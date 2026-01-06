"""
E2E Tests for Chat Completions (Non-streaming).

Tests the happy path for non-streaming chat completion requests.
"""

import pytest
from openai import OpenAI


class TestNonStreamingChatCompletions:
    """Test non-streaming chat completion responses."""

    def test_response_structure(self, client: OpenAI, test_messages: list):
        """Test that response has correct OpenAI-compatible structure."""
        response = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=False,
        )

        # ID format
        assert response.id is not None
        assert response.id.startswith("chatcmpl-")

        # Object type
        assert response.object == "chat.completion"

        # Timestamp
        assert response.created is not None
        assert response.created > 0

        # Model
        assert response.model is not None
        assert len(response.model) > 0

    def test_choices_structure(self, client: OpenAI, test_messages: list):
        """Test that choices array has correct structure."""
        response = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=False,
        )

        # Choices array
        assert response.choices is not None
        assert len(response.choices) >= 1

        choice = response.choices[0]

        # Index
        assert choice.index == 0

        # Finish reason
        assert choice.finish_reason == "stop"

    def test_message_content(self, client: OpenAI, test_messages: list):
        """Test that message has role and content."""
        response = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=False,
        )

        message = response.choices[0].message

        # Role
        assert message.role == "assistant"

        # Content (should contain answer to "what is 1+1?")
        assert message.content is not None
        assert len(message.content) > 0

    def test_usage_statistics(self, client: OpenAI, test_messages: list):
        """Test that usage statistics are present and valid."""
        response = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=False,
        )

        # Usage object
        assert response.usage is not None

        # Token counts
        assert response.usage.prompt_tokens >= 0
        assert response.usage.completion_tokens >= 0
        assert response.usage.total_tokens >= 0

        # Total should be sum of prompt + completion
        assert response.usage.total_tokens == (
            response.usage.prompt_tokens + response.usage.completion_tokens
        )

    def test_system_message_support(self, client: OpenAI):
        """Test that system messages are handled correctly."""
        messages = [
            {"role": "system", "content": "You are a helpful assistant. Always respond in uppercase."},
            {"role": "user", "content": "say hello"},
        ]

        response = client.chat.completions.create(
            model="claude",
            messages=messages,
            stream=False,
        )

        assert response.choices[0].message.content is not None
        assert len(response.choices[0].message.content) > 0
