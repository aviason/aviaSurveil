package checklistintake

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

type ParsedPDF struct {
	PageCount    int    `json:"pageCount"`
	Text         string `json:"text"`
	TextDigest   string `json:"textDigest"`
	OutputDigest string `json:"outputDigest"`
	OutputBytes  int64  `json:"outputBytes"`
}

func ParseBoundedPDF(data []byte, policy IntakePolicy) (ParsedPDF, error) {
	if int64(len(data)) > policy.MaxPDFBytes || len(data) < 5 || !bytes.Equal(data[:5], []byte("%PDF-")) {
		return ParsedPDF{}, errors.New("PDF magic or byte limit failed")
	}
	if bytes.Contains(data, []byte("/Encrypt")) {
		return ParsedPDF{}, errors.New("encrypted PDF is not supported")
	}
	if !utf8.Valid(data) {
		return ParsedPDF{}, errors.New("PDF contains invalid UTF-8 for bounded parser")
	}
	pageCount := bytes.Count(data, []byte("/Type /Page"))
	if pageCount == 0 {
		pageCount = 1
	}
	if pageCount > policy.MaxPDFPages {
		return ParsedPDF{}, ErrArchiveLimit
	}
	text := extractLiteralText(data)
	if text == "" {
		text = strings.TrimSpace(string(bytes.TrimSpace(data[5:])))
	}
	if int64(len([]byte(text))) > policy.MaxExtractedTextBytes {
		return ParsedPDF{}, ErrArchiveLimit
	}
	textSum := sha256.Sum256([]byte(text))
	parsed := ParsedPDF{PageCount: pageCount, Text: text, TextDigest: "sha256:" + hex.EncodeToString(textSum[:])}
	canonical, err := json.Marshal(parsed)
	if err != nil {
		return ParsedPDF{}, err
	}
	if int64(len(canonical)) > policy.MaxParserOutputBytes {
		return ParsedPDF{}, ErrArchiveLimit
	}
	outputSum := sha256.Sum256(canonical)
	parsed.OutputDigest = "sha256:" + hex.EncodeToString(outputSum[:])
	parsed.OutputBytes = int64(len(canonical))
	return parsed, nil
}

func extractLiteralText(data []byte) string {
	var output strings.Builder
	for index := 0; index < len(data); index++ {
		if data[index] != '(' {
			continue
		}
		end := index + 1
		for end < len(data) && data[end] != ')' {
			if data[end] == '\\' && end+1 < len(data) {
				end += 2
				continue
			}
			end++
		}
		if end > index+1 && end <= len(data) {
			if output.Len() > 0 {
				output.WriteByte('\n')
			}
			output.Write(data[index+1 : end])
		}
		index = end
	}
	return strings.TrimSpace(output.String())
}
