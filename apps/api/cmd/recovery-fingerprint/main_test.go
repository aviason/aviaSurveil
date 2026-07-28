package main

import "testing"

func TestFingerprintJSONCanonicalizesObjectKeys(t *testing.T) {
	t.Parallel()

	left, err := fingerprintJSON([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := fingerprintJSON([]byte("{\n  \"a\": 1,\n  \"b\": 2\n}"))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("fingerprints differ: %s != %s", left, right)
	}
}

func TestFingerprintJSONRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := fingerprintJSON([]byte(`{"broken"`)); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
}
