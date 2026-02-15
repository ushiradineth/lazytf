# AGENTS.md

Repository-specific execution guide for coding agents working in `lazytf`.

Precedence
- Nearest `AGENTS.md` wins. This repo currently has one Go project and no nested agent guides.

## Commands
confidence: high

Authoritative CI commands (`.github/workflows/ci.yml`):
- `go mod verify`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run --timeout=5m`
- `go build ./...`

Preferred local commands (`Justfile`):
- `just deps` (download modules)
- `just run` (run app)
- `just dev` (hot reload via `gow`)
- `just build` (build `bin/lazytf`)
- `just fmt` (gofumpt + goimports-reviser + golangci-lint fmt)
- `just vet`
- `just lint` (golangci-lint, timeout 5m)
- `just test` (verbose + race: `go test -v -race ./...`)
- `just test-coverage` (writes `coverage.out` and `coverage.html`)
- `just security` (`govulncheck ./...`)
- `just check-all` (fmt + vet + lint + coverage + security)

## Testing
confidence: high

- Fast default for broad validation: `go test ./...` (matches CI)
- Local stricter run: `just test`
- Integration (tagged): `go test -tags=integration ./test/integration`
- E2E (tagged): `go test -tags=e2e ./test/e2e`
- E2E with race detector: `go test -race -tags=e2e ./test/e2e`
- Notes: integration and e2e tests may require a working `terraform` binary; some e2e flows may require network/provider download.

## Project structure
confidence: high

- `cmd/lazytf/`: CLI entrypoint and wiring
- `internal/ui/`: Bubble Tea model, views, and UI orchestration
- `internal/terraform/`: terraform execution and plan parsing
- `internal/config/`: config models, locking, schema
- `internal/environment/`: folder/workspace detection
- `internal/history/`: SQLite-backed operation history
- `internal/diff/`: diff engine and models
- `internal/integration/`: non-tagged integration-style tests using fake terraform
- `test/integration/`: tagged integration tests against real terraform fixtures
- `test/e2e/`: tagged end-to-end tests
- `testdata/`: fixtures (plans and terraform fixtures)
- `.agents/docs/`: planning notes and migrated legacy docs

Write boundaries
- Prefer edits in `internal/**` and `cmd/lazytf/**` for behavior changes.
- Keep fixtures under `testdata/**` when adding test scenarios.
- Do not treat `bin/`, `coverage.out`, or `coverage.html` as source of truth.

## Code style
confidence: high

- Use `just fmt` before finalizing changes.
- Lint standard is strict, configured in `.golangci.yml`.
- Follow existing Go patterns: small focused functions, explicit error checks, early returns.
- Keep exported identifiers documented when required by lint rules.

Style example:
```go
plan, err := parser.Parse(r)
if err != nil {
	return nil, fmt.Errorf("parse terraform plan: %w", err)
}
return plan, nil
```

## Git workflow
confidence: medium

- Work on a feature branch.
- Keep diffs scoped to the requested change.
- Before every PR push, run `just ci` locally and fix failures before pushing.
- Before PR, run `just check-all` (project contribution guidance).
- If touching CI-sensitive paths, also run CI-equivalent commands (`go mod verify`, `go vet ./...`, `go test ./...`, `golangci-lint run --timeout=5m`, `go build ./...`).

## Boundaries
confidence: high

Always
- Use commands and flags that exist in CI/`Justfile`; do not invent alternatives when verified commands exist.
- Prefer minimal, surgical edits over broad refactors.
- Add or update tests when behavior changes.
- Report what you ran and what could not be run.

Ask first
- Adding/removing dependencies (`go.mod`/`go.sum`) beyond the requested scope.
- Deleting or rewriting large fixture sets under `testdata/`.
- Changes to CI workflows, release logic, or security-sensitive configuration.
- Destructive git operations.

Never
- Commit secrets or credentials.
- Bypass hooks with `--no-verify`.
- Use force-push on protected branches.
- Modify generated artifacts (`bin/*`, coverage files) as a substitute for source fixes.

## Verification checklist

- Confirm changed scope is limited to requested files/areas.
- Run `just ci` before creating or updating a PR branch.
- Run relevant tests first, then broader checks as needed.
- For core-path changes, run at least CI-equivalent validation.
- Ensure formatting and lint pass (`just fmt`, `just lint`).
- Document any skipped checks and why.
