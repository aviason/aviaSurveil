package messagews

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"kindred_server/internal/auth"
	"kindred_server/internal/request"
	"kindred_server/internal/testutil"
	"kindred_server/internal/user"
	"kindred_server/pkg/security"
)

func TestHandlerConnectStoresAuthenticatedConnection(t *testing.T) {
	authService, tokenFor := newTestAuth(t)
	userID, token := tokenFor(t, "u1", "alice@example.com")
	store := newFakeConnectionStore()
	h := NewHandler(authService, fakeRequestLookup{}, store, nil)

	resp, err := h.Handle(context.Background(), wsEvent("$connect", "conn-1", "", token))
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %q", resp.StatusCode, resp.Body)
	}
	conn := store.connections["conn-1"]
	if conn.UserID != userID {
		t.Fatalf("connection user = %q, want %q", conn.UserID, userID)
	}
	if conn.ExpiresAt == 0 {
		t.Fatalf("connection ttl not set")
	}
}

func TestHandlerSubscribeRequiresRequestParticipant(t *testing.T) {
	authService, tokenFor := newTestAuth(t)
	ownerID, ownerToken := tokenFor(t, "owner", "owner@example.com")
	requesterID, requesterToken := tokenFor(t, "requester", "requester@example.com")
	_, strangerToken := tokenFor(t, "stranger", "stranger@example.com")
	reqs := fakeRequestLookup{requests: map[string]request.Request{
		"r1": {ID: "r1", OwnerID: ownerID, RequesterID: requesterID},
	}}
	store := newFakeConnectionStore()
	h := NewHandler(authService, reqs, store, nil)

	connect(t, h, "owner-conn", ownerToken)
	resp, err := h.Handle(context.Background(), wsEvent("subscribe", "owner-conn", `{"type":"subscribe","requestId":"r1"}`, ""))
	if err != nil {
		t.Fatalf("Handle owner subscribe error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner status = %d body = %q", resp.StatusCode, resp.Body)
	}
	if _, ok := store.subscriptions["owner-conn|r1"]; !ok {
		t.Fatalf("owner subscription was not saved")
	}

	connect(t, h, "requester-conn", requesterToken)
	resp, err = h.Handle(context.Background(), wsEvent("subscribe", "requester-conn", `{"type":"subscribe","requestId":"r1"}`, ""))
	if err != nil {
		t.Fatalf("Handle requester subscribe error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("requester status = %d body = %q", resp.StatusCode, resp.Body)
	}

	connect(t, h, "stranger-conn", strangerToken)
	resp, err = h.Handle(context.Background(), wsEvent("subscribe", "stranger-conn", `{"type":"subscribe","requestId":"r1"}`, ""))
	if err != nil {
		t.Fatalf("Handle stranger subscribe error = %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger status = %d body = %q", resp.StatusCode, resp.Body)
	}
	if _, ok := store.subscriptions["stranger-conn|r1"]; ok {
		t.Fatalf("stranger subscription was saved")
	}
}

func TestHandlerDisconnectDeletesConnection(t *testing.T) {
	store := newFakeConnectionStore()
	store.connections["conn-1"] = Connection{ID: "conn-1", UserID: "u1"}
	store.subscriptions["conn-1|r1"] = Subscription{ConnectionID: "conn-1", RequestID: "r1"}
	h := NewHandler(nil, fakeRequestLookup{}, store, nil)

	resp, err := h.Handle(context.Background(), wsEvent("$disconnect", "conn-1", "", ""))
	if err != nil {
		t.Fatalf("Handle error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %q", resp.StatusCode, resp.Body)
	}
	if _, ok := store.connections["conn-1"]; ok {
		t.Fatalf("connection still exists")
	}
	if _, ok := store.subscriptions["conn-1|r1"]; ok {
		t.Fatalf("subscription still exists")
	}
}

func TestHandlerTypingRequiresParticipantAndPublishesAuthenticatedSender(t *testing.T) {
	authService, tokenFor := newTestAuth(t)
	ownerID, ownerToken := tokenFor(t, "owner", "owner@example.com")
	requesterID, _ := tokenFor(t, "requester", "requester@example.com")
	_, strangerToken := tokenFor(t, "stranger", "stranger@example.com")
	reqs := fakeRequestLookup{requests: map[string]request.Request{
		"r1": {ID: "r1", OwnerID: ownerID, RequesterID: requesterID},
	}}
	store := newFakeConnectionStore()
	publisher := &fakeTypingPublisher{}
	h := NewHandler(authService, reqs, store, nil, WithTypingPublisher(publisher))

	connect(t, h, "owner-conn", ownerToken)
	resp, err := h.Handle(context.Background(), wsEvent("typing", "owner-conn", `{"type":"typing","requestId":"r1","isTyping":true,"senderId":"spoofed"}`, ""))
	if err != nil {
		t.Fatalf("Handle typing error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("typing status = %d body = %q", resp.StatusCode, resp.Body)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %+v", publisher.events)
	}
	if publisher.events[0].SenderID != ownerID || publisher.events[0].RequestID != "r1" || !publisher.events[0].IsTyping {
		t.Fatalf("typing event = %+v", publisher.events[0])
	}

	connect(t, h, "stranger-conn", strangerToken)
	resp, err = h.Handle(context.Background(), wsEvent("typing", "stranger-conn", `{"type":"typing","requestId":"r1","isTyping":true}`, ""))
	if err != nil {
		t.Fatalf("Handle stranger typing error = %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger status = %d body = %q", resp.StatusCode, resp.Body)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("stranger typing was published: %+v", publisher.events)
	}
}

func TestHandlerTypingThrottlesFastTrueRefreshes(t *testing.T) {
	authService, tokenFor := newTestAuth(t)
	ownerID, ownerToken := tokenFor(t, "owner", "owner@example.com")
	requesterID, _ := tokenFor(t, "requester", "requester@example.com")
	reqs := fakeRequestLookup{requests: map[string]request.Request{
		"r1": {ID: "r1", OwnerID: ownerID, RequesterID: requesterID},
	}}
	store := newFakeConnectionStore()
	publisher := &fakeTypingPublisher{}
	h := NewHandler(authService, reqs, store, nil, WithTypingPublisher(publisher))
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return now }

	connect(t, h, "owner-conn", ownerToken)
	for i := 0; i < 2; i++ {
		resp, err := h.Handle(context.Background(), wsEvent("$default", "owner-conn", `{"type":"typing","requestId":"r1","isTyping":true}`, ""))
		if err != nil {
			t.Fatalf("Handle typing %d error = %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("typing %d status = %d", i, resp.StatusCode)
		}
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %+v, want one throttled event", publisher.events)
	}

	now = now.Add(time.Second)
	resp, err := h.Handle(context.Background(), wsEvent("$default", "owner-conn", `{"type":"typing","requestId":"r1","isTyping":true}`, ""))
	if err != nil {
		t.Fatalf("Handle typing after interval error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("typing after interval status = %d", resp.StatusCode)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("published events = %+v, want second event after interval", publisher.events)
	}
}

func connect(t *testing.T, h *Handler, connectionID, token string) {
	t.Helper()
	resp, err := h.Handle(context.Background(), wsEvent("$connect", connectionID, "", token))
	if err != nil {
		t.Fatalf("connect error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect status = %d body = %q", resp.StatusCode, resp.Body)
	}
}

func newTestAuth(t *testing.T) (*auth.Service, func(*testing.T, string, string) (string, string)) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	const (
		issuer   = "https://kindred-ws-test.local"
		audience = "kindred-mobile"
	)
	kid := security.JWKThumbprintKID(&priv.PublicKey)
	signer := auth.NewSigner(priv, issuer, audience, kid)
	verifier := auth.NewVerifier(&priv.PublicKey, issuer, audience)
	users := testutil.NewFakeUserRepo()
	service := auth.NewService(users, nil, nil, auth.Config{TokenTTL: time.Hour}, signer, verifier)
	tokenFor := func(t *testing.T, userID, email string) (string, string) {
		t.Helper()
		now := time.Now().UTC()
		if err := users.Create(context.Background(), user.User{
			ID:           userID,
			Email:        email,
			DisplayName:  userID,
			CreatedAt:    now,
			UpdatedAt:    now,
			TokenVersion: 0,
		}); err != nil {
			t.Fatalf("create user: %v", err)
		}
		token, _, err := signer.Sign(userID, email, "", 0, time.Hour)
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return userID, token
	}
	return service, tokenFor
}

func wsEvent(routeKey, connectionID, body, token string) events.APIGatewayWebsocketProxyRequest {
	headers := map[string]string{}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return events.APIGatewayWebsocketProxyRequest{
		Headers: headers,
		Body:    body,
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			RouteKey:     routeKey,
			ConnectionID: connectionID,
		},
	}
}

type fakeRequestLookup struct {
	requests map[string]request.Request
	err      error
}

func (l fakeRequestLookup) GetByID(_ context.Context, id string) (request.Request, error) {
	if l.err != nil {
		return request.Request{}, l.err
	}
	req, ok := l.requests[id]
	if !ok {
		return request.Request{}, request.ErrNotFound
	}
	return req, nil
}

type fakeConnectionStore struct {
	connections   map[string]Connection
	subscriptions map[string]Subscription
}

func newFakeConnectionStore() *fakeConnectionStore {
	return &fakeConnectionStore{
		connections:   map[string]Connection{},
		subscriptions: map[string]Subscription{},
	}
}

func (s *fakeConnectionStore) SaveConnection(_ context.Context, conn Connection) error {
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now().UTC()
	}
	s.connections[conn.ID] = conn
	return nil
}

func (s *fakeConnectionStore) GetConnection(_ context.Context, connectionID string) (Connection, error) {
	conn, ok := s.connections[connectionID]
	if !ok {
		return Connection{}, ErrConnectionNotFound
	}
	return conn, nil
}

func (s *fakeConnectionStore) DeleteConnection(_ context.Context, connectionID string) error {
	delete(s.connections, connectionID)
	for key, sub := range s.subscriptions {
		if sub.ConnectionID == connectionID {
			delete(s.subscriptions, key)
		}
	}
	return nil
}

func (s *fakeConnectionStore) SaveSubscription(_ context.Context, sub Subscription) error {
	s.subscriptions[sub.ConnectionID+"|"+sub.RequestID] = sub
	return nil
}

type fakeTypingPublisher struct {
	events []TypingEvent
}

func (p *fakeTypingPublisher) PublishTyping(_ context.Context, event TypingEvent) error {
	p.events = append(p.events, event)
	return nil
}
