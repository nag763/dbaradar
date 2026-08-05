package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func loadEnvVarOrError(envVarName string) (string, error) {
	value, present := os.LookupEnv(envVarName)
	if !present {
		return "", fmt.Errorf("env var not found : %s", envVarName)
	}
	return value, nil
}

type ConfigService struct {
	TrustedChatIds []string
	QueueURL       string
	QueriesTable   string
}

func Load(ctx context.Context, awsCfg aws.Config) (*ConfigService, error) {

	// loading env var first
	dbaQueueUrl, err := loadEnvVarOrError("DBA_EVENTS_QUEUE_URL")
	if err != nil {
		return nil, err
	}
	queriesTable, err := loadEnvVarOrError("TRUSTED_CHAT_QUERIES_TABLE")
	if err != nil {
		return nil, err
	}
	trustedChatIdsParamName, err := loadEnvVarOrError("TRUSTED_CHAT_IDS_PARAM")
	if err != nil {
		return nil, err
	}

	// Accessing ssm params
	ssmClient := ssm.NewFromConfig(awsCfg)

	trustedChatIdsOut, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &trustedChatIdsParamName,
	})
	if err != nil {
		slog.Error("an error happened while trying to load ssm parameters", "err", err)
		os.Exit(1)
	}
	trustedChatIds := trustedChatIdsOut.Parameter.Value
	trustedChatIdsSplitted := strings.Split(*trustedChatIds, ",")
	return &ConfigService{
		QueueURL:       dbaQueueUrl,
		QueriesTable:   queriesTable,
		TrustedChatIds: trustedChatIdsSplitted,
	}, nil
}
