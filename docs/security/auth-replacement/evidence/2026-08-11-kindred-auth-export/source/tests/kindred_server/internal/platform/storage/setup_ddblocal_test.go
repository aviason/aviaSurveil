//go:build dynamodblocal

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"kindred_server/internal/testutil/ddblocal"
)

// setup connects to DynamoDB Local and returns a freshly-named users + app
// table pair. Tables are created on demand and torn down on test
// completion. Each test gets its own table names so they can run in
// parallel without truncation races.
type ddbFixture struct {
	Client     *dynamodb.Client
	UsersTable string
	AppTable   string
	RateTable  string
}

func setupDDB(t *testing.T) *ddbFixture {
	t.Helper()
	client := ddblocal.Connect(t)
	users, app, rate := ddblocal.UniqueTableNames(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ddblocal.EnsureTables(ctx, client, users, app, rate); err != nil {
		t.Fatalf("ensure tables: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, tbl := range []string{users, app, rate} {
			_, _ = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: &tbl})
		}
	})
	return &ddbFixture{Client: client, UsersTable: users, AppTable: app, RateTable: rate}
}
