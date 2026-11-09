# Tasks: Protect the API with API Keys (US-003)

## Contents

1. Task 1: Declare extension points in a starter, and add items to them with `add_to`
2. Task 2: Render each value of one item field one time with `unique`
3. Task 3: Declare options in a feature, and read them in the templates of the feature
4. Task 4: Set the options of a feature with `--feature <name>:<option>=<value>`
5. Task 5: Render the items of `api.middleware` and `api.config` in the server and the config of the starter
6. Task 6: Add the feature `api-auth`, whose method `api-key` lets only a request with a known key reach a protected route
7. Task 7: Check end to end that a repo with `api-auth` refuses a request without a known key, and that repos with and without it pass their tests

## Story Reference

US-003, "Protect the API with API keys", in `user-stories/feature-compose.md` (Acceptance Criteria and Context).
The design is `PROPOSAL.md`: sections 6 (`extension_points` of `starter.yml`), 7 (`options` and `add_to` of
`feature.yml`, and the `feature.yml` of `api-auth`), 8 (options), 9 (`option`, `items` and `unique`), 10 (rules 1,
2, 6, 7 and 8), 11 (`--feature <name>[:<option>=<value>,...]`) and 15 (the contract of `api-auth`).

## Boundaries

**Out of scope:**

The non-goals of the stories file, as written there:

- The command `add`, and an update of a repo to a new version of the catalog. Both need a three-way merge,
  which is phase 3.
- Generators that you run many times in one repo, for example `go-cli-sub-command`. They stay in scaffold.
- Catalogs from other repos. The catalog is in the binary.
- A starter with more than one folder, for example a Go API in `api/` and a Svelte web app in `web/`. The
  feature `cli` shows G5 in one folder.
- A check of the credential in a repo without `api-auth`. The API of such a repo accepts each request.
- A third method of authentication for Google sign-in, with a session exchange.
- The templates that the proposal has not yet decided: `asdf-plugin`, `go-event-sourcing-cli`, `go-worker`,
  `global/git-hook`, and `global/kustomize`.
- The removal of `repo-templates`.

Added by this breakdown:

- Permissions of a key. With `api-key`, each key that the API knows has every permission, so the API never
  answers 403 (decision 10). The story allows 403 only for a valid key that has no permission, and `api-key`
  has no such key.
- `values` on an option of a starter. No starter option has a list of values, and the tool reads `values` only
  in `feature.yml`.
- A value of a feature option that holds a comma. In `--feature <name>:<option>=<value>,...`, a comma separates
  two options (decision 8).
- Strict checks of `starter.yml` and `feature.yml`, such as unknown keys, a default that is not one of the values
  of its option, or two targets that declare one extension point with different item fields. `check` (US-006,
  US-009) owns the checks of the catalog.

**Deferred:**

- `needs`, and `option "<need>.<option>"` in a template: US-005. In this story, an `add_to` whose extension point
  neither the starter nor a selected feature declares stops `new` (decision 5). US-005 selects the need that
  declares it.
- The value `oauth` of `method`, the entry `files/oauth/` with its `when`, and `when` itself: US-007.
- The CI that makes a repo for each value of `method`: US-004. `show`, which prints the extension points of a
  starter and the options of a feature: US-014. `--dry-run`: US-010. `.feature-compose.yml`: US-011. The TUI:
  US-012.
- `chi-api` as a target of `api-auth`: US-015. `chi-api` must declare `api.middleware` and `api.config` with the
  same item fields, and its config must read the `env` tags of the field that `api.config` adds, as the config
  loader of `go-event-sourcing-service` does.
- The extension point `catchup.consumers` of the starter: US-017, with `process-manager`, the first feature that
  adds to it. The starter does not declare an extension point that no file renders.
- The parts of a repo with `api-auth` that need a change to a file that the starter owns (rule 7). This story
  adds only `api.middleware` and `api.config`, and no extension point takes these parts. Until a story adds one,
  `docs/api-auth.md` of the repo gives the commands:
  - `Taskfile.api.yml`: the requests of `open-account`, `deposit`, `withdraw`, `close-account` and `get-account`
    send no key, so in a repo with `api-auth` they get 401.
  - `README.md`: the quick start calls those tasks.
  - `docker-compose.local.yml` and `.env.local`: no `API_KEY_HASHES`, so the `api` service refuses each request
    to a protected route.
- The parts of go-cli that US-002 deferred: the tasks `run`, `build`, `e2e` and `e2e:local` and the include of
  `Taskfile.release.yml`; the `e2e` job, the `workflow_call` trigger and the release workflows; the tools
  `goreleaser` and `node` in `mise.toml`; and the lines `dist/` and `e2e/node_modules/` in `.gitignore`. `add_to`
  exists after this story, but each part still needs an extension point in a file of the starter, which no story
  plans, and the CI parts are open question 3 of the proposal. US-003 does not need them.

## Codebase Context

**The catalog after US-002.** `Catalog.Starter` reads the features that target the starter
(`catalog/catalog.go:27-64`). `readStarter` reads `options`, `folders` and `delimiters` of `starter.yml`, and no
`extension_points` (`catalog/starter.go:26-90`). `readFeatureFor` reads `targets`, `files`, `delimiters` and
`extension_points`, and keeps only the names of the extension points (`catalog/feature.go:21-71`); so the
`extensionPoint` type that the US-002 design planned does not exist yet. `Starter.Select(name, given,
featureNames)` checks the name and the given values of the starter, selects the features in the order of their
names, places their templates, checks one declarer of each extension point among the selected features
(`catalog/starter.go:163-174`), and renders the defaults. `Selection.render` gives each template the same
functions: `option` reads only `starter.<option>` (`catalog/selection.go:86-94`), and `items` returns nil for a
declared extension point and stops for any other name (`catalog/selection.go:96-105`). A template knows its owner
only as the text of its messages (`catalog/template.go:13-20`). `option` has a default and a pattern
(`catalog/option.go`). `Select` has 27 calls, most of them in tests.

**The command.** `new` parses `--option` as `<name>=<value>` (`internal/cmd/new.go:53-63`), gives each `--feature`
value to `Select` as the name of a feature (`internal/cmd/new.go:33`), and sets `DisableSliceFlagSeparator`, so a
value keeps its commas.

**The API of the starter.** `newHandler` (`internal/api/server/server.go.tmpl:31-41` of the starter) adds the
request logger and the recoverer, and calls `use_cases.Register(r, pool, logger)`. `Register`
(`internal/api/use_cases/register.go.tmpl:19-38`) puts the public routes (only `GET /` of `root`) and the
protected routes (the five routes of `account`) on the same router, with nothing between the protected routes and
their handlers: US-001 removed the shared-token check (commit `ee79978`) and kept the split for `api-auth`.
`newHandler` takes no `cfg` today. `Config` (`internal/api/config.go.tmpl`) has `Port` and `DB`. `config.Load` is
`env.Parse` of caarlos0/env v6, which reads a nested struct field with no prefix, as it reads `DB`; so a field of
type `auth.Config` with `env` tags is read the same way. `cmd/api/main.go.tmpl` makes the pool with
`pgxpool.New`, which (pgx v5.11.0, `MinConns` 0) opens no connection before the first query. So the API of a
written repo starts with DB settings that point to no database, and `POST /accounts` with the body `{}` gets the
answer 400 of the handler (`owner` is required) before any query: an end-to-end test can see that a request got to
the handler.

**The words that the starter must not hold.** `catalog/catalog_test.go:354-366` checks that no file of the
starter alone holds `token`, `bearer` or `authorization`. So the server and the config of the starter name the
place of `api.middleware` without those words.

**Tests.** One umbrella test per type: `TestCatalog` (groups `starter` and `embedded`), `TestStarter`
(`select`), `TestSelection` (`write`, `run after render`) and `TestNewCommand` (`write`, `help`, `stop`). The
`embedded` group (`catalog/catalog_test.go:238-575`) writes the real starter without its `after_render` commands.
A test that needs the real `cli` or the real starter together with a feature that the catalog does not hold can
build its catalog from the catalog folder on disk and a feature of the test: the tests run in `catalog/`, and
`catalog.New` takes any `fs.FS`. `e2e/tests/new.test.ts:55-63` expects the text `the features that target the
starter go-event-sourcing-service are cli`, which a second feature in the catalog changes.
`e2e/tests/new.test.ts:91-114` starts a long-running process and waits on it with `until`.

**No precedent for the check of a key.** Neither this repo nor repo-templates checks an API key on a server.
`global/go-cli-auth` covers only the CLI, which reads `API_KEY` (US-005). The shared-token middleware that US-001
removed compared the header with `==` and answered 403, which the contract replaces with 401.

**Decisions of this breakdown, for the caller to overrule:**

1. **The starter declares `api.middleware` and `api.config` as section 6 of the proposal does**: `item: [code,
   import]` and `item: [name, type, import]`. `import` is the path of a package under the module, as
   `internal/auth`.
2. **The items of `api.middleware` apply to the protected routes only.** `GET /` stays public, so a health check
   needs no key.
3. **The order of the items** is rule 8 without needs: by the name of the feature that adds them, then in the order
   of its `add_to`.
4. **`unique` gives the values in the order in which they first come**, over the extension points in the order
   given and the items in the order of `items`.
5. **An `add_to` whose extension point neither the starter nor a selected feature declares stops `new`.** Before
   needs exist, this is the only way that a mistake in the selection does not write a repo without the item.
6. **An item has exactly the fields of its extension point.** A missing field or an extra field stops `new`.
7. **A feature option has `values` or a `pattern`, and its default renders with the name as the default of a
   starter option does.** A template of a feature reads the options of its own feature by their names, and the
   options of the starter as `starter.<option>`.
8. **`--feature` takes `<name>[:<option>=<value>,...]`.** After the `:`, a `,` separates two options, so a value
   cannot hold a comma. Before the `:`, the name keeps its commas, as in US-002. One option of one feature given
   two different values stops `new`.
9. **The files of `api-auth` are under `files/common/`**, as section 5 of the proposal lays them out, so US-007
   adds `files/oauth/` with its `when` and moves no file.
10. **With `api-key`, a known key has every permission**, so the API never answers 403.
11. **The config of the API reads the key hashes from `API_KEY_HASHES`**: the hex form of the SHA-256 hash of each
    key, with `,` between them. With no hash, the API starts and answers 401 to each request to a protected route.
    An entry that is not the hex form of a SHA-256 hash stops the API at start.
12. **The scheme `Bearer` matches in any case**, as HTTP authentication schemes do.
13. **The tasks, the quick start and the local compose file of the starter stay as they are** in a repo with
    `api-auth` (see Deferred). `docs/api-auth.md` gives the request with a key.

## Tasks

### Task 1: Declare extension points in a starter, and add items to them with `add_to`

**Behavior:** `starter.yml` declares `extension_points` in the form of `feature.yml`. A selected feature adds items
to an extension point of the starter or of a selected feature with `add_to`. Each item has exactly the fields of
its extension point. `items "<name>"` gives the items of the extension point in a fixed order. The root command of
the real `cli` shows a command that a feature adds to `cli.commands`.

**Acceptance Criteria:**
- [x] `starter.yml` declares `extension_points.<name>` with `summary`, `file` and `item`, as `feature.yml` does.
  In a template of the starter or of a selected feature, `items "<name>"` for an extension point of the starter
  renders as it does for an extension point of a selected feature, also when no feature adds an item.
- [x] `feature.yml` declares `add_to.<extension point>`: a list of items, each a map of field names to text, in the
  form of section 7 of `PROPOSAL.md`. A selected feature adds its items to the extension point; a feature that is
  not selected adds nothing.
- [x] `{{ range items "<name>" }}` renders each item with its fields, for example `{{ .code }}`. The items of two
  features come in the order of the names of the features, and the items of one feature in the order of its
  `add_to`. The order of the features in the selection does not change a written file.
- [x] An item that lacks a field of its extension point, or has a field that the extension point does not declare,
  stops the selection before the first file is written. The message names the feature, the extension point and
  the fields that the extension point declares.
- [x] An `add_to` whose extension point neither the starter nor a selected feature declares stops the selection
  before the first file is written. The message names the feature and the extension point.
- [x] The starter and a selected feature that declare one extension point stop the selection before the first
  file is written, and the message names the extension point, the starter and the feature. Two selected features
  that declare one extension point stop as before.
- [x] `items` with a name that neither the starter nor a selected feature declares stops the write before the first
  file is written. The message names the template and the extension point, and lists the extension points that
  the starter and the selected features declare, or says that there are none.
- [x] With the feature `cli` of the catalog in this repo and a feature of the test that adds an item to
  `cli.commands`, the written `internal/cli/root.go` lists the code of that item in the commands of the root
  command, after `newVersionCommand()` and `newUpdateCommand()`, and the file parses as Go. Without that feature,
  the root command lists only those two.
- [x] A selection with no `add_to` and no extension point of the starter writes the same files as before.

**Affected Files/Modules:**
- `catalog/` — the extension points of a starter, `add_to` of a feature, the type of an extension point with its
  fields and items, `items`, and their unit tests (`starter_test.go`, `selection_test.go`, `catalog_test.go`)

**Patterns to Follow:**
- `catalog/feature.go:38-42` and `catalog/feature.go:70` for the read of `extension_points`
- `catalog/starter.go:163-174` for the check of one declarer
- `catalog/selection.go:96-105` for `items`
- `catalog/starter_test.go:193-208` for the test of two declarers
- `catalog/selection_test.go:508-558` for the tests of `items`
- `catalog/features/cli/files/go/internal/cli/root.go.tmpl:29-35` for the range over `cli.commands`
- `catalog/catalog_test.go:448-456` for the test of the root command with no item
- `PROPOSAL.md:117-130` for the extension points of a starter, `PROPOSAL.md:169-177` for `add_to`, and
  `PROPOSAL.md:328-346` for rules 1, 2 and 8

**Testable:** Yes

**Certainty:** medium — follows the read of `extension_points` in `catalog/feature.go:38-42` and `items` in
`catalog/selection.go:96-105`, but no instance gives items with fields, checks the fields of an item, or orders
the items of several features

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes and `go vet ./...` is clean

**Depends on:** None

---

### Task 2: Render each value of one item field one time with `unique`

**Behavior:** `{{ range unique "<field>" "<extension point>" ... }}` gives the values of one field over the items
of one or more extension points, each value one time. So a file imports a package one time when two items need it.

**Acceptance Criteria:**
- [x] `unique "<field>" "<a>" "<b>"` gives each value of the field in the items of the extension points, one time
  each, in the order in which the values first come: the extension points in the order given, and the items in
  the order of `items`.
- [x] With no item in the extension points, the range renders nothing, and the rest of the template renders as
  without it.
- [x] An extension point that neither the starter nor a selected feature declares stops the write before the first
  file is written, and the message names the template and the extension point, as for `items`.
- [x] A field that one of the extension points does not declare stops the write before the first file is written,
  and the message names the template, the field and the extension point.
- [x] `unique` with a field and no extension point stops the write before the first file is written, and the
  message names the template.

**Affected Files/Modules:**
- `catalog/` — the template function `unique`, and its unit tests in `selection_test.go`

**Patterns to Follow:**
- `catalog/selection.go:73-105` for the template functions and `items`
- `catalog/selection_test.go:508-558` for the tests of `items`
- `tasks/learnings.md:80-85` for a new template function of the catalog
- `PROPOSAL.md:306-326` for `unique` and its use in `server.go`
- `task:1` for the items of an extension point

**Testable:** Yes

**Certainty:** high — a template function is one entry beside `items` in `catalog/selection.go:74`, and
`tasks/learnings.md:80-85` records such a task as routine

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 1

---

### Task 3: Declare options in a feature, and read them in the templates of the feature

**Behavior:** `feature.yml` declares `options` as `starter.yml` does, and an option can have `values`. A selection
takes the given values of the options of each selected feature. A value that is not one of the values of its
option stops the selection, and the message names the option and its values. A template of a feature reads the
options of its own feature by their names.

**Acceptance Criteria:**
- [x] `feature.yml` declares `options.<name>` with `default`, and `values` (a list) or `pattern`, in the form of
  `api-auth` in section 7 of `PROPOSAL.md`.
- [x] `Starter.Select` takes each selected feature with the given values of its options. A feature with no given
  value is selected as in US-002. An option with no given value has its default, which renders with the name as
  the default of a starter option does.
- [x] A given value that is not one of the `values` of its option stops the selection before the first file is
  written. The message names the feature and the option, and lists the values of the option. A given value that
  does not match the `pattern` of its option stops as for an option of the starter.
- [x] A given value for an option that the feature does not declare stops the selection before the first file is
  written. The message names the feature and the option, and lists the options of the feature, or says that it has
  none.
- [x] In a template of a feature, `option "<name>"` gives the value of the option `<name>` of that feature, and
  `option "starter.<name>"` gives the option of the starter as before. Two selected features with an option of one
  name each read their own value.
- [x] In a template of the starter, a name that is not `starter.<option>` stops the write as before. In a template
  of a feature, a name that is neither an option of the feature nor `starter.<option>` stops the write before the
  first file is written, and the message names the template and lists the names that the template can read.
- [x] `new` builds and keeps its behaviour: it gives each `--feature` name to the selection with no given values.

**Affected Files/Modules:**
- `catalog/` — the options of a feature, the values of an option, the selection of features with their values,
  `option` in a template of a feature, and their unit tests
- `internal/cmd/new.go` — the call of `Select`, with no given values for a feature
- `internal/cmd/new_test.go` — only if a test needs the new call

**Patterns to Follow:**
- `catalog/starter.go:33-37` and `catalog/starter.go:51-61` for the read of options
- `catalog/starter.go:96-105` for the check of given values
- `catalog/starter.go:119-131` for the defaults that render with the name
- `catalog/option.go:8-19` for the check of a value
- `catalog/selection.go:86-94` for `option`
- `catalog/starter_test.go:45-123` for the tests of options
- `catalog/selection_test.go:338-353` for a stop on an unknown name in `option`
- `PROPOSAL.md:159-162`, `PROPOSAL.md:254-273` and `PROPOSAL.md:300-305` for the options of a feature

**Testable:** Yes

**Certainty:** medium — the options of a starter (`catalog/starter.go:51-61`, `catalog/starter.go:96-131`) are the
precedent, but no option here has a list of values, and no template function gives a value that depends on the
owner of the template

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes and `go vet ./...` is clean

**Depends on:** None

---

### Task 4: Set the options of a feature with `--feature <name>:<option>=<value>`

**Behavior:** `new` takes `--feature <name>[:<option>=<value>,...]`. Each pair after the `:` sets one option of the
feature. A value of `--feature` with no `:` is the name of a feature, as in US-002.

**Acceptance Criteria:**
- [x] `--feature <feature>:<option>=<value>` selects the feature and sets the option. After the `:`, a `,`
  separates two `<option>=<value>` pairs, and each pair sets one option.
- [x] A value of `--feature` with no `:` names one feature and keeps its commas: `--feature a,b` names the feature
  `a,b`, as before.
- [x] A pair with no `=`, or an empty pair, stops `new` before the first file with an exit status other than 0.
  The message names the value of the flag and shows the form `<name>:<option>=<value>`. The output folder is not
  made.
- [x] A feature given in more than one flag is selected one time, with the values of all its flags. One option of
  one feature given two different values, in one flag or in two, stops `new`, and the message names the feature,
  the option and both values. The output folder is not made.
- [x] A value that is not one of the values of its option, or an option that the feature does not declare, stops
  `new` with the message of Task 3, and the output folder is not made.
- [x] `new --help` shows the form `<name>[:<option>=<value>,...]` of `--feature`.

**Affected Files/Modules:**
- `internal/cmd/new.go` — the form of `--feature`
- `internal/cmd/new_test.go` — unit tests with catalogs in memory

**Patterns to Follow:**
- `internal/cmd/new.go:53-63` for the parse of `<name>=<value>`
- `internal/cmd/new.go:18-19` for the usage text of a flag
- `internal/cmd/new_test.go:101-112` for a value of `--feature` that keeps its commas
- `internal/cmd/new_test.go:145-154` for a flag that sets options
- `internal/cmd/new_test.go:235-244` for a value without `=`
- `internal/cmd/new_test.go:262-271` for a stop that makes no output folder
- `internal/cmd/new_test.go:195-203` for the test of `--help`
- `PROPOSAL.md:366-374` for the form of the flag

**Testable:** Yes

**Certainty:** high — repeats the parse of `--option` in `internal/cmd/new.go:53-63` and its tests

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** Task 3

---

### Task 5: Render the items of `api.middleware` and `api.config` in the server and the config of the starter

**Behavior:** `go-event-sourcing-service` declares `api.middleware` and `api.config`. Its server sends each request
to a protected route through the items of `api.middleware`, in their order, and no request to a public route. Its
config has one field for each item of `api.config`. Each file imports the package of its items one time. With no
item, the API accepts each request as before.

**Acceptance Criteria:**
- [x] `starter.yml` declares `api.middleware` with `file: go/internal/api/server/server.go` and `item: [code,
  import]`, and `api.config` with `file: go/internal/api/config.go` and `item: [name, type, import]`. Each
  `summary` gives the contract of an item: for `api.middleware`, an expression of type
  `func(http.Handler) http.Handler` that can use `cfg`; for `api.config`, a field of the config of the API; for
  both, `import` is the path of a package under the module.
- [x] Written with a feature of the test that adds an item to each, `internal/api/server/server.go` applies the
  code of the `api.middleware` item to the protected routes, and `Config` in `internal/api/config.go` has the
  field `<name> <type>` of the `api.config` item. Each file imports `<module>/<import>` of its items one time, also
  when two items have one import.
- [x] The items are in the files that `Write` writes, before a folder command runs. No written file holds the name
  of an extension point.
- [x] In a written repo, a unit test of the starter shows that a request to each protected route goes through the
  middleware that the server gives to the protected routes, and that a request to a public route, such as
  `GET /`, does not.
- [x] With no item, the written files hold no import and no field from an item, the protected routes accept a
  request with no `Authorization` header, and the test at `catalog/catalog_test.go:354-366` passes.
- [x] A repo written from the starter alone builds, `gofmt -l .` lists no file, and its `task test:unit` passes.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — the two extension points
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl` — the items of
  `api.middleware` on the protected routes, and their imports
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl` — the protected
  routes go through the middleware that the server gives them
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl` — the items of `api.config`,
  and their imports
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/` — a unit test of the routes of the starter
- `catalog/catalog_test.go` — the starter with a feature of the test, and the starter alone

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl:31-41` for the router
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:19-38` for the
  public and protected routes
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl:1-8` for the config
- `catalog/features/cli/files/go/internal/cli/root.go.tmpl:29-35` for items in Go code
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/request_logger_test.go.tmpl:25-60`
  for a test of a middleware on a router
- `catalog/catalog_test.go:238-248` for the starter written without its folder commands
- `catalog/catalog_test.go:354-366` for the words that the starter must not hold
- `PROPOSAL.md:117-130` and `PROPOSAL.md:310-326` for the two extension points and their render
- `task:1`, `task:2` for `items` and `unique`

**Testable:** Yes

**Certainty:** medium — `catalog/features/cli/files/go/internal/cli/root.go.tmpl:29-35` renders items in Go code and
`register.go.tmpl:19-38` splits the routes, but no file of the catalog renders items into an import block or a
struct, and no starter puts a middleware on its protected routes alone

**Blast radius:** high — authentication: this decides which routes the check of `api-auth` covers, so a mistake
leaves a protected route open in each repo with `api-auth`, or puts the public route behind the check

**Verification:** `task test:unit` passes. Write the starter alone into a scratch folder and run `go mod tidy`,
`gofmt -l .`, `go vet ./...` and `task test:unit` there. Run the full run of the starter in
`e2e/tests/new.test.ts` on the host (`tasks/learnings.md:59-64`).

**Depends on:** Task 1, Task 2

---

### Task 6: Add the feature `api-auth`, whose method `api-key` lets only a request with a known key reach a protected route

**Behavior:** The catalog in the binary holds `api-auth`. It has the option `method` with the one value `api-key`,
and it adds its middleware to `api.middleware` and its config to `api.config`. In a written repo, a request to a
protected route reaches the handler only with `Authorization: Bearer <key>` for a key whose SHA-256 hash is in the
config of the API. Each other request gets 401 with `WWW-Authenticate: Bearer`.

**Acceptance Criteria:**
- [x] `catalog/features/api-auth/feature.yml` has a `summary` that says what the repo gets; `targets` with only
  `go-event-sourcing-service`; the option `method` with `values: [api-key]` and `default: api-key`; `files` with
  the one entry `files/common/`; and the `add_to` of section 7 of `PROPOSAL.md`: `auth.Middleware(cfg.Auth)` with
  the import `internal/auth` to `api.middleware`, and the field `Auth` of type `auth.Config` with the import
  `internal/auth` to `api.config`. It names no language.
- [x] Written with `go-event-sourcing-service`, the name `my-service` and `api-auth`, the repo has the files of
  `api-auth` under `internal/auth/` and `docs/api-auth.md`. Each file of the starter has the same contents as
  without `api-auth`, except `internal/api/server/server.go` and `internal/api/config.go`, which hold the items of
  `api-auth` and the import `github.com/hpcsc/my-service/internal/auth`. With `module=github.com/acme/ledger`, the
  import is `github.com/acme/ledger/internal/auth`, and no file holds `github.com/hpcsc/my-service`. With the value
  `api-key` given for `method`, the files are the same as with no value.
- [x] No Go file of `api-auth` imports a package of the module other than the packages under `internal/auth`
  (rule 6).
- [x] In the written repo, the unit tests of `internal/auth` run in `task test:unit`, and they show on a handler
  of the test:
  - each request that does not carry `Authorization: Bearer <key>` for a key whose SHA-256 hash is in the config
    gets 401 with `WWW-Authenticate: Bearer`, and the handler does not run: for example no `Authorization` header,
    a scheme other than `Bearer`, `Bearer` and no key, or a key whose hash is not in the config;
  - a request with `Authorization: Bearer <key>` for a key whose hash is in the config gets to the handler, and the
    answer is the answer of the handler; the scheme matches in any case;
  - with no hash in the config, each request gets 401;
  - no request gets 403.
- [x] The config of the API reads the key hashes from `API_KEY_HASHES`: the hex form of the SHA-256 hash of each
  key, with `,` between them. The API keeps the hashes and never a key, and it compares the hash of the key of a
  request with the key hashes in constant time. An entry of `API_KEY_HASHES` that is not the hex form of a SHA-256
  hash stops the API at start, and the message names `API_KEY_HASHES`.
- [x] No written file holds a key or a key hash that the API accepts. A new set of keys needs a new
  `API_KEY_HASHES` and a restart of the API, and no new render.
- [x] `docs/api-auth.md` tells how to make a key and its hash, where the key hashes go, how to change the keys
  without a new render, how a client sends a key, and which answers the API gives. It names no task, file or
  setting that the written repo does not have.
- [x] The end-to-end test of an unknown feature (`e2e/tests/new.test.ts:55-63`) expects the features `api-auth, cli`.
- [x] In this repo, `go build ./...`, `go vet ./...` and `task test:unit` pass with `api-auth` in the catalog, and
  `git status --ignored catalog/` shows no file of `api-auth` as ignored.

**Affected Files/Modules:**
- `catalog/features/api-auth/feature.yml` — new
- `catalog/features/api-auth/files/common/go/internal/auth/` — the middleware, the config and their unit tests;
  each `.go` file has the `.tmpl` suffix
- `catalog/features/api-auth/files/common/go/docs/api-auth.md` — new
- `catalog/catalog_test.go` — the starter with `api-auth`, written without its folder commands
- `e2e/tests/new.test.ts` — the list of the features that target the starter

**Patterns to Follow:**
- `tasks/learnings.md:10-15` for `.tmpl` and `#!`
- `catalog/features/cli/feature.yml` for a `feature.yml`
- `catalog/catalog_test.go:368-404` for the files that a feature adds, and the files of the starter that stay the
  same
- `catalog/catalog_test.go:426-438` for the module in each import
- `catalog/catalog_test.go:468-476` for docs that name nothing that the repo lacks
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/request_logger_test.go.tmpl:25-60`
  for a test of a middleware with `httptest`
- `e2e/tests/new.test.ts:55-63` for the list of the features
- `PROPOSAL.md:154-177` for the `feature.yml` of `api-auth`, and `PROPOSAL.md:488-500` for its contract
- `task:3` for the option `method`, and `task:5` for the extension points of the starter

**Testable:** Yes

**Certainty:** low — no precedent: neither this repo nor repo-templates checks an API key on a server, so the form
of the key hashes, their setting and the answers are decided here

**Blast radius:** high — authentication and credential handling: this is the only check of the API of a repo, and
it handles the keys of the requests and the key hashes of the config

**Verification:** `task test:unit` passes. Until Task 7 checks it, write the starter with `api-auth` into a scratch
folder and run `go mod tidy`, `gofmt -l .`, `go build ./...` and `task test:unit` there. Run the end-to-end test
of an unknown feature on the host (`tasks/learnings.md:59-64`).

**Depends on:** Task 3, Task 5

---

### Task 7: Check end to end that a repo with `api-auth` refuses a request without a known key, and that repos with and without it pass their tests

**Behavior:** An end-to-end test makes a repo with `api-auth`, builds it, runs its unit tests, starts its API, and
sends requests with no key, an unknown key and a known key. The full run of the starter alone also starts its API
and sees that a protected route accepts a request with no key. A third test shows that a value that `method` does
not have stops the real binary.

**Acceptance Criteria:**
- [ ] An end-to-end test runs `feature-compose new go-event-sourcing-service my-service --feature api-auth
  --no-prompt` in an empty scratch folder, and the run exits with 0.
- [ ] In that repo, `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits with 0.
  The packages that pass include `internal/auth` and the packages of the service that have tests.
- [ ] The test starts the API of that repo on a free port, with DB settings that point to no database and
  `API_KEY_HASHES` set to the hash of a key of the test. `GET /` with no `Authorization` header gets 200.
  `POST /accounts` with the body `{}` gets 401 with `WWW-Authenticate: Bearer` with no `Authorization` header and
  with an unknown key, and gets the answer 400 of the handler with `Authorization: Bearer <the key of the test>`.
  The API stops when the test ends.
- [ ] The full run of the starter alone also starts its API, and `POST /accounts` with the body `{}` and no
  `Authorization` header gets the answer 400 of the handler.
- [ ] `feature-compose new go-event-sourcing-service x --feature api-auth:method=oauth` exits with a status other
  than 0, its output names `method` and `api-key`, and the scratch folder stays empty.
- [ ] Only the full runs have a timeout longer than the 30 seconds of `e2e/vitest.config.ts`.
- [ ] `docs/e2e-tests.md` of this repo describes the full run with `api-auth`: what it checks, how it starts the API,
  what it needs, and about how long it takes.

**Affected Files/Modules:**
- `e2e/tests/new.test.ts` — the full run with `api-auth`, the API of the starter alone, and the stop for a value
  that `method` does not have
- `docs/e2e-tests.md` — the full run with `api-auth`

**Patterns to Follow:**
- `e2e/tests/new.test.ts:31-53` and `e2e/tests/new.test.ts:65-89` for the full runs of `new`
- `e2e/tests/new.test.ts:91-114` for a process that the test starts, waits on and stops
- `e2e/tests/new.test.ts:10-18` for a stop that leaves the scratch folder empty
- `e2e/testUtils.ts:38-57` for `runProgram` and `runCli`
- `tasks/learnings.md:38-43` for `MISE_TRUSTED_CONFIG_PATHS`
- `tasks/learnings.md:52-57` for the packages that passed
- `tasks/learnings.md:59-64` for a run on the host
- `docs/e2e-tests.md:80-108` for the description of the full runs

**Testable:** Yes

**Certainty:** medium — follows the full run of `e2e/tests/new.test.ts:65-89` and the process that
`e2e/tests/new.test.ts:91-114` starts, but no test runs a program of a written repo as a server and sends it
requests

**Blast radius:** low — none of the named categories; it adds tests and docs, and changes no code of a repo

**Verification:** On the host: `go build -o <bin> ./cmd/feature-compose`, then
`EXECUTABLE=<bin> BUILD_TAG=dev npx vitest run tests/new.test.ts` in `e2e/` passes.

**Depends on:** Task 4, Task 6

## Summary

**Total:** 7 tasks.

**Order.** Dependency first, then risk. Tasks 1 to 4 build the rules of the catalog on catalogs in memory: the
extension points of a starter and `add_to` (Task 1), `unique` (Task 2), the options of a feature (Task 3) and their
flag (Task 4). Tasks 1 and 3 do not depend on each other. Task 5 changes the starter, and Task 6 adds `api-auth`.
These two are the changes of the story with a high blast radius, so each has its own review: Task 5 decides which
routes the check covers, and Task 6 decides the check. Task 7 proves the full run.

**Coverage of the story:**

| Criterion of US-003 | Tasks |
| --- | --- |
| No header or an unknown key gets 401 with `WWW-Authenticate: Bearer`; a valid key gets to the handler; 403 only for a valid key with no permission | 6 (the middleware and its tests), 5 (the protected routes go through it), 7 (the real API) |
| The API keeps a SHA-256 hash of each key and compares in constant time; the hashes come from the config, so the keys change without a new render | 6, 7 (`API_KEY_HASHES` at the start of the real API) |
| `method` with the default `api-key`; `--feature api-auth:method=<value>`; a value that the option does not have stops, and the message names the option and its values | 3 (the options of a feature), 4 (the flag), 6 (the option of `api-auth`), 7 (the real binary) |
| The server and the config render the items of `api-auth`; no markers; no step other than the folder commands changes a file | 1 (`add_to` and `items`), 2 (`unique`), 5 (the render in the starter, at `Write`, with no marker), 6 (the items of `api-auth`) |
| A repo with `api-auth` and a repo without it build and pass their tests | 5 and 6 (by hand), 7 |
| Context: 401 replaces the 403 of the starters | 6 |
| Context: the code of `api-auth` uses no package of the starter (rule 6) | 6 |
| Decision of the user: a test where a feature adds an item to `cli.commands`, and the root command of the real `cli` shows it | 1 |

No criterion of the story is deferred.

**Criteria that this breakdown adds, for the caller to overrule:**

- A missing or extra field of an item, an `add_to` to an extension point that nobody in the selection declares, and
  two declarers of one extension point stop `new` before the first file (Task 1, decisions 5 and 6).
- The form and the stops of `--feature <name>:<option>=<value>,...` (Task 4, decision 8).
- A unit test in each written repo that the protected routes, and only those, go through the middleware (Task 5,
  decision 2).
- `API_KEY_HASHES`, the stop for an entry that is not a SHA-256 hash, 401 for each request when no hash is set,
  the scheme in any case, and no 403 with `api-key` (Task 6, decisions 10 to 12).
- The API of a written repo is started in the end-to-end tests, with and without `api-auth` (Task 7).

**Notes.**

- The build and the tests of a written repo are checked automatically only in Task 7. Tasks 5 and 6 check them by
  hand, so Task 7 can still need a fix in a file of the starter or of `api-auth`.
- The code of `api-auth` is new, not ported from repo-templates, so the brief of the audit does not mark it as
  ported (`tasks/learnings.md:31-36`). The changed files of the starter are `starter.yml`, `server.go`,
  `register.go`, `config.go` and the new test of the routes.
- The Go code of `api-auth` is in `.go.tmpl` files, so the planned design names only the types of the catalog. The
  names `auth.Middleware` and `auth.Config` come from the items of section 7 of `PROPOSAL.md`, and the words of
  that code are in the `words` of the design.
