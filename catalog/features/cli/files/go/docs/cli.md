# The {{ kebab .Name }} CLI

## Install

The installer downloads the latest release for your platform, verifies its checksum, and puts the
binary where you can start using it. It needs `sh`, `curl`, `jq`, `tar`, and `gzip`.

```shell
sh <(curl -fsSL https://raw.githubusercontent.com/{{ githubRepository (option "starter.module") }}/main/scripts/install.sh)
```

It asks for the release channel and the install directory. To skip the questions on the command line:

```shell
sh <(curl ...) --channel release --dir ~/.local/bin
```

The channel is `release` (the latest stable release) or `prerelease` (the latest build of `main`).
Without `--dir`, the binary lands in `~/.local/bin` (or `$INSTALL_DIR`). When more than one version is
available in the channel, the installer lists them for you to pick.

`GITHUB_TOKEN` or `GH_TOKEN` gives access to a private repository.

`{{ kebab .Name }} update` also installs from a release, and it replaces this very binary, so the
installer and the update command do the same kind of job; use whichever you find convenient on a fresh
machine.

## Build

```shell
task build             # build ./bin/{{ kebab .Name }}
task cli -- --help     # run the CLI from the source, with the arguments after --
```

## Version and update

```shell
{{ kebab .Name }} version                # the tag of a release or a prerelease, or the commit of any other build
{{ kebab .Name }} update                 # install the latest release
{{ kebab .Name }} update --prerelease    # install the latest prerelease, a build of main
{{ kebab .Name }} update --check         # only tell you whether this build is the latest
```

`internal/cli/version` reports the version. A release build gets its tag from goreleaser, through
`-ldflags -X .../internal/cli/version.releaseTag`. Any other build reports the short commit sha that Go
keeps in the build information, with `-dirty` after it when the working tree had changes.

`{{ kebab .Name }} update` downloads the archive for your platform from a GitHub release, checks it
against the `checksums.txt` of that release, and then replaces the binary. It names each step on
stderr, and in a terminal it shows how much of the download has arrived. `GITHUB_TOKEN` or `GH_TOKEN`
gives access to a private repository.

Releases and prereleases are two channels. Each command installs the latest build of its channel when
this build is a different one, so `{{ kebab .Name }} update` on a prerelease goes back to the latest
release. A build from a commit is not a release or a prerelease, so `{{ kebab .Name }} update` does not
replace it unless you add `--force`.

## Goreleaser

GoReleaser builds the releases from `.goreleaser.yaml`.

- Make a snapshot build under `./dist`: `task release:local`
- Publish a release: tag the commit, then run GoReleaser with a `GITHUB_TOKEN` that can write releases.

```shell
git tag vX.X.X
git push origin vX.X.X
GITHUB_TOKEN=... goreleaser release --clean
```

This repo has no workflow that publishes releases or prereleases.

## E2E Test

The end-to-end tests run the built binary in a terminal, send keys to it, and read the screen. They are
in `e2e/`, and [e2e-tests.md](e2e-tests.md) tells how they work.

In Docker:

```shell
docker build -f e2e/Dockerfile --build-arg BUILD_TAG=v0.1.0 -t {{ kebab .Name }}-e2e .
docker container run --rm {{ kebab .Name }}-e2e
```

On this machine, with node:

```shell
CGO_ENABLED=0 go build -ldflags "-X {{ option "starter.module" }}/internal/cli/version.releaseTag=v0.1.0" -o ./bin/{{ kebab .Name }}-e2e ./cmd/{{ kebab .Name }}
cd e2e && npm install --no-audit --no-fund && EXECUTABLE=../bin/{{ kebab .Name }}-e2e BUILD_TAG=v0.1.0 npx vitest run
```
