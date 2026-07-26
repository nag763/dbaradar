package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMGetter defines the interface for fetching SSM parameters.
type SSMGetter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// Config holds the resolved configuration.
type Config struct {
	SecretToken    string
	TrustedChatIDs map[string]bool
}

// Load resolves configuration from environment variables and AWS SSM.
func Load(ctx context.Context, ssmClient SSMGetter) (*Config, error) {
	paramName := os.Getenv("TELEGRAM_SECRET_TOKEN_PARAM")

	var token string

	if paramName != "" {
		// Fetch from SSM
		if ssmClient == nil {
			cfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load AWS config: %w", err)
			}
			ssmClient = ssm.NewFromConfig(cfg)
		}

		withDecryption := true
		output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           &paramName,
			WithDecryption: &withDecryption,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch SSM parameter %q: %w", paramName, err)
		}

		if output.Parameter.Value != nil {
			token = *output.Parameter.Value
		}
	} else {
		// Fallback to direct env var
		token = os.Getenv("TELEGRAM_SECRET_TOKEN")
	}

	if token == "" {
		return nil, fmt.Errorf("telegram secret token not configured")
	}

	// Load trusted chat IDs from SSM
	trustedChatIDsParam := os.Getenv("TRUSTED_CHAT_IDS_PARAM")
	trustedChatIDs := make(map[string]bool)

	if trustedChatIDsParam != "" {
		if ssmClient == nil {
			cfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load AWS config: %w", err)
			}
			ssmClient = ssm.NewFromConfig(cfg)
		}

		output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           &trustedChatIDsParam,
			WithDecryption: nil,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch trusted chat IDs from SSM: %w", err)
		}

		if output.Parameter.Value != nil {
			// Parse comma-separated list
			ids := strings.Split(*output.Parameter.Value, ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				if id != "" {
					trustedChatIDs[id] = true
				}
			}
		}
	}

	return &Config{
		SecretToken:    token,
		TrustedChatIDs: trustedChatIDs,
	}, nil
}
