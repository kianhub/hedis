#!/bin/bash

# This script fetches various bits of Hermes source code
# to the current directory, for exploratory use.

# Exit on non-success
set -e

# How to list Hermes Git source tags:
# git -C ~/hermes fetch; git -C ~/hermes tag -l 'v*' | grep -v rc | sort -V | tr '\n' ' '; echo
# ("sort -V" means versioning-like sort)

# This file is an altered version of P1Secs get_source_codes.sh
# (https://github.com/P1sec/hermes-dec/blob/main/utils/original_hermes_bytecode_c_src/get_source_codes.sh)

cd "$(dirname "$0")"
source ../../git_tags.sh

verlte() {
    [  "$1" = "$(echo -e "$1\n$2" | sort -V | head -n1)" ]
}

for git_tag in "${GIT_TAGS[@]}"; do
    hbc_version=$(get_hbc_version "${git_tag}")

    echo "Downloading files for ${git_tag}..."
    if [ -z "${hbc_version}" ]; then
        # For older versions (v0.8.0 and earlier), directly try BytecodeFileFormat.h
        if verlte "${git_tag}" "v0.8.0"; then
            curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeFileFormat.h" -o "BytecodeVersion-${git_tag}.h"
        else
            # For newer versions, try BytecodeVersion.h first, then fallback
            if ! curl -s -f "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeVersion.h" -o "BytecodeVersion-${git_tag}.h"; then
                echo "  BytecodeVersion.h not found, trying BytecodeFileFormat.h instead..."
                curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeFileFormat.h" -o "BytecodeVersion-${git_tag}.h"
            fi
        fi
        curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeList.def" -o "BytecodeList-${git_tag}.def"
    else
        # First try BytecodeVersion.h, and fallback to BytecodeFileFormat.h if not found
        if ! curl -s -f "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeVersion.h" -o "BytecodeVersion-${hbc_version}.h"; then
            echo "  BytecodeVersion.h not found, trying BytecodeFileFormat.h instead..."
            curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeFileFormat.h" -o "BytecodeVersion-${hbc_version}.h"
        fi
        curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/BCGen/HBC/BytecodeList.def" -o "BytecodeList-${hbc_version}.def"
    fi
done
