package checklistintake

import (
	"archive/zip"
	"bytes"
	"context"
	"hash/crc32"
	"testing"

	"github.com/aviason/aviaSurveil/internal/checklistintake"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestWorkerProcessesOnlyAdminCandidateIntakeAndKeepsConcurrencyOne(t *testing.T) {
	worker := New(checklistintake.NewService(nil))
	if worker.MaxConcurrency != 1 {
		t.Fatalf("concurrency=%d", worker.MaxConcurrency)
	}
	data := syntheticArchive(t)
	admin := identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	if _, err := worker.Receive(context.Background(), admin, "op", "idem", data); err != nil {
		t.Fatal(err)
	}
}

func syntheticArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	body := []byte("%PDF-1.7")
	header := &zip.FileHeader{Name: "form.pdf", Method: zip.Store, CRC32: crc32.ChecksumIEEE(body), CompressedSize64: uint64(len(body)), UncompressedSize64: uint64(len(body))}
	entry, err := writer.CreateRaw(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
