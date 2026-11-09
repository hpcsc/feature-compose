# Tasks: Preview a Repo Before the Tool Writes It (US-010)

## Contents

1. Task 1: Preview the starter and the files with `new --dry-run`, and write nothing
2. Task 2: Show each feature of the preview with its source
3. Task 3: Show each option of the preview with its value and its source

## Story Reference

US-010, "Preview a repo before the tool writes it", in `user-stories/feature-compose.md:336-354` (Acceptance
Criteria and Context). The design is `PROPOSAL.md`: section 8 (the sources of the value of an option,
`PROPOSAL.md:263-270`), section 10 (rule 4, a need with the values that it sets, `PROPOSAL.md:337`; rule 7, one owner
for each file, `PROPOSAL.md:342`; rule 8, the order of the features), section 11 (the flag `--dry-run`,
`PROPOSAL.md:374`, and the steps of `new`), section 12 (the confirm step of the TUI shows the same output,
`PROPOSAL.md:412-413`) and section 13 (the selection: the starter, the features and the options of one repo).

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

Added by this breakdown:

- A check of the output folder in `--dry-run`. The dry run checks the selection and reads no output folder, so an
  output folder that has entries stops `new` but does not stop `--dry-run` (decision 3).
- The contents of the files in the preview. The preview gives the path and the owner of each file.
- The extension points with their items, and the commands of the folders, in the preview. The story names the
  starter, the features, the options and the files.
- A form of the preview for a script, such as JSON. The preview is text for a person, and `show --output json` is
  the output for a script (decision 7).
- A section of `README.md` about `new` and its other flags. The README gets a section about `--dry-run` only.

**Deferred:**

- The source `need` of the value of an option. In the catalog format of today a need sets no value: `needs` is a
  list of names, so each value comes from a flag or from its default (see "Needs set no value" below). US-007 needs
  the map form `needs: {api-auth: {method: oauth}}`, because its fifth criterion has the need of `cli-auth` set the
  `method` of `api-auth` (`user-stories/feature-compose.md:274-276`). US-007 adds the source `need` to the preview
  when it adds the values of a need. The breakdown of US-014 also deferred the map form to US-007
  (`tasks/completed/us-014-show-catalog.md:56-57`).
- The confirm step of the TUI of `new` (US-012), which shows the text of `Selection.Preview`, and the source of an
  answer that the TUI gives.

## Codebase Context

**The steps of `new`.** `newNewCommand` (`internal/cmd/new.go:13-65`) parses each `--option` into a value of the
starter (`internal/cmd/new.go:67-77`) and each `--feature` into the values of a feature
(`internal/cmd/new.go:79-102`; two different values of one option stop it). It loads the catalog, finds the starter
with `Catalog.Starter`, and calls `Starter.Select` with the name of the repo and the given values
(`internal/cmd/new.go:26-45`). Then it calls `Selection.Write` with the output folder (`--output-dir`, else
`./<name>`) and `Selection.RunAfterRender`. At the end it prints `<feature>: needed by <a, b>` for each entry of
`AddedByNeeds`, and `Made <output> from the starter <starter>.` (`internal/cmd/new.go:56-62`). `--no-prompt` is
accepted, and `new` reads no input in each case, because no TUI exists yet. `Run` prints an error on stderr and
returns 1 (`internal/cmd/root.go:18-25`, `internal/cmd/root.go:61-69`).

**Where `new` stops.** `Starter.Select` (`internal/catalog/starter.go:73-103`) stops on a name of the repo that is
not valid; on the values of the starter (`options.withDefaults`, `internal/catalog/option.go:49-75`: an unknown
option, a value that is not one of `values`, a value that does not match `pattern`, a default that fails to
render); on an unknown feature, a need that is not a feature of the starter, and a cycle of needs; on the values of
a feature; on a template outside the folders of the starter; and on the extension points. `Selection.Write`
(`internal/catalog/selection.go:38-62`) first refuses an output folder that has entries (`checkEmpty`,
`internal/catalog/selection.go:82-94`). Then it renders each template, its path and then its contents
(`template.render`, `internal/catalog/template.go:64-79`, which also stops a path that leads out of the repo), and
stops on two templates that write one path with a message that names both templates and their owners. Only then
does it write the files (`file.write`, `internal/catalog/file.go:15-24`). So each error of the render comes before
the first file, but the render is inside `Write`, and no code renders the templates without writing them.
`RunAfterRender` (`internal/catalog/selection.go:64-71`) runs the `after_render` commands of each folder in that
folder of the repo (`internal/catalog/folder.go`).

**The owner of a file.** Rule 7 gives each file one owner: the starter or one feature (`PROPOSAL.md:342`).
`readTemplates` gives each template the owner text `the starter` or `the feature <name>`
(`internal/catalog/template.go:23-27`), and the message of two templates with one path uses that text. The rendered
`file` (`internal/catalog/file.go:10-13`) holds only its path in the repo and its contents.

**What the selection keeps.** `Selection` (`internal/catalog/selection.go:19-28`) keeps the folders, the name of the
repo, the values of the starter and of each feature as maps, the templates (the starter first, then each feature
in the order of rule 8), the extension points, `addedByNeeds`, and the needs of each feature. It keeps no name of
the starter, no order of the features, and no record of which values a flag gave: `withDefaults` returns one map,
in which a given value and a default look the same. `Starter.Select` has each of these at hand: the name of the
starter, the selected features in the order of rule 8 (`inNeedsOrder`, `internal/catalog/starter.go:188-203`), and
the given values. `AddedByNeeds` (`internal/catalog/selection.go:30-36`, built by
`internal/catalog/starter.go:220-236`) gives each feature that a need added, in the order of rule 8, with the
selected features that need it in the order of their names. A feature that a flag gives is not in it, also when a
selected feature needs it.

**Needs set no value.** `needs` is a list of names: `FeatureDefinition.Needs` is a list of strings
(`internal/catalog/definitions.go:61`), and `feature.needs` is that list (`internal/catalog/feature.go:19`,
`internal/catalog/feature.go:63`). The map form that sets values, `needs: {api-auth: {method: oauth}}`
(`PROPOSAL.md:243`, rule 4 at `PROPOSAL.md:337`), does not parse into it. The catalog of this repo has one need list,
`cli-auth` with `needs: [cli, api-auth]`, and no value. So the value of an option comes from a flag or from its
default, and the source `need` cannot occur yet. The stories put the values of a need in later stories: US-007
(`user-stories/feature-compose.md:274-276`) stops when a flag and a need give different values, and US-006
(`user-stories/feature-compose.md:246-247`) checks a need that sets a value that its option does not have. US-010
depends on US-005 only, so this breakdown defers the source `need` (Deferred).

**The names of options.** A template reads an option of its own feature by its name, an option of a need as
`<need>.<option>`, and an option of the starter as `starter.<option>` (`internal/catalog/selection.go:110-136`,
`PROPOSAL.md` section 8). On the command line, `--option <name>=<value>` sets an option of the starter, and
`--feature <name>:<option>=<value>,...` sets options of a feature.

**Tests.** The unit tests have the tag `unit` and run with `task test:unit`. `TestSelection` builds a selection from
a catalog in `fstest.MapFS` (`internal/catalog/selection_test.go:23-44`), and has groups for the write, the run of
the commands, and `AddedByNeeds` (`internal/catalog/selection_test.go:925-975`). `TestStarter` checks the defaults
and the given values (`internal/catalog/starter_test.go:15-62`). `TestNewCommand` runs `new` on a catalog in
`fstest.MapFS` with `runNew`, whose writer is a buffer (`internal/cmd/new_test.go:21-64`). `starterFiles` takes an
`after_render` command, and its group `stop` checks for each error that `new` makes no output folder
(`internal/cmd/new_test.go:350-503`). The e2e tests in `e2e/tests/new.test.ts` run the built binary with `runCli`
(`e2e/testUtils.ts:62-69`), and a stop asserts that the scratch folder holds no entry
(`e2e/tests/new.test.ts:11-19`). They run on the host when Docker does not answer (`tasks/learnings.md:58-63`).

**Docs.** `README.md` has no section about `new` and its flags. "Configure the catalog" (`README.md:24-41`) names
`new`, and "Show the Catalog" (`README.md:43-117`) is the section of a command. `new --help` lists each flag with
its usage (`internal/cmd/new_test.go:339-348`).

**Words.** `owner` is the word of the code for the starter or the feature whose template writes a file
(`template.owner`). `declarer` is taken: the starter or the feature that declares an extension point or an option.
`given` is a feature or a value that a flag gives (the planned design of US-005). `summary` is taken: the sentence
of a `starter.yml` or a `feature.yml`. `needed by` is the text of `new` for the features that need a feature. No Go
code of the tool uses `preview` or `source`. `source` is the word of the story and of `PROPOSAL.md:263-270` for
where a value comes from. The generated CLI of `cli-auth` uses `Source` in the same sense for its key, in another
program.

**Decisions of this breakdown, for the caller to overrule:**

1. **`Selection.Preview` gives the preview as text, or an error.** `--dry-run` prints it, and the confirm step of
   US-012 shows the same text. The text is made in `internal/catalog`, where the owner of a file, the source of a
   value and the features that need a feature are decided, as `Combination.String` makes the arguments of `new`
   there (`internal/catalog/combination.go:15-33`). No exported type holds the parts of the preview: a person reads
   them, and no code does (architecture principle 10).
2. **`Selection.Write` and `Selection.Preview` render the templates through one step**, so each error of the render,
   two templates that write one path included, stops both with the same message. `Write` keeps its order: the check
   of the output folder, the render, and then the files.
3. **`--dry-run` does not read the output folder.** The third criterion of the story names selections, and the
   output folder is not part of a selection (`PROPOSAL.md` section 4). So in a folder with entries, `--dry-run`
   prints the preview, and `new` then stops. The other choice, a dry run that also refuses an output folder with
   entries, cannot preview a repo whose folder is in use.
4. **A feature that a `--feature` flag gives shows the source `command line`, and each other feature shows the
   features that need it**, as `AddedByNeeds` gives them and as `new` prints them (`api-auth: needed by cli-auth`).
   A feature that a flag gives and a selected feature needs shows `command line`. So one rule decides which features
   a need added, for `new` and for the preview.
5. **The preview gives the name of the repo with its starter, then the features, then the options, then the
   files.** The features come in the order of rule 8. The options come by the names with which a template reads
   them: `starter.<option>` in the order of the names, then `<feature>.<option>` for each feature in the order of
   rule 8, and the options of one feature in the order of the names. The files come in the order of their paths in
   the repo, each with its owner as the messages of the tool name it. So each order of the flags gives the same
   text.
6. **The source `need` is deferred to US-007** (Deferred).
7. **The preview is text for a person, not a contract.** `README.md` says so, and points a script to
   `show --output json`. So a change of its layout breaks no consumer outside this repo.
8. **`--dry-run` prints only the preview.** It prints no `needed by` lines of its own, because the preview lists the
   features with their sources, and no `Made` line, because it makes no repo.

## Tasks

### Task 1: Preview the starter and the files with `new --dry-run`, and write nothing

**Behavior:** `new <starter> <name> --dry-run` takes the same steps as `new` up to the render of the templates.
Then it prints the preview: the name of the repo with its starter, and each file of the repo by its path in the
repo, with its owner. It writes no file, makes no folder, and runs no folder command. Each error that stops `new`
before its first file stops `--dry-run` with the same message, before it prints anything; decision 3 names the one
exception. `Selection.Preview` gives the text (decision 1), and `Selection.Write` renders the templates through the
same step (decision 2). Task 2 and Task 3 add the features and the options to the preview.

**Acceptance Criteria:**
- [x] `new <starter> <name> --dry-run` exits with status 0 and prints the name of the repo and its starter, and then
  each file of the repo by its path in the repo, with its owner: `the starter` or `the feature <name>`, the words
  with which the messages of the tool name an owner (`internal/catalog/template.go:23-27`).
- [x] The paths of the preview are the paths of the files that `new` writes for the same arguments, each one time,
  in the order of the paths (decision 5): for example a path that holds the name of the repo shows the rendered
  name, and a name that ends in `.tmpl` shows without the suffix. Each order of the flags gives the same text.
- [x] `--dry-run` writes no file and runs no folder command. After the run, the output folder (`./<name>`, or the
  path of `--output-dir`) does not exist. With `--output-dir` set to a folder that exists and is empty, and a
  starter whose `after_render` command makes a file, the folder stays empty.
- [x] Each error that stops `new` before it writes its first file stops `--dry-run` with the same message and a
  status other than 0, and `--dry-run` then prints nothing on stdout. These are the errors of each step of `new`
  before the write (`internal/cmd/new.go:26-45`) — the parse of `--option` and `--feature`, the load of the catalog,
  the starter, and the selection — and the errors of the render of the templates
  (`internal/catalog/selection.go:42-55`), two templates that write one path included. The one exception is the
  refusal of an output folder that has entries (decision 3).
- [x] `Selection.Preview` gives the text of the preview for a selection, and each error of the render of the
  templates stops it with the message that `Selection.Write` gives for the same selection.
- [x] `Selection.Write` writes the same files, and stops with the same messages, as before: the tests of the group
  `write` of `TestSelection`, of the group `select` of `TestStarter`, and of the groups `write` and `stop` of
  `TestNewCommand` pass with their expected values unchanged.
- [x] `new --help` lists `--dry-run` with its usage.
- [x] `README.md` has a section "Preview a Repo". It tells that `new <starter> <name> --dry-run` prints the preview,
  writes no file, runs no command, and stops with the message of `new`; that the dry run does not check the output
  folder (decision 3); and that the preview is text for a person, so a script reads `show --output json`
  (decision 7).
- [x] `e2e/tests/new.test.ts`: `new go-event-sourcing-service my-service --feature cli --dry-run` exits with status
  0. Its stdout names `go-event-sourcing-service`, `go.mod` with the starter as its owner, and
  `cmd/my-service/main.go` with the feature `cli` as its owner, and the scratch folder holds no entry after the run.
  An unknown feature, and a `go-version` that does not match its pattern, give the same stderr with and without
  `--dry-run`, a status other than 0, and no entry in the scratch folder.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — the name of the starter in the selection; one render step that `Write` and
  `Preview` share; `Preview`
- `internal/catalog/file.go` — the owner of a rendered file
- `internal/catalog/template.go` — the render gives the file the owner of its template
- `internal/catalog/starter.go` — `Select` gives the selection the name of the starter
- `internal/catalog/selection_test.go` — a group `preview` in `TestSelection`
- `internal/cmd/new.go` — the flag `--dry-run`, which prints the preview and stops before the write
- `internal/cmd/new_test.go` — a group `dry run` in `TestNewCommand`, and `--dry-run` in the group `help`
- `README.md` — the section "Preview a Repo"
- `e2e/tests/new.test.ts` — the preview of a repo, and the stops with and without `--dry-run`

**Patterns to Follow:**
- `internal/catalog/selection.go:38-62` for the render of each template and the check of two templates with one
  path before the first file
- `internal/catalog/template.go:23-27` and `internal/catalog/template.go:64-79` for the owner text and the render of
  a path and its contents
- `internal/catalog/combination.go:15-33` for a text in the catalog that a person reads
- `internal/cmd/new.go:13-65` for the flags and the steps of `new`
- `internal/catalog/selection_test.go:23-44`, `internal/catalog/selection_test.go:171-186` and
  `internal/catalog/selection_test.go:266-306` for a selection of a catalog in `fstest.MapFS`, two templates with
  one path, and the errors of the render
- `internal/cmd/new_test.go:21-52` for `starterFiles`, `loaded` and `runNew`
- `internal/cmd/new_test.go:285-296` for an `after_render` command that makes a file
- `internal/cmd/new_test.go:339-348` for the help of a flag
- `internal/cmd/new_test.go:350-503` for each stop of `new`, with no output folder after it
- `e2e/tests/new.test.ts:11-19`, `e2e/tests/new.test.ts:32-41` and `e2e/tests/new.test.ts:163-171` for an e2e stop
  with an empty scratch folder, and `e2e/testUtils.ts:62-69` for `runCli`
- `README.md:43-117` for the section of a command
- `tasks/learnings.md:3-8` for a test that runs a shell command, which starts in a temporary folder
- `tasks/learnings.md:182-187` for a test of the preview in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:58-63` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `Selection.Write` (`internal/catalog/selection.go:38-62`) already renders each template and
checks the paths before its first file, and `newNewCommand` (`internal/cmd/new.go:13-65`) is the command; but no
code renders the templates without writing them, and no command prints a text built from a selection other than the
lines of the needs

**Blast radius:** high — a destructive or irreversible write: this task decides whether `new` writes the repo and
runs the `after_render` commands, of which `gofmt -w .` rewrites files in place, and it changes `Selection.Write`,
whose check keeps `new` out of an output folder that has entries

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant
on the changed lines of `internal/catalog` and `internal/cmd`. Apply one break at a time, read its diff, run the
tests, and put the file back with `git checkout --`: let `new` go on to the write after it prints the preview; drop
the check of the output folder from `Write`; give each file of the preview the starter as its owner. Each break
makes a test fail. Run `e2e/tests/new.test.ts` on the host (`tasks/learnings.md:58-63`). In an empty temporary
folder, run the built binary with `new go-event-sourcing-service my-service --feature cli-auth --dry-run`: it
prints the preview, and the folder stays empty.

**Depends on:** None

---

### Task 2: Show each feature of the preview with its source

**Behavior:** The preview, which `Selection.Preview` gives and `--dry-run` prints, lists each feature of the
selection after the starter, in the order of rule 8. A feature that a `--feature` flag gives shows the source
`command line`. A feature that a need added shows `needed by` and the selected features that need it, as the line
that `new` prints for it (decision 4).

**Acceptance Criteria:**
- [ ] The preview lists each feature of the selection one time, after its needs and otherwise in the order of the
  names (rule 8), which is the order of `AddedByNeeds` and of the items of an extension point.
- [ ] A feature that a `--feature` flag gives shows the source `command line`, also when a selected feature needs
  it. A feature that only a need added shows `needed by` with each selected feature that needs it, in the order of
  their names, as the line that `new` prints for it, for example `api-auth: needed by cli-auth`.
- [ ] A selection with no feature shows that the repo has no feature.
- [ ] The two orders of the flags of a feature and its need give the same text.
- [ ] The section "Preview a Repo" of `README.md` names the features and their two sources.
- [ ] `e2e/tests/new.test.ts`: `new go-event-sourcing-service my-service --feature cli-auth --dry-run` prints
  `api-auth: needed by cli-auth`, `cli: needed by cli-auth`, and `cli-auth` with the source `command line`, and the
  scratch folder holds no entry after the run.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — the features of the selection in the order of rule 8, the features that a flag
  gives, and the features in the preview
- `internal/catalog/starter.go` — `Select` gives the selection the order of the features and the given features
- `internal/catalog/selection_test.go` — the features in the group `preview` of `TestSelection`
- `internal/cmd/new_test.go` — the features in the group `dry run` of `TestNewCommand`
- `README.md` — the features in "Preview a Repo"
- `e2e/tests/new.test.ts` — the features in the preview of a repo with `cli-auth`

**Patterns to Follow:**
- `internal/catalog/selection.go:30-36` and `internal/catalog/starter.go:220-236` for the features that a need
  added, with the features that need them
- `internal/catalog/starter.go:188-203` for the order of rule 8
- `internal/cmd/new.go:56-62` for the line that `new` prints for a feature that a need added
- `internal/catalog/selection_test.go:925-975` for a selection with needs in `TestSelection`
- `internal/cmd/new_test.go:186-237` for a catalog with needs, and the two orders of the flags
- `e2e/tests/new.test.ts:112-125` for the lines that `new` prints with `cli-auth`
- `task:1` for the preview, `Selection.Preview`, and the group `dry run`
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match
- `tasks/learnings.md:147-152` for a walk over the needs, as recursion and not as a queue
- `tasks/learnings.md:182-187` for a test of the preview in `internal/catalog`

**Testable:** Yes

**Certainty:** high — `Starter.addedByNeeds` (`internal/catalog/starter.go:220-236`) finds, in the order of rule 8,
the features that need each feature that a need added, and `new` prints them (`internal/cmd/new.go:56-62`); the
preview lists the same features, with the given features among them

**Blast radius:** low — none of the named categories; the preview is text for a person, and this task changes no
file that `new` writes

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/catalog` and `internal/cmd`. Run `e2e/tests/new.test.ts` on the host.

**Depends on:** Task 1

---

### Task 3: Show each option of the preview with its value and its source

**Behavior:** The preview lists each option after the features: each option of the starter as `starter.<option>`,
and then each option of each feature as `<feature>.<option>`. Each option shows the value that the templates read,
and its source: `command line` for a value that a flag gives, and `default` for each other value. The source `need`
does not occur, because a need sets no value in the catalog format of today (Deferred).

**Acceptance Criteria:**
- [ ] The preview lists each option of the starter by the name `starter.<option>`, in the order of the names, and
  then each option of each feature by the name `<feature>.<option>`, with the features in the order of Task 2 and
  the options of one feature in the order of the names. These are the names with which a template reads the
  options (`internal/catalog/selection.go:110-136`).
- [ ] Each option shows the value that the templates of the repo read. A default shows rendered with the name of the
  repo, for example `github.com/hpcsc/my-service` for the default of `module`.
- [ ] The source of a value is `command line` when `--option <name>=<value>` or
  `--feature <name>:<option>=<value>` gives it, also when the value equals the default, and `default` for each other
  value. Each option of a feature that a need added shows `default`.
- [ ] A starter or a feature with no option adds no option to the preview, and a selection with no option shows
  that the repo has no option.
- [ ] The section "Preview a Repo" of `README.md` names the options, their values, and their two sources.
- [ ] `e2e/tests/new.test.ts`:
  `new go-event-sourcing-service my-service --option go-version=1.28 --feature cli-auth --dry-run` prints
  `starter.go-version` with `1.28` and `command line`, `starter.module` with `github.com/hpcsc/my-service` and
  `default`, and `api-auth.method` with `api-key` and `default`, and the scratch folder holds no entry after the
  run.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — the values that the caller gave, and the options in the preview
- `internal/catalog/starter.go` — `Select` gives the selection the given values of the starter and of each feature
- `internal/catalog/option.go` — only if the choice between the given value and the default also gives its source
- `internal/catalog/selection_test.go` — the options in the group `preview` of `TestSelection`
- `internal/cmd/new_test.go` — the options in the group `dry run` of `TestNewCommand`
- `README.md` — the options in "Preview a Repo"
- `e2e/tests/new.test.ts` — the options in the preview of a repo with `cli-auth`

**Patterns to Follow:**
- `internal/catalog/option.go:49-75` for the choice between the given value and the default, rendered with the name
- `internal/catalog/selection.go:110-136` for the names `starter.<option>` and `<feature>.<option>`
- `internal/catalog/starter.go:73-103` for the given values in `Select`
- `internal/catalog/starter_test.go:15-62` for a default and a given value of the starter
- `internal/cmd/new_test.go:104-135` for the values of the options of a feature on the command line
- `e2e/tests/new.test.ts:32-41` for a value of `go-version` on the command line
- `task:2` for the features of the preview in the order of rule 8
- `task:1` for the preview and the group `dry run`
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match
- `tasks/learnings.md:182-187` for a test of the preview in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `options.withDefaults` (`internal/catalog/option.go:49-75`) chooses between the given value
and the default, and `Selection.option` (`internal/catalog/selection.go:110-136`) names the options as
`starter.<option>` and `<need>.<option>`; but no code keeps which of the two gave a value after the choice

**Blast radius:** low — none of the named categories; the preview is text for a person, and the values that the
templates read do not change

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/catalog` and `internal/cmd`. Run `e2e/tests/new.test.ts` on the host.

**Depends on:** Task 2

## Summary

- **Tasks:** 3.
- **Order:** risk first, and then by dependency. Task 1 carries the risk of the story: it decides whether `new`
  writes and runs commands, changes `Selection.Write`, and makes the dry run stop with the messages of `new`. Task 2
  and Task 3 then add parts of the text, and Task 3 lists the options of the features in the order that Task 2
  gives.
- **Assessment:** no task is low certainty. The story uses again the render of `Selection.Write`, the walk of the
  needs, and the defaults of the options, and it adds no dependency, integration or algorithm that the repo does not
  have. Only Task 1 has a high blast radius.
- **Criteria of the story:**
  - "prints the starter, each feature with the feature that needs it, each option with its value and its source
    (command line, need, or default), and the path of each file with its owner": the starter and the files in Task
    1, the features in Task 2, the options with the sources `command line` and `default` in Task 3. The source
    `need` is deferred to US-007, because a need sets no value in the catalog format of today (Deferred).
  - "writes no file and runs no folder command. The output folder does not exist after the run": Task 1.
  - "A selection that `new` stops on ... stops `--dry-run` with the same message": Task 1, for each step of `new`
    before its first file. Task 2 and Task 3 add no step that can fail. The refusal of an output folder with
    entries is not a selection, and stays out of the dry run (decision 3).
  - Context, "The confirm step of the TUI (US-012) shows the same output": `Selection.Preview` gives that output as
    text (decision 1); the confirm step itself is US-012.
