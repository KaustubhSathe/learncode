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
	fmt.Printf("Received request with path parameters: %+v\n", event.PathParameters)
	fmt.Printf("Query parameters: %+v\n", event.QueryStringParameters)

	// Validate token
	authToken := event.Headers["Authorization"]
	if authToken == "" {
		authToken = event.Headers["authorization"]
	}

	fmt.Printf("Auth token present: %v\n", authToken != "")

	if authToken == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Clean token
	authToken = strings.TrimPrefix(authToken, "Bearer ")

	// Verify token
	fmt.Println("Verifying GitHub token...")
	githubUser, err := utils.GetGithubUser(authToken)
	if err != nil {
		fmt.Printf("GitHub token verification failed: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}
	fmt.Printf("GitHub user verified: %s\n", githubUser.ID)

	// Get submission ID from path parameters
	submissionId := event.QueryStringParameters["submission_id"]
	problemId := event.QueryStringParameters["problem_id"]
	fmt.Printf("Submission ID from path: %s\n", submissionId)
	fmt.Printf("Problem ID from path: %s\n", problemId)

	if submissionId == "" {
		submissionId = "SUBMISSION#"
	}

	// Get submission type from query parameters
	submissionType := event.QueryStringParameters["type"]

	if submissionType == "" {
		submissionType = "RUN" // Default to RUN if not specified
	}
	fmt.Printf("Submission type: %s\n", submissionType)

	// Get submissions from DynamoDB
	submissions, err := db.GetSubmissionsByProblemAndType(ctx, submissionId, problemId, submissionType, githubUser.ID)
	if err != nil {
		fmt.Printf("Failed to fetch submissions: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "%v"}`, err),
		}, nil
	}
	fmt.Printf("Found %d submissions\n", len(submissions))

	// Return submissions
	responseBody, err := json.Marshal(map[string]interface{}{
		"submissions": submissions,
	})
	if err != nil {
		fmt.Printf("Failed to marshal response: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to marshal response: %v"}`, err),
		}, nil
	}

	fmt.Println("Successfully prepared response")
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
