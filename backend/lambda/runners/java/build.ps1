# Build script for Java runner
Write-Host "Building Java runner..."

# Setup Gradle wrapper if needed
Write-Host "Setting up Gradle wrapper..."
docker run --rm -v ${PWD}:/home/gradle/project -w /home/gradle/project gradle:7.4.2 gradle wrapper

# Run Gradle build
.\gradlew.bat clean build

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!"
    exit 1
}

Write-Host "Build successful!" 