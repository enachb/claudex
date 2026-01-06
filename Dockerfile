# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server

# Runtime stage
FROM node:22-alpine

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Install Claude CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Create non-root user with home directory
RUN adduser -D -g '' -h /home/appuser appuser

# Create .claude directory for credentials mount
RUN mkdir -p /home/appuser/.claude && chown -R appuser:appuser /home/appuser

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Change ownership
RUN chown -R appuser:appuser /app

# Set HOME for Claude CLI
ENV HOME=/home/appuser

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/livez || exit 1

# Run the server
ENTRYPOINT ["/app/server"]
