package trusted

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMGetter defines the interface for fetching SSM parameters.
type SSMGetter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// Trusted holds the resolved configuration.
type Trusted struct {
	ChatIDs map[string]bool
}

// Load resolves configuration from environment variables and AWS SSM.
func Load(ctx context.Context, ssmClient ssm.Client) (*Trusted, error) {

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

	return &Trusted{
		ChatIDs: trustedChatIDs,
	}, nil
}
