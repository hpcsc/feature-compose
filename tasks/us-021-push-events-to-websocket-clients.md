# Tasks: Push the Events of a Service to WebSocket Clients (US-021)

## Contents

1. Task 1: Push each new event to the WebSocket clients of the API with the notifications of Postgres, and stop the work of the API when it stops
2. Task 2: Add an item of `add_to` only for some values of the options, with a `when` on the item
3. Task 3: Deliver the events through NATS with the option `delivery`, and start NATS for the integration tests
4. Task 4: Start NATS with `task run` and Docker Compose, and tell how to use the delivery `nats`

## Story Reference

US-021, "Push the events of a service to WebSocket clients", in `user-stories/feature-compose.md:700-742`
(Description, Acceptance Criteria, Context, Depends on US-017 and US-019). The design of the catalog is
`PROPOSAL.md` (sections 5 to 10) and `docs/catalog.md` (Options, Files for One Value of an Option, Extension Points).

The source of the port is the template `go-event-sourcing-service` of `~/repo-templates` (read-only):

- `13b501b`: the constant `PushChannel`, a second `pg_notify` in `SaveInTx`, and `WithListenChannel` of the Postgres
  subscription.
- `017999a` and `4eded84`: `internal/api/use_cases/push_subscriber.go`, `internal/api/websocket/handler.go` and
  `hub.go`, the blocks in `internal/api/use_cases/register.go`, and `github.com/coder/websocket v1.8.15` in `go.mod`.
- `5744ef1`: `Register` returns a function that `Server.Shutdown` calls after the HTTP server stops.
- `06052b3`: `internal/use_cases/push_nats/reaction.go`, `internal/common/natspush/natspush.go`,
  `internal/api/websocket/nats_bridge.go`, `internal/common/config/nats.go`, the blocks in `cmd/catchup/main.go`,
  `internal/api/config.go`, `Taskfile.yml`, `Taskfile.api.yml`, `Taskfile.catchup.yml`,
  `docker-compose.dependencies.yml`, `docker-compose.local.yml` and `go.mod` (`github.com/nats-io/nats.go v1.55.0`).
- `9d35bd8`: `.env.local`, `.env.integration`, `scripts/test/start-test-dependencies.sh`, the port
  `${NATS_HOST_PORT}`, and `internal/use_cases/push_nats/reaction_test.go` (an integration test against NATS).

Read them with `git -C ~/repo-templates show <sha>`. Port the files with the pattern of `tasks/learnings.md`
(render with sentinel values, then put back the template expressions).

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

- **The second notification channel of the template** (`PushChannel`, the second `pg_notify` of `SaveInTx`,
  `WithListenChannel`). The push reader of the API listens on the channel `events`, which `SaveInTx` already
  notifies in the same transaction. A notification only wakes a reader, which then reads by position, so a second
  channel carries no fact that the first one does not. In each repo, the second channel adds one notification on
  each save, and in a repo without push no process reads it. So the event store of the starter does not change.
  The caller can overrule this (decision 1).
- **A credential on the WebSocket route.** `GET /api/ws` is a public route, as in the template, also in a repo with
  `api-auth`: the WebSocket API of a browser cannot send the `Authorization` header that `api-auth` reads. So each
  client that reaches the route gets each event, with its payload, for example the owner of an account. A page in a
  browser connects only from an origin of `ALLOWED_ORIGINS`, with the defaults of the CORS middleware, or from the
  origin of the API, and a client with no `Origin` header connects. The README section says so (decision 4).
- **More than one instance of the API with `pg-notify`.** All instances keep one checkpoint, so `pg-notify` serves
  one instance. The README section says so, and names `nats` for more instances.
- **An ID of the event, or a position, in the message to a client.** The message has the type and the payload of
  the event, as in the template. With `nats`, a client can get an event two times and cannot see that it is the
  same event.
- **A task of the Taskfiles that opens the WebSocket route.** It needs a tool such as `websocat` in `mise.toml`.
  The README shows how to connect.
- **The integration tests of a written repo in the CI of feature-compose.** The job `combinations` makes the repo of
  each combination (each value of `delivery` among them) and runs its `after_render` and `test` commands
  (`task test:unit`). The integration tests need Postgres and NATS in Docker. The tasks run them on this machine as
  proof (Docker answers here), and the CI workflow of the written repo runs them on each push (decision 8).
- **A start of a push API with no Postgres, or with no NATS.** A worker of the API that cannot start stops the
  API (decision 3), so the e2e tests, which start the API with no Postgres, check that stop and do not open the
  WebSocket route.

**Deferred:** none of the criteria of this story.

## Codebase Context

**The API of the starter today.** `cmd/api/main.go.tmpl:17-53` makes the server, waits for a signal in a goroutine,
calls `srv.Shutdown()`, and exits with 0. `internal/api/server/server.go.tmpl:143-207`: `New` builds the handler with
`use_cases.Register(r, protectedMiddleware(cfg), pool, logger)`; `Shutdown` calls `http.Server.Shutdown` with 30
seconds. `http.Server.Shutdown` does not close or wait for a hijacked connection, such as a WebSocket. The API has
no step that runs work beside the server, and no step of its own when it stops: there is no extension point for
the work of the API when it stops. `internal/api/use_cases/register.go.tmpl:73-120`: `Register` takes
`protectedMiddleware` (not `cfg`), and `allRoutes(pool)` builds the routables and appends each item of `api.routes`.
`register_test.go.tmpl:27,100` and `features/process-manager/files/go/internal/api/use_cases/payment_test.go.tmpl:26`
call `Register` with a nil pool, so a Register that starts a reader of Postgres panics in these tests.

**Catchup.** `cmd/catchup/main.go.tmpl:62-124`: a `worker` has `Name()` and `Start(ctx) error`; `run` starts each
worker under the lock of its name, and a worker that returns an error stops catchup with exit 1 and the name of
the worker in the log. `catchup.consumers` (`starter.yml:54-57`) appends `[]worker`. US-017 decision 5 says that a
worker that needs an outside service connects to it in `Start`, and that US-021 follows this for NATS.

**The readers of the events.** `reaction.NewReactor` (`internal/domain/reaction/reactor.go.tmpl`) gives a
`*consumer.Consumer` (`internal/domain/consumer/consumer.go.tmpl`), a worker that reads from its checkpoint, or from
`LastPosition` when it has none, saves the checkpoint after each event, and returns when its context ends. So the
push subscriber of the template is a reaction that this reactor runs, and the publisher of the template in catchup
is one too. `subscription/postgres.go.tmpl:186-206` listens on `event.NotifyChannel` (`events`), and
`stream/postgres.go.tmpl:85-88` notifies it in the transaction of each save. `subscription/fake.go.tmpl` and
`checkpoint/store/memory.go.tmpl` let a unit test run a reactor with no Postgres.

**Extension points and items.** `starter.yml` declares 22 extension points. An item is a `map[string]string`
(`internal/catalog/extension_point.go:10-39`); `Selection.items` and `Selection.unique`
(`internal/catalog/selection.go:348-376`) give every item, and the value goes into the file as it is. A files entry
has a `when` (`internal/catalog/feature.go:63-91`, evaluated at `selection.go:244`, checked at
`feature.go:169-190`), but an item has none. So a feature cannot add an item for one value of an option. Each
NATS service, setting and module of this story must not be in a repo with `pg-notify`, so the catalog needs a
`when` on an item (Task 2, decision 2). Each extension point has one `file`.

**Shared files that the template changes for push.** For both values: `internal/api/use_cases/register.go` (the
hub, the reader, the route, the work when the API stops) and `go.mod` (`coder/websocket`). For `nats` only:
`cmd/catchup/main.go` (config and worker), `internal/api/config.go` (config), `go.mod` (`nats.go`),
`docker-compose.dependencies.yml` (the service `nats`), `docker-compose.local.yml` (`NATS_URL` and `depends_on` of
`api` and `catchup`), `Taskfile.yml` (`run-dependencies` starts `nats`), `Taskfile.api.yml` and
`Taskfile.catchup.yml` (`NATS_URL` of `run`), `.env.local` (`NATS_HOST_PORT`), `.env.integration` (`NATS_HOST_PORT`,
`NATS_URL`), `scripts/test/start-test-dependencies.sh` (`up --wait db nats`). Existing extension points cover
`api.routes`, `api.config`, `catchup.config`, `catchup.consumers`, `gomod.requires`, `readme.sections` and
`compose.catchup.environment`. The others need new ones (decision 7).

**Tests that push touches.** `internal/catalog/catalog_test.go`: `writtenWithGuard` (1651-1686) is a test feature
that adds items to points of the starter; 1745-1759 checks that no written file holds the name of an extension
point (so a point named `env.local` fails it, because `.env.local` holds that text); 1761-1797 checks that
`process-manager` changes only the files of its extension points; 1888-1905 reads parsed YAML of the compose files
and the Taskfiles; 2777-2838 checks that each route of the docs is under `/api`, and finds routes only in
`internal/(api/)?use_cases/`. `e2e/tests/show.test.ts:50-66,101-118,120-135` reads the whole tree with 60 rows,
`└─ process-manager` as the last feature, and counts `j` presses to `api.middleware`. `e2e/tests/new.test.ts:403-441`
is the e2e test of `process-manager`, and `new.test.ts:822` lists the features of the starter. `README.md:55,145`
and `docs/e2e-tests.md:298-310` list the features and the e2e tests.

**CI.** `Starter.combinations` (`internal/catalog/starter.go:143-181`) gives each value of each option, and each
pair of features that add to one extension point. So the job `combinations` writes `push` with each value of
`delivery`, and runs `go mod tidy`, `gofmt -w .` and `task test:unit` in each, with no change to CI.

**Words.** A *client* is a WebSocket connection that the API sends each new event to. A *message* is the JSON text
that a client gets for one event: its type and its payload. *Send* is what the API does to its clients;
*publish* is what catchup does to NATS. A *notification* is a Postgres `NOTIFY` on the channel `events`. A *worker*
is something that runs until its context ends, with `Name` and `Start`: catchup runs each one under the lock of its
name, and the API runs each one while it serves. *delivery* is the option, with the values `pg-notify` and `nats`.
An *instance* is one running process of the API. A *connector file* keeps the meaning of US-017.

**Decisions of this breakdown, for the caller to overrule:**

1. **One notification channel.** See Boundaries. If the caller wants the second channel, it goes into the starter
   for every repo (the criterion lists no extension point for the event store), as a separate change of
   `stream/postgres.go.tmpl`, `subscription/postgres.go.tmpl` and `constants.go.tmpl`.

2. **A `when` on an item** (Task 2). The value of an item goes into the file as it is, and items have no condition,
   so with no change to the tool a repo with `pg-notify` gets the NATS service, its settings and the `nats.go`
   module. Task 2 lets an item of `add_to` carry `when`, with the expression and the readable options of the `when`
   of a files entry. Other ways do not work: a second feature splits the one feature with one option that the story
   asks for, and a need cannot depend on a value.

3. **The work of the API: `api.workers`.** The route and the reader share one set of clients, so an item of
   `api.routes` and an item of the work of the API must reach the same value. The starter gets one extension point,
   `api.workers`, in `internal/api/use_cases/register.go`, with the item `[name, code]`: `Register` declares a
   variable `name` with the value `code`, an item of `api.routes` can use that name, and `Register` gives each worker
   to the server. The server starts each worker when it starts. When it stops, after the HTTP server takes no new
   request, it ends the context of each worker and waits until each `Start` returns, so the work of a worker when
   the API stops (close the clients, close NATS) runs before the API exits. A worker whose `Start` returns an error
   stops the API with exit 1 and the name of the worker in the log, as in catchup. `Register` also takes `cfg`, for
   the NATS setting of Task 3. The value of `code` does no I/O, so the tests that call `Register` with a nil pool
   still pass. The interface is `api.Worker` in `internal/api`, because `server` and `use_cases` both use it.

4. **The route.** `GET /api/ws`, public, for a page of an origin of `ALLOWED_ORIGINS` or of the API, and for a client
   with no `Origin` header. A message is
   `{"type": <event type>, "data": <payload>}`, the same for both values of `delivery`.

5. **The readers.** With `pg-notify`, the push worker of the API runs `reaction.NewReactor` with a reaction that
   sends each event to the clients, over `subscription.NewPostgres` and the Postgres checkpoint store. The
   checkpoint name is `push-subscriber`, the name of the template. With `nats`, catchup runs a worker
   `push-nats` (the name of the template), which connects to NATS in `Start` and runs a reactor whose reaction
   publishes each event; each API instance runs a worker that reads the subject with a consumer of its own that
   gets only new messages, and sends each message to its clients.

6. **The option comes in Task 3.** Task 1 makes `push` with no option, so no value of the option writes a repo that
   does not do what the value says. Task 3 adds `delivery` with both values and puts `when` on the files and items of
   `pg-notify`.

7. **The new extension points of the starter.** Ten, each in one file, declared by the task that first adds an item
   to it. No written file holds any of these names, which the test of `catalog_test.go:1745-1759` needs:

   | Extension point | File | Item | Task | Item of `push` with `nats` |
   | --- | --- | --- | --- | --- |
   | `api.workers` | `go/internal/api/use_cases/register.go` | `[name, code]` | 1 | the push worker of the API (one item for each value, with `when` from Task 3) |
   | `compose.dependencies.services` | `go/docker-compose.dependencies.yml` | `[text]` | 3 | `nats`, with JetStream, a health check and the port `${NATS_HOST_PORT}` |
   | `dotenv.local` | `go/.env.local` | `[name, value]` | 3 | `NATS_HOST_PORT=4222` |
   | `dotenv.integration` | `go/.env.integration` | `[name, value]` | 3 | `NATS_HOST_PORT=44222`, `NATS_URL=nats://127.0.0.1:44222` |
   | `test.dependencies` | `go/scripts/test/start-test-dependencies.sh` | `[service]` | 3 | `nats` |
   | `taskfile.dependencies` | `go/Taskfile.yml` | `[service]` | 4 | `nats` |
   | `api.run.environment` | `go/Taskfile.api.yml` | `[name, value]` | 4 | `NATS_URL=nats://127.0.0.1:4222` |
   | `catchup.run.environment` | `go/Taskfile.catchup.yml` | `[name, value]` | 4 | `NATS_URL=nats://127.0.0.1:4222` |
   | `compose.api.environment` | `go/docker-compose.local.yml` | `[name, value]` | 4 | `NATS_URL=nats://nats:4222` |
   | `compose.depends_on` | `go/docker-compose.local.yml` | `[service]` | 4 | `nats`, for `api` and for `catchup` |

   With `nats`, `push` also adds to `compose.catchup.environment` (`NATS_URL`), `api.config`, `catchup.config`,
   `catchup.consumers` and `gomod.requires`. The tree of `show` gets 11 rows (10 points and the feature), about 54
   of `rowsOfTheWholeTree` (60).

8. **The CI criterion.** "A repo with each value of `delivery` builds and passes its tests" is the job
   `combinations`, which runs the `test` command of the folder, `task test:unit`, for each value with no change.
   If the caller reads "its tests" as the integration tests too, the job needs Postgres and NATS services, which is
   a change to `.github/workflows/ci.yml` and to `TestCombination` that this breakdown does not plan.

9. **The NATS config type** is in a package of the feature (`internal/common/nats_push`), not in
   `internal/common/config`: `config.go` of the API and `main.go` of catchup import `internal/common/config`
   already, and `unique "import"` of `api.config` and `catchup.config` would import it a second time.

10. **The packages of the feature.** `internal/api/push` (both values): the clients, the handler of the route, the
    message; with `pg-notify` the reaction that sends each event to the clients; with `nats` the reader of NATS.
    `internal/common/nats_push` (`nats`): the config, the names of the stream and of its subject, and the reaction
    that publishes each event. Connector files: `internal/api/use_cases/push.go` (one for each value, with the push
    worker of the API) and `cmd/catchup/push.go` (`nats`). The package `websocket` of the template is renamed,
    because it imports `github.com/coder/websocket` of the same name, and `natspush` becomes `nats_push` by the rule
    for packages of more than one word. The template package `push_nats` goes into `nats_push`, so the repo has no
    two packages whose names are the same two words.

## Tasks

### Task 1: Push each new event to the WebSocket clients of the API with the notifications of Postgres, and stop the work of the API when it stops

**Behavior:** `feature-compose new go-event-sourcing-service my-service --feature push --no-prompt` writes a repo whose
API serves `GET /api/ws`. A client that opens the route gets one message for each event that the service stores
after the client connects. The API reads the new events through the notifications of Postgres on the channel
`events` and keeps a checkpoint of the last event that it sent, so after a restart it goes on from that event and
does not send an old event again. The starter gets the extension point `api.workers` (decision 3): the API runs
each worker while it serves, and when it stops it ends the work of each worker and waits for it, so the push worker
closes each WebSocket connection before the API exits.

**Acceptance Criteria:**
- [x] `catalog/features/push/feature.yml` has `targets: [go-event-sourcing-service]` and no option.
  `feature-compose new go my-tool --feature push` and `feature-compose new chi-api my-api --feature push` stop, and the
  message names `push` and the starter. `task test:catalog` gives no mistake.
- [x] `starter.yml` declares `api.workers` with the item `[name, code]` and a summary that names the values that the
  code can use (at least `cfg`, `pool` and `logger`), the type that `Start` has, that an item of `api.routes` can
  use the name, what the API does with a worker when it starts and when it stops, and that a worker whose `Start`
  returns an error stops the API with exit 1. The summary of `api.routes` says that its code can use the name of
  each item of `api.workers`.
- [x] A test feature of `TestCatalog`, with an item of `api.workers` and an item of `api.routes` that uses its name,
  writes a `register.go` that declares the variable, gives the worker to the server, and appends the routes of the
  item of `api.routes`. With no item, the starter alone builds and passes `task test:unit`, and the e2e tests of the
  starter that start its API pass unchanged.
- [x] The register test of the starter and `payment_test.go` of `process-manager` call `Register` in its new form,
  and pass in a written repo of each.
- [x] Written with `push`: `gofmt -l .` lists no file, `go build ./...` and `go vet -tags integration ./...` pass,
  and `task test:unit` passes with the tests of `internal/api/push` among the passed packages.
- [ ] The unit tests of `internal/api/push` show: a client that connects to the route gets each message that the API
  sends after it connects; a message is a JSON object with the keys `type` (the type of the event) and `data` (its
  payload); a client that leaves gets no more messages, and the other clients still get theirs; when the API closes
  its clients, each client gets a close with the status going away.
- [ ] The unit tests run the reaction of `push` with the reactor of the starter, a fake subscription and a checkpoint
  store in memory, and show: with no checkpoint, the first start sends no event that was stored before it; after a
  stop and a new start with the same checkpoint store, the clients get only the events after the last event that
  was sent, and no event that was sent before.
- [ ] Integration tests of the written repo, which `task test:integration` runs against Postgres, show through the
  server of the API: a client gets each event that the service stores after the client connects; after the API
  stops and starts again, a client gets the event stored while the API was down, and no event that the API sent
  before it stopped; when the API stops, its client gets a close with the status going away.
- [x] An API of a repo with `push` whose Postgres does not answer stops with exit status 1, and its output names
  the push worker and the error.
- [x] The repo with `push` differs from the starter-alone repo only in the files of the folders of `push`, and in the
  file of each extension point that `push` adds items to. `push` adds to `api.workers`, `api.routes`,
  `gomod.requires` (`github.com/coder/websocket`) and `readme.sections`.
- [x] The README of the repo ends with a section that names the route `/api/ws`, the form of a message, that a client
  gets each event that the service stores after it connects, that after a restart the API goes on from the last
  event that it sent, that one instance of the API reads the events, and that the route asks for no credential,
  also in a repo with `api-auth`. It names only files and tasks that the repo has.
- [x] The test of `catalog_test.go:2777-2838` includes a repo with `push`, finds the route `/ws`, and passes.
- [x] The e2e tests list `push` among the features of the starter (`new.test.ts:822`), find `└─ push` as the last
  feature and the whole tree in 60 rows (`show.test.ts`), and a new e2e test makes a repo with `push` that builds,
  compiles its integration tests, passes `task test:unit` with `internal/api/push` among the passed packages, and
  whose API stops with exit status 1 when Postgres does not answer. `README.md` (the features table and the list of
  features), `docs/e2e-tests.md` and the form of `docs/commands.md` name `push`.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — `api.workers`, and the summary of `api.routes`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/worker.go.tmpl` — new: the interface of a worker of the API
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl` — `cfg`, the items of `api.workers`, the workers that it gives back
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl` — the new form of `Register`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl` — start and stop the workers
- `catalog/starters/go-event-sourcing-service/files/go/cmd/api/main.go.tmpl` — exit 1 when a worker fails
- `catalog/features/process-manager/files/go/internal/api/use_cases/payment_test.go.tmpl` — the new form of `Register`
- `catalog/features/push/feature.yml` — new
- `catalog/features/push/files/common/go/internal/api/push/` — new: the clients, the handler of the route, the message, and their unit tests
- `catalog/features/push/files/pg-notify/go/internal/api/push/` — new: the reaction that sends each event to the clients, and its unit tests
- `catalog/features/push/files/pg-notify/go/internal/api/use_cases/push.go.tmpl` — new connector file: the push worker of the API
- `catalog/features/push/files/pg-notify/go/internal/api/server/` — new: the integration tests through the server
- `internal/catalog/catalog_test.go` — tests of `api.workers` and of `push`
- `e2e/tests/new.test.ts`, `e2e/tests/show.test.ts` — `push`
- `README.md`, `docs/e2e-tests.md`, `docs/commands.md` — `push`

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main.go.tmpl:62-124` for a worker, its run and a worker that fails
- `catalog/starters/go-event-sourcing-service/files/go/cmd/catchup/main_test.go.tmpl` for a test of the run of workers
- `catalog/features/process-manager/feature.yml` and `catalog/features/process-manager/files/go/internal/api/use_cases/payment.go.tmpl` for a feature with a connector file and items
- `catalog/starters/go-event-sourcing-service/files/go/internal/domain/reaction/reactor.go.tmpl` and `catalog/starters/go-event-sourcing-service/files/go/internal/domain/reaction/reactor_test.go.tmpl` for a reaction that a reactor runs, with a fake subscription
- `catalog/starters/go-event-sourcing-service/files/go/internal/domain/event/subscription/fake.go.tmpl` and `catalog/starters/go-event-sourcing-service/files/go/internal/domain/checkpoint/store/memory.go.tmpl`
- `catalog/starters/go-event-sourcing-service/files/go/internal/common/test/db.go.tmpl` for an integration test against Postgres
- `internal/catalog/catalog_test.go:1651-1686` (a test feature), `internal/catalog/catalog_test.go:1761-1797` (only the files of extension points change), `internal/catalog/catalog_test.go:2777-2838` (routes under `/api`)
- `e2e/tests/new.test.ts:403-441` for the e2e test of a feature of the starter
- Template commits `017999a`, `4eded84` and `5744ef1` of `~/repo-templates` for the code of the clients, the route and the subscriber

**Testable:** Yes — unit tests of `internal/api/push` and of the reaction in the written repo, integration tests
through the server against Postgres, catalog tests, and e2e tests. `clerk mutants` does not change the templates, so
prove the tests with breaks by hand on a written repo (for example: send to no client, send an event before its
checkpoint, close with another status, skip the wait for the workers, exit 0 when a worker fails), run the tests after
each break, and put the file back.

**Certainty:** medium — catchup runs and stops workers (`cmd/catchup/main.go.tmpl:93-124`) and the template has the
WebSocket code, but the API has no run or stop of work today, and a hijacked WebSocket connection outlives
`http.Server.Shutdown`, which no code of this repo handles yet.

**Blast radius:** high — a public route sends each stored event with its payload (for example the owner of an
account) to any client that reaches it, also in a repo with `api-auth`, so it changes who can see personal data.

**Verification:** `task test:unit`, `task test:catalog` and the e2e tests in this repo (`task test:e2e`, or
`task test:e2e:local` when Docker does not answer); in a repo that the built binary writes under `.scratches/`:
`task test:unit`, `task test:integration` with Docker, and the breaks by hand.

**Depends on:** None

---

### Task 2: Add an item of `add_to` only for some values of the options, with a `when` on the item

**Behavior:** A catalog author writes `when: <expression>` in an item of `add_to`. The file of the extension point
writes the item only when the expression is true for the values that the feature that adds the item can read: its
own options, the options of its needs as `<need>.<option>`, and `starter.<option>`. `items` and `unique` give only
those items. `check` finds a `when` of an item that does not parse or that reads an option that the feature cannot
read, with the messages of the `when` of a files entry.

**Acceptance Criteria:**
- [x] An item with `when` is in the written file of its extension point only when the expression is true for the
  values of the selection, read as the feature that adds the item reads them. An item with no `when` is in each
  repo, and each test of the catalog that passed before passes unchanged.
- [x] `unique` gives the values of the items whose `when` is true, and no value of another item.
- [x] `when` is not an item field: an item with the fields of its extension point and a `when` is not a mistake, and
  the order of the items does not change. `check` finds an extension point that declares the item field `when`.
- [x] A `when` of an item that does not parse stops `new`, and `check` gives one mistake that names the feature, the
  extension point, the number of the item, the `when` and the cause.
- [x] A `when` of an item that reads an option that the feature cannot read is a mistake of `check`, with the
  reference and the options that the feature can read, as for the `when` of a files entry.
- [x] The definitions that `show --output json` gives have the `when` of an item as the key `when` of the item.
- [x] `docs/catalog.md` tells how an item applies only to some values of the options, with an example and the rules,
  and the table of `docs/commands.md` has the new mistakes of `check` with an example of each message.

**Affected Files/Modules:**
- `internal/catalog/feature.go` — read the `when` of each item, and check what it reads
- `internal/catalog/extension_point.go` — the `when` of an item, out of its fields; the item field `when`
- `internal/catalog/starter.go` — the items of the selected features
- `internal/catalog/selection.go` — `items` and `unique` give only the items whose `when` is true
- `internal/catalog/definitions.go` — the `when` of an item in the definitions, if the item map does not carry it already
- `internal/catalog/catalog_test.go`, `internal/catalog/starter_test.go` — tests with catalogs of the test
- `docs/catalog.md`, `docs/commands.md`

**Patterns to Follow:**
- `internal/catalog/feature.go:63-91` (read and parse the `when` of a files entry)
- `internal/catalog/feature.go:169-190` (the check of what a `when` reads)
- `internal/catalog/selection.go:243-249` (evaluate a `when` with the values that a feature can read)
- `internal/catalog/selection.go:348-376` (`items` and `unique`)
- `internal/catalog/extension_point.go:17-39` (the item fields)
- `internal/catalog/catalog_test.go:248-263` and `internal/catalog/catalog_test.go:939-980` (tests of the `when` of a files entry)
- `docs/catalog.md:229-243` and `docs/commands.md:358-372`

**Testable:** Yes — through `Starter.Select`, `Selection.Write`, `Catalog.Check` and `Catalog.Definitions` with
catalogs of the test (`fstest.MapFS`). `clerk mutants` checks this Go code.

**Certainty:** medium — the `when` of a files entry is the precedent (`internal/catalog/feature.go:63-91`,
`internal/catalog/selection.go:243-249`), but an item is evaluated inside `items` and `unique` at render, with the
values of the feature that adds it and not of the file that writes it, and its `when` must stay out of its fields.

**Blast radius:** low — a new key of `feature.yml` that no item uses yet, and the output of `show` gets the key only
for an item that has it.

**Verification:** `task test:unit`, `task test:catalog`, `clerk mutants` on the changed Go code.

**Depends on:** None

---

### Task 3: Deliver the events through NATS with the option `delivery`, and start NATS for the integration tests

**Behavior:** `push` gets the option `delivery`, with the values `pg-notify` (the default) and `nats`. With
`pg-notify`, the repo is the repo of Task 1, byte for byte, and has no NATS. With `nats`, catchup runs a worker that
connects to NATS when it starts and publishes each event to a JetStream stream, after the events before it, with a
checkpoint; each instance of the API reads the stream and sends each message to its clients, so a client of any
instance gets each event at least one time. When the API stops, it closes its clients and its connection to NATS.
The integration tests start NATS beside Postgres.

**Acceptance Criteria:**
- [x] `push` declares the option `delivery` with the values `pg-notify` and `nats` and the default `pg-notify`. The
  files entries and the items that only `pg-notify` uses, the README section of Task 1 among them, have
  `when: delivery == "pg-notify"`, and those of `nats` have `when: delivery == "nats"`.
- [ ] With the default and with `delivery=pg-notify`, the repo that the built binary writes from the work tree is the
  same, byte for byte, as the repo with `push` that it writes from the catalog of the commit before (`diff -r`, with
  the `.md` files in the diff). The starter alone is also the same, byte for byte.
- [x] With `pg-notify`, the repo has no NATS: no service of a Docker Compose file, no NATS setting of an `.env`
  file or a Taskfile, no `nats-io` module in `go.mod`, and no Go file that imports one.
- [x] `starter.yml` declares `compose.dependencies.services` (`[text]`), `dotenv.local` (`[name, value]`),
  `dotenv.integration` (`[name, value]`) and `test.dependencies` (`[service]`), each with the file of decision 7 and
  a summary. With no item, each of these files is the same as before, byte for byte.
- [x] With `nats`, the parsed `docker-compose.dependencies.yml` has the service `nats` with JetStream, a health check
  and the port `${NATS_HOST_PORT}`; `.env.local` sets `NATS_HOST_PORT`; `.env.integration` sets `NATS_HOST_PORT` and
  `NATS_URL` to another port than `.env.local`; `scripts/test/start-test-dependencies.sh` starts `nats` and waits for
  it, and passes `task test:shellcheck`.
- [x] With `nats`, the config of the API and of catchup has a NATS field from a package of `push` (decision 9),
  imported one time; `catchup.consumers` adds the push worker; `go.mod` requires `github.com/nats-io/nats.go`.
- [x] A catchup whose NATS does not answer when the push worker starts logs the name of the worker with the error and
  exits with 1.
- [ ] Integration tests of a written repo with `nats`, which `task test:integration` runs with Postgres and NATS, show:
  an event that the push worker of catchup publishes reaches a client of each of two instances of the API at least
  one time, with the same message as with `pg-notify`; after an instance stops, its client gets a close with the
  status going away, and NATS has no connection and no consumer of that instance.
- [x] With each value, the repo with `push` differs from the starter-alone repo only in the files of the folders of
  `push` for that value, and in the file of each extension point that an item of that value adds to.
- [x] `Catalog.Combinations` gives `push` with each value of `delivery`, and `TestCombination` passes for each
  combination with `push`.

**Affected Files/Modules:**
- `catalog/features/push/feature.yml` — the option, the `when` of the files entries and of the items, the items of `nats`
- `catalog/features/push/files/nats/go/internal/common/nats_push/` — new: the config, the stream and the reaction that publishes, with integration tests
- `catalog/features/push/files/nats/go/internal/api/push/` — new: the reader of NATS, with integration tests
- `catalog/features/push/files/nats/go/internal/api/use_cases/push.go.tmpl` — new connector file: the push worker of the API with NATS
- `catalog/features/push/files/nats/go/cmd/catchup/push.go.tmpl` — new connector file: the push worker of catchup
- `catalog/starters/go-event-sourcing-service/starter.yml` — four extension points
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.dependencies.yml`, `catalog/starters/go-event-sourcing-service/files/go/.env.local`, `catalog/starters/go-event-sourcing-service/files/go/.env.integration`, `catalog/starters/go-event-sourcing-service/files/go/scripts/test/start-test-dependencies.sh` — the `items` of each point
- `internal/catalog/catalog_test.go` — tests of the option, of no NATS with `pg-notify`, and of the items of `nats`

**Patterns to Follow:**
- `catalog/features/api-auth/feature.yml` for the files entries of each value of an option
- `catalog/features/process-manager/files/go/cmd/catchup/payment.go.tmpl:24-35` and `catalog/features/process-manager/files/go/cmd/catchup/payment_test.go.tmpl` for a connector file of catchup and its test
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:30-36` and `catalog/starters/go-event-sourcing-service/starter.yml:95-101` for an extension point of a YAML file
- `catalog/starters/go-event-sourcing-service/files/go/internal/common/test/db.go.tmpl` for the environment of an integration test
- `internal/catalog/catalog_test.go:1888-1905` for a test of parsed YAML
- `task:1` (the push worker of the API and its tests), `task:2` (the `when` of an item)
- Template commits `06052b3` and `9d35bd8` of `~/repo-templates` for the NATS code, its integration test and the files of the test stack

**Testable:** Yes — integration tests in the written repo with Postgres and NATS, catalog tests, `TestCombination`
for the combinations of `push`, and breaks by hand on a written repo (for example: publish no event, read no
message, keep the NATS connection open when the API stops, import NATS with `pg-notify`).

**Certainty:** low — NATS JetStream is a new integration for the catalog: the template commits `06052b3` and
`9d35bd8` are the only precedent, and they have no test of the reader of the API or of two instances.

**Blast radius:** high — catchup copies each event, with its payload and the personal data in it, into a NATS
stream that keeps it for one hour, so more personal data leaves Postgres.

**Verification:** `task test:unit`, `task test:catalog`, `task test:combinations` for the combinations of `push`;
`diff -r` of the repos with `pg-notify` and of the starter alone against the commit before; in a written repo with
`nats` under `.scratches/`: `task test:unit`, `task test:integration` with Docker, and the breaks by hand.

**Depends on:** Task 1, Task 2

---

### Task 4: Start NATS with `task run` and Docker Compose, and tell how to use the delivery `nats`

**Behavior:** With `nats`, `task run` starts NATS beside Postgres, and `task api:run` and `task catchup:run`
connect to it. With the Docker Compose files, the services `api` and `catchup` get the address of NATS and wait
until NATS is healthy. The README of a repo with `nats` tells how the events reach the clients, and how to run and
test the repo.

**Acceptance Criteria:**
- [ ] `starter.yml` declares `taskfile.dependencies` (`[service]`), `api.run.environment` (`[name, value]`),
  `catchup.run.environment` (`[name, value]`), `compose.api.environment` (`[name, value]`) and `compose.depends_on`
  (`[service]`), each with the file of decision 7 and a summary. The starter alone and the repo with `pg-notify` are
  the same as before, byte for byte.
- [ ] With `nats`, read from the parsed YAML: the task `run-dependencies` of `Taskfile.yml` starts `nats` after `db`
  and waits for it; the task `run` of `Taskfile.api.yml` and of `Taskfile.catchup.yml` sets `NATS_URL` to the port of
  `.env.local`; in `docker-compose.local.yml`, `api` and `catchup` have `NATS_URL: nats://nats:4222` and wait for
  `nats` with the condition `service_healthy`.
- [ ] A run by hand of `task run` in a written repo with `nats` starts Postgres, NATS, the API and catchup, and a
  WebSocket client of the API gets the message of the event that `task api:open-account` stores.
- [ ] The README of a repo with `nats` ends with a section that names the route `/api/ws`, the form of a message, that
  catchup publishes each event to NATS and that each instance of the API sends it to its clients at least one time,
  that `task run` and the integration tests start NATS, the setting `NATS_URL`, and that the route asks for no
  credential, also in a repo with `api-auth`. It names only files and tasks that the repo has, and each route under
  `/api`.
- [ ] An e2e test makes a repo with `push:delivery=nats` that builds, compiles its integration tests and passes
  `task test:unit`, and whose `task --list-all` names `run` and `run-dependencies`. `docs/e2e-tests.md` tells it.
  The whole tree of `show` fits in `rowsOfTheWholeTree` rows.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/starter.yml` — five extension points
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml`, `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml`, `catalog/starters/go-event-sourcing-service/files/go/Taskfile.catchup.yml`, `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml` — the `items` of each point
- `catalog/features/push/feature.yml` — the items of `nats`, and the README section of `nats`
- `internal/catalog/catalog_test.go` — tests of the parsed Taskfiles and compose files, and of the README section
- `e2e/tests/new.test.ts`, `e2e/tests/show.test.ts`, `docs/e2e-tests.md`

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:22-36` and `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml:8-15` for items in a compose file and in a Taskfile
- `catalog/starters/go-event-sourcing-service/starter.yml:91-101` for the summaries of the points of `.tmuxp.yaml` and of the compose files
- `internal/catalog/catalog_test.go:1888-1905` for a test of the parsed Taskfiles and compose files
- `e2e/tests/new.test.ts:403-441` for the e2e test of a feature of the starter
- `task:3` (the items of `nats` and the byte-for-byte check)

**Testable:** Yes — catalog tests of the parsed YAML and of the README, an e2e test, and a run of `task run` by hand
(the pattern of `tasks/learnings.md`: write the repo under `.scratches/`, a tmux server of its own, then stop the
binaries by their ports and run `docker compose down -v`).

**Certainty:** high — the extension points of `docker-compose.local.yml`, `.tmuxp.yaml` and `Taskfile.yml` of
US-017 (`catalog/starters/go-event-sourcing-service/files/go/docker-compose.local.yml:22-36`) are the precedent, and
Task 3 sets the items of `nats`.

**Blast radius:** low — local run files and docs of a written repo; no data, credential or contract changes.

**Verification:** `task test:unit`, `task test:catalog`, the e2e tests, `diff -r` of the repos with `pg-notify` and
of the starter alone against the commit before, and a run of `task run` by hand in a written repo with `nats`.

**Depends on:** Task 3

## Summary

- **Four tasks.** Task 1 is the main slice: a repo with `push` sends each new event to its WebSocket clients, and the
  API stops its work when it stops. Task 2 is the tool change that lets an item apply for one value. Task 3 adds the
  value `nats` with the Go code and the test stack. Task 4 adds the local run and the docs of `nats`.
- **Order.** Task 1 and Task 2 have no dependency on each other. Task 1 comes first because it has the most risk
  (the work of the API, a public route of personal data) and the most value. Task 3 needs both, and Task 4 needs
  Task 3. The criterion of `task run` and the criterion of the integration tests split Task 3 from Task 4.
- **Coverage of the story:**
  - "`new ... --feature push --no-prompt` makes a repo, and its unit tests pass. A client ... gets each event ..." —
    Task 1.
  - "`push` has the option `delivery`, with the values `pg-notify` and `nats`. The default is `pg-notify`." — Task 3.
  - "With `pg-notify`, the API reads the new events from the notifications of Postgres. After a restart ... The repo
    has no NATS" — Task 1 (reader and restart), Task 3 (no NATS).
  - "With `nats`, the catchup process sends each event to NATS, and each instance of the API sends it ... `task run`
    and the integration tests start NATS." — Task 3 (delivery and integration tests), Task 4 (`task run`).
  - "When the API stops, it closes its WebSocket connections, and with `nats` its connection to NATS." — Task 1
    (`api.workers` and the clients), Task 3 (NATS).
  - "Each change that `push` makes to a shared file of the starter is an item of an extension point ..." — Task 1
    (`api.workers`, the work of the API when it stops, and `api.routes`), Task 3 (consumers of catchup, the services
    of the compose files, the `.env` files, the dependencies of the integration tests), Task 4 (the settings of the
    compose files and of the Taskfiles). Each task checks that `push` changes only the files of its extension points.
  - "`push` targets only `go-event-sourcing-service`, and `check` passes. In the CI of feature-compose, a repo with
    each value of `delivery` builds and passes its tests." — Task 1 (targets, check), Task 3 (combinations of each
    value; decision 8).
- **Deferred:** none. The second notification channel of the template is out of scope (decision 1).
