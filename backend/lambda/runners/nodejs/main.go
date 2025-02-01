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

	"learncode/backend/types"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/momentohq/client-sdk-go/auth"
	"github.com/momentohq/client-sdk-go/config"
	"github.com/momentohq/client-sdk-go/momento"
)

func handleRequest(ctx context.Context) error {
	// Create Momento client
	credentialProvider, err := auth.NewEnvMomentoTokenProvider("MOMENTO_API_KEY")
	if err != nil {
		log.Fatalf("Error loading Momento API key: %v", err)
	}

	client, err := momento.NewTopicClient(config.TopicsDefault(), credentialProvider)
	if err != nil {
		return fmt.Errorf("failed to create Momento client: %v", err)
	}
	defer client.Close()

	// Subscribe to nodejs topic
	subscription, err := client.Subscribe(ctx, &momento.TopicSubscribeRequest{
		CacheName: "default",
		TopicName: "learncode-nodejs",
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %v", err)
	}

	// Process messages
	for {
		event, err := subscription.Event(ctx)
		if err != nil {
			log.Printf("Error receiving event: %v", err)
			continue
		}

		// Handle different event types
		switch e := event.(type) {
		case momento.TopicItem:
			var submission types.Submission
			if err := json.Unmarshal(e.GetValue().(momento.Bytes), &submission); err != nil {
				fmt.Printf("Error unmarshaling message: %v\n", err)
				continue
			}

			// Update status to running
			if err := updateSubmissionStatus(ctx, submission.ID, "running", nil); err != nil {
				fmt.Printf("Error updating status to running: %v\n", err)
				continue
			}

			// Execute code
			result, err := executeNodeJS(submission.Code, submission.ProblemID)
			if err != nil {
				errStr := err.Error()
				updateSubmissionStatus(ctx, submission.ID, "error", &errStr)
				continue
			}

			// Update status to completed
			if err := updateSubmissionStatus(ctx, submission.ID, "completed", &result); err != nil {
				fmt.Printf("Error updating status to completed: %v\n", err)
			}

		case momento.TopicHeartbeat:
			fmt.Printf("Received heartbeat\n")

		case momento.TopicDiscontinuity:
			fmt.Printf("Received discontinuity - some messages may have been missed\n")
		}
	}
}

func updateSubmissionStatus(ctx context.Context, id string, status string, result *string) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	updateExpression := "SET #status = :status, #updated_at = :updated_at"
	expressionAttributeNames := map[string]string{
		"#status":     "status",
		"#updated_at": "updated_at",
	}
	expressionAttributeValues := map[string]dynamodbtypes.AttributeValue{
		":status":     &dynamodbtypes.AttributeValueMemberS{Value: status},
		":updated_at": &dynamodbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix())},
	}

	if result != nil {
		updateExpression += ", #result = :result"
		expressionAttributeNames["#result"] = "result"
		expressionAttributeValues[":result"] = &dynamodbtypes.AttributeValueMemberS{Value: *result}
	}

	_, err = client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("SUBMISSIONS_TABLE")),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          &updateExpression,
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	})

	return err
}

func executeNodeJS(code string, problemID string) (string, error) {
	// Fetch problem details
	problem, err := getProblem(context.Background(), problemID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch problem: %v", err)
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("/tmp", "nodejs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write code to a file with input handling
	codePath := filepath.Join(tmpDir, "code.js")
	wrappedCode := fmt.Sprintf(`
		// Original code
		%s
		
		// Read input
		process.stdin.resume();
		process.stdin.setEncoding('utf-8');
		
		let inputString = '';
		
		process.stdin.on('data', function(inputStdin) {
			inputString += inputStdin;
		});
		
		process.stdin.on('end', function() {
			// Your code will read from this input
			console.log(inputString);
		});
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

	// Execute with Node.js
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", codePath)
	cmd.Dir = tmpDir

	// Setup input and output
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

	// Read the output
	output, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %v", err)
	}

	// Compare with expected output
	actualOutput := strings.TrimSpace(string(output))
	expectedOutput := strings.TrimSpace(problem.Output)

	if actualOutput != expectedOutput {
		return "", fmt.Errorf("output mismatch\nExpected:\n%s\nGot:\n%s", expectedOutput, actualOutput)
	}

	return actualOutput, nil
}

func getProblem(ctx context.Context, problemID string) (*types.Problem, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("PROBLEMS_TABLE")),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: problemID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("problem not found: %s", problemID)
	}

	var problem types.Problem
	if err := attributevalue.UnmarshalMap(result.Item, &problem); err != nil {
		return nil, fmt.Errorf("failed to unmarshal problem: %v", err)
	}

	return &problem, nil
}

func main() {
	lambda.Start(handleRequest)
}
