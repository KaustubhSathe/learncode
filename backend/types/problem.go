package types

type Problem struct {
	ID          string `json:"id" dynamodbav:"id"`
	Title       string `json:"title" dynamodbav:"title"`
	Description string `json:"description" dynamodbav:"description"`
	Difficulty  string `json:"difficulty" dynamodbav:"difficulty"`
	CreatedAt   int64  `json:"created_at" dynamodbav:"created_at"`                     // Unix timestamp
	UpdatedAt   int64  `json:"updated_at" dynamodbav:"updated_at"`                     // Unix timestamp
	DeletedAt   *int64 `json:"deleted_at,omitempty" dynamodbav:"deleted_at,omitempty"` // Optional Unix timestamp
	Input       string `json:"input" dynamodbav:"input"`
	Output      string `json:"output" dynamodbav:"output"`
	ExampleInput       string `json:"example_input" dynamodbav:"example_input"`
	ExampleOutput      string `json:"example_output" dynamodbav:"example_output"`
}
