# Tasks: The CLI Sends the API Key (US-005)

## Contents

1. Task 1: Select the needs of each selected feature, in the order of rule 8
2. Task 2: Name each feature that a need added in the output of `new`
3. Task 3: Read the options of a need in the templates of a feature
4. Task 4: Add the feature `cli-auth`, whose CLI takes the API key from a variable or a credential command and shows its source with `auth status`
5. Task 5: Send the API key on each call of the CLI to the API, and tell the user how to give the key when the API refuses it

## Story Reference

US-005, "The CLI sends the API key", in `user-stories/feature-compose.md:204-231` (Acceptance Criteria and Context).
The design is `PROPOSAL.md`: section 7 (the `feature.yml` of `cli-auth` with `needs: [cli, api-auth]`,
`PROPOSAL.md:194-212`, and the field `needs`, `PROPOSAL.md:243`), section 9 (`option "api-auth.method"`,
`PROPOSAL.md:299-307`), section 10 (rule 4, `PROPOSAL.md:337`; rule 5, `PROPOSAL.md:338-339`; rule 8,
`PROPOSAL.md:343-345`), section 11 (step 1 of `new`, `PROPOSAL.md:378`), section 12 (`api-auth: needed by cli-auth`,
`PROPOSAL.md:409-413`), section 14 (the cycle check of `check`, `PROPOSAL.md:479`) and section 15 (the contract of
`api-auth` and `cli-auth`, `PROPOSAL.md:488-512`). The template to port is the preset `api-key` of
`global/go-cli-auth` in `~/repo-templates/templates/global/go-cli-auth`.

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

- A command of the CLI that calls a protected route of the API. The protected routes are the account routes of the
  starter, which `cli-auth` cannot know (rule 6, and the same files go to `chi-api` in US-015). `cli-auth` gives the
  API client, and the developer writes the commands that use it (decision 1).
- An end-to-end test that starts the API of the written repo and calls it through the built CLI. No command of the
  CLI calls a protected route, so the call is checked by the unit tests of the API client against the middleware
  of `api-auth` in the written repo (decision 1).
- A setting of the CLI for the URL of the API. The API client takes the URL from the code that makes it
  (decision 9).
- The check of `check` for needs: a need that does not target each target of the feature, a cycle anywhere in the
  catalog, and a need that sets a value that its option does not have (US-006). `new` stops only for the needs of
  the features that it selects (decision 3).
- An option of a need of a need in a template. A template reads the options of the needs that its own
  `feature.yml` names (decision 5).
- The second sentence of rule 5: a feature reads a starter option only when each of its targets declares it. It is
  a check of the catalog (US-009).
- `when`. The files of `cli-auth` are in `files/common/`, with no `when`, as the files of `api-auth` are.
- The store of the key, `auth login`, `auth logout` and the option `store-key` (US-013); OAuth in the CLI and
  `auth token` (US-008).
- The section "Authentication" that `go-cli-auth` adds to `README.md`. `README.md` belongs to the starter (rule 7),
  so the docs of `cli-auth` are `docs/cli-auth.md` (decision 8).
- The dry run (US-010), the selection file and its `requested` (US-011), and the step of the TUI that shows the
  needs (US-012). They read the list that Task 2 adds, and they are not built here.
- `chi-api` as a target of `cli-auth` (US-015).

**Deferred:**

- Needs that set option values, in the map form `needs: {api-auth: {method: oauth}}`, and the stop when the command
  line and a need give different values, which names both sources: US-007 has this criterion, and `cli-auth` then
  sets `method: api-key` in its needs. This story reads only the list form `needs: [cli, api-auth]`; the map form
  stops the read of the catalog with the name of the file.
- The setting of the URL of the API (`apiURL` and `<NAME>_API_URL` of `go-cli-auth`): US-013, where the store keeps
  the key under the host of that URL.

## Codebase Context

**The selection today.** `Starter.Select(name, given, givenByFeature)` checks the starter values, takes the given
features in the order of their names (`Starter.selectedFeatures`, `internal/catalog/starter.go:160-170`), gives
each feature its values with the defaults (`options.withDefaults`, `internal/catalog/option.go:49-75`), and builds
the templates and the extension points (`internal/catalog/starter.go:100-128`). `extensionPointsOf` first takes the
extension points that the starter and each selected feature declare, and then adds the items of each `add_to`, by
feature in the order of the names (`internal/catalog/starter.go:184-208`). A name that is not a feature that
targets the starter stops with `unknownFeature`, which lists the features that target it
(`internal/catalog/starter.go:210-216`). `Catalog.Starter` reads only the features whose `targets` name the starter
(`internal/catalog/catalog.go:50-72`), so the starter cannot tell a feature that does not exist from one that
targets another starter. `readFeatureFor` reads no `needs` (`internal/catalog/feature.go:23-90`). No code here walks
a graph or sorts by a dependency.

**The options of a template.** `Selection.option` reads a bare name from the values of the feature of the template,
and `starter.<name>` from the starter values. A wrong name stops the render, and the message lists the names that
the template can read (`internal/catalog/selection.go:100-117`). The `Selection` keeps no needs.

**The output of `new`.** `new` parses `--feature` into a map, so the order of the flags is lost before `Select`
(`internal/cmd/new.go:73-95`). It writes, runs the `after_render` commands, and prints one line,
`Made <output> from the starter <starter>.` (`internal/cmd/new.go:24-57`). `Combination` is the precedent for an
exported struct of names that `internal/cmd` reads (`internal/catalog/combination.go:9-13`).

**Combinations.** `Starter.combinations` gives the starter alone, each feature alone, each other option value, and
each pair of features that add to one extension point (`internal/catalog/starter.go:130-158`). A combination names
only the features that it tests, so `Select` adds its needs. The catalog today has no pair (`api-auth` adds to
`api.middleware` and `api.config`, and `cli-auth` will add to `cli.commands`), so no pair has a need between its
features.

**The catalog.** `cli` declares `cli.commands` with the item field `code` and renders it in
`internal/cli/root.go` after `newVersionCommand()` and `newUpdateCommand()`, with no import field
(`catalog/features/cli/files/go/internal/cli/root.go.tmpl:24-37`). So `newAuthCommand()` must be a function of the
package `cli`, and `cli-auth` adds its file `internal/cli/auth.go` to that folder. `cli` also writes the e2e tests
of the CLI (`e2e/`, with `runCli` and `scratchDir` in `e2e/testUtils.ts`). `api-auth` has the option `method` with
`values: [api-key]`, and `auth.Middleware(cfg auth.Config)` answers 401 with `WWW-Authenticate: Bearer` for a
request without a known key (`catalog/features/api-auth/files/common/go/internal/auth/middleware.go.tmpl`). Its
test builds a config from the hashes of keys of the test with `KeyHashes.UnmarshalText`
(`catalog/features/api-auth/files/common/go/internal/auth/middleware_test.go.tmpl:14-45`). The template functions
are `option`, `items`, `unique`, `githubRepository`, `kebab`, `snake`, `camel` and `pascal`
(`internal/catalog/selection.go:88`); none gives capitals.

**`go-cli-auth`, preset `api-key`.** `internal/credential/credential.go` has `Resolve`, which takes the key from
`<PREFIX>_API_KEY` and then from `credentialCommand` of `$XDG_CONFIG_HOME/<name>/config.yaml`, and returns
`ErrMissing` with the ways to give the key. `command.go` runs the command through `sh` and shows its stderr.
`config.go` finds and reads the user config. `credential_test.go` tests these. `.go-cli-auth/auth.go` has
`newAuthCommand` with `auth status`, which prints `The API key comes from <source>.`. `.go-cli-auth/auth.test.ts`
is its e2e test, and `.go-cli-auth/README.md.part` and `docs/authentication.md` are its docs. The hook renames the
module, the app and the prefix with perl, and adds `newAuthCommand()` to the root command with ast-grep; the item
of `cli.commands` replaces that edit. The preset has no API client: its docs tell the developer to call
`credential.Resolve` before each request and set the header. `api_url.go` (`apiURL`, `<PREFIX>_API_URL`) belongs
to the presets with a store.

**The API of the starter.** The only protected routes are the account routes
(`catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/account/handler.go.tmpl:46-50`). Without
Postgres, `POST /accounts` with `{}` gets 400 from the handler and the other routes fail at the first query
(`tasks/learnings.md:105-110`), so no protected route answers 2xx in the end-to-end tests. The default port is
`3333` (`internal/api/config.go.tmpl` of the starter).

**Tests.** One umbrella test per type in `package catalog_test` with the tag `unit`: `TestStarter` (`select`),
`TestSelection` (`write`, `run after render`, `run tests`) and `TestCatalog` (`starter`, `embedded`,
`combinations`). `TestNewCommand` in `internal/cmd/new_test.go` runs `new` on catalogs in `fstest.MapFS`.
`repoCatalog()` reads the catalog folder of this repo (`internal/catalog/repo_catalog_test.go`), and
`catalogOnDisk` copies it into a `fstest.MapFS` (`internal/catalog/catalog_test.go:907-923`). The e2e tests make
three full repos with the real binary (`e2e/tests/new.test.ts:43-146`).

**Words.** No code uses `need`. The code says `given` for what the flags give (`givenByFeature`). `api` is the
server package of the starter (`internal/api`), and `Client` is the GitHub client of `release`
(`catalog/features/cli/files/go/internal/cli/release/client.go.tmpl:40-48`).

**Decisions of this breakdown, for the caller to overrule:**

1. **The call to the API of the same repo is checked against the middleware of `api-auth` in the written repo.**
   No command of the CLI calls a protected route, and `cli-auth` cannot name one (rule 6). So the unit tests of the
   API client (Task 5) call an `httptest` server whose handler goes through `auth.Middleware` of `internal/auth`,
   with the key hash of a key of the test: this is the whole of what `api-auth` adds to the API. `task test:unit`,
   `task test:combinations` and the full run of Task 4 run these tests. The end-to-end test starts no API. The
   other choice is a protected route that the CLI can call in each repo, for example an extension point
   `api.routes` with a route of `api-auth` and a command `auth check`. That changes the starter and `api-auth`,
   which the story does not name.
2. **The order of rule 8 is: of the orders in which each feature comes after its needs, the first in the order of
   the names.** Rule 8 cannot hold for all pairs: with `app` that needs `zone`, and `base`, the order is `base`,
   `zone`, `app`, so `app` comes after `base`.
3. **A need that is not a feature that targets the starter, and needs that form a cycle, stop `Select`** before the
   first file, with a message that names the feature and the need, or the features of the cycle. `new` returns and
   writes nothing. `check` (US-006) owns the checks over the whole catalog.
4. **`new` prints `<feature>: needed by <features>` for each feature that a need added**, in the order of rule 8,
   before the line that names the output folder. A feature that a flag names gets no line. `Selection.AddedByNeeds`
   gives the list as `Need` values, which the dry run, the selection file and the TUI also read later.
5. **A template reads the options of the needs that its own `feature.yml` names**, as `<need>.<option>`. A need of
   a need is not readable.
6. **A pair in which one feature needs the other, directly or through its needs, gives no combination of its
   own**: its repo is the repo of the feature alone. This closes the first item that US-004 deferred to this story.
7. **The variable of the key is `<NAME>_API_KEY`, with the name of the repo in capitals and `_`**, for example
   `MY_SERVICE_API_KEY`, through a new template function `upper` (`{{ upper (snake .Name) }}`). The config file is
   `$XDG_CONFIG_HOME/<kebab name>/config.yaml`, as in `go-cli-auth`.
8. **The docs of `cli-auth` are `docs/cli-auth.md`**, for a user and for a developer, as `docs/api-auth.md` and
   `docs/cli.md` are.
9. **The API client takes the URL of the API from the code that makes it.** An answer 401 fails the call with the
   source of the refused key and the ways to give the key. Each other answer goes to the caller as it is.
10. **The test of the API client imports `internal/auth` of `api-auth`.** `cli-auth` always needs `api-auth`
    (open question 2 of the stories file), so each repo with `cli-auth` has that package.

## Tasks

### Task 1: Select the needs of each selected feature, in the order of rule 8

**Behavior:** A `feature.yml` names the features that it cannot work without as `needs: [<feature>, ...]`.
`Starter.Select` adds the needs of each given feature, and their needs, to the selection, and the selection writes
their files and their items as it does for a given feature. The items of an extension point come in the order of
rule 8. A need that the starter does not have, and needs that form a cycle, stop the selection. A pair in which one
feature needs the other gives no combination of its own.

**Acceptance Criteria:**
- [x] A feature whose `feature.yml` has `needs: [b, c]`, given alone to `Select`, gives a selection that writes the
  files of the feature, of `b` and of `c`, and the items that each of them adds. A need of a need is also selected.
  A feature with no `needs` selects as before, and the existing tests of `TestStarter`, `TestSelection` and
  `TestCatalog` pass with their expected values unchanged.
- [x] A need that no flag gives takes the defaults of its options, rendered with the name. A need that is also given
  keeps its given values, and the selection holds it one time.
- [x] A need that is not a feature that targets the starter, because it does not exist or because its `targets` do
  not name the starter, stops `Select` before the first file. The message names the feature that needs it, the
  need, and the features that target the starter.
- [x] Needs that form a cycle, also a feature that needs itself, stop `Select`, and `Select` returns. The message
  names each feature of the cycle. A cycle among features that the selection does not hold does not stop it.
- [x] The items of an extension point come in the order of rule 8 (decision 2). The test states the full order for
  a selection where it differs from the order of the names, for example `app` that needs `zone`, with `base`:
  `base`, `zone`, `app`. Each order of the given features gives the same items.
- [x] A feature that adds items to an extension point that only its need declares, for example an item of
  `cli.commands` with `cli` as its need, selects alone, and its items render in the file of the need.
- [x] `Catalog.Combinations` gives no combination for a pair of features in which one needs the other, directly or
  through its needs (decision 6). The combination of a feature alone still names only that feature, and the
  selection of that combination holds the needs of the feature.
- [x] The section "Test the Combinations of the Catalog" of `README.md` says that the repo of a feature has its
  needs, and that a pair in which one feature needs the other has no repo of its own.

**Affected Files/Modules:**
- `internal/catalog/feature.go` — read `needs`
- `internal/catalog/starter.go` — the needs in `Select`, the order of rule 8, the two stops, and the pair rule in
  `combinations`
- `internal/catalog/starter_test.go` — the needs in the group `select` of `TestStarter`
- `internal/catalog/selection_test.go` — the order of the items with needs
- `internal/catalog/catalog_test.go` — the pair with a need in the group `combinations` of `TestCatalog`
- `README.md` — the section about the combinations

**Patterns to Follow:**
- `internal/catalog/starter.go:100-128` for `Select`, and `internal/catalog/starter.go:160-170` for the given features
- `internal/catalog/starter.go:184-216` for the extension points of the selection and the stop for an unknown feature
- `internal/catalog/starter.go:130-158` for the combinations of a starter
- `internal/catalog/feature.go:23-90` for the read of a `feature.yml`, and `internal/catalog/feature.go:104-111` for
  a loop over a map in the order of its keys
- `internal/catalog/starter_test.go:155-197` for the stops that list the features that target the starter
- `internal/catalog/selection_test.go:534-553` for a test of the order of items
- `internal/catalog/catalog_test.go:858-890` for the test of the pairs
- `PROPOSAL.md:243`, `PROPOSAL.md:337`, `PROPOSAL.md:343-345`, `PROPOSAL.md:378` and `PROPOSAL.md:479` for the
  field `needs`, rules 4 and 8, the first step of `new`, and the cycle
- `README.md:91-102` for the section about the combinations
- `tasks/learnings.md:133-138` for a loop over a map that stops at the first match
- `tasks/learnings.md:64-69` for the fixes of an audit round in the umbrella test files

**Testable:** Yes

**Certainty:** low — no code here walks a graph: the closure of the needs, the order of rule 8 and the stop for a
cycle are designed here, and `Starter.selectedFeatures` only sorts the given names

**Blast radius:** low — none of the named categories; it decides which features a repo holds, and CI makes the repo
of each combination

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines of
`internal/catalog`.

**Depends on:** None

---

### Task 2: Name each feature that a need added in the output of `new`

**Behavior:** `Selection.AddedByNeeds` gives each feature that the selection holds only because a selected feature
needs it, with the selected features that need it. `new` prints one line for each, as
`<feature>: needed by <features>`, before the line that names the output folder. The order of the `--feature` flags
changes neither the files nor the output.

**Acceptance Criteria:**
- [x] `new service my-service --feature x`, where `x` needs `y` and `z`, prints `y: needed by x` and
  `z: needed by x`, one line each, in the order of rule 8, and then the line that names the output folder.
- [x] A feature that two selected features need gets one line that names both, for example `y: needed by a, b`. A
  need of a need gets its own line, which names the feature that needs it: with `a` that needs `b`, and `b` that
  needs `c`, the lines are `b: needed by a` and `c: needed by b`.
- [x] A feature that a `--feature` flag names gets no line, also when a selected feature needs it. With no need in
  the selection, `new` prints only the line that names the output folder, as before.
- [x] `Selection.AddedByNeeds` gives the same features, with the same features that need them, in the same order,
  as `Need` values (decision 4).
- [x] `--feature a --feature b` and `--feature b --feature a`, where `a` needs `b` and both add items to one
  extension point, give the same files and the same output. The two orders also give the same files and output
  when the flag of `b` gives a value of an option of `b`.
- [x] A need that stops `Select` (Task 1) stops `new` with the message of the catalog, and `new` makes no output
  folder.

**Affected Files/Modules:**
- `internal/catalog/need.go` — the new type `Need`
- `internal/catalog/selection.go` — `AddedByNeeds`
- `internal/catalog/starter.go` — the given features and the needs that the selection keeps
- `internal/catalog/selection_test.go` — a group for `AddedByNeeds` in `TestSelection`
- `internal/cmd/new.go` — the lines of the needs
- `internal/cmd/new_test.go` — the output and the two orders of the flags in `TestNewCommand`

**Patterns to Follow:**
- `internal/cmd/new.go:24-57` for the steps of `new` and its closing line
- `internal/cmd/new_test.go:176-184` for a test of what `new` prints
- `internal/cmd/new_test.go:256-283` for a test that runs `new` more than one time and compares the files
- `internal/cmd/new_test.go:354-370` for a stop that makes no output folder
- `internal/catalog/combination.go:9-13` for an exported struct of names that `internal/cmd` reads
- `PROPOSAL.md:409-413` for the line `api-auth: needed by cli-auth`
- `task:1` for the needs of a selection

**Testable:** Yes

**Certainty:** medium — `internal/cmd/new.go:24-57` prints a line from a selection and
`internal/catalog/combination.go:9-13` is an exported result that `internal/cmd` reads, but no result of a
selection tells a given feature from a feature that a need added

**Blast radius:** low — none of the named categories; the lines are for a person, and no program reads the output
of `new`

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines.

**Depends on:** Task 1

---

### Task 3: Read the options of a need in the templates of a feature

**Behavior:** A template of a feature reads `{{ option "<need>.<option>" }}` for each option of each need that its
`feature.yml` names (rule 5), in its contents and in its path. The value is the value of that option in the
selection.

**Acceptance Criteria:**
- [ ] A template of a feature reads an option of a need that its `feature.yml` names as `<need>.<option>`, in its
  contents and in its path. The value is the value of that option in the selection: the default of the need, or the
  value that `--feature <need>:<option>=<value>` gives. The test uses a value that is not the default.
- [ ] A template of a feature still reads its own options by their names, and the options of the starter as
  `starter.<option>`.
- [ ] `<feature>.<option>` stops the write before the first file when the feature is selected but is not a need of
  the feature of the template, when it is a need of a need (decision 5), or when the need does not declare the
  option. The message names the template and lists the names that the template can read, which include
  `<need>.<option>` for each option of each need of its feature.
- [ ] A template of the starter still reads only the options of the starter: `<feature>.<option>` in a template of
  the starter stops the write, and the message lists the options of the starter.
- [ ] The default of an option still reads only the name of the repo.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — `option` reads the options of the needs of the feature
- `internal/catalog/starter_test.go` — the options of a need in the group `select` of `TestStarter`

**Patterns to Follow:**
- `internal/catalog/selection.go:86-117` for the render and the read of an option
- `internal/catalog/starter_test.go:310-364` for the options of a feature in a template
- `internal/catalog/starter_test.go:445-475` for the stops that list the names that a template can read
- `PROPOSAL.md:286-307` and `PROPOSAL.md:338-339` for the names of options and rule 5
- `task:1` for the needs of a selection

**Testable:** Yes

**Certainty:** high — `Selection.option` (`internal/catalog/selection.go:100-117`) reads `starter.<option>` by its
prefix and lists the names that a template can read; a need is one more prefix of the same kind

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes. `clerk mutants` finds no live mutant on the changed lines.

**Depends on:** Task 1

---

### Task 4: Add the feature `cli-auth`, whose CLI takes the API key from a variable or a credential command and shows its source with `auth status`

**Behavior:** The catalog holds `cli-auth`, which needs `cli` and `api-auth` and has no option. It adds the `auth`
command to `cli.commands`, and the package that gets the key. The CLI takes the key from `<NAME>_API_KEY`, and else
from the output of `credentialCommand` in its user config, and never stores it. `auth status` shows where the key
comes from, and never the key. The files are the preset `api-key` of `global/go-cli-auth`, ported with sentinel
values. An end-to-end test makes a repo with `--feature cli-auth` with the real binary.

**Acceptance Criteria:**
- [ ] `catalog/features/cli-auth/feature.yml` has a `summary` that says what the repo gets; `targets` with only
  `go-event-sourcing-service`; `needs: [cli, api-auth]`; no option; `files` with the one entry `files/common/`; and
  the item `newAuthCommand()` for `cli.commands` in `add_to`. It names no language.
- [ ] Written with `go-event-sourcing-service`, the name `my-service` and `cli-auth` alone, the repo has the files of
  the starter, of `cli`, of `api-auth` and of `cli-auth`. Each file of the repo written with `cli` and `api-auth`
  and without `cli-auth` has the same contents, except `internal/cli/root.go`, which lists `newAuthCommand()` after
  `newVersionCommand()` and `newUpdateCommand()`. `cli-auth` alone and `cli-auth` with `api-auth` give the same
  files.
- [ ] The template function `upper` gives its text in capitals, so `{{ upper (snake .Name) }}` gives `MY_SERVICE`
  for the names `my-service` and `My.Service` (decision 7).
- [ ] In the written repo, the CLI takes the key from `MY_SERVICE_API_KEY` when it is not empty. Else it runs the
  `credentialCommand` of `$XDG_CONFIG_HOME/my-service/config.yaml` through `sh`, or of
  `~/.config/my-service/config.yaml` when `XDG_CONFIG_HOME` is not an absolute path, and takes what the command
  prints without the white space at its ends. The variable comes before the command. A command that fails, or that
  prints nothing, is an error that names the config file, and the user sees what the command writes to stderr.
  With no source, the error says how to give the key: it names `MY_SERVICE_API_KEY` and `credentialCommand` in the
  path of the config file.
- [ ] The CLI has no flag for the key, and writes no file: after `auth status` and after the unit tests, the config
  folder holds only what the test wrote.
- [ ] The root command of the CLI lists `auth` after `version` and `update`. `auth status` prints
  `The API key comes from <source>.`, where the source is `MY_SERVICE_API_KEY` or
  `the credentialCommand in <path>`, and it never prints the key. With no source, it exits with a status other than
  0, and its message is the error above.
- [ ] `cli-auth` has no option for the method: `--feature cli-auth:method=api-key` stops, and the message says that
  the feature `cli-auth` has no option. `docs/cli-auth.md` names the method of the API, which it reads as
  `api-auth.method` (Task 3), and says that the CLI follows the method of the API.
- [ ] `docs/cli-auth.md` tells a user how to give the key with the variable and with `credentialCommand`, for
  example from a password manager, and what `auth status` shows. It tells a developer where the code is. It names no
  task, file or setting that the written repo does not have.
- [ ] No written file holds a value of scaffold: `.Scaffold.`, `.Computed.`, `go-cli-project`, `GO_CLI_PROJECT`,
  `zzsentinel` or `<no value>`. No Go file of `cli-auth` imports a package of the starter: each package of the
  module that it imports belongs to `cli-auth`, to `cli` or to `api-auth` (rule 6).
- [ ] In the written repo, `task test:unit` runs the unit tests of the package that gets the key, and the e2e test
  `e2e/tests/auth.test.ts` passes on the host against the built CLI.
- [ ] An end-to-end test of this repo runs `feature-compose new go-event-sourcing-service my-service --feature
  cli-auth --no-prompt`. It exits with 0, and its output has the lines `api-auth: needed by cli-auth` and
  `cli: needed by cli-auth`. In that repo, `gofmt -l .` lists no file, `go build ./...` exits with 0, and
  `task test:unit` exits with 0, with the package that gets the key, `internal/auth`, `internal/cli` and the
  packages of the service among the packages that pass. `go run ./cmd/my-service` lists the commands `version`,
  `update`, `auth` and `help`. `auth status` of the built CLI, with `MY_SERVICE_API_KEY` set, names the variable
  and does not print the key. With the variable empty and an empty config folder, it exits with a status other
  than 0, and its output names `MY_SERVICE_API_KEY` and `credentialCommand`.
- [ ] The end-to-end test of an unknown feature expects the features `api-auth, cli, cli-auth`. `docs/e2e-tests.md`
  describes the full run with `cli-auth`.
- [ ] `task test:combinations` passes, with `go-event-sourcing-service --feature cli-auth` among its subtests.
- [ ] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass, and `git status --ignored catalog/`
  shows no file of `cli-auth` as ignored.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — new
- `catalog/features/cli-auth/files/common/go/internal/credential/` — the key from the variable or the command, the
  user config, and their unit tests; each `.go` file has the `.tmpl` suffix
- `catalog/features/cli-auth/files/common/go/internal/cli/auth.go.tmpl` — `newAuthCommand` and `auth status`, in
  the package of the root command of `cli`
- `catalog/features/cli-auth/files/common/go/e2e/tests/auth.test.ts` — the e2e test of `auth status` in the written
  repo
- `catalog/features/cli-auth/files/common/go/docs/cli-auth.md` — new
- `internal/catalog/selection.go` — the template function `upper`
- `internal/catalog/selection_test.go` — a render test of `upper`
- `internal/catalog/catalog_test.go` — the starter with `cli-auth`
- `e2e/tests/new.test.ts` — the full run with `cli-auth`, and the list of the features
- `docs/e2e-tests.md` — the full run with `cli-auth`

**Patterns to Follow:**
- `catalog/features/api-auth/feature.yml` and `catalog/features/cli/feature.yml` for a `feature.yml`
- `catalog/features/cli/files/go/internal/cli/root.go.tmpl:24-37` for the items of `cli.commands`
- `internal/catalog/selection.go:86-98` for the template functions, and `internal/catalog/selection_test.go:78-103`
  for their render tests
- `internal/catalog/catalog_test.go:433-540` for the files that a feature adds, the files that stay the same, the
  imports of a feature and its docs
- `internal/catalog/catalog_test.go:621-657` for the root command of the CLI and the values of scaffold
- `internal/catalog/catalog_test.go:907-923` for a copy of the catalog of this repo
- `e2e/tests/new.test.ts:69-100` and `e2e/tests/new.test.ts:122-146` for a full run, and
  `e2e/tests/new.test.ts:266-281` for the packages that passed and the commands of the CLI
- `e2e/tests/new.test.ts:112-120` for the list of the features
- `e2e/testUtils.ts:40-70` for `runProgram`
- `docs/e2e-tests.md:80-122` for the description of the full runs
- `PROPOSAL.md:194-212` for the `feature.yml` of `cli-auth`, and `PROPOSAL.md:488-512` for its contract
- `tasks/learnings.md:23-28` for the port with sentinel values, and `tasks/learnings.md:30-35` for the files that
  are ported unchanged
- `tasks/learnings.md:10-21` for `.tmpl`, `#!` and the delimiters globs
- `tasks/learnings.md:77-82` for a new template function
- `tasks/learnings.md:84-96` and `tasks/learnings.md:126-131` for the proof on a written repo with breaks by hand
- `tasks/learnings.md:119-124` for a test that reads an environment variable
- `tasks/learnings.md:37-42`, `tasks/learnings.md:51-56` and `tasks/learnings.md:58-63` for the full run
- `task:1`, `task:2` and `task:3` for the needs, their output and `api-auth.method`

**Testable:** Yes

**Certainty:** medium — `catalog/features/api-auth/` and `catalog/features/cli/` are features with Go code, tests and
docs, and `tasks/learnings.md:23-28` gives the port, but no feature yet writes a file into the folder of another
feature or adds an item to an extension point of its need, and `go-cli-auth` is a global template whose hook edits
the root command, which the item of `cli.commands` replaces

**Blast radius:** high — credential handling: the CLI reads the key from the environment, and runs the command of
the user config to get it

**Verification:** `task test:unit` and `go vet ./...` pass. Write a repo with `--feature cli-auth` with the real
binary into a scratch folder, and run `gofmt -l .`, `go build ./...` and `task test:unit` there. Build its CLI and
run its `e2e/tests/auth.test.ts` on the host. On the written files, apply one break at a time, read its diff, run
the unit tests, and copy the file back: the command before the variable, the key in the output of `auth status`, an
error with no source that does not name the variable, a command whose stderr the user does not see. Each break makes
a test fail. Run `e2e/tests/new.test.ts` on the host (`tasks/learnings.md:58-63`), and `task test:combinations`.

**Depends on:** Task 1, Task 2, Task 3

---

### Task 5: Send the API key on each call of the CLI to the API, and tell the user how to give the key when the API refuses it

**Behavior:** `cli-auth` adds the API client of the CLI, through which each command calls the API of the repo.
Before each call, the client gets the key as `auth status` does, and sends `Authorization: Bearer <key>`. With no
key, the call fails before a request leaves the CLI. An answer 401 fails the call with a message that names the
source of the refused key and the ways to give a key. In the written repo, the unit tests of the client call a
test server that runs the `auth.Middleware` of `api-auth`.

**Acceptance Criteria:**
- [ ] The written repo has one API client of the CLI, in a package of `cli-auth` that is not named `api`. It takes
  the URL of the API from the code that makes it (decision 9).
- [ ] Each call through the client gets the key from its sources before the request, and sends it as
  `Authorization: Bearer <key>`, the only `Authorization` header of the request. The client keeps no key: when
  `MY_SERVICE_API_KEY` changes between two calls, the second call sends the new key.
- [ ] With no key, the call fails, no request reaches the server, and the error names `MY_SERVICE_API_KEY` and
  `credentialCommand`, as in Task 4.
- [ ] An answer 401 fails the call. The message says that the API refused the key from its source, and names
  `MY_SERVICE_API_KEY` and `credentialCommand` as the ways to give the key. Each other answer, for example 200, 400
  or 403, goes to the caller as the API gave it.
- [ ] In the written repo, `task test:unit` runs the unit tests of the client against an `httptest` server whose
  handler goes through `auth.Middleware` of `internal/auth`, with a config that holds the key hash of a key of the
  test (decision 1, decision 10). With that key, from the variable and from `credentialCommand`, the call gets the
  answer of the handler. With a wrong key, it fails with the message of a 401. With no key, it fails before a
  request.
- [ ] `docs/cli-auth.md` tells a developer how a command of the CLI calls the API through the client, and names only
  files that the written repo has.
- [ ] The only package of `api-auth` that a file of `cli-auth` imports is `internal/auth`, and only a test file
  imports it. The check of the imports of Task 4 states this.
- [ ] The full run of Task 4 passes, with the package of the client among the packages that pass.

**Affected Files/Modules:**
- `catalog/features/cli-auth/files/common/go/internal/` — the API client and its unit tests, in a package of
  `cli-auth`; each `.go` file has the `.tmpl` suffix
- `catalog/features/cli-auth/files/common/go/docs/cli-auth.md` — a section for the developer about the client
- `internal/catalog/catalog_test.go` — the files and the imports of `cli-auth`
- `e2e/tests/new.test.ts` — the packages that pass in the full run with `cli-auth`

**Patterns to Follow:**
- `catalog/features/api-auth/files/common/go/internal/auth/middleware_test.go.tmpl:14-45` for a config with the key
  hashes of keys of the test, and a handler behind the middleware
- `catalog/features/api-auth/files/common/go/internal/auth/middleware.go.tmpl` for the answer 401
- `catalog/features/cli/files/go/internal/cli/release/client.go.tmpl:40-121` for an HTTP client in a written repo
  that sends a credential
- `internal/catalog/catalog_test.go:506-523` for the check of the imports of a feature
- `tasks/learnings.md:84-96` and `tasks/learnings.md:126-131` for the proof on a written repo with breaks by hand
- `tasks/learnings.md:119-124` for a test that reads an environment variable
- `task:4` for the package that gets the key, the docs and the full run

**Testable:** Yes

**Certainty:** low — no precedent: `go-cli-auth` tells the developer to call `credential.Resolve` before each
request and has no client, so the client, its answer to 401 and its test against the middleware of `api-auth` are
designed here

**Blast radius:** high — credential handling: the client sends the key on each call, and decides what the user sees
when the API refuses it

**Verification:** `task test:unit` passes. Write a repo with `--feature cli-auth` with the real binary, and run
`task test:unit` there. On the written files, apply one break at a time, read its diff, run the tests of the client,
and copy the file back: no `Authorization` header, the scheme `Token`, the key of the first call kept for later
calls, no error for 401, a 401 message without the variable, a request with no key. Each break makes a test fail.
Run `e2e/tests/new.test.ts` on the host, and `task test:combinations`.

**Depends on:** Task 4

## Summary

**Total:** 5 tasks.

**Order.** Dependency first, then risk. Tasks 1 to 3 add needs to the tool on catalogs in memory: the selection of
the needs and the order of rule 8 (Task 1), the output of `new` (Task 2), and the options of a need in a template
(Task 3). Task 1 is first because each other task reads the needs, and it is the task with the most design in the
tool. Tasks 4 and 5 add `cli-auth`, and they are the two tasks with a high blast radius, so each has its own review:
Task 4 is the port of the preset `api-key`, which gets the key, and Task 5 is new code, which sends it.

**Coverage of the story:**

| Criterion of US-005 | Tasks |
| --- | --- |
| `--feature cli-auth --no-prompt` also selects `cli` and `api-auth`; the output names each feature that a need added | 1 (the selection), 2 (the output), 4 (the real binary) |
| The CLI takes the key from a variable or a credential command, sends `Authorization: Bearer <key>` on each call, and does not store the key | 4 (the sources, no store), 5 (the header on each call, no key kept) |
| A call with the right key to the API of the same repo succeeds; no key or a wrong key fails, and the message tells the user how to give the key | 5 (the client against `auth.Middleware` of the repo, decision 1), 4 (`auth status` with no key) |
| The root command shows the `auth` command that `cli-auth` adds through `cli.commands`; `auth status` shows the source and not the key | 1 (an item for an extension point of a need), 4 |
| `cli-auth` has no option for the method, and uses `method` of `api-auth` | 3 (`<need>.<option>`), 4 (no option, and the docs read `api-auth.method`) |
| The order of the `--feature` flags does not change the files | 1 (rule 8), 2 (`new` with both orders) |
| Context: the preset `api-key` of `global/go-cli-auth` | 4 |
| Context: rule 8 | 1 |

No criterion of the story is deferred. The third criterion is checked against the middleware of `api-auth` in the
written repo, and not against the running API (decision 1).

**Criteria that this breakdown adds, for the caller to overrule:**

- A need that the starter does not have, and a cycle, stop `Select` (Task 1, decision 3).
- A pair in which one feature needs the other has no combination (Task 1, decision 6).
- `Selection.AddedByNeeds` and its `Need` values, and no line for a feature that a flag names (Task 2, decision 4).
- The stops for an option of a selected feature that is not a need, and of a need of a need (Task 3, decision 5).
- The template function `upper`, and the variable `<NAME>_API_KEY` (Task 4, decision 7).
- `docs/cli-auth.md` (Task 4, decision 8).
- The API client, its answer to 401, and its test against `auth.Middleware` (Task 5, decisions 1, 9 and 10).

**Notes.**

- Files that Task 4 ports from the preset `api-key` with sentinel values, for the brief of the audit
  (`tasks/learnings.md:30-35`): `credential.go`, `command.go`, `config.go` and `credential_test.go` of
  `internal/credential`, `auth.go` (from `.go-cli-auth/auth.go`, with only `auth status`) and `auth.test.ts`.
  `docs/cli-auth.md` joins `README.md.part` and `docs/authentication.md`. New code: the needs in the tool, `upper`,
  and the API client of Task 5 with its test.
- `clerk mutants` covers the tool code of Tasks 1 to 4. The templates of Tasks 4 and 5 get "not checked", so their
  proof is the written repo with breaks by hand (`tasks/learnings.md:84-89`).
- The needs of `cli-auth` take the map form in US-007, which sets `method: api-key`, so that
  `--feature cli-auth --feature api-auth:method=oauth` stops.
- The full run with `cli-auth` adds a fourth full repo to the e2e tests, about one more minute with a warm module
  cache. `task test:combinations` gets a fourth combination.
