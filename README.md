# feature-compose

## Install

The installer downloads the latest release for your platform, verifies its checksum, and puts the
binary where you can start using it. It needs `sh`, `curl`, `jq`, `tar`, and `gzip`.

```shell
sh <(curl -fsSL https://raw.githubusercontent.com/hpcsc/feature-compose/main/scripts/install.sh)
```

It asks for the release channel and the install directory. To skip the questions on the command line:

```shell
sh <(curl ...) --channel release --dir ~/.local/bin
```

The channel is `release` (the latest stable release) or `prerelease` (the latest build of `main`).
Without `--dir`, the binary lands in `~/.local/bin` (or `$INSTALL_DIR`). When more than one version is
available in the channel, the installer lists them for you to pick.

`GITHUB_TOKEN` or `GH_TOKEN` gives access to a private repository.

## Configure the catalog

`feature-compose new` and `feature-compose show` read the starters and the features from a catalog folder. The config file
`~/.config/feature-compose/config.yml` names the folder (`$XDG_CONFIG_HOME/feature-compose/config.yml` when
`XDG_CONFIG_HOME` is set):

```yaml
catalog: ~/code/feature-compose/catalog
```

The folder holds `starters/` and `features/`. The catalog of this repo is in `catalog/`. A path that starts
with `~/` is in your home folder, and another relative path is relative to the folder of the config file.
Without the config file, `new` and `show` stop and name the path of the file. `version` and `update` do not
read it.

`feature-compose update` also installs from a release, and it replaces this very binary, so the
installer and the update command do the same kind of job; use whichever you find convenient on a fresh
machine.

## Preview a Repo

`feature-compose new <starter> <name> --dry-run` prints a preview of the repo that `new` makes. It writes no file
and runs no command, so the output folder does not exist after the run:

```shell
feature-compose new go-event-sourcing-service my-service --feature cli --dry-run
```

```text
Repo: my-service
Starter: go-event-sourcing-service

Files:
  .dockerignore: the feature cli
  .env.integration: the starter
  ...
  cmd/my-service/main.go: the feature cli
  ...
```

The preview gives the name of the repo and its starter. Then it gives each file of the repo by its path, in the
order of the paths, with its owner: the starter, or the feature whose template writes the file.

Before it prints, `--dry-run` does the same checks as `new`. A selection that stops `new`, for example an unknown
feature or a value that does not match its pattern, stops `--dry-run` with the same message, and `--dry-run` then
prints nothing. `--dry-run` does not read the output folder, so an output folder that has files stops `new` but not
`--dry-run`.

The preview is text for a person, and its layout can change. A script reads the catalog with
`feature-compose show --output json`.

## Show the Catalog

`feature-compose show` shows the starters and the features of the catalog. It has two forms:

- **A TUI, for a person.** When stdin and stdout are both terminals, `show` opens a TUI with two panes. The left
  pane is a tree of the starters. Under each starter, the tree has two groups: its features and its extension
  points. Each starter and each group starts open. The right pane shows the details of the selected row.
  `feature-compose show <starter>` opens the TUI with that starter selected.
- **JSON, for a script.** `--output json` prints the catalog as JSON. When stdin or stdout is not a terminal,
  for example in a pipe, `show` prints JSON with no flag. `feature-compose show <starter>` prints only that
  starter and the features whose `targets` name it.

An unknown starter stops the command before the TUI starts, and the message lists the starters.

```shell
feature-compose show
feature-compose show --output json
feature-compose show go-event-sourcing-service --output json | jq '.features | keys'
```

The keys of the TUI:

| Key | Action |
| --- | --- |
| `j`, `k`, or the up and down arrows | Move down and up in the tree. |
| `l`, Enter, or the right arrow | Open the selected starter or group. At an open starter or group, select its first row. |
| `h`, Esc, or the left arrow | At a feature or an extension point, select its group. At an open starter or group, close it. At a closed group, select its starter. |
| `/` | Start a filter on the names of the starters, the features and the extension points. While you type the filter, each key goes to the filter. Enter keeps the filter, and Esc clears it. |
| Esc, with a filter | Clear the filter. |
| ctrl+d and ctrl+u, or Page Down and Page Up | Scroll the details down and up by half the pane. |
| `q` | Stop the TUI. While you type the filter, `q` goes to the filter, and ctrl+c stops the TUI. |

The filter keeps a starter whose name holds the text, with all its features and extension points. It also keeps
a feature or an extension point whose name holds the text, and opens its group and its starter. The filter ignores
case. When the filter clears, the rows that it opened close again.

The right pane shows these details:

| Row | Details |
| --- | --- |
| A starter | Its summary, its folders with their commands, and its options with their defaults and their values or patterns. |
| The group of the features | Each feature that targets the starter, with its summary. |
| A feature | Its summary, its needs, its options, and the extension points that it declares or adds items to. |
| The group of the extension points | Each extension point of the starter and of its features, with its item fields and its declarer, for example `cli.commands` of `cli`. |
| An extension point | Its summary, its file, its item fields, its declarer, and the features that add items to it. |

When the details do not fit the pane, its bottom border shows how far they are scrolled.

The JSON has the keys of `starter.yml` and `feature.yml`. Each object has each of its fields: a list with no entry
is `[]`, an object with no entry is `{}`, and a text with no value is `""`. Each list keeps the order of its file.
`<name>` is the name of a starter, a feature, a folder, an option or an extension point.

| Field | Meaning |
| --- | --- |
| `starters.<name>.summary` | One sentence about what a repo from the starter gets. |
| `starters.<name>.folders.<name>.path` | The path of the folder in the repo. |
| `starters.<name>.folders.<name>.after_render` | The commands that run in the folder after the tool writes the files. |
| `starters.<name>.folders.<name>.test` | The commands that CI runs in the folder to test a repo. |
| `starters.<name>.options.<name>.default` | The default of the option, as the file writes it, for example `github.com/hpcsc/{{ .Name }}`. |
| `starters.<name>.options.<name>.pattern` | The regular expression that a value must match, or `""`. |
| `starters.<name>.options.<name>.values` | The values that the option accepts, or `[]` for any value. |
| `starters.<name>.extension_points.<name>.summary` | The contract of an item of the extension point. |
| `starters.<name>.extension_points.<name>.file` | The file that renders the items. |
| `starters.<name>.extension_points.<name>.item` | The fields of each item. |
| `features.<name>.summary` | One sentence about what the feature adds. |
| `features.<name>.targets` | The starters that the feature works with. |
| `features.<name>.needs` | The features that the tool also selects when you select this feature. |
| `features.<name>.options.<name>.default` | The default of the option, as the file writes it. |
| `features.<name>.options.<name>.pattern` | The regular expression that a value must match, or `""`. |
| `features.<name>.options.<name>.values` | The values that the option accepts, or `[]` for any value. |
| `features.<name>.extension_points.<name>.summary` | The contract of an item of an extension point that the feature declares. |
| `features.<name>.extension_points.<name>.file` | The file of the feature that renders the items. |
| `features.<name>.extension_points.<name>.item` | The fields of each item. |
| `features.<name>.add_to.<name>` | The items that the feature adds to the extension point, each with the fields of that extension point. |

## Build

```shell
task build            # build ./bin/feature-compose
task run -- --help    # run the CLI from the source
```

## Version and update

```shell
feature-compose version                # the tag of a release or a prerelease, or the commit of any other build
feature-compose update                 # install the latest release
feature-compose update --prerelease    # install the latest prerelease, a build of main
feature-compose update --check         # only tell you whether this build is the latest
```

`internal/version` reports the version. A release build gets its tag from goreleaser, through
`-ldflags -X .../internal/version.releaseTag`. Any other build reports the short commit sha that Go
keeps in the build information, with `-dirty` after it when the working tree had changes.

`feature-compose update` downloads the archive for your platform from a GitHub release, checks it
against the `checksums.txt` of that release, and then replaces the binary. It names each step on
stderr, and in a terminal it shows how much of the download has arrived. `GITHUB_TOKEN` or `GH_TOKEN`
gives access to a private repository.

Releases and prereleases are two channels. Each command installs the latest build of its channel when
this build is a different one, so `feature-compose update` on a prerelease goes back to the latest
release. A build from a commit is not a release or a prerelease, so `feature-compose update` does not
replace it unless you add `--force`.

## Goreleaser

- Run goreleaser in local: `task release:local`. This will generate a snapshot build under `./dist`
- Create a release:

```shell
git tag vX.X.X
git push origin vX.X.X
```

This will trigger the release workflow, which runs the CI checks and then creates a Github Release with
binaries for MacOS and Linux.

Each push to `main` starts the prerelease workflow. It tags the commit with the next patch after the
latest release, the run number and the commit, for example `v0.2.1-42.g4829f92`, publishes that tag as a
prerelease, and then keeps only the 5 newest prereleases.

`On Demand Build` builds a snapshot of any ref from Github Actions and uploads the archives as artifacts.

## Test the Combinations of the Catalog

`task test:combinations` makes one repo for each combination of the catalog in `catalog/`. In each repo, it
runs the `after_render` commands and then the `test` commands of each folder. These are the combinations:

- Each starter alone
- Each starter with each feature that targets it, and the needs of the feature
- One for each value of an option that has `values`, with the other options at their defaults
- Each pair of features that add items to one extension point, except a pair in which one feature needs the
  other: the repo of that feature already holds both

Run it after you change a starter or a feature. CI runs it on each push, in the job `combinations`. The
catalog gives the list, so a new starter, feature or value needs no change to the task or to CI.

The output names each combination that fails as the arguments of `new` that make the same repo, for
example `go-event-sourcing-service --feature api-auth:method=api-key`, and shows the output of the command
that failed.

The task needs Go, Task, gotestsum, a C compiler and the network. Each repo takes some seconds when the Go
module cache has its modules, and some minutes when it does not.

## E2E Test

The end-to-end tests run the built binary in a terminal, send keys to it, and read the screen. They are
in `e2e/`, and [docs/e2e-tests.md](docs/e2e-tests.md) tells how they work.

```shell
task test:e2e          # in Docker, as CI does
task test:e2e:local    # on this machine, needs node
```
