# What Each Command of feature-compose Does

| Command | What it does |
| --- | --- |
| `feature-compose new [<starter> [<name>]]` | Makes a new repo from a starter and the features that you select. |
| `feature-compose show [<starter>]` | Shows the starters, the features and the extension points of the catalog. |
| `feature-compose check [--build]` | Finds the mistakes of the starters, the features and the templates of the catalog. With `--build`, it also makes the repo of each combination and runs its commands. |
| `feature-compose init <folder>` | Writes a small catalog with one starter and one feature, as the start of a catalog of your own. |
| `feature-compose add starter <name>` | Adds a starter to the catalog, with a `starter.yml` to fill in. |
| `feature-compose add feature <name> --target <starter>` | Adds a feature to the catalog, with a `feature.yml` to fill in. |
| `feature-compose version` | Prints the tag that the binary comes from, or its commit. |
| `feature-compose update` | Replaces the binary with the latest release or prerelease. |

`new`, `show` and `check` read the catalog that the config file names, or the catalog folder that `--catalog`
gives.
[Install feature-compose and Point It at a Catalog](install-and-configure.md) tells how to write the config
file, and how `version` and `update` work.

## Make a Repo

```shell
feature-compose new <starter> <name> [--feature <name>[:<option>=<value>,...]]... [--option <name>=<value>]...
```

| Flag | Meaning |
| --- | --- |
| `--feature <name>` | Selects a feature. Give the flag one time for each feature. `--feature api-auth:method=oauth` also sets an option of the feature. |
| `--option <name>=<value>` | Sets an option of the starter. Give the flag one time for each option. |
| `--output-dir <folder>` | The folder of the new repo. The default is `./<name>`. |
| `--no-prompt` | Opens no form. Each option without a value gets its default. |
| `--dry-run` | Prints a preview of the repo, writes no file and runs no command. |
| `--catalog <folder>` | Makes the repo from the catalog in that folder. `new` then does not read the config file. Use it to try a catalog that you write, for example `new --catalog . my-starter my-test --dry-run` in the folder of the catalog. |

`new` writes each file of the starter and of the selected features. Then it runs the `after_render` commands of
each folder of the starter, for example `go mod tidy`. Last, it writes the selection file `.feature-compose.yml` to
the root of the repo. The output folder must be empty or absent.

### The Selection File

`.feature-compose.yml` keeps a record of the choices that made the repo. This is the file of
`feature-compose new go-event-sourcing-service my-service --feature cli-auth`:

```yaml
version: v0.1.0
starter: go-event-sourcing-service
name: my-service
options:
  go-version: 1.27.1
  module: github.com/hpcsc/my-service
features:
  api-auth:
    method: api-key
  cli: {}
  cli-auth:
    store-key: "false"
requested: [cli-auth]
```

| Key | Value |
| --- | --- |
| `version` | The text that `feature-compose version` prints. The catalog is a folder with no version, so this is not a version of the catalog. |
| `starter`, `name` | The starter and the name of the repo. |
| `options` | Each option of the starter, with the value that the templates read. The file also holds a value that came from a default. |
| `features` | Each feature of the repo, with each of its options and the value that the templates read. The file also holds a feature that only a need added, and a value that came from a default or from a need. |
| `requested` | Each feature that you named with `--feature` or selected in the form, also when another feature needs it. A feature that only a need added is in `features` and not in `requested`. |

The options, the features and the requested features are in the order of their names, so one selection gives the
same file in each order of the flags.

## Preview a Repo

`feature-compose new <starter> <name> --dry-run` prints a preview of the repo that `new` makes. It writes no file
and runs no command, so the output folder does not exist after the run:

```shell
feature-compose new go-event-sourcing-service my-service --option go-version=1.28 --feature cli-auth --dry-run
```

```text
Repo: my-service
Starter: go-event-sourcing-service

Features:
  api-auth: needed by cli-auth
  cli: needed by cli-auth
  cli-auth: command line

Options:
  starter.go-version: 1.28 (command line)
  starter.module: github.com/hpcsc/my-service (default)
  api-auth.method: api-key (default)
  cli-auth.store-key: false (default)

Files:
  .dockerignore: the feature cli
  .env.integration: the starter
  .env.local: the starter
  .feature-compose.yml: feature-compose
  ...
  cmd/my-service/main.go: the feature cli
  ...
```

The preview gives the name of the repo and its starter. Then it gives each feature of the repo, after the features
that it needs, with its source:

| Source | Meaning |
| --- | --- |
| `command line` | A `--feature` flag selects the feature. When other selected features also need it, the line ends with `, needed by <features>`. |
| `answered` | You select the feature in the form (see [Answer the Questions in a TUI](#answer-the-questions-in-a-tui)). When other selected features also need it, the line ends with `, needed by <features>`. `--dry-run` opens no form, so it never shows this source. |
| `needed by <features>` | Only a need selects the feature. The line is the line that `new` prints for it. |

Then the preview gives each option by the name with which a template reads it: the options of the starter as
`starter.<option>`, and then the options of each feature as `<feature>.<option>`, with the features in the order
above. Each option shows the value that the templates read, and its source:

| Source | Meaning |
| --- | --- |
| `command line` | `--option <name>=<value>` or `--feature <name>:<option>=<value>` gives the value, also when it is the same as the default. When a need also sets the value, the line ends with `, need of <features>`. |
| `need of <features>` | The `needs` of the features that select this feature set the value, for example a feature `client` with `needs: {auth: {method: api-key}}` sets the `method` of `auth` to `api-key`. A need comes after a flag and before an answer. |
| `answered` | You type or choose the value in the form, also when it is the default. `--dry-run` opens no form, so it never shows this source. |
| `default` | No flag gives the value, so the option has its default. A default that holds `{{ .Name }}` shows with the name of the repo. An option whose `when` is false has its default, also when you gave it another value in the form. |

Last, the preview gives each file of the repo by its path, in the order of the paths, with its owner: the starter,
the feature whose template writes the file, or `feature-compose` for the selection file `.feature-compose.yml`.

Before it prints, `--dry-run` does the same checks as `new`. A selection that stops `new`, for example an unknown
feature, a feature that does not target the starter, a value that does not match its pattern, a flag that gives an
option another value than a need sets, or a flag that gives another value than its default to an option whose `when`
is false, stops `--dry-run` with the same message, and `--dry-run` then prints nothing. For example,
`--feature cli-auth:store-key=true --feature api-auth:method=oauth` stops with `the option store-key of the feature
cli-auth applies only when api-auth.method == "api-key", and api-auth.method is "oauth", but the command line gives it
"true"`.
The message of a feature that does not target the starter names the feature and the starter, and lists the features
that target the starter, for example `the feature api-auth does not target the starter go: the features that target
the starter go are cli`. The message of a flag against a need
names both sources, for example `"oauth" from the command line and "api-key" from the need of client`.
`--dry-run` does not read the output folder, so an output folder that has files stops `new` but not `--dry-run`.

The preview is text for a person, and its layout can change. A script reads the catalog with
`feature-compose show --output json`.

## Answer the Questions in a TUI

When stdin and stdout are both terminals, `feature-compose new` opens a form. The form has the look of the TUI of
`show` (see [Show the Catalog](#show-the-catalog)): each section of the form is a panel with its title in the top
border. The panels come from top to bottom: the repo, the options of the starter, the features, and the options of
each selected feature. Under the panels is the button that writes the repo. You can change any answer in any order.
This is `feature-compose new go-event-sourcing-service my-service --feature cli-auth`, with the focus on the features:

```text
 feature-compose new                                                                     3 features
╭─ Repo ───────────────────────────────────────────────────────────────────────────────────────────╮
│   name     my-service  from the command line                                                     │
│   starter  go-event-sourcing-service  from the command line                                      │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─ Options of go-event-sourcing-service ───────────────────────────────────────────────────────────╮
│   go-version  1.27.1                                                                             │
│   module      github.com/hpcsc/my-service                                                        │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─ Features ────────────────────────────────────────────────────────────────────────────── 3 of 5 ─╮
│ ▌ [x] api-auth         needed by cli-auth                                                        │
│                        The API accepts a request to a protected route only when it carries a     │
│                        credential that the API accepts, an API key or an OAuth access token.     │
│   [x] cli              needed by cli-auth                                                        │
│   [x] cli-auth         from the command line                                                     │
│   [ ] process-manager  The example of a process manager, a payment to another bank, with fakes … │
│   [ ] push             The API sends each new event of the service to its WebSocket clients.     │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─ Options of api-auth ────────────────────────────────────────────────────────────────────────────╮
│   method  api-key                                                                                │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
╭─ Options of cli-auth ────────────────────────────────────────────────────────────────────────────╮
│   store-key  false                                                                               │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
                                                                                   Write the repo

 j/k move • space select • tab next • ctrl+c quit
```

The form has these panels:

| Panel | What it holds |
| --- | --- |
| `Repo` | The name of the repo, as a text, and the starter. |
| `Options of <starter>` | Each option of the starter, in the order of the names. |
| `Features` | Each feature that targets the starter, each after the features that it needs. The count at the right of the top border, for example `3 of 5`, counts the selected features. `[x]` marks a selected feature, and `[ ]` marks a feature that is not selected. |
| `Options of <feature>` | Each option of a selected feature. |

An option is a text, or a list of its values when the option has `values`. When the command line gives no starter,
the starter is a list of the starters of the catalog, each with its summary.

The text after a field tells you more about it:

| Text | Meaning |
| --- | --- |
| `from the command line` | An argument or a flag gives the value, so the form does not change it. |
| `from the need of <features>` | The need of a selected feature sets the value, so the form does not change it. |
| `needed by <features>` | Selected features need the feature, so it stays selected. |
| `missing` | The field has no value yet. This text is red, and the other texts are faint. |
| A summary | The summary of a starter or a feature that has no other text. |

The panel with the focus has a border in the accent color. In it, the field with the focus has the bar `▌` at its
left. A list shows the bar and a colored background on the row under its cursor. Under the field with the focus, the
form shows its default and its pattern, or the whole summary and the needs of the feature under the cursor. A row
wider than its panel ends with `…`. When the form is taller than the terminal, it scrolls to show the cursor. The
footer shows the keys of the field with the focus.

**A list.** A list of values, such as the starters or the values of an option, shows only the chosen value until it
gets the focus. With the focus, it opens: `●` marks the chosen value, `○` marks the other values, and `›` marks the
cursor. The cursor starts at the chosen value. `j` and `k` move the cursor, and Enter chooses the value under the
cursor and moves the focus to the next field. Space chooses the value under the cursor and keeps the list open. A
move of the cursor alone chooses nothing.

The keys of the form:

| Key | Action |
| --- | --- |
| Tab | Move the focus to the next field. The focus skips a value that the command line or a need sets. |
| Shift+Tab | Move the focus to the field before. |
| `j` and `k` | Move the cursor in a list or in the features down and up. The cursor stops at the ends. |
| Down and up arrows | Do what `j` and `k` do. At the ends of a list, and at a field that is not a list, move the focus to the next field or to the field before. |
| Enter | In a list of values, choose the value under the cursor. Then move the focus to the next field. At the button, show the preview. |
| Space | In a list of values, choose the value under the cursor. In the features, select the feature under the cursor, or clear it. At the button, show the preview. |
| Each other key at a text | Edit the text. `j`, `k` and Space also go into the text. The left and right arrows move the cursor of the text. |
| ctrl+c | Stop the form and write nothing. |

**The features and their needs.** Space at a feature selects it, and the form also selects each feature that it
needs, with the text `needed by <features>`. Space at a needed feature changes nothing, and a message under the
feature names the features that need it. Space at a feature that you selected clears it, and also clears each feature
that only its needs selected. A feature that a `--feature` flag gives stays selected, and you can select more features
next to it. When the selection stops, for example because two features declare one extension point, the feature stays
as it was, and a message under it says why. The selection also stops when the need of a feature sets another value
than a flag, for example a Space at a feature `client` whose need sets `auth.method` to `api-key`, with
`--feature auth:method=oauth`. The message names both sources.

**An option with a `when`.** The form shows an option only while its `when` is true, and it changes after each
answer. For example, the answer `oauth` for `auth.method` hides an option whose `when` is
`auth.method == "api-key"`. A value that you gave the option comes back when its `when` is true again. An answer
that makes the `when` false while a flag gives the option another value than its default stops the selection: the
answer stays as it was, and a message under the option says why.

**A text.** The form takes the text at each key while the text passes its check. A text that does not pass stays in
the field, and a message under the field says why, for example that the text does not match the `pattern` of the
option. The form then keeps the last value that passed. The message stays when the focus moves to another field, and
goes away when the text passes.

A value that you type or choose has the source `answered`, also when it is the default. Until you type or choose a
value for an option, the option has its default, and a default that reads the name, for example
`github.com/hpcsc/{{ .Name }}`, follows the name. A clear of a feature drops the values that you gave its options.
While the need of a selected feature sets an option, the option has the value of the need, and a value that you gave
the option comes back when no selected need sets it.

**The starter.** A change of the starter keeps each feature that you selected and that targets the new starter, with
the values that you gave its options. It clears the other features, and the values that you gave the options of the
old starter. A starter that the flags do not fit, or whose features have needs that form a cycle, is not chosen, and a
message under the list says why.

**Write the repo.** Enter at the button shows the preview in a panel, which is the text that `--dry-run` prints, and
asks `Write the repo?`. Enter writes the repo, and Esc goes back to the form. The arrow keys, `j` and `k`, Page Up and
Page Down, and ctrl+u and ctrl+d scroll the preview, and the bottom border shows how far it is scrolled. When an
answer is missing (the name, the starter, or the value of an option with no default), or a text does not pass its
check, the button moves the focus to the first such field, and a message under the field says what is wrong.

An unknown starter, a flag that the starter does not know, and a flag that gives an option another value than a need,
stop `new` before the form opens, with the message that `new` gives without the form.

`new` writes nothing until you press Enter at the preview. A cancel writes no file, makes no output folder and runs no
command. `new` then prints `you canceled new, so it wrote no file` and ends with the status 1.

After `new` writes the repo, it prints the command that makes the same repo:

```text
Made my-service from the starter go-event-sourcing-service.
To make the same repo again, run:
  feature-compose new go-event-sourcing-service my-service --feature cli-auth
```

The command names each value that is not its default, and each feature that you selected. It puts each word that a
shell splits, expands or matches against files in single quotes. That command with `--no-prompt` makes the same
files. When you give a value to an option of a feature that only a need added, the command names that feature with
`--feature`. So the selection file of the repo that the command makes also lists that feature in `requested`.

`new` opens no form when stdin or stdout is not a terminal, or with `--no-prompt` or `--dry-run`. CI uses this mode.
`new` then needs both arguments, `<starter>` and `<name>`.
Each option without a value then gets its default. An option with no default and no value stops `new`, and the
message names the option and the flag that gives it:

```text
the option region of the starter service has no default and no value: give it with --option region=<value>
```

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
`--catalog <folder>` shows the catalog in that folder, and `show` then does not read the config file.

```shell
feature-compose show
feature-compose show --catalog ./catalog
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
| A feature | Its summary, its needs, its options with their defaults, their values or patterns, and their `when`, and the extension points that it declares or adds items to. |
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
| `starters.<name>.options.<name>.when` | Always `""`. Only an option of a feature can have a `when`, and a starter option with a `when` stops the read of the starter. |
| `starters.<name>.extension_points.<name>.summary` | The contract of an item of the extension point. |
| `starters.<name>.extension_points.<name>.file` | The file that renders the items. |
| `starters.<name>.extension_points.<name>.item` | The fields of each item. |
| `features.<name>.summary` | One sentence about what the feature adds. |
| `features.<name>.targets` | The starters that the feature works with. |
| `features.<name>.needs.<name>.<name>` | The value that the need sets for an option of the needed feature. Each key of `needs` is a feature that the tool also selects when you select this feature, and a need that sets no value is `{}`. |
| `features.<name>.options.<name>.default` | The default of the option, as the file writes it. |
| `features.<name>.options.<name>.pattern` | The regular expression that a value must match, or `""`. |
| `features.<name>.options.<name>.values` | The values that the option accepts, or `[]` for any value. |
| `features.<name>.options.<name>.when` | The expression that decides when the option applies, as the file writes it, or `""` for an option that always applies. |
| `features.<name>.extension_points.<name>.summary` | The contract of an item of an extension point that the feature declares. |
| `features.<name>.extension_points.<name>.file` | The file of the feature that renders the items. |
| `features.<name>.extension_points.<name>.item` | The fields of each item. |
| `features.<name>.add_to.<name>` | The items that the feature adds to the extension point, each with the fields of that extension point. |

## Start a Catalog

`feature-compose init <folder>` writes a small catalog that works, as the start of a catalog of your own. The
folder can hold other files, for example `README.md` and `.git`. When it already holds a file of the catalog,
`init` stops, names the file, and writes no file.

```shell
feature-compose init my-catalog
```

The catalog has these files:

| Path | What it holds |
| --- | --- |
| `starters/example/starter.yml` | The starter `example`, with the option `owner`, the folder `app`, and the extension point `readme.sections`. |
| `starters/example/files/app/` | `README.md`, which writes the items of `readme.sections`, and `scripts/test.sh`, which runs each script in `tests/`. |
| `features/greeting/feature.yml` | The feature `greeting`, with the option `language` (`en` or `fr`) and an item of `readme.sections`. |
| `features/greeting/files/app/` | `scripts/greet`, which prints the greeting, and its test `tests/greeting.sh`. |
| `.github/workflows/check.yml` | A GitHub workflow that installs the latest release of feature-compose and runs `check --catalog . --build` on each push. |

Then `init` prints the commands that try the catalog, and the line of the config file that makes it the catalog
of `new`, `show` and `check`:

```text
Made a catalog in my-catalog with the starter example and the feature greeting. Try it:
  feature-compose show --catalog my-catalog
  feature-compose new --catalog my-catalog example my-repo --feature greeting --dry-run
  feature-compose check --catalog my-catalog --build
To use it with no --catalog, write this line in /home/me/.config/feature-compose/config.yml:
  catalog: /home/me/my-catalog
```

Change the starter and the feature into your own, or keep them as an example next to your own.

## Add a Starter or a Feature

`add` writes the files of a new starter or feature, with comments that show the fields to fill in. `check` finds
no mistake in them, and `check --build` passes on their repos, so you can try each change that you make. `add`
writes to the catalog folder of the config file, or to the folder that `--catalog` gives. When the catalog already
has the starter or the feature, `add` stops and writes no file.

```shell
feature-compose add starter service
feature-compose add feature metrics --target service --target example
```

`add starter <name>` writes these files:

| Path | What it holds |
| --- | --- |
| `starters/<name>/starter.yml` | A summary to change, the folder `app` with the path `.`, and the fields `options`, `after_render`, `test` and `extension_points` as comments. |
| `starters/<name>/files/app/README.md` | A heading with the name of the repo. |

`add feature <name> --target <starter>` writes these files. Give `--target` one time for each starter that the
feature targets. A target that is not a starter of the catalog stops `add`, and the message lists the starters.

| Path | What it holds |
| --- | --- |
| `features/<name>/feature.yml` | A summary to change, the targets, a files entry, and the fields `needs` and `options` as comments. A comment lists the extension points of each target with the fields of an item, and gives an `add_to` example with an extension point that each target declares. |
| `features/<name>/files/<folder>/docs/<name>.md` | A file to change into the files of the feature. `<folder>` is the first folder, in the order of the names, that each target has. When no folder is in each target, `add` writes no file and no files entry, and a comment in `feature.yml` says why. |

The name of a starter or a feature must start with a lowercase letter or a digit, and hold only lowercase letters,
digits and `-`. Then `add` prints the files that it wrote, and the commands of `new --dry-run` and `check --build`
that try the starter or the feature.

## Check the Catalog

`feature-compose check` finds each feature of the catalog that does not fit its targets or its needs, each
template that reads an option that it cannot read or that does not parse, each extension point whose file does not
render its items, and each path that two owners write, so that a mistake in the catalog stops before a developer
runs `new`. It reads the catalog folder that the config file
names, and checks each feature against each starter that its `targets` name. `--catalog <folder>` checks the
catalog in that folder, and `check` then does not read the config file. A `--catalog` path that is not a folder
stops `check`, and the message names the path.

```shell
feature-compose check
feature-compose check --catalog ./catalog
```

`task test:catalog` runs `check --catalog catalog` on the catalog of this repo. CI runs it on each push, in the job
`build`, so a change that puts a mistake in `catalog/` fails CI.

`check` finds these mistakes:

| Mistake | Example of the message |
| --- | --- |
| An `add_to` names an extension point that the starter, the feature and its needs do not declare. A need of a need counts as a need. Another feature of the starter does not count, also when `new` selects it. | `the starter service: the feature tools adds to the extension point "cli.commands", which the starter, the feature and its needs do not declare` |
| An item does not have the item fields of its extension point. The order of the fields does not matter. | `the starter service: the item 2 that the feature auth adds to the extension point "api.middleware" has the fields [code], but the extension point declares the fields [code, import]` |
| A feature adds files to a folder that the starter does not have. `check` reads the files of each files entry, also of an entry whose `when` is false. A folder with many files gives one mistake. | `the starter service: the feature web adds files to the folder web, which the starter does not have` |
| A file is directly in a files entry, so it is in no folder. | `the starter service: the template README.md of the files entry files/ of the feature cli is not in a folder` |
| Two targets of a feature declare an extension point of its `add_to` with different item fields. In a target where a need of the feature declares the extension point, the item fields of the need count. A target that does not declare the extension point is not in this message. | `the feature auth adds to the extension point "api.middleware", which its targets declare with different item fields: [code, import] in the starter service, [code] in the starter worker` |
| A need is not a feature of the catalog. | `the starter service: the feature auth needs vault, which is not a feature of the catalog` |
| A need does not target each target of the feature. | `the starter worker: the feature auth needs cli, which does not target the starter` |
| A need sets a value that its option does not have: a value that is not one of the values of the option or does not match its pattern, or an option that the needed feature does not declare. Each wrong value is one mistake, also in a target that the need does not target. | `the starter service: the need of the feature client: the feature auth: the value "basic" of the option method is not one of its values: api-key, oauth` |
| The needs of the features of a starter form a cycle. The message names each feature of the cycle in the order of the needs, and no feature that only needs a feature of the cycle. Each feature that is in a cycle is in at least one message, also when two cycles share it. A cycle gives one mistake for each starter whose features hold it. | `the starter service: the needs of the features form a cycle: app needs base needs app` |
| The `when` of a files entry does not parse. Each such files entry is one mistake of the feature, also when the feature has two targets, and `check` still finds the other mistakes of the feature. `new` stops on the first one. | `the feature auth: the when "method ==" of the files entry "files/": method == needs a quoted value after it` |
| The `when` of an option does not parse. Each such option is one mistake of the feature, also when the feature has two targets. `new` stops on the first one. | `the feature auth: the when "method ==" of the option "store-key": method == needs a quoted value after it` |
| The `when` of an item of `add_to` does not parse. Each such item is one mistake of the feature, also when the feature has two targets, after the `when` of its options, in the order of the extension points and the items. `new` stops on the first one. | `the feature auth: the when "method ==" of the item 2 of the extension point "api.middleware": method == needs a quoted value after it` |
| A `when` reads an option that the feature cannot read. A feature reads its own options by name, the options of each feature in its `needs` as `<need>.<option>`, and `starter.<option>` for an option that each of its targets declares. A need of a need does not count. Each reference that a `when` cannot read is one mistake. For a starter option, the message names each target that does not declare it. | `the feature cli: the when "starter.region == \"eu\"" of the files entry "files/" reads "starter.region", which the starter worker does not declare` |
| The `when` of an option reads an option that it cannot read. The `when` of an option reads the options of each feature in `needs` as `<need>.<option>`, `starter.<option>` for an option that each target declares, and the options of its own feature that have no `when`, by name. It cannot read an option of its own feature that has a `when`, also itself. Each reference is one mistake, after the reads of the `when` of the files entries, in the order of the option names. | `the feature cli-auth: the when "api-auth.port == \"80\"" of the option "store-key" reads "api-auth.port", which the when of an option cannot read: it can read api-auth.method, starter.go-version, starter.module` |
| The `when` of an item of `add_to` reads an option that the feature cannot read, by the rule of the `when` of a files entry. Each reference is one mistake, after the reads of the `when` of the options, in the order of the extension points and the items. | `the feature auth: the when "vault.mode == \"fast\"" of the item 1 of the extension point "api.middleware" reads "vault.mode", which the feature cannot read: it can read method` |
| A template of a feature calls `option` with an option that the feature cannot read, by the rule of the row of the `when` of a files entry. `check` reads the parse tree of the path of each template, with `{{ }}`, and of its contents, with the delimiters of the feature, so it also checks a template whose `when` is false in each repo of CI. Only a call with a quoted option counts, also as a parenthesized argument or in a `define`. Each option is one mistake for each template. | `the feature cli: the template go/cli.go of the files entry files/ reads "starter.region", which the starter worker does not declare` |
| A template of the starter calls `option` with an option other than `starter.<option>` of an option that the starter declares. | `the starter service: the template go/main.go reads "cli.mode", which the starter cannot read: it can read starter.go-version, starter.module` |
| A template does not parse with its delimiters and the template functions of `new`. The message gives the error of the parse. | `the feature cli: the template go/cli.go of the files entry files/ does not parse: template: go/cli.go:1: function "nope" not defined` |
| An extension point declares the item field `when`, which an item of `add_to` uses for its own `when`. | `the starter service: the extension point "api.middleware" declares the item field when, which an item of add_to uses for its own when` |
| The `file` of an extension point is not a template of its declarer, or the extension point names no file. A template whose path is the `file` with `.tmpl` after it is the file. The file of an extension point of a feature can be in any files entry of the feature. | `the starter service: the file "go/config.go" of the extension point "api.config" does not exist` |
| The file of an extension point does not call `items` with the name of the extension point, also when it calls `items` for another extension point or calls only `unique` for it. A feature that has the file in two files entries must call `items` in each copy, and the message names the files entry of a copy that does not. A file that does not parse gives only the mistake of the parse. | `the feature cli: the template go/cli.go of the files entry files/ does not call items "cli.commands"` |
| Two owners write one path of the repo: the starter and a feature that targets it, or two features that target it. The path is the path of the folder joined with the path of the template in the folder, without `.tmpl`, before the render. `check` reads each files entry, also one whose `when` is false. Two templates of one owner are not a mistake. Each pair of owners of a path is one mistake. | `the starter service: the starter and the feature auth both write go.mod` |

Each message names the feature, the starter, and the extension point, the folder or the need that is wrong. A
feature with two targets gives one mistake for each target that it does not fit. A mistake that is not about one
target, for example a `when` that does not parse, comes one time and names no starter. A wrong need of a need is a
mistake of the feature that has the need, not of the features that need that feature.

`check` prints each mistake on its own line to stdout, in this order:

1. The mistakes of each starter, in the order of the starters:
   - The mistakes of each feature that targets it, in the order of the features, the extension points and the
     items.
   - The cycles of the needs.
   - The options that its templates read and cannot read, then its templates that do not parse.
   - The item fields and the files of its extension points, in the order of their names.
   - The paths that two owners write, in the order of the paths.
2. The mistakes of each feature, in the order of the features:
   - Its targets with different item fields.
   - Its `when` expressions that do not parse: those of its files entries, then those of its options, then those
     of its items.
   - The options that its `when` expressions read and cannot read: those of its files entries, then those of its
     options, then those of its items.
   - The options that its templates read and cannot read, then its templates that do not parse.
   - The item fields and the files of its extension points, in the order of their names.

The mistakes of the `when` expressions and the templates of a feature come in the order of its files entries, and
the mistakes of the `when` of its options in the order of the option names.
Then `check` writes the number of the mistakes to stderr, for example `the catalog has 2 mistakes`, and exits
with the status 1. With no mistake, it prints
`The catalog has no mistakes.` and exits with the status 0. A catalog that `check` cannot read, for example a
`starter.yml` that is not valid YAML, stops `check` with the message that `new` and `show` give, and the exit
status is 1.

### Make the Repo of Each Combination

`check` reads the files, but it does not make a repo. A template can parse and still give a repo that does not
build. `check --build` finds this mistake. When the catalog has no mistake, `check --build` makes the repo of each
combination of the catalog in a temporary folder. In each repo, it runs the `after_render` commands and then the
`test` commands of each folder. These are the combinations:

- Each starter alone
- Each starter with each feature that targets it, and the needs of the feature
- One for each value of an option that has `values`, with the other options at their defaults
- One for each value of an option that has `values` of a feature that a feature needs, with both features. The
  option of the need has that value, and each other option of the need has the value that the needs of the feature
  and of its own needs set, or its default. A value that these needs set gives no such combination, and a need of a
  need gives none. When two of these needs set two values of one option, the feature gets no such combination.
- Each pair of features that add items to one extension point, except a pair in which one feature needs the
  other: the repo of that feature already holds both

```shell
feature-compose check --catalog . --build
```

```text
The catalog has no mistakes.
The catalog has 3 combinations. Make the repo of each, and run its after_render and test commands.
[1/3] ok    example (0s)
[2/3] ok    example --feature greeting:language=en (0s)
[3/3] FAIL  example --feature greeting:language=fr (0s)
      the test command "sh ./scripts/test.sh" of the folder app failed: exit status 1
      tests/greeting.sh: the greeting is "Hello, my-repo!", not "Bonjour, my-repo!"
```

Each line names the combination as the arguments of `new` that make the same repo, so you can make it again to
find the cause. The repo of each combination is `my-repo`. `check` removes the temporary folder when it ends. When
a combination fails, `check` goes on with the next one, and then exits with the status 1 and names each combination
that failed. Each repo takes as long as its commands: for a Go repo, some seconds when the Go module cache has its
modules, and some minutes when it does not. The catalog gives the list, so a new starter, feature or value needs no
change to the CI of the catalog.
