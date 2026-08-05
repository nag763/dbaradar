package queue

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/nag763/dbaradar/services/dba-fetcher/internal/fetcher"
)

type QueueClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type QueueService struct {
	client   QueueClient
	queueUrl string
}

func NewQueueService(client QueueClient, queueUrl string) QueueService {
	return QueueService{
		client:   client,
		queueUrl: queueUrl,
	}
}

func (qs *QueueService) PublishEvents(ctx context.Context, query string, clientId string, dbaDocs []fetcher.DbaDoc) error {
	if len(dbaDocs) == 0 {
		return errors.New("no event received in input")
	}
	dbaEvents := make([]SqsEvent, 0, len(dbaDocs))
	for _, doc := range dbaDocs {
		event := ToSqsEvent(doc, uuid.NewString(), query, clientId)
		dbaEvents = append(dbaEvents, event)
	}

	marshalled, err := json.Marshal(dbaEvents)
	if err != nil {
		return err
	}
	_, err = qs.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &qs.queueUrl,
		MessageBody: aws.String(string(marshalled)),
	})
	return err
}
