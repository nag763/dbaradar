package config

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// MockSSMGetter implements SSMGetter for testing.
type MockSSMGetter struct {
	GetParameterFunc func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func (m *MockSSMGetter) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return m.GetParameterFunc(ctx, params, optFns...)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name             string
		envParam         string
		envToken         string
		envChatIDParam   string
		envIPRangesParam string
		ssmMock          *MockSSMGetter
		wantToken        string
		wantChatIDs      map[string]bool
		wantIPNets       []string
		wantError        bool
	}{
		{
			name:      "direct env var",
			envToken:  "secret-token-123",
			wantToken: "secret-token-123",
			wantChatIDs: map[string]bool{},
			wantIPNets: []string{},
		},
		{
			name:     "SSM parameter resolution",
			envParam: "/dba/token",
			ssmMock: &MockSSMGetter{
				GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					tokenValue := "ssm-secret-token"
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{
							Value: &tokenValue,
						},
					}, nil
				},
			},
			wantToken: "ssm-secret-token",
			wantChatIDs: map[string]bool{},
			wantIPNets: []string{},
		},
		{
			name:      "missing token",
			wantError: true,
			wantIPNets: []string{},
		},
		{
			name:     "SSM parameter preferred over direct env var",
			envParam: "/dba/token",
			envToken: "direct-token",
			ssmMock: &MockSSMGetter{
				GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					tokenValue := "ssm-token"
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{
							Value: &tokenValue,
						},
					}, nil
				},
			},
			wantToken: "ssm-token",
			wantChatIDs: map[string]bool{},
			wantIPNets: []string{},
		},
		{
			name:           "load trusted chat IDs from SSM",
			envToken:       "token-123",
			envChatIDParam: "/dba/trusted_chat_ids",
			ssmMock: &MockSSMGetter{
				GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					chatIDsValue := "123,456,789"
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{
							Value: &chatIDsValue,
						},
					}, nil
				},
			},
			wantToken: "token-123",
			wantChatIDs: map[string]bool{
				"123": true,
				"456": true,
				"789": true,
			},
			wantIPNets: []string{},
		},
		{
			name:             "load Telegram IP ranges from SSM",
			envToken:         "token-123",
			envIPRangesParam: "/dba/telegram_ip_ranges",
			ssmMock: &MockSSMGetter{
				GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					ipRangesValue := "149.154.160.0/20,91.108.4.0/22"
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{
							Value: &ipRangesValue,
						},
					}, nil
				},
			},
			wantToken: "token-123",
			wantChatIDs: map[string]bool{},
			wantIPNets: []string{"149.154.160.0/20", "91.108.4.0/22"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalParam := os.Getenv("TELEGRAM_SECRET_TOKEN_PARAM")
			originalToken := os.Getenv("TELEGRAM_SECRET_TOKEN")
			originalChatIDParam := os.Getenv("TRUSTED_CHAT_IDS_PARAM")
			originalIPRangesParam := os.Getenv("TELEGRAM_IP_RANGES_PARAM")

			// Set test env vars
			if tt.envParam != "" {
				os.Setenv("TELEGRAM_SECRET_TOKEN_PARAM", tt.envParam)
			} else {
				os.Unsetenv("TELEGRAM_SECRET_TOKEN_PARAM")
			}

			if tt.envToken != "" {
				os.Setenv("TELEGRAM_SECRET_TOKEN", tt.envToken)
			} else {
				os.Unsetenv("TELEGRAM_SECRET_TOKEN")
			}

			if tt.envChatIDParam != "" {
				os.Setenv("TRUSTED_CHAT_IDS_PARAM", tt.envChatIDParam)
			} else {
				os.Unsetenv("TRUSTED_CHAT_IDS_PARAM")
			}

			if tt.envIPRangesParam != "" {
				os.Setenv("TELEGRAM_IP_RANGES_PARAM", tt.envIPRangesParam)
			} else {
				os.Unsetenv("TELEGRAM_IP_RANGES_PARAM")
			}

			defer func() {
				// Restore original env vars
				if originalParam != "" {
					os.Setenv("TELEGRAM_SECRET_TOKEN_PARAM", originalParam)
				} else {
					os.Unsetenv("TELEGRAM_SECRET_TOKEN_PARAM")
				}
				if originalToken != "" {
					os.Setenv("TELEGRAM_SECRET_TOKEN", originalToken)
				} else {
					os.Unsetenv("TELEGRAM_SECRET_TOKEN")
				}
				if originalChatIDParam != "" {
					os.Setenv("TRUSTED_CHAT_IDS_PARAM", originalChatIDParam)
				} else {
					os.Unsetenv("TRUSTED_CHAT_IDS_PARAM")
				}
				if originalIPRangesParam != "" {
					os.Setenv("TELEGRAM_IP_RANGES_PARAM", originalIPRangesParam)
				} else {
					os.Unsetenv("TELEGRAM_IP_RANGES_PARAM")
				}
			}()

			cfg, err := Load(context.Background(), tt.ssmMock)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if cfg.SecretToken != tt.wantToken {
				t.Errorf("want token %q, got %q", tt.wantToken, cfg.SecretToken)
			}

			if len(cfg.TrustedChatIDs) != len(tt.wantChatIDs) {
				t.Errorf("want %d trusted chat IDs, got %d", len(tt.wantChatIDs), len(cfg.TrustedChatIDs))
			}

			for id := range tt.wantChatIDs {
				if !cfg.TrustedChatIDs[id] {
					t.Errorf("want trusted chat ID %q, but not found", id)
				}
			}

			if len(cfg.TelegramIPNets) != len(tt.wantIPNets) {
				t.Errorf("want %d IP nets, got %d", len(tt.wantIPNets), len(cfg.TelegramIPNets))
			}

			for i, wantCIDR := range tt.wantIPNets {
				_, wantNet, _ := net.ParseCIDR(wantCIDR)
				if cfg.TelegramIPNets[i].String() != wantNet.String() {
					t.Errorf("want IP net %q, got %q", wantNet.String(), cfg.TelegramIPNets[i].String())
				}
			}
		})
	}
}
