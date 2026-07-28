package scenarios

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

var syntheticSubjectPattern = regexp.MustCompile(
	`^synthetic-[a-z0-9-]{1,118}$`,
)
var providerSubjectPattern = regexp.MustCompile(
	`^[A-Za-z0-9._:-]{1,128}$`,
)

type KeycloakEndpointConfig struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type KeycloakEndpoint struct {
	baseURL      *url.URL
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

type scenarioKeycloakUserRepresentation struct {
	ID              string              `json:"id,omitempty"`
	Username        string              `json:"username"`
	Email           string              `json:"email"`
	FirstName       string              `json:"firstName"`
	LastName        string              `json:"lastName"`
	Enabled         bool                `json:"enabled"`
	EmailVerified   bool                `json:"emailVerified"`
	RequiredActions []string            `json:"requiredActions"`
	Attributes      map[string][]string `json:"attributes"`
}

type scenarioKeycloakRoleRepresentation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewKeycloakEndpoint(
	config KeycloakEndpointConfig,
) (*KeycloakEndpoint, error) {
	baseURL, err := url.Parse(strings.TrimRight(
		strings.TrimSpace(config.BaseURL),
		"/",
	))
	if err != nil ||
		(baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.Host == "" {
		return nil, fmt.Errorf("valid Keycloak HTTP(S) base URL is required")
	}
	realm := strings.TrimSpace(config.Realm)
	clientID := strings.TrimSpace(config.ClientID)
	if realm == "" || clientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf(
			"Keycloak realm and service-client credentials are required",
		)
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &KeycloakEndpoint{
		baseURL:      baseURL,
		realm:        realm,
		clientID:     clientID,
		clientSecret: config.ClientSecret,
		httpClient:   httpClient,
	}, nil
}

func (endpoint *KeycloakEndpoint) Preflight(ctx context.Context) error {
	token, err := endpoint.accessToken(ctx)
	if err != nil {
		return err
	}
	users, err := endpoint.listSyntheticUsers(ctx, token)
	if err != nil {
		return err
	}
	if len(users) != 0 {
		return fmt.Errorf(
			"connected-scenario Keycloak target retains %d synthetic users",
			len(users),
		)
	}
	return nil
}

func (endpoint *KeycloakEndpoint) EnsureProviderAccount(
	ctx context.Context,
	account ProviderAccount,
) (ProviderAccount, error) {
	if err := validateProviderAccount(account, false); err != nil {
		return ProviderAccount{}, err
	}
	token, err := endpoint.accessToken(ctx)
	if err != nil {
		return ProviderAccount{}, err
	}
	user, found, err := endpoint.findUserByEmail(ctx, token, account.Email)
	if err != nil {
		return ProviderAccount{}, err
	}
	if found {
		account.SubjectID = user.ID
	} else {
		user = keycloakUserForAccount(account)
		response, err := endpoint.doJSON(
			ctx,
			http.MethodPost,
			endpoint.adminURL("users"),
			token,
			user,
			http.StatusCreated,
		)
		if err != nil {
			return ProviderAccount{}, fmt.Errorf(
				"create exact synthetic Keycloak user: %w",
				err,
			)
		}
		account.SubjectID, err = createdProviderSubjectID(
			response.Header.Get("Location"),
		)
		response.Body.Close()
		if err != nil {
			return ProviderAccount{}, err
		}
		user, found, err = endpoint.readUser(ctx, token, account.SubjectID)
		if err != nil {
			return ProviderAccount{}, err
		}
		if !found {
			return ProviderAccount{}, fmt.Errorf(
				"created synthetic Keycloak user is not readable",
			)
		}
	}
	if !sameKeycloakUser(user, keycloakUserForAccount(account)) {
		return ProviderAccount{}, fmt.Errorf(
			"existing synthetic Keycloak user %s differs from scenario",
			account.SubjectID,
		)
	}

	mapped, err := endpoint.userRoles(ctx, token, account.SubjectID)
	if err != nil {
		return ProviderAccount{}, err
	}
	approved := approvedScenarioRoles(mapped)
	switch {
	case len(approved) == 0:
		role, err := endpoint.realmRole(ctx, token, account.Role)
		if err != nil {
			return ProviderAccount{}, err
		}
		response, err := endpoint.doJSON(
			ctx,
			http.MethodPost,
			endpoint.adminURL(
				"users",
				account.SubjectID,
				"role-mappings",
				"realm",
			),
			token,
			[]scenarioKeycloakRoleRepresentation{role},
			http.StatusNoContent,
		)
		if err != nil {
			return ProviderAccount{}, fmt.Errorf(
				"map synthetic Keycloak role: %w",
				err,
			)
		}
		response.Body.Close()
	case len(approved) != 1 || approved[0] != account.Role:
		return ProviderAccount{}, fmt.Errorf(
			"existing synthetic Keycloak roles differ for %s",
			account.SubjectID,
		)
	}
	if err := endpoint.reconcileAccount(ctx, token, account); err != nil {
		return ProviderAccount{}, err
	}
	return account, nil
}

func (endpoint *KeycloakEndpoint) ReconcileProviderAccounts(
	ctx context.Context,
	expected []ProviderAccount,
) error {
	token, err := endpoint.accessToken(ctx)
	if err != nil {
		return err
	}
	actual, err := endpoint.listSyntheticUsers(ctx, token)
	if err != nil {
		return err
	}
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"Keycloak synthetic account count = %d, expected %d",
			len(actual),
			len(expected),
		)
	}
	expectedByID := make(map[string]ProviderAccount, len(expected))
	for _, account := range expected {
		if err := validateProviderAccount(account, true); err != nil {
			return err
		}
		if _, exists := expectedByID[account.SubjectID]; exists {
			return fmt.Errorf(
				"duplicate expected Keycloak subject %s",
				account.SubjectID,
			)
		}
		expectedByID[account.SubjectID] = account
	}
	for _, user := range actual {
		account, ok := expectedByID[user.ID]
		if !ok {
			return fmt.Errorf("unexpected synthetic Keycloak subject %s", user.ID)
		}
		if err := endpoint.reconcileAccount(ctx, token, account); err != nil {
			return err
		}
	}
	return nil
}

func (endpoint *KeycloakEndpoint) issueActionsEmail(
	ctx context.Context,
	subjectID string,
	actions []string,
) error {
	if !validProviderSubjectID(subjectID) ||
		!sameStrings(
			actions,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		) {
		return fmt.Errorf("invalid synthetic Keycloak invitation request")
	}
	token, err := endpoint.accessToken(ctx)
	if err != nil {
		return err
	}
	requestURL, err := url.Parse(endpoint.adminURL(
		"users",
		subjectID,
		"execute-actions-email",
	))
	if err != nil {
		return err
	}
	query := requestURL.Query()
	query.Set("lifespan", "86400")
	requestURL.RawQuery = query.Encode()
	response, err := endpoint.doJSON(
		ctx,
		http.MethodPut,
		requestURL.String(),
		token,
		actions,
		http.StatusNoContent,
	)
	if err != nil {
		return fmt.Errorf("issue synthetic Keycloak invitation: %w", err)
	}
	response.Body.Close()
	return nil
}

func (endpoint *KeycloakEndpoint) reconcileAccount(
	ctx context.Context,
	token string,
	account ProviderAccount,
) error {
	user, found, err := endpoint.readUser(ctx, token, account.SubjectID)
	if err != nil {
		return err
	}
	if !found || !sameKeycloakUser(user, keycloakUserForAccount(account)) {
		return fmt.Errorf(
			"Keycloak provider account %s differs from scenario",
			account.SubjectID,
		)
	}
	mapped, err := endpoint.userRoles(ctx, token, account.SubjectID)
	if err != nil {
		return err
	}
	approved := approvedScenarioRoles(mapped)
	if len(approved) != 1 || approved[0] != account.Role {
		return fmt.Errorf(
			"Keycloak provider roles differ for %s",
			account.SubjectID,
		)
	}
	return nil
}

func keycloakUserForAccount(
	account ProviderAccount,
) scenarioKeycloakUserRepresentation {
	return scenarioKeycloakUserRepresentation{
		ID:              account.SubjectID,
		Username:        account.Email,
		Email:           account.Email,
		FirstName:       "Synthetic",
		LastName:        strings.ToUpper(account.Role),
		Enabled:         account.Enabled,
		EmailVerified:   false,
		RequiredActions: append([]string(nil), account.RequiredActions...),
		Attributes: map[string][]string{
			"organization_id": {account.OrganizationID},
		},
	}
}

func validateProviderAccount(
	account ProviderAccount,
	requireSubject bool,
) error {
	account.Email = strings.ToLower(strings.TrimSpace(account.Email))
	address, err := mail.ParseAddress(account.Email)
	if err != nil ||
		address.Address != account.Email ||
		!strings.HasSuffix(account.Email, "@synthetic.invalid") ||
		!syntheticSubjectPattern.MatchString(account.ScenarioID) ||
		(requireSubject && !validProviderSubjectID(account.SubjectID)) ||
		(!requireSubject && account.SubjectID != "") ||
		strings.TrimSpace(account.MembershipID) == "" ||
		strings.TrimSpace(account.OrganizationID) == "" ||
		!containsRole(account.Role) ||
		!account.Enabled ||
		!sameStrings(
			account.RequiredActions,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		) {
		return fmt.Errorf("invalid synthetic Keycloak provider account")
	}
	if (account.Role == "auditee") == (account.OrganizationID == "CAA") {
		return fmt.Errorf(
			"synthetic Keycloak role and organization authority differ",
		)
	}
	return nil
}

func validProviderSubjectID(subjectID string) bool {
	return providerSubjectPattern.MatchString(strings.TrimSpace(subjectID))
}

func sameKeycloakUser(
	left,
	right scenarioKeycloakUserRepresentation,
) bool {
	if left.ID != right.ID ||
		left.Username != right.Username ||
		left.Email != right.Email ||
		left.FirstName != right.FirstName ||
		left.LastName != right.LastName ||
		left.Enabled != right.Enabled ||
		left.EmailVerified != right.EmailVerified ||
		!sameStrings(left.RequiredActions, right.RequiredActions) {
		return false
	}
	leftOrganizations := left.Attributes["organization_id"]
	rightOrganizations := right.Attributes["organization_id"]
	return len(leftOrganizations) == 1 &&
		len(rightOrganizations) == 1 &&
		leftOrganizations[0] == rightOrganizations[0]
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func approvedScenarioRoles(
	source []scenarioKeycloakRoleRepresentation,
) []string {
	output := make([]string, 0, len(source))
	for _, role := range source {
		if containsRole(role.Name) {
			output = append(output, role.Name)
		}
	}
	sort.Strings(output)
	return output
}

func (endpoint *KeycloakEndpoint) listSyntheticUsers(
	ctx context.Context,
	token string,
) ([]scenarioKeycloakUserRepresentation, error) {
	const pageSize = 100
	var output []scenarioKeycloakUserRepresentation
	for first := 0; ; first += pageSize {
		pageURL, err := url.Parse(endpoint.adminURL("users"))
		if err != nil {
			return nil, err
		}
		query := pageURL.Query()
		query.Set("first", fmt.Sprintf("%d", first))
		query.Set("max", fmt.Sprintf("%d", pageSize))
		query.Set("briefRepresentation", "false")
		pageURL.RawQuery = query.Encode()
		response, err := endpoint.doJSON(
			ctx,
			http.MethodGet,
			pageURL.String(),
			token,
			nil,
			http.StatusOK,
		)
		if err != nil {
			return nil, fmt.Errorf("list synthetic Keycloak users: %w", err)
		}
		var page []scenarioKeycloakUserRepresentation
		decodeErr := decodeScenarioJSON(response.Body, &page)
		response.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf(
				"decode synthetic Keycloak users: %w",
				decodeErr,
			)
		}
		for _, user := range page {
			if strings.HasSuffix(
				strings.ToLower(user.Email),
				"@synthetic.invalid",
			) {
				output = append(output, user)
			}
		}
		if len(page) < pageSize {
			break
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].ID < output[right].ID
	})
	return output, nil
}

func (endpoint *KeycloakEndpoint) readUser(
	ctx context.Context,
	token,
	subjectID string,
) (scenarioKeycloakUserRepresentation, bool, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.adminURL("users", subjectID),
		nil,
	)
	if err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := endpoint.httpClient.Do(request)
	if err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return scenarioKeycloakUserRepresentation{}, false, nil
	}
	if response.StatusCode != http.StatusOK {
		return scenarioKeycloakUserRepresentation{}, false, fmt.Errorf(
			"read Keycloak user status %d",
			response.StatusCode,
		)
	}
	var user scenarioKeycloakUserRepresentation
	if err := decodeScenarioJSON(response.Body, &user); err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	return user, true, nil
}

func (endpoint *KeycloakEndpoint) findUserByEmail(
	ctx context.Context,
	token,
	email string,
) (scenarioKeycloakUserRepresentation, bool, error) {
	requestURL, err := url.Parse(endpoint.adminURL("users"))
	if err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	query := requestURL.Query()
	query.Set("email", email)
	query.Set("exact", "true")
	query.Set("briefRepresentation", "false")
	query.Set("max", "2")
	requestURL.RawQuery = query.Encode()
	response, err := endpoint.doJSON(
		ctx,
		http.MethodGet,
		requestURL.String(),
		token,
		nil,
		http.StatusOK,
	)
	if err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	defer response.Body.Close()
	var users []scenarioKeycloakUserRepresentation
	if err := decodeScenarioJSON(response.Body, &users); err != nil {
		return scenarioKeycloakUserRepresentation{}, false, err
	}
	switch len(users) {
	case 0:
		return scenarioKeycloakUserRepresentation{}, false, nil
	case 1:
		if users[0].Email != email ||
			!validProviderSubjectID(users[0].ID) {
			return scenarioKeycloakUserRepresentation{}, false, fmt.Errorf(
				"Keycloak email lookup returned an invalid provider account",
			)
		}
		return users[0], true, nil
	default:
		return scenarioKeycloakUserRepresentation{}, false, fmt.Errorf(
			"Keycloak email lookup returned duplicate provider accounts",
		)
	}
}

func (endpoint *KeycloakEndpoint) userRoles(
	ctx context.Context,
	token,
	subjectID string,
) ([]scenarioKeycloakRoleRepresentation, error) {
	response, err := endpoint.doJSON(
		ctx,
		http.MethodGet,
		endpoint.adminURL(
			"users",
			subjectID,
			"role-mappings",
			"realm",
		),
		token,
		nil,
		http.StatusOK,
	)
	if err != nil {
		return nil, fmt.Errorf("list synthetic Keycloak roles: %w", err)
	}
	defer response.Body.Close()
	var roles []scenarioKeycloakRoleRepresentation
	if err := decodeScenarioJSON(response.Body, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (endpoint *KeycloakEndpoint) realmRole(
	ctx context.Context,
	token,
	role string,
) (scenarioKeycloakRoleRepresentation, error) {
	response, err := endpoint.doJSON(
		ctx,
		http.MethodGet,
		endpoint.adminURL("roles", role),
		token,
		nil,
		http.StatusOK,
	)
	if err != nil {
		return scenarioKeycloakRoleRepresentation{}, err
	}
	defer response.Body.Close()
	var representation scenarioKeycloakRoleRepresentation
	if err := decodeScenarioJSON(response.Body, &representation); err != nil {
		return scenarioKeycloakRoleRepresentation{}, err
	}
	if representation.ID == "" || representation.Name != role {
		return scenarioKeycloakRoleRepresentation{}, fmt.Errorf(
			"Keycloak role %s has invalid representation",
			role,
		)
	}
	return representation, nil
}

func createdProviderSubjectID(location string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(location))
	if err != nil || parsed.Path == "" {
		return "", fmt.Errorf(
			"Keycloak create response omitted a valid Location",
		)
	}
	subjectID, err := url.PathUnescape(path.Base(parsed.Path))
	if err != nil || !validProviderSubjectID(subjectID) {
		return "", fmt.Errorf(
			"Keycloak create response omitted a valid provider subject",
		)
	}
	return subjectID, nil
}

func (endpoint *KeycloakEndpoint) accessToken(
	ctx context.Context,
) (string, error) {
	form := url.Values{
		"client_id":     {endpoint.clientID},
		"client_secret": {endpoint.clientSecret},
		"grant_type":    {"client_credentials"},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.url(
			"realms",
			endpoint.realm,
			"protocol",
			"openid-connect",
			"token",
		),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := endpoint.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"Keycloak token status %d",
			response.StatusCode,
		)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := decodeScenarioJSON(response.Body, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("Keycloak token response omitted access_token")
	}
	return payload.AccessToken, nil
}

func (endpoint *KeycloakEndpoint) doJSON(
	ctx context.Context,
	method,
	requestURL,
	token string,
	body any,
	expectedStatus int,
) (*http.Response, error) {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		requestURL,
		requestBody,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := endpoint.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != expectedStatus {
		response.Body.Close()
		return nil, fmt.Errorf(
			"Keycloak %s %s status %d",
			method,
			requestURL,
			response.StatusCode,
		)
	}
	return response, nil
}

func (endpoint *KeycloakEndpoint) adminURL(
	segments ...string,
) string {
	return endpoint.url(
		append([]string{"admin", "realms", endpoint.realm}, segments...)...,
	)
}

func (endpoint *KeycloakEndpoint) url(segments ...string) string {
	output := *endpoint.baseURL
	escaped := make([]string, len(segments))
	for index, segment := range segments {
		escaped[index] = url.PathEscape(segment)
	}
	output.Path = strings.TrimRight(output.Path, "/") +
		"/" + strings.Join(escaped, "/")
	return output.String()
}

func decodeScenarioJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, 2<<20))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("JSON contains trailing content")
	}
	return nil
}
