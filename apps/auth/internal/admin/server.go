package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mail"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/provider"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxJSONBytes        int64 = 64 * 1024
	maxOperationIDBytes       = 160
	operationStaleAfter       = 2 * time.Minute
)

type Config struct {
	Pool       *pgxpool.Pool
	Secret     string
	Issuer     string
	Identity   *identity.PostgresStore
	MFA        *mfa.PostgresStore
	Challenges *challenge.PostgresStore
	Outbox     *mail.Outbox
	Provider   *provider.PostgresStorage
}

type Server struct {
	pool       *pgxpool.Pool
	secret     []byte
	issuer     string
	identity   *identity.PostgresStore
	mfa        *mfa.PostgresStore
	challenges *challenge.PostgresStore
	outbox     *mail.Outbox
	provider   *provider.PostgresStorage
}

func NewServer(configuration Config) (*Server, error) {
	if configuration.Pool == nil || configuration.Identity == nil || configuration.MFA == nil || configuration.Challenges == nil || configuration.Outbox == nil || configuration.Provider == nil {
		return nil, errors.New("provider-admin server dependencies are incomplete")
	}
	secret := strings.TrimSpace(configuration.Secret)
	if len(secret) < 32 || strings.ContainsAny(secret, "\r\n") {
		return nil, errors.New("provider-admin bearer secret must contain at least 32 bytes")
	}
	issuer := strings.TrimRight(strings.TrimSpace(configuration.Issuer), "/")
	if issuer == "" {
		return nil, errors.New("provider-admin issuer is required")
	}
	return &Server{
		pool: configuration.Pool, secret: []byte(secret), issuer: issuer,
		identity: configuration.Identity, mfa: configuration.MFA,
		challenges: configuration.Challenges, outbox: configuration.Outbox,
		provider: configuration.Provider,
	}, nil
}

func (server *Server) Handler() http.Handler {
	return http.HandlerFunc(server.serveHTTP)
}

func (server *Server) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/health/live" {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "alive"})
		return
	}
	if !server.authorized(request) {
		writeAdminError(writer, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED", "provider administration is unavailable")
		return
	}
	if request.Method == http.MethodGet && request.URL.Path == "/v1/directory" {
		server.directory(writer, request)
		return
	}
	if !strings.HasPrefix(request.URL.Path, "/v1/users/") && !(request.Method == http.MethodPost && request.URL.Path == "/v1/users") {
		writeAdminError(writer, http.StatusNotFound, "NOT_FOUND", "provider administration route was not found")
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/users" {
		server.provision(writer, request)
		return
	}
	server.userRoute(writer, request)
}

func (server *Server) authorized(request *http.Request) bool {
	value := strings.TrimSpace(request.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(value) <= len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return false
	}
	provided := strings.TrimSpace(value[len(prefix):])
	if len(provided) != len(server.secret) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), server.secret) == 1
}

func (server *Server) directory(writer http.ResponseWriter, request *http.Request) {
	first, _ := strconv.Atoi(request.URL.Query().Get("first"))
	limit := 25
	if raw := request.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	items, more, err := server.identity.ListProviderAuthorities(request.Context(), first, limit, request.URL.Query().Get("search"))
	if err != nil {
		writeMappedError(writer, err)
		return
	}
	users := make([]directoryUser, 0, len(items))
	for _, item := range items {
		users = append(users, toDirectoryUser(item))
	}
	nextFirst := 0
	if more {
		nextFirst = first + len(users)
	}
	writeJSON(writer, http.StatusOK, map[string]any{"users": users, "nextFirst": nextFirst, "providerCalls": 1})
}

func (server *Server) provision(writer http.ResponseWriter, request *http.Request) {
	body, input, ok := decodeBody[provisionRequest](writer, request)
	if !ok {
		return
	}
	if strings.TrimSpace(input.IdempotencyKey) != "" && input.IdempotencyKey != strings.TrimSpace(request.Header.Get("Idempotency-Key")) {
		writeAdminError(writer, http.StatusBadRequest, "IDEMPOTENCY_KEY_MISMATCH", "idempotency key is invalid")
		return
	}
	if input.ExpectedMembershipRevision != 0 || input.ResultingMembershipRevision != 1 || input.MembershipID == "" {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_REVISION", "provisioning requires membership revision 0 to 1")
		return
	}
	server.writeOperation(writer, request, "PROVISION", body, func(ctx context.Context) (int, any, error) {
		profile := identity.ProviderProfileInput{DisplayName: strings.TrimSpace(input.DisplayName), GivenName: strings.TrimSpace(input.GivenName), FamilyName: strings.TrimSpace(input.FamilyName)}
		account, invitation, err := server.identity.ProvisionProviderInvitation(ctx, identity.InvitationInput{Email: input.Email}, profile, identity.ProviderAuthorityInput{
			MembershipID: input.MembershipID, OrganizationID: input.OrganizationID, Role: input.Role,
			ExpectedRevision: 0, ResultingRevision: 1,
		})
		if err != nil {
			return 0, nil, err
		}
		if err := server.enqueueInvitation(ctx, account.Email, invitation.SubjectID, invitation.Token); err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_USER_PROVISIONED", account.SubjectID, request.Header.Get("Idempotency-Key"), map[string]any{"membershipRevision": 1}); err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, map[string]any{
			"subjectId": account.SubjectID, "membershipId": input.MembershipID,
			"membershipRevision": 1, "authRevision": account.AuthRevision, "state": "INVITED",
		}, nil
	})
}

func (server *Server) userRoute(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(request.URL.Path, "/v1/users/"), "/"), "/")
	if len(parts) == 1 && request.Method == http.MethodGet {
		server.authority(writer, request, parts[0])
		return
	}
	if len(parts) != 2 && len(parts) != 3 {
		writeAdminError(writer, http.StatusNotFound, "NOT_FOUND", "provider administration route was not found")
		return
	}
	subjectID, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(subjectID) == "" {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_SUBJECT", "subject is invalid")
		return
	}
	switch {
	case len(parts) == 2 && parts[1] == "authority" && request.Method == http.MethodGet:
		server.authority(writer, request, subjectID)
	case len(parts) == 2 && parts[1] == "authority" && request.Method == http.MethodPost:
		server.updateAuthority(writer, request, subjectID)
	case len(parts) == 2 && parts[1] == "state" && request.Method == http.MethodPost:
		server.updateState(writer, request, subjectID)
	case len(parts) == 2 && parts[1] == "actions" && request.Method == http.MethodPost:
		server.actions(writer, request, subjectID)
	case len(parts) == 3 && parts[1] == "mfa" && parts[2] == "reset" && request.Method == http.MethodPost:
		server.resetMFA(writer, request, subjectID)
	case len(parts) == 3 && parts[1] == "sessions" && parts[2] == "revoke" && request.Method == http.MethodPost:
		server.revokeSessions(writer, request, subjectID)
	default:
		writeAdminError(writer, http.StatusNotFound, "NOT_FOUND", "provider administration route was not found")
	}
}

func (server *Server) authority(writer http.ResponseWriter, request *http.Request, subjectID string) {
	item, err := server.identity.ObserveProviderAuthority(request.Context(), subjectID)
	if err != nil {
		writeMappedError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, toDirectoryUser(item))
}

func (server *Server) updateAuthority(writer http.ResponseWriter, request *http.Request, subjectID string) {
	body, input, ok := decodeBody[authorityRequest](writer, request)
	if !ok {
		return
	}
	server.writeOperation(writer, request, "AUTHORITY_UPDATE", body, func(ctx context.Context) (int, any, error) {
		item, err := server.identity.UpdateProviderAuthority(ctx, subjectID, identity.ProviderAuthorityInput{
			MembershipID: input.MembershipID, OrganizationID: input.OrganizationID, Role: input.Role, State: input.State,
			ExpectedRevision: input.ExpectedMembershipRevision, ResultingRevision: input.ResultingMembershipRevision,
		})
		if err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_AUTHORITY_UPDATED", subjectID, request.Header.Get("Idempotency-Key"), map[string]any{"membershipRevision": item.MembershipRevision}); err != nil {
			return 0, nil, err
		}
		return http.StatusOK, toDirectoryUser(item), nil
	})
}

func (server *Server) updateState(writer http.ResponseWriter, request *http.Request, subjectID string) {
	body, input, ok := decodeBody[stateRequest](writer, request)
	if !ok {
		return
	}
	server.writeOperation(writer, request, "AUTHORITY_STATE_UPDATE", body, func(ctx context.Context) (int, any, error) {
		item, err := server.identity.SetProviderAuthorityState(ctx, subjectID, input.State, input.ExpectedMembershipRevision, input.ResultingMembershipRevision)
		if err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_AUTHORITY_STATE_UPDATED", subjectID, request.Header.Get("Idempotency-Key"), map[string]any{"membershipRevision": item.MembershipRevision, "state": item.State}); err != nil {
			return 0, nil, err
		}
		return http.StatusOK, toDirectoryUser(item), nil
	})
}

func (server *Server) actions(writer http.ResponseWriter, request *http.Request, subjectID string) {
	body, input, ok := decodeBody[actionRequest](writer, request)
	if !ok {
		return
	}
	if len(input.Actions) != 2 || input.Actions[0] != "UPDATE_PASSWORD" || input.Actions[1] != "VERIFY_EMAIL" || input.LifespanSeconds < 60 || input.LifespanSeconds > 24*60*60 || input.ExpectedMembershipRevision < 1 || input.ResultingMembershipRevision != input.ExpectedMembershipRevision {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_REVISION", "identity action requires a bounded, non-mutating revision")
		return
	}
	server.writeOperation(writer, request, "IDENTITY_ACTION_EMAIL", body, func(ctx context.Context) (int, any, error) {
		item, err := server.identity.ObserveProviderAuthority(ctx, subjectID)
		if err != nil {
			return 0, nil, err
		}
		if item.MembershipRevision != input.ExpectedMembershipRevision {
			return 0, nil, identity.ErrProviderRevisionConflict
		}
		var link string
		if item.State == "INVITED" {
			invitation, resendErr := server.identity.ResendInvitation(ctx, subjectID)
			if resendErr != nil {
				return 0, nil, resendErr
			}
			link = server.issuer + "/activate?" + url.Values{"subject": {subjectID}, "token": {invitation.Token}}.Encode()
		} else {
			issued, issueErr := server.challenges.Issue(ctx, subjectID, challenge.PurposePasswordReset, time.Duration(input.LifespanSeconds)*time.Second, 5)
			if issueErr != nil {
				return 0, nil, issueErr
			}
			link = server.issuer + "/recover/password?" + url.Values{"subject": {subjectID}, "token": {issued.Token}}.Encode()
		}
		if _, err := server.outbox.Enqueue(ctx, mail.Delivery{Recipient: item.Email, Subject: "AviaSurveil360 account access", Body: "Use this one-time access link within the configured lifetime: " + link}); err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_IDENTITY_ACTION_ISSUED", subjectID, request.Header.Get("Idempotency-Key"), map[string]any{"membershipRevision": item.MembershipRevision}); err != nil {
			return 0, nil, err
		}
		return http.StatusAccepted, map[string]any{"subjectId": subjectID, "membershipRevision": item.MembershipRevision, "accepted": true}, nil
	})
}

func (server *Server) resetMFA(writer http.ResponseWriter, request *http.Request, subjectID string) {
	body, input, ok := decodeBody[revisionRequest](writer, request)
	if !ok {
		return
	}
	server.writeOperation(writer, request, "MFA_RESET", body, func(ctx context.Context) (int, any, error) {
		item, err := server.identity.ObserveProviderAuthority(ctx, subjectID)
		if err != nil {
			return 0, nil, err
		}
		if input.ExpectedAuthRevision != item.AuthRevision || input.ResultingAuthRevision != input.ExpectedAuthRevision+1 {
			return 0, nil, identity.ErrRevisionConflict
		}
		resulting, err := server.mfa.ResetAtAuthRevision(ctx, subjectID, input.ExpectedAuthRevision)
		if err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_MFA_RESET", subjectID, request.Header.Get("Idempotency-Key"), map[string]any{"authRevision": item.AuthRevision}); err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"subjectId": subjectID, "authRevision": resulting, "accepted": true}, nil
	})
}

func (server *Server) revokeSessions(writer http.ResponseWriter, request *http.Request, subjectID string) {
	body, input, ok := decodeBody[revisionRequest](writer, request)
	if !ok {
		return
	}
	server.writeOperation(writer, request, "SESSION_REVOCATION", body, func(ctx context.Context) (int, any, error) {
		item, err := server.identity.ObserveProviderAuthority(ctx, subjectID)
		if err != nil {
			return 0, nil, err
		}
		if input.ExpectedAuthRevision != item.AuthRevision || input.ResultingAuthRevision != input.ExpectedAuthRevision {
			return 0, nil, identity.ErrRevisionConflict
		}
		if err := server.provider.RevokeAllSessions(ctx, subjectID); err != nil {
			return 0, nil, err
		}
		if err := server.audit(ctx, "PROVIDER_SESSIONS_REVOKED", subjectID, request.Header.Get("Idempotency-Key"), nil); err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"subjectId": subjectID, "authRevision": item.AuthRevision, "accepted": true}, nil
	})
}

func (server *Server) enqueueInvitation(ctx context.Context, recipient, subjectID, token string) error {
	link := server.issuer + "/activate?" + url.Values{"subject": {subjectID}, "token": {token}}.Encode()
	_, err := server.outbox.Enqueue(ctx, mail.Delivery{Recipient: recipient, Subject: "Activate your AviaSurveil360 account", Body: "Use this one-time activation link within 24 hours: " + link})
	return err
}

func (server *Server) writeOperation(writer http.ResponseWriter, request *http.Request, kind string, body []byte, operation func(context.Context) (int, any, error)) {
	operationID := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if operationID == "" || len(operationID) > maxOperationIDBytes || strings.ContainsAny(operationID, "\r\n") {
		writeAdminError(writer, http.StatusBadRequest, "IDEMPOTENCY_REQUIRED", "Idempotency-Key is required")
		return
	}
	hash := sha256.Sum256(append([]byte(request.Method+"\x00"+request.URL.Path+"\x00"), body...))
	claimed, status, responseBody, err := server.claimOperation(request.Context(), operationID, hash[:], kind)
	if err != nil {
		writeMappedError(writer, err)
		return
	}
	if !claimed {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write(responseBody)
		return
	}
	resultStatus, result, operationErr := operation(request.Context())
	if operationErr != nil {
		statusCode, code, message := mappedError(operationErr)
		payload, _ := json.Marshal(map[string]any{"code": code, "message": message})
		_ = server.finishOperation(request.Context(), operationID, "FAILED", statusCode, payload)
		writeJSONBytes(writer, statusCode, payload)
		return
	}
	payload, err := json.Marshal(result)
	if err != nil {
		payload = []byte(`{"code":"INTERNAL_ERROR","message":"provider administration is unavailable"}`)
		resultStatus = http.StatusServiceUnavailable
		_ = server.finishOperation(request.Context(), operationID, "FAILED", resultStatus, payload)
		writeJSONBytes(writer, resultStatus, payload)
		return
	}
	if err := server.finishOperation(request.Context(), operationID, "SUCCEEDED", resultStatus, payload); err != nil {
		writeMappedError(writer, err)
		return
	}
	writeJSONBytes(writer, resultStatus, payload)
}

func (server *Server) claimOperation(ctx context.Context, operationID string, requestHash []byte, kind string) (bool, int, []byte, error) {
	tx, err := server.pool.Begin(ctx)
	if err != nil {
		return false, 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Now().UTC()
	command, err := tx.Exec(ctx, `INSERT INTO auth_identity.provider_admin_operation_receipts(operation_id, request_hash, operation_kind, state, response_status, response_body, created_at, updated_at) VALUES ($1,$2,$3,'PROCESSING',202, '{"code":"PROCESSING","message":"operation is in progress"}'::jsonb,$4,$4) ON CONFLICT DO NOTHING`, operationID, requestHash, kind, now)
	if err != nil {
		return false, 0, nil, err
	}
	if command.RowsAffected() == 0 {
		var storedHash []byte
		var state string
		var status int
		var response []byte
		var updatedAt time.Time
		if err := tx.QueryRow(ctx, `SELECT request_hash, state, response_status, response_body, updated_at FROM auth_identity.provider_admin_operation_receipts WHERE operation_id = $1 FOR UPDATE`, operationID).Scan(&storedHash, &state, &status, &response, &updatedAt); err != nil {
			return false, 0, nil, err
		}
		if subtle.ConstantTimeCompare(storedHash, requestHash) != 1 {
			return false, 0, nil, operationConflictError{}
		}
		if state != "PROCESSING" || time.Since(updatedAt) < operationStaleAfter {
			if err := tx.Commit(ctx); err != nil {
				return false, 0, nil, err
			}
			return false, status, response, nil
		}
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.provider_admin_operation_receipts SET state = 'PROCESSING', updated_at = $2 WHERE operation_id = $1`, operationID, now); err != nil {
			return false, 0, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, 0, nil, err
	}
	return true, 0, nil, nil
}

func (server *Server) finishOperation(ctx context.Context, operationID, state string, status int, response []byte) error {
	_, err := server.pool.Exec(ctx, `UPDATE auth_identity.provider_admin_operation_receipts SET state = $2, response_status = $3, response_body = $4::jsonb, updated_at = $5 WHERE operation_id = $1`, operationID, state, status, response, time.Now().UTC())
	return err
}

func (server *Server) audit(ctx context.Context, eventType, subjectID, operationID string, fields map[string]any) error {
	if fields == nil {
		fields = map[string]any{}
	}
	redacted, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	_, err = server.pool.Exec(ctx, `INSERT INTO auth_identity.audit_events(event_id, event_at, event_type, outcome, subject_id, request_id, fields) VALUES ($1,$2,$3,'accepted',$4,$5,$6::jsonb)`, auditID(eventType, subjectID, operationID), time.Now().UTC(), eventType, subjectID, operationID, redacted)
	return err
}

type operationConflictError struct{}

func (operationConflictError) Error() string {
	return "operation ID was reused with a different request"
}

type provisionRequest struct {
	IdempotencyKey              string `json:"operationId,omitempty"`
	Email                       string `json:"email"`
	GivenName                   string `json:"givenName"`
	FamilyName                  string `json:"familyName"`
	DisplayName                 string `json:"displayName"`
	OrganizationID              string `json:"organizationId"`
	Role                        string `json:"role"`
	MembershipID                string `json:"membershipId"`
	ExpectedMembershipRevision  int64  `json:"expectedMembershipRevision"`
	ResultingMembershipRevision int64  `json:"resultingMembershipRevision"`
}

type authorityRequest struct {
	MembershipID                string `json:"membershipId"`
	OrganizationID              string `json:"organizationId"`
	Role                        string `json:"role"`
	State                       string `json:"state"`
	ExpectedMembershipRevision  int64  `json:"expectedMembershipRevision"`
	ResultingMembershipRevision int64  `json:"resultingMembershipRevision"`
}

type stateRequest struct {
	State                       string `json:"state"`
	ExpectedMembershipRevision  int64  `json:"expectedMembershipRevision"`
	ResultingMembershipRevision int64  `json:"resultingMembershipRevision"`
}

type actionRequest struct {
	Actions                     []string `json:"actions"`
	LifespanSeconds             int      `json:"lifespanSeconds"`
	ExpectedMembershipRevision  int64    `json:"expectedMembershipRevision"`
	ResultingMembershipRevision int64    `json:"resultingMembershipRevision"`
}

type revisionRequest struct {
	ExpectedMembershipRevision  int64  `json:"expectedMembershipRevision,omitempty"`
	ResultingMembershipRevision int64  `json:"resultingMembershipRevision,omitempty"`
	ExpectedAuthRevision        uint64 `json:"expectedAuthRevision"`
	ResultingAuthRevision       uint64 `json:"resultingAuthRevision"`
}

type directoryUser struct {
	SubjectID          string   `json:"subjectId"`
	Email              string   `json:"email"`
	DisplayName        string   `json:"displayName"`
	OrganizationID     string   `json:"organizationId"`
	Enabled            bool     `json:"enabled"`
	TOTPConfigured     bool     `json:"totpConfigured"`
	RequiredActions    []string `json:"requiredActions"`
	Roles              []string `json:"roles"`
	MembershipID       string   `json:"membershipId"`
	MembershipRevision int64    `json:"membershipRevision"`
	AuthRevision       uint64   `json:"authRevision"`
	State              string   `json:"state"`
}

func toDirectoryUser(item identity.ProviderAuthority) directoryUser {
	required := []string(nil)
	if item.State == "INVITED" || !item.EmailVerified {
		required = []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}
	}
	return directoryUser{
		SubjectID: item.SubjectID, Email: item.Email, DisplayName: item.DisplayName,
		OrganizationID: item.OrganizationID, Enabled: item.State == "ACTIVE" && item.EmailVerified,
		TOTPConfigured: item.MFAEnrolled, RequiredActions: required,
		Roles: []string{item.Role}, MembershipID: item.MembershipID,
		MembershipRevision: item.MembershipRevision, AuthRevision: item.AuthRevision, State: item.State,
	}
}

func decodeBody[T any](writer http.ResponseWriter, request *http.Request) ([]byte, T, bool) {
	var value T
	request.Body = http.MaxBytesReader(writer, request.Body, maxJSONBytes)
	raw, err := io.ReadAll(request.Body)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_JSON", "request JSON is invalid")
		return nil, value, false
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_JSON", "request JSON is invalid")
		return nil, value, false
	}
	canonical, err := json.Marshal(normalized)
	if err != nil || int64(len(canonical)) > maxJSONBytes {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_JSON", "request JSON is invalid")
		return nil, value, false
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		writeAdminError(writer, http.StatusBadRequest, "INVALID_JSON", "request JSON is invalid")
		return nil, value, false
	}
	return canonical, value, true
}

func mappedError(err error) (int, string, string) {
	var operationConflict operationConflictError
	switch {
	case errors.As(err, &operationConflict):
		return http.StatusConflict, "OPERATION_ID_REUSE", "operation ID was reused with a different request"
	case errors.Is(err, identity.ErrProviderRevisionConflict), errors.Is(err, identity.ErrRevisionConflict):
		return http.StatusConflict, "REVISION_CONFLICT", "expected revision does not match provider state"
	case errors.Is(err, mfa.ErrRevisionConflict):
		return http.StatusConflict, "REVISION_CONFLICT", "expected revision does not match provider state"
	case errors.Is(err, identity.ErrDuplicateIdentifier):
		return http.StatusConflict, "DUPLICATE_IDENTIFIER", "provider account already exists"
	case errors.Is(err, identity.ErrProviderNotFound), errors.Is(err, identity.ErrAccountNotFound), errors.Is(err, identity.ErrInvitationNotFound):
		return http.StatusNotFound, "NOT_FOUND", "provider account was not found"
	case errors.Is(err, identity.ErrInvitationExpired), errors.Is(err, identity.ErrInvalidRecovery):
		return http.StatusBadRequest, "EXPIRED_OR_INVALID", "provider operation is expired or invalid"
	case errors.Is(err, mfa.ErrFactorNotFound):
		return http.StatusNotFound, "MFA_NOT_CONFIGURED", "MFA is not configured"
	case errors.Is(err, identity.ErrSessionRevocationUnavailable):
		return http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE", "provider security dependency is unavailable"
	default:
		return http.StatusBadRequest, "PROVIDER_OPERATION_REJECTED", "provider operation was rejected"
	}
}

func writeMappedError(writer http.ResponseWriter, err error) {
	status, code, message := mappedError(err)
	writeAdminError(writer, status, code, message)
}

func writeAdminError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]string{"code": code, "message": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		payload = []byte(`{"code":"INTERNAL_ERROR","message":"provider administration is unavailable"}`)
		status = http.StatusServiceUnavailable
	}
	writeJSONBytes(writer, status, payload)
}

func writeJSONBytes(writer http.ResponseWriter, status int, payload []byte) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func auditID(eventType, subjectID, operationID string) string {
	digest := sha256.Sum256([]byte(eventType + "\x00" + subjectID + "\x00" + operationID))
	return "evt_" + hex.EncodeToString(digest[:])[:32]
}
