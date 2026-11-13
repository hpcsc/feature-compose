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
- Recorded: 2026-11-12
- Observed: this run
- Learning: An audit round reads a ported starter or feature as new code and raises defects that the scaffold template already had: in US-001 eight of 29 findings, in US-017 two of seven, also with a brief that named the ported files. In US-017 the two (webhooks that check no signature, a gateway that blocks past its deadline) were runtime findings with executed evidence, so they earned another round by the severity rule. Decline them with the reason that the code is the template's, put any limit in the docs of the feature, and end the audit with clerk audit accept --early.
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
- Recorded: 2026-11-12
- Observed: this run
- Learning: task test:e2e runs vitest in Docker, and Docker did not always answer on this machine (it answered on 2026-11-12); check with timeout 20 docker info before a breakdown plans a criterion as a manual check. The e2e tests also run on the host: CGO_ENABLED=0 go build -ldflags '-X github.com/hpcsc/feature-compose/internal/version.releaseTag=v0.1.0' -o bin/feature-compose-e2e ./cmd/feature-compose, then EXECUTABLE=../bin/feature-compose-e2e BUILD_TAG=v0.1.0 CATALOG_DIR=../catalog npx vitest run <test file> in e2e/ (task test:e2e:local does the same). The update tests need the tag v0.1.0, and new needs CATALOG_DIR, because runCli writes a config file that names it. A run that skips them leaves the e2e criteria of the story unrun.
- Apply when: a criterion of a story has an e2e test or needs Docker, or task test:e2e fails because Docker does not answer

## Fold the fixes of an audit round hunk by hunk
- Type: pattern
- Recorded: 2026-11-11
- Observed: us-018-form
- Learning: A fix that changes code of several task commits is split by the commit that last changed each hunk, each part a fixup of that commit, built from the fixed tip and checked against the work tree before the rebase; only a part whose fold conflicts stays its own commit, with a message that says what it fixes, never inside another task's commit. A story that grows one bubbletea model over many tasks (US-018: internal/form/draft.go, model.go and model_test.go in six of nine commits) or moves tests between t.Run groups makes clerk fixup replay conflict; then build the task commits with git commit-tree: apply each fix as an exact string replacement to the file of each commit from its target task on (an earlier form of the old text, for example a field that a later task made a method, needs its own pair), gofmt, check that the last tree equals the work tree, and test each changed commit.
- Apply when: folding the fixes of an audit round or a design check, most of all in internal/form/, in the umbrella test files of internal/catalog/, or when clerk fixup mark comes back ambiguous

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
- Recorded: 2026-11-12
- Observed: us-019-api-prefix-cors
- Learning: clerk mutants changes only the Go code of the tool, so a task whose code is a .go.tmpl of a starter or a feature gets 'not checked'. Prove its tests on a written repo: write the repo with the real binary, apply a list of breaks to the written files one at a time, run go test after each, and copy the file back. A break that leaves a variable unused does not build; write it as a condition with && false or || true. The audit also skips such a diff: in US-019 its mutants and design lenses did not run, because their selectors count no .go.tmpl as Go, so the breaks by hand are the only mutation proof of template code.
- Apply when: a task changes the Go templates of a starter or a feature, or adds a feature with its own Go package, or an audit reads a branch whose Go code is only .go.tmpl

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
- Recorded: 2026-11-10
- Observed: us-010-dry-run
- Learning: A fix can need a test or a new file in a place that no task commit touches, for example the round-trip test of Combination.String in internal/cmd/new_test.go, or a new option_values.go of a regrouping. clerk fixup mark answers reason untouched for that file. Stage the files of the fix and run git commit --fixup=<sha of the task commit that the fix corrects>, then clerk fixup replay folds it with the other fixups. clerk fixup replay folds only fixup! commits: to reword a commit message, make it with git commit --fixup=reword:<sha> and fold it with GIT_SEQUENCE_EDITOR=true git rebase -i --autosquash <base>.
- Apply when: folding an audit fix that adds a test, a new file or a change in a file that no commit of the run touches, or rewording the message of a task commit

## A loop over a growing queue makes mutants that time out the whole mutation run
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-005-cli-auth
- Learning: clerk mutants stops at 10 minutes and then refuses clerk finish and clerk audit accept with an incomplete run. In US-005, the walks of the needs were loops over a queue (for len(unvisited) > 0 { ...; unvisited = unvisited[1:] }); removing the shift or the visited check made each loop endless, five mutants waited for their timeout, and the run never completed. The same walks written as recursion over a visited map (selectWithNeeds, addNeeds, cycleFrom in internal/catalog/starter.go) give no endless mutant, and the run completed with 149 mutants in time.
- Apply when: writing a loop in the tool that walks a graph or a queue, most of all the needs of features in internal/catalog

## A Go composite literal with {{ breaks a .go.tmpl of the catalog
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-005-cli-auth
- Learning: The catalog renders each .go.tmpl with text/template and the delimiters {{ and }}, so a nested Go literal such as [][]string{{"Bearer " + key}} in a template stops the render with 'illegal number syntax'. Write the inner literals on their own lines ([][]string{ newline {"Bearer " + key}, newline }), or give the file other delimiters in feature.yml.
- Apply when: writing a Go test or Go code in a .go.tmpl of a starter or a feature

## An accept of a mutant belongs to the task, and a fold or a rename after it opens the row again
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-005-cli-auth
- Learning: clerk mutants accept records the key for the task that is in build. After the audit fixes, a fold or a rename in the same function makes clerk audit accept refuse on the same timed-out rows; accept them again in the audit step, and check that the last run has complete: true in .git/clerk/runs/<run>/mutants.json, because an incomplete run is also an open row.
- Apply when: clerk audit accept refuses on a row of clerk mutants after the audit fixes

## The list of bubbles binds h, l, q and Esc, and its help shows the default bindings
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-014-show-catalog
- Learning: bubbles v1.0.0 list.DefaultKeyMap binds h and l to the previous and next page, and q and Esc to quit, and the full help (?) of a list with more than one page shows each binding with its own help text. A TUI with vim keys in internal/browser replaces PrevPage, NextPage and Quit with bindings that have no h, l or Esc, each with its own help text; the model catches l and Enter before the list. Without the new NextPage, ? tells the user that l goes to the next page.
- Apply when: adding or changing a key of a list of bubbles in the TUI of show (US-014) or of new (US-012)

## A test of a bubbletea model sends the window size first and runs the commands with a short timeout
- Type: pattern
- Recorded: 2026-11-09
- Observed: us-014-show-catalog
- Learning: internal/browser/model_test.go tests the model through Update and View: it sends tea.WindowSizeMsg first, because a list draws nothing before it has a size, and it runs each command that Update returns in a goroutine with a 100 ms limit, because the filter of the list comes back as a message of a command (FilteredItemsMsg) while the blink of the cursor waits about half a second. A tea.QuitMsg from a command is how a test sees that the TUI stops. Remove the escape codes of View before a test compares text, so the tests pass with CLICOLOR_FORCE=1 and NO_COLOR=1.
- Apply when: writing a test of a bubbletea model, in internal/browser or in the TUI of new

## clerk mutants kills a mutant only with the tests of its own package
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-014-show-catalog
- Learning: A mutant in internal/catalog runs only the tests of internal/catalog (and caller gaps report the rest), so the [] and {} of catalog.Definitions lived although TestShowCommand in internal/cmd checks the whole JSON. The package of the code needs its own test of the behaviour, here a JSON test of the definitions in TestCatalog.
- Apply when: a task adds code in one package whose behaviour another package tests, for example internal/catalog and internal/cmd

## fatih/color decides the colour from stdout, also for text that goes to stderr
- Type: constraint
- Recorded: 2026-11-09
- Observed: us-014-show-catalog
- Learning: color.NoColor is set from whether stdout is a terminal, so color.New(...).Fprintln(os.Stderr, ...) colours by the wrong stream. Run in internal/cmd/root.go writes errors through writeError, which disables or enables the colour from its own stream and from NO_COLOR.
- Apply when: writing coloured text to stderr or to any stream other than stdout in the tool

## A criterion that pairs each item with a relation holds for every item
- Type: pattern
- Recorded: 2026-11-10
- Observed: us-010-dry-run
- Learning: US-010 said the preview shows 'each feature with the feature that needs it', and the breakdown decided that a feature a flag gives shows only 'command line', also when another feature needs it. The match-request step found the narrowing, and the user chose to show both ('cli: command line, needed by cli-auth'). Keep the relation for every item, also for the items that have a second source.
- Apply when: decomposing a story whose criterion pairs each item with a relation or a source, for example each feature with the features that need it, or each option with its source

## The words of options, values and needs in internal/catalog
- Type: convention
- Recorded: 2026-11-10
- Observed: us-010-dry-run
- Learning: options is the declared options (type options on Starter and feature); values is the chosen values of a selection (type optionValues, field values); given is what a flag gave (given, givenByFeature, and givenStarterValues/givenFeatureValues in internal/cmd); needs is what a feature needs, and neededBy, asNeed and Need are the features that need it; selectedFeature is the record of a feature in a Selection, and Starter.featuresWithNeeds gives the catalog features. All four design checks of US-010 blocked on names that broke these.
- Apply when: naming a type, field, method or function in internal/catalog or internal/cmd that holds options, option values, flag values or needs

## A full clerk mutants run of a story in internal/catalog passes its 10 minutes
- Type: constraint
- Recorded: 2026-11-11
- Observed: task 3 — us-007-oauth
- Learning: A story that changes several files of internal/catalog, internal/form, internal/screen or internal/browser makes hundreds of mutants against the run base (US-018: 27 files, 309 mutants), and clerk mutants stops at 10 minutes with an INCOMPLETE row. One task can pass the limit too: the parser of when in US-007 made 269 mutants in 5 files, and each runs the tests of internal/catalog (about 10 seconds) on 4 workers. To read only the lines of a fix, run clerk mutants --base HEAD while the fix is still uncommitted (git add -N a new file first), then fold. When a task or a run against the base stops at the limit, run mutants run --base HEAD --tags unit --workers 6 with no --limit and a --json report (about 15 minutes), and accept the INCOMPLETE key with that report as the reason.
- Apply when: running clerk mutants on a task or after the audit fixes of a story that changes internal/catalog, internal/form, internal/screen or internal/browser, or clerk finish or clerk audit accept refuses on an INCOMPLETE mutants row

## Test new in a pseudo-terminal that has a size, with a context that ends
- Type: pattern
- Recorded: 2026-11-11
- Observed: task 9 — us-018-form
- Learning: TestNewCommand starts the TUI only when stdin and stdout are terminals, so runInTerminal opens a creack/pty, sets its size (a pty of size 0 makes the TUI draw nothing), types each key alone with a pause of 50 ms after a text shows (bubbletea reads a burst of letters as one KeyMsg), and gives new a context of 3 seconds as a limit only. Each test ends the form with a key (q, ctrl+c, or w and y): when the context kills the program, bubbletea v1.3.10 closes its cancelreader while the read loop calls Fd(), and task test:unit, which runs -race, fails. runNew gives an empty reader and the same limit, so a TUI that starts by mistake fails a test instead of waiting for keys.
- Apply when: writing a test in internal/cmd of when new starts the TUI, or of what new does after the TUI ends

## tuistory starts each command about 5 seconds late
- Type: constraint
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: In the terminal of tuistory a command that writes to the terminal draws its first output about 5 seconds after the start, so the first waitForText of each screen test passes { timeout: 15_000 }, a rule of docs/e2e-tests.md. Until US-022 the screen tests of show waited the default 5 seconds, and a different one failed with an empty screen in most runs, also when show.test.ts ran alone.
- Apply when: writing an e2e screen test with openCli, or reading a waitForText timeout of about 5 seconds as a defect of the TUI

## The shell refuses > onto a file that exists
- Type: constraint
- Recorded: 2026-11-10
- Observed: us-012-tui
- Learning: The zsh of this machine sets noclobber, so cat > file or cmd > file onto a file that exists fails and the next commands run with the old content; write with >| or with python, and note that mk is an alias and sed is GNU sed (sed -i with no suffix).
- Apply when: writing or rewriting a file with a shell redirect, or naming a shell function, in a command of a run

## A criterion that a breakdown defers needs a story that names it
- Type: pattern
- Recorded: 2026-11-11
- Observed: us-018-form
- Learning: The breakdown of US-018 deferred the when of an option and the values of a need to US-007, but US-007 named neither the form nor when (US-013 adds the first when), so the deferral lived only in the breakdown until match-request found it. When a breakdown defers a criterion because the catalog or the code lacks a capability, name the story that adds the capability and check that its criteria name the deferred behaviour, or ask for a criterion there.
- Apply when: decomposing a story with a criterion that needs a capability of the catalog or of the tool that a later story adds, for example when, values of a need, or a new source of a value

## A move of code into a new package makes each moved line a changed line
- Type: pattern
- Recorded: 2026-11-11
- Observed: task 1 — us-018-form
- Learning: Task 1 of US-018 moved the look of show into internal/screen and was rated high certainty, but it took two attempts: clerk mutants read every moved line as new, and the tests of show had gaps (a terminal of exactly 40 by 8, a title line too narrow for the counts, the width of the tree pane) that the move task then closed with new tests. Size a move by the gaps in the tests of the old code, not by the size of the move.
- Apply when: assessing the certainty or the size of a task that moves code between packages or into a new shared package

## An edit of a value in a bubbletea model of this repo is routine work
- Type: pattern
- Recorded: 2026-11-11
- Observed: task 6 — us-018-form
- Learning: Task 6 of US-018 (Enter edits the name with a bubbles textinput, or lists the values of an option) was rated low certainty and passed in one attempt with no lint finding: a textinput with cursor.CursorStatic sends no blink commands, and a list drawn with screen.RowLine and screen.SelectedRowLine needs no state of a library. The extra attempts that huh fields cost do not apply to plain bubbles.
- Apply when: assessing the certainty of a task that adds a text input or a list of choices to the form of new (internal/form) or to show (internal/browser)

## Fold a fix into the task that first makes its state happen
- Type: convention
- Recorded: 2026-11-11
- Observed: us-018-form
- Learning: In US-018 a round 1 fold put the drop of kept option values into Task 3, where draft.selected came in, but a value can be kept only from Task 6, so the code did nothing for three commits and the message of Task 6 told a change that its diff did not hold; a design-check fix that renamed the fields of Task 3 went into Task 4, where the check ran. run-verifier flagged both. Pick the commit of the task whose behaviour needs the fix, and fold a rename into the task that added the name.
- Apply when: picking the target commit of an audit fix or a design-check fix, with clerk fixup mark or with a git commit-tree rebuild

## clerk audit accept counts the lines of internal/form that the tests of internal/cmd do not run
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-018-form
- Learning: After the audit fixes of US-018, clerk audit accept refused on 14 CALLER GAP rows: internal/cmd calls the form only through form.New, tea.Program and Model.Confirmed, so its tests do not run the details, the edits or the messages of the form. TestModel of internal/form runs each of those lines through the public API, so accept each row with that reason (check it with a coverage profile of internal/form) instead of a pty test in internal/cmd for each row.
- Apply when: clerk audit accept or clerk mutants reports CALLER GAP rows for internal/form, internal/browser or another package that internal/cmd runs through a TUI

## A feature that caches an outside service needs its rules of time and concurrency planned
- Type: pattern
- Recorded: 2026-11-11
- Observed: task 4 — us-007-oauth
- Learning: Task 4 of US-007 (the oauth method, which reads OIDC discovery and the signing keys of an issuer) was rated low certainty and went green at the first attempt, but two audit rounds found six runtime defects in its cache: a lock held across the HTTP calls, no pause after a failed read, keys that never expire, a cancelled request that spends the pause, and a known key that waits for a refresh. Plan them in the task: one read at a time in a goroutine with a context of its own, a known value that never waits, an expiry that keeps the old values while a refresh fails, a pause after each kind of read, and a clock from the constructor with tests of the cache in the package.
- Apply when: decomposing or building a feature whose written code reads an outside service and keeps the result, for example signing keys, tokens, discovery or remote config

## clerk audit run --recheck takes the JSON text, not a file path
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-007-oauth
- Learning: In US-007 a path given to --recheck stopped the round before it began with 'needs a JSON array of findings, and this will not parse as JSON', while the background command still ended with exit 0. Pass --recheck "$(cat <file>)", and read the first lines of the log once to see 'scope · round N' before you wait.
- Apply when: launching a second or later audit round with clerk audit run --recheck

## task test:unit does not build the combinations test
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-007-oauth
- Learning: internal/catalog/combination_test.go has the build tag combinations, so task test:unit and clerk mutants never build it. A call in it that lost its import on main (commit 2c7fc25) showed only when task test:combinations ran, as 'undefined: catalog', and no combination ran. Run go vet -tags combinations ./internal/catalog/ after a change to an API of internal/catalog that the test calls, and before task test:combinations.
- Apply when: changing a signature or a type of internal/catalog that combination_test.go uses, or before running task test:combinations

## A design-check fix that renames a type is a design change of its own
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-007-oauth
- Learning: In US-007 the design check of needValues asked for a new name and a new shape; clerk design note then recorded valuesSetByNeeds as D3, and clerk step asked for a check of D3 before the audit could be accepted again, which cost one more agent, one more fold and one more mutants run against the base. Plan a design-check fix that renames or reshapes a type as its own change, and record its note before you close the earlier check.
- Apply when: fixing a finding of a design check with a new name or a new shape of a type

## clerk mutants finds a caller gap only when the diff changes the caller
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-006-check-links
- Learning: In US-006, the per-task clerk mutants runs of Tasks 2 to 4 changed only internal/catalog and reported no caller gap, but the run against the base after the audit, whose diff also held internal/cmd/check.go, reported 7: the folder, need, cycle and item field paths of the catalog that no test of internal/cmd ran. Closing them cost two more test folds and two more full mutants runs after the audit.
- Apply when: decomposing or building a story that adds a command in internal/cmd in one task and the paths of internal/catalog that the command reaches in later tasks; give each later task a test of the command that reaches its new path

## A rule about one feature belongs on feature in internal/catalog
- Type: pattern
- Recorded: 2026-11-11
- Observed: us-006-check-links
- Learning: The planned design of US-006 put the comparison of the item fields of the targets of a feature on Catalog and the rules of its needs on Starter; the audit found that Catalog.itemFieldsMistakes did not use the catalog, and the design check found that Starter.needMistakes read only the name of the starter, so both moved to feature (itemFieldsMistakes(starters), needMistakes(starterName, catalogFeatures), targetsStarter). That cost an audit fix and two design checks.
- Apply when: planning the design of a check, a rule or a message about one feature of the catalog, its targets or its needs, in internal/catalog

## A second read of the same catalog files hides the error checks of the first from clerk mutants
- Type: constraint
- Recorded: 2026-11-11
- Observed: us-006-check-links
- Learning: Catalog.Check in US-006 first read every feature and then called Catalog.Starter, which read the same files again; a mutant that removed an error check of the first read lived, because the second read failed on the same file with the same error (7 mutants). Building each starter from the one read of the features (readStarter with the features that target it) gave each error one path, and every mutant died.
- Apply when: writing code in internal/catalog that reads the starters or the features of the catalog more than one time in one operation, or reading live mutants on the error checks of a read

## A test of an HTTP answer reads the headers from recorder.Result()
- Type: recurring-finding
- Recorded: 2026-11-12
- Observed: us-019-api-prefix-cors
- Learning: httptest.ResponseRecorder.Header() is the live map, so a test that reads it still sees a header that the code sets after WriteHeader, which a real server never sends. The audit of US-019 raised this two times, in the CORS test of the starter and in the refusal tests of both methods of api-auth. Read the written headers from recorder.Result().Header; then a break that moves a Set after the write fails the test.
- Apply when: writing a Go test, in the tool or in a template, that checks the headers of an answer with httptest.NewRecorder

## env v6 gives a list of only blanks as blank entries
- Type: constraint
- Recorded: 2026-11-12
- Observed: us-019-api-prefix-cors
- Learning: With envSeparator, caarlos0/env v6 gives ALLOWED_ORIGINS=" " as [" "] and "," as ["", ""], and only an unset or empty variable as an empty list. In US-019 the server chose the default origins on an empty list while CORS dropped the blank entries, so a blank value allowed no origin at all (a medium runtime finding of the audit). Decide no value after the trim and the drop of blank entries, in the one place that owns the rule.
- Apply when: adding a list setting with a default to api.Config or to the config of a feature, or any setting that a feature trims before use

## emod validate fails on the account event model of the starter
- Type: constraint
- Recorded: 2026-11-12
- Observed: us-019-api-prefix-cors
- Learning: emod validate docs/event-model-account.emod of go-event-sourcing-service exits 1 with three lints of the model itself: missing-todo-list (WelcomeBonusReaction reads no view), clickbait-event (AccountClosed has one ID field) and god-view (AccountBalanceView reads five events). The breakdown of US-019 asked that it passes, and that criterion stayed open, because no route change can fix these lints.
- Apply when: decomposing a story that changes the event model files of the starter go-event-sourcing-service

## The e2e tests of show and of the form pick a starter by its place in the order of the names
- Type: constraint
- Recorded: 2026-11-12
- Observed: task 1 — US-016 go starter
- Learning: The TUI of show opens on the first starter in the order of the names, and the form of new lists the starters in that order, so its test presses j to reach a starter. A new starter that sorts first changes both: US-016 added go before go-event-sourcing-service and had to change the first test of show in a terminal, the test of show <starter> --output json, the unknown-starter messages, and the j presses of the form test with no argument (e2e/tests/show.test.ts, e2e/tests/new.test.ts). chi-api sorts before go.
- Apply when: decomposing or building a story that adds a starter to catalog/, for example chi-api in US-015

## The scheduler test of go-event-sourcing-service can fail under load
- Type: constraint
- Recorded: 2026-11-12
- Observed: task 2 — US-016 go starter
- Learning: TestScheduler/tick 'ticks at each interval with a later time each time' in the starter go-event-sourcing-service runs the scheduler at 1 ms and failed one time in a full e2e run of new, with two ticks out of time order; the same test passed when it ran again, and US-016 changed no file of the starter. A failure there in an e2e or combinations run is not evidence against the change under test: run the test again before you look for a cause in the branch.
- Apply when: an e2e run of new or task test:combinations fails in internal/domain/scheduler of a repo from go-event-sourcing-service

## Prove task run of a written service repo with Docker and a tmux server of its own
- Type: pattern
- Recorded: 2026-11-12
- Observed: US-017 process-manager
- Learning: Write the repo under .scratches/ of this repo (git ignores it, and Docker shares the home folder; a bind mount from /private/tmp arrives in the container as an empty folder). Run task run-dependencies, list the tables with docker compose exec db psql, start the panes with tmuxp load -d -L <socket> .tmuxp.yaml so that your tmux sessions stay apart, and call the API through the tasks of Taskfile.api.yml. tmux kill-server leaves the binaries of go run alive: stop them by their ports, and run docker compose down -v. In US-017 this proved the criterion of task run that the breakdown had left to a manual check.
- Apply when: decomposing or checking a criterion of a story that names task run, the migrations against Postgres, or the services of Docker Compose of a written go-event-sourcing-service repo

## A break by hand counts only when the broken code still builds
- Type: constraint
- Recorded: 2026-11-12
- Observed: US-017 process-manager
- Learning: A go test that fails to build exits non-zero like a test that catches the break, so a break that leaves an import or a variable unused reads as caught. In US-017 three breaks of a script read as caught until the script looked for [build failed]. Write a break as a condition (if false { ... }, && false) or a slice ([:7]), and read the name of the failed test in the output.
- Apply when: proving a test of a template or of the tool with a break by hand that a script applies

## A port of a template feature with connector files is routine after its first task
- Type: pattern
- Recorded: 2026-11-12
- Observed: US-017 process-manager
- Learning: In US-017 the breakdown called five of eight tasks medium certainty: new extension points, items in Go and in YAML, connector files and their tests. Each went green on the first try with a clean lint, because task 2 set the pattern that the later tasks repeated. The cost was the proof, not the code: breaks by hand found two gaps in the ported tests.
- Apply when: decomposing a story that ports a template feature into the catalog with connector files and items of extension points, for example push in US-021

## A package of more than one word is snake case, and a fake lives beside the code that it fakes
- Type: convention
- Recorded: 2026-11-12
- Observed: US-017 process-manager
- Learning: In the Go code of the catalog, a package folder of more than one word is snake case (use_cases, close_account, welcome_bonus, cmd/fake_services), and a fake has no package of its own: a test double goes into the package of what it fakes (domain.FakeAggregate, command.FakeHandler), and a fake of an outside system goes beside the gateway that calls the system (checks.FakeServices, submission.FakeNetwork). The template of repo-templates broke both rules (fakeservices, closeaccount, welcomebonus, domain/fake), the port of US-017 copied them, and neither the breakdown nor the audit caught them; the user did.
- Apply when: porting a template of repo-templates into the catalog, or adding a Go package, a command or a fake to a starter or a feature

## A job-level concurrency group does not keep the push order of a CI workflow
- Type: constraint
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: A job that an item of ci.jobs adds joins its concurrency group when its needs finish, not in push order, so a publish job on main can publish an older commit after a newer one, and a newly queued job cancels the pending job of a newer commit. The prerelease job of cli reads the head of main with git ls-remote --exit-code and publishes nothing when the head is not GITHUB_SHA; the go-cli template had a workflow-level group, which an item cannot add.
- Apply when: adding a job that publishes on main or on a tag to ci.jobs, for example the images of US-020, or porting a workflow whose concurrency group covers the whole workflow into an item

## The open tree of show must fit rowsOfTheWholeTree
- Type: constraint
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: Each extension point of a starter is one line of the open tree of show, and two tests of e2e/tests/show.test.ts read the whole tree, so they open the terminal with rowsOfTheWholeTree (60) rows and the other screen tests keep 40. After US-022 the tree of go (6 points) and go-event-sourcing-service (22 points) needs about 43 rows; in US-022 the tree passed 40 rows at Task 4 and the two tests lost their last line.
- Apply when: adding extension points to a starter or adding a starter to catalog/, for example in US-015, US-020 or US-021

## Compare a ported CI job with its source by its parsed steps
- Type: pattern
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: TestCatalog compares the steps of the release and prerelease jobs that cli adds to ci.yml with the steps of .github/workflows/release.yml and prerelease.yml of this repo, parsed as YAML, and names each deliberate difference in the test. It found a jq line with two spaces too many during the port, which no review of the YAML text showed.
- Apply when: porting a workflow or a job of this repo or of repo-templates into an item of ci.jobs, for example the image jobs of US-020

## A task that a feature adds must not have the name of a task of any of its targets
- Type: constraint
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: A feature with one set of files writes the same Taskfile into each target, and two tasks of one name stop Task. US-022 named task run for the CLI, but task run of go-event-sourcing-service starts the service, so the user chose task cli in both repos; the breakdown found the conflict only by listing the tasks of each target.
- Apply when: writing or decomposing a story whose feature adds tasks to the Taskfiles of more than one starter, for example US-015 (chi-api) or US-021 (push)

## actionlint and act do not run on this machine
- Type: constraint
- Recorded: 2026-11-13
- Observed: us-022-cli-tasks-ci
- Learning: actionlint and act are not installed, and the permission check refuses go run of a module that it must download, so a written GitHub workflow gets only the YAML checks of TestCatalog here. In US-022 the verification of the release jobs planned actionlint, and it did not run.
- Apply when: a breakdown plans actionlint, act or another tool fetched at run time as the check of a GitHub workflow
