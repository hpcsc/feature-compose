# Tasks: Add a CLI to a Service Repo (US-002)

## Contents

1. Task 1: Write the files of the selected features into the folders of the starter
2. Task 2: Render the templates of a feature with the delimiters of its feature.yml
3. Task 3: Declare extension points in a feature, and render their items with `items`
4. Task 4: Select features with `--feature` on `new`
5. Task 5: Port the go-cli template as the feature `cli` into the catalog in the binary
6. Task 6: Name the GitHub repository of the CLI's releases after the module
7. Task 7: Check end to end that a repo with `cli` builds and passes the unit tests of the service and of the CLI

## Story Reference

US-002, "Add a CLI to a service repo", in `user-stories/feature-compose.md` (Acceptance Criteria and Context).
The design is `PROPOSAL.md`: sections 4, 5 (the `features/` folder), 7 (`summary`, `targets`, `files`,
`extension_points`, `delimiters` of `feature.yml`), 9 (`items`), 10 (rules 1, 2, 6, 7 and 8), 11 (`--feature`)
and 16 (go-cli becomes the starter `go` and the feature `cli`).

## Boundaries

**Out of scope:**

The non-goals of the stories file, as written there:

- The command `add`, and an update of a repo to a new version of the catalog. Both need a three-way merge,
  which is phase 3.
- Generators that you run many times in one repo, for example `go-cli-sub-command`. They stay in scaffold.
- Catalogs from other repos. The catalog is in the binary.
- A starter with more than one folder, for example a Go API in `api/` and a Svelte web app in `web/`. The
  feature `cli` shows G5 in one folder.
- A check of the credential in a repo without `api-auth`. The API of such a repo accepts each request.
- A third method of authentication for Google sign-in, with a session exchange.
- The templates that the proposal has not yet decided: `asdf-plugin`, `go-event-sourcing-cli`, `go-worker`,
  `global/git-hook`, and `global/kustomize`.
- The removal of `repo-templates`.

Added by this breakdown:

- `folders` in `feature.yml`: a feature that adds a folder of its own, with its commands. No feature in the
  stories adds a folder.
- The greeting of the root command of go-cli, which reads a name from stdin. The root command of `cli` has
  no action of its own (decision 4 below).
- The presets of go-cli (`example`) and its "All set" message. `mise use`, `git init` and `npm install` from
  the hook of go-cli, as US-001 left out the mise commands of its hook.
- A release repository of the CLI on a host other than github.com. Task 6 stops `new` for such a module.
- Strict checks of `feature.yml`, such as unknown keys, a `file` that the feature does not have, or a target
  that is not a starter. `check` (US-006, US-009) owns the checks of the catalog.

**Deferred:**

- `add_to`, the extension points of a starter, and `unique`: US-003. So in this story no feature adds an item,
  and `cli.commands` renders with zero items. The first real item of `cli.commands` is the `auth` command of
  `cli-auth` in US-005. That story shows, with a real command, the criterion "The root command of the CLI
  shows each command that a feature adds to the extension point `cli.commands`".
- Feature options, `option "<option of the feature>"`, and `--feature <name>:<option>=<value>`: US-003. In this
  story, the value of `--feature` is the name of a feature, and a value with `:` stops as an unknown feature.
- The CI that makes a repo from each starter with each feature: US-004. Needs: US-005. `when`, also on a
  `files` entry: US-007. `--dry-run`: US-010. `.feature-compose.yml`: US-011. The TUI: US-012.
- `show`: US-014. This story reads the `summary` of a feature and the `summary`, `file` and `item` of an
  extension point, and US-014 prints them.
- The checks of the catalog before a run: a folder that a target does not have (US-006); an extension point
  whose `file` does not exist or does not render its items, two owners of one path, and a template that reads
  a starter option that a target does not declare (US-009). In this story, `new` finds two owners of one path
  only in the selection that it writes.
- The stop for a feature that exists but does not target the starter, with its own message: US-016. In this
  story such a name is not among the features that target the starter, so `new` stops with the message for an
  unknown feature.
- `chi-api` as a target of `cli`: US-015. The starter `go` as a target of `cli`: US-016. Each of them must
  declare the options `module` and `go-version`, because the files of `cli` read both (rule 5).
- The parts of go-cli that need a change to a file that the starter owns (rule 7). No extension point takes
  them, and this story adds none. Each needs `add_to` (US-003) and an extension point that no story plans
  yet; the CI parts are open question 3 of the proposal:
  - `Taskfile.yml`: the tasks `run` and `build` of the CLI, and the include of `Taskfile.release.yml`.
    `Taskfile.release.yml` stays out with the include, because only the include reaches it.
  - `Taskfile.test.yml`: the tasks `e2e` and `e2e:local`, and the var `E2E_TAG`. Until then, the docs of `cli`
    give the commands that run the end-to-end tests of the CLI.
  - `.github/workflows/ci.yml`: the `e2e` job, the `workflow_call` trigger, and `branches-ignore: [main]`.
    `release.yml` and `prerelease.yml` stay out, because each calls `ci.yml` as a reusable workflow, which the
    `ci.yml` of the starter is not. `on-demand-build.yml` stays out, because it runs `task release:local`.
  - `mise.toml`: the tools `goreleaser` and `node`.
  - `.gitignore`: `dist/` and `e2e/node_modules/`.
  - `go.mod`: fixed versions of `github.com/urfave/cli/v3`, `github.com/fatih/color` and
    `github.com/mattn/go-isatty`. `go mod tidy`, an `after_render` command of the starter, adds their latest
    versions when it runs.

## Codebase Context

**The catalog.** `catalog/catalog.go:11-35` embeds `all:starters` and finds a starter by name, with the list of
names in the message for an unknown one. `readStarter` (`catalog/starter.go:24-84`) reads `options`, `folders`
and `delimiters` of `starter.yml`, and `readTemplates` (`catalog/template.go:21-53`) gives each template its
folder from the first path segment, removes `.tmpl`, and picks the delimiters of the first glob that matches
(`catalog/template.go:39-45`). `Selection.Write` (`catalog/selection.go:22-45`) renders every template before
the first write and stops when two templates write one path. `render` (`catalog/selection.go:70-81`) has the
functions `option` and the four case functions, and `option` reads only `starter.<option>`
(`catalog/selection.go:83-91`). A message names a template by its path under the `files` folder, for example
`go/main.go.tmpl`, so a template of the starter and a template of a feature at the same place have the same
text today. `Starter.Select(name, values)` has one caller, `internal/cmd/new.go:23-48`.

**The command.** `internal/cmd/new.go:12-50` is `new` with `--option` (a string slice flag; the command sets
`DisableSliceFlagSeparator`, so a value keeps its commas), `--output-dir` and `--no-prompt`.
`internal/cmd/new_test.go:21-48` runs it on catalogs in memory.

**Tests.** One umbrella test per type: `TestCatalog` (groups `starter` and `embedded`), `TestStarter`
(`select`), `TestSelection` (`write`, `run after render`), `TestNewCommand` (`write`, `stop`) and
`TestRootCommand`. The `embedded` group (`catalog/catalog_test.go:169-289`) writes the real starter without
its `after_render` commands. Its test that no file reads, sets or sends a token applies to the starter alone:
the `update` command of `cli` reads `GITHUB_TOKEN`, so a test of a repo with `cli` cannot reuse it. The
end-to-end tests are vitest in `e2e/tests/`; `e2e/tests/new.test.ts:31-53` is the full run of `new`, and the
e2e image has Go, Task, gotestsum and a C compiler.

**Facts checked for this breakdown:**

- `//go:embed all:starters all:features` does not compile while `catalog/features/` has no file ("pattern
  all:features: no matching files found"). So the embed of `features` comes with the first feature (Task 5),
  Tasks 1 to 4 use catalogs in memory, and a catalog without a `features` folder must still work.
- A package named `cli` can import `github.com/urfave/cli/v3`, whose package is also named `cli`, and it
  builds. So `go/internal/cli/root.go` of the proposal keeps the `*cli.Command` of urfave/cli.
- The starter's `task test:unit` runs `gotestsum -- -tags=unit -race … ./...`, and each test file of go-cli has
  `//go:build unit`. So the unit tests of the CLI run in the repo's `task test:unit` with no change to a file
  of the starter.
- `Dockerfile.api` and `Dockerfile.catchup` of the starter copy only `go.mod`, `go.sum`, `internal` and
  `cmd/<app>`, so the `.dockerignore` of `cli` changes nothing in their images.
- The starter's `test:shellcheck` checks each `*.sh` under `scripts/`, so it also checks `scripts/install.sh`
  of `cli`.
- `scaffold --run-hooks never` renders a template without its hook. The hook of go-cli runs `go get`,
  `go mod tidy`, `gofmt -w .`, `mise use`, `git init` and `npm install`. `go mod tidy` alone adds the modules
  that the CLI imports, so `go get` is not needed.
- This repo was made from go-cli: `internal/release/updater.go` is the one of go-cli with the name
  `feature-compose`. So `internal/cmd/`, `internal/release/`, `internal/version/`, `internal/progress/` and
  `e2e/` of this repo show what most files of `cli` render to.

**The go-cli template.** `/Users/davidnguyen/repo-templates/templates/go-cli` has 34 files under
`{{ .ProjectKebab }}/`, the question `GoVersion`, the preset `example`, and the hook `hooks/post_scaffold`.
Each file has one owner in a service repo (rule 7):

| File of go-cli | Owner in a repo with `cli` |
| --- | --- |
| `cmd/{{ .ProjectKebab }}/main.go` | `cli`, as `go/cmd/{{ .Name }}/main.go.tmpl` |
| `internal/cmd/root.go`, `update.go`, `adapter.go`, `adapter_test.go` | `cli`, under `go/internal/cli/` (package `cli`). `root.go` renders `cli.commands` and has no greeting |
| `internal/release/client.go`, `updater.go`, `updater_test.go` | `cli`, under `go/internal/cli/release/` |
| `internal/version/version.go`, `version_test.go` | `cli`, under `go/internal/cli/version/` |
| `internal/progress/line.go`, `line_test.go` | `cli`, under `go/internal/cli/progress/` |
| `.goreleaser.yaml` | `cli`, with `[[` `]]` delimiters |
| `scripts/install.sh` | `cli` |
| `.dockerignore` | `cli`. No starter has one |
| `e2e/Dockerfile`, `package.json`, `tsconfig.json`, `vitest.config.ts`, `testUtils.ts`, `tests/cli.test.ts`, `tests/update.test.ts` | `cli`. `tests/cli.test.ts` checks the root command without the greeting |
| `docs/e2e-tests.md` | `cli`, with the commands that the repo has |
| `README.md` | `cli`, as `docs/cli.md`, because the starter owns `README.md`; with the commands that the repo has |
| `go.mod` | the starter. `go mod tidy` adds the modules of the CLI |
| `hooks/pre-push` | the starter, with the same contents |
| `.gitignore`, `Taskfile.yml`, `Taskfile.test.yml`, `.github/workflows/ci.yml` | the starter. What go-cli adds to them is Deferred |
| `Taskfile.release.yml`, `.github/workflows/release.yml`, `prerelease.yml`, `on-demand-build.yml` | nobody in this story: Deferred |

The values of go-cli become these values of the catalog:

| In go-cli | In `cli` |
| --- | --- |
| `{{.ProjectKebab}}` as the name of the binary, the command, the archives, the e2e image and the scratch folders | `{{ .Name }}`, as US-001 did |
| `github.com/hpcsc/{{.ProjectKebab}}` in imports, in the `-X` ldflags of `.goreleaser.yaml` and `e2e/Dockerfile`, and in `version_test.go` | `option "starter.module"` |
| `/internal/version.releaseTag` in the ldflags | `/internal/cli/version.releaseTag` |
| `hpcsc/{{.ProjectKebab}}` as the GitHub repository of the releases | `hpcsc/{{ .Name }}` in Task 5; Task 6 makes it follow the module |
| `{{.Scaffold.GoVersion}}` in `e2e/Dockerfile` and `docs/e2e-tests.md` | `option "starter.go-version"` |
| `` {{ `{{ .Tag }}` }} `` and the other escapes in `.goreleaser.yaml` | the text of GoReleaser as it is, in a file with `[[` `]]` delimiters |

**Decisions of this breakdown, for the caller to overrule:**

1. **The split of go-cli (rule 7).** `cli` holds each file of go-cli that works in a service repo without a
   change to a file that the starter owns. The table above gives the owner of each file.
2. **All Go code of `cli` is under `cmd/<name>/` and `internal/cli/`.** `internal/cmd` of go-cli becomes
   `internal/cli`, as the proposal names `go/internal/cli/root.go`. Its packages `release`, `version` and
   `progress` go under `internal/cli/`, so the code of the feature is in folders of its own (rule 6), and
   `internal/version` of a service repo does not read as the version of the service.
3. **The binary is `cmd/<name>/`, as in go-cli.** So the files of `cli` use the name in their paths. It does not
   collide with `cmd/api` and `cmd/catchup` of the starter, except in a repo named `api` or `catchup`. There,
   two owners write `cmd/<name>/main.go`, and `new` stops before the first file (Tasks 1 and 5).
4. **The root command of the CLI has no action of its own.** The root command of go-cli greets a name that it
   reads from stdin. The story asks that the CLI "shows only its own commands", so with no command it prints
   its commands and reads no input, as `feature-compose` does (`internal/cmd/root_test.go:40-51`).
5. **The docs of `cli` describe the repo that they are in.** The README of go-cli becomes `docs/cli.md`.
   `docs/cli.md` and `docs/e2e-tests.md` name no task or workflow that the repo does not have, and give the
   commands that build the CLI, make a snapshot release, and run the end-to-end tests of the CLI.
6. **The end-to-end tests of the CLI are in `cli`**, although no task of the repo runs them until the
   Taskfile parts above are done. The docs give the commands that run them in Docker and on the machine.
7. **The delimiters of a `feature.yml` apply to the templates of that feature only**, and the delimiters of the
   starter apply to the templates of the starter only. So a feature renders the same with each of its targets
   (G3).
8. **A `--feature` name that is not a feature that targets the starter stops `new`**, and the message lists the
   features that target the starter. A feature given twice is selected once.
9. **The GitHub repository of the CLI's releases follows the module (Task 6).** go-cli writes
   `hpcsc/<name>`, and its module is always `github.com/hpcsc/<name>`, so the two agree there. In
   feature-compose, `module` is an option: with `module=github.com/acme/ledger`, the repository
   `hpcsc/ledger` would make `update` replace the CLI with a binary of another owner.

## Tasks

### Task 1: Write the files of the selected features into the folders of the starter

**Behavior:** The catalog holds features under `features/<name>/`, each with a `feature.yml` and folders of
templates. A selection of a starter takes the names of features. Each name must be a feature whose `targets`
names the starter; another name stops the selection with the list of the features that target the starter.
The write renders the templates of the starter and of each selected feature in one pass, puts each template
of a feature into a folder of the starter by its first path segment, and stops before the first file when two
templates write one path.

**Acceptance Criteria:**
- [x] The catalog finds each feature at `features/<name>/feature.yml`. `feature.yml` gives `summary`, `targets`
  (the names of starters) and `files` (a list of entries, each with the `path` of a folder under the folder of
  the feature). A catalog with no `features` folder has no features.
- [x] A selection takes the names of features. A name that is not a feature that names the starter in its
  `targets` stops the selection before the first file is written. The message names the name and lists the
  features that target the starter, or says that no feature targets the starter.
- [x] A feature that is given twice is selected once. A feature that is not given writes no file.
- [x] The write renders the templates under each `files` entry of each selected feature. The first path
  segment of a template after the `path` of its entry names a folder of the starter, and the file goes to the
  `path` of that folder in the repo, as a template of the starter does: with the entry `files/`, the template
  `files/go/cmd/{{ .Name }}/main.go.tmpl` of a feature is written to `cmd/my-service/main.go` for the name
  `my-service` and the folder `go` with `path: .`.
- [x] In a template of a feature, `.Name`, `option "starter.<option>"`, the case functions, the `.tmpl` rule
  and the `#!` rule give the same results as in a template of the starter.
- [x] A template of a selected feature whose first path segment is not a folder of the starter stops the
  selection or the write before the first file is written, and the message names the feature, the template and
  the folders of the starter.
- [x] Two templates that write one path, one of the starter and one of a feature or one of each of two
  features, stop the write before the first file is written. The message names the path, and the owner (the
  starter or the feature) and the template of each.
- [x] A `feature.yml` that is not valid YAML stops, and the message names the file.
- [x] A selection without features writes the same files as before, and `new` keeps its behaviour.

**Affected Files/Modules:**
- `catalog/` — the features of the catalog, the format of `feature.yml`, the selection of features, the folder
  and the owner of each template, the check of one owner for each path, and their unit tests
- `internal/cmd/new.go`, `internal/cmd/new_test.go` — the call of the selection, with no features

**Patterns to Follow:**
- `catalog/starter.go:24-84` for the read of a definition and its templates
- `catalog/template.go:21-53` for the folder of a template
- `catalog/selection.go:22-45` for the render before the first write and the check of one path
- `catalog/catalog.go:26-49` for a lookup whose message lists the names
- `catalog/catalog_test.go:33-167` for tests of a catalog in memory
- `catalog/selection_test.go:158-172` for the test of two templates that write one path
- `PROPOSAL.md:152-252` for `feature.yml`
- `PROPOSAL.md:328-354` for rules 2, 6 and 7

**Testable:** Yes

**Certainty:** medium — follows the read and the write of a starter (`catalog/starter.go:24-84`,
`catalog/selection.go:22-45`), but no instance has templates from two owners, or templates whose folders a
different definition declares

**Blast radius:** low — none of the named categories; the refusal of an output folder that has entries stays as
US-001 built it, and its tests cover it

**Verification:** `task test:unit` passes and `go vet ./...` is clean

**Depends on:** None

---

### Task 2: Render the templates of a feature with the delimiters of its feature.yml

**Behavior:** `feature.yml` declares `delimiters` as `starter.yml` does. A template of a feature that matches
one of its globs renders with those delimiters, so text that GoReleaser reads as a template is written as it
is. The globs of a feature apply only to the templates of that feature, and the globs of the starter only to
the templates of the starter.

**Acceptance Criteria:**
- [x] `feature.yml` declares `delimiters`: entries with a `glob`, a `left` and a `right`. A glob matches the path
  of a template of the feature from its folder segment, `.tmpl` included, as a glob of `starter.yml` does:
  `go/.goreleaser.yaml` matches the template `files/go/.goreleaser.yaml` of a feature with the entry `files/`.
- [x] A template of a feature that matches a glob of its `feature.yml` renders with the `left` and the `right` of
  the first entry that matches, and text such as `{{ .Tag }}` in it is written as it is. `.Name`, `option` and
  the case functions give the same values with these delimiters.
- [x] A template of a feature that matches only a glob of the starter, or only a glob of another selected
  feature, renders with `{{` and `}}`. A template of the starter that matches only a glob of a feature
  renders as the rules of the starter say.
- [x] A glob that is not valid, a glob with an empty segment, or delimiters with only `left` or only `right` stop,
  and the message names the feature and the glob.

**Affected Files/Modules:**
- `catalog/` — the delimiters of a feature, and their unit tests

**Patterns to Follow:**
- `catalog/starter.go:67-74` for the read of `delimiters`
- `catalog/template.go:39-45` for the choice of the delimiters of a template
- `catalog/selection_test.go:354-438` for the tests of delimiters
- `catalog/catalog_test.go:103-137` for the tests of delimiters that are not valid
- `tasks/learnings.md:17-23` for the path that a glob matches

**Testable:** Yes

**Certainty:** high — repeats the delimiters of a starter in `catalog/starter.go:67-74` and
`catalog/template.go:39-45`

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 3: Declare extension points in a feature, and render their items with `items`

**Behavior:** `feature.yml` declares `extension_points`: for each name, a `summary`, the `file` that renders it,
and the fields of an `item`. A template of the selection ranges over the items of an extension point that a
selected feature declares with `items "<name>"`. No feature adds an item in this story, so such a range renders
nothing, and the text around it renders as without it. A name that no selected feature declares stops the
write.

**Acceptance Criteria:**
- [x] `feature.yml` declares `extension_points.<name>` with `summary`, `file` and `item`, in the form of
  `cli.commands` in `PROPOSAL.md`. `new` reads them, and in this story it uses only the name.
- [x] In a template of the starter or of a selected feature, `{{ range items "<name>" }}…{{ end }}` for an
  extension point that a selected feature declares renders nothing between `range` and `end`, and the rest of
  the template renders as without the `range`.
- [x] `items` with a name that no selected feature declares, also the name of an extension point of a feature
  that is not selected, stops the write before the first file is written. The message names the template and
  the extension point.
- [x] Two selected features that declare one extension point stop the selection or the write before the first
  file is written, and the message names the extension point and both features.

**Affected Files/Modules:**
- `catalog/` — the extension points of a feature, the template function `items`, and their unit tests

**Patterns to Follow:**
- `catalog/selection.go:70-91` for the template functions and `option`
- `catalog/selection_test.go:337-373` for a template function whose argument is unknown
- `PROPOSAL.md:295-326` for `items`
- `PROPOSAL.md:179-192` for the declaration of `cli.commands`

**Testable:** Yes

**Certainty:** medium — follows the `option` function of `catalog/selection.go:83-91`, but no template function
here gives a list, and the names that it accepts come from the selected features, not from the starter

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 4: Select features with `--feature` on `new`

**Behavior:** `new` takes `--feature <name>` one or more times, and selects each named feature with the
starter. An unknown feature stops `new` before the first file with the message of Task 1.

**Acceptance Criteria:**
- [x] `new <starter> <name> --feature <feature>` writes the files of the starter and of the feature. The flag can
  be given more than one time, and each gives one feature.
- [x] Without `--feature`, `new` writes the same repo as before.
- [x] A value of `--feature` keeps its commas: `--feature a,b` names one feature, `a,b`.
- [x] An unknown feature stops `new` with an exit status other than 0 and the message of Task 1, also when
  another given feature is known. The output folder is not made.
- [x] `new --help` shows `--feature` and its form.

**Affected Files/Modules:**
- `internal/cmd/new.go` — the flag `--feature`
- `internal/cmd/new_test.go` — unit tests with catalogs in memory

**Patterns to Follow:**
- `internal/cmd/new.go:18-27` for the flag `--option` and its values
- `internal/cmd/new_test.go:91-100` for a flag given more than one time
- `internal/cmd/new_test.go:186-196` for a stop that makes no output folder

**Testable:** Yes

**Certainty:** high — repeats the flag `--option` of `internal/cmd/new.go:18-27` and its tests

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 5: Port the go-cli template as the feature `cli` into the catalog in the binary

**Behavior:** The catalog in the binary holds the feature `cli`: its `feature.yml`, and the files of go-cli that
the table in Codebase Context gives to `cli`, with each value of scaffold changed as the second table says.
Its Go code is under `cmd/<name>/` and `internal/cli/`. `internal/cli/root.go` declares `cli.commands` and
renders its items after the commands of the CLI itself, and the root command has no action of its own. The
GitHub repository of the releases stays `hpcsc/<name>`, as in go-cli, and Task 6 changes it.

**Acceptance Criteria:**
- [ ] `catalog/features/cli/feature.yml` has a `summary` that says what the repo gets; `targets` with only
  `go-event-sourcing-service`; `files` with the one entry `files/`; the extension point `cli.commands` with a
  `summary` that says that an item is an expression of type `*cli.Command`, `file: go/internal/cli/root.go`
  and `item: [code]`; and `[[` and `]]` delimiters for `.goreleaser.yaml`. It names no language.
- [ ] The catalog in the binary holds `cli` and its files whose names start with `.`, with no catalog file on
  disk, from any working folder.
- [ ] Written with `go-event-sourcing-service`, the name `my-service`, no given values and the feature `cli`,
  the repo has each file that the go-cli table in Codebase Context gives to `cli`, at the path that the table
  gives, and each file of the starter with the same contents as without `cli`.
- [ ] The binary is `cmd/my-service/main.go`. Each place where go-cli uses the project name, such as the name
  and the usage texts of the root command, the `project_name`, `main` and archive names of
  `.goreleaser.yaml`, and the e2e image and tests, has `my-service`.
- [ ] Each import of a package of the repo, each `-X` ldflag in `.goreleaser.yaml` and `e2e/Dockerfile`, and the
  module path in the version test start with the module, and the ldflags name `internal/cli/version`. Written
  with `module=github.com/acme/ledger` and `go-version=1.28.0`, the files of `cli` have those values, no file
  contains `github.com/hpcsc/my-service`, and `e2e/Dockerfile` builds from `golang:1.28.0`. (Task 6 stops `new`
  with `cli` for a module that is not on github.com, so this test uses a module that is.)
- [ ] `.goreleaser.yaml` keeps the template syntax of GoReleaser: `{{ .Tag }}`,
  `{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}` and `{{ incpatch .Version }}`.
- [ ] `internal/cli/root.go` renders the items of `cli.commands` in the list of commands of the root command,
  after `version` and `update`. Written with no item, the root command has the commands `version` and
  `update` and no action of its own: with no command, it prints its commands and reads no input. The
  `e2e/tests/cli.test.ts` of `cli` checks this, and no file of `cli` greets a name.
- [ ] `scripts/install.sh` can be run in the written repo.
- [ ] No written file of `cli` holds a value of scaffold (`.Scaffold.`, `.ProjectKebab`), a sentinel value, or
  `<no value>`.
- [ ] `docs/cli.md` and `docs/e2e-tests.md` name no task, workflow or file that the written repo does not have,
  and they give the commands that build the CLI, make a snapshot release with GoReleaser, and run the
  end-to-end tests of the CLI in Docker and on the machine.
- [ ] Written with the name `api` and the feature `cli`, the write stops before the first file is written, and
  the message names `cmd/api/main.go`, the starter and `cli`.
- [ ] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass with `cli` in the catalog, and
  `git status --ignored catalog/` shows no file of `cli` as ignored.

**Affected Files/Modules:**
- `catalog/features/cli/feature.yml` — new
- `catalog/features/cli/files/go/` — the files of go-cli that `cli` owns; each `.go` file has the `.tmpl`
  suffix
- `catalog/catalog.go` — the `go:embed` of `features`
- `catalog/catalog_test.go` — the starter with `cli`, written into a temporary folder without its
  `after_render` commands

**Patterns to Follow:**
- `tasks/learnings.md:24-30` for the port of a scaffold template with sentinel values
- `tasks/learnings.md:10-16` for `.tmpl` and `#!`
- `tasks/learnings.md:31-37` for the brief of the audit of ported files
- `/Users/davidnguyen/repo-templates/templates/go-cli/scaffold.yml` and
  `/Users/davidnguyen/repo-templates/templates/go-cli/{{ .ProjectKebab }}/` for the files
- `catalog/catalog_test.go:169-289` for the tests of the catalog in the binary
- `PROPOSAL.md:179-192` for the `feature.yml` of `cli`
- `internal/cmd/root.go:25-37` and `internal/cmd/root_test.go:40-51` for a root command with no action of its
  own
- `e2e/tests/cli.test.ts:4-12` for the end-to-end test of that root command
- Tasks 2 and 3 for the delimiters and `cli.commands`

**Testable:** Yes

**Certainty:** medium — the port of the starter in US-001 and this repo's own copy of go-cli are the precedent
for each file, but no instance splits a template between a starter and a feature, moves packages under
`internal/cli`, or rewrites the docs to the commands that the repo has

**Blast radius:** low — none of the named categories; the token and update code is the one of go-cli, the same
as `internal/release/` of this repo, and Task 6 owns the repository that `update` reads from

**Verification:** `task test:unit` passes. Until Task 7 checks it automatically, also write the starter with `cli`
into a scratch folder and run `go mod tidy`, `gofmt -l .`, `task test:unit` and `go run ./cmd/my-service` there,
and build and run the image of `e2e/Dockerfile` of that repo one time.

**Depends on:** Task 2, Task 3

---

### Task 6: Name the GitHub repository of the CLI's releases after the module

**Behavior:** Each file of `cli` that names the GitHub repository of the CLI's releases names the owner and the
repository of the module. A module that is not on github.com stops `new` with `cli` before the first file,
because `update` and the installer find releases only on GitHub.

**Acceptance Criteria:**
- [ ] For a module `github.com/<owner>/<repo>`, with or without more path segments after it, each file of `cli`
  that names the GitHub repository of the CLI's releases names `<owner>/<repo>`: `hpcsc/my-service` for the
  default module of `my-service`, and `acme/ledger` for `module=github.com/acme/ledger` and for
  `module=github.com/acme/ledger/tools`.
- [ ] Written with `module=github.com/acme/ledger`, no file of `cli` contains `hpcsc`.
- [ ] With `cli` and a module that does not start with `github.com/<owner>/<repo>`, for example
  `example.com/acme/ledger` or `github.com/acme`, `new` stops before the first file is written, and the message
  names the module and says that the releases of `cli` are on GitHub. Without `cli`, such a module makes a
  repo as before.

**Affected Files/Modules:**
- `catalog/features/cli/files/go/` — each file that names the GitHub repository of the releases; in go-cli these
  are the files that contain `hpcsc/{{.ProjectKebab}}` or `owner: hpcsc`
- `catalog/` — only if the render needs a value that it cannot give today
- `catalog/catalog_test.go` — the tests of `cli` in the catalog in the binary

**Patterns to Follow:**
- `catalog/catalog_test.go:223-236` for given values that replace the defaults in each file
- `catalog/selection_test.go:337-373` for a stop of the write that names the template and the value
- Task 5 for the files of `cli`

**Testable:** Yes

**Certainty:** low — the design is what the task decides: go-cli ties the module and the release repository to
one owner, and no file here derives one from the other

**Blast radius:** high — a destructive write and a credential: `update` replaces the running binary with the
archive of this repository, and sends `GITHUB_TOKEN` with each request for it, so a repository of another
owner installs the binary of that owner

**Verification:** `task test:unit` passes

**Depends on:** Task 5

---

### Task 7: Check end to end that a repo with `cli` builds and passes the unit tests of the service and of the CLI

**Behavior:** An end-to-end test runs the binary as the first criterion of the story says, then builds the new
repo, runs its unit tests and runs its CLI. A second test shows that an unknown feature stops the real binary
with the list of features that target the starter.

**Acceptance Criteria:**
- [ ] An end-to-end test runs `feature-compose new go-event-sourcing-service my-service --feature cli --no-prompt`
  in an empty scratch folder. It exits with 0 and writes `./my-service`.
- [ ] In `./my-service`, `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits
  with 0. The output of `task test:unit` names the packages under `internal/cli` that have tests and the
  packages of the service that have tests.
- [ ] In `./my-service`, `go run ./cmd/my-service` with no command exits with 0, reads no input, and prints the
  commands `version` and `update` and no command other than those and `help`.
- [ ] `feature-compose new go-event-sourcing-service x --feature no-such-feature` exits with a status other than
  0, its output lists `cli`, and the scratch folder stays empty.
- [ ] Only the full runs have a timeout longer than the 30 seconds of `e2e/vitest.config.ts`.
- [ ] `docs/e2e-tests.md` of this repo describes the full run with `cli`: what it checks, what it needs, and
  about how long it takes.

**Affected Files/Modules:**
- `e2e/tests/new.test.ts` — the full run with `cli`, and the unknown feature
- `docs/e2e-tests.md` — what the full run with `cli` needs

**Patterns to Follow:**
- `e2e/tests/new.test.ts:31-53` for the full run of `new`
- `e2e/tests/new.test.ts:10-18` for a stop that leaves the scratch folder empty
- `e2e/testUtils.ts:38-57` for `runProgram` and `runCli`
- `tasks/learnings.md:38-44` for `MISE_TRUSTED_CONFIG_PATHS`

**Testable:** Yes

**Certainty:** high — repeats the full run of `e2e/tests/new.test.ts:31-53` and the stop of
`e2e/tests/new.test.ts:10-18`

**Blast radius:** low — none of the named categories

**Verification:** `task test:e2e:local` passes

**Depends on:** Task 4, Task 6

## Summary

**Total:** 7 tasks.

**Order.** Dependency first, then risk. Tasks 1 to 4 build the rules of features on catalogs in memory, because
the embed of `features` does not compile before a feature exists. Task 1 comes first, because each later task
uses the selection of features and the owner of each template. Tasks 2 and 3 add the delimiters and `items`
that the port needs, and Task 4 adds the flag; the three depend only on Task 1. Task 5 ports go-cli. Task 6 is
the one change of the story with a high blast radius, so it has its own review, and the caller can drop it to
keep `hpcsc/<name>` as go-cli has it. Task 7 proves the full run.

**Coverage of the story:**

| Criterion of US-002 | Tasks |
| --- | --- |
| `new go-event-sourcing-service my-service --feature cli --no-prompt` writes the files of the starter and of `cli`; the repo builds, and the unit tests of the service and of the CLI pass | 1 (the write), 4 (the flag), 5 (the files of `cli`), 7 (the full run) |
| The files of `cli` use the name in their contents and their paths, as the files of the starter do | 1 (`.Name` and the folders in a feature), 5 (the places in `cli`) |
| One feature adds files in more than one language, Go and the release config in YAML; the feature names no language | 1 (the format has no language), 2 (the delimiters of `.goreleaser.yaml`), 5 (the Go code and `.goreleaser.yaml`) |
| The root command shows each command that a feature adds to `cli.commands`; with none, it builds and shows only its own commands | 3 (`items`), 5 (the declaration and the render of `cli.commands`), 7 (only its own commands). A real item: US-005 |
| An unknown feature stops the tool before it writes a file, and the message lists the features that target the starter | 1 (the rule and the message), 4 (the flag), 7 (the real binary lists `cli`) |
| Context: a feature gets the name of the repo, as a starter does | 1 |
| Context: `cli` declares `cli.commands` in its own file | 3, 5 |

No criterion of the story is deferred. The criterion of `cli.commands` is shown here with zero items, as the
story allows ("With no such feature, the CLI builds and shows only its own commands"), and with a real item in
US-005.

**Criteria that this breakdown adds, for the caller to overrule:**

- A template of a feature in a folder that the starter does not have, two owners of one path, and two features
  that declare one extension point each stop `new` before the first file (Tasks 1 and 3). This makes "writes
  nothing" true for mistakes that the story does not name, before `check` finds them in CI.
- The scope of the delimiters of a feature (Task 2, decision 7).
- The layout under `internal/cli/`, the removal of the greeting, and `docs/cli.md` (Task 5, decisions 2, 4
  and 5).
- The release repository that follows the module, and the stop for a module that is not on github.com
  (Task 6, decision 9).

**Notes.**

- The build and the tests of a written repo with `cli` are checked automatically only in Task 7. Task 5 checks
  them by hand, so Task 7 can still need a fix in a file of `cli`.
- The end-to-end tests of the CLI in a written repo run only by hand until the Taskfile parts under Deferred are
  done. Task 5 runs them one time.
- Most files of `cli` are ported from go-cli with no change other than the values of the tables in Codebase
  Context. The changed files are `internal/cli/root.go` (the extension point, no greeting),
  `e2e/tests/cli.test.ts` (no greeting), `docs/cli.md` and `docs/e2e-tests.md` (the commands that the repo has),
  and in Task 6 each file that names the release repository. The brief of the audit can say so
  (`tasks/learnings.md:31-37`).
