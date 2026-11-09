# The API Key of the CLI

`{{ kebab .Name }}` sends an API key to the API of this repo, in the header `Authorization: Bearer <key>`. The API
checks the key with the method `{{ option "api-auth.method" }}`, and the CLI follows the method of the API.
[api-auth.md](api-auth.md) tells how the API checks a key, and how to make a key and its hash.

## Give the CLI Its API Key

`{{ kebab .Name }}` uses the first of these sources that gives an API key:

1. The environment variable `{{ upper (snake .Name) }}_API_KEY`
2. The output of `credentialCommand` in `$XDG_CONFIG_HOME/{{ kebab .Name }}/config.yaml`, or in
   `~/.config/{{ kebab .Name }}/config.yaml` when `XDG_CONFIG_HOME` is not set

`{{ kebab .Name }} auth status` shows which source gives the API key. It does not show the API key.

Use the environment variable in CI and in scripts.

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
and takes the first that gives a value. The code is in `internal/credential`, and the `auth` command is in
`internal/cli/auth.go`, in the package of the root command.

### Use the API Key in an API Client

Call `credential.Resolve` before each request. Send `found.Value` in the header that the API asks for:

```go
found, err := credential.Resolve(ctx)
if err != nil {
	return err
}
request.Header.Set("Authorization", "Bearer "+found.Value)
```

### The Files

| File | Job |
| --- | --- |
| `internal/credential/credential.go` | `Resolve`, which tries the sources in order. `Credential`, which holds the value and the name of its source. |
| `internal/credential/config.go` | Finds the user config, and reads `credentialCommand` from it. |
| `internal/credential/command.go` | Runs `credentialCommand` through `sh`. |
| `internal/cli/auth.go` | The `auth` command. It calls `credential.Resolve` and prints the source. |

### The Order of the Sources

```mermaid
flowchart TD
    resolve["credential.Resolve"] --> variable{"Is {{ upper (snake .Name) }}_API_KEY set?"}
    variable -->|"yes"| fromVariable["The value of the variable"]
    variable -->|"no"| command{"Does the user config set credentialCommand?"}
    command -->|"yes"| fromCommand["What the command prints"]
    command -->|"no"| missing["ErrMissing, with the ways to give the API key"]
```

The order has these reasons:

- **The environment variable comes first.** CI and scripts set it for one run. The value for that run must
  replace a value that the user keeps for all runs.
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
| The CLI does not store the API key. | The key stays in the variable or in the password manager, and the CLI writes no file. |

### Environment Variables

| Variable | Job |
| --- | --- |
| `{{ upper (snake .Name) }}_API_KEY` | Gives the API key. It comes before all other sources. |
| `XDG_CONFIG_HOME` | Holds the folder `{{ kebab .Name }}`, with `config.yaml`. If it is empty or not an absolute path, the folder is `~/.config/{{ kebab .Name }}`. |

### The Tests

The unit tests are in `internal/credential`. Run them with `go test -tags=unit ./internal/credential`.

- Each test calls `isolate`. It sets `XDG_CONFIG_HOME` to a new temporary folder and clears
  `{{ upper (snake .Name) }}_API_KEY`. So no test reads your config.

The e2e tests in `e2e/tests/auth.test.ts` run the binary with the same environment variables.
