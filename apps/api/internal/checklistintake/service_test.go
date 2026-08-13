package checklistintake

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestReceiveArchiveIsAdminOnlyAndReplayStable(t *testing.T) {
	service := NewService(nil)
	data := syntheticArchive(t, map[string][]byte{"form.pdf": []byte("%PDF-1.7\n(question)")})
	admin := identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	manager := identity.Principal{SubjectID: "manager", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}}
	digest := sha256.Sum256(data)
	expectedSHA := fmt.Sprintf("sha256:%x", digest[:])
	if _, err := service.ReceiveArchive(context.Background(), manager, "op", "idem", expectedSHA, "reason", data); err == nil {
		t.Fatal("manager was allowed to receive archive")
	}
	first, err := service.ReceiveArchive(context.Background(), admin, "op", "idem", expectedSHA, "reason", data)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != ImportBatchProcessing || first.FinalizedAt != nil || first.IntakeSafetyEligible {
		t.Fatalf("archive receive bypassed scan/parser finalization boundary: %+v", first)
	}
	receipts, err := service.ListPhaseReceipts(admin, first.ImportBatchID)
	if err != nil || len(receipts) != 1 || receipts[0].Phase != PhaseArchiveValidate || receipts[0].Outcome != ReceiptSucceeded {
		t.Fatalf("unexpected receive receipts: %+v err=%v", receipts, err)
	}
	second, err := service.ReceiveArchive(context.Background(), admin, "op", "idem", expectedSHA, "reason", data)
	if err != nil || first.ManifestDigest != second.ManifestDigest || first.ObservedArchiveSHA != second.ObservedArchiveSHA {
		t.Fatalf("replay changed receipt: first=%+v second=%+v err=%v", first, second, err)
	}
	firstResult, err := service.ReceiveArchiveResult(context.Background(), admin, "op-2", "idem-2", expectedSHA, "reason", data)
	if err != nil || firstResult.Replayed {
		t.Fatalf("first envelope was marked replayed: %+v err=%v", firstResult, err)
	}
	secondResult, err := service.ReceiveArchiveResult(context.Background(), admin, "op-2", "idem-2", expectedSHA, "reason", data)
	if err != nil || !secondResult.Replayed {
		t.Fatalf("replay envelope was not marked replayed: %+v err=%v", secondResult, err)
	}
}
