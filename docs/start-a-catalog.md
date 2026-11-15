# Start a Catalog and Write Its First Feature, Step by Step

This guide makes a catalog with `init`, adds the feature `changelog` with `add`, and tests the feature with
`check` and `check --build`. Each step shows the command and its output.
[Write the Starters and the Features of a Catalog](catalog.md) is the reference for each field that the steps use.

You need feature-compose and `sh`. [Install feature-compose](install-and-configure.md#install) tells how to install
it. The catalog of `init` uses only `sh` scripts, so it needs no other tool.

## 1. Make the Catalog

`init` writes a small catalog that works: the starter `example` and the feature `greeting`.

```shell
feature-compose init my-catalog
```

```text
Made a catalog in my-catalog with the starter example and the feature greeting. Try it:
  feature-compose show --catalog my-catalog
  feature-compose new --catalog my-catalog example my-repo --feature greeting --dry-run
  feature-compose check --catalog my-catalog --build
To use it with no --catalog, write this line in /home/me/.config/feature-compose/config.yml:
  catalog: /home/me/my-catalog
```

[Start a Catalog](commands.md#start-a-catalog) lists each file of the catalog. Two facts about the starter
`example` are important for the next steps:

- The starter has one folder, `app`, with the path `.`. A feature puts its files for this folder under
  `files/app/`.
- The `test` command of the folder is `sh ./scripts/test.sh`. This script runs each script in `tests/`. So a
  feature adds a test when it writes a script to `tests/`.

## 2. Preview a Repo

`--catalog` gives `new` and `show` the catalog folder, so you do not change the config file yet. `--dry-run`
prints the repo and writes nothing:

```shell
feature-compose new --catalog my-catalog example my-repo --feature greeting --dry-run
```

```text
Repo: my-repo
Starter: example

Features:
  greeting: command line

Options:
  starter.owner: nobody (default)
  greeting.language: en (default)

Files:
  .feature-compose.yml: feature-compose
  README.md: the starter
  scripts/greet: the feature greeting
  scripts/test.sh: the starter
  tests/greeting.sh: the feature greeting
  tests/readme.sh: the starter
```

`feature-compose show --catalog my-catalog` opens a tree of the starter, its features and its extension points.

## 3. Add the Feature

`add feature` writes a `feature.yml` to fill in. `--target` names the starter that the feature works with.

```shell
feature-compose add feature changelog --target example --catalog my-catalog
```

```text
Added the feature changelog to the catalog in my-catalog:
  features/changelog/feature.yml
  features/changelog/files/app/docs/changelog.md
Fill in feature.yml, and then try the feature:
  feature-compose new --catalog my-catalog example my-repo --feature changelog --dry-run
  feature-compose check --catalog my-catalog --build
```

The comments in `features/changelog/feature.yml` show the fields to fill in. They also list the extension points
of the starter `example`:

```yaml
# The extension points of the starter example, with the fields of an item:
#   readme.sections: [text]

# add_to:
#   readme.sections:
#     - text: <value>
```

## 4. Write the Feature

The feature `changelog` gives the repo a `CHANGELOG.md`. The option `format` selects the text of the file, and
the feature adds a section to `README.md`. These are the files of the feature:

```text
features/changelog/
  feature.yml
  files/
    app/                    files for the folder "app" of the starter
      CHANGELOG.md
      tests/changelog.sh
```

Remove `files/app/docs/changelog.md`, which `add` wrote as an example. Then write the three files.

**`feature.yml`** declares the option `format` with two values, and adds one item to the extension point
`readme.sections`:

```yaml
summary: A CHANGELOG.md that lists the changes of each release.
targets: [example]

options:
  format:
    values: [keep-a-changelog, plain]
    default: keep-a-changelog

files:
  - path: files/

add_to:
  readme.sections:
    - text: |-
        ## Changes

        CHANGELOG.md lists the changes of each release.
```

**`files/app/CHANGELOG.md`** is a template. `{{ .Name }}` is the name of the repo, and `option "format"` reads the
option of the feature. This first version has a mistake, `fromat`, which step 5 finds:

```text
# Changes of {{ .Name }}
{{- if eq (option "fromat") "keep-a-changelog" }}

The format of this file is from https://keepachangelog.com.

## [Unreleased]
{{- end }}
```

**`files/app/tests/changelog.sh`** is the test of the feature. The test script of the starter runs it in each repo
that has the feature:

```sh
#!/bin/sh
if ! grep -q '^# Changes of {{ .Name }}$' CHANGELOG.md; then
	echo 'tests/changelog.sh: CHANGELOG.md has no heading "# Changes of {{ .Name }}"'
	exit 1
fi
```

## 5. Find the Mistakes Before a Repo Exists

`check` reads each file of the catalog and makes no repo. It finds the option `fromat`, which the feature does not
declare:

```shell
feature-compose check --catalog my-catalog
```

```text
the feature changelog: the template app/CHANGELOG.md of the files entry files/ reads "fromat", which the feature cannot read: it can read format, starter.owner
the catalog has 1 mistake
```

Change `fromat` to `format`. Then `check` prints `The catalog has no mistakes.`

## 6. Make and Test Each Repo

`check --build` makes the repo of each combination of the catalog, and runs its `test` commands:

```shell
feature-compose check --catalog my-catalog --build
```

```text
The catalog has no mistakes.
The catalog has 6 combinations. Make the repo of each, and run its after_render and test commands.
[1/6] ok    example (0s)
[2/6] ok    example --feature changelog:format=keep-a-changelog (0s)
[3/6] ok    example --feature changelog:format=plain (0s)
[4/6] ok    example --feature greeting:language=en (0s)
[5/6] ok    example --feature greeting:language=fr (0s)
[6/6] ok    example --feature changelog:format=keep-a-changelog --feature greeting:language=en (0s)
The repo of each combination passes its commands.
```

The catalog gives the list of the combinations:

- One repo for each value of `format`, so `check --build` tests each value.
- One repo with both features, because `changelog` and `greeting` both add items to `readme.sections`.

For example, a version of `CHANGELOG.md` with the heading `# Changelog` for `keep-a-changelog` passes `check`,
because `check` makes no repo. But the repo of that value fails the test of the feature:

```text
[2/6] FAIL  example --feature changelog:format=keep-a-changelog (0s)
      the test command "sh ./scripts/test.sh" of the folder app failed: exit status 1
      tests/changelog.sh: CHANGELOG.md has no heading "# Changes of my-repo"
```

Each line names the combination as the arguments of `new`. To look at the repo, make it with these arguments:

```shell
feature-compose new --catalog my-catalog example my-repo --feature changelog:format=keep-a-changelog
```

## 7. Use the Catalog

1. Write the line that `init` printed in the config file. Then `new`, `show` and `check` read the catalog with no
   `--catalog`:

   ```yaml
   catalog: /home/me/my-catalog
   ```

2. Make a repo. In a terminal, `new` opens a form, where you select the features and answer the options:

   ```shell
   feature-compose new example my-repo
   ```

3. Push the catalog to GitHub. The workflow `.github/workflows/check.yml` of `init` runs
   `feature-compose check --catalog . --build` on each push.

## Next Steps

- Add a starter for a kind of repo that you make often: `feature-compose add starter <name>`.
  [Add a Starter or a Feature](commands.md#add-a-starter-or-a-feature) tells what it writes.
- Give each value of an option its own files: [Files for One Value of an Option](catalog.md#files-for-one-value-of-an-option).
- Ask an option only for some answers: [An Option That Applies Only Sometimes](catalog.md#an-option-that-applies-only-sometimes).
- Make a feature that needs another feature: [Needs](catalog.md#needs).
- Let features add lines to a file that they do not own: [Extension Points](catalog.md#extension-points).
- Use the template functions, and other delimiters for a file whose own syntax has `{{`:
  [Templates](catalog.md#templates) and [starter.yml](catalog.md#starteryml).
