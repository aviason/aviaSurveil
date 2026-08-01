package checklistintake

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrUnsafeArchivePath = errors.New("unsafe archive path")
	ErrArchiveLimit      = errors.New("AGA_ZIP_PDF_V1 limit exceeded")
)

type IntakePolicy struct {
	Version                        string
	MaxArchiveBytes                int64
	MaxCentralDirectoryRecords     int
	MaxPathDepth                   int
	MaxPathBytes                   int
	MaxFilenameComponentBytes      int
	MaxPDFBytes                    int64
	MaxTotalUncompressedBytes      int64
	MaxExpansionRatio              int64
	MaxPDFPages                    int
	MaxBatchPDFPages               int
	MaxExtractedTextBytes          int64
	MaxBatchTextBytes              int64
	MaxParserOutputBytes           int64
	MaxProposalSpans               int
	MaxProposalTextBytes           int64
	MaxAggregateProposalBytes      int64
	MaxCandidateQuestionSeeds      int
	MaxTranscriptionBytes          int64
	MaxAggregateTranscriptionBytes int64
	MaxParserSeconds               int
	MaxBatchParserSeconds          int
}

func AGAZipPDFV1() IntakePolicy {
	return IntakePolicy{
		Version:                        PolicyAGAZipPDFV1,
		MaxArchiveBytes:                50 * 1024 * 1024,
		MaxCentralDirectoryRecords:     128,
		MaxPathDepth:                   4,
		MaxPathBytes:                   512,
		MaxFilenameComponentBytes:      255,
		MaxPDFBytes:                    20 * 1024 * 1024,
		MaxTotalUncompressedBytes:      100 * 1024 * 1024,
		MaxExpansionRatio:              20,
		MaxPDFPages:                    250,
		MaxBatchPDFPages:               2000,
		MaxExtractedTextBytes:          20 * 1024 * 1024,
		MaxBatchTextBytes:              100 * 1024 * 1024,
		MaxParserOutputBytes:           25 * 1024 * 1024,
		MaxProposalSpans:               1000,
		MaxProposalTextBytes:           64 * 1024,
		MaxAggregateProposalBytes:      8 * 1024 * 1024,
		MaxCandidateQuestionSeeds:      2000,
		MaxTranscriptionBytes:          32 * 1024,
		MaxAggregateTranscriptionBytes: 512 * 1024,
		MaxParserSeconds:               30,
		MaxBatchParserSeconds:          5 * 60,
	}
}

func NormalizeZipPathV1(raw string, directory bool) (string, error) {
	if raw == "" || strings.ContainsRune(raw, '\\') || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || (len(raw) >= 2 && ((raw[0] >= 'A' && raw[0] <= 'Z') || (raw[0] >= 'a' && raw[0] <= 'z')) && raw[1] == ':') {
		return "", ErrUnsafeArchivePath
	}
	if directory {
		if !strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, "//") {
			return "", ErrUnsafeArchivePath
		}
		raw = strings.TrimSuffix(raw, "/")
	} else if strings.HasSuffix(raw, "/") {
		return "", ErrUnsafeArchivePath
	}
	if strings.ContainsRune(raw, '\x00') {
		return "", ErrUnsafeArchivePath
	}
	for _, r := range raw {
		if r == unicode.ReplacementChar || r <= 0x1f || (r >= 0x7f && r <= 0x9f) || r == 0x7f || isBidiControl(r) {
			return "", ErrUnsafeArchivePath
		}
	}
	components := strings.Split(raw, "/")
	if len(components) == 0 || len(components) > AGAZipPDFV1().MaxPathDepth {
		return "", ErrArchiveLimit
	}
	normalized := make([]string, len(components))
	for i, component := range components {
		if component == "" || component == "." || component == ".." {
			return "", ErrUnsafeArchivePath
		}
		normalized[i] = norm.NFC.String(component)
		if len([]byte(normalized[i])) > AGAZipPDFV1().MaxFilenameComponentBytes {
			return "", ErrArchiveLimit
		}
	}
	result := strings.Join(normalized, "/")
	if len([]byte(result)) > AGAZipPDFV1().MaxPathBytes {
		return "", ErrArchiveLimit
	}
	return result, nil
}

func isBidiControl(r rune) bool {
	return (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) || r == 0x200e || r == 0x200f
}

func ValidateExpansionRatio(compressed, uncompressed int64, policy IntakePolicy) error {
	if compressed < 0 || uncompressed < 0 || compressed > policy.MaxPDFBytes || uncompressed > policy.MaxPDFBytes {
		return ErrArchiveLimit
	}
	if compressed == 0 {
		if uncompressed != 0 {
			return fmt.Errorf("%w: zero compressed bytes", ErrArchiveLimit)
		}
		return nil
	}
	if uncompressed > compressed*policy.MaxExpansionRatio {
		return ErrArchiveLimit
	}
	return nil
}
