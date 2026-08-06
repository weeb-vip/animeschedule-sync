# animeschedule-sync

Pulls airing schedules from the [AnimeSchedule.net](https://animeschedule.net)
API and writes them into the weeb.vip MySQL database.

One of the ingest services behind the site: it keeps the "what is airing, and
when" side of the data current, so the front end can show a schedule without
calling a third-party API on every request.

## What it writes

Four tables, each with its own repository under `internal/db/repositories`:

| Repository | Holds |
|---|---|
| `anime` | the series themselves |
| `anime_schedule` | broadcast schedule entries |
| `anime_streaming` | where a series can be streamed |
| `episode_air_time` | per-episode air times |

The API client is `internal/services/animeschedule`; the mapping from its
responses onto those tables is `internal/services/sync`.

## Running it

Requires Go 1.23.

```sh
make sync     # one-shot full sync from the API, then exit
make serve    # run the sync on a recurring interval
make build    # build ./animeschedule-sync
```

Both are Cobra subcommands of the built binary, so
`go run cmd/main.go --help` lists them too.

## Configuration

Config is read by [configor](https://github.com/jinzhu/configor) from
`config/config.<env>.json`, with environment variables overriding it — see
`config/config.go` for the shape. `config.dev.json` is the local example, and
holds the database connection and the AnimeSchedule API details.

Requests to the upstream API are rate limited (`golang.org/x/time/rate`), so a
full sync is deliberately unhurried.

## Building

```sh
docker build -t animeschedule-sync .
```

`.github/workflows/build.yaml` builds and publishes the image on merge.
