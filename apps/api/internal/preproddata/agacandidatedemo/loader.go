package agacandidatedemo

import (
	"context"
	"fmt"
	"time"
)

// OverlayLoadInput intentionally omits every provider/client dependency. The
// only mutable dependency is the PostgreSQL overlay ProjectionStore.
type OverlayLoadInput struct {
	Intent       IntentManifest
	Package      AcceptedPackage
	BaseEvidence BaseRunEvidence
	Store        ProjectionStore
	Clock        func() time.Time
}

// LoadOverlay performs the one-shot, package-bound projection operation. A
// database seal is the only success receipt: callers cannot manufacture a
// result from a control record or a partial projection.
func LoadOverlay(ctx context.Context, input OverlayLoadInput) (ResultManifest, error) {
	if input.Store == nil {
		return ResultManifest{}, fmt.Errorf("AGA demo projection store is required")
	}
	if err := input.Intent.Validate(); err != nil {
		return ResultManifest{}, err
	}
	if err := VerifyBaseEvidence(input.Intent, input.BaseEvidence); err != nil {
		return ResultManifest{}, err
	}
	if err := packageMatchesIntent(input.Package, input.Intent); err != nil {
		return ResultManifest{}, err
	}
	digests, err := RelationshipDigests(input.Package)
	if err != nil {
		return ResultManifest{}, err
	}
	if err := exactDigestSet(input.Intent.ExpectedRelationshipDigests, digests); err != nil {
		return ResultManifest{}, err
	}
	if err := input.Store.Preflight(ctx, input.Intent); err != nil {
		return ResultManifest{}, fmt.Errorf("AGA demo PostgreSQL preflight: %w", err)
	}
	receipt, err := input.Store.Materialize(ctx, input.Intent, input.Package, digests)
	if err != nil {
		return ResultManifest{}, fmt.Errorf("materialize AGA demo projection: %w", err)
	}
	if err := receipt.Validate(input.Intent); err != nil {
		return ResultManifest{}, err
	}
	if receipt.PackageDigest != input.Package.Identity.JSONSHA256 || receipt.ReconciliationDigest != digests["projection"] {
		return ResultManifest{}, fmt.Errorf("AGA demo final seal does not reconcile the accepted package")
	}
	clock := input.Clock
	if clock == nil {
		clock = time.Now
	}
	return BuildResult(ResultInput{RunID: input.Intent.RunID, IntentDigest: input.Intent.IntentDigest, SealDigest: receipt.SealDigest, CompletedAt: clock().UTC()})
}

func packageMatchesIntent(pkg AcceptedPackage, intent IntentManifest) error {
	if pkg.Identity.ZipSHA256 != intent.PackageZipDigest || pkg.Identity.JSONSHA256 != intent.PackageJSONDigest || pkg.Identity.ManifestSHA256 != intent.PackageManifestDigest {
		return fmt.Errorf("accepted package identity does not match AGA demo intent")
	}
	return nil
}
