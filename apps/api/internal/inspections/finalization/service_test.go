package finalization

import "testing"

func TestCanonicalManifestDigestIsVersionedAndStable(t *testing.T) {
	left := []manifestEntry{
		{EntityID: "RESP-2", Revision: 2, Digest: "sha256:b"},
		{EntityID: "RESP-1", Revision: 1, Digest: "sha256:a"},
	}
	right := []manifestEntry{
		{EntityID: "RESP-1", Revision: 1, Digest: "sha256:a"},
		{EntityID: "RESP-2", Revision: 2, Digest: "sha256:b"},
	}
	first, err := canonicalManifestDigest("answers", right)
	if err != nil {
		t.Fatalf("canonical digest: %v", err)
	}
	second, err := canonicalManifestDigest("answers", right)
	if err != nil || first != second {
		t.Fatalf("canonical digest is not stable: %s != %s (err=%v)", first, second, err)
	}
	ordered, err := canonicalManifestDigest("answers", left)
	if err != nil || ordered != first {
		t.Fatalf("manifest order must not change the canonical digest: %s != %s (err=%v)", ordered, first, err)
	}
	changed, err := canonicalManifestDigest("answers", append(left, manifestEntry{EntityID: "RESP-3", Revision: 1, Digest: "sha256:c"}))
	if err != nil || changed == first {
		t.Fatalf("manifest content must change the canonical digest: %s (err=%v)", changed, err)
	}
	if len(first) != len("sha256:")+64 {
		t.Fatalf("digest = %q, want sha256 hex digest", first)
	}
}
