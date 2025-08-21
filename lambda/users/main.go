package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

type eventData struct {
	PK   string
	SK   string
	Data string
}

var tableName = os.Getenv("AUTHLOG_TABLENAME")

func handleAuthEvents(ctx context.Context, event json.RawMessage) (json.RawMessage, error) {

	id := uuid.NewString()
	tstamp := time.Now().Format(time.RFC3339)

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error loading aws config: %w", err)
	}
	client := dynamodb.NewFromConfig(cfg)
	data := eventData{
		PK:   id,
		SK:   tstamp,
		Data: string(event),
	}
	row, err := attributevalue.MarshalMap(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data to map: %w", err)
	}

	_, err = client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      row,
	})
	if err != nil {
		return nil, fmt.Errorf("error inserting row in table: %w", err)
	}
	return event, nil
}

func main() {

	lambda.Start(handleAuthEvents)
}
