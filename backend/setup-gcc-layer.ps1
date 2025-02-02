# Ensure we're in the right directory
Set-Location $PSScriptRoot

# Create layer directory structure
$layerDir = Join-Path $PSScriptRoot "lambda/layers/gcc"
$layerPath = Join-Path $layerDir "gcc/bin"
$includePath = Join-Path $layerDir "gcc/include"

# Clean up existing gcc layer
Write-Host "Cleaning up existing gcc layer..."
if (Test-Path $layerDir) {
    Remove-Item -Path $layerDir -Recurse -Force
}

# Create all necessary directories
Write-Host "Creating directories..."
New-Item -ItemType Directory -Force -Path $layerDir -ErrorAction SilentlyContinue | Out-Null
New-Item -ItemType Directory -Force -Path $layerPath -ErrorAction SilentlyContinue | Out-Null
New-Item -ItemType Directory -Force -Path $includePath -ErrorAction SilentlyContinue | Out-Null

# Create a bash script
$script = @'
#!/bin/bash
set -e

# Enable extra repos and install GCC
amazon-linux-extras enable epel
yum clean metadata
yum install -y gcc-c++ libmpc

# Show GCC version
g++ --version

# Debug: Find header files and GCC version
echo "Looking for GCC installation..."
g++ --version
echo "Looking for C++ headers..."
find /usr/include/c++/ -maxdepth 1
find /usr/include/c++/ -name stdc++.h
find /usr/include/c++/ -name c++config.h

# Get GCC version for paths
GCC_VER=$(g++ -dumpversion)
echo "GCC version: $GCC_VER"

# Create directories
mkdir -p /layer/gcc/bin
mkdir -p /layer/gcc/include/bits
mkdir -p /layer/gcc/include/c++/$GCC_VER
mkdir -p /layer/gcc/include/x86_64-redhat-linux/bits
mkdir -p /layer/gcc/include/sys
mkdir -p /layer/gcc/include/gnu

# Copy compiler and core libraries
cp /usr/bin/g++ /layer/gcc/bin/
cp /usr/lib64/libstdc++.so.6 /layer/gcc/bin/
cp /usr/lib64/libgcc_s.so.1 /layer/gcc/bin/

# Copy additional required libraries
cp /usr/lib64/libmpc.so.3 /layer/gcc/bin/
cp /usr/lib64/libmpfr.so.4 /layer/gcc/bin/
cp /usr/lib64/libgmp.so.10 /layer/gcc/bin/

# Find and copy cc1plus
CC1PLUS=$(find /usr/libexec -name cc1plus)
echo "Found cc1plus at: $CC1PLUS"
cp $CC1PLUS /layer/gcc/bin/

# Copy essential C++ headers
echo "Copying headers..."

# Copy C++ standard library headers
cp -r /usr/include/c++/$GCC_VER/* /layer/gcc/include/c++/$GCC_VER/

# Copy platform-specific headers from correct location
cp -r /usr/include/c++/$GCC_VER/x86_64-redhat-linux/bits/* /layer/gcc/include/x86_64-redhat-linux/bits/
cp -r /usr/include/bits/* /layer/gcc/include/bits/
cp -r /usr/include/sys/* /layer/gcc/include/sys/
cp -r /usr/include/gnu/* /layer/gcc/include/gnu/

# Copy root-level headers
cp /usr/include/*.h /layer/gcc/include/

# Create stdc++.h if it doesn't exist
echo "Creating stdc++.h..."
cat > /layer/gcc/include/bits/stdc++.h << 'EOL'
// C++ includes used for precompiling -*- C++ -*-

// Copyright (C) 2003-2021 Free Software Foundation, Inc.
//
// This file is part of the GNU ISO C++ Library.  This library is free
// software; you can redistribute it and/or modify it under the terms
// of the GNU General Public License as published by the Free Software
// Foundation; either version 3, or (at your option) any later
// version.

// This library is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.

// Under Section 7 of GPL version 3, you are granted additional
// permissions described in the GCC Runtime Library Exception, version
// 3.1, as published by the Free Software Foundation.

// You should have received a copy of the GNU General Public License and
// a copy of the GCC Runtime Library Exception along with this program;
// see the files COPYING3 and COPYING.RUNTIME respectively.

/** @file stdc++.h
 *  This is an implementation file for a precompiled header.
 */

// 17.4.1.2 Headers

// C
#ifndef _GLIBCXX_NO_ASSERT
#include <cassert>
#endif
#include <cctype>
#include <cerrno>
#include <cfloat>
#include <ciso646>
#include <climits>
#include <clocale>
#include <cmath>
#include <csetjmp>
#include <csignal>
#include <cstdarg>
#include <cstddef>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <ctime>
#include <cwchar>
#include <cwctype>

#if __cplusplus >= 201103L
#include <ccomplex>
#include <cfenv>
#include <cinttypes>
#include <cstdalign>
#include <cstdbool>
#include <cstdint>
#include <ctgmath>
#include <cuchar>
#endif

// C++
#include <algorithm>
#include <bitset>
#include <complex>
#include <deque>
#include <exception>
#include <fstream>
#include <functional>
#include <iomanip>
#include <ios>
#include <iosfwd>
#include <iostream>
#include <istream>
#include <iterator>
#include <limits>
#include <list>
#include <locale>
#include <map>
#include <memory>
#include <new>
#include <numeric>
#include <ostream>
#include <queue>
#include <set>
#include <sstream>
#include <stack>
#include <stdexcept>
#include <streambuf>
#include <string>
#include <typeinfo>
#include <utility>
#include <valarray>
#include <vector>

#if __cplusplus >= 201103L
#include <array>
#include <atomic>
#include <chrono>
#include <codecvt>
#include <condition_variable>
#include <forward_list>
#include <future>
#include <initializer_list>
#include <mutex>
#include <random>
#include <ratio>
#include <regex>
#include <scoped_allocator>
#include <system_error>
#include <thread>
#include <tuple>
#include <typeindex>
#include <type_traits>
#include <unordered_map>
#include <unordered_set>
#endif

#if __cplusplus >= 201402L
#include <shared_mutex>
#endif

#if __cplusplus >= 201703L
#include <any>
#include <charconv>
#include <execution>
#include <filesystem>
#include <optional>
#include <memory_resource>
#include <string_view>
#include <variant>
#endif

#if __cplusplus > 201703L
#include <barrier>
#include <bit>
#include <compare>
#include <concepts>
#include <coroutine>
#include <latch>
#include <numbers>
#include <ranges>
#include <span>
#include <stop_token>
#include <semaphore>
#include <source_location>
#include <syncstream>
#include <version>
#endif
EOL

# Debug: Show directory structure
echo "Directory structure:"
find /layer/gcc/include -type d

# Debug: Show some key files
echo "Checking key files:"
ls -l /layer/gcc/include/bits/stdc++.h
ls -l /layer/gcc/include/x86_64-redhat-linux/bits/c++config.h
ls -l /layer/gcc/include/c++/$GCC_VER/iostream

# Set permissions
chmod 755 /layer/gcc/bin/*
chmod -R 755 /layer/gcc/include

# Create test C++ program
cat > /layer/test.cpp << 'EOL'
#include <bits/stdc++.h>
using namespace std;

int main() {
    cout << "Hello from C++!" << endl;
    vector<int> v = {1, 2, 3, 4, 5};
    cout << "Vector sum: " << accumulate(v.begin(), v.end(), 0) << endl;
    return 0;
}
EOL

# Test compilation and execution
echo "Testing C++ compilation and execution..."
g++ -o /layer/test /layer/test.cpp \
    -I/layer/gcc/include \
    -I/layer/gcc/include/c++/$GCC_VER \
    -I/layer/gcc/include/c++/$GCC_VER/x86_64-redhat-linux \
    -B/layer/gcc/bin

# Run the test
/layer/test

# List files and permissions
echo "Layer contents:"
ls -la /layer/gcc/bin/
echo "Include contents by directory:"
for dir in /layer/gcc/include/*/; do
    echo "Contents of $dir:"
    ls -la "$dir"
done
'@

# Save script with Unix line endings
$scriptPath = Join-Path $layerDir "setup.sh"
Write-Host "Creating script at: $scriptPath"
[System.IO.File]::WriteAllText($scriptPath, $script.Replace("`r`n","`n"))

# Run the script in Docker
Write-Host "Running Docker with volume: $layerDir"
docker run --rm -v "${layerDir}:/layer" amazonlinux:2 bash /layer/setup.sh

# Verify local files
Write-Host "`nLocal files:"
Get-ChildItem -Path $layerPath | Select-Object Name, Length

# Cleanup
Remove-Item $scriptPath -ErrorAction SilentlyContinue

Write-Host "`nSetup complete!" 