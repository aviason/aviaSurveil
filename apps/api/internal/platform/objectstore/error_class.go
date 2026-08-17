package objectstore

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"

	"github.com/minio/minio-go/v7"
)

// ErrorClass returns a bounded operational class for object-store failures.
// It deliberately excludes messages, URLs, bucket names, object keys, and
// credentials so callers can use it in diagnostics without leaking data.
func ErrorClass(err error) string {
	if err == nil {
		return "none"
	}

	var response minio.ErrorResponse
	if errors.As(err, &response) {
		if code := safeErrorToken(response.Code); code != "" {
			return "s3:" + code
		}
		if response.StatusCode > 0 {
			return fmt.Sprintf("s3:http-%d", response.StatusCode)
		}
		return "s3:response"
	}

	var urlError *url.Error
	if errors.As(err, &urlError) {
		if errors.Is(urlError.Err, syscall.ECONNREFUSED) {
			return "http:connection-refused"
		}
		if errors.Is(urlError.Err, syscall.ECONNRESET) {
			return "http:connection-reset"
		}
		if errors.Is(urlError.Err, syscall.EHOSTUNREACH) || errors.Is(urlError.Err, syscall.ENETUNREACH) {
			return "http:network-unreachable"
		}
		if errors.Is(urlError.Err, context.DeadlineExceeded) {
			return "http:timeout"
		}
		var networkError net.Error
		if errors.As(urlError.Err, &networkError) && networkError.Timeout() {
			return "http:timeout"
		}
		return "http:url-error"
	}
	return "type:" + strings.TrimPrefix(fmt.Sprintf("%T", err), "*")
}

func safeErrorToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') &&
			character != '-' && character != '_' && character != '.' {
			return "response"
		}
	}
	return value
}
