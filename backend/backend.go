package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"learncode/backend/lib"
	"os"
)

func main() {
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
