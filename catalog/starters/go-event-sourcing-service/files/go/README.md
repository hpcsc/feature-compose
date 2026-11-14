# {{ kebab .Name }}

A base for an event-sourced Go application that keeps its events in Postgres. Consumers read the events
with a loop that wakes on a `LISTEN/NOTIFY` signal, so the application needs no other event store.

The repository contains:

- **An event store in one Postgres table:** optimistic concurrency for each stream, and a global order
  that a late commit cannot break.
- **A subscription with a read loop and a signal:** no catch-up mode and no live mode, and no event lost
  at a switch.
- **One consumer loop for every kind of consumer:**
  - Projections save each batch and its checkpoint in one transaction.
  - Reactions start at the end of the store and save the checkpoint after each event.
- **An example domain:** a bank account with an HTTP API, a balance projection, and a welcome bonus
  reaction that sends a command.

[docs/event-subscription.md](docs/event-subscription.md) explains the design: the write path, the read
loop, and how to choose the kind of consumer. [docs/event-model-account.md](docs/event-model-account.md)
describes the example domain as an event model.

## Requirements

- [mise](https://mise.jdx.dev/) for Go, `task`, `migrate`, `gotestsum`, `shellcheck` and `govulncheck`
- Docker, `jq` and `tmuxp`

```sh
mise install
```

## Run the Example

```sh
task run
```

This starts Postgres on port 45432, runs the migrations, and opens a tmux window with the `api` and
`catchup` processes.

In another terminal:

```sh
task api:open-account -- alice          # returns the new account ID
export ID=<the account ID>
task api:deposit -- 500
task api:withdraw -- 200
task api:get-account                    # balance 1300: 500 - 200, plus the welcome bonus of 1000
task api:close-account                  # a closed account rejects deposits and withdrawals
```

The balance comes from the `account_balances` read model, so it can show the change a short time after
the command.

A client that retries a `POST` sends the same `Idempotency-Key` header as the first request. The API
then makes the same command, with the same account ID or payment ID, and the store records the
command once. The retry gets the same answer as the first request, also when the first request
succeeded and the client did not get the answer. Use a new key for each new request: the API does not
compare the body of a retry with the body of the first request. A request with no key is a new command
each time.

Other tasks:

| Task | What it does |
| --- | --- |
| `task db:cli` | Opens `psql` on the local database. |
| `task db:query -- 'select * from events'` | Runs one query on the local database. |
| `task remove-dependencies` | Stops the local database and deletes its container. |
| `task migrate:create -- <name>` | Creates a new migration in `migrations/`. |

## Call the API From a Web App

Each route of the API is under `/api`, for example `POST /api/accounts`, so one host can serve a web app at
the other paths. A web app on another origin can call the API when its origin is in `ALLOWED_ORIGINS`.

`ALLOWED_ORIGINS` is a list of origins with commas, for example
`https://app.example.com,http://localhost:3000`. An origin has the form `<scheme>://<host>[:<port>]`, with no
path and no `/` at the end, as the browser sends it. The spaces around an origin do not count. With no value,
or with only blank entries, the API allows `http://localhost:8080` and `http://127.0.0.1:8080`.

To give the local API a value, stop the pane of `task api:run` and run it with the variable:

```sh
ALLOWED_ORIGINS=https://app.example.com task api:run
```

The API answers a request from an allowed origin with `Access-Control-Allow-Origin`. To the preflight of an
allowed origin, it allows the method and the headers that the preflight asks for. A request from another
origin gets no CORS header, so the browser does not give the answer to the web app.

## Test

```sh
task test:unit
task test:integration   # starts a separate Postgres on port 45433
```

Unit tests have the `unit` build tag. Integration tests have the `integration` build tag and use the
values in `.env.integration`.

## Publish the Images

The CI workflow publishes an image of the API and an image of catchup to the GitHub container registry of
the repository. It publishes them on a push to `main` and on a push of a tag, after the tests pass.
A push to another branch runs only the tests. The job logs in with the `GITHUB_TOKEN` of the run, so the
repository needs no secret.

| Image | Dockerfile |
| --- | --- |
| `ghcr.io/<owner>/<repository>-api` | `Dockerfile.api` |
| `ghcr.io/<owner>/<repository>-catchup` | `Dockerfile.catchup` |

The names are in lowercase. Each image has the short SHA of its commit as a tag, for example `4829f92`, and:

- An image from the head of `main` also has the tags `main` and `latest`.
- An image from a version tag, for example `v1.2.3`, also has the version as a tag, `1.2.3`.

Only an image from `main` gets `latest`. The runs of one branch or one tag run one at a time, and only the
image of the head of `main` gets the tags `main` and `latest`. A run of an older commit of `main`, for
example a re-run of an older run, publishes its image with the short SHA only.
When the tests of the newest commit of `main` fail, `main` and `latest` stay on an older image until a newer
commit passes its tests.

## Replace the Example

Replace the account example with your own domain:

- `internal/use_cases/account`: the aggregate, the events, the commands, and one package for each use case
- `internal/use_cases/register.go`: the registration of the event types and the command handlers
- `internal/api/use_cases/account`: the HTTP handler
- the migrations of the read model: `account_balances` and its status column
- the consumers in `cmd/catchup/main.go`, and the routes in `internal/api/use_cases/register.go`

## Layout

| Path | Contents |
| --- | --- |
| `cmd/api` | The HTTP API. It sends commands and reads the read models. |
| `cmd/catchup` | The process that runs the projections and the reactions. |
| `internal/domain` | Events, aggregates, commands and their bus. |
| `internal/domain/event/stream` | The write path: append to a stream, read a stream. |
| `internal/domain/event/subscription` | The read loop and the listener. |
| `internal/domain/consumer` | The loop for every consumer. |
| `internal/domain/projection`, `internal/domain/reaction` | The two kinds of consumer. |
| `internal/domain/checkpoint/store` | Checkpoints in Postgres. |
| `internal/use_cases/<domain>` | The aggregate of a domain, its events, its commands and its repository. |
| `internal/use_cases/<domain>/<use case>` | One use case: its command handler, and its projection or reaction. |
| `internal/use_cases/register.go` | The registration of every event type and every command handler. |
| `internal/api` | The HTTP server, routes, middleware and handlers. |
| `migrations` | The schema. The `migrate` container applies it at start. |

## Add an Event

1. Add a file with a struct for the payload to the `event` package of the domain, for example
   `MoneyDeposited` in `internal/use_cases/account/event/money_deposited.go`. Each event has its own file.
2. Give the struct an `EventType` method that returns the name of the event type.
3. Register the struct in the `RegisterEvents` function of the domain, for example in
   `internal/use_cases/account/events.go`.

The store saves the name of the event type with each event. When the store reads the event, the registry
finds the struct from that name. You can rename the struct, but do not change the name of the event type
after the store has an event of that type.

## Change an Event

Each saved event keeps the schema version of its payload struct. A registration with no schema version
has version 1.

| Change | What to do |
| --- | --- |
| Add a field that an old build can ignore | Change the struct only. |
| Add a field that an old build must not ignore | Change the struct. Increase the schema version, and pass `nil` as the upcast. |
| Rename or remove a field, or change what a field means | Change the struct. Increase the schema version, and give an upcast. |

```go
reg.Register[event.MoneyDeposited](registry.WithSchemaVersion(2, upcastMoneyDeposited))
```

The upcast gets the schema version and the data of an older event, and returns the data in the current
shape. The registry calls it before it parses the data.

When an old build reads an event with a newer schema version, the registry returns
`registry.ErrNewerSchemaVersion`, and the read loop stops. Therefore an old `catchup` stops during a
deploy, and does not write wrong data to a read model. The `catchup` of the new build reads the event.

## Add a Command

1. Add a method to the aggregate. The method checks the rule and calls the private `record` method of the
   aggregate with the new event, for example `domain.NewEvent(&event.MoneyDeposited{...})`. `Apply`
   changes the state from that event.
2. Add a file with a payload type for the command, for example `Deposit` in
   `internal/use_cases/account/command/deposit.go`. Each command has its own file.
3. Add a package for the use case, for example `internal/use_cases/account/deposit`. Give it a handler type
   with a `Handle(ctx, domain.Command[Deposit]) error` method. The handler loads the aggregate from its
   repository, calls the method, and saves.
4. Give the package a `Register` function that calls `bus.Register[Deposit](handler{...})`, and call it
   from `internal/use_cases/register.go`.

Send the command with `bus.Dispatch(ctx, domain.NewCommand(Deposit{...}))`. The bus finds the handler from
the payload type, and calls it again on a concurrency conflict. The bus skips a command whose ID the
store already has, so a command that arrives twice appends its events once. Give a repeated request the
same command ID: an HTTP handler takes it from `idempotency.IDFor`, and a reaction builds its command
with `dispatch.CommandFor`.

## Add a Consumer

- **A projection:** implement `projection.Interface`. Write to the read model through the `pgx.Tx` that
  `Handle` gets. For a time in the read model, use `evt.RecordedAt`, not `NOW()`, so that a rebuild
  writes the same times. Add the table in a migration. Add `projection.NewProjector(...)` to the
  workers in `cmd/catchup/main.go`.
- **A process manager:** an aggregate that holds the state of a process and decides each step, plus
  reactions that carry its messages. Each input reaches the aggregate as a command, and each decision
  leaves it as an event. [docs/event-subscription.md](docs/event-subscription.md#a-process-manager)
  tells when a process needs one.
- **A reaction:** implement `reaction.Interface`, or use `external.NewHttpReaction`. Make the effect
  safe to repeat. A message to another service is a contract, not an event, as
  [docs/event-subscription.md](docs/event-subscription.md) tells. A reaction that sends a command builds
  it with `dispatch.CommandFor`, as `welcome_bonus.Reaction` does: the command gets a fixed ID, and the
  correlation ID of the event. Add `reaction.NewReactor(...)` to the workers in `cmd/catchup/main.go`.
- **A scheduler:** implement `scheduler.Interface`. Its `Tick` reads a todo list, a projection, and
  sends a command with a fixed ID for each item that is due. Add `scheduler.New(...)` to the workers in
  `cmd/catchup/main.go`.

The name of each consumer is the key of its checkpoint. Do not rename a consumer after it runs in
production: a projection with a new name rebuilds from the beginning, and a reaction with a new name
starts again at the end of the store.

The name is also the key of a lock. Each consumer and each scheduler runs in one `catchup` process at a
time. A second
process, for example during a rolling deploy, waits until the first process stops.
[docs/event-subscription.md](docs/event-subscription.md#one-process-for-each-consumer) tells how.
{{- range items "readme.sections" }}

{{ .text }}
{{- end }}
