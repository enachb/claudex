# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-02-02

### Added
- MCP (Model Context Protocol) server support with JSON-RPC 2.0 over stdio
- MCP client with connection pooling, automatic reconnection, and health monitoring
- MCP manager for orchestrating multiple server connections
- Tool discovery and execution via MCP protocol
- Configuration via YAML (`config/claudex.yaml`)
- New API endpoints:
  - `GET /v1/mcp/tools` - List all available MCP tools
  - `GET /v1/mcp/servers` - List connected MCP servers
  - `POST /v1/mcp/tools/call` - Execute MCP tools directly
- Environment variable `CLAUDEX_MCP_CONFIG_PATH` for config file location
- Docker support with multi-architecture builds (amd64/arm64)
- Entrypoint script for credential handling in containers

### Changed
- Chat completions handler now integrates MCP tools automatically
- Tool calls in responses are executed via MCP when available
- Improved error handling and logging for tool execution

### Fixed
- Conversation continuation after tool execution now works correctly
- Tool results properly formatted for follow-up requests

## [0.1.0] - 2026-02-02

### Added
- Initial release
- OpenAI-compatible chat completions API (`/v1/chat/completions`)
- Streaming and non-streaming response support
- Vision/image support via base64 data URLs
- Tool calling support with JSON schema validation
- Health check endpoints (`/livez`, `/readyz`, `/healthz`)
- Prometheus metrics endpoint (`/metrics`)
- Structured JSON logging
- Request timeout configuration via `REQUEST_TIMEOUT` environment variable
