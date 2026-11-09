# feature-compose: Make a Repo From One Starter and Reusable Features

feature-compose makes a new repo from one starter and the features that you select. It adds the features
that your features need, asks each question one time, and renders all files in one pass. One feature can
work with many starters.

## 1. Problem

The templates in `repo-templates` use [scaffold](https://hay-kot.github.io/scaffold/). Scaffold renders one
template at a time. This works when a repo needs one template. It fails when a repo needs many templates
that depend on each other.

The first case is authentication for a CLI that calls its own API:

- `global/go-cli-auth` gives a Go CLI its credential. Its `oauth` preset sends an access token from an
  OAuth issuer.
- The server must check that token. `chi-api` and `go-event-sourcing-service` compare the header with one
  shared token from a file.
- So the CLI and the server need two templates that use the same method. Scaffold cannot make sure of this.

These limits of scaffold cause the problem. Tests with scaffold 0.13.1 and 0.14.0 show each limit.

| Limit | Result |
| --- | --- |
| A template cannot run another template. A hook can run one, but the hook variables do not include the path of the template. | The user must know which templates to run, in which order, and with which presets. |
| Two templates cannot share an answer. | The CLI and the server can get different methods of authentication. |
| Scaffold does not merge two versions of one file. 0.13.1 replaces the file and 0.14.0 skips it. Neither version shows a message. | A template that changes a shared file, for example `main.go`, must change it in its hook. |
| The hooks change shared files with bash, perl, jq, and ast-grep. | The `go-cli-auth` hook has 121 lines. Its rule that finds the `Commands` list of the root command has 25 lines. Each new template needs rules like this again. |
| In a global template, `.Project` is always `templates`. | A project template and a global template cannot share a file that uses `{{ .ProjectKebab }}`. The process manager files are copies in two templates, and CI finds differences with `diff -r`. |
| The generated repo has no record of its templates and answers. | Later, you cannot see which presets made the repo. |

## 2. Goals

- **G1:** You select features. The tool also selects the features that they need.
- **G2:** The tool asks each question one time. Two features that must agree read the same option.
- **G3:** One feature works with many starters.
- **G4:** The tool renders all files in one pass. A shared file is a template that renders the items that
  features add to it. No markers and no hooks change the files after the render.
- **G5:** One feature can add files in more than one language.
- **G6:** A check of the catalog finds mistakes in CI, before a user runs the tool.
- **G7:** The generated repo keeps a record of its starter, features, and options.

## 3. Non-Goals for Version 1

- **Add a feature to a repo that you changed.** This needs a three-way merge. Phase 3 adds it.
- **Update a generated repo to a new version of the catalog.** This needs the same merge.
- **Generators that you run many times in a repo, for example `go-cli-sub-command`.** Scaffold does this
  job well, so these generators stay in scaffold.
- **More than one catalog at a time.** The config file names one catalog folder.

## 4. Concepts

| Concept | Meaning |
| --- | --- |
| Starter | The files that a new repo of one kind starts from, for example `chi-api` or `go-event-sourcing-service`. |
| Folder | A named folder of a starter, for example `go` or `web`. Each folder has its own commands for after the render. |
| Extension point | A named list in one file. Features add items to the list, and the file renders the items. |
| Feature | Something that you can select. It adds files to folders, and it adds items to extension points. |
| Option | A question that a starter or a feature asks. |
| Target | A starter that a feature works with. Each feature names its targets. |
| Need | A feature that another feature cannot work without. |
| Selection | The starter, the features, and the options of one repo. |

## 5. The Catalog

```
catalog/
  starters/
    chi-api/
      starter.yml
      files/
        go/
    go-event-sourcing-service/
      starter.yml
      files/
        go/
  features/
    cli/
      feature.yml
      files/
        go/
    api-auth/
      feature.yml
      files/
        common/go/
        oauth/go/
    cli-auth/
      feature.yml
      files/
        common/go/
        oauth/go/
        store/go/
```

Each path in the catalog starts with the name of a folder. A starter maps each folder name to a path in the
repo. So a feature does not know where its files go. In one starter, `go` is the root of the repo. In
another starter, `go` is `api/`.

## 6. starter.yml

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
    summary: An expression of type func(http.Handler) http.Handler. It can use cfg.
    file: go/internal/api/server/server.go
    item: [code, import]
  api.config:
    summary: A field in the config of the API.
    file: go/internal/api/config.go
    item: [name, type, import]
  catchup.consumers:
    summary: An expression of type consumer.Handler. It can use bus and logger.
    file: go/cmd/catchup/main.go
    item: [code, import]

delimiters:
  - glob: "go/**/Taskfile*.yml"
    left: "[["
    right: "]]"
```

| Field | Meaning |
| --- | --- |
| `summary` | One sentence about what the repo gets. `show` shows it. |
| `options` | The questions of the starter. Section 8 tells how options work. |
| `folders.<name>.path` | The path of the folder in the repo. |
| `folders.<name>.after_render` | The commands that run in the folder after the tool renders all files. |
| `folders.<name>.test` | The commands that CI runs in the folder to test a rendered repo. |
| `extension_points.<name>.summary` | The contract for an item: what the code must be, and which names it can use. |
| `extension_points.<name>.file` | The file that renders the extension point. |
| `extension_points.<name>.item` | The fields of each item. |
| `delimiters` | The template delimiters for a file pattern. |

The `summary` of an extension point is a contract that the tool cannot check. CI checks it when it builds
each starter with each feature that targets the starter.

## 7. feature.yml

```yaml
# features/api-auth/feature.yml
summary: The API accepts only requests that have a valid credential.
targets: [chi-api, go-event-sourcing-service]

options:
  method:
    values: [api-key, oauth]
    default: api-key

files:
  - path: files/common/
  - path: files/oauth/
    when: method == "oauth"

add_to:
  api.middleware:
    - code: auth.Middleware(cfg.Auth)
      import: internal/auth
  api.config:
    - name: Auth
      type: auth.Config
      import: internal/auth
```

```yaml
# features/cli/feature.yml
summary: A command line program that calls the API, with releases from GoReleaser.
targets: [chi-api, go-event-sourcing-service]

files:
  - path: files/

extension_points:
  cli.commands:
    summary: An expression of type *cli.Command.
    file: go/internal/cli/root.go
    item: [code]
```

```yaml
# features/cli-auth/feature.yml
summary: The CLI sends the credential that the API asks for.
targets: [chi-api, go-event-sourcing-service]
needs: [cli, api-auth]

options:
  store-key:
    values: [true, false]
    default: false
    when: api-auth.method == "api-key"

files:
  - path: files/common/
  - path: files/oauth/
    when: api-auth.method == "oauth"
  - path: files/store/
    when: api-auth.method == "oauth" || store-key

add_to:
  cli.commands:
    - code: newAuthCommand()
```

A feature can add files in more than one language. This example targets a starter with a Go API in `api/`
and a Svelte web app in `web/`:

```yaml
# features/google-sign-in/feature.yml
summary: Users sign in with Google in the web app, and the API gives them a session token.
targets: [go-api-svelte-web]

files:
  - path: files/

add_to:
  api.routes:
    - code: auth.SignInRoutes(cfg.Google)
      import: internal/auth
  web.routes:
    - path: /sign-in
      component: SignIn
      import: $lib/SignIn.svelte
```

| Field | Meaning |
| --- | --- |
| `summary` | One sentence about what the repo gets. `show` shows it. |
| `targets` | The starters that the feature works with. |
| `needs` | The features that this feature cannot work without. A need can set option values, for example `needs: {api-auth: {method: oauth}}`. |
| `options` | The questions of the feature. |
| `files` | The folders of files that the feature adds. `when` adds a folder only when its expression is true. |
| `add_to` | The items that the feature adds, for each extension point. |
| `extension_points` | The extension points that the feature renders in its own files. |
| `folders` | The new folders that the feature adds to the repo, with their commands. |
| `delimiters` | The template delimiters for a file pattern. |

`method` is an option of `api-auth` only. `cli-auth` reads `api-auth.method`, so the CLI and the API
cannot use different methods. The server decides the method, and the CLI follows it.

## 8. Options

An option is a question. Its answer selects the variant of the starter or the feature that the repo gets.

- An option with `values` accepts only those values.
- An option without `values` accepts any text. `pattern` can limit the text.
- `when` asks the question only when its expression is true. When the expression is false, the option has
  its default value.

The tool takes the value of an option from these sources:

1. The command line, for example `--feature api-auth:method=oauth`
2. A need of another feature, for example `needs: {api-auth: {method: oauth}}`
3. The TUI, which section 12 describes
4. The default

If the command line and a need give different values, the tool stops and names both sources.

An option changes only the generated code. Values that can change after the render stay in the config of
the repo. Examples are the URL of an issuer and a client ID.

### Expressions

`when` uses a small expression language:

| Expression | True when |
| --- | --- |
| `method == "oauth"` | The option has this value. |
| `method != "oauth"` | The option has another value. |
| `store-key` | The option has the value `true`. |
| `!a`, `a && b`, `a \|\| b`, `(a)` | The usual rules of logic apply. |

A name in an expression is one of these:

- An option of the same feature, for example `method`
- An option of a need, for example `api-auth.method`
- An option of the starter, for example `starter.module`

A name can contain `-`. So `store-key` is one name, not a subtraction. This is the reason for a language of
our own.

## 9. Templates

The tool renders files with Go `text/template`. A file path can also contain a template, for example
`go/cmd/{{ .Name }}/main.go`.

| In a template | Value |
| --- | --- |
| `{{ .Name }}` | The name of the repo, from `feature-compose new <starter> <name>`. |
| `{{ option "method" }}` | An option of this feature. |
| `{{ option "api-auth.method" }}` | An option of a need. |
| `{{ option "starter.module" }}` | An option of the starter. |
| `{{ range items "api.middleware" }}` | The items of an extension point, in their fixed order. |
| `{{ range unique "import" "api.middleware" }}` | The values of one field in one or more extension points. Each value comes one time. |
| `kebab`, `snake`, `camel`, `pascal` | Functions that change the case of a name. |

This is how `server.go` renders `api.middleware`:

```go
import (
	"net/http"
{{- range unique "import" "api.middleware" }}
	"{{ option "starter.module" }}/{{ . }}"
{{- end }}
)

func protected(cfg config.Config, next http.Handler) http.Handler {
{{- range items "api.middleware" }}
	next = {{ .code }}(next)
{{- end }}
	return next
}
```

## 10. Rules

1. A starter declares its folders and its extension points. A feature can also declare extension points in
   its own files.
2. A feature names its targets. For each target, the tool checks two things:
   - The starter has each folder that the feature adds files to.
   - The starter, or a need, declares each extension point in `add_to`, with the same item fields.
3. A need must also target each target of the feature. For example, `cli-auth` can target only the starters
   that both `cli` and `api-auth` target.
4. When you select a feature, the tool also selects its needs, with the option values that they set.
5. A feature reads only its own options, the options of its needs, and the options of the starter. It can
   read a starter option only when each of its targets declares that option.
6. A feature puts its code in its own folders, for example `internal/auth`. It connects to the starter only
   through extension points.
7. Each file has one owner: the starter or one feature.
8. The items of an extension point have a fixed order. A feature comes after its needs. Features with no
   need between them come in the order of their names. So the order of the flags does not change a file.
9. The commands of each folder run after the tool renders all files. A feature that adds a folder declares
   the commands for that folder.

Two features that do the same job become one feature with an option. Each value of an option excludes the
other values. So the catalog has no `excludes` field.

Rule 6 makes reuse possible. If `api-auth` uses the `response` package of one starter, it does not build
with another starter. Some features must use the code of a starter, for example `process-manager` uses the
event store. Rule 2 keeps such a feature in the correct starters, because only those starters declare its
extension points.

## 11. Commands

| Command | Job |
| --- | --- |
| `feature-compose new <starter> <name>` | Makes a new repo. In a terminal, a TUI asks for each answer that the arguments do not give. |
| `feature-compose show` | In a terminal, opens a TUI to browse the starters, the features, and the extension points. With `--output json`, or when stdout is not a terminal, prints the catalog as JSON. |
| `feature-compose show <starter>` | Opens the TUI at one starter, or prints that starter and the features that target it as JSON. |
| `feature-compose check` | Checks the catalog. Section 14 lists the checks. |
| `feature-compose add <feature>` | Adds a feature to a repo that you changed. Phase 3 adds this command. |

`new` takes these flags:

| Flag | Job |
| --- | --- |
| `--feature <name>[:<option>=<value>,...]` | Selects a feature and sets its options. You can give the flag more than one time. |
| `--option <name>=<value>` | Sets an option of the starter. |
| `--output-dir <path>` | The folder for the new repo. The default is `./<name>`. |
| `--no-prompt` | Does not start the TUI. Uses the default for each option that has no value. |
| `--dry-run` | Prints the selection and the files, and writes nothing. |

`new` does these steps:

1. It adds the needs of each selected feature, and their needs, to the selection.
2. It checks that each selected feature targets the starter.
3. It asks for each option that has no value and whose `when` is true.
4. It renders the files of the starter and of each feature.
5. It runs the commands of each folder.
6. It writes the selection file.

`show` reads the same files that `new` reads, so its output is always correct. It has two forms:

- **The TUI, for a person.** It has two panes. The left pane is a tree of the starters. Under each starter, it
  has a group of the features and a group of the extension points, and `/` starts a filter on the names. The
  right pane shows the details of the selected row. `j` and `k` move in the tree, as the arrow keys do. `l` or
  Enter opens a row, and `h` or Esc closes it. `q` stops, except while a filter is typed, where ctrl+c stops. The
  details of a starter show its summary, folders and options. The details of a feature show its summary, needs,
  options, and the extension points that it declares or adds items to. The details of an extension point show
  its contract, its file, its item fields, its declarer, and the features that add items to it.
- **JSON, for a script.** `--output json` prints the same data. `show` prints JSON with no flag when stdout is
  not a terminal, so a script never waits on a TUI.

## 12. The TUI

When you run `new` in a terminal, a TUI asks for each answer that the arguments do not give. When you run
`new` with no arguments, the TUI shows all six steps:

1. **Select the starter.** The TUI lists the starters, each with its `summary`.
2. **Enter the name of the repo.**
3. **Select the features.** The TUI lists only the features that target the starter, each with its
   `summary`. You can select more than one feature.
4. **Check the needs.** The TUI shows each feature that the tool adds because of a need, for example
   `api-auth: needed by cli-auth`. You can go back to step 3.
5. **Answer the options.** The TUI shows one group of questions for each feature, in the order of rule 8.
   So a need comes before the feature that needs it. For example, `api-auth.method` comes before
   `cli-auth.store-key`, because the `when` of `store-key` reads `method`.
6. **Confirm.** The TUI shows the starter, the features, the options, and the files. This is the same
   output as `--dry-run`. After you confirm, the tool writes the repo.

After the tool writes the repo, it prints the command that makes the same repo without the TUI:

```shell
feature-compose new go-event-sourcing-service my-service \
  --feature cli-auth --feature api-auth:method=oauth
```

These rules connect the TUI to the flags:

- Flags give answers before the TUI starts, and the TUI asks only for the other answers. For example,
  `feature-compose new chi-api my-api` starts at step 3.
- The TUI does not ask for a value that comes from a flag or from a need. The confirm step shows that
  value as fixed.
- When stdin is not a terminal, or when you give `--no-prompt`, the tool does not start the TUI. It uses
  the default of each option. If an option has no value and no default, the tool stops and names the
  option. CI uses this mode.

The TUI uses [huh](https://github.com/charmbracelet/huh) 1.0.0. Scaffold uses the same library for its
prompts.

| Function of huh | Use in the TUI |
| --- | --- |
| `Group.WithHideFunc` | Hides an option when its `when` is false, for example `store-key` when `method` is `oauth`. |
| `MultiSelect.OptionsFunc` | Builds the list of features from the starter of step 1. |
| `Input.Validate` | Checks text against the `pattern` of an option, for example the Go version. |

## 13. The Selection File

`new` writes `.feature-compose.yml` to the root of the repo:

```yaml
version: 0.1.0
starter: go-event-sourcing-service
name: my-service
options:
  go-version: "1.27.1"
  module: github.com/hpcsc/my-service
features:
  cli: {}
  api-auth:
    method: oauth
  cli-auth: {}
requested: [cli-auth]
```

| Field | Meaning |
| --- | --- |
| `version` | The version of feature-compose. The catalog is a folder that the config file names, so this is not the version of the catalog. |
| `starter`, `name` | The arguments of `new`. |
| `options` | The options of the starter. |
| `features` | Each feature in the repo, with its options. |
| `requested` | The features that you named. The tool added the other features because of needs. |

## 14. The Check of the Catalog

`feature-compose check` finds these mistakes:

- An `add_to` names an extension point that a target and its needs do not declare.
- An item does not have the fields that its extension point declares.
- Two targets of a feature declare one extension point with different item fields.
- A feature adds files to a folder that a target does not have.
- A need does not exist, does not target each target of the feature, or sets a value that its option does not have.
- The needs have a cycle.
- Two owners write the same path.
- A template or an expression reads an option that rule 5 does not allow. The tool finds this in the parse
  tree of the template.
- The `file` of an extension point does not exist or does not call `items` for it.

CI also renders each starter with each feature that targets it. Then it runs the `after_render` and the
`test` commands of each folder. CI also renders each pair of features that add to the same extension point.

## 15. First Features: Authentication for a CLI and Its API

`api-auth` and `cli-auth` follow one contract:

- The CLI sends `Authorization: Bearer <credential>`.
- The API answers 401 with `WWW-Authenticate: Bearer` when the credential is absent or not valid. It answers
  403 only for a valid credential that has no permission.
- With `api-key`, the API keeps a SHA-256 hash of each key, and it compares in constant time.
- With `oauth`, the issuer signs JWT access tokens. The API gets the JWKS through discovery. It checks the
  algorithm against an allowlist, then the signature, `iss`, `aud`, and `exp`.
- With `oauth`, the CLI asks the issuer for a token for the audience of the API. Without the audience, most
  issuers give a token that the API cannot check. `go-cli-auth` now asks only for the scopes
  `openid profile email offline_access`, so `cli-auth` must add the audience.

This command shows how the rules work together:

```shell
feature-compose new go-event-sourcing-service my-service --feature cli-auth
```

1. `cli-auth` needs `cli` and `api-auth`, so the tool selects both.
2. The tool checks that the three features target `go-event-sourcing-service`.
3. The tool asks for `method` one time, because `api-auth` owns that option.
4. If the answer is `oauth`, the tool does not ask for `store-key`.
5. The tool renders the starter and the three features. Then it runs `go mod tidy` and `gofmt`.

## 16. The Move From repo-templates

| Template in repo-templates | In feature-compose |
| --- | --- |
| `go-event-sourcing-service` | The starter `go-event-sourcing-service`, without the process manager |
| `global/go-event-sourcing-service-process-manager` | The feature `process-manager` |
| `chi-api` | The starter `chi-api` |
| `go-cli` | A starter `go` for a Go module, and the feature `cli` |
| `global/go-cli-auth` | The feature `cli-auth`, and the new feature `api-auth` |
| `global/go-cli-sub-command` | Stays in scaffold |

The other templates are not yet decided: `asdf-plugin`, `go-event-sourcing-cli`, `go-worker`,
`global/git-hook`, and `global/kustomize`.

`go-cli` becomes a starter and a feature, so a CLI in its own repo and a CLI in a server repo use the same
files. `repo-templates` stays until each template that moves has its replacement.

## 17. Plan

| Phase | Scope |
| --- | --- |
| 1 | The format of the catalog. The commands `new`, `show`, and `check`. The TUIs for `new` and `show`. The starter `go-event-sourcing-service`. The features `cli`, `api-auth`, and `cli-auth`. A CI job that renders and tests each pair. |
| 2 | The starters `chi-api` and `go`. The feature `process-manager`. The other templates from section 16. |
| 3 | The command `add`, with a three-way merge. The tool renders the old selection and the new selection, and it applies the difference to the repo. |

## 18. Decisions

| ID | Decision | Reason |
| --- | --- | --- |
| D1 | The name is `feature-compose`. | It names the difference from other tools: this tool composes features. `compose` alone conflicts with Docker Compose. The starters also make `docker-compose.yml` files, and the `mailcap` package on Debian and Ubuntu installs `/usr/bin/compose`. |
| D2 | The top level is a starter. | `app` and `component` are too general. In a three-way merge, `base` is the common earlier version. `host` already means the host of the API URL in `go-cli-auth`. `template` is the word for each file in the catalog. |
| D3 | Features are in one shared folder, and each feature names its targets. | One feature works with many starters. An error names the feature and the starter. CI tests exactly the pairs that the features name. The cost is that a new starter needs an edit in each feature that targets it. |
| D4 | Each starter declares its own extension points. | The catalog needs no global list. The check compares the item fields of each extension point across the targets of a feature. |
| D5 | Features have no language. Folders have commands. | A feature can add Go code, Svelte code, YAML, and Markdown. The commands and the delimiters depend on the folder, not on the feature. |
| D6 | `api-auth` owns `method`, and `cli-auth` reads it. | The server decides the method. The CLI and the API cannot use different methods. |
| D7 | The catalog has no `excludes`. | Two features that do the same job become one feature with an option. |
| D8 | The catalog is a folder that the config file `~/.config/feature-compose/config.yml` names, as `catalog: <folder>`. | A catalog author uses a changed starter or feature at once, with no release of the tool. The cost is that the tool and the catalog have different versions, and the tool does not check that a catalog fits it. |
| D9 | `when` uses a small expression language of its own. | Names contain `-`. |
| D10 | The tool uses Go and `text/template`. | The starters are in Go, and `text/template` is in the standard library. |
| D11 | `add` comes in phase 3. | The three-way merge is the largest part of the tool. |
| D12 | The TUI uses huh. | Scaffold uses huh for the same job. Version 1.0.0 can hide a question, build a list from an earlier answer, and check text against a pattern. |

## 19. Open Questions

1. **Which issuer does `oauth` use?** An issuer that signs JWT access tokens works with `api-auth`, for
   example Auth0, Okta, Keycloak, Entra ID, or Zitadel. Google gives no JWT access token for your own API.
   So Google needs a session exchange, as in [stash](https://github.com/hpcsc/stash), and a third method.
2. **Must `cli-auth` work without `api-auth`?** `go-cli-auth` now supports a CLI that calls an API from
   another company. If `cli-auth` keeps this, it needs its own `method` option when the repo has no
   `api-auth`.
3. **How does a feature add a job to a CI workflow?** An item that holds YAML needs a function that sets its
   indent. The other choice is one workflow file for each feature.
4. **Is phase 3 necessary?** If you add features only when you make a repo, the tool needs no merge.
