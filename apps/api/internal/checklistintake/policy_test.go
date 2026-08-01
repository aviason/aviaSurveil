package checklistintake

import "testing"

func TestAGAZipPDFV1LimitsAreFrozen(t *testing.T) {
	policy := AGAZipPDFV1()
	if policy.MaxArchiveBytes != 50*1024*1024 || policy.MaxCentralDirectoryRecords != 128 || policy.MaxPathDepth != 4 || policy.MaxPathBytes != 512 || policy.MaxPDFBytes != 20*1024*1024 || policy.MaxTotalUncompressedBytes != 100*1024*1024 || policy.MaxExpansionRatio != 20 {
		t.Fatalf("unexpected frozen policy: %+v", policy)
	}
}

func TestNormalizeZipPathV1UsesNFCAndRejectsUnsafeForms(t *testing.T) {
	path, err := NormalizeZipPathV1("dir/e\u0301vidence.pdf", false)
	if err != nil || path != "dir/évidence.pdf" {
		t.Fatalf("NFC path=(%q, %v)", path, err)
	}
	for _, unsafe := range []string{"../evidence.pdf", `/absolute.pdf`, `C:\evidence.pdf`, `dir\\evidence.pdf`, "dir/./evidence.pdf", "dir/\x00.pdf", "dir/\u202e.pdf"} {
		if _, err := NormalizeZipPathV1(unsafe, false); err == nil {
			t.Fatalf("unsafe path %q was accepted", unsafe)
		}
	}
}

func TestNormalizeZipPathV1DirectoryOnlyRemovesExactlyOneSlash(t *testing.T) {
	path, err := NormalizeZipPathV1("dir/", true)
	if err != nil || path != "dir" {
		t.Fatalf("directory path=(%q, %v)", path, err)
	}
	if _, err := NormalizeZipPathV1("dir//", true); err == nil {
		t.Fatal("directory with multiple trailing slashes was accepted")
	}
}
