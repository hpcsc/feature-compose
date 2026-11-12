# Tasks: Make a CLI in Its Own Repo (US-016)

## Contents

1. Task 1: Add the starter `go`, a Go module that builds and passes its tests
2. Task 2: Make `cli` target `go` with the files that it adds to a service repo
3. Task 3: Stop `new` for a feature that does not target the starter, with a message that says so

## Open Questions

The tasks follow the recommendation of each question. Overrule a recommendation before Task 2 starts.

1. **Must the CI of a `go` repo with `cli` publish its releases?** go-cli has `release.yml`, `prerelease.yml` and
   `on-demand-build.yml`. **Recommendation: no.** The criterion "a CLI repo whose releases come from GoReleaser"
   holds with the files that `cli` gives a service repo today: `.goreleaser.yaml`, `scripts/install.sh`, the
   `update` command, and the commands in `docs/cli.md` that make a snapshot and publish a release. Task 2 proves it:
   a GoReleaser snapshot of the written repo builds the archives and `checksums.txt` that `update` and the installer
   download. The reasons:
   - The third criterion asks for one set of files for `go` and for the service starters. So a workflow of `cli`
     also goes into each service repo with `cli`.
   - The workflows of go-cli call `ci.yml` as a reusable workflow. The starter owns `ci.yml` (rule 7), so they need
     a change to a file of the starter. That is open question 3 of the proposal, which this story must not settle.
   - US-002 accepted the same files as "a CLI with releases" in a service repo.

   If the caller wants releases from CI in this story, the smallest change is a workflow that `cli` owns, that runs
   GoReleaser on a tag push, and that does not call `ci.yml`. That adds a task after Task 2, and each service repo
   with `cli` gets the same workflow.

2. **Which story adds the parts of go-cli that a `go` repo with `cli` still lacks?** Each part needs a change to a
   file that the starter `go` owns (rule 7), so it needs an extension point that no story plans:
   - `Taskfile.yml`: the tasks `run` and `build` of the CLI, and the include of `Taskfile.release.yml`
     (`task release:local`).
   - `Taskfile.test.yml`: the tasks `e2e` and `e2e:local`, and the var `E2E_TAG`.
   - `.github/workflows/ci.yml`: the `e2e` job, the trigger `workflow_call`, and `branches-ignore: [main]`. Also
     `release.yml`, `prerelease.yml` and `on-demand-build.yml` (open question 1).
   - `mise.toml`: the tools `goreleaser` and `node`.
   - `.gitignore`: `dist/` and `e2e/node_modules/`. Without them, `git status` shows `dist/` after a snapshot.
   - `go.mod`: fixed versions of `github.com/urfave/cli/v3`, `github.com/fatih/color` and
     `github.com/mattn/go-isatty`. `go mod tidy` adds their latest versions.

   US-002 deferred the same parts for a service repo, and named no story. **No story of
   `user-stories/feature-compose.md` takes them, and no criterion names them.** Its Open Questions say: "No story
   adds a job to the CI workflow of a generated repo." US-021 adds extension points for the settings of the
   Taskfiles of `go-event-sourcing-service`, not for tasks, tools or ignore lines, and not for `go`.
   **Recommendation:** add a story after US-016 whose criteria name each part above, for both starters, and which
   settles open question 3 of the proposal for the CI parts. Until then, `docs/cli.md` gives the commands that
   build the CLI, make a snapshot and run its end-to-end tests, so a `go` repo with `cli` works without the tasks.

## Story Reference

US-016, "Make a CLI in its own repo", in `user-stories/feature-compose.md:533-553` (Description, Acceptance Criteria,
Context). The design is `PROPOSAL.md`: sections 5 and 6 (a starter), 7 (`targets` of `feature.yml`), 10 (rules 2, 3,
5 and 7), 11 (step 2 of `new`: "It checks that each selected feature targets the starter") and 16 (go-cli becomes the
starter `go` and the feature `cli`). The reference for the files of a CLI repo is the template `go-cli` of
`repo-templates` (`/Users/davidnguyen/repo-templates/templates/go-cli/`, read-only). The tool and the catalog do not
depend on it.

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

- A workflow that publishes the releases of the CLI (open question 1).
- `api-auth` and `cli-auth` as features of `go`. The story says that `cli-auth` does not target `go` (rule 3), and
  `go` has no API for `api-auth`. Task 3 gives them the stop of the fourth criterion.
- A program of the starter `go` alone. `cli` owns `cmd/<name>/main.go` (rule 7), so the starter has no `cmd/`.
- A package at the root of the module that has the name of the repo. The example package of `go` has a fixed name
  (decision 2).
- A link from the `README.md` of a `go` repo to `docs/cli.md`. The starter owns `README.md` and does not know `cli`,
  as in a service repo.
- A run of the end-to-end tests of the CLI by the tests of feature-compose (decision 7). The Verification of Task 2
  runs them one time by hand.
- The targets of the feature in the new message. `show` lists them.
- A new message for a need that does not target the starter. `the feature <f> needs "<need>", which is not a feature
  that targets the starter` stays as it is.
- The hook of go-cli (`go get`, `mise use`, `git init`, `npm install`), its preset `example` and its "All set"
  message, as US-001 and US-002 left them out.
- The greeting of the root command of go-cli, which reads a name from stdin. The root command of `cli` keeps no
  action of its own (US-002).

**Deferred:**

- The parts of go-cli in open question 2: no story takes them, and no criterion names them. Open question 2 asks the
  caller for a story.
- `check` of rule 5, a template that reads a starter option that a target does not declare: US-009, whose first
  criterion names it. Until then, the embedded tests of Task 2 and `task test:combinations` find a missing option,
  because the render of `cli` with `go` stops on it.
- `chi-api` as a target of `cli`: US-015, whose criteria say that `cli` targets `chi-api` and that the catalog has one
  set of files for each feature. The test of Task 2 takes the targets of `cli` from its `feature.yml`, so it also
  compares the files for `chi-api` when US-015 adds the target.

## Codebase Context

**The catalog.** A starter is `catalog/starters/<name>/starter.yml` with `files/<folder>/`. The only starter is
`go-event-sourcing-service`. Its `starter.yml` (`catalog/starters/go-event-sourcing-service/starter.yml:1-33`)
declares `go-version` and `module`, the folder `go` with `path: .`, `after_render: [go mod tidy, gofmt -w .]` and
`test: [task test:unit]`, two extension points, and `[[` `]]` delimiters for the Taskfiles and for `.github/`. Go files
end in `.tmpl`, and only a rendered file that starts with `#!` is executable (learning). The feature `cli`
(`catalog/features/cli/feature.yml:1-16`) has `targets: [go-event-sourcing-service]`, the one files entry `files/`,
the extension point `cli.commands`, and `[[` `]]` delimiters for `.goreleaser.yaml`. Its files read
`option "starter.module"` and `option "starter.go-version"`, so each target of `cli` must declare both (rule 5). It
writes `cmd/{{ kebab .Name }}/main.go`, `internal/cli/...`, `.goreleaser.yaml`, `scripts/install.sh`,
`.dockerignore`, `e2e/...`, `docs/cli.md` and `docs/e2e-tests.md`. `githubRepository` stops the write of `cli` for a
module that is not on github.com.

**The selection of features.** `Catalog.Starter` (`internal/catalog/catalog.go:19-32`) calls `featuresTargeting`
(`internal/catalog/catalog.go:120-136`), which reads each `feature.yml` one time with `readFeatureFor`
(`internal/catalog/feature.go:32-46`). That function builds only a feature whose `targets` name the starter, and
drops the others. `readStarter` (`internal/catalog/starter.go:25-72`) keeps the features that target the starter.
`Starter.featuresWithNeeds` (`internal/catalog/starter.go:291-306`) stops on a requested name, from a flag or from an
answer, that is not among them, with `unknownFeature` (`internal/catalog/starter.go:444-450`):
`unknown feature "<name>": the features that target the starter <s> are ...`, or `... no feature targets the
starter <s>`. So today a feature of another starter gets the message of an unknown feature
(`internal/catalog/starter_test.go:217-231`), and so does the form (`internal/form/model_test.go:778-793`).
`Catalog.Check` (`internal/catalog/catalog.go:77-102`) also builds each starter with `readStarter`, from its own
read of all features. `new` selects in `selected` with `--no-prompt`, `--dry-run` or no terminal, and in `form.New`
before the form opens in a terminal (`internal/cmd/new.go`); both call `Starter.Select`, so both stop before the
first file.

**What reads the list of starters.** `show`, the form of `new`, `check` and `Catalog.Combinations` take the starters
from the folders under `starters/`, so a new starter needs no change to the tool. These tests assume one starter:
`e2e/tests/show.test.ts:24-28` (the JSON of `show go-event-sourcing-service` equals the JSON of the whole catalog),
`e2e/tests/show.test.ts:38` and `:137` (an unknown starter lists only `go-event-sourcing-service`),
`e2e/tests/show.test.ts:43-67` (the details of the first starter are those of `go-event-sourcing-service`), and
`e2e/tests/new.test.ts:480-513` (the form with no argument takes the first starter of the list). `go` comes before
`go-event-sourcing-service` in the order of the names. `TestCatalog` "the catalog of this repo has no mistake"
(`internal/catalog/catalog_test.go:790-795`) and the e2e test of `check` read `catalog/`.

**go-cli, the reference.** Each file of go-cli has one owner in a `go` repo with `cli` (rule 7):

| File of go-cli | Owner in a `go` repo with `cli` |
| --- | --- |
| `cmd/<name>/main.go`, `internal/cmd/`, `internal/release/`, `internal/version/`, `internal/progress/`, `.goreleaser.yaml`, `scripts/install.sh`, `.dockerignore`, `e2e/`, `docs/e2e-tests.md` | `cli`, as US-002 ported them. No change |
| `README.md` | `cli` as `docs/cli.md`, and the starter `go` with a README of the module |
| `go.mod` | the starter `go`: the module, the Go version, and the version of testify that `go-event-sourcing-service` gives |
| `hooks/pre-push` | the starter `go`, with `scripts/libs/git-hook-prepush.sh` of `go-event-sourcing-service`. The hook sources that script, and go-cli does not have it |
| `.gitignore` | the starter `go`, without `dist/` and `e2e/node_modules/` (open question 2) |
| `Taskfile.yml` | the starter `go`: the include of `Taskfile.test.yml`, `hook:setup`, and `hook:validate-go-changes`, which runs `test:unit`. `run`, `build` and the include of `Taskfile.release.yml`: open question 2 |
| `Taskfile.test.yml` | the starter `go`: `unit`, `shellcheck` and `vulnerabilities`. `e2e`, `e2e:local` and `E2E_TAG`: open question 2 |
| `.github/workflows/ci.yml` | the starter `go`: one job on each push that runs the shell check, `go build ./...`, the unit tests and the vulnerability check. The rest: open question 2 |
| `Taskfile.release.yml`, `release.yml`, `prerelease.yml`, `on-demand-build.yml` | nobody: open questions 1 and 2 |
| The tools of `hooks/post_scaffold` | the starter `go`, as `mise.toml`: Go, Task, gotestsum, shellcheck and govulncheck, at the versions of `go-event-sourcing-service`. `goreleaser` and `node`: open question 2 |
| (none) | the starter `go`: `starter.yml` and the example package `internal/greeting/` (decision 2) |

**Facts checked for this breakdown:**

- GoReleaser 2.18.2 makes a snapshot of a written repo with `cli` outside a git repository:
  `goreleaser release --snapshot --clean` took 14 seconds, wrote `<name>-darwin-amd64.tar.gz`,
  `<name>-darwin-arm64.tar.gz`, `<name>-linux-amd64.tar.gz`, `<name>-linux-arm64.tar.gz` and `checksums.txt` to
  `dist/`, and the binary of the archive of the host printed `v0.0.0` for `version`.
- GoReleaser stops with `git not present in PATH` when no `git` binary is on the `PATH`, also outside a git
  repository. The e2e image (`e2e/Dockerfile`, stage `node:24-trixie-slim`) has no `git` and no `goreleaser`. The
  `mise.toml` of this repo has `goreleaser` 2.18.2, so `task test:e2e:local` on this machine has both.
- The `combinations` job of `.github/workflows/ci.yml` installs only Go, Task and gotestsum, without mise. So the
  `test` command of a starter can use only those tools.
- Docker does not answer on this machine (learning). The e2e image of Task 2 is checked only in CI.

**Words.** A *starter* and a *feature* keep the meaning of `tasks/us-002-add-cli.design.yaml`. A feature *targets* a
starter when its `targets` name the starter. An *unknown feature* is a requested name that no `feature.yml` of the
catalog has. A *feature that does not target the starter* is a feature of the catalog whose `targets` do not name the
starter of the selection. `go` is the name of the starter, not `go-cli` or `go-module`. The *greeting* is the text
that the example package of `go` gives for a name; `welcome` is taken by the welcome bonus of
`go-event-sourcing-service`.

**Decisions of this breakdown, for the caller to overrule:**

1. **What the starter `go` holds** (Task 1): the files that the table gives to the starter `go`. It owns no path that
   `cli` writes (rule 7).
2. **The example package `internal/greeting`** (Task 1). A `go` repo without `cli` must build and pass tests that
   check something, not an empty `./...`. So the starter holds one package, `internal/greeting`, with one exported
   function that gives the greeting for a name, and its unit tests with the build tag `unit`. The reasons:
   - `cli` owns `cmd/<name>/main.go`, so the starter cannot hold a program there.
   - A fixed package name works with each name of the repo. A package named after the repo fails for a name that
     starts with a digit, which `new` accepts.
   - It has the role of the account example of `go-event-sourcing-service`: code that a developer replaces.
   - The example of go-cli was the greeting of its root command, which US-002 left out of `cli`. The greeting comes
     back as a package, with no read from stdin.
3. **The options of `go`** (Task 1): `module` and `go-version`, with the default and the pattern of
   `go-event-sourcing-service`. `cli` reads both (rule 5), and with the same defaults `cli` writes the same files to
   both starters when no value is given.
4. **The commands of the folder `go`** (Task 1): `after_render: [go mod tidy, gofmt -w .]` and
   `test: [task test:unit]`, as `go-event-sourcing-service` has. The CI job of the combinations has only Go, Task and
   gotestsum.
5. **The CI workflow of `go`** (Task 1) runs `go build ./...` where go-cli runs `task build`. So the workflow builds
   each program of the repo, the CLI included, and calls no task that only `cli` needs. Its actions have the
   versions of go-cli, which run on Node 24.
6. **"Releases come from GoReleaser"** (Task 2): open question 1. The proof is a GoReleaser snapshot of the written
   repo in the e2e test, and the e2e image gets `git` and `goreleaser`.
7. **"Its tests pass"** (Tasks 1 and 2) means the `test` command of the folder, `task test:unit`, which CI of
   feature-compose runs for each combination, as US-002 did. The e2e tests of the CLI need Node and `npm install` in
   the written repo, and no task of the repo runs them (open question 2).
8. **The message** (Task 3) has the form of the message of an unknown feature: `the feature api-auth does not target
   the starter go: the features that target the starter go are cli`, or `...: no feature targets the starter go`. A
   name that no `feature.yml` has keeps `unknown feature "<name>"`. The message is the same for a flag and for an
   answer, with `--no-prompt`, with `--dry-run`, before the form opens, and in the details of the form when a change
   of the starter does not fit a `--feature`.
9. **Where the names come from** (Task 3). `featuresTargeting` already reads each `feature.yml` one time and sees
   which features do not target the starter. The `Starter` keeps their names from that read. It does not build those
   features, so a template of a feature of another starter still does not stop `new`, and no file is read two times
   (learning).
10. **The files that are ported** (for the audit). From go-cli, with the changes of the table: `.gitignore`,
    `Taskfile.yml`, `Taskfile.test.yml`, `.github/workflows/ci.yml` and `go.mod`. From `go-event-sourcing-service`
    with no change: `hooks/pre-push`, `scripts/libs/git-hook-prepush.sh`, and the tool versions of `mise.toml`. New:
    `starter.yml`, `README.md` and `internal/greeting/`. `cli` changes only its `targets`.

## Tasks

### Task 1: Add the starter `go`, a Go module that builds and passes its tests

**Behavior:** `feature-compose new go my-tool --no-prompt` makes a Go module from the starter `go`: `go.mod` with the
module and the Go version, the example package `internal/greeting` with its unit tests, the Taskfiles, `mise.toml`,
`.gitignore`, the CI workflow, the pre-push hook and its script, and a README (decisions 1 to 5). The `after_render`
commands run, the module builds, and `task test:unit` runs the tests of `internal/greeting`. `show`, the form of
`new`, `check` and `task test:combinations` take the second starter from the catalog with no change to the tool. The
tests of `show` and of the form of `new` that assume one starter work with two.

**Acceptance Criteria:**
- [x] `catalog/starters/go/starter.yml` has a `summary` that says what the repo gets; the options `module` and
  `go-version` with the default and the pattern of `go-event-sourcing-service`; the folder `go` with `path: .`,
  `after_render: [go mod tidy, gofmt -w .]` and `test: [task test:unit]`; no extension point; and `[[` `]]`
  delimiters for the Taskfiles and for `.github/`.
- [x] A test of the group `embedded` of `TestCatalog` writes `go` with the name `my-tool` and no given values. The
  repo has `go.mod`, `.gitignore`, `mise.toml`, `Taskfile.yml`, `Taskfile.test.yml`, `.github/workflows/ci.yml`,
  `hooks/pre-push`, `scripts/libs/git-hook-prepush.sh`, `README.md`, and the Go file and the test file of
  `internal/greeting`, and no other file. `go.mod` has `module github.com/hpcsc/my-tool` and `go 1.27.1`, and
  `mise.toml` has `1.27.1`.
- [x] Written with `module=github.com/acme/ledger` and `go-version=1.28.0`, `go.mod`, `mise.toml` and each import of a
  package of the module have those values, and no file holds `github.com/hpcsc/my-tool` or `1.27.1`.
- [x] The Taskfiles and the workflow keep the template syntax of Task and of GitHub Actions, for example
  `${{ secrets.GITHUB_TOKEN }}`. `hooks/pre-push` and `scripts/libs/git-hook-prepush.sh` can be run. No written file
  holds a value of scaffold (`.Scaffold.`, `.ProjectKebab`, `<no value>`, `zzsentinel`).
- [x] The README of the written repo names its module and tells how to install the tools, run the unit tests and set
  up the hook. Each path in backticks in it is a file or a folder of the repo, and each `task <name>` in it is a task
  of its Taskfiles.
- [x] `e2e/tests/new.test.ts`: `feature-compose new go my-tool --no-prompt` exits with 0 and writes `./my-tool`. There,
  `gofmt -l .` lists no file, `go build ./...` and `task --list` exit with 0, and `task test:unit` exits with 0 and its
  lines that start with `✓` name `internal/greeting`. `task --list-all` names each task that
  `.github/workflows/ci.yml` and `scripts/libs/git-hook-prepush.sh` of the repo call, taken from those files and not
  from a list of the test.
- [x] The e2e tests of `show` and of the form of `new` pass with two starters. An unknown starter lists
  `go, go-event-sourcing-service`. `show go-event-sourcing-service --output json` prints that starter and the features
  that target it, which is not the whole catalog. The tree of `show` starts with `go`. The form of `new` with no
  argument selects `go-event-sourcing-service` in the list of starters.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass.
- [x] `task test:combinations` passes, and its output names the combination `go`.
- [x] `README.md` of this repo names the starters of its catalog, `go` and `go-event-sourcing-service`.
  `docs/e2e-tests.md` tells the full run of `go`: what it checks and what it needs.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new`, `show` and `check` pass on
  the host.

**Affected Files/Modules:**
- `catalog/starters/go/starter.yml` — new
- `catalog/starters/go/files/go/` — new: `go.mod.tmpl`, `.gitignore`, `mise.toml`, `Taskfile.yml`,
  `Taskfile.test.yml`, `.github/workflows/ci.yml`, `hooks/pre-push`, `scripts/libs/git-hook-prepush.sh`,
  `README.md`, `internal/greeting/` (`.go.tmpl` files)
- `internal/catalog/catalog_test.go` — the group `embedded`: the repo of `go`
- `e2e/tests/new.test.ts` — the full run of `go`, and the form with no argument
- `e2e/tests/show.test.ts` — the tests that assume one starter
- `docs/e2e-tests.md` — the full run of `go`
- `README.md` — the starters of the catalog of this repo

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/starter.yml:1-33` for `starter.yml`
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml:1-22`,
  `catalog/starters/go-event-sourcing-service/files/go/Taskfile.test.yml:1-22`,
  `catalog/starters/go-event-sourcing-service/files/go/mise.toml:1-7`,
  `catalog/starters/go-event-sourcing-service/files/go/.github/workflows/ci.yml:1-30` and
  `catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl:1-12` for the files of a starter
- `catalog/starters/go-event-sourcing-service/files/go/hooks/pre-push:1-18` and
  `catalog/starters/go-event-sourcing-service/files/go/scripts/libs/git-hook-prepush.sh:1-107` for the hook
- `catalog/starters/go-event-sourcing-service/files/go/README.md:1-30` for the README of a starter
- `internal/catalog/catalog_test.go:818-827`, `internal/catalog/catalog_test.go:839-849`,
  `internal/catalog/catalog_test.go:889-910`, `internal/catalog/catalog_test.go:923-931` and
  `internal/catalog/catalog_test.go:1586-1593` for the tests of a written starter
- `internal/catalog/catalog_test.go:790-795` for the check of the catalog of this repo
- `e2e/tests/new.test.ts:43-67` and `e2e/tests/new.test.ts:742-747` for the full run and the packages that passed
- `e2e/tests/show.test.ts:6-40`, `e2e/tests/show.test.ts:43-67` and `e2e/tests/new.test.ts:480-513` for the tests that
  assume one starter
- `docs/e2e-tests.md:80-169` for the docs of a full run
- `README.md:24-41` for the catalog of this repo
- `tasks/learnings.md:10-15` for `.tmpl` and `#!`
- `tasks/learnings.md:16-22` for the delimiters globs
- `tasks/learnings.md:23-29` for the port of go-cli with sentinel values
- `tasks/learnings.md:30-36` for the files that are ported (decision 10)
- `tasks/learnings.md:37-43`, `tasks/learnings.md:51-57` and `tasks/learnings.md:58-64` for the e2e run of a written repo
- `tasks/learnings.md:85-91`, `tasks/learnings.md:92-98` and `tasks/learnings.md:127-133` for the proof of template code
  by breaks by hand
- `tasks/learnings.md:155-161` for a Go composite literal in a `.go.tmpl`
- `tasks/learnings.md:223-228` for the waits of the screen tests of `show`
- `tasks/learnings.md:285-291` for `task test:combinations`

**Testable:** Yes

**Certainty:** medium — `catalog/starters/go-event-sourcing-service/` and its embedded tests
(`catalog_test.go:818-931`) are the precedent for a starter and its files; but this is the second starter, and the
tests of `show` and of the form assume one, and go-cli has no example package for `internal/greeting`

**Blast radius:** low — none of the named categories; a new starter changes no repo that `new` makes from
`go-event-sourcing-service`, and the JSON of `show` keeps its shape

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write `go` with the built binary into a
scratch folder, and run `task test:unit`, `task test:shellcheck` and `task test:vulnerabilities` there. Apply one break
at a time to the written `internal/greeting` Go file, read its diff, run `go test -tags unit ./internal/greeting/`, and
put the file back: give a fixed text; leave out the name. Each break makes a test fail. Run `task test:combinations`,
and the e2e tests of `new`, `show` and `check` on the host.

**Depends on:** None

---

### Task 2: Make `cli` target `go` with the files that it adds to a service repo

**Behavior:** `catalog/features/cli/feature.yml` targets `go` and `go-event-sourcing-service`, with its one files entry
and no change to its files. `feature-compose new go my-tool --feature cli --no-prompt` writes the repo of `go` and the
files of `cli`: the CLI in `cmd/my-tool/` and `internal/cli/` with the commands `version` and `update`,
`.goreleaser.yaml`, `scripts/install.sh`, the e2e tests of the CLI, `docs/cli.md` and `docs/e2e-tests.md`. The repo
builds, and `task test:unit` runs the tests of the starter and of the CLI. A GoReleaser snapshot of the repo builds the
archives and `checksums.txt` that `update` and the installer download (open question 1, decision 6).

**Acceptance Criteria:**
- [x] The `targets` of `catalog/features/cli/feature.yml` name `go` and `go-event-sourcing-service`, and its `files`
  entry is `files/` with no `when`.
- [x] A test of the group `embedded` takes the targets of `cli` from its `feature.yml`, not from a list of the test. For
  each target, it writes the target alone and the target with `cli`, with the name `my-tool`; the files of the target
  stay as without `cli`, and the files that `cli` adds are the rest. For each two targets, the files that `cli` adds
  have the same paths and the same contents: with no given values, and with `module=github.com/acme/ledger` and
  `go-version=1.28.0`.
- [x] Written with `go` and `cli`, the repo has each file that `cli` adds to a service repo
  (`internal/catalog/catalog_test.go:1201-1226`), at the same path.
- [x] For each target of `cli`, `docs/cli.md` and `docs/e2e-tests.md` of the written repo name no `task `, `Taskfile`,
  `.github/workflows` or `release:local`.
- [x] `e2e/tests/new.test.ts`: `feature-compose new go my-tool --feature cli --no-prompt` exits with 0. In `./my-tool`,
  `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits with 0 and its lines that start
  with `✓` name `internal/greeting` and each package under `internal/cli` whose folder holds a test file of the repo.
  `go run ./cmd/my-tool` with no command exits with 0, reads no input, and prints the commands `version`, `update` and
  `help`, and no other.
- [x] In the same test, `goreleaser release --snapshot --clean` in `./my-tool` exits with 0. For each `goos` and
  `goarch` of the `.goreleaser.yaml` of the repo, `dist/` holds `my-tool-<goos>-<goarch>.tar.gz`, the name that
  `update` and `scripts/install.sh` download. `dist/checksums.txt` gives the SHA-256 of each archive, as the test
  computes it from the file. The binary in the archive of the platform of the test prints the tag of the snapshot for
  `version`, so the ldflag of `.goreleaser.yaml` sets the version.
- [x] The runner stage of `e2e/Dockerfile` has `git` and `goreleaser` at the version of `mise.toml` of this repo.
  `docs/e2e-tests.md` tells the full run of `go` with `cli`, and names GoReleaser and git in what the e2e tests need.
- [x] `README.md` of this repo tells that `feature-compose new go my-tool --feature cli` makes a CLI in its own repo.
- [x] `task test:combinations` passes, and its output names the combination `go --feature cli`. `TestCatalog` "the
  catalog of this repo has no mistake" and the e2e test of `check` pass.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new`, `show` and `check` pass on
  the host.

**Affected Files/Modules:**
- `catalog/features/cli/feature.yml` — the target `go`
- `internal/catalog/catalog_test.go` — the files of `cli` for each target, and the docs of `cli` for each target
- `e2e/tests/new.test.ts` — the full run of `go` with `cli`, with the GoReleaser snapshot
- `e2e/Dockerfile` — `git` and `goreleaser` in the runner stage
- `docs/e2e-tests.md` — the full run of `go` with `cli`, and what it needs
- `README.md` — a CLI in its own repo

**Patterns to Follow:**
- `task:1` for the starter `go` and its full run
- `catalog/features/cli/feature.yml:1-16` for the `feature.yml` of `cli`
- `internal/catalog/catalog_test.go:1189-1199` and `internal/catalog/catalog_test.go:1201-1226` for the files that
  `cli` adds
- `internal/catalog/catalog_test.go:1487-1498` for the docs of `cli`
- `e2e/tests/new.test.ts:309-333` and `e2e/tests/new.test.ts:742-763` for the full run with `cli`, the packages that
  passed, and the commands of the CLI
- `e2e/Dockerfile:7-9` and `e2e/Dockerfile:16-23` for the tools of the e2e image
- `docs/e2e-tests.md:80-169` and `docs/e2e-tests.md:171-210` for the docs of a full run and of the e2e image
- `tasks/learnings.md:37-43`, `tasks/learnings.md:51-57` and `tasks/learnings.md:58-64` for the e2e run of a written repo
- `tasks/learnings.md:285-291` for `task test:combinations`

**Testable:** Yes

**Certainty:** medium — the tests of `cli` (`catalog_test.go:1201-1226`) and its full run (`new.test.ts:309-333`) are
the precedent; but no test compares the files of one feature across two targets, and no e2e test runs GoReleaser,
whose run in the e2e image needs two tools that the image does not have and that only CI can check

**Blast radius:** low — none of the named categories; no file of `cli` changes, so `update`, the installer and their
token stay as US-002 made them, and the archive names that `update` reads come from the same `.goreleaser.yaml`

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write `go` with `--feature cli` with the
built binary, and run `goreleaser release --snapshot --clean` there. Run the e2e tests of the CLI in the written repo
one time, with the commands of its `docs/cli.md`. Change the `name_template` of the archives in
`catalog/features/cli/files/go/.goreleaser.yaml`, then remove one `goarch`, run the e2e test of `go` with `cli` after
each change, and put the file back with `git checkout --`; each change makes the test fail. Run `task test:combinations`,
and the e2e tests of `new`, `show` and `check` on the host. CI runs the e2e image, because Docker does not answer here.

**Depends on:** Task 1

---

### Task 3: Stop `new` for a feature that does not target the starter, with a message that says so

**Behavior:** A requested feature, from `--feature` or from an answer of the form, that is a feature of the catalog but
does not target the starter stops the selection before the first file. The message names the feature and the starter,
says that the feature does not target the starter, and lists the features that target the starter, or says that no
feature targets it (decision 8). A name that no `feature.yml` of the catalog has keeps the message of an unknown
feature. The `Starter` takes the names of the features that do not target it from the one read of the `feature.yml`
files (decision 9). `new` stops with the message with `--no-prompt`, with `--dry-run`, and before the form opens; the
form shows it when a change of the starter does not fit a `--feature`.

**Acceptance Criteria:**
- [x] `TestStarter`, group `select`, with a catalog in memory that has two starters: a feature that targets only the
  other starter stops `Select`, from a flag and from an answer, with the same message. The message names the feature
  and the starter, says that the feature does not target the starter, lists the features that target the starter, and
  does not hold `unknown feature`. For a starter that no feature targets, the message says so.
- [x] A name that no `feature.yml` of the catalog has still stops with `unknown feature "<name>"` and the list of the
  features that target the starter, also next to a known feature. A feature that targets both starters is selected
  with each of them.
- [x] A read of a `feature.yml` that fails stops `Catalog.Starter` with its cause, also for a feature that does not
  target the starter (`internal/catalog/catalog_test.go:276-297`).
- [x] `TestNewCommand`: `new <starter> <name> --feature <a feature of the other starter>` stops with the message and
  makes no output folder; with `--dry-run` it stops with the same message and prints nothing; in a terminal it stops
  before the form opens.
- [x] The form test of a starter that a `--feature` does not target (`internal/form/model_test.go:778-793`) shows the
  message that `new` gives, and keeps the tree as it was.
- [x] `e2e/tests/new.test.ts`: for each feature of the catalog whose `targets` do not name `go`, taken from the JSON of
  `show` and not from a list of the test, `feature-compose new go my-tool --feature <feature>` exits with a status
  other than 0. Its output names the feature and `go` and says that the feature does not target the starter, and the
  scratch folder stays empty. `--feature no-such-feature` still gives the message of an unknown feature.
- [x] `README.md` of this repo names a feature that does not target the starter among the selections that stop `new`
  and `--dry-run`.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and the
  e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `internal/catalog/catalog.go` — `featuresTargeting` also gives the names of the features that do not target the
  starter; `Check` builds its starters with the same `readStarter`
- `internal/catalog/starter.go` — the names of the features that do not target the starter, and the message
- `internal/catalog/starter_test.go` — the stop for a feature that does not target the starter
- `internal/cmd/new_test.go` — the stop of `new`, with `--dry-run` and in a terminal
- `internal/form/model_test.go` — the message of a starter that a `--feature` does not target
- `e2e/tests/new.test.ts` — the stop of `new go my-tool` for each feature that does not target `go`
- `README.md` — the stops of `new`

**Patterns to Follow:**
- `task:1` for the starter `go` in the e2e test
- `internal/catalog/starter.go:444-450` and `internal/catalog/starter.go:291-306` for the message of an unknown feature
  and where it stops the selection
- `internal/catalog/catalog.go:19-32`, `internal/catalog/catalog.go:120-136` and `internal/catalog/feature.go:32-46`
  for the one read of each `feature.yml`
- `internal/catalog/catalog.go:77-102` for the other caller of `readStarter`
- `internal/catalog/starter_test.go:217-231`, `internal/catalog/starter_test.go:239-275` and
  `internal/catalog/starter_test.go:295-307` for the tests of the stop
- `internal/catalog/catalog_test.go:276-297` for a read that fails
- `internal/cmd/new_test.go:534-565`, `internal/cmd/new_test.go:730-735` and `internal/cmd/new_test.go:812-821` for the
  stops of `new`
- `internal/form/model_test.go:778-793` for a starter that does not fit
- `e2e/tests/new.test.ts:299-307` for the e2e stop of an unknown feature
- `README.md:97-101` and `README.md:181-196` for the stops of `new` in the README
- `tasks/learnings.md:183-189` for the tests of the package of the code
- `tasks/learnings.md:204-210` for the words of `internal/catalog`
- `tasks/learnings.md:211-216` for a run of `clerk mutants` on `internal/catalog`
- `tasks/learnings.md:285-291` for `go vet -tags combinations`
- `tasks/learnings.md:299-305` for a test of the command that reaches the new path
- `tasks/learnings.md:306-312` for a rule about one feature
- `tasks/learnings.md:313-319` for one read of the files of the catalog

**Testable:** Yes

**Certainty:** high — `Starter.unknownFeature` (`starter.go:444-450`) is the message to follow, the case already has
a test (`starter_test.go:217-231`), and `featuresTargeting` (`catalog.go:120-136`) already reads the `feature.yml` of
each feature that does not target the starter

**Blast radius:** low — none of the named categories; it changes the text of a message that stops `new` before the
first file, and no selection that `new` accepts today

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Run `clerk mutants` on the task. Run `feature-compose new go my-tool --feature api-auth` with the built binary in a
scratch folder, and read the message. Run the e2e tests of `new` on the host.

**Depends on:** Task 1

## Summary

- **Tasks:** 3.
- **Order:** dependency first, then value. Task 1 adds the starter `go`, which each later task needs, and gives the first
  criterion. Task 2 gives the main value of the story, a CLI in its own repo from the files of `cli`, and holds the
  riskiest part: the GoReleaser snapshot in an e2e image that only CI runs. Task 3 changes the one rule of the tool that
  the story needs; its unit tests need no other task, and its e2e test needs `go`.
- **Assessment:** Task 3 is high certainty: the message and the read that it extends exist, and a test of the case
  exists. Tasks 1 and 2 are medium: each repeats a precedent of the catalog, but Task 1 adds the second starter, which
  some tests assume does not exist, and Task 2 adds the first e2e run of GoReleaser. No task is low certainty, because
  this breakdown decides the content of the starter (decisions 1 to 5) and the open questions keep the release
  workflows out. Each task has a low blast radius: the story adds a starter, a target and a message, and changes no
  authentication, credential, money, migration or contract outside this repository. This is a routine story.
- **Criteria of the story:**
  - "`feature-compose new go my-tool --no-prompt` makes a Go module that builds and passes its tests": Task 1.
  - "`feature-compose new go my-tool --feature cli --no-prompt` makes a CLI repo whose releases come from GoReleaser.
    The CLI has the commands `version` and `update`, and the repo has a script that installs the latest release. Its
    tests pass": Task 2, with open question 1 about the release workflows.
  - "`cli` uses one set of files for the starter `go` and for the service starters": Task 2.
  - "`feature-compose new go my-tool --feature api-auth` stops before it writes a file. The message names the feature
    and the starter, and says that the feature does not target the starter": Task 3.
- **Deferred:** no criterion of the story. The parts of go-cli in open question 2 have no story.
