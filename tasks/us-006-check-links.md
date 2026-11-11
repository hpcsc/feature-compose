# Tasks: Check the Links Between Features and Starters (US-006)

## Contents

1. Task 1: Add `check`, which finds an `add_to` that no target or need declares, and an item without the fields of its extension point
2. Task 2: Find a folder that a target does not have, and targets that declare one extension point with other item fields
3. Task 3: Find a need that does not exist, does not target each target, or sets a value that its option does not have
4. Task 4: Find each cycle in the needs, and name each feature in it
5. Task 5: Run `check` on the catalog of this repo in CI

## Story Reference

US-006, "Check the links between features and starters", in `user-stories/feature-compose.md:240-263`
(Description, Acceptance Criteria, Context). The design is `PROPOSAL.md`: section 14, "The Check of the Catalog"
(`PROPOSAL.md:475-491`), which lists the checks; section 10, rules 1 to 3 (`PROPOSAL.md:328-337`), which say what
a target and a need must declare; and the table of commands in section 11 (`PROPOSAL.md:356-366`), which names
`feature-compose check`. US-009 (`user-stories/feature-compose.md:324-347`) adds the checks of templates and of the
owners of files.

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

- A target that names no starter of the catalog. Section 14 does not list it, and `check` does not read such a
  target. A feature whose targets all name no starter is checked against no starter.
- An extension point that two targets of a feature declare with different item fields, when the feature does not
  add to it. Rule 2 (`PROPOSAL.md:332-334`) asks for the same item fields only for the extension points of
  `add_to` (decision 3).
- Two declarers of one extension point in one starter: the starter and a need of the feature, or two needs. `new`
  stops on it (`internal/catalog/starter.go:319-326`), and `task test:combinations` renders each feature with each
  starter that it targets, so CI finds it. Section 14 does not list it.
- The needs of two features that set one option to different values. Only a selection of both features stops
  `new` (`internal/catalog/values_set_by_needs.go:20-34`), so it is a fact of a selection, not of one feature, and
  section 14 does not list it.
- A catalog that `check` cannot read. A `starter.yml` or a `feature.yml` that is not valid YAML, a `needs` of another
  form, a pattern or a delimiters glob that does not compile, a files entry that leads out of its feature or has no
  folder, and a template of a starter outside its folders stop `check` with the message that `new` and `show` give.
  `check` does not report them with the other mistakes (decision 6).
- A workflow or a Taskfile for the CI of a catalog in another repo. `check --catalog <folder>` (Task 5) works there,
  but this story wires only the CI of this repo.

**Deferred:**

- The checks of US-009, each named in its criteria (`user-stories/feature-compose.md:331-340`): a template or a
  `when` that reads an option that rule 5 does not allow; a `when` that does not parse (until then it stops `check`
  as a catalog that `check` cannot read, with the message of `new`); two owners that write one path; and an
  extension point whose `file` does not exist or does not render its items.

## Codebase Context

**Commands today.** `newCommand` registers `new`, `show`, `version` and `update`
(`internal/cmd/root.go:27-40`); there is no `check`. `catalogInConfig` gives the catalog of the folder that the config
file names (`internal/cmd/root.go:53-59`, `internal/config/file.go:18-48`), and without a config file `new` and `show`
stop and name its path. `newShowCommand` takes that function as `load` (`internal/cmd/show.go:15-54`), so
`TestShowCommand` gives it an `fstest.MapFS` catalog (`internal/cmd/show_test.go:25-34`), and `TestRootCommand` checks
the read of the config folder through `newCommand` (`internal/cmd/root_test.go:39-60`). `Run` writes the error of a
command to stderr through `writeError` and returns 1 (`internal/cmd/root.go:18-25`, `internal/cmd/root.go:61-69`).

**The read of the catalog.** `Catalog.Starter` reads one starter with the features whose targets name it
(`internal/catalog/catalog.go:18-31`, `featuresTargeting` at `internal/catalog/catalog.go:76-92`). `readFeatureFor`
reads a feature only for one starter and returns early when its targets do not name it
(`internal/catalog/feature.go:23-75`); the type `feature` has no field for its targets
(`internal/catalog/feature.go:12-21`). `Catalog.Combinations` walks each starter in the order of the names
(`internal/catalog/catalog.go:33-47`). Each read stops at its first error, and `TestCatalog` holds those stops
(`internal/catalog/catalog_test.go:156-262`).

**The rules today, at the time of a selection.** `Starter.Select` applies each rule of this story, but only to the
features that one `new` selects for one starter, and it stops at the first error:

- An `add_to` with no declarer: `Starter.extensionPointsOf` (`internal/catalog/starter.go:317-341`). Any selected
  feature counts as a declarer, not only a need, and the message does not name the starter.
- An item without the fields of its extension point: `extensionPoint.add` (`internal/catalog/extension_point.go:17-28`)
  compares the sorted fields, stops at the first item, and names the feature, the extension point and the number of
  the item, but not the starter.
- A folder: `template.place` (`internal/catalog/template.go:48-64`), which `feature.placedIn`
  (`internal/catalog/feature.go:77-87`) calls from `Starter.templatesOf` (`internal/catalog/starter.go:305-315`) only for
  the selected features.
- A need that does not exist and a need that does not target the starter give one message
  (`internal/catalog/starter.go:213-218`): "is not a feature that targets the starter".
- A need value: `Starter.setByNeeds` (`internal/catalog/starter.go:272-292`) calls `options.check`
  (`internal/catalog/option.go:103-117`), which stops at the first option in the order of the names.
- A cycle: `inNeedsOrder`, `needsCycle` and `cycleFrom` (`internal/catalog/starter.go:226-261`) name the first cycle
  that blocks the order, as `the needs of the features form a cycle: app needs base needs app`. `allNeedsOf` and
  `addNeeds` (`internal/catalog/starter.go:171-188`) walk the needs and the needs of the needs by recursion over a
  visited map.
- No rule compares two starters.

`TestStarter` holds a test of each of these rules: the extension points and the items
(`internal/catalog/starter_test.go:346-417`), a template outside the folders (`internal/catalog/starter_test.go:308-317`),
the need values (`internal/catalog/starter_test.go:781-798`), the needs that do not exist or target another starter
and the cycles (`internal/catalog/starter_test.go:830-881`).

**The catalog of this repo.** One starter, `go-event-sourcing-service`, declares the folder `go` and the extension
points `api.middleware` (`[code, import]`) and `api.config` (`[name, type, import]`). `cli` declares `cli.commands`
(`[code]`). `api-auth` adds to both extension points of the starter, `cli-auth` needs `cli` and `api-auth` (with
`method: api-key`, a value of the option) and adds to `cli.commands`, and each file of each feature is under the folder
`go`. So `check` finds no mistake in `catalog/`.

**CI.** `.github/workflows/ci.yml` runs on each push to a branch other than `main`, and `prerelease.yml` calls it for
`main`. The job `build` (`.github/workflows/ci.yml:14-39`) runs `task build` and `task test:unit`; the jobs
`combinations` and `e2e` run `task test:combinations` and `task test:e2e`. The Taskfile tasks are in
`Taskfile.test.yml`. CI has no config file of feature-compose; the e2e tests write one through `XDG_CONFIG_HOME`
(`e2e/testUtils.ts:30-40`).

**Words.** The story says `mistake` for what `check` finds; no Go code uses the word, and it is not an `error`,
which in this repo stops a command. `check` is the verb of the code for a value against its rules
(`option.check`, `options.check`, `Question.Check`, `CheckRepoName`); `release.Check` is the result of a check for a new
release, in another package. A `target` is a starter that the `targets` of a `feature.yml` names. `declarer`
(`extensionPoint.declarer`) is the starter or the feature that declares an extension point, and `item fields`
(`extensionPoint.itemFields`) are the fields that it declares in `item`. `folder`, `need`, `needs` and `cycle` keep the
meanings of the code (`tasks/learnings.md:204-210`).

**Decisions of this breakdown, for the caller to overrule:**

1. **`check` gathers the mistakes itself, and reuses the rule of one item, one file and one need value.**
   `Catalog.Check` reads each starter with the features that target it, and checks each pair of a starter and a
   feature that targets it. It does not call `Select` for each feature, because `Select` stops at its first error,
   so a feature with two mistakes shows one; it selects the needs, so the mistake of a need shows under each
   feature that needs it, with the wrong feature named; its messages do not name the starter; it gives one message
   for a need that does not exist and one that does not target the starter; and it never compares two starters, so
   it cannot find two targets with different item fields. `Catalog.Starter` also stops at the first feature that it
   cannot read. `check` and `new` still apply one text of each rule: the fields of one item (`extensionPoint.add`),
   the folder of one file (`template.place`), and the value of one option (`options.check`).
2. **The extension points that a feature can add to** (Task 1) are those that the starter, the feature itself, or a
   need of the feature or a need of a need declares, among the features that target the starter. An extension point
   that only another feature of the starter declares, which the feature does not need, is a mistake, although `new`
   accepts it when the developer also selects that feature: rule 2 names only the starter and a need. The feature
   itself counts, because `new` accepts it and rule 1 lets a feature declare extension points.
3. **Two targets with different item fields** (Task 2) are a mistake only for an extension point of `add_to`, and the
   fields are compared in any order, as `extensionPoint.add` compares them. In each target, the declarer is the one of
   decision 2. A target that does not declare the extension point is the mistake of Task 1, not of this one.
4. **A mistake that does not depend on the starter** (a need that does not exist, a need value) gives one mistake for
   each target of the feature, and each names its starter (`tasks/learnings.md:197-203`). The mistake of two targets
   names each of them, and the mistake of a cycle names each of its features.
5. **A cycle** (Task 4) gives one mistake for each starter whose features hold it, not one for each of its features.
   Its message names the starter and each feature of the cycle in the order of the needs, as the message of `new`
   does, and no feature out of the cycle. Each feature that is in a cycle is named in at least one cycle mistake;
   when two cycles share a feature, `check` does not name each cycle through it.
6. **A catalog that `check` cannot read stops `check`**, with the message of `new` and `show` and an exit status that is
   not zero. Only the mistakes of this story are gathered.
7. **The output.** `check` prints each mistake on one line to stdout, in a fixed order, and then stops with an error
   that counts the mistakes, so `Run` writes it to stderr and the exit status is 1. With no mistake, it prints one line
   that says that the catalog has no mistake, and the exit status is 0. Each message names the starter as
   `the starter <name>` and the feature as `the feature <name>`, as the messages of `new` do.
8. **`check --catalog <folder>`** (Task 5) checks the catalog in that folder instead of the folder of the config file,
   so CI, which has no config file, runs `check` on `catalog/`. The other choice is a step that writes a config file
   under a temporary `XDG_CONFIG_HOME`, as the e2e tests do; this breakdown does not take it, because a catalog in
   another repo needs the same thing in its own CI. `new` and `show` do not get the flag.

## Tasks

### Task 1: Add `check`, which finds an `add_to` that no target or need declares, and an item without the fields of its extension point

**Behavior:** `feature-compose check` reads the catalog folder that the config file names and checks each feature
against each starter that it targets. It finds an `add_to` whose extension point the starter, the feature, and its
needs and their needs do not declare (decision 2), and an item that does not have the item fields of its extension
point. It prints each mistake on its own line, with the starter, the feature and the extension point, gathers all
mistakes of the catalog in one run, and stops with an exit status that is not zero when it finds one (decision 7).
With no mistake, it says so and exits with 0.

**Acceptance Criteria:**
- [x] `Catalog.Check` gives a mistake for an `add_to` that names an extension point that neither the starter, nor the
  feature, nor a need of the feature or a need of a need declares. The message names the starter, the feature and the
  extension point.
- [x] An `add_to` to an extension point that only another feature of the starter declares, which the feature does not
  need, is a mistake. An `add_to` to an extension point that a need of a need declares, or that the feature itself
  declares, is not.
- [x] Each item that does not have the item fields of its extension point is a mistake: an item that lacks a field,
  and an item with a field that the extension point does not declare. The message names the starter, the feature,
  the extension point, the number of the item, its fields and the declared fields. An item with the fields in
  another order is not a mistake.
- [x] `Catalog.Check` gives all mistakes of the catalog in one list: two items with wrong fields in one feature give
  two mistakes, the mistakes of two features are both in it, and a feature with two targets that do not declare its
  extension point gives one mistake for each target, each with its own starter.
- [x] `Catalog.Check` gives the mistakes in a fixed order, so two calls on one catalog give the same list.
- [x] A catalog with no mistake gives no mistake, and `Catalog.Check` on the catalog of this repo (`catalog/`) gives
  no mistake.
- [x] A catalog that `Check` cannot read stops it with the cause: a `starter.yml` that is not valid YAML, and a read of
  `starters`, of a `starter.yml`, of `features` or of a `feature.yml` that fails.
- [x] `feature-compose check` prints each mistake on its own line to stdout and then stops with an error that gives the
  number of the mistakes, and `feature-compose` exits with a status that is not zero. With no mistake, it prints one
  line that says that the catalog has no mistake, and exits with 0. `check` takes no argument.
- [x] `check` reads the catalog folder that the config file names. Without a config file it stops and names the path
  of the file, as `new` and `show` do.
- [x] `feature-compose` with no command lists `check` with its usage.
- [x] `README.md` has a section "Check the Catalog" that names the command, the catalog that it reads, each mistake
  that it finds, its output, and its exit status. "Configure the catalog" names `check` with `new` and `show`.
- [x] `e2e/tests/check.test.ts`: `check` on the catalog of this repo exits with 0 and prints the line of no mistake;
  `check` on a catalog in a scratch folder, with a feature that adds to an extension point that nothing declares,
  exits with a status that is not zero, and its output names the starter, the feature and the extension point. The
  tests pass on the host.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/catalog.go` — `Check`, over each starter of the catalog
- `internal/catalog/starter.go` — the mistakes of the features that target the starter: the extension points of
  `add_to` and the fields of each item
- `internal/catalog/extension_point.go` — the rule of the fields of one item, which `new` and `check` both apply
- `internal/catalog/catalog_test.go` — a group `check` in `TestCatalog`
- `internal/cmd/check.go` — the command `check`
- `internal/cmd/check_test.go` — `TestCheckCommand`
- `internal/cmd/root.go`, `internal/cmd/root_test.go` — the command in the list, and its read of the config folder
- `README.md` — "Check the Catalog", and "Configure the catalog"
- `e2e/tests/check.test.ts` — the exit status and the output of the binary

**Patterns to Follow:**
- `internal/cmd/show.go:15-54` for a command that takes `load`, and `internal/cmd/root.go:27-40` and
  `internal/cmd/root.go:53-59` for its registration and the catalog of the config file
- `internal/cmd/show_test.go:25-34` for a test of a command with an `fstest.MapFS` catalog, and
  `internal/cmd/root_test.go:39-60` for the read of the config folder through `newCommand`
- `internal/catalog/catalog.go:33-47` for a walk over each starter of the catalog
- `internal/catalog/starter.go:317-341` for the declarers of the extension points and the `add_to` of a feature
- `internal/catalog/starter.go:171-188` for the needs of a feature and of its needs
- `internal/catalog/extension_point.go:17-28` for the fields of an item
- `internal/catalog/starter_test.go:346-417` for the extension points and the items in tests, and their messages
- `internal/catalog/catalog_test.go:299-311` for a read that fails, with `failingFS`
- `e2e/tests/show.test.ts:5-40` for `runCli`, its exit status and its output, and `e2e/testUtils.ts:30-40` for a
  config file in `XDG_CONFIG_HOME`
- `README.md:218-237` for the section of a command
- `PROPOSAL.md:475-491` and `PROPOSAL.md:328-337` for the checks and rules 1 to 3
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:197-203` for a relation that holds for every message
- `tasks/learnings.md:204-210` for the words of needs and values
- `tasks/learnings.md:211-216` for a mutants run of `internal/catalog` that passes its time limit
- `tasks/learnings.md:58-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `Catalog.Combinations` (`internal/catalog/catalog.go:33-47`) walks each starter and
`Starter.extensionPointsOf` (`internal/catalog/starter.go:317-341`) holds both rules; but each rule in this repo stops
at its first error, and no code gathers the mistakes and goes on

**Blast radius:** low — `check` reads the catalog and prints; it writes no file, and nothing outside this repo reads
its output yet

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd`. Run `e2e/tests/check.test.ts` on the host
(`tasks/learnings.md:58-64`).

**Depends on:** None

---

### Task 2: Find a folder that a target does not have, and targets that declare one extension point with other item fields

**Behavior:** `check` finds a feature that adds files to a folder that a target does not have, from each files entry
of the feature, whatever its `when` gives. It also finds a feature whose targets declare one extension point of its
`add_to` with different item fields (decision 3). Each message names the feature, the folder or the extension point,
and each starter that the mistake is about.

**Acceptance Criteria:**
- [x] A feature with a file in a folder that a target does not have is a mistake. The message names the starter, the
  feature and the folder, and a folder with many files gives one mistake.
- [x] The files of each files entry are checked, also the files of an entry with a `when`, whatever its expression
  gives.
- [x] A file directly in a files entry, which is in no folder, is a mistake, and the message names the starter, the
  feature and the file.
- [x] A feature whose targets declare one extension point of its `add_to` with different item fields is a mistake. The
  message names the feature, the extension point, and each target with its item fields. In a target where a need of
  the feature declares the extension point, the fields of that need count.
- [x] Targets that declare the item fields in another order are not a mistake. A target that does not declare the
  extension point is not named in this mistake.
- [x] An extension point that two targets declare with different item fields, and that the feature does not add to,
  is not a mistake (decision 3).
- [x] The mistakes of Task 1 are still found next to these, and `Catalog.Check` on `catalog/` gives no mistake.
- [x] "Check the Catalog" in `README.md` names both mistakes.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/catalog.go` — the item fields of an extension point across the targets of a feature
- `internal/catalog/starter.go` — the folders of the files of a feature
- `internal/catalog/template.go` — the rule of the folder of one file, which `new` and `check` both apply
- `internal/catalog/catalog_test.go` — the cases above in the group `check`
- `README.md` — "Check the Catalog"

**Patterns to Follow:**
- Task 1 for the walk over each starter and each feature that targets it, and for the messages
- `internal/catalog/template.go:48-64` for the folder of a file
- `internal/catalog/feature.go:41-62` for the templates of each files entry, with or without `when`
- `internal/catalog/extension_point.go:17-28` for fields compared in any order
- `internal/catalog/starter_test.go:308-317` for a template outside the folders of the starter in a test
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`
- `tasks/learnings.md:197-203` for a mistake with more than one starter

**Testable:** Yes

**Certainty:** medium — `template.place` (`internal/catalog/template.go:48-64`) holds the rule of the folder and
`extensionPoint.add` (`internal/catalog/extension_point.go:17-28`) compares the fields; but no rule in this repo spans
two starters, and Task 1 checks one starter at a time

**Blast radius:** low — more mistakes that `check` prints; it writes nothing

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog`.

**Depends on:** Task 1

---

### Task 3: Find a need that does not exist, does not target each target, or sets a value that its option does not have

**Behavior:** `check` finds a need that is not a feature of the catalog, a need that is a feature of the catalog but
does not target a target of the feature, and each value that a need sets and its option does not have. The two
messages of a need tell the two mistakes apart. `check` reads each feature of the catalog whatever its targets, so
it finds a wrong need value also when the need does not target the starter. A mistake that does not depend on the
starter gives one mistake for each target (decision 4).

**Acceptance Criteria:**
- [x] A need that is not a feature of the catalog is a mistake, and the message names the starter, the feature and the
  need, and says that the catalog has no such feature.
- [x] A need that is a feature of the catalog but does not target the starter is a mistake, and the message names the
  starter, the feature and the need, and says that the need does not target the starter. A feature with two targets
  and a need that targets only one of them gives one mistake, for the other target.
- [x] Each value that a need sets and its option does not have is a mistake: a value that is not one of the values of
  the option, a value that does not match the pattern of the option, an option that the needed feature does not
  declare, and an option of a feature with no option. The message names the starter, the feature, the need, the
  option and the value. A need that sets two wrong values gives two mistakes.
- [x] A wrong need value is a mistake also when the need does not target the starter.
- [x] A need that does not exist, in a feature with two targets, gives one mistake for each target, each with its own
  starter.
- [x] A need of a need that is wrong is the mistake of the feature that needs it directly, not of the feature that
  needs its need.
- [x] The mistakes of Tasks 1 and 2 are still found next to these, and `Catalog.Check` on `catalog/` gives no mistake.
  `Catalog.Starter` and `Starter.Select` give the same results as before.
- [x] "Check the Catalog" in `README.md` names the three mistakes.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/feature.go` — the read of a feature whatever its targets, with its targets
- `internal/catalog/catalog.go` — `Check` reads each feature of the catalog
- `internal/catalog/starter.go` — the mistakes of the needs and of the need values
- `internal/catalog/catalog_test.go` — the cases above in the group `check`
- `README.md` — "Check the Catalog"

**Patterns to Follow:**
- Task 1 for the walk over each starter and each feature that targets it, and for the messages
- `internal/catalog/starter.go:208-224` for the needs of a feature that the starter has
- `internal/catalog/starter.go:272-292` and `internal/catalog/option.go:103-117` for the values that a need sets and
  their check
- `internal/catalog/feature.go:23-75` and `internal/catalog/catalog.go:76-92` for the read of a feature and the filter by
  its targets
- `internal/catalog/starter_test.go:781-798` for the wrong need values in tests
- `internal/catalog/starter_test.go:830-855` for the needs that do not exist or target another starter in tests
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`
- `tasks/learnings.md:197-203` for a mistake with one message for each starter
- `tasks/learnings.md:204-210` for the words of needs and values
- `tasks/learnings.md:285-291` for the combinations test after a change to the read of the catalog

**Testable:** Yes

**Certainty:** high — it repeats the walk of Task 1 with the rules of `selectWithNeeds`
(`internal/catalog/starter.go:208-224`) and `setByNeeds` (`internal/catalog/starter.go:272-292`), and the read of a
feature whatever its targets is `readFeatureFor` (`internal/catalog/feature.go:23-75`) without its filter

**Blast radius:** low — more mistakes that `check` prints; the read of a feature for `new` gives the same features

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog`.

**Depends on:** Task 1

---

### Task 4: Find each cycle in the needs, and name each feature in it

**Behavior:** `check` finds the features of a starter whose needs form a cycle. The message names the starter and
each feature of the cycle, in the order of the needs, as the message of `new` does, and no feature out of the
cycle (decision 5). Each feature that is in a cycle is named in a cycle mistake. A cycle does not stop `check`: the
other mistakes are still in the list.

**Acceptance Criteria:**
- [ ] Features of one starter whose needs form a cycle are a mistake. The message names the starter and each feature
  of the cycle in the order of the needs, for example `app needs base needs app`, and names no other feature.
- [ ] A feature that needs itself is a cycle, for example `app needs app`.
- [ ] A cycle of three features that another feature needs gives a message that names the three features of the
  cycle and not the feature that needs it.
- [ ] Two cycles with no feature in common give two mistakes. One cycle gives one mistake for each starter whose
  features hold it, not one for each of its features.
- [ ] Each feature that is in a cycle is named in at least one cycle mistake, also when two cycles share a feature.
- [ ] Features whose needs form a cycle only through a feature that does not target the starter are not a cycle of
  that starter; Task 3 gives the mistake of that need.
- [ ] The mistakes of Tasks 1 to 3 are still found next to a cycle, and `Catalog.Check` on `catalog/` gives no mistake.
- [ ] "Check the Catalog" in `README.md` names the cycle.
- [ ] `clerk mutants` on the changed lines of `internal/catalog` completes within its time limit, with no mutant that
  waits for its timeout.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/starter.go` — the cycles of the needs of the features that target the starter
- `internal/catalog/catalog_test.go` — the cases above in the group `check`
- `README.md` — "Check the Catalog"

**Patterns to Follow:**
- Task 1 for the walk over each starter and each feature that targets it, and for the messages
- `internal/catalog/starter.go:226-261` for the order of the needs, and the cycle and its message
- `internal/catalog/starter.go:171-188` for a walk of the needs by recursion over a visited map
- `internal/catalog/starter_test.go:857-881` for the cycles in tests
- `tasks/learnings.md:148-154` for a walk of a graph that gives no endless mutant
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `cycleFrom` (`internal/catalog/starter.go:249-256`) finds a cycle and gives its features in the
order of the needs; but it finds only the first cycle that blocks a selection, and no walk in this repo finds a cycle
through each feature of a starter

**Blast radius:** low — one more mistake that `check` prints; `new` keeps its own message of a cycle

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` on the changed lines of
`internal/catalog` completes with no live mutant (`tasks/learnings.md:148-154`, `tasks/learnings.md:211-216`).

**Depends on:** Task 1

---

### Task 5: Run `check` on the catalog of this repo in CI

**Behavior:** `check --catalog <folder>` checks the catalog in that folder instead of the folder that the config file
names (decision 8). `task test:catalog` runs it on `catalog/`, and the job `build` of the CI workflow runs that task, so
CI runs `check` on each change and fails when `check` finds a mistake.

**Acceptance Criteria:**
- [ ] `check --catalog <folder>` checks the catalog in that folder, also with no config file. A folder that does not
  exist, or is not a folder, stops `check`, and the message names it.
- [ ] Without the flag, `check` reads the folder that the config file names, as in Task 1.
- [ ] `task test:catalog` runs `check` on `catalog/` and exits with 0. With a feature in `catalog/` that adds to an
  extension point that nothing declares, it exits with a status that is not zero and names that feature; with the
  file restored, it exits with 0 again.
- [ ] The job `build` of `.github/workflows/ci.yml` runs `task test:catalog` in a step of its own, so CI runs `check` on
  each push to a branch and on each run of the workflow for `main`.
- [ ] "Check the Catalog" in `README.md` names `--catalog`, `task test:catalog`, and the job of CI that runs it.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/cmd/check.go`, `internal/cmd/check_test.go` — the flag `--catalog`
- `Taskfile.test.yml` — the task `catalog`
- `.github/workflows/ci.yml` — the step of the job `build`
- `README.md` — "Check the Catalog"

**Patterns to Follow:**
- Task 1 for the command and its test
- `internal/cmd/show.go:15-54` for a flag of a command
- `internal/config/file.go:43-46` for the message of a catalog folder that is not a folder
- `Taskfile.test.yml:11-14` for a test task of the repo
- `.github/workflows/ci.yml:14-39` for the steps of the job `build`
- `README.md:342-360` for a section that names a task and the CI job that runs it

**Testable:** Yes

**Certainty:** high — the flag follows the flag `--output` of `show` (`internal/cmd/show.go:15-54`), the task follows
`test:combinations` (`Taskfile.test.yml:11-14`), and the step follows the steps of the job `build`
(`.github/workflows/ci.yml:14-39`)

**Blast radius:** low — the CI of this repo, where a step that fails only stops a push with a mistake in the catalog;
the flag is new and changes no command that exists

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `task test:catalog` exits with 0, and with
a mistake added to a `feature.yml` of `catalog/` it exits with a status that is not zero (restore it with
`git checkout -- catalog/`). `clerk mutants` finds no live mutant on the changed lines of `internal/cmd`.

**Depends on:** Task 1

---

## Summary

- **Tasks:** 5.
- **Order:** dependency first. Task 1 is the walking skeleton: the command, `Catalog.Check` over each starter and
  each feature that targets it, the gathering of all mistakes, the exit status, the e2e test, and the first two
  mistakes. Tasks 2, 3 and 4 each add the mistakes of one criterion of the story to that walk, and each depends only on
  Task 1. Task 2 adds the one rule that spans two starters; Task 3 makes `check` read each feature whatever its
  targets; Task 4 is the graph walk that the mutation run constrains. Task 5 wires CI last, so its first run checks
  each rule of the story.
- **Assessment:** no task is low certainty: each rule of the story but one exists at the time of a selection, the one
  that does not (two targets with other item fields) compares the fields as `extensionPoint.add` does, and each task
  states its variation. Tasks 1, 2 and 4 are medium: Task 1 gathers mistakes where every rule here stops at
  its first error, Task 2 compares two starters where no rule does, and Task 4 finds a cycle through each feature
  where `cycleFrom` finds only the first. Tasks 3 and 5 are high. No task has a high blast radius: `check` reads the
  catalog and prints, no task touches authentication, money, migrations, secrets or personal data, and no consumer
  outside this repo reads the output of `check` yet.
- **Criteria of the story:**
  - "`check` finds an `add_to` that names an extension point that a target and the needs of the feature do not
    declare. It also finds an item that does not have the fields of its extension point": Task 1.
  - "`check` finds two targets of a feature that declare one extension point with different item fields. It also finds
    a feature that adds files to a folder that a target does not have": Task 2.
  - "`check` finds a need that does not exist, a need that does not target each target of the feature, and a need that
    sets a value that its option does not have": Task 3.
  - "`check` finds a cycle in the needs, and the message names each feature in the cycle": Task 4.
  - "Each message names the feature, the starter, and the extension point, folder, or need that is wrong. `check`
    reports all mistakes in one run, and it exits with a status that is not zero when it finds one": Task 1 for the
    gathering, the order and the exit status, and Tasks 1 to 4 each for the messages of their mistakes.
  - "The CI of feature-compose runs `check` on each change": Task 5.
- **Deferred:** none of the criteria of this story. The checks of templates, of `when` and of the owners of files go to
  US-009, each named in its criteria (see Boundaries).
