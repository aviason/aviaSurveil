package preproddata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

type Generator struct {
	profile  profiles.Profile
	seedHash string
	seed     []byte
}

type ObjectMetadata struct {
	ObjectID      string `json:"objectId"`
	ContentDigest string `json:"contentDigest"`
	SizeBytes     int64  `json:"sizeBytes"`
	ContentType   string `json:"contentType"`
	Bytes         []byte `json:"-"`
}

func NewGenerator(profile profiles.Profile, seed []byte) (*Generator, error) {
	if err := profiles.ValidateFrozen(profile); err != nil {
		return nil, err
	}
	if profile.Name == "" || profile.Version == "" {
		return nil, fmt.Errorf("profile identity is required")
	}
	if profile.ResourceEnvelope.SeedRequired && len(seed) == 0 {
		return nil, fmt.Errorf("an explicit deterministic seed is required")
	}
	if profile.ResourceEnvelope.ClockOrigin.IsZero() {
		return nil, fmt.Errorf("profile clock origin is required")
	}
	digest := sha256.Sum256(seed)
	return &Generator{
		profile: profile, seed: append([]byte(nil), seed...),
		seedHash: "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func (generator *Generator) SeedHash() string {
	return generator.seedHash
}

func (generator *Generator) ID(family string, index int64) string {
	family = strings.ToLower(strings.TrimSpace(family))
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"%x\x00%s\x00%s\x00%s\x00%d",
		generator.seed,
		generator.profile.Name,
		generator.profile.Version,
		family,
		index,
	)))
	return fmt.Sprintf("synthetic-%s-%s", family, hex.EncodeToString(digest[:12]))
}

func (generator *Generator) Instant(sequence int64) time.Time {
	return generator.profile.ResourceEnvelope.ClockOrigin.
		Add(time.Duration(sequence) * time.Second).
		UTC()
}

func (generator *Generator) SyntheticEmail(kind string, index int64) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	return fmt.Sprintf("%s-%04d@synthetic.invalid", kind, index)
}

func (generator *Generator) Text(kind string, index int64) string {
	return fmt.Sprintf(
		"SYNTHETIC %s RECORD %04d — GENERATED FROM APPROVED PROFILE %s@%s",
		strings.ToUpper(strings.TrimSpace(kind)),
		index,
		generator.profile.Name,
		generator.profile.Version,
	)
}

func (generator *Generator) ObjectMetadata(
	recordType string,
	index, sizeBytes int64,
) ObjectMetadata {
	objectID := generator.ID("object-"+recordType, index)
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%d\x00%d",
		generator.seedHash,
		objectID,
		index,
		sizeBytes,
	)))
	return ObjectMetadata{
		ObjectID: objectID, SizeBytes: sizeBytes,
		ContentDigest: "sha256:" + hex.EncodeToString(digest[:]),
		ContentType:   "application/x-aviasurveil360-synthetic-metadata",
		Bytes:         nil,
	}
}

func (generator *Generator) LifecycleState(
	states []string,
	index int64,
) (string, error) {
	if len(states) == 0 {
		return "", fmt.Errorf("at least one lifecycle state is required")
	}
	return states[index%int64(len(states))], nil
}

func (generator *Generator) RelationshipDigest(tuples [][]string) string {
	canonical := make([]string, len(tuples))
	for index, tuple := range tuples {
		canonical[index] = strings.Join(tuple, "\x1f")
	}
	sort.Strings(canonical)
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return "sha256:" + hex.EncodeToString(digest[:])
}
