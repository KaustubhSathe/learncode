# Seed Sample Problems in DynamoDB Problems Table
# This script uses the AWS CLI to put sample items into your Problems table.
# Ensure that AWS CLI is installed and configured.
# The table name is set to "Problems".

$tableName = "Problems"
Write-Host "Using Problems Table: $tableName"

# Create a temporary directory for JSON files
$tempDir = Join-Path $env:TEMP "seed_problems"
if (-not (Test-Path $tempDir)) {
    New-Item -ItemType Directory -Path $tempDir | Out-Null
}

# Create a UTF8 encoding instance that does not include a BOM.
$utf8NoBOM = New-Object System.Text.UTF8Encoding($false)

# Problem 1: Reverse a String
$item1Content = @'
{
    "id": {"S": "prob-001"},
    "title": {"S": "Reverse a String"},
    "description": {"S": "Given a string, return the string reversed."},
    "difficulty": {"S": "Easy"},
    "input": {"S": "Hello World"},
    "output": {"S": "dlroW olleH"}
}
'@
$item1Path = Join-Path $tempDir "item1.json"
[System.IO.File]::WriteAllText($item1Path, $item1Content, $utf8NoBOM)

aws dynamodb put-item --table-name $tableName --item file://$item1Path
Write-Host "Inserted Problem: Reverse a String"

# Problem 2: Sum Two Numbers
$item2Content = @'
{
    "id": {"S": "prob-002"},
    "title": {"S": "Sum of Two Numbers"},
    "description": {"S": "Compute the sum of two integers."},
    "difficulty": {"S": "Easy"},
    "input": {"S": "3 5"},
    "output": {"S": "8"}
}
'@
$item2Path = Join-Path $tempDir "item2.json"
[System.IO.File]::WriteAllText($item2Path, $item2Content, $utf8NoBOM)

aws dynamodb put-item --table-name $tableName --item file://$item2Path
Write-Host "Inserted Problem: Sum of Two Numbers"

# Problem 3: Fibonacci Number
$item3Content = @'
{
    "id": {"S": "prob-003"},
    "title": {"S": "Calculate Fibonacci"},
    "description": {"S": "Return the nth Fibonacci number. Input n is provided."},
    "difficulty": {"S": "Medium"},
    "input": {"S": "7"},
    "output": {"S": "13"}
}
'@
$item3Path = Join-Path $tempDir "item3.json"
[System.IO.File]::WriteAllText($item3Path, $item3Content, $utf8NoBOM)

aws dynamodb put-item --table-name $tableName --item file://$item3Path
Write-Host "Inserted Problem: Calculate Fibonacci"

Write-Host "Sample problems seeded successfully."

# Optionally, clean up the temporary directory
Remove-Item -Path $tempDir -Recurse -Force 