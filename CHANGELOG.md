# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Every entry below was reconstructed from git history on 2026-08-24, so they
record what shipped rather than what was written down at the time.

## [Unreleased]

## [0.3.0] — 2026-09-01

### Added

- `journal keys` command group (`list`, `create`, `revoke`) to manage secret backend and public browser ingest API keys from the terminal.
- Application filter flag `--app` on `journal keys list`.
- Browser public key creation flags (`--public`, `--origins`, `--quota`) on `journal keys create`.
- Confirmation skip flag `--yes` / `-y` on `journal keys revoke`.

## [0.2.1] — 2026-08-24

Release plumbing only. No change to the binary's behaviour.

### Fixed

- The Homebrew formula now publishes. The `brews` block resolves
  `{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}` and the repo holds the secret, but
  `release.yml` never mapped it into the goreleaser step, so the run would
  publish the GitHub release in full and then die on a template error. The
  default `GITHUB_TOKEN` cannot push to `homebrew-tap`, which is why the
  separate token exists.

### Added

- `release.yml` publishes a Homebrew formula to `FacileStudio/homebrew-tap` on
  a tag.

### Changed

- Hooks come from the shared lefthook config in `FacileStudio/hooks`.

### Removed

- The legacy tracked githooks, replaced by that shared config.

## [0.2.0] — 2026-08-21

### Changed

- The config moved from `config.json` to `config.yml`, at
  `${XDG_CONFIG_HOME:-~/.config}/journal/`, which is what CLI-STANDARD §6.1
  mandates for new tools. There is deliberately no fallback to the old file: an
  existing `config.json` is ignored and has to be deleted by hand, and it still
  holds a session token that nothing will clear. Run `journal login` again.
- `JOURNAL_SERVER_URL` is the canonical environment variable, which is the name
  §6.3 gives it. `JOURNAL_URL` stays an accepted alias, and the canonical
  spelling wins when both are set.

### Added

- Tighten-on-read for the config file. One found with any group or other bit set
  is repaired to 0600 before it is read, through the open handle rather than a
  separate stat and chmod that would leave a window between them. `Save` creates
  the file at 0600.

### Fixed

- `logout` clears the credential even when the config cannot be parsed.
  `Clear()` called `Load()` first, so a malformed config aborted the whole
  command with a raw parser error and left the token exactly where somebody had
  just tried to delete it. The URL is unrecoverable in that case, so it resets to
  the default.
- `Save()` re-asserts the file mode on the open handle. `OpenFile`'s perm
  argument applies only at creation, so an existing file kept whatever mode it
  had.

## [0.1.0] — 2026-08-14

### Added

- First release. A read-side terminal client for Journal over its existing
  session-authenticated API, with no server changes: `logs`, `tail`, `context`
  and `apps` against `/api/logs`, `/context` and `/apps`.
- Sign-in through porte's loopback SSO flow, with a password path for instances
  without SSO.
- `tail` polls at the dashboard's 2 second cadence, because no SSE endpoint
  exists.
- `--json` on every data command, and ndjson lines from `tail`.
- An agent-facing `SKILL.md` under `integrations/`, and `facile install journal`.
- GoReleaser config and a GitHub Actions workflow, producing the archive layout
  facile's installer expects on a `v*` tag. The tag round-trips into
  `cmd.version` through ldflags.
- `logs` pages the keyset cursor until it has gathered `--limit` entries, and
  takes `--before-ts` and `--before-id` to resume from a cursor. It previously
  decoded `next_before` and threw it away, so a request for 500 entries returned
  100 and no way to page.

### Changed

- `tail`'s follow loop was doing four jobs at once. `collectFresh` and
  `retryPoll` are split out, leaving the loop to own error handling and
  rendering.

[Unreleased]: https://github.com/FacileStudio/journal-cli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/FacileStudio/journal-cli/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/FacileStudio/journal-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/FacileStudio/journal-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FacileStudio/journal-cli/releases/tag/v0.1.0
