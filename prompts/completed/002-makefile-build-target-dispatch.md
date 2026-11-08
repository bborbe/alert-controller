---
status: completed
spec: [001-cluster-build-arm]
summary: Split the Makefile build target into a local arm and a cluster arm selected by BUILD_TARGET, scoped the local-only guards and goals, removed the duplicate buca, and added a CHANGELOG Unreleased entry.
execution_id: 308a7a48-3966-5b3f-9ee7-9d0296bad013-exec-002-makefile-build-target-dispatch
dark-factory-version: dev
created: "2026-11-08T23:35:00Z"
queued: "2026-11-08T22:47:46Z"
started: "2026-11-08T22:47:47Z"
completed: "2026-11-08T22:55:58Z"
branch: dark-factory/cluster-build-arm
---

# Select the build arm with `BUILD_TARGET`

<summary>
- `make build` can now run either on this machine or inside a Kubernetes cluster, selected by one variable
- The default is unchanged: with no target set, the build runs locally with the same command, the same `--no-cache` recipe and the same tag as today
- A cluster target builds the image as a Job inside the cluster and pushes it to the Docker Hub repository the deployment mirrors from
- The cluster path refuses to run unless the caller supplied a tag, so it can never silently publish over a released image
- The existing version and vendoring guards still protect local builds, and are switched off for cluster builds at parse time
- The publish and cleanup goals now refuse loudly when pointed at a cluster instead of silently doing nothing
- The duplicate definition of the publish goal is removed
- No new build knobs are added: the repository URL, the Dockerfile name and the push-secret name are literals in the dispatch
</summary>

<objective>
Split the `build` target into a local arm and a cluster arm selected by `BUILD_TARGET`, keeping the local arm behaviourally identical. The cluster arm renders and applies the build Job through `hack/build-cluster.sh`, refuses a tag the caller did not supply, and scopes the local-only guards and goals away from cluster builds at parse time.
</objective>

<context>
Read `Makefile` in full before editing. The current `build` target has prerequisites `check-version-tag check-go-mod`; the version guard is `check-version-tag`; the vendoring guard is `check-go-mod`; and `buca` is defined twice (once near the top, once near the bottom). The `VERSION` default block is at the top of the file.

Read `specs/in-progress/001-cluster-build-arm.md` — the Desired Behavior, Constraints and Failure Modes sections are the source of truth for this prompt.

Read `docs/dod.md` for the project's definition of done.

The sibling reference implementation of the same dispatch lives in `bborbe/agent-task-executor`'s `Makefile.docker`; this repo keeps everything in a single `Makefile` and does not add the reference's `BUILD_*` knobs (see constraints). Do not copy the reference's local arm — this repo keeps `docker build --no-cache`, not `docker buildx`.

Relevant coding guide: `/home/node/.claude/plugins/marketplaces/coding/docs/go-makefile-commands.md`.

GNU Make 4.3 is installed. `.dark-factory.yaml` sets `worktree: false`; `.git` is not masked.
</context>

<requirements>
IMPORTANT: every recipe line in the snippets below begins with a literal TAB character (shown as a tab at the start of the line), not spaces. Copy the tabs. A Makefile recipe line indented with spaces fails with `missing separator`.

**Step 1 — add the selector.** Immediately after the existing block

```make
ifeq ($(VERSION),)
	VERSION := $(shell git describe --tags `git rev-list --tags --max-count=1`)
endif
```

and before `include tools.env`, insert (top level, no leading tab):

```make
# BUILD_TARGET selects where `make build` runs. `local` (the default) uses the
# local Docker daemon; any other value names a cluster alias (sun / nukedev /
# nukeprod) or kubectl context and builds inside that cluster as a Job rendered
# from hack/build-job.yaml.
BUILD_TARGET ?= local
```

**Step 2 — add the two goal-gated parse-time refusals.** Immediately after `BUILD_TARGET ?= local`, still before `include tools.env`, insert this block exactly (top level, no leading tabs):

```make
ifneq ($(BUILD_TARGET),local)

# `upload`, `clean` and `buca` push or remove an image that only exists in the
# local Docker daemon. A cluster build pushes from inside the cluster and never
# loads the image here, so those goals have nothing to act on.
ifneq ($(filter upload clean buca,$(MAKECMDGOALS)),)
$(error ERROR: upload, clean and buca are local-only targets; BUILD_TARGET=$(BUILD_TARGET) builds and pushes inside the cluster. Run them without BUILD_TARGET, or use `make build BUILD_TARGET=$(BUILD_TARGET) REF=<sha> VERSION=<tag>`)
endif

# VERSION has a repo-wide default (the newest tag). On the cluster arm that
# default would silently publish over a released image, so require the caller
# to name the tag. `$(origin VERSION)` is `file` when the default assignment
# above ran and `undefined` when it resolved to nothing; both mean the caller
# did not supply a tag.
ifneq ($(filter build,$(MAKECMDGOALS)),)
ifneq ($(filter $(origin VERSION),file undefined),)
$(error ERROR: the cluster arm needs an explicit VERSION. VERSION was not supplied on the command line or in the environment; its file default is $(VERSION). Pass VERSION=<scratch-tag>)
endif
endif

endif
```

Both refusals must stay gated on the goal list (`$(filter upload clean buca,$(MAKECMDGOALS))` and `$(filter build,$(MAKECMDGOALS))`). An ungated `$(error)` fires for every goal and would abort `make build BUILD_TARGET=sun ...` before the cluster recipe is reached.

**Step 3 — rewrite the `build` rule.** Keep the prerequisite list `build: check-version-tag check-go-mod` unchanged. Replace the recipe with (recipe lines start with a TAB):

```make
.PHONY: build
build: check-version-tag check-go-mod
ifeq ($(BUILD_TARGET),local)
	DOCKER_BUILDKIT=1 \
	docker build \
	--no-cache \
	--rm=true \
	--platform=linux/amd64 \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg BRANCH=$(BRANCH) \
	--build-arg BUILD_GIT_VERSION=$$(git describe --tags --always --dirty) \
	--build-arg BUILD_GIT_COMMIT=$$(git rev-parse --short HEAD) \
	--build-arg BUILD_DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
	-t $(DOCKER_REGISTRY)/$(IMAGE):$(VERSION) \
	-f Dockerfile .
else
# The cluster arm publishes at $(VERSION), the same expression the local arm
# tags, so the two cannot drift. The repository URL and push-secret name are
# literals: a configurable clone source on a Job that mounts a push credential
# would let a caller point the credential-bearing Job at another repository.
	BUILD_TARGET='$(BUILD_TARGET)' \
	NAMESPACE='build' \
	REPO_URL='https://github.com/bborbe/alert-controller.git' \
	REF='$(REF)' \
	TAG='$(VERSION)' \
	DOCKERFILE='Dockerfile' \
	IMAGE='$(DOCKER_REGISTRY)/$(IMAGE)' \
	DOCKER_REGISTRY='$(DOCKER_REGISTRY)' \
	PUSH='true' \
	PUSH_SECRET='buildkit-push' \
	TIMEOUT='3600' \
	ALLOW_UNTAGGED_BUILD=1 \
	hack/build-cluster.sh
endif
```

The local branch's command, flags and `-t $(DOCKER_REGISTRY)/$(IMAGE):$(VERSION)` tag must match today's exactly — do not add `--pull`, do not change `--no-cache`, do not switch to `docker buildx`.

**Step 4 — scope `check-version-tag` to the local arm.** Wrap its recipe body in a parse-time conditional; keep the body's text unchanged (recipe lines start with a TAB):

```make
.PHONY: check-version-tag
check-version-tag:
ifeq ($(BUILD_TARGET),local)
	@if [ -n "$(ALLOW_UNTAGGED_BUILD)" ]; then \
		echo "ALLOW_UNTAGGED_BUILD set — skipping version/tag check"; \
	else \
		head_tag=$$(git describe --tags --exact-match HEAD 2>/dev/null); \
		if [ "$$head_tag" != "$(VERSION)" ]; then \
			echo "ERROR: refusing to build $(VERSION) from this tree." >&2; \
			echo "  HEAD is at tag: $${head_tag:-<untagged>}" >&2; \
			echo "  building as:    $(VERSION)" >&2; \
			echo "  An image stamped vX.Y.Z must be built from the vX.Y.Z tag." >&2; \
			echo "  Fix: git checkout $(VERSION)   (or set ALLOW_UNTAGGED_BUILD=1 for a scratch build)" >&2; \
			exit 1; \
		fi; \
	fi
endif
```

**Step 5 — scope `check-go-mod` to the local arm.** Wrap its recipe body in the same parse-time conditional; keep the body's text unchanged (recipe lines start with a TAB):

```make
.PHONY: check-go-mod
check-go-mod:
ifeq ($(BUILD_TARGET),local)
	@if [ -f "go.mod" ]; then \
		echo "go.mod found, running go mod vendor..."; \
		go mod vendor; \
	else \
		echo "go.mod not found, skipping go mod vendor."; \
	fi
endif
```

**Step 6 — remove the duplicate `buca`.** `buca` is defined twice today (once near the top before `build`, once near the bottom after `apply`). Keep exactly one definition — `buca: build upload clean apply` — and delete the second. Leave `upload`, `clean` and `apply` recipes unchanged; the parse-time refusal in step 2 is what stops them on the cluster arm.

**Step 7 — leave everything else alone.** Do not change `DOCKER_REGISTRY`, `IMAGE`, `BRANCH`, `DIRS`, `tools.env`, `precommit`, `ensure`, `format`, `generate`, `test`, `check`, `addlicense`, `apply`, `fix`, or any other target.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- The local arm's command, recipe and tag must not change. `docker build --no-cache` stays; do not convert it to `docker buildx`.
- Add no build knobs beyond `BUILD_TARGET`. Do not add `BUILD_REPO_URL`, `BUILD_PUSH_SECRET`, `BUILD_TIMEOUT`, `BUILD_NAMESPACE`, `BUILD_DOCKERFILE`, or any other `?=` variable. The repository URL (`https://github.com/bborbe/alert-controller.git`), the Dockerfile name (`Dockerfile`), the push-secret name (`buildkit-push`) and the timeout (`3600`) are literals in the dispatch.
- Do NOT add `ALLOW_DEFAULT_VERSION` or any opt-out on the caller-supplied-tag refusal. The refusal is the feature.
- Both parse-time refusals are gated on the goal list as well as on `BUILD_TARGET`: `$(filter upload clean buca,$(MAKECMDGOALS))` for the local-only refusal and `$(filter build,$(MAKECMDGOALS))` for the `VERSION` refusal.
- The `VERSION` refusal keys on `$(origin VERSION)` being `file` or `undefined`. `VERSION` has a repo-wide default, so a guard testing only for `undefined` never fires.
- `check-version-tag` and `check-go-mod` are scoped by `ifeq ($(BUILD_TARGET),local)` at parse time, not by a recipe-level `exit 0`.
- The cluster branch passes `ALLOW_UNTAGGED_BUILD=1` into the runner's environment. It is inert while the guard is scoped to `local`; keep it as a belt.
- `IMAGE`, `DOCKER_REGISTRY`, `REF`, `VERSION` and the other substituted values must each satisfy the runner's character whitelist and must not carry a newline. Do not add a value path that bypasses the runner's check.
- Do NOT modify `hack/` in this prompt — the runner files are added by the next prompt. `make -n` only prints, so the missing script does not affect this prompt's verification.
- Error handling, formatting and generated files are untouched; `make precommit` must still pass.
- Existing tests must still pass.
</constraints>

<verification>
Run `make precommit` — must pass.

Then confirm the dispatch and the guards. Use a full 40-character SHA for `REF` (it is only printed under `-n`):

```bash
SHA=0123456789012345678901234567890123456789

# selector and dispatch present, duplicate buca gone
grep -c '^BUILD_TARGET' Makefile                                   # >= 1
grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile            # >= 1
grep -c '^buca:' Makefile                                          # = 1

# local arm unchanged and cluster arm absent under -n
make -n build 2>&1 | grep -q 'docker build'
! make -n build 2>&1 | grep -q 'hack/build-cluster.sh'
make -n build 2>&1 | grep -q 'refusing to build'                   # recipe text contains the guard

# cluster arm renders the runner with the composed repository
make -n build BUILD_TARGET=sun REF=$SHA VERSION=probe 2>&1 | grep -q "IMAGE='docker.io/bborbe/alert-controller'"
make -n build BUILD_TARGET=sun REF=$SHA VERSION=probe 2>&1 | grep -q 'hack/build-cluster.sh'
! make -n build BUILD_TARGET=sun REF=$SHA VERSION=probe 2>&1 | grep -q 'go mod vendor'
! make -n build BUILD_TARGET=sun REF=$SHA VERSION=probe 2>&1 | grep -q 'refusing to build'

# caller-supplied tag required on the cluster arm
! make -n build BUILD_TARGET=sun REF=$SHA >/dev/null 2>&1
make -n build BUILD_TARGET=sun REF=$SHA 2>&1 | grep -q 'VERSION'

# local-only goals refuse on the cluster arm, while build still runs
! make -n buca BUILD_TARGET=sun REF=$SHA VERSION=probe >/dev/null 2>&1
make -n buca BUILD_TARGET=sun REF=$SHA VERSION=probe 2>&1 | grep -qi 'local-only'
make -n build BUILD_TARGET=sun REF=$SHA VERSION=probe >/dev/null 2>&1   # exits 0

# local buca still shows push and remove
make -n buca 2>&1 | grep -q 'docker push'
make -n buca 2>&1 | grep -q 'docker rmi'
```
</verification>
