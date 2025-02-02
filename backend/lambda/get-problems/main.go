package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"learncode/backend/db"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type User struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

func getGitHubUser(token string) (*User, error) {
	fmt.Printf("Fetching GitHub user with token: %s...\n", token[:10])

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("GitHub API response status: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("GitHub API error response: %s\n", string(body))
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	// Parse response
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return nil, fmt.Errorf("failed to parse user info: %v", err)
	}

	fmt.Printf("Successfully fetched user: %s (ID: %d)\n", user.Login, user.ID)
	return &user, nil
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Lambda - All headers: %+v\n", event.Headers)
	fmt.Printf("Lambda - Auth header: %q\n", event.Headers["Authorization"])
	fmt.Printf("Lambda - auth header: %q\n", event.Headers["authorization"])

	// Check both cases since API Gateway might normalize header names
	authToken := event.Headers["Authorization"]
	if authToken == "" {
		authToken = event.Headers["authorization"]
	}

	if authToken == "" {
		fmt.Println("No authorization token provided")
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "No authorization token provided"}`,
		}, nil
	}

	// Extract token from Bearer format
	parts := strings.Split(authToken, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error": "Invalid token format"}`,
		}, nil
	}
	token := parts[1]

	// Get user info from GitHub
	user, err := getGitHubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf(`{"error": "Failed to validate token: %v"}`, err),
		}, nil
	}

	problems, err := db.GetProblems(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to fetch problems: %v"}`, err),
		}, nil
	}

	// Create response with both problems and user
	response := map[string]interface{}{
		"problems": problems,
		"user":     user,
	}

	responseBody, err := json.Marshal(response)
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
