package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"kindred_server/internal/platform/ddbpagination"
	"kindred_server/internal/points"
	"kindred_server/internal/user"
	"kindred_server/pkg/config"
)

type UserRepository struct {
	client   *dynamodb.Client
	table    string
	appTable string
}

type storedUser struct {
	PK string `dynamodbav:"pk"`
	user.User
}

type emailLock struct {
	PK     string `dynamodbav:"pk"`
	UserID string `dynamodbav:"userID"`
}

type phoneLock struct {
	PK     string `dynamodbav:"pk"`
	UserID string `dynamodbav:"userID"`
}

type signupBonusMarker struct {
	PK        string    `dynamodbav:"pk"`
	SK        string    `dynamodbav:"sk"`
	UserID    string    `dynamodbav:"userID"`
	CreatedAt time.Time `dynamodbav:"createdAt"`
}

func NewUserRepository(ctx context.Context, cfg config.Config) (*UserRepository, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	var opts []func(*dynamodb.Options)
	if cfg.DynamoDBEndpoint != "" {
		opts = append(opts, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(cfg.DynamoDBEndpoint)
		})
	}
	return &UserRepository{client: dynamodb.NewFromConfig(awsCfg, opts...), table: cfg.UsersTable, appTable: cfg.AppTable}, nil
}

// Client exposes the underlying DynamoDB client so other stores in this
// package (e.g. RateLimitStore) can reuse the same connection/credentials.
func (r *UserRepository) Client() *dynamodb.Client {
	return r.client
}

func (r *UserRepository) Create(ctx context.Context, u user.User) error {
	userItem, err := attributevalue.MarshalMap(storedUser{PK: userPK(u.ID), User: u})
	if err != nil {
		return err
	}
	emailItem, err := attributevalue.MarshalMap(emailLock{PK: emailPK(u.Email), UserID: u.ID})
	if err != nil {
		return err
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName:           aws.String(r.table),
					Item:                userItem,
					ConditionExpression: aws.String("attribute_not_exists(pk)"),
				},
			},
			{
				Put: &types.Put{
					TableName:           aws.String(r.table),
					Item:                emailItem,
					ConditionExpression: aws.String("attribute_not_exists(pk)"),
				},
			},
		},
	})
	if isTransactionCanceled(err) {
		return user.ErrAlreadyExists
	}
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (user.User, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: userPK(id)}},
	})
	if err != nil {
		return user.User{}, err
	}
	if len(out.Item) == 0 {
		return user.User{}, user.ErrNotFound
	}
	var stored storedUser
	if err := attributevalue.UnmarshalMap(out.Item, &stored); err != nil {
		return user.User{}, err
	}
	return stored.User, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: emailPK(email)}},
	})
	if err != nil {
		return user.User{}, err
	}
	if len(out.Item) == 0 {
		return user.User{}, user.ErrNotFound
	}
	var lock emailLock
	if err := attributevalue.UnmarshalMap(out.Item, &lock); err != nil {
		return user.User{}, err
	}
	return r.GetByID(ctx, lock.UserID)
}

func (r *UserRepository) List(ctx context.Context, limit int) ([]user.User, error) {
	items, err := ddbpagination.ScanUntilLimit(ctx, r.client, dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("begins_with(pk, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "USER#"},
		},
	}, limit)
	if err != nil {
		return nil, err
	}
	users := make([]user.User, 0, len(items))
	for _, item := range items {
		var stored storedUser
		if err := attributevalue.UnmarshalMap(item, &stored); err != nil {
			return nil, err
		}
		users = append(users, stored.User)
	}
	return users, nil
}

func (r *UserRepository) ListDeletionPending(ctx context.Context, now time.Time, limit int) ([]user.User, error) {
	items, err := ddbpagination.ScanUntilLimit(ctx, r.client, dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("begins_with(pk, :prefix) AND accountStatus = :status AND scheduledPurgeAt <= :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: "USER#"},
			":status": &types.AttributeValueMemberS{
				Value: string(user.AccountStatusDeletionPending),
			},
			":now": &types.AttributeValueMemberS{Value: now.UTC().Format(time.RFC3339Nano)},
		},
	}, limit)
	if err != nil {
		return nil, err
	}
	users := make([]user.User, 0, len(items))
	for _, item := range items {
		var stored storedUser
		if err := attributevalue.UnmarshalMap(item, &stored); err != nil {
			return nil, err
		}
		users = append(users, stored.User)
	}
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, u user.User) error {
	item, err := attributevalue.MarshalMap(storedUser{PK: userPK(u.ID), User: u})
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if isConditionalFailure(err) {
		return user.ErrNotFound
	}
	return err
}

func (r *UserRepository) VerifyPhoneAndGrantSignupBonus(ctx context.Context, u user.User, normalizedPhone string, expectedCodeHash string, bonusPoints int) (bool, error) {
	userItem, err := attributevalue.MarshalMap(storedUser{PK: userPK(u.ID), User: u})
	if err != nil {
		return false, err
	}
	phoneItem, err := attributevalue.MarshalMap(phoneLock{PK: phonePK(normalizedPhone), UserID: u.ID})
	if err != nil {
		return false, err
	}
	tx := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName:                aws.String(r.table),
				Item:                     userItem,
				ConditionExpression:      aws.String("attribute_exists(id) AND #otp = :otpHash"),
				ExpressionAttributeNames: map[string]string{"#otp": "phoneVerificationCodeHash"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":otpHash": &types.AttributeValueMemberS{Value: expectedCodeHash},
				},
			},
		},
		{
			Put: &types.Put{
				TableName:                aws.String(r.table),
				Item:                     phoneItem,
				ConditionExpression:      aws.String("attribute_not_exists(pk) OR #uid = :userID"),
				ExpressionAttributeNames: map[string]string{"#uid": "userID"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":userID": &types.AttributeValueMemberS{Value: u.ID},
				},
			},
		},
	}
	if bonusPoints > 0 {
		now := time.Now().UTC()
		appTable := r.appTable
		if appTable == "" {
			appTable = r.table
		}
		marker, err := signupBonusMarkerRow(u.ID, now)
		if err != nil {
			return false, err
		}
		ledger, err := signupBonusLedgerRow(u.ID, bonusPoints, now)
		if err != nil {
			return false, err
		}
		tx = append(tx,
			types.TransactWriteItem{Put: &types.Put{
				TableName:           aws.String(appTable),
				Item:                marker,
				ConditionExpression: aws.String("attribute_not_exists(pk) AND attribute_not_exists(sk)"),
			}},
			types.TransactWriteItem{Update: &types.Update{
				TableName: aws.String(appTable),
				Key: map[string]types.AttributeValue{
					"pk": &types.AttributeValueMemberS{Value: userPK(u.ID)},
					"sk": &types.AttributeValueMemberS{Value: balanceSK()},
				},
				UpdateExpression: aws.String("SET available = if_not_exists(available, :zero) + :amt, updatedAt = :now, userId = :userID"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":zero":   &types.AttributeValueMemberN{Value: "0"},
					":amt":    &types.AttributeValueMemberN{Value: itoa(bonusPoints)},
					":now":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
					":userID": &types.AttributeValueMemberS{Value: u.ID},
				},
			}},
			types.TransactWriteItem{Put: &types.Put{
				TableName:           aws.String(appTable),
				Item:                ledger,
				ConditionExpression: aws.String("attribute_not_exists(pk)"),
			}},
		)
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: tx})
	if isTransactionCanceled(err) {
		return false, user.ErrAlreadyExists
	}
	return bonusPoints > 0, err
}

func (r *UserRepository) CommitPhoneChange(ctx context.Context, u user.User, oldPhone string, normalizedPhone string, expectedCodeHash string) error {
	userItem, err := attributevalue.MarshalMap(storedUser{PK: userPK(u.ID), User: u})
	if err != nil {
		return err
	}
	phoneItem, err := attributevalue.MarshalMap(phoneLock{PK: phonePK(normalizedPhone), UserID: u.ID})
	if err != nil {
		return err
	}
	tx := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName:                aws.String(r.table),
				Item:                     userItem,
				ConditionExpression:      aws.String("attribute_exists(id) AND #otp = :otpHash"),
				ExpressionAttributeNames: map[string]string{"#otp": "pendingPhoneVerificationCodeHash"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":otpHash": &types.AttributeValueMemberS{Value: expectedCodeHash},
				},
			},
		},
		{
			Put: &types.Put{
				TableName:                aws.String(r.table),
				Item:                     phoneItem,
				ConditionExpression:      aws.String("attribute_not_exists(pk) OR #uid = :userID"),
				ExpressionAttributeNames: map[string]string{"#uid": "userID"},
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":userID": &types.AttributeValueMemberS{Value: u.ID},
				},
			},
		},
	}
	if strings.TrimSpace(oldPhone) != "" && strings.TrimSpace(oldPhone) != normalizedPhone {
		tx = append(tx, types.TransactWriteItem{Delete: &types.Delete{
			TableName: aws.String(r.table),
			Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: phonePK(oldPhone)}},
		}})
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: tx})
	if isTransactionCanceled(err) {
		return user.ErrAlreadyExists
	}
	return err
}

func (r *UserRepository) PurgeAccount(ctx context.Context, found user.User, now time.Time) error {
	if r.appTable != "" {
		keys, err := r.accountPurgeKeys(ctx, found.ID)
		if err != nil {
			return err
		}
		if err := batchDeleteKeys(ctx, r.client, r.appTable, keys); err != nil {
			return err
		}
	}

	tombstone := user.User{
		ID:                  found.ID,
		CreatedAt:           found.CreatedAt,
		UpdatedAt:           now,
		AccountStatus:       user.AccountStatusDeleted,
		DeletionRequestedAt: found.DeletionRequestedAt,
		ScheduledPurgeAt:    found.ScheduledPurgeAt,
	}
	userItem, err := attributevalue.MarshalMap(storedUser{PK: userPK(found.ID), User: tombstone})
	if err != nil {
		return err
	}
	tx := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName: aws.String(r.table),
				Item:      userItem,
			},
		},
	}
	if strings.TrimSpace(found.Email) != "" {
		tx = append(tx, conditionalLockDelete(r.table, emailPK(found.Email), found.ID))
	}
	if strings.TrimSpace(found.Phone) != "" {
		tx = append(tx, conditionalLockDelete(r.table, phonePK(found.Phone), found.ID))
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: tx})
	return err
}

func (r *UserRepository) accountPurgeKeys(ctx context.Context, userID string) ([]map[string]types.AttributeValue, error) {
	rows, err := ddbpagination.ScanUntilLimit(ctx, r.client, dynamodb.ScanInput{
		TableName: aws.String(r.appTable),
		FilterExpression: aws.String(
			"#userId = :userID OR ownerId = :userID OR requesterId = :userID OR senderId = :userID OR reporterId = :userID OR accusedUserId = :userID",
		),
		ExpressionAttributeNames: map[string]string{
			"#userId": "userId",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userID": &types.AttributeValueMemberS{Value: userID},
		},
	}, 0)
	if err != nil {
		return nil, err
	}

	seenKeys := map[string]struct{}{}
	seenRequests := map[string]struct{}{}
	keys := make([]map[string]types.AttributeValue, 0, len(rows))
	for _, row := range rows {
		addDDBKey(&keys, seenKeys, row)
		if req, ok := requestFromAppRow(row); ok {
			seenRequests[req.ID] = struct{}{}
		}
	}

	for reqID := range seenRequests {
		messageRows, err := ddbpagination.QueryAll(ctx, r.client, dynamodb.QueryInput{
			TableName:              aws.String(r.appTable),
			KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk":     &types.AttributeValueMemberS{Value: reqPK(reqID)},
				":prefix": &types.AttributeValueMemberS{Value: "MSG#"},
			},
			ProjectionExpression: aws.String("pk, sk"),
		})
		if err != nil {
			return nil, err
		}
		for _, row := range messageRows {
			addDDBKey(&keys, seenKeys, row)
		}
	}

	return keys, nil
}

func requestFromAppRow(row map[string]types.AttributeValue) (storedRequest, bool) {
	pk, pkOK := row["pk"].(*types.AttributeValueMemberS)
	sk, skOK := row["sk"].(*types.AttributeValueMemberS)
	if !pkOK || !skOK {
		return storedRequest{}, false
	}
	if !(strings.HasPrefix(pk.Value, "REQ#") || strings.HasPrefix(sk.Value, "REQ#")) {
		return storedRequest{}, false
	}
	var stored storedRequest
	if err := attributevalue.UnmarshalMap(row, &stored); err != nil || stored.Request.ID == "" {
		return storedRequest{}, false
	}
	return stored, true
}

func addDDBKey(keys *[]map[string]types.AttributeValue, seen map[string]struct{}, row map[string]types.AttributeValue) {
	pk, pkOK := row["pk"].(*types.AttributeValueMemberS)
	sk, skOK := row["sk"].(*types.AttributeValueMemberS)
	if !pkOK || !skOK || pk.Value == "" || sk.Value == "" {
		return
	}
	id := pk.Value + "\x00" + sk.Value
	if _, ok := seen[id]; ok {
		return
	}
	seen[id] = struct{}{}
	*keys = append(*keys, map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: pk.Value},
		"sk": &types.AttributeValueMemberS{Value: sk.Value},
	})
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	found, err := r.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil
		}
		return err
	}
	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(r.table),
					Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: userPK(id)}},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(r.table),
					Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: emailPK(found.Email)}},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(r.table),
					Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: phonePK(found.Phone)}},
				},
			},
		},
	})
	return err
}

func conditionalLockDelete(table, pk, userID string) types.TransactWriteItem {
	return types.TransactWriteItem{
		Delete: &types.Delete{
			TableName:           aws.String(table),
			Key:                 map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: pk}},
			ConditionExpression: aws.String("attribute_not_exists(pk) OR #uid = :userID"),
			ExpressionAttributeNames: map[string]string{
				"#uid": "userID",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":userID": &types.AttributeValueMemberS{Value: userID},
			},
		},
	}
}

func batchDeleteKeys(ctx context.Context, client *dynamodb.Client, table string, keys []map[string]types.AttributeValue) error {
	for len(keys) > 0 {
		n := min(len(keys), 25)
		writes := make([]types.WriteRequest, 0, n)
		for _, key := range keys[:n] {
			writes = append(writes, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{Key: key},
			})
		}
		requestItems := map[string][]types.WriteRequest{table: writes}
		for len(requestItems[table]) > 0 {
			out, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
				RequestItems: requestItems,
			})
			if err != nil {
				return err
			}
			requestItems = out.UnprocessedItems
		}
		keys = keys[n:]
	}
	return nil
}

func isConditionalFailure(err error) bool {
	var cond *types.ConditionalCheckFailedException
	return errors.As(err, &cond)
}

func isTransactionCanceled(err error) bool {
	var canceled *types.TransactionCanceledException
	return errors.As(err, &canceled)
}

func userPK(id string) string {
	return "USER#" + id
}

func emailPK(email string) string {
	return "EMAIL#" + strings.ToLower(strings.TrimSpace(email))
}

func phonePK(phone string) string {
	return "PHONE#" + strings.TrimSpace(phone)
}

func signupBonusMarkerSK() string {
	return "POINT_BONUS#signup"
}

func signupBonusMarkerRow(userID string, now time.Time) (map[string]types.AttributeValue, error) {
	return attributevalue.MarshalMap(signupBonusMarker{
		PK:        userPK(userID),
		SK:        signupBonusMarkerSK(),
		UserID:    userID,
		CreatedAt: now,
	})
}

func signupBonusLedgerRow(userID string, bonusPoints int, now time.Time) (map[string]types.AttributeValue, error) {
	tx := points.Transaction{
		ID:        "signup_bonus",
		UserID:    userID,
		Delta:     bonusPoints,
		Reason:    points.ReasonSignupBonus,
		RefID:     userID,
		CreatedAt: now,
	}
	return attributevalue.MarshalMap(storedPointTx{
		PK:          userPK(userID),
		SK:          fmt.Sprintf("POINT_TX#%s#%s", timestampKey(now), tx.ID),
		Transaction: tx,
	})
}
