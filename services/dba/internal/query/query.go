package query

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ClientIdQueryItem struct {
	ClientId string `dynamodbav:"clientId"`
	Query    string `dynamodbav:"query"`
}

type QueryService struct {
	client    dynamodb.Client
	tableName string
}

func NewQueryService(client dynamodb.Client, tableName string) QueryService {
	return QueryService{
		client:    client,
		tableName: tableName,
	}
}

func (qs *QueryService) GetQueries(ctx context.Context, clientIds []string) ([]ClientIdQueryItem, error) {

	var queries []ClientIdQueryItem

	// Only fits scenarios with few clients
	for _, clientId := range clientIds {

		keyCond := expression.Key("chatId").Equal(expression.Value(clientId))

		expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build expression %w", err)
		}

		queryPaginator := dynamodb.NewQueryPaginator(&qs.client, &dynamodb.QueryInput{
			TableName:                 &qs.tableName,
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
		})

		for queryPaginator.HasMorePages() {
			response, err := queryPaginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("an error happened while trying to access next page : %v", err)
			}
			var queryItemsPage []ClientIdQueryItem
			if err := attributevalue.UnmarshalListOfMaps(response.Items, &queryItemsPage); err != nil {
				return nil, err
			}
			queries = append(queries, queryItemsPage...)
		}
	}
	return queries, nil
}
