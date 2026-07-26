// Package config handles resolution of configuration parameters, including resolving AWS SSM parameters when configured.
package config

import (
	"context"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMGetter represents an AWS SSM GetParameter client.
type SSMGetter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// Config holds resolved configuration values for the Telegram notifier.
type Config struct {
	BotToken string
	ChatID   string
}

// Load resolves configuration values using environment variables and AWS SSM Parameter Store fallback.
// It checks TELEGRAM_BOT_TOKEN_PARAM first; if present, it fetches the secret value from SSM.
// Otherwise, it falls back to TELEGRAM_BOT_TOKEN.
func Load(ctx context.Context, ssmClient SSMGetter) (*Config, error) {
	botToken, err := resolveParameter(ctx, ssmClient, "TELEGRAM_BOT_TOKEN_PARAM", "TELEGRAM_BOT_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("resolve bot token: %w", err)
	}
	if botToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN or TELEGRAM_BOT_TOKEN_PARAM is required")
	}

	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if chatID == "" {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID is required")
	}

	return &Config{
		BotToken: botToken,
		ChatID:   chatID,
	}, nil
}

// NewSSMClient initializes an AWS SSM client using default AWS config loading.
func NewSSMClient(ctx context.Context) (*ssm.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load default aws config: %w", err)
	}
	return ssm.NewFromConfig(cfg), nil
}

func resolveParameter(ctx context.Context, ssmClient SSMGetter, paramEnvVar, directEnvVar string) (string, error) {
	paramName := os.Getenv(paramEnvVar)
	if paramName != "" {
		if ssmClient == nil {
			var err error
			ssmClient, err = NewSSMClient(ctx)
			if err != nil {
				return "", fmt.Errorf("init ssm client for %s: %w", paramEnvVar, err)
			}
		}
		withDecryption := true
		out, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           &paramName,
			WithDecryption: &withDecryption,
		})
		if err != nil {
			return "", fmt.Errorf("fetch ssm param %s (%s): %w", paramEnvVar, paramName, err)
		}
		if out.Parameter != nil && out.Parameter.Value != nil {
			return *out.Parameter.Value, nil
		}
	}

	return os.Getenv(directEnvVar), nil
}
