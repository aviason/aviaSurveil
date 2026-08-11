package ddbpagination

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrInvalidCursor = errors.New("invalid cursor")

const cursorVersion = 1

const (
	NamespaceOwnerItems                  = "owner_items"
	NamespaceItemQueue                   = "item_queue"
	NamespaceMyRequests                  = "my_requests"
	NamespaceConversations               = "conversations"
	NamespaceRequestConversationBackfill = "request_conversation_backfill"
	NamespacePointsLedger                = "points_ledger"
	NamespaceMessagesForward             = "messages_forward"
	NamespaceMessagesOlder               = "messages_older"
	NamespaceMessageLatestBackfill       = "message_latest_backfill"
	NamespaceMessageReports              = "message_reports"
	NamespaceNotifications               = "notifications"
)

type CursorScope struct {
	Namespace        string
	ExpectedPK       string
	ExpectedSKPrefix string
}

type cursorPayload struct {
	Version   int    `json:"v,omitempty"`
	Namespace string `json:"ns,omitempty"`
	PK        string `json:"pk"`
	SK        string `json:"sk,omitempty"`
}

func EncodeCursor(key map[string]types.AttributeValue, namespace string) (string, error) {
	if len(key) == 0 {
		return "", nil
	}
	if namespace == "" {
		return "", ErrInvalidCursor
	}
	pk, ok := stringValue(key["pk"])
	if !ok || pk == "" {
		return "", ErrInvalidCursor
	}
	payload := cursorPayload{Version: cursorVersion, Namespace: namespace, PK: pk}
	if sk, ok := stringValue(key["sk"]); ok {
		payload.SK = sk
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func EncodeCursorValues(namespace, pk, sk string) (string, error) {
	key := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: pk},
	}
	if sk != "" {
		key["sk"] = &types.AttributeValueMemberS{Value: sk}
	}
	return EncodeCursor(key, namespace)
}

func DecodeCursor(cursor string, scope CursorScope) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var payload cursorPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, ErrInvalidCursor
	}
	legacy := payload.Version == 0 && payload.Namespace == ""
	if !legacy {
		if payload.Version != cursorVersion || payload.Namespace != scope.Namespace || payload.Namespace == "" {
			return nil, ErrInvalidCursor
		}
	}
	if err := validatePayload(payload, scope, legacy); err != nil {
		return nil, ErrInvalidCursor
	}
	key := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: payload.PK},
	}
	if payload.SK != "" {
		key["sk"] = &types.AttributeValueMemberS{Value: payload.SK}
	}
	return key, nil
}

func validatePayload(payload cursorPayload, scope CursorScope, legacy bool) error {
	if payload.PK == "" {
		return ErrInvalidCursor
	}
	if legacy && (scope.Namespace == NamespaceMessagesForward || scope.Namespace == NamespaceMessagesOlder) {
		return ErrInvalidCursor
	}
	if legacy && (scope.ExpectedPK == "" || scope.ExpectedSKPrefix == "") {
		return ErrInvalidCursor
	}
	if scope.ExpectedPK != "" && payload.PK != scope.ExpectedPK {
		return ErrInvalidCursor
	}
	if scope.ExpectedPK != "" && payload.SK == "" {
		return ErrInvalidCursor
	}
	if scope.ExpectedSKPrefix != "" && !strings.HasPrefix(payload.SK, scope.ExpectedSKPrefix) {
		return ErrInvalidCursor
	}
	return nil
}

func QueryPage(ctx context.Context, client *dynamodb.Client, input dynamodb.QueryInput, cursor string, scope CursorScope) ([]map[string]types.AttributeValue, string, error) {
	startKey, err := DecodeCursor(cursor, scope)
	if err != nil {
		return nil, "", err
	}
	input.ExclusiveStartKey = startKey
	out, err := client.Query(ctx, &input)
	if err != nil {
		return nil, "", err
	}
	nextCursor, err := EncodeCursor(out.LastEvaluatedKey, scope.Namespace)
	if err != nil {
		return nil, "", err
	}
	return out.Items, nextCursor, nil
}

func QueryAll(ctx context.Context, client *dynamodb.Client, input dynamodb.QueryInput) ([]map[string]types.AttributeValue, error) {
	var items []map[string]types.AttributeValue
	for {
		out, err := client.Query(ctx, &input)
		if err != nil {
			return nil, err
		}
		items = append(items, out.Items...)
		if len(out.LastEvaluatedKey) == 0 {
			return items, nil
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func QueryUntilLimit(ctx context.Context, client *dynamodb.Client, input dynamodb.QueryInput, limit int) ([]map[string]types.AttributeValue, error) {
	var items []map[string]types.AttributeValue
	for {
		if limit > 0 {
			remaining := limit - len(items)
			if remaining <= 0 {
				return items, nil
			}
			input.Limit = aws.Int32(int32(remaining))
		}
		out, err := client.Query(ctx, &input)
		if err != nil {
			return nil, err
		}
		items = append(items, out.Items...)
		if len(out.LastEvaluatedKey) == 0 {
			return items, nil
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func ScanUntilLimit(ctx context.Context, client *dynamodb.Client, input dynamodb.ScanInput, limit int) ([]map[string]types.AttributeValue, error) {
	var items []map[string]types.AttributeValue
	for {
		if limit > 0 {
			remaining := limit - len(items)
			if remaining <= 0 {
				return items, nil
			}
			input.Limit = aws.Int32(int32(remaining))
		}
		out, err := client.Scan(ctx, &input)
		if err != nil {
			return nil, err
		}
		items = append(items, out.Items...)
		if len(out.LastEvaluatedKey) == 0 {
			return items, nil
		}
		input.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func stringValue(v types.AttributeValue) (string, bool) {
	s, ok := v.(*types.AttributeValueMemberS)
	if !ok {
		return "", false
	}
	return s.Value, true
}
