package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

const GuardDutyMalwareScanStatusTag = "GuardDutyMalwareScanStatus"

type GuardDutyS3ResultProvider struct {
	store objectstore.ExactVersionStore
	clock func() time.Time
}

func NewGuardDutyS3ResultProvider(store objectstore.ExactVersionStore, clock func() time.Time) (*GuardDutyS3ResultProvider, error) {
	if store == nil {
		return nil, errors.New("GuardDuty S3 result provider requires an exact-version object store")
	}
	if clock == nil {
		clock = time.Now
	}
	return &GuardDutyS3ResultProvider{store: store, clock: clock}, nil
}

func (provider *GuardDutyS3ResultProvider) Resolve(ctx context.Context, expected objectstore.ExactObject) (Result, error) {
	tags, _, err := provider.store.ReadTagsExact(ctx, expected)
	if err != nil {
		return Result{}, errors.New("managed malware result exact-version validation failed")
	}
	status, exists := tags[GuardDutyMalwareScanStatusTag]
	if !exists || status == "" {
		return Result{}, errors.New("managed malware result is missing")
	}
	result := Result{
		EngineVersion:    "guardduty-s3-managed",
		SignatureVersion: status,
		ScannedAt:        provider.clock().UTC(),
	}
	switch status {
	case "NO_THREATS_FOUND":
		reader, _, openErr := provider.store.OpenExact(ctx, expected)
		if openErr != nil {
			return Result{}, errors.New("managed malware result exact-version validation failed")
		}
		hasher := sha256.New()
		readBytes, readErr := io.Copy(hasher, io.LimitReader(reader, expected.Size+1))
		closeErr := reader.Close()
		actualSHA256 := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
		if readErr != nil || closeErr != nil || readBytes != expected.Size || actualSHA256 != expected.SHA256 {
			return Result{}, errors.New("managed malware result exact-version content validation failed")
		}
		confirmedTags, _, confirmErr := provider.store.ReadTagsExact(ctx, expected)
		if confirmErr != nil || confirmedTags[GuardDutyMalwareScanStatusTag] != status {
			return Result{}, errors.New("managed malware result changed during exact-version validation")
		}
		result.Clean = true
		return result, nil
	case "THREATS_FOUND":
		result.Reason = "managed malware protection reported threats"
		return result, nil
	case "UNSUPPORTED", "ACCESS_DENIED", "FAILED":
		return Result{}, errors.New("managed malware result did not reach a terminal clean or threat decision")
	default:
		return Result{}, errors.New("managed malware result status is unrecognized")
	}
}

var _ ManagedResultProvider = (*GuardDutyS3ResultProvider)(nil)
