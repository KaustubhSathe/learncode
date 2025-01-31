package main

import (
	"log"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"github.com/joho/godotenv"
	"learncode/backend/lib"
)

func init() {
	log.Println("Starting initialization...")
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found: %v", err)
	}
	log.Println("Environment variables loaded")
}

func main() {
	log.Println("Starting main...")
	defer jsii.Close()
	app := awscdk.NewApp(nil)

	lib.NewBackendStack(app, "LearnCodeStack", &lib.BackendStackProps{
		StackProps: awscdk.StackProps{
			Env: &awscdk.Environment{
				Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
				Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
			},
		},
	})

	app.Synth(nil)
}
