# Tasks: Make a Repo From the go-event-sourcing-service Starter (US-001)

## Contents

1. Task 1: Write a starter's templates into a new repo folder
2. Task 2: Run the after_render commands of each folder
3. Task 3: Set the starter options from their defaults and from given values
4. Task 4: Render the templates that match a delimiters glob with their own delimiters
5. Task 5: Write `.tmpl` templates without the suffix, and scripts as executables
6. Task 6: Port the go-event-sourcing-service starter, without the process manager, into the catalog in the binary
7. Task 7: Remove the shared-token check from the starter
8. Task 8: Replace the sample command with `new`
9. Task 9: Check end to end that a new repo passes `task test:unit` and `task --list`

## Story Reference

US-001, "Make a repo from the go-event-sourcing-service starter", in `user-stories/feature-compose.md`
(Acceptance Criteria and Context). The design is `PROPOSAL.md`: sections 4, 5, 6, 8 (options only), 9, 10
(rules 7 and 9), 11 (`new`, `--option`, `--output-dir`, `--no-prompt`), and decisions D8 and D10.

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

- `mise trust` and `mise install` from the hook of the scaffold template. The folder `go` runs only the two
  commands that the story names. A developer runs mise in the new repo.
- The "All set" message and the presets (`example`, `minimal`) of the scaffold template.
- A check of the form or the case of `<name>`. The tool uses the name as given, as the default of `module`
  does.
- A default of an option that reads another option. A default can read `.Name` and the case functions only.
- Strict checks of `starter.yml`, such as unknown keys. `check` (US-006) owns the checks of the catalog.
- A file mode from the catalog other than the executable bit that Task 5 gives to scripts.

**Deferred:**

- Features, `--feature`, and folders that a feature adds: US-002.
- `extension_points`, `items`, `unique`, `add_to`, and the `api-auth` feature with its `api-key` method: US-003.
  The starter of this story has no `extension_points`.
- Running the `test` commands of each folder in CI: US-004. In this story, `test` is data only.
- Needs: US-005. `check`: US-006. `when` expressions: US-007.
- `--dry-run`: US-010. The selection file `.feature-compose.yml`: US-011.
- The TUI: US-012. Until then, `new` without `--no-prompt` does the same as `new` with it.
- `show`: US-014. The `summary` of a starter is read here and printed there.
- The process manager as the feature `process-manager`: US-017.

## Codebase Context

**The CLI today.** `cmd/feature-compose/main.go` calls `cmd.Run`. `internal/cmd/root.go` builds the root
`cli.Command` (urfave/cli v3) with the commands `version` and `update`, and a root action that greets a
name read from stdin. The `new` command replaces that greeting. `internal/cmd/update.go:21-32` is a command
with flags, and `internal/cmd/adapter.go:28-37` turns a two-argument function into an action that rejects
fewer than two arguments. `asTwoArgumentsAction` has no caller yet.

**Tests.** Unit tests carry `//go:build unit` and run with `task test:unit` (gotestsum, `-race`).
`internal/release/updater_test.go` is the model: an external test package, one umbrella test per type,
`t.Run` groups per operation, scenario names as sentences, testify `require`, and a negative path that
asserts that nothing changed (`internal/release/updater_test.go:255-275`). `internal/cmd/adapter_test.go`
tests inside package `cmd` by running a `cli.Command` with arguments. The end-to-end tests are vitest in
`e2e/tests/`. `runCli` in `e2e/testUtils.ts:36-53` runs the binary in a scratch folder and returns stdout,
stderr and the exit status.

**No precedent in this repo.** The Go code uses no `text/template`, no `go:embed`, no YAML, no `io/fs`, and
starts no external program. `go.yaml.in/yaml/v3` is in `go.mod` as an indirect requirement only.

**Facts checked for this breakdown:**

- A `go.mod` inside the folder that `go:embed` names makes the embed fail ("contains no embeddable files").
  Without that `go.mod`, `go build ./...` of this repo compiles the Go templates and fails on their template
  import paths. So the catalog cannot hold the starter's `go.mod` and Go files under their own names.
  Task 5 adds the `.tmpl` rule for this.
- `go:embed` reports each file as `-r--r--r--`, so the executable bit of `hooks/pre-push` and of the scripts
  under `scripts/` is lost. Task 5 adds the `#!` rule for this.
- `go:embed` leaves out names that start with `.` or `_` unless the pattern starts with `all:`. The starter
  has `.github/`, `.gitignore`, `.env.local`, `.env.integration` and `.tmuxp.yaml`.
- urfave/cli v3 parses flags that come after the arguments, so
  `new go-event-sourcing-service my-service --no-prompt` gives two arguments and the flag.
- `path.Match` has no `**`, and the delimiters glob of the proposal, `go/**/Taskfile*.yml`, needs it.
- The starter's `.gitignore` (`bin`, `tmp`, `tags`, `*.out`, `*.test`, `go.work`, `.idea`, `.junie`, and
  binaries) also applies to its own folder in this repo. No file of the starter matches it today.

**The scaffold template.** `/Users/davidnguyen/repo-templates/templates/go-event-sourcing-service` has 245
files under `{{ .ProjectKebab }}/`. The `features` globs of its `scaffold.yml` select 85 of them for the
process manager. Of the other 160, 118 are `.go` files, plus `go.mod`. The template uses these values:

| Value in the template | Where | In the starter |
| --- | --- | --- |
| `{{ .ProjectKebab }}`, and `[[ .ProjectKebab ]]` in Taskfiles | README title, event `Namespace`, a registry test, compose project names, image tags, the tmuxp session, `start-test-dependencies.sh` | `.Name`, as the default of `module` in the proposal uses it |
| `github.com/hpcsc/{{ .ProjectKebab }}` | `go.mod`, the imports of the Go files, the `-X` ldflags of `Taskfile.api.yml` and `Dockerfile.api` | `option "starter.module"` |
| `{{ .ModulePath }}` | `internal/domain/reaction/conflict_retry.go` and its test | `option "starter.module"`. `scaffold.yml` defines no `ModulePath`, so these imports do not resolve in the template today. |
| `{{ .Scaffold.GoVersion }}` | `go.mod`, `mise.toml`, `Dockerfile.api`, `Dockerfile.catchup` | `option "starter.go-version"`, with the regex of the `GoVersion` question as its pattern |
| `{{ .Scaffold.ProcessManager }}` | 10 shared files | removed. Each shared file keeps its branch for `false`. |

The template uses no `.Computed.*` value. Its hook runs `go mod tidy` (an `after_render` command here) and
`mise trust` and `mise install` (out of scope).

**The process manager.** Its files are the `features` globs of `scaffold.yml`. Its edits to shared files are
listed in `/Users/davidnguyen/repo-templates/templates/global/go-event-sourcing-service-process-manager/hooks/post_scaffold:175-249`:
`account.go`, `events.go`, `balance/projection.go`, `internal/use_cases/register.go`, `cmd/catchup/main.go`,
`internal/api/use_cases/register.go`, `README.md`, `Taskfile.api.yml`, `Taskfile.yml`, `.tmuxp.yaml` and
`docker-compose.local.yml`. In the main template, these edits are the `.Scaffold.ProcessManager` branches.

**The shared token.** The files are `internal/api/middleware/auth.go`, `auth_test.go`,
`internal/api/middleware/testdata/valid-token` and `local/api/token`. The wiring is in
`internal/api/use_cases/register.go` (the middleware on the group of protected routes),
`internal/api/config.go` (`TokenPath`), `Taskfile.api.yml` (the `AUTH` header var and `TOKEN_PATH`),
`docker-compose.local.yml` (the `./local/api` volume of `api`), the remark about auth in
`internal/api/test/common.go`, and "with a bearer token" in `docs/event-model-account.md` and `.emod`.
`internal/api/route/route.go` splits routes into public and protected.

**End-to-end environment.** The runner stage of `e2e/Dockerfile` is `node:24-trixie-slim`. It has no Go, Task
or gotestsum, and it has `g++` for node-pty. The starter's `task test:unit` runs with `-race`, which needs cgo
and a C compiler. `e2e/vitest.config.ts` sets a 30 second timeout for each test. The new repo has its own
`mise.toml`. On a machine where `task` or `go` resolve through mise shims, mise can refuse that untrusted
file in the scratch folder.

**Where each kind of test goes.** Unit tests stay offline and fast: they use catalogs that are built in
memory with `testing/fstest`, and `after_render` commands that are shell builtins. They render the real
starter without its `after_render` commands. The end-to-end tests run the real binary: the error paths of
`new` in Task 8, which stop before the first file, and the one slow run with the network in Task 9.

## Tasks

### Task 1: Write a starter's templates into a new repo folder

**Behavior:** A catalog finds a starter by name in a file system. A selection of that starter with a repo
name renders each template of the starter with Go `text/template` and writes the files into an output
folder. `.Name` and the case functions work in the contents and the paths. The write refuses a folder that
has entries, and it writes nothing when a template fails.

**Acceptance Criteria:**
- [x] The catalog finds the starter `<name>` at `starters/<name>/starter.yml`. An unknown name stops with a
  message that lists the names of all starters in the catalog.
- [x] `starter.yml` gives `summary` and `folders.<name>.path`. The first path segment of each template under
  `starters/<name>/files/` names a folder, and the file goes to the `path` of that folder in the repo
  (`path: .` is the root of the repo).
- [x] The contents and the path of each template render with Go `text/template`. `{{ .Name }}` is the name of
  the repo, in the contents and in a path segment: for the name `my-service`, the template
  `go/cmd/{{ .Name }}/main.go` is written to `cmd/my-service/main.go`.
- [x] `kebab`, `snake`, `camel` and `pascal` are template functions. Each gives one result for a name written
  in any of the four cases: `my-service`, `my_service`, `myService` and `MyService` all give `my_service`
  from `snake`.
- [x] The write makes the output folder when it does not exist, and writes into it when it exists and is
  empty.
- [x] When the output folder has any entry, hidden entries included, the write stops with a message that
  names the folder, and the contents of the folder stay the same.
- [x] A template that fails to parse or to execute stops the write before the first file is written, and the
  message names the template.
- [x] A template whose first path segment is not a folder of `starter.yml` stops the write before the first
  file is written, and the message names the template.

**Affected Files/Modules:**
- `catalog/` (new package at the root of the repo) — the catalog, the starter, the selection, the format of
  `starter.yml`, the render and the write, the case functions, and their unit tests
- `go.mod`, `go.sum` — the YAML library becomes a direct requirement

**Patterns to Follow:**
- `internal/release/updater_test.go:110-137` for the umbrella test in an external test package
- `internal/release/updater_test.go:255-275` for a rejection that leaves the state unchanged
- `PROPOSAL.md:99-150` for the format of `starter.yml`
- `PROPOSAL.md:295-326` for the values in a template

**Testable:** Yes

**Certainty:** low — no precedent: this is the first use of `text/template`, YAML and `io/fs` here, and the
task sets the shape of the package that each later story adds to

**Blast radius:** high — destructive write: the refusal of a folder with entries is the only thing that
stops the tool from writing over the files of a developer

**Verification:** `task test:unit` passes and `go vet ./...` is clean

**Depends on:** None

---

### Task 2: Run the after_render commands of each folder

**Behavior:** After the write, a selection runs the `after_render` commands of each folder of the starter,
in order, in that folder of the repo. The first command that fails stops the run, and the message names the
folder, the command and its output. The `test` commands are data, and the tool does not run them.

**Acceptance Criteria:**
- [x] Each command in `folders.<name>.after_render` runs in that folder of the written repo (`path: .` is the
  root of the repo), in the order of the list, after the last file is written.
- [x] A command that exits with a status other than 0 stops the run, and the commands after it do not run.
  The message names the folder, the command, and the output of the command, stderr included.
- [x] A command that cannot start, for example a program that is not on PATH, stops the run with a message
  that names the same three things.
- [x] The `test` commands of a folder are read from `starter.yml` and do not run.
- [x] No command runs in a folder outside the written repo, also when the tool runs from another folder.
- [x] When the context ends, for example on Ctrl-C, the command that runs stops and the run returns an error.

**Affected Files/Modules:**
- `catalog/` — the commands of a folder, the run of the commands of a selection, and their unit tests

**Patterns to Follow:**
- Task 1 for the selection, the written repo, and its tests

**Testable:** Yes

**Certainty:** low — no precedent: the Go code of this repo starts no external program

**Blast radius:** high — destructive write: the commands rewrite files (`go mod tidy`, `gofmt -w .`), and a
wrong working folder rewrites the files of another project, such as the folder that the tool runs from

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 3: Set the starter options from their defaults and from given values

**Behavior:** `starter.yml` declares `options` with a `default` and a `pattern`. A selection takes a value
for each option from the given values, or else from the default, and a default can be a template of the
name. A template reads a value with `option "starter.<option>"`. An unknown option or a value that does not
match its pattern stops the selection before the first file is written.

**Acceptance Criteria:**
- [ ] An option with no given value takes its `default`. A default that contains a template renders with the
  name: `github.com/hpcsc/{{ .Name }}` gives `github.com/hpcsc/my-service` for `my-service`.
- [ ] A given value replaces the default.
- [ ] `{{ option "starter.<option>" }}` in the contents or the path of a template gives the value of that
  option.
- [ ] A given value for a name that is not an option of the starter stops the selection. The message names
  the unknown option and lists the options of the starter. No file is written.
- [ ] A value that does not match the `pattern` of its option stops the selection. The message names the
  option and its pattern. No file is written. A value that matches is accepted.
- [ ] An option without `pattern` accepts any text.
- [ ] `option` with a name that is not `starter.` and an option of the starter stops the write before the
  first file is written. The message names the template and the name.

**Affected Files/Modules:**
- `catalog/` — the options of a starter, the values of a selection, the `option` template function, and
  their unit tests

**Patterns to Follow:**
- Task 1 for the render and its tests
- `PROPOSAL.md:104-110` for `options` in `starter.yml`
- `PROPOSAL.md:254-273` for how an option gets its value
- `PROPOSAL.md:295-326` for `option "starter.module"`

**Testable:** Yes

**Certainty:** medium — follows the render of Task 1, but no instance there takes values from outside the
catalog, checks them against a pattern, or renders a default that is itself a template

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 4: Render the templates that match a delimiters glob with their own delimiters

**Behavior:** `starter.yml` declares `delimiters`: a glob over catalog paths, and a `left` and a `right`.
A template that matches a glob renders with those delimiters, so text that another tool reads as a
template, such as the `{{ }}` of Task and the `${{ }}` of GitHub Actions, is written as it is.

**Acceptance Criteria:**
- [ ] A template whose catalog path matches the `glob` of a `delimiters` entry renders with the `left` and
  the `right` of that entry. Text in it between `{{` and `}}`, for example `{{.CLI_ARGS}}` and
  `${{ secrets.GITHUB_TOKEN }}`, is written as it is.
- [ ] `**` in a glob matches any number of path segments, also none: `go/**/Taskfile*.yml` matches
  `go/Taskfile.yml` and `go/a/b/Taskfile.test.yml`, and does not match `web/Taskfile.yml`.
- [ ] A template that matches no glob renders with `{{` and `}}`.
- [ ] With other delimiters, `.Name`, `option` and the case functions give the same values as with `{{` and
  `}}`.

**Affected Files/Modules:**
- `catalog/` — the delimiters of a starter, the match of a template against a glob, and their unit tests
- `go.mod`, `go.sum` — only if the task takes a library for `**` globs

**Patterns to Follow:**
- Task 1 for the render and its tests
- `PROPOSAL.md:131-134` for `delimiters` in `starter.yml`
- `/Users/davidnguyen/repo-templates/templates/go-event-sourcing-service/scaffold.yml` (the `delimiters`
  section) for the two globs that the starter needs

**Testable:** Yes

**Certainty:** medium — follows the render of Task 1 with a choice of delimiters for each template, but no
instance here matches a glob with `**`, which `path.Match` does not support

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 5: Write `.tmpl` templates without the suffix, and scripts as executables

**Behavior:** Two rules for how a template becomes a file, so that the catalog can hold a Go module and
scripts. A template whose name ends in `.tmpl` is written without that suffix, so the catalog holds the
`go.mod` and the Go files of a starter where the Go tool of this repo neither treats them as a module nor
compiles them. A rendered file that starts with `#!` is written as an executable, because `go:embed` keeps
no file mode.

**Acceptance Criteria:**
- [ ] A template whose name ends in `.tmpl` is written without that suffix: `go/go.mod.tmpl` becomes
  `go.mod`, and `go/main.go.tmpl` becomes `main.go`. A template with another name keeps its name.
- [ ] A rendered file whose contents start with `#!` can be run by its owner. Other files cannot be run.
- [ ] Two templates that would be written to the same path, for example `go/main.go` and `go/main.go.tmpl`,
  stop the write before the first file is written, and the message names both templates.

**Affected Files/Modules:**
- `catalog/` — the name and the mode of a written file, and their unit tests

**Patterns to Follow:**
- Task 1 for the render, the write, and their tests

**Testable:** Yes

**Certainty:** low — the design is what the task decides: the proposal does not say how the catalog holds a
`go.mod` and Go templates, or how a script keeps its executable bit through `go:embed`

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 6: Port the go-event-sourcing-service starter, without the process manager, into the catalog in the binary

**Behavior:** The binary carries the catalog with `go:embed`. The catalog holds the starter
`go-event-sourcing-service`: its `starter.yml`, and the files of the scaffold template without the process
manager, with each scaffold value changed to `.Name` or a starter option. The token check stays as the
template has it, and Task 7 removes it.

**Acceptance Criteria:**
- [ ] `catalog/starters/go-event-sourcing-service/starter.yml` has a `summary`; the option `go-version` with
  the default `1.27.1` and the regex of the `GoVersion` question of `scaffold.yml` as its pattern; the option
  `module` with the default `github.com/hpcsc/{{ .Name }}`; the folder `go` with `path: .`, the
  `after_render` commands `go mod tidy` and then `gofmt -w .`, and the `test` command `task test:unit`; and
  `[[` and `]]` delimiters for the Taskfiles and for the files under `.github/`. It has no
  `extension_points`.
- [ ] The catalog in the binary finds `go-event-sourcing-service` with no catalog file on disk, from any
  working folder, and it holds the files whose names start with `.`.
- [ ] Written with the name `my-service` and no given values, the repo has `module
  github.com/hpcsc/my-service` and `go 1.27.1` in `go.mod`. Each import of a package of the repo starts with
  the module. `mise.toml`, `Dockerfile.api` and `Dockerfile.catchup` use the Go version. Each place where the
  template used the project name has `my-service`.
- [ ] Written with `module=example.com/acme/ledger` and `go-version=1.28.0`, the repo has those values in the
  same places, and no file contains `github.com/hpcsc/my-service`.
- [ ] Each Taskfile keeps the template syntax of Task, for example `{{.CLI_ARGS}}` and
  `{{ default "latest" .TAG }}`, and `.github/workflows/ci.yml` keeps `${{ secrets.GITHUB_TOKEN }}`.
- [ ] No written file contains a scaffold value (`.Scaffold.`, `.ProjectKebab`, `.ModulePath`) or the text
  `<no value>`.
- [ ] The written repo has none of the files that the `features` globs of `scaffold.yml` select for the
  process manager, and none of the edits that the hook of
  `global/go-event-sourcing-service-process-manager` makes to shared files. Each shared file has the content
  of its branch for `ProcessManager: false`.
- [ ] `hooks/pre-push` and each script under `scripts/` can be run in the written repo.
- [ ] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass with the starter in the
  catalog, and `git status --ignored catalog/` shows no file of the starter as ignored.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — new
- `catalog/starters/go-event-sourcing-service/files/go/` — the 160 files of the template that are not the
  process manager; `go.mod` and each `.go` file have the `.tmpl` suffix
- `catalog/` — the `go:embed` of the starters and the catalog in the binary
- `catalog/` unit tests — the starter written into a temporary folder, without its `after_render` commands

**Patterns to Follow:**
- `PROPOSAL.md:101-135` for the `starter.yml` of this starter
- `/Users/davidnguyen/repo-templates/templates/go-event-sourcing-service/scaffold.yml` for the questions, the
  `features` globs and the delimiters
- `/Users/davidnguyen/repo-templates/templates/go-event-sourcing-service/{{ .ProjectKebab }}/` for the files
- `/Users/davidnguyen/repo-templates/templates/global/go-event-sourcing-service-process-manager/hooks/post_scaffold:175-249`
  for the process-manager edits to shared files
- Tasks 3, 4 and 5 for the options, the delimiters and the `.tmpl` names

**Testable:** Yes

**Certainty:** medium — the scaffold template is the precedent for each file, but no instance covers the
change from scaffold values to `.Name` and `option`, or the `.tmpl` name of each Go file

**Blast radius:** low — none of the named categories; the port copies the token check of the template
without a change, and Task 7 owns its removal

**Verification:** `task test:unit` passes. Until Task 9 checks it automatically, also write the starter into
a scratch folder and run `go mod tidy`, `gofmt -l .` and `task test:unit` there.

**Depends on:** Task 3, Task 4, Task 5

---

### Task 7: Remove the shared-token check from the starter

**Behavior:** The API of the starter has no check of a credential of its own: each route accepts a request
with no `Authorization` header. The token files, the middleware, its config, and each mention of the token
in Taskfiles, compose files and docs are removed.

**Acceptance Criteria:**
- [ ] The written repo has no `internal/api/middleware/auth.go`, no `internal/api/middleware/auth_test.go`, no
  `internal/api/middleware/testdata/valid-token`, and no `local/api/token`.
- [ ] No written file reads, sets or sends the shared token: no `TokenPath`, no `TOKEN_PATH`, no
  `valid-token`, no `Authorization` header in the requests of `Taskfile.api.yml`, and no token volume for the
  `api` service of `docker-compose.local.yml`.
- [ ] The API registers each route with no middleware that reads a credential.
- [ ] No doc of the written repo says that a client of the API sends a token.
- [ ] The unit tests of the starter show that the token files and the token text are absent from a written
  repo, and `go build ./...`, `go vet ./...` and `task test:unit` of this repo pass.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/` — remove the middleware, its
  test and its testdata
- `catalog/starters/go-event-sourcing-service/files/go/local/api/token` — remove
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl` — register
  the routes with no middleware
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl` — remove `TokenPath`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/test/common.go.tmpl` — the remark about
  auth
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml`,
  `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml` — the header, the env and
  the volume
- `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.md`,
  `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.emod` — the client of the API
- `catalog/` unit tests of the starter

**Patterns to Follow:**
- Task 6: the starter's `internal/api/test/common.go.tmpl` registers each route with no middleware

**Testable:** Yes

**Certainty:** medium — the starter's own test router registers routes with no middleware, but no instance
decides whether the API keeps its split of public and protected routes with nothing between them, which is
where `api-auth` (US-003) adds its check

**Blast radius:** high — authentication: this removes the only credential check of the generated API, so a
mistake leaves a check that rejects requests, or a config or doc that claims a check the API does not make

**Verification:** `task test:unit` passes. Write the starter into a scratch folder and run `go mod tidy` and
`task test:unit` there.

**Depends on:** Task 6

---

### Task 8: Replace the sample command with `new`

**Behavior:** `feature-compose new <starter> <name>` selects the starter from the catalog in the binary,
takes the values of `--option`, writes the repo to `./<name>` or to `--output-dir`, and then runs the
`after_render` commands. It accepts `--no-prompt`. The greeting of the root command is removed.

**Acceptance Criteria:**
- [ ] `feature-compose new <starter> <name>` writes the repo to `./<name>` in the working folder, and
  `--output-dir <path>` writes it to that path.
- [ ] With fewer than two arguments, `new` stops with a message that names `<starter>` and `<name>`, and
  writes nothing.
- [ ] `--option <name>=<value>` can be given more than one time, and each sets one starter option. A value
  without `=` stops `new` before the first file is written, with a message that shows the form
  `<name>=<value>`.
- [ ] `new` accepts `--no-prompt`. With or without it, `new` asks no question, reads no input, and gives each
  option without a given value its default.
- [ ] An unknown starter, an unknown option, a value that does not match its pattern, and an output folder
  that has entries each stop `new` with an exit status other than 0. The output folder is not made, or it
  keeps its contents.
- [ ] After the last file is written, `new` runs the `after_render` commands. A command that fails ends `new`
  with an exit status other than 0 and the message of Task 2.
- [ ] From an empty folder, with no catalog files and no `SCAFFOLD_DIR`, `feature-compose new no-such-starter
  x` exits with a status other than 0 and lists `go-event-sourcing-service`, and `feature-compose new
  go-event-sourcing-service x --option go-version=1.20` names `go-version` and its pattern and makes no `./x`.
- [ ] The root command has no action of its own. The greeting and its end-to-end test are removed, and
  `feature-compose` with no command prints its commands and reads no input.

**Affected Files/Modules:**
- `internal/cmd/root.go` — remove the greeting action, and add `new` to the commands
- `internal/cmd/new.go` (new) — the `new` command, its flags, and the catalog it uses
- `internal/cmd/new_test.go` (new) — unit tests with a catalog built in memory and shell-builtin
  `after_render` commands
- `e2e/tests/cli.test.ts` — remove the greeting test
- `e2e/tests/new.test.ts` (new) — the error paths against the real binary

**Patterns to Follow:**
- `internal/cmd/update.go:21-32` for a command with flags
- `internal/cmd/adapter.go:28-37` for an action with two arguments
- `internal/cmd/root.go:25-46` for the list of commands
- `internal/cmd/adapter_test.go:14-20` for running a command with arguments in a test
- `e2e/tests/cli.test.ts:17-30` for an end-to-end test with `runCli`
- `e2e/tests/update.test.ts:91-100` for an end-to-end test that shows nothing changed

**Testable:** Yes

**Certainty:** high — repeats the command of `internal/cmd/update.go:21-32` and the two-argument action of
`internal/cmd/adapter.go:28-37`

**Blast radius:** low — none of the named categories; it calls the steps of Tasks 1, 2 and 3, which own the
refusal of a folder with entries and the folder of each command

**Verification:** `task test:unit` and `task test:e2e:local` pass

**Depends on:** Task 2, Task 6

---

### Task 9: Check end to end that a new repo passes `task test:unit` and `task --list`

**Behavior:** An end-to-end test runs the binary as the first criterion of the story says, and then runs
the unit tests and `task --list` of the new repo. The Docker image of the end-to-end tests has the tools
that this needs.

**Acceptance Criteria:**
- [ ] An end-to-end test runs `feature-compose new go-event-sourcing-service my-service --no-prompt` in an
  empty scratch folder. It exits with 0 and writes `./my-service`.
- [ ] In `./my-service`, `go.mod` declares `module github.com/hpcsc/my-service`, `go.sum` exists,
  `gofmt -l .` lists no file, `task --list` exits with 0, and `task test:unit` exits with 0.
- [ ] `task test:e2e` runs this test in Docker. The e2e image has on PATH Go at the version in the `go.mod` of
  this repo, Task, gotestsum, and a C compiler for the `-race` flag of the starter's `task test:unit`.
- [ ] `task test:e2e:local` runs this test on a machine that has Go, Task and gotestsum on PATH.
- [ ] Only this test has a timeout longer than the 30 seconds of `e2e/vitest.config.ts`, and its timeout is
  long enough for `go mod tidy` and a race build of the repo.
- [ ] `docs/e2e-tests.md` says that this test needs Go, Task, gotestsum and the network, and about how long it
  takes.

**Affected Files/Modules:**
- `e2e/tests/new.test.ts` — the full run
- `e2e/testUtils.ts` — a way to run a program other than the CLI in a folder, if the test needs one
- `e2e/Dockerfile` — Go, Task and gotestsum in the runner stage
- `docs/e2e-tests.md` — what the test needs

**Patterns to Follow:**
- `e2e/tests/cli.test.ts:17-30` for a test with `runCli`
- `e2e/testUtils.ts:36-53` for `runCli`
- `e2e/Dockerfile:1-24` for the two stages of the image
- `Taskfile.test.yml:11-23` for how `test:e2e` and `test:e2e:local` build and run the tests

**Testable:** Yes

**Certainty:** medium — follows the `runCli` tests of `e2e/tests/cli.test.ts:17-30`, but no instance runs a
second toolchain in the e2e image or a test that needs the network and several minutes

**Blast radius:** low — none of the named categories

**Verification:** `task test:e2e` and `task test:e2e:local` pass

**Depends on:** Task 7, Task 8

## Summary

**Total:** 9 tasks.

**Order.** Dependency first, then risk. Tasks 1 to 5 build the rules of the catalog on catalogs in memory, so
each rule has fast, offline tests before a real starter exists. Task 1 comes first because each later task
uses it, and it carries the first of the two high-risk rules: the write over a folder of a developer. Task 2
carries the second, the commands that rewrite files, and needs only Task 1, so it comes next. Tasks 3, 4 and
5 add the options, the delimiters and the `.tmpl` rule that the port needs. Task 6 ports the starter. Task 7
removes the token check as its own change, so the one authentication change of the story has its own
review. Task 8 connects the command, and Task 9 proves the full run.

**Coverage of the story:**

| Criterion of US-001 | Tasks |
| --- | --- |
| `new go-event-sourcing-service my-service --no-prompt` writes `./my-service`, and `task test:unit` passes there | 8 (the default output folder), 9 (the full run) |
| Runs from any folder, with no catalog files, no `SCAFFOLD_DIR` and no URL | 6 (the catalog in the binary), 8 (the test from an empty folder) |
| The name in the contents and the paths; the default of `module` is `github.com/hpcsc/my-service` | 1 (`.Name`), 3 (a default as a template), 6 (the places of the starter) |
| `--option module` and `--option go-version`; an unknown starter or option, or a value that does not match its pattern, stops before the first file, with the option and its pattern or the known names | 1 (the starter), 3 (the option and the pattern), 6 (the two options), 8 (the flag) |
| The Taskfiles keep the syntax of Task, and `task --list` runs | 4 (the delimiters), 6 (the globs of the starter), 9 (`task --list`) |
| `go mod tidy` and `gofmt -w .` run in the folder `go` after the render; a failure names the folder, the command and its output | 2 (the rule), 6 (the commands of the starter), 8 (the run), 9 (the real commands) |
| `--output-dir`, and a folder that has entries stops the tool with nothing written | 1 (the refusal), 8 (the flag) |
| Context: no process manager | 6 |
| Context: no credential check, so the API accepts each request | 7 |

No criterion of the story is deferred.

**Criteria that this breakdown adds, for the caller to overrule:**

- The `.tmpl` rule and the `#!` rule of Task 5. Neither is in the proposal. Both come from checks of
  `go:embed` in this breakdown: a nested `go.mod` blocks the embed, Go templates break `go build ./...`, and
  the embed loses the executable bit.
- A template that fails, a template outside the folders of `starter.yml`, and two templates with one path each
  stop the write before the first file (Tasks 1 and 5). This makes "writes nothing" true for errors that the
  story does not name.
- A second delimiters glob for `.github/` in the starter (Task 6), from `scaffold.yml`. The example in the
  proposal has only the Taskfile glob.

**Notes.**

- The build and the tests of the written starter are checked automatically only in Task 9. Tasks 6 and 7
  check them by hand, so Task 9 can still need a fix in a file of the starter.
- The `{{ .ModulePath }}` imports in `conflict_retry.go` of the scaffold template do not resolve today. The
  port maps them to the `module` option.
