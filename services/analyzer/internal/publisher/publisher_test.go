package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
)

type mockSQS struct {
	sendMessageFunc func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (m *mockSQS) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return m.sendMessageFunc(ctx, params, optFns...)
}

func TestPublish_Success(t *testing.T) {
	var capturedBody string
	mock := &mockSQS{
		sendMessageFunc: func(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
			capturedBody = *params.MessageBody
			if *params.QueueUrl != "https://sqs.example.com/queue" {
				t.Errorf("unexpected queue URL: %s", *params.QueueUrl)
			}
			return &sqs.SendMessageOutput{}, nil
		},
	}

	svc := NewService(mock, "https://sqs.example.com/queue")
	notification := model.DealNotification{
		EventID:      "123",
		Heading:      "Test Item",
		Price:        model.Price{Amount: 100, CurrencyCode: "DKK"},
		CanonicalURL: "https://dba.dk/test",
		Location:     "Copenhagen",
		Verdict:      "GOOD_DEAL",
		Reason:       "cheap",
		Query:        "test",
	}

	err := svc.Publish(context.Background(), notification)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed model.DealNotification
	if err := json.Unmarshal([]byte(capturedBody), &parsed); err != nil {
		t.Fatalf("failed to parse published body: %v", err)
	}
	if parsed.EventID != "123" {
		t.Errorf("unexpected eventId: %s", parsed.EventID)
	}
	if parsed.Verdict != "GOOD_DEAL" {
		t.Errorf("unexpected verdict: %s", parsed.Verdict)
	}
}

func TestPublish_SQSError(t *testing.T) {
	mock := &mockSQS{
		sendMessageFunc: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
			return nil, fmt.Errorf("sqs unavailable")
		},
	}

	svc := NewService(mock, "https://sqs.example.com/queue")
	err := svc.Publish(context.Background(), model.DealNotification{EventID: "123"})

	if err == nil {
		t.Fatal("expected error")
	}
}
