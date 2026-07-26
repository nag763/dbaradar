package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nag763/dbaradar/services/telegram-notifier/internal/model"
)

type mockTelegram struct {
	sendFunc func(ctx context.Context, deal model.DealNotification) error
	calls    []model.DealNotification
}

func (m *mockTelegram) SendDealNotification(ctx context.Context, deal model.DealNotification) error {
	m.calls = append(m.calls, deal)
	if m.sendFunc != nil {
		return m.sendFunc(ctx, deal)
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHandleEvent_Success(t *testing.T) {
	mock := &mockTelegram{}
	h := NewHandler(mock, testLogger())

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "msg-1",
				Body:      `{"eventId":"123","heading":"iPhone 15","price":{"amount":4500,"currencyCode":"DKK"},"canonicalUrl":"https://dba.dk/test","location":"Copenhagen","verdict":"GOOD_DEAL","reason":"cheap","query":"iphone"}`,
			},
		},
	}

	resp, err := h.HandleEvent(context.Background(), sqsEvent)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 0 {
		t.Errorf("expected no failures, got %d", len(resp.BatchItemFailures))
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 send call, got %d", len(mock.calls))
	}
	if mock.calls[0].EventID != "123" {
		t.Errorf("unexpected eventId: %s", mock.calls[0].EventID)
	}
	if mock.calls[0].Heading != "iPhone 15" {
		t.Errorf("unexpected heading: %s", mock.calls[0].Heading)
	}
}

func TestHandleEvent_TelegramError_ReportsFailure(t *testing.T) {
	mock := &mockTelegram{
		sendFunc: func(_ context.Context, _ model.DealNotification) error {
			return fmt.Errorf("telegram unavailable")
		},
	}
	h := NewHandler(mock, testLogger())

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "msg-1", Body: `{"eventId":"123","heading":"Test"}`},
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
		t.Errorf("expected msg-1, got %s", resp.BatchItemFailures[0].ItemIdentifier)
	}
}

func TestHandleEvent_InvalidJSON_ReportsFailure(t *testing.T) {
	mock := &mockTelegram{}
	h := NewHandler(mock, testLogger())

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "msg-1", Body: "not valid json"},
		},
	}

	resp, err := h.HandleEvent(context.Background(), sqsEvent)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.BatchItemFailures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(resp.BatchItemFailures))
	}
	if len(mock.calls) != 0 {
		t.Error("should not call telegram for invalid JSON")
	}
}

func TestHandleEvent_PartialFailure(t *testing.T) {
	callCount := 0
	mock := &mockTelegram{
		sendFunc: func(_ context.Context, _ model.DealNotification) error {
			callCount++
			if callCount == 2 {
				return fmt.Errorf("rate limited")
			}
			return nil
		},
	}
	h := NewHandler(mock, testLogger())

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
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
