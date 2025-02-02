package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"learncode/backend/db"
	"learncode/backend/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse webhook payload; expect a "text" field containing the submission JSON.
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(event.Body), &payload); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid payload: %v"}`, err),
		}, nil
	}

	var submission types.Submission
	if err := json.Unmarshal([]byte(payload.Text), &submission); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid submission: %v"}`, err),
		}, nil
	}

	if err := db.UpdateSubmissionStatus(ctx, submission.ID, "running", nil); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to update status: %v"}`, err),
		}, nil
	}

	result, err := executeJava(ctx, submission.Code, submission.ProblemID)
	if err != nil {
		errStr := err.Error()
		db.UpdateSubmissionStatus(ctx, submission.ID, "error", &errStr)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       fmt.Sprintf(`{"error": "%s"}`, errStr),
		}, nil
	}

	if err := db.UpdateSubmissionStatus(ctx, submission.ID, "completed", &result); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to update status: %v"}`, err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"status": "success"}`,
	}, nil
}

func executeJava(ctx context.Context, code string, problemID string) (string, error) {
	tmpDir, err := os.MkdirTemp("/tmp", "java-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	codePath := filepath.Join(tmpDir, "Solution.java")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write code: %v", err)
	}

	// Compile the Java code
	cmd := exec.CommandContext(ctx, "javac", codePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation error: %v, output: %s", err, string(out))
	}

	// Run the compiled code (assuming the main class is Solution)
	cmd = exec.CommandContext(ctx, "java", "-cp", tmpDir, "Solution")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execution error: %v, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func main() {
	lambda.Start(handleRequest)
} 