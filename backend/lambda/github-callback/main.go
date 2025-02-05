package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"learncode/backend/types"
	"learncode/backend/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GithubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract code from query parameters
	code := request.QueryStringParameters["code"]
	if code == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "No code provided"}`,
		}, nil
	}

	// Exchange code for access token
	token, err := getAccessToken(code)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to get access token: %v"}`, err),
		}, nil
	}

	// Get user info from GitHub
	githubUser, err := utils.GetGithubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to get user info: %v"}`, err),
		}, nil
	}

	// First check if user exists
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to load AWS config: %v"}`, err),
		}, nil
	}

	client := dynamodb.NewFromConfig(cfg)

	// Get user from DynamoDB
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("USERS_TABLE")),
		Key: map[string]dbtypes.AttributeValue{
			"id": &dbtypes.AttributeValueMemberS{Value: githubUser.ID},
		},
	})

	var shouldSaveUser bool
	if err != nil {
		// Handle actual errors
		if !strings.Contains(err.Error(), "ResourceNotFoundException") {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to check user: %v"}`, err),
			}, nil
		}
		shouldSaveUser = true
	} else {
		// If no item found, save the user
		shouldSaveUser = len(result.Item) == 0
	}

	if shouldSaveUser {
		// Save user to DynamoDB
		user := &types.User{
			ID:          githubUser.ID,
			Login:       githubUser.Login,
			CreatedAt:   time.Now().Unix(),
			LastLoginAt: time.Now().Unix(),
			IsAdmin:     false,
		}

		item, err := attributevalue.MarshalMap(user)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to marshal user: %v"}`, err),
			}, nil
		}

		tableName := os.Getenv("USERS_TABLE")
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to save user: %v"}`, err),
			}, nil
		}
	}

	// Return success with token
	return events.APIGatewayProxyResponse{
		StatusCode: 302, // Redirect status code
		Headers: map[string]string{
			"Location": fmt.Sprintf("%s/auth/callback?token=%s", os.Getenv("FRONTEND_URL"), token),
		},
		Body: "",
	}, nil
}

func getAccessToken(code string) (string, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	// Add Accept header for JSON response
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token",
		strings.NewReader(url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"code":          {code},
		}.Encode()))
	if err != nil {
		return "", err
	}

	// Request JSON response
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Debug response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse GithubTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %v, body: %s", err, string(body))
	}

	return tokenResponse.AccessToken, nil
}

func main() {
	lambda.Start(handleRequest)
}
