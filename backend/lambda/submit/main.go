package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/db"
	"learncode/backend/types"
	"learncode/backend/utils"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
)

type SubmitRequest struct {
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Code      string `json:"code"`
	Type      string `json:"type"`
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var req SubmitRequest
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}

	// Validate language
	if !isValidLanguage(req.Language) {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Invalid language. Supported languages: nodejs, cpp, java, python"}`,
		}, nil
	}

	// Validate token
	authToken := event.Headers["Authorization"]
	if authToken == "" {
		authToken = event.Headers["authorization"]
	}

	if authToken == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Clean token before verification
	authToken = strings.TrimPrefix(authToken, "Bearer ")

	// Verify token with GitHub
	githubUser, err := utils.GetGithubUser(authToken)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}

	// Create submission record
	submission := types.Submission{
		ID:        uuid.New().String(),
		UserID:    githubUser.ID,
		ProblemID: req.ProblemID,
		Language:  req.Language,
		Code:      req.Code,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Type:      req.Type,
	}

	// Save to DynamoDB
	if err := db.SaveSubmission(ctx, &submission); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to save submission: %v"}`, err),
		}, nil
	}

	// Publish to Momento topic for processing
	if err := utils.PublishToMomento(ctx, submission); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to publish submission: %v"}`, err),
		}, nil
	}

	// Return the submission ID
	responseBody, err := json.Marshal(map[string]interface{}{
		"submission": submission,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to create response: %v"}`, err),
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

func isValidLanguage(lang string) bool {
	validLanguages := map[string]bool{
		"nodejs": true,
		"cpp":    true,
		"java":   true,
		"python": true,
	}
	return validLanguages[lang]
}

func main() {
	lambda.Start(handleRequest)
}
