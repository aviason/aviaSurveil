package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// RateLimitStore is a DynamoDB-backed fixed-window counter that works across
// multiple instances (e.g. concurrent Lambda containers). Each request
// atomically increments a counter keyed by (client key, window bucket). Items
// are auto-purged by DynamoDB TTL, so no cleanup job is required.
//
// The underlying table needs:
//   - hash key `pk` (string)
//   - TTL attribute `ttl` enabled
type RateLimitStore struct {
	client *dynamodb.Client
	table  string
}

func NewRateLimitStore(ctx context.Context, region, table string) (*RateLimitStore, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &RateLimitStore{client: dynamodb.NewFromConfig(awsCfg), table: table}, nil
}

// NewRateLimitStoreWithClient lets callers reuse an existing dynamodb.Client
// (e.g. the one already created for the users table).
func NewRateLimitStoreWithClient(client *dynamodb.Client, table string) *RateLimitStore {
	return &RateLimitStore{client: client, table: table}
}

func (s *RateLimitStore) Incr(ctx context.Context, key string, windowSeconds int64) (int64, time.Time, error) {
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	now := time.Now().Unix()
	bucket := now / windowSeconds
	resetAtUnix := (bucket + 1) * windowSeconds
	// Give DynamoDB TTL a grace period so a slow reaper cannot recycle the
	// item mid-window.
	expiresAt := resetAtUnix + 60

	pk := "RATE#" + key + "#" + strconv.FormatInt(bucket, 10)

	out, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pk},
		},
		UpdateExpression: aws.String("ADD #c :one SET #ttl = if_not_exists(#ttl, :ttl)"),
		ExpressionAttributeNames: map[string]string{
			"#c":   "count",
			"#ttl": "ttl",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one": &types.AttributeValueMemberN{Value: "1"},
			":ttl": &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAt, 10)},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, time.Time{}, err
	}

	countAttr, ok := out.Attributes["count"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, time.Time{}, nil
	}
	count, err := strconv.ParseInt(countAttr.Value, 10, 64)
	if err != nil {
		return 0, time.Time{}, err
	}
	return count, time.Unix(resetAtUnix, 0), nil
}
