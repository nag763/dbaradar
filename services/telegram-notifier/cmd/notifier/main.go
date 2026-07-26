// Package main is the entrypoint for the Telegram notifier Lambda function.
// It reads configuration from environment variables and starts the Lambda runtime.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nag763/dbaradar/services/telegram-notifier/internal/config"
	"github.com/nag763/dbaradar/services/telegram-notifier/internal/handler"
	"github.com/nag763/dbaradar/services/telegram-notifier/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("Starting telegram notifier lambda...")
	cfg, err := config.Load(context.Background(), nil)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	client := telegram.NewClient(cfg.BotToken, cfg.ChatID)
	h := handler.NewHandler(client, logger)

	lambda.Start(h.HandleEvent)
}
