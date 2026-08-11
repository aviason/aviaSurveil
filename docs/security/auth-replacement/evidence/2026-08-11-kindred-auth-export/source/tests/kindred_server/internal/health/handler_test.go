package health_test

import (
	"net/http"
	"testing"

	"kindred_server/internal/testutil"
)

func TestHealthOK(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp, _ := ts.Do(t, http.MethodGet, "/health", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
