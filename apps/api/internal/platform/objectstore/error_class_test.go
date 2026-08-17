package objectstore

import (
	"fmt"
	"net"
	"net/url"
	"syscall"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestErrorClassBoundsS3ResponseWithoutLeakingDetails(t *testing.T) {
	err := fmt.Errorf("presign failed: %w", minio.ErrorResponse{
		Code:       "AccessDenied",
		Message:    "do-not-log-this-message",
		BucketName: "do-not-log-this-bucket",
		Key:        "do-not-log-this-key",
	})

	if got := ErrorClass(err); got != "s3:AccessDenied" {
		t.Fatalf("ErrorClass() = %q, want bounded S3 class", got)
	}
	if got := ErrorClass(err); got == "do-not-log-this-message" || got == "do-not-log-this-key" {
		t.Fatalf("ErrorClass() leaked error details: %q", got)
	}
}

func TestErrorClassBoundsUnknownErrorType(t *testing.T) {
	if got := ErrorClass(fmt.Errorf("opaque failure")); got != "type:errors.errorString" {
		t.Fatalf("ErrorClass() = %q, want opaque error type", got)
	}
}

func TestErrorClassBoundsCredentialEndpointNetworkFailure(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "http://credential-endpoint.invalid/v2/credentials",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
	}
	if got := ErrorClass(err); got != "http:connection-refused" {
		t.Fatalf("ErrorClass() = %q, want bounded connection class", got)
	}
}
