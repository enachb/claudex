# Contributing to Claudex

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

Before submitting a bug report:

1. Check existing [issues](https://github.com/leeaandrob/claudex/issues) to avoid duplicates
2. Use the bug report template when creating a new issue
3. Include as much detail as possible:
   - Go version (`go version`)
   - Claude CLI version (`claude --version`)
   - Operating system
   - Steps to reproduce
   - Expected vs actual behavior

### Suggesting Features

1. Check existing issues for similar suggestions
2. Use the feature request template
3. Explain the use case and benefits

### Pull Requests

#### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/claudex.git
   cd claudex
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/leeaandrob/claudex.git
   ```

4. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

#### Development Setup

```bash
# Install dependencies
go mod tidy

# Run the server
make run

# Run tests
make test

# Run linter
make lint

# Run E2E tests (requires Claude CLI authenticated)
make test-e2e-setup
make test-e2e
```

#### Code Style

- Follow standard Go conventions ([Effective Go](https://go.dev/doc/effective_go))
- Run `gofmt` before committing
- Keep functions small and focused
- Add comments for exported functions
- Write meaningful commit messages

#### Testing

- Add tests for new functionality
- Ensure all existing tests pass
- For API changes, update E2E tests in `tests/e2e/`

#### Commit Messages

Use clear, descriptive commit messages:

```
feat: add support for multi-turn conversations
fix: resolve streaming timeout issue
docs: update README with Docker instructions
test: add E2E tests for error handling
refactor: simplify message converter logic
```

#### Submitting

1. Ensure all tests pass:
   ```bash
   make test
   make test-e2e
   ```

2. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. Create a Pull Request against `main` branch

4. Fill out the PR template completely

5. Wait for review and address any feedback

## Development Guidelines

### Project Structure

```
.
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── api/handlers/    # HTTP handlers
│   ├── claude/          # Claude CLI integration
│   ├── converter/       # Format conversion (OpenAI <-> Claude)
│   ├── models/          # Data structures
│   └── observability/   # Logging, metrics, tracing
├── tests/e2e/           # End-to-end tests
├── k8s/                 # Kubernetes manifests
└── docs/                # Documentation
```

### Adding New Features

1. Discuss major changes in an issue first
2. Keep changes focused and minimal
3. Update documentation as needed
4. Add appropriate tests

### API Compatibility

This project aims to be compatible with the OpenAI Chat Completions API. When making changes:

- Maintain compatibility with OpenAI SDK clients
- Follow [OpenAI API reference](https://platform.openai.com/docs/api-reference/chat)
- Test with the official OpenAI Python SDK

## Questions?

Feel free to open an issue for any questions about contributing.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
