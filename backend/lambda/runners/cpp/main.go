package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"learncode/backend/db"
	"learncode/backend/types"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/momentohq/client-sdk-go/auth"
	momentoconfig "github.com/momentohq/client-sdk-go/config"
	"github.com/momentohq/client-sdk-go/momento"
)

func handleRequest(ctx context.Context) error {
	credentialProvider, err := auth.NewEnvMomentoTokenProvider("MOMENTO_API_KEY")
	if err != nil {
		log.Fatalf("Error loading Momento API key: %v", err)
	}

	client, err := momento.NewTopicClient(momentoconfig.TopicsDefault(), credentialProvider)
	if err != nil {
		return fmt.Errorf("failed to create Momento client: %v", err)
	}
	defer client.Close()

	subscription, err := client.Subscribe(ctx, &momento.TopicSubscribeRequest{
		CacheName: "default",
		TopicName: "learncode-cpp",
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %v", err)
	}

	for {
		event, err := subscription.Event(ctx)
		if err != nil {
			log.Printf("Error receiving event: %v", err)
			continue
		}

		switch e := event.(type) {
		case momento.TopicItem:
			var submission types.Submission
			if err := json.Unmarshal(e.GetValue().(momento.Bytes), &submission); err != nil {
				fmt.Printf("Error unmarshaling message: %v\n", err)
				continue
			}

			if err := db.UpdateSubmissionStatus(ctx, submission.ID, "running", nil); err != nil {
				fmt.Printf("Error updating status to running: %v\n", err)
				continue
			}

			result, err := executeCPP(submission.Code, submission.ProblemID)
			if err != nil {
				errStr := err.Error()
				db.UpdateSubmissionStatus(ctx, submission.ID, "error", &errStr)
				continue
			}

			if err := db.UpdateSubmissionStatus(ctx, submission.ID, "completed", &result); err != nil {
				fmt.Printf("Error updating status to completed: %v\n", err)
			}

		case momento.TopicHeartbeat:
			fmt.Printf("Received heartbeat\n")

		case momento.TopicDiscontinuity:
			fmt.Printf("Received discontinuity - some messages may have been missed\n")
		}
	}
}

func executeCPP(code string, problemID string) (string, error) {
	problem, err := db.GetProblem(context.Background(), problemID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch problem: %v", err)
	}

	tmpDir, err := os.MkdirTemp("/tmp", "cpp-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write code to file
	codePath := filepath.Join(tmpDir, "code.cpp")
	wrappedCode := fmt.Sprintf(`
		#include <iostream>
		#include <string>
		using namespace std;

		%s

		int main() {
			// Your code will read from stdin
			%s
			return 0;
		}
	`, code)

	if err := os.WriteFile(codePath, []byte(wrappedCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write code file: %v", err)
	}

	// Compile the code
	execPath := filepath.Join(tmpDir, "code")
	cmd := exec.Command("g++", "-o", execPath, codePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("compilation failed: %v\nOutput: %s", err, output)
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

	// Run the compiled program
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, execPath)
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
