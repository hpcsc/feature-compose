# Implementation learnings

## Tests that run shell commands start in a temporary folder
- Type: constraint
- Recorded: 2026-11-08
- Observed: task 2 — us-001-new-from-starter
- Learning: clerk mutants runs the tests of a package in the source tree, so a mutant that drops cmd.Dir runs the commands of a test in the package folder and leaves files there. Each test that runs a shell command calls t.Chdir(t.TempDir()) first.
- Apply when: writing a Go test of code that runs an external command or writes a file relative to the working folder

## The catalog holds Go files as .tmpl and marks scripts with #!
- Type: convention
- Recorded: 2026-11-09
- Observed: this run
- Learning: catalog/ is inside the Go module of the tool, so go build ./... and go vet ./... compile any .go file under it, and a go.mod under it splits the module. So go.mod and each .go file of a starter or a feature end in .tmpl, which the write removes. The write sets the mode from the contents and not from the catalog file: only a rendered file that starts with #! is written as an executable.
- Apply when: adding a starter or a feature to catalog/
## A delimiters glob matches the catalog path, .tmpl included
- Type: constraint
- Recorded: 2026-11-08
- Observed: task 4 — us-001-new-from-starter
- Learning: The starter matches each delimiters glob against the path of the template in the catalog before it removes .tmpl, so go/**/*.go does not match go/main.go.tmpl; write the glob for the name in the catalog.
- Apply when: writing the delimiters of a starter.yml or a feature.yml

## Port a scaffold template by rendering it with sentinel values
- Type: pattern
- Recorded: 2026-11-08
- Observed: task 6 — us-001-new-from-starter
- Learning: Copy the template without its .DS_Store files (Finder writes them, and scaffold fails to parse them), render it with scaffold new --no-prompt and sentinel values such as Project=zzsentinelproject and GoVersion=1.99.77 with the feature questions set, then replace each sentinel with its template expression. scaffold then removes the files and the branches of the other answers with their whitespace, which a hand edit of the conditionals does not do exactly.
- Apply when: porting a template of repo-templates into the catalog as a starter or a feature

## The audit raises the old defects of ported template code
- Type: recurring-finding
- Recorded: 2026-11-08
- Observed: task 6 — us-001-new-from-starter
- Learning: An audit round reads a ported starter as new code and raises defects of the generated service that the scaffold template already had; in US-001 eight of 29 findings were such defects, and the run declined them as out of scope. Say in the brief which files are ported unchanged, so the lenses spend the round on the tool.
- Apply when: auditing a branch that ports a starter or a feature from repo-templates

## MISE_TRUSTED_CONFIG_PATHS needs the real path on macOS
- Type: constraint
- Recorded: 2026-11-08
- Observed: task 9 — us-001-new-from-starter
- Learning: A repo that new makes has its own mise.toml, which mise refuses until someone trusts it; an e2e test sets MISE_TRUSTED_CONFIG_PATHS to realpathSync of its scratch folder, because tmpdir() gives /var/folders and mise compares the real path under /private/var.
- Apply when: writing an e2e test that runs go or task in a repo that new made

## clerk land integrates only with its own --integrate flag
- Type: constraint
- Recorded: 2026-11-08
- Observed: us-001-new-from-starter
- Learning: A bare clerk land reads only the integrate setting of the repo, not the --integrate of the request, so it archives the breakdown and leaves the branch standing; run clerk land --integrate when the request asks for integration, before the learn step, because land refuses after it.
- Apply when: landing an implement run in this repo whose request says --integrate

## gotestsum names a package even when no test of it ran
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 7 — us-002-add-cli
- Learning: gotestsum prints a line for a package whose test files the build tag unit excludes, with ∅ in place of ✓, so an assertion that the output contains '<package> (' passes when no test of that package ran. Assert on the packages whose line starts with ✓.
- Apply when: writing an e2e test or a criterion that says the unit tests of a package of a generated repo pass

## The e2e tests run on the host when Docker does not answer
- Type: constraint
- Recorded: 2026-11-09
- Observed: this run
- Learning: task test:e2e runs vitest in Docker, and Docker does not answer on this machine. The e2e tests also run on the host: CGO_ENABLED=0 go build -ldflags '-X github.com/hpcsc/feature-compose/internal/version.releaseTag=v0.1.0' -o bin/feature-compose-e2e ./cmd/feature-compose, then EXECUTABLE=../bin/feature-compose-e2e BUILD_TAG=v0.1.0 CATALOG_DIR=../catalog npx vitest run <test file> in e2e/ (task test:e2e:local does the same). The update tests need the tag v0.1.0, and new needs CATALOG_DIR, because runCli writes a config file that names it. A run that skips them leaves the e2e criteria of the story unrun.
- Apply when: a criterion of a story has an e2e test, or task test:e2e fails because Docker does not answer
## Fold the fixes of an audit round hunk by hunk
- Type: pattern
- Recorded: 2026-11-09
- Observed: us-003-api-keys
- Learning: A fix that changes code of several task commits is split by the commit that last changed each hunk (git blame at the tip before the fold), and each part becomes a fixup of that commit; only a part whose fold gives a conflict stays its own commit. Build each fixup from the fixed SHA of the tip and never from HEAD, which moves with each fixup commit, and check that the index of the last fixup equals the work tree before the rebase. A test added at the line where a later commit also adds a test gives a conflict, so put it next to lines of its own target commit. clerk fixup mark refuses these files as ambiguous, because catalog_test.go, starter_test.go and selection_test.go hold tests of every task.
- Apply when: folding the fixes of an audit round or a design check, most of all in the umbrella test files of internal/catalog/
## A struct that a template function returns has exported fields in the design
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 7 — us-002-add-cli
- Learning: text/template reads only exported fields, so a template function that returns a struct, such as githubRepository, needs exported fields. Record them in the design at the task that adds the function; the design check at the last task refuses an exported field that the design does not name.
- Apply when: adding a template function to the catalog that returns a struct

## A new template function of the catalog is routine work
- Type: pattern
- Recorded: 2026-11-09
- Observed: task 6 — us-002-add-cli
- Learning: A template function is one FuncMap entry in internal/catalog/selection.go, a unit test of its parse and one render test; in US-002 the task that added githubRepository was called low certainty and passed on the first attempt. A breakdown can call such a task high certainty.
- Apply when: decomposing a story that adds a template function to the catalog

## clerk mutants does not change the templates of the catalog
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 6 — us-003-api-keys
- Learning: clerk mutants changes only the Go code of the tool, so a task whose code is a .go.tmpl of a starter or a feature gets 'not checked'. Prove its tests on a written repo: write the repo with the real binary, apply a list of breaks to the written files one at a time, run go test after each, and copy the file back. A break that leaves a variable unused does not build; write it as a condition with && false or || true.
- Apply when: a task changes the Go templates of a starter or a feature, or adds a feature with its own Go package

## New Go code of a feature costs proof, not construction
- Type: pattern
- Recorded: 2026-11-09
- Observed: task 6 — us-003-api-keys
- Learning: The task that added api-auth, with a new Go package and no precedent, was called low certainty and went green on the first attempt. Its cost was the proof: a written repo, a list of breaks by hand, and an e2e run that starts the API. A breakdown can call the code of such a feature medium, and plan the proof as part of the task.
- Apply when: decomposing a story that adds a feature with new Go code to the catalog

## env v6 reads a config field through UnmarshalText
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 6 — us-003-api-keys
- Learning: The config loader of go-event-sourcing-service is caarlos0/env v6, which reads a field whose type implements encoding.TextUnmarshaler through UnmarshalText, also for a slice type. So a feature puts the parse and the check of a setting in UnmarshalText of its config type, and a bad value stops config.Load at the start of the API; the error of env names the field and holds the text that UnmarshalText returns.
- Apply when: a feature adds a field to api.Config that needs a parse or a check

## An e2e test can start the written API without Postgres
- Type: pattern
- Recorded: 2026-11-09
- Observed: task 7 — us-003-api-keys
- Learning: pgxpool.New opens no connection before the first query, and POST /accounts with the body {} gets the 400 of the handler before any query. So an e2e test builds ./cmd/api of a written repo, starts it on a free port with DB settings that point to no database, and checks the answers of a middleware; startedApi in e2e/tests/new.test.ts does this.
- Apply when: writing an e2e test that checks a request to the API of a written go-event-sourcing-service repo

## task run starts the local API with task api:run
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 6 — us-003-api-keys
- Learning: In go-event-sourcing-service, task run loads .tmuxp.yaml, whose pane runs task api:run, which runs go run ./cmd/api with the env of Taskfile.api.yml; docker-compose.local.yml does not start that API. task passes the variables of the shell to its command. So a doc that tells how to give the local API a setting names task api:run, for example API_KEY_HASHES=<hash> task api:run.
- Apply when: writing the docs of a feature that needs a setting of the API on the machine of a developer

## A test that reads an env variable must clear it first
- Type: recurring-finding
- Recorded: 2026-11-09
- Observed: task 6 — us-003-api-keys
- Learning: A generated test that checks the absence of an environment variable, such as API_KEY_HASHES, fails on a machine whose shell sets it. The audit raised this as a medium runtime finding. The test calls t.Setenv(name, "") and then os.Unsetenv(name), so t.Setenv puts the old value back at the end.
- Apply when: writing a test, in the tool or in a template, that reads or checks the absence of an environment variable

## A break by hand counts only when its diff changes what the test checks
- Type: constraint
- Recorded: 2026-11-09
- Observed: task 3 — us-004-ci-combinations
- Learning: sed 's/A/B/' changes only the first match on each line. In middleware.go.tmpl of api-auth, the first match of http.StatusUnauthorized is inside http.StatusText(...), so the break changed only the body text, the status stayed 401, and task test:combinations passed. Read git diff of the break before the run, and use the g flag when one line holds the value two times.
- Apply when: proving a test of a template or of the tool with a break by hand that sed or a script applies

## A loop over a Go map that returns at the first match goes over sorted keys
- Type: pattern
- Recorded: 2026-11-09
- Observed: task 2 — us-004-ci-combinations
- Learning: clerk mutants puts a break at the end of a for range over a map. With a random key order, a test kills that mutant only in the runs where the matching key comes late, so a run can report it LIVED and the next run KILLED. feature.addsToOneExtensionPointWith loops over slices.Sorted(maps.Keys(f.addTo)), and its test gives the first feature a first sorted key that the second feature does not have.
- Apply when: a task adds a loop over a map that returns or stops at the first match, in catalog/ or anywhere in the tool

## clerk fixup mark refuses a file that no commit of the run touches
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-004-ci-combinations
- Learning: A fix can need a test in a package that no task changed, for example the round-trip test of Combination.String in internal/cmd/new_test.go. clerk fixup mark answers reason untouched for that file. Stage it alone and run git commit --fixup=<sha of the task commit that the fix corrects>, then clerk fixup replay folds it with the other fixups.
- Apply when: folding an audit fix that adds a test or a change in a file that no commit of the run touches
