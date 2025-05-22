# Repository Guidelines

This codebase implements a roguelike game in Go using the Ebiten library.  Each feature lives in its own file (see `README.md` for an overview).

## Coding style
- Use `gofmt -w` on all modified Go files before committing.
- Follow camelCase naming for variables and functions.
- Add comments in Japanese or English matching the surrounding code.
- Keep functions short and related logic grouped in the existing files (e.g. `input.go`, `move.go`).

## Tests
- Unit tests are in `*_test.go` files.  They rely on stub files such as `draw_stub.go` and `fonts_stub.go` when built with the `test` tag.
- Run `go test -tags test ./...` to execute tests in a headless environment.

## Pull requests
- Summaries should briefly describe the change and mention if tests were added or updated.
- Always run `go test -tags test ./...` before submitting a PR.
