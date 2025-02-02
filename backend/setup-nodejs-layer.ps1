# Create layer directory structure
New-Item -ItemType Directory -Force -Path lambda/layers/nodejs/nodejs/bin

# Set working directory
Push-Location lambda/layers/nodejs

# Download Node.js (Linux version)
Invoke-WebRequest -Uri "https://nodejs.org/dist/v16.20.2/node-v16.20.2-linux-x64.tar.xz" -OutFile "node.tar.xz"

# Extract using 7zip (needs to be installed)
7z x -y node.tar.xz
7z x -y node.tar

# Copy only the node binary (skip symlinks)
Copy-Item -Force "node-v16.20.2-linux-x64/bin/node" -Destination "nodejs/bin/"

# Cleanup
Remove-Item -Recurse -Force -ErrorAction SilentlyContinue node.tar.xz, node.tar, node-v16.20.2-linux-x64

# Return to original directory
Pop-Location 