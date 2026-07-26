package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockSSMClient struct {
	paramValue string
	err        error
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Value: aws.String(m.paramValue),
		},
	}, nil
}

func TestLoad_UsingSSMParam(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN_PARAM", "/dba-notifier/telegram_bot_token")
	os.Setenv("TELEGRAM_BOT_TOKEN", "fallback-token")
	os.Setenv("TELEGRAM_CHAT_ID", "123456")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN_PARAM")
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("TELEGRAM_CHAT_ID")
	}()

	mockSSM := &mockSSMClient{paramValue: "ssm-token"}
	cfg, err := Load(context.Background(), mockSSM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BotToken != "ssm-token" {
		t.Errorf("expected BotToken 'ssm-token', got '%s'", cfg.BotToken)
	}
	if cfg.ChatID != "123456" {
		t.Errorf("expected ChatID '123456', got '%s'", cfg.ChatID)
	}
}

func TestLoad_FallbackToEnvVar(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_TOKEN_PARAM")
	os.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	os.Setenv("TELEGRAM_CHAT_ID", "654321")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("TELEGRAM_CHAT_ID")
	}()

	cfg, err := Load(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BotToken != "env-token" {
		t.Errorf("expected BotToken 'env-token', got '%s'", cfg.BotToken)
	}
	if cfg.ChatID != "654321" {
		t.Errorf("expected ChatID '654321', got '%s'", cfg.ChatID)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_TOKEN_PARAM")
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Setenv("TELEGRAM_CHAT_ID", "654321")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

	_, err := Load(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when bot token is missing")
	}
}

func TestLoad_SSMError(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN_PARAM", "/dba-notifier/telegram_bot_token")
	os.Setenv("TELEGRAM_CHAT_ID", "654321")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN_PARAM")
		os.Unsetenv("TELEGRAM_CHAT_ID")
	}()

	mockSSM := &mockSSMClient{err: errors.New("ssm error")}
	_, err := Load(context.Background(), mockSSM)
	if err == nil {
		t.Fatal("expected error when SSM fetch fails")
	}
}
