package user_test

import (
	"testing"
	"time"

	"kindred_server/internal/user"
)

func sampleUser() user.User {
	return user.User{
		ID:           "owner-1",
		Email:        "o@example.com",
		DisplayName:  "Owner",
		Phone:        "+555",
		City:         "NYC",
		PasswordHash: "secret-hash",
		CreatedAt:    time.Unix(1, 0).UTC(),
		UpdatedAt:    time.Unix(2, 0).UTC(),
	}
}

func TestGetFieldPolicy_KnownAndUnknown(t *testing.T) {
	if p := user.GetFieldPolicy("Email"); p != user.PolicyOwnerOnly {
		t.Errorf("Email policy = %v, want OwnerOnly", p)
	}
	if p := user.GetFieldPolicy("DisplayName"); p != user.PolicyPublic {
		t.Errorf("DisplayName policy = %v, want Public", p)
	}
	// Unknown fields default to most restrictive.
	if p := user.GetFieldPolicy("Nope"); p != user.PolicyOwnerOnly {
		t.Errorf("unknown policy = %v, want OwnerOnly", p)
	}
}

// IsFieldVisibleUnderPolicy gates a field on BOTH the field's declared
// policy and the viewing-context policy: the context must grant at least as
// much access as the field requires, and owner-scoped fields additionally
// require requesting == owner.
func TestIsFieldVisibleUnderPolicy(t *testing.T) {
	cases := []struct {
		field, requesting, owner string
		policy                   user.Policy
		want                     bool
	}{
		// DisplayName is Public — visible under any context.
		{"DisplayName", "anyone", "owner-1", user.PolicyPublic, true},
		{"DisplayName", "", "owner-1", user.PolicyAdmin, true},
		// Email is OwnerOnly — needs OwnerOnly+ context AND requester == owner.
		{"Email", "owner-1", "owner-1", user.PolicyOwnerOnly, true},
		{"Email", "owner-1", "owner-1", user.PolicyPublic, false}, // context too low
		{"Email", "stranger", "owner-1", user.PolicyOwnerOnly, false},
		// PasswordHash is OwnerOnly.
		{"PasswordHash", "stranger", "owner-1", user.PolicyOwnerOnly, false},
		// Unknown field defaults to OwnerOnly.
		{"Mystery", "stranger", "owner-1", user.PolicyOwnerOnly, false},
		{"Mystery", "owner-1", "owner-1", user.PolicyOwnerOnly, true},
		{"Mystery", "owner-1", "owner-1", user.PolicyPublic, false}, // context too low
	}
	for _, c := range cases {
		got := user.IsFieldVisibleUnderPolicy(c.field, c.policy, c.requesting, c.owner)
		if got != c.want {
			t.Errorf("IsFieldVisibleUnderPolicy(%s, %s, req=%s) = %v, want %v",
				c.field, c.policy, c.requesting, got, c.want)
		}
	}
}

func TestFilterUserResponseByPolicy_PublicMasksOwnerOnly(t *testing.T) {
	u := sampleUser()
	out := user.FilterUserResponseByPolicy(u, user.PolicyPublic, "stranger").(map[string]interface{})

	// Public fields should be present.
	if _, ok := out["id"]; !ok {
		t.Error("id missing from public view")
	}
	if _, ok := out["displayName"]; !ok {
		t.Error("displayName missing from public view")
	}
	// OwnerOnly fields must be hidden when requester != owner.
	if _, ok := out["email"]; ok {
		t.Error("email should be hidden under public policy")
	}
	if _, ok := out["updatedAt"]; ok {
		t.Error("updatedAt should be hidden under public policy")
	}
	// PasswordHash is never included (json:"-" + explicit skip).
	for _, k := range []string{"PasswordHash", "passwordHash", "-"} {
		if _, ok := out[k]; ok {
			t.Errorf("password hash leaked under key %q", k)
		}
	}
}

func TestFilterUserResponseByPolicy_OwnerSeesAll(t *testing.T) {
	u := sampleUser()
	out := user.FilterUserResponseByPolicy(u, user.PolicyOwnerOnly, u.ID).(map[string]interface{})
	for _, k := range []string{"id", "email", "displayName", "phone", "city", "createdAt", "updatedAt"} {
		if _, ok := out[k]; !ok {
			t.Errorf("owner view missing %q", k)
		}
	}
	// Even for owner, password hash must never appear.
	if _, ok := out["passwordHash"]; ok {
		t.Error("password hash leaked to owner view")
	}
}

func TestGetPolicyForContext(t *testing.T) {
	cases := map[string]user.Policy{
		"admin_dashboard": user.PolicyAdmin,
		"matching":        user.PolicyMatchedUsers,
		"search":          user.PolicyAuthenticated,
		"public_profile":  user.PolicyPublic,
		"owner_profile":   user.PolicyOwnerOnly,
		"unknown":         user.PolicyPublic,
	}
	for ctx, want := range cases {
		if got := user.GetPolicyForContext(ctx); got != want {
			t.Errorf("GetPolicyForContext(%q) = %v, want %v", ctx, got, want)
		}
	}
}
