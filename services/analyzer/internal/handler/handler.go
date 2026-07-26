// Package handler implements the Lambda entrypoint for the deal analyzer.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nag763/dbaradar/services/analyzer/internal/dedup"
	"github.com/nag763/dbaradar/services/analyzer/internal/llm"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
	"github.com/nag763/dbaradar/services/analyzer/internal/publisher"
)

// Handler processes SQS messages from the dba-events queue.
type Handler struct {
	dedup     *dedup.Service
	llm       *llm.Service
	publisher *publisher.Service
	logger    *slog.Logger
}

// NewHandler creates a handler with all required dependencies.
func NewHandler(dedup *dedup.Service, llm *llm.Service, publisher *publisher.Service, logger *slog.Logger) *Handler {
	return &Handler{
		dedup:     dedup,
		llm:       llm,
		publisher: publisher,
		logger:    logger,
	}
}

// HandleEvent processes a batch of SQS messages, returning partial failures.
func (h *Handler) HandleEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	var totalIngested, totalPublished int

	for _, record := range sqsEvent.Records {
		ingested, published, err := h.processRecord(ctx, record)
		totalIngested += ingested
		totalPublished += published

		if err != nil {
			h.logger.Error("failed to process record",
				"messageId", record.MessageId,
				"error", err,
			)
			failures = append(failures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	h.logger.Info("batch processing summary",
		"recordsCount", len(sqsEvent.Records),
		"totalIngested", totalIngested,
		"totalPublished", totalPublished,
		"failuresCount", len(failures),
	)

	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *Handler) processRecord(ctx context.Context, record events.SQSMessage) (int, int, error) {
	events, err := parseEvents(record.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("unmarshal events: %w", err)
	}

	ingested := len(events)
	published := 0

	for _, event := range events {
		wasPublished, err := h.processSingleEvent(ctx, event)
		if err != nil {
			return ingested, published, err
		}
		if wasPublished {
			published++
		}
	}

	return ingested, published, nil
}

func parseEvents(body string) ([]model.DbaEvent, error) {
	trimmed := bytes.TrimSpace([]byte(body))
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var events []model.DbaEvent
		if err := json.Unmarshal(trimmed, &events); err != nil {
			return nil, err
		}
		return events, nil
	}

	var event model.DbaEvent
	if err := json.Unmarshal(trimmed, &event); err != nil {
		return nil, err
	}
	return []model.DbaEvent{event}, nil
}

func (h *Handler) processSingleEvent(ctx context.Context, event model.DbaEvent) (bool, error) {
	h.logger.Info("processing event",
		"eventId", event.EventID,
		"id", event.Id,
		"heading", event.Heading,
		"price", event.Price.Amount,
	)

	processed, err := h.dedup.IsProcessed(ctx, event.Id)
	if err != nil {
		return false, err
	}
	if processed {
		h.logger.Info("event already processed, skipping", "eventId", event.EventID, "id", event.Id)
		return false, nil
	}

	verdict, err := h.llm.Analyze(ctx, event)
	if err != nil {
		return false, fmt.Errorf("llm analyze: %w", err)
	}

	h.logger.Info("llm verdict",
		"eventId", event.EventID,
		"id", event.Id,
		"verdict", verdict.Verdict,
		"reason", verdict.Reason,
	)

	written, err := h.dedup.MarkProcessed(ctx, event.Id, verdict.Verdict)
	if err != nil {
		return false, err
	}
	if !written {
		h.logger.Info("concurrent write detected, skipping notification", "eventId", event.EventID, "id", event.Id)
		return false, nil
	}

	if verdict.Verdict == model.VerdictGoodDeal {
		notification := model.DealNotification{
			EventID:      event.EventID,
			Heading:      event.Heading,
			Price:        event.Price,
			CanonicalURL: event.CanonicalURL,
			Location:     event.Location,
			Verdict:      verdict.Verdict,
			Reason:       verdict.Reason,
			Query:        event.Query,
		}
		if err := h.publisher.Publish(ctx, notification); err != nil {
			return false, err
		}
		h.logger.Info("deal notification published", "eventId", event.EventID, "id", event.Id)
		return true, nil
	}

	return false, nil
}
