---
status: draft
spec: [001-cluster-build-arm]
created: "2026-11-08T23:35:00Z"
branch: dark-factory/cluster-build-arm
---

# Document the two build arms

<summary>
- The README now documents both ways to build the image: on this machine and inside the cluster
- It gives the exact cluster invocation, including the required full commit SHA and caller-supplied tag
- It states that the publish and cleanup goals remain local-only
- It records that the cluster arm pushes Docker Hub and that nuke mirrors the result to the private registry the deployment pulls from
- The changelog carries an unreleased entry for the change
</summary>

<objective>
Update `README.md` § *Build & publish image* so it describes the two build arms that now exist, and add a `## Unreleased` entry to `CHANGELOG.md`. This is the in-repo home for the topology after the spec is archived.
</objective>

<context>
Read `README.md` — the current section is `## Build & publish image`, which only documents `make buca`.

Read `CHANGELOG.md` — it has released headings only (`## v0.3.12` is the newest) and no `## Unreleased` section.

Read `specs/in-progress/001-cluster-build-arm.md` — Desired Behavior 7 and Acceptance Criteria 7 define exactly what this section must say and how it is checked.

Read `docs/dod.md`.

Relevant coding guides:
- `/home/node/.claude/plugins/marketplaces/coding/docs/documentation-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md`
</context>

<requirements>
**Step 1 — rewrite the README build section.** Replace the body of the `## Build & publish image` section in `README.md` (keep the heading text exactly) with text equivalent to the following. The exact cluster invocation line must appear exactly once in the section.

````markdown
## Build & publish image

The image has two build arms, selected by `BUILD_TARGET`.

Local (the default) — builds with the local Docker daemon and loads the image here:

```bash
make build    # builds docker.io/bborbe/alert-controller:<tag> locally
make buca     # builds, pushes and cleans up locally
```

`make buca`, `make upload` and `make clean` are local-only: a cluster build
pushes from inside the cluster and never loads the image on this machine.

Cluster — builds the same image as a Kubernetes Job in the `build` namespace and
pushes it to Docker Hub:

```bash
make build BUILD_TARGET=sun REF=<sha> VERSION=<tag>
```

`REF` must be the full 40-character commit SHA and must exist on `origin`;
`VERSION` must be supplied by the caller, because the cluster arm refuses to fall
back to the newest release tag. The cluster arm pushes
`docker.io/bborbe/alert-controller:<tag>`; `bborbe/nuke` then mirrors that Docker
Hub image to the private registry the deployment pulls from.
````

The string `BUILD_TARGET=sun REF=<sha> VERSION=<tag>` must appear exactly once in this section — do not repeat it in prose or in a second example. The word `local` (lowercase) must appear at least once in this section.

**Step 2 — add the changelog entry.** Add a `## Unreleased` section to `CHANGELOG.md` directly above the existing `## v0.3.12` heading, with one bullet describing the change, matching the bullet style already used under released headings. Use wording equivalent to:

```markdown
## Unreleased

- feat: build the image as a Kubernetes Job with `make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`; the local arm and its guards are unchanged, and the cluster arm pushes Docker Hub for nuke to mirror.
```

There must be exactly one `## Unreleased` heading in the file.

**Step 3 — leave the rest alone.** Do not change any other README section (run locally, HTTP admin endpoints, license) or any other changelog entry.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git.
- Do not touch `Makefile`, `hack/`, or any Go file in this prompt.
- Keep the `## Build & publish image` heading text unchanged — the acceptance check greps it verbatim.
- The section must state that `buca` remains local and that the cluster arm publishes to Docker Hub while nuke mirrors it to the private registry.
- The cluster invocation shown must be exactly `make build BUILD_TARGET=sun REF=<sha> VERSION=<tag>`.
- `CHANGELOG.md` must end up with exactly one `## Unreleased` heading and one bullet under it.
- Existing tests must still pass.
</constraints>

<verification>
Run `make precommit` — must pass.

Then confirm the documentation matches the acceptance criteria:

```bash
# the exact cluster invocation appears exactly once inside the build section
test "$(sed -n '/## Build & publish image/,/^## /p' README.md | grep -c 'BUILD_TARGET=sun REF=<sha> VERSION=<tag>')" = "1"

# the section mentions the local arm
test "$(sed -n '/## Build & publish image/,/^## /p' README.md | grep -c 'local')" -ge 1

# the section mentions the mirror relationship
sed -n '/## Build & publish image/,/^## /p' README.md | grep -qi 'nuke'
sed -n '/## Build & publish image/,/^## /p' README.md | grep -qi 'docker hub'

# exactly one unreleased heading
test "$(grep -c '^## Unreleased$' CHANGELOG.md)" = "1"
```
</verification>
