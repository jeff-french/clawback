# Contributing to clawback

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/jeff-french/clawback.git
cd clawback
go build -o clawback .
go test ./...
```

Requires Go 1.22+.

## Submitting Changes

1. Fork the repo and create a feature branch
2. Make your changes
3. Ensure `go test ./...` and `go vet ./...` pass
4. Submit a pull request with a clear description of the change

## Reporting Issues

Open an issue at https://github.com/jeff-french/clawback/issues with:
- What you expected to happen
- What actually happened
- Steps to reproduce

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Write table-driven tests
- No global mutable state

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
