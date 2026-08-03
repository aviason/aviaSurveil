package agacandidatedemo

import (
	"encoding/json"
	"fmt"
	"time"
)

type ResultManifest struct {
	SchemaVersion string    `json:"schemaVersion"`
	RunID         string    `json:"runId"`
	IntentDigest  string    `json:"intentDigest"`
	SealDigest    string    `json:"sealDigest"`
	CompletedAt   time.Time `json:"completedAt"`
	ResultDigest  string    `json:"resultDigest"`
}
type ResultInput struct {
	RunID        string
	IntentDigest string
	SealDigest   string
	CompletedAt  time.Time
}

func BuildResult(input ResultInput) (ResultManifest, error) {
	result := ResultManifest{SchemaVersion: "preprod-aga-candidate-demo-result/v1", RunID: input.RunID, IntentDigest: input.IntentDigest, SealDigest: input.SealDigest, CompletedAt: input.CompletedAt.UTC()}
	if !runIDPattern.MatchString(result.RunID) || !validDigest(result.IntentDigest) || !validDigest(result.SealDigest) || result.CompletedAt.IsZero() {
		return ResultManifest{}, fmt.Errorf("invalid AGA demo result")
	}
	payload, err := resultPayload(result)
	if err != nil {
		return ResultManifest{}, err
	}
	result.ResultDigest = digestBytes(payload)
	return result, result.Validate()
}
func (result ResultManifest) Validate() error {
	if result.SchemaVersion != "preprod-aga-candidate-demo-result/v1" || !runIDPattern.MatchString(result.RunID) || !validDigest(result.IntentDigest) || !validDigest(result.SealDigest) || !validDigest(result.ResultDigest) || result.CompletedAt.IsZero() {
		return fmt.Errorf("invalid AGA demo result")
	}
	payload, err := resultPayload(result)
	if err != nil || digestBytes(payload) != result.ResultDigest {
		return fmt.Errorf("AGA demo result digest mismatch")
	}
	return nil
}
func resultPayload(result ResultManifest) ([]byte, error) {
	result.ResultDigest = ""
	return json.Marshal(result)
}

type CleanupTombstone struct {
	SchemaVersion   string    `json:"schemaVersion"`
	RunID           string    `json:"runId"`
	IntentDigest    string    `json:"intentDigest"`
	ResultDigest    string    `json:"resultDigest"`
	CleanedAt       time.Time `json:"cleanedAt"`
	TombstoneDigest string    `json:"tombstoneDigest"`
}
type CleanupTombstoneInput struct {
	RunID        string
	IntentDigest string
	ResultDigest string
	CleanedAt    time.Time
}

func BuildCleanupTombstone(input CleanupTombstoneInput) (CleanupTombstone, error) {
	tombstone := CleanupTombstone{SchemaVersion: "preprod-aga-candidate-demo-cleanup-tombstone/v1", RunID: input.RunID, IntentDigest: input.IntentDigest, ResultDigest: input.ResultDigest, CleanedAt: input.CleanedAt.UTC()}
	if !runIDPattern.MatchString(tombstone.RunID) || !validDigest(tombstone.IntentDigest) || !validDigest(tombstone.ResultDigest) || tombstone.CleanedAt.IsZero() {
		return CleanupTombstone{}, fmt.Errorf("invalid AGA demo cleanup tombstone")
	}
	data, err := cleanupPayload(tombstone)
	if err != nil {
		return CleanupTombstone{}, err
	}
	tombstone.TombstoneDigest = digestBytes(data)
	return tombstone, tombstone.Validate()
}
func (tombstone CleanupTombstone) Validate() error {
	if tombstone.SchemaVersion != "preprod-aga-candidate-demo-cleanup-tombstone/v1" || !runIDPattern.MatchString(tombstone.RunID) || !validDigest(tombstone.IntentDigest) || !validDigest(tombstone.ResultDigest) || !validDigest(tombstone.TombstoneDigest) || tombstone.CleanedAt.IsZero() {
		return fmt.Errorf("invalid AGA demo cleanup tombstone")
	}
	data, err := cleanupPayload(tombstone)
	if err != nil || digestBytes(data) != tombstone.TombstoneDigest {
		return fmt.Errorf("AGA demo cleanup tombstone digest mismatch")
	}
	return nil
}
func cleanupPayload(tombstone CleanupTombstone) ([]byte, error) {
	tombstone.TombstoneDigest = ""
	return json.Marshal(tombstone)
}
