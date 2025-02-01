package types

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	AccessToken  string `json:"access_token"`
	CreatedAt    int64  `json:"created_at"`
	LastLoginAt  int64  `json:"last_login_at"`
} 