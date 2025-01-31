# Check if AWS CLI and CDK are installed
function Check-Prerequisites {
    Write-Host "Checking prerequisites..." -ForegroundColor Yellow
    
    # Check AWS CLI
    try {
        aws --version
    } catch {
        Write-Host "AWS CLI is not installed. Please install it first." -ForegroundColor Red
        exit 1
    }

    # Check CDK
    try {
        cdk --version
    } catch {
        Write-Host "AWS CDK is not installed. Please install it first." -ForegroundColor Red
        exit 1
    }
}

# Load environment variables from .env
function Load-EnvVars {
    Write-Host "Loading environment variables..." -ForegroundColor Yellow
    Get-Content .env | ForEach-Object {
        if ($_ -match '^([^=]+)=(.*)$') {
            $key = $matches[1]
            $value = $matches[2]
            [Environment]::SetEnvironmentVariable($key, $value)
        }
    }
}

# Build and deploy
function Deploy-Backend {
    Write-Host "Starting deployment..." -ForegroundColor Yellow

    # Store original location and get absolute paths
    $originalLocation = Get-Location
    $backendPath = $originalLocation
    $buildPath = Join-Path $backendPath "build"

    try {
        # Create build directory
        if (Test-Path $buildPath) {
            Remove-Item $buildPath -Recurse -Force
        }
        New-Item -ItemType Directory -Path $buildPath

        # Build Go Lambda functions
        Write-Host "Building Lambda functions..." -ForegroundColor Yellow
        $lambdas = @("auth", "get-problems", "get-problem", "submit")
        foreach ($lambda in $lambdas) {
            Write-Host "Building $lambda..." -ForegroundColor Yellow
            
            # Create lambda directory
            $lambdaDir = Join-Path $buildPath $lambda
            New-Item -ItemType Directory -Path $lambdaDir

            # Build the lambda
            Push-Location (Join-Path $backendPath "lambda/$lambda")
            
            # Initialize Go modules if needed
            if (-not (Test-Path "go.mod")) {
                go mod init "learncode/backend/lambda/$lambda"
                go get github.com/aws/aws-lambda-go/lambda
                go get github.com/aws/aws-lambda-go/events
                go mod tidy
            }

            # Build for Linux
            $env:GOOS = "linux"
            $env:GOARCH = "amd64"
            $env:CGO_ENABLED = "0"
            
            go build -o (Join-Path $buildPath "$lambda/bootstrap") main.go
            
            if ($LASTEXITCODE -ne 0) {
                Write-Host "Failed to build $lambda" -ForegroundColor Red
                exit 1
            }
            
            Pop-Location
        }

        # Initialize CDK if needed
        if (-not (Test-Path "cdk.json")) {
            Write-Host "Initializing CDK..." -ForegroundColor Yellow
            cdk init app --language go
        }

        # Ensure CDK app can find the build directory
        Write-Host "Building CDK app..." -ForegroundColor Yellow
        Set-Location $backendPath
        go mod tidy
        
        # CDK deploy
        Write-Host "Deploying with CDK..." -ForegroundColor Yellow
        
        # Set AWS environment variables
        $env:CDK_DEFAULT_ACCOUNT = aws sts get-caller-identity --query "Account" --output text
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to get AWS account ID" -ForegroundColor Red
            exit 1
        }
        
        $env:CDK_DEFAULT_REGION = aws configure get region
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to get AWS region" -ForegroundColor Red
            exit 1
        }

        Write-Host "Using AWS Account: $env:CDK_DEFAULT_ACCOUNT"
        Write-Host "Using AWS Region: $env:CDK_DEFAULT_REGION"

        # First synthesize the app
        Write-Host "Synthesizing CDK app..."
        go run backend.go
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to synthesize CDK app" -ForegroundColor Red
            exit 1
        }

        # Then deploy
        Write-Host "Deploying stack..."
        cdk deploy --require-approval never
        if ($LASTEXITCODE -ne 0) {
            Write-Host "CDK deployment failed" -ForegroundColor Red
            exit 1
        }
    }
    finally {
        # Restore original location
        Set-Location $originalLocation
    }
}

# Main deployment flow
try {
    Check-Prerequisites
    Load-EnvVars
    Deploy-Backend
    Write-Host "Deployment completed successfully!" -ForegroundColor Green
} catch {
    Write-Host "Deployment failed: $_" -ForegroundColor Red
    exit 1
} 