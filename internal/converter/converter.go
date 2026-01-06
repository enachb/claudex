package converter

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leeaandrob/claudex/internal/models"
)

// Converter handles format conversion between OpenAI and Claude.
type Converter struct{}

// NewConverter creates a new format converter.
func NewConverter() *Converter {
	return &Converter{}
}

// MessagesToPrompt converts OpenAI messages to Claude prompt format.
// Returns the prompt and system prompt separately.
func (c *Converter) MessagesToPrompt(messages []models.Message) (prompt, systemPrompt string) {
	var systemParts []string
	var conversationParts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemParts = append(systemParts, msg.Content)
		case "user":
			conversationParts = append(conversationParts, "User: "+msg.Content)
		case "assistant":
			conversationParts = append(conversationParts, "Assistant: "+msg.Content)
		}
	}

	systemPrompt = strings.Join(systemParts, "\n")

	// For single user message, use directly without prefix
	// For conversation history, format as dialogue
	if len(conversationParts) == 1 && strings.HasPrefix(conversationParts[0], "User: ") {
		prompt = strings.TrimPrefix(conversationParts[0], "User: ")
	} else {
		prompt = strings.Join(conversationParts, "\n")
	}

	return prompt, systemPrompt
}

// ClaudeToOpenAIResponse converts Claude JSON response to OpenAI format.
func (c *Converter) ClaudeToOpenAIResponse(claudeResp *models.ClaudeJSONResponse, model string) *models.ChatCompletionResponse {
	return &models.ChatCompletionResponse{
		ID:      GenerateCompletionID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.Message{
					Role:    "assistant",
					Content: claudeResp.Result,
				},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}
}

// ClaudeStreamToOpenAIChunk converts Claude streaming message to OpenAI chunk format.
// Note: Role is sent separately via CreateRoleChunk, so isFirst is unused but kept for API compatibility.
func (c *Converter) ClaudeStreamToOpenAIChunk(msg *models.ClaudeStreamMessage, id, model string, isFirst bool, prevContent string) (*models.ChatCompletionChunk, string) {
	chunk := &models.ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChunkChoice{
			{
				Index: 0,
				Delta: models.Delta{},
			},
		},
	}

	// Extract content delta from message
	var currentContent string
	if msg.Message != nil {
		currentContent = msg.Message.GetTextContent()
		// Calculate the delta (new content since last message)
		if len(currentContent) > len(prevContent) {
			chunk.Choices[0].Delta.Content = currentContent[len(prevContent):]
		}
	}

	return chunk, currentContent
}

// CreateRoleChunk creates the first streaming chunk with just the role.
func (c *Converter) CreateRoleChunk(id, model string) *models.ChatCompletionChunk {
	return &models.ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChunkChoice{
			{
				Index: 0,
				Delta: models.Delta{
					Role: "assistant",
				},
			},
		},
	}
}

// CreateContentChunk creates a streaming chunk with content delta.
func (c *Converter) CreateContentChunk(id, model, content string) *models.ChatCompletionChunk {
	return &models.ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChunkChoice{
			{
				Index: 0,
				Delta: models.Delta{
					Content: content,
				},
			},
		},
	}
}

// CreateFinalChunk creates the final streaming chunk with finish_reason.
func (c *Converter) CreateFinalChunk(id, model string) *models.ChatCompletionChunk {
	return &models.ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChunkChoice{
			{
				Index:        0,
				Delta:        models.Delta{},
				FinishReason: "stop",
			},
		},
	}
}

// GenerateCompletionID generates a unique completion ID in OpenAI format.
func GenerateCompletionID() string {
	return "chatcmpl-" + uuid.New().String()
}
