package query

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
)

type DynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type QueryService struct {
	client    DynamoDBAPI
	tableName string
	validate  validator.Validate
}

type ItemCommand struct {
	ClientId string `dynamodbav:"chatId" validate:"required,min=9,max=10"`
	ItemName string `dynamodbav:"query" validate:"required,max=64"`
}

func NewQueryService(client dynamodb.Client, tableName string) *QueryService {
	return &QueryService{
		client:    &client,
		tableName: tableName,
		validate:  *validator.New(),
	}
}

func (q *QueryService) validateAndMarshal(itemCommand ItemCommand) (map[string]types.AttributeValue, error) {

	if err := q.validate.Struct(itemCommand); err != nil {
		return nil, err
	}

	return attributevalue.MarshalMap(itemCommand)
}

func (q *QueryService) AddItem(ctx context.Context, itemCommand ItemCommand) error {

	item, err := q.validateAndMarshal(itemCommand)
	if err != nil {
		return errors.New("failed to create new item")
	}

	_, err = q.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &q.tableName,
		Item:      item,
	})
	return err
}

func (q *QueryService) RemoveItem(ctx context.Context, itemCommand ItemCommand) error {

	item, err := q.validateAndMarshal(itemCommand)
	if err != nil {
		return errors.New("failed to create deletion request")
	}

	_, err = q.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &q.tableName,
		Key:       item,
	})
	return err
}

func (q *QueryService) ListItems(ctx context.Context, chatId string) ([]ItemCommand, error) {
	if len(chatId) < 9 || 11 < len(chatId) {
		return nil, errors.New("chat id is not matching the required format")
	}
	_, err := strconv.Atoi(chatId)
	if err != nil {
		return nil, errors.New("chat id is not numeric")
	}

	keyCond := expression.Key("chatId").Equal(expression.Value(chatId))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(q.client, &dynamodb.QueryInput{
		TableName:                 &q.tableName,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	var items []ItemCommand

	for queryPaginator.HasMorePages() {
		response, err := queryPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var itemsInPage []ItemCommand
		err = attributevalue.UnmarshalListOfMaps(response.Items, &items)
		if err != nil {
			return nil, err
		}
		items = append(items, itemsInPage...)
	}

	return items, nil
}
