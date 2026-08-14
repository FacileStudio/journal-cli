# journal-cli

Terminal client for [Journal](https://github.com/FacileStudio/Journal), the suite's
centralized logging service. Go, cobra, one binary named `journal`.

Lets a shell (or an AI agent) read, filter, and follow the log without opening
the dashboard. Authenticates through the same porte SSO flow the browser uses.

## What it does

- `journal login` — browser SSO flow (one-time code via loopback) or password
- `journal logs` — query newest-first: app, level, full text, request id, time
- `journal tail` — follow new entries as they land (2s poll, the dashboard's own cadence)
- `journal context <id>` — the stream around one entry, with the anchor marked
- `journal apps` — what has logged, and how recently
- `--json` on every command carrying data; ndjson lines from `tail`

## Stack

| Layer | Tech |
|---|---|
| CLI | Go 1.25, cobra, session token in a 0600 config file |
| Auth | porte SSO CLI flow (`?flow=cli` → loopback → `/auth/oidc/exchange`), or password |
| Releases | GoReleaser via facile |

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/FacileStudio/journal-cli/main/install.sh | bash
```

Already have `facile`:

```sh
facile install journal
```

## Setup

```sh
journal login                    # opens the browser, SSO when the instance offers it
journal login --no-browser       # print the URL instead
journal login --email you@x --password …   # password flow, headless
```

The URL defaults to `https://journal.facile.studio`, the suite's shared
instance; `journal login <url>` stores another. `JOURNAL_URL` overrides the
stored URL, `--url` overrides both.

## Usage

```sh
journal apps                          # what is logging, and when last
journal logs --level error --since 30m
journal logs --app sablier --q "upload" --request-id 7f3a
journal tail --app courrier --level error
journal context 48213                 # what led to that entry
journal logs --limit 500 --json | jq '.[].id'   # page through up to the limit
journal logs --level error --before-ts 2026-08-13T00:00:03Z --before-id 48213   # resume a cursor
```

## Rules of the road

- Output newest first for `logs`, chronological for `tail`/`context`.
- Data on stdout, status on stderr — a piped command emits only data.
- `--json` forces colour off; `--no-color` disables it otherwise.
- Exit codes: 0 success, 1 failure, 2 usage, 130 SIGINT.
- Errors are lowercase, name what failed, and end with the fix after an em dash.

## Development

```sh
mise run build      # go build -o bin/journal .
mise run check      # gofmt + vet + test
mise run format     # rewrite Go sources in place
```
