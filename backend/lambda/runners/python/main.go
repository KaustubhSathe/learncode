package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"learncode/backend/db"
	"learncode/backend/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse webhook payload; expect a 'text' field containing the submission JSON.
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

	// Update status to running
	if err := db.UpdateSubmissionStatus(ctx, submission.ProblemID, submission.SubmissionID, "running", nil); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to update status: %v"}`, err),
		}, nil
	}

	// Execute code
	result, err := executePython(ctx, submission.Code, submission.ProblemID)
	if err != nil {
		errStr := err.Error()
		db.UpdateSubmissionStatus(ctx, submission.ProblemID, submission.SubmissionID, "error", &errStr)
		return events.APIGatewayProxyResponse{
			StatusCode: 200, // Still return 200 as the webhook was processed
			Body:       fmt.Sprintf(`{"error": "%s"}`, errStr),
		}, nil
	}

	// Update status to completed
	if err := db.UpdateSubmissionStatus(ctx, submission.ProblemID, submission.SubmissionID, "completed", &result); err != nil {
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

func executePython(ctx context.Context, code string, problemID string) (string, error) {
	problem, err := db.GetProblem(context.Background(), problemID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch problem: %v", err)
	}

	tmpDir, err := os.MkdirTemp("/tmp", "python-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write code to file
	codePath := filepath.Join(tmpDir, "solution.py")
	wrappedCode := fmt.Sprintf(`
import sys

%s

# Your code will read from sys.stdin
`, code)

	if err := os.WriteFile(codePath, []byte(wrappedCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write code file: %v", err)
	}

	// Create input file
	inputPath := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputPath, []byte(problem.Input), 0644); err != nil {
		return "", fmt.Errorf("failed to write input file: %v", err)
	}

	// Create output file
	outputPath := filepath.Join(tmpDir, "output.txt")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Run the Python script
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "solution.py")
	cmd.Dir = tmpDir

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()

	cmd.Stdin = inputFile
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("execution timed out")
		}
		return "", fmt.Errorf("execution failed: %v", err)
	}

	// Read and compare output
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %v", err)
	}

	actualOutput := strings.TrimSpace(string(output))
	expectedOutput := strings.TrimSpace(problem.Output)

	if actualOutput != expectedOutput {
		return "", fmt.Errorf("output mismatch\nExpected:\n%s\nGot:\n%s", expectedOutput, actualOutput)
	}

	return actualOutput, nil
}

func main() {
	lambda.Start(handleRequest)
}
