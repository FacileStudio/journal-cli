---
name: journal
description: >
  Facile centralized logging CLI. Use when the user asks to read, search, or
  follow application logs from the suite, or mentions Journal, log lines,
  errors, or a live tail.
---

# journal — Facile centralized logging

Binary: `journal`
Config: `<config_dir>/journal/config.json` (instance URL + session token)

Journal is the suite's centralized logging service. Every Facile app ships
structured log entries to one instance at `https://journal.facile.studio`; this
CLI reads, filters, and follows them without the dashboard.

## When to apply

Use when the user mentions logs, errors, a request id, "what did app X log",
deploy-time debugging, or a live tail of a suite service.
Triggers: "logs", "log", "error", "tail", "journal", "what happened", "request
id", "deployed but", "is it logging"

## Commands

### Setup
```
journal login [url]                 Authenticate (browser under SSO, or password)
journal logout                      Revoke the stored session
```

### Reading logs
```
journal apps                        Which apps have logged and how recently
journal logs [filters]              Query newest-first, one page
journal tail [filters]              Follow new entries as they land
journal context <id> [--before 50] [--after 50]   Stream around one entry
```

### log filters (logs and tail)
```
--app <name>        Source app, exact match
--level a,b,c       error,warn,info,debug
--q <text>          Full-text search
--request-id <id>   Exact match on meta request_id
--since 30m|2h|RFC3339   Lower bound (relative or absolute)
--until RFC3339     Upper bound
--limit N           Max entries (logs only), default 100
```

## Rules
- A session is required; if none is stored the CLI says so. Run `journal login`
  (browser flow) once, or set `JOURNAL_TOKEN` in a headless/CI context.
- Output is newest first for `logs`, chronological for `tail` and `context`.
- The entry id is the anchor for `context` — `journal logs --json` returns ids.
- `--json` prints one document (logs/apps/context) or one doc per line (tail),
  forcing colour off and leaving colour rules to the consumer.
- Level severity is colour-coded by default; `--no-color` disables it.
- `JOURNAL_URL` overrides the stored instance; `--url` overrides both.
- Exit: `0` success, `1` failure, `2` usage, `130` SIGINT.
- Prefer `journal logs --q <term> --since 30m` to answer "what happened" —
  it is the cheapest probe and returns ids to pivot with `context`.