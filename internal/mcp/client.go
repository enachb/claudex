package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/leeaandrob/claudex/internal/models"
)

const (
	// MCPProtocolVersion is the MCP protocol version we support.
	MCPProtocolVersion = "2024-11-05"

	// DefaultInitTimeout is the default timeout for initialization.
	DefaultInitTimeout = 30 * time.Second

	// DefaultCallTimeout is the default timeout for tool calls.
	DefaultCallTimeout = 60 * time.Second
)

// Client represents an MCP client that communicates with a single MCP server.
type Client struct {
	name        string
	transport   *StdioTransport
	tools       []models.MCPTool
	serverInfo  models.MCPImplementationInfo
	initialized bool
	initTimeout time.Duration
	callTimeout time.Duration
	mu          sync.RWMutex
}

// NewClient creates a new MCP client.
func NewClient(name string) *Client {
	return &Client{
		name:        name,
		transport:   NewStdioTransport(),
		tools:       []models.MCPTool{},
		initTimeout: DefaultInitTimeout,
		callTimeout: DefaultCallTimeout,
	}
}

// SetTimeouts sets the initialization and call timeouts.
func (c *Client) SetTimeouts(initTimeout, callTimeout time.Duration) {
	c.initTimeout = initTimeout
	c.callTimeout = callTimeout
}

// Start starts the MCP server and initializes the connection.
func (c *Client) Start(ctx context.Context, command string, args []string, env map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Start the transport
	if err := c.transport.Start(command, args, env); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// Initialize the MCP connection
	if err := c.initialize(ctx); err != nil {
		c.transport.Stop()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Discover available tools
	if err := c.discoverTools(ctx); err != nil {
		c.transport.Stop()
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	c.initialized = true
	return nil
}

// initialize sends the initialize request to the MCP server.
func (c *Client) initialize(ctx context.Context) error {
	initParams := models.MCPInitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: models.MCPClientCapabilities{
			Roots: &models.MCPRootsCapability{
				ListChanged: false,
			},
		},
		ClientInfo: models.MCPImplementationInfo{
			Name:    "claudex",
			Version: "1.0.0",
		},
	}

	// Create a channel to receive the response
	resultCh := make(chan error, 1)

	go func() {
		response, err := c.transport.Send("initialize", initParams)
		if err != nil {
			resultCh <- fmt.Errorf("initialize request failed: %w", err)
			return
		}

		if response.Error != nil {
			resultCh <- fmt.Errorf("initialize error: %s (code: %d)", response.Error.Message, response.Error.Code)
			return
		}

		// Parse the result
		var result models.MCPInitializeResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			resultCh <- fmt.Errorf("failed to parse initialize result: %w", err)
			return
		}

		c.serverInfo = result.ServerInfo

		// Send initialized notification
		if err := c.transport.SendNotification("notifications/initialized", nil); err != nil {
			resultCh <- fmt.Errorf("failed to send initialized notification: %w", err)
			return
		}

		resultCh <- nil
	}()

	// Wait for response or timeout
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.initTimeout):
		return fmt.Errorf("initialize timeout after %v", c.initTimeout)
	}
}

// discoverTools fetches the list of available tools from the server.
func (c *Client) discoverTools(ctx context.Context) error {
	resultCh := make(chan error, 1)

	go func() {
		response, err := c.transport.Send("tools/list", nil)
		if err != nil {
			resultCh <- fmt.Errorf("tools/list request failed: %w", err)
			return
		}

		if response.Error != nil {
			resultCh <- fmt.Errorf("tools/list error: %s (code: %d)", response.Error.Message, response.Error.Code)
			return
		}

		var result models.MCPToolsListResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			resultCh <- fmt.Errorf("failed to parse tools/list result: %w", err)
			return
		}

		// Tag each tool with the server name
		for i := range result.Tools {
			result.Tools[i].ServerName = c.name
		}

		c.tools = result.Tools
		resultCh <- nil
	}()

	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.initTimeout):
		return fmt.Errorf("tools/list timeout after %v", c.initTimeout)
	}
}

// GetTools returns the list of available tools.
func (c *Client) GetTools() []models.MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// GetName returns the client name.
func (c *Client) GetName() string {
	return c.name
}

// GetServerInfo returns information about the connected server.
func (c *Client) GetServerInfo() models.MCPImplementationInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// IsInitialized returns whether the client is initialized.
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// CallTool executes a tool and returns the result.
func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*models.MCPToolResult, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	params := models.MCPToolsCallParams{
		Name:      name,
		Arguments: arguments,
	}

	resultCh := make(chan struct {
		result *models.MCPToolResult
		err    error
	}, 1)

	go func() {
		response, err := c.transport.Send("tools/call", params)
		if err != nil {
			resultCh <- struct {
				result *models.MCPToolResult
				err    error
			}{nil, fmt.Errorf("tools/call request failed: %w", err)}
			return
		}

		if response.Error != nil {
			// Return error as tool result, not as Go error
			// This allows the conversation to continue
			resultCh <- struct {
				result *models.MCPToolResult
				err    error
			}{
				&models.MCPToolResult{
					Content: []models.MCPContent{{
						Type: "text",
						Text: fmt.Sprintf("Tool error: %s (code: %d)", response.Error.Message, response.Error.Code),
					}},
					IsError: true,
				},
				nil,
			}
			return
		}

		var result models.MCPToolsCallResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			resultCh <- struct {
				result *models.MCPToolResult
				err    error
			}{nil, fmt.Errorf("failed to parse tools/call result: %w", err)}
			return
		}

		resultCh <- struct {
			result *models.MCPToolResult
			err    error
		}{
			&models.MCPToolResult{
				Content: result.Content,
				IsError: result.IsError,
			},
			nil,
		}
	}()

	select {
	case res := <-resultCh:
		return res.result, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.callTimeout):
		return nil, fmt.Errorf("tools/call timeout after %v", c.callTimeout)
	}
}

// HasTool checks if the client has a tool with the given name.
func (c *Client) HasTool(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, tool := range c.tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initialized = false
	return c.transport.Stop()
}
