# AGENTS.md

Stateless Go 1.26+ REST API service for content summarization.

## Commands
- Build: `nix-shell --run "go build ./..."`
- Test: `nix-shell --run "go test ./..."`

## Constraints
- **No git commits/pushes:** Never execute `git commit` or `git push`.
- **Errors:** Always wrap errors using `fmt.Errorf("...: %w", err)`.
- **Architecture:** Layered (`web` -> `summarizer` -> `parser` -> `prompt` -> `llm_gateway`). Keep parsers decoupled from HTTP transport.
