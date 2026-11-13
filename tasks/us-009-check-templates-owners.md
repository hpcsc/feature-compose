# Tasks: Check the Templates and the Owners of Files (US-009)

## Contents

1. Task 1: Report a `when` that does not parse as a mistake of the feature, and go on
2. Task 2: Find a `when` that reads an option that rule 5 does not allow
3. Task 3: Find a template that reads an option that rule 5 does not allow, in its parse tree
4. Task 4: Find an extension point whose `file` does not exist or does not call `items` for it
5. Task 5: Find two owners that write one path of the repo

## Story Reference

US-009, "Check the templates and the owners of files", in `user-stories/feature-compose.md:336-358` (Description,
Acceptance Criteria, Context). The design is `PROPOSAL.md`: section 14, "The Check of the Catalog"
(`PROPOSAL.md:475-491`), whose last three items are this story; section 10, rules 5 and 7 (`PROPOSAL.md:338-342`); the
names of an expression in section 8 (`PROPOSAL.md:286-290`); and the template functions of section 9
(`PROPOSAL.md:295-326`). `docs/catalog.md` tells a catalog author how a template reads options
(`docs/catalog.md:266-284`), how a `when` names them (`docs/catalog.md:172-185`), and how the file of an extension
point renders its items (`docs/catalog.md:187-264`). US-006 (`tasks/completed/us-006-check-links.md`) added `check` and
deferred these four criteria to this story.

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

- A call of `option` or `items` whose argument is not a quoted text: a variable, a value that a pipeline gives to the
  function, or a name that the template builds. `check` cannot know the name, so it reports nothing for such a call
  (decision 6). `new` still stops on an option that the template cannot read.
- A template that calls `items` or `unique` for an extension point that no declarer declares, or `unique` with a field
  that the extension point does not declare. `new` stops on both before it writes a file
  (`internal/catalog/selection.go:303-344`), and no story names them.
- Two templates of one owner that write one path, for example the files entries `api-key` and `oauth` of `api-auth`,
  which both write `internal/auth/config.go` under `when` expressions that exclude each other. Rule 7 names owners,
  and `new` stops when both entries hold (`internal/catalog/selection.go:221-223`).
- Two owners whose paths differ as text but render to one path, for example `cmd/{{ .Name }}/main.go` and
  `cmd/{{ kebab .Name }}/main.go` for a name in lower case. `check` compares the paths before the render (decision 8),
  and `new` still stops.
- A `when` whose references are readable but which can never hold, for example `method == "basic"` for an option
  whose values are `api-key` and `oauth`. It is not a mistake of rule 5.
- The default of an option that calls `option`. A default renders with no readable option
  (`internal/catalog/option.go:71`), so `new` stops on it, and it is not a template of a files entry.
- A template that parses but does not build or does not run. `task test:combinations` finds it, as
  `docs/catalog.md:293-294` says.

**Deferred:**

- The `when` of an option, which the proposal shows for `store-key` (`PROPOSAL.md:201-204`). The catalog has no such
  field today: `featureYML` reads `when` only in a files entry (`internal/catalog/feature_yml.go:14-17`). US-013 adds
  the first one (`user-stories/feature-compose.md:446-448`), but its criteria do not name `check`. The caller must add
  a criterion to US-013: "`check` finds a `when` of an option that does not parse, or that reads an option that
  rule 5 does not allow". The design of Tasks 1 and 2 keeps each `when` of a feature on `feature`, so that US-013 adds
  its expressions to the same checks.

## Codebase Context

**The check today.** `Catalog.Check` (`internal/catalog/catalog.go:77-109`) reads each feature of the catalog one time
(`Catalog.features`, `internal/catalog/catalog.go:111-125`), builds each starter with the features that target it, and
gathers `Starter.mistakes` for each starter (`internal/catalog/starter.go:175-188`): the `add_to`, the folders and the
needs of each feature, then the cycles, each with the prefix `the starter <name>: `. Then it gathers the mistakes of
each feature in the order of the names (`feature.itemFieldsMistakes`, `internal/catalog/catalog.go:105-107`), with no
prefix of a starter. `internal/cmd/check.go:34-66` prints each mistake on a line and stops with the count. A catalog
that `check` cannot read stops it with the message of `new` (US-006 decision 6).

**What each criterion is today:**

| Criterion | `new` today | `check` today | This story |
| --- | --- | --- | --- |
| A template or a `when` reads an option that rule 5 does not allow | Stops at the render, only for the templates and the `when` expressions of the selected features, and only for the selected starter: `Selection.option` (`internal/catalog/selection.go:282-301`) and `expression.trueFor` (`internal/catalog/expression.go:59-62`) | Nothing | New behaviour (Tasks 2 and 3) |
| A `when` does not parse | Stops when it reads a feature that targets the starter: `newFeature` (`internal/catalog/feature.go:68-74`) | Stops at the first such `when`, through the same read; no other mistake is printed | Moves the failure into the list of mistakes, and `check` goes on (Task 1). `new` keeps its stop |
| Two owners write one path | Stops at the render, only for the selected features whose `when` holds: `Selection.files` (`internal/catalog/selection.go:200-228`) | Nothing | Moves the failure of `new` into `check`, for each pair of features that can be selected together (Task 5) |
| The `file` of an extension point does not exist, or does not render its items | Nothing: `new` writes the repo without the items | Nothing | New behaviour (Task 4). The type `extensionPoint` has no field for its file (`internal/catalog/extension_point.go:10-15`), but `ExtensionPointDefinition.File` reads it (`internal/catalog/definitions.go:159-163`) |

**Rule 5 at the render.** `Selection.readable` (`internal/catalog/selection.go:293-301`) gives a template of a feature
its own options by name, the options of each feature in its `needs` as `<need>.<option>`, and the options of the
starter as `starter.<option>`. A need of a need is not readable. A template of the starter has `featureName` empty, so
it reads only `starter.<option>`. `expression.trueFor` reads the same map for a `when`. The rule that a starter option
must be declared by each target exists nowhere in the code: `new` knows only the one starter of the selection.

**Templates and their parse.** `Selection.render` (`internal/catalog/selection.go:257-269`) builds the map of the
template functions (`option`, `items`, `unique`, `indent`, `githubRepository`, `kebab`, `snake`, `camel`, `pascal`,
`upper`), and parses the path with the default delimiters and the contents with the delimiters of the template.
`text/template` refuses to parse a call of a function that the map does not hold, so a parse of `check` needs the
same names. A starter gives its templates their delimiters when it places them (`internal/catalog/starter.go:47-66`,
`template.place` at `internal/catalog/template.go:48-63`). A feature places its templates only for a selected starter
(`feature.placedIn`, `internal/catalog/feature.go:98-108`), so the templates of a feature in `Catalog.Check` have no
delimiters yet; the rules of the feature are in `feature.rules`. No code of this repo walks a `text/template/parse`
tree.

**Owners.** `template.owner` is `the starter` or `the feature <name>` (`internal/catalog/template.go:25-46`), and
`file.owner` carries it to the preview. `Selection.files` keeps a map of the template that writes each rendered path,
and stops on the second one, also when both templates have one owner.

**The catalog of this repo.** A scratch parse of each template of `catalog/` with its delimiters and the template
functions found no template that does not parse, and these reads: `api-auth` reads `method` and `starter.module`;
`cli` (targets `go` and `go-event-sourcing-service`) reads `starter.go-version` and `starter.module`, which both
starters declare; `cli-auth` reads `api-auth.method`, a need, and `starter.module`; `process-manager` reads
`starter.go-version` and `starter.module`; each starter reads only `starter.go-version` and `starter.module`, which it
declares. The only `when` expressions are `method == "api-key"` and `method == "oauth"` in `api-auth`. Each of the 27
extension points (21 of `go-event-sourcing-service`, 5 of `go`, `cli.commands` of `cli`) names a file that is a
template of its declarer and calls `items` with its name; four files render two extension points each. In each
starter, only `api-auth` writes one path from two files entries, and that is one owner. So no template of `catalog/`
breaks rule 5 today, and each task ends with `check` on `catalog/` giving no mistake, with no change to the catalog.

**The test fixtures of US-006.** `withFeatures` in the group `check` of `TestCatalog`
(`internal/catalog/catalog_test.go:354-366`) and in `TestCheckCommand` (`internal/cmd/check_test.go:29-38`) declare
`api.middleware` with the file `go/server.go`, whose contents do not call `items`. Many cases declare extension points of
features (`cli.commands` in `go/cli.go`, `docs.pages` in `go/index.md`, `base.hooks`, `app.routes`, `queue`) with no
file at all. Task 4 makes each of these a mistake, so Task 4 gives those fixtures their files.

**Words.** The code calls the text that a template passes to `option` a `reference` (`Selection.option(featureName,
reference)`, `optionValues.byReference`), and the references that a template can read `readable`
(`Selection.readable`). `owner` is the starter or the feature of a template. `declarer` is the starter or the feature
that declares an extension point, and its `file` is the field of `extension_points` in `starter.yml` and `feature.yml`.
`mistake` is what `check` reports, and does not stop it. The proposal calls the result of a parse the `parse tree`
(`PROPOSAL.md:486-487`). `tasks/learnings.md:206-212` holds the words of options, values and needs.

**Decisions of this breakdown, for the caller to overrule:**

1. **The readable references of a feature** (Tasks 2 and 3) are: each option that the feature declares, by its name;
   `<need>.<option>` for each option of each feature in its `needs`, but not of a need of a need, as at the render; and
   `starter.<option>` for an option that each target of the feature that is a starter of the catalog declares. A target
   that names no starter of the catalog is not counted, as in US-006, and a feature with no such target can read no
   starter option.
2. **The templates of the starter are checked too** (Task 3). They can read only `starter.<option>` of an option that
   the starter declares, as `Selection.readable` gives them. Rule 5 speaks of a feature, but `docs/catalog.md:276` says
   that a template of the starter reads its options with the same form, and the walk is the same.
3. **A mistake of rule 5 is a mistake of the feature, not of each target**, because the readable starter options span
   all its targets. It comes one time in the list, after the mistakes of the starters, and for a starter option it
   names each target that does not declare it (`tasks/learnings.md:199-205`). Each pair of a template, or a `when`, and
   a reference that it cannot read is one mistake, also when it reads the reference many times.
4. **A `when` that does not parse** (Task 1) is one mistake for each files entry, a mistake of the feature, and it
   does not stop `check`. The read of the feature keeps it, and `Catalog.Starter` still stops with the message of
   today, so `new`, `--dry-run`, the form and `task test:combinations` stop as before.
5. **A template that does not parse is a mistake** (Task 3) of its owner, with the error of the parse. `check` needs
   the parse tree to find the reads, and the story's criterion of a `when` that does not parse makes this the same
   kind of mistake. The parse uses the map of template functions of the render, so a template that `new` renders is
   never such a mistake.
6. **Only a call whose argument is a quoted text is a read**, or a call of `items` for an extension point. `check`
   finds `option "<reference>"` and `items "<name>"` wherever they are in the parse tree, also in the path of the
   template and as a parenthesized argument, as in `githubRepository (option "starter.module")`.
7. **The file of an extension point** (Task 4) must be a template of its declarer: the starter for an extension point
   of `starter.yml`, the feature for one of `feature.yml`, in any of its files entries. Its catalog path without
   `.tmpl` is the `file`. It renders the items when its parse tree calls `items` with the name of the extension point,
   as `PROPOSAL.md:488` says; a call of `unique` alone does not count. A file that does not parse gives only the
   mistake of decision 5.
8. **Two owners** (Task 5) are the starter and a feature that targets it, or two features that target one starter.
   `check` compares the paths in the repo before the render: the path of the folder joined with the path in the
   folder, without `.tmpl`. It reads each files entry whatever its `when`. Each path and each pair of owners that
   write it is one mistake of the starter.
9. **The order of the list.** The mistakes of each starter come first, in the order of the starters, as today, and
   the new mistakes of a starter (its templates, its extension points, its owners) come after its cycles. Then the
   mistakes of each feature come in the order of the features: its item fields, its `when` expressions that do not
   parse, its reads, the parse of its templates, and the files of its extension points. Each task keeps a fixed
   order and writes it in `docs/commands.md`.

## Tasks

### Task 1: Report a `when` that does not parse as a mistake of the feature, and go on

**Behavior:** `check` finds each files entry whose `when` does not parse and reports it with the other mistakes,
instead of stopping at the first one. The message names the feature, the files entry, the expression and why it does
not parse. The read of a feature for one starter, which `new` uses, still stops with the message of today
(decision 4).

**Acceptance Criteria:**
- [x] `Catalog.Check` gives a mistake for each files entry whose `when` does not parse. The message names the feature,
  the files entry, the `when` as written, and why it does not parse. Two such entries in one feature give two
  mistakes.
- [x] The mistake comes one time for the feature, also when the feature has two targets, and after the item fields
  mistakes of that feature (decision 9).
- [x] A feature with a `when` that does not parse still gives its other mistakes: a folder that a target does not
  have, among the templates of that files entry too, and a need that does not exist. The mistakes of the other
  features are still in the list.
- [x] `Catalog.Starter` still stops at a `when` that does not parse, with the message of today, for each text of the
  table at `internal/catalog/catalog_test.go:235-262`.
- [x] `feature-compose check` prints the mistake on its own line, with the other mistakes, and stops with the count.
- [x] `e2e/tests/check.test.ts`: `check` on a scratch catalog with a feature whose `when` does not parse exits with 1,
  prints one line that names the feature and the expression, and writes `the catalog has 1 mistake` to stderr. The
  test of the catalog of this repo still exits with 0. Both pass on the host.
- [x] `docs/commands.md` "Check the Catalog" has a row for the mistake, and its paragraph about a catalog that `check`
  cannot read does not include a `when`.
- [x] `Catalog.Check` on `catalog/` gives no mistake. `go build ./...`, `go vet ./...`,
  `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/feature.go` — the read of a feature keeps each `when` that does not parse and goes on; the read for
  one starter stops with it; the mistakes of the feature
- `internal/catalog/catalog.go` — `Check` adds the mistakes of each feature after its item fields
- `internal/catalog/catalog_test.go` — cases in the group `check`; the group `starter` keeps its stop
- `internal/cmd/check_test.go` — `check` prints the mistake
- `e2e/tests/check.test.ts` — the case above
- `docs/commands.md` — "Check the Catalog"

**Patterns to Follow:**
- `internal/catalog/feature.go:68-74` for the parse of a `when` and the message of `new`
- `internal/catalog/feature.go:32-45` for the read of a feature for one starter, which must still stop
- `internal/catalog/catalog.go:105-107` and `internal/catalog/feature.go:140-168` for a mistake of a feature, one time
  in the list
- `internal/catalog/catalog_test.go:235-262` for the texts that do not parse and their reasons
- `internal/cmd/check_test.go:83-112` for a test of the command that prints new mistakes
- `e2e/tests/check.test.ts:19-34` for a scratch catalog in an e2e test
- `docs/commands.md:301-327` for the table of mistakes and the paragraph of the order
- `tasks/learnings.md:318-324` for one read of each feature in one check
- `tasks/learnings.md:185-191` for a test in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:304-310` for a test of the command that reaches the new path
- `tasks/learnings.md:290-296` for the combinations test after a change to the read of a feature
- `tasks/learnings.md:59-65` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `newFeature` parses each `when` (`internal/catalog/feature.go:68-74`) and `itemFieldsMistakes`
is a mistake of a feature in the list (`internal/catalog/catalog.go:105-107`); but no read in this repo keeps a failure
and goes on while another caller of the same read must still stop on it

**Blast radius:** low — `check` prints one more kind of mistake; the read that `new` uses keeps its stop, which the
existing test of `Catalog.Starter` holds

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog` and `internal/cmd`. Run
`e2e/tests/check.test.ts` on the host (`tasks/learnings.md:59-65`).

**Depends on:** None

---

### Task 2: Find a `when` that reads an option that rule 5 does not allow

**Behavior:** `check` finds each reference in a `when` that the feature cannot read under rule 5 (decision 1), and
names the `when`, its files entry, the reference and the feature. For a starter option, it names each target that does
not declare it. It checks every `when`, whatever value its options can have.

**Acceptance Criteria:**
- [x] A `when` that reads a reference that the feature cannot read is a mistake. The message names the `when`, its
  files entry, the reference and the feature. These are such references: an option that the feature does not declare;
  `<need>.<option>` of an option that the need does not declare; `<feature>.<option>` of a feature that the feature does
  not name in `needs`, also a need of a need; and `starter.<option>` of an option that no target declares.
- [x] `starter.<option>` of an option that one target declares and another target does not is a mistake, and the
  message names each target that does not declare it.
- [x] Each reference in the `when` is checked: on both sides of `&&` and `||`, under `!`, in parentheses, in a
  comparison and as a bare name. A `when` that reads one such reference two times gives one mistake.
- [x] An option of the feature, an option of a need, and a starter option that each target declares are not
  mistakes.
- [x] A `when` that does not parse gives only its mistake of Task 1.
- [x] The mistake comes one time for the feature, also with two targets, after its mistakes of Task 1, in a fixed
  order.
- [x] The mistakes of US-006 and Task 1 are still found, and `Catalog.Check` on `catalog/` gives no mistake.
- [x] `feature-compose check` prints the mistake with the other mistakes.
- [x] `docs/commands.md` "Check the Catalog" has a row for the mistake that names the three kinds of readable
  references and the rule of the targets.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/expression.go` — the references that an expression reads
- `internal/catalog/feature.go` — the readable references of the feature, and the mistakes of its `when` expressions
- `internal/catalog/catalog.go` — `Check` gives each feature the starters and the features of the catalog
- `internal/catalog/catalog_test.go` — cases in the group `check`
- `internal/cmd/check_test.go` — `check` prints the mistake
- `docs/commands.md` — "Check the Catalog"

**Patterns to Follow:**
- `internal/catalog/selection.go:293-301` for the references that a feature reads at the render
- `internal/catalog/selection.go:282-291` and `internal/catalog/expression.go:59-62` for the messages of an unknown
  reference at the render
- `internal/catalog/expression.go:38-70` for a walk over the operands of an expression
- `internal/catalog/feature.go:114-138` for the features of the catalog that a feature needs
- `internal/catalog/feature.go:140-168` for the starters that a feature targets, with a mistake that names each of them
- `internal/catalog/option.go:51-54` for the options that a starter or a feature declares
- `internal/catalog/catalog_test.go:648-664` for the messages of options in tests
- `tasks/learnings.md:311-317` for a rule about one feature on `feature`
- `tasks/learnings.md:206-212` for the words of options and needs
- `tasks/learnings.md:199-205` for a relation that each message holds
- `tasks/learnings.md:136-142` for a loop over a map in sorted order
- `tasks/learnings.md:185-191` and `tasks/learnings.md:304-310` for the tests of `internal/catalog` and of the command

**Testable:** Yes

**Certainty:** medium — `Selection.readable` (`internal/catalog/selection.go:293-301`) builds the references of a
feature and `expression.trueFor` looks each name up; but here the set comes from the declarations, with no values, and
a starter option counts only when each target declares it, which no code of this repo computes across the targets

**Blast radius:** low — one more kind of mistake that `check` prints; the render keeps its own lookup

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd`.

**Depends on:** Task 1

---

### Task 3: Find a template that reads an option that rule 5 does not allow, in its parse tree

**Behavior:** `check` parses the path and the contents of each template of each feature and of each starter, with the
delimiters of its owner and the template functions of the render, and finds each `option` call with a reference that
its owner cannot read (decisions 1, 2 and 6). Because it reads the parse tree, it also checks a template that no
combination of CI renders. A template that does not parse is a mistake of its owner (decision 5).

**Acceptance Criteria:**
- [ ] Each `option "<reference>"` in a template of a feature, with a reference that the feature cannot read under
  Task 2, is a mistake. The message names the template with its files entry, the reference and the feature, and for a
  starter option each target that does not declare it.
- [ ] A call counts wherever it is in the parse tree: in an action, in the pipeline, the body or the `else` of `if`,
  `range` and `with`, in a `define`, as a parenthesized argument of another function, and in the path of the template.
- [ ] A template is checked whatever the `when` of its files entry: a template of an entry whose `when` holds for no
  value that a combination of CI selects is still checked.
- [ ] A template is parsed with the delimiters of the first delimiters rule of its owner that matches its catalog
  path: `[[ option "x" ]]` in such a file is a read, and `{{ option "x" }}` in it is text. A template of a feature gets
  these delimiters whatever the starter.
- [ ] A template of the starter that reads a reference other than `starter.<option>` of an option of that starter is
  a mistake of the starter, and the message names the starter, the template and the reference.
- [ ] A template that does not parse is a mistake. The message names the template, its owner and the error of the
  parse, and `check` goes on with the other templates. A template that calls each template function that `new`
  renders with is not such a mistake.
- [ ] A template that reads one reference that it cannot read many times gives one mistake. A template that reads
  only readable references gives none.
- [ ] `new` renders each template with the same delimiters as before: the tests of the Taskfiles, the GitHub workflow
  and `.goreleaser.yaml` (`internal/catalog/catalog_test.go:907-913`, `internal/catalog/catalog_test.go:1648-1654`) still
  pass.
- [ ] The mistakes of US-006 and Tasks 1 and 2 are still found, in a fixed order (decision 9), and `Catalog.Check` on
  `catalog/` gives no mistake.
- [ ] `feature-compose check` prints the mistakes of a template of a feature and of a template of the starter.
- [ ] `e2e/tests/check.test.ts`: `check` on a scratch catalog with a template of a feature that reads a starter option
  that its target does not declare exits with 1, and its line names the template, the option and the feature. The
  test of the catalog of this repo still exits with 0. Both pass on the host.
- [ ] The usage of `check`, its line in the table at `docs/commands.md:7`, the first sentence of "Check the Catalog" in
  `docs/commands.md` and in `docs/catalog.md`, and the `desc` of `test:catalog` in `Taskfile.test.yml` say that `check`
  also checks the templates; `internal/cmd/root_test.go:140` follows the usage. `docs/commands.md` has a row for each
  mistake, and `docs/catalog.md:283-284` says that `check` finds a template that reads an option that it cannot read.
- [ ] `clerk mutants` on the changed lines of `internal/catalog` completes within its time limit, with no mutant that
  waits for its timeout.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/parse_tree.go` — the parse tree of a template, and the quoted texts that each call of one function
  passes
- `internal/catalog/template.go` — the parse tree of the path and the contents of a template, and its delimiters
  whatever the starter
- `internal/catalog/selection.go` — one map of template functions for the render and for the parse
- `internal/catalog/feature.go` — the delimiters of its templates, and the mistakes of the reads and the parse of its
  templates
- `internal/catalog/starter.go` — the mistakes of the reads and the parse of the templates of the starter
- `internal/catalog/catalog.go` — the order of the mistakes of each feature
- `internal/catalog/catalog_test.go` — cases in the group `check`
- `internal/cmd/check.go`, `internal/cmd/check_test.go`, `internal/cmd/root_test.go` — the usage, and `check` prints the
  mistakes
- `e2e/tests/check.test.ts` — the case above
- `docs/commands.md`, `docs/catalog.md`, `Taskfile.test.yml` — the texts above

**Patterns to Follow:**
- `internal/catalog/selection.go:257-269` for the template functions, the delimiters and the parse of the render
- `internal/catalog/template.go:48-63` and `internal/catalog/delimiters_rule.go:14-27` for the delimiters rule that
  matches a catalog path
- `internal/catalog/starter.go:47-66` and `internal/catalog/feature.go:47-57` for the delimiters rules of a starter and of
  a feature at their read
- `internal/catalog/template.go:79-84` for the name of a template in a message
- Task 2 for the readable references and the message of a starter option
- `internal/catalog/catalog_test.go:907-913` and `internal/catalog/catalog_test.go:1648-1654` for the delimiters of `new`
  in tests
- `docs/catalog.md:266-284` for the template functions and the forms of `option`
- `tasks/learnings.md:17-23` for a delimiters glob that matches the catalog path with `.tmpl`
- `tasks/learnings.md:80-86` for a template function as one entry of the map of `internal/catalog/selection.go`
- `tasks/learnings.md:150-156` for a walk by recursion that gives no endless mutant
- `tasks/learnings.md:213-219` for a mutants run of `internal/catalog` that passes its time limit
- `tasks/learnings.md:318-324` for one read of each template in one check
- `tasks/learnings.md:136-142`, `tasks/learnings.md:185-191`, `tasks/learnings.md:304-310` and `tasks/learnings.md:59-65`

**Testable:** Yes

**Certainty:** low — no code of this repo walks a `text/template/parse` tree, and the walk over each kind of node
(`if`, `range`, `with`, `define`, pipelines and parenthesized arguments) is new; the delimiters of a template of a
feature exist today only when a starter places it

**Blast radius:** low — `check` reads and prints; the render of `new` changes only in where it takes its function map
and the delimiters of a template, which the existing tests of the render hold

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` on the changed lines of `internal/catalog` and `internal/cmd` completes with no live mutant
(`tasks/learnings.md:213-219`). Run `e2e/tests/check.test.ts` on the host.

**Depends on:** Task 2

---

### Task 4: Find an extension point whose `file` does not exist or does not call `items` for it

**Behavior:** `check` finds each extension point whose `file` is not a template of its declarer, and each whose file
does not call `items` with the name of the extension point in its parse tree (decision 7). An extension point of the
starter is a mistake of the starter; one of a feature is a mistake of the feature, one time.

**Acceptance Criteria:**
- [ ] An extension point of a starter whose `file` names no template of the starter is a mistake. The message names
  the starter, the extension point and the file. A `file` that only a feature has, and an empty `file`, are such
  mistakes.
- [ ] An extension point of a feature whose `file` names no template of the feature, in any of its files entries, is a
  mistake. The message names the feature, the extension point and the file, and it comes one time, also when the
  feature has two targets.
- [ ] A file whose parse tree does not call `items` with the name of the extension point is a mistake. The message
  names the declarer, the extension point and the file. A file that calls `items` only for another extension point,
  and a file that calls only `unique` for it, are such mistakes. A file that renders two extension points calls
  `items` for each.
- [ ] A `file` matches a template whose catalog path is the `file` with `.tmpl` at its end. A file with other
  delimiters is read with them, as the GitHub workflow of a starter is.
- [ ] A file that does not parse gives only its mistake of Task 3.
- [ ] The fixtures of the group `check` in `TestCatalog` and of `TestCheckCommand` give each extension point that they
  declare a file that calls `items` for it, so each list of mistakes that their US-006 cases and Tasks 1 to 3 expect
  stays the same.
- [ ] The mistakes of US-006 and Tasks 1 to 3 are still found, in a fixed order (decision 9), and `Catalog.Check` on
  `catalog/` gives no mistake: each of its 27 extension points calls `items` in its file.
- [ ] `feature-compose check` prints the mistakes of an extension point of the starter and of a feature.
- [ ] `docs/commands.md` "Check the Catalog" has a row for each mistake, and the "Extension Points" section of
  `docs/catalog.md` says that `check` finds a file that does not exist or does not call `items`.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/extension_point.go` — the file of an extension point, and its mistakes
- `internal/catalog/starter.go` — the file of each extension point of `starter.yml`, and the mistakes of the starter
- `internal/catalog/feature.go` — the file of each extension point of `feature.yml`, and the mistakes of the feature
- `internal/catalog/catalog.go` — the order of the mistakes of each feature
- `internal/catalog/catalog_test.go` — cases in the group `check`, and the files of its fixtures
- `internal/cmd/check_test.go` — `check` prints the mistakes, and the files of its fixtures
- `docs/commands.md`, `docs/catalog.md` — the texts above

**Patterns to Follow:**
- Task 3 for the parse tree of a template and the calls of one function
- `internal/catalog/definitions.go:159-163` for the `file` of an extension point in YAML
- `internal/catalog/starter.go:67-70` and `internal/catalog/feature.go:88-91` for the read of the extension points
- `internal/catalog/extension_point.go:10-15` for the type of an extension point and its declarer
- `internal/catalog/template.go:74-77` for the catalog path of a template without `.tmpl`
- `internal/catalog/catalog_test.go:354-366` and `internal/cmd/check_test.go:29-38` for the fixtures that change
- `docs/catalog.md:187-264` for how a file renders the items of an extension point
- `tasks/learnings.md:318-324` for one parse of each template in one check
- `tasks/learnings.md:311-317` for a rule about one feature on `feature`
- `tasks/learnings.md:136-142`, `tasks/learnings.md:185-191` and `tasks/learnings.md:304-310`

**Testable:** Yes

**Certainty:** medium — Task 3 gives the parse tree and the calls of `items`, and `ExtensionPointDefinition` already
reads the `file`; but no code looks a template up by its catalog path among the templates of a declarer, and every
fixture of the US-006 tests declares an extension point with a file that this task makes a mistake

**Blast radius:** low — more mistakes that `check` prints; `new` does not read the new field

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on the
changed lines of `internal/catalog` and `internal/cmd`.

**Depends on:** Task 3

---

### Task 5: Find two owners that write one path of the repo

**Behavior:** For each starter, `check` finds each path of the repo that two owners write: the starter and a feature
that targets it, or two features that target it (decision 8). It reads each files entry whatever its `when`, and
compares the paths before the render. Each path and each pair of owners is one mistake of the starter, which names the
path and both owners.

**Acceptance Criteria:**
- [ ] A path that the starter and a feature that targets it both write is a mistake. The message names the starter,
  the path and both owners. A path that two features that target the starter both write is a mistake too.
- [ ] The path is the path in the repo: the path of the folder joined with the path in the folder, without `.tmpl`.
  `go.mod.tmpl` of the starter and `go.mod` of a feature write one path, and so do two folders whose paths lead to
  one path.
- [ ] The templates of each files entry count, whatever its `when`.
- [ ] Two templates of one owner that write one path are not a mistake, as the files entries `api-key` and `oauth` of
  `api-auth` write `internal/auth/config.go`.
- [ ] Two features that write one path but have no target in common are not a mistake.
- [ ] Two owners that write many paths give one mistake for each path; three owners of one path give one mistake for
  each pair of them. The mistakes come in a fixed order, after the cycles of the starter (decision 9).
- [ ] A template that is not in a folder of the starter is not in the comparison; US-006 reports it.
- [ ] The mistakes of US-006 and Tasks 1 to 4 are still found, and `Catalog.Check` on `catalog/` gives no mistake.
- [ ] `feature-compose check` prints the mistake.
- [ ] `e2e/tests/check.test.ts`: `check` on a scratch catalog whose starter and feature write one path exits with 1,
  and its line names the path and both owners. The test of the catalog of this repo still exits with 0. Both pass on
  the host.
- [ ] `docs/commands.md` "Check the Catalog" has a row for the mistake, and the rule of one owner in
  `docs/catalog.md:42-43` says that `check` finds two owners of one path.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/starter.go` — the paths that two owners write, among the starter and the features that target it
- `internal/catalog/template.go` — the path in the repo of a template, before the render
- `internal/catalog/catalog_test.go` — cases in the group `check`
- `internal/cmd/check_test.go` — `check` prints the mistake
- `e2e/tests/check.test.ts` — the case above
- `docs/commands.md`, `docs/catalog.md` — the texts above

**Patterns to Follow:**
- `internal/catalog/selection.go:207-227` for the template that writes each path, and the message of `new`
- `internal/catalog/starter.go:208-225` for a walk over the templates of each feature in the folders of the starter
- `internal/catalog/template.go:86-101` for the path in the repo of a template
- `internal/catalog/starter.go:227-244` for a mistake of the starter after the mistakes of its features
- `internal/catalog/catalog_test.go:489-513` for the files entries of one feature with and without `when` in a test
- `e2e/tests/check.test.ts:19-34` for a scratch catalog in an e2e test
- `tasks/learnings.md:136-142` for a loop over a map in sorted order
- `tasks/learnings.md:199-205` for a message that names each owner
- `tasks/learnings.md:185-191`, `tasks/learnings.md:304-310` and `tasks/learnings.md:59-65`

**Testable:** Yes

**Certainty:** high — `Selection.files` (`internal/catalog/selection.go:207-227`) finds the second template that writes a
path, and `Starter.folderMistakes` (`internal/catalog/starter.go:208-225`) walks each template of each feature in the
folders of the starter; this task joins the two over each feature of the starter, with the path that
`template.render` joins

**Blast radius:** low — one more kind of mistake that `check` prints; `new` keeps its own check of the rendered paths

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on the
changed lines of `internal/catalog` and `internal/cmd`. Run `e2e/tests/check.test.ts` on the host.

**Depends on:** None

---

## Summary

- **Tasks:** 5.
- **Order:** dependency first. Task 1 moves a failure that exists into the list of mistakes, and keeps each `when` of a
  feature on `feature`, where Task 2 reads it. Task 2 builds the readable references of rule 5 on the expressions,
  which have a parser already. Task 3 is the risk of the story, the walk of the parse tree, and it reuses the readable
  references of Task 2. Task 4 reuses the parse tree of Task 3 for the calls of `items`. Task 5 depends on no other
  task and can run at any point; it is last because it is the most routine.
- **Assessment:** one task is low certainty: Task 3, because no code here walks a `text/template/parse` tree. Tasks 1,
  2 and 4 are medium, each with its variation: a read that goes on past a failure that another caller must still stop
  on, readable starter options across all the targets, and a template looked up by its catalog path with every US-006
  fixture to change. Task 5 is high. No task has a high blast radius: `check` reads the catalog and prints, its exit
  status does not change, no task touches authentication, money, migrations, secrets or personal data, and the read
  that `new` uses keeps its stops.
- **The catalog of this repo:** no template breaks rule 5, each extension point calls `items` in its file, and no two
  owners write one path, so the breakdown plans no change to `catalog/`.
- **Criteria of the story:**
  - "`check` finds a template or a `when` expression that reads an option that rule 5 does not allow. The message
    names the file or the expression, the option, and the feature": Task 2 for a `when`, Task 3 for a template, also
    one that a CI repo does not render, as the Context asks.
  - "`check` finds a `when` expression that does not parse, and the message names the feature and the expression":
    Task 1. Today `check` stops on it with the message of `new`; Task 1 makes it a mistake in the list.
  - "`check` finds two owners that write one path, and the message names the path and both owners": Task 5. Today
    only `new` stops on it, for the selected features.
  - "`check` finds an extension point whose `file` does not exist, or does not render the items of that extension
    point": Task 4.
- **Deferred:** none of the criteria of this story. The `when` of an option does not exist in the code yet; US-013 adds
  it, and needs a criterion of `check` (see Boundaries).
