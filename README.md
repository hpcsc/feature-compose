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

## Show the Catalog

`feature-compose show` shows the starters and the features of the catalog. It has two forms:

- **A TUI, for a person.** When stdin and stdout are both terminals, `show` opens a list of the starters, each
  with its summary. `feature-compose show <starter>` opens the TUI at the view of that starter.
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
| `j`, `k`, or the arrow keys | Move down and up in a list. |
| `l` or Enter | Open the view of the selected starter, or of the selected feature in the view of a starter. |
| `h` or Esc | Go back one view. At the list of the starters, Esc clears the filter. |
| `/` | Start a filter on the names of the starters. While you type the filter, each key goes to the filter. Enter ends the typing and keeps the filtered list. |
| `q` | Stop the TUI. While you type the filter, `q` goes to the filter, and ctrl+c stops the TUI. |

The view of a starter shows its summary, its folders with their commands, its options with their defaults and
their values or patterns, its extension points with their item fields and their declarers, and the features that
target it. The extension points include those that a feature declares, for example `cli.commands` of `cli`.
The view of a feature shows its summary, its needs, its options, and the extension points that it declares or
adds items to.

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
