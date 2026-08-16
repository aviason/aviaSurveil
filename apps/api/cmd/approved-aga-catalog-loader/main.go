// Command approved-aga-catalog-loader imports the immutable Aviation-approved
// AGA source package. It is a bootstrap-only binary and is not linked into the
// ordinary API, worker, or web images.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/preproddata/canonicalaga"
	"github.com/jackc/pgx/v5"
)

const loaderActorSubjectID = "avia-bootstrap"

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type catalogManifest struct {
	SchemaVersion         int    `json:"schemaVersion"`
	ManifestVersion       string `json:"manifestVersion"`
	AdvisoryLockKey       int64  `json:"advisoryLockKey"`
	Target                string `json:"target"`
	Enabled               bool   `json:"enabled"`
	CatalogVersion        string `json:"catalogVersion"`
	CatalogUsageClass     string `json:"catalogUsageClass"`
	CatalogOrigin         string `json:"catalogOrigin"`
	ProviderScopeID       string `json:"providerScopeId"`
	RegulatedTargetID     string `json:"regulatedTargetId"`
	PackagePath           string `json:"packagePath"`
	PackageVersion        string `json:"packageVersion"`
	PackageZipSHA256      string `json:"packageZipSha256"`
	PackageJSONSHA256     string `json:"packageJsonSha256"`
	SourceManifestSHA256  string `json:"sourceManifestSha256"`
	CatalogRootDigest     string `json:"catalogRootDigest"`
	FormCount             int    `json:"formCount"`
	QuestionCount         int    `json:"questionCount"`
	AIEnrichmentPath      string `json:"aiEnrichmentPath"`
	AIEnrichmentSHA256    string `json:"aiEnrichmentSha256"`
	AIEnrichmentVersion   string `json:"aiEnrichmentVersion"`
	AIEnrichmentDigest    string `json:"aiEnrichmentDigest"`
	AIEnrichmentItemCount int    `json:"aiEnrichmentItemCount"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "approved AGA catalog loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: approved-aga-catalog-loader validate|load --package PATH --manifest PATH --manifest-sha256 DIGEST --target TARGET --ai-enrichment PATH --ai-enrichment-sha256 DIGEST [--database-url-file PATH]")
	}
	command := args[0]
	flags := flag.NewFlagSet("approved-aga-catalog-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	packagePath := flags.String("package", os.Getenv("AVIA_APPROVED_CATALOG_PACKAGE_PATH"), "absolute approved AGA package path")
	manifestPath := flags.String("manifest", os.Getenv("AVIA_CATALOG_MANIFEST_PATH"), "absolute approved catalog manifest path")
	manifestSHA256 := flags.String("manifest-sha256", os.Getenv("AVIA_CATALOG_MANIFEST_SHA256"), "release-pinned catalog manifest digest")
	target := flags.String("target", os.Getenv("AVIA_BOOTSTRAP_TARGET"), "exact target")
	catalogVersion := flags.String("catalog-version", "aga-approved-source@2.0.0", "immutable catalog version")
	databaseURLFile := flags.String("database-url-file", os.Getenv("AVIA_DATABASE_URL_FILE"), "private target database URL file")
	actorSubjectID := flags.String("actor-subject-id", os.Getenv("AVIA_APPROVED_CATALOG_LOADER_ACTOR"), "bootstrap service identity")
	aiEnrichmentPath := flags.String("ai-enrichment", os.Getenv("AVIA_AI_RECOMMENDATION_ARTIFACT_PATH"), "absolute offline AI recommendation artifact path")
	aiEnrichmentSHA256 := flags.String("ai-enrichment-sha256", os.Getenv("AVIA_AI_RECOMMENDATION_ARTIFACT_SHA256"), "release-pinned offline AI recommendation artifact file digest")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid approved AGA loader arguments")
	}
	if !filepath.IsAbs(strings.TrimSpace(*packagePath)) || !filepath.IsAbs(strings.TrimSpace(*manifestPath)) || !filepath.IsAbs(strings.TrimSpace(*aiEnrichmentPath)) || strings.TrimSpace(*target) == "" {
		return errors.New("--package, --manifest, and --ai-enrichment must be absolute paths and an exact target is required")
	}
	if strings.TrimSpace(*catalogVersion) != "aga-approved-source@2.0.0" {
		return errors.New("approved AGA loader accepts only catalog version aga-approved-source@2.0.0")
	}
	if strings.TrimSpace(*actorSubjectID) != "" && strings.TrimSpace(*actorSubjectID) != loaderActorSubjectID {
		return errors.New("--actor-subject-id must use the fixed bootstrap service identity")
	}
	*actorSubjectID = loaderActorSubjectID
	manifest, manifestDigest, err := readCatalogManifest(*manifestPath, *manifestSHA256, *target)
	if err != nil {
		return fmt.Errorf("validate approved catalog manifest: %w", err)
	}
	if manifest.CatalogVersion != *catalogVersion || manifest.PackagePath != "apps/surveil/deliverables/AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip" || manifest.PackageVersion != "AGA_ALL_FORMS_APPROVED_SOURCE_V2" || manifest.CatalogUsageClass != "GOVERNED_OPERATIONAL" || manifest.CatalogOrigin != "IMPORTED_APPROVED_SOURCE" || manifest.AIEnrichmentPath != "apps/surveil/deliverables/aga-ai-checklist-recommendations-v1/AGA_AI_CHECKLIST_RECOMMENDATIONS_V1.json" || manifest.AIEnrichmentVersion != "aga-ai-checklist-recommendations/v1" || manifest.AIEnrichmentItemCount != 1310 {
		return errors.New("approved catalog manifest is not the governed Aviation source contract")
	}
	pkg, err := canonicalaga.ReadApprovedSourcePackage(ctx, *packagePath, canonicalaga.ExactApprovedSourcePackage())
	if err != nil {
		return fmt.Errorf("validate approved source package: %w", err)
	}
	releaseManifest, err := canonicalaga.BuildApprovedImportManifest(pkg, *catalogVersion)
	if err != nil {
		return fmt.Errorf("build approved catalog manifest: %w", err)
	}
	if manifest.PackageZipSHA256 != pkg.Identity.ZipSHA256 || manifest.PackageJSONSHA256 != pkg.Identity.JSONSHA256 || manifest.SourceManifestSHA256 != releaseManifest.SourceManifestSHA256 || manifest.CatalogRootDigest != releaseManifest.CatalogRootDigest || manifest.FormCount != len(releaseManifest.Forms) || manifest.QuestionCount != len(releaseManifest.Rows) {
		return errors.New("approved catalog package provenance differs from the release manifest")
	}
	artifact, artifactFileSHA256, err := canonicalaga.ReadAIRecommendationArtifact(*aiEnrichmentPath)
	if err != nil {
		return fmt.Errorf("validate offline AI recommendation artifact: %w", err)
	}
	if strings.TrimSpace(*aiEnrichmentSHA256) == "" || artifactFileSHA256 != strings.TrimSpace(*aiEnrichmentSHA256) || manifest.AIEnrichmentSHA256 != artifactFileSHA256 || manifest.AIEnrichmentDigest != artifact.ArtifactDigest {
		return errors.New("offline AI recommendation artifact provenance differs from the release manifest")
	}
	if command == "validate" {
		_, err := fmt.Fprintf(output, "approved AGA import validated: catalog=%s forms=%d questions=%d aiEnrichment=%d root=%s sourceManifest=%s aiDigest=%s manifest=%s\n", releaseManifest.CatalogVersion, len(releaseManifest.Forms), len(releaseManifest.Rows), artifact.ItemCount, releaseManifest.CatalogRootDigest, releaseManifest.SourceManifestSHA256, artifact.ArtifactDigest, manifestDigest)
		return err
	}
	databaseURL, err := readSecretFile(*databaseURLFile)
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open target database: %w", err)
	}
	defer pool.Close()
	var issuer, displayName string
	err = pool.QueryRow(ctx, `SELECT issuer, display_name FROM identity_references WHERE subject_id=$1`, *actorSubjectID).Scan(&issuer, &displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = pool.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ($1,'avia:bootstrap','Avia deployment bootstrap')`, *actorSubjectID)
	} else if err == nil && (issuer != "avia:bootstrap" || displayName != "Avia deployment bootstrap") {
		return fmt.Errorf("bootstrap actor %s drifted", *actorSubjectID)
	}
	if err != nil {
		return fmt.Errorf("register bootstrap actor: %w", err)
	}
	result, err := canonicalaga.LoadApprovedCatalog(ctx, pool, pkg, *catalogVersion, *actorSubjectID, manifest.ProviderScopeID, manifest.RegulatedTargetID, manifest.AdvisoryLockKey, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("load approved catalog: %w", err)
	}
	aiResult, err := canonicalaga.LoadAIRecommendationEnrichment(ctx, pool, artifact, *catalogVersion, manifest.AdvisoryLockKey, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("load offline AI recommendation enrichment: %w", err)
	}
	_, err = fmt.Fprintf(output, "approved AGA catalog loaded: catalog=%s forms=%d questions=%d aiEnrichment=%d root=%s aiDigest=%s\n", result.CatalogVersion, result.FormCount, result.QuestionCount, aiResult.ItemCount, result.ImportDigest, aiResult.ArtifactDigest)
	return err
}

func readCatalogManifest(path, expectedDigest, target string) (catalogManifest, string, error) {
	var manifest catalogManifest
	if !filepath.IsAbs(strings.TrimSpace(path)) {
		return manifest, "", errors.New("approved catalog manifest path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 2 || info.Size() > 64*1024 {
		return manifest, "", errors.New("approved catalog manifest must be a bounded regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, "", fmt.Errorf("read approved catalog manifest: %w", err)
	}
	digestBytes := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	if digest != strings.TrimSpace(expectedDigest) {
		return manifest, "", errors.New("approved catalog manifest digest mismatch")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, "", fmt.Errorf("decode approved catalog manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Target != target || !manifest.Enabled || manifest.ManifestVersion == "" || manifest.AdvisoryLockKey <= 0 || !digestPattern.MatchString(manifest.PackageZipSHA256) || !digestPattern.MatchString(manifest.PackageJSONSHA256) || !digestPattern.MatchString(manifest.SourceManifestSHA256) || !digestPattern.MatchString(manifest.CatalogRootDigest) || !digestPattern.MatchString(manifest.AIEnrichmentSHA256) || !digestPattern.MatchString(manifest.AIEnrichmentDigest) || manifest.AIEnrichmentVersion != "aga-ai-checklist-recommendations/v1" || manifest.AIEnrichmentItemCount != 1310 || manifest.FormCount != 52 || manifest.QuestionCount != 1310 || strings.TrimSpace(manifest.AIEnrichmentPath) == "" || strings.TrimSpace(manifest.ProviderScopeID) == "" || strings.TrimSpace(manifest.RegulatedTargetID) == "" {
		return manifest, "", errors.New("approved catalog manifest identity or counts are invalid")
	}
	return manifest, digest, nil
}

func readSecretFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", errors.New("database URL file must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > 8192 {
		return "", errors.New("database URL file is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read database URL file: %w", err)
	}
	raw := string(data)
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("database URL file is empty or malformed")
	}
	return value, nil
}
