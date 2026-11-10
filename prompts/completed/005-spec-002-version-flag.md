---
status: completed
spec: [002-version-flag]
summary: Added an exact `-version` path in main.go that prints build version/commit/date from the application struct's env tags and defaults (unknown for unset date) and exits 0 before required-flag validation, with unit tests, docs/dod.md carve-out, README/CHANGELOG updates, argument/v2 promoted to a direct dependency, and an anchored depguard rule so v2 imports are allowed.
execution_id: 38c65f80-d1cd-5a2b-b7fa-920e00562852-exec-005-spec-002-version-flag
dark-factory-version: dev
created: "2026-11-10T22:48:40Z"
queued: "2026-11-10T23:10:46Z"
started: "2026-11-10T23:10:48Z"
completed: "2026-11-10T23:18:23Z"
branch: dark-factory/version-flag
---

# Add the `-version` path

<summary>
- Running the binary with `-version` prints the build version, commit and date on one line and exits successfully.
- The check runs before the service's required flags are validated, so it works on a machine where the service could not start.
- The values come from the same environment variables and struct-tag defaults the application already declares, so there is no second copy of the defaults.
- An unset date prints `unknown`; a malformed date prints its raw value instead of failing.
- `--version` and every other unknown flag keep producing the existing flag error.
- The normal service path is untouched: the same required flags are enforced and the same exit codes are returned.
- The definition-of-done print rule gains a one-sentence carve-out for this path, because it writes before logging is initialised.
- Unit tests cover exact flag matching, the environment path, the defaults path and the malformed-date path.
</summary>

<objective>
Make `alert-controller -version` print the build version, commit and date and exit 0, without requiring `SENTRY_DSN`, `LISTEN` or `PROMETHEUS_URL`, while leaving the existing service startup path byte-for-byte equivalent when the flag is absent.
</objective>

<context>
Read `main.go` — the sole entrypoint. `main` builds an `application`, then calls `service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy)`. `service.Main` calls `argument.ParseAndPrint` on the application struct, and `SentryDSN`, `Listen` and `PrometheusURL` are `required:"true"`. That is why registering `-version` on the struct alone would fail with a required-field error. The three build fields are:

```go
BuildGitVersion string            `required:"false" arg:"build-git-version" env:"BUILD_GIT_VERSION" usage:"..." default:"dev"`
BuildGitCommit  string            `required:"false" arg:"build-git-commit"  env:"BUILD_GIT_COMMIT"  usage:"..." default:"none"`
BuildDate       *libtime.DateTime `required:"false" arg:"build-date"        env:"BUILD_DATE"        usage:"..."`
```

Read `docs/dod.md` — § Code Quality forbids "print statements, `fmt.Printf`".

Read `specs/in-progress/002-version-flag.md` — Desired Behavior 1-4, Acceptance Criteria 1-4 and the Failure Modes table.

Relevant coding guides:
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-cli-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-build-args-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-glog-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md`

Verified library API (do not paraphrase; these are the real exported functions):
- `github.com/bborbe/argument/v2` exposes `func DefaultValues(ctx context.Context, data interface{}) (map[string]interface{}, error)`. It reflects over `default` struct tags and returns a map keyed by Go field name.
- `github.com/bborbe/errors` exposes `func Wrap(ctx context.Context, err error, message string) error`.
- `github.com/bborbe/argument/v2` is already pinned in `go.mod` as `github.com/bborbe/argument/v2 v2.13.10 // indirect`.
</context>

<requirements>
**Step 1 — add `version.go` (new file, `package main`).** Create the file with exactly this content:

```go
// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"

	"github.com/bborbe/argument/v2"
	"github.com/bborbe/errors"
)

const versionFlag = "-version"

// hasVersionFlag reports whether args contains the exact -version argument.
func hasVersionFlag(args []string) bool {
	return slices.Contains(args, versionFlag)
}

// printVersion writes the build version, commit and date to w. The values come
// from the environment variables and struct-tag defaults declared on
// application; an unset date renders as "unknown".
func (a *application) printVersion(ctx context.Context, w io.Writer) error {
	defaults, err := argument.DefaultValues(ctx, a)
	if err != nil {
		return errors.Wrap(ctx, err, "read argument defaults failed")
	}
	typ := reflect.TypeOf(a).Elem()
	version := buildValue(typ, defaults, "BuildGitVersion")
	commit := buildValue(typ, defaults, "BuildGitCommit")
	date := buildValue(typ, defaults, "BuildDate")
	if date == "" {
		date = "unknown"
	}
	_, _ = fmt.Fprintf(w, "version=%s commit=%s date=%s\n", version, commit, date)
	return nil
}

// buildValue returns the environment value for field when set, otherwise the
// field's struct-tag default.
func buildValue(typ reflect.Type, defaults map[string]interface{}, field string) string {
	f, ok := typ.FieldByName(field)
	if !ok {
		return ""
	}
	if value := os.Getenv(f.Tag.Get("env")); value != "" {
		return value
	}
	if value, ok := defaults[field].(string); ok {
		return value
	}
	return ""
}
```

Rationale for the pieces (keep them, do not substitute literals):
- `argument.DefaultValues` reads `default:"dev"` / `default:"none"` from the `application` struct tags. Do **not** hardcode `"dev"` or `"none"` in `version.go`.
- The environment variable names are read from the struct's `env` tags via `reflect.Type.FieldByName(...).Tag.Get("env")`. Do **not** hardcode `"BUILD_GIT_VERSION"`, `"BUILD_GIT_COMMIT"` or `"BUILD_DATE"`.
- `BuildDate` has no `default` tag, so an unset date is the empty string; render `"unknown"` explicitly, matching the Dockerfile's `ARG BUILD_DATE=unknown`.
- The date is rendered as the raw environment value and is never parsed. This is what makes the unparseable-date failure mode exit 0 by construction.
- `slices.Contains` is exact string equality, so only `-version` matches. `--version`, `-version=true` and every other spelling are not matched.

**Step 2 — wire the check into `main.go` before `service.Main`.** Replace the current `main` function:

```go
func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}
```

with:

```go
func main() {
	app := &application{}
	if hasVersionFlag(os.Args[1:]) {
		if err := app.printVersion(context.Background(), os.Stdout); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "print version failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}
```

Add `"fmt"` to the `main.go` import block. Do not change anything else in `main.go`: not the `application` struct, not its tags, not the `Run` method, not any factory call, not the `service.Main` arguments.

**Step 3 — promote the argument dependency.** `version.go` imports `github.com/bborbe/argument/v2` directly, so it must leave the `// indirect` block in `go.mod`. Running `go mod tidy` (which `make precommit` runs through `ensure`) moves it into the first `require` block as `github.com/bborbe/argument/v2 v2.13.10`. Confirm the resulting `go.mod` has it in the direct `require` block with no `// indirect` comment. Do not add any other module.

**Step 4 — amend `docs/dod.md`.** Under § Code Quality, after the "No debug output (print statements, `fmt.Printf`) — use structured logging" bullet, add exactly one sentence:

```markdown
- **Carve-out:** the `-version` path in `main.go` writes the build version, commit and date to stdout with `fmt.Fprintf` before logging is initialised; this is the only permitted direct print in the binary.
```

Do not remove or reword the existing bullet.

**Step 5 — add unit tests.** Create two files.

`main_suite_test.go`:

```go
// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMainSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}
```

`version_test.go`:

```go
// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"os"
	"reflect"

	"github.com/bborbe/argument/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("version flag", func() {
	envKeys := []string{"BUILD_GIT_VERSION", "BUILD_GIT_COMMIT", "BUILD_DATE"}

	BeforeEach(func() {
		for _, key := range envKeys {
			Expect(os.Unsetenv(key)).To(Succeed())
		}
	})

	AfterEach(func() {
		for _, key := range envKeys {
			Expect(os.Unsetenv(key)).To(Succeed())
		}
	})

	Describe("hasVersionFlag", func() {
		It("matches only the exact argument", func() {
			Expect(hasVersionFlag([]string{"-version"})).To(BeTrue())
			Expect(hasVersionFlag([]string{"-listen", "x", "-version"})).To(BeTrue())
			Expect(hasVersionFlag([]string{"--version"})).To(BeFalse())
			Expect(hasVersionFlag([]string{"-version=true"})).To(BeFalse())
			Expect(hasVersionFlag(nil)).To(BeFalse())
		})
	})

	Describe("printVersion", func() {
		It("prints the struct-tag defaults when no environment is set", func() {
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(buf.String()).To(Equal("version=dev commit=none date=unknown\n"))
		})

		It("prints the values from the environment", func() {
			Expect(os.Setenv("BUILD_GIT_VERSION", "v9.9.9")).To(Succeed())
			Expect(os.Setenv("BUILD_GIT_COMMIT", "abc1234")).To(Succeed())
			Expect(os.Setenv("BUILD_DATE", "2026-01-02T03:04:05Z")).To(Succeed())
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(buf.String()).To(Equal("version=v9.9.9 commit=abc1234 date=2026-01-02T03:04:05Z\n"))
		})

		It("renders an unparseable date raw and does not fail", func() {
			Expect(os.Setenv("BUILD_DATE", "not-a-date")).To(Succeed())
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(buf.String()).To(Equal("version=dev commit=none date=not-a-date\n"))
		})
	})

	Describe("buildValue", func() {
		It("reads the default from the struct tag rather than a literal", func() {
			type tagged struct {
				BuildGitVersion string `env:"TEST_BUILD_GIT_VERSION" default:"zzz"`
			}
			Expect(os.Unsetenv("TEST_BUILD_GIT_VERSION")).To(Succeed())
			value := &tagged{}
			defaults, err := argument.DefaultValues(context.Background(), value)
			Expect(err).NotTo(HaveOccurred())
			Expect(buildValue(reflect.TypeOf(value).Elem(), defaults, "BuildGitVersion")).
				To(Equal("zzz"))
		})
	})
})
```

<!-- OPEN QUESTION (for the human reviewer, not an implementation instruction): the spec's Failure Modes table uses `BUILD_DATE=2026-01-02` as its example of an unparseable date. `libtime.ParseTime` (github.com/bborbe/time v1.27.26) accepts `time.DateOnly`, so `2026-01-02` is parseable and would render as `2026-01-02T00:00:00Z`. The test therefore uses `not-a-date` to exercise the raw-render path. The implementation never parses the date, so this distinction does not change behaviour. -->

</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- The existing flag set, its names, its defaults and its validation are unchanged. `-version` is additive.
- Do not register `version` on the `application` struct. The check must run before `service.Main` / `argument.ParseAndPrint`, otherwise the required-field error wins.
- The new path must not read or require `SENTRY_DSN`, `LISTEN` or `PROMETHEUS_URL`.
- Do not add a `version` subcommand, a `-json` mode, a `-short` mode, or a `--version` spelling.
- Do not add `-ldflags` injection, a `version` package, or a generated file. The values keep their current `ARG` → `ENV` provenance.
- Write only to stdout for the version line, with `fmt.Fprintf`. Do not use `glog` on this path; logging is not initialised.
- Exit 0 on success. Do not reuse the service's non-zero exit codes.
- Match only the exact string `-version`. `--version` must continue to reach the normal flag parser and produce `flag provided but not defined`.
- No new module. Promoting `github.com/bborbe/argument/v2` from indirect to direct is expected.
- Do not change the metrics output, the `SetBuildInfo` call, the Kubernetes manifests, the Dockerfile, or the build arms.
- Do not refactor `main.go`, the `application` struct, or the `service.Main` call beyond the `main` wiring in Step 2.
- `cmd/alert-yaml-to-k8s` is a separate binary and is out of scope; do not add `-version` there.
- Existing tests must still pass.
</constraints>

<verification>
Run `make precommit` — must pass.

Then build and exercise the binary:

```bash
go build -o /tmp/alert-controller .

# AC1/AC3: no other flag, no build environment -> dev / none / unknown, exit 0.
# The execution environment may already export BUILD_* (the repo's own CI does),
# so unset them explicitly; inheriting them would make this check non-deterministic.
env -u BUILD_GIT_VERSION -u BUILD_GIT_COMMIT -u BUILD_DATE \
  /tmp/alert-controller -version

# AC2: the built values are the ones the binary was given
BUILD_GIT_VERSION=v9.9.9 BUILD_GIT_COMMIT=abc1234 BUILD_DATE=2026-01-02T03:04:05Z \
  /tmp/alert-controller -version

# Failure mode: a malformed date still exits 0 and renders raw
BUILD_DATE=not-a-date /tmp/alert-controller -version

# AC4: the service path still enforces required flags. SENTRY_DSN may be exported
# by the execution environment; unset all three required inputs so the first
# missing field is deterministic and names sentry-dsn.
if env -u SENTRY_DSN -u LISTEN -u PROMETHEUS_URL /tmp/alert-controller \
  >/tmp/alert-controller-noargs.out 2>&1; then
  echo "expected a non-zero exit without required flags"; exit 1
fi
grep -q 'sentry-dsn' /tmp/alert-controller-noargs.out

# Security: --version remains an unknown flag, not a version print
if /tmp/alert-controller --version >/tmp/alert-controller-dashdash.out 2>&1; then
  echo "expected --version to fail as an unknown flag"; exit 1
fi
grep -q 'flag provided but not defined' /tmp/alert-controller-dashdash.out

# The print carve-out is recorded, and argument is a direct dependency
grep -q 'Carve-out' docs/dod.md
grep -q 'github.com/bborbe/argument/v2 v2.13.10' go.mod
if grep -q 'github.com/bborbe/argument/v2 v2.13.10 // indirect' go.mod; then
  echo "argument must be a direct require after this change"; exit 1
fi
```

All four version invocations must exit 0 and the two negative invocations must exit non-zero with the expected text. The `make precommit` run must also leave `go.mod` with `github.com/bborbe/argument/v2` in the direct `require` block.
</verification>
