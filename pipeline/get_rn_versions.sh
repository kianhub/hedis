#!/bin/bash

# Define the range of React Native versions
versions=("0.69", "0.70" "0.71" "0.72" "0.73" "0.74" "0.75" "0.76" "0.77" "0.78" "0.79")

# Define the base directory for all projects, relative to the current directory (pipeline)
base_dir="react-natives"

echo "---------------------------------------------------"
echo "Ensuring base directory exists: ./$base_dir (relative to pipeline/)"
echo "---------------------------------------------------"

# Create the base directory if it doesn't exist (-p flag prevents error if it exists)
mkdir -p "$base_dir"

# Check if directory creation was successful
if [ $? -ne 0 ]; then
  echo "Failed to create or access directory: ./$base_dir"
  exit 1
fi

# Store the current directory (which is pipeline/) to return later
original_dir=$(pwd)

# Change to the base directory (react-natives/ inside pipeline/)
cd "$base_dir" || { echo "Failed to change directory to ./$base_dir"; exit 1; }

# Loop through each version
for version in "${versions[@]}"; do
  echo "---------------------------------------------------"
  echo "Creating project for React Native version: $version"
  echo "---------------------------------------------------"

  # Define the project directory name within the base_dir (react-natives/)
  # Remove dots from the version for the directory name
  version_without_dots="${version//./}"
  project_name="rn${version_without_dots}" # Project name will be like rn069, rn070 etc.

  # npx react-native init will create this directory
  # Create a new React Native project with the specified version
  # We use npx to execute the react-native command without installing it globally
  # The project will be created inside the current directory ($base_dir)
  # We use --name explicitly here to ensure the project name *inside* package.json is correct,
  # while the directory name is controlled by the first argument.
  npx @react-native-community/cli init "$project_name" --version "$version"

  # Check if the init command was successful
  if [ $? -eq 0 ]; then
    echo "Successfully created project for React Native version: $version in ./$base_dir/$project_name"
  else
    echo "Failed to create project for React Native version: $version"
    # You might want to add more robust error handling here, e.g., clean up failed directory
  fi

done

# Navigate back to the original directory (pipeline/)
cd "$original_dir"

echo "---------------------------------------------------"
echo "Script finished. Projects are located in ./$base_dir (relative to pipeline/)"
echo "---------------------------------------------------"
