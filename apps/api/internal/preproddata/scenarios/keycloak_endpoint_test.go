package scenarios_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

func TestKeycloakEndpointCreatesExactProviderAccountAndRejectsDrift(
	t *testing.T,
) {
	fixture := newScenarioKeycloakFixture(t)
	endpoint, err := scenarios.NewKeycloakEndpoint(
		scenarios.KeycloakEndpointConfig{
			BaseURL:      fixture.server.URL + "/identity",
			Realm:        "task7-realm",
			ClientID:     "task7-loader",
			ClientSecret: "task7-secret",
			HTTPClient:   &http.Client{Timeout: 2 * time.Second},
		},
	)
	if err != nil {
		t.Fatalf("new Keycloak endpoint: %v", err)
	}
	ctx := context.Background()
	if err := endpoint.Preflight(ctx); err != nil {
		t.Fatalf("preflight Keycloak endpoint: %v", err)
	}
	account := scenarios.ProviderAccount{
		ScenarioID:     "synthetic-provideraccounts-0001",
		MembershipID:   "synthetic-membership-0001",
		Email:          "user-0001@synthetic.invalid",
		OrganizationID: "AUDITEE-A",
		Role:           "auditee",
		Enabled:        true,
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	account, err = endpoint.EnsureProviderAccount(ctx, account)
	if err != nil {
		t.Fatalf("ensure provider account: %v", err)
	}
	if account.SubjectID != "provider-subject-001" {
		t.Fatalf("provider-assigned subject = %q", account.SubjectID)
	}

	gotUser, gotRoles := fixture.account(account.SubjectID)
	wantUser := scenarioKeycloakUser{
		ID:              account.SubjectID,
		Username:        account.Email,
		Email:           account.Email,
		FirstName:       "Synthetic",
		LastName:        "AUDITEE",
		Enabled:         true,
		EmailVerified:   false,
		RequiredActions: account.RequiredActions,
		Attributes: map[string][]string{
			"organization_id": {"AUDITEE-A"},
		},
	}
	if !reflect.DeepEqual(gotUser, wantUser) ||
		!slices.Equal(gotRoles, []string{"auditee"}) {
		t.Fatalf(
			"Keycloak state = user %#v roles %#v, expected user %#v roles [auditee]",
			gotUser,
			gotRoles,
			wantUser,
		)
	}
	if err := endpoint.ReconcileProviderAccounts(
		ctx,
		[]scenarios.ProviderAccount{account},
	); err != nil {
		t.Fatalf("reconcile exact provider account: %v", err)
	}
	fixture.setOrganization(account.SubjectID, "WRONG-ORG")
	if err := endpoint.ReconcileProviderAccounts(
		ctx,
		[]scenarios.ProviderAccount{account},
	); err == nil {
		t.Fatalf("Keycloak organization drift was accepted")
	}
}

func TestMailpitInvitationEndpointDeliversOnceAndReconcilesRecipient(
	t *testing.T,
) {
	fixture := newScenarioKeycloakFixture(t)
	keycloak, err := scenarios.NewKeycloakEndpoint(
		scenarios.KeycloakEndpointConfig{
			BaseURL:      fixture.server.URL + "/identity",
			Realm:        "task7-realm",
			ClientID:     "task7-loader",
			ClientSecret: "task7-secret",
			HTTPClient:   &http.Client{Timeout: 2 * time.Second},
		},
	)
	if err != nil {
		t.Fatalf("new Keycloak endpoint: %v", err)
	}
	account := scenarios.ProviderAccount{
		ScenarioID:     "synthetic-provideraccounts-0001",
		MembershipID:   "synthetic-membership-0001",
		Email:          "user-0001@synthetic.invalid",
		OrganizationID: "AUDITEE-A",
		Role:           "auditee",
		Enabled:        true,
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	ctx := context.Background()
	account, err = keycloak.EnsureProviderAccount(ctx, account)
	if err != nil {
		t.Fatalf("ensure provider account: %v", err)
	}
	invitations, err := scenarios.NewMailpitInvitationEndpoint(
		scenarios.MailpitInvitationEndpointConfig{
			Keycloak:   keycloak,
			BaseURL:    fixture.server.URL + "/mailpit",
			HTTPClient: &http.Client{Timeout: 2 * time.Second},
		},
	)
	if err != nil {
		t.Fatalf("new Mailpit invitation endpoint: %v", err)
	}
	if err := invitations.Preflight(ctx); err != nil {
		t.Fatalf("preflight Mailpit: %v", err)
	}
	delivery := scenarios.InvitationDelivery{
		InvitationID: "synthetic-invitation-0001",
		DeliveryID:   "synthetic-delivery-0001",
		SubjectID:    account.SubjectID,
		Email:        account.Email,
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	if err := invitations.EnsureInvitationDelivery(ctx, delivery); err != nil {
		t.Fatalf("ensure invitation delivery: %v", err)
	}
	if err := invitations.EnsureInvitationDelivery(ctx, delivery); err != nil {
		t.Fatalf("replay invitation delivery: %v", err)
	}
	if got := fixture.messageRecipients(); !slices.Equal(
		got,
		[]string{account.Email},
	) {
		t.Fatalf("Mailpit recipients after replay = %#v", got)
	}
	if err := invitations.ReconcileInvitationDeliveries(
		ctx,
		[]scenarios.InvitationDelivery{delivery},
	); err != nil {
		t.Fatalf("reconcile invitation delivery: %v", err)
	}
	fixture.setMessageRecipient("wrong@synthetic.invalid")
	if err := invitations.ReconcileInvitationDeliveries(
		ctx,
		[]scenarios.InvitationDelivery{delivery},
	); err == nil {
		t.Fatalf("Mailpit recipient drift was accepted")
	}
}

type scenarioKeycloakFixture struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	users    map[string]scenarioKeycloakUser
	roles    map[string][]string
	messages []scenarioMailpitMessage
	nextUser int
}

type scenarioKeycloakUser struct {
	ID              string              `json:"id"`
	Username        string              `json:"username"`
	Email           string              `json:"email"`
	FirstName       string              `json:"firstName"`
	LastName        string              `json:"lastName"`
	Enabled         bool                `json:"enabled"`
	EmailVerified   bool                `json:"emailVerified"`
	RequiredActions []string            `json:"requiredActions"`
	Attributes      map[string][]string `json:"attributes"`
}

type scenarioKeycloakRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type scenarioMailpitMessage struct {
	ID string                   `json:"ID"`
	To []scenarioMailpitAddress `json:"To"`
}

type scenarioMailpitAddress struct {
	Address string `json:"Address"`
}

func newScenarioKeycloakFixture(t *testing.T) *scenarioKeycloakFixture {
	t.Helper()
	fixture := &scenarioKeycloakFixture{
		t:     t,
		users: make(map[string]scenarioKeycloakUser),
		roles: make(map[string][]string),
	}
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.handle))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (fixture *scenarioKeycloakFixture) handle(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.URL.Path == "/mailpit/api/v1/messages" {
		if request.Method != http.MethodGet {
			http.Error(writer, "method", http.StatusMethodNotAllowed)
			return
		}
		fixture.writeMessages(writer)
		return
	}
	if request.URL.Path ==
		"/identity/realms/task7-realm/protocol/openid-connect/token" {
		if request.Method != http.MethodPost {
			http.Error(writer, "method", http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"access_token":"task7-token"}`))
		return
	}
	if request.Header.Get("Authorization") != "Bearer task7-token" {
		http.Error(writer, "authorization", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(
		request.URL.Path,
		"/identity/admin/realms/task7-realm/",
	)
	segments := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case len(segments) == 1 && segments[0] == "users" &&
		request.Method == http.MethodGet:
		fixture.writeUsers(writer, request)
	case len(segments) == 1 && segments[0] == "users" &&
		request.Method == http.MethodPost:
		fixture.createUser(writer, request)
	case len(segments) == 2 && segments[0] == "users" &&
		request.Method == http.MethodGet:
		fixture.writeUser(writer, segments[1])
	case len(segments) == 4 && segments[0] == "users" &&
		segments[2] == "role-mappings" && segments[3] == "realm" &&
		request.Method == http.MethodGet:
		fixture.writeUserRoles(writer, segments[1])
	case len(segments) == 4 && segments[0] == "users" &&
		segments[2] == "role-mappings" && segments[3] == "realm" &&
		request.Method == http.MethodPost:
		fixture.mapUserRoles(writer, request, segments[1])
	case len(segments) == 3 && segments[0] == "users" &&
		segments[2] == "execute-actions-email" &&
		request.Method == http.MethodPut:
		fixture.issueActionsEmail(writer, request, segments[1])
	case len(segments) == 2 && segments[0] == "roles" &&
		request.Method == http.MethodGet:
		fixture.writeRole(writer, segments[1])
	default:
		http.Error(writer, "not found", http.StatusNotFound)
	}
}

func (fixture *scenarioKeycloakFixture) writeMessages(
	writer http.ResponseWriter,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	writeScenarioJSON(
		fixture.t,
		writer,
		map[string]any{"messages": fixture.messages},
		http.StatusOK,
	)
}

func (fixture *scenarioKeycloakFixture) issueActionsEmail(
	writer http.ResponseWriter,
	request *http.Request,
	subjectID string,
) {
	var actions []string
	if err := json.NewDecoder(request.Body).Decode(&actions); err != nil ||
		!sameScenarioStrings(
			actions,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		) ||
		request.URL.Query().Get("lifespan") != "86400" {
		http.Error(writer, "invalid actions", http.StatusBadRequest)
		return
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	user, ok := fixture.users[subjectID]
	if !ok {
		http.Error(writer, "not found", http.StatusNotFound)
		return
	}
	fixture.messages = append(
		fixture.messages,
		scenarioMailpitMessage{
			ID: "message-" + subjectID,
			To: []scenarioMailpitAddress{{Address: user.Email}},
		},
	)
	writer.WriteHeader(http.StatusNoContent)
}

func (fixture *scenarioKeycloakFixture) writeUsers(
	writer http.ResponseWriter,
	request *http.Request,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	users := make([]scenarioKeycloakUser, 0, len(fixture.users))
	for _, user := range fixture.users {
		email := request.URL.Query().Get("email")
		if email != "" && user.Email != email {
			continue
		}
		users = append(users, user)
	}
	sort.Slice(users, func(left, right int) bool {
		return users[left].ID < users[right].ID
	})
	writeScenarioJSON(fixture.t, writer, users, http.StatusOK)
}

func (fixture *scenarioKeycloakFixture) createUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var user scenarioKeycloakUser
	if err := json.NewDecoder(request.Body).Decode(&user); err != nil {
		http.Error(writer, "decode", http.StatusBadRequest)
		return
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if user.ID != "" {
		http.Error(writer, "provider subject is server-assigned", http.StatusBadRequest)
		return
	}
	fixture.nextUser++
	user.ID = fmt.Sprintf("provider-subject-%03d", fixture.nextUser)
	if _, exists := fixture.users[user.ID]; exists {
		http.Error(writer, "conflict", http.StatusConflict)
		return
	}
	fixture.users[user.ID] = user
	writer.Header().Set(
		"Location",
		fixture.server.URL+"/identity/admin/realms/task7-realm/users/"+user.ID,
	)
	writer.WriteHeader(http.StatusCreated)
}

func (fixture *scenarioKeycloakFixture) writeUser(
	writer http.ResponseWriter,
	subjectID string,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	user, ok := fixture.users[subjectID]
	if !ok {
		http.Error(writer, "not found", http.StatusNotFound)
		return
	}
	writeScenarioJSON(fixture.t, writer, user, http.StatusOK)
}

func (fixture *scenarioKeycloakFixture) writeUserRoles(
	writer http.ResponseWriter,
	subjectID string,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	roles := make([]scenarioKeycloakRole, 0, len(fixture.roles[subjectID]))
	for _, role := range fixture.roles[subjectID] {
		roles = append(roles, scenarioKeycloakRole{
			ID:   "role-" + role,
			Name: role,
		})
	}
	writeScenarioJSON(fixture.t, writer, roles, http.StatusOK)
}

func (fixture *scenarioKeycloakFixture) mapUserRoles(
	writer http.ResponseWriter,
	request *http.Request,
	subjectID string,
) {
	var roles []scenarioKeycloakRole
	if err := json.NewDecoder(request.Body).Decode(&roles); err != nil {
		http.Error(writer, "decode", http.StatusBadRequest)
		return
	}
	names := make([]string, len(roles))
	for index, role := range roles {
		names[index] = role.Name
	}
	slices.Sort(names)
	fixture.mu.Lock()
	fixture.roles[subjectID] = names
	fixture.mu.Unlock()
	writer.WriteHeader(http.StatusNoContent)
}

func (fixture *scenarioKeycloakFixture) writeRole(
	writer http.ResponseWriter,
	role string,
) {
	writeScenarioJSON(
		fixture.t,
		writer,
		scenarioKeycloakRole{ID: "role-" + role, Name: role},
		http.StatusOK,
	)
}

func (fixture *scenarioKeycloakFixture) account(
	subjectID string,
) (scenarioKeycloakUser, []string) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	user := fixture.users[subjectID]
	user.RequiredActions = append([]string(nil), user.RequiredActions...)
	user.Attributes = cloneScenarioAttributes(user.Attributes)
	roles := append([]string(nil), fixture.roles[subjectID]...)
	return user, roles
}

func (fixture *scenarioKeycloakFixture) setOrganization(
	subjectID,
	organizationID string,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	user := fixture.users[subjectID]
	user.Attributes = cloneScenarioAttributes(user.Attributes)
	user.Attributes["organization_id"] = []string{organizationID}
	fixture.users[subjectID] = user
}

func (fixture *scenarioKeycloakFixture) messageRecipients() []string {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	var output []string
	for _, message := range fixture.messages {
		for _, recipient := range message.To {
			output = append(output, recipient.Address)
		}
	}
	slices.Sort(output)
	return output
}

func (fixture *scenarioKeycloakFixture) setMessageRecipient(
	email string,
) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if len(fixture.messages) == 0 {
		return
	}
	fixture.messages[0].To = []scenarioMailpitAddress{{Address: email}}
}

func cloneScenarioAttributes(
	source map[string][]string,
) map[string][]string {
	output := make(map[string][]string, len(source))
	for key, values := range source {
		output[key] = append([]string(nil), values...)
	}
	return output
}

func sameScenarioStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func writeScenarioJSON(
	t *testing.T,
	writer http.ResponseWriter,
	value any,
	status int,
) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Errorf("encode fixture response: %v", err)
	}
}
