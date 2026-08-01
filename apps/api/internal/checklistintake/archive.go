package checklistintake

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
)

type ArchiveEntryKind string

const (
	ArchiveDirectory ArchiveEntryKind = "DIRECTORY"
	ArchivePDF       ArchiveEntryKind = "PDF"
)

type ArchiveEntry struct {
	Ordinal           int              `json:"ordinal"`
	Kind              ArchiveEntryKind `json:"kind"`
	Path              string           `json:"path"`
	SHA256            string           `json:"sha256,omitempty"`
	CompressedBytes   uint64           `json:"compressedBytes"`
	UncompressedBytes uint64           `json:"uncompressedBytes"`
	CRC32             uint32           `json:"crc32"`
}

type ArchiveInventory struct {
	ArchiveSHA256     string
	ArchiveBytes      int64
	Entries           []ArchiveEntry
	PDFCount          int
	DirectoryCount    int
	TotalUncompressed uint64
	ManifestDigest    string
}

type ArchiveError struct {
	Code string
	Err  error
}

func (e *ArchiveError) Error() string { return e.Code }
func (e *ArchiveError) Unwrap() error { return e.Err }

func InventoryArchive(data []byte) (ArchiveInventory, error) {
	return InventoryArchiveWithPolicy(data, AGAZipPDFV1())
}

func InventoryArchiveWithPolicy(data []byte, policy IntakePolicy) (ArchiveInventory, error) {
	if int64(len(data)) > policy.MaxArchiveBytes {
		return ArchiveInventory{}, &ArchiveError{Code: "ARCHIVE_SIZE_EXCEEDED", Err: ErrArchiveLimit}
	}
	if len(data) < 4 || !bytes.Equal(data[:4], []byte("PK\x03\x04")) && !bytes.Equal(data[:4], []byte("PK\x05\x06")) {
		return ArchiveInventory{}, &ArchiveError{Code: "ZIP_MAGIC_MISMATCH", Err: errors.New("archive magic is not ZIP")}
	}
	if err := validateStrictZIPStructure(data); err != nil {
		return ArchiveInventory{}, err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ArchiveInventory{}, &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: err}
	}
	if len(reader.File) > policy.MaxCentralDirectoryRecords {
		return ArchiveInventory{}, &ArchiveError{Code: "ENTRY_COUNT_EXCEEDED", Err: ErrArchiveLimit}
	}
	inventory := ArchiveInventory{
		ArchiveSHA256: "sha256:" + hex.EncodeToString(sha256Sum(data)),
		ArchiveBytes:  int64(len(data)),
		Entries:       make([]ArchiveEntry, 0, len(reader.File)),
	}
	seen := make(map[string]ArchiveEntryKind, len(reader.File))
	fold := cases.Fold()
	for ordinal, file := range reader.File {
		directory := file.FileInfo().IsDir()
		path, pathErr := NormalizeZipPathV1(file.Name, directory)
		if pathErr != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSAFE_PATH", Err: pathErr}
		}
		if !utf8.ValidString(file.Name) {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSAFE_PATH", Err: ErrUnsafeArchivePath}
		}
		collisionKey := fold.String(path)
		if _, exists := seen[collisionKey]; exists {
			return ArchiveInventory{}, &ArchiveError{Code: "DUPLICATE_NORMALIZED_PATH", Err: ErrUnsafeArchivePath}
		}
		for parent := parentPath(path); parent != ""; parent = parentPath(parent) {
			if kind, exists := seen[fold.String(parent)]; exists && kind == ArchivePDF {
				return ArchiveInventory{}, &ArchiveError{Code: "FILE_DIRECTORY_PREFIX_COLLISION", Err: ErrUnsafeArchivePath}
			}
		}
		seen[collisionKey] = ArchiveDirectory
		if file.Flags&^uint16(0x808) != 0 || file.Flags&0x1 != 0 {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_ENTRY_FLAGS", Err: errors.New("encrypted or unsupported ZIP flags")}
		}
		if file.Method != zip.Store && file.Method != zip.Deflate {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_COMPRESSION", Err: errors.New("ZIP method is not STORE or DEFLATE")}
		}
		if file.CompressedSize64 > math.MaxUint32 || file.UncompressedSize64 > math.MaxUint32 {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: ErrArchiveLimit}
		}
		if directory {
			if file.Method != zip.Store || file.CompressedSize64 != 0 || file.UncompressedSize64 != 0 || file.CRC32 != 0 || file.Flags&0x8 != 0 {
				return ArchiveInventory{}, &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("directory carries data")}
			}
			inventory.DirectoryCount++
			inventory.Entries = append(inventory.Entries, ArchiveEntry{Ordinal: ordinal + 1, Kind: ArchiveDirectory, Path: path})
			continue
		}
		seen[collisionKey] = ArchivePDF
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_ENTRY_TYPE", Err: errors.New("only PDF files are allowed")}
		}
		if file.CompressedSize64 > uint64(policy.MaxPDFBytes) || file.UncompressedSize64 > uint64(policy.MaxPDFBytes) || inventory.TotalUncompressed+file.UncompressedSize64 > uint64(policy.MaxTotalUncompressedBytes) {
			return ArchiveInventory{}, &ArchiveError{Code: "ENTRY_SIZE_EXCEEDED", Err: ErrArchiveLimit}
		}
		if err := ValidateExpansionRatio(int64(file.CompressedSize64), int64(file.UncompressedSize64), policy); err != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "EXPANSION_RATIO_EXCEEDED", Err: err}
		}
		reader, err := file.Open()
		if err != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP_READ_FAILED", Err: err}
		}
		limited := io.LimitReader(reader, policy.MaxPDFBytes+1)
		content, readErr := io.ReadAll(limited)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP_READ_FAILED", Err: firstError(readErr, closeErr)}
		}
		if int64(len(content)) > policy.MaxPDFBytes || len(content) < 5 || !bytes.Equal(content[:5], []byte("%PDF-")) {
			return ArchiveInventory{}, &ArchiveError{Code: "PDF_MAGIC_MISMATCH", Err: errors.New("entry is not a PDF")}
		}
		inventory.TotalUncompressed += uint64(len(content))
		inventory.PDFCount++
		inventory.Entries = append(inventory.Entries, ArchiveEntry{Ordinal: ordinal + 1, Kind: ArchivePDF, Path: path, SHA256: "sha256:" + hex.EncodeToString(sha256Sum(content)), CompressedBytes: file.CompressedSize64, UncompressedBytes: uint64(len(content)), CRC32: file.CRC32})
	}
	manifestBytes, err := json.Marshal(inventory.Entries)
	if err != nil {
		return ArchiveInventory{}, fmt.Errorf("marshal archive manifest: %w", err)
	}
	inventory.ManifestDigest = "sha256:" + hex.EncodeToString(sha256Sum(manifestBytes))
	return inventory, nil
}

// InventoryArchiveReaderAt applies the same AGA_ZIP_PDF_V1 inventory rules as
// InventoryArchiveWithPolicy while reading ZIP metadata and PDF contents from
// a bounded ReaderAt. The intake service uses this form after hashing a
// mode-0600 scratch file, so the complete archive is never materialized in
// process memory.
func InventoryArchiveReaderAt(reader io.ReaderAt, archiveBytes int64, archiveSHA string, policy IntakePolicy) (ArchiveInventory, error) {
	if reader == nil || archiveBytes < 0 {
		return ArchiveInventory{}, &ArchiveError{Code: "ZIP_READ_FAILED", Err: errors.New("archive reader is unavailable")}
	}
	if archiveBytes > policy.MaxArchiveBytes {
		return ArchiveInventory{}, &ArchiveError{Code: "ARCHIVE_SIZE_EXCEEDED", Err: ErrArchiveLimit}
	}
	if strings.TrimSpace(archiveSHA) == "" {
		return ArchiveInventory{}, &ArchiveError{Code: "ARCHIVE_HASH_MISSING", Err: errors.New("archive hash is required")}
	}
	magic, err := readAtExact(reader, 0, 4)
	if err != nil || (!bytes.Equal(magic, []byte("PK\x03\x04")) && !bytes.Equal(magic, []byte("PK\x05\x06"))) {
		return ArchiveInventory{}, &ArchiveError{Code: "ZIP_MAGIC_MISMATCH", Err: errors.New("archive magic is not ZIP")}
	}
	if err := validateStrictZIPStructureReaderAt(reader, archiveBytes, policy.MaxCentralDirectoryRecords); err != nil {
		return ArchiveInventory{}, err
	}
	zipReader, err := zip.NewReader(reader, archiveBytes)
	if err != nil {
		return ArchiveInventory{}, &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: err}
	}
	if len(zipReader.File) > policy.MaxCentralDirectoryRecords {
		return ArchiveInventory{}, &ArchiveError{Code: "ENTRY_COUNT_EXCEEDED", Err: ErrArchiveLimit}
	}
	inventory := ArchiveInventory{
		ArchiveSHA256: strings.TrimSpace(archiveSHA),
		ArchiveBytes:  archiveBytes,
		Entries:       make([]ArchiveEntry, 0, len(zipReader.File)),
	}
	seen := make(map[string]ArchiveEntryKind, len(zipReader.File))
	fold := cases.Fold()
	for ordinal, file := range zipReader.File {
		directory := file.FileInfo().IsDir()
		path, pathErr := NormalizeZipPathV1(file.Name, directory)
		if pathErr != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSAFE_PATH", Err: pathErr}
		}
		if !utf8.ValidString(file.Name) {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSAFE_PATH", Err: ErrUnsafeArchivePath}
		}
		collisionKey := fold.String(path)
		if _, exists := seen[collisionKey]; exists {
			return ArchiveInventory{}, &ArchiveError{Code: "DUPLICATE_NORMALIZED_PATH", Err: ErrUnsafeArchivePath}
		}
		for parent := parentPath(path); parent != ""; parent = parentPath(parent) {
			if kind, exists := seen[fold.String(parent)]; exists && kind == ArchivePDF {
				return ArchiveInventory{}, &ArchiveError{Code: "FILE_DIRECTORY_PREFIX_COLLISION", Err: ErrUnsafeArchivePath}
			}
		}
		seen[collisionKey] = ArchiveDirectory
		if file.Flags&^uint16(0x808) != 0 || file.Flags&0x1 != 0 {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_ENTRY_FLAGS", Err: errors.New("encrypted or unsupported ZIP flags")}
		}
		if file.Method != zip.Store && file.Method != zip.Deflate {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_COMPRESSION", Err: errors.New("ZIP method is not STORE or DEFLATE")}
		}
		if file.CompressedSize64 > math.MaxUint32 || file.UncompressedSize64 > math.MaxUint32 {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: ErrArchiveLimit}
		}
		if directory {
			if file.Method != zip.Store || file.CompressedSize64 != 0 || file.UncompressedSize64 != 0 || file.CRC32 != 0 || file.Flags&0x8 != 0 {
				return ArchiveInventory{}, &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("directory carries data")}
			}
			inventory.DirectoryCount++
			inventory.Entries = append(inventory.Entries, ArchiveEntry{Ordinal: ordinal + 1, Kind: ArchiveDirectory, Path: path})
			continue
		}
		seen[collisionKey] = ArchivePDF
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return ArchiveInventory{}, &ArchiveError{Code: "UNSUPPORTED_ENTRY_TYPE", Err: errors.New("only PDF files are allowed")}
		}
		if file.CompressedSize64 > uint64(policy.MaxPDFBytes) || file.UncompressedSize64 > uint64(policy.MaxPDFBytes) || inventory.TotalUncompressed+file.UncompressedSize64 > uint64(policy.MaxTotalUncompressedBytes) {
			return ArchiveInventory{}, &ArchiveError{Code: "ENTRY_SIZE_EXCEEDED", Err: ErrArchiveLimit}
		}
		if err := ValidateExpansionRatio(int64(file.CompressedSize64), int64(file.UncompressedSize64), policy); err != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "EXPANSION_RATIO_EXCEEDED", Err: err}
		}
		contentReader, err := file.Open()
		if err != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP_READ_FAILED", Err: err}
		}
		content, readErr := io.ReadAll(io.LimitReader(contentReader, policy.MaxPDFBytes+1))
		closeErr := contentReader.Close()
		if readErr != nil || closeErr != nil {
			return ArchiveInventory{}, &ArchiveError{Code: "ZIP_READ_FAILED", Err: firstError(readErr, closeErr)}
		}
		if int64(len(content)) > policy.MaxPDFBytes || len(content) < 5 || !bytes.Equal(content[:5], []byte("%PDF-")) {
			return ArchiveInventory{}, &ArchiveError{Code: "PDF_MAGIC_MISMATCH", Err: errors.New("entry is not a PDF")}
		}
		inventory.TotalUncompressed += uint64(len(content))
		inventory.PDFCount++
		inventory.Entries = append(inventory.Entries, ArchiveEntry{Ordinal: ordinal + 1, Kind: ArchivePDF, Path: path, SHA256: "sha256:" + hex.EncodeToString(sha256Sum(content)), CompressedBytes: file.CompressedSize64, UncompressedBytes: uint64(len(content)), CRC32: file.CRC32})
	}
	manifestBytes, err := json.Marshal(inventory.Entries)
	if err != nil {
		return ArchiveInventory{}, fmt.Errorf("marshal archive manifest: %w", err)
	}
	inventory.ManifestDigest = "sha256:" + hex.EncodeToString(sha256Sum(manifestBytes))
	return inventory, nil
}

func readAtExact(reader io.ReaderAt, offset int64, size int) ([]byte, error) {
	if offset < 0 || size < 0 {
		return nil, errors.New("invalid archive reader range")
	}
	data := make([]byte, size)
	read, err := reader.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	if read != size {
		return nil, io.ErrUnexpectedEOF
	}
	return data, nil
}

type strictZIPRange struct {
	start int
	end   int
}

const (
	zipLocalFileSignature       = uint32(0x04034b50)
	zipCentralFileSignature     = uint32(0x02014b50)
	zipEndOfCentralSignature    = uint32(0x06054b50)
	zip64EndOfCentralSignature  = uint32(0x06064b50)
	zip64LocatorSignature       = uint32(0x07064b50)
	zip64ExtraFieldID           = uint16(0x0001)
	zipLocalFileHeaderBytes     = 30
	zipCentralFileHeaderBytes   = 46
	zipEndOfCentralDirectoryMin = 22
)

func validateStrictZIPStructure(data []byte) error {
	eocd := findZIPEndOfCentral(data)
	if eocd < 0 {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("ZIP end-of-central-directory record is missing or has trailing data")}
	}
	diskNumber := binary.LittleEndian.Uint16(data[eocd+4 : eocd+6])
	centralDisk := binary.LittleEndian.Uint16(data[eocd+6 : eocd+8])
	diskEntries := binary.LittleEndian.Uint16(data[eocd+8 : eocd+10])
	totalEntries := binary.LittleEndian.Uint16(data[eocd+10 : eocd+12])
	centralSize := binary.LittleEndian.Uint32(data[eocd+12 : eocd+16])
	centralOffset := binary.LittleEndian.Uint32(data[eocd+16 : eocd+20])
	commentLength := int(binary.LittleEndian.Uint16(data[eocd+20 : eocd+22]))
	if diskNumber != 0 || centralDisk != 0 || diskEntries != totalEntries {
		return &ArchiveError{Code: "ZIP_MULTI_DISK_UNSUPPORTED", Err: errors.New("multi-disk ZIP is not supported")}
	}
	if diskEntries == 0xffff || totalEntries == 0xffff || centralSize == math.MaxUint32 || centralOffset == math.MaxUint32 || commentLength < 0 || eocd+zipEndOfCentralDirectoryMin+commentLength != len(data) {
		return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("ZIP64 sentinel or trailing ZIP data is not supported")}
	}
	centralStart := int(centralOffset)
	centralEnd := centralStart + int(centralSize)
	if centralStart > len(data) || centralEnd > len(data) {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory range is outside the archive")}
	}
	if centralEnd <= eocd && (bytes.Contains(data[centralEnd:eocd], uint32Bytes(zip64EndOfCentralSignature)) || bytes.Contains(data[centralEnd:eocd], uint32Bytes(zip64LocatorSignature))) {
		return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("ZIP64 end records are not supported")}
	}
	if centralStart < 0 || centralEnd < centralStart || centralEnd > eocd || centralEnd != eocd {
		return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("central directory has an unaccounted gap or trailing data")}
	}
	if totalEntries == 0 {
		if centralStart != 0 {
			return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("empty ZIP has unaccounted bytes")}
		}
		return nil
	}

	ranges := make([]strictZIPRange, 0, totalEntries)
	centralCursor := centralStart
	for ordinal := 0; ordinal < int(totalEntries); ordinal++ {
		if centralCursor+zipCentralFileHeaderBytes > centralEnd || binary.LittleEndian.Uint32(data[centralCursor:centralCursor+4]) != zipCentralFileSignature {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory header is malformed")}
		}
		centralVersionNeeded := binary.LittleEndian.Uint16(data[centralCursor+6 : centralCursor+8])
		centralFlags := binary.LittleEndian.Uint16(data[centralCursor+8 : centralCursor+10])
		centralMethod := binary.LittleEndian.Uint16(data[centralCursor+10 : centralCursor+12])
		centralCRC := binary.LittleEndian.Uint32(data[centralCursor+16 : centralCursor+20])
		centralCompressed := binary.LittleEndian.Uint32(data[centralCursor+20 : centralCursor+24])
		centralUncompressed := binary.LittleEndian.Uint32(data[centralCursor+24 : centralCursor+28])
		nameLength := int(binary.LittleEndian.Uint16(data[centralCursor+28 : centralCursor+30]))
		extraLength := int(binary.LittleEndian.Uint16(data[centralCursor+30 : centralCursor+32]))
		commentLength := int(binary.LittleEndian.Uint16(data[centralCursor+32 : centralCursor+34]))
		diskStart := binary.LittleEndian.Uint16(data[centralCursor+34 : centralCursor+36])
		localOffset := binary.LittleEndian.Uint32(data[centralCursor+42 : centralCursor+46])
		centralRecordEnd := centralCursor + zipCentralFileHeaderBytes + nameLength + extraLength + commentLength
		if centralRecordEnd < centralCursor || centralRecordEnd > centralEnd {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory record exceeds its range")}
		}
		if centralVersionNeeded >= 45 || centralCompressed == math.MaxUint32 || centralUncompressed == math.MaxUint32 || localOffset == math.MaxUint32 || hasZIP64Extra(data[centralCursor+zipCentralFileHeaderBytes+nameLength:centralCursor+zipCentralFileHeaderBytes+nameLength+extraLength]) {
			return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("central directory uses ZIP64, data descriptors, or multiple disks")}
		}
		if centralFlags&0x8 != 0 {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: fmt.Errorf("central directory uses a data descriptor: flags=%#x", centralFlags)}
		}
		if diskStart != 0 {
			return &ArchiveError{Code: "ZIP_MULTI_DISK_UNSUPPORTED", Err: errors.New("central directory entry is on another disk")}
		}
		centralNameStart := centralCursor + zipCentralFileHeaderBytes
		centralName := data[centralNameStart : centralNameStart+nameLength]
		localStart := int(localOffset)
		if localStart < 0 || localStart+zipLocalFileHeaderBytes > centralStart || binary.LittleEndian.Uint32(data[localStart:localStart+4]) != zipLocalFileSignature {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header range is malformed")}
		}
		localVersionNeeded := binary.LittleEndian.Uint16(data[localStart+4 : localStart+6])
		localFlags := binary.LittleEndian.Uint16(data[localStart+6 : localStart+8])
		localMethod := binary.LittleEndian.Uint16(data[localStart+8 : localStart+10])
		localCRC := binary.LittleEndian.Uint32(data[localStart+14 : localStart+18])
		localCompressed := binary.LittleEndian.Uint32(data[localStart+18 : localStart+22])
		localUncompressed := binary.LittleEndian.Uint32(data[localStart+22 : localStart+26])
		localNameLength := int(binary.LittleEndian.Uint16(data[localStart+26 : localStart+28]))
		localExtraLength := int(binary.LittleEndian.Uint16(data[localStart+28 : localStart+30]))
		localNameStart := localStart + zipLocalFileHeaderBytes
		localDataStart := localNameStart + localNameLength + localExtraLength
		localDataEnd := localDataStart + int(localCompressed)
		if localDataStart < localNameStart || localDataEnd < localDataStart || localDataEnd > centralStart || localNameStart+localNameLength > centralStart || localNameStart+localNameLength+localExtraLength > centralStart {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header data range is malformed")}
		}
		if localVersionNeeded >= 45 || localCompressed == math.MaxUint32 || localUncompressed == math.MaxUint32 || hasZIP64Extra(data[localNameStart+localNameLength:localNameStart+localNameLength+localExtraLength]) {
			return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("local header uses ZIP64 or a data descriptor")}
		}
		if localFlags&0x8 != 0 {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header uses a data descriptor")}
		}
		if !bytes.Equal(centralName, data[localNameStart:localNameStart+localNameLength]) || centralFlags != localFlags || centralMethod != localMethod || centralCRC != localCRC || centralCompressed != localCompressed || centralUncompressed != localUncompressed {
			return &ArchiveError{Code: "ZIP_HEADER_MISMATCH", Err: errors.New("local and central ZIP headers disagree")}
		}
		ranges = append(ranges, strictZIPRange{start: localStart, end: localDataEnd})
		centralCursor = centralRecordEnd
	}
	if centralCursor != centralEnd {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory has unaccounted records")}
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	if ranges[0].start != 0 || ranges[len(ranges)-1].end != centralStart {
		return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("archive contains unaccounted bytes outside local entries")}
	}
	for i := 1; i < len(ranges); i++ {
		if ranges[i].start < ranges[i-1].end {
			return &ArchiveError{Code: "ZIP_RANGE_OVERLAP", Err: errors.New("local entry byte ranges overlap")}
		}
	}
	return nil
}

// validateStrictZIPStructureReaderAt is the ReaderAt equivalent of the
// byte-slice validator above. It deliberately reads only fixed headers and
// bounded variable fields, then checks that local entry ranges account for
// every byte before the central directory.
func validateStrictZIPStructureReaderAt(reader io.ReaderAt, archiveBytes int64, maxRecords int) error {
	eocd, err := findZIPEndOfCentralReaderAt(reader, archiveBytes)
	if err != nil || eocd < 0 {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("ZIP end-of-central-directory record is missing or has trailing data")}
	}
	eocdHeader, err := readAtExact(reader, eocd, zipEndOfCentralDirectoryMin)
	if err != nil || binary.LittleEndian.Uint32(eocdHeader[:4]) != zipEndOfCentralSignature {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("ZIP end-of-central-directory record is malformed")}
	}
	diskNumber := binary.LittleEndian.Uint16(eocdHeader[4:6])
	centralDisk := binary.LittleEndian.Uint16(eocdHeader[6:8])
	diskEntries := binary.LittleEndian.Uint16(eocdHeader[8:10])
	totalEntries := binary.LittleEndian.Uint16(eocdHeader[10:12])
	centralSize := binary.LittleEndian.Uint32(eocdHeader[12:16])
	centralOffset := binary.LittleEndian.Uint32(eocdHeader[16:20])
	commentLength := int64(binary.LittleEndian.Uint16(eocdHeader[20:22]))
	if diskNumber != 0 || centralDisk != 0 || diskEntries != totalEntries {
		return &ArchiveError{Code: "ZIP_MULTI_DISK_UNSUPPORTED", Err: errors.New("multi-disk ZIP is not supported")}
	}
	if diskEntries == 0xffff || totalEntries == 0xffff || centralSize == math.MaxUint32 || centralOffset == math.MaxUint32 || eocd+zipEndOfCentralDirectoryMin+commentLength != archiveBytes {
		return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("ZIP64 sentinel or trailing ZIP data is not supported")}
	}
	centralStart := int64(centralOffset)
	centralEnd := centralStart + int64(centralSize)
	if centralStart < 0 || centralEnd < centralStart || centralEnd > archiveBytes || centralEnd != eocd {
		return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("central directory has an unaccounted gap or trailing data")}
	}
	if totalEntries == 0 {
		if centralStart != 0 {
			return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("empty ZIP has unaccounted bytes")}
		}
		return nil
	}
	if maxRecords > 0 && int(totalEntries) > maxRecords {
		return &ArchiveError{Code: "ENTRY_COUNT_EXCEEDED", Err: ErrArchiveLimit}
	}

	type zipRange64 struct{ start, end int64 }
	ranges := make([]zipRange64, 0, int(totalEntries))
	centralCursor := centralStart
	for ordinal := 0; ordinal < int(totalEntries); ordinal++ {
		centralHeader, headerErr := readAtExact(reader, centralCursor, zipCentralFileHeaderBytes)
		if headerErr != nil || binary.LittleEndian.Uint32(centralHeader[:4]) != zipCentralFileSignature {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory header is malformed")}
		}
		centralVersionNeeded := binary.LittleEndian.Uint16(centralHeader[6:8])
		centralFlags := binary.LittleEndian.Uint16(centralHeader[8:10])
		centralMethod := binary.LittleEndian.Uint16(centralHeader[10:12])
		centralCRC := binary.LittleEndian.Uint32(centralHeader[16:20])
		centralCompressed := binary.LittleEndian.Uint32(centralHeader[20:24])
		centralUncompressed := binary.LittleEndian.Uint32(centralHeader[24:28])
		nameLength := int(binary.LittleEndian.Uint16(centralHeader[28:30]))
		extraLength := int(binary.LittleEndian.Uint16(centralHeader[30:32]))
		commentLength := int(binary.LittleEndian.Uint16(centralHeader[32:34]))
		diskStart := binary.LittleEndian.Uint16(centralHeader[34:36])
		recordLength := zipCentralFileHeaderBytes + nameLength + extraLength + commentLength
		if recordLength < zipCentralFileHeaderBytes || centralCursor+int64(recordLength) > centralEnd {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory record exceeds its range")}
		}
		variable, variableErr := readAtExact(reader, centralCursor+zipCentralFileHeaderBytes, nameLength+extraLength+commentLength)
		if variableErr != nil {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory variable fields are truncated")}
		}
		centralName := variable[:nameLength]
		centralExtra := variable[nameLength : nameLength+extraLength]
		if centralVersionNeeded >= 45 || centralCompressed == math.MaxUint32 || centralUncompressed == math.MaxUint32 || binary.LittleEndian.Uint32(centralHeader[42:46]) == math.MaxUint32 || hasZIP64Extra(centralExtra) {
			return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("central directory uses ZIP64")}
		}
		if centralFlags&0x8 != 0 {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: fmt.Errorf("central directory uses a data descriptor: flags=%#x", centralFlags)}
		}
		if diskStart != 0 {
			return &ArchiveError{Code: "ZIP_MULTI_DISK_UNSUPPORTED", Err: errors.New("central directory entry is on another disk")}
		}
		localStart := int64(binary.LittleEndian.Uint32(centralHeader[42:46]))
		if localStart < 0 || localStart+zipLocalFileHeaderBytes > centralStart {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header range is malformed")}
		}
		localHeader, localErr := readAtExact(reader, localStart, zipLocalFileHeaderBytes)
		if localErr != nil || binary.LittleEndian.Uint32(localHeader[:4]) != zipLocalFileSignature {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header range is malformed")}
		}
		localVersionNeeded := binary.LittleEndian.Uint16(localHeader[4:6])
		localFlags := binary.LittleEndian.Uint16(localHeader[6:8])
		localMethod := binary.LittleEndian.Uint16(localHeader[8:10])
		localCRC := binary.LittleEndian.Uint32(localHeader[14:18])
		localCompressed := binary.LittleEndian.Uint32(localHeader[18:22])
		localUncompressed := binary.LittleEndian.Uint32(localHeader[22:26])
		localNameLength := int(binary.LittleEndian.Uint16(localHeader[26:28]))
		localExtraLength := int(binary.LittleEndian.Uint16(localHeader[28:30]))
		localVariableLength := localNameLength + localExtraLength
		if localVariableLength < 0 || localStart+zipLocalFileHeaderBytes+int64(localVariableLength) > centralStart {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header data range is malformed")}
		}
		localVariable, localVariableErr := readAtExact(reader, localStart+zipLocalFileHeaderBytes, localVariableLength)
		if localVariableErr != nil {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header variable fields are truncated")}
		}
		localName := localVariable[:localNameLength]
		localExtra := localVariable[localNameLength:]
		localDataStart := localStart + zipLocalFileHeaderBytes + int64(localVariableLength)
		localDataEnd := localDataStart + int64(localCompressed)
		if localDataEnd < localDataStart || localDataEnd > centralStart || localVersionNeeded >= 45 || localCompressed == math.MaxUint32 || localUncompressed == math.MaxUint32 || hasZIP64Extra(localExtra) {
			return &ArchiveError{Code: "ZIP64_UNSUPPORTED", Err: errors.New("local header uses ZIP64")}
		}
		if localFlags&0x8 != 0 {
			return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("local header uses a data descriptor")}
		}
		if !bytes.Equal(centralName, localName) || centralFlags != localFlags || centralMethod != localMethod || centralCRC != localCRC || centralCompressed != localCompressed || centralUncompressed != localUncompressed {
			return &ArchiveError{Code: "ZIP_HEADER_MISMATCH", Err: errors.New("local and central ZIP headers disagree")}
		}
		ranges = append(ranges, zipRange64{start: localStart, end: localDataEnd})
		centralCursor += int64(recordLength)
	}
	if centralCursor != centralEnd {
		return &ArchiveError{Code: "ZIP_STRUCTURE_MISMATCH", Err: errors.New("central directory has unaccounted records")}
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	if len(ranges) == 0 || ranges[0].start != 0 || ranges[len(ranges)-1].end != centralStart {
		return &ArchiveError{Code: "ZIP_TRAILING_DATA", Err: errors.New("archive contains unaccounted bytes outside local entries")}
	}
	for i := 1; i < len(ranges); i++ {
		if ranges[i].start != ranges[i-1].end {
			return &ArchiveError{Code: "ZIP_RANGE_OVERLAP", Err: errors.New("local entry byte ranges overlap or leave a gap")}
		}
	}
	return nil
}

func findZIPEndOfCentralReaderAt(reader io.ReaderAt, archiveBytes int64) (int64, error) {
	if archiveBytes < zipEndOfCentralDirectoryMin {
		return -1, io.ErrUnexpectedEOF
	}
	windowBytes := int64(65557)
	if archiveBytes < windowBytes {
		windowBytes = archiveBytes
	}
	window, err := readAtExact(reader, archiveBytes-windowBytes, int(windowBytes))
	if err != nil {
		return -1, err
	}
	for offset := len(window) - zipEndOfCentralDirectoryMin; offset >= 0; offset-- {
		if binary.LittleEndian.Uint32(window[offset:offset+4]) != zipEndOfCentralSignature {
			continue
		}
		commentLength := int(binary.LittleEndian.Uint16(window[offset+20 : offset+22]))
		if offset+zipEndOfCentralDirectoryMin+commentLength == len(window) {
			return archiveBytes - windowBytes + int64(offset), nil
		}
	}
	return -1, nil
}

func findZIPEndOfCentral(data []byte) int {
	minimum := len(data) - 65557
	if minimum < 0 {
		minimum = 0
	}
	for offset := len(data) - zipEndOfCentralDirectoryMin; offset >= minimum; offset-- {
		if offset >= 0 && binary.LittleEndian.Uint32(data[offset:offset+4]) == zipEndOfCentralSignature {
			commentLength := int(binary.LittleEndian.Uint16(data[offset+20 : offset+22]))
			if offset+zipEndOfCentralDirectoryMin+commentLength == len(data) {
				return offset
			}
		}
	}
	return -1
}

func hasZIP64Extra(extra []byte) bool {
	for offset := 0; offset < len(extra); {
		if offset+4 > len(extra) {
			return true
		}
		fieldID := binary.LittleEndian.Uint16(extra[offset : offset+2])
		fieldLength := int(binary.LittleEndian.Uint16(extra[offset+2 : offset+4]))
		offset += 4
		if offset+fieldLength > len(extra) {
			return true
		}
		if fieldID == zip64ExtraFieldID {
			return true
		}
		offset += fieldLength
	}
	return false
}

func uint32Bytes(value uint32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, value)
	return bytes
}

func parentPath(path string) string {
	index := strings.LastIndexByte(path, '/')
	if index < 0 {
		return ""
	}
	return path[:index]
}

func sha256Sum(value []byte) []byte {
	digest := sha256.Sum256(value)
	return digest[:]
}

func firstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
