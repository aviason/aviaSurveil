package datafeed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"

	"golang.org/x/text/unicode/norm"
)

var v3EventContentProjection = []string{
	"contract_id", "contract_version", "event_id", "event_type", "event_version",
	"occurred_at", "emitted_at", "effective_at", "known_at", "source_module",
	"source_system", "tenant_id", "owning_organization_id", "actor_organization_id",
	"visibility_purpose_code", "business_key", "correlation_id", "causation_id",
	"aggregate_type", "aggregate_id", "aggregate_revision", "entity_refs",
	"state_before", "state_after", "privacy_class", "payload", "payload_sha256",
}

var typedLengthPrefixedDomain = []byte("aviacore.typed_length_prefixed.v1\x00")
var eventContentV3Domain = []byte("aviacore.event_content_hash.v3\x00")

// EventContentDigest is the producer-side implementation of the locked
// event_content_hash.v3 contract. It deliberately reconstructs the declared
// projection at publication time rather than trusting a mutable transport
// header or a legacy JSON-byte digest.
func EventContentDigest(event map[string]any) (string, error) {
	projection := make(map[string]any, len(v3EventContentProjection))
	for _, field := range v3EventContentProjection {
		value, ok := event[field]
		if !ok {
			return "", fmt.Errorf("v3 event content projection is missing %q", field)
		}
		projection[field] = value
	}
	encoded, err := typedLengthPrefixed(projection)
	if err != nil {
		return "", fmt.Errorf("encode v3 event content projection: %w", err)
	}
	sum := sha256.Sum256(append(append([]byte{}, eventContentV3Domain...), append(typedLengthPrefixedDomain, encoded...)...))
	return hex.EncodeToString(sum[:]), nil
}

func typedLengthPrefixed(value any) ([]byte, error) {
	switch current := value.(type) {
	case nil:
		return typedFrame('N', nil), nil
	case bool:
		if current {
			return typedFrame('B', []byte("1")), nil
		}
		return typedFrame('B', []byte("0")), nil
	case string:
		return typedFrame('S', []byte(norm.NFC.String(current))), nil
	case int:
		return typedFrame('I', []byte(strconv.Itoa(current))), nil
	case int64:
		return typedFrame('I', []byte(strconv.FormatInt(current, 10))), nil
	case float64:
		if math.Trunc(current) != current || math.IsNaN(current) || math.IsInf(current, 0) {
			return nil, fmt.Errorf("JSON float is forbidden by the typed v3 hash contract")
		}
		return typedFrame('I', []byte(strconv.FormatInt(int64(current), 10))), nil
	case []any:
		var payload []byte
		for _, item := range current {
			encoded, err := typedLengthPrefixed(item)
			if err != nil {
				return nil, err
			}
			payload = append(payload, encoded...)
		}
		return typedFrame('A', payload), nil
	case map[string]any:
		type normalizedItem struct {
			key   []byte
			value any
		}
		items := make([]normalizedItem, 0, len(current))
		seen := make(map[string]struct{}, len(current))
		for key, item := range current {
			normalized := []byte(norm.NFC.String(key))
			stableKey := string(normalized)
			if _, exists := seen[stableKey]; exists {
				return nil, fmt.Errorf("object keys collide after NFC normalization")
			}
			seen[stableKey] = struct{}{}
			items = append(items, normalizedItem{key: normalized, value: item})
		}
		sort.Slice(items, func(left, right int) bool { return string(items[left].key) < string(items[right].key) })
		var payload []byte
		for _, item := range items {
			encoded, err := typedLengthPrefixed(item.value)
			if err != nil {
				return nil, err
			}
			payload = append(payload, typedFrame('K', item.key)...)
			payload = append(payload, encoded...)
		}
		return typedFrame('O', payload), nil
	default:
		return nil, fmt.Errorf("unsupported canonical value type %T", value)
	}
}

func typedFrame(tag byte, payload []byte) []byte {
	framed := make([]byte, 0, 1+20+1+len(payload))
	framed = append(framed, tag)
	framed = append(framed, strconv.AppendInt(nil, int64(len(payload)), 10)...)
	framed = append(framed, ':')
	return append(framed, payload...)
}
