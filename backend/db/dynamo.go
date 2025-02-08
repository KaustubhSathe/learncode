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

func UpdateSubmissionStatus(ctx context.Context, problemId string, submissionId string, status string, result *string) error {
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
			"problem_id":    &dbtypes.AttributeValueMemberS{Value: problemId},
			"submission_id": &dbtypes.AttributeValueMemberS{Value: submissionId},
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

func GetUser(ctx context.Context, userID string) (*types.User, error) {
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USERS_TABLE")),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if result.Item == nil {
		return nil, nil // User not found, but not an error
	}

	var user types.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %v", err)
	}

	return &user, nil
}

func GetSubmissionsByProblemAndType(ctx context.Context, submissionID string, problemID string, submissionType string, userId string) ([]types.Submission, error) {
	// Initialize DynamoDB client
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("Failed to load AWS config: %v\n", err)
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Query submissions using begins_with and filter by type
	input := &dynamodb.QueryInput{
		TableName:              aws.String("SubmissionsV2"),
		KeyConditionExpression: aws.String("problem_id = :problem_id AND begins_with(submission_id, :prefix)"),
		ExpressionAttributeValues: map[string]dbtypes.AttributeValue{
			":problem_id": &dbtypes.AttributeValueMemberS{Value: problemID},
			":prefix":     &dbtypes.AttributeValueMemberS{Value: submissionID},
			":type":       &dbtypes.AttributeValueMemberS{Value: submissionType},
			":userId":     &dbtypes.AttributeValueMemberS{Value: userId},
		},
		FilterExpression: aws.String("#t = :type AND user_id = :userId"),
		ExpressionAttributeNames: map[string]string{
			"#t": "type",
		},
	}

	fmt.Printf("Executing DynamoDB query with input: %+v\n", input)
	result, err := client.Query(ctx, input)
	if err != nil {
		fmt.Printf("DynamoDB query failed: %v\n", err)
		return nil, fmt.Errorf("failed to query submissions: %v", err)
	}

	fmt.Printf("Query returned %d items\n", len(result.Items))

	// Convert DynamoDB items to Submission structs
	var submissions []types.Submission
	for _, item := range result.Items {
		var submission types.Submission
		if err := attributevalue.UnmarshalMap(item, &submission); err != nil {
			fmt.Printf("Failed to unmarshal item: %v\n", err)
			return nil, fmt.Errorf("failed to unmarshal submission: %v", err)
		}
		submissions = append(submissions, submission)
	}

	fmt.Printf("Successfully processed %d submissions\n", len(submissions))
	return submissions, nil
}
