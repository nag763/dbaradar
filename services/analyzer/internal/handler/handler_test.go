package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	lambdaevents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/nag763/dbaradar/services/analyzer/internal/dedup"
	"github.com/nag763/dbaradar/services/analyzer/internal/llm"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
	"github.com/nag763/dbaradar/services/analyzer/internal/publisher"
)

// --- Mock implementations ---

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
	if m.batchGetItemFunc == nil {
		return &dynamodb.BatchGetItemOutput{}, nil
	}
	return m.batchGetItemFunc(ctx, params, optFns...)
}

type mockBedrock struct {
	converseFunc func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

func (m *mockBedrock) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return m.converseFunc(ctx, params, optFns...)
}

type mockSQS struct {
	sendMessageFunc func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (m *mockSQS) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	if m.sendMessageFunc == nil {
		return &sqs.SendMessageOutput{}, nil
	}
	return m.sendMessageFunc(ctx, params, optFns...)
}

// --- Test helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func makeSQSEvent(items ...model.DbaEvent) lambdaevents.SQSEvent {
	var records []lambdaevents.SQSMessage
	for i, e := range items {
		body, _ := json.Marshal(e)
		records = append(records, lambdaevents.SQSMessage{
			MessageId: fmt.Sprintf("msg-%d", i),
			Body:      string(body),
		})
	}
	return lambdaevents.SQSEvent{Records: records}
}

// --- Tests ---

func TestHandleEvent_GoodDeal_PublishesNotification(t *testing.T) {
	published := false

	dynamo := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	bedrock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"GOOD_DEAL","reason":"great price"}`},
						},
					},
				},
			}, nil
		},
	}

	sqsMock := &mockSQS{
		sendMessageFunc: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
			published = true
			return &sqs.SendMessageOutput{}, nil
		},
	}

	h := NewHandler(
		dedup.NewService(dynamo, "test-table", 7),
		llm.NewService(bedrock),
		publisher.NewService(sqsMock, "https://queue.example.com"),
		testLogger(),
	)

	event := makeSQSEvent(model.DbaEvent{EventID: "123", Heading: "Test", Price: model.Price{Amount: 100, CurrencyCode: "DKK"}})
	resp, err := h.HandleEvent(context.Background(), event)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected no failures, got %d", len(resp.BatchItemFailures))
	}
	if !published {
		t.Error("expected notification to be published")
	}
}

func TestHandleEvent_NotADeal_DoesNotPublish(t *testing.T) {
	published := false

	dynamo := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	bedrock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"NOT_A_DEAL","reason":"overpriced"}`},
						},
					},
				},
			}, nil
		},
	}

	sqsMock := &mockSQS{
		sendMessageFunc: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
			published = true
			return &sqs.SendMessageOutput{}, nil
		},
	}

	h := NewHandler(
		dedup.NewService(dynamo, "test-table", 7),
		llm.NewService(bedrock),
		publisher.NewService(sqsMock, "https://queue.example.com"),
		testLogger(),
	)

	event := makeSQSEvent(model.DbaEvent{EventID: "456", Heading: "Expensive thing"})
	resp, err := h.HandleEvent(context.Background(), event)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected no failures, got %d", len(resp.BatchItemFailures))
	}
	if published {
		t.Error("should not publish notification for NOT_A_DEAL")
	}
}

func TestHandleEvent_AlreadyProcessed_Skips(t *testing.T) {
	bedrockCalled := false

	dynamo := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]dbtypes.AttributeValue{
					"id": &dbtypes.AttributeValueMemberS{Value: "123"},
				},
			}, nil
		},
	}

	bedrock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			bedrockCalled = true
			return nil, nil
		},
	}

	h := NewHandler(
		dedup.NewService(dynamo, "test-table", 7),
		llm.NewService(bedrock),
		publisher.NewService(&mockSQS{}, "https://queue.example.com"),
		testLogger(),
	)

	event := makeSQSEvent(model.DbaEvent{EventID: "123"})
	resp, err := h.HandleEvent(context.Background(), event)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected no failures, got %d", len(resp.BatchItemFailures))
	}
	if bedrockCalled {
		t.Error("bedrock should not be called for already-processed events")
	}
}

func TestHandleEvent_PartialFailure(t *testing.T) {
	callCount := 0

	dynamo := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			callCount++
			if callCount == 2 {
				return nil, fmt.Errorf("dynamodb unavailable")
			}
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	bedrock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"NOT_A_DEAL","reason":"ok"}`},
						},
					},
				},
			}, nil
		},
	}

	h := NewHandler(
		dedup.NewService(dynamo, "test-table", 7),
		llm.NewService(bedrock),
		publisher.NewService(&mockSQS{}, "https://queue.example.com"),
		testLogger(),
	)

	sqsEvent := lambdaevents.SQSEvent{
		Records: []lambdaevents.SQSMessage{
			{MessageId: "msg-0", Body: `{"eventId":"100","heading":"Item 1"}`},
			{MessageId: "msg-1", Body: `{"eventId":"200","heading":"Item 2"}`},
			{MessageId: "msg-2", Body: `{"eventId":"300","heading":"Item 3"}`},
		},
	}

	resp, err := h.HandleEvent(context.Background(), sqsEvent)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}
	if resp.BatchItemFailures[0].ItemIdentifier != "msg-1" {
		t.Errorf("expected msg-1 to fail, got %s", resp.BatchItemFailures[0].ItemIdentifier)
	}
}

func TestHandleEvent_ArrayBatchPayload(t *testing.T) {
	publishedCount := 0

	dynamo := &mockDynamoDB{
		getItemFunc: func(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
		putItemFunc: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	bedrock := &mockBedrock{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &brtypes.ConverseOutputMemberMessage{
					Value: brtypes.Message{
						Content: []brtypes.ContentBlock{
							&brtypes.ContentBlockMemberText{Value: `{"verdict":"GOOD_DEAL","reason":"great price"}`},
						},
					},
				},
			}, nil
		},
	}

	sqsMock := &mockSQS{
		sendMessageFunc: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
			publishedCount++
			return &sqs.SendMessageOutput{}, nil
		},
	}

	h := NewHandler(
		dedup.NewService(dynamo, "test-table", 5),
		llm.NewService(bedrock),
		publisher.NewService(sqsMock, "https://queue.example.com"),
		testLogger(),
	)

	items := []model.DbaEvent{
		{EventID: "1", Heading: "Item 1", Price: model.Price{Amount: 100, CurrencyCode: "DKK"}},
		{EventID: "2", Heading: "Item 2", Price: model.Price{Amount: 200, CurrencyCode: "DKK"}},
	}
	body, _ := json.Marshal(items)

	sqsEvent := lambdaevents.SQSEvent{
		Records: []lambdaevents.SQSMessage{
			{MessageId: "msg-batch", Body: string(body)},
		},
	}

	resp, err := h.HandleEvent(context.Background(), sqsEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(resp.BatchItemFailures))
	}
	if publishedCount != 2 {
		t.Errorf("expected 2 published notifications, got %d", publishedCount)
	}
}
