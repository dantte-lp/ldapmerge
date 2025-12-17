# Contributing to ldapmerge

Thank you for your interest in contributing to ldapmerge!

## Development Setup

### Prerequisites

- Go 1.25+
- Make
- Docker (optional, for container builds)

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ldapmerge.git
   cd ldapmerge
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Install pre-commit hooks:
   ```bash
   pre-commit install
   ```

### Building

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
```

### Testing

```bash
make test           # Run tests
make test-coverage  # Run tests with coverage
```

### Linting

```bash
make lint           # Run golangci-lint
```

## Pull Request Process

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and ensure:
   - All tests pass (`make test`)
   - Code passes linting (`make lint`)
   - Code is formatted (`go fmt ./...`)

3. Commit your changes with a clear message:
   ```bash
   git commit -m "Add feature: brief description"
   ```

4. Push to your fork and create a Pull Request

5. Wait for review and address any feedback

## Code Style

- Follow standard Go conventions
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and concise

## Reporting Issues

When reporting issues, please include:

- ldapmerge version (`ldapmerge version`)
- Operating system and architecture
- Steps to reproduce the issue
- Expected vs actual behavior
- Relevant logs or error messages

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
