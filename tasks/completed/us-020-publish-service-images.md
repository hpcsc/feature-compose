# Tasks: Publish the Images of a Service Repo From CI (US-020)

## Contents

1. Task 1: Publish the images of the API and of catchup from CI on `main` and on tags, after the tests, with only the permissions that each job needs
2. Task 2: Keep `.env.*` files, `.git` and build output out of the build of each image
3. Task 3: Run the migrations with the SSL mode that `DB_SSLMODE` names, and with `disable` when it has no value
4. Task 4: Check the shell scripts in the root folder of the repo with `task test:shellcheck`

## Open Questions

The tasks follow the recommendation of each question. Overrule a recommendation before the task that it names starts.

1. **Where do the ignore lines of the images go, when `cli` owns `.dockerignore`?** (Task 2.) The template adds a
   `.dockerignore` at the root of the repo. In the catalog, the feature `cli` writes `.dockerignore`
   (`catalog/features/cli/files/go/.dockerignore:1-4`), and rule 7 of `PROPOSAL.md` gives each file one owner. A
   `.dockerignore` of the starter makes `check` report that the starter and the feature `cli` both write
   `.dockerignore` (the check of `internal/catalog/catalog_test.go:1405-1426`), fails "the catalog of this repo has no
   mistake", and stops `new go-event-sourcing-service --feature cli`.
   **Recommendation: one ignore file for each Dockerfile that `publish` builds, `Dockerfile.api.dockerignore` and
   `Dockerfile.catchup.dockerignore`, with the seven lines of the template.** Docker reads `<Dockerfile>.dockerignore`
   beside the Dockerfile in place of the `.dockerignore` at the root of the context (fact below). The lines of `cli`
   (`.git/`, `bin/`, `dist/`, `e2e/node_modules/`) are all in the lines of the template, so the images of a repo with
   `cli` lose no line. The starter owns two new paths, `cli` keeps its file, and no feature changes.

   The other choice: the starter owns `.dockerignore` with a new extension point for its lines, and `cli` adds its four
   lines as items. Rule 2 then needs the point in each target of `cli`, so `go` and `chi-api` also get a
   `.dockerignore`, and `cli` and three starters change for one criterion of one starter.

2. **Which images get `latest`?** (Task 1.) The criterion gives `latest` to an image from `main`. With the tags of the
   template, `docker/metadata-action@v5` also gives `latest` to the image of a version tag such as `v1.2.3`: its
   default `flavor` is `latest=auto`, which adds `latest` for each `type=semver` tag that is not a prerelease (its
   README, "Latest tag", and the example row "push tag | refs/tags/v1.2.3 | 1.2.3, 1.2, latest"). Then `latest` is
   the image that was published last, of `main` or of a version tag, and a tag on an older commit moves `latest` back.
   **Recommendation: the `flavor` input `latest=false` on the step `Docker meta`.** Only the rule `type=raw` on the
   default branch then gives `latest`, so `latest` is always an image of `main`. It is the one difference between the
   steps of `publish` and the steps of the template, and the test names it. If the caller wants the tags of the
   template as they are, Task 1 leaves out the input, the test compares the steps with no difference, and the README
   tells that a version tag also moves `latest`.

3. **How does the publish keep the order of the pushes to `main`?** (Task 1.) The tags `latest` and `main` move with
   each publish. Two pushes to `main` start two runs, and the run of the older commit can publish after the run of the
   newer commit when its tests take longer (`tasks/learnings.md:388-393`).
   **Recommendation: a concurrency group at the level of the workflow in `ci.yml` of the starter, keyed by the
   workflow and the ref, with no cancel of a run in progress**, as `.github/workflows/prerelease.yml:10-13` of this
   repo has. A run of a ref then starts only after the run before it ends, so the newer commit publishes last. When a
   run waits and a newer run of the same ref queues, GitHub cancels the waiting run, so `main` gets the images of its
   newest commit. An item of `ci.jobs` cannot add a group at the level of the workflow, but `publish` is a job of the
   starter's own file, which can. The costs: the runs of one branch run one at a time, and a run that waits is
   cancelled, with its tests, when a newer push comes. Each version tag is its own ref and its own group, and with
   question 2 a version tag moves no tag.
   The other choice is the head check of the job `prerelease` of `cli` (`catalog/features/cli/feature.yml:75-138`). It
   runs the tests of each commit, but it adds a step that the template does not have, it must apply only to a branch
   (`git ls-remote` gives an annotated tag the SHA of the tag object, not of the commit), and an older commit of `main`
   then pushes no image, also not with its SHA tag.

## Story Reference

US-020, "Publish the images of a service repo from CI", in `user-stories/feature-compose.md:674-698` (Description,
Acceptance Criteria, Context). The reference is the template `go-event-sourcing-service` of `repo-templates`
(`/Users/davidnguyen/repo-templates/`, read-only), in its commits `321d3dd` (the CI workflow: permissions, newer
actions, `setup-go`, and the job `publish` in place of `build`) and `5c8d77f` (`.dockerignore`, `DB_SSLMODE` in the
entrypoint of the migrations, and the root scripts in `test:shellcheck`). The tool and the catalog do not depend on it.

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

- The newer `golang.org/x` modules of the Context. No criterion names them, and no task needs them. `go.mod.tmpl` of
  the starter keeps its versions.
- The CI workflow, the ignore files, the migrations and the shell check of the starters `go` and `chi-api`. The story
  names the repo of `go-event-sourcing-service`. (`go` and `chi-api` already use `checkout@v7` and `mise-action@v5`.)
- An image of `cmd/fake_services` of `process-manager`. It fakes outside systems for `task run`;
  `Dockerfile.fake_services` gets no ignore file and no publish.
- The job `e2e` of `cli` in the `needs` of `publish`. `e2e` exists only with `cli`, and a `needs` that names no job of
  the workflow stops the workflow, so `publish` needs `test`, as in the template. In a repo with `cli`, the images do
  not wait for the end-to-end tests of the CLI.
- `DB_SSLMODE` for `task migrate:*` (`Taskfile.migrate.yml:9`) and for the `migrate` service of Docker Compose. Both
  run the migrations against the local Postgres, at `127.0.0.1` or at the service `db`, which has no SSL (decision 5).
- `DB_SSLMODE` for the API and catchup. `config.DB.ConnectionString` gives no `sslmode`
  (`catalog/starters/go-event-sourcing-service/files/go/internal/common/config/db.go.tmpl:13-22`), so pgx uses its
  default, `prefer`, which also connects to a database that needs SSL.
- A change to the Dockerfiles. The last stage of each already copies only the binary.
- Images for more than one platform. The template sets up QEMU and gives `build-push-action` no `platforms`, so it
  builds for the platform of the runner; the step stays as the template has it (decision 2).
- A test that pushes an image to the GitHub container registry, and actionlint or act, which do not run on this machine
  (`tasks/learnings.md:416-421`). The tests check the workflow as written; the Verification of Task 1 tells how to see
  a real publish by hand.

**Deferred:** none of the criteria of this story.

## Codebase Context

**The starter `go-event-sourcing-service`, in `catalog/starters/go-event-sourcing-service/`.**

- `files/go/.github/workflows/ci.yml:1-54`: `on: push: {}` and `permissions: contents: read` at the level of the
  workflow (US-022). The job `test` uses `checkout@v4` and `mise-action@v3`, has no permissions of its own, and runs
  `task test:shellcheck`, `test:unit`, `test:integration` and `test:vulnerabilities`. The job `build` needs `test`, runs
  on every push, and runs `task <app>:docker:build` for the matrix `api`, `catchup`; it pushes nothing. Then the items
  of `ci.jobs`, with indent 2. `starter.yml:104-110` gives `.github/**` the delimiters `[[` `]]`, so the expressions
  `{{version}}` and `{{is_default_branch}}` of metadata-action and each `${{ ... }}` of GitHub Actions reach the
  written file as text.
- `starter.yml:79-82`: the summary of `ci.jobs` says that the workflow runs on each push with the permissions of the
  starter, and that a job that writes sets its own permissions. `cli` adds `e2e` (no permissions), `release` and
  `prerelease` (each `contents: write`) (`catalog/features/cli/feature.yml:23-138`). No other feature adds a job.
- `files/go/Dockerfile.api:1-24` and `files/go/Dockerfile.catchup:1-24`: the build stage copies `go.mod`, `go.sum`,
  `internal` and `cmd/<app>`; the last stage is `gcr.io/distroless/static` with the binary only.
- `files/go/Taskfile.yml:18-21`: `test:shellcheck` checks each `.sh` under `scripts/` and each file of `hooks/`. The
  written repo has no `.sh` file in its root folder, also with each feature (`cli` writes `scripts/install.sh`).
- `files/go/scripts/local/migrate/entrypoint.sh:31-35`: `migrate` with `sslmode=disable` in the database URL.
  `docker-compose.dependencies.yml` mounts the script into the service `migrate`, which `task run-dependencies` and,
  through `docker-compose.test.yml`, `scripts/test/start-test-dependencies.sh` start.
- `files/go/.env.local` and `files/go/.env.integration` are files of the starter at the root of the repo.
- `files/go/README.md`: the sections Requirements, Run the Example, Call the API From a Web App, Test (`:90-98`),
  Replace the Example, Layout (`:110-126`, with the row of `migrations`), and the sections on events, commands and
  consumers, then the items of `readme.sections`.

**The tests of a written repo**, in the group `embedded` of `TestCatalog` (`internal/catalog/catalog_test.go`):
`writtenStarter` (`:1511-1520`), `parsedYAML` (`:1881-1886`), `targetsOf` and `writtenTarget` (`:2029-2045`). The
tests of `ci.yml`: the jobs and the last line with no item (`:2975-2989`, which names the job `build` and its last
line, so it changes in Task 1), the triggers and permissions with `cli` (`:2991-3009`), the `if` and `needs` of the
release jobs (`:3025-3039`), the permission to write the contents (`:3061-3079`), and the steps of the release jobs
compared, parsed, with the workflows of this repo, with each deliberate difference named (`:3081-3119`). The test of the
installer of `cli` runs a written script with stubs on `PATH` that record their arguments (`:3196-3216`). "the catalog
of this repo has no mistake" is at `:1471-1476`.

**Facts checked for this breakdown:**

- Docker 27.3.1 (`docker build -f Dockerfile.api .`, which `task api:docker:build` runs) and Docker Compose 2.30.3
  (`build.dockerfile: Dockerfile.api`, as `docker-compose.yml` has) read `Dockerfile.api.dockerignore` in place of the
  root `.dockerignore`. In a context with the four lines of `cli` in `.dockerignore` and the seven lines of the template
  in `Dockerfile.api.dockerignore`, the build got no `.env.local`, no `internal/x/.env.secret`, no `bin/`, no `.git/`
  and no `cov.out`. The docs of Docker name the convention (docs.docker.com/build/concepts/context/): the
  Dockerfile-specific file takes precedence over the root file. `docker/build-push-action` builds with buildx.
- docker/metadata-action: `latest=auto` gives `latest` to a semver tag that is not a prerelease (open question 2). The
  action writes the image name in lowercase, as the GitHub container registry needs. A tag push matches no
  `type=ref,event=branch` rule, so the image of a version tag gets its short SHA and its version, and no branch tag.
- The `find` of the template's line, which gives the root scripts to shellcheck in one call, exits 0 when no file
  matches and 1 when shellcheck finds an error (shellcheck 0.11.0, `find` of macOS). So the line without `|| true`
  meets the criterion, and with `|| true` it passes whatever shellcheck finds.
- The e2e image of this repo (`e2e/Dockerfile`) has no shellcheck, so an e2e test that runs `task test:shellcheck` of a
  written repo fails in Docker; the tests of Task 4 are in `TestCatalog`.
- Docker answered `timeout 20 docker info` on 2026-11-14.

**Words.** *The images* are the images of the API and of catchup: one for each app of the matrix of `publish`. To
*publish* an image is to push it to the GitHub container registry; the job is `publish`, as in the template. `release`
and `prerelease` stay the jobs of `cli` that publish GitHub releases of the CLI. *The ignore file of an image* is
`Dockerfile.<app>.dockerignore`; `.dockerignore` stays the file of `cli`. *The SSL mode* is the `sslmode` of the
database URL of the migrations. *The root folder* is the folder of `Taskfile.yml`.

**Decisions of this breakdown, for the caller to overrule:**

1. **The workflow** (Task 1): `ci.yml` of the template at `321d3dd`, with the items of `ci.jobs` after the jobs, as
   today. `test` gets `contents: read` of its own, `checkout@v7`, `setup-go@v7` with the Go version of `go.mod`, and
   `mise-action@v5`; `build` becomes the template's `publish`. Open questions 2 and 3 add the `flavor` input and the
   concurrency group. The test holds the jobs of the template as text and compares them, parsed, with the written jobs
   (`tasks/learnings.md:402-407`), because no test can read `repo-templates`.
2. **The step `Set up QEMU`** (Task 1) stays as the template has it, also with no `platforms` input. It is the
   template's (`tasks/learnings.md:31-36`).
3. **The ignore files** (Task 2): open question 1, with the seven lines of the template in its order. The last stage of
   each Dockerfile already copies only the binary, so the image holds no file of the machine; the ignore file also
   keeps those files out of the context that Docker sends and out of the build stage.
4. **The shell check** (Task 4): the template's line with no `|| true`. With `|| true` the task passes whatever
   shellcheck finds in a root script, which does not meet the criterion. With no root script, `find` runs nothing and
   exits 0, so the line needs no guard.
5. **The SSL mode** (Task 3): only the entrypoint, as the template. An empty `DB_SSLMODE` also gives `disable`, as the
   default form of the template does. `Taskfile.migrate.yml` and the `migrate` service of Docker Compose reach only the
   local Postgres, which has no SSL (Boundaries).
6. **The README of the written repo** (Tasks 1 and 3): a section after Test tells that CI publishes the two images to
   the GitHub container registry, with their names and tags (Task 1). A sentence tells that the `migrate` container
   reads `DB_SSLMODE`, with `disable` as the default (Task 3). No criterion names the README; a developer needs the
   names of the images to deploy from the registry.
7. **No render with sentinel values.** The ported text names no value of scaffold (the workflow, the ignore lines and
   the two script lines), so the port takes the text of the two commits as it is.

## Tasks

### Task 1: Publish the images of the API and of catchup from CI on `main` and on tags, after the tests, with only the permissions that each job needs

**Behavior:** The CI workflow of a `go-event-sourcing-service` repo has the jobs `test` and `publish` of the template
(decision 1). On a push to `main` and on a tag, `publish` runs after `test` passes, once for each app of its matrix
(`api`, `catchup`): it builds `Dockerfile.<app>` and pushes the image `ghcr.io/<owner>/<repo>-<app>` to the GitHub
container registry of the repo. Each image has the tag of the short SHA of its commit and the tag of its branch; an
image from a version tag also has the version, and an image from `main` also has `latest`, which no other image gets
(open question 2). On a push to another branch, only the tests run. The workflow and each job read the contents, and
only `publish` also writes the packages. The runs of one ref run one at a time (open question 3). The README of the
repo tells where the images are and which tags they have (decision 6).

**Acceptance Criteria:**
- [x] `TestCatalog`: with no feature, the written `ci.yml` has the jobs `test` and `publish` and no other, and ends
  with the last line of `publish`. The test "with no item, ci.yml of each target of cli has only the jobs of the starter
  and ends with the last line of the starter" names `publish` and its last line for `go-event-sourcing-service`.
- [x] `TestCatalog`: the jobs `test` and `publish` of the written `ci.yml`, parsed, are the jobs of the template's
  `ci.yml` at `321d3dd`, which the test holds as text, apart from the `flavor` input of the step `Docker meta`, which the
  test names as the one difference. The written workflow has the `name`, `on` and `permissions` of the template, and a
  concurrency group at the level of the workflow that holds the workflow and the ref and cancels no run in progress.
- [x] `TestCatalog`: `publish` needs `test`, and its `if` lets it run only for `refs/heads/main` and for a ref that
  starts with `refs/tags/`. `test` has no `if`. The workflow runs on each push.
- [x] `TestCatalog`: the tags of `Docker meta` in the written file are the four rules of the template: the short SHA
  with no prefix, the branch, the version of a semver tag, and `latest` on the default branch. They hold `{{version}}`
  and `{{is_default_branch}}` as text, and `flavor` turns off the `latest` that the action adds by itself.
- [x] `TestCatalog`: for each app of the matrix of `publish`, the written repo has `Dockerfile.<app>`.
- [x] `TestCatalog`: with every feature that targets `go-event-sourcing-service`, the permissions of the workflow are
  `contents: read`, `test` has `contents: read`, `publish` has `contents: read` and `packages: write`, and no other job
  has a permission on the packages. The test that only `release` and `prerelease` can write the contents passes.
- [x] The README of the written repo tells that a push to `main` and a tag publish the images of the API and of catchup
  to the GitHub container registry after the tests pass, names the images in the form that the workflow pushes them,
  and names their tags, `latest` only for `main`. A test of `TestCatalog` reads the section, and the test that the
  README with no item ends with the last line of the starter passes.
- [x] `TestCatalog` "the catalog of this repo has no mistake" and the e2e test of `check` pass, and
  `task test:combinations` passes.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass, and
  the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/.github/workflows/ci.yml` — the jobs `test` and `publish`, the
  concurrency group
- `catalog/starters/go-event-sourcing-service/files/go/README.md` — the section about the images
- `internal/catalog/catalog_test.go` — the group `embedded`: the written workflow and the README

**Patterns to Follow:**
- `internal/catalog/catalog_test.go:3081-3119` for jobs compared, parsed, with their source, and the named differences
- `internal/catalog/catalog_test.go:3061-3079` for the permissions of each job of the written workflow
- `internal/catalog/catalog_test.go:3025-3039` for the `if` and the `needs` of a job
- `internal/catalog/catalog_test.go:2975-2989` for the jobs and the last line with no item
- `internal/catalog/catalog_test.go:2991-3009` for the triggers and the permissions with `cli`
- `internal/catalog/catalog_test.go:2029-2045` for `targetsOf` and `writtenTarget`
- `internal/catalog/catalog_test.go:1597-1603` for the template syntax of GitHub Actions in a written file
- `internal/catalog/catalog_test.go:1605-1614` for a section of the README of the written repo
- `catalog/starters/go/files/go/.github/workflows/ci.yml:1-35` for the versions of the actions of a starter
- `catalog/features/cli/feature.yml:39-73` for a job with its own permissions and `setup-go`
- `.github/workflows/prerelease.yml:10-13` for a concurrency group at the level of a workflow
- `catalog/starters/go-event-sourcing-service/starter.yml:104-110` for the delimiters of `.github/`
- `catalog/starters/go-event-sourcing-service/files/go/README.md:90-98` for the place of the section
- `tasks/learnings.md:402-407` for the comparison of a ported job by its parsed steps
- `tasks/learnings.md:388-393` for the order of the pushes and a publish on `main`
- `tasks/learnings.md:416-421` for actionlint and act
- `tasks/learnings.md:31-36` for the old defects of ported template code
- `tasks/learnings.md:129-134`, `tasks/learnings.md:367-372` and `tasks/learnings.md:535-540` for the breaks by hand
- `tasks/learnings.md:290-295` for `go vet -tags combinations`
- `tasks/learnings.md:59-64` for the e2e tests on the host

**Testable:** Yes

**Certainty:** low — no precedent here for an image publish: it is the first job of the catalog that pushes to a
registry, with docker actions that no file of this repo uses, and the tags and the order of `latest` come from the docs
of metadata-action and of GitHub concurrency, since no test here can run the job

**Blast radius:** high — a token permission and a contract consumed outside this repository: `publish` gives
`GITHUB_TOKEN` `packages: write`, and the names and tags of the images are what a deployment pulls

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. Write a repo of `go-event-sourcing-service` with the built binary and read its `.github/workflows/ci.yml`. Apply
one break at a time to the `ci.yml` of the starter, with no e2e run or `task test:combinations` at the same time, read
its diff, run `TestCatalog`, and put the file back with `git checkout --`: let `publish` also run on each branch; move
`packages: write` to the level of the workflow; remove the `flavor` input; remove `test` from the `needs` of `publish`;
let the concurrency group cancel a run in progress. Each break makes a test fail by name. By hand, when the caller wants
it: push a written repo to a scratch GitHub repository; a push to a branch runs only `test`; a push to `main` publishes
two images with the short SHA, `main` and `latest`; a tag `v0.1.0` publishes them with the short SHA and `0.1.0`, and
`latest` stays on the image of `main`. Run `task test:combinations`, and the e2e tests of `new` and `check` on the
host.

**Depends on:** None

---

### Task 2: Keep `.env.*` files, `.git` and build output out of the build of each image

**Behavior:** Beside each Dockerfile that `publish` builds, the starter writes an ignore file that Docker reads for that
Dockerfile in place of `.dockerignore` (open question 1), with the lines of the template: `.git`, `.DS_Store`, and
`node_modules`, `dist`, `bin`, `.env.*` and `*.out` at any depth. The build of each image gets none of those files,
from `docker/build-push-action` in CI, from `task <app>:docker:build`, and from Docker Compose, also in a repo with
`cli`, whose `.dockerignore` does not change. The last stage of each Dockerfile copies only the binary of its build
stage, so the image holds no file of the machine (decision 3).

**Acceptance Criteria:**
- [x] `TestCatalog`: for each app of the matrix of `publish`, the written repo has `Dockerfile.<app>.dockerignore`
  beside `Dockerfile.<app>`, and its lines are the lines of the template's `.dockerignore` at `5c8d77f`, which the test
  holds as text.
- [x] `TestCatalog`: the last stage of each Dockerfile that `publish` builds copies only the binary from its build
  stage, and copies or adds no other file.
- [x] `TestCatalog`: with `cli`, the written repo has the `.dockerignore` of `cli` and the ignore file of each image,
  and the write does not stop. "the catalog of this repo has no mistake" and the e2e test of `check` pass.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and `task test:combinations` passes.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/Dockerfile.api.dockerignore` — new
- `catalog/starters/go-event-sourcing-service/files/go/Dockerfile.catchup.dockerignore` — new
- `internal/catalog/catalog_test.go` — the group `embedded`: the ignore file and the last stage of each image

**Patterns to Follow:**
- `task:1` for the matrix of `publish`, which names the images
- `catalog/features/cli/files/go/.dockerignore:1-4` for an ignore file of the catalog
- `catalog/starters/go-event-sourcing-service/files/go/Dockerfile.api:1-24` and
  `catalog/starters/go-event-sourcing-service/files/go/Dockerfile.catchup:1-24` for the stages of each image
- `internal/catalog/catalog_test.go:1586-1591` for a test of each Dockerfile of the written repo
- `internal/catalog/catalog_test.go:2326-2345` for the files that `cli` writes
- `internal/catalog/catalog_test.go:1405-1426` for the check of a path that two owners write
- `internal/catalog/catalog_test.go:1471-1476` for the check of the catalog of this repo
- `tasks/learnings.md:360-365` for a written service repo under `.scratches/` with Docker
- `tasks/learnings.md:129-134`, `tasks/learnings.md:367-372` and `tasks/learnings.md:535-540` for the breaks by hand
- `tasks/learnings.md:31-36` for the old defects of ported template code

**Testable:** Yes

**Certainty:** medium — the `.dockerignore` of `cli` is the precedent for an ignore file of the catalog; the variation is
the name for one Dockerfile, which Docker reads in place of the root file and which no file here uses (checked in this
breakdown with Docker 27.3.1 and Compose 2.30.3)

**Blast radius:** low — none of the named categories in effect: the last stage of each Dockerfile already copies only
the binary, and the task changes no Dockerfile, so a wrong ignore file changes what the build stage gets and not what a
published image holds

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. With Docker: write a repo of
`go-event-sourcing-service` with `--feature cli` under `.scratches/` with the built binary, make it a git repository
with one commit, run `task api:build`, `task catchup:build` and `task test:unit`, and add an `.env.*` file and a `bin/`
folder in `internal/` and in `cmd/api/`. For each app, build only the build stage of `Dockerfile.<app>` and list its
`/build`: no `.env.*` file, no `.git`, no `bin`, no `.out` file. Build the whole image and list its files with
`docker create` and `docker export`: the files of the distroless base and the binary. Move `Dockerfile.api.dockerignore`
away and build the build stage again: the added files are there, so Docker read the file. Put it back, run
`docker compose build api catchup` once, and remove the images. Apply one break at a time to the catalog, read its
diff, run `TestCatalog`, and put the file back: remove the line of `.env.*` from `Dockerfile.catchup.dockerignore`;
add a copy of the whole context to the last stage of `Dockerfile.api`. Each break makes a test fail by name. Run
`task test:combinations`.

**Depends on:** Task 1

---

### Task 3: Run the migrations with the SSL mode that `DB_SSLMODE` names, and with `disable` when it has no value

**Behavior:** `scripts/local/migrate/entrypoint.sh`, which runs the migrations in the `migrate` container, puts the
value of `DB_SSLMODE` into the `sslmode` of the database URL, as the template does. With no value, or with an empty
value, the mode is `disable`, so `task run-dependencies` and the integration tests run the migrations against the local
Postgres as before (decision 5). The README of the repo tells that the `migrate` container reads `DB_SSLMODE`
(decision 6).

**Acceptance Criteria:**
- [x] `TestCatalog`: the written entrypoint, run with stubs of `nc` and `migrate` on `PATH` and values for the `DB_`
  variables, gives `migrate` a database URL with the SSL mode of `DB_SSLMODE` for `require` and for `verify-full`, and
  with `disable` when `DB_SSLMODE` is not set and when it is empty. In each case the user, the password, the host, the
  port and the name of the URL, the folder of the migrations and the command are those of the values. The result does
  not change with a `DB_SSLMODE` in the shell that runs the test.
- [x] `TestCatalog` "hooks/pre-push and the scripts can be run" passes.
- [x] The README of the written repo tells that the `migrate` container reads `DB_SSLMODE` for a database that needs
  SSL, and that the mode is `disable` with no value.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/scripts/local/migrate/entrypoint.sh` — the SSL mode of the URL
- `catalog/starters/go-event-sourcing-service/files/go/README.md` — the sentence about `DB_SSLMODE`
- `internal/catalog/catalog_test.go` — the group `embedded`: the database URL of the entrypoint

**Patterns to Follow:**
- `internal/catalog/catalog_test.go:3196-3216` for a written script that runs with stubs on `PATH` that record their
  arguments
- `internal/catalog/catalog_test.go:3236-3242` for the scripts that can be run
- `catalog/starters/go-event-sourcing-service/files/go/scripts/local/migrate/entrypoint.sh:31-35` for the URL
- `catalog/starters/go-event-sourcing-service/files/go/README.md:110-126` for the row of `migrations`
- `tasks/learnings.md:3-8` for a test that runs a shell command
- `tasks/learnings.md:122-127` for a test that reads an environment variable
- `tasks/learnings.md:360-365` for the migrations of a written service repo against Postgres
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134` and `tasks/learnings.md:367-372` for the breaks by hand

**Testable:** Yes

**Certainty:** high — the test of the installer of `cli` runs a written script with stubs on `PATH` that record their
arguments (`internal/catalog/catalog_test.go:3196-3216`), and the change is one value of one line

**Blast radius:** high — credential handling: the mode decides whether the connection that carries the database
password and the schema changes uses SSL, and a script that ignored `DB_SSLMODE` would send the password in clear to a
database that also accepts a connection with no SSL, with no error

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. With Docker: write a repo of
`go-event-sourcing-service` under `.scratches/` with the built binary, run `task run-dependencies`, and list the tables
with `docker compose exec db psql`: the migrations ran with the default mode. Run the service `migrate` again with
`DB_SSLMODE=require` against the same Postgres, which has no SSL: it fails with the error of Postgres that SSL is not
on, so the mode reaches the server. Run `docker compose down -v`. Apply one break at a time to the written entrypoint,
read its diff, run the test, and put the file back: write `disable` in place of the variable; keep the variable and
remove its default. Each break makes a test fail by name.

**Depends on:** None

---

### Task 4: Check the shell scripts in the root folder of the repo with `task test:shellcheck`

**Behavior:** `task test:shellcheck` of a written repo also gives each `.sh` file in the root folder of the repo to
shellcheck at the level `error`, and fails when shellcheck finds an error in one. With no such file, it passes as
before. The line is the template's, with no `|| true` (decision 4). The scripts under `scripts/` and the hooks are
still checked, and the job `test` of CI runs the task.

**Acceptance Criteria:**
- [x] `TestCatalog`: in a written repo, a `.sh` file in the root folder in which shellcheck finds an error makes
  `test:shellcheck` fail, and the file is among the files that shellcheck gets. A root script with no error, and no
  root script, let `test:shellcheck` pass.
- [x] `TestCatalog`: each `.sh` file under `scripts/` and each file of `hooks/` are still among the files that
  shellcheck gets.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass.

**Affected Files/Modules:**
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml` — `test:shellcheck`
- `internal/catalog/catalog_test.go` — the group `embedded`: the shell check of the written repo

**Patterns to Follow:**
- `catalog/starters/go-event-sourcing-service/files/go/Taskfile.yml:18-21` for `test:shellcheck`
- `internal/catalog/catalog_test.go:3196-3216` for a written script that runs with stubs on `PATH`
- `internal/catalog/catalog_test.go:1881-1886` for a parsed Taskfile of the written repo
- `tasks/learnings.md:3-8` for a test that runs a shell command
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134` and `tasks/learnings.md:367-372` for the breaks by hand

**Testable:** Yes

**Certainty:** high — `test:shellcheck` already checks the scripts and the hooks with the same command of shellcheck
(`Taskfile.yml:18-21`), and the test of the installer of `cli` runs written commands with stubs on `PATH`
(`internal/catalog/catalog_test.go:3196-3216`)

**Blast radius:** low — none of the named categories; it adds files to a check that runs in the tests

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass. Write a repo of
`go-event-sourcing-service` with the built binary, and in it run `task test:shellcheck`: it passes. Add a root script
with a parse error: the task fails and names the script. Correct the script: the task passes. Apply one break at a
time to the `Taskfile.yml` of the starter, read its diff, run `TestCatalog`, and put the file back with
`git checkout --`: add `|| true` to the line; make the line search `scripts/` in place of the root folder. Each break
makes a test fail by name.

**Depends on:** None

## Summary

- **Tasks:** 4.
- **Order:** risk first, then dependency. Task 1 is the one task of low certainty and the main value: the images in
  the registry. Task 2 reads the matrix of `publish` from Task 1 to know the images. Tasks 3 and 4 are one line each of
  `5c8d77f`, with no edge to the other tasks, so they can run in either order.
- **Assessments:** Task 1 is low certainty and high blast radius (a token permission and the names and tags that a
  deployment pulls). Task 3 is high certainty and high blast radius (the SSL of the connection that carries the
  database password). Task 2 is medium certainty with a named variation, and Task 4 is high certainty and low blast
  radius.
- **Criteria of the story:**
  - On a push to `main` and on a tag, CI builds and pushes the images after the tests; another branch runs only the
    tests: Task 1.
  - The tags of the short SHA, the branch, the version and `latest`: Task 1, with open question 2.
  - The permissions of each job: Task 1.
  - No `.env.*` file, no `.git` and no build output in the images: Task 2, with open question 1.
  - The migrations with `DB_SSLMODE`, and `disable` with no value: Task 3.
  - The shell check of the root scripts: Task 4.
- **Deferred:** none. The newer `golang.org/x` modules of the Context are no criterion and no task needs them
  (Boundaries).
