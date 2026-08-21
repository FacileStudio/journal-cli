# journal-cli

Terminal client for a [Journal](https://github.com/FacileStudio/Journal) instance. Go, cobra, one
binary named `journal`.

## Commands

| Task | Command |
|---|---|
| Build | `mise run build` |
| Quality gate | `mise run check` |
| Format Go | `mise run format` |
| Enable git hooks | `mise run hooks` |

## Structure

```
main.go            hands off to cmd
cmd/               one file per command; root.go owns flags and exit codes,
                   format.go the rendering shared between them
internal/
  client/          the HTTP surface
  config/          the instance URL and the session token, as YAML at
                   ${XDG_CONFIG_HOME:-~/.config}/journal/config.yml
  loopback/        the porte SSO one-time-code listener
  ui/              CLI-STANDARD §4 output vocabulary, copied verbatim
integrations/      SKILL.md, the AI agent registration
install.sh         generated from Wiki/install.sh.template; only the config block differs
scripts/check.sh   the quality gate, copied from antenne-cli
```

Dependencies are cobra, `fatih/color`, `golang.org/x/term` and `gopkg.in/yaml.v3`.
The last one is there because CLI-STANDARD §6.1 mandates YAML for the config file;
the standard is normative and Go has no YAML in its standard library. Adding a
fifth needs a reason — a client for one API does not need a framework.

## Conventions

These come from `Wiki/CLI-STANDARD.md`, which is normative. When this repo
disagrees with it, this repo is wrong.

- **`Short` and flag help: capitalized, imperative, no trailing period.** `"List what the
  instance is seeing"`, never `"Lists what the instance is seeing."`.
- **No emoji, anywhere.** Not in help, not at runtime.
- **All output through `internal/ui`.** `▸` step, `✓` success, `!` warning, `✗` error, and
  hints indented two spaces. Warnings and errors go to stderr.
- **Data on stdout, everything else on stderr**, so a piped command emits only data.
- **`--json` on every command carrying data**, printing one document and nothing else (one
  document per line for `tail`, which is a stream). It forces colour off.
- **Exit codes**: `0` success, `1` failure, `2` usage, `130` SIGINT. `root.go` maps them;
  `commandStarted` is what distinguishes a usage error from a failed one.
- Errors are lowercase, name what failed, and end with the fix after an em dash — the glyph
  is added by the printer, never baked into the message.
- `--version` prints exactly `journal <semver>` — the installer parses that line.

## The client is read-only

The read endpoints (`/api/logs`, `/api/logs/{id}/context`, `/api/apps`) are all this CLI
talks to after login. Ingest is deliberately out of scope: the Go SDK (`sdk/journal`), the
browser SDK, and `curl` cover shipping, and a CLI adds nothing there.

## The login flow

`journal login` asks the instance what it accepts (`GET /api/auth/config`), then either:

- **SSO**: binds a loopback listener, opens `…/api/auth/oidc?flow=cli&port=N&cli_state=S`,
  waits for the redirect with the one-time code, and trades it at
  `/api/auth/oidc/exchange` — the porte CLI flow casier-cli uses. The state nonce is
  echoed and verified, so a callback from a different login is refused.
- **Password**: prompts for an address and a password, exchanged at `/api/auth/login`.

Both end with a bearer token stored in `<config_dir>/journal/config.yml` (0600), and both
logout paths revoke it server-side.

## Gotchas

- Journal mounts porte under `/api`, so every path carries the prefix — the login URL is
  `/api/auth/oidc`, not `/auth/oidc`. `internal/loopback` builds it; the test locks the
  shape.
- There is no SSE endpoint: `tail` polls `/api/logs` every two seconds with a
  newest-first page and keeps entries with a higher id. The dashboard does the same.
- A bearer token is exempt from the porte CSRF rule that a cookie-authenticated mutating
  request needs `X-Facile-CSRF`. The CLI sends only reads, and the bearer header is
  attached by the client, so the two can never collide.
- `JOURNAL_TOKEN` in the environment wins over the stored token for read commands — the
  headless/CI path, and how an agent with its own token avoids touching the config file.
