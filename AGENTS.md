# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## Project

Official MongoDB Go Driver (`go.mongodb.org/mongo-driver/v2`). Correctness, backward compatibility, and conformance to [MongoDB Driver Specifications](https://github.com/mongodb/specifications) are top priorities.

- Module: `go.mongodb.org/mongo-driver/v2`
- Issue tracker: [JIRA GODRIVER project](https://jira.mongodb.org/browse/GODRIVER) — GitHub Issues is not used for bugs/features
- Active major version: v2 on `master`. v1 lives on `release/1.x` branches.

## Build, test, and lint

Uses **[Task](https://taskfile.dev/)**, not make.

```
task                 # default: build + check-license + check-fmt + check-modules + lint + test-short
task fmt             # gofumpt -w . — use this, not plain gofmt
task build           # includes compilecheck (all supported Go versions, needs Docker)
task lint            # golangci-lint across linux/{386,arm,arm64,amd64,ppc64le,s390x}
task test-short      # race detector, ~60s timeout — fast feedback
task test            # full suite, serial (-p 1), 1800s timeout, requires mongod
task api-report      # required when public API changes — include output in PR description
```

**Tests require a running `mongod`** on `localhost:27017` (or `MONGODB_URI`). The full test suite is **serial** (`-p 1`) — integration tests share server state. Do not parallelize.

## Go version requirements

- **Go 1.25**: minimum to compile/use the driver. The Go Driver supports the last two Go minor versions. Public packages (`mongo/`, `bson/`, `event/`, `tag/`) must build on 1.25. The `compilecheck` task enforces this at build time, compiling against every version in `goVersions` (`internal/test/compilecheck/compile_check_test.go`).
- **Go 1.26+**: required to run the test suite and develop the driver.

## CI (Evergreen)

All CI lives in `.evergreen/config.yml`. Validate any change with
`evergreen validate -p mongo-go-driver -f .evergreen/config.yml` (authenticate with `evergreen login`, not an API key). Warnings about unused tasks and merge-queue aliases are pre-existing.

**Two submodules, both pinned by commit.** `.evergreen/drivers-evergreen-tools` (DET) provides every server-provisioning script; `testdata/specifications` provides spec-test fixtures. CI checks out the pinned commits (`git submodule update --init` in the `setup-system` function) — it does **not** track DET `master`. So a fix that has landed upstream in DET is not live here until the pin moves. Run `task init-submodule` locally before inspecting DET; it is often left uninitialized.

**Adding a new server version:**

1. Add a value at the top of the `version` axis (`id`/`display_name`/`variables: VERSION:`).
2. Append it to each relevant `matrix_spec` version list. Matrices tagged `pullrequest` run on every PR; untagged ones are waterfall-only. Convention is that the newest version gets the PR-tagged matrix and the previous newest falls back to the waterfall matrix.
3. Variants not expanded across the `version` axis need editing by hand — notably `testoidc_task_group`, which hardcodes `MONGODB_VERSION`. Task groups cannot take a matrix axis, and a task may appear only once per buildvariant.

**When a version won't download, the fix usually belongs in DET, not here.** `VERSION` flows to `mongodl` unchanged (`bootstrap-mongo-orchestration` → `run-orchestration.sh` → `drivers_orchestration.py`), and `mongodl` resolves a bare `X.Y` to the newest `X.Y.*` in `full.json`, release candidates included. Pre-GA versions are often absent from `full.json`, and DET carries the workaround (e.g. `UNPUBLISHED_VERSIONS` maps such a version to its branch nightly). Prefer bumping the DET pin over adding driver-side overrides such as `MONGODB_DOWNLOAD_SOURCE`:

```
cd .evergreen/drivers-evergreen-tools
git fetch origin
git checkout <upstream-commit>      # or origin/master
cd -
git add .evergreen/drivers-evergreen-tools
```

Dependabot bumps this pin routinely, so prefer waiting for its PR when the fix is not urgent. Bumping by hand pulls in every intervening upstream commit — review `git log <old>..<new>` for unrelated CI changes and validate in a patch build (`evergreen patch -p mongo-go-driver -u`), not just `evergreen validate`.

## Gotchas

1. **`x/` packages are importable but not semver-covered.** Users do import `x/mongo/driver` etc. Flag breaking changes explicitly in PR descriptions even though semver doesn't require it.
1. **Atomic alignment on 32-bit.** `int64`/`uint64` fields accessed via `sync/atomic` must be at struct start or use `atomic.Int64`. The lint pass on `GOARCH=386` catches this — don't skip cross-arch lint.
1. **GridFS lives in `mongo/`.** v2 merged `gridfs.Bucket` into `mongo.GridFSBucket`. Don't resurrect the old `gridfs` import path.
1. **CSOT timeouts.** Use context deadlines; do not introduce per-call wall-clock timers that race with the user's context.
1. **Command monitoring redacts credentials.** When adding commands that handle credentials (`authenticate`, `saslStart`, etc.), add them to the redaction list.
1. **`libmongocrypt` required for CSFLE/QE tests.** Install with `task install-libmongocrypt`. CSFLE code requires the `cse` build tag.

## Code conventions

- **Formatter**: `gofumpt` (stricter superset of gofmt). Always use `task fmt`, never plain gofmt.
- **Errors**: `fmt.Errorf("…: %w", err)` for wrapping. Use `errors.Is`/`errors.As` for comparisons — never `strings.Contains` on error messages. Sentinel errors live in the `mongo` package (`mongo.ErrNoDocuments`, etc.).
- **Context**: Every blocking public method takes `context.Context` as its first argument. Use `mongo.SessionFromContext(ctx)` (v2 pattern) — not the old `mongo.SessionContext`.
- **BSON**: `bson.D` for ordered documents and commands. `bson.M` only when order genuinely doesn't matter. `bsoncore.Document` in hot paths (zero-copy, byte-slice-backed).
- **Exported names**: Full words, not abbreviations (`ClientBulkWriteResult`, not `CBWResult`). Doc comments required on all exported identifiers using the "name is subject" convention.
- **New options**: Follow the setter pattern: `func (o *XxxOptionsBuilder) SetFoo(v T) *XxxOptionsBuilder`.

## Tests

- Use `internal/assert` (wraps testify) for assertions. Prefer `assert.ErrorIs` over `assert.ErrorContains`.
- Integration tests: use `internal/integration/mtest` for topology-aware setup, fail-point injection, and event capture.
- Spec tests: JSON/YAML fixtures under `testdata/` (git submodule — run `task init-submodule` after cloning if missing).
- Prose tests: number them to match the spec (e.g., `"3. bulkWrite batch splits…"`) for cross-referencing.
- Unit tests live next to the code (`foo.go` ↔ `foo_test.go`). Integration tests live in `internal/integration/`.

## PR and commit conventions

- **JIRA prefix required**: `GODRIVER-1234 Short description` for all commit messages and PR titles.
- Features → `master`. Bug fixes → latest stable release branch (e.g., `release/2.5`). The "Merge up" GitHub Action propagates fixes to newer branches automatically.
- Run `task` (default target) before every PR.
- Changed public API (`mongo/`, `bson/`, `event/`, `tag/`)? Run `task api-report` and include the output in the PR description.
- User-visible breaking change? Update `docs/migration-2.0.md`.
- Spec compliance changes? Quote the relevant spec requirement in the PR description.
