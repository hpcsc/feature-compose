# Tasks: Answer the Questions in a TUI (US-012)

## Contents

1. Task 1: Stop `new` on an option that has no value and no default
2. Task 2: Give the answers of the TUI to the selection, with the source `answered`
3. Task 3: List the questions of a selection, each with its default and its check
4. Task 4: Give the arguments of `new` that make the same repo
5. Task 5: Confirm the repo in a TUI before `new` writes it
6. Task 6: Ask for the features, and show the features that the needs add
7. Task 7: Ask for the options, in one group for the starter and one for each feature
8. Task 8: Ask for the starter and the name when the arguments do not give them

## Story Reference

US-012, "Answer the questions in a TUI", in `user-stories/feature-compose.md:379-412` (Description, Acceptance
Criteria and Context). The design is `PROPOSAL.md`: section 8 (the sources of the value of an option, with the TUI
as source 3, `PROPOSAL.md:263-270`), section 11 (the flags and the steps of `new`, `PROPOSAL.md:366-383`), section 12
(the six steps of the TUI, the printed command and the rules that connect the TUI to the flags,
`PROPOSAL.md:398-439`) and decision D12 (the TUI uses huh 1.0.0, `PROPOSAL.md:552`).

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

- `--dry-run` with the TUI. `--dry-run` needs both arguments and starts no TUI, because the confirm step of the TUI
  is already the preview (decision 2).
- A step for the output folder. `--output-dir` gives it, and the TUI does not ask for it.
- A way back from the options step or from the confirm step. The story names one way back: from the needs step to
  the feature step.
- A check of the output folder before the TUI starts. `Selection.Write` checks it after the confirm step, as `new`
  does without the TUI (decision 12).
- The printed command after a run of `new` without the TUI. The developer typed that command (decision 9).
- Quotes for a shell that is not POSIX, for example fish or PowerShell.
- A value of an option of a feature that holds a comma. The `--feature` flag cannot give such a value, so the TUI
  does not accept it (decision 5).
- A value for an option with no default in the combinations of CI. No option of the catalog has an empty default,
  and a combination with such an option stops with the message of Task 1.
- The accessible mode of huh, for a screen reader.

**Deferred:**

- A value that a need sets, and its source `need`. In the catalog format of today a need sets no value (`needs` is a
  list of names, `internal/catalog/definitions.go:61`), so no value of an option comes from a need, and the clause
  "the TUI does not ask for a value that comes ... from a need" holds because no such value exists. US-007 adds the
  values of a need; the question of an option that a need sets then leaves `Selection.Questions` there.
- Questions that `when` hides. No option has `when` yet; US-007 adds `when`, and US-013 is the first option that it
  hides (`user-stories/feature-compose.md:424-425`). This story does not use `Group.WithHideFunc`.

## Codebase Context

**The steps of `new`.** `newNewCommand` (`internal/cmd/new.go:13-74`) takes exactly two arguments through
`asTwoArgumentsAction` (`internal/cmd/adapter.go:28-37`), whose message names `<starter>` and `<name>`. It parses each
`--option` into a value of the starter (`internal/cmd/new.go:76-86`) and each `--feature` into the values of a feature
(`internal/cmd/new.go:88-111`). The parse splits a `--feature` value at its commas (`internal/cmd/new.go:97`), while
`--option` keeps its commas (`DisableSliceFlagSeparator`, `internal/cmd/new.go:25`). It loads the catalog, finds the
starter, and calls `Starter.Select` with the name and the given values. With `--dry-run` it prints
`Selection.Preview` (`internal/cmd/new.go:47-54`). Else it calls `Selection.Write` and `Selection.RunAfterRender`, and
prints one `needed by` line for each entry of `AddedByNeeds` and then `Made <output> from the starter <starter>.`
(`internal/cmd/new.go:55-71`). `--no-prompt` is accepted, and `new` reads no input with or without it
(`internal/cmd/new_test.go:271-283`). `Run` prints an error on stderr through `writeError` and returns 1
(`internal/cmd/root.go:18-25`, `internal/cmd/root.go:61-69`).

**The TUI of `show`.** `show` opens its TUI only when stdin and stdout are both terminals
(`internal/cmd/show.go:46-48`, `isTerminal` at `internal/cmd/update.go:89-92`), and runs a bubbletea program with the
input and the output of the command (`browse`, `internal/cmd/show.go:56-67`). `TestShowCommand` calls `browse`
directly with a context that has ended (`internal/cmd/show_test.go:221-239`). The TUI is `browser.Model`
(`internal/browser/model.go:22-66`), a `tea.Model` whose details pane scrolls with ctrl+d and ctrl+u
(`internal/browser/model.go:32-33`, `internal/browser/model.go:108-111`). `TestModel` drives it through `Update` and
`View`, sends the window size first, and runs each command with a 100 ms limit (`internal/browser/model_test.go:469-630`,
`tasks/learnings.md:175-180`).

**The selection and its sources.** `Selection` (`internal/catalog/selection.go:19-26`) holds the starter and its
values (`selectedStarter`), and each selected feature in the order of rule 8 (`selectedFeature`,
`internal/catalog/selection.go:254-286`), with `given` for a feature that a `--feature` flag gives. `line` gives the
source of a feature: `command line`, `command line, needed by ...`, or `needed by ...`
(`internal/catalog/selection.go:262-270`). `optionValues` (`internal/catalog/option_values.go:1-32`) holds the values
by name and the `given` set, and `lines` gives `command line` or `default`. `AddedByNeeds` gives each feature that a
flag does not give (`internal/catalog/selection.go:28-36`). `Preview` gives the repo, the starter, the features, the
options and the files (`internal/catalog/selection.go:54-65`). `Write` checks the output folder and then calls
`files`, which renders each template and stops before the first file on each error of the render
(`internal/catalog/selection.go:129-145`); `Preview` calls the same `files`, so both stop with one message.

**Options.** `options.withDefaults` (`internal/catalog/option.go:49-77`) checks each given value (an unknown option,
then `option.check`, `internal/catalog/option.go:30-38`: one of `values`, or a match of `pattern`), and renders the
default of each other option with the name of the repo. An option whose `default` is empty, or absent, silently gets
the empty text; no code knows that an option has no default. `Starter.Select` (`internal/catalog/starter.go:73-116`)
checks the name of the repo against `repoName` (`internal/catalog/starter.go:14`), the values of the starter, the
features and their needs (`internal/catalog/starter.go:167-216`), and the values of each feature.

**What the TUI can read.** `Catalog.Definitions` (`internal/catalog/catalog.go:49-74`) gives the `summary` of each
starter and each feature, and `Definitions.ForStarter` (`internal/catalog/definitions.go:81-93`) gives the features
whose targets name a starter. `Catalog.Starter` (`internal/catalog/catalog.go:18-31`) stops on an unknown starter and
lists the starters. `Combination.String` (`internal/catalog/combination.go:15-33`) writes the flags of `new` in the
form that `new` parses, and `TestNewCommand` runs `new` with them and compares the files with `Selection.Write`
(`internal/cmd/new_test.go:309-337`). No code quotes a word for a shell.

**The catalog of this repo.** One starter, `go-event-sourcing-service`, with the options `go-version` (default
`1.27.1`, a pattern) and `module` (default `github.com/hpcsc/{{ .Name }}`, a pattern), and the `after_render`
commands `go mod tidy` and `gofmt -w .`. The features are `api-auth` (option `method`, values `[api-key]`), `cli` (no
option) and `cli-auth` (needs `cli` and `api-auth`, no option). No option has an empty default, no option has
`when`, and no need sets a value. So the tests of an option with no default use a catalog of their own.

**huh 1.0.0.** It is in the module cache and builds with bubbletea v1.3.10, bubbles v1.0.0 and lipgloss v1.1.0 of
go.mod. A `huh.Form` is a `tea.Model`: a model that holds one can pass each message to it and read `State`
(`StateCompleted`, `StateAborted`). Its `Quit` binding is ctrl+c only. `Select` and `MultiSelect` start a filter with
`/` and clear it with Esc, `MultiSelect` toggles an item with space or `x`, `Confirm` takes `y` and `n`, and
`Input.Validate` checks the value when the field is left with Enter, not while the value is typed.

**Tests.** The unit tests have the tag `unit` and run with `task test:unit`. `TestNewCommand` runs `new` on a catalog
in `fstest.MapFS` with a writer that is a buffer (`internal/cmd/new_test.go:23-77`); its reader is not a terminal, so
each of its tests runs `new` without a TUI. `TestSelection` selects from a catalog in `fstest.MapFS`
(`internal/catalog/selection_test.go:24-63`), with groups for the preview (`internal/catalog/selection_test.go:805-1012`)
and `AddedByNeeds` (`internal/catalog/selection_test.go:1146-1185`). `TestStarter` checks the defaults and the given
values (`internal/catalog/starter_test.go:15-62`). `Starter.Select` has about 70 callers, nearly all in tests. The e2e
tests run the binary with `runCli` (pipes, so stdin is not a terminal, `e2e/testUtils.ts:62-69`) or in a terminal of
160x40 with `openCli` (`e2e/testUtils.ts:71-90`, `docs/e2e-tests.md:31-55`, `docs/e2e-tests.md:177-192`); the tests
of `show` in a terminal are `e2e/tests/show.test.ts:43-82`. A run that writes a repo of the real catalog runs
`go mod tidy`, so those tests have a limit of 15 minutes (`e2e/tests/new.test.ts:43-67`). The e2e tests run on the host
when Docker does not answer (`tasks/learnings.md:58-63`).

**Docs.** `README.md` has "Preview a Repo" (`README.md:43-100`), with a table of the sources of a feature and a
table of the sources of an option, and "Show the Catalog" (`README.md:102-176`). It has no section about the TUI of
`new`.

**Words.** `given` is a feature or a value that a flag gives; `values` is the chosen values of a selection;
`options` is the declared options; `needs`, `neededBy` and `Need` are the needs of a feature; `selectedFeature` is a
feature of a selection (`tasks/learnings.md:203-208`). `source` is where a feature or a value comes from, `owner` is
the starter or the feature whose template writes a file, `declarer` is the starter or the feature that declares an
option or an extension point, and `summary` is the sentence of a `starter.yml` or a `feature.yml`. `browser` is the
TUI of `show`. No Go code uses `answer`, `question`, `step`, `confirm` or `prompt` as a name; the flag `--no-prompt`
uses `prompt` for the questions of `new`, and the field `Prompt` of the text input of bubbles (in
`internal/browser/model.go:53`) is the text before an input, in a dependency.

**Decisions of this breakdown, for the caller to overrule:**

1. **The TUI of `new` is one bubbletea model, `prompt.Model`, in a new package `internal/prompt`**, as `browser.Model`
   is the TUI of `show`. It shows one huh form at a time, one for each step, and builds the form of a step when the
   step starts, from the answers of the steps before it. The needs step has two choices, "Continue" and "Change the
   features"; the second shows the feature step again with the selected features kept. A single static form would
   need a hidden group for each option of each feature of each starter, and the order of those groups changes with
   the features. Because the model is a `tea.Model`, its tests drive it through `Update` and `View` as `TestModel` of
   `internal/browser` does, and the e2e tests drive the same steps in a terminal with `openCli`. The package name is
   the word of the flag `--no-prompt`.
2. **When the TUI starts.** `new` starts the TUI only when stdin and stdout are both terminals (as `show` does),
   without `--no-prompt`, and without `--dry-run` (decision 3 of the user). In each other case `new` acts as today,
   with the stop of Task 1. The flags answer whole steps before the TUI starts (decision 2 of the user): the first
   argument answers the starter step, the second the name step, and any `--feature` flag the feature step. A
   `--feature` flag also skips the needs step, because the developer cannot change the features in the TUI then. The
   needs step is also skipped when no need adds a feature. The options step asks only the options that no `--option`
   or `--feature` flag gives.
3. **The source `answered`** (decision 1 of the user). A feature that the developer selects in the TUI, and a value
   that the developer gives in the TUI, show `answered` in the preview, also a default that the developer accepts with
   Enter. A selected feature that another selected feature needs shows `answered, needed by <features>`, as a given
   one shows `command line, needed by <features>` (`tasks/learnings.md:196-201`). `--dry-run` gives no answer, so it
   never shows `answered`. The confirm step shows the text of `Selection.Preview`, where a value from a flag shows
   `command line`: that is how the confirm step shows a value as fixed.
4. **An option with no default.** An option whose `default` is empty or absent has no default, and the empty text is
   no value for it, also when a flag gives it (`--option <name>=`). So the TUI cannot accept the empty text for it,
   and the printed command always names it. `Starter.Select` still gives the selection, so that the TUI can ask the
   option; `Selection.Write` and `Selection.Preview` stop before the first file, with a message that names each such
   option and the flag that gives it. So `new` without the TUI and `--dry-run` stop with the same message (US-010).
5. **The questions come from the catalog.** `Selection.Questions` gives the options that the TUI asks, each with its
   default and its accepted values, and `Question.Check` checks an answer with the rule with which `Select` checks the
   same value from a flag, and with the rule of decision 4. `Check` also refuses a comma in a value of an option of a
   feature: the parse of `--feature` splits at commas (`internal/cmd/new.go:97`), so no printed command could give
   that value. A value of an option of the starter keeps its commas.
6. **The options step** has one group for the options of the starter, first, and then one group for each feature
   with a question, in the order of rule 8. A group with no question is not shown, and with no question at all the
   TUI goes on to the confirm step. A text option starts with its default, and an option with `values` starts at its
   default. While the value of a text option fails `Check`, the TUI shows the message of `Check`, and Enter does not
   accept the value. huh's `Input.Validate` checks only when the field is left, so the message while the value is
   typed is the TUI's own.
7. **The confirm step** shows the text of `Selection.Preview` in a pane that scrolls with ctrl+d and ctrl+u, and with
   pgdown and pgup, as the details of `show` do, because the preview of the real starter is far longer than 40 rows.
   It asks whether to write the repo, with "Yes" and "No". An error of the preview ends the TUI with the message that
   stops `--dry-run`.
8. **A cancel** is Esc or ctrl+c at each step, or "No" at the confirm step. `new` then writes no file, makes no output
   folder and runs no command. It ends through the error path of `Run`, with a message on stderr that says that the
   developer canceled and that `new` wrote no file, and the status 1, so `feature-compose new && cd my-service` does
   not go on into a folder that does not exist.
9. **The printed command.** Only after the TUI writes the repo, `new` prints the lines that it prints today and then
   one line with `feature-compose new` and the arguments that `Selection.Arguments` gives. A run without the TUI
   prints only the lines of today, which the e2e tests check exactly (`e2e/tests/new.test.ts:112-161`). The arguments
   are the starter, the name, `--output-dir <output>` when the output folder is not the name, `--option <name>=<value>`
   for each option of the starter whose value is not its default (in the order of the names), and then
   `--feature <feature>[:<option>=<value>,...]` for each feature that a flag or an answer selected (in the order of
   the names) and for each other feature with a value that is not its default (in the order of rule 8), each with its
   options whose value is not the default (in the order of the names), in the form of `Combination.String`. So the
   example of the story, `go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=oauth`,
   comes out in that order. A value that equals its default is left out, also when a flag or an answer gave it. Each
   word that a POSIX shell would split, expand or match against files is in single quotes. Like the example, the
   command has no `--no-prompt`.
10. **The errors of the flags.** When both arguments are given, `new` makes the selection of the flags before the TUI
    starts, so each error of a flag stops `new` before the TUI, as today. When the TUI asks for the starter or the name
    (Task 8), the TUI makes the first selection after the name step, and an error of a flag ends the TUI there with
    the message that stops `new` without the TUI.
11. **The name step** checks the name with the rule of `Select`, which `catalog.CheckName` gives to the TUI.
12. **The output folder** is checked by `Selection.Write` after the confirm step, as today. A check before the TUI is
    out of scope.

## Tasks

### Task 1: Stop `new` on an option that has no value and no default

**Behavior:** An option whose `default` is empty, or absent, has no default (decision 4). When no flag gives such an
option a value that is not empty, `new` and `new --dry-run` stop before the first file with a status other than 0,
and the message names each such option with the flag that gives it: `--option <name>=<value>` for an option of the
starter, and `--feature <feature>:<name>=<value>` for an option of a feature. Today such an option silently gets the
empty text. `Starter.Select` still gives the selection; `Selection.Write` and `Selection.Preview` stop.

**Acceptance Criteria:**
- [x] With a catalog whose starter has an option with no default, `new <starter> <name>`, with and without
  `--no-prompt`, stops with a status other than 0. The message names the option and the starter, and shows
  `--option <name>=<value>`. No output folder is made, and no `after_render` command runs.
- [x] An option with no default of a feature stops `new` in the same way, also when only a need selects the feature.
  The message names the option and the feature, and shows `--feature <feature>:<name>=<value>`.
- [x] When more than one option has no value, the message names each of them, in the order of the options of the
  preview.
- [x] `--option <name>=` and `--feature <feature>:<name>=`, with the empty text, stop `new` for an option with no
  default in the same way. A value that is not empty and passes the check of the option makes the repo with that
  value.
- [x] An option that has a default acts as today: with no flag it gets its default, and a flag can give it the empty
  text when its pattern allows it.
- [x] `--dry-run` stops with the same message and status, prints nothing on stdout, and makes no output folder.
- [x] In `internal/catalog`, `Starter.Select` gives a selection for such a catalog, and `Selection.Write` and
  `Selection.Preview` stop with the same message, before the first file.
- [x] The tests of `TestSelection`, `TestStarter` and `TestNewCommand`, and the e2e tests, pass with their expected
  values unchanged: no option of the catalog in `catalog/` has an empty default.

**Affected Files/Modules:**
- `internal/catalog/option.go` — an option knows that it has no default
- `internal/catalog/option_values.go` — the options of the starter or of a feature that have no value
- `internal/catalog/selection.go` — `Write` and `Preview` stop on an option with no value, before the render
- `internal/catalog/selection_test.go` — the stop in the groups `write` and `preview` of `TestSelection`
- `internal/catalog/starter_test.go` — `Select` gives the selection with an option that has no default
- `internal/cmd/new_test.go` — the stop in the groups `stop` and `dry run` of `TestNewCommand`

**Patterns to Follow:**
- `internal/catalog/option.go:30-38` and `internal/catalog/option.go:49-77` for the check of one value and the choice
  between a given value and the default
- `internal/catalog/selection.go:129-145` and `internal/catalog/selection.go:38-65` for one stop that `Write` and
  `Preview` share before the first file
- `internal/catalog/option_values.go:1-32` for the values of one declarer and the `given` set
- `internal/catalog/starter_test.go:15-62` for a starter with options in `fstest.MapFS`
- `internal/cmd/new_test.go:23-51` and `internal/cmd/new_test.go:499-653` for a stop of `new` with no output folder
- `internal/cmd/new_test.go:339-478` for a stop of `--dry-run` with the message of `new`
- `tasks/learnings.md:182-187` for a test in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match

**Testable:** Yes

**Certainty:** high — `Selection.files` (`internal/catalog/selection.go:129-145`) already stops `Write` and `Preview`
with one message before the first file, and `option.check` (`internal/catalog/option.go:30-38`) is the check of one
value; this adds one more check of the same kind

**Blast radius:** low — none of the named categories; `new` stops in more cases and writes nothing in them, and no
option of the catalog has an empty default

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` (`tasks/learnings.md:210-215`). Apply one break at a time, read its diff, run
the tests, and put the file back with `git checkout --`: give an option with no default the empty text again; let a
flag with the empty text count as a value; drop the stop from `Preview`. Each break makes a test fail.

**Depends on:** None

---

### Task 2: Give the answers of the TUI to the selection, with the source `answered`

**Behavior:** `Starter.Select` takes the answers of the TUI next to the values that the flags give: the features that
the developer selects, and the values of options of the starter and of the features (`catalog.Answers`). A feature
or a value that an answer gives has the source `answered` in the preview (decision 3). `AddedByNeeds` gives only the
features that neither a flag nor an answer selects. An answered value is checked as a value of a flag is, and an
answered feature as a feature of a flag is. With no answers, each result is as before.

**Acceptance Criteria:**
- [x] A selection whose answers select features holds those features and their needs, in the order of rule 8, and
  writes the same files as the selection that `--feature` flags with the same features give.
- [x] In the preview, a feature that an answer selects shows `<feature>: answered`, and
  `<feature>: answered, needed by <features>` when other selected features need it, with the features that need it in
  the order of their names. A feature that only a need selects keeps `needed by <features>`, and a feature that a flag
  gives keeps `command line`.
- [x] In the preview, each value that an answer gives shows the source `answered`, also when it equals the default.
  A value that a flag gives keeps `command line`, and each other value keeps `default`.
- [x] `AddedByNeeds` gives no feature that an answer selects, also when a selected feature needs it.
- [x] An answered value for an unknown option, a value that is not one of the `values` of its option, and a value that
  does not match the pattern of its option stop `Select` with the message that the same value from a flag gives. An
  answered feature that does not target the starter stops `Select` with the message of an unknown feature, which lists
  the features that target the starter.
- [x] With no answers, `Select`, `Preview`, `Write` and `AddedByNeeds` give what they gave before: the tests of
  `TestSelection`, `TestStarter`, `TestCatalog` and `TestNewCommand`, and the e2e tests of `--dry-run`, pass with their
  expected values unchanged. `new` gives no answers, so `new --dry-run` shows no `answered`.

**Affected Files/Modules:**
- `internal/catalog/answers.go` — new: `Answers`
- `internal/catalog/starter.go` — `Select` takes the answers, selects the answered features with their needs, and
  checks the answered values
- `internal/catalog/option.go` — `withDefaults` takes the answered values next to the given ones
- `internal/catalog/option_values.go` — the answered values, and the source `answered`
- `internal/catalog/selection.go` — an answered feature, its line in the preview, and `AddedByNeeds`
- `internal/catalog/selection_test.go`, `internal/catalog/starter_test.go`, `internal/catalog/catalog_test.go`,
  `internal/catalog/combination_test.go` — the callers of `Select`, and the groups `preview` and `added by needs`
- `internal/cmd/new.go` — `new` calls `Select` with no answers
- `internal/cmd/new_test.go` — the caller of `Select` in the round trip of the combinations

**Patterns to Follow:**
- `internal/catalog/option_values.go:1-32` for the `given` set and the source `command line` of a value
- `internal/catalog/selection.go:254-286` for `selectedFeature`, its `given` flag and its line
- `internal/catalog/selection.go:28-36` for `AddedByNeeds`
- `internal/catalog/starter.go:73-116` and `internal/catalog/starter.go:167-199` for `Select` and the walk of the needs
  from the given features
- `internal/catalog/option.go:49-77` for the check of a given value
- `internal/catalog/combination.go:9-13` for the fields `StarterValues` and `FeatureValues`
- `internal/catalog/selection_test.go:805-1012` and `internal/catalog/selection_test.go:1146-1185` for the preview and
  `AddedByNeeds` of a selection
- `tasks/learnings.md:203-208` for the words of options, values, given and needs
- `tasks/learnings.md:196-201` for a source that keeps the relation `needed by` for every feature
- `tasks/learnings.md:147-152` for a walk over the needs, as recursion and not as a queue
- `tasks/learnings.md:182-187` for a test in `internal/catalog`

**Testable:** Yes

**Certainty:** high — the `given` set of `optionValues` (`internal/catalog/option_values.go:1-32`) and the `given`
flag of `selectedFeature` (`internal/catalog/selection.go:254-270`) give the source `command line` today; `answered`
is a second source of the same kind, and the many callers of `Select` change mechanically

**Blast radius:** low — none of the named categories; with no answers the files and the preview do not change, and
the preview is text for a person

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd` (`tasks/learnings.md:210-215`). Run the `--dry-run` tests of
`e2e/tests/new.test.ts` on the host (`tasks/learnings.md:58-63`).

**Depends on:** None

---

### Task 3: List the questions of a selection, each with its default and its check

**Behavior:** `Selection.Questions` gives each option of the selection whose value no flag gives: the options of the
starter in the order of their names, and then the options of each feature, with the features in the order of rule 8
and the options of one feature in the order of their names. Each `Question` names its feature (none for an option of
the starter) and its option, and gives its default rendered with the name of the repo and the values that it
accepts. `Question.Check` accepts a value that `Select` accepts as an answer, and else gives the message that
`Select` gives for that value; it also refuses the empty text for an option with no default (Task 1), and a comma in
a value of an option of a feature (decision 5).

**Acceptance Criteria:**
- [x] `Questions` gives the options of the starter, and then the options of each feature, in the order of the
  options of the preview. A need comes before the feature that needs it.
- [x] An option that `--option` or `--feature` gives is not a question. Another option of the same feature is.
- [x] Each option of the selection is a question at most one time, also when two selected features need its feature.
- [x] The default of a question is the default of its option rendered with the name of the repo, for example
  `github.com/hpcsc/my-service` for `github.com/hpcsc/{{ .Name }}`, and the empty text for an option with no default.
  The accepted values are the `values` of the option, in the order of the file, and none for a text option.
- [x] `Check` gives no error for a value that the option accepts. For a value that does not match the pattern, that
  is not one of the values, or that is the empty text of an option with no default, it gives a message that names the
  option and the pattern, the values, or the missing default.
- [x] `Check` refuses a value with a comma for an option of a feature, and the message says that `--feature` cannot
  give it. It accepts a comma in a value of an option of the starter.
- [x] Each answer that `Check` accepts, given back to `Select` as the answer to its question, gives a selection that
  `Preview` and `Write` accept, with that value in the files.
- [x] A selection with no option, or whose each value a flag gives, gives no question.

**Affected Files/Modules:**
- `internal/catalog/question.go` — new: `Question` and `Check`
- `internal/catalog/selection.go` — `Questions`
- `internal/catalog/option.go` — the check of an answer, with the rule of an option with no default
- `internal/catalog/option_values.go` — the options that no flag gives, with their rendered defaults
- `internal/catalog/selection_test.go` — a group `questions` in `TestSelection`

**Patterns to Follow:**
- `internal/catalog/option.go:30-38` and `internal/catalog/option.go:49-77` for the check of a value and the default
  rendered with the name
- `internal/catalog/selection.go:104-110` and `internal/catalog/option_values.go:22-32` for the options in the order of
  the preview
- `internal/catalog/definitions.go:22-26` and `internal/catalog/definitions.go:76-93` for an exported value of the
  catalog that a TUI reads
- `internal/cmd/new.go:88-111` for the parse of `--feature`, which splits at commas
- `internal/catalog/selection_test.go:918-1012` for the options of a selection in the preview
- `task:1` for an option with no default
- `task:2` for the answers that `Select` takes
- `tasks/learnings.md:203-208` for the words of options and values
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match

**Testable:** Yes

**Certainty:** medium — `options.withDefaults` (`internal/catalog/option.go:49-77`) renders the defaults and
`option.check` checks a value, and `Definitions` (`internal/catalog/definitions.go:76-93`) is a value of the catalog
that a TUI reads; but no exported value yet carries a check that a caller outside `internal/catalog` runs, and `Check`
also holds the rule of the parse of `--feature` about commas

**Blast radius:** low — none of the named categories; the questions are read by the TUI, and no file changes

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog`.

**Depends on:** Task 1, Task 2

---

### Task 4: Give the arguments of `new` that make the same repo

**Behavior:** `Selection.Arguments` gives, for an output folder, the arguments of `new` that make the same selection
in that folder, quoted for a POSIX shell (decision 9). `new` with those arguments and `--no-prompt` writes the same
files as `Selection.Write` of the selection.

**Acceptance Criteria:**
- [x] The arguments are the starter, the name, `--output-dir <output>` only when the output folder is not the name,
  `--option <name>=<value>` for each option of the starter whose value is not its default, and `--feature` for each
  feature that a flag or an answer selected and for each other feature with a value that is not its default, in the
  orders of decision 9. A `--feature` holds `:<option>=<value>,...` with each option of the feature whose value is not
  its default, and only the name when each value is its default.
- [x] A value that equals its default is not in the arguments, also when a flag or an answer gave it. A feature that
  only a need selects, and whose each value is its default, is not in the arguments.
- [x] With a catalog where the feature `a` needs `b` and `b` has an option `mode` whose default is `fast`, the answers
  that select `a` and give `mode` the value `slow` give the arguments `service my-service --feature a --feature
  b:mode=slow`.
- [x] A word that a POSIX shell would split, expand or match against files is in single quotes, and a single quote in
  it is kept: `sh -c` with `printf '%s\n'` and the arguments prints each word as the selection has it, for a value
  with a space, with `$HOME`, with `*`, and with `'`.
- [x] In `internal/cmd`, for selections made with flags and with answers — features that an answer selects, a feature
  that a need selects with an answered value that is not its default, values of the starter with a space and with a
  quote, and an output folder that is not the name — `new` run with the arguments as `sh` splits them, and with
  `--no-prompt`, writes the same files as `Selection.Write` of the selection.
- [x] `internal/catalog` has its own test of the arguments of each case above.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — `Arguments`
- `internal/catalog/option_values.go` — the values that are not their default
- `internal/catalog/selection_test.go` — a group `arguments` in `TestSelection`
- `internal/cmd/new_test.go` — the round trip of the arguments through `sh` and `new --no-prompt`

**Patterns to Follow:**
- `internal/catalog/combination.go:15-33` for the flags of `new` written by the catalog
- `internal/cmd/new_test.go:309-337` for a run of `new` with the arguments that the catalog gives, compared with
  `Selection.Write`
- `internal/cmd/new.go:76-111` for the parse of `--option` and `--feature` that the arguments must fit
- `internal/catalog/option.go:49-77` for the default of an option, rendered with the name
- `internal/catalog/starter.go:201-216` for the order of rule 8
- `task:2` for the features and the values that an answer gives
- `tasks/learnings.md:3-8` for a test that runs a shell command, which starts in a temporary folder
- `tasks/learnings.md:182-187` for a test in `internal/catalog`, not only in `internal/cmd`

**Testable:** Yes

**Certainty:** medium — `Combination.String` (`internal/catalog/combination.go:15-33`) writes the flags of `new`, and
`TestNewCommand` already runs `new` with them (`internal/cmd/new_test.go:309-337`); but no code here quotes a word for
a shell, and the choice of what to name depends on the defaults, which a selection does not keep after the choice of
a value

**Blast radius:** high — a contract consumed outside this repository: a shell runs the printed command, and a word
that is not quoted makes another repo, or runs a command, when the developer pastes it

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd`. Apply one break at a time, read its diff, run the tests,
and put the file back with `git checkout --`: leave out the quotes; name a value that equals its default; leave out a
feature that an answer selected. Each break makes a test fail.

**Depends on:** Task 2

---

### Task 5: Confirm the repo in a TUI before `new` writes it

**Behavior:** In a terminal, without `--no-prompt` and without `--dry-run`, `new <starter> <name>` starts the TUI of
`new` after the checks of the flags (decisions 1, 2 and 10). In this task the TUI has one step, the confirm step
(decision 7): it shows the text of `Selection.Preview`, and asks whether to write the repo. "Yes" ends the TUI, and
`new` writes the repo, runs the `after_render` commands, prints the lines that it prints today, and then the command
`feature-compose new <arguments>` of Task 4. A cancel writes nothing (decision 8). With `--no-prompt`, `--dry-run`,
or a stdin or a stdout that is not a terminal, `new` starts no TUI and acts as before, with the stop of Task 1. Task 6
to Task 8 add the steps before the confirm step.

**Acceptance Criteria:**
- [x] go.mod requires `github.com/charmbracelet/huh` v1.0.0, and keeps bubbletea v1.3.10, bubbles v1.0.0 and lipgloss
  v1.1.0. `go mod tidy` leaves go.mod and go.sum as they are.
- [x] `new` starts the TUI only when stdin and stdout are both terminals, without `--no-prompt` and without
  `--dry-run`. Else it reads no input and starts no TUI: the tests of `TestNewCommand` pass with their expected values
  unchanged, and the e2e tests that run `new` with `runCli` pass unchanged.
- [x] A flag that stops `new` without the TUI, for example an unknown starter, an unknown feature, or a value that
  does not match its pattern, stops `new` before the TUI starts, with the same message and status.
- [x] The confirm step shows the text of `Selection.Preview` for the selection, where each value of a flag shows
  `command line`. ctrl+d and pgdown scroll the text down, and ctrl+u and pgup scroll it up, so each line of a preview
  that is longer than the screen can be shown.
- [x] An error of the preview, for example a template that fails to render, ends the TUI with the message that stops
  `--dry-run`, and `new` writes nothing.
- [x] Until the developer chooses "Yes", `new` writes no file, makes no output folder and runs no command.
- [x] "Yes" (also `y`) ends the TUI. Then `new` writes the repo, runs the `after_render` commands, prints the `needed by`
  lines and the line `Made <output> from the starter <starter>.` as today, and then a line with `feature-compose new`
  and `Selection.Arguments` of the output folder. Its status is 0.
- [x] "No" (also `n`) at the confirm step, Esc, and ctrl+c are a cancel: `new` writes no file, makes no output folder,
  runs no command, prints on stderr that the developer canceled and that `new` wrote no file, and its status is 1.
- [x] The tests of `internal/prompt` drive `prompt.Model` through `Update` and `View`: the confirm step shows the lines
  of the preview, "Yes" gives the selection, and "No", Esc and ctrl+c give the cancel and stop the TUI. They pass when
  the shell sets `CLICOLOR_FORCE=1`, and when it sets `NO_COLOR=1`.
- [x] `new --help` says that `--no-prompt` starts no TUI, gives each option without a value its default, and stops
  on an option that has no default and no value.
- [x] `README.md` has a section "Answer the Questions in a TUI". It tells when `new` starts the TUI, that the confirm
  step shows the preview of `--dry-run` with the values of the flags as `command line`, how to scroll it, how to
  confirm and how to cancel, the status of a cancel, the printed command, and that this command with `--no-prompt`
  makes the same files. It also tells that with `--no-prompt`, or with no terminal, an option with no default and no
  value stops `new`, and that the message names the flag that gives it.
- [x] `e2e/tests/new.test.ts`, in a terminal with `openCli`:
  `new go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=api-key --option
  go-version=1.28 --option module=github.com/acme/my-service` shows the confirm step with
  `cli-auth: command line`, `api-auth: command line, needed by cli-auth`, `cli: needed by cli-auth` and
  `starter.go-version: 1.28 (command line)`; ctrl+d scrolls until `go.mod: the starter` shows. `n` ends with a status
  other than 0, the screen shows the message of the cancel, and the scratch folder holds no entry. Esc ends in the
  same way.
- [x] `e2e/tests/new.test.ts`, in a terminal: `new go-event-sourcing-service my-service --feature cli --option
  go-version=1.28 --option module=github.com/acme/my-service` and `y` end with `EXIT:0`, the repo has `go.mod` with
  `module github.com/acme/my-service`, and the screen shows `feature-compose new go-event-sourcing-service my-service
  --option go-version=1.28 --option module=github.com/acme/my-service --feature cli`.
- [x] `e2e/tests/new.test.ts`, in a terminal: with stdin from `/dev/null`, with stdout to `cat`, and with
  `--no-prompt`, `new go-event-sourcing-service my-service --output-dir <a folder that has a file>` shows no confirm
  step, and ends with the message that the output folder is not empty and a status other than 0.
- [x] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `go.mod`, `go.sum` — huh v1.0.0
- `internal/prompt/model.go` — new: `prompt.Model`, `New`, the confirm step, the cancel, and `Confirmed`
- `internal/prompt/model_test.go` — new: `TestModel`
- `internal/cmd/new.go` — the check for a terminal, the start of the TUI, the write after the confirm step, and the
  printed command; the usage of `--no-prompt`
- `internal/cmd/new_test.go` — the TUI does not start without a terminal; the run of the TUI with a context that has
  ended; the help of `--no-prompt`
- `README.md` — the section "Answer the Questions in a TUI"
- `e2e/tests/new.test.ts` — the screen tests of the confirm step, the cancel and the write, and the runs with no TUI

**Patterns to Follow:**
- `internal/cmd/show.go:46-48` and `internal/cmd/update.go:89-92` for the check of a terminal
- `internal/cmd/show.go:56-67` for the run of a bubbletea program with the input and the output of the command
- `internal/cmd/show_test.go:221-239` for a test of the run of a TUI in `internal/cmd`
- `internal/browser/model.go:22-66` for a `tea.Model` of this repo, its keys and its constructor
- `internal/browser/model.go:32-33` and `internal/browser/model.go:108-111` for a pane that scrolls with ctrl+d and
  ctrl+u
- `internal/browser/model_test.go:469-630` for a test that drives a model through `Update` and `View`
- `internal/cmd/new.go:13-74` for the steps of `new`, its write and its closing lines
- `internal/cmd/root.go:18-25` and `internal/cmd/root.go:61-69` for an error on stderr and the status 1
- `internal/cmd/new_test.go:271-283` and `internal/cmd/new_test.go:480-497` for a run that reads no input, and the help
  of a flag
- `e2e/tests/show.test.ts:43-82` and `e2e/testUtils.ts:71-90` for a screen test, and a run that starts no TUI
- `e2e/tests/new.test.ts:43-67` for an e2e run that writes a repo of the real catalog
- `docs/e2e-tests.md:31-55` and `docs/e2e-tests.md:177-192` for the rules of a screen test
- `README.md:102-176` for the section of a TUI
- `task:4` for the arguments of the printed command
- `task:1` for the stop of `--no-prompt` on an option with no default
- `tasks/learnings.md:175-180` for a test of a bubbletea model, in the TUI of `new`
- `tasks/learnings.md:168-173` for keys that a component of Charm binds by default
- `tasks/learnings.md:189-194` for text on stderr and its colour
- `tasks/learnings.md:37-42` for an e2e test that writes a repo with its own `mise.toml`
- `tasks/learnings.md:58-63` for the e2e tests on the host

**Testable:** Yes

**Certainty:** low — huh is a new dependency, and no code here holds a huh form in a bubbletea model, runs the
commands of huh in a test of a model, or puts a pane that scrolls beside a form; the shape of the TUI is decided
here (decision 1)

**Blast radius:** high — a destructive or irreversible write: this task decides whether `new` writes the repo and runs
the `after_render` commands, of which `gofmt -w .` rewrites files; and a contract consumed outside this repository: it
decides when `new` waits for input, and a wrong check makes a script wait on a TUI

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/prompt` and `internal/cmd`. Run
`e2e/tests/new.test.ts` on the host. In a terminal, run the built binary with
`new go-event-sourcing-service my-service --feature cli-auth` in an empty temporary folder, scroll the preview, and
press `n`: the folder stays empty. Run the same command with `< /dev/null` in another empty folder: it starts no TUI.

**Depends on:** Task 1, Task 4

---

### Task 6: Ask for the features, and show the features that the needs add

**Behavior:** When no `--feature` flag is given, the TUI asks for the features before the confirm step: a list of the
features that target the starter, each with its summary, where the developer selects none, one or more. When a need
adds a feature, the needs step then shows each such feature with the features that need it, and the developer chooses
"Continue", or "Change the features", which shows the feature step again with the same features selected (decision
1). The selected features have the source `answered`. With one or more `--feature` flags, the TUI skips the feature
step and the needs step (decision 2). So `new go-event-sourcing-service my-service` starts at the feature step.

**Acceptance Criteria:**
- [ ] In a terminal, `new <starter> <name>` with no `--feature` flag starts at the feature step. The list holds each
  feature whose targets name the starter, in the order of the names, each with its summary, and no other feature of
  the catalog. Space or `x` selects or clears a feature, the developer can select more than one, and Enter goes on,
  also with no feature selected.
- [ ] A starter that no feature targets has no feature step.
- [ ] After the feature step, when a need adds at least one feature, the needs step shows each line of
  `AddedByNeeds`, for example `api-auth: needed by cli-auth` and `cli: needed by cli-auth`. "Continue" goes on to the
  next step. "Change the features" shows the feature step again, with the features that the developer selected still
  selected. When no need adds a feature, the TUI goes on from the feature step with no needs step.
- [ ] With one or more `--feature` flags, the TUI shows no feature step and no needs step, and the confirm step shows
  those features with `command line`.
- [ ] The confirm step shows each feature that the developer selected with `answered`, or with `answered, needed by
  <features>` when another selected feature needs it, and each feature that only a need adds with `needed by
  <features>`.
- [ ] Esc and ctrl+c are a cancel at the feature step and at the needs step, as at the confirm step.
- [ ] After "Yes", the printed command names each feature that the developer selected.
- [ ] The tests of `internal/prompt` drive these steps through `Update` and `View` with a catalog of their own, which
  has a feature that targets another starter.
- [ ] The section "Answer the Questions in a TUI" of `README.md` tells the feature step, its keys, the needs step and
  its way back, and that `--feature` skips both steps. The table of the sources of a feature in "Preview a Repo" has
  `answered`.
- [ ] `e2e/tests/new.test.ts`, in a terminal:
  `new go-event-sourcing-service my-service --option go-version=1.28 --option module=github.com/acme/my-service` shows
  `api-auth`, `cli` and `cli-auth`, each with its summary. Selecting `cli-auth` and Enter show the needs step with
  `api-auth: needed by cli-auth` and `cli: needed by cli-auth`. "Change the features" shows the feature step with
  `cli-auth` selected. Clearing `cli-auth`, selecting `cli` and Enter show the confirm step with `cli: answered`, and
  `n` ends with a status other than 0 and a scratch folder that holds no entry.

**Affected Files/Modules:**
- `internal/prompt/model.go` — the feature step and the needs step, and the selection of the answered features
- `internal/prompt/model_test.go` — the feature step and the needs step in `TestModel`
- `internal/cmd/new.go` — the TUI gets the given features and values of the flags
- `README.md` — the feature step and the needs step, and the source `answered` of a feature
- `e2e/tests/new.test.ts` — the screen test of the feature step and the needs step

**Patterns to Follow:**
- `task:5` for `prompt.Model`, its steps, its cancel and its tests
- `internal/catalog/definitions.go:81-93` for the features that target one starter, with their summaries
- `internal/catalog/selection.go:28-36` for the features that a need adds
- `internal/browser/model.go:51-64` for a TUI that starts at a given starter
- `e2e/tests/show.test.ts:84-102` for a screen test that moves in a list and checks the screen
- `README.md:74-80` for the table of the sources of a feature
- `task:2` for the answers that select features, and the source `answered`
- `tasks/learnings.md:175-180` for a test of a bubbletea model
- `tasks/learnings.md:196-201` for the relation `needed by` of every feature

**Testable:** Yes

**Certainty:** medium — the model and the confirm step of `task:5`, `Definitions.ForStarter`
(`internal/catalog/definitions.go:81-93`) and `AddedByNeeds` (`internal/catalog/selection.go:28-36`) give each part;
but no TUI here returns to an earlier step with its answers kept, or selects more than one item of a list

**Blast radius:** low — none of the named categories; the step changes only which features the developer selects
before the confirm step

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/prompt` and `internal/cmd`. Run
`e2e/tests/new.test.ts` on the host.

**Depends on:** Task 2, Task 5

---

### Task 7: Ask for the options, in one group for the starter and one for each feature

**Behavior:** After the needs step (or the feature step, or at the start when flags answer the features), the TUI asks
each question of `Selection.Questions` (decision 6): a group for the options of the starter, and then one group for
each feature with a question, in the order of rule 8, so a need comes before the feature that needs it. An option
with `values` is a list of its values; a text option is a text that starts with its default. While a text fails
`Question.Check`, the TUI shows the message and does not accept it. Each answer has the source `answered` in the
confirm step, and the printed command names each answer that is not its default. An option that a flag gives is not
asked, and shows `command line`.

**Acceptance Criteria:**
- [ ] The options step shows a group for the options of the starter, whose title names the starter, and then one
  group for each feature with a question, whose title names the feature, in the order of rule 8: with a catalog where
  the feature `alpha` needs `zed` and both have options, the group of `zed` comes before the group of `alpha`.
- [ ] Each option is asked one time, also when two selected features need its feature.
- [ ] An option that `--option` or `--feature` gives is not asked, and the confirm step shows its value with
  `command line`. A group with no question is not shown, and with no question at all the TUI goes on to the confirm
  step.
- [ ] An option with `values` shows each value, with its default chosen at the start, and the chosen value is the
  answer.
- [ ] A text option starts with its default rendered with the name of the repo. While its value does not match its
  pattern, the TUI shows a message that names the option and the pattern, and Enter keeps the TUI at that option. When
  the value matches, the message goes away, and Enter accepts the value. An option with no default starts empty, and
  the TUI does not accept the empty text. A value with a comma for an option of a feature is not accepted, and the
  message says why.
- [ ] The confirm step shows each value that the developer gave in the TUI with `(answered)`, also a default that the
  developer accepted.
- [ ] Esc and ctrl+c are a cancel at the options step.
- [ ] After "Yes", the printed command names each answered value that is not its default.
- [ ] The tests of `internal/prompt` drive the options step through `Update` and `View` with a catalog of their own.
- [ ] The section "Answer the Questions in a TUI" of `README.md` tells the options step, its groups and their order,
  the check of a text, and that a flag answers an option before the TUI starts. The table of the sources of an option
  in "Preview a Repo" has `answered`.
- [ ] `e2e/tests/new.test.ts`, in a terminal: `new go-event-sourcing-service my-service --feature cli-auth` shows the
  group of `go-event-sourcing-service` with `go-version` at `1.27.1` and `module` at `github.com/hpcsc/my-service`, and
  then the group of `api-auth` with `method`. A `go-version` of `1.20` shows the pattern
  `^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$` on the screen, and Enter stays at `go-version`. A `go-version` of `1.28`, and
  Enter through the other options, show the confirm step with `starter.go-version: 1.28 (answered)`,
  `starter.module: github.com/hpcsc/my-service (answered)`, `api-auth.method: api-key (answered)` and
  `cli-auth: command line`. `n` ends with a status other than 0 and a scratch folder that holds no entry. With
  `--option go-version=1.28`, the TUI does not ask for `go-version`, and the confirm step shows
  `starter.go-version: 1.28 (command line)`.

**Affected Files/Modules:**
- `internal/prompt/model.go` — the options step, its groups, the check of an answer, and the answered values
- `internal/prompt/model_test.go` — the options step in `TestModel`
- `README.md` — the options step, and the source `answered` of an option
- `e2e/tests/new.test.ts` — the screen test of the options step

**Patterns to Follow:**
- `task:5` and `task:6` for `prompt.Model`, its steps and its tests
- `task:3` for the questions, their defaults and `Check`
- `task:2` for the answered values and their source
- `internal/catalog/selection.go:104-110` for the options in the order of the preview
- `e2e/tests/new.test.ts:32-41` for the pattern of `go-version` in a message
- `README.md:82-89` for the table of the sources of an option
- `tasks/learnings.md:175-180` for a test of a bubbletea model

**Testable:** Yes

**Certainty:** medium — the steps of `task:5` and `task:6` and the questions of `task:3` give each part; but the
groups of this step come from the selection that the earlier steps made, and huh checks a text only when the field
is left, while the story asks for the message while the value is typed

**Blast radius:** low — none of the named categories; an answer goes through the check that a flag goes through, and
the confirm step comes before the write

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/prompt`. Run
`e2e/tests/new.test.ts` on the host.

**Depends on:** Task 3, Task 6

---

### Task 8: Ask for the starter and the name when the arguments do not give them

**Behavior:** In a terminal, without `--no-prompt` and without `--dry-run`, `new` with no argument asks for the
starter, from a list of the starters with their summaries, and then for the name of the repo, checked with the rule
of `Select` (decision 11). Then come the steps of Task 6 and Task 7, and the confirm step. `new <starter>` starts at
the name step. Without the TUI, `new` with fewer than two arguments stops as today. A flag that the starter of the
TUI does not know ends the TUI after the name step with the message of `new` without the TUI (decision 10).

**Acceptance Criteria:**
- [ ] In a terminal, `new` with no argument asks for the starter, the name of the repo, the features and the options,
  in that order, and then shows the confirm step. The list of the starters holds each starter of the catalog, in the
  order of the names, each with its summary.
- [ ] The name step does not accept a name that `Select` refuses, for example the empty text, `../escape` or
  `team/service`, and shows the message of `Select` while the name is not valid.
- [ ] `new <starter>` starts at the name step. An unknown starter stops `new` before the TUI starts, and the message
  lists the starters of the catalog.
- [ ] `new <starter> <name>` starts at the feature step, as in Task 6.
- [ ] With no argument or one argument, `new` with `--no-prompt`, with `--dry-run`, or with a stdin or a stdout that is
  not a terminal stops with the message that names `<starter>` and `<name>`, as today, and writes nothing.
- [ ] A `--feature` or `--option` flag that the starter from the TUI does not know, or a value that does not match,
  ends the TUI after the name step. `new` then prints the message that `new <starter> <name>` with the same flags
  prints without the TUI, writes nothing, and its status is 1.
- [ ] Esc and ctrl+c are a cancel at the starter step and at the name step.
- [ ] After "Yes", the printed command names the starter and the name that the developer gave in the TUI.
- [ ] `catalog.CheckName` gives the message of `Select` for a name that `Select` refuses, and `Select` uses it.
- [ ] `new --help` shows the arguments as optional.
- [ ] The tests of `internal/prompt` drive the starter step and the name step through `Update` and `View` with a
  catalog of their own.
- [ ] The section "Answer the Questions in a TUI" of `README.md` tells the six steps in their order, which argument
  or flag answers which step, and the keys of each step.
- [ ] `e2e/tests/new.test.ts`, in a terminal, with a limit of 15 minutes: `new` shows `go-event-sourcing-service` and
  its summary. Enter, `my-service`, Enter, the selection of `cli-auth`, Enter, "Continue", Enter through the options,
  and `y` end with `EXIT:0`, and the screen shows `feature-compose new go-event-sourcing-service my-service --feature
  cli-auth`. Run in a second scratch folder with `--no-prompt`, that command writes a repo with the same files and
  the same contents as the repo of the TUI.

**Affected Files/Modules:**
- `internal/prompt/model.go` — the starter step and the name step, and the first selection after the name step
- `internal/prompt/model_test.go` — the starter step and the name step in `TestModel`
- `internal/catalog/starter.go` — `CheckName`, which `Select` uses
- `internal/catalog/starter_test.go` — `CheckName`
- `internal/cmd/new.go` — `new` with fewer than two arguments in a terminal, and the usage of the arguments
- `internal/cmd/new_test.go` — the stop with fewer than two arguments without the TUI; the help
- `README.md` — the six steps
- `e2e/tests/new.test.ts` — the full screen test, the write and the round trip of the printed command

**Patterns to Follow:**
- `task:6` and `task:7` for the steps of `prompt.Model`
- `internal/catalog/definitions.go:76-79` for the starters with their summaries
- `internal/catalog/starter.go:14` and `internal/catalog/starter.go:73-76` for the rule of the name of the repo
- `internal/catalog/catalog.go:18-31` for the stop on an unknown starter, which lists the starters
- `internal/cmd/adapter.go:28-37` for the stop on fewer than two arguments, which names `<starter>` and `<name>`
- `internal/cmd/new_test.go:499-527` for the stops on the arguments, and the names that `Select` refuses
- `e2e/tests/new.test.ts:112-161` for an e2e run that writes a repo of the real catalog and compares what it prints
- `tasks/learnings.md:37-42` for an e2e test that writes a repo with its own `mise.toml`
- `tasks/learnings.md:175-180` for a test of a bubbletea model
- `tasks/learnings.md:58-63` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — the steps of `task:6` and `task:7`, the starters of `Definitions`
(`internal/catalog/definitions.go:76-79`) and the stop of `asTwoArgumentsAction` (`internal/cmd/adapter.go:28-37`) give
each part; but the number of arguments that `new` accepts now depends on the terminal and the flags, and the flags
are checked only after the TUI has a starter and a name

**Blast radius:** low — none of the named categories; without a terminal `new` keeps its arguments and its messages

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/prompt`, `internal/catalog` and
`internal/cmd` (`tasks/learnings.md:210-215`). Run `e2e/tests/new.test.ts` on the host. In a terminal, run the built
binary with `new` in an empty temporary folder, answer each step, and confirm; then run the printed command with
`--no-prompt` in another empty folder, and compare the two repos with `diff -r`.

**Depends on:** Task 7

## Summary

- **Tasks:** 8.
- **Order:** dependency first, and the rules of the catalog before the TUI that uses them. Task 1 to Task 4 are the
  rules that need no terminal, each tested through the exported API of `internal/catalog` and through `new` without a
  TUI: the stop on an option with no default, the source `answered`, the questions with their check, and the
  arguments of the printed command with their round trip. Task 5 carries the risk of the story: huh, the model of the
  TUI, the start of the TUI, the write after the confirm step, and the cancel. Task 6, Task 7 and Task 8 then add the
  steps before the confirm step: the features and the needs, the options, and last the starter and the name. Each e2e
  test of an earlier task gives the flags that skip the steps of the later tasks, so it stays valid when they land.
- **Assessment:** Task 5 is low certainty, because huh and the shape of the TUI are new here. Task 4 and Task 5 have a
  high blast radius: a shell runs the printed command, and Task 5 decides whether `new` writes and when it waits for
  input.
- **Criteria of the story:**
  - "with no arguments asks for the starter, the name of the repo, the features, and the options, in that order ...
    summary ... only the features that target the starter ... more than one": Task 6 (the features), Task 7 (the
    options), Task 8 (the starter, the name, and the order of the whole).
  - "After the feature step, the TUI shows each feature that a need adds ... go back to the feature step": Task 6.
  - "one group of options for each feature, and a need comes before the feature that needs it. It asks each option
    one time. A text option shows an error ... does not accept that value": Task 3 (the questions and their check) and
    Task 7 (the step).
  - "Arguments and flags give answers before the TUI starts ... starts at the feature step. The TUI does not ask for
    a value that comes from a flag or from a need, and the confirm step shows that value as fixed": Task 5 (the
    confirm step shows a value of a flag as `command line`), Task 6 (both arguments start at the feature step, and
    `--feature` skips it), Task 7 (an option of a flag is not asked), Task 8 (one argument starts at the name step). A
    value from a need does not exist in the catalog format of today (Deferred, to US-007).
  - "The confirm step shows the same output as `--dry-run`. The tool writes nothing until the developer confirms, and
    a cancel writes nothing": Task 5, with the source `answered` of Task 2 in Task 6 and Task 7.
  - "After the tool writes the repo, it prints the command ... That command with `--no-prompt` makes the same files":
    Task 4 (the arguments and their round trip through `sh` and `new --no-prompt`), Task 5 (the printed line), Task 8
    (the round trip of the full TUI in a terminal).
  - "When stdin is not a terminal, or with `--no-prompt`, the TUI does not start. An option with no value and no
    default then stops the tool, and the message names the option": Task 5 (no TUI) and Task 1 (the stop).
  - Context, the printed command names the selected features and each other feature whose options do not have their
    defaults, as in the example: Task 4 (decision 9).
