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

`task test:combinations` runs `feature-compose check --catalog catalog --build`. It makes one repo for each
combination of the catalog in `catalog/`, and runs the `after_render` and `test` commands of each repo.
[Make the Repo of Each Combination](commands.md#make-the-repo-of-each-combination) lists the combinations and
tells how to read the output. Run the task after you change a starter or a feature.

The task sets `MISE_TRUSTED_CONFIG_PATHS` to the temporary folder, because mise refuses the `mise.toml` of a repo
in a folder that it does not trust. The task needs Go, Task, a C compiler and the network.

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
