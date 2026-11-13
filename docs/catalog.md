# Write the Starters and the Features of a Catalog

A catalog is a folder with two parts: `starters/` and `features/`. Each starter is the base of one kind of
repo. Each feature is a part that a repo can add. This document tells how to write both. The catalog of this
repo, in `catalog/`, has an example of each part.

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
  `new` stops, and the message names both.

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
| `folders.<name>.after_render` | The commands that `new` runs in the folder after it writes all files. |
| `folders.<name>.test` | The commands that a test of the catalog runs in the folder, for example in CI. `new` does not run them. |
| `extension_points.<name>.summary` | The contract of an item: what its fields must hold, and which names its code can use. |
| `extension_points.<name>.file` | The file that writes the items, as a path that starts with a folder name. |
| `extension_points.<name>.item` | The fields of each item. |
| `delimiters` | Other template delimiters for the files whose catalog path matches `glob`. The glob also matches the folder name and `.tmpl`. |

Use other delimiters for a file whose own syntax has `{{`, for example a Taskfile or a GitHub workflow.

## feature.yml

```yaml
summary: The API accepts a request to a protected route only when it carries a credential that the API accepts.
targets: [go-event-sourcing-service]

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
summary: The CLI sends the API key that the API asks for.
targets: [go-event-sourcing-service]
needs:
  cli:
  api-auth:
    method: api-key

files:
  - path: files/common/

add_to:
  cli.commands:
    - code: newAuthCommand()
```

| Field | Meaning |
| --- | --- |
| `summary` | One sentence about what the feature adds. `show` shows it. |
| `targets` | The starters that the feature works with. `new` stops when you select the feature with another starter. |
| `needs` | The features that this feature cannot work without. A need can set options of the feature that it names, for example `method: api-key`. |
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

The tool takes the value of an option from the first of these sources that gives one:

1. The command line, for example `--option go-version=1.28` or `--feature api-auth:method=oauth`
2. A need of a selected feature, for example the need of `cli-auth` sets `api-auth.method` to `api-key`
3. The form of `new`
4. The default

When the command line and a need give different values, `new` stops, and the message names both sources.

Two features that do the same job are one feature with an option. For example, `api-auth` has one option,
`method`, and no second feature for OAuth. So the catalog has no list of features that exclude each other.

## Needs

A need is a feature that another feature cannot work without. When you select a feature, `new` also selects
each feature that it needs, and the needs of those features. A need must target each target of the feature
that needs it.

A need can set options of the feature that it names. `cli-auth` sets `api-auth.method` to `api-key`, because
the CLI sends an API key. So the CLI and the API cannot use different methods: the API decides, and the
feature that needs it follows.

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
edits the file after the render.

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

- The value of an item goes into the file as it is. The tool does not render it as a template.
- The items have a fixed order: a feature comes after the features that it needs, and other features come in
  the order of their names. So the order of the `--feature` flags does not change a file.
- With no item, the `range` writes nothing. Put the `{{-` and `-}}` markers so that the file of a repo with no
  item is the same as a file with no extension point.
- A feature can add items only to an extension point that the starter, the feature, or one of its needs
  declares.

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
writes the first file. The message names the template.

## Check the Catalog

`feature-compose check` finds each feature that does not fit its targets or its needs, for example an item with
the wrong fields, or a need that does not target the starter.
[What Each Command of feature-compose Does](commands.md#check-the-catalog) lists each mistake that it finds.
Run it in the CI of the catalog, so that a mistake stops before somebody runs `new`.

`check` reads the files, but it does not build a repo. To find a template that does not build, make a repo of
each combination and run its commands. This repo does this with `task test:combinations`.
