# Tasks: Build, Test and Release the CLI With the Tasks and the CI of the Repo (US-022)

## Contents

1. Task 1: Build, run and snapshot the CLI with `task build`, `task cli` and `task release:local`
2. Task 2: Run the end-to-end tests of the CLI with `task test:e2e` and `task test:e2e:local`
3. Task 3: Install GoReleaser and Node with `mise install`, and keep `dist/` and `e2e/node_modules/` out of git
4. Task 4: Fix the version of each module that the CLI imports in `go.mod`
5. Task 5: Add the extension point `ci.jobs`, and run the end-to-end tests of the CLI in the CI workflow
6. Task 6: Publish a release for each version tag and a prerelease for each push to `main`, after the tests pass

## Open Questions

The tasks follow the recommendation of the question. Overrule it before the task that it names starts.

1. **How does `cli` add its jobs to the CI workflow?** (Tasks 5 and 6; open question 3 of the proposal.)
   **Recommendation: with YAML items of a new extension point `ci.jobs` in `.github/workflows/ci.yml` of each
   starter, for the three jobs: `e2e`, `release` and `prerelease`.** The file writes each item under `jobs:` with
   `indent 2`, as `docker-compose.local.yml` writes `compose.services`. `release` runs for a tag that starts with
   `v`, `prerelease` runs for a push to `main`, and both need the job `test` of the starter and the job `e2e`.
   Each has `permissions: contents: write` for its job alone. The reasons:
   - The function that sets the indent of a YAML item exists (`indent`, US-017), so the cost that the proposal
     names for this choice is paid.
   - The triggers of the workflow of the starter stay `push: {}`. Each change of `cli` to `ci.yml` is an added
     job, which is what an item can express.
   - The release workflows of go-cli call `ci.yml` as a reusable workflow. That needs `workflow_call` in the
     starter and `branches-ignore: [main]` on its push trigger, so that the tests do not run two times on `main`.
     An item can add a trigger but cannot change one, so the starter must change its own trigger when a caller
     exists. Then a repo with `cli` runs no `ci.yml` of its own on `main` or on a tag, and a called workflow gets
     no permission beyond its caller (`contents: write`). US-020 publishes the images of a service repo from its
     CI on `main` and on tags, with `packages: write`, so that design breaks US-020 in a service repo with `cli`.
     With `ci.jobs`, the jobs of US-020 and of `cli` sit side by side, each with its own permissions.
   - The other choice of the proposal, one workflow file for each feature, cannot name the job `test` of the
     starter in `needs`, so a release of `cli` would copy the test steps of the starter, or call `ci.yml`, which
     comes back to the reusable workflow.

   The cost: a repo with `cli` has one workflow, `CI`, with the release jobs in it, where go-cli has three. The
   GitHub page of the runs shows the release jobs as skipped on each other branch.

## Story Reference

US-022, "Build, test and release the CLI with the tasks and the CI of the repo", in
`user-stories/feature-compose.md:734-770` (Description, Acceptance Criteria, Context). The design is `PROPOSAL.md`:
sections 6 (extension points of a starter), 9 (templates), 10 (rules 2, 7 and 8) and 19 (open question 3). The
reference for the parts of go-cli is the template `go-cli` of `repo-templates`
(`/Users/davidnguyen/repo-templates/templates/go-cli/`, read-only): `.github/workflows/{ci,release,prerelease}.yml`,
`Taskfile.yml`, `Taskfile.release.yml`, `Taskfile.test.yml`, `.gitignore`, and the tools and modules of
`hooks/post_scaffold`. This repo's own `.github/workflows/`, `Taskfile*.yml`, `mise.toml`, `.gitignore` and
`go.mod` follow the same template, with fixed versions.

## Boundaries

**Out of scope:**

The non-goals of the stories file, as written there:

- The command `add`, and an update of a repo to a new version of the catalog. Both need a three-way merge,
  which is phase 3.
- Generators that you run many times in one repo, for example `go-cli-sub-command`. They stay in scaffold.
- More than one catalog at a time. The config file names one catalog folder.
- A starter with more than one folder, for example a Go API in `api/` and a Svelte web app in `web/`. The
  feature `cli` shows G5 in one folder.
- A check of the credential in a repo without `api-auth`. The API of such a repo accepts each request.
- A third method of authentication for Google sign-in, with a session exchange.
- The templates that the proposal has not yet decided: `asdf-plugin`, `go-event-sourcing-cli`, `go-worker`,
  `global/git-hook`, and `global/kustomize`.
- The removal of `repo-templates`.
- A sync with `repo-templates`. The tool reads any catalog, and the catalog of this repo is one of them. A
  change in `repo-templates` comes into the catalog only through a story that gives the change its own
  value.

Added by this breakdown:

- `on-demand-build.yml` of go-cli, a workflow that a person starts to build the archives of a ref. No criterion
  names it. It is a file of its own that changes no file of a starter, so a later story can add it to `cli`
  with no extension point.
- `test:e2e` in the pre-push hook. In go-cli, `hook:validate-go-changes` runs it. The hook of each starter keeps
  the tests of the starter, and no criterion names the hook.
- A task `run` of `cli`. `task run` of `go-event-sourcing-service` starts the service, so the CLI runs with
  `task cli` in each repo (decision 8).
- `workflow_call` and `branches-ignore: [main]` in the `ci.yml` of a starter, and the separate files
  `release.yml` and `prerelease.yml` (open question 1).
- Fixed versions of the modules that other features import. Criterion 4 names the modules that the CLI imports.
  `api-auth` with the method `oauth` imports `github.com/golang-jwt/jwt/v5`, which no `go.mod` of a starter
  requires; `cli-auth` imports `go.yaml.in/yaml/v3`, which `go-event-sourcing-service` already requires.
- A section of the README of a starter about the tasks of the CLI. `docs/cli.md` is the documentation of `cli`,
  and the starter `go` has no extension point for its README (US-016).
- A run of `task test:e2e` of a written repo by the tests of feature-compose. The e2e image of this repo has no
  Docker. The Verification of Task 2 runs it on the host.
- A test that publishes a real GitHub release. The tests of feature-compose check the workflow as written
  (Tasks 5 and 6); the Verification of Task 6 tells how to see a real release by hand.
- A change to the `test` command of the folders of the starters (`task test:unit`). The `combinations` job of
  this repo's CI has no Node, so the end-to-end tests of the CLI run in the e2e test of `go` with `cli`.

**Deferred:** no criterion of the story. The first criterion names `task run -- <arguments>`; the caller chose
`task cli -- <arguments>` for the CLI in each repo (decision 8).

## Codebase Context

**Extension points and items.** A starter declares an extension point in its `starter.yml` with a `file` and its
item fields (`catalog/starters/go-event-sourcing-service/starter.yml:59-82`). A feature adds items with `add_to`
(`catalog/features/process-manager/feature.yml:20-113`). The file writes the items with `items` and `range`, and a
YAML block with `indent` (`internal/catalog/selection.go:271-280`):
`catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:30-36` writes a two-field item per
line and a YAML block at indent 2, and `Taskfile.api.yml:64-67` writes a YAML block with `[[` `]]` delimiters.
**The value of an item goes into the file as it is; the tool does not render it as a template**
(`docs/catalog.md:221`). So an item cannot hold the name of the repo or the module. A task that builds
`./bin/<name>`, or that sets the ldflag `<module>/internal/cli/version.releaseTag`, must be in a file of `cli`,
which the tool renders. `check` compares the item fields of an extension point across the targets of a feature,
and finds a file of an extension point that does not call `items` for it. The items of an extension point have a
fixed order (rule 8). With no item, the file must be the same as a file with no extension point
(`docs/catalog.md:224-225`).

**The two targets of `cli` today.**

| File | `go` | `go-event-sourcing-service` |
| --- | --- | --- |
| `Taskfile.yml` | includes `test`; tasks `hook:setup`, `hook:validate-go-changes`; no extension point | includes `migrate`, `test`, `api`, `catchup` and the items of `taskfile.includes`; tasks `test:shellcheck`, `hook:*`, `db:*`, `run-dependencies`, `remove-dependencies`, `run` |
| `Taskfile.test.yml` | `unit`, `shellcheck`, `vulnerabilities` | `unit`, `integration`, `all`, `vulnerabilities` |
| `mise.toml` | Go, Task 3.42.1, gotestsum, shellcheck, govulncheck | the same and `golang-migrate` |
| `.gitignore` | has `bin` | has `bin` |
| `go.mod.tmpl` | one line `require github.com/stretchr/testify v1.12.1` | two `require` blocks, the second of indirect modules |
| `.github/workflows/ci.yml` | `on: push: {}`, `permissions: contents: read`, job `test` | `on: push: {}`, no `permissions`, jobs `test` and `build` |

Both starters have the delimiters `[[` `]]` for `go/**/Taskfile*.yml` and `go/.github/**`. `cli` has them only for
`go/.goreleaser.yaml` (`catalog/features/cli/feature.yml:13-16`); the delimiters glob matches the path in the
catalog (learning). `cli` has no `add_to` today; its only extension point is `cli.commands` in its own `root.go`.

**Where each missing part of go-cli goes.**

| Part of go-cli | Place in a repo with `cli` |
| --- | --- |
| `run` and `build` of `Taskfile.yml` | the tasks `cli` and `build` of `Taskfile.cli.yml` of `cli`, which `Taskfile.yml` includes with `flatten: true` through `taskfile.flat_includes` (Task 1) |
| `Taskfile.release.yml` (`release:local`) | the task `release:local` of `Taskfile.cli.yml` (Task 1) |
| `e2e`, `e2e:local` and `E2E_TAG` of `Taskfile.test.yml` | the tasks `test:e2e` and `test:e2e:local` of `Taskfile.cli.yml`. A task name with a colon in a flat include sits next to the namespace `test`, as `test:shellcheck` of `go-event-sourcing-service` does (Task 2) |
| `goreleaser` and `node` of `mise use` in the hook | items of `mise.tools` (Task 3) |
| `dist/` and `e2e/node_modules/` of `.gitignore` | items of `gitignore.patterns` (Task 3) |
| `go get` of `urfave/cli/v3`, `fatih/color`, `mattn/go-isatty` in the hook | items of `gomod.requires` (Task 4) |
| the job `e2e` of `ci.yml` | an item of `ci.jobs` (Task 5) |
| `release.yml` and `prerelease.yml` | the items `release` and `prerelease` of `ci.jobs` (Task 6, open question 1) |
| `workflow_call` and `branches-ignore: [main]` of `ci.yml`, `on-demand-build.yml`, `test:e2e` in the hook | nobody (Boundaries) |

**The tests that this story changes.**

- `addedByCLI` (`internal/catalog/catalog_test.go:1528-1551`) requires that each file of a target stays as
  without `cli`. From Task 1, `cli` changes the file of each extension point that it adds items to. The test of
  `process-manager` (`catalog_test.go:1071-1107`) is the form: the changed files are the files of the extension
  points of its `add_to`, taken from the definitions.
- "the docs of cli name no task or workflow that a repo of any target of cli lacks" (`catalog_test.go:1848-1859`)
  requires that the docs of `cli` hold no `task `, `Taskfile`, `.github/workflows` or `release:local`. The second
  criterion asks for the opposite: the docs of `cli` name the tasks.
- `tasksOf` (`catalog_test.go:2223-2242`) reads each include as a string. A flat include is a map, so the parse
  stops on it.
- The e2e run of `go` with `cli` (`e2e/tests/new.test.ts:111-152`) calls `goreleaser` directly; the run of
  `go-event-sourcing-service` with `cli` is `new.test.ts:443-467`; the run of `go` alone checks the tasks that its
  CI workflow calls (`new.test.ts:82-109`).
- The tree of `show` in a terminal (`e2e/tests/show.test.ts:48-66`) expects `└─ use_cases.handlers`, the last
  extension point of `go-event-sourcing-service`, on a screen of 40 lines. This story adds five extension points to
  each starter, so the tree grows by about ten lines and can push its last lines off the screen.

**Facts checked for this breakdown.**

- Task 3.42.1 (the version in the `mise.toml` of both starters) and Task 3.53.1 (the CI and the e2e image of this
  repo) list the tasks of an include with `flatten: true` under their own names; `task test:e2e` of a flat
  include and `task test:unit` of the namespace `test` both run; a task `run` in a flat include next to a `run`
  of `Taskfile.yml` stops Task.
- Docker answers on this machine (`timeout 20 docker info`, 2026-11-13). The e2e image of this repo
  (`e2e/Dockerfile:17-35`) has Go, Task, gotestsum, GoReleaser, git, Node, npm and a C compiler, and no Docker,
  mise or actionlint.
- An e2e test already runs `new` with `GOPROXY` and `GOSUMDB=off` in its environment (`new.test.ts:469-494`).
- The modules that the Go files of `cli` import, other than the standard library and the module of the repo:
  `github.com/urfave/cli/v3`, `github.com/fatih/color`, `github.com/mattn/go-isatty` and
  `github.com/stretchr/testify` (tests). The starters require testify; none requires the other three, so
  `go mod tidy` takes their latest versions today.
- US-020 is not built. It changes the `ci.yml` of `go-event-sourcing-service` to publish images on `main` and on
  tags. With `ci.jobs`, US-020 keeps the `range` of the items and the name of the job `test`, which the summary
  of `ci.jobs` gives to the items.

**Words.** A *flat include* is an include of `Taskfile.yml` with `flatten: true`, whose tasks keep the names that
their Taskfile gives them; not a root include, a merged include or an inline include. A *CI job* is a job of `.github/workflows/ci.yml`; an item of `ci.jobs` is one CI job; not a
workflow, a pipeline or a stage. A *tool* is what `mise.toml` names and `mise install` installs; not a
dependency, a plugin or a package. A *pattern* is a line of `.gitignore`; not a rule or an entry. A *required
module* is a module that a `require` of `go.mod` names with a fixed version; not a dependency, a library or a
pin. The item field `module` of `gomod.requires` is the path of a required module, the same kind of value as the
option `module` of a starter, which is the path of the module of the repo. A *release*, a *prerelease* and a
*snapshot* keep the meaning of `docs/cli.md` of `cli`: the GitHub release of a tag `v<major>.<minor>.<patch>`,
the GitHub release of a build of `main`, and the output of GoReleaser in `dist/` with no publish.

**The extension points that this story adds.** Each starter declares each one with the same item fields, so
`check` accepts the `add_to` of `cli` for both targets.

| Name | File | Item | What the item is |
| --- | --- | --- | --- |
| `taskfile.flat_includes` | `go/Taskfile.yml` | `[name, taskfile]` | A Taskfile of a feature that `Taskfile.yml` includes with `flatten: true` |
| `mise.tools` | `go/mise.toml` | `[name, version]` | A tool, with the name that `mise.toml` gives it and a fixed version |
| `gitignore.patterns` | `go/.gitignore` | `[pattern]` | A pattern of `.gitignore` for files that a command of a feature makes |
| `gomod.requires` | `go/go.mod` | `[module, version]` | A module that the Go code of a feature imports, with a fixed version |
| `ci.jobs` | `go/.github/workflows/ci.yml` | `[text]` | A CI job as YAML with the job name at the left edge. `needs` can name `test`, the job of the starter that runs its tests |

**Decisions of this breakdown, for the caller to overrule:**

1. **One Taskfile of `cli`** (Tasks 1 and 2). `Taskfile.cli.yml` holds `build`, `cli`, `release:local`,
   `test:e2e` and `test:e2e:local`, and the tool renders it with `kebab .Name` and `option "starter.module"`. A
   flat include keeps the task names of the criteria. The starters get one extension point for their Taskfiles,
   `taskfile.flat_includes`, and `Taskfile.test.yml` of each starter does not change.
2. **The tool versions** (Task 3): `aqua:goreleaser/goreleaser` 2.18.2 and `node` 26.10.0, from the
   `mise.toml` of this repo. go-cli takes the latest versions when scaffold runs its hook, so it gives no version.
3. **The module versions** (Task 4): `github.com/urfave/cli/v3` v3.14.0, `github.com/fatih/color` v1.19.0 and
   `github.com/mattn/go-isatty` v0.0.24, from the `go.mod` of this repo, whose CLI uses the same code as `cli`.
4. **The proof that a new release of a module does not change the repo** (Task 4): a second `new` of the same
   repo with `GOPROXY=off`, after a first `new` filled the module cache. `go mod tidy` then reads only the cache,
   so it stops if any import needs a lookup of a latest version.
5. **The release jobs** (Task 6) have the steps of the jobs `release` and `prerelease` of this repo's
   `.github/workflows/release.yml` and `prerelease.yml`, which are the steps of go-cli. A test compares them, so a
   port slip shows.
6. **What the tests of feature-compose run** (Tasks 2, 3, 5 and 6): the end-to-end tests of the CLI through
   `task test:e2e:local` in the e2e run of `go` with `cli`; `git status` after a snapshot and an `npm install`; the
   tasks that the written `ci.yml` calls. By hand: `task test:e2e` in Docker, `mise install`, actionlint on the
   written workflows, and a real release.
7. **The files that are ported** (for the audit): the tasks of `Taskfile.yml`, `Taskfile.release.yml` and
   `Taskfile.test.yml`, the job `e2e` of `ci.yml`, the jobs of `release.yml` and `prerelease.yml`, and the lines
   of `.gitignore` of go-cli. The audit can raise old defects of this code; decline them with the reason that the
   code is the template's (learning).
8. **The run task of the CLI is `cli` in each repo** (Task 1), chosen by the caller. `task cli -- <arguments>`
   runs the CLI from the source in a `go` repo and in a `go-event-sourcing-service` repo, so the commands are the
   same in both. `task run` of `go-event-sourcing-service` keeps starting the service, and a `go` repo has no
   `task run`. A task `run` of `cli` beside the `run` of the service starter stops Task, and an `excludes` on the
   include of one starter gives the two repos different commands.

## Tasks

### Task 1: Build, run and snapshot the CLI with `task build`, `task cli` and `task release:local`

**Behavior:** Each starter that `cli` targets declares the extension point `taskfile.flat_includes` in
`Taskfile.yml`. `cli` writes its own `Taskfile.cli.yml` with the tasks `build`, `cli` and `release:local`, and adds
one item that includes it. In each repo with `cli`, `task build` builds `bin/<name>`, `task cli -- <arguments>`
runs the CLI from the source with the arguments, and `task release:local` makes a GoReleaser snapshot in `dist/`.
In a `go-event-sourcing-service` repo, `task run` still starts the service (decision 8). A repo without `cli` has
none of these tasks, and its `Taskfile.yml`
is the same as without the extension point. Of the files of a target, `cli` changes only the file of each
extension point that it adds items to. `docs/cli.md` names the three tasks.

**Acceptance Criteria:**
- [x] The `starter.yml` of `go` and of `go-event-sourcing-service` declares `taskfile.flat_includes` with the file
  `go/Taskfile.yml`, the item fields `[name, taskfile]`, and a summary that says that the tasks of the Taskfile
  keep their own names, and that a task name of the Taskfile must not be a task name of the starter.
- [x] `catalog/features/cli/feature.yml` adds one item of `taskfile.flat_includes` for `Taskfile.cli.yml`, and
  gives `[[` `]]` delimiters to its Taskfile. Written with the name `my-tool`, `Taskfile.cli.yml` builds
  `./bin/my-tool` from `./cmd/my-tool` with `CGO_ENABLED=0`, runs `./cmd/my-tool` with `{{.CLI_ARGS}}`, and runs
  `goreleaser release --snapshot --clean`. It keeps the template syntax of Task.
- [x] `TestCatalog`, for each target of `cli` taken from its `feature.yml`: of the files of the target, `cli`
  changes only the file of each extension point that its `add_to` names, taken from the definitions and not from a
  list of the test. The other files that `cli` writes are new files, at the same paths and with the same contents
  for each two targets, `Taskfile.cli.yml` among them.
- [x] `TestCatalog`: the tasks of the written Taskfiles, read with the flat include, hold `build`, `cli` and
  `release:local` for each target with `cli`, and no task name of `Taskfile.cli.yml` is a task of the starter.
  Written without `cli`, no target has a task `build`, `cli` or `release:local`, and
  `Taskfile.yml` of each starter holds only the lines of the starter, with no empty line that the extension point
  adds.
- [x] `TestCatalog`: for each target, each `task <name>` that `docs/cli.md` and `docs/e2e-tests.md` of `cli` name
  is a task of the written repo. `docs/cli.md` names `task build`, `task cli -- <arguments>` and
  `task release:local`.
- [x] `e2e/tests/new.test.ts`, the run of `go` with `cli`: `task build` exits with 0, and `bin/my-tool` with no
  command prints the commands `version`, `update` and `help`, and no other. `task cli -- --help` prints the same
  commands. `task release:local` makes the snapshot in place of the direct call of GoReleaser, and the checks of
  the archives, the checksums and the version of the snapshot pass. `task --list-all` names each task that the
  written `docs/cli.md` names.
- [x] `e2e/tests/new.test.ts`, the run of `go` alone: `task --list-all` names no task `build`, `cli` or
  `release:local`.
- [x] `e2e/tests/new.test.ts`, the run of `go-event-sourcing-service` with `cli`: `task build` writes
  `bin/my-service`, which prints the commands of the CLI, and `task cli -- --help` prints the same commands.
  `task --dry run` prints the `tmuxp load` of the starter. The run of `go-event-sourcing-service` alone has no
  task `build`, `cli` or `release:local` in `task --list-all`.
- [x] `docs/e2e-tests.md` of this repo tells the new steps of the runs of `new`.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass.
  `task test:combinations` passes, and its output names `go --feature cli` and
  `go-event-sourcing-service --feature cli`.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass,
  and the e2e tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go/starter.yml` — the extension point `taskfile.flat_includes`
- `catalog/starters/go/files/go/Taskfile.yml` — writes the items of `taskfile.flat_includes`
- `catalog/starters/go-event-sourcing-service/starter.yml` — the extension point `taskfile.flat_includes`
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml` — writes the items of `taskfile.flat_includes`
- `catalog/features/cli/feature.yml` — the item of `taskfile.flat_includes`, and the delimiters of its Taskfile
- `catalog/features/cli/files/go/Taskfile.cli.yml` — new: `build`, `cli`, `release:local`
- `catalog/features/cli/files/go/docs/cli.md` — the sections "Build" and "Goreleaser" name the tasks
- `internal/catalog/catalog_test.go` — the changed files of `cli`, the tasks with and without `cli`, the tasks of the
  docs of `cli`, and `tasksOf` with a flat include
- `e2e/tests/new.test.ts` — the runs of `go` and of `go-event-sourcing-service`, with and without `cli`
- `e2e/tests/show.test.ts` — the tree of `show`, if its last lines leave the screen
- `docs/e2e-tests.md` — the runs of `new`

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml:3-10` and
  `catalog/starters/go-event-sourcing-service/starter.yml:67-70` for an include that an item gives
- `catalog/features/process-manager/feature.yml:91-93` for the item of an include
- `catalog/features/cli/feature.yml:13-16` for the delimiters of a file of `cli`
- `Taskfile.yml:7-17` and `Taskfile.release.yml:1-6` of this repo for the tasks of go-cli
- `internal/catalog/catalog_test.go:1071-1107` for the files that a feature changes
- `internal/catalog/catalog_test.go:1229-1240` for a file with no item
- `internal/catalog/catalog_test.go:1528-1584` for the files of `cli` for each target
- `internal/catalog/catalog_test.go:1848-1859` and `internal/catalog/catalog_test.go:2223-2242` for the docs of `cli`
  and the tasks of a written repo
- `e2e/tests/new.test.ts:82-109`, `e2e/tests/new.test.ts:111-152` and `e2e/tests/new.test.ts:443-467` for the runs
  of `new`
- `e2e/tests/show.test.ts:48-66` for the tree of `show`
- `docs/e2e-tests.md:80-110` for the docs of a run of `new`
- `tasks/learnings.md:16-21` for the delimiters glob
- `tasks/learnings.md:30-35` for the old defects of ported code
- `tasks/learnings.md:37-42` and `tasks/learnings.md:58-63` for the e2e run of a written repo
- `tasks/learnings.md:127-132` and `tasks/learnings.md:362-367` for breaks by hand
- `tasks/learnings.md:285-290` for `task test:combinations`
- `tasks/learnings.md:341-346` for the e2e tests of `show`

**Testable:** Yes

**Certainty:** medium — `taskfile.includes` (`go-event-sourcing-service/files/go/Taskfile.yml:3-10`) and its item
in `process-manager` are the precedent; but no include of the catalog has `flatten`, and this is the first
extension point of `go`

**Blast radius:** low — none of the named categories; it adds tasks to the Taskfiles of a repo with `cli`, and the
files of a repo without `cli` stay the same

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write each target with `--feature cli` with the built binary, and run `task --list-all`, `task build`,
`task cli -- version` and `task release:local` there, with Task 3.42.1 of the repo's `mise.toml` and with Task
3.53.1. Write each starter alone with the binary of this branch and with a binary built from `main`, and diff the
two folders: no file differs. Apply one break at a time to the catalog, read its diff, run the test that it
names, and put the file back with `git checkout --`: remove `flatten: true` from the Taskfile of `go` (the e2e run
of `go` with `cli`); rename the task `cli` of `Taskfile.cli.yml` to `run` (`TestCatalog`, and the e2e run of
`go-event-sourcing-service` with `cli`); build into `./bin/other` (the e2e run of `go` with `cli`).
Each break makes a test fail. Run `task test:combinations`, and the e2e tests of `new`, `show` and `check` on the
host.

**Depends on:** None

---

### Task 2: Run the end-to-end tests of the CLI with `task test:e2e` and `task test:e2e:local`

**Behavior:** `Taskfile.cli.yml` also has `test:e2e`, which builds `e2e/Dockerfile` with the tag `v0.1.0` and runs
the image, and `test:e2e:local`, which builds the CLI with the tag `v0.1.0` in its ldflag, installs the packages of
`e2e/` with npm, and runs vitest with `EXECUTABLE` and `BUILD_TAG`. Both run in each target, next to `test:unit`.
The docs of `cli` name both tasks in place of the raw commands.

**Acceptance Criteria:**
- [x] `Taskfile.cli.yml` has `test:e2e` and `test:e2e:local` with the commands of the e2e section of
  `docs/cli.md` of `cli`, and one tag, `v0.1.0`, for both. Written with `module=github.com/acme/ledger`, the
  ldflag of `test:e2e:local` names `github.com/acme/ledger/internal/cli/version`, and no file of the repo holds
  `github.com/hpcsc/my-tool`.
- [x] `TestCatalog`: the tasks of each target with `cli` hold `test:e2e` and `test:e2e:local` beside `test:unit`.
  Without `cli`, no target has `test:e2e` or `test:e2e:local`.
- [x] `e2e/tests/new.test.ts`, the run of `go` with `cli`: `task test:e2e:local` exits with 0, and its output names
  each test file under `e2e/tests` of the written repo as passed, taken from the folder and not from a list of the
  test.
- [x] `e2e/tests/new.test.ts`, the run of `go` alone: `task --list-all` names no task `test:e2e` or
  `test:e2e:local`.
- [x] The section "E2E Test" of `docs/cli.md` of `cli` names `task test:e2e`, which runs the tests in Docker, and
  `task test:e2e:local`, which runs them on the machine with Node. The table "Where the tests run" of
  `docs/e2e-tests.md` of `cli` names the same two tasks. The test of the tasks of the docs of `cli` (Task 1)
  passes for each target.
- [x] `docs/e2e-tests.md` of this repo tells the new step of the run of `go` with `cli`.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `check` pass on the
  host.

**Affected Files/Modules:**
- `catalog/features/cli/files/go/Taskfile.cli.yml` — `test:e2e` and `test:e2e:local`
- `catalog/features/cli/files/go/docs/cli.md` — the section "E2E Test"
- `catalog/features/cli/files/go/docs/e2e-tests.md` — the table "Where the tests run"
- `internal/catalog/catalog_test.go` — the e2e tasks with and without `cli`, and the module in the ldflag
- `e2e/tests/new.test.ts` — `task test:e2e:local` in the run of `go` with `cli`, and the run of `go` alone
- `docs/e2e-tests.md` — the run of `go` with `cli`

**Patterns to Follow:**
- `task:1` for `Taskfile.cli.yml` and the tests of its tasks
- `Taskfile.test.yml:3-4` and `Taskfile.test.yml:21-33` of this repo for the e2e tasks of go-cli
- `catalog/features/cli/files/go/docs/cli.md:73-90` and `catalog/features/cli/files/go/docs/e2e-tests.md:78-87` for the
  commands that the tasks replace
- `catalog/features/cli/files/go/e2e/Dockerfile` for the image that `test:e2e` builds
- `internal/catalog/catalog_test.go:1607-1619` for the module in the files of `cli`
- `e2e/tests/new.test.ts:111-152` and `e2e/tests/new.test.ts:930-933` for the run of `go` with `cli` and the folders
  that hold tests
- `e2e/Dockerfile:17-35` for the tools of the e2e image
- `docs/e2e-tests.md:80-110` for the docs of a run of `new`
- `tasks/learnings.md:51-56` for the output of a test run that names what passed
- `tasks/learnings.md:58-63` for the e2e run on the host
- `tasks/learnings.md:127-132` and `tasks/learnings.md:362-367` for breaks by hand

**Testable:** Yes

**Certainty:** medium — this repo's `Taskfile.test.yml:21-33` has the same two tasks, and Task 1 sets the Taskfile
of `cli`; but no test of this repo runs the vitest suite of a written repo, and its `npm install` of node-pty in the
e2e image is new

**Blast radius:** low — none of the named categories; it adds two test tasks

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write `go` with `--feature cli` and
`go-event-sourcing-service` with `--feature cli` with the built binary, and run `task test:e2e` (Docker) and
`task test:e2e:local` in each; both exit with 0. Apply one break at a time to `Taskfile.cli.yml`, read its diff,
run the e2e run of `go` with `cli`, and put the file back with `git checkout --`: set the tag of the build of
`test:e2e:local` to `v0.2.0` and leave `BUILD_TAG` at `v0.1.0` (the version test of the CLI fails); remove
`dir: e2e`. Each break makes the test fail. Run `task test:combinations`, and the e2e tests of `new` and `check`
on the host.

**Depends on:** Task 1

---

### Task 3: Install GoReleaser and Node with `mise install`, and keep `dist/` and `e2e/node_modules/` out of git

**Behavior:** Each starter that `cli` targets declares `mise.tools` in `mise.toml` and `gitignore.patterns` in
`.gitignore`. `cli` adds GoReleaser and Node with fixed versions to `mise.tools`, and `dist/` and
`e2e/node_modules/` to `gitignore.patterns`. So `mise install` in a repo with `cli` installs GoReleaser and Node,
and after `task release:local` and an `npm install` in `e2e/`, `git status` shows no file under `dist/` or
`e2e/node_modules/`. A repo without `cli` has the `mise.toml` and the `.gitignore` of its starter.

**Acceptance Criteria:**
- [ ] The `starter.yml` of each target of `cli` declares `mise.tools` with the file `go/mise.toml` and the item
  fields `[name, version]`, and `gitignore.patterns` with the file `go/.gitignore` and the item field `[pattern]`.
- [ ] `TestCatalog`: for each target with `cli`, `mise.toml` parses as TOML and has the tools of the starter at
  their versions, `aqua:goreleaser/goreleaser` at `2.18.2` and `node` at `26.10.0`. `.gitignore` has the lines
  `dist/` and `e2e/node_modules/`. Without `cli`, `mise.toml` has neither tool, `.gitignore` has neither line, and
  both files hold only the lines of the starter.
- [ ] `e2e/tests/new.test.ts`, the run of `go` with `cli`: after `task release:local` and
  `npm install --no-audit --no-fund` in `e2e/`, `git init` and `git status --porcelain --untracked-files=all` in the
  repo list `go.mod`, and no path under `dist/` or `e2e/node_modules/`.
- [ ] `docs/e2e-tests.md` of this repo tells the step of `git status`.
- [ ] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new`, `show` and `check` pass
  on the host.

**Affected Files/Modules:**
- `catalog/starters/go/starter.yml` and `catalog/starters/go-event-sourcing-service/starter.yml` — `mise.tools` and
  `gitignore.patterns`
- `catalog/starters/go/files/go/mise.toml` and `catalog/starters/go-event-sourcing-service/files/go/mise.toml` —
  write the items
- `catalog/starters/go/files/go/.gitignore` and `catalog/starters/go-event-sourcing-service/files/go/.gitignore` —
  write the items
- `catalog/features/cli/feature.yml` — the items
- `internal/catalog/catalog_test.go` — the tools and the patterns with and without `cli`
- `e2e/tests/new.test.ts` — `git status` after the snapshot and the `npm install`
- `e2e/tests/show.test.ts` — the tree of `show`, if its last lines leave the screen
- `docs/e2e-tests.md` — the run of `go` with `cli`

**Patterns to Follow:**
- `task:1` for the run of `go` with `cli` and `task release:local`
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:30-32` for an item of two fields on
  one line
- `catalog/starters/go-event-sourcing-service/starter.yml:75-78` and `catalog/features/process-manager/feature.yml:96-104`
  for the declaration and the items
- `mise.toml:1-7` and `.gitignore:22-27` of this repo for the tool names, the versions and the patterns
- `internal/catalog/catalog_test.go:1229-1240` for a file with no item
- `e2e/tests/new.test.ts:111-152` for the run of `go` with `cli`
- `tasks/learnings.md:37-42` for mise in a written repo
- `tasks/learnings.md:127-132` for breaks by hand

**Testable:** Yes

**Certainty:** high — the items of `compose.catchup.environment`
(`go-event-sourcing-service/files/go/docker-compose.local.yml:30-32`) are the same kind of item, one line of two
fields, and Task 1 gives the extension points of both starters and the changed-file test of `cli`

**Blast radius:** low — none of the named categories; it adds two tools and two patterns

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write `go` with `--feature cli` with
the built binary, and run `mise install` and `mise ls --current` there: GoReleaser 2.18.2 and Node 26.10.0 are
installed. Apply one break at a time to the catalog, read its diff, run the test that it names, and put the file
back with `git checkout --`: remove the item `dist/` (the e2e run of `go` with `cli`); write the patterns of the
items with no new line between them (`TestCatalog`). Each break makes a test fail. Run `task test:combinations`,
and the e2e tests of `new`, `show` and `check` on the host.

**Depends on:** Task 1

---

### Task 4: Fix the version of each module that the CLI imports in `go.mod`

**Behavior:** Each starter that `cli` targets declares `gomod.requires` in `go.mod`. `cli` adds a required module
for each module that its Go code imports and that no starter requires: `github.com/urfave/cli/v3`,
`github.com/fatih/color` and `github.com/mattn/go-isatty`, at the versions of this repo's `go.mod`. `go mod tidy`
of the folder commands keeps those versions, so a new release of such a module does not change the repo that `new`
makes. A repo without `cli` has the `go.mod` of its starter.

**Acceptance Criteria:**
- [ ] The `starter.yml` of each target of `cli` declares `gomod.requires` with the file `go/go.mod` and the item
  fields `[module, version]`.
- [ ] `TestCatalog`, for each target of `cli`: the `go.mod` written with `cli`, before the folder commands, has a
  `require` with a version for each module that a Go file of the written repo imports, other than the standard
  library and the module of the repo. The test takes the imports from the written Go files with `go/parser`, and
  not from a list. Without `cli`, `go.mod` holds only the lines of the starter.
- [ ] `e2e/tests/new.test.ts`, the run of `go` with `cli`: the written `go.mod` gives each required module of the
  items of `cli` in its `feature.yml` the version of the item. A second `new go my-tool --feature cli --no-prompt`
  into another folder, with `GOPROXY=off`, `GOSUMDB=off` and the module cache of the first, exits with 0, and its
  `go.mod` is the same as the `go.mod` of the first.
- [ ] `docs/e2e-tests.md` of this repo tells the second `new`.
- [ ] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new`, `show` and `check` pass
  on the host.

**Affected Files/Modules:**
- `catalog/starters/go/starter.yml` and `catalog/starters/go-event-sourcing-service/starter.yml` — `gomod.requires`
- `catalog/starters/go/files/go/go.mod.tmpl` and `catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl` —
  write the items
- `catalog/features/cli/feature.yml` — the items
- `internal/catalog/catalog_test.go` — the imports and the required modules for each target
- `e2e/tests/new.test.ts` — the versions after `go mod tidy`, and the second `new` with `GOPROXY=off`
- `e2e/tests/show.test.ts` — the tree of `show`, if its last lines leave the screen
- `docs/e2e-tests.md` — the run of `go` with `cli`

**Patterns to Follow:**
- `task:1` for the changed-file test of `cli` and the run of `go` with `cli`
- `catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl:5-13` and
  `catalog/starters/go/files/go/go.mod.tmpl:1-5` for the `require` lines of the starters
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:30-32` for an item of two fields on
  one line
- `go.mod:5-16` of this repo for the versions
- `internal/catalog/catalog_test.go:1452-1471` and `internal/catalog/catalog_test.go:1717-1738` for the imports of the
  written Go files
- `e2e/tests/new.test.ts:469-494` for `new` with `GOPROXY` and `GOSUMDB` in its environment
- `e2e/tests/new.test.ts:195-200` for two `go.mod` files of two runs of `new`
- `tasks/learnings.md:58-63` for the e2e run on the host
- `tasks/learnings.md:127-132` for breaks by hand

**Testable:** Yes

**Certainty:** high — the `require` block of `go-event-sourcing-service/files/go/go.mod.tmpl:5-13` and the
two-field items of `docker-compose.local.yml:30-32` are the precedent, and the imports test of
`catalog_test.go:1452-1471` already parses the written Go files

**Blast radius:** low — none of the named categories; it fixes the versions of three modules that the written repo
already uses

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write `go` with `--feature cli` with
the built binary, and compare its `go.mod` with the `go.mod` of a write with the binary of `main`: the three
modules have the versions of the items. Apply one break at a time to `catalog/features/cli/feature.yml`, read its
diff, run the test that it names, and put the file back with `git checkout --`: remove the item of
`github.com/fatih/color` (`TestCatalog`, and the second `new` of the e2e run stops); give `github.com/urfave/cli/v3`
a version that is not a release. Each break makes a test fail. Run `task test:combinations`, and the e2e tests of
`new`, `show` and `check` on the host.

**Depends on:** Task 1

---

### Task 5: Add the extension point `ci.jobs`, and run the end-to-end tests of the CLI in the CI workflow

**Behavior:** Each starter that `cli` targets declares `ci.jobs` in `.github/workflows/ci.yml`. The file writes each
item under `jobs:` with `indent 2`, after the jobs of the starter. `cli` adds the CI job `e2e`, which checks out the
repo, installs Task and runs `task test:e2e` on `ubuntu-latest`. The triggers and the permissions of the workflow
stay as the starter gives them. The docs of `cli` tell that CI runs the end-to-end tests. `docs/catalog.md` and
`PROPOSAL.md` give the answer to open question 3 of the proposal (open question 1 of this breakdown).

**Acceptance Criteria:**
- [ ] The `starter.yml` of each target of `cli` declares `ci.jobs` with the file `go/.github/workflows/ci.yml`, the
  item field `[text]`, and a summary that says that an item is a CI job as YAML with the job name at the left edge,
  and that `needs` can name `test`, the job of the starter that runs its tests.
- [ ] `TestCatalog`: without an item, `ci.yml` of each starter holds only the lines of the starter, and its jobs are
  the jobs of the starter.
- [ ] `TestCatalog`: for each target with `cli`, `ci.yml` parses as YAML; its `on`, its `permissions` and each job of
  the starter are the same as without `cli`; it has the job `e2e`, whose steps run `task test:e2e`; and its text
  keeps `${{ secrets.GITHUB_TOKEN }}`.
- [ ] `e2e/tests/new.test.ts`, the runs of `go` with `cli` and of `go-event-sourcing-service` with `cli`:
  `task --list-all` names each task that the written `.github/workflows/ci.yml` calls, taken from the file, and
  `test:e2e` is among them.
- [ ] `docs/e2e-tests.md` of `cli` tells that the job `e2e` of `.github/workflows/ci.yml` runs `task test:e2e` on each
  push, and has no sentence that says that no workflow runs the tests.
- [ ] `docs/catalog.md`, in "Extension Points", tells that an item can hold a block of YAML, which the file writes
  with `indent`, with `ci.jobs` as the example. `PROPOSAL.md` has a decision that a feature adds a job to the CI
  workflow of a starter with an item of `ci.jobs`, and open question 3 names that decision as its answer.
- [ ] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new`, `show` and `check` pass
  on the host.

**Affected Files/Modules:**
- `catalog/starters/go/starter.yml` and `catalog/starters/go-event-sourcing-service/starter.yml` — `ci.jobs`
- `catalog/starters/go/files/go/.github/workflows/ci.yml` and
  `catalog/starters/go-event-sourcing-service/files/go/.github/workflows/ci.yml` — write the items
- `catalog/features/cli/feature.yml` — the job `e2e`
- `catalog/features/cli/files/go/docs/e2e-tests.md` — the job that runs the tests
- `internal/catalog/catalog_test.go` — the workflow with and without `cli`
- `e2e/tests/new.test.ts` — the tasks that the written workflow calls
- `e2e/tests/show.test.ts` — the tree of `show`, if its last lines leave the screen
- `docs/catalog.md` — a YAML item, with `ci.jobs`
- `PROPOSAL.md` — the decision, and open question 3

**Patterns to Follow:**
- `task:2` for `task test:e2e`
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:33-36` and
  `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml:64-67` for a YAML item with `indent` and with
  `[[` `]]` delimiters
- `catalog/starters/go-event-sourcing-service/starter.yml:79-82` and `catalog/features/process-manager/feature.yml:105-109`
  for the declaration and the item of a YAML block
- `.github/workflows/ci.yml:62-74` of this repo for the job `e2e` of go-cli
- `internal/catalog/catalog_test.go:1198-1216` for a parsed YAML file with the item of a feature
- `internal/catalog/catalog_test.go:1229-1240` for a file with no item
- `e2e/tests/new.test.ts:82-109` for the tasks that the workflow calls
- `docs/catalog.md:187-231` for the docs of extension points
- `PROPOSAL.md:544-571` for the decisions and the open questions
- `tasks/learnings.md:16-21` for the delimiters glob
- `tasks/learnings.md:127-132` for breaks by hand

**Testable:** Yes

**Certainty:** high — `compose.services` (`go-event-sourcing-service/files/go/docker-compose.local.yml:33-36`) and
`api.tasks` (`Taskfile.api.yml:64-67`, with the same `[[` `]]` delimiters as `.github/**`) write a YAML block with
`indent 2`, and the job of the item is the job `e2e` of this repo's `ci.yml:62-74`

**Blast radius:** low — none of the named categories; the job runs tests, with the read permission of its
workflow

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write each target with `--feature cli`
with the built binary, and run actionlint on its `.github/workflows/ci.yml`
(`go run github.com/rhysd/actionlint/cmd/actionlint@<a release>`): no finding. Apply one break at a time, read its
diff, run the test that it names, and put the file back with `git checkout --`: write the items with `indent 4` in
the `ci.yml` of `go`; remove the step of `task test:e2e` from the item. Each break makes a test fail. Run
`task test:combinations`, and the e2e tests of `new`, `show` and `check` on the host.

**Depends on:** Task 2

---

### Task 6: Publish a release for each version tag and a prerelease for each push to `main`, after the tests pass

**Behavior:** `cli` adds two more items of `ci.jobs`: `release`, which runs for a push of a tag that starts with
`v`, and `prerelease`, which runs for a push to `main`. Both need `test` and `e2e`, so a release starts only after
the tests pass, and each has `permissions: contents: write` for its job alone. `release` finds the previous
release and runs GoReleaser for the tag. `prerelease` tags the commit with the next patch, the run number and the
commit, runs GoReleaser, and keeps the five newest prereleases, with one run at a time. GoReleaser publishes the
archives and `checksums.txt` of `.goreleaser.yaml`, whose names `update` and `scripts/install.sh` download. The
docs of `cli` tell how a release and a prerelease come.

**Acceptance Criteria:**
- [ ] `TestCatalog`: for each target with `cli`, `ci.yml` has the jobs `release` and `prerelease`. The `if` of
  `release` lets it run only for a ref that starts with `refs/tags/v`, and the `if` of `prerelease` only for
  `refs/heads/main`. The `needs` of each holds `test` and `e2e`.
- [ ] `TestCatalog`: each `needs` of each job of the written workflow names a job of the workflow.
- [ ] `TestCatalog`: only `release` and `prerelease` have the permission `contents: write`, and the `on` and the
  `permissions` of the workflow are the same as without `cli`.
- [ ] `TestCatalog`: the steps of `release` and of `prerelease` are the steps of the jobs `release` of
  `.github/workflows/release.yml` and `prerelease` of `.github/workflows/prerelease.yml` of this repo, and
  `prerelease` has the concurrency group `prerelease` with no cancel of a run in progress.
- [ ] `docs/cli.md` of `cli` tells that a tag `v<major>.<minor>.<patch>` publishes a release after the tests pass,
  that each push to `main` publishes a prerelease, and that `task release:local` makes a snapshot. No sentence says
  that the repo has no workflow that publishes releases or prereleases.
- [ ] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `check` pass on the
  host.

**Affected Files/Modules:**
- `catalog/features/cli/feature.yml` — the jobs `release` and `prerelease`
- `catalog/features/cli/files/go/docs/cli.md` — the section about releases
- `internal/catalog/catalog_test.go` — the release jobs of the written workflow, and the steps of this repo's
  workflows

**Patterns to Follow:**
- `task:5` for the items of `ci.jobs` and the tests of the written workflow
- `.github/workflows/release.yml:1-45` and `.github/workflows/prerelease.yml:1-66` of this repo for the steps of
  go-cli
- `catalog/features/cli/files/go/.goreleaser.yaml:18-38` for the archives, the checksums and the prerelease of a
  tag with a prerelease part
- `catalog/features/cli/files/go/docs/cli.md:58-71` for the section that tells how to publish
- `internal/catalog/catalog_test.go:1198-1216` for a parsed YAML file with the items of a feature
- `e2e/tests/new.test.ts:111-152` for the snapshot whose archives and checksums the release publishes
- `tasks/learnings.md:30-35` for the old defects of ported code

**Testable:** Yes

**Certainty:** medium — the steps are those of this repo's `release.yml:1-45` and `prerelease.yml:1-66`; but here
they are jobs of `ci.yml` that a job-level `if` on the ref and `needs` on the job `test` of the starter hold back,
with job-level permissions and concurrency, which no workflow of this repo or of the catalog has, and no test can
run GitHub Actions

**Blast radius:** high — a contract consumed outside this repository and a token permission: the jobs publish the
archives and `checksums.txt` that `update` and the installer download on the machines of users, and they give the
token `contents: write`, which pushes tags and deletes releases

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write each target with `--feature cli`
with the built binary, and run actionlint on its `.github/workflows/ci.yml`: no finding. Apply one break at a time
to `catalog/features/cli/feature.yml`, read its diff, run `TestCatalog`, and put the file back with
`git checkout --`: change the `if` of `prerelease` to `refs/heads/mainx`; remove `e2e` from the `needs` of
`release`; move `contents: write` to the level of the workflow. Each break makes a test fail. By hand, when the
caller wants it: push a written `go` repo with `cli` to a scratch GitHub repository, push to `main` and then a tag
`v0.1.0`, and see a prerelease and a release, each with the four archives and `checksums.txt`, after the jobs
`test` and `e2e`; then `update --prerelease` from the binary of the release. Run `task test:combinations`, and the
e2e tests of `new` and `check` on the host.

**Depends on:** Task 5

## Summary

- **Tasks:** 6.
- **Order:** dependency first, then value. Task 1 gives the main value, the tasks that build, run and snapshot the
  CLI, and sets the Taskfile of `cli`, the first extension point of `go`, and the test that `cli` changes only the
  files of its extension points, which each later task relies on. Task 2 adds the test tasks to that Taskfile.
  Tasks 3 and 4 add the tools, the patterns and the required modules, each with its own extension points. Task 5
  adds `ci.jobs` with the job that runs the end-to-end tests, and Task 6, the only task with a high blast radius,
  adds the release jobs last, on a workflow whose test jobs exist.
- **Assessment:** Tasks 3, 4 and 5 are high certainty: each repeats an item form of `go-event-sourcing-service`.
  Tasks 1, 2 and 6 are medium: Task 1 is the first flat include and the first extension point of `go`, Task 2 is the first run of the vitest suite of a written repo, and Task 6 turns two workflows into
  two jobs gated by `if` and `needs`. No task is low certainty, because decision 8 and the open question decide
  the two designs that the story leaves open (the run task of the CLI, and how `cli` adds its CI jobs), and each task repeats
  config of go-cli and of this repo. Task 6 has a high blast radius: it publishes the releases that installed CLIs
  download, with a token that writes.
- **Criteria of the story:**
  - "`task build` builds the CLI into `bin/<name>`, `task run -- <arguments>` runs the CLI from the source, and
    `task release:local` makes a GoReleaser snapshot in `dist/`. A repo without `cli` has none of these tasks":
    Task 1, with `task cli -- <arguments>` in place of `task run -- <arguments>` (decision 8).
  - "`task test:e2e` runs the end-to-end tests of the CLI in Docker, and `task test:e2e:local` runs them on the
    machine with Node. The docs of `cli` name these tasks": Task 2, and the docs test of Task 1.
  - "`mise install` in a repo with `cli` installs GoReleaser and Node. After `task release:local` and an
    `npm install` in `e2e/`, `git status` shows no file under `dist/` or `e2e/node_modules/`": Task 3.
  - "The `go.mod` of a repo with `cli` gives a fixed version of each module that the CLI imports": Task 4.
  - "The CI workflow of a repo with `cli` also runs the end-to-end tests of the CLI": Task 5. "A tag publishes a
    GitHub release ..., each push to `main` publishes a prerelease. A release starts only after the tests pass":
    Task 6, with open question 1.
  - "Each change that `cli` makes to a file of the starter is an item of an extension point of the starter ...
    `check` passes, and in the CI of feature-compose a repo of each target with `cli` passes its tests": the
    changed-file test of Task 1 holds it for each later item; each task keeps `check` and
    `task test:combinations` green; the extension points are added in Tasks 1, 3, 4 and 5.
- **Deferred:** no criterion of the story.
- **Checked only by hand:** `mise install` (the e2e image has no mise; `TestCatalog` checks `mise.toml`),
  `task test:e2e` in Docker (the e2e image has no Docker; the e2e test runs `task test:e2e:local`), the run of the
  written workflows on GitHub (the tests check the workflow as written, and actionlint runs by hand), and a real
  release and prerelease.
