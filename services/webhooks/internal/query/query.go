package query

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type QueryService struct {
	client    DynamoDBAPI
	tableName string
}

type UserQuery struct {
	ChatId string `dynamodbav:"chatId"`
	Query  string `dynamodbav:"query"`
}

func NewQueryService(client dynamodb.Client, tableName string) *QueryService {
	return &QueryService{
		client:    &client,
		tableName: tableName,
	}
}

func (q *QueryService) AddItem(ctx context.Context, clientId string, itemName string) error {

	if len(strings.TrimSpace(itemName)) == 0 {
		return errors.New("itemName is empty")
	}
	if _, err := strconv.Atoi(clientId); err != nil {
		return errors.New("client id can't be null or alphabetical")
	}

	newUserQuery := UserQuery{
		ChatId: clientId,
		Query:  itemName,
	}

	item, err := attributevalue.MarshalMap(newUserQuery)
	if err != nil {
		return errors.New("failed to create new item")
	}

	_, err = q.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &q.tableName,
		Item:      item,
	})
	return err
}
