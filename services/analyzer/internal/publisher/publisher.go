// Package publisher sends deal notifications to the downstream SQS queue.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/nag763/dbaradar/services/analyzer/internal/model"
)

// SQSAPI is the subset of the SQS client used by the publisher.
type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// Service publishes deal notifications to SQS.
type Service struct {
	client   SQSAPI
	queueURL string
}

// NewService creates a publisher targeting the given SQS queue URL.
func NewService(client SQSAPI, queueURL string) *Service {
	return &Service{
		client:   client,
		queueURL: queueURL,
	}
}

// Publish sends a deal notification to the downstream queue.
func (s *Service) Publish(ctx context.Context, notification model.DealNotification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("publish to %s: %w", s.queueURL, err)
	}

	return nil
}
