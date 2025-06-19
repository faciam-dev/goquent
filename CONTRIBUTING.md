# Contributing

Thank you for considering a contribution to this project.

## Development Setup

1. Install Go 1.23 or later.
2. Start the MySQL container:
   ```bash
   docker-compose up -d
   ```
3. Run the test suite:
   ```bash
   go test ./...
   ```

## Coding Style

- Follow the directory structure and dependency rules described in `AGENT.MD`.
- Keep comments and documentation in English.
- Use `gofmt` before committing changes.

