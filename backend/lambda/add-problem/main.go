package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/types"
	"learncode/backend/utils"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

var dynamoClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

type CreateProblemRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Difficulty    string `json:"difficulty"`
	Input         string `json:"input"`
	Output        string `json:"output"`
	ExampleInput  string `json:"example_input"`
	ExampleOutput string `json:"example_output"`
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Check authorization
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

	// Verify token with GitHub and check admin status
	githubUser, err := utils.GetGithubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}

	// Get user's admin status from DynamoDB
	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USERS_TABLE")),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: githubUser.ID},
		},
	})

	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to get user from database: %v"}`, err),
		}, nil
	}

	var dbUser types.User
	if err := attributevalue.UnmarshalMap(result.Item, &dbUser); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to parse user data: %v"}`, err),
		}, nil
	}

	if !dbUser.IsAdmin {
		return events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       `{"error": "Unauthorized: Admin access required"}`,
		}, nil
	}

	// Parse request body
	var req CreateProblemRequest
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}

	// Validate required fields
	if req.Title == "" || req.Description == "" || req.Difficulty == "" ||
		req.Input == "" || req.Output == "" || req.ExampleInput == "" || req.ExampleOutput == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "All fields are required"}`,
		}, nil
	}

	// Validate difficulty
	if req.Difficulty != "Easy" && req.Difficulty != "Medium" && req.Difficulty != "Hard" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Difficulty must be Easy, Medium, or Hard"}`,
		}, nil
	}

	// Create problem
	now := time.Now().Unix()
	problem := &types.Problem{
		ID:            fmt.Sprintf("prob-%s", uuid.New().String()[:8]),
		Title:         req.Title,
		Description:   req.Description,
		Difficulty:    req.Difficulty,
		Input:         req.Input,
		Output:        req.Output,
		ExampleInput:  req.ExampleInput,
		ExampleOutput: req.ExampleOutput,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Save to DynamoDB
	item, err := attributevalue.MarshalMap(problem)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to marshal problem: %v"}`, err),
		}, nil
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("PROBLEMS_TABLE")),
		Item:      item,
	})

	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to save problem: %v"}`, err),
		}, nil
	}

	// Return success response
	response := map[string]interface{}{
		"message": "Problem created successfully",
		"problem": problem,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to marshal response: %v"}`, err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 201,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
