# Contributing to Themis

## Quick Start

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/themis.git
cd themis

# Setup
go mod download
cp .env.example .env
# Add your LLM provider key to .env

# Test
go test ./...
```

## Development Workflow

1. **Create a branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

   Use prefixes: `feature/`, `fix/`, `docs/`, `refactor/`, `test/`

2. **Make changes**

   - Write clean, idiomatic Go code
   - Add tests for new functionality
   - Update documentation if needed

3. **Test**

   ```bash
   go test ./...
   go test -race ./...
   ```

4. **Commit**

   Use conventional commits:

   ```
   feat: add weighted_product aggregation method
   fix: resolve API timeout on large contexts
   docs: update installation guide
   ```

5. **Push and create PR**

   ```bash
   git push origin feature/your-feature-name
   ```

   Then create a pull request on GitHub.

## Pull Request Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Code follows Go conventions
- [ ] Documentation updated if needed
- [ ] Commit messages are clear
- [ ] No breaking changes (or documented in PR)

## Adding a New Judge

1. Add judge definition to `configs/judges.yaml`:

   ```yaml
   - name: your-judge
     enabled: true
     weight: 0.2
     prompt: |
       Your evaluation prompt...
   ```

2. Add test case in `docs/testing/api-tests.md`

3. Test locally:

   ```bash
   go run cmd/api/main.go
   curl -X POST http://localhost:18082/api/v1/evaluate/judge/your-judge -d '{...}'
   ```

## Adding a New Aggregation Method

1. Implement in `internal/aggregator/aggregator.go`
2. Add to `JUDGE_AGGREGATION_METHOD` options in config
3. Update `docs/getting-started/configuration.md`
4. Add test cases

## Code Style

Follow standard Go conventions:

- Use `gofmt` for formatting
- Run `golangci-lint run` before committing
- Keep functions small and focused
- Write clear comments for exported functions
- Return errors, don't panic

## Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/judge/...

# With race detection
go test -race ./...
```

Write table-driven tests:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    Input
        expected Output
    }{
        {"valid input", validInput, expectedOutput},
        {"edge case", edgeInput, edgeOutput},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Function(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Documentation

- Add YAML frontmatter to new docs:

  ```yaml
  ---
  title: Document Title
  description: Brief description
  version: 1.0.0
  tags: [tag1, tag2]
  ---
  ```

- Update `docs/INDEX.md` if adding new sections
- Include code examples that work

## Reporting Issues

**Bug report**: Include environment, steps to reproduce, expected vs actual behavior.

**Feature request**: Describe the use case and proposed solution.

Use labels: `bug`, `enhancement`, `documentation`, `good first issue`

## Community

- Be respectful and constructive
- Help newcomers
- Read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Report security issues to security@terminus-lab.com (see [SECURITY.md](SECURITY.md))

## Questions?

- Open a [GitHub Discussion](https://github.com/Terminus-Lab/themis/discussions)
- Check [documentation](docs/INDEX.md)
- Review existing [issues](https://github.com/Terminus-Lab/themis/issues)

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
