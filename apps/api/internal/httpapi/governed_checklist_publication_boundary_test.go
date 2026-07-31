package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalCanonicalRouterDoesNotRegisterDirectChecklistPublication(t *testing.T) {
	// Production break: registering the direct-publication route would make an
	// Admin request reach a handler instead of the required normal 404 boundary.
	api := NewCanonicalAPI(CanonicalAPIDependencies{})
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/checklist-template-versions", nil)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("normal POST direct checklist publication status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
