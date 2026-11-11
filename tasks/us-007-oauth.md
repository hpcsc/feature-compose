# Tasks: Protect the API with OAuth Access Tokens (US-007)

## Contents

1. Task 1: Read the values that a need sets from `feature.yml`, and give them in `show`
2. Task 2: Give an option the value that a need sets, and stop on a flag or a need with another value
3. Task 3: Write the files of a files entry only while its `when` holds
4. Task 4: Add the method `oauth` to `api-auth`
5. Task 5: Mark a value that a need sets as fixed in the form of `new`

## Story Reference

US-007, "Protect the API with OAuth access tokens", in `user-stories/feature-compose.md:265-297` (Description,
Acceptance Criteria, Context). The design is `PROPOSAL.md`: section 7 (a need that sets option values, and a files
entry with `when`, `PROPOSAL.md:152-252`), section 8 (the four sources of a value, the stop when the command line and
a need give different values, and the expression language of `when`, `PROPOSAL.md:254-293`), section 15 (the contract
of `api-auth`: 401 with `WWW-Authenticate: Bearer`, discovery, the allowlist, the signature, `iss`, `aud` and `exp`,
`PROPOSAL.md:493-517`) and open question 1 (which issuer, `PROPOSAL.md:563-565`). The breakdown of US-018 left three
items to this story (`tasks/completed/us-018-form.md:58-72`): the values of a need with their source `need`, the
mark `fixed` of such a value in the form, and the message of a need against a flag in the form.

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

- The scopes, roles and other claims of an access token. With `oauth`, each valid access token has each permission,
  so the API never answers 403, as with `api-key` (`catalog/features/api-auth/files/common/go/docs/api-auth.md:70-71`).
- An access token that is not a JWT (an opaque token, or token introspection), discovery other than OpenID Connect
  discovery at `<issuer>/.well-known/openid-configuration`, and an allowlist of algorithms in the config of the repo.
  The allowlist is in the code of the repo (decision 6).
- The settings of the issuer in `.env.local` and `docker-compose.local.yml` of the starter. `docs/api-auth.md` tells
  how to set them for `task api:run`, as it does for `API_KEY_HASHES`.
- A `when` that compares an option with a value that the option does not have, for example a typo in
  `method == "oauht"`. At run time such a `when` never holds; nothing refuses it.
- A change of a value that a need sets. The form starts no edit there (Task 5), and a flag with another value stops
  `new` (Task 2).
- `chi-api` as a target of `api-auth`. Rule 6 still holds for the files of `oauth`, so US-015 adds the target only.

**Deferred:**

- The `when` of an option, and a tree of the form that shows an option only while its `when` holds. US-013 adds them:
  its second criterion says that the tool asks for `store-key` only when `api-auth.method` is `api-key`, and that the
  tree of the form shows or hides `store-key` after each change of `method`. Task 3 builds the expression language
  that the `when` of an option also uses.
- The check in `check` of a need that sets a value that its option does not have: US-006, third criterion. At run
  time, `new` already stops on such a need (Task 2).
- The checks in `check` of a `when` that does not parse, and of a `when` that reads an option that rule 5 does not
  allow: US-009, first and second criteria. At run time, `new` already stops on both (Task 3).
- The CLI with `oauth`: US-008. Until then the need of `cli-auth` sets `api-key` (Task 2), so a repo with `cli-auth`
  and `oauth` stops before a file (Task 4).

## Codebase Context

**Needs today.** `FeatureDefinition.Needs` is a list of names (`internal/catalog/definitions.go:58-74`), read from the
key `needs` of `feature.yml` (`internal/catalog/feature_yml.go:11-21`), and `feature.needs` holds the same names
(`internal/catalog/feature.go:12-20`). A need selects a feature and sets no value. The walks of the needs
(`selectWithNeeds`, `inNeedsOrder`, `cycleFrom` and `neededBy`, `internal/catalog/starter.go:186-277`) go over that
list, and the message of a cycle follows its order. `Need` (`internal/catalog/need.go:5-16`) is a feature that only
needs add, and `selectedFeature.source` (`internal/catalog/selection.go:363-386`) gives `command line`, `answered`,
`needed by <features>` or `command line, needed by <features>`.

**Values today.** `options.withDefaults` (`internal/catalog/option.go:56-89`) checks the values of the flags and the
answers against the declared options, and gives each option the value of a flag, else of an answer, else its default.
`optionValues.source` (`internal/catalog/option_values.go:46-54`) gives `command line`, `answered` or `default`, and
the preview prints `<feature>.<option>: <value> (<source>)` (`internal/catalog/option_values.go:24-30`).
`Starter.Select` (`internal/catalog/starter.go:73-123`) calls `withDefaults` for the starter and for each selected
feature, with the values of that feature only. `Question` (`internal/catalog/question.go:8-19`) has `Given` for a value
from a flag, which `options.questions` fills (`internal/catalog/option.go:107-124`). One option with two values from
flags stops `new` with `the option method of the feature auth has two values: "oauth" and "api-key"`
(`internal/cmd/new.go:136-138`).

**Files today.** A files entry of a `feature.yml` has only `path` (`internal/catalog/feature_yml.go:13-15`).
`readFeatureFor` reads the templates under each entry (`internal/catalog/feature.go:40-51`), and `Selection.files`
renders each template and stops when two of them write one path (`internal/catalog/selection.go:200-222`). A template
of a feature reads the options of its feature, `<need>.<option>` and `starter.<option>`; another name stops the write,
and the message lists the names that it can read (`internal/catalog/selection.go:265-284`). The catalog has no `when`
and no expression language. `internal/glob/pattern.go` is the one small language of the repo: `Compile` refuses a bad
glob, and `Match` recurses over slices.

**The catalog of this repo.** `api-auth` (`catalog/features/api-auth/feature.yml`) has `method` with
`values: [api-key]`, the one files entry `files/common/`, and adds `auth.Middleware(cfg.Auth)` to `api.middleware` and
the field `Auth` of type `auth.Config` to `api.config`. An `add_to` does not change with an option, so each method
must give `auth.Config` and `auth.Middleware`. The Go code of `api-auth`
(`catalog/features/api-auth/files/common/go/internal/auth/`) holds `Config` with `KeyHashes` from `API_KEY_HASHES`,
which env v6 reads through `UnmarshalText`, the middleware, and their tests; `docs/api-auth.md` reads
`{{ option "method" }}`. `cli-auth` has `needs: [cli, api-auth]` and no option. Its docs read `api-auth.method`, and the
test of its API client calls `auth.Middleware` with the `auth.Config` of `api-key`
(`catalog/features/cli-auth/files/common/go/internal/apiclient/client_test.go.tmpl`). The modules of a feature reach
`go.mod` only through `go mod tidy`, an `after_render` command of the starter, as the modules of `cli` do
(`catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl`, `tasks/completed/us-002-add-cli.md:84`).

**Tests that this story changes.** `internal/catalog/catalog_test.go:548-555` uses `oauth` as a value that `method`
does not have; `internal/catalog/catalog_test.go:800-813` gives `cli-auth` a method other than `api-key` with a flag;
`internal/catalog/catalog_test.go:495-516` lists the files that `api-auth` adds. `e2e/tests/new.test.ts:102-110` uses
`api-auth:method=oauth` as the value that stops `new`, and `e2e/tests/new.test.ts:185-195` and
`e2e/tests/new.test.ts:337-362` expect `api-auth.method: api-key (default)` in a repo with `cli-auth`.
`e2e/tests/show.test.ts:16` expects the needs of `cli-auth` as a list.

**The JSON of `show`.** `Catalog.Definitions` (`internal/catalog/catalog.go:49-74`) gives the definitions with no null
(`internal/catalog/catalog_test.go:258-292`). `show --output json` prints them, its test holds each field
(`internal/cmd/show_test.go:85-120`), and `README.md:259-283` names each field and says that the JSON has the keys of
`starter.yml` and `feature.yml`. The details of a feature list its needs by name in the TUI of `show`
(`internal/browser/details.go:78-97`) and in the form of `new` (`internal/form/draft.go:361-368`).

**The form.** `draft.isGiven` and `draft.givenMessage` (`internal/form/draft.go:137-175`) make a value from an argument
or a flag fixed, and `draft.optionNodes` marks an option `fixed` from `Question.Given`
(`internal/form/draft.go:263-281`). Enter at such a row starts no edit, and the details say
`The option region is fixed: the command line gives it.` A Space whose selection `Select` refuses keeps the feature as
it was and shows the message of `Select` (`internal/form/draft.go:74-92`). `form.New` stops with the message of
`Select` for flags that `Select` refuses (`internal/form/model_test.go:1083-1104`).

**CI and the e2e tests.** `Starter.combinations` (`internal/catalog/starter.go:137-165`) gives one combination for each
value of an option with `values`, so `go-event-sourcing-service --feature api-auth:method=oauth` joins
`task test:combinations` with no change to CI (`internal/catalog/combination_test.go`, `.github/workflows/ci.yml`).
`startedApi` (`e2e/tests/new.test.ts:468-500`) starts the API of a written repo with no Postgres. `fakeReleases`
(`e2e/tests/update.test.ts:24-74`) serves a fake outside service from node on `127.0.0.1`, and `fakeGitHub.serve`
(`catalog/features/cli/files/go/internal/cli/release/updater_test.go.tmpl:35-84`) does the same with `httptest` in a
test of a template.

**Words.** `options` are the declared options, `values` the chosen values, `given` a value or a feature of a flag,
`needs` and `neededBy` the needs of a feature (`tasks/learnings.md:204-210`). `fixed` is the mark of the form for a
value that the developer cannot change. `source` is where a value comes from, with the texts of the preview. `key` is
the API key of `api-key` (`tasks/us-003-api-keys.design.yaml`), so the keys of the issuer are `signing keys`, as the
story says. No Go code of the tool uses `when`, `expression`, `issuer`, `audience` or `need of`.

**Decisions of this breakdown, for the caller to overrule:**

1. **The values of a need in `feature.yml` and in the JSON of `show`** (Task 1). `needs` takes a list of names, as
   today, or a map from each need to the values that it sets for the options of that need, as section 7 of the
   proposal writes it. Both forms give the same names in the same order. In the definitions and in the JSON of
   `show`, `features.<name>.needs` becomes an object from each need to its values, with `{}` for a need that sets no
   value. This changes the type of a field of the JSON from a list to an object. The other choice is to keep the list
   and add a key for the values; this breakdown does not take it, because `README.md:259` promises the keys of
   `feature.yml` and `feature.yml` has no such key.
2. **The order of the sources** (Task 2), from section 8: a flag, then a need, then an answer, then the default. A flag
   and a need with different values stop. Two needs with different values also stop, and the message names both
   needs. A value of a need that its option does not have, or a need value for an option that the feature does not
   declare, stops with the message of a flag, and the message names the feature whose need sets it. An answer for an
   option that a need sets does not stop: the selection has the value of the need, and the answer comes back when no
   selected need sets the option (the context of US-018: a need comes before the TUI).
3. **The source text** (Task 2): `need of <features>`, with the features whose needs set the value in the order of
   their names, and `command line, need of <features>` when a flag gives the same value, as a feature shows
   `command line, needed by <features>` (`tasks/learnings.md:197-203`). `Question` gives those features as `NeedOf`;
   `Given` stays true only for a value from a flag.
4. **`when` on a files entry, with the whole expression language of section 8** (Task 3): `==`, `!=`, a bare name that
   holds for the value `true`, `!`, `&&`, `||` and parentheses. US-007 itself needs only `==`, but US-008 and US-013
   write their `when` with `||` and a bare name (`PROPOSAL.md:206-211`), and no criterion of theirs names the
   language, so a part of it would leave them to grow it with no criterion (`tasks/learnings.md:237-243`). The parser
   is unexported in `internal/catalog`, and its tests go through `Write` and `Preview`. A name is a name that a
   template of the feature can read.
5. **The files of `api-auth` in three entries** (Task 4): `files/common/` with no `when`, `files/api-key/` with
   `when: method == "api-key"`, and `files/oauth/` with `when: method == "oauth"`. Each method gives `auth.Config`
   and `auth.Middleware`, so the `add_to` of the feature stays as it is.
6. **The check of an access token** (Task 4). The settings are `OAUTH_ISSUER_URL` and `OAUTH_AUDIENCE`, read by the
   config loader of the starter. The API reads `<issuer>/.well-known/openid-configuration`, refuses a discovery whose
   `issuer` is not the issuer URL of the config, and reads the signing keys from its `jwks_uri`. It reads both when it
   first checks a token, not at start, so the API starts while the issuer does not answer. It reads the signing keys
   again for a key ID that it does not have, but not for each request. The allowlist holds `RS256`, the default of each
   issuer that open question 1 names; an operator who sets another algorithm at the issuer changes the allowlist in
   the code. `iss` must be the issuer URL, `aud` must hold the audience (as a text or in a list), and `exp` must be
   after now. With no `OAUTH_ISSUER_URL` or no `OAUTH_AUDIENCE`, the API starts and answers 401 to each request to a
   protected route, as with no `API_KEY_HASHES`. An issuer URL that is not an absolute `https` URL, or an `http` URL
   of a loopback host, stops the API at start. A call to the issuer has a time limit. The task picks the JWT library;
   it reaches `go.mod` only through `go mod tidy` of an `oauth` repo.
7. **The need of `cli-auth` sets `method: api-key`** (Task 2), in the catalog of this repo. So a repo with `cli-auth`
   shows `api-auth.method: api-key (need of cli-auth)`, and with `oauth` in `values` (Task 4),
   `--feature cli-auth --feature api-auth:method=oauth` stops.
8. **The details of a feature list each need with the values that it sets** (Task 1), in the TUI of `show` and in the
   form of `new`, because the type of `FeatureDefinition.Needs` changes under both.

## Tasks

### Task 1: Read the values that a need sets from `feature.yml`, and give them in `show`

**Behavior:** The key `needs` of a `feature.yml` takes a list of names, as today, or a map from each need to the values
that it sets for the options of that need (decision 1). Both forms select the same needs. `Catalog.Definitions` gives
each need of a feature with its values, and `show --output json` prints `features.<name>.needs` as an object from each
need to its values. The details of a feature, in the TUI of `show` and in the form of `new`, list each need with the
values that it sets (decision 8). A selection does not use the values yet: Task 2 does.

**Acceptance Criteria:**
- [x] A `feature.yml` with `needs: [cli, api-auth]` and one with `needs` as a map from `cli` to no value and from
  `api-auth` to `method: api-key` select the same features, with the same files and the same items.
- [x] The needs of the list form keep the order of the file: the tests of the needs in `TestStarter` and
  `TestSelection`, the messages of a cycle and of a need that does not target the starter included, pass with their
  expected values unchanged.
- [x] `Catalog.Definitions` gives `Needs` of each feature as each need with the values that it sets: `{}` for a need
  that sets no value and for each need of the list form, and an empty object for a feature with no need. The JSON of
  the definitions holds no null (`internal/catalog/catalog_test.go:258-292` holds a need with a value and a need
  without).
- [x] A `needs` that is neither a list of names nor a map from names to option values, for example a text, or a need
  whose values are a list, stops `Catalog.Starter` and `Catalog.Definitions`, and the message names the `feature.yml`.
- [x] `show --output json` prints `features.<name>.needs` as an object from each need to its values. The test of each
  field of the JSON (`internal/cmd/show_test.go:85-120`) holds a need with a value and a need without, and
  `README.md` names `features.<name>.needs.<name>` with its meaning in the table of the fields.
- [x] The details of a feature in the TUI of `show` list each need by its name, in the order of the names, and under a
  need each option that it sets with its value. A need with no value shows only its name. The details of a feature in
  the form of `new` list the needs in the same way.
- [x] `e2e/tests/show.test.ts` expects the needs of `cli-auth` as an object of `cli` and `api-auth`, each with no
  value, and passes on the host.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/definitions.go` — `FeatureDefinition.Needs` holds each need with its values
- `internal/catalog/feature_yml.go` — the two forms of `needs`
- `internal/catalog/feature.go` — the names of the needs from either form
- `internal/catalog/catalog_test.go`, `internal/catalog/starter_test.go` — the two forms, a `needs` of another form,
  and the JSON of the definitions
- `internal/browser/details.go`, `internal/browser/model_test.go` — the needs in the details of a feature
- `internal/form/draft.go`, `internal/form/model_test.go` — the needs in the details of a feature
- `internal/cmd/show_test.go` — the JSON of the fields
- `README.md` — the table of the fields of the JSON
- `e2e/tests/show.test.ts` — the needs of `cli-auth`

**Patterns to Follow:**
- `internal/catalog/definitions.go:58-74` for a field of the definitions and its `filled` value with no null
- `internal/catalog/feature_yml.go:11-38` for the read of a `feature.yml` and its message
- `internal/catalog/catalog_test.go:156-206` for a `feature.yml` that stops the read, and a message that names it
- `internal/catalog/catalog_test.go:258-292` for the JSON of the definitions with no null
- `internal/cmd/show_test.go:85-120` and `README.md:259-283` for the fields of the JSON and their docs
- `internal/browser/details.go:78-97` and `internal/browser/model_test.go:325-421` for the needs in the details of a
  feature and their tests
- `internal/form/draft.go:361-368` for the details of a feature in the form
- `PROPOSAL.md:194-252` for the two forms of `needs`
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`, not only in `internal/cmd`
- `tasks/learnings.md:204-210` for the words of needs and values

**Testable:** Yes

**Certainty:** medium — `FeatureDefinition.filled` (`internal/catalog/definitions.go:58-74`) and the test of each field
of the JSON (`internal/cmd/show_test.go:85-120`) give the shape and its test; but no key of a `feature.yml` in this repo
takes two YAML forms

**Blast radius:** high — a contract outside this repository: the type of `features.<name>.needs` in the JSON of `show`
changes from a list to an object, and a script reads that JSON

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog`, `internal/browser` and `internal/form`. Run `e2e/tests/show.test.ts` on the
host (`tasks/learnings.md:58-64`). `feature-compose show --output json | jq '.features["cli-auth"].needs'` prints an
object.

**Depends on:** None

---

### Task 2: Give an option the value that a need sets, and stop on a flag or a need with another value

**Behavior:** When the need of a selected feature sets a value for an option of the feature that it needs, the
selection gives that option that value: after a flag, and before an answer and the default (decision 2). The preview
of `--dry-run` and `Selection.Questions` give the source `need of <features>`, or `command line, need of <features>`
when a flag gives the same value (decision 3). A flag with another value stops `Select`, so `new`, `--dry-run` and the
start of the form stop before the first file, and the message names both sources. Two needs with different values
stop, and the message names both needs. In the catalog of this repo, the need of `cli-auth` sets `method: api-key`
(decision 7).

**Acceptance Criteria:**
- [x] With a catalog where `cli-auth` needs `auth` and sets `method: oauth`, and the default of `method` is `api-key`,
  `--feature cli-auth` writes the files of `auth` with `oauth`, and the preview shows
  `auth.method: oauth (need of cli-auth)`. A need that sets the value of the default also shows
  `need of <feature>`.
- [x] When the needs of two selected features set one option to the same value, the source names both features in the
  order of their names, for example `need of a, b`.
- [x] A flag that gives an option the value that a need sets does not stop, and the source is
  `command line, need of cli-auth`.
- [x] A flag that gives an option another value than a need stops `Select`, and the message names the option, its
  feature, the value of the flag with the command line, and the value of the need with the need of its feature, for
  example `oauth` from the command line and `api-key` from the need of `cli-auth`. `new` and `new --dry-run` stop with
  that message, make no output folder, run no command and print no preview, in each order of the flags.
- [x] The needs of two selected features that set one option to different values stop `Select`, and the message names
  the option, its feature, and each value with the need that sets it.
- [x] A need value that its option does not have, a need value for an option that the feature does not declare, and a
  need value for a feature with no option stop `Select` with the message of the same flag, and the message also names
  the feature whose need sets the value.
- [x] An answer for an option that a need sets does not stop `Select`: the option has the value of the need, with the
  source `need of <features>`. With no selected feature whose need sets it, the option has the answer again.
- [x] A template of the needed feature reads the value of the need with `option`, and a template of the feature that
  needs it reads it as `<need>.<option>`.
- [x] `Selection.Questions` gives such an option with the value of the need, the source of the preview, and `NeedOf`
  with the features whose needs set it, in the order of their names. `Given` is true only for a value from a flag, and
  `NeedOf` is empty for a value that no need sets. The groups `preview` and `questions` of `TestSelection` and the
  group `select` of `TestStarter` hold these cases.
- [x] `catalog/features/cli-auth/feature.yml` writes its need of `api-auth` with `method: api-key`, and its need of
  `cli` with no value. `new go-event-sourcing-service my-service --feature cli-auth --dry-run` shows
  `api-auth.method: api-key (need of cli-auth)`. The test of the docs of `cli-auth`
  (`internal/catalog/catalog_test.go:800-813`) sets its other method through the need of `cli-auth` in its copy of the
  catalog, since a flag with another value now stops.
- [x] `README.md`, in "Preview a Repo": the example shows `api-auth.method: api-key (need of cli-auth)`; the table of the
  sources of an option has `need of <features>` and says when the line starts with `command line, `; and the paragraph
  of the stops of `--dry-run` names a flag that gives an option another value than a need.
- [x] `e2e/tests/new.test.ts:185-195` and `e2e/tests/new.test.ts:337-362` expect
  `api-auth.method: api-key (need of cli-auth)`, and the e2e tests of `new` pass on the host.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/starter.go` — `Select` collects the values of the needs of the selected features, and stops on a
  conflict
- `internal/catalog/feature.go` — the values that the needs of a feature set
- `internal/catalog/option.go`, `internal/catalog/option_values.go` — the value of a need between a flag and an answer,
  and its source
- `internal/catalog/question.go` — `NeedOf`
- `internal/catalog/starter_test.go`, `internal/catalog/selection_test.go`, `internal/catalog/catalog_test.go` — the
  cases above, and the test of the docs of `cli-auth`
- `internal/cmd/new_test.go` — the stop of `new` and `--dry-run`, and the source in the preview
- `catalog/features/cli-auth/feature.yml` — the need of `api-auth` with `method: api-key`
- `README.md` — "Preview a Repo"
- `e2e/tests/new.test.ts` — the source of `api-auth.method`

**Patterns to Follow:**
- `task:1` for the values of a need in the definitions of a feature
- `internal/catalog/option.go:56-105` for the order of a flag before an answer before the default, and the check of a
  value against its option
- `internal/catalog/option_values.go:24-54` for the source of a value and its line in the preview
- `internal/catalog/selection.go:363-386` for a source with `command line, ` and a relation to other features
- `internal/cmd/new.go:136-138` for the message of one option with two values
- `internal/catalog/starter_test.go:678-704` for the tests of a need with `method`
- `internal/catalog/selection_test.go:1045-1063` and `internal/catalog/selection_test.go:1130-1140` for the sources in
  the preview, and a flag and an answer for one option
- `internal/catalog/selection_test.go:1199-1275` for the tests of the questions
- `internal/cmd/new_test.go:497-504` and `internal/cmd/new_test.go:815-832` for the sources of `--dry-run`, and a stop
  that names both values and makes no output folder
- `internal/catalog/catalog_test.go:800-813` for the test of the docs of `cli-auth`
- `README.md:43-100` for the preview and its tables of sources
- `tasks/learnings.md:197-203` for the relation of every item, also with a second source
- `tasks/learnings.md:204-210` for the words of options, values, given and needs
- `tasks/learnings.md:134-140` for a loop over a map that stops at the first match
- `tasks/learnings.md:183-189` for tests in `internal/catalog`
- `tasks/learnings.md:58-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `options.withDefaults` (`internal/catalog/option.go:56-89`) already ranks a flag before an
answer before the default, and `optionValues.source` (`internal/catalog/option_values.go:46-54`) names the source; but
the value of a need comes from another selected feature, so the values and their conflicts are collected across the
features of the selection before an option gets its value

**Blast radius:** low — none of the named categories; it decides the option values of a written repo, and each
conflict stops before the first file

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` finds no live mutant on
the changed lines of `internal/catalog`. Run `e2e/tests/new.test.ts` on the host. Run the built binary with
`new go-event-sourcing-service my-service --feature cli-auth --dry-run` and read the line of `api-auth.method`.

**Depends on:** Task 1

---

### Task 3: Write the files of a files entry only while its `when` holds

**Behavior:** A files entry of a `feature.yml` can have `when`, with the expression language of section 8 of the
proposal (decision 4). `Selection.Write` and `Selection.Preview` render the templates of an entry only while its `when`
holds for the values of the options that the feature can read; an entry without `when` is always written. A `when`
that does not parse stops the read of the feature. A name in a `when` that the feature cannot read stops the write and
the preview before the first file.

**Acceptance Criteria:**
- [x] `<name> == "<value>"` holds when the option has that value, and `<name> != "<value>"` when it has another. A bare
  name holds when the option has the value `true`, and not for another value. `!`, `&&`, `||` and parentheses combine
  expressions, with `!` before `&&` and `&&` before `||`: `a || b && c` holds as `a || (b && c)`. A name can hold `-`
  and `.`, so `store-key` is one name. Spaces between the parts do not change the result.
- [x] A name is an option of the feature, `<need>.<option>` of an option of one of its needs, or `starter.<option>`,
  with the value of the selection, also a value from a flag, an answer or a need.
- [x] The preview lists no file of an entry whose `when` does not hold, and `Write` writes none of its files. The other
  entries of the feature, and the starter, write their files as before.
- [x] Two entries of one feature whose `when` never hold together can each write one path. When both hold, the write
  stops with the message of two templates that write one path.
- [x] A `when` that does not parse stops `Catalog.Starter`, and the message names the `feature.yml`, the files entry
  and the text of the `when`: for example `method ==`, `method = "oauth"`, `(method == "oauth"`, `a &&`, a lone
  `"oauth"`, and a text with no closing quote.
- [x] A `when` with a name that the feature cannot read stops `Write` and `Preview` before the first file, and the
  message names the `when`, the feature, and the names that it can read.
- [x] The tests of `TestCatalog`, `TestStarter` and `TestSelection` pass with their expected values unchanged: no
  catalog file has a `when` yet.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/feature_yml.go` — `when` of a files entry
- `internal/catalog/feature.go` — each template of an entry carries its `when`
- `internal/catalog/template.go` — `when` of a template
- `internal/catalog/expression.go` — new: the parsed `when`, and whether it holds
- `internal/catalog/selection.go` — `files` renders a template only while its `when` holds
- `internal/catalog/selection_test.go`, `internal/catalog/catalog_test.go` — the cases above, through `Write`,
  `Preview` and `Catalog.Starter`

**Patterns to Follow:**
- `internal/glob/pattern.go:13-47` for a small language that is parsed once and matched with recursion over slices
- `internal/catalog/delimiters_rule.go:9-27` for a rule of a `feature.yml` that is parsed at the read and applied to
  its templates
- `internal/catalog/feature.go:40-51` for the files entries of a feature
- `internal/catalog/selection.go:200-222` for the files of a selection and the stop on two writers of one path
- `internal/catalog/selection.go:265-284` for the names that a template of a feature can read, and the message of
  another name
- `internal/catalog/catalog_test.go:156-206` for a `feature.yml` that stops the read, and a message that names it
- `internal/catalog/starter_test.go:585-614` for a template that reads an option of its need, and the message of a
  name that it cannot read
- `PROPOSAL.md:275-293` for the expression language
- `tasks/learnings.md:148-154` for a loop whose mutants never end and stop the mutation run
- `tasks/learnings.md:134-140` for a loop over a map in sorted order
- `tasks/learnings.md:183-189` for tests in `internal/catalog`

**Testable:** Yes

**Certainty:** medium — `internal/glob/pattern.go:13-47` is a small language of this repo, parsed once and matched
with recursion over slices; but the expression language adds operators with an order, parentheses, and names that it
reads against the option values of a selection

**Blast radius:** low — none of the named categories; no catalog file has a `when` until Task 4, and a wrong result
there shows as a missing `auth.Middleware` or as two writers of one path, which stop the build or the write

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. `clerk mutants` completes within its
limit and finds no live mutant on the changed lines of `internal/catalog`.

**Depends on:** None

---

### Task 4: Add the method `oauth` to `api-auth`

**Behavior:** The option `method` of `api-auth` has the values `api-key` and `oauth`, with the default `api-key`. The
files of `api-auth` are in three entries (decision 5). With `oauth`, the API of a written repo reads the URL of the
issuer and the audience from its config, gets the signing keys of the issuer through discovery, and lets a request
reach a protected route only with a JWT access token whose algorithm is in the allowlist, whose signature is valid,
and whose `iss`, `aud` and `exp` are correct (decision 6). Each other request gets 401 with
`WWW-Authenticate: Bearer`. With `api-key`, the repo is as before, with no OAuth code and no JWT library. In the
catalog of this repo, `--feature cli-auth --feature api-auth:method=oauth` stops before a file. A repo with each value
of `method` builds and passes its tests, in `task test:combinations` and in an e2e test that starts the API with a test
issuer.

**Acceptance Criteria:**
- [x] `catalog/features/api-auth/feature.yml` has `method` with `values: [api-key, oauth]` and `default: api-key`, the
  files entries `files/common/` with no `when`, `files/api-key/` with `when: method == "api-key"` and `files/oauth/`
  with `when: method == "oauth"`, and the `add_to` that it has today.
- [x] With `oauth`, the unit tests of `internal/auth` of the written repo run in `task test:unit`, against a test issuer
  that serves discovery and the signing keys and signs tokens with a key of the test, and show:
  - a request with `Authorization: Bearer <token>`, for a token signed with `RS256` by a signing key that the issuer
    publishes, with `iss` the issuer URL, `aud` that holds the audience as a text or in a list, and `exp` after now,
    gets to the handler, and the answer is the answer of the handler; the scheme matches in any case;
  - each other request gets 401 with `WWW-Authenticate: Bearer`, and the handler does not run: no `Authorization`
    header, a scheme other than `Bearer`, `Bearer` with no token, a text that is not a JWT, the algorithm `none`,
    `HS256` with the public signing key as its secret, an algorithm that the allowlist does not hold with a signing key
    that the issuer publishes, a signature by a key that the issuer does not publish, a changed payload, another
    `iss`, an `aud` without the audience, an `exp` before now, and no `exp`;
  - the API reads discovery at `<issuer>/.well-known/openid-configuration` and the signing keys at its `jwks_uri`; with
    a discovery whose `issuer` is not the issuer URL of the config, each token gets 401;
  - many requests read discovery and the signing keys one time. A token signed with a key that the issuer publishes
    after that first read gets to the handler. Requests with key IDs that the issuer does not publish do not read the
    signing keys for each request;
  - while the issuer does not answer, or answers with an error, a request with a token gets 401, and after the issuer
    answers again, a request with a valid token gets to the handler;
  - no request gets 403.
- [x] The config of the API reads the issuer URL from `OAUTH_ISSUER_URL` and the audience from `OAUTH_AUDIENCE`. With
  no `OAUTH_ISSUER_URL` or no `OAUTH_AUDIENCE`, the API starts and answers 401 to each request to a protected route. An
  issuer URL that is not an absolute `https` URL, or an `http` URL of a loopback host, stops the config loader at
  start, and the message names `OAUTH_ISSUER_URL`. A call of the API to the issuer has a time limit.
- [x] No written file holds an issuer URL or an audience that the API uses: a new issuer or a new audience needs new
  values of the two settings and a restart of the API, and no new render.
- [x] With `api-key`, a request with a known key gets to the handler and each other request gets 401 with
  `WWW-Authenticate: Bearer`, as the tests of `internal/auth` of an `api-key` repo show. The repo holds no file of
  `files/oauth/`: no written file names an issuer, an audience, discovery or a JWT, and no Go file imports the JWT
  module that the files of `oauth` import.
- [x] With each method, no Go file of `api-auth` imports a package of the module other than the packages under
  `internal/auth` (rule 6).
- [x] With `oauth`, `docs/api-auth.md` tells: that the issuer must sign JWT access tokens and publish discovery, and
  that Google gives no JWT access token for your own API; the two settings, and how to change them with no new render,
  for example `OAUTH_ISSUER_URL=<url> OAUTH_AUDIENCE=<audience> task api:run`; the allowlist and the checks; how a
  client sends the token; and the answers of the API. It names only files that the repo has. With `api-key`, the docs
  pass the checks of `internal/catalog/catalog_test.go:576-598`.
- [x] The test of a value that `method` does not have (`internal/catalog/catalog_test.go:548-555`) uses a value other
  than `oauth`, and its message lists `api-key, oauth`.
- [x] In the catalog of this repo, `--feature cli-auth --feature api-auth:method=oauth` stops `Select`, and the message
  names `oauth` with the command line and `api-key` with the need of `cli-auth`.
- [x] `task test:combinations` passes, and its combinations hold
  `go-event-sourcing-service --feature api-auth:method=oauth`.
- [x] `e2e/tests/new.test.ts`:
  - `new go-event-sourcing-service my-service --feature api-auth:method=oauth --no-prompt` makes a repo in which
    `gofmt -l .` lists no file, `go build ./...` builds, and `task test:unit` passes with `internal/auth` among the
    packages that passed. The test starts a test issuer on `127.0.0.1` that serves discovery and the signing keys and
    signs tokens with a key of the test, and starts the API with `OAUTH_ISSUER_URL` and `OAUTH_AUDIENCE`. `GET /` gets
    200 with no token. `POST /accounts` gets 401 with `WWW-Authenticate: Bearer` with no token, with a token for
    another audience and with a token whose `exp` is before now, and the answer 400 of the handler with a valid token.
    The test then starts the API again with another `OAUTH_AUDIENCE` and no new render: the first token gets 401, and
    a token for the new audience gets 400.
  - The test of the `api-key` repo (`e2e/tests/new.test.ts:69-100`) also checks that `go.mod` of the repo requires the
    same modules as `go.mod` of a repo without `api-auth`.
  - The test of a value that `method` does not have (`e2e/tests/new.test.ts:102-110`) uses a value other than `oauth`,
    and expects `api-key, oauth`.
  - `new go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=oauth`, with `--no-prompt`
    and with `--dry-run`, ends with a status other than 0, a message that names `oauth` with the command line and
    `api-key` with the need of `cli-auth`, and a scratch folder with no entry.
  - `docs/e2e-tests.md` tells the test of the `oauth` repo and its test issuer.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/features/api-auth/feature.yml` — the value `oauth`, and the three files entries
- `catalog/features/api-auth/files/common/` — the files that both methods use
- `catalog/features/api-auth/files/api-key/` — the files of `api-key`, from `files/common/`
- `catalog/features/api-auth/files/oauth/go/internal/auth/` — new: the config, the middleware, discovery, the signing
  keys, and their tests; each `.go` file has the `.tmpl` suffix
- `catalog/features/api-auth/files/oauth/go/docs/api-auth.md` — new, or `docs/api-auth.md` of `files/common/` for both
  methods
- `internal/catalog/catalog_test.go` — the files of each method, the imports, the docs, the value that `method` does not
  have, and the stop of `cli-auth` with `oauth`
- `e2e/tests/new.test.ts` — the `oauth` repo with a test issuer, `go.mod` of the `api-key` repo, and the stop of
  `cli-auth` with `oauth`
- `docs/e2e-tests.md` — the test of the `oauth` repo

**Patterns to Follow:**
- `task:2` for the need of `cli-auth` and the stop of a flag against a need
- `task:3` for the `when` of a files entry
- `catalog/features/api-auth/files/api-key/go/internal/auth/middleware.go.tmpl:1-18` for the middleware and its 401
- `catalog/features/api-auth/files/api-key/go/internal/auth/middleware_test.go.tmpl:16-104` for the tests of a request
  that is refused or reaches the handler
- `catalog/features/api-auth/files/api-key/go/internal/auth/key_hashes.go.tmpl:13-24` and
  `catalog/features/api-auth/files/api-key/go/internal/auth/key_hashes_test.go.tmpl:22-72` for a setting that the config
  loader reads through `UnmarshalText`, and its tests
- `catalog/features/cli/files/go/internal/cli/release/updater_test.go.tmpl:35-84` for a fake outside service on
  `httptest` in a test of a template
- `internal/catalog/catalog_test.go:484-598` for the files, the imports and the docs of `api-auth`
- `e2e/tests/new.test.ts:69-100` and `e2e/tests/new.test.ts:468-500` for a repo whose API an e2e test starts
- `e2e/tests/update.test.ts:24-74` for a fake outside service from node on `127.0.0.1`
- `docs/e2e-tests.md:80-125` for the docs of the e2e tests that start an API
- `PROPOSAL.md:493-517` for the contract of `api-auth`
- `tasks/learnings.md:10-15` for `.tmpl`
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof of a template by breaks by hand on a
  written repo
- `tasks/learnings.md:92-98` for the cost of the proof of new Go code of a feature
- `tasks/learnings.md:99-105` for a setting of `api.Config` that `UnmarshalText` checks
- `tasks/learnings.md:106-112` for an e2e test that starts the API without Postgres
- `tasks/learnings.md:113-119` for the docs of a setting of the local API
- `tasks/learnings.md:120-126` for a test of an absent variable
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:37-43`, `tasks/learnings.md:51-57` and `tasks/learnings.md:58-64` for an e2e test that runs `go`
  and `task` in a written repo, on the host

**Testable:** Yes

**Certainty:** low — the first JWT library and the first call of a written API to an outside service in this repo:
discovery, the signing keys and the allowlist have no precedent, and `api-key` (the standard library only) gives only
the middleware, its config and its tests

**Blast radius:** high — authentication: this is the check of each request to a protected route of an `oauth` repo, and
it handles the access tokens of the requests

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo with each method with the
built binary into scratch folders, and run `go mod tidy`, `gofmt -l .`, `go build ./...` and `task test:unit` there.
In the `oauth` repo, apply one break at a time to the written files, read its diff, run `go test -tags unit
./internal/auth/`, and copy the file back (`tasks/learnings.md:85-91`, `tasks/learnings.md:127-133`): accept each
algorithm; accept `none`; skip the signature; skip `iss`; skip `aud`; skip `exp`; drop `WWW-Authenticate`; read the
signing keys from a fixed path instead of `jwks_uri`; read the signing keys again for each request; accept a discovery
with another `issuer`. Each break makes a test fail. Run `task test:combinations` and `e2e/tests/new.test.ts` on the
host.

**Depends on:** Task 2, Task 3

---

### Task 5: Mark a value that a need sets as fixed in the form of `new`

**Behavior:** In the tree of the form, an option whose value a need sets is marked `fixed`, and its details give the
source `need of <features>`. Enter at that option starts no edit, and the details say which need sets the value. A
Space at a feature whose need sets another value than a flag leaves the feature unselected, and the details show the
message of `Select`, which names both sources. Flags that conflict with a need stop `new` before the form opens.

**Acceptance Criteria:**
- [x] An option whose value the need of a selected feature sets shows `fixed` in the tree, for example
  `method  api-key  fixed` under `api-auth` after a Space at `cli-auth`. When no selected feature's need sets the
  option, the mark goes, unless a flag gives the value.
- [x] The details of that option show its value, its default, its values or its pattern, and the source
  `need of <features>`, or `command line, need of <features>` when a flag gives the same value.
- [x] Enter at that option starts no edit, and the first line of the details says that the option is fixed and names
  the features whose need sets it, and also the command line when a flag gives the value.
- [x] A value that the developer kept for an option gives way to the value of a need while the need of a selected
  feature sets it, and the option shows that value again when no selected feature's need sets it.
- [x] With `--feature auth:method=oauth` and a feature whose need sets `method: api-key`, a Space at that feature leaves
  its row `[ ]`, and the details show the message of `Select`, which names `oauth` with the command line and `api-key`
  with the need of that feature.
- [x] `form.New` with the two features in `--feature`, whose need and flag give different values, stops with that
  message (the group `start` of `TestModel`). In a terminal, `new` with those flags stops before the form opens and
  writes nothing (the group `tui` of `TestNewCommand`).
- [x] `README.md`, in "Answer the Questions in a TUI": the mark `fixed` names an argument, a flag or a need; the example
  of the screen shows `method  api-key  fixed`; Enter at a value that an argument, a flag or a need gives starts no
  edit; and a Space at a feature whose need gives another value than a flag leaves the feature unselected and shows the
  message.
- [x] `e2e/tests/new.test.ts`, describe "feature-compose new in a terminal", on the catalog of this repo:
  `new go-event-sourcing-service my-service --feature cli-auth` shows `method  api-key  fixed`, and Enter at that row
  shows a message that names `cli-auth`, with the source `need of cli-auth`;
  `new go-event-sourcing-service my-service --feature api-auth:method=oauth` and a Space at `cli-auth` leave
  `[ ] cli-auth` and show a message that names `oauth` and the need of `cli-auth`; `q` ends with a status other than
  0 and a scratch folder with no entry.
- [x] The tests of `internal/form` drive these keys through `Update` and `View`, with catalogs in `fstest.MapFS`.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, also with `CLICOLOR_FORCE=1` and with `NO_COLOR=1`,
  and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `internal/form/draft.go` — a value of a need is fixed: its mark, its refused edit and its message
- `internal/form/model_test.go` — the groups `options`, `edit`, `features` and `start` of `TestModel`
- `internal/cmd/new_test.go` — the stop before the form opens
- `README.md` — "Answer the Questions in a TUI"
- `e2e/tests/new.test.ts` — the describe "feature-compose new in a terminal"

**Patterns to Follow:**
- `task:2` for `Question.NeedOf` and the source of a need
- `task:4` for the value `oauth` of the catalog of this repo
- `internal/form/draft.go:137-175` for a fixed value and the message of a refused edit
- `internal/form/draft.go:263-281` and `internal/form/draft.go:370-381` for the marks of an option and its details
- `internal/form/draft.go:74-92` for a Space whose selection `Select` refuses
- `internal/form/model_test.go:409-416` and `internal/form/model_test.go:529-535` for the tests of a fixed option and
  its refused edit
- `internal/form/model_test.go:318-337` for a Space that `Select` refuses
- `internal/form/model_test.go:1083-1104` for the stops of `form.New`
- `internal/cmd/new_test.go:706-730` for a stop of `new` before the form opens
- `e2e/tests/new.test.ts:276-362` for the screen tests of the form
- `README.md:104-200` for the docs of the form
- `tasks/learnings.md:176-182` for a test of a bubbletea model
- `tasks/learnings.md:217-222` for a test of `new` in a pseudo-terminal
- `tasks/learnings.md:223-229` for the late start of a command in tuistory
- `tasks/learnings.md:265-270` for the lines of `internal/form` that the tests of `internal/cmd` do not run

**Testable:** Yes

**Certainty:** high — the mark and the refused edit of a value from a flag (`draft.isGiven` and `draft.givenMessage`,
`internal/form/draft.go:137-175`, and the marks of `draft.optionNodes`, `internal/form/draft.go:263-281`) are the
pattern; the value of a need is a second source of the same mark, and a Space that `Select` refuses already shows its
message (`internal/form/draft.go:74-92`)

**Blast radius:** low — none of the named categories; the form writes no file, and `Select` holds the rule of the
sources

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, also with `CLICOLOR_FORCE=1` and with
`NO_COLOR=1`. `clerk mutants` finds no live mutant on the changed lines of `internal/form`. Run `e2e/tests/new.test.ts`
on the host. In a terminal, run the built binary with `new go-event-sourcing-service my-service --feature cli-auth`,
read the row of `method`, press Enter at it, and press `q`.

**Depends on:** Task 2, Task 4

## Summary

- **Tasks:** 5.
- **Order:** dependency first. Task 1 reads the values of a need and gives them in `show`; Task 2 gives them to the
  options of a selection, with the source `need of <features>` and the stops of section 8, and sets the need of
  `cli-auth`. Task 3 adds `when` to a files entry, with no dependency on Task 1 or Task 2. Task 4 is the riskiest task
  and comes as soon as its two capabilities exist: the method `oauth` of `api-auth`, its files behind `when`, its
  proof on written repos, and the stop of `cli-auth` with `oauth` on the catalog of this repo. Task 5 marks a value of
  a need as fixed in the form; it comes last, because its screen tests use `oauth` of the catalog of this repo.
- **Assessment:** Task 4 is low certainty, because it adds the first JWT library and the first call of a written API to
  an outside service, with no precedent for discovery, the signing keys or the allowlist. Task 1 and Task 4 have a high
  blast radius: Task 1 changes the type of a field of the JSON of `show`, which scripts read, and Task 4 is the check of
  each request to a protected route of an `oauth` repo. Task 5 is high certainty: it repeats the mark of a value from a
  flag.
- **Criteria of the story:**
  - "The option `method` of `api-auth` has the values `api-key` and `oauth`. The default stays `api-key`": Task 4.
  - "With `oauth`, the API gets the signing keys of the issuer through discovery. It accepts a JWT access token only
    when the algorithm is in the allowlist, the signature is valid, and `iss`, `aud`, and `exp` are correct. Each other
    request gets 401 with `WWW-Authenticate: Bearer`": Task 4, in the unit tests of the written repo and in the e2e
    test with a test issuer.
  - "The URL of the issuer and the audience of the API come from the config of the repo, not from options. An operator
    can change them without a new render": Task 4, with the e2e test that starts the API again with another audience.
  - "A repo with `method=api-key` has no OAuth code and no JWT library in its module": Task 3 (`when`) and Task 4 (the
    files of each method, and `go.mod` of the `api-key` repo in the e2e test).
  - "`--feature cli-auth --feature api-auth:method=oauth` stops before the tool writes a file. The message names both
    sources of the value": Task 2 (the stop and its message, and the need of `cli-auth`), Task 4 (on the catalog of this
    repo, with `--no-prompt` and `--dry-run`), Task 5 (before the form opens).
  - "`new --dry-run` shows the source `need` for each option value that a need sets, with the feature whose need sets
    it": Task 2.
  - "In the form of `new` (US-018), the tree marks an option value that a need sets as `fixed`, and the details name the
    need as its source. Enter at that option starts no edit, and the details say which need sets the value": Task 5.
  - "A repo with each value of `method` builds and passes its tests": Task 4 (`task test:combinations` and the e2e
    tests).
  - The items that US-018 left to this story: the values of a need and their source (Task 2), the mark `fixed` of such
    a value (Task 5), and the message of a need against a flag in the form (Task 5).
- **Deferred:** none of the criteria of this story. The `when` of an option goes to US-013, the checks of `check` to
  US-006 and US-009, and the CLI with `oauth` to US-008, each named in the criteria of its story (see Boundaries).
