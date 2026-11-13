# {{ kebab .Name }}

An HTTP API in the Go module `{{ option "starter.module" }}`, with the router chi. Each route of the API is under
/api. `internal/use_cases` has an example of a public route and an example of a protected route. Replace them
with your own routes.

## Requirements

[mise](https://mise.jdx.dev/) installs Go and the tools that the tasks use: Task, gotestsum, ShellCheck and
govulncheck. `mise.toml` gives their versions.

```sh
mise install
```

## Run the API

```sh
task api:run
```

The API listens on port 3333. `task api:run` passes the variables of the shell to the API, so you give the API a
setting before the command:

```sh
PORT=8080 task api:run
```

The example routes:

| Route | Kind | Answer |
| --- | --- | --- |
| `GET /api/` | public | The version of the API |
| `POST /api/users` | protected | 400 with the validation errors of the body, or 200 for a valid user |

A request to a public route goes to its handler. A request to a protected route first goes through the middleware
of the protected routes, which a feature can add. With no such middleware, the API gives each request to its
handler.

```sh
curl -s http://127.0.0.1:3333/api/
curl -s -X POST http://127.0.0.1:3333/api/users -d '{"name": "example-user", "email": "user@example.com", "age": 30}'
```

## Build

```sh
task api:build          # builds the API with the version in VERSION
task api:docker:build   # builds the image of the API
task api:docker:run     # builds the image and runs it on port 3333
```

## Test

```sh
task test:unit              # the unit tests, with the race detector and coverage
task test:shellcheck        # the shell scripts and the hooks
task test:vulnerabilities   # the known vulnerabilities of the module and of its dependencies
```

Most tests have the `unit` build tag. `task test:unit` passes `-tags=unit`, so it runs them with the tests that have no
build tag.

## Set Up the Hook

```sh
task hook:setup
```

Before a push, the hook `hooks/pre-push` runs `task test:shellcheck` when the push changes a shell script or a
hook, and `task hook:validate-go-changes` when the push changes a Go file. For a new branch, it compares the branch
with the main branch. Set `SKIP_PREPUSH_HOOK=true` to skip the hook for one push.

## Add a Route

1. Add a package for the use case under `internal/use_cases`, for example `internal/use_cases/user`. Give it a
   handler with a `Routes` method, which returns each route of the use case with `route.Public` or
   `route.Protected`. A pattern does not start with /api, because the API adds it.
2. Add the handler to the list in `allRoutes` of `internal/use_cases/register.go`.

## Layout

| Path | Contents |
| --- | --- |
| `cmd/api` | The program of the API. |
| `internal/config` | The settings of the API, which it reads from the environment. |
| `internal/server` | The HTTP server, and the middleware of each request. |
| `internal/use_cases` | The routes under /api, and a package for each use case. |
| `internal/middleware` | The log of each request. |
| `internal/response`, `internal/route` | The JSON body of an answer, and the kinds of route. |
| `internal/test` | The helpers of the tests of the handlers. |
| `hooks/pre-push`, `scripts/libs/git-hook-prepush.sh` | The pre-push hook. |
| `.github/workflows/ci.yml` | The CI workflow. On each push, it checks the shell scripts, builds the module, and runs the unit tests and the check for vulnerabilities. |
| `Taskfile.yml`, `Taskfile.api.yml`, `Taskfile.test.yml` | The tasks. |
| `Dockerfile` | The image of the API. |
