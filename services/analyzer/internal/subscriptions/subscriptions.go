package subscriptions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"log/slog"
)

// Service holds subscription data loaded from DynamoDB keyed by chatId and query.
type Service struct {
	tableName       string
	db              *dynamodb.Client
	logger          *slog.Logger
	chatIDToQueries map[string]map[string]struct{}
	queryToChatIDs  map[string][]string
}

// NewService initializes the subscriptions service: it resolves TRUSTED_CHAT_IDS
// (via SSM param or env var) and queries DynamoDB per chatId to build in-memory maps.
func NewService(db *dynamodb.Client, ssmClient *ssm.Client, logger *slog.Logger) (*Service, error) {
	tableName := os.Getenv("TRUSTED_CHAT_QUERIES_TABLE")
	if tableName == "" {
		tableName = "trusted-chat-queries"
	}

	svc := &Service{
		tableName:       tableName,
		db:              db,
		logger:          logger,
		chatIDToQueries: make(map[string]map[string]struct{}),
		queryToChatIDs:  make(map[string][]string),
	}

	// Resolve trusted chat IDs (SSM param name or direct env)
	trustedStr, err := resolveParameter(context.Background(), ssmClient, "TRUSTED_CHAT_IDS_PARAM", "TRUSTED_CHAT_IDS")
	if err != nil {
		return nil, fmt.Errorf("resolve trusted chat ids: %w", err)
	}
	if trustedStr == "" {
		logger.Info("no TRUSTED_CHAT_IDS found; subscriptions will be empty")
		return svc, nil
	}

	chatIDs := parseCSV(trustedStr)
	for _, cid := range chatIDs {
		if err := svc.loadQueriesForChatID(context.Background(), cid); err != nil {
			logger.Error("failed to load queries for chatId", "chatId", cid, "error", err)
			// continue loading others
		}
	}

	return svc, nil
}

func (s *Service) loadQueriesForChatID(ctx context.Context, chatID string) error {
	// Query DynamoDB for items with PK=chatID
	out, err := s.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: awsString("chatId = :cid"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":cid": &ddbtypes.AttributeValueMemberS{Value: chatID},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamo query: %w", err)
	}

	for _, it := range out.Items {
		var item struct{
			ChatID string `dynamodbav:"chatId"`
			Query  string `dynamodbav:"query"`
		}
		if err := attributevalue.UnmarshalMap(it, &item); err != nil {
			s.logger.Error("unmarshal item", "error", err)
			continue
		}
		if s.chatIDToQueries[item.ChatID] == nil {
			s.chatIDToQueries[item.ChatID] = make(map[string]struct{})
		}
		s.chatIDToQueries[item.ChatID][item.Query] = struct{}{}
		s.queryToChatIDs[item.Query] = appendIfMissing(s.queryToChatIDs[item.Query], item.ChatID)
	}

	return nil
}

// IsQuerySubscribed returns true if any trusted chat is subscribed to the query.
func (s *Service) IsQuerySubscribed(query string) bool {
	_, ok := s.queryToChatIDs[query]
	return ok
}

// GetChatIDsForQuery returns the list of chat IDs subscribed to a query.
func (s *Service) GetChatIDsForQuery(query string) []string {
	if v, ok := s.queryToChatIDs[query]; ok {
		return append([]string(nil), v...)
	}
	return nil
}

// helpers
func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func appendIfMissing(slice []string, v string) []string {
	for _, s := range slice {
		if s == v {
			return slice
		}
	}
	return append(slice, v)
}

func awsString(s string) *string { return &s }

// resolveParameter mirrors the logic used elsewhere: check paramEnvVar for an SSM
// parameter name and fetch it, else fallback to directEnvVar.
func resolveParameter(ctx context.Context, ssmClient *ssm.Client, paramEnvVar, directEnvVar string) (string, error) {
	paramName := os.Getenv(paramEnvVar)
	if paramName != "" {
		if ssmClient == nil {
			return "", fmt.Errorf("ssm client required to resolve %s", paramEnvVar)
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
