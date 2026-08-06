// Package main is the entrypoint for the deal analyzer Lambda function.
// It wires up AWS SDK clients and starts the Lambda runtime.
package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/nag763/dbaradar/services/analyzer/internal/dedup"
	"github.com/nag763/dbaradar/services/analyzer/internal/handler"
	"github.com/nag763/dbaradar/services/analyzer/internal/llm"
	"github.com/nag763/dbaradar/services/analyzer/internal/publisher"
	"github.com/nag763/dbaradar/services/analyzer/internal/subscriptions"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	tableName := os.Getenv("DEDUP_TABLE_NAME")
	if tableName == "" {
		tableName = "dba-analyzed-events"
	}

	queueURL := os.Getenv("DEAL_NOTIFICATIONS_QUEUE_URL")
	if queueURL == "" {
		logger.Error("DEAL_NOTIFICATIONS_QUEUE_URL is required")
		os.Exit(1)
	}

	ttlDays := 10
	if ttlStr := os.Getenv("DEDUP_TTL_DAYS"); ttlStr != "" {
		if parsed, err := strconv.Atoi(ttlStr); err == nil && 0 < parsed {
			ttlDays = parsed
		} else {
			logger.Warn("invalid DEDUP_TTL_DAYS, defaulting to 5", "value", ttlStr)
		}
	}

	dedupSvc := dedup.NewService(dynamodb.NewFromConfig(cfg), tableName, ttlDays)
	llmSvc := llm.NewService(bedrockruntime.NewFromConfig(cfg))
	pubSvc := publisher.NewService(sqs.NewFromConfig(cfg), queueURL)

	// Create SSM client for resolving TRUSTED_CHAT_IDS (user prefers SSM)
	ssmClient := ssm.NewFromConfig(cfg)

	// Subscriptions service loads TRUSTED_CHAT_IDS and fetches queries from DynamoDB
	subsSvc, err := subscriptions.NewService(dynamodb.NewFromConfig(cfg), ssmClient, logger)
	if err != nil {
		logger.Error("failed to init subscriptions service", "error", err)
		os.Exit(1)
	}

	h := handler.NewHandlerWithSubscriptions(dedupSvc, llmSvc, pubSvc, subsSvc, logger)

	lambda.Start(h.HandleEvent)
}
