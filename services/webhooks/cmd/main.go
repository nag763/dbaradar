package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nag763/dbaradar/services/webhooks/internal/config"
	"github.com/nag763/dbaradar/services/webhooks/internal/handler"
)

var (
	logger *slog.Logger
	h      *handler.Handler
	cfg    *config.Config
)

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	var err error
	cfg, err = config.Load(context.Background(), nil)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	h = handler.NewHandler(logger, cfg)
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
