package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/leeaandrob/claudex/internal/models"
)

// StdioTransport handles communication with an MCP server via stdio.
// It implements JSON-RPC 2.0 over newline-delimited JSON (NDJSON).
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	stderr    io.ReadCloser
	mu        sync.Mutex
	requestID int64
	running   bool
	serverEnv map[string]string
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{}
}

// Start starts the MCP server process.
func (t *StdioTransport) Start(command string, args []string, env map[string]string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.cmd = exec.Command(command, args...)
	t.serverEnv = env

	// Set up environment
	t.cmd.Env = os.Environ()
	for key, value := range env {
		// Expand environment variables in the value
		expandedValue := os.ExpandEnv(value)
		t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", key, expandedValue))
	}

	// Create pipes for stdin, stdout, stderr
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		t.stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	t.stdout = bufio.NewScanner(stdout)
	// Increase scanner buffer for large JSON responses
	t.stdout.Buffer(make([]byte, 64*1024), 1024*1024)

	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		t.stdin.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	t.stderr = stderr

	// Start the process
	if err := t.cmd.Start(); err != nil {
		t.stdin.Close()
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	t.running = true
	t.requestID = 0

	// Drain stderr in background to prevent blocking
	go t.drainStderr()

	return nil
}

// drainStderr reads and discards stderr to prevent the process from blocking.
func (t *StdioTransport) drainStderr() {
	if t.stderr == nil {
		return
	}
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// Could log stderr here if needed for debugging
		_ = scanner.Text()
	}
}

// Stop stops the MCP server process.
func (t *StdioTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	// Close stdin to signal the server to shut down
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for the process to exit
	if t.cmd != nil && t.cmd.Process != nil {
		// Try graceful shutdown first
		t.cmd.Process.Signal(os.Interrupt)

		// Wait for process (with timeout handled by caller)
		err := t.cmd.Wait()
		if err != nil {
			// Force kill if graceful shutdown failed
			t.cmd.Process.Kill()
		}
	}

	return nil
}

// IsRunning returns whether the transport is running.
func (t *StdioTransport) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// Send sends a JSON-RPC request and returns the response.
func (t *StdioTransport) Send(method string, params interface{}) (*models.JSONRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil, fmt.Errorf("transport not running")
	}

	// Generate unique request ID
	id := int(atomic.AddInt64(&t.requestID, 1))

	request := models.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Marshal request to JSON
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request with newline (NDJSON format)
	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	if !t.stdout.Scan() {
		if err := t.stdout.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("connection closed")
	}

	line := t.stdout.Text()
	if line == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Parse response
	var response models.JSONRPCResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (line: %s)", err, truncate(line, 100))
	}

	// Verify response ID matches request ID
	if response.ID != id {
		return nil, fmt.Errorf("response ID mismatch: expected %d, got %d", id, response.ID)
	}

	return &response, nil
}

// SendNotification sends a JSON-RPC notification (no response expected).
func (t *StdioTransport) SendNotification(method string, params interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return fmt.Errorf("transport not running")
	}

	// Notifications don't have an ID
	notification := struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ExpandEnvVars expands environment variables in a string.
// Supports ${VAR} and $VAR syntax.
func ExpandEnvVars(s string) string {
	// Use os.ExpandEnv which handles ${VAR} and $VAR
	return os.ExpandEnv(s)
}

// BuildEnvSlice builds an environment slice from a map.
func BuildEnvSlice(envMap map[string]string) []string {
	baseEnv := os.Environ()
	for key, value := range envMap {
		// Check if value references other env vars
		if strings.Contains(value, "${") || strings.Contains(value, "$") {
			value = ExpandEnvVars(value)
		}
		baseEnv = append(baseEnv, fmt.Sprintf("%s=%s", key, value))
	}
	return baseEnv
}
