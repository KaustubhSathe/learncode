package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"learncode/backend/types"
	"os"
	"sync"

	"github.com/momentohq/client-sdk-go/auth"
	"github.com/momentohq/client-sdk-go/config"
	"github.com/momentohq/client-sdk-go/momento"
)

var momentoClient *momento.TopicClient
var momentoOnce sync.Once

func initMomentoClient() error {
	// Skip initialization if MOMENTO_AUTH_TOKEN is not set (e.g., during testing or local development)
	if os.Getenv("MOMENTO_AUTH_TOKEN") == "" {
		return fmt.Errorf("MOMENTO_AUTH_TOKEN not set")
	}

	credentialProvider, err := auth.NewEnvMomentoTokenProvider("MOMENTO_AUTH_TOKEN")
	if err != nil {
		return fmt.Errorf("failed to load Momento auth token: %v", err)
	}

	client, err := momento.NewTopicClient(config.TopicsDefault(), credentialProvider)
	if err != nil {
		return fmt.Errorf("failed to create Momento client: %v", err)
	}
	momentoClient = &client
	return nil
}

func PublishToMomento(ctx context.Context, submission types.Submission) error {
	var initErr error
	momentoOnce.Do(func() {
		initErr = initMomentoClient()
	})

	if initErr != nil {
		return fmt.Errorf("failed to initialize momento client: %v", initErr)
	}

	if momentoClient == nil {
		return fmt.Errorf("momento client not initialized")
	}

	// Create topic name based on language
	topicName := fmt.Sprintf("learncode-%s", submission.Language)

	// Publish submission to appropriate topic
	message, _ := json.Marshal(submission)
	if _, err := (*momentoClient).Publish(ctx, &momento.TopicPublishRequest{
		CacheName: "learncode-cache",
		TopicName: topicName,
		Value:     momento.Bytes(message),
	}); err != nil {
		return fmt.Errorf("failed to publish to topic %s: %v", topicName, err)
	}

	return nil
}
