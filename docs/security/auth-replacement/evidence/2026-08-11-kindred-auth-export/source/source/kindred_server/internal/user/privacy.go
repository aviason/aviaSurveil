package user

import (
	"reflect"
	"strings"
)

// Privacy policies define what data is visible in different contexts
type Policy string

const (
	// PolicyPublic - visible to everyone
	PolicyPublic Policy = "public"
	// PolicyAuthenticated - visible to authenticated users only
	PolicyAuthenticated Policy = "authenticated"
	// PolicyMatchedUsers - visible to matched/connected users
	PolicyMatchedUsers Policy = "matched_users"
	// PolicyOwnerOnly - visible to data owner only
	PolicyOwnerOnly Policy = "owner_only"
	// PolicyAdmin - visible to admin users only
	PolicyAdmin Policy = "admin"
)

// FieldPolicy defines privacy policy for a specific field
type FieldPolicy struct {
	FieldName string
	Policy    Policy
}

// jsonFieldName returns the JSON object key from a struct tag,
// stripping options like ",omitempty".
func jsonFieldName(tag string) string {
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		return tag[:idx]
	}
	return tag
}

// GetFieldPolicies returns the privacy policy for each field in User struct
func GetFieldPolicies() []FieldPolicy {
	return []FieldPolicy{
		{FieldName: "ID", Policy: PolicyPublic},
		{FieldName: "Email", Policy: PolicyOwnerOnly},
		{FieldName: "DisplayName", Policy: PolicyPublic},
		{FieldName: "Phone", Policy: PolicyMatchedUsers},
		{FieldName: "City", Policy: PolicyPublic},
		{FieldName: "PasswordHash", Policy: PolicyOwnerOnly},
		{FieldName: "CreatedAt", Policy: PolicyPublic},
		{FieldName: "UpdatedAt", Policy: PolicyOwnerOnly},
	}
}

// GetFieldPolicy returns the privacy policy for a specific field
func GetFieldPolicy(fieldName string) Policy {
	for _, fp := range GetFieldPolicies() {
		if fp.FieldName == fieldName {
			return fp.Policy
		}
	}
	return PolicyOwnerOnly // default to most restrictive
}

// policyRank orders policies from least to most privileged. A viewing
// context can see a field only if the context's rank is >= the field's
// declared rank.
func policyRank(p Policy) int {
	switch p {
	case PolicyPublic:
		return 0
	case PolicyAuthenticated:
		return 1
	case PolicyMatchedUsers:
		return 2
	case PolicyOwnerOnly:
		return 3
	case PolicyAdmin:
		return 4
	default:
		return -1
	}
}

// IsFieldVisibleUnderPolicy checks if fieldName is visible when the request
// is evaluated under the given viewing-context policy. The field is visible
// when the context grants at least as much access as the field requires; for
// owner-scoped fields, the requesting user must also be the owner.
func IsFieldVisibleUnderPolicy(fieldName string, policy Policy, requestingUserID, ownerUserID string) bool {
	fieldPolicy := GetFieldPolicy(fieldName)
	ctxRank, fieldRank := policyRank(policy), policyRank(fieldPolicy)
	if ctxRank < 0 || fieldRank < 0 || ctxRank < fieldRank {
		return false
	}

	switch fieldPolicy {
	case PolicyPublic:
		return true
	case PolicyAuthenticated:
		return requestingUserID != ""
	case PolicyMatchedUsers:
		return requestingUserID != ""
	case PolicyOwnerOnly:
		return requestingUserID == ownerUserID
	case PolicyAdmin:
		return false
	default:
		return false
	}
}

// FilterUserByPolicy returns a filtered version of User based on policy
func FilterUserByPolicy(u User, policy Policy, requestingUserID string) interface{} {
	var result map[string]interface{} = make(map[string]interface{})

	userVal := reflect.ValueOf(u)
	userType := reflect.TypeOf(u)

	for i := 0; i < userType.NumField(); i++ {
		field := userType.Field(i)
		fieldValue := userVal.Field(i).Interface()
		fieldName := field.Name

		if IsFieldVisibleUnderPolicy(fieldName, policy, requestingUserID, u.ID) {
			key := jsonFieldName(field.Tag.Get("json"))
			if key != "" && key != "-" {
				result[key] = fieldValue
			} else {
				result[fieldName] = fieldValue
			}
		}
	}

	return result
}

// FilterUserResponseByPolicy returns appropriate response based on policy
func FilterUserResponseByPolicy(u User, policy Policy, requestingUserID string) interface{} {
	policies := GetFieldPolicies()
	result := make(map[string]interface{})

	for _, fp := range policies {
		if fp.FieldName == "PasswordHash" {
			// Never include password hash in response
			continue
		}

		if IsFieldVisibleUnderPolicy(fp.FieldName, policy, requestingUserID, u.ID) {
			val := reflect.ValueOf(u).FieldByName(fp.FieldName).Interface()

			field, _ := reflect.TypeOf(u).FieldByName(fp.FieldName)
			key := jsonFieldName(field.Tag.Get("json"))
			if key == "" || key == "-" {
				key = fp.FieldName
			}

			result[key] = val
		}
	}

	return result
}

// GetPolicyForContext determines which privacy policy to apply based on context
func GetPolicyForContext(contextType string) Policy {
	switch contextType {
	case "admin_dashboard":
		return PolicyAdmin
	case "matching":
		return PolicyMatchedUsers
	case "search":
		return PolicyAuthenticated
	case "public_profile":
		return PolicyPublic
	case "owner_profile":
		return PolicyOwnerOnly
	default:
		return PolicyPublic
	}
}
