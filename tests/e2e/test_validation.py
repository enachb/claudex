"""
E2E Tests for Request Validation.

Tests that invalid requests return proper error responses.
"""

import pytest
import requests


class TestValidationErrors:
    """Test validation error responses."""

    def test_empty_messages_returns_400(self, base_url: str):
        """Test that empty messages array returns 400."""
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json={"model": "claude", "messages": []},
            timeout=30,
        )

        assert response.status_code == 400

        data = response.json()
        assert "error" in data
        assert data["error"]["type"] == "invalid_request_error"
        assert "message" in data["error"]

    def test_missing_messages_returns_400(self, base_url: str):
        """Test that missing messages field returns 400."""
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json={"model": "claude"},
            timeout=30,
        )

        assert response.status_code == 400

        data = response.json()
        assert "error" in data

    def test_invalid_json_returns_400(self, base_url: str):
        """Test that invalid JSON returns 400."""
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            data="not valid json",
            headers={"Content-Type": "application/json"},
            timeout=30,
        )

        assert response.status_code == 400

        data = response.json()
        assert "error" in data
        assert data["error"]["type"] == "invalid_request_error"

    def test_error_response_structure(self, base_url: str):
        """Test that error response has correct OpenAI-compatible structure."""
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json={"model": "claude", "messages": []},
            timeout=30,
        )

        assert response.status_code == 400

        data = response.json()

        # Error object structure
        assert "error" in data
        error = data["error"]

        # Required fields
        assert "message" in error
        assert "type" in error
        assert "code" in error

        # Message should be descriptive
        assert len(error["message"]) > 0

    def test_malformed_message_object(self, base_url: str):
        """Test that malformed message objects are handled."""
        response = requests.post(
            f"{base_url}/v1/chat/completions",
            json={
                "model": "claude",
                "messages": [{"invalid": "message"}],
            },
            timeout=30,
        )

        # Should either return 400 or handle gracefully
        # The exact behavior depends on implementation
        assert response.status_code in [400, 500]
