package canonicalaga

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkedInAIRecommendationArtifactPath(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve test working directory: %v", err)
	}
	for directory := workingDirectory; directory != filepath.Dir(directory); directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, "deliverables", "aga-ai-checklist-recommendations-v1", "AGA_AI_CHECKLIST_RECOMMENDATIONS_V1.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("checked-in AI recommendation artifact not found from %s", workingDirectory)
	return ""
}

func TestCheckedInAIRecommendationArtifactMatchesGoContract(t *testing.T) {
	artifact, fileDigest, err := ReadAIRecommendationArtifact(checkedInAIRecommendationArtifactPath(t))
	if err != nil {
		t.Fatalf("read checked-in AI recommendation artifact: %v", err)
	}
	if artifact.ItemCount != aiRecommendationQuestionCount || len(artifact.Items) != aiRecommendationQuestionCount {
		t.Fatalf("artifact item count = %d/%d, want %d", artifact.ItemCount, len(artifact.Items), aiRecommendationQuestionCount)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(filepath.Dir(checkedInAIRecommendationArtifactPath(t)), "MANIFEST.sha256"))
	if err != nil {
		t.Fatalf("read artifact manifest: %v", err)
	}
	parts := strings.Fields(string(manifestBytes))
	if len(parts) != 2 || len(parts[0]) != 64 {
		t.Fatalf("artifact manifest has unexpected format: %q", string(manifestBytes))
	}
	if fileDigest != "sha256:"+parts[0] {
		t.Fatalf("artifact file digest = %s, manifest does not match", fileDigest)
	}
}

func TestAIRecommendationArtifactRejectsNeutralStateDrift(t *testing.T) {
	artifact, _, err := ReadAIRecommendationArtifact(checkedInAIRecommendationArtifactPath(t))
	if err != nil {
		t.Fatalf("read checked-in AI recommendation artifact: %v", err)
	}
	artifact.Items[0].AdvisoryState = "MANAGER_REVIEW_REQUIRED"
	if err := ValidateAIRecommendationArtifact(artifact); err == nil {
		t.Fatal("expected legacy approval state to be rejected")
	}
}
