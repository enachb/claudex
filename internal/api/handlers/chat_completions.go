package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/leeaandrob/claudex/internal/claude"
	"github.com/leeaandrob/claudex/internal/converter"
	"github.com/leeaandrob/claudex/internal/models"
	"github.com/leeaandrob/claudex/internal/observability"
	"github.com/valyala/fasthttp"
)

// ChatCompletionsHandler handles chat completion requests.
type ChatCompletionsHandler struct {
	executor  *claude.Executor
	parser    *claude.Parser
	converter *converter.Converter
	metrics   *observability.Metrics
	logger    *observability.Logger
}

// NewChatCompletionsHandler creates a new chat completions handler.
func NewChatCompletionsHandler(
	executor *claude.Executor,
	parser *claude.Parser,
	conv *converter.Converter,
	metrics *observability.Metrics,
	logger *observability.Logger,
) *ChatCompletionsHandler {
	return &ChatCompletionsHandler{
		executor:  executor,
		parser:    parser,
		converter: conv,
		metrics:   metrics,
		logger:    logger,
	}
}

// Handle processes chat completion requests.
func (h *ChatCompletionsHandler) Handle(c *fiber.Ctx) error {
	start := time.Now()
	h.metrics.IncrementActive()
	defer h.metrics.DecrementActive()

	// Parse request body
	var req models.ChatCompletionRequest
	if err := c.BodyParser(&req); err != nil {
		h.metrics.RecordError("parse_error")
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: models.ErrorDetail{
				Message: "Invalid request body: " + err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_json",
			},
		})
	}

	// Validate messages
	if len(req.Messages) == 0 {
		h.metrics.RecordError("validation_error")
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: models.ErrorDetail{
				Message: "Messages array is required and cannot be empty",
				Type:    "invalid_request_error",
				Code:    "invalid_messages",
			},
		})
	}

	// Convert messages to Claude prompt
	prompt, systemPrompt := h.converter.MessagesToPrompt(req.Messages)

	// Branch on stream parameter
	if req.Stream {
		return h.handleStreaming(c, prompt, systemPrompt, req.Model, start)
	}
	return h.handleNonStreaming(c, prompt, systemPrompt, req.Model, start)
}

// handleNonStreaming handles non-streaming requests.
func (h *ChatCompletionsHandler) handleNonStreaming(c *fiber.Ctx, prompt, systemPrompt, model string, start time.Time) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Minute)
	defer cancel()

	claudeStart := time.Now()

	// Execute Claude CLI
	output, err := h.executor.ExecuteNonStreaming(ctx, prompt, systemPrompt)
	if err != nil {
		h.metrics.RecordError("claude_error")
		h.metrics.RecordRequest("error", false, time.Since(start).Seconds())
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: models.ErrorDetail{
				Message: "Failed to execute Claude: " + err.Error(),
				Type:    "server_error",
				Code:    "claude_error",
			},
		})
	}

	h.metrics.RecordClaudeDuration(time.Since(claudeStart).Seconds())

	// Parse Claude response
	claudeResp, err := h.parser.ParseJSONResponse(output)
	if err != nil {
		h.metrics.RecordError("parse_error")
		h.metrics.RecordRequest("error", false, time.Since(start).Seconds())
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: models.ErrorDetail{
				Message: "Failed to parse Claude response: " + err.Error(),
				Type:    "server_error",
				Code:    "parse_error",
			},
		})
	}

	// Convert to OpenAI format
	openaiResp := h.converter.ClaudeToOpenAIResponse(claudeResp, model)

	h.metrics.RecordRequest("success", false, time.Since(start).Seconds())

	return c.JSON(openaiResp)
}

// handleStreaming handles streaming requests with SSE.
func (h *ChatCompletionsHandler) handleStreaming(c *fiber.Ctx, prompt, systemPrompt, model string, start time.Time) error {
	// Set SSE headers - CRITICAL: must be set before any writes
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no") // For nginx proxy

	completionID := converter.GenerateCompletionID()

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer func() {
			h.metrics.RecordRequest("success", true, time.Since(start).Seconds())
		}()

		claudeStart := time.Now()

		// Start streaming from Claude CLI
		chunks, errChan, err := h.executor.ExecuteStreaming(context.Background(), prompt, systemPrompt)
		if err != nil {
			h.metrics.RecordError("claude_error")
			h.writeSSEError(w, "Failed to start Claude: "+err.Error())
			return
		}

		h.metrics.RecordClaudeDuration(time.Since(claudeStart).Seconds())

		isFirst := true

		for line := range chunks {
			msg, err := h.parser.ParseStreamLine(line)
			if err != nil {
				continue
			}

			// Handle stream_event messages with content deltas
			if msg.Type == "stream_event" {
				deltaText := msg.GetDeltaText()
				if deltaText == "" {
					continue
				}

				// Send role-only chunk first (OpenAI SDK expects this)
				if isFirst {
					roleChunk := h.converter.CreateRoleChunk(completionID, model)
					data, _ := json.Marshal(roleChunk)
					fmt.Fprintf(w, "data: %s\n\n", data)
					w.Flush()
					isFirst = false
				}

				// Create chunk with delta text
				chunk := h.converter.CreateContentChunk(completionID, model, deltaText)
				data, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			}
		}

		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				h.metrics.RecordError("claude_error")
				h.writeSSEError(w, err.Error())
				return
			}
		default:
		}

		// Send final chunk with finish_reason
		finalChunk := h.converter.CreateFinalChunk(completionID, model)
		data, _ := json.Marshal(finalChunk)
		fmt.Fprintf(w, "data: %s\n\n", data)

		// Send [DONE] marker
		fmt.Fprintf(w, "data: [DONE]\n\n")
		w.Flush()
	}))

	return nil
}

// writeSSEError writes an error as an SSE event.
func (h *ChatCompletionsHandler) writeSSEError(w *bufio.Writer, message string) {
	errResp := models.ErrorResponse{
		Error: models.ErrorDetail{
			Message: message,
			Type:    "server_error",
			Code:    "claude_error",
		},
	}
	data, _ := json.Marshal(errResp)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	w.Flush()
}
