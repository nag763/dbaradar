package handler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/config"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/fetcher"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/query"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/queue"
)

type HandlerService struct {
	configSvc  config.ConfigService
	fetcherSvc fetcher.FetcherService
	queueSvc   queue.QueueService
	querySvc   query.QueryService
	logger     slog.Logger
}

func NewHandlerService(configSvc config.ConfigService, fetcherSvc fetcher.FetcherService, queueSvc queue.QueueService, querySvc query.QueryService, logger slog.Logger) HandlerService {
	return HandlerService{
		configSvc:  configSvc,
		fetcherSvc: fetcherSvc,
		queueSvc:   queueSvc,
		querySvc:   querySvc,
		logger:     logger,
	}
}

func (h *HandlerService) Handle(ctx context.Context, event events.EventBridgeEvent) error {
	var sqsErrors int
	h.logger.Info("event received...")
	trustedClientIds := h.configSvc.TrustedChatIds
	if len(trustedClientIds) == 0 {
		h.logger.Info("no trusted client id registered found, exiting...")
		return nil
	}
	queries, err := h.querySvc.GetQueries(ctx, trustedClientIds)
	if err != nil {
		return fmt.Errorf("an error happened while trying to access user queries : %v", err)
	}

	sqsErrorsChan := make(chan int, len(queries))
	// We limit concurrency to 7
	sem := make(chan struct{}, 7)

	// Syncing asynchronously
	var wg sync.WaitGroup

	for _, q := range queries {
		wg.Add(1)
		sem <- struct{}{}

		go func(query query.ClientIdQueryItem) {
			defer wg.Done()
			defer func() { <-sem }()

			h.logger.Info("fetching dba for client", "clientId", query.ClientId, "query", query.Query)
			dbaDocs, err := h.fetcherSvc.FetchDba(ctx, query.Query)
			if err != nil {
				h.logger.Warn("an error happened while attempting to fetch query, no event will be published for the item", "clientId", query.ClientId, "query", query.Query, "err", err)
				return
			}
			if len(dbaDocs) == 0 {
				h.logger.Warn("no dba doc found for query", "clientId", query.ClientId, "query", query.Query)
				return
			}
			h.logger.Info("attempting to publish events in queue", "clientId", query.ClientId, "query", query.Query, "numberOfEvents", len(dbaDocs))

			if err := h.queueSvc.PublishEvents(ctx, query.Query, query.ClientId, dbaDocs); err != nil {
				h.logger.Error("an error happened while publishing events in queue", "err", err)
				sqsErrorsChan <- 1
			}
		}(q)

	}
	go func() {
		wg.Wait()
		close(sqsErrorsChan)
	}()
	for res := range sqsErrorsChan {
		sqsErrors += res
	}

	if sqsErrors != 0 {
		h.logger.Warn("errors happened while trying to push events in sqs", "number", sqsErrors)
	}
	return nil
}
