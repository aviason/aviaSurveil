// Package v3 validates the Task 3B locked AviaCore producer contract without
// importing AviaCore implementation code.
package v3

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type ValidationError struct {
	Code    string `json:"code"`
	Pointer string `json:"pointer"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidateEventJSON(raw []byte, authenticatedTenantID string) []ValidationError {
	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil {
		return []ValidationError{{Code: "SCHEMA_INVALID", Pointer: "/"}}
	}
	return ValidateEvent(event, authenticatedTenantID)
}

func ValidateEvent(event map[string]any, authenticatedTenantID string) []ValidationError {
	eventType, _ := event["event_type"].(string)
	schemaName, known := GeneratedEventSchemaByType[eventType]
	if !known {
		return []ValidationError{{Code: "UNKNOWN_EVENT_TYPE", Pointer: "/event_type"}}
	}
	schema, ok := decodedSchema(schemaName)
	if !ok {
		return []ValidationError{{Code: "SCHEMA_INVALID", Pointer: "/"}}
	}
	errors := validateSchema(schema, event, "/")
	if authenticatedTenantID != "" && event["tenant_id"] != authenticatedTenantID {
		errors = append(errors, ValidationError{Code: "TENANT_IDENTITY_MISMATCH", Pointer: "/tenant_id"})
	}
	occurred, occurredOK := parseTime(event["occurred_at"])
	emitted, emittedOK := parseTime(event["emitted_at"])
	effective, effectiveOK := parseTime(event["effective_at"])
	knownAt, knownOK := parseTime(event["known_at"])
	if (occurredOK && emittedOK && emitted.Before(occurred)) || (effectiveOK && knownOK && knownAt.Before(effective)) {
		errors = append(errors, ValidationError{Code: "TEMPORAL_ORDER_INVALID", Pointer: "/known_at"})
	}
	payload, payloadOK := event["payload"].(map[string]any)
	if !payloadOK {
		return append(errors, ValidationError{Code: "SCHEMA_INVALID", Pointer: "/payload"})
	}
	if encoded, err := json.Marshal(payload); err != nil || event["payload_sha256"] != sha256Hex(encoded) {
		errors = append(errors, ValidationError{Code: "PAYLOAD_SHA256_MISMATCH", Pointer: "/payload_sha256"})
	}
	if eventType == "correction.recorded" && payload["corrected_event_id"] == event["event_id"] {
		errors = append(errors, ValidationError{Code: "CORRECTION_REQUIRES_DISTINCT_EVENT_ID", Pointer: "/payload/corrected_event_id"})
	}
	if eventType == "event.superseded" && payload["superseded_event_id"] == payload["replacement_event_id"] {
		errors = append(errors, ValidationError{Code: "SUPERSESSION_REQUIRES_DISTINCT_EVENT_IDS", Pointer: "/payload/replacement_event_id"})
	}
	return errors
}

func decodedSchema(name string) (map[string]any, bool) {
	raw, ok := GeneratedSchemaDocuments[name]
	if !ok {
		return nil, false
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return nil, false
	}
	return schema, true
}

func validateSchema(schema map[string]any, value any, pointer string) []ValidationError {
	errors := make([]ValidationError, 0)
	if ref, ok := schema["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "./")
		if resolved, exists := decodedSchema(name); exists {
			return validateSchema(resolved, value, pointer)
		}
		return []ValidationError{{Code: "SCHEMA_INVALID", Pointer: pointer}}
	}
	if allOf, ok := schema["allOf"].([]any); ok {
		for _, raw := range allOf {
			if child, ok := raw.(map[string]any); ok {
				errors = append(errors, validateSchema(child, value, pointer)...)
			}
		}
	}
	if oneOf, ok := schema["oneOf"].([]any); ok {
		matched := 0
		for _, raw := range oneOf {
			if child, ok := raw.(map[string]any); ok && len(validateSchema(child, value, pointer)) == 0 {
				matched++
			}
		}
		if matched != 1 {
			errors = append(errors, ValidationError{Code: "SCHEMA_ONE_OF", Pointer: pointer})
		}
	}
	if !matchesType(schema["type"], value) {
		return append(errors, ValidationError{Code: "SCHEMA_TYPE", Pointer: pointer})
	}
	if expected, exists := schema["const"]; exists && !reflect.DeepEqual(expected, value) {
		errors = append(errors, ValidationError{Code: "SCHEMA_CONST", Pointer: pointer})
	}
	if choices, ok := schema["enum"].([]any); ok {
		matched := false
		for _, choice := range choices {
			if reflect.DeepEqual(choice, value) {
				matched = true
				break
			}
		}
		if !matched {
			errors = append(errors, ValidationError{Code: "SCHEMA_ENUM", Pointer: pointer})
		}
	}
	if object, ok := value.(map[string]any); ok {
		properties, _ := schema["properties"].(map[string]any)
		if required, ok := schema["required"].([]any); ok {
			for _, raw := range required {
				field, _ := raw.(string)
				if _, exists := object[field]; !exists {
					errors = append(errors, ValidationError{Code: "SCHEMA_REQUIRED", Pointer: childPointer(pointer, field)})
				}
			}
		}
		if schema["additionalProperties"] == false {
			for field := range object {
				if _, exists := properties[field]; !exists {
					errors = append(errors, ValidationError{Code: "SCHEMA_ADDITIONAL_PROPERTY", Pointer: childPointer(pointer, field)})
				}
			}
		}
		for field, raw := range properties {
			if child, ok := raw.(map[string]any); ok {
				if childValue, exists := object[field]; exists {
					errors = append(errors, validateSchema(child, childValue, childPointer(pointer, field))...)
				}
			}
		}
		checkSize(&errors, schema, len(object), pointer, "properties")
	}
	if array, ok := value.([]any); ok {
		checkSize(&errors, schema, len(array), pointer, "items")
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range array {
				errors = append(errors, validateSchema(itemSchema, item, childPointer(pointer, stringIndex(index)))...)
			}
		}
	}
	if text, ok := value.(string); ok {
		length := utf8.RuneCountInString(text)
		checkSize(&errors, schema, length, pointer, "length")
		if pattern, ok := schema["pattern"].(string); ok {
			if compiled, err := regexp.Compile(pattern); err != nil || !compiled.MatchString(text) {
				errors = append(errors, ValidationError{Code: "SCHEMA_PATTERN", Pointer: pointer})
			}
		}
		if format, ok := schema["format"].(string); ok && !validFormat(format, text) {
			errors = append(errors, ValidationError{Code: "SCHEMA_FORMAT", Pointer: pointer})
		}
	}
	if number, ok := value.(float64); ok {
		if minimum, exists := schema["minimum"].(float64); exists && number < minimum {
			errors = append(errors, ValidationError{Code: "SCHEMA_MINIMUM", Pointer: pointer})
		}
		if maximum, exists := schema["maximum"].(float64); exists && number > maximum {
			errors = append(errors, ValidationError{Code: "SCHEMA_MAXIMUM", Pointer: pointer})
		}
	}
	return errors
}

func matchesType(raw any, value any) bool {
	if raw == nil {
		return true
	}
	if types, ok := raw.([]any); ok {
		for _, candidate := range types {
			if matchesType(candidate, value) {
				return true
			}
		}
		return false
	}
	kind, _ := raw.(string)
	switch kind {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && math.Trunc(number) == number
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	default:
		return false
	}
}

func checkSize(errors *[]ValidationError, schema map[string]any, size int, pointer, kind string) {
	minimum, maximum := "min"+strings.Title(kind), "max"+strings.Title(kind)
	if kind == "length" {
		minimum, maximum = "minLength", "maxLength"
	}
	if value, ok := schema[minimum].(float64); ok && size < int(value) {
		*errors = append(*errors, ValidationError{Code: "SCHEMA_MIN", Pointer: pointer})
	}
	if value, ok := schema[maximum].(float64); ok && size > int(value) {
		*errors = append(*errors, ValidationError{Code: "SCHEMA_MAX", Pointer: pointer})
	}
}

func validFormat(format, value string) bool {
	switch format {
	case "uuid":
		return uuidPattern.MatchString(value)
	case "date-time":
		_, ok := parseTime(value)
		return ok
	default:
		return true
	}
}
func childPointer(pointer, field string) string {
	if pointer == "/" {
		return "/" + field
	}
	return pointer + "/" + field
}
func stringIndex(index int) string { return strconv.Itoa(index) }

func ValidateRecoveryContract(value map[string]any) []ValidationError {
	bootstrap, ok := value["bootstrap"].(map[string]any)
	if !ok || bootstrap["strategy"] != "historical_event_api_backfill_from_source_consistent_cut" || bootstrap["realtime_table_dump"] != "forbidden" || bootstrap["cursor_resume"] != "required" || bootstrap["reconciliation"] != "required_before_resume" {
		return []ValidationError{{Code: "BOOTSTRAP_RESUME_CONTRACT_INVALID", Pointer: "/bootstrap/cursor_resume"}}
	}
	return nil
}

func ValidateVersionCompatibility(value map[string]any) []ValidationError {
	compatibility, ok := value["version_compatibility"].(map[string]any)
	if !ok || compatibility["predeployment_overlap"] != "forbidden" || compatibility["v1_v3_concurrent_delivery"] != "forbidden" || compatibility["negotiation"] != "v3_only_before_first_deployment" || compatibility["rollback"] != "forward_fix_only" {
		return []ValidationError{{Code: "VERSION_COMPATIBILITY_OVERLAP_INVALID", Pointer: "/version_compatibility/v1_v3_concurrent_delivery"}}
	}
	return nil
}

func parseTime(value any) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, stringValue(value))
	return parsed, err == nil
}
func stringValue(value any) string { text, _ := value.(string); return text }
func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
