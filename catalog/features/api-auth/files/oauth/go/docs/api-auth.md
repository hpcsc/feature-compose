# OAuth access tokens

The API accepts a request to a protected route only when the request carries an access token that the issuer
of the API signs. `GET /` is public, so a health check needs no token.

## The issuer

The issuer must sign JWT access tokens and publish OpenID Connect discovery at
`<issuer URL>/.well-known/openid-configuration`. Most OAuth servers do both, for example Auth0, Okta, Keycloak
and Microsoft Entra ID. Google gives no JWT access token for your own API, so Google cannot be the issuer.

Register the API at the issuer, and give it an audience: the value that the `aud` claim of its access tokens
holds.

## Give the issuer and the audience to the API

The API reads two environment variables when it starts:

| Variable | Value |
| --- | --- |
| `OAUTH_ISSUER_URL` | The URL of the issuer, for example `https://issuer.example.com/`. It must be the same text as the `issuer` of its discovery and the `iss` claim of its access tokens. |
| `OAUTH_AUDIENCE` | The audience of the API at the issuer, for example `https://api.example.com`. |

- With no `OAUTH_ISSUER_URL` or no `OAUTH_AUDIENCE`, the API starts and answers 401 to each request to a
  protected route.
- An `OAUTH_ISSUER_URL` that is not an absolute `https` URL stops the API at start. An `http` URL is accepted
  only for a loopback host, for example `http://127.0.0.1:8080`, for an issuer on this machine.

To change the issuer or the audience, change the variables and restart the API. The code does not change, and
the repo needs no new render.

## Run the API on this machine

`task run` in `README.md` starts the API with `task api:run`, which sets neither variable. So that API refuses
each request to a protected route. To start the API with your issuer, stop the pane of `task api:run` and run
it with the variables:

```shell
OAUTH_ISSUER_URL=<url> OAUTH_AUDIENCE=<audience> task api:run
```

`task api:run` passes the variables of your shell to the API.

The requests in `Taskfile.api.yml`, such as `task api:open-account` in `README.md`, send no token, so the API
answers them with 401. Send the token with curl, as the next section shows.

`.env.local` and `docker-compose.local.yml` set neither variable either. To run the `api` service of
`docker-compose.local.yml` with your issuer, add the two variables to its `environment`.

## Send a request with an access token

A client gets an access token from the issuer, for the audience of the API, and sends it in the
`Authorization` header with the scheme `Bearer`:

```shell
curl -X POST http://localhost:3333/accounts \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d '{"owner": "Jane"}'
```

## The checks of an access token

The API accepts an access token only when each of these is true:

- It is a JWT, and the algorithm in its header is in the allowlist. The allowlist is `RS256`, the default of
  the issuers above. To accept `RS384` or `RS512` too, add it to `allowlist` in `internal/auth/middleware.go`.
  The API reads only the RSA keys of the issuer, so an algorithm of another key type, for example `ES256`,
  also needs a change to `internal/auth/issuer.go`.
- One of the signing keys that the issuer publishes signs it. The `kid` in its header names the key.
- Its `iss` claim is the issuer URL.
- Its `aud` claim holds the audience, as a text or in a list.
- Its `exp` claim is after now. A token with no `exp` gets 401.

The API reads discovery and the signing keys when it checks the first token, and not when it starts. So the API
starts also when the issuer does not answer. Then it reads the signing keys again:

- one hour after the last read. The first request after that hour still gets its key, and the API reads the
  signing keys while it answers. A key that the issuer removed gets 401 after that read. While the issuer does
  not answer, the API keeps the signing keys that it has.
- when a token names a key that the API does not have, at most one time in 5 minutes.
- after a read that fails, not before 10 seconds have passed.

The API makes one read of the issuer at a time, and a request with a known key never waits for that read. Each
call to the issuer stops after 10 seconds.

## The answers of the API

| Request to a protected route | Answer |
| --- | --- |
| No `Authorization` header, a scheme other than `Bearer`, or no token | 401 with `WWW-Authenticate: Bearer` |
| A token that fails a check above, or a token when the API cannot read the issuer | 401 with `WWW-Authenticate: Bearer` |
| `Authorization: Bearer <token>` for a token that passes each check | The answer of the route |

Each answer 401 has the JSON body of the other errors of the API, with a message that tells the client to
sign in:

```json
{"successful": false, "messages": ["sign in to continue"]}
```

The scheme `Bearer` matches in any case. With the method `{{ option "method" }}`, each access token that the
API accepts has every permission, so the API never answers 403.
