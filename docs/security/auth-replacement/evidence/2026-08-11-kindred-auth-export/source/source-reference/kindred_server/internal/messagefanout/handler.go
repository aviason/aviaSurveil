package messagefanout

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"kindred_server/internal/message"
	"kindred_server/pkg/logger"
)

type Publisher interface {
	PublishMessage(ctx context.Context, topicARN string, payload MessageEvent) error
}

type RealtimePublisher interface {
	PublishRealtime(ctx context.Context, payload MessageEvent) error
}

type NotificationProjector interface {
	ProjectRecord(ctx context.Context, record events.DynamoDBEventRecord) error
}

type Handler struct {
	topicARN              string
	publisher             Publisher
	realtimePublisher     RealtimePublisher
	notificationProjector NotificationProjector
	logger                *logger.Logger
}

type MessageEvent struct {
	Type            string `json:"type"`
	ID              string `json:"id"`
	RequestID       string `json:"requestId"`
	SenderID        string `json:"senderId"`
	CreatedAt       string `json:"createdAt"`
	ClientMessageID string `json:"clientMessageId,omitempty"`

	ProtocolVersion     int                          `json:"protocolVersion"`
	SessionID           string                       `json:"sessionId"`
	RatchetHeader       message.RatchetHeader        `json:"ratchetHeader"`
	Nonce               string                       `json:"nonce"`
	Ciphertext          string                       `json:"ciphertext"`
	Bootstrap           *message.BootstrapData       `json:"bootstrap,omitempty"`
	CiphertextHash      string                       `json:"ciphertextHash"`
	PlaintextHash       string                       `json:"plaintextHash"`
	SenderSignature     string                       `json:"senderSignature"`
	SenderKeyVersion    int                          `json:"senderKeyVersion"`
	NotificationPreview *message.NotificationPreview `json:"notificationPreview,omitempty"`
}

func NewHandler(topicARN string, publisher Publisher, logger *logger.Logger) *Handler {
	return &Handler{topicARN: topicARN, publisher: publisher, logger: logger}
}

func (h *Handler) WithRealtimePublisher(publisher RealtimePublisher) *Handler {
	h.realtimePublisher = publisher
	return h
}

func (h *Handler) WithNotificationProjector(projector NotificationProjector) *Handler {
	h.notificationProjector = projector
	return h
}

func (h *Handler) Handle(ctx context.Context, event events.DynamoDBEvent) (events.DynamoDBEventResponse, error) {
	var failures []events.DynamoDBBatchItemFailure

	for _, record := range event.Records {
		if h.notificationProjector != nil {
			if err := h.notificationProjector.ProjectRecord(ctx, record); err != nil {
				if h.logger != nil {
					h.logger.Error("project notification event", err, map[string]any{"sequence": record.Change.SequenceNumber})
				}
				failures = append(failures, events.DynamoDBBatchItemFailure{ItemIdentifier: record.Change.SequenceNumber})
				continue
			}
		}
		msg, ok := messageFromRecord(record)
		if !ok {
			continue
		}
		if err := h.publisher.PublishMessage(ctx, h.topicARN, msg); err != nil {
			if h.logger != nil {
				h.logger.Error("publish message event", err, map[string]any{"requestId": msg.RequestID, "messageId": msg.ID})
			}
			failures = append(failures, events.DynamoDBBatchItemFailure{ItemIdentifier: record.Change.SequenceNumber})
			continue
		}
		if h.realtimePublisher != nil {
			if err := h.realtimePublisher.PublishRealtime(ctx, msg); err != nil && h.logger != nil {
				h.logger.Error("publish realtime message event", err, map[string]any{"requestId": msg.RequestID, "messageId": msg.ID})
			}
		}
	}

	return events.DynamoDBEventResponse{BatchItemFailures: failures}, nil
}

func messageFromRecord(record events.DynamoDBEventRecord) (MessageEvent, bool) {
	if record.EventName != "INSERT" {
		return MessageEvent{}, false
	}

	pk := stringValue(record.Change.Keys, "pk")
	sk := stringValue(record.Change.Keys, "sk")
	if !strings.HasPrefix(pk, "REQ#") || !strings.HasPrefix(sk, "MSG#") {
		return MessageEvent{}, false
	}

	image := record.Change.NewImage
	msg := MessageEvent{
		Type:                "message.created",
		ID:                  stringValue(image, "id"),
		RequestID:           stringValue(image, "requestId"),
		SenderID:            stringValue(image, "senderId"),
		CreatedAt:           stringValue(image, "createdAt"),
		ClientMessageID:     stringValue(image, "clientMessageId"),
		ProtocolVersion:     numberValue(image, "protocolVersion"),
		SessionID:           stringValue(image, "sessionId"),
		RatchetHeader:       ratchetHeaderValue(image, "ratchetHeader"),
		Nonce:               stringValue(image, "nonce"),
		Ciphertext:          stringValue(image, "ciphertext"),
		Bootstrap:           bootstrapValue(image, "bootstrap"),
		CiphertextHash:      stringValue(image, "ciphertextHash"),
		PlaintextHash:       stringValue(image, "plaintextHash"),
		SenderSignature:     stringValue(image, "senderSignature"),
		SenderKeyVersion:    numberValue(image, "senderKeyVersion"),
		NotificationPreview: notificationPreviewValue(image, "notificationPreview"),
	}
	if msg.ID == "" || msg.RequestID == "" || msg.SenderID == "" {
		return MessageEvent{}, false
	}
	return msg, true
}

func stringValue(attrs map[string]events.DynamoDBAttributeValue, key string) string {
	value, ok := attrs[key]
	if !ok || value.IsNull() {
		return ""
	}
	return value.String()
}

func numberValue(attrs map[string]events.DynamoDBAttributeValue, key string) int {
	value, ok := attrs[key]
	if !ok || value.IsNull() {
		return 0
	}
	n, err := strconv.Atoi(value.Number())
	if err != nil {
		return 0
	}
	return n
}

func ratchetHeaderValue(attrs map[string]events.DynamoDBAttributeValue, key string) message.RatchetHeader {
	m, ok := mapValue(attrs, key)
	if !ok {
		return message.RatchetHeader{}
	}
	header := message.RatchetHeader{
		DHPublicKey:         stringValue(m, "dhPublicKey"),
		PreviousChainLength: numberValue(m, "previousChainLength"),
		MessageNumber:       numberValue(m, "messageNumber"),
	}
	if pq, ok := pqHeaderValue(m, "pq"); ok {
		header.PQ = &pq
	}
	return header
}

func pqHeaderValue(attrs map[string]events.DynamoDBAttributeValue, key string) (message.PQRatchetHeader, bool) {
	m, ok := mapValue(attrs, key)
	if !ok {
		return message.PQRatchetHeader{}, false
	}
	return message.PQRatchetHeader{
		Epoch:         numberValue(m, "epoch"),
		Type:          stringValue(m, "type"),
		MessageNumber: numberValue(m, "messageNumber"),
		Data:          stringValue(m, "data"),
	}, true
}

func bootstrapValue(attrs map[string]events.DynamoDBAttributeValue, key string) *message.BootstrapData {
	m, ok := mapValue(attrs, key)
	if !ok {
		return nil
	}
	return &message.BootstrapData{
		SignedX25519PrekeyID:     stringValue(m, "signedX25519PrekeyId"),
		OneTimeMLKEMPrekeyID:     stringValue(m, "oneTimeMLKEMPrekeyId"),
		SenderEphemeralPublicKey: stringValue(m, "senderEphemeralPublicKey"),
		MLKEMCiphertext:          stringValue(m, "mlkemCiphertext"),
	}
}

func notificationPreviewValue(attrs map[string]events.DynamoDBAttributeValue, key string) *message.NotificationPreview {
	m, ok := mapValue(attrs, key)
	if !ok {
		return nil
	}
	return &message.NotificationPreview{
		Version:       numberValue(m, "version"),
		KeyID:         stringValue(m, "keyId"),
		KEMCiphertext: stringValue(m, "kemCiphertext"),
		Nonce:         stringValue(m, "nonce"),
		Ciphertext:    stringValue(m, "ciphertext"),
		Signature:     stringValue(m, "signature"),
	}
}

func mapValue(attrs map[string]events.DynamoDBAttributeValue, key string) (map[string]events.DynamoDBAttributeValue, bool) {
	value, ok := attrs[key]
	if !ok || value.IsNull() {
		return nil, false
	}
	if value.DataType() != events.DataTypeMap {
		return nil, false
	}
	return value.Map(), true
}

func EncodeMessageEvent(payload MessageEvent) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
