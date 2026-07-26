// Package dedup provides DynamoDB-backed deduplication for processed events.
package dedup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI is the subset of the DynamoDB client used by the dedup service.
type DynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

// Service handles event deduplication via DynamoDB lookups.
type Service struct {
	client    DynamoDBAPI
	tableName string
	ttlDays   int
}

// NewService creates a dedup service targeting the given DynamoDB table.
func NewService(client DynamoDBAPI, tableName string, ttlDays int) *Service {
	return &Service{
		client:    client,
		tableName: tableName,
		ttlDays:   ttlDays,
	}
}

// IsProcessed returns true if the event has already been analyzed.
func (s *Service) IsProcessed(ctx context.Context, id string) (bool, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ProjectionExpression: aws.String("id"),
	})
	if err != nil {
		return false, fmt.Errorf("dedup check for %s: %w", id, err)
	}
	return out.Item != nil, nil
}

// FilterUnprocessed returns only the offer IDs that have not been analyzed yet.
func (s *Service) FilterUnprocessed(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]map[string]types.AttributeValue, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		})
	}

	out, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			s.tableName: {
				Keys:                 keys,
				ProjectionExpression: aws.String("id"),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("batch dedup check: %w", err)
	}

	processedSet := make(map[string]bool)
	if items, ok := out.Responses[s.tableName]; ok {
		for _, item := range items {
			if attr, exists := item["id"]; exists {
				if sAttr, isS := attr.(*types.AttributeValueMemberS); isS {
					processedSet[sAttr.Value] = true
				}
			}
		}
	}

	var unprocessed []string
	for _, id := range ids {
		if !processedSet[id] {
			unprocessed = append(unprocessed, id)
		}
	}

	return unprocessed, nil
}

// MarkProcessed records the offer as analyzed with a TTL for automatic expiry.
// Returns false if the offer was already recorded by another invocation (conditional write failed).
func (s *Service) MarkProcessed(ctx context.Context, id string, verdict string) (bool, error) {
	expiresAt := time.Now().Add(time.Duration(s.ttlDays) * 24 * time.Hour).Unix()

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item: map[string]types.AttributeValue{
			"id":         &types.AttributeValueMemberS{Value: id},
			"analyzedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			"verdict":    &types.AttributeValueMemberS{Value: verdict},
			"expiresAt":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expiresAt)},
		},
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})

	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return false, nil
		}
		return false, fmt.Errorf("mark processed %s: %w", id, err)
	}
	return true, nil
}
