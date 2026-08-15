package canonicalaga

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApprovedSourcePackageHasExactImmutableBoundary(t *testing.T) {
	packagePath, err := filepath.Abs("../../../../../deliverables/AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip")
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := ReadApprovedSourcePackage(context.Background(), packagePath, ExactApprovedSourcePackage())
	if err != nil {
		t.Fatalf("read approved package: %v", err)
	}
	if len(accepted.Forms) != 52 || accepted.CatalogRootDigest != ExactApprovedSourcePackage().CatalogRootDigest {
		t.Fatalf("approved package boundary = forms %d/root %s", len(accepted.Forms), accepted.CatalogRootDigest)
	}
	questions := 0
	for _, form := range accepted.Forms {
		questions += len(form.Questions)
	}
	if questions != 1310 {
		t.Fatalf("approved question count = %d", questions)
	}
}

func TestApprovedSourcePackageRejectsV1AndRelativePaths(t *testing.T) {
	if _, err := ReadApprovedSourcePackage(context.Background(), "relative.zip", ExactApprovedSourcePackage()); err == nil {
		t.Fatal("accepted relative approved package path")
	}
	v1, err := filepath.Abs("../../../../../deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(v1); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadApprovedSourcePackage(context.Background(), v1, ExactApprovedSourcePackage()); err == nil {
		t.Fatal("accepted historical V1 package as approved V2")
	}
}
