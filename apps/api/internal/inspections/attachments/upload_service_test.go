package attachments

import "testing"

func TestResumablePartLayoutRequiresEveryOrderedPart(t *testing.T) {
	if got := expectedPartCount(10, 4); got != 3 {
		t.Fatalf("part count = %d, want 3", got)
	}
	complete := []UploadPartReceipt{
		{PartNumber: 1, ByteSize: 4, SHA256: "sha256:part-1", AcknowledgedOffset: 4, ObjectVersion: "v1"},
		{PartNumber: 2, ByteSize: 4, SHA256: "sha256:part-2", AcknowledgedOffset: 8, ObjectVersion: "v2"},
		{PartNumber: 3, ByteSize: 2, SHA256: "sha256:part-3", AcknowledgedOffset: 10, ObjectVersion: "v3"},
	}
	if err := validateResumablePartLayout(10, 4, complete); err != nil {
		t.Fatalf("complete ordered layout: %v", err)
	}
	if err := validateResumablePartLayout(10, 4, complete[:2]); err == nil {
		t.Fatal("missing final part must not complete an upload")
	}
}

func TestResumablePartLayoutRejectsConflictingPartIdentity(t *testing.T) {
	parts := []UploadPartReceipt{
		{PartNumber: 1, ByteSize: 4, SHA256: "sha256:part-1", AcknowledgedOffset: 4, ObjectVersion: "v1"},
		{PartNumber: 1, ByteSize: 4, SHA256: "sha256:other", AcknowledgedOffset: 4, ObjectVersion: "v2"},
	}
	if err := validateResumablePartLayout(4, 4, parts); err == nil {
		t.Fatal("conflicting same-number part identities must be rejected")
	}
}
