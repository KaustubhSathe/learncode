package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"learncode/backend/types"
	"log"
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

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var submitReq SubmitRequest
	if err := json.Unmarshal([]byte(request.Body), &submitReq); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}

	// Validate language
	if !isValidLanguage(submitReq.Language) {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Invalid language. Supported languages: nodejs, cpp, java, python"}`,
		}, nil
	}

	// Get user ID from token
	userID := getUserIDFromToken(request.Headers["Authorization"])
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "Invalid token"}`,
		}, nil
	}

	// Create submission record
	submission := types.Submission{
		ID:        uuid.New().String(),
		UserID:    userID,
		ProblemID: submitReq.ProblemID,
		Language:  submitReq.Language,
		Code:      submitReq.Code,
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
	credentialProvider, err := auth.NewEnvMomentoTokenProvider("MOMENTO_API_KEY")
	if err != nil {
		log.Fatalf("Error loading Momento API key: %v", err)
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
		CacheName: "default",
		TopicName: topicName,
		Value:     momento.Bytes(message),
	}); err != nil {
		return fmt.Errorf("failed to publish to topic %s: %v", topicName, err)
	}

	return nil
}

func getUserIDFromToken(token string) string {
	if token == "" {
		return ""
	}

	// Parse JWT token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	// Parse the JSON payload
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	return claims.Sub
}

func main() {
	lambda.Start(handleRequest)
}
