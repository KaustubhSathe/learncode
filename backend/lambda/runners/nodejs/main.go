package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"learncode/backend/db"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type RunRequest struct {
	Cache  string `json:"cache"`
	Topic  string `json:"topic"`
	Binary string `json:"binary"`
}

type Submission struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Code      string `json:"code"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type Problem struct {
	ID     string `json:"id" dynamodbav:"id"`
	Input  string `json:"input" dynamodbav:"input"`
	Output string `json:"output" dynamodbav:"output"`
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Received request: %+v\n", event)

	// Parse request body
	var req RunRequest
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		fmt.Printf("Error parsing request body: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid request body: %v"}`, err),
		}, nil
	}

	// Decode base64 binary data
	binaryData, err := base64.StdEncoding.DecodeString(req.Binary)
	if err != nil {
		fmt.Printf("Error decoding binary data: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid binary data: %v"}`, err),
		}, nil
	}

	// Parse submission data
	var submission Submission
	if err := json.Unmarshal(binaryData, &submission); err != nil {
		fmt.Printf("Error parsing submission data: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid submission data: %v"}`, err),
		}, nil
	}

	fmt.Printf("Parsed submission: ID=%s, ProblemID=%s, Code length=%d\n",
		submission.ID, submission.ProblemID, len(submission.Code))

	fmt.Printf("Submission code: %s\n", submission.Code)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "nodejs-runner-*")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to create temp directory: %v"}`, err),
		}, nil
	}
	fmt.Printf("Created temp directory: %s\n", tmpDir)
	defer os.RemoveAll(tmpDir)

	// Write code to file
	codePath := filepath.Join(tmpDir, "solution.js")
	if err := os.WriteFile(codePath, []byte(submission.Code), 0644); err != nil {
		fmt.Printf("Error writing code file: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to write code file: %v"}`, err),
		}, nil
	}
	fmt.Printf("Wrote code to file: %s\n", codePath)

	// Get problem from DynamoDB
	problem, err := getProblem(ctx, submission.ProblemID)
	if err != nil {
		fmt.Printf("Error getting problem: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to get problem: %v"}`, err),
		}, nil
	}
	fmt.Printf("Retrieved problem: %+v\n", problem)

	// Run code
	cmd := exec.Command("/opt/nodejs/bin/node", codePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("Executing code...")
	err = cmd.Run()
	fmt.Printf("Execution complete. stdout: %s, stderr: %s\n", stdout.String(), stderr.String())

	if err != nil {
		fmt.Printf("Error executing code: %v\n", err)
		// Update submission status
		errOutput := stderr.String()
		if err := db.UpdateSubmissionStatus(ctx, submission.ID, "error", &errOutput); err != nil {
			fmt.Printf("Error updating submission status: %v\n", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to update submission: %v"}`, err),
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       fmt.Sprintf(`{"status": "error", "output": %q}`, stderr.String()),
		}, nil
	}

	output := stdout.String()
	// Trim whitespace and newlines from both outputs
	cleanOutput := strings.TrimSpace(output)
	cleanExpected := strings.TrimSpace(problem.Output)

	fmt.Printf("Code execution successful. Output: %s\n", output)

	// Compare output with expected output
	fmt.Printf("Clean Output: %q\n", cleanOutput)
	fmt.Printf("Clean Expected: %q\n", cleanExpected)
	if cleanOutput == cleanExpected {
		fmt.Println("Output matches expected output")
		if err := db.UpdateSubmissionStatus(ctx, submission.ID, "success", &output); err != nil {
			fmt.Printf("Error updating submission status: %v\n", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to update submission: %v"}`, err),
			}, nil
		}
	} else {
		fmt.Printf("Output does not match expected output. Expected: %s, Got: %s\n", problem.Output, output)
		if err := db.UpdateSubmissionStatus(ctx, submission.ID, "wrong_answer", &output); err != nil {
			fmt.Printf("Error updating submission status: %v\n", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf(`{"error": "Failed to update submission: %v"}`, err),
			}, nil
		}
	}

	fmt.Println("Request handling complete")
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf(`{"status": "completed", "output": %q}`, output),
	}, nil
}

func executeNodeJS(ctx context.Context, code string, problemID string) (string, error) {
	// Write the JS code to a temporary file and execute it using Node.js.
	tmpDir, err := os.MkdirTemp("/tmp", "nodejs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	codePath := filepath.Join(tmpDir, "solution.js")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write code: %v", err)
	}

	// Execute the code using "node"
	cmd := exec.CommandContext(ctx, "node", codePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execution error: %v, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func getProblem(ctx context.Context, id string) (*Problem, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("PROBLEMS_TABLE")),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	var problem Problem
	if err := attributevalue.UnmarshalMap(result.Item, &problem); err != nil {
		return nil, err
	}
	return &problem, nil
}

func main() {
	lambda.Start(handleRequest)
}
