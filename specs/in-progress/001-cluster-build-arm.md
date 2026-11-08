---
status: verifying
approved: "2026-11-08T22:22:57Z"
generating: "2026-11-08T22:29:10Z"
prompted: "2026-11-08T22:38:45Z"
verifying: "2026-11-08T22:57:08Z"
branch: dark-factory/cluster-build-arm
---

## Summary

- This repo's image can currently be produced on one machine only: the Mac.
- Give it a second build path: `make build BUILD_TARGET=sun …` runs the build as a Kubernetes Job in the `build` namespace and pushes the result to Docker Hub.
- The local arm keeps its command, its `--no-cache` recipe and its `$(VERSION)` tag exactly as they are.
- Both arms compose their image reference from the same two variables, so they cannot drift apart.
- The version guard that refuses an untagged build stays on the local arm, where it protects a human's build, and is scoped away from the cluster arm, which builds a named `REF` rather than `HEAD`.

## Problem

`alert-controller` runs on nukedev, nukeprod and quant, and its image is pulled from the private registry `docker.prod.nuke.benjamin-borbe.de:443/bborbe/alert-controller`, which nuke fills by mirroring `docker.io/bborbe/alert-controller`. Both ends of that chain start with a build on this Mac: `make build` runs `docker build` against the local daemon, and nothing else can produce the image. Every rebuild, every release cut while the operator is away from this machine, and every unattended pipeline needs a human at the keyboard for this one repo.

The sibling repos already carry the second arm, and one of them is the reference implementation. Copying it here is not mechanical, because four properties of this repo differ:

- **The build file is `Makefile`, not `Makefile.docker`.** The reference arm lives in a separate included file. Here the split goes into the single `Makefile`, beside the `build` target it replaces.
- **`check-version-tag` is a prerequisite of `build`.** It compares `git describe --tags --exact-match HEAD` against `$(VERSION)` and exits non-zero on a mismatch, so on any untagged tree — including the one a cluster build runs from — a naive paste of the cluster branch never reaches the runner.
- **`VERSION` has a repo-wide default.** `VERSION := $(shell git describe --tags …)` resolves to the newest tag when the caller passes nothing, so a cluster build that forgot `VERSION` would silently publish under a tag the built `REF` does not correspond to.
- **`check-go-mod` runs `go mod vendor`** in the local tree. It is meaningful only for a local build; on the cluster arm the build happens in a clone and the vendoring would only dirty the operator's checkout.

## Goal

`make build` on this repo produces the image locally exactly as it does today. `make build BUILD_TARGET=sun REF=<full sha> VERSION=<scratch-tag>` produces the same image inside the sun cluster as a Kubernetes Job and pushes `docker.io/bborbe/alert-controller:<scratch-tag>`, the Docker Hub repository nuke mirrors from. Both arms work from the same checkout, the local arm is unchanged in behaviour, and the cluster arm works from a fresh clone with no local preparation.

## Non-goals

- Changing the local arm's command, its `--no-cache` recipe or its tag. All three stay as they are.
- Moving the local arm to `docker buildx` with timestamp rewriting, as `go-skeleton`'s local arm does for reproducibility. This repo's local arm keeps `docker build`; the cluster arm derives its build clock from the built commit, which is where reproducibility matters here.
- Pushing the private-registry copy from the cluster. The `buildkit-push` credential is a Docker Hub credential; the private copy stays nuke's mirror step.
- Bumping the mirror pin in `bborbe/nuke` to the tag this work releases. That is a deploy-side edit in another repository.
- Refreshing the local clone's commit drift.
- Adding build knobs beyond `BUILD_TARGET` and `VERSION`. The repository URL and the push-secret name are literals in the dispatch, not configurable inputs.
- Carrying `ALLOW_DEFAULT_VERSION` across from the copy source. It is an opt-out on exactly the caller-supplied-tag refusal this spec ships, and the reference spec deliberately dropped it for the same reason.

## Acceptance Criteria

- [ ] **The local arm's command and tag are unchanged, and its version guard still fires** — container-executable half: with no `BUILD_TARGET` set, `make -n build` prints the `docker build` recipe and no reference to `hack/build-cluster.sh`, and `make -n build` on an untagged tree prints `refusing to build`. Operator-executable half: `make build ALLOW_UNTAGGED_BUILD=1` exits 0 and `docker image inspect docker.io/bborbe/alert-controller:v0.3.12` exits 0; bare `make build` **fails non-zero** with stderr containing `refusing to build`. Evidence: the two `make -n` outputs, the two real exit codes, and the inspected image id. ⚠️ `build: check-version-tag check-go-mod` (`Makefile:105`) makes a bare build fail by design here, and GNU make returns 2 for a failed prerequisite, so the criterion is "non-zero", never "1". ⚠️ The tag in the `docker image inspect` probe is a literal, not `$(VERSION)` — make syntax is not shell syntax and would substitute empty.
- [ ] **Post-Deploy (Rung-2):** the cluster arm builds and pushes from a fresh clone — `make build BUILD_TARGET=sun REF=<full 40-character sha> VERSION=<scratch-tag>` exits 0, `crane digest docker.io/bborbe/alert-controller:<scratch-tag>` resolves to a digest, and `docker image inspect` on the same reference fails on this Mac with `No such image` — a stopped daemon or a malformed reference does not satisfy this. Evidence: the exit code, the Job name, the build pod's `spec.nodeName`, the resolved digest, and the inspect failure text.
  - `deploy_check:` `kubectlsun -n build get secret buildkit-push -o jsonpath='{.metadata.name}'`
  - `deploy_target:` `buildkit-push`
- [ ] **The cluster arm pushes to the repository the deployment consumes** — `make -n build BUILD_TARGET=sun REF=<sha> VERSION=probe` prints `IMAGE='docker.io/bborbe/alert-controller'` as the value handed to the runner, and the digest above resolves at `docker.io/bborbe/alert-controller:<scratch-tag>`. Evidence: the printed `IMAGE=` line, and the digest row naming that exact repository.
- [ ] **The cluster arm refuses a tag it did not receive from the caller** — `make -n build BUILD_TARGET=sun REF=<sha>` with no `VERSION` on the command line exits non-zero and prints a message naming `VERSION`. Evidence: the exit code and the message. The refusal is at parse time, so no recipe runs and no Job can be created — the exit code and message are the whole observable, and a Job probe here would be theatre.
- [ ] **`hack/` holds the five files, the cluster branch invokes the runner, and the local guards are scoped off it** — at `HEAD`, `git ls-tree --name-only HEAD hack/` lists `build-cluster.sh`, `cluster-alias.sh`, `build-job.yaml`, `build-cluster-bootstrap.sh` and `dockerconfig.json.tpl`; `grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile` returns ≥ 1 on an invocation line rather than a comment; and `make -n build BUILD_TARGET=sun REF=<sha> VERSION=probe` prints neither `go mod vendor` nor `refusing to build`. Evidence: the five names, the invocation count, and the two absent strings.
- [ ] **`upload`, `clean` and `buca` stay local, refused at parse time** — `make -n buca` prints `docker push` and `docker rmi`; `make -n buca BUILD_TARGET=sun REF=<sha> VERSION=probe` exits non-zero and prints a message naming those targets as local-only, while `make -n build BUILD_TARGET=sun REF=<sha> VERSION=probe` still succeeds. Evidence: both outputs and exit codes, and the third command's success — a refusal keyed on `BUILD_TARGET` alone would kill the cluster build too, so the third command is the discriminator.
- [ ] **The documented build path matches reality** — `README.md` § *Build & publish image* shows the exact cluster invocation `make build BUILD_TARGET=sun REF=<sha> VERSION=<tag>`, states that `buca` remains local, and records that the cluster arm pushes Docker Hub while nuke mirrors it to the private registry; `CHANGELOG.md` carries a `## Unreleased` section with a bullet for this change. Evidence: `sed -n '/## Build & publish image/,/^## /p' README.md | grep -c 'BUILD_TARGET=sun REF=<sha> VERSION=<tag>'` = 1 and `… | grep -c 'local'` ≥ 1 (scoped to the section — a repo-wide `grep -c buca` already passes on today's README and proves nothing), and `grep -c '^## Unreleased$' CHANGELOG.md` = 1.

## Verification

### Container-executable (runs inside the YOLO container at prompt time)

- `make precommit` — lint, format, generate, test and license clean
- `make test` — unit and integration suites pass
- `bash -n hack/build-cluster.sh && bash -n hack/cluster-alias.sh && bash -n hack/build-cluster-bootstrap.sh` — the copied scripts parse
- `grep -c '^BUILD_TARGET' Makefile` returns ≥ 1
- `grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile` returns ≥ 1
- `git ls-tree --name-only HEAD hack/` lists the five expected files
- `make -n build` prints a `docker build` recipe and no `hack/build-cluster.sh`
- `make -n build` on an untagged tree prints `refusing to build`
- `make -n build BUILD_TARGET=sun REF=<sha> VERSION=probe` prints `hack/build-cluster.sh` and `IMAGE='docker.io/bborbe/alert-controller'`, and prints neither `go mod vendor` nor `refusing to build`
- `make -n build BUILD_TARGET=sun REF=<sha>` (no `VERSION`) exits non-zero naming `VERSION`
- `make -n buca BUILD_TARGET=sun REF=<sha> VERSION=probe` exits non-zero naming the local-only targets, while `make -n build BUILD_TARGET=sun REF=<sha> VERSION=probe` exits 0
- `grep -c '^## Unreleased$' CHANGELOG.md` = 1

### Operator-executable (runs on the host after PR merge, spec verification ladder)

- `make build BUILD_TARGET=sun REF=<full sha> VERSION=<scratch-tag>` — one real cluster build, Job name, node and digest recorded
- `crane digest docker.io/bborbe/alert-controller:<scratch-tag>` — the push landed at the repository nuke mirrors from
- `docker image inspect docker.io/bborbe/alert-controller:<scratch-tag>` — fails with `No such image`, proving the build did not run here
- `make build ALLOW_UNTAGGED_BUILD=1` then `docker image inspect docker.io/bborbe/alert-controller:v0.3.12` — the local arm still works
- `make build` — still refused with `refusing to build`, proving the guard survived

## Desired Behavior

1. **A build target selects the arm.** `BUILD_TARGET` defaults to `local`. With `local`, `make build` behaves exactly as it does today: it runs `check-version-tag` and `check-go-mod`, then `docker build --no-cache`, tagging `$(DOCKER_REGISTRY)/$(IMAGE):$(VERSION)`.
2. **Any other value routes to a cluster Job.** With `BUILD_TARGET` set to a cluster alias or kubectl context, `make build` prints the composed image repository, then renders and applies the Job through `hack/build-cluster.sh`, passing the repository URL, the Dockerfile name and the push-secret name as literals. It waits for the Job and reports success or failure with the Job's own log lines. The build runs from a fresh clone: the Dockerfile already builds with `-mod=mod`, so no `vendor/` directory is needed. Of the five `hack/` files, three are invoked by the build (`build-cluster.sh`, `cluster-alias.sh`, `build-job.yaml`); the other two (`build-cluster-bootstrap.sh`, `dockerconfig.json.tpl`) provision the `build` namespace and its `buildkit-push` secret on a cluster that does not yet carry them, and are shipped so that a new cluster needs no second change.
3. **The cluster arm needs a caller-supplied tag.** The cluster arm refuses to run unless `VERSION` came from the command line or the environment. It refuses in particular when `VERSION` would resolve from the file's own default, so a tag nobody chose can never be pushed. The refusal names the variable and happens before any Job is created.
4. **Both arms publish to one repository.** The cluster arm uses `$(DOCKER_REGISTRY)/$(IMAGE):$(VERSION)` — the same expression the local arm tags, so the two cannot drift. With the file's defaults that is `docker.io/bborbe/alert-controller`.
5. **The version and vendor guards cover the local arm only.** `check-version-tag` and `check-go-mod` are scoped to `BUILD_TARGET=local` at parse time: a cluster build neither compares `HEAD`'s tag (it builds `REF`) nor vendors into the operator's working tree. A bare local build still fails on an untagged tree.
6. **`upload`, `clean` and `buca` remain local-only.** A remote target pushes from inside the cluster and never loads the image here, so there is nothing to push or remove. Given a cluster `BUILD_TARGET` **and** one of those goals they refuse at parse time, naming themselves as local-only, rather than silently doing nothing — and `build` with the same `BUILD_TARGET` still runs.
7. **The documented build path matches reality.** The README's build section shows both arms, says `buca` stays local, and records that the cluster arm publishes to Docker Hub while nuke mirrors it onward. The CHANGELOG records the change.

## Constraints

- The local arm's command, recipe and tag must not change. `docker build --no-cache` stays; do not convert it to `docker buildx`.
- Both parse-time refusals are gated on the goal list as well as on `BUILD_TARGET`: `$(filter upload clean buca,$(MAKECMDGOALS))` for the local-only refusal and `$(filter build,$(MAKECMDGOALS))` for the `VERSION` refusal. An ungated top-level `$(error …)` fires for every goal, so `make build BUILD_TARGET=sun …` would abort before reaching the cluster recipe — killing the arm it ships.
- `check-version-tag` and `check-go-mod` are scoped by an `ifeq ($(BUILD_TARGET),local)` at parse time, not by a recipe-level `exit 0`. A recipe-level skip would still run and still print, and the AC could not distinguish the two arms under `-n`.
- The cluster branch also passes `ALLOW_UNTAGGED_BUILD=1` into the runner's environment. With the guard scoped to `local` this is inert, and it is kept deliberately as a belt for a future invocation path that reaches the guard; the scoping is the mechanism, not this variable.
- The `VERSION` refusal keys on `$(origin VERSION)` being `file` or `undefined`. `VERSION` has a repo-wide default here, so a guard testing only for `undefined` never fires and the caller gets the runner's generic `TAG is required` instead.
- `buca` is defined twice in `Makefile` today (lines 100-101 and 165-166). Remove the duplicate; keep one definition.
- The five `hack/` files are copied unmodified from `origin/master` of their named sources: `build-cluster.sh`, `cluster-alias.sh` and `build-job.yaml` from `bborbe/agent-task-executor` (its Job template carries `automountServiceAccountToken: false`, the two `/meta` lines and the `DOCKER_REGISTRY=docker.io` build arg, which the older `go-skeleton` template lacks); `build-cluster-bootstrap.sh` and `dockerconfig.json.tpl` from `bborbe/go-skeleton`. Both repositories are public, so the agent can clone them; do not invent a variant.
- The Job template passes `DOCKER_REGISTRY=docker.io` as a build argument, so the Dockerfile's base images resolve from the public registry inside the cluster. Do not point it at the private mirror.
- `IMAGE`, `DOCKER_REGISTRY`, `REF`, `VERSION` and the other substituted values must each satisfy the runner's character whitelist; no value may carry a newline. The runner enforces this — do not add a value path that bypasses it.
- `.gitignore` needs no change: it ignores only `/specs/log` and `/prompts/log`, so the spec file itself is committable without an ignore edit.
- `README.md` § *Build & publish image* is the in-repo home for the two-arm topology; the spec dies after implementation, the README does not.
- No new scenario. The behaviour is reachable by checks on the rendered invocation plus one real cluster build.

## Assumptions

These must hold before the cluster ACs can pass; none is created by this spec.

- The `build` namespace and the `buildkit-push` dockerconfigjson secret exist on sun, and that credential authenticates against **Docker Hub**. Measured on the pilot 2026-11-08: a private-mirror push returned `401 Unauthorized` after a successful build, while a sibling pushed `docker.io/bborbe/git-rest:v0.40.5` on the same cluster. The private-registry copy the deployment pulls comes from nuke's mirror step, not from this build.
- The repository is public, so the Job's anonymous `git fetch` succeeds; the init container carries no clone credential.
- `crane` is installed on the host running the ACs.
- `REF` exists in the local repository **and on `origin`**: the runner derives `SOURCE_DATE_EPOCH` from the local repository, and the Job's init container fetches the ref from the remote. A commit that exists only locally fails at the clone.
- The build pod can reach the Go module proxy.
- `kubectl` is configured for sun through the alias `hack/cluster-alias.sh` resolves, and sun is up.
- The nuke mirror pin in `bborbe/nuke`'s `alert-controller/Makefile` still names an older tag (`v0.3.0` while this repo is at `v0.3.12`); deploying the newly built tag is a separate deploy-side edit.

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---|---|---|
| `BUILD_TARGET` names an unknown cluster or context | The runner fails before applying anything and lists the accepted values | Re-run with a listed target; no Job exists to clean up |
| `VERSION` omitted on the cluster arm | Refusal at parse time, naming `VERSION`, before any Job is created | Re-run passing `VERSION=<scratch-tag>` |
| `REF` is a short SHA, or is not in the local repository | The runner dies resolving `SOURCE_DATE_EPOCH` | `git fetch`, then re-run with the full 40-character SHA |
| `REF` exists locally but was never pushed | The clone init container fails: the ref does not exist on `origin` | Push the branch or commit, then re-run |
| The clone's commit epoch differs from the locally derived one | The build container refuses before `buildctl` runs and names both epochs | `git fetch`, then re-run |
| A Job with the derived name already exists in `build` | The runner refuses and names the existing Job rather than overwriting it | Delete the finished Job, then re-run |
| The runner is interrupted after `apply` | The Job keeps running in the cluster while the local invocation exits; the next run then hits the existing-Job refusal | Inspect `kubectlsun -n build get jobs`, read the pod's log, delete the Job once finished |
| The build pod cannot reach the module proxy | The Job fails; the runner prints the last 100 log lines and the Job name | Inspect the Job logs; the local arm is unaffected |
| The build pod is OOM-killed or throttled | The Job reports `failed`; the pod status carries `OOMKilled` or a CPU-throttling event | Raise the template's limits in the same change, or build locally for that one image |
| The registry rejects the push | The build fails at the push step with the registry's own error | Re-check the `build` namespace secret, then re-run |
| A caller-supplied `VERSION` names an already-released tag | No guard — the push replaces that tag irreversibly | Re-push the image from the released tag, then cut a new release |
| The local arm regresses | `make build ALLOW_UNTAGGED_BUILD=1` fails or tags differently | Revert the arm; the local path is the regression gate |

## Security / Abuse Cases

- The runner substitutes caller-supplied values into a manifest and into embedded shell, so every value is checked against a character whitelist and rejected for embedded newlines before rendering. Do not add a value path that bypasses that check.
- The repository URL and the push-secret name are literals in the dispatch, not configurable inputs. A configurable clone source on a build that mounts a push credential would let a caller point the credential-bearing Job at another repository.
- The build Job mounts a push credential from a `build`-namespace secret and runs with `automountServiceAccountToken: false`. The credential is scoped to pushing an image; it grants nothing else.
- The build container runs as an unprivileged user with sandbox profiles relaxed because rootless BuildKit requires it. That relaxation is deliberate and must not be copied to the clone init container.
- `VERSION` is caller-controlled and becomes an image tag and a Job name. The tag is whitelisted and the Job name is sanitized to a valid DNS name, truncated, and stripped of trailing separators.

## Suggested Decomposition

Prompts should be generated in this order — each row is a single prompt with a clear scope.

| # | Prompt focus | Covers DBs | Covers ACs | Depends on |
|---|---|---|---|---|
| 1 | `Makefile` dispatch: the `BUILD_TARGET` split, the cluster branch and its goal-gated `VERSION` refusal, the `check-version-tag` / `check-go-mod` scoping, the goal-gated local-only `upload` / `clean` / `buca` refusal, and removing the duplicate `buca` | 1, 2, 3, 4, 5, 6 | 1, 3, 4, 6 | — |
| 2 | The build path itself: the five `hack/` files copied from their named sources | 2 | 2, 5 | prompt 1 (the dispatch calls the runner) |
| 3 | Docs: the README build section and the CHANGELOG bullet | 7 | 7 | — |

Rationale: prompt 1 establishes the dispatch and the guards that make any of it reachable; prompt 2 supplies the runner the dispatch calls; prompt 3 is independent and touches no build logic.

## Do-Nothing Option

Leaving this repo without a cluster arm keeps one image tied to the Mac. Every rebuild, every release cut while the operator is away from this machine, and every future unattended pipeline needs a human at the keyboard for this one repo — while its siblings build themselves. The cost is not the one build; it is that the automation has a hole shaped exactly like the repo nobody is watching.
