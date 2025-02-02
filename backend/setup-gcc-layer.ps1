# Create layer directory structure
New-Item -ItemType Directory -Force -Path lambda/layers/gcc/gcc/bin -ErrorAction SilentlyContinue

# Set working directory
Push-Location lambda/layers/gcc

# Create a bash script
$script = @'
#!/bin/bash
set -e
yum install -y gcc-c++
mkdir -p /layer/gcc/bin

# Copy g++ and libraries
cp /usr/bin/g++ /layer/gcc/bin/
cp /usr/lib64/libstdc++.so.6 /layer/gcc/bin/
cp /usr/lib64/libgcc_s.so.1 /layer/gcc/bin/

# Find and copy cc1plus
CC1PLUS=$(find /usr/libexec -name cc1plus)
echo "Found cc1plus at: $CC1PLUS"
cp $CC1PLUS /layer/gcc/bin/
'@

# Save script with Unix line endings
[System.IO.File]::WriteAllText("$PWD/setup.sh", $script.Replace("`r`n","`n"))

# Run the script in Docker
docker run --rm -v ${PWD}:/layer amazonlinux:2 bash /layer/setup.sh

# Cleanup
Remove-Item setup.sh

# Return to original directory
Pop-Location 