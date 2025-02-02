package main

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/types"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/momentohq/client-sdk-go/auth"
	"github.com/momentohq/client-sdk-go/config"
	"github.com/momentohq/client-sdk-go/momento"
)

type SubmitRequest struct {
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Code      string `json:"code"`
}

func verifyGitHubToken(token string) (string, error) {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	// Make request to GitHub API to verify token
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid token")
	}

	// Parse the response to get user ID
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return fmt.Sprintf("%d", user.ID), nil
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Received submit request with headers: %+v\n", event.Headers)
	fmt.Printf("Request body: %s\n", event.Body)

	// Parse request body
	var req SubmitRequest
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		fmt.Printf("Error parsing request body: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}

	fmt.Printf("Parsed request: ProblemID=%s, Language=%s, Code length=%d\n",
		req.ProblemID, req.Language, len(req.Code))

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
	fmt.Printf("Auth token present: %v\n", authToken != "")

	if authToken == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Verify token with GitHub
	userID, err := verifyGitHubToken(authToken)
	if err != nil {
		fmt.Printf("Token verification failed: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Invalid token: %v"}`, err),
		}, nil
	}

	fmt.Printf("Token verified for user: %s\n", userID)

	// Create submission record
	submission := types.Submission{
		ID:        uuid.New().String(),
		UserID:    userID,
		ProblemID: req.ProblemID,
		Language:  req.Language,
		Code:      req.Code,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// Save to DynamoDB
	if err := saveSubmission(ctx, &submission); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to save submission: %v"}`, err),
		}, nil
	}

	// Publish to Momento topic
	if err := publishToMomento(ctx, submission); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to publish to Momento: %v"}`, err),
		}, nil
	}

	fmt.Println("Returning successful response")
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: fmt.Sprintf(`{"submission_id": "%s", "status": "pending"}`, submission.ID),
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

func saveSubmission(ctx context.Context, submission *types.Submission) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)
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

func publishToMomento(ctx context.Context, submission types.Submission) error {
	credentialProvider, err := auth.NewEnvMomentoTokenProvider("MOMENTO_AUTH_TOKEN")
	if err != nil {
		log.Printf("Error loading Momento auth token: %v", err)
		return fmt.Errorf("failed to load Momento auth token: %v", err)
	}

	client, err := momento.NewTopicClient(config.TopicsDefault(), credentialProvider)
	if err != nil {
		return fmt.Errorf("failed to create Momento client: %v", err)
	}
	defer client.Close()

	// Create topic name based on language
	topicName := fmt.Sprintf("learncode-%s", submission.Language)

	// Publish submission to appropriate topic
	message, _ := json.Marshal(submission)
	if _, err := client.Publish(ctx, &momento.TopicPublishRequest{
		CacheName: "learncode-cache",
		TopicName: topicName,
		Value:     momento.Bytes(message),
	}); err != nil {
		return fmt.Errorf("failed to publish to topic %s: %v", topicName, err)
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
