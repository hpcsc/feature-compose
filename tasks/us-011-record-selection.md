# Tasks: Record the Selection in the Repo (US-011)

## Contents

1. Task 1: Write `.feature-compose.yml` with the version, the starter, the name and the options of the starter
2. Task 2: Write each feature with its options, and the requested features, to `.feature-compose.yml`
3. Task 3: List `.feature-compose.yml` in the preview, and stop a template that writes its path

## Story Reference

US-011, "Record the selection in the repo", in `user-stories/feature-compose.md:382-401` (Description, Acceptance
Criteria and Context), goal G7. The design is `PROPOSAL.md`: section 13, "The Selection File"
(`PROPOSAL.md:448-473`), which shows the file and tells each field; step 6 of `new` in section 11, "It writes the
selection file" (`PROPOSAL.md:378-383`); section 8, the sources of the value of an option (`PROPOSAL.md:263-268`);
section 12, the command that `new` prints after the TUI (`PROPOSAL.md:427-431`); rule 7, one owner for each file
(`PROPOSAL.md:342`); and decision D8, the catalog is a folder (`PROPOSAL.md:555`). US-010
(`tasks/completed/us-010-dry-run.md`) made the preview, whose paths are the paths of the files that `new` writes.

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

- A command that reads `.feature-compose.yml`, for example `add` or an update of a repo. This story writes the
  file, and no code of the tool reads it.
- The source of each value (`command line`, `need of <features>`, `answered`, `default`) in the selection file. The
  file holds the values, as section 13 shows it, and the preview shows the sources. `requested` is the one record of
  where a feature came from.
- The output folder (`--output-dir`) in the selection file. It is not part of a selection (`PROPOSAL.md` section 4).
- A version of the catalog in the selection file. The catalog is a folder with no version (decision D8).
- A selection file from `--dry-run`. The dry run writes no file (US-010); its preview names the file (Task 3).
- A mistake of `check` for a template that writes `.feature-compose.yml` at the root of the repo. `new` and
  `--dry-run` stop on it before the first file (Task 3), and no story names it for `check`.
- A flag that sets an option of a feature and does not select the feature, so that the command that `new` prints
  after the form keeps a feature that only a need added out of `requested` (decision 6).

**Deferred:** none.

## Codebase Context

**The steps of `new`.** `newNewCommand` (`internal/cmd/new.go:16-88`) parses `--option` and `--feature` into the
given values (`internal/cmd/new.go:110-145`), loads the catalog, and gets a `Selection` from `Starter.Select`
(`selected`, `internal/cmd/new.go:90-96`) or, in a terminal, from the form (`ask`, `internal/cmd/new.go:98-108`). With
`--dry-run` it prints `Selection.Preview` and stops (`internal/cmd/new.go:58-65`). Else it calls `Selection.Write`
with the output folder, then `Selection.RunAfterRender`, then prints a line for each feature that a need added and
`Made <output> from the starter <starter>.`, and after the form the command that makes the same repo
(`internal/cmd/new.go:66-85`). No step writes a selection file. The root command gives `newNewCommand` the load of
the catalog as an argument (`internal/cmd/root.go:27-41`).

**What a selection holds.** `Selection` (`internal/catalog/selection.go:22-29`) holds the name of the repo, the
starter as `selectedStarter` (its name, its options and its values, `internal/catalog/selection.go:351-367`), and the
features as `selectedFeature`, in the order of rule 8 (`internal/catalog/selection.go:369-422`). The values of each
are an `optionValues` (`internal/catalog/option_values.go:10-31`), whose `byName` holds, for each declared option,
the value that the templates read. `options.withDefaults` (`internal/catalog/option.go:64-109`) takes that value from
a flag, a need, an answer of the form, or the default, in that order, and renders a default with the name of the
repo. So each value of the selection file is at hand, also a default and a value that a need sets. `Name` and
`StarterName` exist (`internal/catalog/selection.go:31-37`). `Combination` (`internal/catalog/combination.go:9-13`)
names the same parts `StarterName`, `StarterValues` and `FeatureValues`.

**The requested features.** `selectedFeature.requested` is true when a `--feature` flag gives the feature (`given`)
or the form selects it (`answered`) (`internal/catalog/selection.go:393-395`). `Starter.Select` sets `given` from
`givenByFeature` and `answered` from `Answers.Features` (`internal/catalog/starter.go:98-114`). `AddedByNeeds` gives
each other feature (`internal/catalog/selection.go:39-47`). A feature that a flag gives and another selected feature
needs is requested, and the preview shows it as `command line, needed by <features>`
(`internal/catalog/selection.go:383-391`).

**The form path.** Space at a feature adds it to `Answers.Features` (`draft.toggled`,
`internal/form/draft.go:74-92`), and the draft selects again with those answers (`internal/form/draft.go:63-72`).
`Model.Confirmed` returns that selection (`internal/form/model.go:109-114`), and `new` then takes the same steps as
without the form. So a feature that the form selects reaches `new` as `answered`, and `requested` is right for the
form with no change in `internal/form`. A value that the form answers for a feature that only a need added does not
select the feature. But the command that `new` prints after the form names such a feature with
`--feature <name>:<option>=<value>` (`Selection.featuresOfArguments`, `internal/catalog/selection.go:162-174`), and
`--feature` selects it, so the repo of that command lists the feature in `requested` (decision 6). The test of
`TestNewCommand` that runs `new` with the arguments of a selection has this case: answers that select `a` and `w`,
and a value of `b`, which only a need adds (`internal/cmd/new_test.go:343-379`).

**The version.** `version.Current` (`internal/version/version.go:18-50`) gives the tag that the release build sets
with `-ldflags` (the e2e binary is built with `v0.1.0`, `tasks/learnings.md:59-65`), else the module version of a
tagged build, else the short commit with `-dirty` for a build with changes, else `unknown`.
`feature-compose version` and `--version` print it (`internal/cmd/root.go:31`, `internal/cmd/root.go:43-52`), and
`TestRootCommand` compares `--version` with `version.Current()` and not with a fixed text
(`internal/cmd/root_test.go:119-128`). `internal/catalog` does not import `internal/version`. Section 13 shows
`version: 0.1.0`, without the `v` of a tag.

**Files and YAML.** `file.write` (`internal/catalog/file.go:16-32`) writes one file under the output folder, makes
its folders, and sets the mode from the contents. `Selection.Write` (`internal/catalog/selection.go:64-78`) renders
each template through `files` (`internal/catalog/selection.go:200-228`), which stops on two templates that write one
path with a message that names both, then refuses an output folder that has entries, and then writes. No code of the
tool writes YAML: `starter.yml`, `feature.yml` and the config file are read with `go.yaml.in/yaml/v3`
(`internal/catalog/starter_yml.go:31`, `internal/config/file.go:33`), and `show` writes JSON with an indent of two
spaces (`internal/cmd/show.go:49-51`). `internal/catalog/definitions.go:43-64` has exported fields with `yaml` tags.
The YAML encoder of `go.yaml.in/yaml/v3` writes the keys of a map in the order of the names, writes an empty map as
`{}` and an empty list as `[]`, and quotes a text that YAML would read as another type, such as `1.28`, `true`,
`on` or `n`.

**The preview.** `Selection.Preview` (`internal/catalog/selection.go:80-91`) lists each file that `files` gives, in
the order of the paths, with its owner (`fileLines`, `internal/catalog/selection.go:192-198`), and the confirm step of
the form shows the same text (`internal/form/model.go:144-157`). US-010 made the paths of the preview the paths of
the files that `new` writes (`tasks/completed/us-010-dry-run.md:182-184`). Two tests of `TestSelection` compare the
paths of the preview with the files of `Write` (`internal/catalog/selection_test.go:1003-1015`,
`internal/catalog/selection_test.go:1877-1902`). The owner text is `the starter` or `the feature <name>`
(`internal/catalog/template.go:57-61`).

**Tests.** The unit tests have the tag `unit` and run with `task test:unit`. `TestSelection` makes selections from a
catalog in `fstest.MapFS` and writes them to `t.TempDir()` (`internal/catalog/selection_test.go:25-67`); its group
`write` calls `Write` about 60 times and compares the whole map of written files. `TestNewCommand` runs `new` with
`runNew` and, in a pseudo-terminal, with `runInTerminal` (`internal/cmd/new_test.go:27-90`). About 16 of its tests
compare the whole map of a written repo with `filesIn`, and two compare the repo of `new` with the repo of `Selection.Write`
(`internal/cmd/new_test.go:343-379`, `internal/cmd/new_test.go:409-437`). The form test of `TestModel` compares the
whole text of the preview (`internal/form/model_test.go:870-876`). The e2e test of `cli-auth` makes a real repo with
`--no-prompt` (`e2e/tests/new.test.ts:354-370`). The form test with no argument compares the repo of the form with
the repo of the printed command, file by file, dot files included (`e2e/tests/new.test.ts:662-698`, `filesOf` at
`e2e/tests/new.test.ts:746-755`), so it compares the two selection files with no change. Docker answered on
2026-11-13; the e2e tests also run on the host (`tasks/learnings.md:59-65`).

**Docs.** `docs/commands.md` tells the steps of `new` (`docs/commands.md:15-30`), the preview
(`docs/commands.md:32-96`), and that the command that `new` prints after the form makes the same files
(`docs/commands.md:189-199`). `docs/catalog.md` tells the rule of one owner for each file (`docs/catalog.md:42-45`).

**Words.** `selection file` is the word of `PROPOSAL.md` for `.feature-compose.yml` (the heading of section 13 and
step 6 of `new`), and of the delivery order of the stories (`US-011 Selection file`); no Go code uses it or
`record`. `config file` is taken: the file that names the catalog (`internal/config`). `selected` is taken for each
feature of a selection (`selectedFeature`), so the features that the developer named are `requested`, the key of
section 13 and the word of `selectedFeature.requested`. `values` is the word for the chosen values, and `options` for
the declared options (`tasks/learnings.md:206-212`); the keys `options` and `features` of the file are the keys of
section 13, and the Go names that hold them follow `Combination`. `owner` is the starter or the feature whose
template writes a file (`template.owner`). `version` is the text of `version.Current`.

**Decisions of this breakdown, for the caller to overrule:**

1. **`new` writes the selection file after the commands of each folder, with a method of its own,
   `Selection.WriteSelectionFile`,** as step 6 of `new` in `PROPOSAL.md:383`. `Selection.Write` keeps its files, so
   its about 60 tests in `TestSelection` keep their expected values. The cost: an `after_render` command that fails
   stops `new`, and that repo then has no selection file. The other choice, a selection file that `new` writes with
   the files and before the commands, keeps a record in such a repo, and puts the file in a commit that a command
   makes; it changes the signature of `Write` and each map of written files in `TestSelection`.
2. **`version` is the text that `feature-compose version` prints**: `v0.1.0` for a release, the short commit for a
   build of a commit with no tag, `unknown` for a build with no information of version control. Section 13 shows
   `0.1.0`; this choice gives one text for one version in the output of `version` and in the file.
   `internal/catalog` takes the version from its caller and does not read the build information.
3. **The layout of the file:** the keys in the order of section 13 (`version`, `starter`, `name`, `options`,
   `features`, `requested`), each key also when it is empty (`options: {}`, `features: {}`, `requested: []`, and
   `cli: {}` for a feature with no option), an indent of two spaces, and the options of the starter, the features,
   the options of each feature and the requested features in the order of their names. So each order of the flags
   gives the same bytes. The features are in the order of the names and not of rule 8: the example of section 13
   follows neither order, and a reader looks a feature up by its name. Each value reads back with a YAML parser as
   the same text.
4. **The values are the values that the templates read**, from each source, with no source in the file.
5. **`requested` is each feature that a `--feature` flag gives or that the form selects, also when another selected
   feature needs it** (`tasks/learnings.md:199-205`). It is `selectedFeature.requested`, so the rule has one owner
   for `new`, the preview and the selection file.
6. **The command that `new` prints after the form can make a selection file with more requested features.** When an
   answer gives a value to a feature that only a need added, the command names that feature with `--feature`, which
   selects it. The rendered files are the same, and `docs/commands.md` says that the selection file of that repo also
   lists the feature in `requested`. A flag that sets a value with no selection is out of scope.
7. **`--dry-run` and the confirm step of the form list `.feature-compose.yml` with the owner `feature-compose`**,
   because `new` writes it: US-010 made the paths of the preview the paths of the files that `new` writes, and
   section 11 says that `--dry-run` prints the files. `--dry-run` still writes no file.
8. **The path `.feature-compose.yml` at the root of the repo has the owner feature-compose (rule 7).** A template
   that renders to it stops `new` and `--dry-run` before the first file, with the message of two owners. With no
   stop, the preview would list the path two times, and the selection file would replace the file of the template.

## Tasks

### Task 1: Write `.feature-compose.yml` with the version, the starter, the name and the options of the starter

**Behavior:** After `new` writes the files of the repo and runs the commands of each folder, it writes
`.feature-compose.yml` to the root of the output folder (decision 1). The file has, in this order, `version`, the
text that `feature-compose version` prints (decision 2); `starter`, the name of the starter; `name`, the name of the
repo; and `options`, each option of the starter by its name with the value that the templates read, also a default
(decision 3). `Selection.WriteSelectionFile` writes it with the version that its caller gives. Task 2 adds the
features and `requested`, and Task 3 adds the file to the preview.

**Acceptance Criteria:**
- [x] `new <starter> <name>` writes `.feature-compose.yml` to the root of the output folder (`./<name>`, or the path
  of `--output-dir`), also when the form gives the selection. The file is YAML with the keys `version`, `starter`,
  `name` and `options`, in this order, with an indent of two spaces, as `PROPOSAL.md` section 13 shows them.
- [x] `version` is the text that `feature-compose version` prints for the same binary. `starter` and `name` are the
  starter and the name of the repo that `new` makes, from its arguments or from the form.
- [x] `options` holds each option of the starter by its name, in the order of the names, with the value that the
  templates read: the value of `--option`, the value that the form answers, or the default, rendered with the name
  of the repo (for example `github.com/hpcsc/my-service` for the default of `module`). A starter with no option gives
  an empty map.
- [x] Each value reads back with a YAML parser as the same text, also a value that YAML would read as another type,
  for example `1.28`, `true`, `on`, or the empty text.
- [x] The two orders of two `--option` flags give the same bytes in `.feature-compose.yml`.
- [x] `new` writes the file after the last `after_render` command (`PROPOSAL.md:383`): an `after_render` command that
  lists the files of the repo does not list `.feature-compose.yml`, the repo has the file after `new`, and an
  `after_render` command that fails stops `new` before it writes the file.
- [x] `Selection.WriteSelectionFile` writes `.feature-compose.yml` with the version that its caller gives, writes no
  other file, and leaves each other file of the output folder as it was. An output path that it cannot write, for
  example a path that is a file, stops it with an error that names `.feature-compose.yml`.
- [x] `--dry-run` writes no selection file, and a stop of `new` before its first file makes no output folder: the
  tests of the groups `dry run` and `stop` of `TestNewCommand` pass with their expected values unchanged.
- [x] `Selection.Write` writes the same files as before: the group `write` of `TestSelection` passes with its
  expected values unchanged. Each test of `TestNewCommand` that compares the whole map of a written repo also expects
  `.feature-compose.yml`, and the other files of each map keep their expected contents.
- [x] `docs/commands.md`: "Make a Repo" says that `new` writes `.feature-compose.yml` after the commands of each
  folder. A section "The Selection File" tells the keys `version`, `starter`, `name` and `options`; that `version` is
  the text of `feature-compose version` and not a version of the catalog (decision D8); and that `options` holds each
  value of an option of the starter, also a default.
- [x] `e2e/tests/new.test.ts`: the test that makes a repo with `cli-auth` finds in `my-service/.feature-compose.yml`
  the version of the e2e build (`buildTag()`, `v0.1.0`), the starter `go-event-sourcing-service`, the name
  `my-service`, and the options `go-version` with `1.27.1` and `module` with `github.com/hpcsc/my-service`.

**Affected Files/Modules:**
- `internal/catalog/selection_file.go` — the new type `selectionFile`: the contents of `.feature-compose.yml` for one
  selection
- `internal/catalog/selection.go` — `Selection.WriteSelectionFile`
- `internal/catalog/selection_test.go` — a group `write selection file` in `TestSelection`
- `internal/cmd/new.go` — `new` writes the selection file, with the version, after `RunAfterRender`
- `internal/cmd/root.go` — only if `new` takes the version from the root command
- `internal/cmd/new_test.go` — the selection file in the groups `write` and `tui`, and `.feature-compose.yml` in each
  whole map of a written repo
- `internal/cmd/root_test.go` — the version in the selection file of a repo that `new` makes through the root command
- `docs/commands.md` — "Make a Repo" and "The Selection File"
- `e2e/tests/new.test.ts` — the selection file of the repo with `cli-auth`

**Patterns to Follow:**
- `internal/cmd/new.go:66-85` for the steps of `new` after the selection
- `internal/catalog/selection.go:64-78` and `internal/catalog/file.go:16-32` for a file of the repo under the output
  folder
- `internal/catalog/combination.go:9-13` for the names `StarterName` and `StarterValues`
- `internal/catalog/definitions.go:43-64` for exported fields with `yaml` tags
- `internal/catalog/option_values.go:10-31` for the value that the templates read, by the name of the option
- `internal/cmd/show.go:49-51` for an encoder with an indent of two spaces
- `internal/cmd/root.go:27-52` and `internal/cmd/root_test.go:119-128` for `version.Current` in a command and in its
  test
- `internal/cmd/root_test.go:20-37` for `new` through the root command, with a config file and a catalog folder
- `internal/catalog/selection_test.go:25-67` for a selection of a catalog in `fstest.MapFS`
- `internal/cmd/new_test.go:27-58` for `starterFiles`, `loaded` and `runNew`
- `internal/cmd/new_test.go:309-329` for the values of `--option`
- `internal/cmd/new_test.go:385-407` for an `after_render` command that makes a file, and one that fails
- `internal/cmd/new_test.go:628-640` for the form path that writes the repo
- `e2e/tests/new.test.ts:354-370` and `e2e/testUtils.ts:17-23` for the repo with `cli-auth` and `buildTag`
- `tasks/learnings.md:3-8` for a test that runs a shell command, which starts in a temporary folder
- `tasks/learnings.md:185-191` for a test of the selection file in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:73-79` for exported fields that a library reads, which the design names
- `tasks/learnings.md:206-212` and `tasks/learnings.md:430-435` for the words and the names of `internal/catalog`
- `tasks/learnings.md:220-226` for a test of `new` in a pseudo-terminal
- `tasks/learnings.md:59-65` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `Selection.Write` and `file.write` (`internal/catalog/selection.go:64-78`,
`internal/catalog/file.go:16-32`) write the files of a repo, and `Combination` holds the same parts; but no code of
the tool writes YAML, and no file of the repo comes from the tool and not from a template

**Blast radius:** high — a contract consumed outside this repository: the file goes into each repo that `new` makes
and stays there, so a wrong key, order or form of a value outlives its fix, and phase 3 reads it; the task also adds
a write step to `new`

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd`. Apply one break at a time, read its diff, run the tests,
and put the file back with `git checkout --`: let `new` write the selection file before `RunAfterRender`; give the
selection file a fixed version text; write the options of the starter without the default of an option. Each break
makes a test fail. Run `e2e/tests/new.test.ts` (Docker, or the host run of `tasks/learnings.md:59-65`). In an empty
temporary folder, run the built binary with
`new go-event-sourcing-service my-service --option go-version=1.28 --no-prompt`, and read
`my-service/.feature-compose.yml`.

**Depends on:** None

---

### Task 2: Write each feature with its options, and the requested features, to `.feature-compose.yml`

**Behavior:** The selection file gets `features` and then `requested`, after `options`. `features` holds each feature
of the selection, also a feature that only a need added, with each of its options and the value that the templates
read (decision 4). `requested` lists each feature that a `--feature` flag gives or that the form selects, also when
another selected feature needs it (decision 5). A feature that only a need added is in `features` and not in
`requested`.

**Acceptance Criteria:**
- [ ] `.feature-compose.yml` has `features` and then `requested` after `options`, as `PROPOSAL.md` section 13 shows
  them.
- [ ] `features` holds each feature of the selection one time, by its name, in the order of the names: each feature
  that a flag gives or that the form selects, and each feature that a need added. Each feature holds each of its
  options by its name, in the order of the names, with the value that the templates read: from
  `--feature <name>:<option>=<value>`, from the need of another feature, from an answer of the form, or the default.
  A feature with no option holds an empty map, as `cli: {}` in section 13. A selection with no feature gives an empty
  map.
- [ ] `requested` lists, in the order of the names, each feature that a `--feature` flag gives or that the form
  selects, also when another selected feature needs it. A feature that only a need added is in `features` and not in
  `requested`, also when an answer of the form gives a value to one of its options. A selection with no requested
  feature gives an empty list.
- [ ] In a terminal, a feature that space selects in the form is in `requested` of the repo that `new` writes, and
  each feature that only its needs add is in `features` and not in `requested`.
- [ ] The two orders of the `--feature` flags of a feature and its need, with the defaults and with a value of an
  option, give the same bytes in `.feature-compose.yml`.
- [ ] `new` with the arguments that `Selection.Arguments` gives for a selection, and `--no-prompt`, writes the files
  of that selection and its selection file, with one difference: a feature that only a need added and whose option an
  answer gives a value is in `requested` of the repo of the arguments, because the arguments name it with
  `--feature` (decision 6).
- [ ] `docs/commands.md`: "The Selection File" tells `features` and `requested`, with the file of
  `new go-event-sourcing-service my-service --feature cli-auth` as its example. The text about the command that `new`
  prints after the form says that the selection file of its repo also lists in `requested` each feature that the
  command names only to give a value to its option.
- [ ] `e2e/tests/new.test.ts`: in the test that makes a repo with `cli-auth`, `.feature-compose.yml` holds `api-auth`
  with `method` `api-key`, and `cli` and `cli-auth` with no option, and lists only `cli-auth` under `requested`. In
  the form test with no argument, the selection file of the repo of the form lists only `cli-auth` under `requested`,
  and it is the same file as the selection file of the repo of the printed command.

**Affected Files/Modules:**
- `internal/catalog/selection_file.go` — `features` and `requested`
- `internal/catalog/selection.go` — the features and the requested features of the selection file
- `internal/catalog/selection_test.go` — the features and `requested` in the group of the selection file, from
  flags, from the answers of the form, and from needs
- `internal/cmd/new_test.go` — the features and `requested` of a repo that `new` makes from flags and from the form,
  the two orders of the flags, and the repo of the arguments of a selection
- `docs/commands.md` — `features` and `requested` in "The Selection File", and the selection file of the command that
  `new` prints after the form
- `e2e/tests/new.test.ts` — the features and `requested` of the repo with `cli-auth` and of the repo of the form

**Patterns to Follow:**
- `internal/catalog/selection.go:369-422` for `selectedFeature`, its values and `requested`
- `internal/catalog/selection.go:146-160` for the features and the options of a selection in the preview
- `internal/catalog/starter.go:98-114` for `given` and `answered` in `Select`
- `internal/catalog/option.go:64-109` for the value of a flag, a need, an answer and a default
- `internal/catalog/selection.go:162-174` for the features that `Arguments` names
- `internal/catalog/selection_test.go:2030-2041` for selections with given and answered features
- `internal/catalog/selection_test.go:944-971` for answers that select a feature, with a need and an answered value
- `internal/cmd/new_test.go:106-117` for `withNeeds`
- `internal/cmd/new_test.go:267-286` for the two orders of the flags of a feature and its need
- `internal/cmd/new_test.go:343-379` for the repo of the arguments of a selection
- `internal/cmd/new_test.go:59-90` and `internal/cmd/new_test.go:628-640` for a test of the form in a pseudo-terminal
- `internal/form/draft.go:74-92` for a feature that space selects
- `e2e/tests/new.test.ts:354-370` and `e2e/tests/new.test.ts:662-698` for the repo with `cli-auth` and the repo of
  the form
- `task:1` for the selection file and its group in `TestSelection`
- `tasks/learnings.md:199-205` for a feature that a flag gives and another feature needs
- `tasks/learnings.md:304-310` for a test of `new` that reaches the new path of `internal/catalog`
- `tasks/learnings.md:311-317` for a rule about one feature on the type of the feature
- `tasks/learnings.md:206-212` and `tasks/learnings.md:430-435` for the words and the names of `internal/catalog`
- `tasks/learnings.md:220-226` for a test of `new` in a pseudo-terminal
- `tasks/learnings.md:59-65` for the e2e tests on the host

**Testable:** Yes

**Certainty:** high — `selectedFeature.requested` (`internal/catalog/selection.go:393-395`) already decides the
requested features for flags and for the form, and the preview lists the same features and values
(`internal/catalog/selection.go:146-160`); Task 1 gives the file

**Blast radius:** high — a contract consumed outside this repository: `features` and `requested` are what a reader of
the repo, and phase 3, take as the choices that made the repo, and a wrong list stays in each repo that `new` made
with it

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/catalog` and `internal/cmd`. Apply one break at a time and put the file back with `git checkout --`: list
each feature under `requested`; leave out the features that a need added; let `requested` take only the features
that a flag gives. Each break makes a test fail. Run `e2e/tests/new.test.ts` on the host or in Docker. In an empty
temporary folder, run the built binary with `new go-event-sourcing-service my-service --feature cli-auth --no-prompt`,
and read `my-service/.feature-compose.yml`.

**Depends on:** Task 1

---

### Task 3: List `.feature-compose.yml` in the preview, and stop a template that writes its path

**Behavior:** The preview, which `--dry-run` prints and the confirm step of the form shows, lists
`.feature-compose.yml` with the owner `feature-compose` among the files, in the order of the paths, because `new`
writes it (decision 7). The path `.feature-compose.yml` at the root of the repo has one owner, feature-compose: a
template of the starter or of a feature that renders to it stops `new` and `--dry-run` before the first file, with the
message of two owners (decision 8).

**Acceptance Criteria:**
- [ ] The preview lists `.feature-compose.yml` one time, with the owner `feature-compose`, among the files in the
  order of the paths. A starter with no template previews a repo with this one file.
- [ ] The paths of the preview are the paths of the files that `new` writes for the same arguments: the files of
  `Selection.Write` and `.feature-compose.yml`. The tests of `TestSelection` that compare the paths of the preview
  with the written files also write the selection file.
- [ ] The confirm step of the form shows the line `.feature-compose.yml: feature-compose`, as its preview is the text
  of `--dry-run`.
- [ ] A template of the starter or of a feature whose path in the repo, after the render, is `.feature-compose.yml`
  stops `new` and `--dry-run` before the first file, with the same message and a status other than 0. The message
  names the template, its owner and `feature-compose`, and says that both write `.feature-compose.yml`. `new` makes
  no output folder. A template that writes `.feature-compose.yml` in a folder of the repo below its root is a file
  like any other.
- [ ] `docs/commands.md`: "Preview a Repo" names `.feature-compose.yml` with the owner `feature-compose`, in its text
  and in its example. `docs/catalog.md`: the rule of one owner for each file says that `.feature-compose.yml` at the
  root of the repo belongs to feature-compose, and that a template that writes it stops `new`.
- [ ] `e2e/tests/new.test.ts`: the `--dry-run` test of `go-event-sourcing-service` with `cli` finds
  `.feature-compose.yml: feature-compose` among the files, and the scratch folder holds no entry after the run.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — `.feature-compose.yml` in the files of the preview and in the check of two owners
- `internal/catalog/selection_test.go` — the groups `preview`, `write` and `write and preview`
- `internal/cmd/new_test.go` — the group `dry run`, and the stop of `new` and `--dry-run` on a template that writes
  `.feature-compose.yml`
- `internal/form/model_test.go` — the expected text of the preview of the form
- `docs/commands.md` — the selection file in "Preview a Repo"
- `docs/catalog.md` — the owner of `.feature-compose.yml`
- `e2e/tests/new.test.ts` — the selection file in the preview of `--dry-run`

**Patterns to Follow:**
- `internal/catalog/selection.go:200-228` for the check of two templates that write one path, before the first file
- `internal/catalog/selection.go:192-198` for the files of the preview with their owners
- `internal/catalog/template.go:57-61` and `internal/catalog/template.go:120-129` for the owner text and the
  description of a template in a message
- `internal/catalog/selection_test.go:246-260` and `internal/catalog/selection_test.go:507-523` for two owners of one
  path
- `internal/catalog/selection_test.go:974-1053` for the group `preview`
- `internal/catalog/selection_test.go:1877-1902` for the group `write and preview`
- `internal/cmd/new_test.go:439-449` for the whole text of `--dry-run`
- `internal/cmd/new_test.go:530-568` for each stop of `new` that stops `--dry-run` with the same message
- `internal/form/model_test.go:870-876` for the preview of the form
- `e2e/tests/new.test.ts:405-415` for the files of the preview in an e2e test
- `task:1` for the selection file and the step of `new` that writes it
- `tasks/learnings.md:185-191` for a test of the preview in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:304-310` for a test of `new` that reaches the new path of `internal/catalog`
- `tasks/learnings.md:430-435` for the names of `internal/catalog`
- `tasks/learnings.md:59-65` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `Selection.files` (`internal/catalog/selection.go:200-228`) stops on two templates that write
one path, and `fileLines` lists the files with their owners; but the selection file is not a template, so the check
and the list get one path that no template renders

**Blast radius:** low — none of the named categories: the preview is text for a person (US-010 decision 7), and the
new stop comes before the first file, for a path that no template of the catalog writes

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/catalog` and `internal/cmd`. Apply one break at a time and put the file back with `git checkout --`: drop
the line of the selection file from the preview; drop the stop on a template that writes `.feature-compose.yml`;
compare the path of the template before the render. Each break makes a test fail. Run `e2e/tests/new.test.ts` on the
host or in Docker.

**Depends on:** Task 1

## Summary

- **Tasks:** 3.
- **Order:** dependency first, and then risk. Task 1 fixes the file, its place in the steps of `new`, its layout and
  its version. Task 2 adds `features` and `requested`, whose rule is the part of the story that a reader of a repo
  depends on: a feature that a flag gives and a need also adds, the form path, and the command that `new` prints
  after the form. Task 3 then puts the file into the preview and into the rule of one owner; it depends only on
  Task 1.
- **Assessment:** no task is low certainty. The story uses again the values of a selection, the rule of the
  requested features that US-010 and US-012 made, and the YAML library that the tool already reads with; it adds no
  dependency, integration or algorithm. Task 1 and Task 2 have a high blast radius, because the file stays in each
  repo that `new` makes; Task 3 is low.
- **Criteria of the story:**
  - "`new` writes `.feature-compose.yml` to the root of the repo ... the version of feature-compose, the starter, the
    name, the options of the starter, and each feature with its options": the version, the starter, the name and the
    options in Task 1; the features with their options in Task 2.
  - "The file holds each value that the repo uses, also a value that came from a default or from a need": the
    values of the starter, also a default, in Task 1; the values of each feature, also a default and a value that a
    need sets, in Task 2.
  - "`requested` lists only the features that the developer named with `--feature` or selected in the TUI": Task 2,
    for flags and for the form, also for a feature that a flag gives and another feature needs.
  - "The same selection gives the same file, in each order of the flags": the order of `--option` in Task 1, and
    the order of `--feature` in Task 2.
  - Context, "the version of feature-compose is not the version of the catalog": Task 1 (decision 2), in the file
    and in `docs/commands.md`.
  - Task 3 is not a criterion of the story. It keeps the promise of US-010 that the preview lists each file that
    `new` writes, and rule 7 for the one path that no template owns (decisions 7 and 8).
