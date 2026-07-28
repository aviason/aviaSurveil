package preproddata

import (
	"bytes"
	"testing"
)

func TestAuthoritativeCommandValidationRejectsUnsafePayloads(t *testing.T) {
	valid := AuthoritativeCommand{
		Family:      "organizations",
		OperationID: "synthetic-operation-000001",
		Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0001"}`),
	}
	if err := validateAuthoritativeCommand(valid); err != nil {
		t.Fatalf("valid command: %v", err)
	}

	for name, payload := range map[string][]byte{
		"invalid JSON":  []byte(`{"organizationId":`),
		"nested secret": []byte(`{"provider":{"client_secret":"forbidden"}}`),
		"token variant": []byte(`{"access-token":"forbidden"}`),
		"oversized":     bytes.Repeat([]byte{'x'}, 1<<20+1),
	} {
		t.Run(name, func(t *testing.T) {
			command := valid
			command.Payload = payload
			if err := validateAuthoritativeCommand(command); err == nil {
				t.Fatalf("unsafe authoritative command was accepted")
			}
		})
	}
}
