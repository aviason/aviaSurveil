package checklistintake

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"os"
	"testing"
)

func syntheticArchive(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range entries {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		if name[len(name)-1] != '/' {
			header.SetMode(0o600)
		}
		header.Flags = 0
		header.CRC32 = crc32.ChecksumIEEE(body)
		header.CompressedSize64 = uint64(len(body))
		header.UncompressedSize64 = uint64(len(body))
		entry, err := writer.CreateRaw(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInventoryArchiveAcceptsBoundedPDFAndDirectory(t *testing.T) {
	data := syntheticArchive(t, map[string][]byte{"forms/": nil, "forms/FSS-AGA-FORM-048.pdf": []byte("%PDF-1.7\nsynthetic")})
	inventory, err := InventoryArchive(data)
	if err != nil {
		if archiveErr, ok := err.(*ArchiveError); ok {
			t.Fatalf("inventory error: %s: %v", archiveErr.Code, archiveErr.Err)
		}
		t.Fatal(err)
	}
	if len(inventory.Entries) != 2 || inventory.PDFCount != 1 || inventory.DirectoryCount != 1 || inventory.ManifestDigest == "" {
		t.Fatalf("unexpected archive inventory: %+v", inventory)
	}
}

func TestInventoryArchiveReaderAtMatchesBoundedInventory(t *testing.T) {
	data := syntheticArchive(t, map[string][]byte{"forms/": nil, "forms/FSS-AGA-FORM-048.pdf": []byte("%PDF-1.7\nreader-at")})
	file, err := os.CreateTemp(t.TempDir(), "aga-reader-at-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	expectedSHA := "sha256:" + hex.EncodeToString(digest[:])
	want, err := InventoryArchive(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := InventoryArchiveReaderAt(file, int64(len(data)), expectedSHA, AGAZipPDFV1())
	if err != nil {
		t.Fatal(err)
	}
	if got.ArchiveSHA256 != expectedSHA || got.ArchiveBytes != int64(len(data)) || got.ManifestDigest != want.ManifestDigest || got.PDFCount != want.PDFCount || got.DirectoryCount != want.DirectoryCount {
		t.Fatalf("reader-at inventory mismatch: got=%+v want=%+v", got, want)
	}
}

func TestInventoryArchiveRejectsTraversalDuplicateAndNonPDF(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"traversal", syntheticArchive(t, map[string][]byte{"../escape.pdf": []byte("%PDF-1.7")})},
		{"non-pdf", syntheticArchive(t, map[string][]byte{"notes.txt": []byte("not a PDF")})},
		{"duplicate-normalized", syntheticArchive(t, map[string][]byte{"Forms/A.pdf": []byte("%PDF-1.7"), "forms/a.pdf": []byte("%PDF-1.7")})},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := InventoryArchive(testCase.data); err == nil {
				t.Fatal("unsafe archive was accepted")
			}
		})
	}
}

func TestInventoryArchiveEnforcesRatioAndPDFBytes(t *testing.T) {
	data := syntheticArchive(t, map[string][]byte{"large.pdf": []byte("%PDF-1.7\n1234567890")})
	policy := AGAZipPDFV1()
	policy.MaxPDFBytes = 8
	if _, err := InventoryArchiveWithPolicy(data, policy); err == nil {
		t.Fatal("per-PDF byte limit was not enforced")
	}
}

func signatureOffset(data []byte, signature uint32, start int) int {
	for offset := start; offset+4 <= len(data); offset++ {
		if binary.LittleEndian.Uint32(data[offset:offset+4]) == signature {
			return offset
		}
	}
	return -1
}

func TestInventoryArchiveRejectsStrictZipStructureViolations(t *testing.T) {
	base := syntheticArchive(t, map[string][]byte{"one.pdf": []byte("%PDF-1.7\nbody")})
	eocd := signatureOffset(base, 0x06054b50, 0)
	central := signatureOffset(base, 0x02014b50, 0)
	if eocd < 0 || central < 0 {
		t.Fatal("synthetic ZIP did not contain central directory and EOCD")
	}

	cases := []struct {
		name string
		data []byte
	}{
		{name: "trailing-data", data: append(append([]byte(nil), base...), []byte("trailing")...)},
		{name: "zip64-sentinel", data: func() []byte {
			mutated := append([]byte(nil), base...)
			binary.LittleEndian.PutUint16(mutated[eocd+10:eocd+12], 0xffff)
			return mutated
		}()},
		{name: "zip64-locator", data: func() []byte {
			marker := make([]byte, 20)
			binary.LittleEndian.PutUint32(marker, 0x07064b50)
			mutated := append([]byte(nil), base[:eocd]...)
			mutated = append(mutated, marker...)
			return append(mutated, base[eocd:]...)
		}()},
		{name: "multi-disk", data: func() []byte {
			mutated := append([]byte(nil), base...)
			binary.LittleEndian.PutUint16(mutated[eocd+4:eocd+6], 1)
			return mutated
		}()},
		{name: "local-central-method-mismatch", data: func() []byte {
			mutated := append([]byte(nil), base...)
			binary.LittleEndian.PutUint16(mutated[8:10], zip.Deflate)
			return mutated
		}()},
		{name: "local-central-crc-mismatch", data: func() []byte {
			mutated := append([]byte(nil), base...)
			binary.LittleEndian.PutUint32(mutated[14:18], 1)
			return mutated
		}()},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := InventoryArchive(testCase.data); err == nil {
				t.Fatal("strict ZIP violation was accepted")
			}
		})
	}

	two := syntheticArchive(t, map[string][]byte{"one.pdf": []byte("%PDF-1.7\none"), "two.pdf": []byte("%PDF-1.7\ntwo")})
	twoCentral := signatureOffset(two, 0x02014b50, 0)
	if twoCentral < 0 {
		t.Fatal("two-entry ZIP did not contain a central directory")
	}
	secondCentral := signatureOffset(two, 0x02014b50, twoCentral+46)
	if secondCentral < 0 {
		t.Fatal("two-entry ZIP did not contain a second central header")
	}
	overlap := append([]byte(nil), two...)
	firstLocalOffset := binary.LittleEndian.Uint32(overlap[twoCentral+42 : twoCentral+46])
	binary.LittleEndian.PutUint32(overlap[secondCentral+42:secondCentral+46], firstLocalOffset)
	if _, err := InventoryArchive(overlap); err == nil {
		t.Fatal("overlapping local entry ranges were accepted")
	}
}
