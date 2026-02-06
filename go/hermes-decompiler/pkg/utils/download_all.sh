#!/bin/bash

# This script runs all the download scripts in the subdirectories
# to fetch all required Hermes source code files.

# Exit on non-success
set -e

echo "Starting download of all Hermes source code files..."

# Store the current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
cd "$SCRIPT_DIR"

# Run function builtins downloader
echo -e "\n===== Downloading function builtins definitions ====="
bash defversions/function_builtins_src/get_source.sh

# Run bytecode downloader
echo -e "\n===== Downloading bytecode definitions ====="
bash defversions/hermes_bytecode_src/get_source.sh

# Run regex bytecode downloader
echo -e "\n===== Downloading regex bytecode definitions ====="
bash defversions/regex_bytecode_src/get_source.sh

echo -e "\n===== All downloads completed ====="
