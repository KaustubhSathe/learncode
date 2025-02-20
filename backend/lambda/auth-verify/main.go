package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/types"
	"learncode/backend/utils"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dynamoClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func getUserFromDynamoDB(ctx context.Context, githubID string) (*types.User, error) {
	tableName := os.Getenv("USERS_TABLE")

	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: githubID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user from DynamoDB: %v", err)
	}

	if result.Item == nil {
		return nil, nil
	}

	var user types.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %v", err)
	}

	return &user, nil
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get token from Authorization header
	authHeader := event.Headers["Authorization"]
	if authHeader == "" {
		authHeader = event.Headers["authorization"]
	}

	if authHeader == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Extract token from Bearer format
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "Invalid token format"}`,
		}, nil
	}
	token := parts[1]

	// Verify token with GitHub
	githubUser, err := utils.GetGithubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}

	// Get user's admin status from DynamoDB
	dbUser, err := getUserFromDynamoDB(ctx, githubUser.ID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to get user: %v"}`, err),
		}, nil
	}

	// Set admin status from DB (false if user doesn't exist)
	if dbUser != nil {
		githubUser.IsAdmin = dbUser.IsAdmin
	} else {
		githubUser.IsAdmin = false
	}

	// Return user info
	userJSON, err := json.Marshal(githubUser)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to marshal user: %v"}`, err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(userJSON),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
