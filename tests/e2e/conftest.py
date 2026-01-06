"""
E2E Test Fixtures for OpenAI-Claude Proxy.

Provides fixtures for testing with real Claude CLI.
"""

import os
import pytest
from openai import OpenAI


def get_base_url() -> str:
    """Get the base URL for the proxy server."""
    host = os.environ.get("PROXY_HOST", "localhost")
    port = os.environ.get("PROXY_PORT", "8080")
    return f"http://{host}:{port}"


@pytest.fixture(scope="session")
def base_url() -> str:
    """Base URL fixture for direct HTTP requests."""
    return get_base_url()


@pytest.fixture(scope="session")
def client() -> OpenAI:
    """OpenAI client configured to use the proxy."""
    return OpenAI(
        base_url=f"{get_base_url()}/v1",
        api_key="not-needed",  # Auth handled by Claude CLI
        timeout=120.0,
    )


@pytest.fixture
def test_prompt() -> str:
    """Standard test prompt for deterministic responses."""
    return "what is 1+1?"


@pytest.fixture
def test_messages(test_prompt) -> list:
    """Standard test messages."""
    return [{"role": "user", "content": test_prompt}]
