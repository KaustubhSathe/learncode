package main

import (
	"bytes"
	"context"
	"encoding/base64"
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
	fmt.Println("Received request:", event.Body)

	// Parse webhook payload from Momento
	var payload struct {
		Cache               string `json:"cache"`
		Topic               string `json:"topic"`
		EventTimestamp      int64  `json:"event_timestamp"`
		PublishTimestamp    int64  `json:"publish_timestamp"`
		TopicSequenceNumber int    `json:"topic_sequence_number"`
		TokenID             string `json:"token_id"`
		Binary              string `json:"binary"`
	}
	if err := json.Unmarshal([]byte(event.Body), &payload); err != nil {
		fmt.Printf("Error parsing payload: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid payload: %v"}`, err),
		}, nil
	}
	fmt.Printf("Decoded Momento message: %+v\n", payload)

	// Decode base64 binary
	submissionJSON, err := base64.StdEncoding.DecodeString(payload.Binary)
	if err != nil {
		fmt.Printf("Error decoding base64: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid base64: %v"}`, err),
		}, nil
	}

	var submission types.Submission
	if err := json.Unmarshal(submissionJSON, &submission); err != nil {
		fmt.Printf("Error parsing submission: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error": "Invalid submission: %v"}`, err),
		}, nil
	}
	fmt.Printf("Processing submission: %+v\n", submission)

	// Update submission status to "running"
	if err := db.UpdateSubmissionStatus(ctx, submission.ID, "running", nil); err != nil {
		fmt.Printf("Error updating status to running: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to update status: %v"}`, err),
		}, nil
	}
	fmt.Println("Updated status to running")

	// Execute C++ code – call a helper function
	fmt.Printf("Starting C++ execution for submission %s\n", submission.ID)
	result, err := executeCpp(ctx, submission.Code, submission.ProblemID)
	if err != nil {
		fmt.Printf("Execution failed for submission %s: %v\n", submission.ID, err)
		errStr := err.Error()
		if updateErr := db.UpdateSubmissionStatus(ctx, submission.ID, "error", &errStr); updateErr != nil {
			fmt.Printf("Failed to update error status: %v\n", updateErr)
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       fmt.Sprintf(`{"error": "%s"}`, errStr),
		}, nil
	}
	fmt.Printf("Execution completed successfully for submission %s\n", submission.ID)

	// Update status to "completed"
	fmt.Printf("Updating final status for submission %s\n", submission.ID)
	if err := db.UpdateSubmissionStatus(ctx, submission.ID, "completed", &result); err != nil {
		fmt.Printf("Failed to update final status: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to update status: %v"}`, err),
		}, nil
	}
	fmt.Printf("Successfully completed submission %s\n", submission.ID)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"status": "success"}`,
	}, nil
}

func executeCpp(ctx context.Context, code string, problemID string) (string, error) {
	tmpDir, err := os.MkdirTemp("/tmp", "cpp-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	codePath := filepath.Join(tmpDir, "solution.cpp")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write code: %v", err)
	}

	fmt.Printf("Wrote code to file: %s\n", codePath)

	// Get problem from DynamoDB
	problem, err := db.GetProblem(ctx, problemID)
	if err != nil {
		return "", fmt.Errorf("failed to get problem: %v", err)
	}

	exePath := filepath.Join(tmpDir, "solution")
	fmt.Printf("Retrieved problem: %+v\n", problem)

	fmt.Println("Compiling code...")
	cmd := exec.Command("/opt/gcc/bin/g++", "-B/opt/gcc/bin", "-std=c++14", "-o", exePath, codePath)
	cmd.Env = append(os.Environ(),
		"LD_LIBRARY_PATH=/opt/gcc/bin",
		"CPLUS_INCLUDE_PATH=/opt/gcc/include",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Compilation error: %v, output: %s\n", err, string(out))
		return "", fmt.Errorf("compilation error: %v, output: %s", err, string(out))
	}
	fmt.Println("Compilation successful")

	// Run the compiled program
	fmt.Println("Executing code...")
	cmd = exec.Command(exePath)
	cmd.Stdin = strings.NewReader(problem.Input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Execution error: %v, stderr: %s\n", err, stderr.String())
		return "", fmt.Errorf("execution error: %v, output: %s", err, stderr.String())
	}
	fmt.Printf("Execution complete. stdout: %s, stderr: %s\n", stdout.String(), stderr.String())

	return strings.TrimSpace(stdout.String()), nil
}

func main() {
	lambda.Start(handleRequest)
}
