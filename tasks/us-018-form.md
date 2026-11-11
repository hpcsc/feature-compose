# Tasks: Answer the Questions in One Form (US-018)

## Contents

1. Task 1: Share the look of the TUI of `show` in a package of its own
2. Task 2: Give a selection with no name, and stop its write and its preview until it has one
3. Task 3: Open the form of `new` with the name, the starter and the features
4. Task 4: Select and clear features with Space, with their needs
5. Task 5: Show the options of the starter and of each selected feature in the tree
6. Task 6: Edit the name and the options with Enter
7. Task 7: Choose the starter with Enter
8. Task 8: Show the preview with `w`, and confirm the write
9. Task 9: Open the form in `new`, and remove the steps of huh

## Story Reference

US-018, "Answer the questions in one form", in `user-stories/feature-compose.md:564-612` (Description, Acceptance
Criteria and Context). The design is `PROPOSAL.md`: section 8 (the sources of the value of an option,
`PROPOSAL.md:254-273`), section 11 (the flags and the steps of `new`, and the keys of `show`, `PROPOSAL.md:356-396`),
section 12 (the form, its tree, its keys, and the rules that connect it to the flags, `PROPOSAL.md:398-446`) and
decision D12 (one form with the look and the keys of `show`, Bubble Tea and Bubbles for both TUIs,
`PROPOSAL.md:559`). The user chose, through questions: one form like `show` (not the steps of US-012 with a new
look), the details in the right pane with the preview on `w`, and Bubble Tea and Bubbles for the form, so huh leaves
the TUI and `go.mod`.

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

- The output folder in the form. `--output-dir` gives it, the form does not ask for it, and `Selection.Write` checks
  it after the developer confirms the write, as in US-012.
- A filter (`/`) in the form. The tree holds one starter, so it has few rows.
- A change of a fixed value in the form. A developer who wants another value runs `new` without that argument or
  flag.
- The answered values of a feature after the developer clears it, and the answered values of the options of the
  starter after a change of the starter: the form drops them (decision 10).
- The form with `--dry-run`, `--no-prompt`, or a stdin or stdout that is not a terminal: `new` opens no form, as in
  US-012.
- Quotes for a shell that is not POSIX in the printed command, as in US-012.
- Input from a mouse.

**Deferred:**

- `when`. The catalog has no `when` today: an option declares only `default`, `pattern` and `values`
  (`internal/catalog/definitions.go:22-26`), and US-007 adds `when`. The form reads `Selection.Questions` again after
  each answer (Task 5 and Task 6), so the clause "it changes after each answer" holds for each option that the
  selection holds; the clause "the tree shows an option only while its `when` is true" comes when US-007 makes
  `Questions` leave out an option whose `when` is false.
- A value that a need sets, its source `need`, and its mark `fixed`. A need is a name and sets no value
  (`internal/catalog/definitions.go:61`), and the sources of a value are `command line`, `answered` and `default`
  (`internal/catalog/option_values.go:42-50`). US-007 adds the values of a need; `Question.Given` (Task 5) marks only
  a value from a flag, so a value from a need needs a source of its own in `Question` there.
- The message of a need whose value differs from a flag, which names both sources. `Starter.Select` gives no such
  message today, because no need sets a value. Task 4 already handles a Space whose selection `Select` refuses: the
  feature stays as it was, and the details show the message of `Select`. So the message of US-007 shows there with
  no change to the form.

## Codebase Context

**The TUI of `new` today.** `prompt.Model` (`internal/prompt/model.go:50-69`) is the TUI of US-012: one huh form at a
time, in the steps starter, name, features, needs, options and confirm. `prompt.New` (`internal/prompt/model.go:71-88`)
takes the catalog, the starter and the name from the arguments, and the values of `--option` and `--feature`. It
calls `Starter.Select` only after it has a name (`internal/prompt/model.go:209-222`), and again after each step with
the answers so far (`internal/prompt/model.go:240-251`, `internal/prompt/model.go:337-345`). `Confirmed`
(`internal/prompt/model.go:142-147`) gives the selection after "Yes", and otherwise the error
`you canceled new, so it wrote no file` (`internal/prompt/model.go:20`) or the error that ended the TUI. The name step
shows the message of `catalog.CheckRepoName` after each key (`internal/prompt/model.go:189-207`). huh is used only in
`internal/prompt`. Its tests (`internal/prompt/model_test.go`, umbrella `TestModel`) drive the model through `Update`
and `View` with catalogs in `fstest.MapFS`.

**The TUI of `show`.** `browser.Model` (`internal/browser/model.go:39-49`) draws a title line (`appTitleStyle` and
the counts, `internal/browser/model.go:209-217`), a tree pane and a details pane side by side
(`internal/browser/model.go:170-194`) as `panel`s with a rounded border, a title and a scroll note
(`internal/browser/panel.go:10-40`), and a footer of the keys with the help of bubbles
(`internal/browser/model.go:219-229`). A terminal smaller than 40 by 8 shows only a message
(`internal/browser/model.go:16-20`, `internal/browser/model.go:174-176`). The tree pane takes at most two fifths of
the width and keeps the selected row in view (`internal/browser/model.go:140-161`). The tree draws `▾` and `▸`, the
branches `├─ └─ │`, and the selected row with `▌` and its style (`internal/browser/tree.go:223-275`). The details lay
out sections of entries with labelled fields, and wrap them to the pane (`internal/browser/details.go:15-30`,
`internal/browser/details.go:144-244`). The styles are in `internal/browser/styles.go:5-22`. All of these are
unexported. The keys: `j`/`k` and the arrows move, `l`/Enter/right opens (and at an open row selects the first
child), `h`/Esc/left closes (and at another row selects the parent), ctrl+d/ctrl+u and pgdown/pgup scroll the
details, `/` starts a filter in a text input of bubbles whose keys all go to it (`internal/browser/model.go:120-138`),
`q` stops, and ctrl+c always stops (`internal/browser/model.go:22-35`, `internal/browser/model.go:70-118`).

**What the catalog gives the form, and what it does not.**
- `Starter.Select(name, given, givenByFeature, answers)` (`internal/catalog/starter.go:73-121`) refuses each name
  that `CheckRepoName` refuses, the empty name included (`internal/catalog/starter.go:306-311`). So no selection, no
  question and no default that reads the name exists before the developer gives a name.
- `Selection.Write` checks the output folder first and then renders (`internal/catalog/selection.go:49-63`);
  `Selection.Preview` renders (`internal/catalog/selection.go:65-76`). Both stop through `files` and `checkValues` on
  an option with no default and no value (`internal/catalog/selection.go:185-217`).
- `Selection.Questions` (`internal/catalog/selection.go:78-84`, `internal/catalog/option.go:100-115`) gives only the
  options that no flag gives. A `Question` (`internal/catalog/question.go:8-31`) has `Option`, `Default` (rendered
  with the name) and `Accepted`, an unexported `feature`, `Declarer()` (text such as `the feature api-auth`), and
  `Check`. It does not give the value, the source, the pattern or the feature name. The value and the source of each
  option live in `optionValues` (`internal/catalog/option_values.go:9-50`) and reach a caller only as the text of the
  preview.
- The features that a selected feature needs: `selectedFeature.neededBy` (`internal/catalog/selection.go:331-364`).
  A caller sees it only through `AddedByNeeds` (`internal/catalog/selection.go:39-47`), which leaves out each feature
  that a flag or an answer selects, and through the preview.
- The order of rule 8 (a feature after its needs, otherwise by name) is `Starter.inNeedsOrder`
  (`internal/catalog/starter.go:208-223`), private, over the selected features only. It stops on a cycle of needs
  (`internal/catalog/starter.go:225-238`).
- `Starter.checkAnsweredFeatures` (`internal/catalog/starter.go:240-247`) refuses an answered value of a feature that
  the selection does not hold. So a form that clears a feature must drop its answered values before the next
  `Select`.
- `Answers` (`internal/catalog/answers.go:1-23`) has the exported fields `Features`, `StarterValues` and
  `FeatureValues`; `WithValues` fills the values from the questions of the options step of huh, and its only caller
  outside the package is `internal/prompt/model.go:338`.
- `Catalog.Definitions` and `Definitions.ForStarter` (`internal/catalog/definitions.go:76-93`) give the summaries, the
  needs, and the declared options with their raw default, pattern and values.

**The catalog of this repo.** One starter, `go-event-sourcing-service`, with the options `go-version` (default
`1.27.1`, a pattern) and `module` (default `github.com/hpcsc/{{ .Name }}`, a pattern). The features are `api-auth`
(option `method`, values `[api-key]`), `cli` (no option) and `cli-auth` (needs `cli` and `api-auth`, no option). The
order of the names is also the order of rule 8 here, so a test of that order needs a catalog of its own.

**`new`.** `asked` (`internal/cmd/new.go:30`) is true only with stdin and stdout terminals, without `--no-prompt` and
without `--dry-run`; then `new` takes zero, one or two arguments. `ask` (`internal/cmd/new.go:98-108`) runs
`prompt.Model` in a bubbletea program with the alt screen and reads `Confirmed`. After the write, `new` prints the
`needed by` lines, `Made <output> from the starter <starter>.`, and, only after the TUI, the command of
`Selection.Arguments` (`internal/cmd/new.go:66-85`). `Run` prints an error on stderr and returns 1
(`internal/cmd/root.go:18-25`).

**Tests.** The unit tests have the tag `unit` (`task test:unit`). `TestNewCommand` runs `new` in a pseudo-terminal
with `runInTerminal` (`internal/cmd/new_test.go:60-89`), which types keys after a text shows, and its group `tui`
(`internal/cmd/new_test.go:612-732`) types the keys of the steps of huh. `e2e/tests/new.test.ts:276-466` (describe
"feature-compose new in a terminal") drives the steps with tuistory, and `e2e/tests/show.test.ts:43-102` drives the
tree of `show`. `README.md:104-218` ("Answer the Questions in a TUI") tells the steps; `README.md:74-89` holds the
tables of the sources, where `answered` says "also when you keep the default".

**Words.** `browser` is the TUI of `show`; `row`, `tree` and `details` are its words for a line of the tree, the tree,
and the right pane. `panel` is its word for a box, while the stories say "pane". `given` is a value or a feature
that a flag gives; `values` are the chosen values of a selection; `options` are the declared options; `needs` and
`neededBy` are the needs of a feature (`tasks/learnings.md:205-210`). `source` is where a value comes from, with the
texts `command line`, `answered`, `default` and `needed by <features>`. `cancel` is the end of `new` with no write.
`question` is an option of a selection that the TUI of `new` shows. `prompt` is the TUI of US-012 and the word of the
flag `--no-prompt`; `form` is used only for the forms of huh, inside `internal/prompt`, which this story removes. No
Go code uses `screen`, `pane`, `fixed`, `missing`, `keep` or `drop` as a name.

**Decisions of this breakdown, for the caller to overrule:**

1. **The form is `form.Model`, in a new package `internal/form`**, next to `browser.Model` of `show`. The stories and
   D12 call the TUI of `new` a form, so the word of the code follows them; the flag keeps its name `--no-prompt`,
   because scripts use it. `internal/prompt` stays, and `new` uses it, until Task 9 opens the form in `new` and
   removes `internal/prompt` and huh in the same commit. So `new` keeps a working TUI after each task.
2. **The look of `show` moves to a new package `internal/screen`** (Task 1): the panes, the styles, the title line,
   the message of a small terminal, the lines of a tree, and the layout of the details. `browser` keeps what is
   particular to `show`: its tree of starters, groups, features and extension points, the content of its details, its
   keys and its filter. The box is renamed from `panel` to `Pane`, the word of the stories. Each TUI keeps its own
   keys, because Enter opens a row in `show` and edits a value in the form.
3. **A selection with no name** (Task 2). `Starter.Select` accepts the empty text as a name that is missing, and
   refuses each other name that `CheckRepoName` refuses, as today. `Selection.Write` and `Selection.Preview` then stop
   with the message of `CheckRepoName` for the empty text, before any other check: `Write` before it reads or makes
   the output folder. So the form has a selection, its features, its options and the defaults that read the name
   before the developer gives a name, and `new <starter> ''` without the form still stops with the same message and
   writes nothing.
4. **The tree.** The first row is the name of the repo. The second is the starter, open at the start. Under the
   starter come the options of the starter, in the order of their names (Task 5), and then each feature that targets
   the starter, selected or not, each after the features that it needs and otherwise in the order of the names (rule
   8 over all the features of the starter, `Starter.FeatureNames`). So the options of a need come before the options
   of the feature that needs it. A selected feature shows `[x]` and its options under it; a feature that is not
   selected shows `[ ]` and no option. A feature with options opens and closes with `l` and `h`. With no starter
   argument, no starter is chosen at the start: the starter row says so, and the tree has no feature and no option
   until the developer chooses a starter (Task 7). The form does not choose the first starter for the developer.
5. **The marks.** A row whose value an argument or a flag gives shows `fixed`: the name and the starter of the
   arguments, each feature of `--feature`, and each option that `--option` or `--feature` gives. Each selected feature
   that a selected feature needs shows `needed by <features>`, with the features in the order of their names, also
   when the developer or a flag selected it (`tasks/learnings.md:198-203`). An option with no default and no value
   shows that its value is missing. The details give the source of each row with the texts of the preview.
6. **The look.** The title line shows `feature-compose new` in the title style of `show`, and at its right the count
   of the selected features as `show` counts (`1 feature`, `3 features`). The tree pane has the title `Repo`. The
   details pane has the name of the selected row as its title, and `<feature> › <option>` or `starter › <option>`
   for an option. The footer shows the keys that work at that moment.
7. **Keys outside an edit.** `j`/`k` and down/up move. `l`/right opens a closed starter or feature, and at an open one
   selects its first child; `h`/left/Esc close an open starter or feature, and at another row select its parent, as in
   `show`. Enter at a feature row does what `l` does, because a feature has no value to edit. ctrl+d/pgdown and
   ctrl+u/pgup scroll the details. `q` and ctrl+c stop the form with the cancel: `Confirmed` gives
   `you canceled new, so it wrote no file`, and `new` ends with the status 1, as in US-012.
8. **Space** (Task 4) adds the feature to the answered features, or removes it, and the form runs `Select` again. It
   changes nothing, and the details say why, at a feature of `--feature` and at a feature that a selected feature
   needs. A Space whose selection `Select` refuses leaves the selection as it was, and the details show the message of
   `Select`.
9. **An edit** (Task 6, Task 7). Enter at the name or at a text option starts an edit of the value, which starts with
   the current value; each key goes to the edit; the details show the message of `CheckRepoName` or `Question.Check`
   while the text does not pass; Enter keeps the value only when it passes and `Select` accepts it; Esc drops the
   change; ctrl+c stops the form. Enter at the starter or at an option with `values` lists the starters (each with its
   summary) or the values in the details pane, with the current one chosen; `j`/`k` move, Enter keeps, Esc drops.
   Enter at a fixed row starts no edit, and the details say that the value is fixed and name its source.
10. **Answered values.** An option has the source `answered` once the developer keeps a value for it with Enter, also
    when the value equals the default. Until then it has its default, and a default that reads the name follows the
    name. This differs from US-012, where each option of the options step became `answered`. A clear of a feature
    drops the answered values of each feature that the selection no longer holds, and a change of the starter also
    drops the answered values of the options of the starter.
11. **A change of the starter** (Task 7) keeps each feature that the developer selected and that targets the new
    starter, and clears the others. The flags must fit the new starter: a starter whose selection `Select` refuses
    with the flags, or whose needs form a cycle, is not kept, and the details show the message.
12. **`w`** (Task 8). When no answer is missing, the details pane gets the title `Preview` and the focused border, and
    shows the text of `Selection.Preview` and the question `Write the repo?`. `y` stops the form, and `Confirmed`
    gives the selection of the preview. `n` and Esc go back to the form. An answer is missing when no starter is
    chosen, when the repo has no name, or when an option has no default and no value, which are the cases in which
    the form has no selection or `Preview` stops. Then `w` selects the first such row in the order of the tree, opens
    the rows above it, and the details show the message: of `CheckRepoName` for the name, of `Question.Check` for the
    empty text of an option, and a message that names the missing starter. A preview that stops for another reason,
    for example a template that fails to render, shows its message in the details pane with no question.
13. **Errors before the form opens.** With a starter argument, `form.New` stops with the message of the catalog for an
    unknown starter, for a flag that `Select` refuses (with the name, or with no name), and for a cycle of needs among
    the features of the starter. `new` then writes nothing, as in US-012.

## Tasks

### Task 1: Share the look of the TUI of `show` in a package of its own

**Behavior:** The parts that the TUIs of `show` and `new` draw the same way move from `internal/browser` to a new
package `internal/screen` (decision 2): the panes with a rounded border, a title and a scroll note; the styles; the
title line with its counts; the message of a terminal that is too small, which names the command; the lines of a tree
(`▾`, `▸`, `├─`, `└─`, `│` and the selected row); and the layout of the details in sections, entries and fields,
wrapped to the pane. `show` looks and acts as before.

**Acceptance Criteria:**
- [x] `internal/screen` holds the panes, the styles, the title line, the message of a small terminal, the lines of a
  tree and the layout of the details. `internal/browser` holds no copy of them and keeps only its tree of starters,
  groups, features and extension points, the content of its details, its keys and its filter.
- [x] The tests of `TestModel` in `internal/browser` pass with their expected values unchanged, also with
  `CLICOLOR_FORCE=1` and with `NO_COLOR=1`.
- [x] `internal/screen` has its own tests through its exported names, one umbrella test for each type: a pane draws
  its title in the top border and cuts a long title with `…`, cuts each line at its inner width, fills its height,
  and shows a note in its bottom border; the title line puts the counts at the right and keeps only the title when
  they do not fit; the message of a small terminal names the command and the smallest size; the details put a value
  under its name when the pane is too narrow for both, and wrap a long text at a space.
- [x] The message of a small terminal for `show` is the same text as before.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/screen/pane.go` — new: `Pane`, from `internal/browser/panel.go`
- `internal/screen/` — new: the styles, the title line, the message of a small terminal, the lines of a tree, and the
  layout of the details, each in the file of the type or function that it declares, with its tests
- `internal/browser/model.go`, `internal/browser/tree.go`, `internal/browser/details.go` — use `internal/screen`
- `internal/browser/panel.go`, `internal/browser/styles.go` — removed

**Patterns to Follow:**
- `internal/browser/panel.go:10-40` for the pane that moves
- `internal/browser/styles.go:5-22` for the styles that move
- `internal/browser/model.go:170-236` for the title line, the small terminal, the two panes and the footer
- `internal/browser/tree.go:223-275` for the lines of a tree
- `internal/browser/details.go:15-30` and `internal/browser/details.go:144-244` for the layout of the details
- `internal/textcase/textcase.go` and `internal/textcase/textcase_test.go` for code that moved to a package of its own
  with exported names and its own tests
- `internal/browser/model_test.go:66-105` and `internal/browser/model_test.go:469-630` for tests that read the text of
  a drawn screen without its escape codes
- `tasks/learnings.md:184-189` for tests in the package of the code
- `tasks/learnings.md:177-182` for the escape codes of `View`

**Testable:** Yes

**Certainty:** high — commit f020865 moved the general code of `internal/catalog` to packages of their own, such as
`internal/textcase/textcase.go`, with the names exported and the code and messages unchanged; this moves drawing
code the same way

**Blast radius:** low — none of the named categories; the TUI of `show` writes no file, and its tests hold its look

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/screen` and `internal/browser`. In
a terminal, `feature-compose show` looks as before.

**Depends on:** None

---

### Task 2: Give a selection with no name, and stop its write and its preview until it has one

**Behavior:** `Starter.Select` accepts the empty text as a name that the developer has not given yet, and refuses
each other name that `CheckRepoName` refuses, as today (decision 3). The defaults that read the name render with the
empty text. `Selection.Write` and `Selection.Preview` of a selection with no name stop with the message of
`CheckRepoName` for the empty text, before any other check and before the first file. `new` without a form acts as
before: `new <starter> ''` stops with the same message and writes nothing.

**Acceptance Criteria:**
- [x] `Starter.Select` with the empty name gives a selection. Its `Questions` give the default of an option that reads
  the name rendered with the empty text, for example `github.com/hpcsc/` for `github.com/hpcsc/{{ .Name }}`.
- [x] Each other name that `CheckRepoName` refuses, for example `../escape`, `team/service` and `-flag`, still stops
  `Select` with the message that names the name.
- [x] `Selection.Write` of a selection with no name stops with the message of `CheckRepoName` for the empty text. It
  does not read or make the output folder and writes no file, also when the output folder is the working folder and
  holds files.
- [x] `Selection.Preview` of a selection with no name stops with the same message.
- [x] When the name is missing and an option has no default and no value, `Write` and `Preview` stop with the message
  of the name.
- [x] `new <starter> ''`, with and without `--no-prompt`, stops with the message that names `the name "" of the repo`,
  makes no folder, runs no `after_render` command, and leaves the files of the working folder as they are. With
  `--dry-run` it stops with the same message and prints nothing. The test "the name \"\" stops new, and the working
  folder keeps its files" of `TestNewCommand` passes unchanged.
- [x] The other tests of `TestSelection`, `TestStarter` and `TestNewCommand`, and the tests of `internal/prompt`, pass
  with their expected values unchanged.

**Affected Files/Modules:**
- `internal/catalog/starter.go` — `Select` accepts the empty name
- `internal/catalog/selection.go` — `Write` and `Preview` stop on the empty name first
- `internal/catalog/starter_test.go` — the empty name leaves the names that `Select` refuses, and gets its own test
- `internal/catalog/selection_test.go` — the stop of `Write` and `Preview` in the groups `write` and `preview`
- `internal/cmd/new_test.go` — the empty name in the table of the stops of `--dry-run`

**Patterns to Follow:**
- `internal/catalog/selection.go:185-217` for one stop that `Write` and `Preview` share before the first file
- `internal/catalog/selection.go:49-63` and `internal/catalog/selection.go:219-231` for the check of the output
  folder, which the new stop must come before
- `internal/catalog/starter.go:73-76` and `internal/catalog/starter.go:306-311` for the check of the name
- `internal/catalog/starter_test.go:45-62` and `internal/catalog/starter_test.go:146-153` for the tests of the names
- `internal/cmd/new_test.go:520-551` for the stops that `new` and `--dry-run` share
- `internal/cmd/new_test.go:750-762` for a stop that leaves the files of the working folder
- `tasks/learnings.md:3-8` for a test that runs a command, which starts in a temporary folder
- `tasks/learnings.md:184-189` for a test in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:212-217` for a small change in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `Selection.checkValues` (`internal/catalog/selection.go:206-217`) already stops `Write` and
`Preview` before the first file on a missing value; but `Write` runs that check after the check of the output folder
(`internal/catalog/selection.go:49-56`), and the stop of the name must come before it

**Blast radius:** high — a destructive write: this changes the check that keeps `new` from writing a repo with the
working folder as its output folder, where the `after_render` commands, `gofmt -w .` among them, would then run

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog`. Apply one break at a time, read its diff, run the tests, and put the file
back with `git checkout --`: drop the stop from `Write`; move it after the check of the output folder; drop it from
`Preview`; accept `../escape` in `Select`. Each break makes a test fail.

**Depends on:** None

---

### Task 3: Open the form of `new` with the name, the starter and the features

**Behavior:** `form.New` takes what `prompt.New` takes: the catalog, the starter and the name of the arguments (each
can be empty), and the values of `--option` and `--feature`. It gives a model with the look of `show` (decisions 4,
5, 6 and 7): a title line, a tree pane with the name of the repo, the starter, and each feature that targets the
starter, and a details pane with the details of the selected row. The features of `--feature` and their needs show as
selected. The developer moves in the tree, opens and closes rows, scrolls the details, and stops the form with `q`.
`new` does not open this form yet (decision 1).

**Acceptance Criteria:**
- [x] After the size of the terminal, the screen has a title line with `feature-compose new` and the count of the
  selected features, a tree pane with the title `Repo`, a details pane whose title names the selected row, and a
  footer with the keys of the form. The panes, the colours and the selected row are those of `internal/screen`.
- [x] The tree holds the name row first, and then the starter row, open, with each feature that targets the starter
  under it, each after the features that it needs and otherwise in the order of the names: with a catalog where
  `alpha` needs `zed`, `zed` comes before `alpha`. A feature that targets only another starter is not in the tree. A
  selected feature shows `[x]`, another `[ ]`.
- [x] Each value that an argument or a flag gives is marked `fixed`: the name and the starter of the arguments, and
  each feature of `--feature`. Each selected feature that a selected feature needs shows `needed by <features>` with
  those features in the order of their names, also a feature of `--feature`, which then shows both marks.
- [x] With no name argument, the name row shows that the name is missing. With no starter argument, the starter row
  shows that no starter is chosen, and the tree has no feature.
- [x] The details of the name row show the name and its source, or the message of `CheckRepoName` for the empty text
  when the name is missing. The details of the starter row show the summary of the starter and its source; with no
  starter, they list each starter of the catalog with its summary. The details of a feature show its summary, its
  needs, and its source with the texts of the preview (`command line`, `needed by <features>`,
  `command line, needed by <features>`), or that it is not selected.
- [x] `j` and `k` move down and up by one row, as the arrow keys do, and stop at the ends. `l` and right at an open
  row select its first child, and at a closed row open it. `h`, left and Esc at an open row close it, and at another
  row select its parent. Enter at a feature row does what `l` does.
- [x] ctrl+d and pgdown scroll the details down by half the pane, ctrl+u and pgup scroll them up, and the bottom
  border of the details pane shows how far they are scrolled. A new selection shows its details from the top.
- [x] The tree pane takes at most two fifths of the width, and a tree taller than its pane follows the selected row.
  A terminal smaller than the smallest size of `show` shows only the message that asks for a larger terminal, and it
  names `new`.
- [x] `q` and ctrl+c stop the form, and `Confirmed` then gives the error `you canceled new, so it wrote no file`. The
  form writes no file.
- [x] With a starter argument, `form.New` stops with the message of the catalog for an unknown starter (which lists
  the starters), for a flag that `Select` refuses with the name or with no name (an unknown feature, an unknown option,
  a value that does not match its pattern), and for features of the starter whose needs form a cycle.
- [x] In `internal/catalog`, `Starter.FeatureNames` gives the names of the features that target the starter, each after
  the features that it needs and otherwise in the order of the names, and stops on a cycle of needs with the message
  that `Select` gives for it. `Selection.NeededBy` gives the selected features that need a feature, in the order of
  their names, and none for a feature that no selected feature needs or that the selection does not hold. Both have
  tests in `internal/catalog`.
- [x] The tests of `internal/form` drive the model through `Update` and `View` with catalogs in `fstest.MapFS`, and
  pass with `CLICOLOR_FORCE=1` and with `NO_COLOR=1`. The tests of `internal/prompt` and `internal/cmd` pass unchanged.

**Affected Files/Modules:**
- `internal/form/model.go` — new: `form.Model`, `New`, `Confirmed`, the keys, the layout
- `internal/form/tree.go` — new: the rows of the form, the selected row, the open rows
- `internal/form/` — the details of the rows, in the file of the type that declares them
- `internal/form/model_test.go` — new: `TestModel`
- `internal/catalog/starter.go` — `FeatureNames`
- `internal/catalog/selection.go` — `NeededBy`
- `internal/catalog/starter_test.go`, `internal/catalog/selection_test.go` — tests of `FeatureNames` and `NeededBy`

**Patterns to Follow:**
- `task:1` for the panes, the styles, the title line, the tree lines and the details
- `task:2` for a selection with no name
- `internal/browser/model.go:22-161` for the keys, the model, the moves, the scroll of the details and the layout
- `internal/browser/tree.go:74-175` for the rows of a tree, the selected row, and open and close
- `internal/browser/details.go:98-118` for the details of a feature
- `internal/prompt/model.go:71-88` and `internal/prompt/model.go:176-187` for the constructor that takes the arguments
  and the flags, the starter and its definitions
- `internal/prompt/model.go:20` and `internal/prompt/model.go:142-147` for the cancel and `Confirmed`
- `internal/catalog/starter.go:208-238` for the order of rule 8 and the message of a cycle
- `internal/catalog/starter.go:249-258` and `internal/catalog/selection.go:331-364` for the features that need a feature,
  and the source of a feature
- `internal/browser/model_test.go:18-207` and `internal/browser/model_test.go:469-630` for the tests of a two-pane
  model and their helpers
- `internal/prompt/model_test.go:22-82` for catalogs in `fstest.MapFS` with needs
- `tasks/learnings.md:177-182` for a test of a bubbletea model
- `tasks/learnings.md:198-203` for the relation `needed by` of every needed feature
- `tasks/learnings.md:205-210` for the words of options, values and needs
- `tasks/learnings.md:149-154` for a walk over the needs
- `tasks/learnings.md:184-189` for tests in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `browser.Model` (`internal/browser/model.go:22-161`, `internal/browser/tree.go:74-175`) is a
two-pane tree with details and the same keys; but its rows come from the definitions, while the rows of the form
come from a selection, with checkboxes and the marks `fixed` and `needed by`, and the order of rule 8 over all
features of a starter is new to the catalog

**Blast radius:** low — none of the named categories; `new` does not open the form yet, and the form writes no file

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form` and `internal/catalog`.

**Depends on:** Task 1, Task 2

---

### Task 4: Select and clear features with Space, with their needs

**Behavior:** Space at a feature row selects the feature, or clears it (decision 8). A selected feature brings its
needs, which show `[x]` and `needed by <features>`. A clear also clears each feature that only its needs selected. A
feature of `--feature`, and a feature that a selected feature needs, cannot be cleared. A Space whose selection
`Select` refuses changes nothing, and the details show the message of `Select`.

**Acceptance Criteria:**
- [x] Space at a feature that is not selected selects it: its row shows `[x]`, its details show the source
  `answered`, and each feature that it needs, and their needs, show `[x]` and `needed by <features>`. The count of the
  title line follows.
- [x] Space at a feature that the developer selected and that no selected feature needs clears it, and also clears
  each feature that only its needs selected. A feature that the developer selected before stays: after Space at
  `api-auth` and then at `cli-auth`, a Space at `cli-auth` clears `cli-auth` and `cli`, and `api-auth` stays selected
  with the source `answered`.
- [x] Space at a feature that a selected feature needs changes nothing, whether only the need or also the developer
  selected it, and the details show a message that names the features that need it.
- [x] Space at a feature of `--feature` changes nothing, and the details say that the command line fixes it. The
  developer can select more features next to it.
- [x] Space at a feature whose selection `Select` refuses, for example two features that declare the same extension
  point, or a need that does not target the starter, leaves the selection and the tree as they were, and the details
  show the message of `Select`.
- [x] Space at the name row or at the starter row changes nothing.
- [x] The footer shows Space with its help.
- [x] The tests of `internal/form` drive these keys through `Update` and `View`.

**Affected Files/Modules:**
- `internal/form/model.go` — Space, the answered features, and the message of a feature that cannot change
- `internal/form/tree.go` — the rows after a new selection
- `internal/form/model_test.go` — a group `features` in `TestModel`

**Patterns to Follow:**
- `task:3` for the form, its tree and its tests
- `internal/prompt/model.go:240-251` for a new selection with the answered features
- `internal/catalog/starter.go:172-206` for the needs that `Select` adds, and its messages
- `internal/catalog/selection.go:341-364` for the source of a feature
- `internal/browser/model_test.go:107-207` for tests of the keys of a tree
- `tasks/learnings.md:198-203` for the relation `needed by` of every needed feature
- `tasks/learnings.md:177-182` for a test of a bubbletea model

**Testable:** Yes

**Certainty:** medium — `prompt.Model.withFeatures` (`internal/prompt/model.go:240-251`) makes a new selection from the
answered features; but there the TUI ends on an error of `Select`, while here a refused Space keeps the old
selection, and the rule that a needed feature or a feature of a flag cannot be cleared is new

**Blast radius:** low — none of the named categories; the form writes no file, and `Select` holds the needs

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form`.

**Depends on:** Task 3

---

### Task 5: Show the options of the starter and of each selected feature in the tree

**Behavior:** The tree shows each option of the selection with its value: the options of the starter under the
starter row, before its features, and the options of each selected feature under that feature (decision 4). A
selected feature with options opens and closes. An option that a flag gives is marked `fixed`, and an option with no
default and no value shows that its value is missing. The details of an option show its value, its default, its
values or its pattern, and its source. `Selection.Questions` gives each option of the selection for this, the
options of the flags included, each with its value, its source and whether a flag fixes it.

**Acceptance Criteria:**
- [x] Under the starter row, before the features, the tree shows each option of the starter in the order of the
  names, with its value. Under each selected feature, it shows each option of that feature in the order of the names,
  with its value. A feature that is not selected shows no option. Each option of the selection is in the tree one
  time, also when two selected features need its feature.
- [x] The options of a need come before the options of the feature that needs it.
- [x] A Space that selects a feature shows its options and those of its needs, and a Space that clears it removes
  them.
- [x] `l` and `h` open and close a selected feature with options, and a starter with options, as other rows.
- [x] An option whose value `--option` or `--feature` gives is marked `fixed`. An option with no default and no value
  shows that its value is missing.
- [x] The details of an option show its value, its default rendered with the name (`github.com/hpcsc/my-service`, and
  `github.com/hpcsc/` while the name is missing), its values in the order of the file or its pattern, and its source:
  `command line` or `default`. The title of the details pane is `starter › <option>` or `<feature> › <option>`.
- [x] In `internal/catalog`, `Selection.Questions` gives each option of the selection, also each option that a flag
  gives, in the order of the options of the preview. Each question gives its feature (none for an option of the
  starter), its option, its value, its source with the text of the preview, whether a flag fixes it, its default, its
  values and its pattern. `Check` acts as before. The group `questions` of `TestSelection` holds these: an option that
  a flag gives is a fixed question with the source `command line` and the value of the flag.
- [x] `internal/prompt` still asks only the options that no flag gives: its tests pass unchanged.
- [x] The tests of `internal/form` drive the options in the tree through `Update` and `View`.

**Affected Files/Modules:**
- `internal/catalog/question.go` — the feature, the value, the source, whether a flag fixes it, and the pattern
- `internal/catalog/option.go` — `questions` gives each option, the given ones included
- `internal/catalog/option_values.go` — the value and the source of one option
- `internal/catalog/selection_test.go` — the group `questions`
- `internal/prompt/model.go` — leaves out a fixed question
- `internal/form/model.go`, `internal/form/tree.go` — the option rows, their marks and their details
- `internal/form/model_test.go` — a group `options` in `TestModel`

**Patterns to Follow:**
- `task:3` and `task:4` for the form, its tree and its tests
- `internal/catalog/option.go:100-115` and `internal/catalog/question.go:8-31` for the questions of a selection
- `internal/catalog/option_values.go:24-50` for the value and the source of each option
- `internal/catalog/selection.go:78-84` and `internal/catalog/selection.go:139-145` for the options in the order of the
  preview
- `internal/browser/details.go:144-158` for the details of an option with its default, its values and its pattern
- `internal/catalog/selection_test.go:1162-1313` for the tests of the questions
- `internal/prompt/model.go:287-294` for the questions that the options step asks
- `tasks/learnings.md:205-210` for the words of options, values and given
- `tasks/learnings.md:135-140` for a loop over a map that stops at the first match
- `tasks/learnings.md:184-189` for tests in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `options.questions` (`internal/catalog/option.go:100-115`) gives the questions and
`optionValues.source` (`internal/catalog/option_values.go:42-50`) the source of each value; but no question yet gives
its value or its source, or holds an option that a flag gives, and the rows of the tree nest under their features

**Blast radius:** low — none of the named categories; the files and the preview do not change, and the form writes no
file

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog` and `internal/form`.

**Depends on:** Task 4

---

### Task 6: Edit the name and the options with Enter

**Behavior:** Enter at the name row or at an option row starts an edit (decision 9). The name and a text option take
typed text, checked after each key; an option with `values` lists its values in the details pane. Enter keeps a value
that passes its check and that `Select` accepts, and Esc drops the change. A kept value has the source `answered`. A
default that reads the name follows the name until the developer keeps a value for that option (decision 10). Enter
at a fixed row starts no edit.

**Acceptance Criteria:**
- [x] Enter at the name row starts an edit that holds the current name, and the screen shows the text as it is typed.
  While the text is edited, each key goes to it, also `q`, `j`, `k`, `h`, `l`, `w` and Space.
- [x] While `CheckRepoName` refuses the text, the details show its message, and Enter keeps nothing and the edit goes
  on. Enter with a text that it accepts keeps the name: the edit ends, the name row shows the name, and its details
  show the source `answered`. Esc drops the change, and the name row shows the name before the edit.
- [x] Enter at an option without `values` starts an edit that holds the current value. While `Question.Check` refuses
  the text, the details show its message, which names the option and its pattern, its missing default, or the comma
  that `--feature` cannot give, and Enter keeps nothing. Enter with a text that passes keeps the value.
- [x] Enter at an option with `values` lists its values in the details pane, in the order of the file, with the current
  value chosen. `j`, `k` and the arrow keys move, Enter keeps the chosen value, and Esc drops the change.
- [x] A kept value has the source `answered`, also when it equals the default, and the details show it.
- [x] An edit whose selection `Select` refuses is not kept, and the details show the message of `Select`.
- [x] Enter at the name of an argument, or at an option of a flag, starts no edit, and the details say that the value
  is fixed and name its source.
- [x] An option whose default reads the name follows the name until the developer keeps a value for it: with the
  default `github.com/hpcsc/{{ .Name }}`, keeping the name `my-service` shows `github.com/hpcsc/my-service`, and then
  keeping `other` shows `github.com/hpcsc/other`. After the developer keeps a value for that option, a new name does
  not change it.
- [x] A feature that a Space clears loses the values that the developer kept for it, also a need that the clear
  removes: after a new Space selects it, its options show their defaults.
- [x] ctrl+c stops the form during an edit, and `Confirmed` gives the cancel.
- [x] During an edit, the footer shows the keys of the edit: Enter keeps, Esc drops, ctrl+c stops.
- [x] The tests of `internal/form` drive these edits through `Update` and `View`.

**Affected Files/Modules:**
- `internal/form/model.go` — the edit of the name and of an option, the list of values, the answered values
- `internal/form/` — the edit, in the file of the type that declares it
- `internal/form/model_test.go` — a group `edit` in `TestModel`

**Patterns to Follow:**
- `task:5` for the option rows and their details
- `internal/browser/model.go:51-55` and `internal/browser/model.go:120-138` for a text input of bubbles whose keys all
  go to it, kept with Enter and dropped with Esc
- `internal/prompt/model.go:189-207` for the check of the name after each key
- `internal/prompt/model.go:296-335` for an option with `values` and a text option checked with `Question.Check`
- `internal/prompt/model.go:121-140` for the message of a check under the text
- `internal/catalog/answers.go:1-23` for the answers that `Select` takes
- `internal/catalog/starter.go:240-247` for the answered values of a feature that the selection does not hold
- `internal/prompt/model_test.go:382-534` for the tests of the options and their checks
- `tasks/learnings.md:170-175` for the keys that a component of bubbles binds by default
- `tasks/learnings.md:177-182` for the blink of the cursor in a test of a bubbletea model
- `tasks/learnings.md:247-252` for the attempts that a new kind of input costs

**Testable:** Yes

**Certainty:** low — the design of the edit is decided here: no TUI of this repo edits a value inside a tree, lists
values in the details pane, or keeps an answer that it must drop again when a default that reads the name follows a
new name; the filter of `show` is the only input that shares its keys with a tree

**Blast radius:** low — none of the named categories; an answer goes through the check that a flag goes through, and
the form writes no file

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form`.

**Depends on:** Task 5

---

### Task 7: Choose the starter with Enter

**Behavior:** Enter at the starter row lists the starters of the catalog, each with its summary, in the details pane.
Enter keeps the chosen starter, and Esc drops the change. A change of the starter keeps each feature that the
developer selected and that targets the new starter, and clears the others (decision 11). With no starter argument,
the developer chooses the starter here, and the tree then shows its options and its features. A starter that does
not fit the flags is not kept.

**Acceptance Criteria:**
- [x] Enter at the starter row lists each starter of the catalog in the details pane, in the order of the names, each
  with its summary, with the current starter chosen, or the first when none is chosen. `j`, `k` and the arrow keys
  move, Enter keeps the chosen starter, and Esc drops the change.
- [x] With no starter argument, a kept starter shows its options and its features in the tree, with the defaults that
  read the name.
- [x] A change of the starter keeps each feature that the developer selected and that targets the new starter, with
  the values that the developer kept for it, and clears each other feature. It drops the values that the developer
  kept for the options of the starter. The tree shows the features and the options of the new starter.
- [x] A starter whose selection `Select` refuses with the flags, for example with a `--feature` that does not target
  it or an `--option` that it does not declare, or whose features have needs that form a cycle, is not kept, the tree
  stays as it was, and the details show the message.
- [x] Keeping the current starter changes nothing.
- [x] Enter at the starter of an argument starts no list, and the details say that the starter is fixed and name its
  source.
- [x] The tests of `internal/form` drive the choice of the starter through `Update` and `View` with a catalog of two
  starters, with a feature that targets both and a feature that targets one.

**Affected Files/Modules:**
- `internal/form/model.go` — the list of the starters and the change of the starter
- `internal/form/model_test.go` — a group `starter` in `TestModel`

**Patterns to Follow:**
- `task:6` for a list in the details pane, kept with Enter and dropped with Esc
- `internal/prompt/model.go:154-187` for the starters with their summaries, and the starter with its definitions
- `internal/catalog/definitions.go:76-93` for the features that target a starter
- `internal/catalog/catalog.go:18-31` for the starter of a name
- `internal/browser/details.go:78-84` for names with their summaries in the details
- `internal/prompt/model_test.go:535-632` for the tests of the starter step
- `tasks/learnings.md:170-175` for the keys that a component of bubbles binds by default
- `tasks/learnings.md:247-252` for the attempts that a new kind of input costs

**Testable:** Yes

**Certainty:** medium — the list of `task:6` and the starters with their summaries of the starter step
(`internal/prompt/model.go:154-174`) give the parts; but no TUI here changes the starter of a selection and keeps the
features that still fit

**Blast radius:** low — none of the named categories; the form writes no file, and `Select` checks the flags against
the new starter

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form`.

**Depends on:** Task 6

---

### Task 8: Show the preview with `w`, and confirm the write

**Behavior:** `w` shows the text of `Selection.Preview` in the details pane and asks whether to write the repo
(decision 12). `y` stops the form, and `Confirmed` gives the selection of the preview; `n` and Esc go back to the
form. While an answer is missing, `w` selects its row and the details show the message.

**Acceptance Criteria:**
- [x] With no answer missing, `w` gives the details pane the title `Preview` and the focused border, and shows the text
  of `Selection.Preview` of the current selection, with the source of each feature and each option (`command line`,
  `answered`, `default`, `needed by <features>`), and the question `Write the repo?` with its keys.
- [x] ctrl+d and pgdown scroll the preview down, ctrl+u and pgup scroll it up, and the bottom border shows how far it
  is scrolled, so each line of a preview that is longer than the pane can be shown.
- [x] `y` stops the form, and `Confirmed` then gives the selection of the preview, with the answers of the developer.
- [x] `n` and Esc go back to the form, with the same row selected and its details. `q` and ctrl+c stop the form with
  the cancel, also while the preview shows.
- [x] An answer is missing when no starter is chosen, when the repo has no name, or when an option has no default and
  no value. Then `w` shows no preview: it selects the row of the first missing answer in the order of the tree, opens
  the rows above it, and the details show the message: of `CheckRepoName` for the empty name, of `Question.Check` for
  the empty text of the option, and a message that names the missing starter.
- [x] A preview that stops, for example on a template that fails to render, shows its message in the details pane
  with no question, and Esc goes back to the form.
- [x] The form writes no file. `Confirmed` gives a selection only after `y`.
- [x] The footer shows `w` with its help, and the keys of the preview while it shows.
- [x] The tests of `internal/form` drive `w`, `y`, `n` and Esc through `Update` and `View`, with a catalog that has an
  option with no default and a catalog with a template that fails to render.

**Affected Files/Modules:**
- `internal/form/model.go` — `w`, the preview, the question, the missing answers, and `Confirmed`
- `internal/form/model_test.go` — a group `write` in `TestModel`

**Patterns to Follow:**
- `task:7` for the form with every answer
- `internal/prompt/model.go:355-374` for the preview of the confirm step and its scroll
- `internal/prompt/model.go:142-147` and `internal/prompt/model.go:384-387` for `Confirmed` and the end of the form
- `internal/browser/model.go:140-157` and `internal/browser/model.go:189-193` for the scroll of the details pane and
  its note
- `internal/catalog/selection.go:65-76` and `internal/catalog/selection.go:206-217` for the preview and its stops
- `internal/catalog/question.go:20-31` for the message of an option with no default
- `internal/prompt/model_test.go:123-229` for the tests of the confirm step
- `tasks/learnings.md:177-182` for a `tea.QuitMsg` in a test of a bubbletea model

**Testable:** Yes

**Certainty:** medium — the confirm step (`internal/prompt/model.go:355-374`) shows the preview in a viewport and
`Confirmed` (`internal/prompt/model.go:142-147`) gives the selection after "Yes"; but here the preview takes the place
of the details, Esc goes back instead of a cancel, and a missing answer moves the selection to its row

**Blast radius:** high — a destructive write: `Confirmed` decides whether `new` writes the repo and runs the
`after_render` commands, `gofmt -w .` among them

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form`. Apply one break at a time,
read its diff, run the tests, and put the file back with `git checkout --`: give the selection after `n`; let `w`
show the preview with no name; drop the answers from the selection that `Confirmed` gives. Each break makes a test
fail.

**Depends on:** Task 7

---

### Task 9: Open the form in `new`, and remove the steps of huh

**Behavior:** In a terminal, without `--no-prompt` and without `--dry-run`, `new` opens the form of `internal/form`
in place of the steps of huh. After `y`, `new` writes the repo, runs the `after_render` commands, and prints the lines
and the command that it prints today. A cancel ends with the status 1 and writes nothing. `internal/prompt` and huh
go away (decision 1). The tests in a terminal, the e2e tests and the README tell the form.

**Acceptance Criteria:**
- [x] `new` opens the form only when stdin and stdout are both terminals, without `--no-prompt` and without
  `--dry-run`, with zero, one or two arguments. Else it starts no form and acts as today: the tests of
  `TestNewCommand` that run without a terminal pass unchanged.
- [x] In a terminal, `w` and then `y` make `new` write the repo, run the `after_render` commands, and print the
  `needed by` lines, `Made <output> from the starter <starter>.` and the command `feature-compose new` with
  `Selection.Arguments` of the output folder, with the status 0.
- [x] In a terminal, `q`, and ctrl+c, stop `new` with `you canceled new, so it wrote no file` on stderr and the status
  1, and `new` writes no file, makes no output folder and runs no command.
- [x] An unknown starter, and a flag that `Select` refuses, stop `new` before the form opens, with the message of `new`
  without the form.
- [x] The group `tui` of `TestNewCommand` types the keys of the form in a pseudo-terminal: `w` and `y` write the repo
  and print the command; `q` cancels; with no argument and with one argument the form opens and `q` cancels;
  `--no-prompt` and `--dry-run` open no form; a stdin or a stdout that is not a terminal opens no form.
- [x] No Go file imports `github.com/charmbracelet/huh`, `go.mod` does not require it, `internal/prompt` is gone, and
  `go mod tidy` leaves `go.mod` and `go.sum` as they are.
- [x] No exported name of `internal/catalog` is left without a caller outside its package, for example
  `Answers.WithValues` when the form does not call it.
- [x] The section "Answer the Questions in a TUI" of `README.md` tells the form: when `new` opens it; the tree and its
  rows; the keys (`j`, `k`, `l`, `h`, Esc, Space, Enter, `w`, `y`, `n`, ctrl+d, ctrl+u, `q`, and ctrl+c during an
  edit); the marks `fixed` and `needed by`; the rules of the needs; the edit with its check, keep and drop; the
  change of the starter; the default that follows the name; `w` with the preview and the missing answers; the cancel
  and its status; the printed command; and when `new` opens no form. The rows `answered` of the tables of the sources
  in "Preview a Repo" say that a value is answered when the developer keeps it with Enter, also when it is the
  default.
- [x] `e2e/tests/new.test.ts`, describe "feature-compose new in a terminal", on the catalog of this repo:
  - `new go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=api-key --option
    go-version=1.28 --option module=github.com/acme/my-service` shows the tree with `my-service`, the starter,
    `cli-auth` and `go-version` marked `fixed`, and `api-auth` and `cli` with `needed by cli-auth`. `w` shows
    `cli-auth: command line`, `api-auth: command line, needed by cli-auth`, `cli: needed by cli-auth` and
    `starter.go-version: 1.28 (command line)`, and ctrl+d scrolls until `go.mod: the starter` shows. Esc goes back to
    the tree, and `q` ends with a status other than 0, the message of the cancel, and a scratch folder with no entry.
  - `new go-event-sourcing-service my-service`: Space at `cli-auth` shows `api-auth` and `cli` with
    `needed by cli-auth`; Space at `api-auth` leaves it selected and shows a message that names `cli-auth`; Space at
    `cli-auth` clears the three. Space at `cli` and `w` show `cli: answered`.
  - `new go-event-sourcing-service my-service --feature cli-auth`: Enter at `go-version` and the text `1.20` show the
    pattern `^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$`, and Enter keeps the edit; the text `1.28` and Enter keep it. `w`
    shows `starter.go-version: 1.28 (answered)`, `starter.module: github.com/hpcsc/my-service (default)` and
    `api-auth.method: api-key (default)`. `q` ends with a status other than 0 and a scratch folder with no entry.
  - With a limit of 15 minutes, `new` with no argument: `w` selects the name row and shows the message of the name;
    Enter, `my-service` and Enter keep the name; Enter at the starter row shows `go-event-sourcing-service` with its
    summary, and Enter keeps it; the details of `module` show `github.com/hpcsc/my-service`; Space at `cli-auth`, `w`
    and `y` end with `EXIT:0`, and the screen shows `feature-compose new go-event-sourcing-service my-service
    --feature cli-auth`. That command with `--no-prompt`, run in a second scratch folder, writes a repo with the same
    files and the same contents.
  - With stdin from `/dev/null`, with stdout to `cat`, with `--no-prompt`, and with `--dry-run`, `new` opens no form.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the
  host.

**Affected Files/Modules:**
- `internal/cmd/new.go` — `ask` runs `form.Model`
- `internal/cmd/new_test.go` — the group `tui`
- `internal/prompt/` — removed
- `internal/catalog/answers.go` — exported names that lost their last caller go away
- `go.mod`, `go.sum` — huh and its own dependencies go away
- `README.md` — "Answer the Questions in a TUI", and the rows `answered` of "Preview a Repo"
- `e2e/tests/new.test.ts` — the describe "feature-compose new in a terminal"

**Patterns to Follow:**
- `task:8` for `form.New` and `Confirmed`
- `internal/cmd/new.go:29-108` for the start of the TUI, the write after it, and the printed command
- `internal/cmd/new_test.go:60-89` and `internal/cmd/new_test.go:612-732` for the tests in a pseudo-terminal
- `internal/cmd/show.go:56-67` for the run of a two-pane TUI
- `e2e/tests/new.test.ts:276-466` for the screen tests of `new` and the round trip of the printed command
- `e2e/tests/show.test.ts:43-102` for the screen tests of a tree with vim keys
- `e2e/testUtils.ts:71-90` and `docs/e2e-tests.md:175-192` for the rules of a screen test
- `README.md:74-89`, `README.md:104-218` and `README.md:239-265` for the sources, the TUI of `new`, and the keys of
  the TUI of `show`
- `tasks/learnings.md:233-238` for a test of `new` in a pseudo-terminal
- `tasks/learnings.md:240-245` for the late start of a command in tuistory
- `tasks/learnings.md:58-63` for the e2e tests on the host
- `tasks/learnings.md:37-42` for an e2e test that writes a repo with its own `mise.toml`
- `tasks/learnings.md:191-196` for the message of the cancel on stderr

**Testable:** Yes

**Certainty:** high — `ask` (`internal/cmd/new.go:98-108`) already runs a model and reads `Confirmed`, and the tests in
a pseudo-terminal (`internal/cmd/new_test.go:612-732`) and on the screen (`e2e/tests/new.test.ts:276-466`) exist for
the steps; this changes the model and the keys that they type, and removes code that loses its last caller

**Blast radius:** high — a destructive write: `new` writes the repo and runs the `after_render` commands after the form;
and a contract outside this repository: it decides when `new` waits for input, and a wrong check makes a script wait
on the form

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `go mod tidy` leaves `go.mod` and `go.sum` unchanged. `clerk mutants` finds no live mutant on the
changed lines of `internal/cmd` and `internal/catalog`. Run `e2e/tests/new.test.ts` and `e2e/tests/show.test.ts` on
the host. In a terminal, run the built binary with `new` in an empty temporary folder, answer each row, `w` and `y`;
then run the printed command with `--no-prompt` in another empty folder, and compare the two repos with `diff -r`. Run
the binary with `new go-event-sourcing-service my-service < /dev/null` in another empty folder: it opens no form.

**Depends on:** Task 8

## Summary

- **Tasks:** 9.
- **Order:** dependency first, and the shared parts before the form that uses them. Task 1 moves the look of `show` to
  `internal/screen`, proved by the tests of `show`. Task 2 is the one change of a rule of the catalog: a selection
  with no name, whose write and preview stop. Task 3 to Task 8 build the form in `internal/form` one behaviour at a
  time, each tested through `Update` and `View`: the tree with the flags, Space and the needs, the options, the edit
  of the name and the options, the choice of the starter, and `w`. During these tasks `new` keeps the steps of huh, so
  each commit leaves `new` usable in a terminal. Task 9 opens the form in `new` and removes huh in one commit, with
  the tests in a terminal, the e2e tests and the README.
- **Assessment:** Task 6 is low certainty, because it decides how a value is edited inside a tree and when an answer
  stops following the name. Task 2, Task 8 and Task 9 have a high blast radius: each decides whether, or where, `new`
  writes a repo and runs `gofmt -w .`, and Task 9 also decides when `new` waits for input.
- **Criteria of the story:**
  - "one form with two panes ... the header, the borders, the colors, and the key help of `show` ... a tree with the
    name of the repo, the starter, the features that target the starter, and each option under its starter or its
    feature ... the details of the selected row": Task 1 (the look), Task 3 (the panes, the tree, the details),
    Task 5 (the options), Task 9 (`new` opens it).
  - "`j` and `k` ... `l` and `h` ... Space ... Enter edits ... each key goes to it, Enter keeps the value, and Esc drops
    the change. ctrl+d and ctrl+u scroll the details": Task 3 (moves, open and close, scroll), Task 4 (Space), Task 6
    and Task 7 (Enter, the edit, keep and drop).
  - "Enter on the starter row lists the starters with their summaries. A change of the starter keeps each selected
    feature that targets the new starter, and clears the other features": Task 7.
  - "selects each feature that it needs and marks it ... cannot clear a needed feature ... A clear of the feature also
    clears each feature that only its need selected. The options of a need come before the options of the feature that
    needs it": Task 3 (the marks and the order of rule 8), Task 4 (Space and its rules), Task 5 (the options in that
    order).
  - "The tree shows an option only while its `when` is true, and it changes after each answer. The details of an
    option show its default, its values or its pattern, and its source. Enter on an option with `values` lists the
    values. A typed value that does not match the pattern shows a message ... An option whose default reads the name
    ... follows the name until the developer edits the option": Task 5 (the options of the selection after each
    answer, and their details), Task 6 (the list of values, the check, the default that follows the name). The clause
    about `when` is deferred to US-007, which adds `when`.
  - "Arguments and flags give answers before the form opens. The tree marks a value from an argument, a flag, or a need
    as fixed, and the details name its source. The developer cannot change a fixed value. A feature from `--feature`
    stays selected, and the developer can select more features. When a feature needs a value that differs from the
    value of a flag, the form does not select the feature, and the details show the message of `new`": Task 3 (the
    values of the arguments and the flags, the marks, the stops before the form opens), Task 4 (a feature of
    `--feature` stays, and a refused Space shows the message of `Select`), Task 5 (the options of a flag), Task 6 and
    Task 7 (a fixed value starts no edit). The values of a need, and the message of a need against a flag, are
    deferred to US-007, because no need sets a value today; the message then shows through the path of Task 4.
  - "`w` shows the output of `--dry-run` in the right pane and asks whether to write ... Esc goes back ... While an
    answer is missing ... `w` selects that row and the details show the message. `q` stops the form and writes
    nothing, except while a value is typed, where ctrl+c stops it": Task 2 (a selection with a missing name), Task 3
    (`q`), Task 6 (ctrl+c during an edit), Task 8 (`w`, the missing answers, the confirm), Task 9 (the write and the
    cancel of `new`).
  - "After the tool writes the repo, it prints the command that makes the same repo without the TUI, as in US-012. With
    `--no-prompt`, with `--dry-run`, or when stdin is not a terminal, the form does not open": Task 9, which keeps the
    printed command of US-012 and its start rules.
  - Context, one set of keys for the two TUIs, and huh out of the TUI and `go.mod` (D12): Task 1 and Task 9.
