# Catalog Registry: Implementation Notes

These notes support the goals G1 to G10 of the flow proposal. Phase 1 delivers G1 to G6 and G10. Phase 2 delivers
G7 to G9.

## Constraints From the Current Code

| Where | What the code does now | What the change must respect |
| --- | --- | --- |
| `internal/config/file.go:18` | `Load` reads one key, `catalog`, and stops when the file does not exist | `Load` reads `catalogs:`. With no file, the config has no catalogs |
| `internal/cmd/root.go:34` | Each command gets a loader of one `catalog.Catalog` | Each loader gives a `catalog.Catalogs` |
| `internal/cmd/catalog_flag.go:17` | `--catalog` takes a folder | `--catalog` takes a folder or the name of a catalog in the config |
| `internal/form/draft.go:38`, `:50` | The form uses only `Definitions` and `Starter` of the catalog | `Catalogs` gives both methods, so the form changes only the type of its parameter |
| `internal/catalog/selection.go:139` | The preview lists only the `generate` commands, in the section `Generate` | The preview lists each command that `new` runs, in the section `Commands` |
| `internal/catalog/selection_file.go:15` | `.feature-compose.yml` records the starter, but not its catalog | The file gets a `catalog` block |
| `internal/cmd/init.go:39` | `init` tells the user to write `catalog:` in the config | `init` names `catalog add` |
| `Taskfile.test.yml:18`, `:23` | CI runs `check --catalog catalog` | A catalog from a folder keeps the short names, so the output of CI stays as it is |
| `e2e/testUtils.ts:38`, `e2e/tests/check.test.ts:45` | The e2e tests write `catalog:` in the config | They write `catalogs:` |
| `README.md:101` | Get Started clones the repo by hand | Get Started runs `catalog add` |
| `internal/release/client.go:102` | `update` sends `GITHUB_TOKEN` or `GH_TOKEN` to GitHub | Phase 2 gives the same token to git (G9) |

## Decision Ledger

"Decided" means that the user made the decision. "Proposed" means that the user has not confirmed it yet.

| ID | Decision | Reason | Status |
| --- | --- | --- | --- |
| D1 | git gets each catalog, not the GitHub API | git reads a private repo with the SSH key or the credential helper that the user already has. It works with each host, and it has no limit of 60 requests an hour | Decided |
| D2 | A source is a git URL or a local folder. The tool uses a local folder where it is | A catalog author changes a local catalog and tests it with no commit | Proposed |
| D3 | A repo takes its features only from the catalog of its starter | A feature that targets a starter of another catalog needs targets and versions across catalogs | Decided |
| D4 | The subcommands are `add`, `list`, `sync`, `remove` and `set` | `update` already means "replace the binary", and one word must have one meaning | Decided |
| D5 | `new` gives no warning for a catalog that the user has not synced for a long time | The user asked for no warning | Decided |
| D6 | The key `catalog:` goes away, and the config holds only `catalogs:`. The tool stops on the old key, and names `catalog add` | The tool has one user, so no fallback is needed. The message saves a search | Decided |
| D7 | A starter of a catalog in the config has the full name `<catalog>/<starter>`. A short name works while one catalog has that starter. A catalog from `--catalog <folder>` has no name, so its starters keep short names | Two catalogs can have a starter with one name. CI and the tests run `check --catalog catalog` | Proposed |
| D8 | `.feature-compose.yml` records the name, the source, the path, the ref and the commit of the catalog | A user can make the same repo again (G5), also after a sync | Proposed |
| D9 | A clone lives in `$XDG_DATA_HOME/feature-compose/catalogs/<name>`, or in `~/.local/share/feature-compose/catalogs/<name>` with no `XDG_DATA_HOME` | A clone is data, not config. A user can delete it, and `sync` makes it again | Proposed |
| D10 | git runs with `GIT_TERMINAL_PROMPT=0`, with no input, and in a session of its own with no terminal | The tool keeps the output of git, so a question of git or ssh waits where nobody sees it. With no terminal, git and ssh stop at once. A `GIT_SSH_COMMAND` from the tool replaces the `core.sshCommand` of the user, which can name the key of a private repo | Proposed |
| D11 | `--catalog` takes a folder first, and then the name of a catalog in the config | CI passes the folder `catalog`, and that must keep its meaning | Proposed |
| D12 | `add starter` and `add feature` write only into a catalog from a local folder | The next `sync` of a clone removes each change in it | Proposed |
| D13 | `new` never gets new content. Only `catalog sync` does | A catalog cannot change between two runs of `new`, and `new` works with no network | Proposed |
| D14 | `catalog add` checks only that the path holds a `starters/` folder, and does not run the full `check` | The user asked for this check only. The full `check` of a starter with a generator also needs `ast-grep` | Decided |
| D15 | Phase 2 holds `show` by catalog, a central index of known catalogs, and a token for git in CI | Phase 1 is complete without them | Proposed |
| D16 | The preview of `new` lists each command that `new` runs, in the section `Commands`: the `generate` command and each `after_render` command of each folder, in the order in which `new` runs them. This section replaces the section `Generate` | A catalog runs its commands on the machine of the user, so the user sees each one before `new` runs it (G10) | Decided |
| D17 | The index of known catalogs is the file `catalogs.yml` on the branch `main` of this repo. The tool gets it over HTTPS for each `catalog add <name>` with no source, and for each `catalog list --known`. `FEATURE_COMPOSE_INDEX_URL` gives another index | A new entry needs no release of the tool, and a team can keep an index of its own (G8) | Proposed |
| D18 | With `GITHUB_TOKEN` or `GH_TOKEN`, git sends the token as the header `Authorization: Basic` for each source on the GitHub server, which is `GITHUB_SERVER_URL` or `https://github.com`. The header goes in `GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_<n>` and `GIT_CONFIG_VALUE_<n>` | The token is not in the URL, so it is not in `.git/config` or in a message. The process list does not show the environment. A source on another server does not get the token (G9) | Proposed |
| D19 | The tree of `show` has a row for each named catalog, with its starters under it. In a catalog, the tree and the details use the names that the catalog gives. The JSON keeps the full names | A person reads the names in a group. A script needs one key for each starter and each feature (G7) | Proposed |
| D20 | Any server gets a token through `FEATURE_COMPOSE_GIT_SERVER`, `FEATURE_COMPOSE_GIT_TOKEN` and `FEATURE_COMPOSE_GIT_USERNAME`, in the header of D18. A token goes only to its scheme, host and port, only over `https` or to `localhost`, and only to the git commands that reach the server | CI on GitLab or Bitbucket needs a token too. A token in the URL goes into `config.yml` and into the `.git/config` of the clone | Decided |

## The Config File

```yaml
catalogs:
  hpcsc:
    source: https://github.com/hpcsc/feature-compose
    path: catalog
    ref: main
  team:
    source: /Users/me/code/team-catalog
```

- A name matches `^[A-Za-z0-9][A-Za-z0-9._-]*$`, the rule of a repo name. So it cannot hold `/`.
- `catalog add` writes a local source as a full path. A source that starts with `~/` is in the home folder.
- A source is a git URL when it holds `://`, or when it has the form `user@host:path`. Each other source is a local
  folder.
- The old key stops `Load` with this message: "<path of config.yml> has the key catalog, which feature-compose does
  not read: remove the key, and then add the catalog with feature-compose catalog add <name> <folder>". The key also
  stops `catalog add`, so the message names the removal first.

## The Git Commands

| Step | Commands |
| --- | --- |
| `add` | `git clone --quiet -- <source> <clone>`, then the checkout of the ref below |
| Find the revision of a ref | No ref: `origin/HEAD`. A ref for which `git rev-parse --verify --quiet origin/<ref>^{commit}` succeeds: `origin/<ref>`, so a branch follows its newest commit. Each other ref: the ref, as a tag or a commit |
| Checkout | `git -C <clone> checkout --quiet --detach <revision>` |
| `sync` | `git -C <clone> fetch --quiet --tags --force origin`, then `git -C <clone> remote set-head origin --auto`, then the checkout |
| The commit | `git -C <clone> rev-parse HEAD` |

When git fails, the message names the source and tells the user to check `git clone <source>`. It includes the
output of git.

## Types

```go
// package catalog
type Origin struct {
	Name, Source, Path, Ref, Commit string
}

func FromOrigin(fsys fs.FS, origin Origin) Catalog

type Catalogs struct{ all []Catalog }

func NewCatalogs(catalogs ...Catalog) Catalogs
func (cs Catalogs) Starter(name string) (Starter, error)
func (cs Catalogs) Definitions() (Definitions, error)
func (cs Catalogs) Check(ctx context.Context) ([]string, error)
func (cs Catalogs) Combinations() ([]Combination, error)
func (cs Catalogs) TestCombination(ctx context.Context, combination Combination, name, output string) error
func SplitFullName(fullName string) (catalogName, name string)

// package registry
type Instance struct {
	file   config.File
	clones string
}

func Open() (*Instance, error)
func (r *Instance) Add(ctx context.Context, name string, entry config.Entry) (catalog.Origin, error)
func (r *Instance) Sync(ctx context.Context, names []string) ([]Synced, error)
func (r *Instance) Set(ctx context.Context, name string, change Change) (catalog.Origin, error)
func (r *Instance) Remove(name string) error
func (r *Instance) List(ctx context.Context) ([]catalog.Origin, error)
func (r *Instance) Catalogs(ctx context.Context) (catalog.Catalogs, error)
func (r *Instance) Catalog(ctx context.Context, name string) (catalog.Catalog, error)
func (r *Instance) Folder(name string) (string, error)
```

- `Catalogs.Definitions` gives full names for the starters and the features of each named catalog, and for the
  targets and the needs of each feature. Then `Definitions.ForStarter` and the browser work with no change.
- `Catalogs.Check` puts `the catalog <name>: ` before each mistake of a named catalog.
- `Combination` gets the field `CatalogName`, and its text starts with the full starter name.
- `Instance.Folder` gives the folder of a catalog from a local folder, for `add starter` and `add feature` (D12).
- The source in `.feature-compose.yml` and in the output of `catalog list` has no password and no token: an HTTP or
  HTTPS URL loses its user part, and another URL loses its password.

## Changes by Layer

| Layer | Change |
| --- | --- |
| `config` | The schema of D6, with `Save`. With no file, the config has no catalogs |
| `registry` (new) | `add`, `sync`, `set`, `remove` and `list`. It also opens the catalogs with their commits. It runs git as D10 and D18 say |
| `index` (new) | Gets and reads the index of known catalogs (D17) |
| `browser` | A row for each catalog in the tree of `show`, and the names that a catalog gives (D19) |
| `catalog` | `Origin` in `Catalog`, `Starter`, `Selection` and `Combination`. `Catalogs`. The `catalog` block in `.feature-compose.yml`. The section `Commands` of the preview |
| `form` | The type of its parameter is an interface with `Definitions` and `Starter` |
| `cmd` | The `catalog` command and its subcommands. Loaders that give `Catalogs`. `--catalog` as D11 says. `add` as D12 says. The message of `init`. `new` prints the full starter name |
| e2e | The helpers write `catalogs:`. A test adds a catalog from a local bare repo, syncs it to a new commit, and makes a repo from it |
| Docs | The Get Started section of the README, the config section of `docs/install-and-configure.md`, the `catalog` commands and `--catalog` in `docs/commands.md`, and the message of `init` in `docs/start-a-catalog.md` |

## Test Plan

- **`config`:** a load and a save give the same entries. With no file, the config has no catalogs. The old key stops with
  its message. A source with `~/` is in the home folder. Invalid YAML stops, and the message names the file.
- **`registry`:** each test makes a bare repo in a temporary folder. `GIT_CONFIG_GLOBAL=/dev/null` and
  `GIT_CONFIG_NOSYSTEM=1` keep the git config of the machine out of the test. The cases:
  - `add` of a git source records the commit, and `add` of a local folder needs no clone.
  - `add` with a name that the config has stops.
  - A `file://` URL to a repo that does not exist stops at once, with the message about `git clone`.
  - A path with no `starters/` folder stops, and the clone is gone.
  - An unknown ref stops, and the entry stays as it was.
  - `sync` of a branch gets a new commit, and `sync` of a tag keeps its commit.
  - `remove` deletes the clone and the entry.
  - `set --ref` checks out the new ref.
  - `sync` clones the source again when the clone is gone.
- **`catalog`:** `Starter` with a full name, a unique short name, a shared short name and an unknown name.
  `Definitions` with full names. `Check` with the catalog name before each mistake. The text of each
  `Combination`. The `catalog` block of `.feature-compose.yml`. The full name in `Arguments`. The preview lists the
  `generate` command and each `after_render` command, in their order. A catalog with no name keeps each output
  as it is now.
- **`cmd`:** the output of each subcommand. `new` with two catalogs. `--catalog` with a name and with a folder.
  `add starter` into a clone stops.
- **e2e:** `catalog add`, a new commit in the source, `catalog sync`, and `new` from the catalog.
- **Phase 2:** the tree of `show` with two catalogs. `catalog add <name>` and `catalog list --known` with an index
  from a test server. `catalog add` from a test HTTP server that serves the repo only with the token, and a source
  on another server that gets no token.

## Technical Acceptance Criteria

- Each unit test and each e2e test passes.
- `task test:catalog` and `task test:combinations` give the same output as now.
- Each check of the definition of done of phase 1 in the flow proposal passes (G1 to G6, G10).
- Each check of the definition of done of phase 2 in the flow proposal passes (G7 to G9).
