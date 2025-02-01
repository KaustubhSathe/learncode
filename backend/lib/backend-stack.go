package lib

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdkapigatewayv2alpha/v2"
	"github.com/aws/aws-cdk-go/awscdkapigatewayv2integrationsalpha/v2"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
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
		BillingMode:         awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:           jsii.String("Problems"),
		TimeToLiveAttribute: jsii.String("deleted_at"),
	})

	submissionsTable := awsdynamodb.NewTable(stack, jsii.String("Submissions"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:   jsii.String("Submissions"),
	})

	usersTable := awsdynamodb.NewTable(stack, jsii.String("Users"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode: awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:   jsii.String("Users"),
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
	usersTable.GrantReadWriteData(lambdaRole)

	// Lambda Functions
	submitLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("SubmitFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/submit"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_TOKEN":     jsii.String(os.Getenv("MOMENTO_TOKEN")),
		},
	})

	getProblemsLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("GetProblemsFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/get-problems"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
		},
	})

	getProblemLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("GetProblemFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/get-problem"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
		},
	})

	authLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("AuthFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/auth"),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"GITHUB_CLIENT_ID": jsii.String(os.Getenv("GITHUB_CLIENT_ID")),
		},
	})

	githubCallbackLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("GithubCallbackFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/github-callback"),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"GITHUB_CLIENT_ID":     jsii.String(os.Getenv("GITHUB_CLIENT_ID")),
			"GITHUB_CLIENT_SECRET": jsii.String(os.Getenv("GITHUB_CLIENT_SECRET")),
			"FRONTEND_URL":         jsii.String(os.Getenv("FRONTEND_URL")),
			"USERS_TABLE":          usersTable.TableName(),
		},
	})

	// Runner Lambdas
	nodejsRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("NodeJSRunner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("lambda/runners/nodejs"),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_API_KEY":   jsii.String(os.Getenv("MOMENTO_API_KEY")),
		},
	})

	// Grant Docker permissions
	nodejsRunner.Role().AddManagedPolicy(
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AWSLambdaExecute")),
	)

	cppRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("cpp-runner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("../lambda/runners/cpp"),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_API_KEY":   jsii.String(os.Getenv("MOMENTO_API_KEY")),
		},
	})

	javaRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("java-runner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("../lambda/runners/java"),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_API_KEY":   jsii.String(os.Getenv("MOMENTO_API_KEY")),
		},
	})

	pythonRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("python-runner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("../lambda/runners/python"),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":    problemsTable.TableName(),
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"MOMENTO_API_KEY":   jsii.String(os.Getenv("MOMENTO_API_KEY")),
		},
	})

	// Grant DynamoDB permissions
	problemsTable.GrantReadData(cppRunner)
	submissionsTable.GrantWriteData(cppRunner)
	problemsTable.GrantReadData(javaRunner)
	submissionsTable.GrantWriteData(javaRunner)
	problemsTable.GrantReadData(pythonRunner)
	submissionsTable.GrantWriteData(pythonRunner)

	// Create HTTP API for runners
	runnersApi := awscdkapigatewayv2alpha.NewHttpApi(stack, jsii.String("RunnersApi"), &awscdkapigatewayv2alpha.HttpApiProps{
		ApiName: jsii.String("Code Runners API"),
		CorsPreflight: &awscdkapigatewayv2alpha.CorsPreflightOptions{
			AllowHeaders: jsii.Strings("momento-signature", "Content-Type"),
			AllowMethods: &[]awscdkapigatewayv2alpha.CorsHttpMethod{
				awscdkapigatewayv2alpha.CorsHttpMethod_POST,
				awscdkapigatewayv2alpha.CorsHttpMethod_OPTIONS,
			},
			AllowOrigins: jsii.Strings("*"),
		},
	})

	// Add routes for each language runner
	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/nodejs"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("NodeJSRunnerIntegration"),
			nodejsRunner,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/cpp"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("CppRunnerIntegration"),
			cppRunner,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/java"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("JavaRunnerIntegration"),
			javaRunner,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/python"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("PythonRunnerIntegration"),
			pythonRunner,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	// HTTP API
	httpApi := awscdkapigatewayv2alpha.NewHttpApi(stack, jsii.String("LearnCodeApi"), &awscdkapigatewayv2alpha.HttpApiProps{
		ApiName: jsii.String("LearnCode API"),
		CorsPreflight: &awscdkapigatewayv2alpha.CorsPreflightOptions{
			AllowHeaders: jsii.Strings("Authorization", "Content-Type"),
			AllowMethods: &[]awscdkapigatewayv2alpha.CorsHttpMethod{
				awscdkapigatewayv2alpha.CorsHttpMethod_GET,
				awscdkapigatewayv2alpha.CorsHttpMethod_POST,
				awscdkapigatewayv2alpha.CorsHttpMethod_OPTIONS,
			},
			AllowOrigins: jsii.Strings("*"),
		},
	})

	// Routes for GitHub auth
	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/auth/github"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_GET,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("AuthIntegration"),
			authLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/auth/github/callback"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_GET,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("GithubCallbackIntegration"),
			githubCallbackLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/problems"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_GET,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("GetProblemsIntegration"),
			getProblemsLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/problems/{id}"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_GET,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("GetProblemIntegration"),
			getProblemLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/submit"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("SubmitIntegration"),
			submitLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	// Stack Outputs
	awscdk.NewCfnOutput(stack, jsii.String("ApiEndpoint"), &awscdk.CfnOutputProps{
		Value: httpApi.Url(),
	})

	return stack
}
