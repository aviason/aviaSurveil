//go:build canonicaltest

package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"hash/crc32"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistintake"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestAGAForm048CandidateIntake(t *testing.T) {
	// Synthetic-only mechanism proof. The supplied AGA archive is intentionally
	// excluded here; the path-driven verifier is the only archive reader.
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	body := []byte("%PDF-1.7\n(Checklist for the surveillance of an aerodrome)")
	header := &zip.FileHeader{Name: "FSS-AGA-FORM-048.pdf", Method: zip.Store, CRC32: crc32.ChecksumIEEE(body), CompressedSize64: uint64(len(body)), UncompressedSize64: uint64(len(body))}
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
	service := checklistintake.NewService(nil)
	admin := identity.Principal{SubjectID: "synthetic-admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	digest := sha256.Sum256(archive.Bytes())
	expectedSHA := fmt.Sprintf("sha256:%x", digest[:])
	if _, err := service.ReceiveArchive(context.Background(), admin, "aga-048-synthetic", "aga-048-synthetic-1", expectedSHA, "candidate-only synthetic proof", archive.Bytes()); err != nil {
		t.Fatal(err)
	}
}
