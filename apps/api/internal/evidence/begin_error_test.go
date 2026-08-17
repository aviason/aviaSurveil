package evidence

import (
	"errors"
	"testing"
)

func TestBeginUploadErrorKeepsCauseOutOfPublicMessage(t *testing.T) {
	cause := errors.New("database password and object key must not be exposed")
	err := &BeginUploadError{Stage: "upload-session", Cause: cause}
	if got := err.Error(); got != "begin Evidence upload failed at upload-session" {
		t.Fatalf("bounded error = %q", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("begin upload error did not preserve its cause")
	}
}
