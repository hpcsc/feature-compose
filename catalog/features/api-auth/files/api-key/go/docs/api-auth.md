# API keys

The API accepts a request to a protected route only when the request carries a key that the API knows.
`GET /api/` is public, so a health check needs no key.

## Make a key and its hash

A key is a random secret. The API keeps only the SHA-256 hash of each key, and never the key.

```shell
key=$(openssl rand -hex 32)
printf '%s' "$key" | shasum -a 256 | cut -d ' ' -f 1
```

Give the key to the client, and keep the hash for the API. On Linux, `sha256sum` gives the same hash as
`shasum -a 256`.

## Give the key hashes to the API

The API reads the key hashes from the environment variable `API_KEY_HASHES` when it starts. The value is the
hex form of the SHA-256 hash of each key, with a comma between two hashes:

```shell
API_KEY_HASHES=<hash of the first key>,<hash of the second key>
```

- With no `API_KEY_HASHES`, the API starts and answers 401 to each request to a protected route.
- An entry that is not the hex form of a SHA-256 hash, which has 64 hex digits, stops the API at start. The
  message gives the number of the entry, and not its text.

To add or remove a key, change `API_KEY_HASHES` and restart the API. The code does not change.

## Run the API on this machine

`task run` in `README.md` starts the API with `task api:run`, which sets no `API_KEY_HASHES`. So that API
refuses each request to a protected route. To start the API with your key hashes, stop the pane of
`task api:run` and run it with the variable:

```shell
API_KEY_HASHES=<hash of your key> task api:run
```

`task api:run` passes the variables of your shell to the API.

The requests in `Taskfile.api.yml`, such as `task api:open-account` in `README.md`, send no key, so the API
answers them with 401. Send the key with curl, as the next section shows.

`.env.local` and `docker-compose.local.yml` set no `API_KEY_HASHES` either. To run the `api` service of
`docker-compose.local.yml` with your key hashes, add `API_KEY_HASHES` to its `environment`.

## Send a request with a key

A client sends the key in the `Authorization` header, with the scheme `Bearer`:

```shell
curl -X POST http://localhost:3333/api/accounts \
  -H "Authorization: Bearer $key" \
  -H 'Content-Type: application/json' \
  -d '{"owner": "Jane"}'
```

## The answers of the API

| Request to a protected route | Answer |
| --- | --- |
| No `Authorization` header, a scheme other than `Bearer`, or no key | 401 with `WWW-Authenticate: Bearer` |
| A key whose hash is not in `API_KEY_HASHES` | 401 with `WWW-Authenticate: Bearer` |
| `Authorization: Bearer <key>` for a key whose hash is in `API_KEY_HASHES` | The answer of the route |

Each answer 401 has the JSON body of the other errors of the API, with a message that tells the client to
sign in:

```json
{"successful": false, "messages": ["sign in to continue"]}
```

The scheme `Bearer` matches in any case. With the method `{{ option "method" }}`, each key that the API
knows has every permission, so the API never answers 403.
