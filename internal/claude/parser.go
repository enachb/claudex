package claude

import (
	"encoding/json"
	"fmt"

	"github.com/leeaandrob/claudex/internal/models"
)

// Parser handles parsing of Claude CLI output.
type Parser struct{}

// NewParser creates a new Claude output parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseJSONResponse parses a non-streaming Claude CLI JSON response.
func (p *Parser) ParseJSONResponse(output string) (*models.ClaudeJSONResponse, error) {
	var resp models.ClaudeJSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse claude json response: %w", err)
	}
	return &resp, nil
}

// ParseStreamLine parses a single line from Claude CLI stream-json output.
func (p *Parser) ParseStreamLine(line string) (*models.ClaudeStreamMessage, error) {
	var msg models.ClaudeStreamMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse claude stream line: %w", err)
	}
	return &msg, nil
}
