package trusted

import (
	"context"
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
		name           string
		envChatIDParam string
		ssmMock        *MockSSMGetter
		wantChatIDs    map[string]bool
		wantError      bool
	}{
		{
			name:        "empty config",
			wantChatIDs: map[string]bool{},
		},
		{
			name:           "load trusted chat IDs from SSM",
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
			wantChatIDs: map[string]bool{
				"123": true,
				"456": true,
				"789": true,
			},
		},
		{
			name:           "load trusted chat IDs with whitespace",
			envChatIDParam: "/dba/trusted_chat_ids",
			ssmMock: &MockSSMGetter{
				GetParameterFunc: func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					chatIDsValue := "123, 456 , 789"
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{
							Value: &chatIDsValue,
						},
					}, nil
				},
			},
			wantChatIDs: map[string]bool{
				"123": true,
				"456": true,
				"789": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalChatIDParam := os.Getenv("TRUSTED_CHAT_IDS_PARAM")

			if tt.envChatIDParam != "" {
				os.Setenv("TRUSTED_CHAT_IDS_PARAM", tt.envChatIDParam)
			} else {
				os.Unsetenv("TRUSTED_CHAT_IDS_PARAM")
			}

			defer func() {
				if originalChatIDParam != "" {
					os.Setenv("TRUSTED_CHAT_IDS_PARAM", originalChatIDParam)
				} else {
					os.Unsetenv("TRUSTED_CHAT_IDS_PARAM")
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

			if len(cfg.ChatIDs) != len(tt.wantChatIDs) {
				t.Errorf("want %d trusted chat IDs, got %d", len(tt.wantChatIDs), len(cfg.ChatIDs))
			}

			for id := range tt.wantChatIDs {
				if !cfg.ChatIDs[id] {
					t.Errorf("want trusted chat ID %q, but not found", id)
				}
			}
		})
	}
}
