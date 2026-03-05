# Operational Memory

- `main.go` -> Entrypoint, routing, CGI translation, subprocess sidecar execution.
- `errors.go` -> Centralized error reporting via `ReportError` and `ReportFatal`.
- `mise.toml` -> Task and tool version management, dependencies.
- `.github/workflows/` -> CI and Release processes (specifically autorelease.yml when present).

# Conventions

## Documentation
- Use `/** ... */` syntax for docstrings on types and functions, even in Go, to maintain consistency for LLMs and avoid line comment ambiguity unless inside a function body.
- Docstrings must explain the *why*, nuances (like `%PORT%` substitutions), and execution flow, avoiding obvious repetitions of the signature.

## Error Handling
- Every error must be routed to `ReportError` or `ReportFatal` in `errors.go`.
- Never leave an empty catch block or discard an error implicitly (e.g., in a `defer` call to `Close()`).
