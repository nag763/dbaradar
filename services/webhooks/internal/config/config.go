package config

import (
	"context"
	"fmt"
	"net"
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
	SecretToken      string
	TrustedChatIDs   map[string]bool
	TelegramIPNets   []*net.IPNet
}

// Load resolves configuration from environment variables and AWS SSM.
func Load(ctx context.Context, ssmClient SSMGetter) (*Config, error) {
	if ssmClient == nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		ssmClient = ssm.NewFromConfig(cfg)
	}

	paramName := os.Getenv("TELEGRAM_SECRET_TOKEN_PARAM")
	var token string

	if paramName != "" {
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
		token = os.Getenv("TELEGRAM_SECRET_TOKEN")
	}

	if token == "" {
		return nil, fmt.Errorf("telegram secret token not configured")
	}

	trustedChatIDs := make(map[string]bool)
	trustedChatIDsParam := os.Getenv("TRUSTED_CHAT_IDS_PARAM")
	if trustedChatIDsParam != "" {
		output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           &trustedChatIDsParam,
			WithDecryption: nil,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch trusted chat IDs from SSM: %w", err)
		}

		if output.Parameter.Value != nil {
			ids := strings.Split(*output.Parameter.Value, ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				if id != "" {
					trustedChatIDs[id] = true
				}
			}
		}
	}

	var telegramIPNets []*net.IPNet
	telegramIPRangesParam := os.Getenv("TELEGRAM_IP_RANGES_PARAM")
	if telegramIPRangesParam != "" {
		output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           &telegramIPRangesParam,
			WithDecryption: nil,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Telegram IP ranges from SSM: %w", err)
		}

		if output.Parameter.Value != nil {
			ranges := strings.Split(*output.Parameter.Value, ",")
			for _, cidr := range ranges {
				cidr = strings.TrimSpace(cidr)
				if cidr != "" {
					_, ipNet, err := net.ParseCIDR(cidr)
					if err != nil {
						return nil, fmt.Errorf("failed to parse Telegram IP range %q: %w", cidr, err)
					}
					telegramIPNets = append(telegramIPNets, ipNet)
				}
			}
		}
	}

	return &Config{
		SecretToken:    token,
		TrustedChatIDs: trustedChatIDs,
		TelegramIPNets: telegramIPNets,
	}, nil
}
