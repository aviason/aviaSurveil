package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestKeycloakAdminClientProvisionsPasswordlessUserWithoutRequiredTOTP(t *testing.T) {
	t.Parallel()
	var created map[string]any
	var mappedRoles []map[string]any
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users":
			if request.URL.Query().Get("email") != "new.user@example.test" ||
				request.URL.Query().Get("exact") != "true" {
				t.Errorf("duplicate-email query = %q", request.URL.RawQuery)
			}
			writeKeycloakJSON(writer, http.StatusOK, []any{})
		case request.Method == http.MethodPost &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users":
			if err := json.NewDecoder(request.Body).Decode(&created); err != nil {
				t.Errorf("decode created user: %v", err)
			}
			writer.Header().Set(
				"Location",
				server.URL+"/identity/admin/realms/aviasurveil360/users/provider-subject-001",
			)
			writer.WriteHeader(http.StatusCreated)
		case request.Method == http.MethodGet &&
			strings.HasPrefix(
				request.URL.Path,
				"/identity/admin/realms/aviasurveil360/roles/",
			):
			role := request.URL.Path[strings.LastIndex(request.URL.Path, "/")+1:]
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"id": "role-" + role, "name": role,
			})
		case request.Method == http.MethodPost &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			if err := json.NewDecoder(request.Body).Decode(&mappedRoles); err != nil {
				t.Errorf("decode role mapping: %v", err)
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	subjectID, err := client.ProvisionUser(
		context.Background(),
		identity.KeycloakUser{
			Email:          "new.user@example.test",
			FirstName:      "New",
			LastName:       "User",
			OrganizationID: "CAA",
			Roles: []identity.Role{
				identity.RoleInspector,
			},
		},
	)
	if err != nil {
		t.Fatalf("provision Keycloak user: %v", err)
	}
	if subjectID != "provider-subject-001" {
		t.Fatalf("provider subject = %q", subjectID)
	}
	if created["username"] != "new.user@example.test" ||
		created["email"] != "new.user@example.test" ||
		created["firstName"] != "New" ||
		created["lastName"] != "User" ||
		created["enabled"] != true ||
		created["emailVerified"] != false {
		t.Fatalf("created representation = %#v", created)
	}
	if _, exists := created["credentials"]; exists {
		t.Fatalf("created representation contains credentials: %#v", created)
	}
	attributes, ok := created["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("created attributes = %#v", created["attributes"])
	}
	if values, ok := attributes["organization_id"].([]any); !ok ||
		len(values) != 1 ||
		values[0] != "CAA" {
		t.Fatalf("organization attributes = %#v", attributes)
	}
	requiredActions, ok := created["requiredActions"].([]any)
	if !ok ||
		!slices.Equal(requiredActions, []any{"UPDATE_PASSWORD", "VERIFY_EMAIL"}) ||
		slices.Contains(requiredActions, any("CONFIGURE_TOTP")) {
		t.Fatalf("required actions = %#v", created["requiredActions"])
	}
	if len(mappedRoles) != 1 ||
		mappedRoles[0]["name"] != string(identity.RoleInspector) {
		t.Fatalf("mapped roles = %#v", mappedRoles)
	}
}

func TestKeycloakAdminClientIssuesExecuteActionsResetsMFAAndForcesLogout(t *testing.T) {
	t.Parallel()
	var transcript []string
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodPut &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/execute-actions-email":
			if request.URL.Query().Get("lifespan") != "86400" {
				t.Errorf("execute-actions lifespan = %q", request.URL.RawQuery)
			}
			var actions []string
			if err := json.NewDecoder(request.Body).Decode(&actions); err != nil {
				t.Errorf("decode execute actions: %v", err)
			}
			if !slices.Equal(actions, []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}) {
				t.Errorf("execute actions = %#v", actions)
			}
			transcript = append(transcript, "execute-actions")
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/credentials":
			writeKeycloakJSON(writer, http.StatusOK, []map[string]string{
				{"id": "otp-credential-001", "type": "otp"},
				{"id": "password-credential-001", "type": "password"},
			})
		case request.Method == http.MethodDelete &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/credentials/otp-credential-001":
			transcript = append(transcript, "delete-otp")
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/logout":
			transcript = append(transcript, "logout")
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	if err := client.IssueExecuteActionsEmail(
		context.Background(),
		"provider-subject-001",
		[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		24*60*60,
	); err != nil {
		t.Fatalf("issue execute actions: %v", err)
	}
	if err := client.ResetUserMFA(
		context.Background(),
		"provider-subject-001",
	); err != nil {
		t.Fatalf("reset user MFA: %v", err)
	}
	if err := client.ForceUserLogout(
		context.Background(),
		"provider-subject-001",
	); err != nil {
		t.Fatalf("force user logout: %v", err)
	}
	if !slices.Equal(
		transcript,
		[]string{"execute-actions", "delete-otp", "logout"},
	) {
		t.Fatalf("identity lifecycle transcript = %#v", transcript)
	}
}

func TestKeycloakAdminClientClassifiesProviderHTTPFailures(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name          string
		status        int
		expectedClass identity.KeycloakFailureClass
	}{
		{"rate-limit", http.StatusTooManyRequests, identity.KeycloakFailureRetryable},
		{"server-error", http.StatusServiceUnavailable, identity.KeycloakFailureRetryable},
		{"unauthorized", http.StatusUnauthorized, identity.KeycloakFailureManualReview},
		{"forbidden", http.StatusForbidden, identity.KeycloakFailureManualReview},
		{"bad-request", http.StatusBadRequest, identity.KeycloakFailurePermanent},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				assertKeycloakBearer(t, request)
				if request.URL.Path ==
					"/identity/realms/aviasurveil360/protocol/openid-connect/token" {
					assertKeycloakAdminTokenRequest(t, request)
					writeKeycloakJSON(writer, http.StatusOK, map[string]any{
						"access_token": "admin-access-token",
						"expires_in":   60,
					})
					return
				}
				http.Error(writer, "classified provider failure", testCase.status)
			}))
			t.Cleanup(server.Close)
			err := newKeycloakAdminTestClient(t, server.URL).ForceUserLogout(
				context.Background(),
				"provider-subject-001",
			)
			if err == nil {
				t.Fatal("provider HTTP failure was accepted")
			}
			if actual := identity.ClassifyKeycloakError(err); actual !=
				testCase.expectedClass {
				t.Fatalf(
					"provider HTTP %d class = %q, want %q: %v",
					testCase.status,
					actual,
					testCase.expectedClass,
					err,
				)
			}
		})
	}
}

func TestTask4KeycloakAdminClientObservesExactUserAuthorityAndLockout(
	t *testing.T,
) {
	t.Parallel()
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path ==
			"/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path ==
				"/identity/admin/realms/aviasurveil360/users/provider-subject-001":
			providerCalls++
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"id":              "provider-subject-001",
				"enabled":         true,
				"totp":            true,
				"requiredActions": []string{},
				"attributes": map[string][]string{
					"organization_id": {"CAA"},
				},
			})
		case request.Method == http.MethodGet &&
			request.URL.Path ==
				"/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			providerCalls++
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{"id": "role-default", "name": "offline_access"},
				{"id": "role-inspector", "name": "inspector"},
			})
		case request.Method == http.MethodGet &&
			request.URL.Path ==
				"/identity/admin/realms/aviasurveil360/attack-detection/brute-force/users/provider-subject-001":
			providerCalls++
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"disabled":    true,
				"numFailures": 5,
			})
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	observation, err := newKeycloakAdminTestClient(
		t,
		server.URL,
	).ObserveUserAuthority(
		context.Background(),
		"provider-subject-001",
	)
	if err != nil {
		t.Fatalf("observe Keycloak user authority: %v", err)
	}
	if providerCalls != 3 ||
		observation.SubjectID != "provider-subject-001" ||
		!observation.Enabled ||
		!observation.Locked ||
		!observation.MFAEnrolled ||
		observation.OrganizationID != "CAA" ||
		!slices.Equal(
			observation.Roles,
			[]identity.Role{identity.RoleInspector},
		) ||
		len(observation.RequiredActions) != 0 {
		t.Fatalf(
			"provider calls = %d, observation = %#v",
			providerCalls,
			observation,
		)
	}
}

func TestKeycloakAdminClientRejectsCrossAuthorityRoleOrganizationMappings(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			requestCount++
			http.Error(writer, "must not call provider", http.StatusInternalServerError)
		},
	))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	if _, err := client.ProvisionUser(
		context.Background(),
		identity.KeycloakUser{
			Email:          "mixed-authority@example.test",
			FirstName:      "Mixed",
			LastName:       "Authority",
			OrganizationID: "auditee-org-001",
			Roles: []identity.Role{
				identity.RoleAuditee,
				identity.RoleInspector,
			},
		},
	); err == nil {
		t.Fatal("mixed Auditee and CAA roles were accepted")
	}
	if err := client.UpdateUserAuthority(
		context.Background(),
		"provider-subject-001",
		"auditee-org-001",
		[]identity.Role{identity.RoleInspector},
	); err == nil {
		t.Fatal("CAA role outside the exact CAA organization was accepted")
	}
	if requestCount != 0 {
		t.Fatalf("invalid authority mapping made %d provider requests", requestCount)
	}
}

func TestKeycloakAdminClientRejectsDuplicateEmailBeforeCreate(t *testing.T) {
	t.Parallel()
	createCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users":
			writeKeycloakJSON(writer, http.StatusOK, []map[string]string{
				{"id": "existing-subject"},
			})
		case request.Method == http.MethodPost:
			createCalled = true
			http.Error(writer, "must not create", http.StatusInternalServerError)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	_, err := client.ProvisionUser(context.Background(), identity.KeycloakUser{
		Email:          "existing@example.test",
		FirstName:      "Existing",
		LastName:       "User",
		OrganizationID: "CAA",
		Roles:          []identity.Role{identity.RoleInspector},
	})
	if !errors.Is(err, identity.ErrKeycloakDuplicateEmail) {
		t.Fatalf("duplicate-email error = %v", err)
	}
	if createCalled {
		t.Fatal("duplicate email reached create endpoint")
	}
}

func TestKeycloakAdminClientReconcilesOnlyAnExactExistingUser(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users":
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{
					"id":        "existing-subject",
					"username":  "existing@example.test",
					"email":     "existing@example.test",
					"firstName": "Existing",
					"lastName":  "User",
					"enabled":   true,
					"attributes": map[string][]string{
						"organization_id": {"airline-xyz"},
					},
				},
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/existing-subject/role-mappings/realm":
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{"id": "role-auditee", "name": "auditee"},
				{"id": "role-offline", "name": "offline_access"},
			})
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	subjectID, matched, err := client.ReconcileProvisionedUser(
		context.Background(),
		identity.KeycloakUser{
			Email:          "existing@example.test",
			FirstName:      "Existing",
			LastName:       "User",
			OrganizationID: "airline-xyz",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	)
	if err != nil || !matched || subjectID != "existing-subject" {
		t.Fatalf(
			"exact reconciliation = subject %q matched %t err %v",
			subjectID,
			matched,
			err,
		)
	}
	_, matched, err = client.ReconcileProvisionedUser(
		context.Background(),
		identity.KeycloakUser{
			Email:          "existing@example.test",
			FirstName:      "Existing",
			LastName:       "User",
			OrganizationID: "other-airline",
			Roles:          []identity.Role{identity.RoleAuditee},
		},
	)
	if err != nil || matched {
		t.Fatalf("mismatched reconciliation = matched %t err %v", matched, err)
	}
}

func TestKeycloakAdminClientDisablesUserAndRevokesProviderSessions(t *testing.T) {
	t.Parallel()
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodPut &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001":
			var update map[string]any
			if err := json.NewDecoder(request.Body).Decode(&update); err != nil {
				t.Errorf("decode disable update: %v", err)
			}
			if update["enabled"] != false {
				t.Errorf("disable representation = %#v", update)
			}
			requests = append(requests, "disable")
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPost &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/logout":
			requests = append(requests, "logout")
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	if err := client.DisableUser(
		context.Background(),
		"provider-subject-001",
	); err != nil {
		t.Fatalf("disable Keycloak user: %v", err)
	}
	if !slices.Equal(requests, []string{"disable", "logout"}) {
		t.Fatalf("disable transcript = %#v", requests)
	}
}

func TestKeycloakAdminClientUpdatesOrganizationRolesAndReactivatesUser(t *testing.T) {
	t.Parallel()
	var transcript []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001":
			transcript = append(transcript, "read-user")
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"id":            "provider-subject-001",
				"username":      "existing.user@example.test",
				"email":         "existing.user@example.test",
				"firstName":     "Existing",
				"lastName":      "User",
				"enabled":       true,
				"emailVerified": true,
				"totp":          true,
				"requiredActions": []string{
					"VERIFY_EMAIL",
				},
				"attributes": map[string][]string{
					"organization_id": {"CAA"},
					"retained_fact":   {"preserve-me"},
				},
			})
		case request.Method == http.MethodPut &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001":
			var update map[string]any
			if err := json.NewDecoder(request.Body).Decode(&update); err != nil {
				t.Errorf("decode user update: %v", err)
			}
			if rawAttributes, ok := update["attributes"]; ok {
				for field, expected := range map[string]any{
					"username":      "existing.user@example.test",
					"email":         "existing.user@example.test",
					"firstName":     "Existing",
					"lastName":      "User",
					"emailVerified": true,
				} {
					if update[field] != expected {
						t.Errorf("organization update lost %s: %#v", field, update)
					}
				}
				attributes, ok := rawAttributes.(map[string]any)
				if !ok {
					t.Errorf("organization update = %#v", update)
				} else if values, ok := attributes["organization_id"].([]any); !ok ||
					len(values) != 1 ||
					values[0] != "auditee-org-002" {
					t.Errorf("organization update = %#v", update)
				} else if retained, ok := attributes["retained_fact"].([]any); !ok ||
					len(retained) != 1 ||
					retained[0] != "preserve-me" {
					t.Errorf("organization update lost retained attributes: %#v", update)
				}
				transcript = append(transcript, "organization")
			} else if enabled, ok := update["enabled"]; ok {
				if enabled != true {
					t.Errorf("reactivation update = %#v", update)
				}
				transcript = append(transcript, "enable")
			} else {
				t.Errorf("unexpected user update = %#v", update)
			}
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{"id": "role-inspector", "name": "inspector"},
				{"id": "role-offline-access", "name": "offline_access"},
			})
		case request.Method == http.MethodDelete &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			var removed []map[string]any
			if err := json.NewDecoder(request.Body).Decode(&removed); err != nil {
				t.Errorf("decode removed roles: %v", err)
			}
			if len(removed) != 1 || removed[0]["name"] != "inspector" {
				t.Errorf("removed roles = %#v", removed)
			}
			transcript = append(transcript, "remove-approved")
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/roles/auditee":
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"id": "role-auditee", "name": "auditee",
			})
		case request.Method == http.MethodPost &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			var added []map[string]any
			if err := json.NewDecoder(request.Body).Decode(&added); err != nil {
				t.Errorf("decode added roles: %v", err)
			}
			if len(added) != 1 || added[0]["name"] != "auditee" {
				t.Errorf("added roles = %#v", added)
			}
			transcript = append(transcript, "add-approved")
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	if err := client.UpdateUserAuthority(
		context.Background(),
		"provider-subject-001",
		"auditee-org-002",
		[]identity.Role{identity.RoleAuditee},
	); err != nil {
		t.Fatalf("update Keycloak authority: %v", err)
	}
	if err := client.EnableUser(
		context.Background(),
		"provider-subject-001",
	); err != nil {
		t.Fatalf("enable Keycloak user: %v", err)
	}
	if !slices.Equal(
		transcript,
		[]string{"read-user", "organization", "remove-approved", "add-approved", "enable"},
	) {
		t.Fatalf("authority transcript = %#v", transcript)
	}
}

func TestKeycloakAdminClientListsBoundedDirectoryWithProviderRoles(t *testing.T) {
	t.Parallel()
	providerCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertKeycloakBearer(t, request)
		switch {
		case request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token":
			assertKeycloakAdminTokenRequest(t, request)
			writeKeycloakJSON(writer, http.StatusOK, map[string]any{
				"access_token": "admin-access-token",
				"expires_in":   60,
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users":
			providerCalls++
			if request.URL.Query().Get("first") != "0" ||
				request.URL.Query().Get("max") != "2" ||
				request.URL.Query().Get("search") != "David" ||
				request.URL.Query().Get("briefRepresentation") != "false" {
				t.Errorf("directory query = %q", request.URL.RawQuery)
			}
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{
					"id":              "provider-subject-001",
					"username":        "david.inspector@example.test",
					"email":           "david.inspector@example.test",
					"firstName":       "David",
					"lastName":        "Demir",
					"enabled":         true,
					"totp":            true,
					"requiredActions": []string{},
					"attributes": map[string][]string{
						"organization_id": {"CAA"},
					},
				},
				{
					"id":              "provider-subject-002",
					"username":        "david.auditee@example.test",
					"email":           "david.auditee@example.test",
					"firstName":       "David",
					"lastName":        "Air",
					"enabled":         false,
					"totp":            false,
					"requiredActions": []string{"CONFIGURE_TOTP", "VERIFY_EMAIL"},
					"attributes": map[string][]string{
						"organization_id": {"airline-xyz"},
					},
				},
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-001/role-mappings/realm":
			providerCalls++
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{"id": "role-offline", "name": "offline_access"},
				{"id": "role-lead", "name": "leadInspector"},
				{"id": "role-inspector", "name": "inspector"},
			})
		case request.Method == http.MethodGet &&
			request.URL.Path == "/identity/admin/realms/aviasurveil360/users/provider-subject-002/role-mappings/realm":
			providerCalls++
			writeKeycloakJSON(writer, http.StatusOK, []map[string]any{
				{"id": "role-auditee", "name": "auditee"},
			})
		default:
			http.Error(writer, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	client := newKeycloakAdminTestClient(t, server.URL)
	page, err := client.ListDirectory(
		context.Background(),
		identity.KeycloakDirectoryQuery{First: 0, Limit: 2, Search: "David"},
	)
	if err != nil {
		t.Fatalf("list Keycloak directory: %v", err)
	}
	if page.ProviderCalls != 3 || providerCalls != 3 {
		t.Fatalf("provider calls = page %d server %d", page.ProviderCalls, providerCalls)
	}
	if page.NextFirst != 2 || len(page.Users) != 2 {
		t.Fatalf("directory page = %#v", page)
	}
	first := page.Users[0]
	if first.SubjectID != "provider-subject-001" ||
		first.Email != "david.inspector@example.test" ||
		first.DisplayName != "David Demir" ||
		first.OrganizationID != "CAA" ||
		!first.Enabled ||
		!first.TOTPConfigured ||
		len(first.RequiredActions) != 0 ||
		!slices.Equal(first.Roles, []identity.Role{
			identity.RoleInspector,
			identity.RoleLeadInspector,
		}) {
		t.Fatalf("first directory user = %#v", first)
	}
	second := page.Users[1]
	if second.SubjectID != "provider-subject-002" ||
		second.OrganizationID != "airline-xyz" ||
		second.Enabled ||
		second.TOTPConfigured ||
		!slices.Equal(second.RequiredActions, []string{
			"CONFIGURE_TOTP",
			"VERIFY_EMAIL",
		}) ||
		!slices.Equal(second.Roles, []identity.Role{identity.RoleAuditee}) {
		t.Fatalf("second directory user = %#v", second)
	}
}

func newKeycloakAdminTestClient(
	t *testing.T,
	baseURL string,
) *identity.KeycloakAdminClient {
	t.Helper()
	client, err := identity.NewKeycloakAdminClient(identity.KeycloakAdminConfig{
		BaseURL:      baseURL + "/identity",
		Realm:        "aviasurveil360",
		ClientID:     "aviasurveil360-lifecycle",
		ClientSecret: "lifecycle-client-secret",
		HTTPClient:   http.DefaultClient,
	})
	if err != nil {
		t.Fatalf("new Keycloak admin client: %v", err)
	}
	return client
}

func assertKeycloakAdminTokenRequest(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodPost {
		t.Errorf("token method = %s", request.Method)
	}
	if err := request.ParseForm(); err != nil {
		t.Errorf("parse token request: %v", err)
		return
	}
	expected := url.Values{
		"client_id":     {"aviasurveil360-lifecycle"},
		"client_secret": {"lifecycle-client-secret"},
		"grant_type":    {"client_credentials"},
	}
	if request.Form.Encode() != expected.Encode() {
		t.Errorf("token form = %q", request.Form.Encode())
	}
}

func assertKeycloakBearer(t *testing.T, request *http.Request) {
	t.Helper()
	if request.URL.Path == "/identity/realms/aviasurveil360/protocol/openid-connect/token" {
		return
	}
	if request.Header.Get("Authorization") != "Bearer admin-access-token" {
		t.Errorf("authorization = %q", request.Header.Get("Authorization"))
	}
}

func writeKeycloakJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
