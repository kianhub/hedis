#!/bin/bash

# Function to map commit hash to HBC version number
get_hbc_version() {
    local git_tag="$1"

    case "$git_tag" in
        "ee7a2db") echo "hbc61" ;;
        "8eb5923") echo "hbc63" ;;
        "2d326de") echo "hbc64" ;;
        "28436fb") echo "hbc65" ;;
        "892bf2c") echo "hbc66" ;;
        "f9e263d") echo "hbc67" ;;
        "883eb4d") echo "hbc68" ;;
        "5a402ac") echo "hbc69" ;;
        "51969f2") echo "hbc70" ;;
        "d1637f2") echo "hbc72" ;;
        "ecded4d") echo "hbc73" ;;
        "6871396") echo "hbc75" ;;
        "701b9dd") echo "hbc77" ;;
        "c57fde2") echo "hbc78" ;;
        "05b2972") echo "hbc79" ;;
        "e70045d") echo "hbc80" ;;
        "40aa0f7") echo "hbc81" ;;
        "65de349") echo "hbc82" ;;
        "b823515") echo "hbc86" ;;
        "41752c6") echo "hbc87" ;;
        "2a55135") echo "hbc88" ;;
        "0763eee") echo "hbc90" ;;
        "4985960") echo "hbc91" ;;
        "b544ff4") echo "hbc92" ;;
        "1c71748") echo "hbc94" ;;
        "f6b56d3") echo "hbc95" ;;
        "2afc7b0") echo "hbc96" ;;
        *) echo "" ;;
    esac
}

# Hermes GitHub repository URL
HERMES_REPO_URL="https://github.com/facebook/hermes.git"

# Array of commit hashes (tags) extracted from the function
declare -a TAGS=("ee7a2db" "8eb5923" "2d326de" "28436fb" "892bf2c" "f9e263d" "883eb4d" "5a402ac" "51969f2" "d1637f2" "ecded4d" "6871396" "701b9dd" "c57fde2" "05b2972" "e70045d" "40aa0f7" "65de349" "b823515" "41752c6" "2a55135" "0763eee" "4985960" "b544ff4" "1c71748" "f6b56d3" "2afc7b0")

# Base directory for version-specific outputs
BASE_OUTPUT_DIR="./compilers/hermes-" # Example: subdirectories of the current location

# Temporary directory for the single clone
TEMP_CLONE_DIR="./hermes_repo_clone"

echo "--- Starting Hermes Version Checkout, Copy, and Build Script ---"

# 1. Clone the repository once into a temporary directory
echo "Cloning the repository once into '$TEMP_CLONE_DIR'..."
git clone "$HERMES_REPO_URL" "$TEMP_CLONE_DIR"

if [ $? -ne 0 ]; then
    echo "Error: Failed to clone the repository into '$TEMP_CLONE_DIR'. Exiting."
    exit 1
fi
echo "Repository cloned successfully."
echo ""

# Store the original directory where the script was run
ORIGINAL_DIR=$(pwd)

# Loop through each tag, checkout, copy contents, and build
for tag in "${TAGS[@]}"
do
    version=$(get_hbc_version "$tag")

    if [ -z "$version" ]; then
        echo "Warning: No version found for tag '$tag'. Skipping."
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue
    fi

    TARGET_DIR="${ORIGINAL_DIR}/${BASE_OUTPUT_DIR}${version}"

    echo "--- Processing tag '$tag' (version: ${version}) ---"

    # Create the target directory for this version
    mkdir -p "$TARGET_DIR"
    if [ $? -ne 0 ]; then
        echo "Error: Could not create target directory '$TARGET_DIR'. Skipping build for this tag."
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue
    fi
    echo "Target directory '$TARGET_DIR' created."

    # 2. Navigate into the temporary cloned directory and checkout the tag
    echo "Navigating to '$TEMP_CLONE_DIR' and checking out tag '$tag'..."
    cd "$TEMP_CLONE_DIR" || {
        echo "Error: Could not change directory to '$TEMP_CLONE_DIR'. Skipping build for this tag."
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue
    }

    git checkout "$tag"
    if [ $? -ne 0 ]; then
        echo "Error: Failed to checkout tag '$tag'. Skipping build."
        cd "$ORIGINAL_DIR" # Return to original directory
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue
    fi
    echo "Checked out tag '$tag'."

    # 3. Navigate back to the original directory and copy contents
    # Using rsync is generally safer and more robust than cp for directory contents
    echo "Navigating back to original directory and copying contents (excluding .git) to '$TARGET_DIR'..."
    cd "$ORIGINAL_DIR" || {
        echo "Error: Could not change back to the original directory. Exiting."
        # Fatal error if we can't get back to the original dir
        exit 1
    }

    # Copy contents using rsync, excluding the .git directory
    # The trailing slash on TEMP_CLONE_DIR/ is important to copy contents *into* TARGET_DIR
    rsync -a --exclude .git/ "$TEMP_CLONE_DIR/" "$TARGET_DIR/"
    if [ $? -ne 0 ]; then
        echo "Error during copying contents from '$TEMP_CLONE_DIR' to '$TARGET_DIR'. Skipping build."
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue
    fi
    echo "Contents copied successfully."

    # 4. Navigate into the target directory and build
    echo "Navigating to '$TARGET_DIR' and building hermesc..."
    cd "$TARGET_DIR" || {
        echo "Error: Could not change directory to '$TARGET_DIR'. Skipping build for this tag."
        # Navigate back to original directory is not needed as we are already here
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue # Skip to the next tag
    }

    # --- Build Steps ---
    echo "Running CMake configuration..."
    # Source directory is "." as we are inside $TARGET_DIR
    cmake -S . -B build -G Ninja
    if [ $? -ne 0 ]; then
        echo "Error during CMake configuration for tag '$tag'. Skipping build."
        cd "$ORIGINAL_DIR" > /dev/null # Go back to the original directory silently
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue # Skip to the next tag
    fi

    echo "Running CMake build for target 'hermesc'..."
    cmake --build ./build --target hermesc
    if [ $? -ne 0 ]; then
        echo "Error during building hermesc for tag '$tag'."
        # Build might have failed, but we can still try to clean up or just report error.
        cd "$ORIGINAL_DIR" > /dev/null # Go back to the original directory silently
        echo "--- Finished processing tag '$tag' ---"
        echo ""
        continue # Skip to the next tag
    fi

    echo "Successfully built hermesc for tag '$tag'."

    # Navigate back to the original directory before the next loop iteration
    cd "$ORIGINAL_DIR" > /dev/null || {
        echo "Error: Could not change back to the original directory after build."
        # This is less critical, just print a warning
    }

    echo "--- Finished processing tag '$tag' ---"
    echo ""

done

# 5. Clean up the temporary cloned repository directory
echo "Cleaning up temporary repository clone directory '$TEMP_CLONE_DIR'..."
rm -rf "$TEMP_CLONE_DIR"
if [ $? -eq 0 ]; then
    echo "Temporary directory removed successfully."
else
    echo "Warning: Could not remove temporary directory '$TEMP_CLONE_DIR'."
fi
echo ""

echo "--- Script Finished ---"
