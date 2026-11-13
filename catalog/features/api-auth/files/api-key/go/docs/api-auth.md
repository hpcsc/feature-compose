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

`task api:run` starts the API on port 3333, and passes the variables of your shell to the API. Give it the
hashes of your keys in the variable:

```shell
API_KEY_HASHES=<hash of your key> task api:run
```

With no `API_KEY_HASHES`, the API refuses each request to a protected route. Each other way to start the API
that `README.md` tells, for example in Docker, also needs `API_KEY_HASHES` in the environment of the API.

## Send a request with a key

A client sends the key in the `Authorization` header, with the scheme `Bearer`. Each route of the API is under
`/api`, so the path of a request is `/api/<route>`:

```shell
curl http://localhost:3333/api/<route> \
  -H "Authorization: Bearer $key"
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
