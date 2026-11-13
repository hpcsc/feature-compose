# Tasks: The CLI Signs In with OAuth (US-008)

## Contents

1. Task 1: Add the files of the method `oauth` to `cli-auth`: `auth login` at the issuer for the audience of the API, and the tokens in the OS credential store
2. Task 2: Give each feature a combination with each other value of an option of its needs
3. Task 3: Let `cli-auth` follow the method of `api-auth`, and show that the API of the repo accepts the token that the CLI gets

## Story Reference

US-008, "The CLI signs in with OAuth", in `user-stories/feature-compose.md:313-336` (Description, Acceptance
Criteria, Context). The design is `PROPOSAL.md`: section 7 (the `feature.yml` of `cli-auth` with
`needs: [cli, api-auth]`, `files/oauth/` and `files/store/`, `PROPOSAL.md:194-216`), section 8 (an option changes
only the generated code, and the URL of an issuer and a client ID stay in the config of the repo,
`PROPOSAL.md:272-273`; the expression language of `when`, `PROPOSAL.md:275-293`), section 9
(`option "api-auth.method"`, `PROPOSAL.md:295-326`), section 15 (the CLI asks the issuer for a token for the
audience of the API, `PROPOSAL.md:493-517`) and section 19 (open questions 1 and 2, `PROPOSAL.md:562-569`). The
template to port is the preset `oauth` of `global/go-cli-auth` in `~/repo-templates/templates/global/go-cli-auth`.
US-007 (done) gave `api-auth` the method `oauth` and made the need of `cli-auth` set `method: api-key`
(`tasks/completed/us-007-oauth.md`, decision 7), so that a CLI of `api-key` never met an API of `oauth`.

## Boundaries

**Out of scope:**

The non-goals of the stories file, as written there:

- The command `add`, and an update of a repo to a new version of the catalog. Both need a three-way merge,
  which is phase 3.
- Generators that you run many times in one repo, for example `go-cli-sub-command`. They stay in scaffold.
- More than one catalog at a time. The config file names one catalog folder.
- A starter with more than one folder, for example a Go API in `api/` and a Svelte web app in `web/`. The
  feature `cli` shows G5 in one folder.
- A check of the credential in a repo without `api-auth`. The API of such a repo accepts each request.
- A third method of authentication for Google sign-in, with a session exchange.
- The templates that the proposal has not yet decided: `asdf-plugin`, `go-event-sourcing-cli`, `go-worker`,
  `global/git-hook`, and `global/kustomize`.
- The removal of `repo-templates`.
- A sync with `repo-templates`. The tool reads any catalog, and the catalog of this repo is one of them. A
  change in `repo-templates` comes into the catalog only through a story that gives the change its own
  value.

Added by this breakdown:

- A `when` on an item of `add_to`. The three items of `gomod.requires` of `cli-auth` are in each repo with
  `cli-auth`, and `go mod tidy` removes them from a repo of `api-key` (decision 3).
- A fixed version of `github.com/golang-jwt/jwt/v5`, which `api-auth` imports with `oauth`. `api-auth` has no item
  of `gomod.requires`, so `go mod tidy` takes the latest version. That is a change of `api-auth`.
- The parameter `resource` of RFC 8707, and an audience in the scopes, as Microsoft Entra ID
  (`api://<id>/.default`) and Zitadel ask for it. The docs of `oauth` tell a developer how to add a scope in the
  code (decision 5).
- A check of the scheme of the issuer URL in the CLI. The API checks `OAUTH_ISSUER_URL`; the preset checks
  nothing in the CLI, and this story ports the preset.
- Two processes of the CLI that refresh one refresh token at the same time. With an issuer that changes the
  refresh token at each refresh, one of the two asks the user to log in again, as in the preset.
- A client secret. The CLI is a public client with PKCE, as in the preset.
- An API client that takes its URL from the settings. `apiclient.New` keeps its argument, and the docs of `oauth`
  tell a command to give it `credential.APIURL()` (decision 7).
- A test against a real OS credential store. The unit tests use the mock of `go-keyring`, and each e2e test sets
  `MY_SERVICE_CREDENTIAL_STORE=file`, so no test changes the keychain of the person who runs it.
- A combination of each feature with each other value of an option of the starter. No starter of the catalog has
  an option with `values` (Task 2).
- A change of the answer of the API, `sign in to continue`, which `api-auth` owns. The CLI says `log in`, as its
  command `auth login` does (decision 12).
- A change of the CLI of `api-key`: its `auth` command keeps only `auth status`, and it stores nothing.

**Deferred:**

- The option `store-key`, the stored API key, and `auth login` and `auth logout` with `api-key`: US-013. Its
  context says "The CLI with oauth keeps its tokens in the same store (US-008), so the files of the store serve
  both methods". US-013 widens the `when` of `files/store/` to `api-auth.method == "oauth" || store-key`, as
  `PROPOSAL.md:210-211` writes it. This story gives `files/store/` the API URL too, which US-005 deferred to
  US-013 (decision 7), so US-013 reuses it.

## Codebase Context

**`cli-auth` today.** `catalog/features/cli-auth/feature.yml` needs `cli` with no value and `api-auth` with
`method: api-key`, has no option, the one files entry `files/common/`, and the item `newAuthCommand()` of
`cli.commands`. Its files are the preset `api-key` of `go-cli-auth`: `internal/credential/` (`credential.go` with
`Resolve`, `ErrMissing`, `ErrRefused` and `Credential.Refused`, `command.go`, `config.go`, `credential_test.go`),
`internal/cli/auth.go` (`auth status`), `internal/apiclient/` (`client.go`, and `client_test.go`, which calls
`auth.Middleware` of `api-auth` with the key hashes of `api-key`), `e2e/tests/auth.test.ts` and `docs/cli-auth.md`.
The texts `API key` and `MY_SERVICE_API_KEY` are in `credential.go`, `command.go`, `auth.go`, the tests and the
docs.

**What the catalog supports (point 1 of the request).** A need without values in the list form
`needs: [cli, api-auth]`, or as a key with no value in the map form (`internal/catalog/feature_yml.go:47-83`). A
files entry with `when` (`internal/catalog/feature.go:63-90`), whose names can be the options of the feature,
`<need>.<option>` and `starter.<option>` (`Selection.readable`, `internal/catalog/selection.go:333-341`), and which
`Selection.files` evaluates per template (`internal/catalog/selection.go:244`). Two entries of one feature whose
`when` never hold together can each write one path (US-007 Task 3), and the owner check of `check` counts one owner
per feature (`internal/catalog/starter.go:249-274`). `check` finds a `when` that does not parse or reads a name that
the feature cannot read (US-009). So criterion 1 needs no change of the tool: `cli-auth` drops its need value and
puts its files behind a `when` that reads `api-auth.method`. The catalog lacks two things: a `when` on an item of
`add_to` (an item is a map of text fields, and `extensionPoint.checkItem` refuses a field that the extension point
does not declare, `internal/catalog/extension_point.go:32-39`), and a combination of a feature with a value of an
option of its need (`Starter.combinations`, `internal/catalog/starter.go:145-173`). This breakdown does without the
first (decision 3), and Task 2 adds the second.

**Settings of a written repo today (point 2).** The API of the starter reads environment variables through
`api.Config` and env v6; `api-auth` with `oauth` reads `OAUTH_ISSUER_URL` and `OAUTH_AUDIENCE` that way
(`catalog/features/api-auth/files/oauth/go/internal/auth/config.go.tmpl`), and an operator changes them with no new
render. The CLI has no config of the repo: `cli` writes the name of the release repository as a constant of
`internal/cli/root.go`, and `cli-auth` reads only the user config `$XDG_CONFIG_HOME/<name>/config.yaml` for
`credentialCommand`. `go-cli-auth` keeps `issuerURL` and `clientID` as empty constants of
`internal/credential/issuer.go` and `apiURL` as an empty constant of `internal/credential/api_url.go`; the
variables `<PREFIX>_OAUTH_ISSUER_URL`, `<PREFIX>_OAUTH_CLIENT_ID` and `<PREFIX>_API_URL` replace them, and an empty
setting stops `auth login` with a message that names the constant and its variable.

**The keyring module (point 3).** The preset `oauth` imports `github.com/zalando/go-keyring` (`store.go`, and
`keyring.MockInit` in `store_test.go`), `golang.org/x/oauth2` (`issuer.go`, `login.go`, `tokens.go`) and
`github.com/cli/browser` (`login.go`). The source fixes no version: `go-cli-auth` has no `go.mod`, and its hook runs
`go mod tidy`, which takes the latest version at render time. On 2026-11-14, `go list -m -versions` gives
`github.com/zalando/go-keyring` `v0.2.8`, `golang.org/x/oauth2` `v0.38.0` (its `go.mod` says `go 1.26.0`, below
the `1.27.1` of the starter) and `github.com/cli/browser` `v1.3.0`. A repo gets a module of a feature with a fixed
version through an item of `gomod.requires` of `go-event-sourcing-service`, which `go.mod.tmpl` writes in its
first `require` block (`catalog/starters/go-event-sourcing-service/files/go/go.mod.tmpl:5-16`), as `cli` does for
its three modules (`catalog/features/cli/feature.yml:16-22`). A check in the scratchpad showed that
`go mod tidy` removes a requirement that no package imports. go-keyring needs `golang.org/x/sys` `v0.27.0` and
`cli/browser` needs `v0.13.0`, both below the `v0.43.0` of the starter, so in a repo of `api-key` the three items
raise no version of another module.

**The split by method (point 4).** Five files of `cli-auth` differ between the methods: `credential.go` (the
sources, the variable `MY_SERVICE_API_KEY` or `MY_SERVICE_TOKEN`, the fields `Account` and `Expiry`, the message
of a refused credential), `credential_test.go`, `internal/cli/auth.go` (with `oauth` also `login`, `logout` and
`token`), `client_test.go` (the config of `auth.Middleware` differs per method), `e2e/tests/auth.test.ts` and
`docs/cli-auth.md`. `command.go` holds the text `API key`. Only the files with the same text for both methods can
stay in `files/common/`. In `go-cli-auth`, `api_url.go`, `store.go` and `store_test.go` belong to each preset with a
store (`api-key-stored` and `oauth`), and `issuer.go`, `login.go`, `tokens.go`, their tests and
`fake_issuer_test.go` belong to `oauth` only (the `features` of its `scaffold.yml`). The precedent for the split and
its proof is `api-auth`: three entries (`catalog/features/api-auth/feature.yml:9-14`), a test of the files of each
method (`internal/catalog/catalog_test.go:1862-1893`), and a test that no file of `api-key` names an OAuth word
(`internal/catalog/catalog_test.go:1895-1903`).

**The proof (point 5).** `clerk mutants` does not change a `.go.tmpl`, so the template code of Task 1 gets its
mutation proof from breaks by hand on a written repo (`tasks/learnings.md:87-92`, `tasks/learnings.md:129-134`,
`tasks/learnings.md:367-372`); each task lists its breaks under **Verification**. The test issuer of US-007 is
`startedTestIssuer` in `e2e/tests/new.test.ts:856-880` (discovery and signing keys from node, RS256 tokens signed
with `node:crypto`) and `testIssuer` in `catalog/features/api-auth/files/oauth/go/internal/auth/test_issuer_test.go.tmpl`
(package `auth`, so only the tests of `internal/auth` can use it). The fake issuer of the preset
(`fake_issuer_test.go`, and `fakeIssuer` in `.go-cli-auth/auth.test.ts`) serves the authorization, device, token
and revocation endpoints, and gives opaque tokens `access-<n>` that no API can check. `startedApi` starts the API of
a written repo with no Postgres (`e2e/tests/new.test.ts:777-809`), and `builtCall` builds a program that calls the
API through `apiclient.Do` (`e2e/tests/new.test.ts:903-927`). Docker answered `docker info` on 2026-11-14, and the
e2e tests also run on the host (`tasks/learnings.md:59-64`).

**Tests that rely on the need value of `cli-auth`.** `e2e/tests/show.test.ts:16` (the needs of `cli-auth` in the
JSON of `show`); `e2e/tests/new.test.ts:342-352` (the stop of `cli-auth` with `oauth`), `e2e/tests/new.test.ts:432-440`
(`api-auth.method: api-key (need of cli-auth)`), `e2e/tests/new.test.ts:610-638` (the mark `fixed` of a need value
and the Space that `Select` refuses, in the form), `e2e/tests/new.test.ts:659`; `internal/catalog/catalog_test.go:1957-1964`,
`internal/catalog/catalog_test.go:2347-2363` and `internal/catalog/catalog_test.go:2365-2375`. The docs name the need
value of `cli-auth` in `docs/catalog.md:117-136`, `docs/catalog.md:156`, `docs/catalog.md:171-173`,
`docs/commands.md:85`, `docs/commands.md:113`, `docs/commands.md:126`, `docs/commands.md:190-192` and `README.md:71`.
`internal/form` tests the need values on a catalog of its own (`withNeedValue`, `internal/form/model_test.go:44-52`),
and `e2e/tests/check.test.ts:20-25` writes a catalog into a scratch folder with a config file that names it.

**Combinations.** `Starter.combinations` gives the starter alone, each other value of a starter option with
`values`, each feature alone with its defaults, each other value of an option with `values` of the feature, and each
pair of features that add to one extension point, except a pair in which one feature needs the other
(`internal/catalog/starter.go:145-173`, `internal/catalog/option.go:147-171`). So no combination makes a repo with
`cli-auth` and `api-auth:method=oauth`: `cli-auth` has no option, and it needs `api-auth`.
`internal/catalog/combination_test.go` makes the repo of each combination of the catalog of this repo and has the
build tag `combinations` (`tasks/learnings.md:290-295`). `docs/development.md:29-41` lists the combinations.

**Words.** `credential` is what a client sends after `Bearer`, and the package of `cli-auth` that resolves it
(US-007 and US-005). `key` is the API key of `api-key`. `issuer`, `discovery`, `audience`, `access token` and
`signing keys` are the words of `tasks/us-007-oauth.design.yaml`, and its test double is the `test issuer`
(`test_issuer_test.go.tmpl`, `startedTestIssuer`), so the port calls the fake issuer of the preset a test issuer.
`option` is a question of the catalog; the issuer URL, the client ID, the audience and the API URL are `settings`
of the written repo. `checkpoint.Store` of the starter is the store of the catchup checkpoints; `credential.Store`
of the preset reads with its package as the store of the credential. `combination` is the word of
`internal/catalog/combination.go`, and `needs` and `neededBy` are the words of `tasks/learnings.md:206-211`.

**Decisions of this breakdown, for the caller to overrule:**

1. **`cli-auth` follows the method through `when`, and its need sets no value** (Task 3). `needs` becomes
   `[cli, api-auth]`, the list form of section 7. The method is the one question of `api-auth`; `cli-auth` has no
   option. Each path below reads `api-auth.method` in a `when` or in a template.
2. **Four files entries** (Task 1): `files/common/` with no `when`, `files/api-key/` with
   `when: api-auth.method == "api-key"`, `files/oauth/` with `when: api-auth.method == "oauth"`, and `files/store/`
   with `when: api-auth.method == "oauth"`. `PROPOSAL.md:206-211` has no `files/api-key/`, but five files differ per
   method and would have two writers with `oauth`. `files/store/` holds `store.go`, `store_test.go` and `api_url.go`,
   as in the presets of `go-cli-auth`, and its files hold no text of one method, so that US-013 changes only its
   `when`.
3. **The three modules are items of `gomod.requires` in each repo with `cli-auth`** (Task 1):
   `golang.org/x/oauth2` `v0.38.0`, `github.com/zalando/go-keyring` `v0.2.8` and `github.com/cli/browser` `v1.3.0`.
   An item has no `when`, so a repo of `api-key` gets the three lines in the `go.mod` that the tool renders, and
   `go mod tidy`, an `after_render` command of the starter, removes them. The other choice is a `when` on an item of
   `add_to`: a new key of `feature.yml` that `Select`, `check`, the JSON of `show` and the docs must know, and that
   no story or section of the proposal names.
4. **The settings are constants of the repo with a variable each** (Task 1), the reading of criterion 4:
   `issuerURL`, `clientID` and `audience` in `internal/credential/issuer.go`, and `apiURL` in
   `internal/credential/api_url.go`, empty in the render. `MY_SERVICE_OAUTH_ISSUER_URL`,
   `MY_SERVICE_OAUTH_CLIENT_ID`, `MY_SERVICE_OAUTH_AUDIENCE` and `MY_SERVICE_API_URL` replace them for one run, for
   example for a staging issuer, the tests and the e2e tests. The reason: the CLI is a binary that its users
   install with `update` or `scripts/install.sh`, and a user has no repo, so the values must be in the binary. A
   constant needs no step in `go run`, `task build`, GoReleaser or the tests, and the preset does the same. A new
   issuer needs a change of a constant and a new build, and no new render. No option of `cli-auth` or `api-auth`
   gives a setting. The audience of the CLI must be the `OAUTH_AUDIENCE` of the API; the docs say so.
5. **The CLI sends the audience as the parameter `audience`** (Task 1), in the authorization request of the browser
   login and in the device authorization request, beside the scopes `openid profile email offline_access` of the
   preset. Auth0 needs this parameter. RFC 6749, section 3.1, says that an issuer must ignore a parameter that it
   does not know, so the parameter does no harm at Okta or Keycloak, which take the audience from the
   registration of the client. `resource` (RFC 8707) is not sent: some issuers refuse a value that is not an
   absolute URI. A refresh sends no audience; the refresh token keeps the audience of the login.
6. **`auth login` needs all three OAuth settings** (Task 1). With no issuer URL, client ID or audience, it sends no
   request, and the message names each missing constant and its variable. A refresh and `auth logout` need the
   issuer URL and the client ID, as in the preset.
7. **The API URL comes with the store** (Task 1). US-005 deferred `apiURL` to US-013, but the store of the preset
   keeps the tokens under the host of the API URL, so `files/store/` brings `api_url.go`. `apiclient.New(baseURL)`
   stays for both methods, and the docs of `oauth` tell a command to give it `credential.APIURL()`, so the client
   sends the stored token only to the host that it belongs to.
8. **The written repo checks the token against its own API** (Task 1), as US-005 decision 1 did for the key. Its
   test issuer serves discovery, the signing keys and the endpoints of the login, and signs RS256 JWT access tokens
   whose `aud` is the audience of the login request, with no `aud` for a request without one. A test sends the
   token that a login stores through `apiclient.Do` to a server whose handler goes through `auth.Middleware` of
   `api-auth` with the issuer URL and the audience of the test.
9. **The need of `cli-auth` keeps `method: api-key` until Task 3.** Task 1 adds the files of `oauth`, and its tests
   reach them through a copy of the catalog whose need of `cli-auth` sets no value
   (`internal/catalog/catalog_test.go:2347-2363`). So no state of the branch gives a CLI of `api-key` to an API of
   `oauth`, and the test and doc changes of the need value stay in Task 3.
10. **A combination for each other value of an option of a need** (Task 2). A `when` that reads an option of a need
    makes a variant of a feature that no combination makes today. Task 2 adds, for each feature, one combination
    with each other value of each option with `values` of each feature that it needs, except a value that its own
    need sets. Only direct needs count, because a feature reads only the options of its own needs (rule 5). In the
    catalog of this repo, this gives `go-event-sourcing-service --feature api-auth:method=oauth --feature cli-auth`
    after Task 3. The e2e test of Task 3 also makes that repo; the combination keeps CI right for the next feature
    whose files follow a need.
11. **The e2e tests of a need value move to a catalog in a scratch folder** (Task 3). After Task 3 no feature of
    the catalog of this repo has a need value, and `new` still has the source `need of`, the mark `fixed` and the
    stop of a flag against a need (US-007). The tests keep them on a starter and two features that the test
    writes, as `e2e/tests/check.test.ts:20-25` does.
12. **The CLI says `log in`**, as its commands `auth login` and `auth logout` and the messages of the preset do.
    The story says "signs in"; the API of `api-auth` answers `sign in to continue`, which this story does not change.
13. **The full run of the repo of `oauth` runs its own e2e tests** with `task test:e2e:local` (Task 3), as the full
    run of `cli` does (`e2e/tests/new.test.ts:116-170`), so CI runs the e2e tests of `auth login`, `auth token` and
    `auth logout` of the written repo.

**Open questions for the user:**

1. **Where do the issuer URL, the client ID and the audience of the CLI live?** Decision 4 reads "the config of the
   repo" as constants of `internal/credential` with a variable each, as the preset does. Another reading is a file
   of settings in the repo that the CLI embeds with `go:embed`, or values that `Taskfile.cli.yml` and
   `.goreleaser.yaml` give with `-ldflags -X`. The code of the repo does not settle this.
2. **Which parameter carries the audience?** Decision 5 sends `audience`. Microsoft Entra ID and Zitadel take the
   audience from a scope, and an issuer of RFC 8707 from `resource`. Say if the CLI must also send `resource`, or
   if the docs must name an issuer.
3. **Must an item of `add_to` take a `when`?** Decision 3 puts the three modules into the rendered `go.mod` of each
   repo with `cli-auth`, and lets `go mod tidy` remove them with `api-key`. A `when` on an item is the other choice,
   and it changes the format of `feature.yml`.

## Tasks

### Task 1: Add the files of the method `oauth` to `cli-auth`: `auth login` at the issuer for the audience of the API, and the tokens in the OS credential store

**Behavior:** The files of `cli-auth` are in four files entries (decision 2). With `api-key`, `cli-auth` writes the
files that its tests in `TestCatalog` list, and the CLI is as before. With `oauth`, the CLI is the preset `oauth`
of `global/go-cli-auth`, ported with sentinel values, with one change: each login asks the issuer for a token for
the audience of the API (decision 5). `auth login` logs the user in at the issuer in a browser with PKCE, or with a
device code, and keeps the tokens in the OS credential store under the host of the API URL, or in
`credentials.json` when the OS credential store fails. Each call of the CLI to its API takes `MY_SERVICE_TOKEN`,
else the stored tokens, refreshed when the access token expires, else the output of `credentialCommand`.
`auth status` shows the source, the account and the expiry; `auth token` prints the access token; `auth logout`
revokes the refresh token and removes the tokens. The issuer URL, the client ID, the audience and the API URL are
settings of the repo (decision 4). `cli-auth` adds three items to `gomod.requires` (decision 3). The need of
`cli-auth` still sets `method: api-key` (decision 9), so the tests reach the files of `oauth` through a copy of the
catalog whose need of `cli-auth` sets no value.

**Acceptance Criteria:**
- [x] `catalog/features/cli-auth/feature.yml` has the four files entries of decision 2 with their `when`, the need
  of `api-auth` with `method: api-key`, the item `newAuthCommand()` of `cli.commands`, and the items of
  `gomod.requires` `golang.org/x/oauth2` `v0.38.0`, `github.com/zalando/go-keyring` `v0.2.8` and
  `github.com/cli/browser` `v1.3.0`. `task test:catalog` finds no mistake.
- [x] With `api-key`, the files that `cli-auth` adds are `internal/credential/credential.go`, `command.go`,
  `config.go`, `credential_test.go`, `internal/cli/auth.go`, `e2e/tests/auth.test.ts`, `docs/cli-auth.md`,
  `internal/apiclient/client.go` and `client_test.go`, and the only files of the starter, `cli` and `api-auth` that
  change are `internal/cli/root.go` and `go.mod` (`internal/catalog/catalog_test.go:2207-2231`). The other tests of
  `cli-auth` in `TestCatalog` pass with their expected values unchanged.
- [x] With `api-key`, no file that `cli-auth` writes holds `issuer`, `audience`, `oauth`, `refresh`, `keyring`,
  `credential store` or `auth login` (in any case), and no Go file that it writes imports `golang.org/x/oauth2`,
  `github.com/zalando/go-keyring` or `github.com/cli/browser`.
- [x] The e2e full run of `cli-auth` with `api-key` (`e2e/tests/new.test.ts:354-407`) also checks that the
  `go.mod` of the repo, after `new`, requires none of the three modules.
- [x] With `oauth`, from a copy of the catalog whose need of `cli-auth` sets no value and with
  `api-auth:method=oauth`, the files that `cli-auth` adds are the files of `files/common/`, `files/oauth/` and
  `files/store/`, none of `files/api-key/`, and the only files of the starter, `cli` and `api-auth` that change are
  `internal/cli/root.go` and `go.mod`.
- [x] With each method, no written file holds `.Scaffold.`, `.Computed.`, `go-cli-project`, `GO_CLI_PROJECT`,
  `zzsentinel` or `<no value>`. The Go files of `cli-auth` import only its own packages and `cli`, and only a test
  imports `internal/auth` of `api-auth` (rule 6). The `go.mod` that the selection writes requires a version of each
  module outside the standard library that a Go file of `cli-auth` imports.
- [x] With `oauth`, no file that `cli-auth` writes holds a value of the issuer URL, the client ID, the audience or
  the API URL: each is an empty constant in `internal/credential`, and `MY_SERVICE_OAUTH_ISSUER_URL`,
  `MY_SERVICE_OAUTH_CLIENT_ID`, `MY_SERVICE_OAUTH_AUDIENCE` and `MY_SERVICE_API_URL` replace them. Neither `cli-auth`
  nor `api-auth` has an option for them.
- [x] A repo of `oauth` that the built binary writes from that copy of the catalog, with
  `--feature cli-auth --feature api-auth:method=oauth --no-prompt`, passes `gofmt -l .` with no file,
  `go build ./...`, and `task test:unit` with `internal/credential`, `internal/apiclient`, `internal/auth` and
  `internal/cli` among the packages that passed. Its unit tests run against a test issuer (decision 8) and show:
  - **login:** the browser login sends an S256 challenge and the verifier, and checks the state; another state, an
    error of the issuer and an answer with no code stop the login; it prints the URL also when the browser does
    not open. The device login prints the verification URL and the user code, and an issuer with no device
    authorization endpoint stops it. Each login request carries the scopes `openid profile email offline_access`
    and the audience of the setting. With no issuer URL, client ID or audience, `auth login` sends no request, and
    the message names each missing constant and its variable. A discovery that names another issuer stops the
    login.
  - **resolve:** `MY_SERVICE_TOKEN` comes first, then the stored tokens of the host of the API URL, then the output
    of `credentialCommand`. Tokens stored for another host are not given. With no source, the error is
    `ErrMissing`, and it names `my-service auth login`, `MY_SERVICE_TOKEN` and `credentialCommand`.
  - **refresh:** an access token that expires in less than 10 seconds gets a new one with the refresh token; the
    new tokens are stored, and the next resolve uses them with no second refresh. When the issuer sends no new
    refresh token, the CLI keeps the old one, and with no new ID token it keeps the account. When the issuer
    refuses the refresh, or the stored tokens have no refresh token, the error tells the user to run
    `my-service auth login`.
  - **store:** with the OS credential store (the mock of `go-keyring`), the tokens are there and no file is written;
    when the OS credential store fails, `credentials.json` holds them with the mode `0600`;
    `MY_SERVICE_CREDENTIAL_STORE=file` uses only the file, and another value is an error; each host has its own
    value; a delete removes the value from both places; a set while the OS credential store works removes the copy
    in the file.
  - **logout:** it revokes the refresh token, or the access token when there is no refresh token, and removes the
    tokens; when the revocation fails, it writes a warning and removes the tokens; with no stored tokens, it
    reports that it removed nothing.
  - **API client:** the access token that a login stores reaches the handler behind `auth.Middleware` of
    `api-auth`, with the issuer URL and the audience of the test, through `apiclient.Do`, and so does the access
    token of a refresh. With an API whose audience is another value, the call fails, and the message names the
    source of the token and `my-service auth login`. With no token, no request reaches the API.
- [x] With `oauth`, the root command lists `auth` after `version` and `update`, and `auth` has `status`, `login`
  (with the flag `--device`), `logout` and `token`. `auth status` prints the source, the account and the expiry, and
  never the token. The CLI has no flag for a token or a setting.
- [x] With `oauth`, `e2e/tests/auth.test.ts` of the written repo runs the built CLI against a test issuer in node,
  with `MY_SERVICE_CREDENTIAL_STORE=file`: `auth status` names `MY_SERVICE_TOKEN` and not the token;
  `auth login --device` prints the code and stores the tokens in a file with the mode `0600`; the login in a
  browser that `BROWSER` names sends the PKCE verifier; `auth token` after the expiry prints a new access token;
  `auth logout` revokes the refresh token; the test issuer gets the audience of `MY_SERVICE_OAUTH_AUDIENCE` with
  each login. `task test:e2e:local` of the written repo passes on the host with `tests/auth.test.ts` among the
  files that passed.
- [x] With `oauth`, `docs/cli-auth.md` tells a user how to log in in a browser and with `--device`, what
  `auth status`, `auth token` and `auth logout` do, where the tokens are, and the order of `MY_SERVICE_TOKEN`, the
  stored tokens and `credentialCommand`. It tells a developer the four settings with their constants and variables,
  that the audience must be the `OAUTH_AUDIENCE` of the API, how to register the client at the issuer (a public
  client, the redirect URL `http://127.0.0.1/callback` with any port, and the grants of the authorization code, the
  refresh token and the device code), the parameter `audience` and the scope for an issuer that takes the audience
  from a scope, how a command gives the API client the API URL of `credential.APIURL`, the refresh, the
  store, and the tests. It links [api-auth.md](api-auth.md) and names only files that the repo has.
- [x] `docs/cli-auth.md` of each method names that method, `api-key` or `oauth`. With `api-key`, the test of the
  docs of `cli-auth` (`internal/catalog/catalog_test.go:2279-2295`) passes with its expected values unchanged.
- [x] `go build ./...`, `go vet ./...` and `task test:unit` pass, and the e2e tests of `new` pass on the host.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — the four files entries and the items of `gomod.requires`
- `catalog/features/cli-auth/files/common/go/` — the files with the same text for both methods
- `catalog/features/cli-auth/files/api-key/go/` — the files of `api-key`, from `files/common/`
- `catalog/features/cli-auth/files/oauth/go/internal/credential/` — new: `credential.go`, `issuer.go`, `login.go`,
  `tokens.go`, their tests and the test issuer; each `.go` file has the `.tmpl` suffix
- `catalog/features/cli-auth/files/oauth/go/internal/cli/auth.go.tmpl` — new: the `auth` command of `oauth`
- `catalog/features/cli-auth/files/oauth/go/internal/apiclient/client_test.go.tmpl` — new: the client against
  `auth.Middleware` of `oauth`
- `catalog/features/cli-auth/files/oauth/go/e2e/tests/auth.test.ts` — new: the e2e tests of the written repo
- `catalog/features/cli-auth/files/oauth/go/docs/cli-auth.md` — new
- `catalog/features/cli-auth/files/store/go/internal/credential/` — new: `store.go`, `store_test.go`, `api_url.go`
- `internal/catalog/catalog_test.go` — the files, the words, the imports, `go.mod`, the settings and the docs of each
  method
- `e2e/tests/new.test.ts` — `go.mod` of the repo of `api-key`

**Patterns to Follow:**
- `catalog/features/api-auth/feature.yml:9-14` for files entries with `when`
- `catalog/features/cli/feature.yml:16-22` for the items of `gomod.requires`
- `internal/catalog/catalog_test.go:1862-1903` for the files of each method and the words that `api-key` must not hold
- `internal/catalog/catalog_test.go:2207-2295` for the files, the imports, the values of scaffold and the docs of
  `cli-auth`
- `internal/catalog/catalog_test.go:2347-2363` for a copy of the catalog with a changed `feature.yml`
- `internal/catalog/catalog_test.go:2463-2493` for a `go.mod` that requires each imported module
- `internal/catalog/catalog_test.go:1905-1919` for a setting with no value in the written files
- `catalog/features/cli-auth/files/api-key/go/internal/apiclient/client_test.go.tmpl:35-56` for a client test against
  `auth.Middleware`
- `catalog/features/api-auth/files/oauth/go/internal/auth/test_issuer_test.go.tmpl:1-80` for a test issuer that
  publishes signing keys and signs RS256 tokens
- `catalog/features/api-auth/files/oauth/go/internal/auth/config.go.tmpl:1-6` for the config of `auth.Middleware`
  with `oauth`
- `catalog/features/cli-auth/files/api-key/go/internal/credential/credential_test.go.tmpl:14-20` for the isolation of
  a test from the config and the variables of the person who runs it
- `e2e/tests/new.test.ts:354-407` for the full run of `cli-auth` and the checks of its `go.mod`
- `tasks/learnings.md:24-29` for the port with sentinel values
- `tasks/learnings.md:10-15` for `.tmpl`
- `tasks/learnings.md:157-162` for a Go composite literal with `{{` in a template
- `tasks/learnings.md:122-127` for a test that reads an environment variable
- `tasks/learnings.md:381-386` for snake case packages and a fake beside the code that it fakes
- `tasks/learnings.md:276-281` for the rules of time of a feature that keeps what it reads from an outside service
- `tasks/learnings.md:87-92`, `tasks/learnings.md:129-134` and `tasks/learnings.md:367-372` for the breaks by hand
- `tasks/learnings.md:94-99` and `tasks/learnings.md:374-379` for the cost of a port, which is its proof
- `tasks/learnings.md:31-36` for the old defects of the preset that an audit raises
- `tasks/learnings.md:38-43`, `tasks/learnings.md:52-57` and `tasks/learnings.md:59-64` for the runs of `go` and
  `task` in a written repo, on the host

**Testable:** Yes

**Certainty:** medium — the port of the preset `api-key` (`catalog/features/cli-auth/`, US-005 Task 4) and the split
of `api-auth` by method (`catalog/features/api-auth/feature.yml:9-14`) are the precedents; the variation is the
first `when` that reads an option of a need, the audience that the preset does not send, and a test issuer that
signs tokens that `auth.Middleware` of the written repo checks

**Blast radius:** high — credential and token handling: the CLI gets, refreshes, stores and revokes the tokens of
the user, and sends the access token to the API

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, and `task test:catalog` finds no
mistake. Copy `catalog/` to a scratch folder, drop `method: api-key` from the need of `cli-auth` there, and point a
config file of the tool at it. Write a repo with each method with the built binary, and run `gofmt -l .`,
`go build ./...`, `task test:unit` and `task test:e2e:local` there. In the repo of `api-key`, read the `go.mod`
after `new`: it names none of the three modules, and each module has the version that it has in the repo written
from that copy with the three items removed. In the repo of `oauth`, apply one break at a time to the written files,
read its diff, run `go test -tags unit ./internal/credential/ ./internal/apiclient/`, read the name of the failed
test, and copy the file back. Write each break as a condition, so that the code still builds: the browser login
sends no audience; the device login sends no audience; `auth login` with no audience sends its request; the
callback accepts another state; the exchange sends no PKCE verifier; resolve uses an expired access token; a
refresh that fails gives the old token; the refreshed tokens are not stored; a discovery with another issuer is
accepted; `MY_SERVICE_TOKEN` comes after the stored tokens; the store writes the file while the OS credential store
works; `credentials.json` gets the mode `0644`; the store gives the value of another host; `auth logout` does not
revoke; `auth logout` keeps the tokens when the revocation fails; the API client gives the answer 401 to the
caller. Each break makes a test fail. Break `auth status` so that it prints the token, and run
`task test:e2e:local`: a test of `auth.test.ts` fails. On macOS, run `auth login --device` of the repo of `oauth`
against a test issuer without `MY_SERVICE_CREDENTIAL_STORE`, read the item `my-service` with
`security find-generic-password -s my-service`, run `auth logout`, and read that the item is gone. Run the e2e
tests of `new` on the host.

**Depends on:** None

---

### Task 2: Give each feature a combination with each other value of an option of its needs

**Behavior:** `Catalog.Combinations` gives, for each feature, one combination for each other value of each option
with `values` of each feature that it needs (decision 10). The combination holds the feature with the defaults of
its options, and the need with that value; each other option with `values` of the need has the value that the need
of the feature sets, else its default. A value of an option that the need of the feature sets gives no such
combination. A need of a need gives none. The name of the combination is the arguments of `new` that make the same
repo, so `task test:combinations` makes the repo of each of them with no change to the task or to CI.

**Acceptance Criteria:**
- [x] In a catalog where `client` needs `auth`, and `auth` has `method` with `values: [api-key, oauth, none]` and
  `default: api-key`, `Catalog.Combinations` holds `api --feature auth:method=oauth --feature client` and
  `api --feature auth:method=none --feature client`, beside the combinations of `client` alone and of `auth` with
  each value. The test states the whole list in its order.
- [x] When the need of `client` sets `method: api-key`, no combination varies `method` with `client`. When the need
  sets one option of `auth` and `auth` has a second option with `values`, each combination of `client` with another
  value of the second option holds the first option at the value of the need.
- [x] A need with no option with `values`, a feature with no need, and a need of a need give no combination of this
  kind. The tests of the group `combinations` of `TestCatalog` pass with their expected values unchanged.
- [x] `Select` of each such combination does not stop, and its repo holds the values that its name gives (the
  pattern of `internal/catalog/catalog_test.go:2921-2947`).
- [x] In the catalog of this repo, no combination holds `cli-auth` with `api-auth:method=oauth`, because the need of
  `cli-auth` sets `method: api-key`.
- [x] `docs/development.md`, in "Test the Combinations of the Catalog", lists the combination of a feature with each
  other value of an option of a feature that it needs, and the values that the need sets.
- [x] `go build ./...`, `go vet ./...`, `go vet -tags combinations ./internal/catalog/` and `task test:unit` pass.

**Affected Files/Modules:**
- `internal/catalog/starter.go` — `combinations`
- `internal/catalog/catalog_test.go` — the group `combinations` of `TestCatalog`
- `docs/development.md` — the list of the combinations

**Patterns to Follow:**
- `internal/catalog/starter.go:145-173` for the combinations of a feature alone, of its option values, and of a pair
- `internal/catalog/option.go:147-171` for the defaults and the other values of the options with `values`
- `internal/catalog/feature.go:124-148` for the values that the need of a feature sets
- `internal/catalog/combination.go:9-21` for the name of a combination
- `internal/catalog/catalog_test.go:2858-2990` for the tests of the combinations and of their repos
- `docs/development.md:29-41` for the list of the combinations
- `tasks/learnings.md:136-141` for a loop over a map in the order of its keys
- `tasks/learnings.md:290-295` for `go vet -tags combinations`
- `tasks/learnings.md:311-316` for a rule about one feature on `feature`
- `tasks/learnings.md:185-190` for the tests in the package of the code

**Testable:** Yes

**Certainty:** high — `Starter.combinations` (`internal/catalog/starter.go:145-173`) already gives each other value of
an option of a feature and each pair of two features; this combination is the pair of a feature and its need, with
the other values of the need from `options.otherValuesOfOptionsWithValues`

**Blast radius:** low — none of the named categories; it adds repos that CI makes and tests

**Verification:** `task test:unit`, `go build ./...`, `go vet ./...` and `go vet -tags combinations ./internal/catalog/`
pass. `clerk mutants` finds no live mutant on the changed lines of `internal/catalog`.

**Depends on:** None

---

### Task 3: Let `cli-auth` follow the method of `api-auth`, and show that the API of the repo accepts the token that the CLI gets

**Behavior:** The need of `cli-auth` sets no value (decision 1), so `--feature cli-auth --feature api-auth:method=oauth`
makes a repo whose CLI logs in at the issuer, and the API of that repo accepts the token that the CLI gets. The
tool asks for the method one time, as the option of `api-auth`; `cli-auth` has no question. An answer of the form
for `method` changes the files of `cli-auth` as a flag does. The tests and the docs that used the need value of
`cli-auth` as their example use the default of `method`, or a catalog in a scratch folder (decision 11). An e2e
test makes the repo of `oauth`, logs in with the built CLI at a test issuer, calls the running API through the API
client, and gets the answer of the handler.

**Acceptance Criteria:**
- [ ] `catalog/features/cli-auth/feature.yml` has `needs: [cli, api-auth]` and a `summary` that names the API key
  and the access token of a login at the issuer.
- [ ] `new go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=oauth --dry-run` ends with
  the exit status 0 and prints `api-auth: command line, needed by cli-auth`, `cli: needed by cli-auth`,
  `cli-auth: command line` and `api-auth.method: oauth (command line)`, the files of `files/oauth/` and
  `files/store/` of `cli-auth` with the owner `the feature cli-auth`, and none of `files/api-key/`. With
  `--feature cli-auth` alone, the preview prints `api-auth.method: api-key (default)` and the files of `api-key`.
- [ ] `Selection.Questions` of a selection of the catalog of this repo with `cli-auth`, with each method, holds no
  question of `cli-auth`, and `method` of `api-auth` is its only question of a feature. With `--feature cli-auth`
  and the answer `oauth` for `method` of `api-auth`, the selection writes the files of `oauth` of `cli-auth`, and the
  preview prints `api-auth.method: oauth (answered)`.
- [ ] `Catalog.Combinations` of the catalog of this repo holds
  `go-event-sourcing-service --feature api-auth:method=oauth --feature cli-auth`, and `task test:combinations`
  passes.
- [ ] `show --output json` prints `features["cli-auth"].needs` as `{"cli": {}, "api-auth": {}}`
  (`e2e/tests/show.test.ts:16`).
- [ ] The tests of `TestCatalog` that used the need value of `cli-auth` (`internal/catalog/catalog_test.go:1957-1964`
  and `internal/catalog/catalog_test.go:2365-2375`) check the selection of `cli-auth` with `oauth` and the source
  `default` of `api-auth.method`.
- [ ] The e2e tests of a need value use a catalog in a scratch folder with a starter, a feature with the option
  `method`, and a feature whose need sets `method: api-key`: `--dry-run` prints the source `need of <feature>`; a
  flag with another value stops `new` with `--no-prompt` and with `--dry-run`, names both sources, and makes no
  folder; in the form, the option shows `fixed`, Enter starts no edit and names the need, and a Space at the feature
  with a flag of another value leaves it `[ ]` and shows the message that names both sources.
- [ ] The e2e tests of the form on the catalog of this repo: with `--feature cli-auth`, the row of `method` of
  `api-auth` shows `method  api-key` with no mark `fixed`, Enter lists `api-key` and `oauth`, and after `oauth` the
  preview of `w` prints `api-auth.method: oauth (answered)` and a file of `files/oauth/` of `cli-auth`. With
  `--feature api-auth:method=oauth`, a Space at `cli-auth` selects it and `cli`. The preview test of
  `e2e/tests/new.test.ts:640-665` expects `api-auth.method: api-key (default)`.
- [ ] An e2e full run, `new go-event-sourcing-service my-service --feature cli-auth --feature api-auth:method=oauth --no-prompt`:
  - ends with the exit status 0, prints `cli: needed by cli-auth` before the line that names the output folder, and
    writes `.feature-compose.yml` with `api-auth` at `method: oauth`;
  - gives a repo in which `gofmt -l .` lists no file, `go build ./...` builds, and `task test:unit` passes with
    `internal/credential`, `internal/apiclient`, `internal/auth` and `internal/cli` among the packages that passed;
  - gives a `go.mod` that requires each module of the items of `gomod.requires` of `cli-auth` at the version of the
    item, read from the JSON of `show`;
  - passes `task test:e2e:local` of the repo, with `tests/auth.test.ts` among the files that passed;
  - starts a test issuer on `127.0.0.1` that serves discovery, the signing keys, the device authorization endpoint
    and the token endpoint, and signs RS256 access tokens whose `aud` is the audience of the device authorization
    request; and starts the API with `OAUTH_ISSUER_URL` of that issuer and `OAUTH_AUDIENCE=my-api`;
  - runs `auth login --device` of the built CLI with `MY_SERVICE_OAUTH_ISSUER_URL`, `MY_SERVICE_OAUTH_CLIENT_ID`,
    `MY_SERVICE_OAUTH_AUDIENCE=my-api`, `MY_SERVICE_API_URL` of the API, `MY_SERVICE_CREDENTIAL_STORE=file` and an
    empty config folder: it ends with 0, and the test issuer got the audience `my-api`;
  - calls `POST /api/accounts` with `{}` through `apiclient.Do` (`builtCall`) and gets the answer 400 of the handler,
    so the API accepted the token that the CLI got;
  - after the access token expires, a second call refreshes it at the test issuer, and the API accepts the new
    token; when the test issuer refuses the refresh, the call fails, and the message tells the user to run
    `my-service auth login`;
  - with `MY_SERVICE_OAUTH_AUDIENCE=another-api` and another empty config folder, the same binary logs in, and the
    call fails with a message that names `my-service auth login`: the API refused a token for another audience, and
    the CLI changed its audience with no new render.
- [ ] `docs/catalog.md`: the `feature.yml` of `cli-auth` shows `needs: [cli, api-auth]` and the files entries that read
  `api-auth.method`; the text of needs and of the sources of an option names a need value with an example that is
  not a feature of the catalog of this repo, and says that `cli-auth` follows `api-auth.method` through `when`.
  `docs/commands.md`: the preview prints `api-auth.method: api-key (default)`, and the examples of `need of`, of the
  stop of a flag against a need, and of the Space that the form refuses name no need of `cli-auth`. `README.md`: the
  diagram shows `cli-auth` needs `api-auth` with no value, and the table of the features says that `cli-auth` sends
  the API key or the access token of a login. `docs/e2e-tests.md` tells the full run of `oauth` with its test issuer,
  and the tests of a need value on a catalog in a scratch folder. No doc says that the need of `cli-auth` sets a
  value.
- [ ] `go build ./...`, `go vet ./...` and `task test:unit` pass, `task test:catalog` finds no mistake, and the e2e
  tests of `new`, `show` and `check` pass on the host.

**Affected Files/Modules:**
- `catalog/features/cli-auth/feature.yml` — the need of `api-auth` with no value, and the summary
- `internal/catalog/catalog_test.go` — the selection of `cli-auth` with `oauth`, its questions, the source of
  `api-auth.method`, and the combination of the catalog of this repo
- `e2e/tests/new.test.ts` — the full run of `oauth` with a test issuer, the preview of `oauth`, the tests of a need
  value on a catalog in a scratch folder, and the tests of the form
- `e2e/tests/show.test.ts` — the needs of `cli-auth`
- `docs/catalog.md`, `docs/commands.md`, `README.md`, `docs/e2e-tests.md` — the need of `cli-auth`, and the tests

**Patterns to Follow:**
- `task:1` for the files of `oauth` and their settings
- `task:2` for the combination of `cli-auth` with `api-auth:method=oauth`
- `internal/catalog/catalog_test.go:1957-1964` and `internal/catalog/catalog_test.go:2365-2375` for the tests of the
  need of `cli-auth` on the catalog of this repo
- `internal/catalog/selection_test.go:1452-1520` for the tests of `Selection.Questions`
- `e2e/tests/check.test.ts:20-25` for a catalog in a scratch folder and a config file that names it
- `internal/form/model_test.go:44-52` for a catalog with a need value
- `e2e/tests/new.test.ts:247-288` for a repo of `oauth` whose API an e2e test starts with a test issuer
- `e2e/tests/new.test.ts:354-407` for the full run of `cli-auth`, its CLI and its call through `apiclient.Do`
- `e2e/tests/new.test.ts:116-170` for the items of `gomod.requires` from the JSON of `show`, and
  `task test:e2e:local` of a written repo
- `e2e/tests/new.test.ts:856-880` for the test issuer in node
- `e2e/tests/new.test.ts:903-927` for `builtCall`
- `e2e/tests/new.test.ts:549-665` for the tests of the form
- `docs/e2e-tests.md:80-235` for the docs of the full runs
- `tasks/learnings.md:59-64` for the e2e tests on the host
- `tasks/learnings.md:108-113` for an e2e test that starts the API without Postgres
- `tasks/learnings.md:38-43` and `tasks/learnings.md:52-57` for `go` and `task` in a written repo
- `tasks/learnings.md:220-225` and `tasks/learnings.md:227-232` for the tests of `new` in a pseudo-terminal
- `tasks/learnings.md:346-351` for the j presses of the e2e tests of the form
- `tasks/learnings.md:353-358` for a failure of the scheduler test of the starter
- `tasks/learnings.md:199-204` for the source of each item, also with a second source

**Testable:** Yes

**Certainty:** medium — US-007 changed the need of `cli-auth` and its tests and docs, and the e2e test of the API of
`oauth` starts a test issuer (`e2e/tests/new.test.ts:247-288`); the variation is a test issuer that also serves the
device login and signs a token for the audience that the CLI sends, and the e2e tests of a need value on a catalog
that the test writes

**Blast radius:** high — authentication: it decides which method the CLI of a written repo uses with each method of
the API, and it removes the stop that kept a CLI of `api-key` away from an API of `oauth`

**Verification:** `task test:unit`, `go build ./...` and `go vet ./...` pass, and `task test:catalog` finds no
mistake. Run `go vet -tags combinations ./internal/catalog/` and `task test:combinations`. Run
`e2e/tests/new.test.ts`, `e2e/tests/show.test.ts` and `e2e/tests/check.test.ts` on the host, or `task test:e2e` when
`timeout 20 docker info` answers. Apply one break at a time to the catalog, read its diff, run the full run of
`oauth`, and copy the file back: the device login of `files/oauth/` sends no audience (the call gets 401); the need
of `cli-auth` sets `method: api-key` again (the full run stops at `new`). Each break makes the e2e test fail.

**Depends on:** Task 1, Task 2

## Summary

- **Tasks:** 3.
- **Order:** risk first, then dependency. Task 1 is the port of the preset `oauth` with the audience, the riskiest
  task and the one whose proof costs most: a written repo from a copy of the catalog, its unit and e2e tests, and
  sixteen breaks by hand. It keeps the need value, so no state of the branch gives a CLI of `api-key` to an API of
  `oauth`. Task 2 is a small change of the tool, with no dependency, and it is the only task that `clerk mutants`
  checks. Task 3 drops the need value, which makes the files of Task 1 and the combination of Task 2 reachable in the
  catalog of this repo, moves the e2e tests of a need value to a catalog that the test writes, and proves the story
  end to end against the running API.
- **Assessment:** Task 1 and Task 3 have a high blast radius: Task 1 handles the tokens of the user, and Task 3
  decides which method of authentication the CLI of each repo uses. Task 2 is high certainty, because it composes
  two loops that `Starter.combinations` already has. No task is low certainty: the story ports a preset with a
  precedent for each part (the port of `api-key` in US-005, the split by method and the test issuer in US-007).
  What stays open is a decision for the user, not a design that a task must find: where the settings live
  (decision 4) and which parameter carries the audience (decision 5).
- **Criteria of the story:**

| Criterion of US-008 | Tasks |
| --- | --- |
| `--feature cli-auth --feature api-auth:method=oauth` makes a repo, and the tool asks no question about the method for `cli-auth` | 3 (the need with no value, the preview, `Selection.Questions`, the form, the full run) |
| `auth login` signs the user in at the issuer and asks for a token for the audience of the API; the API of the same repo accepts the token that the CLI gets | 1 (the login, the audience, the client against `auth.Middleware` in the written repo), 3 (the built CLI against the running API) |
| The CLI keeps the tokens in the OS credential store and refreshes the access token when it expires; the user signs in again only when the refresh fails | 1 (the store, the refresh and its failure), 3 (the refresh against the running API) |
| The URL of the issuer, the client ID and the audience come from the config of the repo, not from options | 1 (the constants and their variables, no option), 3 (another audience with no new render) |
| A repo with `method=api-key` has no OAuth code in the CLI | 1 (the files of each method, the words and the imports of `api-key`, and its `go.mod` after `new`) |
| Context: the preset `oauth` of `global/go-cli-auth`, with the audience | 1 |

- **Deferred:** none of the criteria of this story. The store of the API key and `store-key` go to US-013, which
  names the store of this story in its context.
- **Criteria that this breakdown adds, for the caller to overrule:** the files entry `files/api-key/` (decision 2);
  the three items of `gomod.requires` in each repo with `cli-auth` (decision 3); the API URL with the store
  (decision 7); the combination of a feature with the values of an option of its need (Task 2, decision 10); the
  e2e tests of a need value on a catalog in a scratch folder (decision 11); the e2e tests of the written repo of
  `oauth` in the full run (decision 13).
- **Notes:**
  - Files that Task 1 ports from the preset `oauth` with sentinel values, for the brief of the audit
    (`tasks/learnings.md:31-36`): `issuer.go`, `login.go`, `tokens.go`, `store.go`, `api_url.go`, `credential.go`
    and their tests, `fake_issuer_test.go` as the test issuer, `.go-cli-auth/auth.go` as `internal/cli/auth.go`,
    `.go-cli-auth/auth.test.ts`, and `docs/authentication.md` with `.go-cli-auth/README.md.part` as
    `docs/cli-auth.md`. New code: the audience, its setting, the signed tokens of the test issuer, the client test
    against `auth.Middleware`, and the message of a refused token.
  - The full run of `oauth` adds about three minutes to the e2e tests, with `npm install` and the e2e tests of the
    written repo. `task test:combinations` gets one more repo.
