# Tasks: Call the API of a Service from a Web App (US-019)

## Contents

1. Task 1: Serve the routes of the API under `/api`
2. Task 2: Answer the web apps of the allowed origins with the CORS headers
3. Task 3: Answer a request that `api-auth` refuses with the JSON error shape of the API
4. Task 4: Name each route with `/api` in the docs of the repo, of `api-auth` and of `cli-auth`

## Story Reference

US-019, "Call the API of a service from a web app", in `user-stories/feature-compose.md:638-666` (Description,
Acceptance Criteria, Context). The reference implementation is the template `go-event-sourcing-service` of
`repo-templates` (read-only), in its commits `fa6d77d` (CORS and `ALLOWED_ORIGINS`), `43427b6` (the JSON answer of
401), `5744ef1` (the routes under `/api`), `b692c2e` (`/api` in `Taskfile.api.yml`) and `d75aba1` (`/api` in the
docs and the event model). The catalog takes from those commits only what this story names; see Boundaries.

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

- The parts of the five template commits that this story does not name: the cleanup function that `Register`
  returns and that `Server.Shutdown` calls (`5744ef1`), the shared-token middleware and the caller in the request
  context, `WithUser` and `UserFromContext` (`43427b6`), the `AUTH` variable and the NATS settings of
  `Taskfile.api.yml`, the webhook callback of `docker-compose.local.yml` and the payment test (`b692c2e`), and the
  payment and process-manager docs (`d75aba1`). The starter has no shared-token check (US-001) and no process
  manager; the work of the API when it stops comes with `push` (US-021).
- Credentials across origins: the API sends no `Access-Control-Allow-Credentials`, so a web app sends its
  credential in the `Authorization` header and not in a cookie.
- `Access-Control-Expose-Headers`. A script of another origin reads the status and the JSON body of an answer;
  it does not read `WWW-Authenticate`.
- A wildcard origin (`*`), a pattern of origins, and a check of the form of each origin when the API starts. An
  entry of `ALLOWED_ORIGINS` that a browser never sends, for example with a path or a trailing slash, never
  matches; the README of the repo tells the form of an origin (Task 2).
- An extension point through which a feature adds a CORS header or method. Decision 3 makes it unneeded.
- A new message of `cli-auth` from the JSON body of 401. The API client keeps its `credential.ErrRefused`
  (`catalog/features/cli-auth/files/common/go/internal/apiclient/client.go.tmpl:36-39`).
- A web server or a proxy in the repo that serves a web app and the API on one host. The prefix `/api` makes such a
  host possible; the repo does not include one.

**Deferred:** none of the criteria of this story. US-015 (`chi-api` under `/api`), US-017 and US-021 name their
own routes under `/api` in their own criteria.

## Codebase Context

**The routes today.** `use_cases.Register`
(`catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:20-42`) puts the public
routes on the router, and the protected routes in a group whose middleware is the items of `api.middleware`
(`auth.Middleware(cfg.Auth)` with `api-auth`). The patterns come from each `route.Routable`: `GET /` of
`use_cases/root` and the five `/accounts` routes of `use_cases/account/handler.go.tmpl:44-52`. `register_test.go.tmpl`
walks the router and expects each protected route to go through the middleware and `GET /` not to. The handler tests
and the middleware tests of `api-auth` build their own router or call the middleware directly with `/accounts`, so
the prefix does not touch them.

**The server and the config.** `server.newHandler`
(`catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl:34-44`) adds the request
logger and `Recoverer` with `Use`, then calls `Register`. `api.Config` (`internal/api/config.go.tmpl:10-16`) has
`Port`, `DB` and the items of `api.config`; `config.Load` is `caarlos0/env` v6, which leaves a slice field empty for a
variable that is set to the empty text. The starter has no CORS. `RequestLogger`
(`internal/api/middleware/request_logger.go.tmpl:14-34`) is the one middleware of the starter, a function that
returns the middleware.

**The answers.** The handlers answer with `response.Response` (`internal/api/response/response.go.tmpl:9-13`):
`successful`, `messages`, and `data` only when it has a value; `response.Fail` gives an error.
`apiTest.RequireErrorResponse` (`internal/api/test/common.go.tmpl:22-28`) reads that shape in a test. `api-auth`
refuses with `refuse` (`catalog/features/api-auth/files/common/go/internal/auth/refuse.go.tmpl:7-10`), which both
methods call: 401, `WWW-Authenticate: Bearer`, and the text body of `http.Error`. Rule 6 forbids `api-auth` to import
the package `response` of the starter; `internal/catalog/catalog_test.go:1123-1141` checks the imports of each method.

**The guard of US-001.** `internal/catalog/catalog_test.go:933-946` checks that no file of a repo from the starter
alone holds `token`, `bearer` or `authorization` in any case. So the CORS code, its test and the README of the
starter must not name the `Authorization` header (decision 3).

**The callers of the routes.** `Taskfile.api.yml` calls `{{.API}}/accounts…` with `API: http://127.0.0.1:3333`.
`e2e/tests/new.test.ts` starts the API of a written repo (`startedApi`, lines 555-588) and waits for `GET /` to answer
(`answers`, lines 616-622); `openAccount` (lines 669-675) posts to `/accounts`; the cli-auth test builds a program
that calls `apiclient.New(<api>)` with the path `/accounts` (lines 637-667). `docs/e2e-tests.md:80-142` tells these
tests with their routes. The docs of `api-auth` say "`GET /` is public" and curl `http://localhost:3333/accounts`
(both methods); the docs of `cli-auth` show `apiclient.New("http://localhost:3333")` with `/accounts`. The event model
of the starter (`docs/event-model-account.emod`, `.md`, `.svg`) names the routes without `/api`; its `.svg` is the
file of the template before `d75aba1`, and `d75aba1` changes only the four trigger texts in it. The README of the
starter and the README of this repo name no route.

**The starter sends `Idempotency-Key`.** The README of the starter tells a client that retries a `POST` to send
`Idempotency-Key` (`catalog/starters/go-event-sourcing-service/files/go/README.md:54-59`). The CORS of the template
allows only `Content-Type` and `Authorization`, so a web app of another origin could not send it.

**Words.** A *route* is a method and a pattern that `Register` serves; a *protected route* goes through the items of
`api.middleware` (`tasks/us-003-api-keys.design.yaml`). An *allowed origin* is an entry of `ALLOWED_ORIGINS`, or a
default origin. A *preflight* is the `OPTIONS` request that a browser sends before a request of another origin. The
*base URL* of the API is `http://<host>:<port>/api`, which a client puts before a path such as `/accounts`. The
*error shape* is `response.Response` with `successful` false and its `messages`. `credential` keeps the meaning of
`tasks/us-007-oauth.design.yaml`.

**Decisions of this breakdown, for the caller to overrule:**

1. **One prefix for all routes** (Task 1). `Register` serves every route of every `route.Routable` under `/api`, in
   one place, as `5744ef1` does; the patterns of the handlers stay as they are. A route at the root path, for any
   method, gets 404. The e2e tests call the API at its base URL under `/api`, as the docs of `cli-auth` tell a client
   to (Task 4).
2. **The answer to `OPTIONS`** (Task 2). The API answers each `OPTIONS` with 204 before the routes and before the
   items of `api.middleware`, at any path and from any origin, as `fa6d77d` does; only an allowed origin gets the
   CORS headers in it. So an `OPTIONS` at the root path also gets 204, and only the other methods there get 404.
3. **The method and the headers of a preflight** (Task 2). To the preflight of an allowed origin, the API allows the
   method and the headers that the preflight asks for, and `Access-Control-Max-Age` is 86400, as in the template.
   The template has a fixed list of `Content-Type` and `Authorization`. This breakdown does not take it: the list
   leaves out `Idempotency-Key`, which the README of the starter tells a client to send; a repo without `api-auth`
   must not name `Authorization` (the guard of US-001); and a feature that adds a header would need an extension
   point to add it to the list.
4. **`Vary: Origin` on each answer** (Task 2), also to another origin and to a request with no `Origin`, because the
   CORS headers of an answer depend on the origin. The template adds it only for an allowed origin.
5. **The value of `ALLOWED_ORIGINS`** (Task 2). Entries are separated by commas, and the spaces around an entry do
   not count, as in `API_KEY_HASHES` of `api-auth`
   (`catalog/features/api-auth/files/api-key/go/internal/auth/key_hashes.go.tmpl:13-24`). An origin matches only
   with the same scheme, host and port. A variable that is not set, or set to the empty text, gives the default
   `http://localhost:8080` and `http://127.0.0.1:8080`. With a value, a default origin is allowed only when the value
   lists it.
6. **Names** (Task 2). The middleware is `middleware.CORS(allowedOrigins)`, beside `middleware.RequestLogger`, and
   not the `NewCORSMiddleware` of the template, which repeats its package. The field is `api.Config.AllowedOrigins`.
   The default origins live in the server of the starter, as in the template, and the e2e test of the started API
   checks them.
7. **The JSON answer of 401** (Task 3). `refuse` of `files/common` writes it, so both methods give the same answer:
   401, `WWW-Authenticate: Bearer`, a JSON `Content-Type`, and the error shape with one message that tells the client
   to sign in (the template writes `sign in to continue`). `api-auth` writes the shape itself (rule 6). The template
   sends no `WWW-Authenticate`; the catalog keeps it.
8. **The files that are ported** (for the audit). Ported with no change of behaviour: the prefix of `Register`
   (`5744ef1`, without the cleanup), the `API` variable of `Taskfile.api.yml` (`b692c2e`), the field of
   `api.Config` and the use of the middleware with its default (`fa6d77d`), and the account docs and event model
   (`d75aba1`). Ported with changes: the CORS middleware (`fa6d77d`, decisions 2-5) and the JSON of 401 (`43427b6`,
   decision 7). The rest of each file of the starter and of `api-auth` is as US-001, US-003 and US-007 left it.

## Tasks

### Task 1: Serve the routes of the API under `/api`

**Behavior:** A repo from `go-event-sourcing-service` serves each route of its API under `/api`, for example
`POST /api/accounts`, and the public root route at `GET /api/` (decision 1). A request with the method and the path of
a route at the root path, for example `POST /accounts`, gets 404, and does not reach the middleware of the protected
routes. The protected routes still go through the items of `api.middleware`, and the public route still does not. The
requests of `Taskfile.api.yml` call the routes under `/api`. The e2e tests of `new` call each written API at its base
URL under `/api`.

**Acceptance Criteria:**
- [x] The tests of `use_cases.Register` in the written repo (`register_test.go`) take the routes from the router that
  `Register` fills, not from a list of the test, and show: each route that `Register` serves has a path under
  `/api/`; the protected routes are the five account routes under `/api`, and each goes through the middleware of the
  protected routes; `GET /api/` does not go through it; and a request with the method and the path of each route
  without `/api`, `GET /` included, gets 404.
- [x] With no middleware for the protected routes, `POST /api/accounts` with the body `{}` gets the answer 400 of the
  handler.
- [x] `Taskfile.api.yml` of the written repo calls each route under `http://127.0.0.1:3333/api`.
- [x] `e2e/tests/new.test.ts`: the API of each written repo is ready when `GET /api/` answers; the repo from the starter
  alone answers 200 to `GET /api/`, 400 to `POST /api/accounts` with `{}`, and 404 to `POST /accounts` with `{}`; the
  tests of `api-auth` with each method send their requests to `/api/accounts` and `GET /api/`; and the program of the
  cli-auth test calls `apiclient.New` with the base URL of the API under `/api` and the path `/accounts`, and still
  prints 400.
- [x] `docs/e2e-tests.md` names each route that the tests call under `/api`, and tells that `POST /accounts` gets 404.
- [x] With the starter alone, with `api-auth` with each value of `method`, with `cli` and with `cli-auth`, the written
  repo builds, `gofmt -l .` lists no file, and its unit tests pass (`task test:combinations`).
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl` — every route under
  `/api`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl` — the routes
  under `/api`, and 404 at the root path
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml` — the `API` variable
- `e2e/tests/new.test.ts` — the base URL under `/api` in `startedApi`, `answers`, `openAccount`, the program of the
  cli-auth test, and the 404 at the root path
- `docs/e2e-tests.md` — the routes of the tests

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register.go.tmpl:20-42` for the public
  and the protected routes
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/use_cases/register_test.go.tmpl:19-76` for the
  walk of the routes and a middleware that refuses
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.api.yml:3-36` for the requests of the Taskfile
- `e2e/tests/new.test.ts:43-67`, `e2e/tests/new.test.ts:69-106` and `e2e/tests/new.test.ts:169-218` for the e2e tests
  that start the API
- `e2e/tests/new.test.ts:555-588`, `e2e/tests/new.test.ts:616-622` and `e2e/tests/new.test.ts:637-675` for
  `startedApi`, `answers`, the program of the cli-auth test and `openAccount`
- `docs/e2e-tests.md:80-142` for the docs of the e2e tests that start an API
- `tasks/learnings.md:30-36` for the files that are ported (decision 8)
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof of a template by breaks by hand
- `tasks/learnings.md:106-112` for an e2e test that starts the API without Postgres
- `tasks/learnings.md:51-57` and `tasks/learnings.md:58-64` for gotestsum and the e2e tests on the host
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:285-291` for `task test:combinations`

**Testable:** Yes

**Certainty:** medium — `register.go.tmpl:20-42` and its test hold the public and the protected routes; but no route
of this repo sits under a prefix, and each caller of the API in the e2e tests moves to the base URL under `/api`

**Blast radius:** high — authorization: the task moves the group of protected routes whose middleware is `api-auth`,
and a protected route outside that group is reachable with no credential; also a contract outside this repository,
the paths of the API of each written repo

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo from the starter alone and
one with `--feature api-auth` with the built binary into scratch folders, and run `go test -tags unit
./internal/api/...` there. Apply one break at a time to the written `register.go`, read its diff, run the tests, and
put the file back: serve the protected routes at the root path; serve the public route at the root path; serve the
protected routes under `/api` without their middleware; serve each route both at the root path and under `/api`.
Each break makes a test fail. Run `task test:combinations` and `e2e/tests/new.test.ts` on the host.

**Depends on:** None

---

### Task 2: Answer the web apps of the allowed origins with the CORS headers

**Behavior:** The API of a written repo answers a request from an allowed origin with the CORS headers, also when the
answer is a 401 of `api-auth` or a 404. It answers each `OPTIONS` with 204 before the routes and before `api-auth`,
with the CORS headers only for an allowed origin (decision 2); the preflight of an allowed origin gets the method and
the headers that it asks for (decision 3). A request from another origin, or with no `Origin`, gets no CORS header.
Each answer has `Vary: Origin` (decision 4). `ALLOWED_ORIGINS` gives the allowed origins as a list with commas, with
the default `http://localhost:8080` and `http://127.0.0.1:8080` (decision 5). The README of the repo tells the
setting.

**Acceptance Criteria:**
- [ ] The unit tests of `middleware.CORS` in the written repo (`internal/api/middleware/cors_test.go`, build tag
  `unit`) show:
  - a request with the `Origin` of an allowed origin gets `Access-Control-Allow-Origin` with that origin, and the
    status and the body of the handler;
  - an `OPTIONS` from an allowed origin with `Access-Control-Request-Method` and `Access-Control-Request-Headers` gets
    204, `Access-Control-Allow-Origin` with that origin, the asked method in `Access-Control-Allow-Methods`, the asked
    headers in `Access-Control-Allow-Headers`, and `Access-Control-Max-Age: 86400`, and the handler does not run;
  - an `OPTIONS` from another origin, and one with no `Origin`, gets 204 with no `Access-Control-` header, and the
    handler does not run;
  - a request from another origin, and one with no `Origin`, gets no `Access-Control-` header, and the status and the
    body of the handler;
  - each of these answers has `Vary: Origin`;
  - with two allowed origins, each of them gets the headers, and an origin that differs from one of them only in its
    scheme, its host or its port gets none.
- [ ] `api.Config` reads `AllowedOrigins` from `ALLOWED_ORIGINS`, separated by commas; the spaces around an entry do not
  count.
- [ ] The started API, in `e2e/tests/new.test.ts`:
  - in the repo with `api-auth` (`api-key`), an `OPTIONS` to `/api/accounts` from `http://localhost:8080` that asks for
    `POST` and the headers `authorization` and `content-type` gets 204 with `Access-Control-Allow-Origin:
    http://localhost:8080` and both headers allowed; `POST /api/accounts` from that origin with no key gets 401 with
    `Access-Control-Allow-Origin`; and the same requests from `https://another.example.com` get no
    `Access-Control-Allow-Origin`;
  - with no `ALLOWED_ORIGINS`, and with `ALLOWED_ORIGINS` set to the empty text, `http://localhost:8080` and
    `http://127.0.0.1:8080` get `Access-Control-Allow-Origin`;
  - with `ALLOWED_ORIGINS` set to two origins with a space after the comma, each of the two gets
    `Access-Control-Allow-Origin`, and `http://localhost:8080` gets none.
- [ ] The README of the written repo names `ALLOWED_ORIGINS`, the form of an origin (`<scheme>://<host>[:<port>]`,
  with no path), the default origins, and how to set it for the local API, for example
  `ALLOWED_ORIGINS=<origins> task api:run`. It names only files that the repo has.
- [ ] The test of the starter alone that finds no `token`, `bearer` or `authorization` in a written file
  (`internal/catalog/catalog_test.go:933-946`) passes with its words unchanged.
- [ ] The test of the fields of `api.Config` with no item of `api.config` (`internal/catalog/catalog_test.go:986-991`)
  expects `AllowedOrigins` among the fields of the starter. With `api-auth` with each value of `method`, the files of
  the starter that change are still only the server and the config of the API
  (`internal/catalog/catalog_test.go:1029-1049` passes unchanged).
- [ ] `docs/e2e-tests.md` tells the CORS checks of the started API.
- [ ] With each combination, the written repo builds, `gofmt -l .` lists no file, and its unit tests pass
  (`task test:combinations`).
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/cors.go.tmpl` — new: `CORS`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/cors_test.go.tmpl` — new: its tests
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl` — `AllowedOrigins`
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl` — the CORS before the
  routes, and the default origins
- `catalog/starters/go-event-sourcing-service/files/go/README.md` — `ALLOWED_ORIGINS`
- `internal/catalog/catalog_test.go` — the fields of `api.Config`
- `e2e/tests/new.test.ts` — the CORS answers of the started API
- `docs/e2e-tests.md` — the CORS checks

**Patterns to Follow:**
- `task:1` for the routes under `/api` and the base URL of the e2e tests
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/request_logger.go.tmpl:14-34` for a
  middleware of the starter
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/middleware/request_logger_test.go.tmpl:25-60` for
  a test of a middleware of the starter through a router
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/server/server.go.tmpl:34-44` for the middleware of
  every request
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/config.go.tmpl:10-16` for a field of `api.Config`
- `catalog/features/api-auth/files/api-key/go/internal/auth/key_hashes.go.tmpl:13-24` for a list with commas whose
  entries ignore the spaces around them
- `catalog/starters/go-event-sourcing-service/files/go/README.md:31-59` for the docs of the local API
- `internal/catalog/catalog_test.go:933-946` for the words that a repo without `api-auth` must not hold
- `internal/catalog/catalog_test.go:986-991` and `internal/catalog/catalog_test.go:1029-1049` for the written server
  and config
- `e2e/tests/new.test.ts:69-106` and `e2e/tests/new.test.ts:555-588` for the e2e tests that start the API with
  settings
- `docs/e2e-tests.md:80-142` for the docs of the e2e tests
- `tasks/learnings.md:30-36` for the files that are ported, and the changes of decisions 2-5
- `tasks/learnings.md:85-91`, `tasks/learnings.md:92-98` and `tasks/learnings.md:127-133` for the proof of new Go code
  of the catalog by breaks by hand
- `tasks/learnings.md:99-105` for a setting that `config.Load` reads
- `tasks/learnings.md:106-112` for an e2e test that starts the API without Postgres
- `tasks/learnings.md:113-119` for the docs of a setting of the local API
- `tasks/learnings.md:120-126` for a test that reads an env variable
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:285-291` for `task test:combinations`

**Testable:** Yes

**Certainty:** medium — `RequestLogger` (`request_logger.go.tmpl:14-34`) is a middleware that `server.go.tmpl:34-44`
adds to every request; but this one answers `OPTIONS` itself before the routes and `api-auth`, and its headers depend
on a setting of `api.Config` with a default

**Blast radius:** high — authorization: it decides which web origins can read the answers of the API, and a preflight
passes before `api-auth`

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo from the starter alone and
one with `--feature api-auth` with the built binary, and run `go test -tags unit ./internal/api/...` there. Apply one
break at a time to the written files, read its diff, run the tests, and put the file back: give the headers to each
origin; give no header to an allowed origin; pass an `OPTIONS` to the routes; answer an `OPTIONS` with 200; drop
`Vary: Origin` for another origin; allow a fixed list of headers instead of the asked headers; keep the spaces around
an entry. Each break makes a unit test fail. Then, on the host: drop the default origins from the server; add the
CORS after the routes or only to the protected routes; each makes an e2e test fail. Run `task test:combinations` and
`e2e/tests/new.test.ts` on the host.

**Depends on:** Task 1

---

### Task 3: Answer a request that `api-auth` refuses with the JSON error shape of the API

**Behavior:** With `api-auth`, each method answers a refused request with 401, `WWW-Authenticate: Bearer`, a JSON
`Content-Type`, and a body in the error shape of the other answers of the API: `successful` false and one message that
tells the client to sign in (decision 7). `api-auth` writes the body itself, and imports no package of the starter.
The docs of `api-auth` tell the body.

**Acceptance Criteria:**
- [ ] For each value of `method` in `catalog/features/api-auth/feature.yml`, the unit tests of `internal/auth` in the
  written repo show that each request that the middleware refuses gets 401, `WWW-Authenticate: Bearer`, a
  `Content-Type` of `application/json`, and a body that is a JSON object with exactly the keys `successful`, with the
  value false, and `messages`, with one text that tells the client to sign in; and that the handler does not run.
- [ ] Each accepted request gets the answer of the handler, as before, in the tests of each method.
- [ ] With each value of `method`, no Go file of `api-auth` imports a package of the module other than the packages
  under `internal/auth` (`internal/catalog/catalog_test.go:1123-1141` passes unchanged).
- [ ] The docs of `api-auth` with each value of `method` tell, in the answers of the API, that a refused request gets
  401 with `WWW-Authenticate: Bearer` and the JSON body with its message, and the tests of the docs
  (`internal/catalog/catalog_test.go:1143-1176`) check it.
- [ ] `e2e/tests/new.test.ts`: in the repo with `api-auth` with each value of `method`, the 401 of `POST /api/accounts`
  has a JSON `Content-Type` and a body with the same keys as the body of the answer 400 of the handler, with
  `successful` false in both, and a message that tells the client to sign in. The cli-auth test still shows its
  message for a refused key.
- [ ] The tests of `internal/apiclient` of `cli-auth` pass unchanged: `Do` still turns 401 into
  `credential.ErrRefused`.
- [ ] `docs/e2e-tests.md` tells the body of 401 that the tests check.
- [ ] With each combination, the written repo builds and its unit tests pass (`task test:combinations`).
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/features/api-auth/files/common/go/internal/auth/refuse.go.tmpl` — the JSON body
- `catalog/features/api-auth/files/api-key/go/internal/auth/middleware_test.go.tmpl` — the body of each refused request
- `catalog/features/api-auth/files/oauth/go/internal/auth/middleware_test.go.tmpl` — the body of each refused request
- `catalog/features/api-auth/files/api-key/go/docs/api-auth.md`, `catalog/features/api-auth/files/oauth/go/docs/api-auth.md`
  — the answers of the API
- `internal/catalog/catalog_test.go` — the tests of the docs of `api-auth`
- `e2e/tests/new.test.ts` — the body of 401 of the started API
- `docs/e2e-tests.md` — the body of 401

**Patterns to Follow:**
- `task:1` for the routes under `/api` in the e2e tests
- `catalog/features/api-auth/files/common/go/internal/auth/refuse.go.tmpl:7-10` for the one refusal of both methods
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/response/response.go.tmpl:9-13` and
  `catalog/starters/go-event-sourcing-service/files/go/internal/api/response/response.go.tmpl:37-42` for the error
  shape
- `catalog/starters/go-event-sourcing-service/files/go/internal/api/test/common.go.tmpl:22-28` for a test that reads
  the error shape
- `catalog/features/api-auth/files/api-key/go/internal/auth/middleware_test.go.tmpl:27-87` and
  `catalog/features/api-auth/files/oauth/go/internal/auth/middleware_test.go.tmpl:115-135` for the refused requests of
  each method
- `catalog/features/api-auth/files/api-key/go/docs/api-auth.md:62-71` and
  `catalog/features/api-auth/files/oauth/go/docs/api-auth.md:87-96` for the answers of the API in the docs
- `internal/catalog/catalog_test.go:1123-1176` for the imports and the docs of `api-auth`
- `e2e/tests/new.test.ts:69-145` for the e2e tests of each method
- `tasks/learnings.md:30-36` for the file that is ported, and the change of decision 7
- `tasks/learnings.md:85-91` and `tasks/learnings.md:127-133` for the proof by breaks by hand
- `tasks/learnings.md:155-161` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:197-203` for a criterion that holds for each method
- `tasks/learnings.md:285-291` for `task test:combinations`

**Testable:** Yes

**Certainty:** high — `refuse` (`refuse.go.tmpl:7-10`) is already the one refusal of both methods, and the error shape
is `response.Response` (`response.go.tmpl:9-13`); the refused cases of each method are in their middleware tests

**Blast radius:** high — authentication: it changes the answer to each refused request of a protected route with
both methods; also a contract outside this repository, the body of 401 that the clients of a written repo read

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo with each value of
`method` with the built binary, and run `go test -tags unit ./internal/auth/` there. Apply one break at a time to the
written `refuse.go`, read its diff, run the tests of both repos, and put the file back: write the text body of
`http.Error`; drop the JSON `Content-Type`; write `successful` true; rename the key `messages`; write no message; drop
`WWW-Authenticate`; answer 403. Each break makes a test of each method fail. Run `task test:combinations` and
`e2e/tests/new.test.ts` on the host.

**Depends on:** Task 1

---

### Task 4: Name each route with `/api` in the docs of the repo, of `api-auth` and of `cli-auth`

**Behavior:** The event model of the starter (`docs/event-model-account.emod`, `.md` and `.svg`) names each route of
the API with `/api`, as `d75aba1` does. The docs of `api-auth`, with each method, name the public root route
`GET /api/` and send their example request to `http://localhost:3333/api/accounts`. The docs of `cli-auth` give the
API client the base URL `http://localhost:3333/api` and the path `/accounts`. A test of the embedded catalog finds no
route in the docs and the Taskfiles of a written repo without `/api`.

**Acceptance Criteria:**
- [ ] `docs/event-model-account.emod`, `docs/event-model-account.md` and `docs/event-model-account.svg` of the written
  repo name each route of the API with `/api`, in each trigger, table, timeline and description. `emod validate
  docs/event-model-account.emod` passes.
- [ ] The docs of `api-auth` with each value of `method` name the public root route `GET /api/`, and their curl
  example calls `http://localhost:3333/api/accounts`. The tests of the docs of `api-auth`
  (`internal/catalog/catalog_test.go:1143-1176`) check both.
- [ ] The docs of `cli-auth` call `apiclient.New` with `http://localhost:3333/api` and `Do` with the path `/accounts`,
  and say that the base URL of the API ends with `/api`. The test of the docs of `cli-auth`
  (`internal/catalog/catalog_test.go:1366-1381`) checks it.
- [ ] A test of the group `embedded` of `TestCatalog` writes a repo from the starter alone, with `api-auth` with each
  value of `method`, and with `cli-auth`. It takes the routes from the route patterns in the written Go files of
  `internal/api/use_cases`, not from a list of the test, and finds none of them without `/api` before its path in a
  file under `docs/`, in `README.md`, or in a Taskfile of the repo; it also finds no `GET /` alone there.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.emod`,
  `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.md`,
  `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.svg` — the routes under `/api`
- `catalog/features/api-auth/files/api-key/go/docs/api-auth.md`, `catalog/features/api-auth/files/oauth/go/docs/api-auth.md`
  — the root route and the curl example
- `catalog/features/cli-auth/files/common/go/docs/cli-auth.md` — the base URL of the API client
- `internal/catalog/catalog_test.go` — the docs of `api-auth` and `cli-auth`, and the routes in the docs and the
  Taskfiles

**Patterns to Follow:**
- `task:1` for the routes under `/api` and the `API` variable of `Taskfile.api.yml`
- `catalog/starters/go-event-sourcing-service/files/go/docs/event-model-account.md:32-64` for the routes in the event
  model
- `catalog/features/api-auth/files/api-key/go/docs/api-auth.md:3-4` and
  `catalog/features/api-auth/files/api-key/go/docs/api-auth.md:51-60` for the routes in the docs of `api-auth`
- `catalog/features/api-auth/files/oauth/go/docs/api-auth.md:3-4` and
  `catalog/features/api-auth/files/oauth/go/docs/api-auth.md:50-60` for the same in the docs of `oauth`
- `catalog/features/cli-auth/files/common/go/docs/cli-auth.md:37-56` for the API client in the docs of `cli-auth`
- `internal/catalog/catalog_test.go:1143-1176` and `internal/catalog/catalog_test.go:1366-1381` for the tests of the
  docs
- `internal/catalog/catalog_test.go:933-946` for a check of each written file
- `tasks/learnings.md:30-36` for the files that are ported

**Testable:** Yes

**Certainty:** high — the tests of the docs of `api-auth` and `cli-auth` (`catalog_test.go:1143-1176`,
`catalog_test.go:1366-1381`) and the check of each written file (`catalog_test.go:933-946`) are the pattern, and
`d75aba1` makes the same edits to the same event model

**Blast radius:** low — none of the named categories; it changes the docs, and a wrong route in them shows as a 404
to the reader

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo with `--feature cli-auth`
and one with `--feature api-auth:method=oauth` with the built binary, and read the routes in `docs/`. Run `emod
validate docs/event-model-account.emod` in the written repo. Put `/accounts` without `/api` back into one doc at a
time, and `GET /` into the docs of `api-auth`, and see the test of the routes fail for each.

**Depends on:** Task 1

## Summary

- **Tasks:** 4.
- **Order:** dependency first, then risk. Task 1 moves the routes under `/api` and every caller that the tests and the
  Taskfile hold, so each later task tests against the routes under `/api`. Task 2 is the riskiest and the main value
  of the story: the CORS of the API, before the routes and before `api-auth`. Task 3 gives the 401 of `api-auth` the
  JSON error shape. Task 4 brings the event model and the docs of the features to the routes under `/api`, and adds
  the test that keeps them there; it comes last because it is the cheapest and only docs depend on it.
- **Assessment:** No task is low certainty. Each ports a change that the template already made, and this breakdown
  decides the open rules of the CORS (decisions 2-6), so no task decides its own design. Task 1 and Task 2 are medium:
  no route of this repo sits under a prefix, and no middleware of the starter answers a request itself. Tasks 1-3
  have a high blast radius: Task 1 moves the group that `api-auth` protects, Task 2 decides which origins can read
  the answers, and Task 3 changes the answer to each refused request. Task 4 changes only docs.
- **Criteria of the story:**
  - "A repo from `go-event-sourcing-service` serves each route of its API under `/api`, for example
    `POST /api/accounts`, and the public root route at `GET /api/`. A request to a route at the root path, for example
    `POST /accounts`, gets 404": Task 1.
  - "The requests in `Taskfile.api.yml`, the docs of the repo, and its event model name each route with `/api`":
    Task 1 (`Taskfile.api.yml`) and Task 4 (the event model, the docs, and the test of the routes in the docs and the
    Taskfiles).
  - "The API answers a request from an allowed origin with the CORS headers. It answers a preflight `OPTIONS` with
    204, also on a route that `api-auth` protects. A request from another origin gets no CORS headers": Task 2.
  - "`ALLOWED_ORIGINS` gives the allowed origins as a list with commas. With no value, the API allows
    `http://localhost:8080` and `http://127.0.0.1:8080`": Task 2.
  - "With `api-auth`, each method answers a refused request with 401, `WWW-Authenticate: Bearer`, and a JSON body in
    the error shape of the other answers of the API. The message of the body tells the client to sign in": Task 3.
  - "The docs of `api-auth` and `cli-auth` name the routes and the base URL of the API under `/api`": Task 4.
- **Deferred:** none.
