package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/db"
	"learncode/backend/utils"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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

	// Verify token with GitHub
	githubUser, err := utils.GetGithubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}

	// Get submission ID from path parameters
	submissionID := event.PathParameters["id"]
	if submissionID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Submission ID is required"}`,
		}, nil
	}

	// Get submission from DynamoDB
	submission, err := db.GetSubmission(ctx, submissionID, githubUser.ID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       fmt.Sprintf(`{"error": "Submission not found: %v"}`, err),
		}, nil
	}

	// Return submission
	responseBody, err := json.Marshal(submission)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to marshal response: %v"}`, err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
