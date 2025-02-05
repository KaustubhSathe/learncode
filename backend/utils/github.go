package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"learncode/backend/types"
	"net/http"
	"time"
)

func GetGithubUser(token string) (*types.User, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var user types.User
	var userMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userMap); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %v", err)
	}

	user.ID = fmt.Sprintf("%d", int64(userMap["id"].(float64))) // Convert float64 to string
	user.Login = userMap["login"].(string)
	createdAt, err := time.Parse(time.RFC3339, userMap["created_at"].(string))
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %v", err)
	}
	user.CreatedAt = createdAt.Unix()

	return &user, nil
}
