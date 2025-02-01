package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"learncode/backend/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var client *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	client = dynamodb.NewFromConfig(cfg)
}

func UpdateSubmissionStatus(ctx context.Context, id string, status string, result *string) error {
	updateExpression := "SET #status = :status, #updated_at = :updated_at"
	expressionAttributeNames := map[string]string{
		"#status":     "status",
		"#updated_at": "updated_at",
	}
	expressionAttributeValues := map[string]dbtypes.AttributeValue{
		":status":     &dbtypes.AttributeValueMemberS{Value: status},
		":updated_at": &dbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
	}

	if result != nil {
		updateExpression += ", #result = :result"
		expressionAttributeNames["#result"] = "result"
		expressionAttributeValues[":result"] = &dbtypes.AttributeValueMemberS{Value: *result}
	}

	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("SUBMISSIONS_TABLE")),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          &updateExpression,
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	})

	return err
}

func GetProblem(ctx context.Context, problemID string) (*types.Problem, error) {
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("PROBLEMS_TABLE")),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: problemID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("problem not found: %s", problemID)
	}

	var problem types.Problem
	if err := attributevalue.UnmarshalMap(result.Item, &problem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal problem: %v", err)
	}

	return &problem, nil
}

func SaveSubmission(ctx context.Context, submission *types.Submission) error {
	item, err := attributevalue.MarshalMap(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal submission: %v", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("SUBMISSIONS_TABLE")),
		Item:      item,
	})
	return err
}

func GetProblems(ctx context.Context) ([]types.Problem, error) {
	result, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("PROBLEMS_TABLE")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan problems: %v", err)
	}

	var problems []types.Problem
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &problems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal problems: %v", err)
	}

	return problems, nil
}

func SaveUser(ctx context.Context, user *types.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %v", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("USERS_TABLE")),
		Item:      item,
	})
	return err
}
