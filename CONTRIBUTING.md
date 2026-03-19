# Contributing to clawback

Thanks for your interest in contributing! This document covers how to get started, submit changes, and what to expect during review.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/<your-username>/clawback.git
   cd clawback
   ```
3. Build and verify everything works:
   ```bash
   go build -o clawback .
   go test ./...
   go vet ./...
   ```
4. Install pre-commit hooks:
   ```bash
   git config core.hooksPath githooks
   ```
   This runs `go build`, `go vet`, `gofmt`, and `golangci-lint` (if installed) before each commit.

Requires **Go 1.22+**.

## Making Changes

1. Create a feature branch from `main`:
   ```bash
   git checkout -b my-feature
   ```
2. Make your changes
3. Ensure tests, vet, and lint pass:
   ```bash
   go test ./...
   go vet ./...
   golangci-lint run  # install: https://golangci-lint.run/welcome/install/
   ```
4. Commit with a clear, concise message describing the change

## Submitting a Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin my-feature
   ```
2. Open a pull request against the `main` branch
3. Include a clear description of:
   - What the change does
   - Why it is needed
   - How it was tested
4. A maintainer will review your PR. You may be asked to make revisions.

## Reporting Issues

Open an issue at https://github.com/jeff-french/clawback/issues with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Your Go version and OS

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Write table-driven tests
- No global mutable state
- Keep functions focused and testable

## Questions and Discussion

If you have questions about the project or want to discuss a potential change before writing code, open a GitHub issue. There is no separate chat or mailing list at this time.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
