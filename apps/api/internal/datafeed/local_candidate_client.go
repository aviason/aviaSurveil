package datafeed

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// LocalCandidateClient is the explicitly named Workspace-dev transport. It
// exists only for the isolated local candidate and never replaces the direct
// mTLS client used by a released AviaData endpoint.
type LocalCandidateClient struct {
	endpoint string
	http     *http.Client
}

func NewLocalCandidateClient(endpoint string) (*LocalCandidateClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "/v3/aviasurveil/event-batches" {
		return nil, fmt.Errorf("local AviaData candidate requires an internal http event endpoint")
	}
	return &LocalCandidateClient{
		endpoint: parsed.String(),
		http: &http.Client{
			Transport: &http.Transport{Proxy: nil, DisableCompression: true, MaxConnsPerHost: 2},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return fmt.Errorf("local AviaData candidate rejects redirect responses")
			},
		},
	}, nil
}

func (client *LocalCandidateClient) Submit(ctx context.Context, request BatchRequest) (*http.Response, error) {
	if client == nil || client.http == nil || client.endpoint == "" {
		return nil, fmt.Errorf("local AviaData candidate client is not configured")
	}
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" ||
		strings.TrimSpace(request.ReplayID) == "" || len(request.Body) == 0 {
		return nil, fmt.Errorf("local AviaData candidate request is incomplete")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, strings.NewReader(string(request.Body)))
	if err != nil {
		return nil, fmt.Errorf("create local AviaData candidate request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", aviaCoreV3MediaType)
	httpRequest.Header.Set("X-Request-Id", request.RequestID)
	httpRequest.Header.Set("Idempotency-Key", request.IdempotencyKey)
	httpRequest.Header.Set("X-Replay-Id", request.ReplayID)
	return client.http.Do(httpRequest)
}
