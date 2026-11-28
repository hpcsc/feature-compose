# Write the Starters and the Features of a Catalog

A catalog is a folder with two parts: `starters/` and `features/`. Each starter is the base of one kind of
repo. Each feature is a part that a repo can add. This document tells how to write both. The catalog of this
repo, in `catalog/`, has an example of each part. To start a catalog of your own, run
`feature-compose init <folder>`: it writes a small catalog with one starter and one feature that work. Then
`feature-compose add starter <name>` and `feature-compose add feature <name> --target <starter>` write the files of
a new starter or feature, with comments that show the fields to fill in.
[Start a Catalog and Write Its First Feature, Step by Step](start-a-catalog.md) goes through these commands.

## The Folders of a Catalog

```text
catalog/
  starters/
    go-event-sourcing-service/
      starter.yml
      files/
        go/                         the folder "go" of the starter
          cmd/api/main.go.tmpl
          Taskfile.yml
  features/
    cli/
      feature.yml
      files/
        go/                         files for the folder "go"
          cmd/{{ kebab .Name }}/main.go.tmpl
    api-auth/
      feature.yml
      files/
        common/go/                  a files entry that the repo always gets
        api-key/go/                 a files entry for one value of an option
        oauth/go/
```

These rules apply to each file under `files/`:

- **The path starts with the name of a folder.** The starter maps each folder name to a path in the repo,
  for example `go` to `.`. So a feature does not know the place of its files in the repo.
- **The tool renders each file as a template,** and also each path. `cmd/{{ kebab .Name }}/main.go.tmpl`
  writes `cmd/my-tool/main.go`. [Templates](#templates) tells what a template can read.
- **The tool removes `.tmpl` at the end of a name.** Use it for Go files and `go.mod` in a catalog that is in a
  Go module, so that `go build ./...` of the module does not build them.
- **A written file that starts with `#!` can run.** The tool writes it with the mode `0755`, and each other
  file with `0644`.
- **Each file of the repo has one owner,** the starter or one feature. When two of them write the same path,
  `new` stops, and the message names both. `check` finds two owners of one path for each starter, also when
  the `when` of a files entry is false. One feature can write one path from two files entries, for example
  with `when` expressions that exclude each other. `.feature-compose.yml` at the root of the repo has the owner
  feature-compose, because `new` writes the selection file there. A template that writes this path stops `new`.

## starter.yml

```yaml
summary: A Go service with an event store, an HTTP API, and a catchup process.

options:
  go-version:
    default: "1.27.1"
    pattern: '^1\.(2[7-9]|[3-9][0-9])(\.[0-9]+)?$'
  module:
    default: github.com/hpcsc/{{ .Name }}

folders:
  go:
    path: .
    after_render: [go mod tidy, gofmt -w .]
    test: [task test:unit]

extension_points:
  api.middleware:
    summary: An expression of type func(http.Handler) http.Handler, which each request to a protected route goes through.
    file: go/internal/api/server/server.go
    item: [code, import]

delimiters:
  - glob: "go/**/Taskfile*.yml"
    left: "[["
    right: "]]"
```

| Field | Meaning |
| --- | --- |
| `summary` | One sentence about what a repo from the starter gets. `show` shows it. |
| `options` | The questions of the starter. [Options](#options) tells how they work. |
| `folders.<name>.path` | The path of the folder in the repo. |
| `folders.<name>.pinned_tools` | With `true`, each command of the folder runs with the tools that the `mise.toml` of the folder pins. [Pinned Tools](#pinned-tools) tells how it works. |
| `folders.<name>.generate` | A command that writes the first files of the folder, for example `mix phx.new`. [A Starter on Top of a Generator](#a-starter-on-top-of-a-generator) tells how it works. |
| `folders.<name>.after_render` | The commands that `new` runs in the folder after it writes all files. |
| `folders.<name>.test` | The commands that a test of the catalog runs in the folder, for example in CI. `new` does not run them. |
| `extension_points.<name>.summary` | The contract of an item: what its fields must hold, and which names its code can use. |
| `extension_points.<name>.file` | The file that writes the items, as a path that starts with a folder name. |
| `extension_points.<name>.item` | The fields of each item. |
| `extension_points.<name>.language`, `pattern`, `rule`, `after`, `before`, `separator`, `insert` | The place of an extension point in a file that the generator writes. [Extension Points in Generated Files](#extension-points-in-generated-files) tells how they work. |
| `remove` | The files, and the code, that the starter removes from the output of its generator. |
| `add_to` | The items that the starter adds to its own extension points, in the form of the `add_to` of a feature. They come before the items of the features. |
| `delimiters` | Other template delimiters for the files whose catalog path matches `glob`. The glob also matches the folder name and `.tmpl`. |

Use other delimiters for a file whose own syntax has `{{`, for example a Taskfile or a GitHub workflow.

## feature.yml

```yaml
summary: The API accepts a request to a protected route only when it carries a credential that the API accepts.
targets: [chi-api, go-event-sourcing-service]

options:
  method:
    values: [api-key, oauth]
    default: api-key

files:
  - path: files/common/
  - path: files/api-key/
    when: method == "api-key"
  - path: files/oauth/
    when: method == "oauth"

add_to:
  api.middleware:
    - code: auth.Middleware(cfg.Auth)
      import: internal/auth
```

```yaml
summary: The CLI sends the credential that the API asks for, in the method of api-auth.
targets: [chi-api, go-event-sourcing-service]
needs: [cli, api-auth]

options:
  store-key:
    values: ["true", "false"]
    default: "false"
    when: api-auth.method == "api-key"

files:
  - path: files/common/
  - path: files/api-key/
    when: api-auth.method == "api-key"
  - path: files/oauth/
    when: api-auth.method == "oauth"
  - path: files/store/
    when: api-auth.method == "oauth" || store-key

add_to:
  cli.commands:
    - code: newAuthCommand()
  gomod.requires:
    - module: golang.org/x/oauth2
      version: v0.38.0
```

| Field | Meaning |
| --- | --- |
| `summary` | One sentence about what the feature adds. `show` shows it. |
| `targets` | The starters that the feature works with. `new` stops when you select the feature with another starter. |
| `needs` | The features that this feature cannot work without, as a list, or as a map whose values set options of the feature that each key names. |
| `options` | The questions of the feature. |
| `files` | The files entries of the feature. `path` is a folder under the folder of the feature. `when` adds the entry only when its expression is true. |
| `add_to` | The items that the feature adds to each extension point. |
| `extension_points` | The extension points that the feature declares in its own files. Other features can add items to them. |
| `delimiters` | Other template delimiters for the files of the feature, as in `starter.yml`. |

## Options

An option is a question. Its answer selects a variant of the files.

| Field | Meaning |
| --- | --- |
| `values` | The values that the option accepts. With no `values`, the option accepts any text. |
| `pattern` | A regular expression that a text must match. |
| `default` | The value when nobody gives one. A default can read the name of the repo, for example `github.com/hpcsc/{{ .Name }}`. An option with no default must get a value. |
| `when` | An expression that decides when the option applies, in the language of the `when` of a files entry. Only an option of a feature can have a `when`. |

The tool takes the value of an option from the first of these sources that gives one:

1. The command line, for example `--option go-version=1.28` or `--feature api-auth:method=oauth`
2. A need of a selected feature, for example a feature `client` with `needs: {auth: {method: api-key}}` sets
   `auth.method` to `api-key`
3. The form of `new`
4. The default

When the command line and a need give different values, `new` stops, and the message names both sources.

### An Option That Applies Only Sometimes

An option with a `when` applies only while its expression is true. The expression reads the options of the needs of
the feature as `<need>.<option>`, the options of the starter as `starter.<option>`, and the options of the feature
itself that have no `when`, by their names. For example, a feature `client` that needs `auth` declares:

```yaml
options:
  store-key:
    values: ["true", "false"]
    default: "false"
    when: auth.method == "api-key"
```

While the `when` is false, the option does not apply:

- The tool asks no question for it, and the form of `new` shows no field for it.
- It has its default, also when you gave it another value in the form. That value comes back when the `when` is true
  again.
- An option with no default is not missing.
- A flag or a need that gives it its default is accepted. A flag or a need that gives it another value stops `new`,
  and the message says when the option applies, for example `the option store-key of the feature client applies only
  when auth.method == "api-key", and auth.method is "oauth", but the command line gives it "true"`.

The `when` of an option cannot read an option of its own feature that has a `when` of its own, so the `when`
expressions of one feature form no cycle. A `when` that reads a name that it cannot read stops `new`.

Two features that do the same job are one feature with an option. For example, `api-auth` has one option,
`method`, and no second feature for OAuth. So the catalog has no list of features that exclude each other.

## Needs

A need is a feature that another feature cannot work without. When you select a feature, `new` also selects
each feature that it needs, and the needs of those features. A need must target each target of the feature
that needs it.

A need can set options of the feature that it names. A feature `client` that works only with the method
`api-key` of a feature `auth` writes:

```yaml
needs:
  auth:
    method: api-key
```

Then `new` stops when a flag gives `auth.method` another value, and the form does not change the value.

A feature that works with each value reads the option of its need in place of a value in its need. `cli-auth`
needs `api-auth` with no value, and the `when` of its files entries reads `api-auth.method`. So the CLI uses the
method of the API: the tool asks for `method` one time, as the option of `api-auth`. The `when` of the option
`store-key` of `cli-auth` also reads `api-auth.method`, so the tool asks for `store-key` only with the method
`api-key`.

## Files for One Value of an Option

Each files entry of a feature has its own path. `when` in a files entry adds the entry only when its expression
is true:

| Expression | True when |
| --- | --- |
| `method == "oauth"` | The option has this value. |
| `method != "oauth"` | The option has another value. |
| `store-key` | The option has the value `true`. |
| `!a`, `a && b`, `a \|\| b`, `(a)` | The rules of logic apply. |

A name in an expression is an option of the feature, for example `method`, an option of a need, for example
`api-auth.method`, or an option of the starter, for example `starter.module`. A name can hold `-`, so
`store-key` is one name.

## Extension Points

An extension point is a named list in one file. The starter, or a feature, declares it. Other features add
items to it, and the file writes each item. So a feature changes a file that it does not own, and no step
edits the file after the render. The starter, or a feature, can also declare an extension point in a file
that a generator writes: [Extension Points in Generated Files](#extension-points-in-generated-files) tells
how.

A feature adds items with `add_to`. Each item has the fields that the extension point declares:

```yaml
add_to:
  api.middleware:
    - code: auth.Middleware(cfg.Auth)
      import: internal/auth
```

The file of the extension point writes the items with `items`, and the values of one field with `unique`:

```go
import (
	"net/http"
{{- range unique "import" "api.middleware" }}
	"{{ option "starter.module" }}/{{ . }}"
{{- end }}
)

func protectedMiddleware(cfg *api.Config) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
{{- range items "api.middleware" }}
		{{ .code }},
{{- end }}
	}
}
```

- The value of an item goes into the file as it is. The tool does not render it as a template. An item of an
  extension point in a generated file is different: the tool renders its values.
- The items have a fixed order: a feature comes after the features that it needs, and other features come in
  the order of their names. So the order of the `--feature` flags does not change a file.
- With no item, the `range` writes nothing. Put the `{{-` and `-}}` markers so that the file of a repo with no
  item is the same as a file with no extension point.
- A feature can add items only to an extension point that the starter, the feature, or one of its needs
  declares.
- The `file` of an extension point is the path of a template of its declarer, without `.tmpl`. It starts with
  a folder name: for a starter, it is the path under `files/`, and for a feature, the path in a files entry.
  The template must call `items` with the name of the extension point. `check` finds a `file` that does not
  exist, and a file that does not call `items` for its extension point, also when it calls only `unique`.

### An Item That Applies Only Sometimes

`when` in an item adds the item only when its expression is true. The expression has the language of the
`when` of a files entry, and reads the options that the feature that adds the item can read. For example, a
feature with the option `method` adds a second middleware only for one value:

```yaml
add_to:
  api.middleware:
    - code: auth.Middleware(cfg.Auth)
      import: internal/auth
    - code: auth.Audit(cfg.Auth)
      import: internal/auth
      when: method == "oauth"
```

- An item with no `when` is in each repo.
- `items` gives only the items whose `when` is true, and `unique` gives only the values of these items. The
  items keep their order.
- `when` is not an item field. An extension point cannot declare the item field `when`, and `check` finds one
  that does.
- A `when` that does not parse stops `new`. `check` finds it, and also a `when` that reads an option that the
  feature cannot read.

An item can hold a block of YAML, for example a job of a CI workflow. The feature writes the block with each
key at the left edge, and the file of the extension point sets the indent with `indent`:

```yaml
add_to:
  ci.jobs:
    - text: |-
        e2e:
          runs-on: ubuntu-latest
          steps:
            - uses: actions/checkout@v7
            - name: Run E2E Test
              run: task test:e2e
```

```yaml
jobs:

  test:
    runs-on: ubuntu-latest
    steps:
      - name: Run Unit Test
        run: task test:unit
[[- range items "ci.jobs" ]]

[[ indent 2 .text ]]
[[- end ]]
```

So a feature adds a job to the CI workflow of the starter, and the workflow keeps its triggers and its
permissions. A job of an item can name the jobs of the starter in `needs`. The workflow uses the delimiters
`[[` `]]`, because GitHub Actions uses `${{ }}`.

The `summary` of an extension point is a contract that the tool cannot check. A test of each combination of
the catalog checks it: [Build, Test and Release feature-compose](development.md) tells how this repo does
this.

## Pinned Tools

A command of a folder runs with the tools on the `PATH` of the machine. So a repo can get another version of Go,
Elixir or a generator than its catalog expects. With `pinned_tools: true`, each command of the folder runs with the
versions that the `mise.toml` of the folder pins, with [mise](https://mise.jdx.dev):

```yaml
folders:
  app:
    path: .
    pinned_tools: true
    after_render: [mix deps.get]
```

```toml
# files/app/mise.toml
[tools]
erlang = "29.1"
elixir = "1.20.4-otp-29"
```

- The tool reads the section `[tools]` of the `mise.toml` that the starter or a feature writes in the folder, and
  runs each `generate`, `after_render` and `test` command as `mise exec erlang@29.1 elixir@1.20.4-otp-29 -- sh -c
  '<command>'`. The `mise.toml` is the one place of the versions. The repo gets the same file, so its own commands
  use the same versions.
- mise installs a version that the machine does not have. So a user needs mise, and not the tools themselves.
- mise reads no `mise.toml` in the folder of the command, or in a folder above it. So mise does not trust and does
  not track the new repo, and a `mise.toml` of a parent folder does not change the tools. The tool sets
  `MISE_CEILING_PATHS` and `MISE_IGNORED_CONFIG_PATHS` for this.
- The tool also gives mise the version of each tool with a plain name as `MISE_<TOOL>_VERSION`, for example
  `MISE_ERLANG_VERSION=29.1`. mise gives the install of a tool only the tools of its config, and these variables
  count as config. So the install of `elixir` finds the `erl` of the pinned Erlang.
- Each version is a text, for example `elixir = "1.20.4-otp-29"`. Another form stops `new` and names the tool.
- The preview gives each command with its `mise exec` part.
- `check` finds a folder with `pinned_tools` whose starter writes no `mise.toml` in it.

## A Starter on Top of a Generator

Some kinds of repo start from the output of a generator, for example `mix phx.new` for a Phoenix app. A copy of
that output in the catalog gets old when the generator changes. So a starter can run the generator, and then
change its output:

```yaml
options:
  app:
    default: "{{ snake .Name }}"
  phx-new-version:
    default: 1.8.17

folders:
  app:
    path: .
    pinned_tools: true
    generate: >-
      MIX_ARCHIVES="$PWD/.archives" mix local.hex --force &&
      MIX_ARCHIVES="$PWD/.archives" mix archive.install hex phx_new {{ option "starter.phx-new-version" }} --force &&
      MIX_ARCHIVES="$PWD/.archives" mix phx.new {{ .Folder }} --app {{ option "starter.app" }} --no-install
    after_render: [mix local.hex --force --if-missing, mix deps.get, mix format]

remove:
  - file: app/lib/{{ option "starter.app" }}_web/controllers/page_controller.ex
  - file: app/lib/{{ option "starter.app" }}_web/router.ex
    language: elixir
    pattern: get "/", PageController, :home
```

`new` makes the repo in these steps:

1. It runs the `generate` command of each folder that has one.
2. It removes the files and the code that `remove` names.
3. It writes the files of the starter and of the features.
4. It adds the items of each extension point in a generated file.
5. It runs the `after_render` commands, and then writes `.feature-compose.yml`.

- `{{ .Folder }}` is a path in an empty temporary folder, and the path does not exist yet. Its last part is the
  name of the repo. The command must write its files there, and the tool then moves them into the folder of the
  starter. Some generators stop when their folder exists, so the tool does not give them the output folder.
- The command gets no input. Give it the flags that stop each question, for example `--no-install`.
- The command runs in the temporary folder, and the tool deletes that folder after the command. So a generator
  that the command installs there, for example into `MIX_ARCHIVES="$PWD/.archives"`, does not change the machine.
  An option for the version of the generator, such as `phx-new-version`, makes the same command give the same repo
  again.
- The command is a template, and it can read the options of the starter.
- A file of the starter at the path of a generated file replaces the generated file. A file of a feature at such
  a path stops `new`.
- A removal with only `file` removes the file, and each folder that it leaves empty. A removal with a `pattern`
  or a `rule` removes the one place that it matches, and also the line of that place when the line is then
  empty. The `file` of a removal starts with a folder name, as the `file` of an extension point does.
- A starter with a generator does not need a `files/` folder.

### Extension Points in Generated Files

A file of the generator has no `items` call. So the starter tells where the items go, with a pattern or a rule
of [ast-grep](https://ast-grep.github.io):

```yaml
extension_points:
  mix.deps:
    summary: A dependency of the app, as a tuple of deps in mix.exs.
    file: app/mix.exs
    item: [dep]
    language: elixir
    pattern: defp deps do [$$$DEPS] end
    after: DEPS
    separator: ","
    insert: "{{ .dep }}"
```

| Field | Meaning |
| --- | --- |
| `language` | The ast-grep language of the file, for example `elixir` or `markdown`. The tool gives ast-grep the text of the file, so the name of the file does not change the language. |
| `pattern` | An ast-grep pattern. It must match exactly one place in the file. |
| `rule` | An ast-grep rule, in place of `pattern`, for a place that a pattern cannot match alone. For example, a rule can match a node only inside another node. |
| `after` | The items go after the last node of this metavariable. `match` is the whole match. |
| `before` | The items go before the first node of this metavariable, and above the comment lines directly above that node. `match` is the whole match. |
| `separator` | The text between two items of a list, for example `,`. |
| `insert` | A template for the text of one item. `{{ .dep }}` reads a field of the item. |

This rule matches the endpoint only in the list of children, and the items go before it:

```yaml
extension_points:
  application.children:
    summary: A child that the supervisor of the app starts, before the endpoint.
    file: app/lib/{{ option "starter.app" }}/application.ex
    item: [child]
    language: elixir
    rule:
      pattern: '{{ pascal (option "starter.app") }}Web.Endpoint'
      inside:
        pattern: children = [$$$CHILDREN]
        stopBy: end
    before: match
    separator: ","
    insert: "{{ .child }}"
```

- Each item goes on its own line, with the indentation of the node next to it. An `after_render` command, for
  example `mix format`, fixes the rest of the layout.
- The value of an item is a template, and it can read the options that its feature can read. Code in a
  generated file often names the app, for example `config :{{ option "starter.app" }}, Oban`.
- So `{{` in the value of an item starts a template action. For code that holds `{{`, for example a nested
  tuple of Elixir, write `{ {` when the formatter of the `after_render` commands joins it, as `mix format`
  does. Otherwise write `{{ "{{" }}`.
- The `file`, the `pattern` and the strings of the `rule` are templates that can read the options of the
  starter.
- A pattern that matches no place, or more than one place, stops `new`. The message names the extension point
  and the file. A new version of the generator can change its files, so run `check --build` on a schedule.
- The tool runs ast-grep only for an extension point that has items.
- A feature can declare an extension point in a generated file too. `check` finds it as a mistake for each
  target whose folder has no `generate` command. The `file` and the pattern of the extension point read only
  the options of the starter.
- `new` and `check` need `ast-grep` 0.45.3 or later on the `PATH` when a starter changes the files of its
  generator.

`check` finds these mistakes without the generator: an unknown language, a rule that ast-grep refuses, a pattern
that ast-grep reads with an `ERROR` node, an `after` or a `before` that is not a metavariable of the pattern or
of the rule, and a template of a generated file that does not parse or reads an option that its owner cannot
read. Only `check --build` finds a pattern that matches no place, because only it runs the generator.

## Templates

The tool renders each file and each path with Go [`text/template`](https://pkg.go.dev/text/template). A
template can read these values and functions:

| In a template | Value |
| --- | --- |
| `{{ .Name }}` | The name of the repo, from `feature-compose new <starter> <name>`. |
| `{{ option "method" }}` | An option of the feature of the template. |
| `{{ option "api-auth.method" }}` | An option of a need of the feature. |
| `{{ option "starter.module" }}` | An option of the starter. A template of the starter reads its options with this form too. |
| `{{ range items "api.middleware" }}` | The items of an extension point, in their fixed order. `{{ .code }}` reads a field of the item. |
| `{{ range unique "import" "api.middleware" "api.config" }}` | The values of one field over one or more extension points. Each value comes one time. |
| `{{ indent 2 .text }}` | The text with 2 spaces before each line that is not empty. Use it for an item that holds a block of YAML. |
| `{{ githubRepository (option "starter.module") }}` | `<account>/<repo>` of a module `github.com/<account>/<repo>`. Another module stops the render. |
| `kebab`, `snake`, `camel`, `pascal`, `upper` | Functions that change the case of a text, for example `{{ kebab .Name }}`. |

A template that reads an unknown option, or an extension point that nobody declares, stops `new` before it
writes the first file. The message names the template. `check` finds a template that reads an option that it
cannot read, also a template that no repo of CI renders, because it reads the parse tree of each template.

## Check the Catalog

`feature-compose check` finds each feature that does not fit its targets or its needs, for example an item with
the wrong fields, or a need that does not target the starter. It also finds each template that reads an option
that it cannot read, or that does not parse.
[What Each Command of feature-compose Does](commands.md#check-the-catalog) lists each mistake that it finds.
Run it in the CI of the catalog, so that a mistake stops before somebody runs `new`.

`check` reads the files, but it does not make a repo. To find a template that does not build, run
`check --build`. It makes the repo of each combination and runs its commands.
[Make the Repo of Each Combination](commands.md#make-the-repo-of-each-combination) lists the combinations.
