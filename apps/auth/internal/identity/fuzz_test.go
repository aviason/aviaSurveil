package identity_test

import (
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
)

func FuzzIdentifierNormalizationNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"pilot@example.invalid",
		"  pilot@example.invalid  ",
		"synthetic-user-2026",
		"\x00\xff",
		strings.Repeat("a", 4096),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(_ *testing.T, value string) {
		_, _ = identity.DetectIdentifier(value)
		_, _ = identity.NormalizeIdentifier(identity.IdentifierEmail, value)
		_, _ = identity.NormalizeIdentifier(identity.IdentifierUsername, value)
	})
}
