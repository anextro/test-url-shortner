# URL Shortener

Create permanent short links with a code you choose. Opening a short link redirects to the original URL.

- **Service** (`service/`): Go HTTP API backed by Postgres, with an in-memory cache for redirects.
- **Client** (`client/`): React web app for creating links.

## Run it locally

Requires Docker.

```bash
make up      # starts Postgres, the service (:8080) and the web client (:3000)
make down    # stops everything and removes the data
```

Open http://localhost:3000 to use the web client.

## API

Base URL: `http://localhost:8080`

### Create a short link

`POST /links`

```json
{ "url": "https://example.com", "code": "abc123" }
```

| Status | Meaning |
|--------|---------|
| 201 | Link created |
| 200 | This URL already has this code; the existing link is returned |

Response:

```json
{ "shortUrl": "t/abc123" }
```

### Follow a short link

`GET /{code}`

| Status | Meaning |
|--------|---------|
| 301 | Redirect; the `Location` header holds the original URL |
| 404 | Unknown code |

Codes are case-insensitive: `/ABC123` and `/abc123` are the same link.

### Rules

| Field | Rule |
|-------|------|
| `code` | Required. 6–10 characters: `a-z`, `0-9`, `-`, `_`. Some words are reserved (e.g. `health`, `status`, `signin`). |
| `url` | `http` or `https`, at most 2048 characters, not a link to this service itself. |
| Rate limit | 10 creates per IP per minute. |

Links are permanent: they cannot be edited or deleted.

### Errors

Every error has this shape. Only the first problem found is reported.

```json
{ "error": { "code": "CODE_TAKEN", "message": "..." } }
```

| HTTP | `error.code` | When |
|------|--------------|------|
| 400 | `INVALID_URL` | URL is not http/https, too long, or malformed |
| 400 | `SELF_REFERENCE` | URL points to this service |
| 400 | `INVALID_CODE` | Code has the wrong length or characters |
| 400 | `RESERVED_CODE` | Code is a reserved word |
| 404 | `NOT_FOUND` | Unknown code on redirect |
| 409 | `CODE_TAKEN` | Code already points to a different URL |
| 429 | `RATE_LIMITED` | More than 10 creates per minute from your IP |
| 503 | `UNAVAILABLE` | Service or storage problem; safe to retry |

Retrying a create is safe: sending the same URL and code again returns the existing link.

## Examples

Create a link:

```bash
curl -i -X POST http://localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com","code":"abc123"}'
```

Follow it:

```bash
curl -i http://localhost:8080/abc123
```

## Tests

```bash
make test-service   # API tests, concurrency, load (needs Docker)
make test-client    # web client tests against a mock API
make test           # everything, including an end-to-end browser test
```

## Configuration (service)

| Variable | Default | Meaning |
|----------|---------|---------|
| `DATABASE_URL` | — | Postgres connection string (required) |
| `PORT` | `8080` | Port to listen on |
