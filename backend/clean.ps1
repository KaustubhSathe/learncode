# Clean build artifacts
Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow

$lambdas = @("auth", "get-problems", "get-problem", "submit")
foreach ($lambda in $lambdas) {
    $path = "lambda/$lambda/bootstrap"
    if (Test-Path $path) {
        Remove-Item $path
        Write-Host "Cleaned $lambda" -ForegroundColor Green
    }
}

# Clean CDK outputs
if (Test-Path "cdk.out") {
    Remove-Item "cdk.out" -Recurse -Force
    Write-Host "Cleaned cdk.out" -ForegroundColor Green
} 