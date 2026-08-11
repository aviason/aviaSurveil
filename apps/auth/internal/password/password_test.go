package password

import (
	"errors"
	"sync"
	"testing"
)

func testHasher(t *testing.T, capacity int) *Hasher {
	t.Helper()
	hasher, err := New(Params{MemoryKiB: 16 * 1024, Time: 1, Threads: 1, KeyLength: 32, SaltLen: 16, MaxBytes: 1024, Capacity: capacity})
	if err != nil {
		t.Fatal(err)
	}
	return hasher
}

func TestArgon2idHashVerifyAndUnknownDummyPath(t *testing.T) {
	hasher := testHasher(t, 2)
	hash, err := hasher.Hash([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := hasher.Verify(hash, []byte("correct horse battery staple")); err != nil || !ok {
		t.Fatalf("known password verification = %t/%v", ok, err)
	}
	if ok, err := hasher.Verify(hash, []byte("wrong password")); err != nil || ok {
		t.Fatalf("wrong password verification = %t/%v", ok, err)
	}
	if ok, err := hasher.Verify(hasher.DummyHash(), []byte("wrong password")); err != nil || ok {
		t.Fatalf("dummy verification = %t/%v", ok, err)
	}
	if _, err := hasher.Hash(make([]byte, 1025)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("long password error = %v", err)
	}
}

func TestPasswordPolicyRejectsReuseAndCompromisedValue(t *testing.T) {
	hasher := testHasher(t, 2)
	current, err := hasher.Hash([]byte("current password 1"))
	if err != nil {
		t.Fatal(err)
	}
	policy := Policy{MinBytes: 12, MaxBytes: 1024, Compromised: func(value []byte) bool { return string(value) == "compromised password" }}
	if err := policy.Validate([]byte("current password 1"), hasher, current, nil); !errors.Is(err, ErrPasswordReused) {
		t.Fatalf("current reuse error = %v", err)
	}
	if err := policy.Validate([]byte("compromised password"), hasher, current, nil); !errors.Is(err, ErrCompromised) {
		t.Fatalf("compromised error = %v", err)
	}
}

func TestHashAdmissionIsBounded(t *testing.T) {
	hasher := testHasher(t, 1)
	gate, err := hasher.acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer gate()
	if _, err := hasher.Hash([]byte("this request is admitted after the gate")); !errors.Is(err, ErrHashCapacity) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestHashConcurrentCapacityDoesNotRace(t *testing.T) {
	hasher := testHasher(t, 2)
	var wait sync.WaitGroup
	for index := 0; index < 8; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _ = hasher.Hash([]byte("bounded concurrent password"))
		}()
	}
	wait.Wait()
}
