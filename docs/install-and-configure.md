# Install feature-compose and Point It at a Catalog

feature-compose is one binary. It reads the starters and the features from a catalog folder, which a config
file names.

## Install

The installer downloads the latest release for your platform, checks its checksum, and puts the binary in an
install folder. It needs `sh`, `curl`, `jq`, `tar` and `gzip`.

```shell
sh <(curl -fsSL https://raw.githubusercontent.com/hpcsc/feature-compose/main/scripts/install.sh)
```

It asks for the release channel and the install folder. To give both on the command line:

```shell
sh <(curl -fsSL https://raw.githubusercontent.com/hpcsc/feature-compose/main/scripts/install.sh) --channel release --dir ~/.local/bin
```

| Flag | Meaning |
| --- | --- |
| `--channel release` | The latest stable release. |
| `--channel prerelease` | The latest build of `main`. |
| `--dir <folder>` | The install folder. With no `--dir`, the installer uses `$INSTALL_DIR`, or `~/.local/bin` when `INSTALL_DIR` is not set. |

When the channel has more than one version, the installer lists them, and you select one. `GITHUB_TOKEN` or
`GH_TOKEN` gives access to a private repository.

## The Config File

`new`, `show` and `check` read the catalog folder that the config file names. The config file is
`~/.config/feature-compose/config.yml`. When `XDG_CONFIG_HOME` is set, it is
`$XDG_CONFIG_HOME/feature-compose/config.yml`.

```yaml
catalog: ~/code/feature-compose/catalog
```

The file has one field:

| Field | Meaning |
| --- | --- |
| `catalog` | The catalog folder. A path that starts with `~/` is in your home folder. Another relative path is relative to the folder of the config file. |

Without the config file, `new`, `show` and `check` stop, and the message names the path of the file. A config
file with no `catalog`, or a `catalog` that is not a folder, also stops them, and the message says why.
`version` and `update` do not read the config file. `check --catalog <folder>` checks a catalog with no config
file.

The catalog of this repo is in `catalog/`. A team can keep its own catalog in its own repo, and its members
point the config file at their copy. [Write the Starters and the Features of a Catalog](catalog.md) tells how
a catalog works.

## Update feature-compose

```shell
feature-compose version                # the tag of a release or a prerelease, or the commit of another build
feature-compose update                 # install the latest release
feature-compose update --prerelease    # install the latest prerelease, a build of main
feature-compose update --check         # only tell whether this build is the latest
```

`update` downloads the archive for your platform from a GitHub release, checks it against the
`checksums.txt` of that release, and replaces the binary. It names each step on stderr. In a terminal, it
also shows the progress of the download. `GITHUB_TOKEN` or `GH_TOKEN` gives access to a private repository.

Releases and prereleases are two channels. Each command installs the latest build of its channel when this
build is a different one. So `update` on a prerelease goes back to the latest release. A build from a commit
is not a release or a prerelease, so `update` does not replace it unless you add `--force`.

The installer and `update` do the same job. On a new machine, use the installer. After that, `update` keeps
the binary current.
