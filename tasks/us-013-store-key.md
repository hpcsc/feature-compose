# Tasks: Keep the API Key in the Credential Store (US-013)

## Contents

1. Task 1: Read the `when` of an option of a feature, find one that does not parse, and show it
2. Task 2: Ask an option only while its `when` is true, and stop a flag that gives it another value
3. Task 3: Find a `when` of an option that reads an option that it cannot read
4. Task 4: Give `cli-auth` the option `store-key`, which the tool asks only with the method `api-key`
5. Task 5: Keep the API key in the OS credential store with `store-key=true`

## Story Reference

US-013, "Keep the API key in the credential store", in `user-stories/feature-compose.md:438-461` (Description,
Acceptance Criteria, Context). The design is `PROPOSAL.md`: section 7 (the `feature.yml` of `cli-auth` with the option
`store-key`, its `when: api-auth.method == "api-key"`, and `files/store/` with
`when: api-auth.method == "oauth" || store-key`, `PROPOSAL.md:195-216`), section 8 (an option with `when` is asked
only while its expression is true, else it has its default value, and the expression language,
`PROPOSAL.md:254-293`), section 10 rule 5 (`PROPOSAL.md:339-340`), section 12 (the tree shows an option only while its
`when` is true, `PROPOSAL.md:410-412`) and section 15 (if the answer is `oauth`, the tool does not ask for
`store-key`, `PROPOSAL.md:512-515`). The template to port is the preset `api-key-stored` of `global/go-cli-auth` in
`~/repo-templates/templates/global/go-cli-auth`. US-008 (done today) put the store of the tokens of `oauth` in
`files/store/` with no text of one method, so that this story changes only its `when`
(`tasks/completed/us-008-cli-oauth.md`, decision 2). US-009 kept each `when` of a feature on `feature`, so that this
story adds the `when` of an option to the same checks (`tasks/completed/us-009-check-templates-owners.md`, Deferred).

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

- A `when` on an option of a starter. `starter.yml` with such a `when` stops the read of the starter (decision 3).
- A `when` of an option that reads another option of its own feature. The `when` of an option reads only the options
  of the needs of its feature and the starter options (decision 2). `check` reports such a read as a mistake.
- A check of a need that sets a value of an option whose `when` can be false. `new` stops on such a need when the
  `when` is false (decision 5), and `check` does not look for it.
- Combinations that look for the values of the needs that make the `when` of an option true. An other value of an
  option whose `when` is false with the defaults of the needs gives a combination that `Select` refuses, and
  `task test:combinations` names it. No option of the catalog of this repo is such an option (decision 9).
- A seam in `Store` for an OS credential store that fails one call and not another. The mock of `go-keyring` fails
  each call or none (`tasks/learnings.md:458-463`), so the docs of the stored key name the limits, as the docs of
  `oauth` do.
- A `when` on an item of `add_to`. `golang.org/x/term` goes into the `go.mod` that the tool renders for each repo
  with `cli-auth`, and `go mod tidy` removes it where no file imports it (decision 12, as US-008 decision 3).
- A check of the API key before `auth login` stores it, and a flag of `auth login` that gives the key. The preset
  stores the key that it reads from the terminal or from stdin, so the key is not in the history of the shell.
- A change of the CLI of `oauth`. Its files and its tests stay as they are; only the `when` of `files/store/` reads
  `store-key` too.

**Deferred:** none.

## Codebase Context

**The `when` of a files entry today.** `featureYML` reads `when` only in a files entry
(`internal/catalog/feature_yml.go:14-17`). `newFeature` parses it with `newExpression` and keeps a text that does not
parse in `whenParseMistakes` (`internal/catalog/feature.go:77-83`). `readFeatureFor`, the read for one starter that
`new` uses, stops on the first one (`internal/catalog/feature.go:44-46`), and `feature.mistakes` gives each one to
`check` (`internal/catalog/feature.go:150-160`). `Selection.files` evaluates the `when` with `Selection.readable`: the
options of the feature by name, `<need>.<option>` for each direct need, and `starter.<option>`
(`internal/catalog/selection.go:244-250`, `internal/catalog/selection.go:333-341`). `check` finds a reference that rule
5 does not allow with `feature.readMistakes`, `feature.readable` and `feature.whyCannotRead`
(`internal/catalog/feature.go:162-217`); its messages list the references that the feature can read.

**Options today.** `OptionDefinition` is the YAML of an option in `starter.yml` and in `feature.yml`, and the JSON of
`show` for both (`internal/catalog/definitions.go:22-31`). `newOption` makes the type `option`
(`internal/catalog/option.go:11-28`). `options.withDefaults` gives each option the value of a flag, else of a need,
else of an answer, else its default (`internal/catalog/option.go:64-109`), and a flag that gives another value than a
need stops it (`internal/catalog/values_set_by_needs.go:41-48`). `options.questions` gives one `Question` for each
declared option (`internal/catalog/option.go:127-145`). `optionValues.missing` and `Selection.checkValues` stop an
option with no value and no default (`internal/catalog/option_values.go:34-61`,
`internal/catalog/selection.go:267-278`). `Starter.Select` computes the values of the features in the order of the
needs (`internal/catalog/starter.go:77-131`, `inNeedsOrder` at `internal/catalog/starter.go:398-413`), so the values of
the needs of a feature exist before the values of the feature.

**The form of `new`.** The tree lists the options of each selected feature from `Selection.Questions`
(`internal/form/draft.go:270-292`), and the draft runs `Select` again after each answer
(`internal/form/draft.go:94-116`) and each Space (`internal/form/draft.go:74-92`). When `Select` stops, the form keeps
the old draft and shows the message. So an option that `Selection.Questions` leaves out has no row, and the tree
changes after each answer, with no change in `internal/form` (decision 6). A value that the form kept for an option
comes back when no selected need sets the option (`internal/form/model_test.go:583-589`).

**`show`.** The JSON is `json.Encode` of `Definitions` (`internal/cmd/show.go:34-51`). The TUI lists the default, the
values and the pattern of each option (`internal/browser/details.go:121-135`).

**Combinations.** `Starter.combinations` gives each feature with the defaults of its options with values, each other
value of such an option, and (US-008) each other value of an option of a need with the feature at its defaults
(`internal/catalog/starter.go:145-183`). Each combination gives its values as flags to `Select`
(`internal/catalog/combination_test.go:33`). For `cli-auth` with `store-key`, this gives
`--feature cli-auth:store-key=false`, `--feature cli-auth:store-key=true` (with `api-auth.method` at its default
`api-key`), and `--feature api-auth:method=oauth --feature cli-auth:store-key=false`. The last one gives `store-key`
its default with a flag while its `when` is false (decision 5). No combination gives `store-key=true` with `oauth`.

**`cli-auth` today.** `catalog/features/cli-auth/feature.yml` has `needs: [cli, api-auth]`, no option, four files
entries (`files/common/`; `files/api-key/` when `api-auth.method == "api-key"`; `files/oauth/` and `files/store/` when
`api-auth.method == "oauth"`) and three items of `gomod.requires`. `files/api-key/` holds six files: `credential.go`
(`Resolve`, `ErrMissing`, `ErrRefused`, `Credential.Refused`), `credential_test.go`, `internal/cli/auth.go`
(`auth status` only), `internal/apiclient/client_test.go` (calls against `auth.Middleware` of `api-auth`),
`e2e/tests/auth.test.ts` and `docs/cli-auth.md`. `files/store/` holds `store.go` (`Store` over `go-keyring` and
`credentials.json`, `ErrNotStored` with an inline conditional on the method), `store_test.go`, `api_url.go` (`APIURL`,
`APIHost`, `setting`) and `api_url_test.go`. The catalog has no block conditional in a template; two templates have an
inline conditional of one word (`catalog/features/cli-auth/files/store/go/internal/credential/store.go.tmpl:19`,
`catalog/features/cli-auth/files/common/go/internal/credential/command.go.tmpl:20`).

**The preset `api-key-stored`.** Its `scaffold.yml` sets `stored` and keeps `api_url.go`, `store.go` and
`store_test.go`. Its differences from `api-key`: `credential.go` has `storedKey()` (the stored key of the host of the
API URL) between the variable and `credentialCommand`, and the message of `ErrMissing` names `auth login`;
`.go-cli-auth/auth.go` has `auth login` (the key from the terminal with no echo through `golang.org/x/term`, else from
stdin), `auth logout` and other usage texts; `credential_test.go` has the cases of the stored key, of another host and
of an API URL that is not set; `.go-cli-auth/auth.test.ts` has the tests of `auth login` (the key on stdin) and
`auth logout`; `docs/authentication.md` and `.go-cli-auth/README.md.part` have their own text. Without an API URL,
`Resolve` of the preset stops after the variable with a message that names `apiURL`, also when a `credentialCommand`
is set. The catalog adds to the preset of `api-key`: `Credential.Refused`, the client test against `auth.Middleware`,
and its own e2e tests and docs.

**`golang.org/x/term`.** No file of the catalog imports it. On 2026-11-14, `go mod download` gives `v0.48.0`, whose
`go.mod` requires `golang.org/x/sys v0.50.0`, and `v0.42.0`, which requires `golang.org/x/sys v0.43.0`, the version
of `go.mod.tmpl` of `go-event-sourcing-service`.

**Tests that change with `store-key`.** In `internal/catalog/catalog_test.go`: the questions of a selection with
`cli-auth` (`1974-1990`), the stop of a value for `cli-auth`, which has no option today (`2400-2407`), and the names of
the combinations of the catalog of this repo (`3373-3383`). In `e2e/tests/new.test.ts`: the `.feature-compose.yml` of
the full runs (`cli-auth: {}` at lines 377 and 436), the list of the items of `gomod.requires` and their check in
`go.mod` (lines 379-383 and 444-452), and the form tests, where `cli-auth` gets a row under it
(`e2e/tests/new.test.ts:705-800`). The YAML encoder writes the value `false` as `"false"`
(`tasks/learnings.md:451-456`).

**The proof.** `clerk mutants` checks the tool code of Tasks 1 to 3. It does not change a `.go.tmpl`, so the template
code of Task 5 gets its mutation proof from breaks by hand on a written repo (`tasks/learnings.md:87-92`,
`tasks/learnings.md:129-134`, `tasks/learnings.md:367-372`). `docker info` answered on 2026-11-14, and the e2e tests
also run on the host (`tasks/learnings.md:59-64`).

**Words.** `when` is the field of a files entry, and now of an option. An option whose `when` is false "does not
apply": the tool asks no question for it, and it has its default; the story says that `store-key` "applies only to the
method `api-key`". No identifier of `internal/catalog` uses `apply` today. `store`, `OS credential store`, `API URL`
and `log in` are the words of `tasks/us-008-cli-oauth.design.yaml`. The key that `auth login` keeps is the "stored
key", as `storedKey()` of the preset calls it. `options`, `values`, `given` and `needs` are the words of
`tasks/learnings.md:206-211`.

**Decisions of this breakdown, for the caller to overrule:**

1. **The `when` of an option is a field of the option in `feature.yml`** (Task 1), with the expression language of a
   files entry. The read of the feature parses it. A `when` that does not parse is one mistake of `check` for each
   option, and `new` stops on the first one, as US-009 did for a files entry.
2. **The `when` of an option reads the options of the needs of its feature, as `<need>.<option>`, the starter
   options, as `starter.<option>`, and the options of its own feature that have no `when`, by name** (Tasks 2 and 3).
   It does not read an option of its own feature that has a `when`, so the `when` expressions of one feature form no
   cycle (an option that reads itself). The needs come before the feature (rule 8), so their values exist. In `check`, a starter option
   counts only when each target declares it, as for a files entry, and a read of an option of the own feature that has
   a `when` is a mistake.
3. **A `when` on an option of `starter.yml` stops the read of the starter** (Task 1), so `new`, `--dry-run` and
   `check` stop with a message that names the `starter.yml` and the option. `OptionDefinition` is the JSON of `show`
   for both, so each option of a starter has `"when": ""`, as each text with no value has.
4. **While the `when` of an option is false, the option does not apply** (Task 2). `Selection.Questions` holds no
   question for it, so the form shows no row. It has its default, also an empty default, and then it is not missing.
   A value that the form kept for it is not used, and it comes back when the `when` is true again, as a kept value
   comes back after a need (`internal/form/model_test.go:583-589`). The preview prints it with its default and the
   source `default`, and `.feature-compose.yml` holds the default, so a template reads the value that the preview
   shows.
5. **A flag or a need that gives another value than the default to an option that does not apply stops `Select`**
   (Task 2), so `new`, `--no-prompt`, `--dry-run` and the start of the form stop before any file. The message names the
   option, its feature, the `when` as written, the value of each option that the `when` reads, and the source, for
   example: the option `store-key` of the feature `cli-auth` applies only when `api-auth.method == "api-key"`, and
   `api-auth.method` is `"oauth"`, but the command line gives it `"true"`. A flag that gives the default is accepted,
   with the source `command line`, as a flag that gives the value of a need is accepted
   (`internal/catalog/values_set_by_needs.go:41-48`). The combination
   `--feature api-auth:method=oauth --feature cli-auth:store-key=false` needs this.
6. **The form needs no code** (Task 2). Its tree follows `Selection.Questions` after each answer. An answer for the
   option of a need that makes the `when` false, while a flag gives the option another value than the default, stops
   `Select`, so the form keeps the old value and shows the message, as it does for a Space that a need refuses.
7. **`show` prints the `when` of each option** (Task 1): the key `when` in the JSON of each option (`""` with no
   `when`), and a line `when` in the details of the TUI for an option that has one.
8. **The order of the mistakes of a feature** (Tasks 1 and 3): the `when` expressions of the files entries that do
   not parse, then those of the options, in the order of the option names; then the reads of the `when` of the files
   entries, then the reads of the `when` of the options, in the order of the option names; then the templates.
9. **No change of `Starter.combinations`** (Tasks 2 and 4). The combinations give `store-key=true` with the default
   method `api-key`, and `oauth` with the default `store-key=false`, which decision 5 accepts. Task 2 proves this on a
   catalog of the test, and Task 4 on the catalog of this repo.
10. **The files of `api-key` hold both values of `store-key`** (Task 5). The six files of `files/api-key/` get blocks
    that the render keeps only when `store-key` is `true`, with the text of the preset `api-key-stored`, and with
    `store-key=false` each written file is the same as today, byte for byte, which the tests of `cli-auth` in
    `TestCatalog` hold with their expected values unchanged. The other choice: a files entry `files/api-key-stored/`
    with `when: api-auth.method == "api-key" && store-key` beside `files/api-key/` with `&& !store-key`, which copies
    six files whose texts are mostly the same.
11. **`files/store/` gets `when: api-auth.method == "oauth" || store-key`** (Task 5), as `PROPOSAL.md:210-211` writes
    it. Its files do not change.
12. **`golang.org/x/term` `v0.29.0` is an item of `gomod.requires`** (Task 5), because `auth login` of the stored key
    reads the key from the terminal with it. `v0.29.0` is the version that the module graph of the starter already
    selects, so `go mod tidy` removes the item from a repo where no file imports it, and the `go.mod` of a repo with
    `api-key` or `oauth` and no stored key does not change. A higher version, for example `v0.42.0`, stays in each
    `go.mod` as an indirect requirement.
13. **The messages of the stored key name `auth login`** (Task 5): `ErrMissing`, as the preset does, and
    `Credential.Refused`, which the catalog adds, for example
    `the API refused the API key from the OS credential store: run "my-service auth login"`. As in the preset, the
    stored key needs the API URL: without it and without the variable, `Resolve` stops with the message that names
    `apiURL` and `MY_SERVICE_API_URL`, and the docs say so.

**Open questions for the user:**

1. **Can the `when` of an option read an option of its own feature?** Decision 2 says no, which is narrower than rule
   5 for a files entry. Criterion 6 says "an option that rule 5 does not allow"; with decision 2, `check` also
   reports an option of the own feature.
2. **Does a flag with the default value stop?** Decision 5 accepts it, so a script that always gives
   `store-key=false` works with each method. The story names only `store-key=true`.
3. **One text for each file of `api-key`, or a files entry for the stored key?** Decision 10 picks block conditionals,
   which no template of the catalog has yet.

## Tasks

### Task 1: Read the `when` of an option of a feature, find one that does not parse, and show it

**Behavior:** An option in `feature.yml` can have `when`, with the expression language of a files entry. The read of
the feature parses it (decision 1). A `when` that does not parse stops `new` with the first one, and `check` reports
each one with the other mistakes. A `when` on an option of `starter.yml` stops the read of the starter (decision 3).
`show` prints the `when` of each option in the JSON and in the details of the TUI (decision 7). The `when` does not
change a selection yet; Task 2 adds that.

**Acceptance Criteria:**
- [x] `Catalog.Definitions` gives the `when` of each option of a feature as the file writes it, and `""` for an option
  with no `when` and for each option of a starter. The JSON of the definitions holds `"when"` in each option
  (the group `definitions` of `TestCatalog`).
- [x] `feature-compose show --output json` prints `"when"` in each option of each starter and each feature
  (`TestShowCommand`, the whole JSON).
- [x] The details of a feature in the TUI of `show` show a line `when` with the expression under an option that has
  one, after its default, its values and its pattern. An option with no `when` shows no such line.
- [x] `Catalog.Starter` of a catalog with a feature that targets the starter and has an option whose `when` does not
  parse stops. The message names the `feature.yml`, the option, the `when` as written, and why, for each text of the
  table at `internal/catalog/catalog_test.go:235-247`.
- [x] `Catalog.Check` gives one mistake for each option whose `when` does not parse; two such options of one feature
  give two mistakes. The message names the feature, the option, the `when` as written, and why. It comes one time for
  the feature, also with two targets, after the `when` expressions of the files entries that do not parse
  (decision 8). The other mistakes of the feature and of the catalog are still in the list.
- [x] An option with `when` in `starter.yml` stops `Catalog.Starter` and `Catalog.Check`. The message names the
  `starter.yml` and the option, and says that only an option of a feature can have a `when`.
- [x] `feature-compose check` prints the mistake with the other mistakes, and stops with the count
  (`TestCheckCommand`). `e2e/tests/check.test.ts`: `check` on a scratch catalog with an option whose `when` does not
  parse exits with 1, prints one line that names the feature, the option and the expression, and writes
  `the catalog has 1 mistake` to stderr.
- [x] `docs/commands.md`: the table of the JSON of `show` has `starters.<name>.options.<name>.when` (always `""`) and
  `features.<name>.options.<name>.when`; the table of the details of the TUI names the `when` of an option; "Check
  the Catalog" has a row for a `when` of an option that does not parse, and its order names it.
- [x] `Catalog.Check` on `catalog/` gives no mistake. `go build ./...`, `go vet ./...`,
  `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and `e2e/tests/check.test.ts` and
  `e2e/tests/show.test.ts` pass.

**Affected Files/Modules:**
- `internal/catalog/definitions.go` — `OptionDefinition.When`
- `internal/catalog/option.go` — the `when` of an option
- `internal/catalog/feature.go` — the parse of the `when` of each option, and its mistake
- `internal/catalog/starter.go` — `readStarter` stops on an option with `when`
- `internal/browser/details.go` — the line `when` of an option
- `internal/catalog/catalog_test.go` — the groups `starter`, `definitions` and `check`
- `internal/cmd/show_test.go`, `internal/cmd/check_test.go`, `internal/browser/model_test.go`
- `e2e/tests/check.test.ts`
- `docs/commands.md` — "Show the Catalog" and "Check the Catalog"

**Patterns to Follow:**
- `internal/catalog/feature.go:77-83` for the parse of a `when` and its mistake
- `internal/catalog/feature.go:32-48` for the read for one starter, which stops on the first mistake
- `internal/catalog/feature.go:150-160` for the mistakes of a feature, one time in the list
- `internal/catalog/starter.go:33-39` for the read of the options of a starter
- `internal/catalog/definitions.go:22-31` for the fields of `OptionDefinition` and their JSON
- `internal/browser/details.go:121-135` for the lines of an option in the details
- `internal/catalog/catalog_test.go:235-262` for the texts that do not parse
- `internal/catalog/catalog_test.go:302-352` for the JSON of the definitions
- `internal/catalog/catalog_test.go:808-858` for the mistakes of a `when` that does not parse
- `internal/cmd/show_test.go:95-121` for the whole JSON of `show`
- `internal/browser/model_test.go:256-276` for the details of the options
- `internal/cmd/check_test.go:117-131` for a test of the command
- `e2e/tests/check.test.ts:36-51` for a scratch catalog with a `when` that does not parse
- `tasks/learnings.md:185-190` for a test in the package of the code
- `tasks/learnings.md:304-309` for a test of the command that reaches the new path
- `tasks/learnings.md:318-323` for one read of each feature in one check
- `tasks/learnings.md:290-295` for `go vet -tags combinations`
- `tasks/learnings.md:59-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** high — it repeats the parse of the `when` of a files entry into `whenParseMistakes`
(`internal/catalog/feature.go:77-83`) and the fields of `OptionDefinition` in the JSON and the details of `show`
(`internal/catalog/definitions.go:22-31`, `internal/browser/details.go:121-135`)

**Blast radius:** high — a contract consumed outside this repository: it adds the key `when` to each option in the JSON
of `show`, which scripts read

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog`, `internal/cmd` and
`internal/browser`. Run `e2e/tests/check.test.ts` and `e2e/tests/show.test.ts` on the host or with `task test:e2e`.

**Depends on:** None

---

### Task 2: Ask an option only while its `when` is true, and stop a flag that gives it another value

**Behavior:** `Select` evaluates the `when` of each option of each selected feature against the values of the needs
of the feature and of the starter (decision 2). While the `when` is false, the option does not apply (decision 4): no
question, its default, and no use of a kept answer. A flag or a need that gives it another value than the default
stops `Select` before any file, and the message says when the option applies (decision 5). The form shows the option
only while its `when` is true, and the tree changes after each answer (decision 6).

**Acceptance Criteria:**
- [x] In a catalog of the test where `client` needs `auth` (`method` with `values: [api-key, oauth]`,
  `default: api-key`) and `client` has `store-key` (`values: [true, false]`, `default: false`,
  `when: auth.method == "api-key"`): with `auth.method` at `api-key` from the default, a flag or an answer,
  `Selection.Questions` holds the question of `store-key`, and `--feature client:store-key=true` gives the templates
  `true` with the source `command line`.
- [x] With `auth.method` at `oauth` from a flag or an answer, `Selection.Questions` holds no question of `store-key`,
  the templates of `client` read `false`, and the preview prints `client.store-key: false (default)`.
- [x] With `oauth`, an answer `true` for `store-key` is not used: the value is `false` with the source `default`, and
  `Selection.Arguments` names no `store-key`. With `api-key` again, the answer gives `true` with the source
  `answered`.
- [x] With `oauth`, `--feature client:store-key=true` stops `Select`. The message names the option `store-key`, the
  feature `client`, the `when` `auth.method == "api-key"`, `auth.method` with the value `"oauth"`, and the command
  line with the value `"true"`. `--feature client:store-key=false` does not stop, and its source is `command line`.
- [x] With `oauth`, a selected feature whose need sets `store-key: "true"` of `client` stops `Select`, and the message
  names that need. A need that sets `"false"` does not stop.
- [x] An option that does not apply and has no default is not missing: `Select` and `Selection.Preview` do not stop on
  it, and `Selection.Questions` holds no question for it.
- [x] A `when` of an option that reads `starter.<option>` reads the value of the starter option. A `when` that reads a
  name that it cannot read (decision 2) stops `Select` with a message that names the option, the feature and the
  `when`. The tests take one name of each form of the expression language: a name with no prefix (an option of its own
  feature), `<feature>.<option>` (of a need of a need, of a feature that is not a need, and of an option that a need
  does not declare), and `starter.<option>` (of an option that the starter does not declare).
- [x] A feature with no option with `when` gives the same values, questions, preview and files as before: the tests
  of the groups `starter` and `combinations` of `TestCatalog` and of `TestSelection` pass with their expected values
  unchanged.
- [x] `Catalog.Combinations` of the catalog of the test holds `api --feature client:store-key=true` and
  `api --feature auth:method=oauth --feature client:store-key=false`, no combination gives `store-key=true` with
  `auth:method=oauth`, and `Select` of each combination does not stop (decision 9).
- [x] `feature-compose new` with `--feature client:store-key=true --feature auth:method=oauth`, with `--no-prompt` and
  with `--dry-run`, in both orders of the flags, stops with the message, prints nothing, and makes no output folder.
  In a terminal, it stops before the form starts and writes nothing.
- [x] The form (`TestModel` with that catalog): with `--feature client`, the tree shows the row `store-key  false`
  under `client` while `auth.method` is `api-key`. After Enter at `method` and the choice `oauth`, the tree has no row
  of `store-key`; after `api-key` again, the row shows again, with the value that the developer kept. The preview of
  `w` with `oauth` prints `client.store-key: false (default)`. With `--feature client:store-key=true`, the choice
  `oauth` for `method` leaves `method  api-key`, and the details show the message of `Select`.
- [x] `docs/catalog.md`, "Options": the table has the row `when`, and the text says what a `when` of an option reads
  (decision 2), that the option has its default and no question while its `when` is false, and that a flag or a need
  with another value then stops `new`, with an example. `docs/commands.md`: the source `default` of the preview
  covers an option whose `when` is false; the stop of `--dry-run` lists the flag of an option that does not apply;
  "Answer the Questions in a TUI" says that the tree shows an option only while its `when` is true, and that a kept
  value comes back when the `when` is true again.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  `Catalog.Check` on `catalog/` gives no mistake.

**Affected Files/Modules:**
- `internal/catalog/option.go` — `withDefaults` and `questions` with the `when` of each option
- `internal/catalog/option_values.go` — an option that does not apply, its source and `missing`
- `internal/catalog/starter.go` — `Select` gives each feature the values that the `when` of its options read
- `internal/catalog/catalog_test.go`, `internal/catalog/selection_test.go` — the selection, the preview, the stops and
  the combinations of an option with `when`
- `internal/cmd/new_test.go` — the stop of `new`, `--dry-run` and the terminal
- `internal/form/model_test.go` — the tree, the kept value and the refused answer
- `docs/catalog.md` — "Options"
- `docs/commands.md` — "Preview a Repo" and "Answer the Questions in a TUI"

**Patterns to Follow:**
- `internal/catalog/option.go:64-109` for the value of each option and the stop of a flag against a need
- `internal/catalog/values_set_by_needs.go:41-48` for a flag with another value that stops, and a flag with the same
  value that does not
- `internal/catalog/option.go:127-145` for the questions of a feature
- `internal/catalog/option_values.go:34-61` for `missing` and the source of a value
- `internal/catalog/starter.go:77-131` for the values of the features in the order of the needs
- `internal/catalog/selection.go:333-341` for the values that a feature reads from its needs and the starter
- `internal/catalog/expression.go:38-70` for the evaluation of an expression and the message of an unknown name
- `internal/catalog/catalog_test.go:3269-3299` for `Select` of each combination of a catalog of the test
- `internal/cmd/new_test.go:975-993` for a stop of `new` and `--dry-run` that makes no folder
- `internal/cmd/new_test.go:818-829` for a stop before the TUI starts
- `internal/form/model_test.go:44-52` for a catalog of the form with a need
- `internal/form/model_test.go:344-349` for a change that `Select` refuses and the message in the details
- `internal/form/model_test.go:583-589` for a kept value that comes back
- `tasks/learnings.md:206-211` for the words of options, values and needs
- `tasks/learnings.md:136-141` for a loop over a map in the order of its keys
- `tasks/learnings.md:199-204` for the source of each option, also of an option that does not apply
- `tasks/learnings.md:269-274` for the caller gaps of `internal/form`
- `tasks/learnings.md:213-218` for a `clerk mutants` run of `internal/catalog` that passes its limit
- `tasks/learnings.md:430-435` for the names of `internal/catalog` that the audit renames
- `tasks/learnings.md:220-225` for the tests of `new` in a pseudo-terminal
- `tasks/learnings.md:290-295` for `go vet -tags combinations`

**Testable:** Yes

**Certainty:** medium — a flag that gives another value than a need stops `withDefaults`
(`internal/catalog/values_set_by_needs.go:41-48`) and `Selection.files` evaluates a `when` against the values of the
needs (`internal/catalog/selection.go:244-250`); the variation is that the values of one feature now depend on the
values of another feature, which `withDefaults` has never read, and a kept answer that `Select` does not use

**Blast radius:** low — none of the named categories; `Select` stops before any write, and `new` writes only into an
empty folder

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog`; when it stops at its limit, run
it with `--base HEAD` on the uncommitted change (`tasks/learnings.md:213-218`). Read the lines of `internal/form` that
`clerk audit accept` reports as caller gaps against a coverage profile of `TestModel` (`tasks/learnings.md:269-274`).

**Depends on:** Task 1

---

### Task 3: Find a `when` of an option that reads an option that it cannot read

**Behavior:** `check` finds each reference in the `when` of an option that the `when` cannot read (decision 2), and
names the feature, the option, the `when`, the reference and the references that the `when` can read. It checks every
`when`, whatever value its options can have.

**Acceptance Criteria:**
- [x] A `when` of an option that reads a reference that it cannot read (decision 2) is a mistake. The tests take one
  reference of each form that the expression language gives: a name with no prefix (an option of its own feature that
  has a `when`, also the option itself, and a name that is no option); `<feature>.<option>` (of an option that a need does not declare,
  of a feature that is not in `needs`, and of a need of a need); and `starter.<option>` (of an option that no target
  declares). The message names the feature, the option, the `when` as written, the reference, and the references
  that the `when` of an option can read.
- [x] `starter.<option>` of an option that one target declares and another target does not is a mistake, and the
  message names each target that does not declare it.
- [x] Each reference in the `when` is checked: on both sides of `&&` and `||`, under `!`, in parentheses, in a
  comparison and as a bare name. A `when` that reads one such reference two times gives one mistake.
- [x] A `when` that reads only options of the needs, starter options that each target declares, and options of its
  own feature that have no `when` is not a mistake.
  A `when` that does not parse gives only its mistake of Task 1.
- [x] The mistake comes one time for the feature, also with two targets, after the reads of the `when` of the files
  entries and before the reads of the templates, in the order of the option names (decision 8).
- [x] The mistakes of US-006, US-009 and Task 1 are still found, and `Catalog.Check` on `catalog/` gives no mistake.
- [x] `feature-compose check` prints the mistake with the other mistakes (`TestCheckCommand`).
- [x] `docs/commands.md`, "Check the Catalog": a row for the mistake that names what the `when` of an option reads
  and the rule of the targets, and the order list names it.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/feature.go` — the references that the `when` of an option can read, and its read mistakes
- `internal/catalog/catalog_test.go` — cases in the group `check`
- `internal/cmd/check_test.go` — `check` prints the mistake
- `docs/commands.md` — "Check the Catalog"

**Patterns to Follow:**
- `internal/catalog/feature.go:162-183` for the read mistakes of the `when` of the files entries
- `internal/catalog/feature.go:185-217` for the readable references and the message of a reference that the feature
  cannot read
- `internal/catalog/expression.go:72-85` for the references of an expression
- `internal/catalog/catalog_test.go:862-906` for the cases of the references that a `when` cannot read
- `internal/cmd/check_test.go:132-144` for a test of the command that prints a read mistake
- `task:1` for the `when` of an option and the order of the mistakes of a feature
- `tasks/learnings.md:311-316` for a rule about one feature on `feature`
- `tasks/learnings.md:136-141` for a loop over a map in the order of its keys
- `tasks/learnings.md:185-190` and `tasks/learnings.md:304-309` for the tests of `internal/catalog` and of the command
- `tasks/learnings.md:430-435` for the names of `internal/catalog` that the audit renames

**Testable:** Yes

**Certainty:** high — it repeats the check of the references of the `when` of a files entry (US-009 Task 2,
`internal/catalog/feature.go:162-217`) over the `when` of the options, with the references of the needs and of the
starter that `feature.readable` already builds

**Blast radius:** low — none of the named categories; `check` prints one more kind of mistake, and `Select` keeps its
own lookup

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog` and `internal/cmd`.

**Depends on:** Task 1, Task 2

---

### Task 4: Give `cli-auth` the option `store-key`, which the tool asks only with the method `api-key`

**Behavior:** `catalog/features/cli-auth/feature.yml` declares `store-key` with the values `true` and `false`, the
default `false`, and `when: api-auth.method == "api-key"`. The tool asks for `store-key` with `api-key`, does not ask
for it with `oauth`, and stops `--feature cli-auth:store-key=true` with `method=oauth` before it writes a file. No file
of `cli-auth` reads `store-key` yet; Task 5 gives it its files.

**Acceptance Criteria:**
- [x] `feature-compose show --output json` prints `features["cli-auth"].options` as
  `{"store-key": {"default": "false", "pattern": "", "values": ["true", "false"], "when": "api-auth.method == \"api-key\""}}`
  (`e2e/tests/show.test.ts`).
- [x] `Selection.Questions` of a selection of the catalog of this repo with `cli-auth`: with `api-key`, the questions
  of the features are `api-auth.method` and `cli-auth.store-key`; with `oauth`, only `api-auth.method`
  (`internal/catalog/catalog_test.go:1974-1990`).
- [x] A value of `method` for `cli-auth` stops `Select`, and the message lists `store-key` as the option of
  `cli-auth` (`internal/catalog/catalog_test.go:2400-2407`).
- [x] `Catalog.Combinations` of the catalog of this repo holds `go-event-sourcing-service --feature cli-auth:store-key=false`,
  `go-event-sourcing-service --feature cli-auth:store-key=true` and
  `go-event-sourcing-service --feature api-auth:method=oauth --feature cli-auth:store-key=false`; no combination holds
  both `api-auth:method=oauth` and `cli-auth:store-key=true`; and `Select` of each combination of the catalog of this
  repo does not stop.
- [x] `new go-event-sourcing-service my-service --feature cli-auth:store-key=true --feature api-auth:method=oauth`,
  with `--no-prompt` and with `--dry-run`, exits with a status that is not 0, makes no folder, and the message names
  `store-key`, `api-auth.method == "api-key"` and `api-auth.method` with `"oauth"`. In a terminal, it stops before the
  form starts.
- [x] `--dry-run --feature cli-auth` prints `cli-auth.store-key: false (default)`; with `api-auth:method=oauth` too it
  prints `cli-auth.store-key: false (default)`; `--dry-run --feature cli-auth:store-key=true` prints
  `cli-auth.store-key: true (command line)` and `api-auth.method: api-key (default)`.
- [x] The e2e tests of the form: with `--feature cli-auth`, the row `store-key  false` is under `cli-auth`; after Enter
  at `method` and the choice `oauth`, the tree has no row of `store-key`; after `api-key` again, the row shows again;
  the preview of `w` with `oauth` prints `cli-auth.store-key: false (default)`. The other form tests pass with the new
  row.
- [x] The full runs of `cli-auth` with `api-key` and with `oauth` write `.feature-compose.yml` with
  `cli-auth:\n    store-key: "false"` and pass.
- [x] `docs/catalog.md`: the `feature.yml` of `cli-auth` shows the option `store-key` with its `when`, and "Needs"
  says that `cli-auth` asks `store-key` only with `api-key`. `docs/commands.md`: the examples of the preview and of
  the form of `cli-auth` show `cli-auth.store-key`, and the stop of a flag of an option that does not apply has the
  example of `store-key` with `oauth`.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass,
  `task test:catalog` finds no mistake, and the e2e tests of `new`, `show` and `check` pass.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — the option `store-key`
- `internal/catalog/catalog_test.go` — the questions, the stop and the combinations of `cli-auth`
- `e2e/tests/new.test.ts` — the stop, the preview, the form, and the `.feature-compose.yml` of the full runs
- `e2e/tests/show.test.ts` — the options of `cli-auth`
- `docs/catalog.md`, `docs/commands.md` — the option `store-key`

**Patterns to Follow:**
- `catalog/features/api-auth/feature.yml:1-14` for an option with `values` and a `default`
- `task:2` for the `when` of an option, its stop and its form
- `internal/catalog/catalog_test.go:1974-1990` for the questions of a selection with `cli-auth`
- `internal/catalog/catalog_test.go:3373-3383` for the combinations of the catalog of this repo
- `internal/catalog/catalog_test.go:3269-3299` for `Select` of each combination
- `e2e/tests/new.test.ts:332-363` for a stop of `new` and `--dry-run` that makes no folder
- `e2e/tests/new.test.ts:520-550` for the preview of `cli-auth`
- `e2e/tests/new.test.ts:774-800` for the form test of `method` with `--feature cli-auth`
- `e2e/tests/new.test.ts:364-380` for the `.feature-compose.yml` of a full run
- `e2e/tests/show.test.ts:6-19` for the JSON of `show` of the catalog of this repo
- `tasks/learnings.md:451-456` for the YAML of `false`
- `tasks/learnings.md:444-449` for the tests that compare the whole `.feature-compose.yml`
- `tasks/learnings.md:346-351` for the j presses of the e2e tests of the form
- `tasks/learnings.md:227-232` for the late start of each command in tuistory
- `tasks/learnings.md:59-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** high — it repeats the change of `cli-auth/feature.yml` and of its tests of the questions, the preview,
the form and the combinations that US-008 Task 3 made (`internal/catalog/catalog_test.go:1974-1990`,
`e2e/tests/new.test.ts:774-800`)

**Blast radius:** low — none of the named categories; it adds an option whose value changes no file until Task 5, and
the shape of `.feature-compose.yml` and of the JSON of `show` stays

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass, and `task test:catalog` finds no mistake. Run `e2e/tests/new.test.ts`, `e2e/tests/show.test.ts` and
`e2e/tests/check.test.ts` on the host, or `task test:e2e`. Break the catalog one time and copy the file back: remove
the `when` of `store-key`; the e2e test of the form and the e2e test of the stop fail.

**Depends on:** Task 2

---

### Task 5: Keep the API key in the OS credential store with `store-key=true`

**Behavior:** With `api-key` and `store-key=true`, the CLI is the preset `api-key-stored` of `global/go-cli-auth`,
ported with sentinel values into blocks on `option "store-key"` in the files of `files/api-key/` (decision 10), with the
additions of the catalog. `auth login` reads the API key from the terminal with no echo, or from stdin, and stores it
under the host of the API URL in the OS credential store, or in `credentials.json` when the OS credential store fails.
Each call of the CLI to its API takes `MY_SERVICE_API_KEY`, else the stored key of the host of the API URL, else the
output of `credentialCommand`. `auth logout` removes the stored key, and `auth status` names the source.
`files/store/` gets `when: api-auth.method == "oauth" || store-key` (decision 11), and `golang.org/x/term` is an item
of `gomod.requires` (decision 12). With `store-key=false`, the repo is the same as today, with no code for the store.

**Acceptance Criteria:**
- [x] `catalog/features/cli-auth/feature.yml` has `files/store/` with `when: api-auth.method == "oauth" || store-key`,
  the item `golang.org/x/term` `v0.29.0` of `gomod.requires`, and a `summary` that names the stored API key.
  `task test:catalog` finds no mistake.
- [x] With `store-key=false`, the files that `cli-auth` writes with `api-key` and their contents are the same as
  today: the tests of `cli-auth` in `TestCatalog` with `api-key` (the files at
  `internal/catalog/catalog_test.go:2276-2300`, the words at `internal/catalog/catalog_test.go:2324-2351`, the docs at
  `internal/catalog/catalog_test.go:2450-2466`) pass with their expected values unchanged. No file holds `keyring`,
  `credential store` or `auth login`, and no Go file imports `github.com/zalando/go-keyring` or `golang.org/x/term`.
- [x] With `store-key=true`, the files that `cli-auth` adds are those of `files/common/`, `files/api-key/` and
  `files/store/` (`store.go`, `store_test.go`, `api_url.go`, `api_url_test.go`), none of `files/oauth/`, and only
  `internal/cli/root.go` and `go.mod` of the starter, `cli` and `api-auth` change. No file holds `issuer`,
  `audience`, `oauth` or `refresh` (in any case), and no Go file imports `golang.org/x/oauth2` or
  `github.com/cli/browser`.
- [x] With `store-key=true`, no written file holds `.Scaffold.`, `.Computed.`, `go-cli-project`, `GO_CLI_PROJECT`,
  `zzsentinel` or `<no value>`; the Go files of `cli-auth` import only its own packages and `cli`, and only a test
  imports `internal/auth` of `api-auth`; the `go.mod` of the selection requires a version of each module outside the
  standard library that a Go file of `cli-auth` imports.
- [x] A repo with `store-key=true`, from the built binary with `--feature cli-auth:store-key=true --no-prompt`, passes
  `gofmt -l .` with no file, `go build ./...`, and `task test:unit` with `internal/credential`, `internal/apiclient`,
  `internal/auth` and `internal/cli` among the packages that passed. Its unit tests show:
  - **resolve:** `MY_SERVICE_API_KEY` comes first, then the stored key of the host of the API URL, then the output of
    `credentialCommand`. A key stored for another host is not given. With no source, the error is `ErrMissing`, and it
    names `my-service auth login`, `MY_SERVICE_API_KEY` and `credentialCommand`. Without the API URL and the variable,
    the error names `apiURL` and `MY_SERVICE_API_URL` (decision 13).
  - **store:** the tests of `store_test.go` pass as they do with `oauth`.
  - **API client:** a key stored for the host of the API reaches the handler behind `auth.Middleware` of `api-auth`
    with `api-key` through `apiclient.Do`; `MY_SERVICE_API_KEY` wins over the stored key; a stored key that the API
    does not know fails the call, and the message names the source of the key and `my-service auth login`.
- [x] With `store-key=true`, the root command lists `auth` after `version` and `update`, and `auth` has `status`,
  `login` and `logout`, and no `token`. `auth status` never prints the key. The CLI has no flag for a key.
- [x] With `store-key=true`, `e2e/tests/auth.test.ts` of the written repo, with `MY_SERVICE_CREDENTIAL_STORE=file`:
  `auth login` stores the key that it reads from stdin in `credentials.json` with the mode `0600`, and `auth status`
  then names that file; `MY_SERVICE_API_KEY` wins over the stored key; a key stored for another host is not given;
  `auth logout` removes the key, and with no stored key it says that none is stored; with no source, the message
  names `my-service auth login`. `task test:e2e:local` of the written repo passes on the host with
  `tests/auth.test.ts` among the files that passed.
- [x] With `store-key=true`, `docs/cli-auth.md` names the method `api-key` and tells a user how to store the key with
  `auth login` (in the terminal, or on stdin), what `auth status` and `auth logout` do, where the key is, and the order
  of `MY_SERVICE_API_KEY`, the stored key and `credentialCommand`. It tells a developer the setting of the API URL
  with its constant and its variable, how a command gives the API client `credential.APIURL()`, the store and its
  limits (`tasks/learnings.md:458-463`), and the tests. It names only files that the repo has.
- [x] An e2e full run, `new go-event-sourcing-service my-service --feature cli-auth:store-key=true --no-prompt`:
  - ends with the exit status 0, and writes `.feature-compose.yml` with `cli-auth:\n    store-key: "true"`;
  - gives a `go.mod` that requires `github.com/zalando/go-keyring` and `golang.org/x/term` at the versions of their
    items, read from the JSON of `show`, and neither `golang.org/x/oauth2` nor `github.com/cli/browser`;
  - gives a repo in which `gofmt -l .` lists no file, `go build ./...` builds, `task test:unit` passes with
    `internal/credential`, `internal/apiclient`, `internal/auth` and `internal/cli` among the packages that passed, and
    `task test:e2e:local` passes with `tests/auth.test.ts` among the files that passed;
  - starts the API with the hash of a key in `API_KEY_HASHES`, and runs `auth login` of the built CLI with that key on
    stdin, `MY_SERVICE_API_URL` of the API, `MY_SERVICE_CREDENTIAL_STORE=file`, an empty `MY_SERVICE_API_KEY` and an
    empty config folder: it ends with 0;
  - calls `POST /api/accounts` with `{}` through `apiclient.Do` (`builtCall`) with no `MY_SERVICE_API_KEY`, and gets
    the answer 400 of the handler, so the API accepted the stored key;
  - with `MY_SERVICE_API_KEY` of a key that the API does not know, the call fails and the message names
    `MY_SERVICE_API_KEY`, so the variable still comes first;
  - runs `auth logout`, which ends with 0, and the next call fails with a message that names `my-service auth login`.
- [x] The full run of `cli-auth` with `api-key` checks that the `go.mod` of the repo requires none of
  `github.com/zalando/go-keyring`, `golang.org/x/term`, `golang.org/x/oauth2` and `github.com/cli/browser`. The full
  run of `oauth` checks the modules that its Go files import, at the versions of their items, and passes.
- [x] `README.md`: the table of the features says that `cli-auth` can keep the API key in the OS credential store with
  `store-key`. `docs/catalog.md` shows `files/store/` with its `when` in the `feature.yml` of `cli-auth`.
  `docs/e2e-tests.md` tells the full run of the stored key.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, `task test:catalog` finds no mistake,
  `task test:combinations` passes with `go-event-sourcing-service --feature cli-auth:store-key=true` among the
  combinations, and the e2e tests of `new`, `show` and `check` pass.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — the `when` of `files/store/`, the item `golang.org/x/term`, the summary
- `catalog/features/cli-auth/files/api-key/go/internal/credential/credential.go.tmpl` — the stored key, and the
  messages that name `auth login`
- `catalog/features/cli-auth/files/api-key/go/internal/credential/credential_test.go.tmpl` — the cases of the stored key
- `catalog/features/cli-auth/files/api-key/go/internal/cli/auth.go.tmpl` — `auth login` and `auth logout`
- `catalog/features/cli-auth/files/api-key/go/internal/apiclient/client_test.go.tmpl` — the stored key against
  `auth.Middleware`
- `catalog/features/cli-auth/files/api-key/go/e2e/tests/auth.test.ts` — the e2e tests of the written repo
- `catalog/features/cli-auth/files/api-key/go/docs/cli-auth.md` — the docs of the stored key
- `internal/catalog/catalog_test.go` — the files, the words, the imports, `go.mod` and the docs with `store-key=true`
- `e2e/tests/new.test.ts` — the full run of the stored key, and the modules of the other full runs
- `README.md`, `docs/catalog.md`, `docs/e2e-tests.md`

**Patterns to Follow:**
- `task:4` for the option `store-key`
- `catalog/features/cli-auth/files/store/go/internal/credential/store.go.tmpl:19` for a conditional on an option in a
  template of `cli-auth`
- `catalog/features/cli-auth/files/oauth/go/internal/credential/credential.go.tmpl:28-54` for `Resolve` with a stored
  credential before `credentialCommand`, and `Refused` that names `auth login`
- `catalog/features/cli-auth/files/oauth/go/e2e/tests/auth.test.ts:1-40` for the isolation of the e2e tests of a repo
  with a store
- `catalog/features/cli-auth/files/api-key/go/internal/apiclient/client_test.go.tmpl:1-70` for a client test against
  `auth.Middleware`
- `catalog/features/cli/feature.yml:16-22` for the items of `gomod.requires`
- `internal/catalog/catalog_test.go:2240-2330` for the files and the words of each variant of `cli-auth`
- `internal/catalog/catalog_test.go:2409-2466` for the values of scaffold, the imports and the docs of `cli-auth`
- `e2e/tests/new.test.ts:364-423` for the full run of `cli-auth` with the API and `builtCall`
- `e2e/tests/new.test.ts:425-452` for `task test:e2e:local` of a written repo and the modules of the items
- `docs/e2e-tests.md` for the docs of the full runs
- `tasks/learnings.md:24-29` for the port with sentinel values
- `tasks/learnings.md:10-15` for `.tmpl`
- `tasks/learnings.md:157-162` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:122-127` for a test that reads an environment variable
- `tasks/learnings.md:458-463` for the limits of the mock of `go-keyring`
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134`, `tasks/learnings.md:367-372` and
  `tasks/learnings.md:479-484` for the breaks by hand
- `tasks/learnings.md:94-99` and `tasks/learnings.md:374-379` for the cost of a port, which is its proof
- `tasks/learnings.md:31-36` for the old defects of the preset that an audit raises
- `tasks/learnings.md:38-43`, `tasks/learnings.md:52-57` and `tasks/learnings.md:108-113` for the runs of `go`,
  `task` and the API in a written repo
- `tasks/learnings.md:381-386` for snake case packages and a fake beside the code that it fakes

**Testable:** Yes

**Certainty:** medium — the port of the preset `oauth` with the store (US-008 Task 1) and the client test against
`auth.Middleware` (`catalog/features/cli-auth/files/api-key/go/internal/apiclient/client_test.go.tmpl:1-70`) are the
precedents; the variation is blocks of Go code under a conditional on an option, where the catalog has only an inline
conditional of one word (`catalog/features/cli-auth/files/store/go/internal/credential/store.go.tmpl:19`), and a key
that the CLI reads from the terminal with `golang.org/x/term`

**Blast radius:** high — credential handling: `auth login` keeps the API key of the user in the OS credential store or
in a file, the CLI sends it to the API, and `auth logout` removes it

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, and `task test:catalog` finds no mistake.
Write a repo with `store-key=true` and one with `store-key=false` with the built binary, and run `gofmt -l .`,
`go build ./...`, `task test:unit` and `task test:e2e:local` in each. In the repo with `store-key=true`, apply one
break at a time to the written files, read its diff, run `go test -tags unit -timeout 60s ./internal/credential/
./internal/apiclient/`, read the name of the failed test, and copy the file back. Write each break as a condition, so
that the code still builds: `Resolve` takes the stored key before the variable; `Resolve` gives a key stored for
another host; `Resolve` skips the stored key; `Resolve` without an API URL goes on to `credentialCommand`; the message
of `ErrMissing` does not name `auth login`; `Refused` does not name `auth login`. Break `auth login` so that it stores
an empty key, and `auth logout` so that it keeps the key, and run `task test:e2e:local`: a test of `auth.test.ts`
fails. Break the catalog one time and copy it back: `files/store/` with `when: api-auth.method == "oauth"`; the full
run of the stored key fails. On macOS, run `auth login` of the repo with `store-key=true` without
`MY_SERVICE_CREDENTIAL_STORE`, read the item `my-service` with `security find-generic-password -s my-service`, run
`auth logout`, and read that the item is gone. Run `task test:combinations` and the e2e tests of `new` on the host.

**Depends on:** Task 4

## Summary

- **Tasks:** 5.
- **Order:** dependency first, then risk. Task 1 makes the `when` of an option a field that the read, `show` and
  `check` know, with no change to a selection. Task 2 is the riskiest change of the tool: the values of one feature
  start to depend on the values of another, and `Select` gets a new stop; the form follows it with no code. Task 3
  adds the second half of the `check` criterion on the rule that Task 2 settles. Task 4 adds `store-key` to `cli-auth`
  and proves criteria 1, 2 and 5 on the catalog of this repo. Task 5 ports the preset `api-key-stored` and proves
  criteria 3 and 4 with a written repo, its unit and e2e tests, breaks by hand and the running API.
- **Assessment:** Task 1 has a high blast radius because it adds a key to the JSON of `show`, and Task 5 because it
  handles the API key of the user. Tasks 1, 3 and 4 are high certainty: each repeats a change that US-009 or US-008
  made. Tasks 2 and 5 are medium, with the variation named. No task is low certainty: each part has a precedent in
  this repo. What stays open are decisions for the user (decisions 2, 5 and 10), not a design that a task must find.
- **Criteria of the story:**

| Criterion of US-013 | Tasks |
| --- | --- |
| `cli-auth` has the option `store-key`, with the values `true` and `false`; the default is `false` | 4 |
| The tool asks for `store-key` only with `api-key`; with `oauth`, the tree does not show it and it has its default; the tree shows or hides it after each change of `method` | 2 (the rule, the form on a catalog of the test), 4 (the catalog of this repo, the e2e tests of the form) |
| With `store-key=true`, `auth login` stores the key, later calls send it, `auth logout` removes it, and the variable still works | 5 |
| With `store-key=false`, the CLI has no code for the store | 5 (the files and words of `api-key` unchanged, the `go.mod` after `new`) |
| `--feature cli-auth:store-key=true` with `method=oauth` stops the tool before it writes a file, and the message says that `store-key` applies only to `api-key` | 2 (the stop and its message), 4 (the catalog of this repo, e2e) |
| `check` finds a `when` of an option that does not parse, or that reads an option that rule 5 does not allow | 1 (the parse), 3 (the reads) |
| Context: the preset `api-key-stored`, and the files of the store serve both methods | 5 |

- **Deferred:** none of the criteria of this story.
- **Criteria that this breakdown adds, for the caller to overrule:** the stop of a `when` on an option of a starter
  (decision 3); the key `when` in the JSON and the details of `show` (decision 7); a flag or a need with the default
  value is accepted (decision 5); the stop of a need with another value (decision 5); the item `golang.org/x/term`
  (decision 12); `Refused` names `auth login` (decision 13); the e2e full run of the stored key against the running
  API.
- **Notes:**
  - Files that Task 5 ports from the preset `api-key-stored`, for the brief of the audit (`tasks/learnings.md:31-36`):
    `credential.go` (`storedKey`), `credential_test.go` (the cases of the stored key), `.go-cli-auth/auth.go`
    (`auth login`, `readAPIKey`, `auth logout`) as `internal/cli/auth.go`, `.go-cli-auth/auth.test.ts` (the tests of
    `auth login` and `auth logout`), and the stored parts of `docs/authentication.md` and `.go-cli-auth/README.md.part`
    as `docs/cli-auth.md`. New code: the conditionals on `store-key`, `Refused` with `auth login`, the client test of
    the stored key, and the e2e full run.
  - The full run of the stored key adds about three minutes to the e2e tests, with `npm install` and the e2e tests of
    the written repo. `task test:combinations` gets one more repo.
