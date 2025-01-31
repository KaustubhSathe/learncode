package lib

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"os"
)

type BackendStackProps struct {
	awscdk.StackProps
}

func NewBackendStack(scope constructs.Construct, id string, props *BackendStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// DynamoDB Tables
	problemsTable := awsdynamodb.NewTable(stack, jsii.String("Problems"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:   jsii.String("Problems"),
	})

	submissionsTable := awsdynamodb.NewTable(stack, jsii.String("Submissions"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:   jsii.String("Submissions"),
	})

	// Lambda execution role
	lambdaRole := awsiam.NewRole(stack, jsii.String("LambdaExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AWSLambdaBasicExecutionRole")),
		},
	})

	// Grant DynamoDB permissions
	problemsTable.GrantReadWriteData(lambdaRole)
	submissionsTable.GrantReadWriteData(lambdaRole)

	// Lambda Functions
	submitLambda := awslambda.NewFunction(stack, jsii.String("SubmitFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Handler: jsii.String("bootstrap"),
		Code:    awslambda.Code_FromAsset(jsii.String("build/submit"), &awss3assets.AssetOptions{}),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_TOKEN":     jsii.String(os.Getenv("MOMENTO_AUTH_TOKEN")),
		},
	})

	getProblemsLambda := awslambda.NewFunction(stack, jsii.String("GetProblemsFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Handler: jsii.String("bootstrap"),
		Code:    awslambda.Code_FromAsset(jsii.String("build/get-problems"), &awss3assets.AssetOptions{}),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
		},
	})

	getProblemLambda := awslambda.NewFunction(stack, jsii.String("GetProblemFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Handler: jsii.String("bootstrap"),
		Code:    awslambda.Code_FromAsset(jsii.String("build/get-problem"), &awss3assets.AssetOptions{}),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
		},
	})

	// Auth Lambda
	authLambda := awslambda.NewFunction(stack, jsii.String("AuthFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Handler: jsii.String("bootstrap"),
		Code:    awslambda.Code_FromAsset(jsii.String("build/auth"), &awss3assets.AssetOptions{}),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"GITHUB_CLIENT_ID":     jsii.String(os.Getenv("GITHUB_CLIENT_ID")),
			"GITHUB_CLIENT_SECRET": jsii.String(os.Getenv("GITHUB_CLIENT_SECRET")),
			"FRONTEND_URL":         jsii.String(os.Getenv("FRONTEND_URL")),
		},
	})

	// API Gateway Authorizer
	authorizer := awsapigateway.NewTokenAuthorizer(stack, jsii.String("GithubAuthorizer"), &awsapigateway.TokenAuthorizerProps{
		Handler: authLambda,
	})

	// API Gateway with default authorization
	api := awsapigateway.NewRestApi(stack, jsii.String("LearnCodeApi"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String("LearnCode API"),
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins: awsapigateway.Cors_ALL_ORIGINS(),
			AllowMethods: awsapigateway.Cors_ALL_METHODS(),
			AllowHeaders: &[]*string{
				jsii.String("Authorization"),
				jsii.String("Content-Type"),
			},
		},
		DefaultMethodOptions: &awsapigateway.MethodOptions{
			Authorizer: authorizer,
		},
	})

	// Auth endpoints (no authorizer)
	auth := api.Root().AddResource(jsii.String("auth"), nil)
	githubAuth := auth.AddResource(jsii.String("github"), nil)
	githubAuth.AddMethod(jsii.String("GET"), 
		awsapigateway.NewLambdaIntegration(authLambda, nil), 
		&awsapigateway.MethodOptions{
			Authorizer: nil, // No auth required for login
		},
	)

	// Protected endpoints (with authorizer)
	problems := api.Root().AddResource(jsii.String("problems"), nil)
	problems.AddMethod(jsii.String("GET"), 
		awsapigateway.NewLambdaIntegration(getProblemsLambda, nil), nil)

	problem := problems.AddResource(jsii.String("{id}"), nil)
	problem.AddMethod(jsii.String("GET"),
		awsapigateway.NewLambdaIntegration(getProblemLambda, nil), nil)

	submit := api.Root().AddResource(jsii.String("submit"), nil)
	submit.AddMethod(jsii.String("POST"),
		awsapigateway.NewLambdaIntegration(submitLambda, nil), nil)

	// Stack Outputs
	awscdk.NewCfnOutput(stack, jsii.String("ApiEndpoint"), &awscdk.CfnOutputProps{
		Value: api.Url(),
	})

	return stack
} 

