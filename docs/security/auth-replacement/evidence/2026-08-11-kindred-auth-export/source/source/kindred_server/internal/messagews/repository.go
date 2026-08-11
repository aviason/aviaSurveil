package messagews

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"kindred_server/internal/platform/ddbpagination"
)

type Repository struct {
	client *dynamodb.Client
	table  string
}

type Connection struct {
	ID        string    `dynamodbav:"id"`
	UserID    string    `dynamodbav:"userId"`
	ExpiresAt int64     `dynamodbav:"ttl"`
	CreatedAt time.Time `dynamodbav:"createdAt"`
}

type Subscription struct {
	ConnectionID string `dynamodbav:"connectionId"`
	RequestID    string `dynamodbav:"requestId"`
	UserID       string `dynamodbav:"userId"`
	ExpiresAt    int64  `dynamodbav:"ttl"`
}

type connectionRow struct {
	PK string `dynamodbav:"pk"`
	SK string `dynamodbav:"sk"`
	Connection
}

type subscriptionRow struct {
	PK string `dynamodbav:"pk"`
	SK string `dynamodbav:"sk"`
	Subscription
}

func NewRepository(client *dynamodb.Client, table string) *Repository {
	return &Repository{client: client, table: table}
}

func (r *Repository) SaveConnection(ctx context.Context, conn Connection) error {
	row, err := attributevalue.MarshalMap(connectionRow{
		PK:         connPK(conn.ID),
		SK:         "META",
		Connection: conn,
	})
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      row,
	})
	return err
}

func (r *Repository) GetConnection(ctx context.Context, connectionID string) (Connection, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: connPK(connectionID)},
			"sk": &types.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil {
		return Connection{}, err
	}
	if len(out.Item) == 0 {
		return Connection{}, ErrConnectionNotFound
	}
	var row connectionRow
	if err := attributevalue.UnmarshalMap(out.Item, &row); err != nil {
		return Connection{}, err
	}
	return row.Connection, nil
}

func (r *Repository) DeleteConnection(ctx context.Context, connectionID string) error {
	subs, err := r.listSubscriptionsByConnection(ctx, connectionID)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if err := r.DeleteSubscription(ctx, sub.RequestID, connectionID); err != nil {
			return err
		}
	}
	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: connPK(connectionID)},
			"sk": &types.AttributeValueMemberS{Value: "META"},
		},
	})
	return err
}

func (r *Repository) SaveSubscription(ctx context.Context, sub Subscription) error {
	byRequest, err := attributevalue.MarshalMap(subscriptionRow{
		PK:           requestSubPK(sub.RequestID),
		SK:           connSK(sub.ConnectionID),
		Subscription: sub,
	})
	if err != nil {
		return err
	}
	byConnection, err := attributevalue.MarshalMap(subscriptionRow{
		PK:           connPK(sub.ConnectionID),
		SK:           requestSubSK(sub.RequestID),
		Subscription: sub,
	})
	if err != nil {
		return err
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: aws.String(r.table), Item: byRequest}},
			{Put: &types.Put{TableName: aws.String(r.table), Item: byConnection}},
		},
	})
	return err
}

func (r *Repository) ListSubscriptionsByRequest(ctx context.Context, requestID string, limit int) ([]Subscription, error) {
	input := dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: requestSubPK(requestID)},
			":prefix": &types.AttributeValueMemberS{Value: "CONN#"},
		},
	}
	var items []map[string]types.AttributeValue
	var err error
	if limit > 0 {
		items, err = ddbpagination.QueryUntilLimit(ctx, r.client, input, limit)
	} else {
		items, err = ddbpagination.QueryAll(ctx, r.client, input)
	}
	if err != nil {
		return nil, err
	}
	return subscriptionsFromItems(items)
}

func (r *Repository) DeleteSubscription(ctx context.Context, requestID, connectionID string) error {
	_, err := r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(r.table),
					Key: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: requestSubPK(requestID)},
						"sk": &types.AttributeValueMemberS{Value: connSK(connectionID)},
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(r.table),
					Key: map[string]types.AttributeValue{
						"pk": &types.AttributeValueMemberS{Value: connPK(connectionID)},
						"sk": &types.AttributeValueMemberS{Value: requestSubSK(requestID)},
					},
				},
			},
		},
	})
	return err
}

func (r *Repository) listSubscriptionsByConnection(ctx context.Context, connectionID string) ([]Subscription, error) {
	items, err := ddbpagination.QueryAll(ctx, r.client, dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: connPK(connectionID)},
			":prefix": &types.AttributeValueMemberS{Value: "SUB#"},
		},
	})
	if err != nil {
		return nil, err
	}
	return subscriptionsFromItems(items)
}

func subscriptionsFromItems(items []map[string]types.AttributeValue) ([]Subscription, error) {
	subs := make([]Subscription, 0, len(items))
	for _, item := range items {
		var row subscriptionRow
		if err := attributevalue.UnmarshalMap(item, &row); err != nil {
			return nil, err
		}
		subs = append(subs, row.Subscription)
	}
	return subs, nil
}

func connPK(connectionID string) string    { return fmt.Sprintf("WSCONN#%s", connectionID) }
func connSK(connectionID string) string    { return fmt.Sprintf("CONN#%s", connectionID) }
func requestSubPK(requestID string) string { return fmt.Sprintf("WSREQ#%s", requestID) }
func requestSubSK(requestID string) string { return fmt.Sprintf("SUB#%s", requestID) }
