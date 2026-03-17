# ncgi Operational Memory & Project Conventions

## Where Things Live (Architecture Map)
- `main.go` -> Main entrypoint and CGI handling logic (`ServeHTTP`). Includes `buildArgs` and `buildEnv`.
- `errors.go` -> Centralized error reporting functions (`ReportError`, `ReportFatal`).

## Global Rules
1. **Centralized Error Handling:** Empty catch blocks, unhandled return values, and silent failures are forbidden. Every unexpected error, or unhandled return value must be routed through `ReportError` or `ReportFatal` in `errors.go`. This includes `io` return values like `fmt.Fprint` or `stdout.Close()`.
2. **Subprocess/Port Behavior:** The application manages external commands via `handleSubprocess` replacing the token `%PORT%` with the dynamically allocated server port.
3. **Security (CGI Input Sanitation):** The `ncgi` handler passes URL path segments as CLI arguments. You MUST reject any path segment starting with a dash (`-`) with a `400 Bad Request` to prevent Argument/Flag Injection. Do not inject `--` unconditionally.
4. **Tooling Requirements:** `mise` is the official task runner and version manager. Tools in `mise.toml` must be pinned to specific versions. Use `mise run test`, `mise run lint`, `mise run fmt`, etc.
5. **Formatting:** Go code modifications must be formatted using `gofmt` (`mise run fmt:go`) before committing.
