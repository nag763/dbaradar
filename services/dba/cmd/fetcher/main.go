package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/config"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/fetcher"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/handler"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/query"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/queue"
)

var (
	handlerSvc handler.HandlerService
)

func init() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	logger.Info("starting the handler...")

	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-central-1"))
	if err != nil {
		slog.Error("an error happened while trying to load aws config", "err", err)
		os.Exit(1)
	}

	config, err := config.Load(ctx, cfg)
	if err != nil {
		logger.Error("an error happened while trying to load the config...", "err", err)
		os.Exit(1)
	}

	querySvc := query.NewQueryService(*dynamodb.NewFromConfig(cfg), config.QueriesTable)
	fetcherSvc := fetcher.NewFetcherService()
	queueSvc := queue.NewQueueService(sqs.NewFromConfig(cfg), config.QueueURL)

	handlerSvc = handler.NewHandlerService(*config, fetcherSvc, queueSvc, querySvc, *logger)
}

func main() {
	lambda.Start(handlerSvc.Handle)
}
