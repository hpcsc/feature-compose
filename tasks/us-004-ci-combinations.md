# Tasks: Test Each Starter With Each Feature in CI (US-004)

## Contents

1. Task 1: Run the `test` commands of each folder of a written repo with `RunTests`
2. Task 2: List the combinations of the catalog that CI makes a repo from
3. Task 3: Make and test the repo of each combination in CI, and name each combination that fails

## Story Reference

US-004, "Test each starter with each feature in CI", in `user-stories/feature-compose.md:178-202` (Acceptance
Criteria and Context). The design is `PROPOSAL.md`: sections 6 (`folders.<name>.test`, `PROPOSAL.md:143`, and the
contract of the `summary` of an extension point, `PROPOSAL.md:148-149`), 8 (an option with `values`,
`PROPOSAL.md:256-262`), 10 (rule 4, needs, `PROPOSAL.md:337`), 12 (CI uses `--no-prompt`, `PROPOSAL.md:430-432`) and
14 (the repos that CI makes, `PROPOSAL.md:485-486`).

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

- The product of option values. CI makes one repo for each value of an option, with the other options at their
  defaults, and not one repo for each set of values of two options.
- A value of a starter option together with a feature. CI tests each value of a starter option on the starter
  alone.
- Repos with three or more features. CI makes the repo of each pair of features that add to one extension point,
  and no larger set.
- A command that prints the combinations, and a GitHub matrix with one job for each combination (decision 1). The
  commands `check` (US-006, US-009) and `show` (US-014) are later stories.
- Checks of a written repo other than its `after_render` and `test` commands, for example `gofmt -l .`, `go build`
  or a start of its API. The three full runs of `new` in `e2e/tests/new.test.ts` keep those checks, and stay as they
  are.
- The test commands in `new`. `new` runs the `after_render` commands, as before, and never the `test` commands
  (decision 9).
- Combinations that run at the same time. The CI job makes the repos one after another.
- The folders and the commands of a feature (rule 9 of the proposal). No feature adds a folder, and a starter has
  one folder.
- A check that each feature targets a starter that exists, or that the default of an option with `values` is one
  of its values. `check` (US-006) owns the checks of the catalog. CI shows the second mistake only as a failed
  combination.

**Deferred:**

- Needs (US-005). A combination names only the features that it tests, and `Select` adds their needs after US-005
  with no change to the list. A pair in which one feature needs the other then makes the same repo as the feature
  alone. The list does not see needs, so it does not merge the two, and CI makes that repo two times.
- A feature that adds items to an extension point that another feature declares, for example `cli-auth` to
  `cli.commands`: its repo gets the declarer only through its needs (US-005). Until then `Select` stops for its
  combination, and CI fails for it. No feature of the catalog does this today.
- The values of an option of a need in the repo of the feature that needs it, for example `cli-auth` with
  `api-auth:method=oauth`: US-008 decides whether CI needs that repo. This story makes `api-auth` with each value of
  `method`.
- `when` (US-007). A combination gives each value of an option, also when a `when` of the feature would not ask for
  it. US-007 decides whether such a value makes a combination.

## Codebase Context

**The catalog.** `Catalog.Starter` reads one starter with the features that target it (`catalog/catalog.go:27-40`).
`featuresTargeting` reads each `feature.yml` and keeps the features whose `targets` name the starter
(`catalog/catalog.go:42-64`), and `starterNames` lists the folders under `starters/` in the order of their names
(`catalog/catalog.go:66-78`). Nothing exported lists the starters or the features today. `Starter.Select(name, given,
givenByFeature)` checks the given values, selects the features in the order of their names, and adds the items of
each `add_to` (`catalog/starter.go:99-177`). `new` goes from names and values to a written repo through
`Starter`, `Select`, `Write` and `RunAfterRender` (`internal/cmd/new.go:33-51`), and the combinations of this story
go through the same steps. An `option` has `defaultValue`, `pattern` and `accepted`, the list of its `values`
(`catalog/option.go:11-16`). `options.withDefaults` checks the given values and renders the default of each other
option with the name of the repo (`catalog/option.go:45-71`). A starter option can have `values`
(`catalog/starter_test.go:93-104`). A feature keeps its `add_to` as a map from the name of an extension point to its
items (`catalog/feature.go:13-20`).

**The folder commands.** `readStarter` reads `path` and `after_render` of each folder, and not `test`
(`catalog/starter.go:40-43`, `catalog/starter.go:65-71`). `folder.run` starts each command with `sh -c` in its own
process group, kills the group when the context ends, and its message always says `after_render`
(`catalog/folder.go:28-40`). `Selection.RunAfterRender` runs the folders in the order of their names
(`catalog/selection.go:51-58`). The test at `catalog/selection_test.go:860-881` checks that `RunAfterRender` runs no
`test` command.

**The catalog today.** One starter, `go-event-sourcing-service`, whose options `go-version` and `module` have a
`pattern` and no `values`, and whose folder `go` declares `test: [task test:unit]`
(`catalog/starters/go-event-sourcing-service/starter.yml`). The feature `cli` has no option, declares
`cli.commands` and adds to nothing. The feature `api-auth` has the option `method` with `values: [api-key]` and
adds to `api.middleware` and `api.config`. So the list of this story is three combinations today: the starter
alone, the starter with `api-auth` and `method=api-key`, and the starter with `cli`. No two features add to one
extension point, so the tests of the pair rule build their catalogs in `fstest.MapFS`, as
`catalog/starter_test.go:236-241` does.

**A written repo.** Its `task test:unit` runs gotestsum with `go test -race`, which needs a C compiler. Its
`mise.toml` pins `go`, `task` and `gotestsum`, and on a machine where these run through mise shims, mise refuses
the file until someone trusts the folder (`tasks/learnings.md:38-43`, `docs/e2e-tests.md:120-122`). `go mod tidy`
in `after_render` downloads modules. One repo takes about a minute with a warm module cache, and some minutes with
a cold one (`docs/e2e-tests.md:115-116`). `go test` stops a test binary after 10 minutes, unless `-timeout` gives
another limit.

**CI.** `.github/workflows/ci.yml` has the job `build` (`setup-go`, `jdx/mise-action`, `task test:unit`,
`ci.yml:14-39`) and the job `e2e` (`arduino/setup-task`, `task test:e2e` in the image of `e2e/Dockerfile`,
`ci.yml:41-53`). That image installs Task and gotestsum with `go install` (`e2e/Dockerfile:7-9`) and has no mise.
`prerelease.yml` and `release.yml` call `ci.yml`, so each job of `ci.yml` also gates a prerelease and a release.

**Tests.** One umbrella test per type, all in `package catalog_test` with the tag `unit`: `TestCatalog` (groups
`starter` and `embedded`), `TestStarter` (`select`) and `TestSelection` (`write`, `run after render`).
`catalog/catalog_test.go:220-240` stops a read of the catalog with a `failingFS`. Each test that runs a shell
command starts with `t.Chdir(t.TempDir())` (`tasks/learnings.md:3-8`).

**Words.** No code uses `combination`. `case` is taken by the case functions of `catalog/case.go`, `check` by the
command of US-006, and `PROPOSAL.md:256` uses `variant` for what the answer to one option selects.

**Decisions of this breakdown, for the caller to overrule:**

1. **CI runs a Go test under its own build tag over the list of the catalog, through one task and one job**: shape
   (a) of the brief, and not a command that prints a GitHub matrix. The test adds no command that a user sees, it
   runs the same way on a machine and in CI, and its subtests name each combination. The cost is time: one job makes
   the repos one after another, so the job grows by a minute or more for each combination.
2. **The starter alone is a combination.** A new starter that no feature targets yet gets its repo, and a value of
   a starter option is tested on the starter alone.
3. **One combination for each value of each option with `values`, with the other options at their defaults.** A
   combination gives a value to each option with `values` of its starter and of its features, so the feature alone
   and the feature with the default value of its option are one combination. It gives no value to an option
   without `values`, and the selection renders that default with the name, as `new --no-prompt` does.
4. **One combination for each pair of features that target one starter and add items to one extension point** in
   their `add_to`, with both features at their defaults. The extension point can be one of the starter or one of a
   feature. Three such features give three pairs, and two features that add to two of the same extension points
   give one.
5. **A combination names only the features that it tests.** `Select` adds their needs (US-005), so the list needs
   no change for needs.
6. **The name of a combination reads as the arguments of `new` that make the same repo**, for example
   `go-event-sourcing-service --feature api-auth:method=api-key`. A catalog author can then make the repo of a
   failed combination with `new`. `go test` shows each space of a subtest name as `_`.
7. **The CI job installs Go from `go.mod`, Task and gotestsum without mise**, as the `e2e` job and its image do. So
   the `mise.toml` of a written repo does not choose the tools in CI. On a machine with mise shims, the test trusts
   its temporary folder, as the e2e tests do.
8. **A combination that fails does not stop the others.** Each combination is a subtest, and the run reports each
   one that fails.
9. **`RunTests` runs only the `test` commands**, and the caller runs `RunAfterRender` before it. `new` runs no `test`
   command.

## Tasks

### Task 1: Run the `test` commands of each folder of a written repo with `RunTests`

**Behavior:** `starter.yml` declares `folders.<name>.test`, a list of commands. `Selection.RunTests` runs them in
the folders of a written repo, as `RunAfterRender` runs the `after_render` commands, and stops at the first command
that fails with a message that says that a `test` command failed.

**Acceptance Criteria:**
- [x] `RunTests` runs the `test` commands of each folder in that folder of the repo, in their order, and the folders
  in the order of their names. It runs no `after_render` command.
- [x] A `test` command that fails stops the run, and no later command runs. The message says that a `test` command
  failed, and names the folder, the command and its output. A program that is not on PATH stops the run the same
  way, and the message gives the cause.
- [x] A context that ends stops the command and its children, and `RunTests` returns an error.
- [x] A starter whose folders declare no `test` command runs nothing, and `RunTests` returns no error.
- [x] `RunAfterRender` still runs no `test` command, and the test at `catalog/selection_test.go:860-881` passes. The
  message of a failed `after_render` command still says `after_render`, and names the folder, the command and its
  output.

**Affected Files/Modules:**
- `catalog/starter.go` — read `test` of each folder
- `catalog/folder.go` — the `test` commands of a folder, and a message that names which list the command is in
- `catalog/selection.go` — `RunTests`
- `catalog/selection_test.go` — the tests of `RunTests` in `TestSelection`

**Patterns to Follow:**
- `catalog/folder.go:18-40` for the run of the commands of a folder and its message
- `catalog/selection.go:51-58` for `RunAfterRender`
- `catalog/starter.go:40-43` and `catalog/starter.go:65-71` for the read of a folder
- `catalog/selection_test.go:784-947` for the tests of `RunAfterRender`, and `catalog/selection_test.go:860-881` for
  a folder with both lists
- `PROPOSAL.md:143` for `folders.<name>.test`
- `tasks/learnings.md:3-8` for a test that runs shell commands
- `tasks/learnings.md:66-71` for the fixes of an audit round in `selection_test.go`

**Testable:** Yes

**Certainty:** high — the `test` commands are a second list of the same folder, run as the `after_render` commands
in `catalog/folder.go:18-40` and `catalog/selection.go:51-58`, with tests to copy at
`catalog/selection_test.go:784-947`

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** None

---

### Task 2: List the combinations of the catalog that CI makes a repo from

**Behavior:** `Catalog.Combinations` gives each combination of a starter, features and option values that CI makes
a repo from, in a fixed order. A combination gives what `Starter` and `Select` take: the name of the starter, the
values of the starter options and the features with the values of their options. Its name gives the starter, the
features and the values.

**Acceptance Criteria:**
- [ ] For each starter of the catalog, in the order of the names, the list has one combination of the starter alone,
  and one combination of each feature that targets the starter, with that feature alone. A feature that does not
  target a starter is in no combination of that starter. A starter that no feature targets has its combination of
  the starter alone.
- [ ] For each option with `values`, of a starter or of a feature, the list has one combination for each of its
  values: with the starter alone for an option of the starter, and with the feature alone for an option of a
  feature. The other options take their defaults, and no combination varies two options.
- [ ] Each combination gives a value to each option with `values` of its starter and of its features, and gives no
  value to an option without `values`. Combinations that give the same starter, features and values are one: the
  feature alone and the feature with the default value of its option are one combination, and so are the starter
  alone and the starter with the default value of its option.
- [ ] When two features that target one starter add items to one extension point in their `add_to`, the list has
  one combination with both features, at the defaults of their options. This holds for an extension point of the
  starter and for one that a feature declares. Each pair of such features gets one combination: three features
  that add to one extension point give three, and two features that add to two of the same extension points give
  one. Two features that add to no common extension point get no combination together.
- [ ] A combination names only the features that it tests. `Starter` with its starter name, and `Select` with its
  values, give the selection of its repo. For a combination that gives a value other than the default, the file
  that the feature writes holds that value.
- [ ] The name of a combination gives the starter, each feature and each value that the combination gives, as the
  arguments of `new` that make the same repo (decision 6).
- [ ] The list has a fixed order, which the test states in full, and the order of the keys in `starter.yml` and
  `feature.yml` does not change it.
- [ ] A read of the catalog that fails stops the list with its cause, as `Catalog.Starter` does.

**Affected Files/Modules:**
- `catalog/combination.go` — the new type `Combination` and its name
- `catalog/starter.go` — the combinations of one starter
- `catalog/catalog.go` — `Combinations` over the starters of the catalog
- `catalog/catalog_test.go` — a group `combinations` in `TestCatalog`, with catalogs in `fstest.MapFS`

**Patterns to Follow:**
- `catalog/catalog.go:27-78` for the read of the starters and of the features that target a starter
- `catalog/starter.go:129-139` and `catalog/starter.go:153-177` for the features in the order of their names and
  their `add_to`
- `catalog/option.go:11-16` and `catalog/option.go:45-71` for the values and the defaults of options
- `catalog/starter_test.go:93-104` for a starter option with `values`
- `catalog/starter_test.go:236-241` and `catalog/starter_test.go:310-321` for catalogs with `add_to` and with
  feature options
- `catalog/catalog_test.go:220-240` for a read of the catalog that fails
- `internal/cmd/new.go:33-51` for the steps from the names and the values to a written repo
- `PROPOSAL.md:485-486` and `PROPOSAL.md:256-262` for the repos that CI makes and the options with `values`
- `tasks/learnings.md:66-71` for the fixes of an audit round in `catalog_test.go`

**Testable:** Yes

**Certainty:** medium — `catalog/catalog.go:42-78` reads the starters and the features that target each, and
`catalog/starter.go:129-177` walks features and their `add_to` in the order of their names, but no code derives
selections from the catalog: the rules for values, for pairs and for one combination of one repo have no instance

**Blast radius:** low — none of the named categories

**Verification:** `task test:unit` passes

**Depends on:** None

---

### Task 3: Make and test the repo of each combination in CI, and name each combination that fails

**Behavior:** A Go test under its own build tag takes the combinations of the catalog in the binary. For each one,
in a subtest with the name of the combination, it makes the repo in a temporary folder through `Starter`, `Select`
and `Write`, and runs its `after_render` commands and then its `test` commands. `task test:combinations` runs the
test, and a job of the CI workflow runs the task on each push.

**Acceptance Criteria:**
- [ ] A test file in `catalog/` under its own build tag, and not `unit`, takes `Combinations` of `catalog.Embedded()`.
  For each combination it gets the starter, selects with the values of the combination, writes the repo into a
  temporary folder, and runs `RunAfterRender` and then `RunTests`. A step that returns an error fails the subtest of
  that combination with that error.
- [ ] Each combination is its own subtest, whose name is the name of the combination. A combination that fails does
  not stop the others.
- [ ] `task test:combinations` runs the test, and the 10 minutes that `go test` allows by default do not cut the run
  short. On this machine it passes, with one subtest for each combination of the catalog in the binary: today the
  starter alone, the starter with `api-auth` and `method=api-key`, and the starter with `cli`.
- [ ] When a change to a template of one feature makes the tests of its repo fail, `task test:combinations` exits with
  a status that is not zero. Its output names the starter, the features and the option values of that combination,
  with the output of the `test` command that failed, and the other combinations pass.
- [ ] On a machine where `go` and `task` run through mise shims, the test trusts its temporary folder, so mise accepts
  the `mise.toml` of each written repo.
- [ ] `.github/workflows/ci.yml` has a job beside `build` and `e2e` that runs `task test:combinations` on
  `ubuntu-latest`. It installs Go from `go.mod`, and Task and gotestsum without mise (decision 7). The workflow and
  the task name no starter, no feature and no option value, so a new starter or a new feature gets its repos with no
  change to them. `actionlint` finds no error in the workflow.
- [ ] `task test:unit` passes, and runs none of these subtests.
- [ ] `README.md` says what `task test:combinations` makes and runs, what it needs (Go, Task, gotestsum, a C
  compiler and the network), and that a catalog author runs it after a change to a starter or a feature.

**Affected Files/Modules:**
- `catalog/` — a test file under its own build tag, named for what it tests
- `Taskfile.test.yml` — the task `combinations`
- `.github/workflows/ci.yml` — the job that runs it
- `README.md` — the section about the task

**Patterns to Follow:**
- `e2e/tests/new.test.ts:32-56` for a repo that the tool makes and whose `task test:unit` runs, with mise trust
- `catalog/catalog_test.go:243-253` for a written repo of the catalog in the binary
- `internal/cmd/new.go:33-51` for the steps from a starter to a repo whose `after_render` commands ran
- `.github/workflows/ci.yml:41-53` for a job that installs Task without mise
- `e2e/Dockerfile:7-9` for the install of Task and gotestsum with `go install`
- `Taskfile.test.yml:7-9` for a task that runs Go tests with a build tag
- `docs/e2e-tests.md:115-122` for what a full run needs and how long it takes
- `tasks/learnings.md:3-8` for a test that runs shell commands
- `tasks/learnings.md:38-43` for the real path that `MISE_TRUSTED_CONFIG_PATHS` needs on macOS
- `tasks/learnings.md:86-91` for the proof of a test of the catalog templates with a break by hand
- `task:1` for `RunTests`, and `task:2` for `Combinations`

**Testable:** Yes

**Certainty:** medium — `e2e/tests/new.test.ts:32-56` makes the repo of the starter and runs its `task test:unit`
with mise trust, and `.github/workflows/ci.yml:41-53` installs Task without mise, but no Go test here writes the
repos of the catalog and runs their commands, none runs under a tag other than `unit` or past the default timeout
of `go test`, and a new CI job shows that it passes only after a push

**Blast radius:** low — none of the named categories; the new job also gates a prerelease and a release, because
`prerelease.yml` and `release.yml` call `ci.yml`

**Verification:** `task test:unit` passes. `task test:combinations` passes on this machine with three subtests.
Break one template of `api-auth` by hand so that a test of its repo fails, run `task test:combinations`, check that
only the combination of `api-auth` fails and that the output names it, and put the file back with
`git checkout -- <file>`. `actionlint .github/workflows/ci.yml` finds no error.

**Depends on:** Task 1, Task 2

## Summary

**Total:** 3 tasks.

**Order.** Dependency first. Task 1 (`RunTests`) and Task 2 (`Combinations`) do not depend on each other, and each
has its own exported contract that its unit tests check on catalogs in memory. Task 3 is their consumer: it runs
the list of the catalog in the binary and adds the task and the CI job. Task 3 is also the task with the most risk,
because its result in CI shows only after a push, so it gets the time of a full run on this machine and a break by
hand.

**Coverage of the story:**

| Criterion of US-004 | Tasks |
| --- | --- |
| For each starter and each feature that targets it, CI makes a repo with that feature and its needs, runs the `after_render` and the `test` commands of each folder, and fails when a command fails | 2 (the combinations; `Select` adds the needs after US-005), 1 (the `test` commands), 3 (the run in CI) |
| One repo for each value of each option that has `values`, for example each value of `method` of `api-auth` | 2, 3 |
| Two features that add items to one extension point get one repo with both | 2 (on catalogs in memory; the catalog has no such pair today), 3 |
| CI takes the list of repos from the catalog; a new starter or feature needs no change to the CI config | 2 (the list from the catalog), 3 (the workflow and the task name no starter, feature or value) |
| For each repo that fails, the CI output names the starter, the features and the option values | 2 (the name of a combination), 3 (the subtest of each combination) |
| Context: these repos check the contract of the `summary` of each extension point, and CI builds the files of each value | 2, 3 |

No criterion of the story is deferred.

**Criteria that this breakdown adds, for the caller to overrule:**

- The starter alone is a combination, and each value of a starter option is tested on the starter alone (Task 2,
  decision 2).
- The name of a combination reads as the arguments of `new` (Task 2, decision 6).
- The CI job installs its tools without mise, and the test trusts its temporary folder for mise on a machine
  (Task 3, decision 7).
- `RunTests` is a method of its own, and `new` does not run the `test` commands (Task 1, decision 9).

**Notes.**

- No task touches a category of a high blast radius. The story changes the read of `starter.yml`, adds a list that
  only a test reads, and adds a CI job of this repo. This is the honest assessment, and not a step that was skipped.
- The tasks do not change a template of the catalog, so `clerk mutants` covers Tasks 1 and 2. Task 3 changes no
  production code, and its proof is the full run and the break by hand (`tasks/learnings.md:86-91`).
- The three combinations of today take some minutes on this machine with a warm module cache. Each new feature
  and each new value adds a repo, so the time of the CI job grows with the catalog. A run of the combinations at the same
  time, or a matrix of jobs, is the next step when the job gets too slow.
