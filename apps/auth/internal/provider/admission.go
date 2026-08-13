package provider

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

var ErrProviderRateLimited = errors.New("provider authorization admission denied")

// AdmissionPolicy contains only bounded, operation-specific provider policy.
// The values are deliberately explicit so a deployment cannot silently fall
// back to a shared socket-peer quota.
type AdmissionPolicy struct {
	Window                    time.Duration
	GlobalLimit               int
	ClientLimit               int
	BrowserLimit              int
	RequestLimit              int
	OutstandingLimit          int
	AnonymousOutstandingLimit int
}

func DefaultAdmissionPolicy() AdmissionPolicy {
	return AdmissionPolicy{
		Window:                    time.Minute,
		GlobalLimit:               600,
		ClientLimit:               120,
		BrowserLimit:              60,
		RequestLimit:              20,
		OutstandingLimit:          100,
		AnonymousOutstandingLimit: 20,
	}
}

func normalizeAdmissionPolicy(policy AdmissionPolicy) (AdmissionPolicy, error) {
	if policy.Window <= 0 {
		policy = DefaultAdmissionPolicy()
	}
	if policy.AnonymousOutstandingLimit <= 0 {
		policy.AnonymousOutstandingLimit = DefaultAdmissionPolicy().AnonymousOutstandingLimit
	}
	if policy.GlobalLimit < 1 || policy.ClientLimit < 1 || policy.BrowserLimit < 1 || policy.RequestLimit < 1 || policy.OutstandingLimit < 1 || policy.AnonymousOutstandingLimit < 1 {
		return AdmissionPolicy{}, ErrProviderInvalid
	}
	return policy, nil
}

type browserBindingContextKey struct{}

// WithBrowserBinding carries the server-issued browser binding to the storage
// boundary without allowing raw forwarding headers or gateway peer identity to
// enter provider admission.
func WithBrowserBinding(ctx context.Context, binding string) context.Context {
	return context.WithValue(ctx, browserBindingContextKey{}, strings.TrimSpace(binding))
}

func browserBindingFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(browserBindingContextKey{}).(string); ok && value != "" {
		return value
	}
	return "missing"
}

func providerAdmissionRules(policy AdmissionPolicy, ctx context.Context, request *oidc.AuthRequest, userID string) []throttle.Rule {
	requestFingerprint := strings.Join([]string{
		request.ClientID, request.RedirectURI, request.State, request.Nonce,
		request.CodeChallenge, string(request.CodeChallengeMethod), userID,
	}, "\x00")
	return []throttle.Rule{
		{Key: throttle.Key("provider:global", "authorization-request"), Window: policy.Window, Limit: policy.GlobalLimit, Global: true},
		{Key: throttle.Key("provider:client", request.ClientID), Window: policy.Window, Limit: policy.ClientLimit},
		{Key: throttle.Key("provider:browser", browserBindingFromContext(ctx)), Window: policy.Window, Limit: policy.BrowserLimit},
		{Key: throttle.Key("provider:request", requestFingerprint), Window: policy.Window, Limit: policy.RequestLimit},
	}
}

func admitProviderRequest(ctx context.Context, limiter throttle.Limiter, policy AdmissionPolicy, request *oidc.AuthRequest, userID string) error {
	if limiter == nil {
		return ErrProviderUnavailable
	}
	if err := limiter.Allow(ctx, providerAdmissionRules(policy, ctx, request, userID)...); err != nil {
		if errors.Is(err, throttle.ErrRateLimited) {
			return ErrProviderRateLimited
		}
		return ErrProviderUnavailable
	}
	return nil
}
