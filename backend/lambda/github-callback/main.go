package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"learncode/backend/db"
	"learncode/backend/types"
	"learncode/backend/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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
	token, err := utils.GetAccessToken(code)
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

	// Get user from DynamoDB
	existingUser, err := db.GetUser(ctx, githubUser.ID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to check user: %v"}`, err),
		}, nil
	}

	if existingUser == nil {
		// Save user to DynamoDB
		user := &types.User{
			ID:          githubUser.ID,
			Login:       githubUser.Login,
			CreatedAt:   time.Now().Unix(),
			LastLoginAt: time.Now().Unix(),
			IsAdmin:     false,
		}
		err = db.SaveUser(ctx, user)
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

func main() {
	lambda.Start(handleRequest)
}
