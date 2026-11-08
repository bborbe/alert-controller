---
status: completed
spec: [001-cluster-build-arm]
summary: Added the five hack/ build-runner files copied byte-identically from bborbe/agent-task-executor and bborbe/go-skeleton with correct modes, extended the CHANGELOG Unreleased section, and verified via make precommit.
execution_id: 308a7a48-3966-5b3f-9ee7-9d0296bad013-exec-003-hack-build-runner
dark-factory-version: dev
created: "2026-11-08T23:35:00Z"
queued: "2026-11-08T22:53:13Z"
started: "2026-11-08T22:55:59Z"
completed: "2026-11-08T22:57:08Z"
branch: dark-factory/cluster-build-arm
---

# Ship the cluster build runner under `hack/`

<summary>
- The repository now carries the scripts and Job template the cluster build arm needs
- Three files render and apply the build Job and resolve a cluster alias to a kubectl context
- Two files provision the build namespace and its push credential on a cluster that does not yet have them
- All five are copied verbatim from the sibling repositories that already ship this arm, so the behaviour matches the fleet instead of a hand-written variant
- The cluster build arm can now reach an existing script instead of a missing file
</summary>

<objective>
Add the five `hack/` files the cluster build arm depends on, copied unmodified from their named upstream repositories, with executable bits preserved. This makes the invocation added to `Makefile` in the previous prompt resolvable.
</objective>

<context>
Read `Makefile` and confirm the cluster branch of the `build` target invokes `hack/build-cluster.sh` (added by the previous prompt).

Read `specs/in-progress/001-cluster-build-arm.md` — see Desired Behavior 2, the Constraints section (the paragraph naming the five files and their sources), and the Security / Abuse Cases section.

Read `docs/dod.md`.

The five files and their sources:
- `build-cluster.sh`, `cluster-alias.sh`, `build-job.yaml` — from `https://github.com/bborbe/agent-task-executor` (`hack/` on `master`). Its Job template carries `automountServiceAccountToken: false`, the two `/meta` lines and the `DOCKER_REGISTRY=docker.io` build arg; do not substitute the older `go-skeleton` template.
- `build-cluster-bootstrap.sh`, `dockerconfig.json.tpl` — from `https://github.com/bborbe/go-skeleton` (`hack/` on `master`).

Both repositories are public. The files must be byte-identical to those sources; do not edit, reformat or "improve" them. In particular, the comments in `build-cluster.sh` mention `Makefile.docker` — that is the upstream text and stays.

Relevant coding guide: `/home/node/.claude/plugins/marketplaces/coding/docs/go-makefile-commands.md`.
</context>

<requirements>
1. Create the directory `hack/` at the repository root.

2. Fetch the sources into a temporary directory (do not leave the clones in the repo):

   ```bash
   rm -rf /tmp/ate-src /tmp/gosk-src
   git clone --depth 1 --branch master https://github.com/bborbe/agent-task-executor.git /tmp/ate-src
   git clone --depth 1 --branch master https://github.com/bborbe/go-skeleton.git /tmp/gosk-src
   ```

   If a clone fails because the network is unavailable, fetch the same `master` refs with `curl` into the same paths:

   ```bash
   mkdir -p /tmp/ate-src/hack /tmp/gosk-src/hack
   curl -fsSL https://raw.githubusercontent.com/bborbe/agent-task-executor/master/hack/build-cluster.sh        -o /tmp/ate-src/hack/build-cluster.sh
   curl -fsSL https://raw.githubusercontent.com/bborbe/agent-task-executor/master/hack/cluster-alias.sh        -o /tmp/ate-src/hack/cluster-alias.sh
   curl -fsSL https://raw.githubusercontent.com/bborbe/agent-task-executor/master/hack/build-job.yaml          -o /tmp/ate-src/hack/build-job.yaml
   curl -fsSL https://raw.githubusercontent.com/bborbe/go-skeleton/master/hack/build-cluster-bootstrap.sh      -o /tmp/gosk-src/hack/build-cluster-bootstrap.sh
   curl -fsSL https://raw.githubusercontent.com/bborbe/go-skeleton/master/hack/dockerconfig.json.tpl           -o /tmp/gosk-src/hack/dockerconfig.json.tpl
   ```

3. Copy the five files, unmodified, into `hack/`, preserving the upstream mode:
   - from `/tmp/ate-src/hack/`: `build-cluster.sh`, `cluster-alias.sh`, `build-job.yaml`
   - from `/tmp/gosk-src/hack/`: `build-cluster-bootstrap.sh`, `dockerconfig.json.tpl`

   Preserve executable bits: `build-cluster.sh`, `cluster-alias.sh` and `build-cluster-bootstrap.sh` are mode `755`; `build-job.yaml` and `dockerconfig.json.tpl` are mode `644`. After copying, run `chmod 755 hack/build-cluster.sh hack/cluster-alias.sh hack/build-cluster-bootstrap.sh` and `chmod 644 hack/build-job.yaml hack/dockerconfig.json.tpl`.

4. Do not modify the copied contents in any way. The runner's character whitelist, newline rejection, Job-name sanitisation, `SOURCE_DATE_EPOCH` derivation and push-credential mount are load-bearing security controls from upstream (see the spec's Security / Abuse Cases). Do not relax or restate them.

5. Do not execute any of the copied scripts and do not run `make build BUILD_TARGET=...` — that requires a live cluster and belongs to the operator verification ladder, not this prompt.

6. Leave `Makefile`, `README.md`, `CHANGELOG.md` and all Go files untouched in this prompt.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- The five files are copied unmodified from `origin/master` of their named sources: `build-cluster.sh`, `cluster-alias.sh` and `build-job.yaml` from `bborbe/agent-task-executor`; `build-cluster-bootstrap.sh` and `dockerconfig.json.tpl` from `bborbe/go-skeleton`. Do not invent a variant.
- The Job template passes `DOCKER_REGISTRY=docker.io` as a build argument; do not point it at the private mirror.
- The build container's relaxed sandbox profiles are deliberate for rootless BuildKit and must not be copied to the clone init container. Do not edit the security contexts.
- Do not run `go mod vendor`, do not run the scripts, do not create any Kubernetes object.
- Do not add any other file to `hack/` beyond the five named above.
- No new scenario. The behaviour is reachable by checks on the rendered invocation plus one real cluster build, which is operator-executable.
- Existing tests must still pass.
</constraints>

<verification>
Run `make precommit` — must pass.

Then confirm the five files are present, parse, are executable where required, and match upstream:

```bash
# exactly the five expected files (the committed-tree form of this check, run after
# dark-factory commits, is `git ls-tree --name-only HEAD hack/`; the files are
# uncommitted at prompt time, so list the working tree here)
ls -1 hack/ | sort
# expected: build-cluster-bootstrap.sh, build-cluster.sh, build-job.yaml, cluster-alias.sh, dockerconfig.json.tpl

# shell scripts parse
bash -n hack/build-cluster.sh
bash -n hack/cluster-alias.sh
bash -n hack/build-cluster-bootstrap.sh

# executable bits
test -x hack/build-cluster.sh
test -x hack/cluster-alias.sh
test -x hack/build-cluster-bootstrap.sh

# unmodified versus the fetched sources (works with either the clone or the curl fallback)
diff /tmp/ate-src/hack/build-cluster.sh           hack/build-cluster.sh
diff /tmp/ate-src/hack/cluster-alias.sh           hack/cluster-alias.sh
diff /tmp/ate-src/hack/build-job.yaml             hack/build-job.yaml
diff /tmp/gosk-src/hack/build-cluster-bootstrap.sh hack/build-cluster-bootstrap.sh
diff /tmp/gosk-src/hack/dockerconfig.json.tpl     hack/dockerconfig.json.tpl

# the cluster branch from the previous prompt invokes the runner
grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile    # >= 1
```

Remove `/tmp/ate-src` and `/tmp/gosk-src` before finishing.
</verification>
