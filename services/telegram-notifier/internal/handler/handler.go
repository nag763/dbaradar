// Package handler implements the Lambda entrypoint for the Telegram notifier.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nag763/dbaradar/services/telegram-notifier/internal/model"
)

// TelegramSender is the interface for sending deal notifications via Telegram.
type TelegramSender interface {
	SendDealNotification(ctx context.Context, deal model.DealNotification) error
}

// Handler processes SQS messages from the deal-notifications queue.
type Handler struct {
	telegram TelegramSender
	logger   *slog.Logger
}

// NewHandler creates a handler with the given Telegram client.
func NewHandler(telegram TelegramSender, logger *slog.Logger) *Handler {
	return &Handler{
		telegram: telegram,
		logger:   logger,
	}
}

// HandleEvent processes a batch of SQS messages containing deal notifications.
func (h *Handler) HandleEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	h.logger.Info("Checking the queue...")
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.Error("failed to send notification",
				"messageId", record.MessageId,
				"error", err,
			)
			failures = append(failures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *Handler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var deal model.DealNotification
	if err := json.Unmarshal([]byte(record.Body), &deal); err != nil {
		return fmt.Errorf("unmarshal notification: %w", err)
	}

	h.logger.Info("sending deal notification",
		"eventId", deal.EventID,
		"heading", deal.Heading,
		"price", deal.Price.Amount,
	)

	if err := h.telegram.SendDealNotification(ctx, deal); err != nil {
		return fmt.Errorf("send telegram: %w", err)
	}

	h.logger.Info("notification sent successfully", "eventId", deal.EventID)
	return nil
}
