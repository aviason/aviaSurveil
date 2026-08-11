package messagews

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"kindred_server/internal/auth"
	"kindred_server/internal/request"
	"kindred_server/pkg/logger"
)

type RequestLookup interface {
	GetByID(ctx context.Context, id string) (request.Request, error)
}

type ConnectionStore interface {
	SaveConnection(ctx context.Context, conn Connection) error
	GetConnection(ctx context.Context, connectionID string) (Connection, error)
	DeleteConnection(ctx context.Context, connectionID string) error
	SaveSubscription(ctx context.Context, sub Subscription) error
}

type Handler struct {
	auth            *auth.Service
	requests        RequestLookup
	repo            ConnectionStore
	typingPublisher TypingPublisher
	logger          *logger.Logger
	now             func() time.Time
	typingMu        sync.Mutex
	lastTypingSent  map[string]time.Time
}

type subscribeMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
}

type typingMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	IsTyping  bool   `json:"isTyping"`
}

type messageTypeEnvelope struct {
	Type string `json:"type"`
}

type TypingPublisher interface {
	PublishTyping(ctx context.Context, event TypingEvent) error
}

type HandlerOption func(*Handler)

func WithTypingPublisher(publisher TypingPublisher) HandlerOption {
	return func(h *Handler) {
		h.typingPublisher = publisher
	}
}

func NewHandler(authService *auth.Service, requests RequestLookup, repo ConnectionStore, logger *logger.Logger, opts ...HandlerOption) *Handler {
	h := &Handler{
		auth:           authService,
		requests:       requests,
		repo:           repo,
		logger:         logger,
		now:            func() time.Time { return time.Now().UTC() },
		lastTypingSent: map[string]time.Time{},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Handler) Handle(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch event.RequestContext.RouteKey {
	case "$connect":
		return h.connect(ctx, event)
	case "$disconnect":
		return h.disconnect(ctx, event)
	case "subscribe":
		return h.subscribe(ctx, event)
	case "typing":
		return h.typing(ctx, event)
	case "$default":
		return h.routeDefault(ctx, event)
	default:
		return response(200, ""), nil
	}
}

func (h *Handler) connect(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	token := bearerToken(event.Headers)
	if token == "" {
		return response(401, "missing token"), nil
	}
	claims, err := h.auth.VerifyToken(ctx, token)
	if err != nil {
		return response(401, "invalid token"), nil
	}
	now := h.now()
	if err := h.repo.SaveConnection(ctx, Connection{
		ID:        event.RequestContext.ConnectionID,
		UserID:    claims.UserID,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour).Unix(),
	}); err != nil {
		if h.logger != nil {
			h.logger.Error("save websocket connection", err, map[string]any{"connectionId": event.RequestContext.ConnectionID})
		}
		return response(500, "connection failed"), nil
	}
	return response(200, ""), nil
}

func (h *Handler) disconnect(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	if err := h.repo.DeleteConnection(ctx, event.RequestContext.ConnectionID); err != nil && h.logger != nil {
		h.logger.Error("delete websocket connection", err, map[string]any{"connectionId": event.RequestContext.ConnectionID})
	}
	return response(200, ""), nil
}

func (h *Handler) subscribe(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	var msg subscribeMessage
	if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
		return response(400, "invalid message"), nil
	}
	if msg.Type != "subscribe" || strings.TrimSpace(msg.RequestID) == "" {
		return response(400, "invalid subscription"), nil
	}

	conn, err := h.repo.GetConnection(ctx, event.RequestContext.ConnectionID)
	if err != nil {
		return response(401, "connection not found"), nil
	}
	req, err := h.requests.GetByID(ctx, msg.RequestID)
	if err != nil {
		if errors.Is(err, request.ErrNotFound) {
			return response(404, "request not found"), nil
		}
		if h.logger != nil {
			h.logger.Error("load websocket request", err, map[string]any{"requestId": msg.RequestID})
		}
		return response(500, "subscription failed"), nil
	}
	if conn.UserID != req.OwnerID && conn.UserID != req.RequesterID {
		return response(403, "not a participant"), nil
	}

	if err := h.repo.SaveSubscription(ctx, Subscription{
		ConnectionID: conn.ID,
		RequestID:    msg.RequestID,
		UserID:       conn.UserID,
		ExpiresAt:    h.now().Add(24 * time.Hour).Unix(),
	}); err != nil {
		if h.logger != nil {
			h.logger.Error("save websocket subscription", err, map[string]any{"connectionId": conn.ID, "requestId": msg.RequestID})
		}
		return response(500, "subscription failed"), nil
	}
	return response(200, `{"type":"subscribed"}`), nil
}

func (h *Handler) routeDefault(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	var msg messageTypeEnvelope
	if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
		return response(400, "invalid message"), nil
	}
	switch msg.Type {
	case "typing":
		return h.typing(ctx, event)
	default:
		return response(200, ""), nil
	}
}

func (h *Handler) typing(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	var msg typingMessage
	if err := json.Unmarshal([]byte(event.Body), &msg); err != nil {
		return response(400, "invalid message"), nil
	}
	if msg.Type != "typing" || strings.TrimSpace(msg.RequestID) == "" {
		return response(400, "invalid typing event"), nil
	}
	conn, req, resp, ok := h.authorizedConnectionForRequest(ctx, event.RequestContext.ConnectionID, msg.RequestID)
	if !ok {
		return resp, nil
	}
	if conn.UserID != req.OwnerID && conn.UserID != req.RequesterID {
		return response(403, "not a participant"), nil
	}
	if msg.IsTyping && h.isTypingRateLimited(conn.ID) {
		return response(200, `{"type":"typing.accepted"}`), nil
	}
	if h.typingPublisher != nil {
		if err := h.typingPublisher.PublishTyping(ctx, TypingEvent{
			RequestID: msg.RequestID,
			SenderID:  conn.UserID,
			IsTyping:  msg.IsTyping,
		}); err != nil && h.logger != nil {
			h.logger.Error("publish typing websocket event", err, map[string]any{"connectionId": conn.ID, "requestId": msg.RequestID})
		}
	}
	return response(200, `{"type":"typing.accepted"}`), nil
}

func (h *Handler) authorizedConnectionForRequest(ctx context.Context, connectionID, requestID string) (Connection, request.Request, events.APIGatewayProxyResponse, bool) {
	conn, err := h.repo.GetConnection(ctx, connectionID)
	if err != nil {
		return Connection{}, request.Request{}, response(401, "connection not found"), false
	}
	req, err := h.requests.GetByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, request.ErrNotFound) {
			return Connection{}, request.Request{}, response(404, "request not found"), false
		}
		if h.logger != nil {
			h.logger.Error("load websocket request", err, map[string]any{"requestId": requestID})
		}
		return Connection{}, request.Request{}, response(500, "request lookup failed"), false
	}
	if conn.UserID != req.OwnerID && conn.UserID != req.RequesterID {
		return Connection{}, request.Request{}, response(403, "not a participant"), false
	}
	return conn, req, events.APIGatewayProxyResponse{}, true
}

func (h *Handler) isTypingRateLimited(connectionID string) bool {
	const minInterval = 500 * time.Millisecond
	now := h.now()
	h.typingMu.Lock()
	defer h.typingMu.Unlock()
	last, ok := h.lastTypingSent[connectionID]
	if ok && now.Sub(last) < minInterval {
		return true
	}
	h.lastTypingSent[connectionID] = now
	return false
}

func bearerToken(headers map[string]string) string {
	for key, value := range headers {
		if strings.EqualFold(key, "authorization") {
			token, ok := strings.CutPrefix(value, "Bearer ")
			if !ok {
				return ""
			}
			return strings.TrimSpace(token)
		}
	}
	return ""
}

func response(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}
