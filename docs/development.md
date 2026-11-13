# Build, Test and Release feature-compose

This document is for the people who change feature-compose or the catalog of this repo. The tasks need
[Task](https://taskfile.dev). `mise.toml` gives the versions of the tools.

## Build and Run

```shell
task build            # build ./bin/feature-compose
task run -- --help    # run the CLI from the source
```

`internal/version` gives the version that `feature-compose version` prints. A release build gets its tag from
GoReleaser, through `-ldflags -X .../internal/version.releaseTag`. Each other build gives the short commit that
Go keeps in the build information, with `-dirty` after it when the working tree had changes.

## Test

| Task | What it runs |
| --- | --- |
| `task test:unit` | The unit tests of the tool. |
| `task test:catalog` | `feature-compose check --catalog catalog` on the catalog of this repo. |
| `task test:combinations` | One repo for each combination of the catalog, with its commands. |
| `task test:e2e` | The end-to-end tests in Docker, as CI runs them. |
| `task test:e2e:local` | The end-to-end tests on this machine. They need Node. |

CI runs each of them on each push.

### Test the Combinations of the Catalog

`task test:combinations` makes one repo for each combination of the catalog in `catalog/`. In each repo, it
runs the `after_render` commands and then the `test` commands of each folder. These are the combinations:

- Each starter alone
- Each starter with each feature that targets it, and the needs of the feature
- One for each value of an option that has `values`, with the other options at their defaults
- One for each value of an option that has `values` of a feature that a feature needs, with both features. The
  option of the need has that value, and each other option of the need has the value that the needs of the feature
  and of its own needs set, or its default. A value that these needs set gives no such combination, and a need of a
  need gives none. When two of these needs set two values of one option, the feature gets no such combination.
- Each pair of features that add items to one extension point, except a pair in which one feature needs the
  other: the repo of that feature already holds both

Run it after you change a starter or a feature. The catalog gives the list, so a new starter, feature or value
needs no change to the task or to CI.

The output names each combination that fails as the arguments of `new` that make the same repo, for example
`go-event-sourcing-service --feature api-auth:method=api-key`, and shows the output of the command that failed.

The task needs Go, Task, gotestsum, a C compiler and the network. Each repo takes some seconds when the Go
module cache has its modules, and some minutes when it does not.

### The End-to-End Tests

The end-to-end tests run the built binary in a terminal, send keys to it, and read the screen. They are in
`e2e/`, and [How the end-to-end tests work](e2e-tests.md) tells more.

## Release

To run GoReleaser on this machine, run `task release:local`. It makes a snapshot build in `./dist`.

To make a release, push a tag:

```shell
git tag vX.X.X
git push origin vX.X.X
```

The tag starts the release workflow. It runs the CI checks, and then makes a GitHub release with the binaries
for macOS and Linux.

Each push to `main` starts the prerelease workflow. It tags the commit with the next patch after the latest
release, the run number and the commit, for example `v0.2.1-42.g4829f92`. It publishes that tag as a
prerelease, and then keeps only the 5 newest prereleases.

The workflow `On Demand Build` builds a snapshot of any ref from GitHub Actions, and uploads the archives as
artifacts.
