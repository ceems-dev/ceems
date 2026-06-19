#!/bin/bash

# Set repository information
REPO_OWNER="ceems-dev"
REPO_NAME="ceems"
API_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases"

# Fetch all release information
ALL_RELEASES=$(curl -s $API_URL)

# Check if a valid release list is found
if [[ $ALL_RELEASES == *"Not Found"* ]]; then
    echo "Releases not found for the $REPO_NAME repository."
    exit 1
fi

# Extract the latest release tag from the release information
LATEST_RELEASE_TAG=$(echo "$ALL_RELEASES" | grep -Eo '"tag_name": "v[^"]*' | sed -E 's/"tag_name": "//' | sed -E 's/^.//' | head -n 1)
LATEST_RELEASE_DATE=$(echo "$ALL_RELEASES" | grep -Eo '"created_at": "[^T]*' | sed -E 's/"created_at": "//' | head -n 1)

# Print latest tag
echo "Latest tag is ${LATEST_RELEASE_TAG} released on ${LATEST_RELEASE_DATE}"

# Replace the version string in README
sed -Ei "s/version: (.*)/version: ${LATEST_RELEASE_TAG}/g" CITATION.cff
sed -Ei "s/date-released: (.*)/date-released: ${LATEST_RELEASE_DATE}/g" CITATION.cff
