package security

import (
	"strings"
	"testing"
)

// TestHashPasswordFormat verifies that hashes are in correct format
func TestHashPasswordFormat(t *testing.T) {
	password := "TestPassword123"
	hash := HashPassword(password)

	// Format should be: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Hash doesn't start with $argon2id$, got: %s", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("Expected 6 parts in hash format, got %d", len(parts))
	}

	if parts[1] != "argon2id" {
		t.Errorf("Expected algorithm 'argon2id', got: %s", parts[1])
	}

	if !strings.Contains(parts[3], "m=65536") {
		t.Errorf("Expected m=65536 in parameters, got: %s", parts[3])
	}
}

// TestVerifyPassword verifies correct password verification
func TestVerifyPassword(t *testing.T) {
	password := "MySecurePassword123"
	hash := HashPassword(password)

	// Correct password should verify
	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword failed for correct password")
	}

	// Wrong password should not verify
	if VerifyPassword("WrongPassword", hash) {
		t.Error("VerifyPassword succeeded for wrong password")
	}

	// Empty password should not verify
	if VerifyPassword("", hash) {
		t.Error("VerifyPassword succeeded for empty password")
	}
}

// TestVerifyPasswordTimingResistance verifies timing resistance
// (This is a basic test, real timing analysis requires specialized tools)
func TestVerifyPasswordTimingResistance(t *testing.T) {
	password := "CorrectPassword"
	hash := HashPassword(password)

	wrongPassword := "WrongPassword"

	// Both should return false but take similar time
	result1 := VerifyPassword(wrongPassword, hash)
	result2 := VerifyPassword("AnotherWrong", hash)

	if result1 || result2 {
		t.Error("Wrong passwords should not verify")
	}
	// Timing comparison would require benchmark tools
}

// TestHashDiversity verifies different hashes for same password
func TestHashDiversity(t *testing.T) {
	password := "TestPassword"

	hash1 := HashPassword(password)
	hash2 := HashPassword(password)

	// Hashes should be different (different salts)
	if hash1 == hash2 {
		t.Error("Same password produced identical hashes (salt reuse)")
	}

	// But both should verify the password
	if !VerifyPassword(password, hash1) {
		t.Error("Hash1 verification failed")
	}
	if !VerifyPassword(password, hash2) {
		t.Error("Hash2 verification failed")
	}
}

// TestInvalidHashFormat verifies handling of invalid hash formats
func TestInvalidHashFormat(t *testing.T) {
	testCases := []string{
		"",                                    // Empty
		"invalid",                             // No format
		"$invalid$format",                     // Wrong algorithm
		"$argon2id$v=19$incomplete$hash",     // Incomplete format
		"$argon2id$v=19$m=invalid,t=3,p=4$salt$hash", // Invalid parameters
	}

	for _, invalidHash := range testCases {
		if VerifyPassword("password", invalidHash) {
			t.Errorf("VerifyPassword should fail for invalid hash: %s", invalidHash)
		}
	}
}

// BenchmarkHashPassword benchmarks password hashing
func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

// BenchmarkVerifyPassword benchmarks password verification
func BenchmarkVerifyPassword(b *testing.B) {
	password := "BenchmarkPassword123"
	hash := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(password, hash)
	}
}

// BenchmarkVerifyPasswordWrong benchmarks wrong password verification
func BenchmarkVerifyPasswordWrong(b *testing.B) {
	password := "BenchmarkPassword123"
	wrongPassword := "WrongPassword"
	hash := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(wrongPassword, hash)
	}
}
