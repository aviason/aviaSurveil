package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const firstPartyAdminMaxResponseBytes int64 = 128 * 1024

type FirstPartyAdminConfig struct {
	BaseURL    string
	Secret     string
	SecretFile string
	HTTPClient *http.Client
}

type FirstPartyAdminClient struct {
	baseURL    *url.URL
	secret     []byte
	httpClient *http.Client
}

func NewFirstPartyAdminClient(configuration FirstPartyAdminConfig) (*FirstPartyAdminClient, error) {
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(configuration.BaseURL), "/"))
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("first-party admin URL must be an absolute HTTP(S) URL")
	}
	secret := strings.TrimSpace(configuration.Secret)
	if strings.TrimSpace(configuration.SecretFile) != "" {
		secretBytes, readErr := readFirstPartySecret(configuration.SecretFile)
		if readErr != nil {
			return nil, readErr
		}
		secret = strings.TrimSpace(string(secretBytes))
	}
	if len(secret) < 32 || strings.ContainsAny(secret, "\r\n") {
		return nil, fmt.Errorf("first-party admin secret must contain at least 32 bytes")
	}
	httpClient := configuration.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &FirstPartyAdminClient{baseURL: baseURL, secret: []byte(secret), httpClient: httpClient}, nil
}

func (client *FirstPartyAdminClient) ListDirectory(ctx context.Context, query ProviderDirectoryQuery) (ProviderDirectoryPage, error) {
	if query.First < 0 || query.Limit < 1 || query.Limit > 100 {
		return ProviderDirectoryPage{}, ErrProviderPermanent
	}
	endpoint := client.endpoint("/v1/directory")
	values := endpoint.Query()
	values.Set("first", strconv.Itoa(query.First))
	values.Set("limit", strconv.Itoa(query.Limit))
	if strings.TrimSpace(query.Search) != "" {
		values.Set("search", strings.TrimSpace(query.Search))
	}
	endpoint.RawQuery = values.Encode()
	var response struct {
		Users         []firstPartyDirectoryUser `json:"users"`
		NextFirst     int                       `json:"nextFirst"`
		ProviderCalls int                       `json:"providerCalls"`
	}
	if err := client.do(ctx, http.MethodGet, endpoint.String(), nil, "", &response); err != nil {
		return ProviderDirectoryPage{}, err
	}
	page := ProviderDirectoryPage{NextFirst: response.NextFirst, ProviderCalls: response.ProviderCalls, Users: make([]ProviderDirectoryUser, 0, len(response.Users))}
	for _, user := range response.Users {
		page.Users = append(page.Users, user.toProviderDirectoryUser())
	}
	return page, nil
}

func (client *FirstPartyAdminClient) ObserveUserAuthority(ctx context.Context, subjectID string) (AuthorityObservation, error) {
	var user firstPartyDirectoryUser
	if err := client.do(ctx, http.MethodGet, client.endpoint("/v1/users", subjectID, "authority").String(), nil, "", &user); err != nil {
		return AuthorityObservation{}, err
	}
	return user.toAuthorityObservation(), nil
}

func (client *FirstPartyAdminClient) ProvisionUser(ctx context.Context, user ProviderUser) (string, error) {
	return client.ProvisionUserAtRevision(ctx, user, 0, 1)
}

func (client *FirstPartyAdminClient) ProvisionUserAtRevision(ctx context.Context, user ProviderUser, expected, resulting int64) (string, error) {
	membershipID := strings.TrimSpace(user.MembershipID)
	if membershipID == "" {
		membershipID = "membership-provider-" + digestString(strings.ToLower(strings.TrimSpace(user.Email)))[:24]
	}
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	}
	body := map[string]any{
		"email": user.Email, "givenName": user.FirstName, "familyName": user.LastName,
		"displayName":    displayName,
		"organizationId": user.OrganizationID, "role": roleForProvider(user.Roles), "membershipId": membershipID,
		"expectedMembershipRevision": expected, "resultingMembershipRevision": resulting,
	}
	var response struct {
		SubjectID string `json:"subjectId"`
	}
	if err := client.do(ctx, http.MethodPost, client.endpoint("/v1/users").String(), body, "", &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.SubjectID) == "" {
		return "", ErrProviderManualReview
	}
	return response.SubjectID, nil
}

func (client *FirstPartyAdminClient) ReconcileProvisionedUser(ctx context.Context, user ProviderUser) (string, bool, error) {
	page, err := client.ListDirectory(ctx, ProviderDirectoryQuery{First: 0, Limit: 25, Search: user.Email})
	if err != nil {
		return "", false, err
	}
	matches := make([]ProviderDirectoryUser, 0, 1)
	for _, candidate := range page.Users {
		if strings.EqualFold(candidate.Email, strings.TrimSpace(user.Email)) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return "", false, nil
	}
	candidate := matches[0]
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	}
	membershipID := strings.TrimSpace(user.MembershipID)
	if membershipID == "" {
		membershipID = "membership-provider-" + digestString(strings.ToLower(strings.TrimSpace(user.Email)))[:24]
	}
	if candidate.DisplayName != displayName || candidate.OrganizationID != user.OrganizationID ||
		candidate.MembershipID != membershipID || candidate.MembershipRevision != 1 ||
		len(candidate.Roles) != 1 || candidate.Roles[0] != Role(roleForProvider(user.Roles)) {
		return candidate.SubjectID, false, nil
	}
	return candidate.SubjectID, true, nil
}

func (client *FirstPartyAdminClient) DisableUser(ctx context.Context, subjectID string) error {
	return client.setUserState(ctx, subjectID, "DISABLED")
}

func (client *FirstPartyAdminClient) EnableUser(ctx context.Context, subjectID string) error {
	// Reactivation deliberately returns the account to INVITED so the
	// dedicated auth-mail flow establishes a fresh password before activation.
	return client.setUserState(ctx, subjectID, "INVITED")
}

func (client *FirstPartyAdminClient) SetUserStateAtRevision(ctx context.Context, subjectID, state string, expected, resulting int64) error {
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "state").String(), map[string]any{
		"state": state, "expectedMembershipRevision": expected, "resultingMembershipRevision": resulting,
	})
}

func (client *FirstPartyAdminClient) setUserState(ctx context.Context, subjectID, state string) error {
	observation, err := client.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		return err
	}
	return client.SetUserStateAtRevision(ctx, subjectID, state, observation.MembershipRevision, observation.MembershipRevision+1)
}

func (client *FirstPartyAdminClient) UpdateUserAuthority(ctx context.Context, subjectID, organizationID string, roles []Role) error {
	observation, err := client.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		return err
	}
	return client.UpdateUserAuthorityAtRevision(ctx, subjectID, organizationID, roles, observation.MembershipID, observation.MembershipRevision, observation.MembershipRevision+1)
}

func (client *FirstPartyAdminClient) UpdateUserAuthorityAtRevision(ctx context.Context, subjectID, organizationID string, roles []Role, membershipID string, expected, resulting int64) error {
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "authority").String(), map[string]any{
		"membershipId": membershipID, "organizationId": organizationID, "role": roleForProvider(roles), "state": "ACTIVE",
		"expectedMembershipRevision": expected, "resultingMembershipRevision": resulting,
	})
}

func (client *FirstPartyAdminClient) IssueExecuteActionsEmail(ctx context.Context, subjectID string, actions []string, lifespanSeconds int) error {
	observation, err := client.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		return err
	}
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "actions").String(), map[string]any{
		"actions": actions, "lifespanSeconds": lifespanSeconds,
		"expectedMembershipRevision":  observation.MembershipRevision,
		"resultingMembershipRevision": observation.MembershipRevision,
	})
}

func (client *FirstPartyAdminClient) ActivateUserAtRevision(ctx context.Context, subjectID, password string, expected, resulting int64) error {
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "activate").String(), map[string]any{
		"password": password, "expectedMembershipRevision": expected, "resultingMembershipRevision": resulting,
	})
}

func (client *FirstPartyAdminClient) ResetUserMFA(ctx context.Context, subjectID string) error {
	observation, err := client.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		return err
	}
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "mfa", "reset").String(), map[string]any{
		"expectedAuthRevision":  observation.AuthRevision,
		"resultingAuthRevision": observation.AuthRevision + 1,
	})
}

func (client *FirstPartyAdminClient) ForceUserLogout(ctx context.Context, subjectID string) error {
	observation, err := client.ObserveUserAuthority(ctx, subjectID)
	if err != nil {
		return err
	}
	return client.postRevisioned(ctx, http.MethodPost, client.endpoint("/v1/users", subjectID, "sessions", "revoke").String(), map[string]any{
		"expectedAuthRevision":  observation.AuthRevision,
		"resultingAuthRevision": observation.AuthRevision,
	})
}

func (client *FirstPartyAdminClient) postRevisioned(ctx context.Context, method, endpoint string, body any) error {
	return client.do(ctx, method, endpoint, body, "", &struct{}{})
}

func (client *FirstPartyAdminClient) do(ctx context.Context, method, endpoint string, body any, operationID string, output any) error {
	var requestBody io.Reader
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return ErrProviderPermanent
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
	if err != nil {
		return ErrProviderPermanent
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+string(client.secret))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		if operationID == "" {
			operationID = operationKey(method, endpoint, encoded)
		}
		request.Header.Set("Idempotency-Key", operationID)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("first-party provider request: %w: %w", err, ErrProviderUnavailable)
	}
	defer response.Body.Close()
	payload, readErr := io.ReadAll(io.LimitReader(response.Body, firstPartyAdminMaxResponseBytes+1))
	if readErr != nil || int64(len(payload)) > firstPartyAdminMaxResponseBytes {
		return ErrProviderUnavailable
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return mapFirstPartyError(response.StatusCode, payload)
	}
	if output != nil && len(bytes.TrimSpace(payload)) > 0 {
		if err := json.Unmarshal(payload, output); err != nil {
			return ErrProviderManualReview
		}
	}
	return nil
}

func (client *FirstPartyAdminClient) endpoint(parts ...string) *url.URL {
	endpoint := *client.baseURL
	segments := make([]string, 0, len(parts))
	for _, segment := range parts {
		for _, part := range strings.Split(strings.Trim(segment, "/"), "/") {
			if part != "" {
				segments = append(segments, part)
			}
		}
	}
	endpoint.Path = path.Join(strings.TrimRight(client.baseURL.Path, "/"), strings.Join(segments, "/"))
	return &endpoint
}

type firstPartyDirectoryUser struct {
	SubjectID          string   `json:"subjectId"`
	Email              string   `json:"email"`
	DisplayName        string   `json:"displayName"`
	OrganizationID     string   `json:"organizationId"`
	Enabled            bool     `json:"enabled"`
	TOTPConfigured     bool     `json:"totpConfigured"`
	RequiredActions    []string `json:"requiredActions"`
	Roles              []Role   `json:"roles"`
	MembershipID       string   `json:"membershipId"`
	MembershipRevision int64    `json:"membershipRevision"`
	AuthRevision       uint64   `json:"authRevision"`
	State              string   `json:"state"`
}

func (user firstPartyDirectoryUser) toProviderDirectoryUser() ProviderDirectoryUser {
	return ProviderDirectoryUser{SubjectID: user.SubjectID, Email: user.Email, DisplayName: user.DisplayName, OrganizationID: user.OrganizationID, Enabled: user.Enabled, TOTPConfigured: user.TOTPConfigured, RequiredActions: append([]string(nil), user.RequiredActions...), Roles: append([]Role(nil), user.Roles...), MembershipID: user.MembershipID, MembershipRevision: user.MembershipRevision, AuthRevision: user.AuthRevision, State: user.State}
}

func (user firstPartyDirectoryUser) toAuthorityObservation() AuthorityObservation {
	return AuthorityObservation{SubjectID: user.SubjectID, Enabled: user.Enabled, OrganizationID: user.OrganizationID, Roles: append([]Role(nil), user.Roles...), RequiredActions: append([]string(nil), user.RequiredActions...), MFAEnrolled: user.TOTPConfigured, State: user.State, MembershipID: user.MembershipID, MembershipRevision: user.MembershipRevision, AuthRevision: user.AuthRevision}
}

func roleForProvider(roles []Role) string {
	if len(roles) != 1 {
		return ""
	}
	return string(roles[0])
}

func operationKey(method, endpoint string, body []byte) string {
	digest := sha256.Sum256(append([]byte(method+"\x00"+endpoint+"\x00"), body...))
	return "op_" + hex.EncodeToString(digest[:])[:48]
}

func digestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func mapFirstPartyError(status int, payload []byte) error {
	var response struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(payload, &response)
	var mapped error
	switch response.Code {
	case "DUPLICATE_IDENTIFIER":
		mapped = ErrProviderDuplicateEmail
	case "REVISION_CONFLICT", "INVALID_REVISION":
		mapped = ErrProviderRevisionConflict
	case "NOT_FOUND", "MFA_NOT_CONFIGURED":
		mapped = ErrProviderPermanent
	case "PROVIDER_UNAVAILABLE", "ADMIN_UNAUTHORIZED":
		mapped = ErrProviderUnavailable
	case "OPERATION_ID_REUSE":
		mapped = ErrProviderManualReview
	}
	if mapped == nil {
		if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
			mapped = ErrProviderUnavailable
		} else if status == http.StatusUnauthorized || status == http.StatusForbidden {
			mapped = ErrProviderManualReview
		} else {
			mapped = ErrProviderPermanent
		}
	}
	return mapped
}

func readFirstPartySecret(name string) ([]byte, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("first-party admin secret file is required")
	}
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o222 != 0 || info.Mode()&os.ModeSymlink != 0 || info.Size() < 32 || info.Size() > 4096 {
		return nil, fmt.Errorf("first-party admin secret file is invalid")
	}
	contents, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read first-party admin secret file: %w", err)
	}
	return contents, nil
}
