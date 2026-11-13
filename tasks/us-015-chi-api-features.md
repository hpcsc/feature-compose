# Tasks: Make a chi-api Repo With the Same Features (US-015)

## Contents

1. Task 1: Add the starter `chi-api`, an HTTP API whose routes are under `/api` and that checks no credential
2. Task 2: Make `api-auth` target `chi-api` through `api.middleware` and `api.config`, with one text of its docs for each target
3. Task 3: Make `cli` target `chi-api` through the extension points of the tasks, the tools, CI, `go.mod` and `.gitignore`
4. Task 4: Make `cli-auth` target `chi-api`, with one text of its docs for each target

## Open Questions

The tasks follow the recommendation of each question. Overrule a recommendation before Task 2 starts.

1. **Can the docs of `api-auth` and `cli-auth` change, so that they read true in a `chi-api` repo?** The third
   criterion says: "The only change to a feature for `chi-api` is the new target." The fifth says that the docs of
   `api-auth` and `cli-auth` name one path for both starters. Today the docs name parts that only
   `go-event-sourcing-service` has:
   - `docs/api-auth.md` of both methods, in "Run the API on this machine": the pane of `task run` that runs
     `task api:run`, the requests of `Taskfile.api.yml` such as `task api:open-account`, `.env.local`, and the
     `api` service of `docker-compose.local.yml`
     (`catalog/features/api-auth/files/api-key/go/docs/api-auth.md:33-49`,
     `catalog/features/api-auth/files/oauth/go/docs/api-auth.md:32-48`). In "Send a request", the example is
     `POST /api/accounts` with `{"owner": "Jane"}` (`:51-60` and `:50-60`).
   - `docs/cli-auth.md` of both methods: the example call of `client.Do` to the route `/accounts`
     (`catalog/features/cli-auth/files/api-key/go/docs/cli-auth.md:92`,
     `catalog/features/cli-auth/files/oauth/go/docs/cli-auth.md:106`).

   A `chi-api` repo has none of them. To keep these docs with no change, the starter `chi-api` must get a tmux
   window, a request task `api:open-account`, `.env.local`, a Docker Compose file and an account route, which a
   plain HTTP API does not need. **Recommendation: one text of each doc for every target.** The docs name only
   what each target has: `task api:run`, which both starters have (decision 3), the variables of the API, the
   routes under `/api` on port 3333, `GET /api/`, and `README.md` of the repo for the rest of the local run. An
   example request or call names a placeholder route. The breakdown reads the third criterion as "no change of a
   feature that only `chi-api` needs": the new text is the same for each target, so the catalog keeps one set of
   files, and the Context of the story puts such a fix into the feature (rule 6). A `go-event-sourcing-service`
   repo loses the hints about the pane, the request tasks and Docker Compose in these docs; its `README.md` still
   tells how to give the local API a setting with `task api:run`
   (`catalog/starters/go-event-sourcing-service/files/go/README.md:80-84`). The Go files of the three features
   do not change.

   If the caller reads the third criterion literally: Tasks 2 and 4 change only the `targets`. Then the docs of a
   `chi-api` repo name four files and two tasks that the repo does not have, and the test of the docs of
   `api-auth` and `cli-auth` runs only for `go-event-sourcing-service`.

2. **Does a `chi-api` repo get CORS, as US-019 gave `go-event-sourcing-service`?** **Recommendation: no.** No
   criterion of US-015 names it. The fifth criterion takes from US-019 only the routes under `/api`, and the
   template `chi-api` has no CORS. If the caller wants it, the smallest change is one task after Task 1 that
   ports `middleware.CORS`, its test and `ALLOWED_ORIGINS` from `go-event-sourcing-service`
   (`catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/cors.go.tmpl`) into
   `internal/middleware` of `chi-api`, with its section of the README.

## Story Reference

US-015, "Make a chi-api repo with the same features", in `user-stories/feature-compose.md:511-537` (Description,
Acceptance Criteria, Context). The design is `PROPOSAL.md`: sections 8 (options), 9 (templates, and how `server.go`
renders `api.middleware`) and 10 (rules 2, 3, 5, 6 and 7). The reference for the files of the starter is the
template `chi-api` of `repo-templates` (`/Users/davidnguyen/repo-templates/templates/chi-api/`, read-only). The tool
and the catalog do not depend on it.

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

- A per-starter copy of a file of a feature, and a `when` on the starter in a feature (decision of the user).
- A change to the Go files of `api-auth`, `cli` and `cli-auth`. None of them imports a package of
  `go-event-sourcing-service` (Codebase Context), so no rule 6 fix is needed in code.
- CORS and `ALLOWED_ORIGINS` in `chi-api` (open question 2).
- `internal/resilient` and the question `EnableResilientHttp` of the template: an HTTP client with retries and a
  circuit breaker, which no criterion names and which the API does not call. It can become a feature of its own.
- The extension points of `go-event-sourcing-service` that no feature that targets `chi-api` adds to, for example
  `api.routes`, `api.tasks` and `readme.sections`. `chi-api` declares the seven points that `api-auth`, `cli` and
  `cli-auth` add to.
- `process-manager` as a feature of `chi-api`. It needs the event store, and only `go-event-sourcing-service`
  declares its extension points (rule 2).
- A task `run` and a task `build` at the top of the Taskfile of `chi-api`. `task api:run` starts the API, and
  `cli` owns `build` (decision 3).
- The steps of the hook `post_scaffold` of the template (`go get`, `mise use`) and its "All set" message, as US-001
  and US-016 left them out.
- A GoReleaser snapshot and `task test:e2e:local` of a `chi-api` repo with `cli` in the e2e tests of
  feature-compose. Task 3 proves that `cli` writes the same files into each target, and the full run of `go` with
  `cli` runs both. The Verification of Tasks 3 and 4 runs `task test:e2e:local` of a written repo one time.
- An e2e test of a `chi-api` repo with `cli-auth:store-key=true` or with the method `oauth` of `cli-auth`. The
  combinations of CI build and test both, and Task 4 proves that their files are the same as for
  `go-event-sourcing-service`.

**Deferred:** none of the criteria of this story.

## Codebase Context

**The starters.** `catalog/starters/go/` (US-016) is the closest precedent for a new starter: `starter.yml` with the
options `go-version` and `module`, the folder `go` with `after_render: [go mod tidy, gofmt -w .]` and
`test: [task test:unit]`, five extension points (`taskfile.flat_includes`, `mise.tools`, `ci.jobs`,
`gomod.requires`, `gitignore.patterns`), and `[[` `]]` delimiters for the Taskfiles and `.github/`
(`catalog/starters/go/starter.yml:1-45`). `go-event-sourcing-service` declares `api.middleware` with the item
`[code, import]` in `go/internal/api/server/server.go`, and `api.config` with the item `[name, type, import]` in
`go/internal/api/config.go` (`catalog/starters/go-event-sourcing-service/starter.yml:42-49`). `server.go.tmpl`
imports the package of each item one time (`:17-19`) and gives each `code` to the protected routes in
`protectedMiddleware(cfg)` (`:48-54`). `config.go.tmpl` adds a field and an import for each item of `api.config`
(`:1-17`). `Register` serves each route under `/api`, with a public group and a protected group that gets the
middleware of the protected routes (`catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:20-44`),
and its test checks both (`register_test.go.tmpl:18-108`).

**What the three features need of a target.**

| Feature | Reads | Adds items to | Writes |
| --- | --- | --- | --- |
| `api-auth` | `starter.module` (tests) | `api.middleware` (`auth.Middleware(cfg.Auth)`, `internal/auth`), `api.config` (`Auth auth.Config`) | `internal/auth/`, `docs/api-auth.md` |
| `cli` | `starter.module`, `starter.go-version` | `taskfile.flat_includes`, `mise.tools`, `gomod.requires`, `ci.jobs`, `gitignore.patterns` | `cmd/{{ kebab .Name }}/main.go`, `internal/cli/`, `.goreleaser.yaml`, `.dockerignore`, `Taskfile.cli.yml`, `scripts/install.sh`, `e2e/`, `docs/cli.md`, `docs/e2e-tests.md` |
| `cli-auth` | `starter.module`, `api-auth.method` | `cli.commands` (of `cli`), `gomod.requires` | `internal/credential/`, `internal/apiclient/`, `internal/cli/auth.go`, `e2e/tests/auth.test.ts`, `docs/cli-auth.md` |

- The Go files of the three features import only their own packages, `cli`, the standard library and modules.
  `api-auth` writes its refusal in `refuse.go` of `files/common` with its own JSON body, not with the `response`
  package of the starter.
- The item `auth.Middleware(cfg.Auth)` needs a variable `cfg` with a field `Auth` where the server renders it. The
  config loader must be `caarlos0/env/v6`, which reads the env tags of `auth.Config` and calls `UnmarshalText`
  (learning). The test `issuer_url_test.go` of oauth imports `github.com/caarlos0/env/v6`.
- `Taskfile.cli.yml` has the tasks `build`, `cli`, `release:local`, `test:e2e` and `test:e2e:local`. The jobs
  `release` and `prerelease` that `cli` adds to `ci.yml` need the jobs `test` and `e2e`.
- `cli-auth` reads `go.yaml.in/yaml/v3` in `files/common`, and no item of `gomod.requires` gives it; the `go.mod`
  of each target requires it.
- `cli-auth` needs `cli` and `api-auth`, so it can target `chi-api` only after both do (rule 3).

**The template `chi-api`, and its place in the starter.**

| File of the template | In the starter `chi-api` |
| --- | --- |
| `cmd/{{.ProjectKebab}}/main.go` | `cmd/api/main.go`, because `cli` writes `cmd/<name>/main.go` (decision 2) |
| `internal/config/config.go` | The same path, without `TokenPath` (decisions 5 and 6) |
| `internal/server/server.go` | The same path. It gives `Register` the middleware of the protected routes: none in Task 1, the items of `api.middleware` from Task 2 |
| `internal/usecase/register.go`, `internal/usecase/root/`, `internal/usecase/user/` | `internal/use_cases/` (decision 8). `Register` serves the routes under `/api`, and takes the middleware of the protected routes (decision 4) |
| `internal/middleware/auth.go`, `auth_test.go`, `testdata/valid-token`, `local/token` | None (decision 5) |
| `internal/middleware/request_logger*.go`, `internal/middleware/request/`, `internal/response/`, `internal/route/`, `internal/test/` | The same paths |
| `internal/resilient/http.go` | None (Boundaries) |
| `Taskfile.yml` | The include of the tests and of `Taskfile.api.yml`, `hook:setup`, `hook:validate-go-changes`. `run`, `build`, `docker:build` and `docker:run` move to `Taskfile.api.yml` (decision 3) |
| `Taskfile.test.yml` | `unit`, `shellcheck`, `vulnerabilities` |
| `Dockerfile` | The same path. It builds `./cmd/api` (decision 11) |
| `.github/workflows/ci.yml` | The workflow of the starter `go`, with the job `test` (decision 10) |
| `go.mod` | The modules of the hook at fixed versions (decision 9) |
| `hooks/pre-push` | The same, with `scripts/libs/git-hook-prepush.sh` of the starter `go`, which the hook sources and the template does not have |
| `.gitignore` | The template's lines and `bin`, as the starter `go` has |
| `README.md` | A README of the API (decision 12) |
| The tools of `hooks/post_scaffold` | `mise.toml` of the starter `go` |
| (none) | `starter.yml`, and `internal/use_cases/register_test.go` (decision 4) |

**What reads the list of starters.** `show`, the form of `new`, `check` and `Catalog.Combinations` take the starters
from the folders under `starters/`, so a new starter needs no change to the tool. `chi-api` sorts before `go`, so
the e2e tests that pick a starter by its place change (learning): the message of an unknown starter
(`e2e/tests/show.test.ts:37-46`, `:149-157`), the first screen test of `show`, which opens on the first starter and
reads the whole tree (`e2e/tests/show.test.ts:49-69`), and the form of `new` with no argument, which presses `j` in
the list of starters (`e2e/tests/new.test.ts:935-972`). Each extension point and each feature of a starter is one
line of the open tree, and the tree of `go` and `go-event-sourcing-service` already needs about 43 of the 60 rows of
`rowsOfTheWholeTree` (learning). `chi-api` adds about 14 rows by the end of Task 4.

**Facts checked for this breakdown:**

- A module with the five modules of the template at the versions of `go-event-sourcing-service`
  (`caarlos0/env/v6 v6.10.1`, `go-chi/chi/v5 v5.3.2`, `gookit/validate v1.6.0`, `stretchr/testify v1.12.1`,
  `unrolled/render v1.8.1`), tidied with `GOPROXY=off` from the module cache, gets the indirect requires
  `fsnotify/fsnotify v1.10.0`, `gookit/filter v1.2.3`, `gookit/goutil v0.7.6`, `go.yaml.in/yaml/v3 v3.0.5`,
  `golang.org/x/sys v0.43.0` and `golang.org/x/text v0.22.0`. Its module graph selects `golang.org/x/term v0.29.0`,
  the version of the item of `cli-auth`, so `go mod tidy` removes that item where no file imports it (learning).
- The production files `request_logger.go`, the package `request`, `response.go`, the package `route`,
  `root/handler.go` and `test/fake_log_handler.go` of the template are the files of `go-event-sourcing-service` in
  the catalog with another import path. Their tests and `test/common.go` differ. So the JSON body of a refusal of
  `api-auth`, `{"successful": false, "messages": [...]}`, has the keys of the error answer of the handlers of
  `chi-api` (`response.Fail`), as US-019 needs.
- Of the tests of the template, only `user/handler_test.go` and `response/response_test.go` have the build tag
  `unit`. `request_logger_test.go` and `route_matcher_test.go` have no tag, so `-tags=unit` also runs them.
- The template has no `.DS_Store` file, and scaffold 0.13.1 is installed. Docker answered `docker info` on
  2026-11-14.

**Words.** `chi-api` is the name of the starter, not `chi-service` or `http-api`. *The API* is the program in
`cmd/api`, as in `go-event-sourcing-service`. A *public route* and a *protected route* are `route.Public` and
`route.Protected`; *the middleware of the protected routes* is what `Register` gives the protected group. The
example use case of the template is *user*. A feature *targets* a starter when its `targets` name it.

**Decisions of this breakdown, for the caller to overrule:**

1. **What the starter `chi-api` holds** (Task 1): the files that the table gives to the starter. It owns no path
   that `api-auth`, `cli` or `cli-auth` writes (rule 7), and no `.dockerignore`, which `cli` writes.
2. **The program of the API is `cmd/api`** (Task 1), as in `go-event-sourcing-service`. `cli` owns
   `cmd/<name>/main.go`, so the template path cannot stay. With one path, `startedApi` of the e2e tests builds the
   API of both starters, and a repo named `api` with `cli` stops the write in both, as it does today for
   `go-event-sourcing-service`.
3. **The tasks of the API are in `Taskfile.api.yml`** (Task 1), which `Taskfile.yml` includes as `api`, as in
   `go-event-sourcing-service`: `api:run` (`go run ./cmd/api`), `api:build` (`bin/api`, with the ldflag of
   `root.Version`), `api:docker:build` and `api:docker:run` (with no mount of a token). The template task `build`
   would have the name of a task of `cli`, which stops Task (learning), and `task api:run` is then one command for
   the docs of `api-auth` in both starters (open question 1).
4. **The routes under `/api`** (Task 1). `Register` serves each route of each `route.Routable` under `/api`, with
   the public routes in one group and the protected routes in a group that gets the middleware of the protected
   routes as an argument, as `register.go.tmpl` of `go-event-sourcing-service` does. The routes of the example are
   `GET /api/` (public, the version) and `POST /api/users` (protected, the validation example of the template). A
   new `register_test.go` checks the groups and the prefix with a middleware of its own, as
   `register_test.go.tmpl` of `go-event-sourcing-service` does.
5. **No credential check** (Task 1). The starter has no `internal/middleware/auth.go`, its test and test data, no
   `local/token`, no `TokenPath` or `TOKEN_PATH`, and no mount of `./local` in `api:docker:run`. Without `api-auth`,
   each request to a protected route reaches its handler, as US-001 decided for `go-event-sourcing-service`.
6. **The config** (Task 1): `config.Config` in `internal/config`, with `Port` (`PORT`, default `3333`), read by
   `config.Load` with `caarlos0/env/v6`, the loader of `go-event-sourcing-service`. Port 3333 is the port of the
   docs of `api-auth` and `cli-auth`.
7. **The options** (Task 1): `module` and `go-version`, with the default and the pattern of the other starters. `cli`
   and `cli-auth` read both (rule 5), and with the same defaults each feature writes the same files to each target.
   No option for `EnableResilientHttp`.
8. **Package names** (Task 1): `internal/usecase` becomes `internal/use_cases`, a package of two words in snake case
   (learning). `internal/test` keeps `FakeLogHandler`, as `internal/common/test` of `go-event-sourcing-service`
   does.
9. **`go.mod`** (Task 1): the five modules of the hook at the versions of `go-event-sourcing-service`, and the
   indirect requires of the fact above. So `go mod tidy` takes no latest version, and the module graph gives
   `cli-auth` the versions that it gets in `go-event-sourcing-service`.
10. **`mise.toml`, `.gitignore`, `ci.yml`, the hook and its script** (Task 1): those of the starter `go`. The CI job
    is `test`, because the jobs of `cli` need it, and it runs `go build ./...` where the template runs `task build`.
11. **`Dockerfile`** (Task 1): the template's, with `./cmd/api` and the binary `api`.
12. **The README** (Task 1) names the module, `mise install`, `task api:run` and that it passes the variables of the
    shell to the API, the routes under `/api`, the unit tests and the hook. It says nothing about a credential,
    because a feature can add one.
13. **The render of `api.middleware` and `api.config`** (Task 2): `internal/server/server.go` imports the package of
    each item of `api.middleware` one time and gives each `code`, in the order of the items, to the protected routes
    in `protectedMiddleware(cfg)`; `internal/config/config.go` adds a field and an import for each item of
    `api.config`. The summaries name `cfg, the *config.Config of the API`, and the item fields are those of
    `go-event-sourcing-service`.
14. **The five extension points of `cli`** (Task 3): the summaries, the item fields and the renders of the starter
    `go`, in `Taskfile.yml`, `mise.toml`, `.github/workflows/ci.yml`, `go.mod` and `.gitignore`.
15. **The docs of `api-auth` and `cli-auth`** (Tasks 2 and 4): open question 1.
16. **"Its unit tests pass"** (Task 1) and **"builds and passes its tests" in CI** (Tasks 2 to 4) mean the `test`
    command of the folder, `task test:unit`, which the job `combinations` of this repo runs for each combination.
17. **The files that are ported** (for the audit). From the template, through a render with sentinel values: the
    files of the table, with the changes of decisions 2 to 11. From the starter `go` with no change:
    `scripts/libs/git-hook-prepush.sh`, `mise.toml`, the CI workflow, and the extension points of Task 3. New:
    `starter.yml`, `README.md`, `Taskfile.api.yml` and `register_test.go`.

## Tasks

### Task 1: Add the starter `chi-api`, an HTTP API whose routes are under `/api` and that checks no credential

**Behavior:** `feature-compose new chi-api my-api --no-prompt` makes a chi HTTP API from the starter `chi-api`: the
API in `cmd/api`, its config, server, middleware, routes, response package and the example use cases `root` and
`user` with their unit tests, the Taskfiles, `Dockerfile`, `mise.toml`, `.gitignore`, the CI workflow, the pre-push
hook and its script, and a README (decisions 1 to 12). The `after_render` commands run, the module builds, and
`task test:unit` passes. The API serves each route under `/api`, and a request to a protected route reaches its
handler with no credential. `show`, the form of `new`, `check` and `task test:combinations` take the third starter
from the catalog with no change to the tool, and the e2e tests that pick a starter by its place work with `chi-api`
first.

**Acceptance Criteria:**
- [x] `catalog/starters/chi-api/starter.yml` has a `summary` that says what the repo gets; the options `module` and
  `go-version` with the default and the pattern of `go-event-sourcing-service`; the folder `go` with `path: .`,
  `after_render: [go mod tidy, gofmt -w .]` and `test: [task test:unit]`; and `[[` `]]` delimiters for the
  Taskfiles and for `.github/`.
- [x] A test of the group `embedded` of `TestCatalog` writes `chi-api` with the name `my-api` and no given values. The
  repo has the files of decision 1 and no other file. `go.mod` has `module github.com/hpcsc/my-api`, `go 1.27.1` and
  a fixed version of each module that a Go file of the repo imports, other than the standard library and the module
  of the repo; `mise.toml` has `1.27.1`; and `cmd/api/main.go` exists while no path starts with `cmd/my-api`.
- [x] Written with `module=github.com/acme/ledger` and `go-version=1.28.0`, `go.mod`, `mise.toml`, `Dockerfile`,
  `Taskfile.api.yml` and each import of a package of the module have those values, and no file holds
  `github.com/hpcsc/my-api` or `1.27.1`.
- [x] The Taskfiles and the workflow keep the template syntax of Task and of GitHub Actions. `hooks/pre-push` and
  `scripts/libs/git-hook-prepush.sh` can be run. No written file holds a value of scaffold (`.Scaffold.`,
  `.ProjectKebab`, `<no value>`, `zzsentinel`).
- [x] The repo has no credential check: no `internal/middleware/auth.go`, no test data of a token and no
  `local/token`, and no written file holds `token`, `bearer` or `authorization` in any case, other than the GitHub
  token of the workflow.
- [x] The tasks of the repo, from its Taskfiles, have `api:run`, `api:build`, `api:docker:build`, `test:unit`,
  `hook:setup` and `hook:validate-go-changes`, and none of `build`, `cli`, `release:local`, `test:e2e` and
  `test:e2e:local`.
- [x] The README of the written repo names its module and tells how to install the tools, run the API with
  `task api:run` and give it a setting from the shell, run the unit tests and set up the hook. Each path in
  backticks in it is a file or a folder of the repo, and each `task <name>` in it is a task of its Taskfiles.
- [x] The test of `TestCatalog` that reads the routes of the API also reads a `chi-api` repo: its README and
  Taskfiles name each route under `/api`, and do not name `GET /` alone.
- [x] In the written repo, the unit test of `Register` passes: a request to `GET /api/` reaches its handler, a
  request to `POST /api/users` goes through the middleware that the test gives for the protected routes, and each
  route that `Register` serves is under `/api`.
- [x] `e2e/tests/new.test.ts`: `feature-compose new chi-api my-api --no-prompt` exits with 0. In `./my-api`,
  `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits with 0, and its lines that
  start with `✓` name each folder of the repo that holds a test file. `task --list-all` names `api:run` and each task
  that `.github/workflows/ci.yml` and `scripts/libs/git-hook-prepush.sh` call, and no task of `cli`. The test starts
  the API: `GET /api/` gets 200, `POST /api/users` with `{}` and no `Authorization` header gets the answer 400 of the
  handler, and `POST /users` gets 404.
- [x] The e2e tests of `show` and of the form of `new` pass with three starters. An unknown starter lists
  `chi-api, go, go-event-sourcing-service`. The tree of `show` starts with `chi-api`, and the two tests that read the
  whole tree see its last line. The form of `new` with no argument still selects `go-event-sourcing-service`.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass. `task test:combinations`
  passes, and its output names the combination `chi-api`.
- [x] `README.md` of this repo names `chi-api` among the starters of its catalog. `docs/e2e-tests.md` tells the full
  run of `chi-api`: what it checks and what it needs.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/chi-api/starter.yml` — new
- `catalog/starters/chi-api/files/go/` — new: `go.mod.tmpl`, `.gitignore`, `mise.toml`, `Taskfile.yml`,
  `Taskfile.api.yml`, `Taskfile.test.yml`, `Dockerfile`, `.github/workflows/ci.yml`, `hooks/pre-push`,
  `scripts/libs/git-hook-prepush.sh`, `README.md`, `cmd/api/`, `internal/config/`, `internal/server/`,
  `internal/middleware/`, `internal/route/`, `internal/response/`, `internal/use_cases/`, `internal/test/`
  (`.go.tmpl` files)
- `internal/catalog/catalog_test.go` — the group `embedded`: the repo of `chi-api`, and the test of the routes under
  `/api`
- `e2e/tests/new.test.ts` — the full run of `chi-api`, and the form with no argument
- `e2e/tests/show.test.ts` — the unknown starter, the first starter of the tree, and the rows of the whole tree
- `docs/e2e-tests.md` — the full run of `chi-api`
- `README.md` — the starters of the catalog of this repo

**Patterns to Follow:**
- `catalog/starters/go/starter.yml:1-15` for `starter.yml`
- `catalog/starters/go/files/go/go.mod.tmpl:1-8`, `catalog/starters/go/files/go/mise.toml:1-9`,
  `catalog/starters/go/files/go/.gitignore:1-26`, `catalog/starters/go/files/go/.github/workflows/ci.yml:1-35`,
  `catalog/starters/go/files/go/Taskfile.yml:1-18` and `catalog/starters/go/files/go/Taskfile.test.yml:1-15` for
  the files of a starter
- `catalog/starters/go/files/go/hooks/pre-push:1-18` and
  `catalog/starters/go/files/go/scripts/libs/git-hook-prepush.sh:1-107` for the hook
- `catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl:1-29` for the versions and the indirect requires
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml:38-63` for the tasks of the API
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:20-44` and
  `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl:18-108` for the
  routes under `/api` and their test
- `catalog/starters/go-event-sourcing-service/files/go/internal/common/config/loader.go.tmpl:1-15` for the loader
- `catalog/starters/go/files/go/README.md:1-42` and
  `catalog/starters/go-event-sourcing-service/files/go/README.md:70-88` for the README
- `internal/catalog/catalog_test.go:3197-3287` for the tests of a written starter
- `internal/catalog/catalog_test.go:1637-1649` for the repo with no credential check
- `internal/catalog/catalog_test.go:2745-2794` for the routes under `/api` in the docs
- `internal/catalog/catalog_test.go:2840-2853` for the tasks of a target
- `internal/catalog/catalog_test.go:1471-1476` for the check of the catalog of this repo
- `e2e/tests/new.test.ts:43-84` and `e2e/tests/new.test.ts:86-114` for the full run, the API that the test starts,
  and the packages that passed
- `e2e/tests/new.test.ts:1060-1096` for `startedApi`
- `e2e/tests/show.test.ts:37-69`, `e2e/tests/show.test.ts:149-157` and `e2e/tests/new.test.ts:935-972` for the tests
  that pick a starter by its place
- `docs/e2e-tests.md:80-138` for the docs of a full run
- `README.md:50-58` and `README.md:132-144` for the starters of the catalog of this repo
- `tasks/learnings.md:10-15` for `.tmpl` and `#!`
- `tasks/learnings.md:17-22` for the delimiters globs
- `tasks/learnings.md:24-29` for the port with sentinel values
- `tasks/learnings.md:31-36` for the old defects of ported code (decision 17)
- `tasks/learnings.md:38-43`, `tasks/learnings.md:52-57` and `tasks/learnings.md:59-64` for the e2e run of a written
  repo
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134`, `tasks/learnings.md:367-372` and
  `tasks/learnings.md:479-484` for the proof of template code by breaks by hand
- `tasks/learnings.md:108-113` for an API that the e2e test starts
- `tasks/learnings.md:157-162` for a Go composite literal in a `.go.tmpl`
- `tasks/learnings.md:227-232`, `tasks/learnings.md:346-351` and `tasks/learnings.md:395-400` for the screen tests of
  `show` and the form
- `tasks/learnings.md:381-386` for the names of the packages and of the fakes
- `tasks/learnings.md:409-414` for the task names of a starter and a feature
- `tasks/learnings.md:290-295` for `go vet -tags combinations`

**Testable:** Yes

**Certainty:** medium — `catalog/starters/go/` with its tests (`catalog_test.go:3197-3287`) is the precedent for a new
starter, and `register.go.tmpl:20-44` for the routes under `/api`; but `chi-api` is the first starter that sorts
before `go`, so the screen tests of `show` and the form of `new` change, and the port moves the program and the tasks
of the template to new paths

**Blast radius:** high — authentication and a public API shape: the task decides that a protected route of a
`chi-api` repo accepts each request, builds the split of public and protected routes that `api-auth` relies on, and
sets the paths under `/api` that the clients of the API call

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write `chi-api` with the built binary into a scratch folder, and run `task test:unit`, `task test:shellcheck`
and `task test:vulnerabilities` there, and `task api:docker:build` when `docker info` answers. Apply one break at a
time to the written `internal/use_cases/register.go`, read its diff, run
`go test -tags unit -timeout 60s ./internal/use_cases/`, and put the file back: serve the routes at the root path;
give the protected group no middleware with `&& false`; register a protected route in the public group. Each break
makes a test fail by name. With the binary of HEAD, write `go` and `go-event-sourcing-service` into a second folder,
and `diff -r` them with the repos of the work tree: no difference. Run `task test:combinations`, and the e2e tests of
`new`, `show` and `check` on the host.

**Depends on:** None

---

### Task 2: Make `api-auth` target `chi-api` through `api.middleware` and `api.config`, with one text of its docs for each target

**Behavior:** `chi-api` declares `api.middleware` and `api.config` with the item fields of
`go-event-sourcing-service`. Its server imports the package of each item of `api.middleware` one time and gives
each item, in their order, to the protected routes; its config has a field for each item of `api.config`
(decision 13). `catalog/features/api-auth/feature.yml` targets `chi-api` and `go-event-sourcing-service`, with its
files entries as they are. `feature-compose new chi-api my-api --feature api-auth` writes a repo whose API answers
401 to a request to a protected route without a credential that it accepts, with each method. `docs/api-auth.md`
has one text for every target, which names only what the repo of each target has (open question 1).

**Acceptance Criteria:**
- [x] `catalog/starters/chi-api/starter.yml` declares `api.middleware` with the item `[code, import]` in
  `go/internal/server/server.go`, and `api.config` with the item `[name, type, import]` in
  `go/internal/config/config.go`, each with a summary that names what the item can use.
- [x] The `targets` of `catalog/features/api-auth/feature.yml` name `chi-api` and `go-event-sourcing-service`, and
  its options, files entries and `add_to` do not change.
- [x] With a feature of a test catalog that targets `chi-api` and adds two items to `api.middleware` and one to
  `api.config`, the server of the written repo gives the two items in their order to the protected routes and
  imports their package one time, and the config has the field and imports its package one time. With no item, the
  server gives no middleware to the protected routes, the config has only its own fields, and no written file holds
  the name of an extension point.
- [x] A test of the group `embedded` takes the targets of `api-auth` from its `feature.yml`. For each target and each
  method, `api-auth` adds the files of the method, and of the files of the target it changes only the file of each
  extension point that it adds items to. For each two targets and each method, the files that `api-auth` adds have
  the same paths and the same contents.
- [x] For each target of `api-auth` and each method, `docs/api-auth.md` names the settings of the method,
  `Authorization: Bearer`, the answers 401 with `WWW-Authenticate: Bearer`, `` `GET /api/` is public ``, the JSON body
  of the refusal, and how to start the API on this machine with the settings through `task api:run`. Each file and
  each folder that it names in backticks is in the repo of that target, and each `task <name>` that it names is a
  task of that repo.
- [x] The test of `TestCatalog` that reads the routes of the API reads a `chi-api` repo with `api-auth` of each
  method: its docs name each route under `/api`, and do not name `GET /` alone.
- [x] `e2e/tests/new.test.ts`: `feature-compose new chi-api my-api --feature api-auth --no-prompt` exits with 0. In
  `./my-api`, `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits with 0 and its
  lines that start with `✓` name `internal/auth` and `internal/use_cases/user`. The test starts the API with
  `API_KEY_HASHES` of a key of the test. `GET /api/` gets 200 with no key. `POST /api/users` gets 401 with
  `WWW-Authenticate: Bearer` with no `Authorization` header and with an unknown key, and the answer 400 of the
  handler with the key. The body of the 401 is JSON with the keys of the body of the 400, `successful` false, and a
  message that tells the client to sign in. `go.mod` of the repo is the `go.mod` of a `chi-api` repo without
  `api-auth`.
- [x] `e2e/tests/new.test.ts`: with `--feature api-auth:method=oauth`, the API that the test starts with the test
  issuer and the audience `my-api` answers `POST /api/users` with 401 with no token and with a token for another
  audience, and with the answer 400 of the handler with a token for `my-api`.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass. `task test:combinations`
  passes, and its output names `chi-api --feature api-auth` and `chi-api --feature api-auth:method=oauth`.
- [x] The e2e tests of `show` pass, and the two tests that read the whole tree see its last line.
- [x] `README.md` of this repo gives `chi-api` and `go-event-sourcing-service` as the targets of `api-auth`.
  `docs/e2e-tests.md` tells the two full runs of `chi-api` with `api-auth`.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/chi-api/starter.yml` — `api.middleware` and `api.config`
- `catalog/starters/chi-api/files/go/internal/server/server.go.tmpl` — the render of `api.middleware`
- `catalog/starters/chi-api/files/go/internal/config/config.go.tmpl` — the render of `api.config`
- `catalog/features/api-auth/feature.yml` — the target `chi-api`
- `catalog/features/api-auth/files/api-key/go/docs/api-auth.md`, `catalog/features/api-auth/files/oauth/go/docs/api-auth.md`
  — one text for every target
- `internal/catalog/catalog_test.go` — the points of `chi-api`, the files and the docs of `api-auth` for each target,
  and the routes under `/api`
- `e2e/tests/new.test.ts` — the full runs of `chi-api` with `api-auth` of each method
- `docs/e2e-tests.md` — the two full runs
- `README.md` — the targets of `api-auth`

**Patterns to Follow:**
- `task:1` for the starter `chi-api` and its full run
- `catalog/starters/go-event-sourcing-service/starter.yml:42-49` for the two extension points
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl:17-19` and
  `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl:48-54` for the render of
  `api.middleware`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl:1-17` for the render of
  `api.config`
- `catalog/features/api-auth/feature.yml:1-23` for the `feature.yml` of `api-auth`
- `internal/catalog/catalog_test.go:1651-1700`, `internal/catalog/catalog_test.go:1725-1743` and
  `internal/catalog/catalog_test.go:1745-1759` for the items of the two points and the repo with no item
- `internal/catalog/catalog_test.go:2029-2070` for the files that `api-auth` adds
- `internal/catalog/catalog_test.go:2258-2346` for the tests that take the targets of a feature from its `feature.yml`
- `internal/catalog/catalog_test.go:2211-2245` for the docs of `api-auth`
- `internal/catalog/catalog_test.go:2818-2838` for the docs of a feature that name only the tasks and the paths of
  each target
- `internal/catalog/catalog_test.go:2745-2794` for the routes under `/api` in the docs
- `catalog/starters/go-event-sourcing-service/files/go/README.md:80-84` for the README that tells how to give the
  local API a setting
- `e2e/tests/new.test.ts:196-245` and `e2e/tests/new.test.ts:247-288` for the full runs with `api-auth`
- `e2e/tests/new.test.ts:1151-1216`, `e2e/tests/new.test.ts:1271-1285` for the test issuer and the answers of the API
- `docs/e2e-tests.md:139-170` for the docs of the full runs with `api-auth`
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134`, `tasks/learnings.md:367-372` and
  `tasks/learnings.md:479-484` for the proof by breaks by hand
- `tasks/learnings.md:94-99` for the cost of the proof of the code of a feature
- `tasks/learnings.md:101-106` for the config loader and `UnmarshalText`
- `tasks/learnings.md:108-113` for the API that the e2e test starts
- `tasks/learnings.md:115-120` for `task api:run` in the docs
- `tasks/learnings.md:122-127` for a test that reads an environment variable
- `tasks/learnings.md:325-330` for the headers of an answer in a Go test
- `tasks/learnings.md:395-400` for the rows of the whole tree
- `tasks/learnings.md:500-505` for a change of a template with a repo written from the commit before
- `tasks/learnings.md:290-295` for `go vet -tags combinations`

**Testable:** Yes

**Certainty:** medium — `server.go.tmpl:17-19,48-54` and `config.go.tmpl:1-17` of `go-event-sourcing-service` are the
precedent for the two renders, and `catalog_test.go:2258-2346` for a feature with two targets; but no doc of a
feature yet names the local run of a starter for two starters whose local runs differ, and the e2e tests of
`api-auth` run against a second API with another example route

**Blast radius:** high — authentication: the render of `api.middleware` in the server of `chi-api` decides whether
each request to a protected route of a `chi-api` repo with `api-auth` goes through `auth.Middleware`

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write `chi-api` with `--feature api-auth`, and with `--feature api-auth:method=oauth`, with the built binary,
and run `task test:unit` and `task test:vulnerabilities` there. With the binary of HEAD, write
`go-event-sourcing-service` with `api-auth` of each method into a second folder, and `diff -r` them with the repos of
the work tree: only `docs/api-auth.md` differs. Read the rendered `docs/api-auth.md` of both starters and both
methods. Apply one break at a time to `catalog/starters/chi-api/files/go/internal/server/server.go.tmpl`, read its
diff, run the e2e test of `chi-api` with `api-auth` and `go test -tags unit -run TestCatalog/embedded ./internal/catalog/`,
and copy the file back from a copy made before the break: give the protected routes an empty list with `[:0]`; write the items in
the reverse order. Each break makes a test fail by name: the first the e2e test, the second the test with the test
catalog. Run `task test:combinations`, and the e2e tests of `new`, `show`
and `check` on the host.

**Depends on:** Task 1

---

### Task 3: Make `cli` target `chi-api` through the extension points of the tasks, the tools, CI, `go.mod` and `.gitignore`

**Behavior:** `chi-api` declares `taskfile.flat_includes`, `mise.tools`, `ci.jobs`, `gomod.requires` and
`gitignore.patterns` with the summaries and the item fields of the starter `go`, and its `Taskfile.yml`, `mise.toml`,
`ci.yml`, `go.mod` and `.gitignore` write their items as the files of `go` do (decision 14).
`catalog/features/cli/feature.yml` targets `chi-api`, `go` and `go-event-sourcing-service`, with no change to its
files. `feature-compose new chi-api my-api --feature cli` writes the repo of `chi-api` and the files of `cli`: the
CLI builds, its tasks sit beside the tasks of the API with no name in common, and `task test:unit` runs the tests of
the API and of the CLI.

**Acceptance Criteria:**
- [x] `catalog/starters/chi-api/starter.yml` declares `taskfile.flat_includes`, `mise.tools`, `ci.jobs`,
  `gomod.requires` and `gitignore.patterns` with the summaries, the files and the item fields of
  `catalog/starters/go/starter.yml`.
- [x] The `targets` of `catalog/features/cli/feature.yml` name `chi-api`, `go` and `go-event-sourcing-service`, and
  no other line of the file changes.
- [x] The tests of the group `embedded` that take the targets of `cli` from its `feature.yml` pass with `chi-api`: the
  files that `cli` adds to each target, and the same files with the same contents for each two targets, with no
  given values and with a given module and Go version; the docs of `cli`, which name only tasks and paths of each
  target; the tasks of `Taskfile.cli.yml` beside the tasks of the target, with no name in common; the tools of
  `mise.toml`; the patterns of `.gitignore`; a version in `go.mod` of each module that a Go file imports; and the
  jobs of `ci.yml`, whose `needs` each name a job of the workflow.
- [x] Each test of the group `embedded` that names the targets of `cli` in a list of its own also names `chi-api`:
  with no item, `ci.yml` of `chi-api` has only the job `test` and ends with the last line of the starter, and
  `go.mod` holds only the lines of the starter.
- [x] A `chi-api` repo named `api` with `cli` stops the write before the first file, because the starter and `cli`
  both write `cmd/api/main.go`, as a `go-event-sourcing-service` repo does.
- [x] `e2e/tests/new.test.ts`: `feature-compose new chi-api my-api --feature cli --no-prompt` exits with 0. In
  `./my-api`, `gofmt -l .` lists no file, `go build ./...` exits with 0, and `task test:unit` exits with 0 and its
  lines that start with `✓` name `internal/cli`, `internal/cli/release`, `internal/cli/version`,
  `internal/cli/progress` and `internal/use_cases/user`. The CLI that `task build` builds, and the CLI that
  `task cli -- --help` runs from the source, list only the commands `version`, `update` and `help`.
  `task --dry api:run` prints `go run ./cmd/api`, so the task of the API stays the starter's. `task --list-all`
  names each task that `.github/workflows/ci.yml` calls, `test:unit` and `test:e2e` among them.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass. `task test:combinations`
  passes, and its output names `chi-api --feature cli`.
- [x] The e2e tests of `show` pass, and the two tests that read the whole tree see its last line.
- [x] `README.md` of this repo gives `chi-api`, `go` and `go-event-sourcing-service` as the targets of `cli`.
  `docs/e2e-tests.md` tells the full run of `chi-api` with `cli`.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/chi-api/starter.yml` — the five extension points
- `catalog/starters/chi-api/files/go/Taskfile.yml`, `mise.toml`, `.github/workflows/ci.yml`, `go.mod.tmpl`,
  `.gitignore` — the render of the items
- `catalog/features/cli/feature.yml` — the target `chi-api`
- `internal/catalog/catalog_test.go` — the lists of targets of `cli`, and the repo named `api`
- `e2e/tests/new.test.ts` — the full run of `chi-api` with `cli`
- `docs/e2e-tests.md` — the full run of `chi-api` with `cli`
- `README.md` — the targets of `cli`

**Patterns to Follow:**
- `task:1` for the starter `chi-api` and its full run
- `catalog/starters/go/starter.yml:17-37` for the five extension points
- `catalog/starters/go/files/go/Taskfile.yml:1-18`, `catalog/starters/go/files/go/mise.toml:1-9`,
  `catalog/starters/go/files/go/.github/workflows/ci.yml:1-35`, `catalog/starters/go/files/go/go.mod.tmpl:1-8` and
  `catalog/starters/go/files/go/.gitignore:1-26` for the render of the items
- `catalog/features/cli/feature.yml:1-16` for the `feature.yml` of `cli`
- `internal/catalog/catalog_test.go:2258-2346` for the files that `cli` adds to each target
- `internal/catalog/catalog_test.go:2818-2853` for the docs and the tasks of `cli` in each target
- `internal/catalog/catalog_test.go:2894-2944` for `go.mod` and `ci.yml` of each target
- `internal/catalog/catalog_test.go:3174-3187` for the repo named `api`
- `e2e/tests/new.test.ts:693-723` for the full run of a starter with `cli`
- `docs/e2e-tests.md:171-184` for the docs of that full run
- `tasks/learnings.md:290-295` for `go vet -tags combinations`
- `tasks/learnings.md:395-400` for the rows of the whole tree
- `tasks/learnings.md:409-414` for the task names of a starter and a feature
- `tasks/learnings.md:416-421` for the check of a written workflow
- `tasks/learnings.md:493-498` for an item of `gomod.requires` and the module graph

**Testable:** Yes

**Certainty:** high — the starter `go` declares and writes the same five points (`catalog/starters/go/starter.yml:17-37`
and its files), and the tests of `cli` already take its targets from its `feature.yml`
(`catalog_test.go:2258-2346`, `catalog_test.go:2818-2853`)

**Blast radius:** low — none of the named categories; no file of `cli` changes, and the release jobs that `cli` adds
to `ci.yml` are the same items as for `go`

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write `chi-api` with `--feature cli` with the built binary, and run `task test:unit`, `task release:local` and
`NO_COLOR=1 task test:e2e:local` there one time. With the binary of HEAD, write `go` and `go-event-sourcing-service`
with `cli` into a second folder, and `diff -r` them with the repos of the work tree: no difference. Rename the job
`test` to `build` in `catalog/starters/chi-api/files/go/.github/workflows/ci.yml`, run
`go test -tags unit -run TestCatalog/embedded ./internal/catalog/`, check that the test of the `needs` fails by name,
and copy the file back from a copy made before the break. Run `task test:combinations`, and the e2e tests of `new`, `show` and `check` on the host.

**Depends on:** Task 1

---

### Task 4: Make `cli-auth` target `chi-api`, with one text of its docs for each target

**Behavior:** `catalog/features/cli-auth/feature.yml` targets `chi-api` and `go-event-sourcing-service`, with its
needs, its option and its files entries as they are. Its needs `cli` and `api-auth` target `chi-api` (Tasks 2 and
3), so `check` finds no mistake (rule 3). `feature-compose new chi-api my-api --feature cli-auth` selects `cli` and
`api-auth`, and the CLI of the repo calls the API of the repo with the API key. `docs/cli-auth.md` has one text for
every target: its example call names a placeholder route (open question 1).

**Acceptance Criteria:**
- [ ] The `targets` of `catalog/features/cli-auth/feature.yml` name `chi-api` and `go-event-sourcing-service`, and
  no other line of the file changes.
- [ ] Tests of the group `embedded` take the targets of `cli-auth` from its `feature.yml`. For each target and each
  variant (the method `api-key`, the method `oauth`, and `store-key=true`), `cli-auth` adds its files, and of the
  files of the repo with `cli` and `api-auth` it changes only `internal/cli/root.go`, which lists `auth` after
  `version` and `update`, and `go.mod`. For each two targets and each variant, the files that `cli-auth` adds have
  the same paths and the same contents.
- [ ] For each target and each variant, `go.mod` of the selection requires a version of each module that a Go file
  of `cli-auth` imports, other than the standard library and the module of the repo, and the Go files of `cli-auth`
  import only its own packages and `cli`, and only a test imports `internal/auth`.
- [ ] For each target and each variant, `docs/cli-auth.md` names what it names today for
  `go-event-sourcing-service` (the variable, `credentialCommand`, `auth status`, the method of `api-auth`, the API
  URL that ends with `/api`), and each path that it names in backticks is in the repo of that target. Its example
  call names no route that only one target serves.
- [ ] The test of `TestCatalog` that reads the routes of the API reads a `chi-api` repo with `cli-auth`, and with
  `cli-auth` and the method `oauth`: its docs name each route under `/api`, and do not name `GET /` alone.
- [ ] The test of `TestCatalog` of the combinations of this repo finds, for `chi-api` and for
  `go-event-sourcing-service`, `cli-auth` with each value of `store-key` and with the method `oauth`, and no
  combination of `store-key=true` with `oauth`; `Select` takes each combination.
- [ ] `e2e/tests/new.test.ts`: `feature-compose new chi-api my-api --feature cli-auth --no-prompt` exits with 0, and
  its output names `api-auth` and `cli` as needed by `cli-auth` and `chi-api` as the starter. `go.mod` holds no module
  of the items of `gomod.requires` of `cli-auth`. `gofmt -l .` lists no file, the CLI builds, and `task test:unit`
  exits with 0 and its lines that start with `✓` name `internal/credential`, `internal/apiclient`, `internal/auth`
  and `internal/cli`. With the API that the test starts with `API_KEY_HASHES` of a key of the test, a call of
  `internal/apiclient` to `POST /users` with `{}` gets the answer 400 of the handler with the key in `MY_API_API_KEY`,
  stops with the message of a refused key with an unknown key, and stops with the message of no key with no key.
- [ ] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass. `task test:combinations`
  passes, and its output names `chi-api --feature cli-auth:store-key=false`, `chi-api --feature cli-auth:store-key=true`
  and `chi-api --feature api-auth:method=oauth --feature cli-auth:store-key=false`.
- [ ] The e2e tests of `show` pass, and the two tests that read the whole tree see its last line.
- [ ] `README.md` of this repo gives `chi-api` and `go-event-sourcing-service` as the targets of `cli-auth`.
  `docs/e2e-tests.md` tells the full run of `chi-api` with `cli-auth`.
- [ ] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — the target `chi-api`
- `catalog/features/cli-auth/files/api-key/go/docs/cli-auth.md`, `catalog/features/cli-auth/files/oauth/go/docs/cli-auth.md`
  — the example call
- `internal/catalog/catalog_test.go` — the files, the imports, `go.mod` and the docs of `cli-auth` for each target, the
  routes under `/api`, and the combinations of this repo
- `e2e/tests/new.test.ts` — the full run of `chi-api` with `cli-auth`, and the call of the API client with the module
  and the route of the repo
- `docs/e2e-tests.md` — the full run of `chi-api` with `cli-auth`
- `README.md` — the targets of `cli-auth`

**Patterns to Follow:**
- `task:2` for one text of the docs of a feature for each target, and the e2e run against the API of `chi-api`
- `task:3` for the targets of `cli`
- `catalog/features/cli-auth/feature.yml:1-31` for the `feature.yml` of `cli-auth`
- `internal/catalog/catalog_test.go:2420-2508` for the files that `cli-auth` adds with each variant
- `internal/catalog/catalog_test.go:2539-2575` and `internal/catalog/catalog_test.go:2619-2646` for `go.mod` and the
  imports of `cli-auth`
- `internal/catalog/catalog_test.go:2648-2664`, `internal/catalog/catalog_test.go:2694-2743` for the docs of `cli-auth`
- `internal/catalog/catalog_test.go:2332-2344` for the same files for each two targets
- `internal/catalog/catalog_test.go:3651-3672` for the combinations of this repo
- `e2e/tests/new.test.ts:364-423` for the full run with `cli-auth`
- `e2e/tests/new.test.ts:1239-1269` for the call of the API client
- `docs/e2e-tests.md:185-202` for the docs of that full run
- `tasks/learnings.md:290-295` for `go vet -tags combinations`
- `tasks/learnings.md:395-400` for the rows of the whole tree
- `tasks/learnings.md:493-498` for an item of `gomod.requires` and the module graph
- `tasks/learnings.md:500-505` for a change of a template block on an option, with a repo written from the commit
  before
- `tasks/learnings.md:507-512` for the CLI of a written repo in an e2e test

**Testable:** Yes

**Certainty:** medium — Task 2 is the precedent for a feature with docs for each target, and the tests of `cli-auth`
(`catalog_test.go:2420-2575`) for each variant; but those tests build every variant from one starter, and the call
of the API client in the e2e tests (`new.test.ts:1239-1269`) holds the module and the route of
`go-event-sourcing-service`

**Blast radius:** low — none of the named categories; no Go file of `cli-auth` changes, so the CLI of a `chi-api`
repo keeps, sends and stores the credential with the same code as in `go-event-sourcing-service`, and the check of
the credential is `api-auth`'s (Task 2)

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write `chi-api` with `--feature cli-auth:store-key=true` with the built binary, and run `task test:unit` and
`NO_COLOR=1 task test:e2e:local` there one time. With the binary of HEAD, write `go-event-sourcing-service` with
`cli-auth` of each variant into a second folder, and `diff -r` them with the repos of the work tree: only
`docs/cli-auth.md` differs, and the diff of each holds only the example call. Run `task test:combinations`, and the
e2e tests of `new`, `show` and `check` on the host.

**Depends on:** Task 2, Task 3

## Summary

- **Tasks:** 4.
- **Order:** dependency first, then risk. Task 1 adds the starter, which each later task needs, and gives the first
  criterion and the routes under `/api`. Task 2 comes next because it holds the most risk: the protected routes of a
  `chi-api` repo with `api-auth`, and the docs that open question 1 decides. Task 3 repeats the pattern of the
  starter `go`. Task 4 needs both, because `cli-auth` needs `cli` and `api-auth` (rule 3).
- **Assessment:** Task 3 is high certainty: the starter `go` declares and writes the same five points, and the tests
  of `cli` already iterate its targets. Tasks 1, 2 and 4 are medium: each repeats a precedent of the catalog with a
  named variation (the first starter before `go`, docs for two starters with different local runs, an e2e call that
  holds the module and the route of one starter). No task is low certainty, because this breakdown decides the
  layout of the starter (decisions 1 to 14) and open question 1 decides the docs. Tasks 1 and 2 have a high blast
  radius: they decide which requests reach a protected route of a `chi-api` repo. Tasks 3 and 4 change no code of a
  feature.
- **Criteria of the story:**
  - "`feature-compose new chi-api my-api --no-prompt` makes a repo, and its unit tests pass. The API has no check of
    the credential of its own": Task 1.
  - "`api-auth`, `cli`, and `cli-auth` each target `chi-api`. In CI, a `chi-api` repo with each of them, and with
    each value of their options, builds and passes its tests": Tasks 2, 3 and 4, through the combinations of
    `task test:combinations`.
  - "The catalog has one set of files for each feature. The only change to a feature for `chi-api` is the new
    target": Tasks 2, 3 and 4, with open question 1 about the docs.
  - "`chi-api` declares `api.middleware` and `api.config` with the same item fields as `go-event-sourcing-service`,
    and `check` passes": Task 2.
  - "A `chi-api` repo serves the routes of its API under `/api`, as a `go-event-sourcing-service` repo does (US-019).
    So the docs of `api-auth` and `cli-auth` name one path for both starters": Task 1 for the routes, Tasks 2 and 4
    for the docs.
- **Deferred:** no criterion of the story.
