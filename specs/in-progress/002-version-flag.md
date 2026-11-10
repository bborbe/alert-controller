---
status: approved
approved: "2026-11-10T22:41:11Z"
branch: dark-factory/version-flag
---

## Summary

- The binary carries its build version, commit and date, but the only way to read them is the metrics endpoint.
- A `-version` flag that prints them and exits makes the deployed artefact identifiable from a shell, a pod, or a log line.
- It must work with no other flag set — today the binary refuses to start without its required flags, so the check has to run before argument validation.
- The service path is unchanged: same flags, same validation, same behaviour when `-version` is absent.

## Problem

`alert-controller` knows exactly what it was built from. `main.go` declares `BuildGitVersion`, `BuildGitCommit` and `BuildDate`, and `Run` hands all three to `libmetrics.NewBuildInfoMetrics().SetBuildInfo(...)`. That is the only reader. Recovering them from a running pod means scraping the metrics endpoint and joining three series, and from a built image that is not running anywhere it means inspecting the image's environment by hand.

The obvious fix — a `-version` flag on the application struct — does not work here. `main` calls `service.Main`, which calls `argument.Parse` on the struct before the application runs, and `SentryDSN`, `Listen` and `PrometheusURL` are all `required:"true"`. A binary invoked as `alert-controller -version` therefore exits with a required-field error and prints nothing, which is precisely the wrong answer for a flag whose whole purpose is to be usable on a machine where the service cannot start.

## Goal

`alert-controller -version` prints the build version, commit and date on one line and exits 0, with no other flag, environment variable or configuration present. Every existing invocation behaves exactly as it does today.

## Non-goals

- Changing the metrics output, the `SetBuildInfo` call, or the metric names and labels.
- Adding a `version` subcommand, a `-json` output mode, a `-short` mode, or a `--version` (double-dash) spelling.
- Injecting the version at link time via `-ldflags`. The Dockerfile passes only `-ldflags "-s"`; the values keep arriving through `ARG` → `ENV`, exactly as they do now.
- Touching the Kubernetes manifests, the Dockerfile, or the build arms.
- Refactoring `main.go`, the application struct, or the `service.Main` call.

## Acceptance Criteria

- [ ] **`-version` works with no other flag set.** `/tmp/alert-controller -version` exits 0 and prints the build version, commit and date. Evidence: the exit code and the printed line. The discriminator is that the same command today exits **2** with `flag provided but not defined: -version` — so a passing check proves the flag is recognised *and* the version path exits before required-field validation, not that a flag was merely registered. (An argument-less run is the one that exits 4 with the `sentry-dsn` required-field error; that is AC4's baseline, not this one.)
- [ ] **The printed version is the one the binary was built with.** With `BUILD_GIT_VERSION=v9.9.9 BUILD_GIT_COMMIT=abc1234 BUILD_DATE=2026-01-02T03:04:05Z /tmp/alert-controller -version`, the output contains all three values. Evidence: the output line and the three values quoted in it.
- [ ] **The defaults are the ones the build declares, not literals in the new path.** With none of the three variables set, `-version` prints `dev`, `none` and `unknown` and exits 0. Note the provenance differs per field: `BuildGitVersion` and `BuildGitCommit` carry `default:"dev"` / `default:"none"` as struct tags, while `BuildDate` has **no** struct default — `unknown` comes from `Dockerfile`'s `ARG BUILD_DATE=unknown`. Discriminator: change the `default:"dev"` tag to `default:"zzz"`, rebuild, and re-run with no environment — the output must print `zzz`; a hardcoded `"dev"` in the new path fails that and passes the plain check, so both are required. **Revert the tag edit before the next AC.** Evidence: the two output lines, the rebuild, and `git diff` clean afterwards.
- [ ] **The service path is unchanged.** With `-version` absent, the binary still refuses to start when a required flag is missing, naming that flag, and still starts normally when all required flags are supplied. Evidence: the two exit codes and the error text naming `sentry-dsn`.
- [ ] **The flag is documented, and the repo-wide flag search is clean.** `grep -rn -- '-version' . --include='*.md' --include='*.go' --include='Makefile*'` shows the flag named in `README.md` and nowhere it contradicts the implementation (no stale `--version` spelling, no doc claiming a different output), and `CHANGELOG.md` carries a `## Unreleased` bullet. Evidence: the grep output and `grep -c '^## Unreleased$' CHANGELOG.md` = 1.

## Verification

- `go build -o /tmp/alert-controller .` — the binary compiles
- `/tmp/alert-controller -version` — exits 0, prints one line carrying `dev`, `none` and `unknown`
- `BUILD_GIT_VERSION=v9.9.9 BUILD_GIT_COMMIT=abc1234 BUILD_DATE=2026-01-02T03:04:05Z /tmp/alert-controller -version` — exits 0, prints all three
- `/tmp/alert-controller` — exits non-zero, error names `sentry-dsn`
- `make precommit` — exits 0

## Desired Behavior

1. `-version` is recognised as a flag of the binary, and supplying it prints the build version, the build commit and the build date on a single line, then exits 0.
2. The three values are read from the same sources the application struct already declares — `BUILD_GIT_VERSION`, `BUILD_GIT_COMMIT`, `BUILD_DATE` — and fall back to the same defaults the struct declares (`dev`, `none`) plus `unknown` for an unset date, matching the Dockerfile's own `ARG` defaults. No second source of truth is introduced: the defaults are not restated as literals in the new path.
3. The check runs before argument validation, so `-version` succeeds on a host or in a container where the service's required flags are absent and the service could not start.
4. When `-version` is absent, the binary parses its flags, validates them and runs the service exactly as it does today — including the same required-flag errors and the same exit codes.
5. The flag is documented in the README, so a reader can discover it without reading the source.

## Constraints

- The existing flag set, its names, its defaults and its validation are unchanged. `-version` is additive.
- **`docs/dod.md`'s no-print rule does not apply to this path.** § Code Quality forbids "print statements, `fmt.Printf`" in favour of structured logging; this path exits before logging is initialised, so it writes with `fmt.Fprintf(os.Stdout, …)` and nothing else in the binary gains a print. Amend `docs/dod.md` with that one-sentence carve-out in the same change, so the next reader does not have to rediscover why.
- The version values keep their current provenance. Do not add `-ldflags` injection, a `version` package, or a generated file.
- The new output goes to stdout and must not depend on `glog` being initialised.
- The exit code on success is 0. Do not reuse the service's non-zero codes for this path.
- Match only the exact `-version` argument. `--version` is deliberately not supported (see Non-goals) and must keep behaving as an unknown flag, not silently print.
- No new module. `github.com/bborbe/argument` is already pinned in `go.mod` but marked `// indirect`; promoting it to a direct require is expected and is not a new dependency — do not reinvent reflection to avoid it.
- `-version` must not require or read `SENTRY_DSN`, `LISTEN` or `PROMETHEUS_URL`.

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---|---|---|
| `-version` is registered on the application struct only | The binary exits with a required-field error and prints nothing — the change looks done and is useless | Move the check ahead of argument validation and assert the no-other-flag case |
| The defaults are hardcoded in the new path instead of read from the struct tags | The printed defaults drift from the struct the moment either is edited | Read the same three values the application struct declares; assert with the `default:"zzz"` discriminator |
| An unset date renders as `<nil>` or `0001-01-01T00:00:00Z` | The output line is unusable and differs from the Dockerfile's `unknown` default | Render `unknown` explicitly when the value is unset |
| Output goes through `glog` before logging is initialised | Nothing is printed, or the line lands in a file instead of the terminal | Write to stdout directly, per the Constraints carve-out |
| The check swallows unknown flags | A typo elsewhere in the flag set stops being reported | Match only the exact `-version` argument; leave all other parsing untouched |
| `BUILD_DATE` is set but unparseable (e.g. `2026-01-02` with no time) | The flag still exits 0 and renders the raw value; a parse failure must never make the version path fail | Render the raw string when parsing fails, and assert the exit code stays 0 |
| The service path regresses | A required flag stops being enforced | Assert the missing-flag error and its exit code |

## Security / Abuse Cases

- The only input this path reads is the process argument list. It matches the **exact** string `-version` and nothing else, so no argument content is interpreted, expanded or passed to a shell.
- An unknown or misspelled flag — including the deliberate `--version` non-goal — must keep producing the existing `flag provided but not defined` error. A prefix or substring match would silently swallow typos for the rest of the flag set, so exact equality is the requirement, not a convenience.
- The path prints only the three build values, which are already public through the metrics endpoint. It reads no secret and opens no file or socket.

## Assumptions

- The build values reach the binary through `ARG` → `ENV` in the Dockerfile (`Dockerfile:11-20`), not through `-ldflags` — the build line passes only `-ldflags "-s"`. So a pod always carries non-empty values (`dev` / `none` / `unknown` unless overridden), and the unset case is a local-build concern.
- `main()` in the repo root is the sole entrypoint; `cmd/alert-yaml-to-k8s` is a separate binary and out of scope.
- The defaults live in the `application` struct's field tags, which is why AC3's discriminator edits a tag rather than a constant.

## Suggested Decomposition

Prompts should be generated in this order — each row is a single prompt with a clear scope.

| # | Prompt focus | Covers DBs | Covers ACs | Depends on |
|---|---|---|---|---|
| 1 | The `-version` path in `main.go`: pre-parse before `service.Main`, read the three values from the struct's declared sources and defaults, render `unknown` for an unset date, exit 0 — plus the one-sentence `docs/dod.md` carve-out | 1, 2, 3, 4 | 1, 2, 3, 4 | — |
| 2 | Docs: the `README.md` flag mention and the `CHANGELOG.md` bullet | 5 | 5 | prompt 1 (the docs describe what it ships) |

Rationale: prompt 1 is the whole behaviour and must land first; prompt 2 is prose that would describe a flag that does not exist if it ran first.

## Do-Nothing Option

The version stays reachable only through the metrics endpoint of a running instance. Identifying a built image or a stopped pod stays a manual inspection, and the first question in any "which build is this?" investigation keeps costing a scrape and a join instead of one command.
