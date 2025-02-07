package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/db"
	"learncode/backend/types"
	"learncode/backend/utils"
	"log"
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
	log.Printf("Received request: %+v", event)

	// Parse request body
	var req SubmitRequest
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		log.Printf("Failed to parse request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}
	log.Printf("Parsed request: %+v", req)

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
	log.Printf("Auth token: %v", authToken)
	log.Printf("Auth token present: %v", authToken != "")

	if authToken == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Clean token before verification
	authToken = strings.TrimPrefix(authToken, "Bearer ")
	log.Printf("Token after cleaning: %s...", authToken[:10])

	// Verify token with GitHub
	githubUser, err := utils.GetGithubUser(authToken)
	if err != nil {
		log.Printf("Failed to verify GitHub token: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to verify token: %v"}`, err),
		}, nil
	}
	log.Printf("GitHub user verified: %s", githubUser.ID)

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
	log.Printf("Created submission record: %+v", submission)

	// Save to DynamoDB
	if err := db.SaveSubmission(ctx, &submission); err != nil {
		log.Printf("Failed to save submission: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to save submission: %v"}`, err),
		}, nil
	}
	log.Printf("Saved submission to DynamoDB")

	// Publish to Momento topic for processing
	if err := utils.PublishToMomento(ctx, submission); err != nil {
		log.Printf("Failed to publish to Momento: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to publish submission: %v"}`, err),
		}, nil
	}
	log.Printf("Published to Momento successfully")

	// Return the submission ID
	responseBody, err := json.Marshal(map[string]interface{}{
		"submission": submission,
	})
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to create response: %v"}`, err),
		}, nil
	}

	log.Printf("Returning successful response")
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
