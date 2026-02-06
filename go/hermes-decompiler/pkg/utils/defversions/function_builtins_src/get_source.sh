#!/bin/bash

# This script fetches various bits of Hermes source code
# to the current directory, for exploratory use.

# Exit on non-success
set -e

# How to list Hermes Git source tags:
# git -C ~/hermes fetch; git -C ~/hermes tag -l 'v*' | grep -v rc | sort -V | tr '\n' ' '; echo
# ("sort -V" means versioning-like sort)

# This file is an altered version of P1Secs get_source_codes.sh
# (https://github.com/P1sec/hermes-dec/blob/main/utils/original_function_builtins_c_src/get_source_codes.sh)

cd "$(dirname "$0")"
source ../../git_tags.sh

verlte() {
    [  "$1" = "$(echo -e "$1\n$2" | sort -V | head -n1)" ]
}

for git_tag in "${GIT_TAGS[@]}"; do
    hbc_version=$(get_hbc_version "${git_tag}")

    echo "Downloading files for ${git_tag}..."
    if [ -z "${hbc_version}" ]; then
        if verlte "${git_tag}" "v0.3.0"; then
            curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/Inst/Builtins.def" -o "Builtins-${git_tag}.def"
        else
            curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/FrontEndDefs/Builtins.def" -o "Builtins-${git_tag}.def"
        fi
    else
        curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/FrontEndDefs/Builtins.def" -o "Builtins-${hbc_version}.def" || curl -s "https://raw.githubusercontent.com/facebook/hermes/${git_tag}/include/hermes/Inst/Builtins.def" -o "Builtins-${hbc_version}.def"
    fi
done
