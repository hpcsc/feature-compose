# Implementation learnings

## Tests that run shell commands start in a temporary folder
- Type: constraint
- Recorded: 2026-11-08
- Observed: task 2 — us-001-new-from-starter
- Learning: clerk mutants runs the tests of a package in the source tree, so a mutant that drops cmd.Dir runs the commands of a test in the package folder and leaves files there. Each test that runs a shell command calls t.Chdir(t.TempDir()) first.
- Apply when: writing a Go test of code that runs an external command or writes a file relative to the working folder

## The catalog holds Go files as .tmpl and marks scripts with #!
- Type: convention
- Recorded: 2026-11-08
- Observed: task 5 — us-001-new-from-starter
- Learning: go:embed refuses a folder with its own go.mod, and go build ./... compiles any .go file under catalog/, so go.mod and each .go file of a starter or feature end in .tmpl, which the write removes. go:embed keeps no file mode, so only a rendered file that starts with #! is written as an executable.
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
- Observed: task 7 — us-002-add-cli
- Learning: task test:e2e runs vitest in Docker, and Docker does not answer on this machine. The e2e tests also run on the host: go build -o <bin> ./cmd/feature-compose, then EXECUTABLE=<bin> BUILD_TAG=dev npx vitest run <test file> -t <name> in e2e/. A run that skips them leaves the e2e criteria of the story unrun.
- Apply when: a criterion of a story has an e2e test, or task test:e2e fails because Docker does not answer

## Fold the fixes of an audit round hunk by hunk
- Type: pattern
- Recorded: 2026-11-09
- Observed: us-002-add-cli
- Learning: A fix that changes code of several task commits is split by the commit that last changed each hunk (git blame on the parent of the fixes), and each part becomes a fixup of that commit; only a part whose fold gives a conflict stays its own commit. A test added at the line where a later commit also adds a test gives a conflict, so put it next to lines of its own target commit.
- Apply when: folding the fixes of an audit round, most of all in catalog/catalog_test.go, which holds tests of every task

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
- Learning: A template function is one FuncMap entry in catalog/selection.go, a unit test of its parse and one render test; in US-002 the task that added githubRepository was called low certainty and passed on the first attempt. A breakdown can call such a task high certainty.
- Apply when: decomposing a story that adds a template function to the catalog
