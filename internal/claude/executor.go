package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Executor handles Claude CLI execution.
type Executor struct{}

// NewExecutor creates a new Claude CLI executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// ExecuteNonStreaming executes Claude CLI and returns the complete response.
func (e *Executor) ExecuteNonStreaming(ctx context.Context, prompt, systemPrompt string) (string, error) {
	args := []string{"-p", "--output-format", "json"}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("claude cli error: %s", stderrStr)
		}
		return "", fmt.Errorf("claude cli error: %w", err)
	}

	return stdout.String(), nil
}

// ExecuteStreaming executes Claude CLI with streaming output.
// Returns a channel that emits each line of output.
func (e *Executor) ExecuteStreaming(ctx context.Context, prompt, systemPrompt string) (<-chan string, <-chan error, error) {
	args := []string{"-p", "--verbose", "--output-format", "stream-json", "--include-partial-messages"}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start claude cli: %w", err)
	}

	chunks := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errChan)

		// Read stderr in a goroutine
		var stderrBuf bytes.Buffer
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				stderrBuf.WriteString(scanner.Text())
				stderrBuf.WriteString("\n")
			}
		}()

		// Read stdout line by line (NDJSON)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				chunks <- line
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("scanner error: %w", err)
			return
		}

		if err := cmd.Wait(); err != nil {
			if stderrBuf.Len() > 0 {
				errChan <- fmt.Errorf("claude cli error: %s", stderrBuf.String())
			} else {
				errChan <- fmt.Errorf("claude cli error: %w", err)
			}
			return
		}
	}()

	return chunks, errChan, nil
}

// IsAvailable checks if the Claude CLI is available.
func (e *Executor) IsAvailable() bool {
	cmd := exec.Command("claude", "--version")
	return cmd.Run() == nil
}
