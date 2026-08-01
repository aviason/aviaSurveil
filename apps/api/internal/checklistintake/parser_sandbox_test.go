package checklistintake

import "testing"

func TestParserSandboxPolicyDeniesNetworkAndSecrets(t *testing.T) {
	policy := DefaultParserSandboxPolicy()
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
	policy.AllowedSyscalls = append(policy.AllowedSyscalls, "socket")
	if err := policy.Validate(); err == nil {
		t.Fatal("network-capable syscall was accepted")
	}
	policy = DefaultParserSandboxPolicy()
	policy.Environment = append(policy.Environment, "DATABASE_URL=postgres://secret")
	if err := policy.Validate(); err == nil {
		t.Fatal("secret-bearing environment was accepted")
	}
}
