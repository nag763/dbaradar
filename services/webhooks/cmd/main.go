package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/nag763/dbaradar/services/webhooks/internal/handler"
	"github.com/nag763/dbaradar/services/webhooks/internal/query"
	"github.com/nag763/dbaradar/services/webhooks/internal/trusted"
)

var (
	logger                         *slog.Logger
	h                              *handler.Handler
	trustedService                 *trusted.Trusted
	chatQueriesTableNameEnvVarName = "CHAT_QUERIES_TABLE_NAME"
)

func init() {
	ctx := context.Background()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"))
	if err != nil {
		logger.Error("could not load aws cfg", "err", err)
		os.Exit(1)
	}

	trustedService, err = trusted.Load(context.Background(), *ssm.NewFromConfig(cfg))
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	querySvcTableName := os.Getenv(chatQueriesTableNameEnvVarName)
	if len(querySvcTableName) == 0 {
		logger.Error("env var is not set, exiting...", "varName", chatQueriesTableNameEnvVarName)
		os.Exit(1)
	}
	querySvc := query.NewQueryService(*dynamodb.NewFromConfig(cfg), querySvcTableName)

	h = handler.NewHandler(logger, trustedService, querySvc)
}

func handleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if request.RouteKey != "POST /webhook" {
		logger.Warn("invalid route",
			"method", request.RequestContext.HTTP.Method,
			"ip", request.RequestContext.HTTP.SourceIP,
		)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 404,
			Body:       `{"error":"not found"}`,
		}, nil
	}
	return handleWebhook(ctx, request)
}

func handleWebhook(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	// Parse and handle webhook
	response, statusCode := h.HandleWebhookRequest(ctx, request.Body)

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      statusCode,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            response,
		IsBase64Encoded: false,
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
