# API

Go API workspace for the site.

## Development

Run the API from the repository root:

```bash
pnpm api:dev
```

The server listens on `:8080` by default. Override it with `API_ADDR`:

```bash
API_ADDR=:8081 pnpm api:dev
```

Admin routes require a bearer token configured with `API_ADMIN_TOKEN`:

```bash
API_ADMIN_TOKEN=change-me pnpm api:dev
```

The API stores data in `.data/api.json` by default. Override it with `API_DATA_PATH`.
In production, point `API_DATA_PATH` to a persistent volume, for example `/data/my-space-api.json`.

CORS allows `http://localhost:3000` by default. Override it with `API_ALLOW_ORIGIN`.

## Container Image

Build the API image from the repository root:

```bash
pnpm api:docker:build
```

The Dockerfile expects the repository root as build context:

```bash
docker build -f apps/api/Dockerfile -t my-space-api .
```

For deployment, configure these environment variables in the API host:

- `API_ADMIN_TOKEN`: bearer token used by Postman for admin routes.
- `API_ALLOW_ORIGIN`: public frontend URL allowed by CORS.
- `API_ADDR`: server address, optional. Defaults to `:8080`.
- `API_DATA_PATH`: JSON database file path. Defaults to `.data/api.json`.
- `SPOTIFY_CLIENT_ID`: Spotify app client ID, optional.
- `SPOTIFY_CLIENT_SECRET`: Spotify app client secret, optional.
- `SPOTIFY_REFRESH_TOKEN`: Spotify refresh token with `user-read-currently-playing user-top-read`, optional.
- `SPOTIFY_MARKET`: Spotify market code. Defaults to `BR`.

When Spotify variables are configured, the API uses Spotify for the `currently.listening` value, includes the current track metadata and progress, and exposes top tracks for the About page. If Spotify is unavailable, the API falls back to the JSON `currently.listening` value.

## Routes

- `GET /healthz` returns `{ "status": "ok" }`.
- `GET /widgets/home` returns currently, latest approved signatures, and visitor count.
- `GET /status/currently` returns listening, online status, and current Spotify track metadata when configured.
- `GET /spotify/top-tracks?limit=5` returns top Spotify tracks when configured.
- `PUT /admin/status/currently` updates the currently widget.
- `POST /visits` increments the visitor counter.
- `GET /stats/visitors` returns the visitor counter.
- `POST /guestbook/signatures` creates a pending guestbook signature.
- `GET /guestbook/signatures` lists approved signatures.
- `GET /guestbook/signatures/latest?limit=3` lists the latest approved signatures for widgets.
- `GET /admin/guestbook/signatures/pending` lists pending signatures.
- `PATCH /admin/guestbook/signatures/{id}/approve` approves a signature.
- `PATCH /admin/guestbook/signatures/{id}/reject` rejects a signature.
