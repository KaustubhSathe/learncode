package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"learncode/backend/types"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func GetGithubUser(token string) (*types.User, error) {
	log.Printf("Starting GitHub user verification")

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	log.Printf("Request headers set, Authorization: Bearer %s...", token[:10])

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make request to GitHub: %v", err)
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("GitHub API response status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("GitHub API error response: %s", string(body))
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var user types.User
	var userMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userMap); err != nil {
		log.Printf("Failed to decode GitHub response: %v", err)
		return nil, fmt.Errorf("failed to parse user info: %v", err)
	}
	log.Printf("GitHub response decoded: %+v", userMap)

	user.ID = fmt.Sprintf("%d", int64(userMap["id"].(float64))) // Convert float64 to string
	user.Login = userMap["login"].(string)
	createdAt, err := time.Parse(time.RFC3339, userMap["created_at"].(string))
	if err != nil {
		log.Printf("Failed to parse created_at timestamp: %v", err)
		return nil, fmt.Errorf("failed to parse created_at: %v", err)
	}
	user.CreatedAt = createdAt.Unix()

	log.Printf("Successfully verified GitHub user: %s (%s)", user.Login, user.ID)
	return &user, nil
}

type GithubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func GetAccessToken(code string) (string, error) {
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
