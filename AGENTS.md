# AGENTS.md

Guidance for coding agents working in `gymctl`.
This file is repository-specific and should override generic habits when they conflict.

## Project Snapshot

- Language: Go (`go 1.24.x`)
- Module: `gymctl`
- Entry point: `cmd/gymctl/main.go`
- Primary packages: `internal/cli`, `internal/checks`, `internal/environment`, `internal/scenario`, `internal/progress`
- Build artifact: `build/gymctl` (via Makefile)
- CI baseline: `.github/workflows/ci.yml` currently builds `./cmd/gymctl`

## Cursor / Copilot Rules

I checked for repository-level agent instruction files:

- `.cursorrules`
- `.cursor/rules/`
- `.github/copilot-instructions.md`

None were found in this repo at the time this file was generated.
If any are added later, treat them as required constraints and update this file.

## Canonical Commands

Run commands from repo root: `/home/jg/git/shart-cloud-gh/containers/gymctl`.

### Build

- `make build` - build binary into `build/gymctl` with version ldflags
- `go build ./cmd/gymctl` - minimal CI-equivalent build
- `go build -o build/gymctl ./cmd/gymctl` - explicit local build output
- `make build-all` - cross-compile linux/darwin/windows targets

### Run

- `make run`
- `go run ./cmd/gymctl`

### Format / Vet / Lint

- `make fmt` - runs `go fmt ./...`
- `go fmt ./...` - format packages
- `make vet` - runs `go vet ./...`
- `go vet ./...`
- `make lint` - runs `golangci-lint run` if installed
- `golangci-lint run` - preferred lint command when available

### Test

- `make test` - runs `go test -v ./...`
- `go test ./...` - run all tests
- `go test -v ./...` - verbose all tests
- `make test-coverage` - quick coverage view
- `make coverage` - writes `coverage.out` and `coverage.html`

### Run Exactly One Test (Important)

Use `-run` with a regex and limit package scope whenever possible.

- Single test in one package:
  - `go test -v ./internal/checks -run '^TestCompareValue$'`
- Single subtest:
  - `go test -v ./internal/checks -run 'TestRunHTTPCheck/health_check_success'`
- Multiple related tests by prefix:
  - `go test -v ./internal/checks -run '^TestRun(Dockerfile|File)Check$'`
- Run one package only:
  - `go test ./internal/cli`
- Disable test caching when debugging flaky behavior:
  - `go test -count=1 -v ./internal/checks -run '^TestRunScriptCheck$'`

### Useful Dev Shortcuts

- `make dev` - fmt + vet + test + build
- `make ci` - deps + fmt + vet + lint + test + build
- `make tidy` - `go mod tidy`
- `make deps` - `go mod download`

## Code Style Guidelines

These rules are inferred from existing code and should be preserved.

### General

- Keep changes small and package-local unless refactor is required.
- Prefer extending existing abstractions over introducing new frameworks.
- Preserve current CLI behavior and output tone unless task explicitly changes UX.

### Formatting

- Always run `go fmt ./...` after edits.
- Keep code gofmt-clean; do not manually align spacing.
- Keep functions focused; extract helpers when branches become hard to scan.

### Imports

- Group imports in gofmt default order:
  1) stdlib
  2) third-party
  3) local module imports (`gymctl/...`)
- Avoid alias imports unless required for disambiguation.
- Remove unused imports; do not leave commented imports.

### Types and Structs

- Use explicit struct types for schema-heavy data (`scenario`, `progress`).
- Preserve existing YAML/JSON tags and omitempty semantics.
- Add pointer fields (`*bool`, `*int`) only when tri-state/optional behavior is needed.
- Prefer zero-value-friendly structs and maps initialized where needed.

### Naming

- Exported identifiers: PascalCase with clear domain meaning.
- Unexported identifiers: camelCase.
- Command constructors follow `newXCmd()` convention.
- Option structs follow `<command>Options` pattern (for Cobra commands).
- Avoid abbreviations unless already conventional (`ctx`, `cmd`, `err`).

### Error Handling

- Return errors instead of panicking (except top-level guarded recovery patterns already used).
- Wrap errors with context using `%w` when propagating underlying causes.
  - Example pattern used: `fmt.Errorf("read progress: %w", err)`
- Use plain `%s` messages when not preserving a nested error.
- For CLI command handlers, return errors and let Cobra/root execution path handle exit.
- Prefer actionable error text for user-facing failures.

### Context and External Commands

- Thread `context.Context` through operations that can block.
- For subprocesses, use existing runner/environment helpers instead of ad hoc `exec.Command` wiring.
- Respect timeout fields in checks/specs (`time.ParseDuration` patterns exist).

### File and Path Safety

- Resolve user paths via helper functions (`resolveGymDir`, `resolveProgressFile`, etc.).
- Use `filepath.Join` and avoid hardcoded path separators.
- Match existing file mode conventions:
  - dirs: `0o755`
  - files: `0o644`

### CLI / UX Conventions

- Keep Cobra command shape consistent: `Use`, `Short`, `Args`, `RunE`.
- Keep output user-friendly and concise; existing code uses colored helpers and emoji in places.
- Avoid printing directly to stdout in deep packages; prefer returning values/errors.

### Testing Conventions

- Prefer table-driven tests with `name` fields and `t.Run(...)`.
- Keep assertions straightforward with clear failure messages.
- Use temp dirs/files for filesystem tests (`os.MkdirTemp`, `defer os.RemoveAll`).
- Keep tests deterministic; avoid network/external dependency unless using local test servers (`httptest`).

## Change Validation Checklist (for agents)

Before finishing, run the smallest relevant set:

1. `go fmt ./...`
2. `go test ./...` (or targeted package/test when iterating)
3. `go vet ./...` for non-trivial logic changes
4. `golangci-lint run` when available or when touching broad surfaces
5. `go build ./cmd/gymctl` for CLI-impacting changes

If you only changed a narrow area, run targeted tests first, then broaden as needed.

## Notes for Future Updates

- If CI starts running lint/tests, mirror those exact commands here.
- If Cursor/Copilot instruction files are added, add a dedicated section summarizing mandatory rules.
- Keep this file practical and command-focused; avoid generic Go advice not used by this repo.
