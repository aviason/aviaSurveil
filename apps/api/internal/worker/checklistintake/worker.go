package checklistintake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistintake"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

type Worker struct {
	Service        *checklistintake.Service
	MaxConcurrency int
}

func New(service *checklistintake.Service) *Worker {
	return &Worker{Service: service, MaxConcurrency: 1}
}

func (worker *Worker) Receive(ctx context.Context, principal identity.Principal, operationID, idempotencyKey string, archive []byte) (checklistintake.ImportBatch, error) {
	if worker == nil || worker.Service == nil {
		return checklistintake.ImportBatch{}, checklistintake.ErrAppendOnlyViolation
	}
	digest := sha256.Sum256(archive)
	expectedSHA := "sha256:" + hex.EncodeToString(digest[:])
	return worker.Service.ReceiveArchive(ctx, principal, operationID, idempotencyKey, expectedSHA, "worker received candidate-only archive", archive)
}
