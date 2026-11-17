#!/bin/bash

# Test Triage Factory on Genie Repo
# Usage: ./test-triage.sh <github_token> <issue_number>
# Example: ./test-triage.sh ghp_ABC123XYZ 1

if [ -z "$1" ]; then
    echo "❌ Error: GitHub token required"
    echo ""
    echo "Usage: ./test-triage.sh <token> <issue_number>"
    echo ""
    echo "Example:"
    echo "  ./test-triage.sh ghp_ABC123XYZ 1"
    echo ""
    echo "Get token at: https://github.com/settings/tokens"
    exit 1
fi

if [ -z "$2" ]; then
    echo "❌ Error: Issue number required"
    echo ""
    echo "Usage: ./test-triage.sh <token> <issue_number>"
    exit 1
fi

TOKEN=$1
ISSUE=$2
OWNER="PratikDhanave"
REPO="genie"
API="http://localhost:8080"

echo "🧪 Testing Triage on Genie Repo"
echo "================================"
echo ""
echo "📋 Parameters:"
echo "  Owner:  $OWNER"
echo "  Repo:   $REPO"
echo "  Issue:  $ISSUE"
echo "  Token:  ${TOKEN:0:15}..."
echo ""
echo "🚀 Calling: GET /api/github/triage"
echo ""

# Make the API call
curl -s -N "$API/api/github/triage?owner=$OWNER&repo=$REPO&issue=$ISSUE&token=$TOKEN" | while IFS= read -r line; do
    if [[ $line == data:* ]]; then
        json="${line#data: }"
        status=$(echo "$json" | jq -r '.status' 2>/dev/null)
        progress=$(echo "$json" | jq -r '.progress' 2>/dev/null)
        message=$(echo "$json" | jq -r '.message' 2>/dev/null)

        # Print progress
        printf "\r  [%-50s] %d%% - %s" "$(printf '#%.0s' $(seq 1 $((progress/2))))" "$progress" "$message"
    fi
done

echo ""
echo ""
echo "✅ Triage Complete!"
echo ""
echo "📊 Results Posted to:"
echo "   https://github.com/$OWNER/$REPO/issues/$ISSUE"
echo ""
echo "🔍 Check the issue for:"
echo "   ✓ Triage comment"
echo "   ✓ Applied labels"
echo "   ✓ Category & priority"
echo ""
