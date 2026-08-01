package checklistintake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type RegisterRow struct {
	Page      int
	RowNumber int
	FormCode  string
	Title     string
	Version   string
	Status    string
}

type RegisterParseResult struct {
	Rows         []RegisterRow
	ResultDigest string
}

func ParseRegister(rows []RegisterRow) (RegisterParseResult, error) {
	if len(rows) == 0 {
		return RegisterParseResult{}, errors.New("register contains no rows")
	}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.Page <= 0 || row.RowNumber <= 0 || strings.TrimSpace(row.FormCode) == "" {
			return RegisterParseResult{}, errors.New("register row is incomplete")
		}
		if _, exists := seen[row.FormCode]; exists {
			return RegisterParseResult{}, fmt.Errorf("duplicate register form code %q", row.FormCode)
		}
		seen[row.FormCode] = struct{}{}
	}
	canonical, err := json.Marshal(rows)
	if err != nil {
		return RegisterParseResult{}, err
	}
	digest := sha256.Sum256(canonical)
	return RegisterParseResult{Rows: append([]RegisterRow(nil), rows...), ResultDigest: "sha256:" + hex.EncodeToString(digest[:])}, nil
}

// MatchRegister is the only function that can derive initial identity-match
// facts. It requires the successful REGISTER_PARSE receipt and never mutates
// the immutable file facts supplied by the caller.
func MatchRegister(receipt PhaseReceipt, rows []RegisterRow, files []ImportFile) ([]RegisterEntry, error) {
	if receipt.Phase != PhaseRegisterParse || receipt.Outcome != ReceiptSucceeded || strings.TrimSpace(receipt.ResultDigest) == "" {
		return nil, errors.New("identity matching requires successful register parse")
	}
	parsed, err := ParseRegister(rows)
	if err != nil {
		return nil, err
	}
	if parsed.ResultDigest != receipt.ResultDigest {
		return nil, errors.New("register digest does not match terminal receipt")
	}
	byCode := make(map[string]ImportFile, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.RegisterFormCode) != "" {
			if _, exists := byCode[file.RegisterFormCode]; exists {
				return nil, fmt.Errorf("duplicate file form code %q", file.RegisterFormCode)
			}
			byCode[file.RegisterFormCode] = file
		}
	}
	matched := make([]RegisterEntry, 0, len(rows))
	matchedFiles := make(map[string]struct{}, len(rows))
	for ordinal, row := range rows {
		file, exists := byCode[row.FormCode]
		if !exists {
			return nil, fmt.Errorf("register form %q is unmatched", row.FormCode)
		}
		matchedFiles[file.ImportFileID] = struct{}{}
		matched = append(matched, RegisterEntry{Ordinal: ordinal + 1, Page: row.Page, RowNumber: row.RowNumber, FormCode: row.FormCode, TitleText: row.Title, VersionText: row.Version, StatusText: row.Status, MatchedImportFileID: file.ImportFileID, MatchState: "MATCHED"})
	}
	for _, file := range files {
		if file.RegisterFormCode == "" {
			continue
		}
		if _, exists := matchedFiles[file.ImportFileID]; !exists {
			return nil, fmt.Errorf("file %q is extra", file.ImportFileID)
		}
	}
	return matched, nil
}
