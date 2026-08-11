package messagews

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	apigwtypes "github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"

	"kindred_server/internal/message"
	"kindred_server/internal/messagefanout"
	"kindred_server/pkg/logger"
)

type RealtimeFanout struct {
	repo   SubscriptionStore
	client ConnectionPoster
	logger *logger.Logger
}

type SubscriptionStore interface {
	ListSubscriptionsByRequest(ctx context.Context, requestID string, limit int) ([]Subscription, error)
	DeleteConnection(ctx context.Context, connectionID string) error
}

type ConnectionPoster interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

func NewRealtimeFanout(repo SubscriptionStore, client ConnectionPoster, logger *logger.Logger) *RealtimeFanout {
	return &RealtimeFanout{repo: repo, client: client, logger: logger}
}

func (f *RealtimeFanout) PublishRealtime(ctx context.Context, payload messagefanout.MessageEvent) error {
	return f.publish(ctx, payload.RequestID, payload)
}

func (f *RealtimeFanout) PublishMessageRead(ctx context.Context, state message.MessageReadState) error {
	return f.publish(ctx, state.RequestID, messageReadEvent{
		Type:                     "message.read",
		RequestID:                state.RequestID,
		UserID:                   state.UserID,
		LastReadMessageID:        state.LastReadMessageID,
		LastReadMessageCreatedAt: state.LastReadMessageCreatedAt,
		LastReadAt:               state.LastReadAt,
	})
}

func (f *RealtimeFanout) PublishTyping(ctx context.Context, event TypingEvent) error {
	return f.publish(ctx, event.RequestID, messageTypingEvent{
		Type:      "message.typing",
		RequestID: event.RequestID,
		SenderID:  event.SenderID,
		IsTyping:  event.IsTyping,
	})
}

func (f *RealtimeFanout) publish(ctx context.Context, requestID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	subs, err := f.repo.ListSubscriptionsByRequest(ctx, requestID, 0)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		_, err := f.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
			ConnectionId: aws.String(sub.ConnectionID),
			Data:         body,
		})
		if err == nil {
			continue
		}
		var gone *apigwtypes.GoneException
		if errors.As(err, &gone) {
			if deleteErr := f.repo.DeleteConnection(ctx, sub.ConnectionID); deleteErr != nil && f.logger != nil {
				f.logger.Error("delete stale websocket connection", deleteErr, map[string]any{"connectionId": sub.ConnectionID})
			}
			continue
		}
		if f.logger != nil {
			f.logger.Error("post websocket message", err, map[string]any{"connectionId": sub.ConnectionID, "requestId": requestID})
		}
	}
	return nil
}

type TypingEvent struct {
	RequestID string
	SenderID  string
	IsTyping  bool
}

type messageTypingEvent struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	SenderID  string `json:"senderId"`
	IsTyping  bool   `json:"isTyping"`
}

type messageReadEvent struct {
	Type                     string    `json:"type"`
	RequestID                string    `json:"requestId"`
	UserID                   string    `json:"userId"`
	LastReadMessageID        string    `json:"lastReadMessageId"`
	LastReadMessageCreatedAt time.Time `json:"lastReadMessageCreatedAt"`
	LastReadAt               time.Time `json:"lastReadAt"`
}
