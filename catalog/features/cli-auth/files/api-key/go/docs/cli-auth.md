{{- $stored := eq (option "store-key") "true" -}}
# The API Key of the CLI

`{{ kebab .Name }}` sends an API key to the API of this repo, in the header `Authorization: Bearer <key>`. The API
checks the key with the method `{{ option "api-auth.method" }}`, and the CLI follows the method of the API.
[api-auth.md](api-auth.md) tells how the API checks a key, and how to make a key and its hash.

## Give the CLI Its API Key

`{{ kebab .Name }}` uses the first of these sources that gives an API key:

1. The environment variable `{{ upper (snake .Name) }}_API_KEY`
{{- if $stored }}
2. The API key that `{{ kebab .Name }} auth login` stores
3. The output of `credentialCommand` in `$XDG_CONFIG_HOME/{{ kebab .Name }}/config.yaml`, or in
   `~/.config/{{ kebab .Name }}/config.yaml` when `XDG_CONFIG_HOME` is not set
{{- else }}
2. The output of `credentialCommand` in `$XDG_CONFIG_HOME/{{ kebab .Name }}/config.yaml`, or in
   `~/.config/{{ kebab .Name }}/config.yaml` when `XDG_CONFIG_HOME` is not set
{{- end }}

`{{ kebab .Name }} auth status` shows which source gives the API key. It does not show the API key.

Use the environment variable in CI and in scripts.
{{- if $stored }}

### Store the API Key

`{{ kebab .Name }} auth login` stores the API key, so that you give it one time and not on each call. In a terminal, it asks
for the key and does not show what you paste. Without a terminal, it reads the key from stdin:

```shell
{{ kebab .Name }} auth login                            # paste the key
printf '%s' "$KEY" | {{ kebab .Name }} auth login       # send the key on stdin
{{ kebab .Name }} auth status                           # show where the API key comes from
{{ kebab .Name }} auth logout                           # remove the stored API key
```

`auth login` keeps the key in the OS credential store: the macOS Keychain, the Windows Credential Manager, or the
Linux Secret Service. If the machine has no credential store, it keeps the key in `credentials.json`, in the same
folder as `config.yaml`. Only you can read that file. `{{ upper (snake .Name) }}_CREDENTIAL_STORE=file` makes `{{ kebab .Name }}` use only
`credentials.json`. The tests use it, so that they do not change your OS credential store.

The store keeps each key under the host of the API URL. Set the API URL before the first login:

| Constant | File | Environment variable | Value |
| --- | --- | --- | --- |
| `apiURL` | `internal/credential/api_url.go` | `{{ upper (snake .Name) }}_API_URL` | The base URL of the API, which ends with `/api`, for example `https://api.example.com/api`. |

The variable replaces the constant for one run, for example for a staging server. Without the API URL and without
`{{ upper (snake .Name) }}_API_KEY`, `{{ kebab .Name }}` stops with a message that names `apiURL` and `{{ upper (snake .Name) }}_API_URL`, also when `credentialCommand` is
set.
{{- end }}

### Read the API Key from a Password Manager

`credentialCommand` runs through `sh`, and `{{ kebab .Name }}` uses what the command prints. With it, the API key
stays in your password manager:

```yaml
credentialCommand: op read op://Private/{{ kebab .Name }}/api-key
```

`{{ kebab .Name }}` reads `credentialCommand` only from your own config file, and never from a file in a
repository.

## How the CLI Gets Its API Key

`credential.Resolve` gives the API key that `{{ kebab .Name }}` sends to its API. It tries a fixed list of sources
and takes the first that gives a value. The code is in `internal/credential`, the API client is in
`internal/apiclient`, and the `auth` command is in `internal/cli/auth.go`, in the package of the root command.

### Call the API from a Command

A command of the CLI calls the API through `apiclient.Client`. Before each request, the client calls
`credential.Resolve` and sends the key as `Authorization: Bearer <key>`.
{{- if $stored }} Give the client the API URL of
`credential.APIURL`, and give each call the path of the route after `/api`:

```go
apiURL, err := credential.APIURL()
if err != nil {
	return err
}
client := apiclient.New(apiURL.String())
{{- else }} Give the client the base URL of the
API, which ends with `/api`, and give each call the path of the route after `/api`:

```go
client := apiclient.New("http://localhost:3333/api")
{{- end }}
response, err := client.Do(ctx, http.MethodPost, "/<route>", body)
if err != nil {
	return err
}
defer response.Body.Close()
```

- With no API key, `Do` sends no request, and its error is `credential.ErrMissing` with the ways to give a key.
- When the API answers 401, `Do` returns no response. Its error is `credential.ErrRefused`, with the source of the
  refused key and the ways to give a key.
{{- if $stored }}
- The store keeps the key under the host of the API URL, so the client must call that host. A client with another
  URL sends the stored key to a server that it does not belong to.
{{- end }}
- `Do` gives each other answer to the command as the API gave it, for example 400 or 403.
- The client keeps no key. Each call resolves the key again, so a changed variable or password manager takes effect
  at the next call.

### The Files

| File | Job |
| --- | --- |
| `internal/credential/credential.go` | `Resolve`, which tries the sources in order. `Credential`, which holds the value and the name of its source. |
| `internal/credential/config.go` | Finds the user config, and reads `credentialCommand` from it. |
| `internal/credential/command.go` | Runs `credentialCommand` through `sh`. |
{{- if $stored }}
| `internal/credential/api_url.go` | `APIURL` and `APIHost`, from the constant `apiURL` or from `{{ upper (snake .Name) }}_API_URL`. |
| `internal/credential/store.go` | `Store`, which keeps a value in the OS credential store or in `credentials.json`. |
{{- end }}
| `internal/apiclient/client.go` | `Client`, which sends the API key on each call to the API, and turns the answer 401 into `credential.ErrRefused`. |
{{- if $stored }}
| `internal/cli/auth.go` | The `auth` commands: `status` prints the source, `login` stores the key, and `logout` removes it. |
{{- else }}
| `internal/cli/auth.go` | The `auth` command. It calls `credential.Resolve` and prints the source. |
{{- end }}

### The Order of the Sources

```mermaid
flowchart TD
    resolve["credential.Resolve"] --> variable{"Is {{ upper (snake .Name) }}_API_KEY set?"}
    variable -->|"yes"| fromVariable["The value of the variable"]
{{- if $stored }}
    variable -->|"no"| stored{"Does the store have a key for the host of the API URL?"}
    stored -->|"yes"| fromStore["The stored API key"]
    stored -->|"no"| command{"Does the user config set credentialCommand?"}
{{- else }}
    variable -->|"no"| command{"Does the user config set credentialCommand?"}
{{- end }}
    command -->|"yes"| fromCommand["What the command prints"]
    command -->|"no"| missing["ErrMissing, with the ways to give the API key"]
```

The order has these reasons:

- **The environment variable comes first.** CI and scripts set it for one run. The value for that run must
  replace a value that the user keeps for all runs.
{{- if $stored }}
- **The stored key comes before `credentialCommand`.** A user who runs `auth login` wants the CLI to use that key. A
  user who wants `credentialCommand` runs `auth logout` first.
{{- end }}
- **`credentialCommand` comes last.** It is for a user who keeps the API key in a password manager. It runs a
  program each time, so the CLI runs it only when no other source gives a value.

`Credential.Source` names the source: the name of the variable, or `the credentialCommand in <path>`. `auth status`
prints it. No command prints the API key itself.

### The Rules That Keep the API Key Safe

| Rule | Reason |
| --- | --- |
| The CLI has no flag for the API key. | The shell history keeps the value of a flag, and other users of the machine can see it in `ps`. Tests and CI use the environment variable. |
| The CLI reads `credentialCommand` only from the user config. | A file in a repository can come from another person. If that file can set a command, the CLI runs a command of that person on your machine. |
| `credentialCommand` writes its stderr to the terminal. | When a password manager fails, for example because it is locked, the user sees its message. |
{{- if $stored }}
| `auth login` reads the key from the terminal with no echo, or from stdin. | The key is not in the history of the shell, and the screen does not show it. |
| The store keeps each key under the host of the API URL. | If `{{ upper (snake .Name) }}_API_URL` names another server, the store has no key for it, so the CLI sends nothing to a server that the key does not belong to. |
| `credentials.json` has mode `0600`. | Only the user can read it. |
{{- else }}
| The CLI does not store the API key. | The key stays in the variable or in the password manager, and the CLI writes no file. |
{{- end }}

### Environment Variables

| Variable | Job |
| --- | --- |
| `{{ upper (snake .Name) }}_API_KEY` | Gives the API key. It comes before all other sources. |
{{- if $stored }}
| `{{ upper (snake .Name) }}_API_URL` | Replaces the constant `apiURL`. |
| `{{ upper (snake .Name) }}_CREDENTIAL_STORE` | `file` makes the store use only `credentials.json`. |
{{- end }}
| `XDG_CONFIG_HOME` | Holds the folder `{{ kebab .Name }}`, with `config.yaml`. If it is empty or not an absolute path, the folder is `~/.config/{{ kebab .Name }}`. |

### The Tests

The unit tests are in `internal/credential` and `internal/apiclient`. Run them with
`go test -tags=unit ./internal/credential ./internal/apiclient`. The tests of the client call a test server that
runs `auth.Middleware` of `internal/auth`, with the hashes of two keys of the test, so they check the client
against the check of the API of this repo.

{{ if $stored -}}
- Each test calls `isolate`. It sets `XDG_CONFIG_HOME` to a new temporary folder, clears `{{ upper (snake .Name) }}_API_KEY`, sets
  `{{ upper (snake .Name) }}_API_URL`, and sets `{{ upper (snake .Name) }}_CREDENTIAL_STORE` to `file`. So no test reads your config or changes your OS
  credential store.
- The tests of `Store` replace the OS credential store with `keyring.MockInit` and `keyring.MockInitWithError`. The
  mock fails each call or none, so no test shows a store that fails one call and works for another. When the OS
  credential store refuses a new key but keeps an older one for the same host, `Get` reads the older key first:
  run `auth logout` and then `auth login` again. `Delete` treats each error of the OS credential store as no stored
  key, so when the OS credential store is locked, `auth logout` can say that no key is stored while the store keeps
  it.
{{- else -}}
- Each test calls `isolate`. It sets `XDG_CONFIG_HOME` to a new temporary folder and clears
  `{{ upper (snake .Name) }}_API_KEY`. So no test reads your config.
{{- end }}

The e2e tests in `e2e/tests/auth.test.ts` run the binary with the same environment variables.
