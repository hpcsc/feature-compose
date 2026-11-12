# Tasks: Add the Process Manager to a Service Repo (US-017)

## Contents

1. Task 1: Declare `catchup.config` and `catchup.consumers`, and show that catchup stops with the cause when a worker cannot start
2. Task 2: Add the feature `process-manager` with the holds of the account
3. Task 3: Record the steps of a payment with the payment aggregate and its command handlers
4. Task 4: Run the consumers of the payment in catchup, with the tables `payments` and `payment_deadlines`
5. Task 5: Serve the payment routes and the webhook routes under `/api`
6. Task 6: Start the fake outside systems with `task run` and with Docker Compose
7. Task 7: Tell how to run a payment: the tasks `api:pay` and `api:get-payment`, the README section and the docs
8. Task 8: Check end to end that a repo with `process-manager` builds, passes its tests and runs its fakes, and that the catalog has one copy of its files

## Story Reference

US-017, "Add the process manager to a service repo", in `user-stories/feature-compose.md:557-589` (Description,
Acceptance Criteria, Context). The design of the catalog is `PROPOSAL.md`: sections 5 to 10 (`starter.yml`,
`feature.yml`, extension points, items, `add_to`, `items` and `unique`, rules 2, 6, 7 and 8) and 16 (the template
`global/go-event-sourcing-service-process-manager` becomes the feature `process-manager`).

The source of the port is the copy of the process manager in the template `go-event-sourcing-service` of
`repo-templates` (`~/repo-templates/templates/go-event-sourcing-service/{{ .ProjectKebab }}/`, read-only), rendered
with `ProcessManager` true and `PushSubscriber` and `PushNATS` false. Its files are the globs of the first `features`
entry of its `scaffold.yml`, and its changes to the shared files are the `{{ if .Scaffold.ProcessManager }}` blocks of
eleven files. That copy has the routes under `/api` and `reaction.NewConflictRetry` in catchup, as the starter has
since US-019. The copy in `global/go-event-sourcing-service-process-manager` is older (routes at the root path, no
`NewConflictRetry`); its hook `hooks/post_scaffold` is the list of the changes to the files of the starter, and it
agrees with the eleven blocks.

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

- A credential that the fake outside systems send to the webhook routes. The starter has no shared token (US-001),
  so the fakes send no `Authorization` header. In a repo with `api-auth` too, `api-auth` refuses the webhooks, and each
  payment ends at its deadlines; the README section of the feature says so (decision 10). A credential for the fakes,
  or a check of the signature of a webhook, is a story of its own.
- The parts of the shared files of the template for `PushSubscriber` and `PushNATS`. They come with `push` (US-021).
  US-021 also adds the first item of `catchup.consumers` that can fail when it starts.
- A run of `task run`, of the migrations against Postgres, or of the integration tests of the written repo in this
  repo or in its CI. Docker does not answer on this machine, and the CI of feature-compose runs only the `test`
  commands of each folder (`task test:unit`). Decision 8 tells how the story proves the criterion of `task run`.
- `emod validate` of `docs/event-model-payment.emod`. The model of the account of the starter already fails it
  (`tasks/learnings.md:334-340`), and the port keeps the payment model as it is.
- A step that renames the migrations of the feature. Their versions come after each migration of the starter
  (decision 7).
- An item of `api.config`, an env setting of `Taskfile.catchup.yml` or of `Taskfile.api.yml`, and an item of `.env.*`.
  The process manager needs none: the defaults of `outside.Config` point to the local fakes and the local API.
- A change to `PROPOSAL.md`. Its section 6 shows `catchup.consumers` with the item `[code, import]` of a
  `consumer.Handler`; this story declares it with `[code]` of `[]worker` (decision 5). The caller can update the
  proposal.
- A second files entry or an option of `process-manager`. It has one set of files and no question.

**Deferred:** none of the criteria of this story.

## Codebase Context

**The starter today.** `catalog/starters/go-event-sourcing-service/starter.yml:17-25` declares two extension points,
`api.middleware` and `api.config`. `internal/api/config.go.tmpl:3-17` and `internal/api/server/server.go.tmpl` render
them: `{{- range unique "import" "<name>" }}` in the import block, and `{{- range items "<name>" }}` where the items
go. With no item, a range renders nothing, so the written file is the same as a file without the range.
`catalog/features/api-auth/feature.yml` is the example of `add_to`. The starter has the migrations
`20261004130000` to `20261005120100`.

**The machinery of the catalog.** An item is a `map[string]string` (`internal/catalog/extension_point.go:10-49`), and
its values go into the file as they are: the catalog does not render an item as a template. A YAML block scalar in
`feature.yml` gives a value with several lines. The template functions are `option`, `items`, `unique`,
`githubRepository`, `kebab`, `snake`, `camel`, `pascal` and `upper` (`internal/catalog/selection.go:257-269`). No
function sets the indent of a text, so an item of several lines that must go into YAML at a depth other than the left
edge has no way in today (open question 3 of the proposal names the same need for CI jobs).

**The shared files that the template changes.** The eleven blocks of the template, and the hook:

- `internal/use_cases/account/account.go`: the field `holds map[uuid.UUID]hold` of `Account`, and a case of `Apply`
  for the four hold events (`account.go.tmpl:26-33`, `account.go.tmpl:111-125`).
- `internal/use_cases/account/events.go`: the four hold events (`events.go.tmpl:8-14`).
- `internal/use_cases/account/balance/projection.go`: `HoldCaptured` in `EventTypes`, and a case of `Handle` that
  takes the amount off the balance (`projection.go.tmpl:26-61`).
- `internal/use_cases/register.go`: `payment.RegisterEvents(reg)`, the registration of the hold handler and of the
  three payment handlers, five imports of the module and `time` (`register.go.tmpl:15-27`).
- `cmd/catchup/main.go`: the field `Outside outside.Config` of `Config`, eight workers (five reactors with
  `NewConflictRetry`, two projectors, one scheduler), and ten imports (`main.go.tmpl:28-30`, `main.go.tmpl:62-77`).
- `internal/api/use_cases/register.go`: the payment handler and the webhook handler in `allRoutes`
  (`register.go.tmpl:46-64`), with three imports.
- `README.md` (a section at the end), `Taskfile.api.yml` (the tasks `pay` and `get-payment`), `Taskfile.yml` (the
  include of `Taskfile.fakeservices.yml`), `.tmuxp.yaml` (a pane), and `docker-compose.local.yml` (four variables of
  `catchup` and the service `fakeservices`).

**The imports of an item.** Most of the Go lines above use more than one package of the module, and one of them uses
`time.Now`. An item has one `import`, the path of one package under the module, so a Go item cannot carry these
lines. Decision 2 settles it.

**The catchup process.** `cmd/catchup/main.go.tmpl:56-110`: `workers` builds a fixed `[]worker`, and `run` starts
each worker under the Postgres lock of its name. When `Start` of a worker returns an error, `run` logs
`<name> failed: <error>`, cancels the other workers, waits for them, and returns true; `main` then exits with 1. So
a worker that cannot start already stops catchup with its cause. `run` takes a `*lock.Postgres`, which needs
Postgres, so no unit test reaches `run` today, and the starter has no test of a `main` package.

**Tests that the feature touches.** `register_test.go.tmpl:45-77` of the starter expects exactly the six routes of
the starter, so a route of a feature makes it fail. `internal/catalog/catalog_test.go:936-945` checks that a repo of
the starter alone has no file and no edit of the process manager, `catalog_test.go:947-958` that it holds no
`token`, `bearer` or `authorization`, and `catalog_test.go:1009-1018` that no written file holds the name of an
extension point. `catalog_test.go:1043-1063` checks which files of the starter `api-auth` changes.
`catalog_test.go:1441-1487` checks that the docs and the Taskfiles name each route under `/api`, for a list of repos.
`e2e/tests/new.test.ts:371-379` lists the features that target the starter, `new.test.ts:381-399` checks each
feature that does not target `go`, and `e2e/tests/show.test.ts:47-66` and `show.test.ts:119-135` read the last
feature, the last extension point and the place of `api.middleware` in the tree of `show`.

**CI.** `Catalog.Combinations` (`internal/catalog/starter.go:145-173`) takes the combinations from the catalog: the
starter alone, each feature alone, each value of an option, and each pair of features that add to one extension
point. So `process-manager` gets its combination with no change to CI. The combinations test
(`internal/catalog/combination_test.go`) runs the `after_render` and the `test` commands of each folder, which are
`go mod tidy`, `gofmt -w .` and `task test:unit`. Nothing in the CI of feature-compose runs Docker, Postgres or the
integration tests of a written repo.

**Words.** A *worker* is a consumer or a scheduler that catchup runs under the lock of its name (the comment of
`main.go.tmpl:56`); the extension point keeps the name `catchup.consumers` of the story. A *hold* is an amount that
an account keeps for one payment until the payment ends. The *outside systems* are the fraud service, the sanctions
service and the payment network; the *fakes* are `cmd/fakeservices` and `internal/fakeservices`. A *protected route*
keeps its meaning of `tasks/us-003-api-keys.design.yaml`. A *connector file* is a file that a feature adds to a package
of the starter, whose function an item calls (decision 2). An item field `text` holds a block of YAML or Markdown.

**Decisions of this breakdown, for the caller to overrule:**

1. **The source copy.** The port takes the files and the blocks of the template `go-event-sourcing-service`
   (`/api`, `NewConflictRetry`), not the older copy of `global/`. It follows the pattern of
   `tasks/learnings.md:23-29`: render the template with sentinel values, `ProcessManager` true, and the push questions
   false, then replace each sentinel with its template expression.

2. **Connector files instead of imports in items** (question 1). The Go lines of the process manager need several
   packages of the module, `time`, and the alias `apiPayment`, and an item carries one `import`. This breakdown
   weighed three ways: a new catalog rule that an `import` holds several paths (with a second rule for the standard library),
   new constructor functions in each package of the feature (new code that the ported tests do not cover), and this
   one. The feature adds a *connector file* to each package of the starter whose function needs its code, and the
   item is one call of the function of that file. The file has the lines of the template block in one function, with
   its own imports, so Go resolves the imports and the ported code does not change. The connector files are
   `internal/use_cases/payment.go` (`registerPaymentEvents`, `registerPaymentHandlers`), `cmd/catchup/payment.go`
   (`paymentWorkers`) and `internal/api/use_cases/payment.go` (`paymentRoutables`). This bends rule 6 in three more
   packages; the hold code of decision 4 bends it in the same way, and the context of the story says that the process
   manager cannot follow rule 6 in full. The four Go extension points of a composition root (`use_cases.events`,
   `use_cases.handlers`, `catchup.consumers`, `api.routes`) therefore have the item `[code]`, and their summary
   names the values that the code can use and says that it can call the functions of a connector file.

3. **The new extension points of the starter** (question 1). Fourteen, each in the file that the template changes,
   declared in `starter.yml` by the task that first adds an item to it:

   | Extension point | File | Item | Contract (its `summary`) | Items of `process-manager` |
   | --- | --- | --- | --- | --- |
   | `account.state` | `go/internal/use_cases/account/account.go` | `[name, type, apply]` | A field of the Account aggregate, and a statement that Apply runs for each event after the events of the account. type can use uuid and the types of the package account. apply can use a, the *Account, and e, the domain.Event, and the methods that a file of the feature in the package account declares. | `holds`, `map[uuid.UUID]hold`, `a.applyHold(e)` |
   | `account.events` | `go/internal/use_cases/account/events.go` | `[event]` | The name of a payload type of the package event of the account, which RegisterEvents registers. | `HoldPlaced`, `HoldRejected`, `HoldReleased`, `HoldCaptured` |
   | `balance.changes` | `go/internal/use_cases/account/balance/projection.go` | `[event, amount]` | An event of the account that changes the balance in account_balances. event is the name of its payload type in the package event of the account, and that type has the field AccountID. amount is an expression of type int64 that the balance changes by, and it can use payload, the *event.<event>. | `HoldCaptured`, `-payload.Amount` |
   | `use_cases.events` | `go/internal/use_cases/register.go` | `[code]` | A statement of RegisterEvents that registers the event types of a domain. It can use reg, the *registry.OfEvents, and the functions that a feature declares in a file of its own in the package use_cases. | `registerPaymentEvents(reg)` |
   | `use_cases.handlers` | `go/internal/use_cases/register.go` | `[code]` | A statement of RegisterHandlers that registers command handlers. It can use bus, the *command.Bus, stream, the event.Stream, and accounts, the *account.Repository, and the functions that a feature declares in a file of its own in the package use_cases. | `registerPaymentHandlers(bus, stream, accounts)` |
   | `catchup.config` | `go/cmd/catchup/main.go` | `[name, type, import]` | A field of the Config of catchup, whose type the config loader reads with its env tags. import is the path of the package of its type under the module. | `Outside`, `outside.Config`, `internal/use_cases/payment/outside` |
   | `catchup.consumers` | `go/cmd/catchup/main.go` | `[code]` | An expression of type []worker: the consumers and the schedulers that catchup runs, each under the lock of its name. It can use pool, cfg, logger, bus, eventStream, checkpointStore and eventSubscription, and the functions that a feature declares in a file of its own in package main. A worker that cannot start returns an error from Start: catchup then stops each worker, logs the name of the worker with the error, and exits with 1. | one call of `paymentWorkers` |
   | `api.routes` | `go/internal/api/use_cases/register.go` | `[code]` | An expression of type []route.Routable, whose routes Register serves under /api as public or protected routes. It can use bus and pool, and the functions that a feature declares in a file of its own in the package use_cases of the API. | `paymentRoutables(bus, pool)` |
   | `taskfile.includes` | `go/Taskfile.yml` | `[name, taskfile]` | A Taskfile that Taskfile.yml includes. name is the namespace of its tasks, and taskfile is its path in the repo. | `fakeservices`, `Taskfile.fakeservices.yml` |
   | `api.tasks` | `go/Taskfile.api.yml` | `[text]` | Tasks of Taskfile.api.yml, as YAML with each task name at the left edge. A task can use the var API. | `pay`, `get-payment` |
   | `tmux.panes` | `go/.tmuxp.yaml` | `[command]` | A command that task run runs in a pane of its tmux window. | `task fakeservices:run` |
   | `compose.catchup.environment` | `go/docker-compose.local.yml` | `[name, value]` | A variable of the environment of the catchup service of docker-compose.local.yml. | `FRAUD_SERVICE_URL`, `SANCTIONS_SERVICE_URL`, `PAYMENT_NETWORK_URL`, `WEBHOOK_URL` |
   | `compose.services` | `go/docker-compose.local.yml` | `[text]` | A service of docker-compose.local.yml, as YAML with the service name at the left edge. | `fakeservices` |
   | `readme.sections` | `go/README.md` | `[text]` | A section of the README after the sections of the starter, as Markdown that starts with a heading of level 2. | `## Process Manager Example` |

   `Dockerfile.fakeservices`, `Taskfile.fakeservices.yml`, the migrations and the docs are files of the feature, not
   changes to files of the starter.

4. **The account aggregate** (question 2). The hold stays in the `Account` aggregate: `HoldPlaced` takes the amount
   off `available`, which `Withdraw` checks, so a hold in another aggregate or another stream changes the rules of the
   example. The feature adds the hold files of the template to the package `account` and its sub-packages
   (`hold.go`, `hold_test.go`, `command/*_hold.go`, `event/hold_*.go`, `holds/`), and its items add the field, the
   statement of `Apply`, the four events and the change of the balance. `apply` runs after the switch of `Apply`, for
   each event; `applyHold` has its own switch on the four hold events, and no case of the starter matches them, so the
   state after each event is the same as with the case of the template. The template tests `hold_test.go`,
   `holds/handler_test.go` and the integration test `process_test.go` are the specification, and the port keeps
   them as they are.

5. **`catchup.consumers` and a worker that cannot start** (question 3). The item is `[code]`, an expression of type
   `[]worker`, and `workers` appends the items to the workers of the starter. A worker fails when it starts by a
   return of an error from `Start`; `run` already stops the other workers, logs `<name> failed: <error>`, and `main`
   exits with 1. A worker that needs an outside service connects to it in `Start` (US-021 follows this for NATS). The
   story proves the path with a unit test of the written repo: `run` takes the lock of each worker through a small
   interface of `cmd/catchup` that `*lock.Postgres` satisfies, and a test of `run` (`cmd/catchup/main_test.go`, tag
   `unit`) gives it a lock of the test, a worker whose `Start` returns an error, and a worker that runs until its
   context ends. A catalog test with a feature of the test shows that an item of `catchup.consumers` is among the
   workers that `main` gives `run`.

6. **Items of YAML and Markdown** (question 1). Items of one line have their own fields (`taskfile.includes`,
   `tmux.panes`, `compose.catchup.environment`), so they need no new function. An item of several lines has the field
   `text`. A section of the README starts at the left edge, so `readme.sections` needs no new function. The tasks of
   `Taskfile.api.yml` and a service of `docker-compose.local.yml` go at a depth of 2 spaces, so the catalog gets the
   template function `indent`: `indent <spaces> <text>` puts the spaces before each line of the text that is not
   empty, and leaves an empty line empty. It is one entry of the FuncMap (`tasks/learnings.md:78-84`). US-022 can use
   it for the jobs of a CI workflow.

7. **The migrations** (question 4). The feature owns `migrations/20261005124000_create-payments-table.{up,down}.sql`
   and `migrations/20261005124100_create-payment-deadlines-table.{up,down}.sql`, the versions of the template. They
   come after `20261005120100`, the last migration of the starter, so golang-migrate applies them after the migrations
   of the starter, with no rename. A catalog test checks that each migration of each feature that targets the starter
   has a version after each migration of the starter, so a later migration of the starter that breaks the order
   fails a test.

8. **The proof of `task run`** (question 5). `task run` runs `run-dependencies` (Postgres, then the `migrate` service,
   which applies each file of `migrations/`), then `tmuxp load .tmuxp.yaml`, whose panes run `task api:run`,
   `task catchup:run` and, with the feature, `task fakeservices:run`. Docker does not answer here, so the proof is in
   three parts. (a) Catalog tests read the written repo: `migrations/` holds the two migrations of decision 7, whose
   `up` files create the tables `payments` and `payment_deadlines`; `Taskfile.yml` includes `Taskfile.fakeservices.yml`
   as `fakeservices`, whose task `run` runs `go run ./cmd/fakeservices`; `.tmuxp.yaml` has the pane
   `task fakeservices:run`; `docker-compose.local.yml` has the service `fakeservices` and the four variables of
   `catchup`. Each YAML file parses, and the test reads these values from the parsed YAML. (b) The e2e test lists the
   tasks of the written repo with `task --list-all`, starts the written fakes, and sends them a fraud check whose
   webhook goes to a server of the test. (c) The migrations and `process_test.go` are the files of the template; `go
   vet -tags integration ./...` compiles the integration tests, and the CI of the written repo runs them against
   Postgres. A run of `task run` on a machine with Docker stays a manual check for the reviewer.

9. **One copy** (question 6). A catalog test walks the catalog folder: each file whose path in its folder is a path of
   the process manager (the paths of `catalog_test.go:939`, plus the three connector files and their tests) is under `catalog/features/process-manager/files/go/`, and no two files of the catalog give the same
   path in the repo. `process-manager` has one files entry, `files/`, and no option. The test of the starter alone
   (`catalog_test.go:936-945`) still finds no file and no edit of the process manager.

10. **The webhook routes.** They stay protected routes, as in the template, so a repo with `api-auth` checks the
    caller of a webhook. The fakes send no credential (the starter has no shared token), so a webhook reaches its
    handler only in a repo without `api-auth`. The README section of the feature says this (Boundaries).

11. **The starter alone.** A repo of the starter alone stays the same, except in three files, each named in the task
    that changes it: `cmd/catchup/main.go` (`workers` collects the workers in a variable for the items of
    `catchup.consumers`, and `run` takes the lock through the interface of decision 5), the new
    `cmd/catchup/main_test.go`, and `internal/api/use_cases/register_test.go` (the test expects the routes of the
    starter among the routes that `Register` serves, so a route of a feature does not fail it). Each other extension
    point leaves no trace with no item: no line, no blank line and no import.

12. **The files that the port takes** (for the audit). The port changes only the module and the name in the
    Go files of `internal/use_cases/payment/` except `process_test.go`, of `internal/use_cases/account/holds/`, of
    `internal/api/use_cases/payment/` and `internal/api/use_cases/webhook/`, the hold files of
    `internal/use_cases/account/`, the four migrations, `Dockerfile.fakeservices`, `docs/event-model-payment.emod`,
    `.md` and `.svg`, and the lines of the eleven blocks, which go into items and connector files. The port changes more in
    `internal/fakeservices/handler.go` and its test, `cmd/fakeservices/main.go` and `Taskfile.fakeservices.yml` (no
    token: decision 10), `process_test.go` (the `Register` of the starter, no token), the tasks `pay` and
    `get-payment` (no `AUTH` header), the service `fakeservices` (no volume of the token),
    `docs/process-manager-example.md` and the README section (the sentence on the bearer token, the connector files, and
    decision 10). The new code is the connector files and their tests, the extension points, `indent`, the test of `run`, and the
    tests of the catalog and of e2e.

## Tasks

### Task 1: Declare `catchup.config` and `catchup.consumers`, and show that catchup stops with the cause when a worker cannot start

**Behavior:** `go-event-sourcing-service` declares `catchup.config` and `catchup.consumers` in `cmd/catchup/main.go`
(decisions 3 and 5). `Config` of catchup has a field for each item of `catchup.config`, and `main.go` imports the
package of each item one time. `workers` appends the workers of each item of `catchup.consumers` to the workers of the
starter, and `run` starts them all. When a worker cannot start, catchup stops each worker, the log names the worker and
the error, and the process exits with 1. A unit test of the written repo shows this path.

**Acceptance Criteria:**
- [x] `starter.yml` declares `catchup.config` with `file: go/cmd/catchup/main.go` and `item: [name, type, import]`, and
  `catchup.consumers` with the same file and `item: [code]`, each with the `summary` of decision 3.
- [x] Written with a feature of the test that adds two items to `catchup.config` with one `import`, and an item to
  `catchup.consumers` whose code calls a function of a file that the feature adds to `cmd/catchup`,
  `cmd/catchup/main.go` has the field `<name> <type>` of each config item in `Config`, imports
  `<module>/<import>` one time, and appends the code of the consumer item to the workers that `main` gives `run`.
- [x] With no item, `Config` has only `DB config.DB`, the import block of `main.go` is the import block of the starter,
  and `workers` gives the projector of the balance and the reactor of the welcome bonus.
- [x] The unit tests of `run` in the written repo (`cmd/catchup/main_test.go`, build tag `unit`) show, with a lock of the
  test: when the `Start` of one worker returns an error, `run` returns that catchup failed, the other worker stops, and
  the log holds a line with the name of the failed worker and its error; when each worker returns from `Start` with no
  error after the context ends, `run` returns that catchup did not fail.
- [x] `run` takes the lock of each worker through an interface of `cmd/catchup` that `*lock.Postgres` satisfies, and
  `main` gives it the Postgres lock of its pool.
- [x] No written file holds the name `catchup.config` or `catchup.consumers`, with or without items
  (`catalog_test.go:1009-1018` names them).
- [x] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — the two extension points
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main.go.tmpl` — the items in `Config`, the imports
  and `workers`, and the lock of `run` through an interface
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main_test.go.tmpl` — new: the tests of `run`
- `internal/catalog/catalog_test.go` — the render with a feature of the test, the render with no item, and the names
  of the extension points

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl:3-17` for the items of a config and
  their imports
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main.go.tmpl:56-110` for `workers` and `run`
- `catalog/starters/go-event-sourcing-service/starter.yml:17-25` for the declaration of an extension point
- `internal/catalog/catalog_test.go:961-1018` for a feature of the test, the render with and without items, and the
  names of the extension points
- `catalog/starters/go-event-sourcing-service/files/go/internal/domain/scheduler/scheduler_test.go.tmpl` for a test
  of a worker that runs until its context ends
- `catalog/starters/go-event-sourcing-service/files/go/internal/common/test/fake_log_handler.go.tmpl:1-35` for a log of
  a test
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof of template code by breaks by hand
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:285-291` for `go vet -tags combinations` and `task test:combinations`
- `tasks/learnings.md:348-354` for the scheduler test that can fail under load

**Testable:** Yes

**Certainty:** medium — `config.go.tmpl:3-17` renders the items of `api.config` and `catalog_test.go:961-1018` tests
them with a feature of the test; but no item of the catalog is a slice that a function appends, and the starter has no
test of a `main` package, so the lock of `run` must go through an interface first

**Blast radius:** low — none of the named categories; it changes how catchup takes the lock of a worker, not who can
reach the API or what a worker writes

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the starter alone with the built
binary into a scratch folder and run `go test -tags unit ./cmd/catchup/` there. Apply one break at a time to the
written `main.go`, read its diff, run the test, and put the file back: `run` returns false after a failed worker;
`run` does not cancel the other workers after a failure; the log line of a failure leaves out the error. Each break
makes a test fail. Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** None

---

### Task 2: Add the feature `process-manager` with the holds of the account

**Behavior:** The catalog holds the feature `process-manager`, which targets only `go-event-sourcing-service`. In this
task it adds the holds of the account (decision 4): the account places a hold for a payment when its available amount
covers it, and records a rejection when it does not; it releases and captures a hold; a captured hold takes the amount
off the balance of the read model. The starter declares `account.state`, `account.events`, `balance.changes` and
`use_cases.handlers`, and the feature adds its items to them and the connector file `internal/use_cases/payment.go`
(decision 2). `feature-compose new go my-tool --feature process-manager` stops, and the message names the feature and
the starter.

**Acceptance Criteria:**
- [x] `catalog/features/process-manager/feature.yml` has a `summary` that says what the repo gets, `targets` with only
  `go-event-sourcing-service`, the one files entry `files/`, no option and no need, and the items of decision 3 for
  `account.state`, `account.events`, `balance.changes` and `use_cases.handlers`.
- [x] `starter.yml` declares the four extension points of decision 3, each with its file, its item fields and its
  `summary`.
- [x] Written with `process-manager`, the repo has the hold files of decision 4, from the template with the module
  and the name of the repo, and the connector file `internal/use_cases/payment.go`. `Account` has the field `holds`,
  `Apply` runs `a.applyHold(e)` after its switch, `account.RegisterEvents` registers the four hold events, the balance
  projection lists `HoldCaptured` in `EventTypes` and changes the balance by `-payload.Amount` for it in `Handle`, and
  `RegisterHandlers` registers the hold handler.
- [x] Written from the starter alone, the four files hold no line, no blank line and no import from an item, and the
  test at `catalog_test.go:936-945` passes, with its check of `fakeservices` and `HoldPlaced`.
- [x] With `process-manager`, each file of the starter that differs from the repo of the starter alone is the `file` of
  an extension point that `process-manager` adds items to, and each other file that differs is a file of
  `process-manager`. The test takes the files of the extension points from the definitions of the catalog, so it holds
  in each later task.
- [x] In the written repo, the unit tests of `internal/use_cases/account` (with `hold_test.go`) and of
  `internal/use_cases/account/holds` run in `task test:unit` and pass.
- [x] A unit test of the connector file in the written repo (`internal/use_cases/payment_test.go`, tag `unit`) shows,
  with the memory stream of the starter, that the bus that `use_cases.RegisterHandlers` fills takes the hold commands:
  a hold for an open account with money records `HoldPlaced` on the stream of the account, and its capture records
  `HoldCaptured`.
- [x] `feature-compose check --catalog catalog` finds no mistake.
- [x] `e2e/tests/new.test.ts:371-379` expects `api-auth, cli, cli-auth, process-manager`, and the test at
  `new.test.ts:381-399` expects `process-manager` among the features that do not target `go`. The tests of `show` in a
  terminal (`e2e/tests/show.test.ts:47-66`, `show.test.ts:119-135`) expect the new last feature, the new last extension
  point and the new place of `api.middleware` in the tree.
- [x] No written file holds the name of one of the four extension points (`catalog_test.go:1009-1018`).
- [x] With each combination, `process-manager` among them, the written repo builds, `gofmt -l .` lists no file, and its
  unit tests pass (`task test:combinations`).
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/features/process-manager/feature.yml` — new
- `catalog/features/process-manager/files/go/internal/use_cases/account/` — new: `hold.go.tmpl`, `hold_test.go.tmpl`,
  the hold commands, the hold events, and `holds/`
- `catalog/features/process-manager/files/go/internal/use_cases/payment.go.tmpl` — new: the connector file with
  `registerPaymentHandlers`
- `catalog/features/process-manager/files/go/internal/use_cases/payment_test.go.tmpl` — new: the test of the connector
  file
- `catalog/starters/go-event-sourcing-service/starter.yml` — four extension points
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/account/account.go.tmpl` — `account.state`
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/account/events.go.tmpl` — `account.events`
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/account/balance/projection.go.tmpl` —
  `balance.changes` in `EventTypes` and in `Handle`
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/register.go.tmpl` — `use_cases.handlers`
- `internal/catalog/catalog_test.go` — the repo with `process-manager`, the starter alone, and the changed files
- `e2e/tests/new.test.ts`, `e2e/tests/show.test.ts` — the new feature and the new extension points

**Patterns to Follow:**
- `tasks/learnings.md:23-29` for the port with sentinel values, and `tasks/learnings.md:10-15` for `.tmpl`
- `catalog/features/api-auth/feature.yml` for a `feature.yml` with `add_to`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl:10-17` for items in a struct
- `catalog/features/cli/files/go/internal/cli/root.go.tmpl:29-35` for items in Go code
- `internal/catalog/catalog_test.go:1021-1063` for the files that a feature adds and the files of the starter that it
  changes
- `internal/catalog/catalog_test.go:936-945` for the starter alone with no file of the process manager
- `internal/catalog/catalog_test.go:793-798` for the check of the catalog of this repo
- `e2e/tests/new.test.ts:371-399` for the lists of features, and `e2e/tests/show.test.ts:47-66` and
  `show.test.ts:119-135` for the tree of `show`
- `tasks/learnings.md:341-347` for the e2e tests of `show` that read places in the tree
- `tasks/learnings.md:30-36` for the files that the port takes (decision 12)
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand
- `tasks/learnings.md:58-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** medium — `api-auth` (`catalog/features/api-auth/feature.yml`, `catalog_test.go:1021-1063`) is a feature
with Go code and items, and `config.go.tmpl:10-17` renders items in a struct; but no feature adds files to a package of
the starter, no item goes after a switch, and `balance.changes` renders one item in two places of one file

**Blast radius:** high — balances: the hold changes the available amount that a withdrawal checks, and
`balance.changes` changes the balance of the read model

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the repo with
`--feature process-manager` with the built binary into a scratch folder, and run `gofmt -l .`, `go build ./...` and
`go test -tags unit ./internal/use_cases/account/...` there. Apply one break at a time to the written files, read its
diff, run the tests, and put the file back: leave `a.applyHold(e)` out of `Apply` with `if false`; register
`HoldPlaced` with no `HoldCaptured`; let a hold of an amount equal to the available amount fail; leave the hold handler
out of `registerPaymentHandlers`. Each break makes a test fail. The change of the balance in `Handle` has only the
integration tests of the written repo; the catalog test reads its text. Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** None

---

### Task 3: Record the steps of a payment with the payment aggregate and its command handlers

**Behavior:** In a repo with `process-manager`, the command bus takes the commands of a payment: it records a request,
the funding, the results of the fraud check and of the sanctions screening, the submission, the settlement, the
return, and the timeouts, and the payment aggregate decides each step. The registry knows the payment events. The
starter declares `use_cases.events`, and the connector file registers the payment events and the three payment handlers.

**Acceptance Criteria:**
- [x] `starter.yml` declares `use_cases.events` of decision 3, and `process-manager` adds `registerPaymentEvents(reg)`
  to it.
- [x] Written with `process-manager`, the repo has the payment aggregate, its events, its commands, its repository, and
  the handlers `request`, `record` and `timeout` of the template under `internal/use_cases/payment/`, except the
  packages of Task 4 and `process_test.go`. `internal/use_cases/payment.go` registers the payment events and the
  three handlers with the default timeouts and `time.Now`, as the template block does.
- [x] Written from the starter alone, `internal/use_cases/register.go` holds no line, no blank line and no import from
  an item.
- [x] In the written repo, the unit tests of `internal/use_cases/payment` and of its packages `request`, `record` and
  `timeout` run in `task test:unit` and pass.
- [x] The test of the connector file (`internal/use_cases/payment_test.go`) also shows, with the memory stream and the
  registry that `use_cases.RegisterEvents` fills, that the bus takes the commands of the three payment handlers: a
  request for an open account with money records `PaymentRequested`, the record of its funding records
  `PaymentFunded`, and a timeout of its checks reaches the handler of the timeouts.
- [x] No written file holds the name `use_cases.events`, and `feature-compose check --catalog catalog` finds no
  mistake.
- [x] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/features/process-manager/files/go/internal/use_cases/payment/` — new: the aggregate, the events, the
  commands, the repository, `request/`, `record/` and `timeout/`, with their tests
- `catalog/features/process-manager/files/go/internal/use_cases/payment.go.tmpl` — `registerPaymentEvents`, and the
  payment handlers in `registerPaymentHandlers`
- `catalog/features/process-manager/files/go/internal/use_cases/payment_test.go.tmpl` — the commands of the payment
- `catalog/features/process-manager/feature.yml` — the item of `use_cases.events`
- `catalog/starters/go-event-sourcing-service/starter.yml` — `use_cases.events`
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/register.go.tmpl` — `use_cases.events`
- `internal/catalog/catalog_test.go` — the registration in the written repo, and the starter alone

**Patterns to Follow:**
- `task:2` for the port, the connector file, the items and the tests of the written repo
- `catalog/starters/go-event-sourcing-service/files/go/internal/use_cases/register.go.tmpl:15-27` for the registrations
- `tasks/learnings.md:23-29` for the port with sentinel values
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand

**Testable:** Yes

**Certainty:** high — Task 2 sets the port, the connector file and the items of `register.go.tmpl`, and the code and the
tests of the payment are the files of the template

**Blast radius:** high — money: the payment aggregate decides whether a payment to another bank goes on, fails, is
refused, or settles

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the repo with
`--feature process-manager` into a scratch folder, and run `go build ./...` and
`go test -tags unit ./internal/use_cases/...` there. Apply one break at a time to the written `payment.go` and
`internal/use_cases/payment.go`, read its diff, run the tests, and put the file back: let a payment settle twice; refuse
no payment after a failed fraud check; leave the record handler out of `registerPaymentHandlers`; leave
`payment.RegisterEvents` out of `registerPaymentEvents`. Each break makes a test fail. Run `task test:combinations` and
the e2e tests of `new` and `show` on the host.

**Depends on:** Task 2

---

### Task 4: Run the consumers of the payment in catchup, with the tables `payments` and `payment_deadlines`

**Behavior:** In a repo with `process-manager`, catchup runs the eight workers of the payment: the funding reaction and
the holds reaction, the gateways of the checks, of the submission and of the status, which call the outside systems
with the outside client, the projections of the summary and of the deadlines, and the scheduler of the deadlines.
`Config` of catchup reads the URLs of the outside systems. The repo has the two migrations of the feature, which create
the tables `payments` and `payment_deadlines` after the migrations of the starter (decision 7).

**Acceptance Criteria:**
- [x] `process-manager` adds `Outside outside.Config` with the import `internal/use_cases/payment/outside` to
  `catchup.config`, and one call of `paymentWorkers` to `catchup.consumers`. The connector file `cmd/catchup/payment.go`
  gives the eight workers of the template block, in its order, with `NewConflictRetry` and `StartAtBeginning` as there.
- [x] Written with `process-manager`, the repo has the packages `funding`, `holds`, `checks`, `submission`, `status`,
  `outside`, `summary` and `deadlines` of the template under `internal/use_cases/payment/`, and the two migrations of
  decision 7 with their `up` and `down` files.
- [x] The `up` migration `20261005124000_create-payments-table` creates the table `payments`, and
  `20261005124100_create-payment-deadlines-table` creates `payment_deadlines`; each `down` migration drops its table.
- [x] For each feature of the catalog that targets `go-event-sourcing-service`, each migration that it adds has a
  version after each migration of the starter (a catalog test).
- [x] In the written repo, the unit tests of `funding`, `holds`, `checks`, `submission`, `status` and `outside` run in
  `task test:unit` and pass.
- [x] A unit test of the connector file in the written repo (`cmd/catchup/payment_test.go`, tag `unit`) shows that the
  workers of catchup, with a pool that points to no database, have the two names of the workers of the starter and the
  eight names of the workers of the payment of the template, each one time. A name is the key of the checkpoint and of
  the lock of a worker, so two workers with one name is a mistake.
- [x] `feature-compose check --catalog catalog` finds no mistake.
- [x] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/features/process-manager/files/go/internal/use_cases/payment/` — new: `funding/`, `holds/`, `checks/`,
  `submission/`, `status/`, `outside/`, `summary/`, `deadlines/`
- `catalog/features/process-manager/files/go/cmd/catchup/payment.go.tmpl` — new: the connector file with `paymentWorkers`
- `catalog/features/process-manager/files/go/cmd/catchup/payment_test.go.tmpl` — new: the names of the workers
- `catalog/features/process-manager/files/go/migrations/` — new: the four migration files
- `catalog/features/process-manager/feature.yml` — the items of `catchup.config` and `catchup.consumers`
- `internal/catalog/catalog_test.go` — the workers and the config of catchup, the migrations, and their versions

**Patterns to Follow:**
- `task:1` for `catchup.config` and `catchup.consumers`
- `task:2` for the port, the connector file and the tests of the written repo
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main.go.tmpl:62-77` for the workers of the starter
- `catalog/starters/go-event-sourcing-service/files/go/migrations/20261005120100_add-schema-version-to-events.up.sql`
  for a migration of the starter
- `tasks/learnings.md:23-29` for the port with sentinel values
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand
- `tasks/learnings.md:348-354` for the scheduler test that can fail under load

**Testable:** Yes

**Certainty:** medium — Task 1 and Task 2 are the precedent for the items and the port; but no feature of the catalog
adds a migration, so this task checks the order of the versions against the starter for the first time

**Blast radius:** high — a schema migration, and the gateways send the payments to the outside systems

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the repo with
`--feature process-manager` into a scratch folder, and run `go build ./...`, `go vet -tags integration ./...` and
`go test -tags unit ./internal/use_cases/payment/...` there. Apply one break at a time to the written files, read its
diff, run the tests, and put the file back: let the funding reaction place no hold; let the checks gateway send no
key of the request; leave the scheduler of the deadlines out of `paymentWorkers`; give a migration of the feature the
version `20261004120000` in the catalog. Each break makes a test fail. The projections of the summary and of the
deadlines have only the integration tests of the written repo. Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** Task 1, Task 3

---

### Task 5: Serve the payment routes and the webhook routes under `/api`

**Behavior:** The API of a repo with `process-manager` serves `POST /api/payments` and `GET /api/payments/{id}`, and the
three webhook routes under `/api/webhooks/`, as protected routes (decision 10). The starter declares `api.routes`, and
the connector file `internal/api/use_cases/payment.go` gives the payment handler and the webhook handler. The test of
`Register` of the starter expects the routes of the starter among the routes that `Register` serves, so it passes with
the routes of a feature (decision 11).

**Acceptance Criteria:**
- [ ] `starter.yml` declares `api.routes` of decision 3, and `process-manager` adds `paymentRoutables(bus, pool)` to it.
  `allRoutes` appends the routables of each item to the routables of the starter.
- [ ] Written with `process-manager`, the repo has the packages `payment` and `webhook` of the template under
  `internal/api/use_cases/`, and the connector file `internal/api/use_cases/payment.go`.
- [ ] The test of `Register` in the written repo (`register_test.go`) shows, for the starter alone and with the routes
  of a feature: a request to `GET /api/` reaches its handler, and a request to each of the five account routes goes
  through the middleware of the protected routes; each route that `Register`
  serves has a path under `/api/`; and a request to the path of each route without `/api` gets 404.
- [ ] A unit test of the connector file in the written repo (`internal/api/use_cases/payment_test.go`, tag `unit`) shows
  that `Register` serves the five routes of the process manager under `/api`, and that a request to each goes through
  the middleware of the protected routes.
- [ ] In the written repo, the unit tests of `internal/api/use_cases/payment` and `internal/api/use_cases/webhook` run
  in `task test:unit` and pass.
- [ ] Written from the starter alone, `internal/api/use_cases/register.go` holds no line, no blank line and no import
  from an item, and no written file holds the name `api.routes`.
- [ ] `feature-compose check --catalog catalog` finds no mistake.
- [ ] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — `api.routes`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl` — the items in
  `allRoutes`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl` — the routes of
  the starter among the served routes
- `catalog/features/process-manager/files/go/internal/api/use_cases/` — new: `payment/`, `webhook/`, the connector file
  `payment.go.tmpl` and its test `payment_test.go.tmpl`
- `catalog/features/process-manager/feature.yml` — the item of `api.routes`
- `internal/catalog/catalog_test.go` — the routes in the written repo, and the starter alone

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:46-64` for `allRoutes`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl:19-77` for the walk
  of the routes and a middleware that refuses
- `task:2` for the port and the connector file
- `tasks/learnings.md:23-29` for the port with sentinel values
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand
- `tasks/learnings.md:320-326` for the headers of an HTTP answer in a test

**Testable:** Yes

**Certainty:** medium — `register.go.tmpl:46-64` and its test hold the public and the protected routes, and US-019 put
them under `/api`; but no item adds routes, and the test of the starter must stop to expect a fixed list of routes

**Blast radius:** high — authorization: the webhook routes must stay behind the middleware of the protected routes;
also a contract outside this repository, the routes of the API of each written repo

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the starter alone and the repo with
`--feature process-manager` into scratch folders, and run `go test -tags unit ./internal/api/...` in each. Apply one
break at a time to the written files, read its diff, run the tests, and put the file back: make one webhook route
public; serve the routes of the item at the root path; leave the item out of `allRoutes`; drop a route of the starter.
Each break makes a test fail. Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** Task 4

---

### Task 6: Start the fake outside systems with `task run` and with Docker Compose

**Behavior:** A repo with `process-manager` has the fakes of the three outside systems, which answer a request and
then call the webhook of the request, and which send no credential (decision 10). `Taskfile.yml` includes
`Taskfile.fakeservices.yml` as `fakeservices`, and `.tmuxp.yaml` runs `task fakeservices:run` in a pane, so `task run`
starts the fakes beside the API and catchup. `docker-compose.local.yml` has the service `fakeservices` from
`Dockerfile.fakeservices`, and gives `catchup` the URLs of the fakes and of the webhooks of the API. The catalog gets
the template function `indent` (decision 6). The integration test of the template, `process_test.go`, is in the repo
and compiles.

**Acceptance Criteria:**
- [ ] The template function `indent <spaces> <text>` puts the spaces before each line of the text that is not empty,
  and leaves an empty line empty. Unit tests of the render show it with a text of one line, of several lines, with an
  empty line, and with no text, also with the delimiters `[[` and `]]`.
- [ ] `starter.yml` declares `taskfile.includes`, `tmux.panes`, `compose.catchup.environment` and `compose.services` of
  decision 3, and `process-manager` adds the items of decision 3 to them.
- [ ] Written with `process-manager`, the repo has `cmd/fakeservices/main.go`, `internal/fakeservices/handler.go` and
  its test, `Taskfile.fakeservices.yml` and `Dockerfile.fakeservices`. No file of the fakes reads, holds or sends a
  token, and the fakes send each webhook with the key of its request.
- [ ] A catalog test parses the written `Taskfile.yml`, `Taskfile.fakeservices.yml`, `.tmuxp.yaml` and
  `docker-compose.local.yml` as YAML, and finds: the include `fakeservices: Taskfile.fakeservices.yml` after the
  includes of the starter; a task `run` of `Taskfile.fakeservices.yml` that runs `go run ./cmd/fakeservices`; the panes
  `task api:run`, `task catchup:run` and `task fakeservices:run`; the four variables of `catchup` with the values of
  the template, `WEBHOOK_URL` under `/api`, beside `<<: *x-db-config`; and a service `fakeservices` whose build uses
  `Dockerfile.fakeservices`.
- [ ] Written from the starter alone, the four files hold no line and no blank line from an item, and they parse with
  the values of the starter.
- [ ] Written with `process-manager`, the repo has `internal/use_cases/payment/process_test.go`, which calls the
  `Register` of the starter, and `go vet -tags integration ./...` passes in the repo.
- [ ] In the written repo, the unit tests of `internal/fakeservices` run in `task test:unit` and pass.
- [ ] No written file holds the name of one of the four extension points, and `feature-compose check --catalog catalog`
  finds no mistake.
- [ ] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `internal/catalog/selection.go` — `indent` in the FuncMap of `render`
- `internal/catalog/selection_test.go` — the render of `indent`
- `catalog/starters/go-event-sourcing-service/starter.yml` — four extension points
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml`, `.tmuxp.yaml`, `docker-compose.local.yml` — the
  items
- `catalog/features/process-manager/files/go/cmd/fakeservices/`, `internal/fakeservices/` — new: the fakes with no token
- `catalog/features/process-manager/files/go/Taskfile.fakeservices.yml`, `Dockerfile.fakeservices` — new
- `catalog/features/process-manager/files/go/internal/use_cases/payment/process_test.go.tmpl` — new
- `catalog/features/process-manager/feature.yml` — the items
- `internal/catalog/catalog_test.go` — the parsed YAML of the repo with `process-manager` and of the starter alone

**Patterns to Follow:**
- `internal/catalog/selection.go:257-269` for the FuncMap of the templates
- `internal/catalog/selection_test.go:103-130` and `internal/catalog/selection_test.go:587-645` for the tests of a
  template function
- `tasks/learnings.md:78-84` for a new template function
- `tasks/learnings.md:16-22` for the delimiters of the Taskfiles
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml:3-7`, `.tmuxp.yaml:1-8` and
  `docker-compose.local.yml:22-29` for the places of the items
- `internal/catalog/catalog_test.go:907-913` for the Taskfiles that keep the syntax of Task
- `task:2` for the port and the tests of the written repo
- `tasks/learnings.md:23-29` for the port with sentinel values, and `tasks/learnings.md:30-36` for the files that are
  ported with changes (decision 12)
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand

**Testable:** Yes

**Certainty:** medium — a template function is one FuncMap entry (`selection.go:257-269`, `tasks/learnings.md:78-84`)
and the items of Go files are the precedent; but no item of the catalog goes into a YAML file yet, and a block of
several lines must keep the YAML valid with and without items

**Blast radius:** low — none of the named categories; the fakes stand in for the outside systems on the machine of a
developer, and the files are local run config

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, and `clerk mutants` covers `indent`.
Write the starter alone and the repo with `--feature process-manager` into scratch folders. In each, run
`task --list-all`, and in the repo with the feature run `go vet -tags integration ./...` and
`go test -tags unit ./internal/fakeservices/`. Apply one break at a time to the written fakes, read its diff, run the
tests, and put the file back: let a fraud check pass above the limit; send the webhook with no key. Each break makes a
test fail. Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** Task 5

---

### Task 7: Tell how to run a payment: the tasks `api:pay` and `api:get-payment`, the README section and the docs

**Behavior:** A repo with `process-manager` has the tasks `api:pay` and `api:get-payment`, which call
`POST /api/payments` and `GET /api/payments/{id}`. Its README ends with the section "Process Manager Example", which
tells how to run a payment, the paths that the amount and the IBAN select, the files to change for another process,
and that the fakes send no credential (decision 10). The repo has `docs/process-manager-example.md` and the payment
event model.

**Acceptance Criteria:**
- [ ] `starter.yml` declares `api.tasks` and `readme.sections` of decision 3, and `process-manager` adds the tasks
  `pay` and `get-payment` and the README section to them.
- [ ] A catalog test parses the written `Taskfile.api.yml` and finds the tasks of the starter and the tasks `pay` and
  `get-payment`, whose commands call `{{.API}}/payments` and `{{.API}}/payments/${PAYMENT}` with no `AUTH` header.
- [ ] The README of the repo with `process-manager` ends with the section `## Process Manager Example`. It names
  `task api:pay`, `task api:get-payment`, the table of the amount and of the last digits of the IBAN, the three connector
  files, and that the fakes send no credential, so `api-auth` refuses their webhooks. It names only files that the
  repo has.
- [ ] Written with `process-manager`, the repo has `docs/process-manager-example.md`, `docs/event-model-payment.emod`,
  `.md` and `.svg`. `docs/process-manager-example.md` does not say that an outside system sends the bearer token of the
  API, names the connector files, and names only files that the repo has.
- [ ] The test at `catalog_test.go:1441-1487` also reads the repo with `process-manager`, and finds each route of its
  API under `/api` in its docs, its README and its Taskfiles.
- [ ] Written from the starter alone, `Taskfile.api.yml` and `README.md` hold no line and no blank line from an item.
- [ ] No written file holds the name `api.tasks` or `readme.sections`, and `feature-compose check --catalog catalog`
  finds no mistake.
- [ ] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` and `show` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — two extension points
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml`, `README.md` — the items
- `catalog/features/process-manager/feature.yml` — the tasks and the README section
- `catalog/features/process-manager/files/go/docs/` — new: `process-manager-example.md`, `event-model-payment.emod`,
  `event-model-payment.md`, `event-model-payment.svg`
- `internal/catalog/catalog_test.go` — the tasks, the README and the docs of the repo with `process-manager`

**Patterns to Follow:**
- `task:6` for items of YAML with `indent`
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml:6-63` for the tasks of the starter
- `catalog/starters/go-event-sourcing-service/files/go/README.md:181-207` for the end of the README
- `internal/catalog/catalog_test.go:1157-1173` for docs that name only files that the repo has
- `internal/catalog/catalog_test.go:1441-1487` for the routes under `/api` in the docs
- `tasks/learnings.md:113-119` for how the docs name the local API
- `tasks/learnings.md:334-340` for `emod validate`

**Testable:** Yes

**Certainty:** high — Task 6 sets the items of YAML with `indent`, and `catalog_test.go:1157-1173` and
`catalog_test.go:1441-1487` are the tests of docs to follow

**Blast radius:** low — none of the named categories; docs and local tasks

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write the repo with
`--feature process-manager` into a scratch folder, run `task --list-all` there, and read the README and the docs.
Run `task test:combinations` and the e2e tests of `new` and `show` on the host.

**Depends on:** Task 6

---

### Task 8: Check end to end that a repo with `process-manager` builds, passes its tests and runs its fakes, and that the catalog has one copy of its files

**Behavior:** An e2e test runs `feature-compose new go-event-sourcing-service my-service --feature process-manager
--no-prompt`, and checks the repo: `gofmt` lists no file, it builds, its integration tests compile, its unit tests pass, it
has the tasks that `task run` needs, its API serves the payment routes and the webhook routes, and its fakes call the
webhook of a request. A catalog test shows that the catalog has one copy of each file of the process manager
(decision 9).

**Acceptance Criteria:**
- [ ] The e2e test makes the repo, and the run exits with 0. In the repo, `gofmt -l .` lists no file,
  `go build ./...` and `go vet -tags integration ./...` exit with 0, and `task test:unit` exits with 0. The packages
  that pass include each folder with unit tests under `internal/use_cases`, `internal/api/use_cases`,
  `internal/fakeservices` and `cmd/catchup`, with the connector files of `internal/use_cases`, `internal/api/use_cases` and
  `cmd/catchup` among them.
- [ ] `task --list-all` in the repo names `run`, `fakeservices:run`, `api:pay` and `api:get-payment`.
- [ ] The test starts the API of the repo with DB settings that point to no database. `POST /api/payments` and
  `POST /api/webhooks/fraud-checks` with the body `{}` get the answer 400 of their handlers, and `POST /payments` gets
  404.
- [ ] The test starts the fakes of the repo on a free port with no webhook delay, and sends `POST /fraud/checks` with a
  key of the request and an amount under the limit, whose webhook URL is a server of the test. The fakes answer 202,
  and the server gets a webhook with the payment ID, the result `passed` true, and the same key. The test stops the
  API, the fakes and the server when it ends.
- [ ] A catalog test walks the catalog folder: each file whose path in the repo is a path of the process manager is
  under `catalog/features/process-manager/files/go/`, and no two files of the catalog give the same path in the repo.
- [ ] Only the full runs have a timeout longer than the 30 seconds of `e2e/vitest.config.ts`.
- [ ] `docs/e2e-tests.md` describes the full run with `process-manager`: what it checks, what it starts, what it
  needs, and that it does not run `task run`, the migrations or the integration tests.
- [ ] `task test:unit` passes, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `e2e/tests/new.test.ts` — the full run with `process-manager`
- `docs/e2e-tests.md` — the full run with `process-manager`
- `internal/catalog/catalog_test.go` — one copy of the files of the process manager

**Patterns to Follow:**
- `e2e/tests/new.test.ts:43-80` and `e2e/tests/new.test.ts:154-204` for the full runs of `new`
- `e2e/tests/new.test.ts:681-714`, `new.test.ts:741-760` and `new.test.ts:836-847` for `startedApi`, `answers`,
  `freePort`, `passedPackages` and `foldersWithTests`
- `e2e/tests/new.test.ts:869-875` for `tasksIn`
- `docs/e2e-tests.md:80-195` for the description of the full runs
- `internal/catalog/catalog_test.go:936-945` for the paths of the process manager
- `tasks/learnings.md:37-43` for `MISE_TRUSTED_CONFIG_PATHS`
- `tasks/learnings.md:51-57` for the packages that passed
- `tasks/learnings.md:58-64` for the e2e tests on the host
- `tasks/learnings.md:106-112` for the API without Postgres
- `tasks/learnings.md:348-354` for the scheduler test that can fail under load

**Testable:** Yes

**Certainty:** high — repeats the full run with `api-auth` (`e2e/tests/new.test.ts:154-204`) and its started API
(`new.test.ts:681-714`); the fakes start as the API does

**Blast radius:** low — none of the named categories; it adds tests and docs, and changes no file of a written repo

**Verification:** `task test:unit` passes. Run `e2e/tests/new.test.ts` on the host (`tasks/learnings.md:58-64`). Apply
one break at a time and read its diff: remove the include of `Taskfile.fakeservices.yml` from `feature.yml`; copy one
file of the process manager into the starter. Each break makes a test fail. Put each file back.

**Depends on:** Task 7

---

## Summary

**Eight tasks.** The order is risk-first, then dependency-first. Task 1 comes first: it settles the shape of
`catchup.consumers` and the path of a worker that cannot start, with no file of the process manager, so a mistake
there costs no port. Tasks 2 to 5 port the process manager in the order of its packages (the account, the payment
commands, the consumers, the API), so each task adds packages whose imports already exist, and the written repo
builds and passes its unit tests after each task. Task 6 brings the first items of YAML and the fakes, Task 7 the
docs, and Task 8 the e2e test and the check of one copy.

**Assessments.** No task is `low` certainty: this breakdown takes the design decisions (the connector files, the shape of
`catchup.consumers`, the item of `text`, `indent`), and each task has a precedent in this repo or in an earlier task.
Each task names its variation. Five tasks are `high` blast radius: the holds and the balance (Task 2), the
payment aggregate (Task 3), the migrations and the gateways (Task 4), and the protected webhook routes (Task 5).

**The criteria of the story:**

| Criterion | Tasks |
| --- | --- |
| `new ... --feature process-manager --no-prompt` makes a repo with the example, and its unit tests pass | 2, 3, 4, 5, 6 (the files and their unit tests), 8 (the e2e run) |
| `task run` creates `payments` and `payment_deadlines`, and starts the fakes | 4 (the migrations and their order), 6 (the Taskfiles, the pane, the compose file, the fakes), 8 (the tasks and the started fakes); decision 8 |
| catchup starts the consumers through `catchup.consumers`; each other change to a file of the starter is an item; no step other than the folder commands changes a file | 1, 4 (`catchup.consumers`), 2 (the test of the changed files of the starter), 2 to 7 (the items) |
| an item of `catchup.consumers` can fail when it starts; catchup stops, and the message says why | 1 |
| the catalog has one copy of the files of the process manager | 8; decision 9 |
| `process-manager` targets only `go-event-sourcing-service`; `new go my-tool --feature process-manager` stops with the feature and the starter | 2 |

This breakdown defers no criterion. The manual run of `task run` with Docker is the one check that this repo cannot run; the
written repo runs `process_test.go` against Postgres in its own CI.
