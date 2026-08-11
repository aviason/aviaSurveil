package analytics_test

import (
	"net/http"
	"testing"
	"time"

	"kindred_server/internal/analytics"
	"kindred_server/internal/testutil"
)

func TestConsentEndpointsAndEventIngest(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_, token := ts.RegisterAndLogin(t, "analytics@example.com", "fixture-secret", "Analyst")

	var consents analytics.ConsentStateResponse
	if s := ts.DoJSON(t, http.MethodPut, "/me/consents", token, map[string]any{
		"consents": map[string]bool{
			string(analytics.PurposeAnalytics): true,
			string(analytics.PurposeHeatmap):   true,
		},
	}, &consents); s != http.StatusOK {
		t.Fatalf("put consents status = %d", s)
	}
	if !consents.Consents[analytics.PurposeAnalytics].Granted {
		t.Fatalf("analytics consent not granted: %+v", consents)
	}

	var current analytics.ConsentStateResponse
	if s := ts.DoJSON(t, http.MethodGet, "/me/consents", token, nil, &current); s != http.StatusOK {
		t.Fatalf("get consents status = %d", s)
	}
	if current.Consents[analytics.PurposeHeatmap].Version != 1 {
		t.Fatalf("heatmap consent = %+v, want v1", current.Consents[analytics.PurposeHeatmap])
	}

	var out analytics.BatchResponse
	if s := ts.DoJSON(t, http.MethodPost, "/analytics/events", token, map[string]any{
		"events": []map[string]any{
			eventBody("018f05b1-bd66-7d2f-bb14-f0d393d1a101", "screen_viewed", []string{"analytics"}),
			eventBody("018f05b1-bd66-7d2f-bb14-f0d393d1a102", "location_observed", []string{"analytics", "precise_location"}),
		},
	}, &out); s != http.StatusOK {
		t.Fatalf("ingest status = %d", s)
	}
	if out.Accepted != 1 || out.Rejected != 1 {
		t.Fatalf("ingest response = %+v, want one missing-consent rejection", out)
	}
	published := ts.Analytics.Published()
	if len(published) != 1 || published[0].UserID == "" || published[0].UserID == "client-user" {
		t.Fatalf("published = %#v", published)
	}
}

func eventBody(id, name string, purposes []string) map[string]any {
	return map[string]any{
		"eventId":       id,
		"eventName":     name,
		"schemaVersion": 1,
		"eventTime":     time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"sessionId":     "018f05b1-bd66-7d2f-bb14-f0d393d1a103",
		"anonymousId":   "install-id",
		"source":        "ios",
		"appVersion":    "1.0.0",
		"screen":        "home",
		"purposes":      purposes,
		"userId":        "client-user",
		"properties": map[string]any{
			"itemId": "item-1",
		},
	}
}
