package types

type User struct {
	ID          string  `json:"id" dynamodbav:"id"`
	Login       string `json:"login" dynamodbav:"login"`
	CreatedAt   int64  `json:"created_at" dynamodbav:"created_at"`
	LastLoginAt int64  `json:"last_login_at" dynamodbav:"last_login_at"`
	IsAdmin     bool   `json:"isAdmin" dynamodbav:"is_admin"`
}
