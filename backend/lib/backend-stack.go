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

	// Grant CloudFormation execution role access to the Node.js layer
	cfnExecRole := awsiam.Role_FromRoleArn(stack, jsii.String("CfnExecRole"),
		jsii.String("arn:aws:iam::473539126755:role/cdk-hnb659fds-cfn-exec-role-473539126755-ap-south-1"), nil)

	cfnExecRole.AttachInlinePolicy(awsiam.NewPolicy(stack, jsii.String("NodejsLayerAccess"), &awsiam.PolicyProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: &[]*string{
					jsii.String("lambda:GetLayerVersion"),
				},
				Resources: &[]*string{
					jsii.String("*"),
				},
			}),
		},
	}))

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

	submissionsTable := awsdynamodb.NewTable(stack, jsii.String("SubmissionsTable"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("problem_id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("submission_id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode:   awsdynamodb.BillingMode_PAY_PER_REQUEST,
		TableName:     jsii.String("SubmissionsV2"),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
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

	// Runner role with CloudWatch permissions
	runnerRole := awsiam.NewRole(stack, jsii.String("RunnerExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AWSLambdaBasicExecutionRole")),
		},
	})

	// Add Lambda layer access permission
	runnerRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Effect: awsiam.Effect_ALLOW,
		Actions: &[]*string{
			jsii.String("lambda:GetLayerVersion"),
		},
		Resources: &[]*string{
			jsii.String("*"),
		},
	}))

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
			"PROBLEMS_TABLE":     problemsTable.TableName(),
			"SUBMISSIONS_TABLE":  submissionsTable.TableName(),
			"MOMENTO_AUTH_TOKEN": jsii.String(os.Getenv("MOMENTO_AUTH_TOKEN")),
			"USERS_TABLE":        usersTable.TableName(),
		},
	})

	// Get Problems Lambda
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
			"USERS_TABLE":    usersTable.TableName(),
		},
	})

	problemsTable.GrantReadData(getProblemsLambda)
	usersTable.GrantReadData(getProblemsLambda)

	// Delete Problem Lambda
	deleteProblemLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("DeleteProblemLambda"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/delete-problem"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
			"USERS_TABLE":    usersTable.TableName(),
		},
	})

	problemsTable.GrantWriteData(deleteProblemLambda)
	usersTable.GrantReadData(deleteProblemLambda)

	// Add Problem Lambda
	addProblemLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("AddProblemLambda"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/add-problem"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE": problemsTable.TableName(),
			"USERS_TABLE":    usersTable.TableName(),
		},
	})

	problemsTable.GrantWriteData(addProblemLambda)
	usersTable.GrantReadData(addProblemLambda)

	// Get Problem Lambda
	getProblemLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("GetProblemLambda"), &awscdklambdagoalpha.GoFunctionProps{
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
			"USERS_TABLE":    usersTable.TableName(),
		},
	})

	authLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("AuthFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/auth"),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"GITHUB_CLIENT_ID": jsii.String(os.Getenv("GITHUB_CLIENT_ID")),
			"USERS_TABLE":      usersTable.TableName(),
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

	// Auth verify Lambda
	authVerifyLambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("AuthVerifyFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/auth-verify"),
		Role:    lambdaRole,
		Environment: &map[string]*string{
			"GITHUB_CLIENT_ID": jsii.String(os.Getenv("GITHUB_CLIENT_ID")),
			"USERS_TABLE":      usersTable.TableName(),
		},
	})

	// Create Node.js Lambda Layer
	nodejsLayer := awslambda.NewLayerVersion(stack, jsii.String("NodejsLayer"), &awslambda.LayerVersionProps{
		LayerVersionName: jsii.String("nodejs18"),
		Description:      jsii.String("Node.js 18.x runtime"),
		Code:             awslambda.Code_FromAsset(jsii.String("lambda/layers/nodejs"), nil),
		CompatibleRuntimes: &[]awslambda.Runtime{
			awslambda.Runtime_PROVIDED_AL2(),
		},
	})

	// Runner Lambdas
	nodejsRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("nodejs-runner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("lambda/runners/nodejs"),
		ModuleDir:  jsii.String("."),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Role:       runnerRole,
		Layers: &[]awslambda.ILayerVersion{
			nodejsLayer,
		},
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":     problemsTable.TableName(),
			"SUBMISSIONS_TABLE":  submissionsTable.TableName(),
			"MOMENTO_AUTH_TOKEN": jsii.String(os.Getenv("MOMENTO_AUTH_TOKEN")),
			"USERS_TABLE":        usersTable.TableName(),
		},
	})

	pythonRunner := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("python-runner"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime:    awslambda.Runtime_PROVIDED_AL2(),
		Entry:      jsii.String("lambda/runners/python"),
		ModuleDir:  jsii.String("."),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Role:       runnerRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":     problemsTable.TableName(),
			"SUBMISSIONS_TABLE":  submissionsTable.TableName(),
			"MOMENTO_AUTH_TOKEN": jsii.String(os.Getenv("MOMENTO_AUTH_TOKEN")),
			"USERS_TABLE":        usersTable.TableName(),
		},
	})

	// Create Java runner
	javaRunner := awslambda.NewFunction(stack, jsii.String("java-runner"), &awslambda.FunctionProps{
		Runtime:    awslambda.Runtime_JAVA_11(),
		Handler:    jsii.String("main.java.Handler::handleRequest"),
		Code:       awslambda.Code_FromAsset(jsii.String("lambda/runners/java/build/libs/java-runner.zip"), nil),
		Timeout:    awscdk.Duration_Seconds(jsii.Number(30)),
		MemorySize: jsii.Number(512),
		Role:       runnerRole,
		Environment: &map[string]*string{
			"PROBLEMS_TABLE":     problemsTable.TableName(),
			"SUBMISSIONS_TABLE":  submissionsTable.TableName(),
			"MOMENTO_AUTH_TOKEN": jsii.String(os.Getenv("MOMENTO_AUTH_TOKEN")),
			"USERS_TABLE":        usersTable.TableName(),
		},
	})

	// Grant DynamoDB permissions
	problemsTable.GrantReadData(pythonRunner)
	submissionsTable.GrantWriteData(pythonRunner)
	problemsTable.GrantReadData(nodejsRunner)
	submissionsTable.GrantWriteData(nodejsRunner)
	problemsTable.GrantReadData(javaRunner)
	submissionsTable.GrantWriteData(javaRunner)

	nodejsRunner.Role().AddManagedPolicy(
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AWSLambdaExecute")),
	)

	javaRunner.Role().AddManagedPolicy(
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AWSLambdaExecute")),
	)

	pythonRunner.Role().AddManagedPolicy(
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AWSLambdaExecute")),
	)

	// Create Runners API
	runnersApi := awscdkapigatewayv2alpha.NewHttpApi(stack, jsii.String("runners-api"), &awscdkapigatewayv2alpha.HttpApiProps{
		CorsPreflight: &awscdkapigatewayv2alpha.CorsPreflightOptions{
			AllowHeaders: jsii.Strings("*"),
			AllowMethods: &[]awscdkapigatewayv2alpha.CorsHttpMethod{
				awscdkapigatewayv2alpha.CorsHttpMethod_ANY,
			},
			AllowOrigins: jsii.Strings("*"),
		},
	})

	// Add Node.js runner integration
	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/nodejs"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("NodejsIntegration"),
			nodejsRunner,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	// Add Python runner integration
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

	// Add Java runner integration
	runnersApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/runners/java"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("JavaIntegration"),
			javaRunner,
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
				awscdkapigatewayv2alpha.CorsHttpMethod_DELETE,
				awscdkapigatewayv2alpha.CorsHttpMethod_PUT,
				awscdkapigatewayv2alpha.CorsHttpMethod_OPTIONS,
				awscdkapigatewayv2alpha.CorsHttpMethod_PATCH,
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
		Path: jsii.String("/admin/add"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_POST,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("AddProblemIntegration"),
			addProblemLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/admin/problems/{id}"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_DELETE,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("DeleteProblemIntegration"),
			deleteProblemLambda,
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

	// Add routes
	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path: jsii.String("/auth/verify"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{
			awscdkapigatewayv2alpha.HttpMethod_GET,
		},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("AuthVerifyIntegration"),
			authVerifyLambda,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	// Get Submission Lambda
	getSubmissionFunction := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("GetSubmissionFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2(),
		Entry:   jsii.String("lambda/get-submission"),
		Role:    lambdaRole,
		Bundling: &awscdklambdagoalpha.BundlingOptions{
			Environment: &map[string]*string{
				"GOOS":   jsii.String("linux"),
				"GOARCH": jsii.String("amd64"),
			},
		},
		Timeout: awscdk.Duration_Seconds(jsii.Number(30)),
		Environment: &map[string]*string{
			"SUBMISSIONS_TABLE": submissionsTable.TableName(),
			"USERS_TABLE":       usersTable.TableName(),
		},
	})

	submissionsTable.GrantReadData(getSubmissionFunction)
	usersTable.GrantReadData(getSubmissionFunction)

	httpApi.AddRoutes(&awscdkapigatewayv2alpha.AddRoutesOptions{
		Path:    jsii.String("/submissions"),
		Methods: &[]awscdkapigatewayv2alpha.HttpMethod{awscdkapigatewayv2alpha.HttpMethod_GET},
		Integration: awscdkapigatewayv2integrationsalpha.NewHttpLambdaIntegration(
			jsii.String("GetSubmissionIntegration"),
			getSubmissionFunction,
			&awscdkapigatewayv2integrationsalpha.HttpLambdaIntegrationProps{},
		),
	})

	// Output the API endpoints
	awscdk.NewCfnOutput(stack, jsii.String("MainApiEndpoint"), &awscdk.CfnOutputProps{
		Value:       httpApi.Url(),
		Description: jsii.String("Main API endpoint URL"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("RunnersApiEndpoint"), &awscdk.CfnOutputProps{
		Value:       runnersApi.Url(),
		Description: jsii.String("Runners API endpoint URL"),
	})

	return stack
}
