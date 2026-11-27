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

### Tools for a Starter That Runs a Generator

A starter can run a generator, for example `mix phx.new`, and then change the files that the generator writes.
For such a starter, `new` and `check` need these tools on the `PATH`:

- [ast-grep](https://ast-grep.github.io), version 0.45.3 or later. For example: `brew install ast-grep`, or
  `mise use -g ast-grep`.
- The generator and its toolchain, for example Elixir and the archive `phx_new`.

A starter without a generator needs neither.

## The Catalogs

`new`, `show` and `check` read the catalogs of the config file. `feature-compose catalog add` adds a catalog to
the config file, so you do not write the file by hand:

```shell
feature-compose catalog add hpcsc https://github.com/hpcsc/feature-compose --path catalog
feature-compose catalog add team ~/code/team-catalog
```

The first argument is the name of the catalog. The second argument is its source:

| Source | Example | What the tool does |
| --- | --- | --- |
| A git URL | `https://github.com/hpcsc/feature-compose`, `git@github.com:team/catalog.git`, `file:///srv/git/catalog.git` | Clones the repo into the folder of the clones, and checks out the ref. `catalog sync` gets the newest commit of the ref |
| A local folder | `~/code/team-catalog` | Uses the folder where it is, with no clone. A change in the folder shows at the next `new`, with no `sync` |

- `--path <folder>` names the folder of the catalog in the source, for example `catalog` for this repo. With no
  `--path`, the catalog is at the root of the source. The folder must hold the folder `starters`.
- `--ref <ref>` names a branch, a tag or a commit of a git URL. With no `--ref`, the tool uses the default branch
  of the repo. A branch gets its newest commit at each `catalog sync`. A commit keeps its content, and so does a
  tag, unless the owner of the repo moves the tag to another commit.
- A name starts with a letter or a digit, and holds only letters, digits, `.`, `_` and `-`.

[What Each Command of feature-compose Does](commands.md#manage-the-catalogs) tells how `catalog list`, `sync`,
`set` and `remove` work.

### Names Across Catalogs

A starter has a full name, `<catalog>/<starter>`, for example `hpcsc/phoenix`. A short name, for example
`phoenix`, works while only one catalog has a starter with that name. When two catalogs have it, `new` stops and
lists the full names. A repo takes its features only from the catalog of its starter, so `--feature oban` names
the feature `oban` of that catalog.

`--catalog <name>` gives `new`, `show` and `check` only that catalog of the config file. `--catalog <folder>` gives
them the catalog in that folder, with no config file. The starters of a catalog from a folder keep their short
names.

### Private Repos

git reads a private repo with the credentials that it already has: an SSH key, the SSH agent, or a credential
helper for HTTPS. If `git clone <source>` works on the machine with no question on the terminal, `catalog add` works too. The tool runs git and ssh
with no terminal, so they cannot ask for a password, a passphrase or a host key on the terminal. A program that
`GIT_ASKPASS` or `SSH_ASKPASS` names can still ask, for example in the terminal of an editor. When git has no
credential for the source, `catalog add` and `catalog sync` stop at once, and the message tells you to check that
`git clone <source>` works. A password or a token in the URL of a source does not go into the messages, the output of
`catalog list` or `.feature-compose.yml`.

### Trust

A catalog runs its commands on your machine: the `generate` command and the `after_render` commands of its
starters. So add only a catalog whose owner you trust. The preview of `new --dry-run` lists each command that
`new` runs, in the section `Commands`. `new` never gets new content of a catalog. Only `catalog sync` does.

### The Files

The config file is `~/.config/feature-compose/config.yml`. When `XDG_CONFIG_HOME` is set, it is
`$XDG_CONFIG_HOME/feature-compose/config.yml`:

```yaml
catalogs:
  hpcsc:
    source: https://github.com/hpcsc/feature-compose
    path: catalog
  team:
    source: /home/me/code/team-catalog
```

| Field | Meaning |
| --- | --- |
| `catalogs.<name>.source` | The git URL or the local folder of the catalog. A folder that starts with `~/` is in your home folder. Another relative folder is relative to the folder of the config file. `catalog add` writes the full path of a local folder |
| `catalogs.<name>.path` | The folder of the catalog in the source. With no `path`, the root of the source |
| `catalogs.<name>.ref` | The branch, the tag or the commit of a git URL. With no `ref`, the default branch |

The clones are in `~/.local/share/feature-compose/catalogs/<name>`. When `XDG_DATA_HOME` is set, they are in
`$XDG_DATA_HOME/feature-compose/catalogs/<name>`. You can delete a clone. Then `new` stops and names
`catalog sync`, and `catalog sync` clones the source again.

Without a catalog in the config file, `new`, `show` and `check` stop, and the message names `catalog add`. A
config file with the key `catalog` also stops them, and `catalog add` too. The message tells you to remove the key,
and then to run `catalog add`. `version` and `update`
do not read the config file.

A team can keep its own catalog in its own repo, and each member adds it with `catalog add`.
[Write the Starters and the Features of a Catalog](catalog.md) tells how a catalog works.

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
