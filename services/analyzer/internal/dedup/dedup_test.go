package dedup

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type mockDynamoDB struct {
	getItemFunc      func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	putItemFunc      func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	batchGetItemFunc func(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

func (m *mockDynamoDB) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemFunc(ctx, params, optFns...)
}

func (m *mockDynamoDB) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItemFunc(ctx, params, optFns...)
}

func (m *mockDynamoDB) BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	return m.batchGetItemFunc(ctx, params, optFns...)
}

func TestIsProcessed_NotFound(t *testing.T) {
	mock := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	svc := NewService(mock, "test-table", 5)
	processed, err := svc.IsProcessed(context.Background(), "event-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Error("expected event to not be processed")
	}
}

func TestIsProcessed_Found(t *testing.T) {
	mock := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: "offer-123"},
				},
			}, nil
		},
	}

	svc := NewService(mock, "test-table", 5)
	processed, err := svc.IsProcessed(context.Background(), "offer-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected offer to be processed")
	}
}

func TestMarkProcessed_Success(t *testing.T) {
	mock := &mockDynamoDB{
		putItemFunc: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			if *params.TableName != "test-table" {
				t.Errorf("unexpected table: %s", *params.TableName)
			}
			if params.ConditionExpression == nil || *params.ConditionExpression != "attribute_not_exists(id)" {
				t.Error("expected conditional write")
			}
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	svc := NewService(mock, "test-table", 5)
	written, err := svc.MarkProcessed(context.Background(), "offer-123", "GOOD_DEAL")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !written {
		t.Error("expected write to succeed")
	}
}

func TestMarkProcessed_ConditionalFailure(t *testing.T) {
	mock := &mockDynamoDB{
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: ptrString("condition not met")}
		},
	}

	svc := NewService(mock, "test-table", 5)
	written, err := svc.MarkProcessed(context.Background(), "offer-123", "GOOD_DEAL")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written {
		t.Error("expected write to fail due to condition")
	}
}

func TestFilterUnprocessed(t *testing.T) {
	mock := &mockDynamoDB{
		batchGetItemFunc: func(_ context.Context, params *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
			return &dynamodb.BatchGetItemOutput{
				Responses: map[string][]map[string]types.AttributeValue{
					"test-table": {
						{
							"id": &types.AttributeValueMemberS{Value: "processed-1"},
						},
					},
				},
			}, nil
		},
	}

	svc := NewService(mock, "test-table", 5)
	unprocessed, err := svc.FilterUnprocessed(context.Background(), []string{"processed-1", "unprocessed-2", "unprocessed-3"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(unprocessed) != 2 {
		t.Fatalf("expected 2 unprocessed items, got %d", len(unprocessed))
	}
	if unprocessed[0] != "unprocessed-2" || unprocessed[1] != "unprocessed-3" {
		t.Errorf("unexpected unprocessed items: %v", unprocessed)
	}
}

func ptrString(s string) *string {
	return &s
}
