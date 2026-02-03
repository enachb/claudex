#!/bin/bash
# Refresh Claude credentials in K8s from local ~/.claude/.credentials.json
# Usage: ./scripts/refresh-k8s-credentials.sh [namespace]
#
# This script updates the claude-credentials secret in K8s with fresh
# credentials from the local Claude CLI. The local CLI auto-refreshes
# tokens, so this keeps the K8s deployment working.
#
# Run this when you see "claude cli error: exit status 1" in Claudex logs.

set -e

NAMESPACE="${1:-leandro}"
CREDENTIALS_FILE="$HOME/.claude/.credentials.json"

# Check if credentials file exists
if [ ! -f "$CREDENTIALS_FILE" ]; then
    echo "Error: $CREDENTIALS_FILE not found"
    echo "Make sure you're logged into Claude CLI locally"
    exit 1
fi

# Check expiry
EXPIRES_AT=$(jq -r '.claudeAiOauth.expiresAt // 0' "$CREDENTIALS_FILE")
NOW=$(date +%s)
EXPIRES_SEC=$((EXPIRES_AT / 1000))

if [ "$EXPIRES_SEC" -lt "$NOW" ]; then
    echo "Warning: Local credentials are expired!"
    echo "Run 'claude' locally to refresh them first"
    exit 1
fi

EXPIRES_IN=$((EXPIRES_SEC - NOW))
echo "Local credentials valid for: $((EXPIRES_IN / 3600))h $((EXPIRES_IN % 3600 / 60))m"

# Update secret
echo "Updating claude-credentials secret in namespace: $NAMESPACE"
kubectl create secret generic claude-credentials \
    --from-file=.credentials.json="$CREDENTIALS_FILE" \
    -n "$NAMESPACE" \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Secret updated successfully"

# Restart deployment to pick up new credentials
echo "Restarting claudex-api deployment..."
kubectl rollout restart -n "$NAMESPACE" deployment/claudex-api
kubectl rollout status -n "$NAMESPACE" deployment/claudex-api --timeout=90s

echo ""
echo "Done! Claudex API should now have fresh credentials."
echo "Test with: curl http://localhost:8081/v1/chat/completions -X POST -H 'Content-Type: application/json' -d '{\"model\":\"claude-opus-4\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}'"
