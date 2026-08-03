package agacandidatedemo

import "fmt"

// BaseRunEvidence is a read-only predecessor receipt supplied by the existing
// preprod control store; the AGA overlay never creates or reconciles it.
type BaseRunEvidence struct {
	RunID                   string
	IntentDigest            string
	ResultDigest            string
	TargetFingerprintDigest string
	Outcome                 string
	Disposable              bool
}

func VerifyBaseEvidence(intent IntentManifest, evidence BaseRunEvidence) error {
	if err := intent.Validate(); err != nil {
		return err
	}
	if evidence.RunID != intent.BaseRunID || evidence.IntentDigest != intent.BaseIntentDigest || evidence.ResultDigest != intent.BaseResultDigest || evidence.TargetFingerprintDigest != intent.BaseTargetDigest || evidence.Outcome != "SUCCEEDED" || !evidence.Disposable {
		return fmt.Errorf("base preprod evidence does not exactly satisfy the overlay intent")
	}
	return nil
}
