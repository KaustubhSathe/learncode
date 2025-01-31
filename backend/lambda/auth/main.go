package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type GithubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

type GithubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Handle different paths
	switch event.Path {
	case "/auth/github":
		return handleGithubInit(event)
	case "/auth/github/callback":
		return handleGithubCallback(event)
	default:
		// Handle as authorizer request
		return handleAuthorizer(event)
	}
}

func handleGithubInit(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirectURI := fmt.Sprintf("%s/auth/github/callback", event.RequestContext.DomainName)
	
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user",
		clientID,
		url.QueryEscape(redirectURI),
	)

	return events.APIGatewayProxyResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": authURL,
		},
	}, nil
}

func handleGithubCallback(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	code := event.QueryStringParameters["code"]
	if code == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "No code provided",
		}, nil
	}

	// Exchange code for token
	token, err := getGithubToken(code)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error getting token: %v", err),
		}, nil
	}

	// Get user info
	user, err := getGithubUser(token.AccessToken)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Error getting user: %v", err),
		}, nil
	}

	// Create response with token and user info
	response := map[string]interface{}{
		"token": token.AccessToken,
		"user":  user,
	}

	responseJSON, _ := json.Marshal(response)

	// In production, redirect to frontend with token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL != "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 302,
			Headers: map[string]string{
				"Location": fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, token.AccessToken),
			},
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseJSON),
	}, nil
}

func getGithubToken(code string) (*GithubTokenResponse, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	// Exchange code for token
	resp, err := http.PostForm("https://github.com/login/oauth/access_token",
		url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"code":          {code},
		})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token GithubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func getGithubUser(token string) (*GithubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func handleAuthorizer(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token := event.Headers["Authorization"]
	if token == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "No token provided",
		}, nil
	}

	// Validate GitHub token
	user, err := getGithubUser(token)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       fmt.Sprintf("Invalid token: %v", err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: fmt.Sprintf(`{"userId": %d, "login": "%s"}`, user.ID, user.Login),
	}, nil
}

func main() {
	lambda.Start(handleRequest)
} 