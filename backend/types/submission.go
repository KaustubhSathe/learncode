package types

type Submission struct {
	ID          string  `json:"id" dynamodbav:"id"`
	UserID      string  `json:"user_id" dynamodbav:"user_id"`
	ProblemID   string  `json:"problem_id" dynamodbav:"problem_id"`
	Language    string  `json:"language" dynamodbav:"language"`
	Code        string  `json:"code" dynamodbav:"code"`
	Status      string  `json:"status" dynamodbav:"status"` // pending, running, completed, error
	CreatedAt   int64   `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   int64   `json:"updated_at" dynamodbav:"updated_at"`
	Result      *string `json:"result,omitempty" dynamodbav:"result,omitempty"`
	Type        string  `json:"type" dynamodbav:"type"` // RUN, SUBMIT
} 