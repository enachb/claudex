#!/bin/bash
# Script to update Kubernetes secret with Claude OAuth credentials
# Usage: ./update-k8s-credentials.sh [namespace]

set -e

NAMESPACE="${1:-leandro}"
SECRET_NAME="claude-credentials"

echo "=== Claude Code Kubernetes Credentials Updater ==="

# Try to find credentials file
CREDS_FILE=""

# Check common locations
LOCATIONS=(
    "$HOME/.claude/.credentials.json"
    "$HOME/.cc-switch/profiles/magic/.credentials.json"
    "$HOME/.cc-switch/profiles/leandro/.credentials.json"
)

for loc in "${LOCATIONS[@]}"; do
    if [ -f "$loc" ]; then
        CREDS_FILE="$loc"
        echo "Found credentials at: $loc"
        break
    fi
done

if [ -z "$CREDS_FILE" ]; then
    echo "ERROR: No credentials file found!"
    echo ""
    echo "Please run 'claude setup-token' first to generate a long-lived token."
    echo "On macOS, the token is stored in Keychain. You can export it with:"
    echo "  security find-generic-password -s 'claude-code' -w | pbcopy"
    echo ""
    echo "Alternatively, create the file manually:"
    echo "  ~/.claude/.credentials.json"
    exit 1
fi

# Validate JSON
if ! jq -e '.claudeAiOauth.accessToken' "$CREDS_FILE" > /dev/null 2>&1; then
    echo "ERROR: Invalid credentials file - missing accessToken"
    exit 1
fi

# Check expiration
EXPIRES_AT=$(jq -r '.claudeAiOauth.expiresAt' "$CREDS_FILE")
CURRENT_MS=$(($(date +%s) * 1000))

if [ "$EXPIRES_AT" -lt "$CURRENT_MS" ]; then
    echo "WARNING: Token has already expired!"
    echo "Please run 'claude setup-token' to generate a new token."
    exit 1
fi

EXPIRES_DATE=$(date -d @$((EXPIRES_AT/1000)) 2>/dev/null || date -r $((EXPIRES_AT/1000)) 2>/dev/null || echo "unknown")
echo "Token expires: $EXPIRES_DATE"

# Read credentials
OAUTH_TOKEN=$(cat "$CREDS_FILE")

echo ""
echo "Updating secret '$SECRET_NAME' in namespace '$NAMESPACE'..."

# Create/update secret
kubectl create secret generic "$SECRET_NAME" \
    --namespace="$NAMESPACE" \
    --from-literal="oauth-token=$OAUTH_TOKEN" \
    --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "Secret updated successfully!"
echo ""
echo "To restart pods and pick up new credentials:"
echo "  kubectl rollout restart deployment -n $NAMESPACE"
