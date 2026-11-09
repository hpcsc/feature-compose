# Tasks: Show the Catalog (US-014)

## Contents

1. Task 1: Read the definitions of the catalog, and print them as JSON with `show`
2. Task 2: Open the TUI of `show` in a terminal, with the list of the starters, its filter, and the view of a starter
3. Task 3: Open the view of a feature from the view of a starter

## Story Reference

US-014, "Show the catalog", in `user-stories/feature-compose.md:436-473` (Acceptance Criteria and Context). The
design is `PROPOSAL.md`: section 6 (the fields of `starter.yml`, `PROPOSAL.md:137-147`), section 7 (the fields of
`feature.yml`, `PROPOSAL.md:239-249`), section 11 (the two forms of `show`, `PROPOSAL.md:361-362` and
`PROPOSAL.md:385-393`), section 12 (no TUI without a terminal, `PROPOSAL.md:425-427`, and huh 1.0.0,
`PROPOSAL.md:429-437`) and decisions D8 and D12 (`PROPOSAL.md:545`, `PROPOSAL.md:549`).

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

- A text form of `show`, such as a table of the starters and the extension points. `--output` takes only `json`
  (decision 9).
- `files`, `delimiters` and the `folders` of a feature in the JSON and in the TUI. `files` and `delimiters` tell how
  the tool renders, not what a person selects, and the tool reads no `folders` of a feature (rule 9 of the proposal;
  no feature adds a folder).
- A check of the catalog in `show`. `show` reads only `starter.yml` and `feature.yml` (decision 7). It shows a
  feature whose `targets` name a starter that does not exist, and a need that is not a feature, with no message;
  `check` (US-006, US-009) owns these checks.
- A filter in any list other than the list of the starters. The story names the filter for that list only.
- A key that opens the view of a need from the view of a feature. The view of a feature names its needs.
- A widget that the TUI of `show` and the TUI of `new` share. The two TUIs share the definitions of the catalog,
  which give the starters with their summaries and the features of one starter (decision 6).
- The mouse, colour themes, and a help screen beyond the key hints that the list shows.

**Deferred:**

- A view that is taller than the terminal. The TUI draws each view whole and does not scroll it, so a terminal
  shorter than the view of a starter cuts lines of it (decision 12). The e2e terminal has 160 columns and 40 rows,
  and the view of `go-event-sourcing-service` fits in it. Scrolling needs keys that the story does not name.
- `when` of an option, and the values that a need sets in the map form `needs: {api-auth: {method: oauth}}`
  (US-007). Here `needs` is a list of names in the JSON. US-007 adds fields for both, and adds no rename.
- The TUI of `new` (US-012). It reads `Catalog.Definitions` for its lists, and adds huh 1.0.0 on the versions of
  decision 1.

## Codebase Context

**The catalog today.** `Catalog.Starter` checks the name against `starterNames`, the folders under `starters/` in
the order of their names, and stops an unknown name with `unknown starter "<name>": the catalog has <names>`
(`internal/catalog/catalog.go:19-32`, `internal/catalog/catalog.go:74-86`). `featuresTargeting` reads each folder
under `features/`, skips a file, and gives no features when `features/` does not exist
(`internal/catalog/catalog.go:50-72`). `readStarter` parses `starter.yml` into an anonymous struct named
`definition` (`internal/catalog/starter.go:34-55`): it reads the options, the folders with `path`, `after_render`
and `test`, the delimiters and the extension points, and keeps of each extension point only its item fields and a
declarer text such as `the starter go-event-sourcing-service` (`internal/catalog/extension_point.go:10-15`). It
reads no `summary`, and drops the `summary` and the `file` of each extension point. `readFeatureFor` parses
`feature.yml` the same way (`internal/catalog/feature.go:31-53`), returns nothing for a feature whose `targets` do
not name the starter, and keeps no `targets` and no `summary` (`internal/catalog/feature.go:24-92`). It also reads
each template of the feature. An `option` keeps its default, its compiled pattern and its values as `accepted`
(`internal/catalog/option.go:11-16`). Nothing exported lists the starters or the features, or gives a summary.
`Combination` and `Need` are the exported structs of plain values (`internal/catalog/combination.go:9-13`,
`internal/catalog/need.go:3-6`).

**The commands.** `newCommand` lists `new`, `version` and `update` (`internal/cmd/root.go:26-38`). `new` takes the
loader `catalogInConfig`, which reads the config file and opens its catalog folder, so `version` and `update` need
no config file (`internal/cmd/root.go:51-57`, `internal/cmd/new.go:13-65`). `Run` prints an error in red and
returns 1. `isTerminal` checks a writer with `go-isatty`; `update` is its only caller, and the only user of
`go-isatty` (`internal/cmd/update.go:90-93`). `adapter.go` has helpers for one and two arguments, and none for
zero or one (`internal/cmd/adapter.go:17-26`). The tests are in `package cmd` with the tag `unit`: `TestNewCommand`
runs `new` on catalogs in `fstest.MapFS` through `loaded` and `runNew` (`internal/cmd/new_test.go:40-50`), and
`TestRootCommand` runs the root command with a config file and a catalog folder on disk under `XDG_CONFIG_HOME`
(`internal/cmd/root_test.go:19-36`). A test gives the command a `bytes.Buffer` as its writer, so in a unit test
stdout is never a terminal.

**The TUI libraries.** go.mod has no TUI library. huh v1.0.0 (D12) requires bubbletea v1.3.6 and a pseudo-version
of bubbles v0.21.1; bubbles v1.0.0 requires bubbletea v1.3.10 and lipgloss v1.1.0. A scratch module with huh
v1.0.0, bubbletea v1.3.10, bubbles v1.0.0 and lipgloss v1.1.0 builds with Go 1.27.1. The list of bubbles v1.0.0
has these default keys: `/` starts its filter, `k` and up, `j` and down move, `h`, left, pgup, `b` and `u` go to
the previous page, `l`, right, pgdown, `f` and `d` go to the next page, and `q` and Esc quit. So `j` and `k` already
fit the story, and `h`, `l` and Esc do not. The filter of the list matches the text that each item gives for the
filter, with fuzzy matching.

**The e2e tests.** `runCli` gives the CLI pipes, so stdout is not a terminal (`e2e/testUtils.ts:62-69`). `openCli`
runs the CLI in a pseudo-terminal of 160 columns and 40 rows, and the screen ends with `EXIT:<status>`
(`e2e/testUtils.ts:71-90`); no test uses it yet. `docs/e2e-tests.md:31-55` tells how screen tests work and why
`openCli` turns off new line mode for a Bubble Tea program, and `docs/e2e-tests.md:177-192` tells how to write one.
Both helpers give the CLI a config file that names `CATALOG_DIR`. The CI job `e2e` runs each file in `e2e/tests/`
in the image of `e2e/Dockerfile`, which copies `catalog/` to `/catalog`.

**The catalog in this repo.** One starter, `go-event-sourcing-service`, with a summary, the options `go-version`
and `module` (a pattern each, no values), the folder `go` (`after_render: [go mod tidy, gofmt -w .]`,
`test: [task test:unit]`), and the extension points `api.middleware` and `api.config`. Three features: `api-auth`
(the option `method` with `values: [api-key]`, adds items to `api.middleware` and `api.config`), `cli` (declares
`cli.commands` with the item field `code`), and `cli-auth` (`needs: [cli, api-auth]`, adds an item to
`cli.commands`). So the view of the starter lists three extension points, one of them declared by `cli`, and three
features.

**Docs.** "Configure the catalog" in `README.md:24-40` says that `new` reads the config file, and that `version`
and `update` do not. No doc names the output of a command field by field. `internal/catalog/catalog_test.go:525-538`
is a test that reads a doc and checks the names in it.

**Words.** `definition` is the word of `readStarter` and `readFeatureFor` for what `starter.yml` and `feature.yml`
declare (`definitionPath`, `definitionYAML`, `definition`). `declarer` is the word for the starter or the feature
that declares an extension point or an option (`extensionPoint.declarer`, `options.declarer`). `owner` is taken:
the starter or the feature that writes a file (`template.owner`). `Need` is taken: a feature that a selection holds
because of needs; the needs of one feature are `needs`, as in `feature.needs`. `output` is the word of `new` for
the folder of the repo (`--output-dir`), and the story names the flag of `show` `--output`, so the code of `show`
calls its value the format. `view`, `filter` and `browser` are not used in the Go code.

**Decisions of this breakdown, for the caller to overrule:**

1. **The TUI uses bubbletea v1.3.10, with the list of bubbles v1.0.0, and lipgloss v1.1.0**, at these exact
   versions. They are the versions that huh 1.0.0 builds on (D12), so US-012 adds huh with no second major version
   of bubbletea in the binary; the scratch build above checks it. Each is more than two weeks old: bubbletea v1.3.10
   is of 2025-09-17, bubbles v1.0.0 of 2026-02-09, lipgloss v1.1.0 of 2025-03-12, and huh v1.0.0 of 2026-02-23. The
   other choice is the v2 line under `charm.land`: bubbletea v2.1.0 (2026-10-08), bubbles v2.2.1 (2026-08-24),
   lipgloss v2.0.6 (2026-08-11) and huh v2.0.3 (2026-03-10). It is newer and maintained, but it changes D12 to huh
   v2. bubbletea v2.1.1 (2026-10-28) is less than two weeks old.
2. **`/` starts the filter**, as in vim and as in the list of bubbles. The story says "Typing filters the list by
   name", but `h`, `j`, `k`, `l` and `q` are keys of the TUI, and the names hold these letters (`cli`, `chi-api`).
   While the filter is typed, each key goes to the filter, also these letters. Enter ends the typing and keeps the
   filtered list, where the keys work again. Esc clears the filter. The filter matches the name, not the summary.
   The other choice, a filter that takes each key except `h`, `j`, `k`, `l` and `q`, cannot filter for `cli`.
3. **`show` opens the TUI only when stdin and stdout are both terminals**, and else prints JSON. The story names
   stdout. Stdin follows the rule of `new` (`PROPOSAL.md:425-427`): with no keyboard, a TUI waits.
4. **The JSON has the keys of `starter.yml` and `feature.yml`.** Its top level has `starters` and `features`, each
   an object by name. A starter has `summary`, `folders`, `options` and `extension_points`. A feature has
   `summary`, `targets`, `needs`, `options`, `extension_points` and `add_to`. A folder has `path`, `after_render` and
   `test`; an option has `default`, `pattern` and `values`; an extension point has `summary`, `file` and `item`;
   `add_to` gives, for each extension point, the items as the `feature.yml` writes them. Each object has each of
   its fields: a list with no entry is `[]`, an object with no entry is `{}`, and a text with no value is `""`. The
   default of an option is its text in the file, for example `github.com/hpcsc/{{ .Name }}`, because `show` has no
   repo name. Each list keeps the order of the file. `show <starter>` gives the same shape, with one starter and the
   features whose `targets` name it. The JSON is indented by two spaces. A catalog author and a script then read
   the same names.
5. **In the JSON, the extension points of a starter are only those that its `starter.yml` declares**, and an
   extension point that a feature declares is under that feature. The view of a starter in the TUI lists both, each
   with its declarer, as the story says. A script knows the declarer from the place of the extension point, with no
   text to parse.
6. **`Catalog.Definitions` gives a `Definitions` value, which both forms of `show` read.** `definition` is already
   the word of the code for what `starter.yml` and `feature.yml` declare, and the types carry the JSON names of
   decision 4. The TUI of `new` (US-012) reads the same definitions for its lists of starters and features with
   their summaries. That is the share that the Context of the story names: a huh form and the list of `show` are two
   widgets, so the two TUIs share the data and not a widget.
7. **`show` reads only `starter.yml` and `feature.yml`**, and no template, so a mistake in a template does not stop
   `show`.
8. **`README.md` names each field of the JSON in a table, by its path**, for example
   `starters.<name>.folders.<name>.after_render`. One test compares the paths of the JSON of a catalog in which each
   field has a value with the paths of that table. A second test asserts the full JSON of a small catalog with
   literal values.
9. **`--output` takes only `json`**: another value stops `show` and names `json`. `show` takes at most one argument.
10. **At the list of the starters, `h` does nothing, and Esc clears the filter.** `show <starter>` opens the view of
    that starter over the list of all starters, so `h` goes back to that list. ctrl+c stops from each view, also
    while the filter is typed.
11. **The TUI is the package `internal/browser`**, and `browser.Model` is its bubbletea model, as `list.Model` is in
    bubbles. Tests send keys to it and read its `View`, which is the text on the screen.
12. **A view is drawn whole, and not scrolled** (Deferred).
13. **The lists of the TUI are in the order of the names.** The commands, the values, the targets, the needs and the
    item fields keep the order of the file.

## Tasks

### Task 1: Read the definitions of the catalog, and print them as JSON with `show`

**Behavior:** `Catalog.Definitions` reads each `starter.yml` and each `feature.yml` of the catalog, and gives what
they declare, by the name of each starter and feature. `Definitions.ForStarter` keeps one starter and the features
whose `targets` name it. The new command `show` prints the definitions of the catalog folder that the config file
names as JSON (decision 4), and `show <starter>` prints that starter and the features that target it. An unknown
starter stops `show`, and the message lists the starters. `README.md` names each field of the JSON, and a test
fails when a field changes its name or goes away. In this task `show` prints JSON also in a terminal; Task 2 adds
the TUI.

**Acceptance Criteria:**
- [x] `show --output json` prints one JSON document with the fields of decision 4, for the catalog that it loads. A
  test in `TestShowCommand` runs `show --output json` on a catalog in `fstest.MapFS` with two starters and two
  features, in which each field has a value, and asserts the whole JSON against literal expected values, so a field
  that changes its name or goes away fails it.
- [x] Each object has each of its fields: a list with no entry is `[]`, an object with no entry is `{}`, and a text
  with no value is `""`, never `null` and never absent. The default of an option is its text in the file, not
  rendered. The commands, values, targets, needs and item fields keep the order of the file.
- [x] `features` holds each feature of the catalog, also one whose `targets` name no starter of the catalog. A file
  directly in `features/` is not a feature. A catalog with no `features/` folder gives `"features": {}`.
- [x] `show <starter> --output json` prints the same shape with only that starter in `starters` and only the
  features whose `targets` name it in `features`. `Definitions.ForStarter` gives the same starter and features.
- [x] An unknown starter stops `show`, with or without `--output json`, before it prints anything on stdout. The
  message lists the starters of the catalog, as the message of `new` does. A second argument stops `show`, and the
  message shows the form `show [<starter>]`. An `--output` value other than `json` stops `show`, and the message
  names `json` (decision 9).
- [x] With no flag, and a stdout that is not a terminal, `show` and `show <starter>` print the same JSON as with
  `--output json`.
- [x] A `starter.yml` or a `feature.yml` that is not valid YAML stops `show`, and the message names the file. A read
  of the catalog that fails stops `show` with its cause. A catalog that cannot load, for example with no config
  file, stops `show` with its error.
- [x] `show` reads no template: a template that is not in a folder of its starter does not stop `show`
  (decision 7).
- [x] `show` reads the catalog folder that the config file names: a test writes a config file and a catalog folder,
  adds a starter folder and a feature folder to the catalog, and `show` lists both with no other change. `version`
  and `update` still need no config file. The root command lists `show` with its usage.
- [x] `README.md` has a section "Show the catalog", which tells the two forms, `--output json`, and the rule for a
  stdout that is not a terminal, and names each field of the JSON in a table by its path, with its meaning
  (decision 8). "Configure the catalog" says that `show` also reads the config file.
- [x] A test compares the field paths of the JSON of a catalog in which each field has a value with the paths in the
  table of `README.md`, and fails when a path is in only one of them.
- [x] `e2e/tests/show.test.ts` runs `show` with `runCli`, whose stdout is not a terminal, and parses its output as
  JSON: `starters` holds `go-event-sourcing-service` with its summary and the folder `go`, whose `after_render`
  starts with `go mod tidy`, and `features` holds `cli-auth` with the needs `cli` and `api-auth`, and `cli` with the
  extension point `cli.commands`. `show go-event-sourcing-service --output json` gives the same starter and
  features. `show no-such-starter` exits with a status other than 0, its output names `go-event-sourcing-service`,
  and the working folder stays empty.

**Affected Files/Modules:**
- `internal/catalog/definitions.go` — new: `Definitions`, the definition types of a starter, a feature, a folder,
  an option and an extension point, and `ForStarter`
- `internal/catalog/catalog.go` — `Definitions`
- `internal/catalog/starter.go`, `internal/catalog/feature.go` — the parse of `starter.yml` and `feature.yml`, if
  the definitions share it
- `internal/catalog/catalog_test.go` — a group `definitions` in `TestCatalog`
- `internal/catalog/definitions_test.go` — new: `TestDefinitions`
- `internal/cmd/show.go` — new: the command `show`
- `internal/cmd/show_test.go` — new: `TestShowCommand`
- `internal/cmd/root.go` — `show` in the commands, with the loader of the config file
- `internal/cmd/root_test.go` — `show` reads the catalog of the config file
- `README.md` — the section "Show the catalog", and "Configure the catalog"
- `e2e/tests/show.test.ts` — new

**Patterns to Follow:**
- `internal/cmd/new.go:13-65` for a command that takes the loader of the catalog
- `internal/cmd/root.go:26-38` and `internal/cmd/root.go:51-57` for the commands and the loader
- `internal/cmd/adapter.go:17-26` for the stop when an argument is missing
- `internal/cmd/new_test.go:40-50` for a catalog in `fstest.MapFS` and a run of a command, and
  `internal/cmd/new_test.go:468-480` for a catalog that cannot load
- `internal/cmd/root_test.go:19-36` for a config file and a catalog folder on disk
- `internal/catalog/catalog.go:19-32`, `internal/catalog/catalog.go:50-72` and `internal/catalog/catalog.go:74-86`
  for the starters, the features, and the message of an unknown starter
- `internal/catalog/starter.go:34-55` and `internal/catalog/feature.go:31-53` for the parse of the two files
- `internal/catalog/combination.go:9-13` for an exported struct of plain values
- `internal/catalog/catalog_test.go:39-49`, `internal/catalog/catalog_test.go:62-71`,
  `internal/catalog/catalog_test.go:208-217` and `internal/catalog/catalog_test.go:220-241` for the unknown starter,
  YAML that is not valid, a file in `features/` and a read that fails
- `internal/catalog/catalog_test.go:525-538` for a test that reads a doc and checks the names in it
- `README.md:24-40` and `README.md:91-110` for a section of the README about the catalog
- `e2e/tests/new.test.ts:11-19` and `e2e/testUtils.ts:62-69` for an e2e test with `runCli`
- `PROPOSAL.md:137-147`, `PROPOSAL.md:239-249` and `PROPOSAL.md:385-393` for the fields and the two forms
- `tasks/learnings.md:58-63` for the e2e tests on the host
- `tasks/learnings.md:64-69` for the fixes of an audit round in `internal/catalog/catalog_test.go`
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match
- `tasks/learnings.md:119-124` for a test that sets `XDG_CONFIG_HOME`

**Testable:** Yes

**Certainty:** medium — `newNewCommand` (`internal/cmd/new.go:13-65`) is a command over the loaded catalog, and
`readStarter` and `readFeatureFor` parse the same files, but no command prints JSON for a script, nothing reads
the catalog without a starter or reads a `summary`, and no test holds a doc of the tool to an output

**Blast radius:** high — a contract consumed outside this repository: the JSON of `show` is an output format of the
CLI, which scripts read

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant
on the changed lines of `internal/catalog` and `internal/cmd`. Apply one break at a time, read its diff, run the
tests, and put the file back with `git checkout --`: rename one JSON field in the code, remove one row from the
table of `README.md`, and change `[]` to `null` for a list with no entry. Each break makes a test fail. Run
`e2e/tests/show.test.ts` on the host (`tasks/learnings.md:58-63`).

**Depends on:** None

---

### Task 2: Open the TUI of `show` in a terminal, with the list of the starters, its filter, and the view of a starter

**Behavior:** In a terminal, `show` opens a TUI. Its first view is the list of the starters, each with its summary.
`/` starts a filter on the name (decision 2). `l` or Enter opens the view of the selected starter: its summary, its
folders with their commands, its options with their defaults and their values or patterns, its extension points
and those that a feature that targets it declares, each with its summary, its item fields and its declarer, and
the features that target it, each with its summary. `j` and `k` move in a list as the arrow keys do, `h` or Esc goes
back one view, and `q` stops. `show <starter>` opens the TUI at the view of that starter. The TUI writes no file.
`l` on a feature opens nothing until Task 3.

**Acceptance Criteria:**
- [x] go.mod requires `github.com/charmbracelet/bubbletea` v1.3.10 and `github.com/charmbracelet/bubbles` v1.0.0,
  with `github.com/charmbracelet/lipgloss` at v1.1.0 (decision 1). `go mod tidy` leaves go.mod and go.sum as they
  are.
- [x] The first view lists each starter of the definitions that the browser gets, in the order of the names, each
  with its summary. The browser names no starter of its own: a starter that the definitions hold is in the list.
- [x] `/` starts the filter. While the filter is typed, each key goes to the filter, also `h`, `j`, `k`, `l` and
  `q`, and the list holds only the starters whose name matches it. A starter whose summary holds the text, but whose
  name does not, is not in the list. Enter ends the typing and keeps the filtered list, where `j`, `k`, `l` and `q`
  work as before. Esc clears the filter, and the list holds each starter again.
- [x] `j` and `k` move the selection down and up, as the down and up arrows do, in the list of the starters and in
  the list of the features of a starter: from the first starter, `j` and then `l` opens the view of the second
  starter, and `j`, `k` and then `l` opens the first.
- [x] `l` or Enter on a starter opens the view of that starter. It shows the summary of the starter; each folder
  with its path, its `after_render` commands and its `test` commands; each option with its default and its values or
  its pattern; each extension point with its summary, its item fields and its declarer, where the list also holds
  each extension point that a feature that targets the starter declares, with that feature as its declarer; and
  each feature that targets the starter, with its summary. A starter that no feature targets says so.
- [x] `h` or Esc in the view of a starter goes back to the list of the starters, with the same starter selected and
  the same filter. At the list of the starters, `h` does nothing, and Esc clears the filter (decision 10).
- [x] `q` stops the TUI from the list of the starters, when no filter is typed, and from the view of a starter.
  ctrl+c stops it from each view.
- [x] `show <starter>` in a terminal opens the TUI at the view of that starter, and `h` then shows the list of all
  starters. An unknown starter stops `show` before the TUI starts, and the message lists the starters.
- [x] `show` opens the TUI only when stdin and stdout are both terminals (decision 3). Else, and with
  `--output json` in a terminal, it prints the JSON of Task 1 and starts no TUI. The tests of Task 1 pass with their
  expected values unchanged.
- [x] The TUI writes no file: after a session that opens views and stops with `q`, the working folder and the config
  folder hold only what they held before.
- [x] The tests of `internal/browser` pass when the shell sets `CLICOLOR_FORCE=1`, and when it sets `NO_COLOR=1`.
- [x] `e2e/tests/show.test.ts` opens `show` with `openCli`. The screen shows `go-event-sourcing-service` and its
  summary. `/` and a text that matches no name of the catalog hide the starter, and Esc shows it again. `l` opens the
  view of the starter, which shows `go mod tidy`, `task test:unit`, `1.27.1`, `api.middleware`, `cli.commands` with
  the declarer `cli`, and `cli-auth`. `h` shows the list again, and `q` ends with `EXIT:0`.
  `show go-event-sourcing-service` opens at the view of the starter. `show no-such-starter` shows the starters of
  the catalog and an exit status other than 0, and no view of the TUI. The working folder holds no file after each
  test.
- [x] The section "Show the catalog" of `README.md` tells when `show` opens the TUI, the keys of the TUI, and the
  filter with `/`.
- [x] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `go.mod`, `go.sum` — bubbletea, bubbles and lipgloss
- `internal/browser/model.go` — new: `Model`, `New`, `NewAtStarter`, the list of the starters and its filter, and
  the keys
- `internal/browser/starter_view.go` — new: the view of a starter
- `internal/browser/model_test.go` — new: `TestModel`
- `internal/cmd/show.go` — the check for a terminal, and the start of the TUI
- `internal/cmd/update.go` — `isTerminal`, which `show` also calls
- `internal/cmd/show_test.go` — the JSON with a stdin or a stdout that is not a terminal
- `README.md` — the TUI in the section "Show the catalog"
- `e2e/tests/show.test.ts` — the screen tests

**Patterns to Follow:**
- `internal/cmd/update.go:90-93` for the check of a terminal
- `internal/cmd/new.go:13-65` for the steps of a command over the loaded catalog
- `e2e/testUtils.ts:71-90` for `openCli`, and `docs/e2e-tests.md:31-55` and `docs/e2e-tests.md:177-192` for a
  screen test
- `PROPOSAL.md:385-393` for the TUI of `show`
- `task:1` for the definitions, `ForStarter`, the stop for an unknown starter, and the command `show`
- `tasks/learnings.md:58-63` for the e2e tests on the host
- `tasks/learnings.md:119-124` for a test that depends on a variable of the shell
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match
- `tasks/learnings.md:147-152` for a loop over a growing queue, such as the open views or the messages of a test

**Testable:** Yes

**Certainty:** low — no TUI and no Charm module exist in this repo: bubbletea, the list of bubbles, and the clash of
the keys of the story with the keys of that list (`h`, `l` and Esc) are first used here, and the filter is designed
here (decision 2)

**Blast radius:** high — a contract consumed outside this repository: this task decides when `show` gives the TUI
and when it gives JSON, and a wrong check makes a script wait on a TUI

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/browser` and `internal/cmd`.
Run `e2e/tests/show.test.ts` on the host. Run `feature-compose show | cat` and `feature-compose show < /dev/null`
with the built binary in a terminal: each prints the JSON and does not wait.

**Depends on:** Task 1

---

### Task 3: Open the view of a feature from the view of a starter

**Behavior:** `l` or Enter on a feature in the view of a starter opens the view of the feature: its summary, its
needs, its options with their defaults and their values or patterns, the extension points that it declares, and
the extension points that it adds items to. `h` or Esc goes back to the view of the starter with the same feature
selected, and `q` stops.

**Acceptance Criteria:**
- [x] `l` or Enter on a feature in the view of a starter opens the view of that feature. It shows the summary of the
  feature; its needs, or that it has none; each option with its default and its values or its pattern; each
  extension point that it declares, with its summary and its item fields; and each extension point that it adds
  items to.
- [x] `j` and then `l` in the view of a starter opens the view of the second feature in the list.
- [x] `h` or Esc in the view of a feature goes back to the view of the starter, with the same feature selected. A
  second `h` goes back to the list of the starters. The same holds after `show <starter>`.
- [x] `q` stops the TUI from the view of a feature.
- [x] `e2e/tests/show.test.ts` opens the view of `go-event-sourcing-service`, selects `cli-auth`, and opens it with
  `l`: the screen shows the needs `cli` and `api-auth` and the extension point `cli.commands`. `h` shows the view of
  the starter again, and `q` ends with `EXIT:0`.
- [x] The section "Show the catalog" of `README.md` names the view of a feature.

**Affected Files/Modules:**
- `internal/browser/feature_view.go` — new: the view of a feature
- `internal/browser/starter_view.go` — `l` and Enter on a feature
- `internal/browser/model_test.go` — the view of a feature in `TestModel`
- `e2e/tests/show.test.ts` — the view of a feature on the screen
- `README.md` — the view of a feature

**Patterns to Follow:**
- `task:2` for a view that `l` opens from a list, and that `h` and Esc leave with the selection kept
- `task:1` for the definition of a feature
- `e2e/testUtils.ts:71-90` for `openCli`
- `tasks/learnings.md:58-63` for the e2e tests on the host

**Testable:** Yes

**Certainty:** high — the view of a starter of Task 2 (`task:2`) is a view that `l` opens from a list and that `h`
leaves, and the view of a feature is one more of the same kind

**Blast radius:** low — none of the named categories; the view is for a person, and reads only

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/browser`. Run `e2e/tests/show.test.ts` on the host.

**Depends on:** Task 2

## Summary

**Total:** 3 tasks.

**Order.** Dependency first. Task 1 comes first because both forms read the definitions, and it is the task with the
contract: the JSON for a script, its docs, and the test that holds the two together. It gives a whole slice with no
TUI, because `show` prints JSON also in a terminal until Task 2. Task 2 is the riskiest task: the first TUI and the
first Charm modules of the repo, the clash of the keys, and the check that keeps a script from a TUI. Task 3 adds the
last view on the pattern that Task 2 makes.

**Coverage of the story:**

| Criterion of US-014 | Tasks |
| --- | --- |
| In a terminal, `show` opens a TUI with the list of the starters and their summaries; typing filters the list by name | 2 (the filter starts with `/`, decision 2) |
| `l` or Enter on a starter opens its view: summary, folders with commands, options, extension points with summary, item fields and declarer, also those that a feature declares, and the features that target it | 1 (the definitions), 2 |
| `l` or Enter on a feature opens its view: summary, needs, options, the extension points that it declares or adds items to | 1 (the definitions), 3 |
| `j` and `k` move as the arrow keys do; `h` or Esc goes back one view; `q` stops from each view; the TUI writes no file | 2 (the list and the view of a starter), 3 (the view of a feature) |
| `show <starter>` opens the TUI at that starter; an unknown starter stops before the TUI, and the message lists the starters | 1 (the stop), 2 (the TUI) |
| `--output json` prints the catalog; `show <starter> --output json` prints that starter and its features; with a stdout that is not a terminal, `show` prints JSON with no flag | 1, 2 (the check for a terminal) |
| The docs name each field of the JSON, and a test fails when a field changes its name or goes away | 1 |
| The TUI and the JSON come from the catalog folder of the config file; a new starter or feature appears in both with no other change | 1 (the JSON), 2 (the TUI) |
| Context: US-014 and US-012 can share the lists of starters and features | 1 (`Catalog.Definitions`, decision 6) |

No criterion of the story is deferred. The first criterion has `/` before the text of the filter (decision 2).

**Criteria that this breakdown adds, for the caller to overrule:**

- The filter starts with `/`, takes each key while it is typed, and matches the name only (Task 2, decision 2).
- The TUI starts only when stdin and stdout are both terminals (Task 2, decision 3).
- Each field of the JSON on each object, with `[]`, `{}` and `""` for no value, and the default of an option as its
  text in the file (Task 1, decision 4).
- The extension points of a starter in the JSON are only its own (Task 1, decision 5).
- `show` reads no template (Task 1, decision 7).
- `--output` takes only `json`, and `show` takes at most one argument (Task 1, decision 9).
- At the list of the starters, `h` does nothing and Esc clears the filter; ctrl+c stops from each view (Task 2,
  decision 10).
- The tests of the browser pass with `CLICOLOR_FORCE=1` and with `NO_COLOR=1` (Task 2).

**Notes.**

- `clerk mutants` covers the new code of all three tasks: `internal/catalog`, `internal/cmd` and
  `internal/browser`. No task changes a template of the catalog.
- The e2e tests of `show` make no repo, so they fit in the 30 seconds of `e2e/vitest.config.ts`. They are the first
  tests that use `openCli`.
- The definitions are a second read of `starter.yml` and `feature.yml` beside `readStarter` and `readFeatureFor`.
  Task 1 can make the three share one parse, so that a new key of the files reaches `new` and `show` in one place.
- go.mod gets the Charm modules in Task 2, and the Docker image of the e2e job downloads them with `go mod download`.
