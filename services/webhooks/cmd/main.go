package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
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
		Level: slog.LevelInfo,
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
			"route", request.RouteKey,
			"method", request.RequestContext.HTTP.Method,
			"ip", request.RequestContext.HTTP.SourceIP,
		)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 404,
			Body:       fmt.Sprintf(`{"error":"not found", "route": "%s"}`, request.RouteKey),
		}, nil
	}
	return handleWebhook(ctx, request)
}

func handleWebhook(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	sourceIP := request.RequestContext.HTTP.SourceIP

	// Verify source IP is from Telegram if ranges are configured
	if len(cfg.TelegramIPNets) > 0 {
		ip := net.ParseIP(sourceIP)
		if ip == nil {
			logger.Warn("failed to parse source IP",
				"ip", sourceIP,
				"request_id", request.RequestContext.RequestID,
			)
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 401,
				Body:       `{"error":"unauthorized"}`,
			}, nil
		}

		allowed := false
		for _, ipNet := range cfg.TelegramIPNets {
			if ipNet.Contains(ip) {
				allowed = true
				break
			}
		}

		if !allowed {
			logger.Warn("webhook IP verification failed: IP not in Telegram range",
				"ip", sourceIP,
				"request_id", request.RequestContext.RequestID,
			)
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 401,
				Body:       `{"error":"unauthorized"}`,
			}, nil
		}
	}

	logger.Debug("webhook IP verified", "ip", sourceIP)

	// Verify token
	token := request.Headers["x-telegram-bot-api-secret-token"]
	if token != cfg.SecretToken {
		logger.Warn("webhook token verification failed",
			"ip", sourceIP,
			"token_provided", token != "",
			"request_id", request.RequestContext.RequestID,
		)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 401,
			Body:       `{"error":"unauthorized"}`,
		}, nil
	}

	logger.Debug("webhook token verified", "ip", sourceIP)

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
