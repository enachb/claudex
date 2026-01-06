"""
E2E Tests for Streaming (SSE) Chat Completions.

Tests the streaming response format and behavior.
"""

import pytest
from openai import OpenAI


class TestStreamingChatCompletions:
    """Test streaming chat completion responses."""

    def test_streaming_returns_chunks(self, client: OpenAI, test_messages: list):
        """Test that streaming returns multiple chunks."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        chunks = list(stream)

        # Must have at least one chunk
        assert len(chunks) > 0

    def test_chunk_structure(self, client: OpenAI, test_messages: list):
        """Test that each chunk has correct structure."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        for chunk in stream:
            # ID format
            assert chunk.id is not None
            assert chunk.id.startswith("chatcmpl-")

            # Object type for streaming
            assert chunk.object == "chat.completion.chunk"

            # Timestamp
            assert chunk.created is not None
            assert chunk.created > 0

            # Model
            assert chunk.model is not None

            # Choices
            assert chunk.choices is not None
            assert len(chunk.choices) >= 1

    def test_first_chunk_has_role(self, client: OpenAI, test_messages: list):
        """Test that first chunk contains the assistant role."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        chunks = list(stream)
        first_chunk = chunks[0]

        # First chunk should have role in delta
        assert first_chunk.choices[0].delta.role == "assistant"

    def test_final_chunk_has_finish_reason(self, client: OpenAI, test_messages: list):
        """Test that final chunk has finish_reason=stop."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        chunks = list(stream)
        final_chunk = chunks[-1]

        # Final chunk should have finish_reason
        assert final_chunk.choices[0].finish_reason == "stop"

    def test_accumulated_content_not_empty(self, client: OpenAI, test_messages: list):
        """Test that accumulated content from all chunks is not empty."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        content_parts = []
        for chunk in stream:
            delta = chunk.choices[0].delta
            if delta.content:
                content_parts.append(delta.content)

        accumulated_content = "".join(content_parts)

        # Should have some content
        assert len(accumulated_content) > 0

    def test_chunk_ids_consistent(self, client: OpenAI, test_messages: list):
        """Test that all chunks have the same ID."""
        stream = client.chat.completions.create(
            model="claude",
            messages=test_messages,
            stream=True,
        )

        chunks = list(stream)
        first_id = chunks[0].id

        for chunk in chunks:
            assert chunk.id == first_id

    def test_streaming_with_system_message(self, client: OpenAI):
        """Test streaming with system message."""
        messages = [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "what is 1+1?"},
        ]

        stream = client.chat.completions.create(
            model="claude",
            messages=messages,
            stream=True,
        )

        chunks = list(stream)

        # Should work with system message
        assert len(chunks) > 0

        # Accumulate content
        content = "".join(
            chunk.choices[0].delta.content or ""
            for chunk in chunks
        )
        assert len(content) > 0
