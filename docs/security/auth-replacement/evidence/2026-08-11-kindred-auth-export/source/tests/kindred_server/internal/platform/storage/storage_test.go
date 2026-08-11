// Package storage holds DynamoDB-backed Repository implementations.
//
// No hermetic unit tests live here: every function in this package issues
// real calls against the AWS SDK DynamoDB client and exercises behavior
// that depends on table schema and conditional expressions. Real
// DynamoDB conditional-expression tests live behind build tag
// `dynamodblocal` (see *_ddblocal_test.go) and run against DynamoDB Local.
package storage

import "testing"

func TestSkipped(t *testing.T) {
	t.Skip("Real DynamoDB conditional-expression tests live behind build tag `dynamodblocal`. Run with `make test-ddb` after `make dev-up`.")
}
