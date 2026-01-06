# Claudex

[![CI](https://github.com/leeaandrob/claudex/actions/workflows/ci.yml/badge.svg)](https://github.com/leeaandrob/claudex/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/leeaandrob/claudex)](https://goreportcard.com/report/github.com/leeaandrob/claudex)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)

**Use Claude with any OpenAI-compatible client.**

Claudex is a lightweight proxy that exposes an OpenAI-compatible Chat Completions API, powered by the Claude CLI. Drop-in replacement for OpenAI API - works with existing SDKs, tools, and integrations.

## Why Claudex?

- **Zero code changes** - Use your existing OpenAI SDK code with Claude
- **Real-time streaming** - Full SSE support with token-by-token delivery
- **Production ready** - OpenTelemetry tracing, Prometheus metrics, structured logging
- **Kubernetes native** - Health checks, graceful shutdown, easy deployment

## Quick Start

### Prerequisites

- Go 1.22+
- Claude CLI authenticated: `npm install -g @anthropic-ai/claude-code && claude login`

### Installation

```bash
# Clone the repository
git clone https://github.com/leeaandrob/claudex.git
cd claudex

# Build and run
make run
```

### Usage

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"  # Auth handled by Claude CLI
)

# Non-streaming
response = client.chat.completions.create(
    model="claude",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)
print(response.choices[0].message.content)

# Streaming
stream = client.chat.completions.create(
    model="claude",
    messages=[{"role": "user", "content": "Tell me a story"}],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

## API Compatibility

| Feature | Status |
|---------|--------|
| Chat Completions | ✅ |
| Streaming (SSE) | ✅ |
| System messages | ✅ |
| Multi-turn conversations | ✅ |
| Multimodal content (text) | ✅ |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | OpenAI-compatible chat completions |
| `/livez` | GET | Liveness probe |
| `/readyz` | GET | Readiness probe |
| `/metrics` | GET | Prometheus metrics |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | - | OpenTelemetry endpoint |
| `SERVICE_NAME` | `claudex` | Service name for tracing |

## Deployment

### Docker

```bash
# Build
docker build -t claudex .

# Run (mount Claude CLI credentials)
docker run -p 8080:8080 \
  -v ~/.claude:/home/appuser/.claude:ro \
  claudex
```

### Kubernetes

```bash
# Create secret with Claude credentials
kubectl create secret generic claude-credentials \
  --from-file=credentials.json=$HOME/.claude/credentials.json

# Deploy
kubectl apply -f k8s/
```

## Development

```bash
make build       # Build binary
make test        # Run unit tests
make test-e2e    # Run E2E tests (requires Claude CLI)
make lint        # Run linter
make clean       # Clean build artifacts
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────┐
│  OpenAI Client  │────▶│     Claudex     │────▶│  Claude CLI │
│  (Python SDK)   │◀────│   (Go + Fiber)  │◀────│ (claude -p) │
└─────────────────┘     └─────────────────┘     └─────────────┘
                               │
                               ├── OpenTelemetry Traces
                               ├── Prometheus Metrics
                               └── Structured Logs (JSON)
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## License

[MIT](LICENSE) - Use it freely in your projects.

## Acknowledgments

Built with [Fiber](https://gofiber.io/), powered by [Claude](https://claude.ai/).
