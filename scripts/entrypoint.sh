#!/bin/sh
set -e

CLAUDE_DIR="${HOME}/.claude"
CREDENTIALS_FILE="${CLAUDE_DIR}/.credentials.json"
MCP_FILE="${CLAUDE_DIR}/.mcp.json"

# Ensure .claude directory exists with correct permissions
mkdir -p "${CLAUDE_DIR}"

# Remove or reset MCP config to avoid hanging on unavailable servers
# The MCP servers configured on the host won't be available in the container
# Create empty MCP config if not exists (ensures clean MCP state)
if [ ! -f "${MCP_FILE}" ]; then
    echo "Creating minimal MCP config for container environment"
    echo '{"mcpServers":{}}' > "${MCP_FILE}" 2>/dev/null || true
fi

# If CLAUDE_CODE_OAUTH_TOKEN is set, create credentials file from it
if [ -n "${CLAUDE_CODE_OAUTH_TOKEN}" ]; then
    echo "Using CLAUDE_CODE_OAUTH_TOKEN environment variable"

    # Extract components from the token if it's a full JSON
    if echo "${CLAUDE_CODE_OAUTH_TOKEN}" | grep -q "accessToken"; then
        # Full JSON provided
        echo "${CLAUDE_CODE_OAUTH_TOKEN}" > "${CREDENTIALS_FILE}"
    else
        # Just the access token provided, create minimal credentials
        # Expiry set to 1 year from now (in milliseconds)
        EXPIRY=$(( $(date +%s) * 1000 + 31536000000 ))
        cat > "${CREDENTIALS_FILE}" << EOF
{
  "claudeAiOauth": {
    "accessToken": "${CLAUDE_CODE_OAUTH_TOKEN}",
    "expiresAt": ${EXPIRY},
    "scopes": ["user:inference"],
    "subscriptionType": "max",
    "rateLimitTier": "default_claude_max_20x"
  }
}
EOF
    fi
    chmod 600 "${CREDENTIALS_FILE}"
fi

# Check if credentials exist
if [ -f "${CREDENTIALS_FILE}" ]; then
    # Check token expiration
    EXPIRES_AT=$(cat "${CREDENTIALS_FILE}" | grep -o '"expiresAt":[0-9]*' | grep -o '[0-9]*' || echo "0")
    CURRENT_MS=$(( $(date +%s) * 1000 ))

    if [ "${EXPIRES_AT}" -gt 0 ] && [ "${EXPIRES_AT}" -lt "${CURRENT_MS}" ]; then
        echo "WARNING: OAuth token has expired! Token expired at $(date -d @$((EXPIRES_AT/1000)))"
        echo "Please update CLAUDE_CODE_OAUTH_TOKEN or mount fresh credentials"
    else
        EXPIRES_DATE=$(date -d @$((EXPIRES_AT/1000)) 2>/dev/null || echo "unknown")
        echo "OAuth token valid until: ${EXPIRES_DATE}"
    fi
else
    echo "WARNING: No credentials found at ${CREDENTIALS_FILE}"
    echo "Set CLAUDE_CODE_OAUTH_TOKEN env var or mount credentials file"
fi

# Start the server
exec /app/server "$@"
