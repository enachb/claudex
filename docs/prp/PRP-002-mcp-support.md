name: "PRP-002: MCP Server Support & Anthropic API Deprecation"
description: |

## Purpose

Add native Model Context Protocol (MCP) server support to Claudex, enabling generic tool execution via any MCP-compliant server. This PRP also formalizes the deprecation of the Anthropic API backend, consolidating Claudex to use Claude CLI exclusively.

## Core Principles

1. Claudex becomes an MCP client - manages MCP server lifecycle
2. Generic MCP configuration - not tied to any specific use case
3. Tool discovery at startup - inject tools to Claude CLI via system prompt
4. Tool execution at runtime - Claudex executes tool_calls via MCP
5. Claude CLI remains the only LLM backend after Anthropic API deprecation

---

## Discovery Summary

### Initial Task Analysis

User requires MCP support in Claudex for the Pokemon AI League project. The MCP server (`mcp_server.py`) is already built using FastMCP and provides tools for emulator control. Current tool calling works via prompt injection but lacks native MCP protocol support.

### User Clarifications Received

- **Question**: Should MCP support be Pokemon-specific or generic?
- **Answer**: Generic - Claudex should be capable of embedding any MCP server
- **Impact**: Configuration-driven MCP server management, not hardcoded

- **Question**: Is Anthropic API deprecation a blocker for MCP?
- **Answer**: No - MCP client lives in Claudex, not Claude CLI
- **Impact**: Design focuses on Claude CLI backend only

### Missing Requirements Identified

- MCP server configuration format (how users specify servers)
- Tool execution lifecycle (when/how to call MCP tools)
- Error handling for MCP server failures
- Multiple MCP server support

## Goal

Enable Claudex to:
1. Load MCP server configurations from a config file
2. Start/manage MCP server processes (stdio transport)
3. Discover tools from MCP servers at startup
4. Inject discovered tools into Claude CLI requests
5. Parse Claude's tool_calls and execute them via MCP
6. Return tool results to Claude for continued conversation

Additionally: Remove Anthropic API backend code (deprecation).

## Why

- **Unified Tool Architecture**: MCP is the standard protocol for AI tool calling
- **Generic Extensibility**: Any MCP server works with Claudex
- **Simplification**: Single backend (Claude CLI) reduces complexity
- **Pokemon AI League**: Enables end-to-end agent with emulator tools
- **Future-Proof**: MCP ecosystem growing rapidly

## What

### User-Visible Behavior

1. Users configure MCP servers in `claudex.yaml` or environment variables
2. On startup, Claudex starts configured MCP servers
3. Tools from all MCP servers appear in `/v1/chat/completions` tool definitions
4. Claude can call any MCP tool, Claudex executes and returns results
5. Transparent to API consumers - same OpenAI-compatible interface

### Technical Requirements

1. MCP client implementation in Go
2. Stdio transport for MCP server communication
3. Configuration file support for MCP servers
4. Tool discovery via `tools/list` MCP method
5. Tool execution via `tools/call` MCP method
6. Remove `internal/anthropic/` package

### Success Criteria

- [ ] MCP servers start successfully on Claudex startup
- [ ] Tools are discovered and listed via `/v1/models` or config endpoint
- [ ] Claude can successfully call MCP tools
- [ ] Tool results are returned to Claude correctly
- [ ] End-to-end test with Pokemon MCP server passes
- [ ] Anthropic API code removed
- [ ] No regression in existing Claude CLI functionality

## All Needed Context

### Research Phase Summary

- **Codebase patterns found**: Claude CLI executor in `internal/claude/executor.go`
- **External research needed**: MCP specification for Go client implementation
- **Knowledge gaps identified**: MCP stdio protocol specifics

### Documentation & References

```yaml
- url: https://modelcontextprotocol.io/specification
  why: MCP protocol specification for tools/list and tools/call

- url: https://github.com/mark3labs/mcp-go
  why: Potential Go MCP client library (if available)

- file: internal/claude/executor.go
  why: Pattern for CLI invocation and tool prompt injection

- file: internal/models/openai.go
  why: OpenAI tool definitions to map from MCP

- file: internal/converter/converter.go
  why: Response parsing and tool extraction logic
```

### Current Codebase tree

```bash
.
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── anthropic/               # TO BE REMOVED (deprecation)
│   │   └── client.go
│   ├── api/
│   │   ├── handlers/            # HTTP handlers
│   │   ├── middleware/
│   │   └── routes.go
│   ├── claude/
│   │   ├── executor.go          # Claude CLI execution
│   │   └── parser.go            # Response parsing
│   ├── converter/
│   │   └── converter.go         # OpenAI <-> Claude conversion
│   ├── models/
│   │   ├── anthropic.go         # TO BE REMOVED
│   │   ├── claude.go
│   │   └── openai.go            # OpenAI-compatible models
│   └── observability/
├── docs/prp/
└── tests/e2e/
```

### Desired Codebase tree

```bash
.
├── cmd/server/main.go           # Initialize MCP manager
├── config/
│   └── claudex.yaml             # MCP server configuration
├── internal/
│   ├── api/
│   │   ├── handlers/            # Updated for MCP tools
│   │   ├── middleware/
│   │   └── routes.go
│   ├── claude/
│   │   ├── executor.go          # Inject MCP tools
│   │   └── parser.go
│   ├── converter/
│   │   └── converter.go
│   ├── mcp/                     # NEW: MCP client package
│   │   ├── client.go            # MCP client implementation
│   │   ├── manager.go           # Manages multiple MCP servers
│   │   ├── transport.go         # Stdio transport
│   │   └── types.go             # MCP protocol types
│   ├── models/
│   │   ├── claude.go
│   │   ├── mcp.go               # NEW: MCP models
│   │   └── openai.go
│   └── observability/
├── docs/prp/
└── tests/e2e/
    └── test_mcp_tools.py        # NEW: MCP integration tests
```

### Known Gotchas

```go
// CRITICAL: MCP uses JSON-RPC 2.0 over stdio
// Each message is a single line of JSON (newline-delimited)
// Responses may arrive out of order - use request IDs

// CRITICAL: MCP server processes must be managed
// - Start on Claudex startup
// - Restart on failure
// - Graceful shutdown on exit

// CRITICAL: Tool arguments from Claude are JSON strings
// Must parse before passing to MCP tools/call

// GOTCHA: Claude CLI tool calling uses prompt injection
// MCP tools must be converted to the same JSON schema format
```

## Implementation Blueprint

### Data models and structure

```go
// internal/models/mcp.go

// MCPConfig represents MCP server configuration.
type MCPConfig struct {
    Servers []MCPServerConfig `yaml:"servers"`
}

// MCPServerConfig represents a single MCP server.
type MCPServerConfig struct {
    Name    string            `yaml:"name"`
    Command string            `yaml:"command"`
    Args    []string          `yaml:"args,omitempty"`
    Env     map[string]string `yaml:"env,omitempty"`
    Enabled bool              `yaml:"enabled"`
}

// MCPTool represents a tool discovered from MCP.
type MCPTool struct {
    Name        string          `json:"name"`
    Description string          `json:"description,omitempty"`
    InputSchema json.RawMessage `json:"inputSchema"`
    ServerName  string          `json:"-"` // Track which server owns this
}

// MCPToolCall represents a request to execute a tool.
type MCPToolCall struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}

// MCPToolResult represents the result of a tool execution.
type MCPToolResult struct {
    Content []MCPContent `json:"content"`
    IsError bool         `json:"isError,omitempty"`
}

// MCPContent represents content in a tool result.
type MCPContent struct {
    Type string `json:"type"` // "text" | "image" | "resource"
    Text string `json:"text,omitempty"`
    Data string `json:"data,omitempty"`
}
```

### List of tasks

```yaml
Task 1: Create MCP types and models
  CREATE internal/models/mcp.go:
    - Define MCPConfig, MCPServerConfig structs
    - Define MCPTool, MCPToolCall, MCPToolResult
    - Define JSON-RPC message types (Request, Response, Error)
    - MIRROR pattern from: internal/models/openai.go

Task 2: Implement MCP transport layer
  CREATE internal/mcp/transport.go:
    - StdioTransport struct with cmd, stdin, stdout
    - Send() method for JSON-RPC requests
    - Receive() method for JSON-RPC responses
    - Start() and Stop() for process lifecycle
    - CRITICAL: Handle newline-delimited JSON

Task 3: Implement MCP client
  CREATE internal/mcp/client.go:
    - MCPClient struct wrapping transport
    - Initialize() - send initialize request
    - ListTools() - call tools/list method
    - CallTool() - call tools/call method
    - Close() - graceful shutdown
    - MIRROR error handling from: internal/claude/executor.go

Task 4: Implement MCP manager
  CREATE internal/mcp/manager.go:
    - MCPManager struct managing multiple clients
    - LoadConfig() - parse claudex.yaml
    - StartAll() - start all enabled servers
    - GetAllTools() - aggregate tools from all servers
    - CallTool() - route to correct server by tool name
    - StopAll() - graceful shutdown
    - CRITICAL: Handle server crashes with restart

Task 5: Configuration file support
  CREATE config/claudex.yaml.example:
    - Example MCP server configuration
    - Documentation comments
  MODIFY cmd/server/main.go:
    - Load MCP config on startup
    - Initialize MCPManager
    - Pass to handlers

Task 6: Integrate MCP tools with Claude executor
  MODIFY internal/claude/executor.go:
    - Accept MCPManager in constructor
    - Convert MCP tools to OpenAI tool format
    - Include in buildToolsPrompt() when MCP tools present
    - PRESERVE existing tool handling logic

Task 7: Implement tool execution loop
  MODIFY internal/api/handlers/chat.go:
    - After Claude response, check for tool_calls
    - Parse tool calls from response
    - Execute via MCPManager.CallTool()
    - Format results and continue conversation
    - MIRROR streaming pattern from existing code

Task 8: Remove Anthropic API code (deprecation)
  DELETE internal/anthropic/client.go
  DELETE internal/models/anthropic.go
  MODIFY internal/api/handlers/*:
    - Remove Anthropic backend references
    - Update model routing to Claude CLI only
  MODIFY go.mod:
    - Remove anthropic SDK dependency if present

Task 9: Add MCP integration tests
  CREATE tests/e2e/test_mcp_tools.py:
    - Test MCP server startup
    - Test tool discovery
    - Test tool execution
    - Test multi-server support
    - Test error handling

Task 10: Documentation
  UPDATE README.md:
    - Add MCP configuration section
    - Document supported MCP servers
    - Add Pokemon MCP example
```

### Per task pseudocode

```go
// Task 2: Transport layer pseudocode
type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout *bufio.Scanner
    mu     sync.Mutex
}

func (t *StdioTransport) Start(command string, args []string) error {
    // PATTERN: Follow cmd setup from executor.go
    t.cmd = exec.Command(command, args...)
    t.stdin, _ = t.cmd.StdinPipe()
    stdout, _ := t.cmd.StdoutPipe()
    t.stdout = bufio.NewScanner(stdout)
    return t.cmd.Start()
}

func (t *StdioTransport) Send(msg JSONRPCRequest) error {
    // CRITICAL: Must be single line, newline terminated
    t.mu.Lock()
    defer t.mu.Unlock()
    data, _ := json.Marshal(msg)
    _, err := t.stdin.Write(append(data, '\n'))
    return err
}

func (t *StdioTransport) Receive() (JSONRPCResponse, error) {
    // CRITICAL: Blocking read, handle concurrent responses
    if t.stdout.Scan() {
        var resp JSONRPCResponse
        json.Unmarshal(t.stdout.Bytes(), &resp)
        return resp, nil
    }
    return JSONRPCResponse{}, io.EOF
}
```

```go
// Task 7: Tool execution loop pseudocode
func (h *ChatHandler) handleToolCalls(ctx context.Context, toolCalls []models.ToolCall) ([]Message, error) {
    var results []Message

    for _, tc := range toolCalls {
        // Parse arguments from JSON string
        var args json.RawMessage
        json.Unmarshal([]byte(tc.Function.Arguments), &args)

        // Execute via MCP
        result, err := h.mcpManager.CallTool(ctx, tc.Function.Name, args)
        if err != nil {
            // CRITICAL: Return error as tool result, don't fail
            results = append(results, Message{
                Role: "tool",
                ToolCallID: tc.ID,
                Content: fmt.Sprintf("Error: %v", err),
            })
            continue
        }

        // Format result for Claude
        results = append(results, Message{
            Role: "tool",
            ToolCallID: tc.ID,
            Content: result.Content[0].Text,
        })
    }

    return results, nil
}
```

### Integration Points

```yaml
API/ROUTES:
  - add to: internal/api/routes.go
  - pattern: Inject MCPManager into handlers
  - new endpoint: GET /v1/mcp/tools (optional, for debugging)

CONFIG:
  - add to: config/claudex.yaml
  - pattern: YAML configuration with defaults
  - env override: CLAUDEX_MCP_CONFIG_PATH

STARTUP:
  - add to: cmd/server/main.go
  - pattern: Initialize MCPManager before HTTP server
  - graceful shutdown: Stop MCP servers on SIGTERM
```

## Validation Loop

### Level 1: Syntax & Style

```bash
# Run these FIRST - fix any errors before proceeding
go fmt ./...
go vet ./...
golangci-lint run

# Type checking
go build ./...

# Expected: No errors
```

### Level 2: Unit Tests

```bash
# Run unit tests for new MCP package
go test ./internal/mcp/... -v

# Expected: All tests pass
```

### Level 3: Integration Tests

```bash
# Start test MCP server (Pokemon emulator)
cd /path/to/pokemon-mcp && python mcp_server.py &

# Run E2E tests
cd tests/e2e && pytest test_mcp_tools.py -v

# Expected: All tests pass
```

## Final Validation Checklist

- [ ] All tests pass: `go test ./...`
- [ ] No linting errors: `golangci-lint run`
- [ ] Build succeeds: `go build ./cmd/server`
- [ ] MCP server starts: Config loads and servers initialize
- [ ] Tool discovery works: Tools appear in requests
- [ ] Tool execution works: Claude calls tools successfully
- [ ] E2E Pokemon test: Full agent loop with emulator
- [ ] Anthropic code removed: No references remain
- [ ] Streaming works: Tool results in streaming mode
- [ ] Error handling: Graceful degradation on MCP failures

---

## Anti-Patterns to Avoid

- Do not hardcode MCP server paths - use configuration
- Do not block startup on slow MCP servers - use timeouts
- Do not panic on MCP errors - return as tool results
- Do not mix Anthropic and Claude CLI code paths - removed entirely
- Do not assume single MCP server - design for multiple
- Do not parse tool arguments manually - use json.RawMessage
- Do not ignore MCP server crashes - implement restart logic

---

## Configuration Example

```yaml
# config/claudex.yaml
mcp:
  servers:
    - name: pokemon-emulator
      command: python
      args:
        - /path/to/claude-plays-pokemon/mcp_server.py
        - --rom
        - /path/to/pokemon.gb
      enabled: true

    - name: filesystem
      command: npx
      args:
        - -y
        - "@modelcontextprotocol/server-filesystem"
        - /allowed/path
      enabled: false
```

## Deprecation Notice

### Anthropic API Removal

The following will be removed as part of this PRP:

1. `internal/anthropic/client.go` - Direct Anthropic API client
2. `internal/models/anthropic.go` - Anthropic-specific models
3. Any Anthropic SDK dependencies in `go.mod`
4. Backend selection logic (Claude CLI becomes the only option)

**Rationale**:
- Claude CLI provides all necessary functionality
- MCP support requires execution control (Claude CLI provides this)
- Reduces maintenance burden
- Aligns with project focus on CLI-based AI agents

**Migration**: No user action required - Claude CLI backend unchanged.

---

*PRP-002 v1.0 | MCP Server Support | Claude CLI Only | Generic Configuration*
