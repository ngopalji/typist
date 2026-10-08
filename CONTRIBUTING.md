# Contributing

Issues and PRs are welcome. Run `make test` before sending one; CI also
checks `gofmt` and `go vet`.

## Development

```sh
make dev        # run from source against .dev/typist.db (throwaway, gitignored)
make dev-stats  # same, straight to the stats screen
make dev-db     # open the dev database in sqlite3
make test       # run all tests
make install    # install to $(go env GOPATH)/bin for real use
```

`make dev` never touches your real history. A plain `go run ./cmd/typist`
does, because it uses the default database path.

## How it's built

```
cmd/typist         CLI entrypoint: flags, subcommands, wiring
internal/content   text generators; one endless Source per mode
internal/typing    the test engine: cursor, keystrokes, timing (no UI)
internal/stats     pure analytics over keystrokes: summary, timeline, per-key breakdown, history
internal/store     SQLite persistence and migrations
internal/config    where the database lives
internal/ui        Bubble Tea screens: home, typer, results, stats
internal/ui/chart  braille line charts and eighth-block bar charts
```

Dependencies only point inward: `ui` uses everything, `stats` and `store`
use `typing`, and `typing` uses nothing but `content`'s Mode type. Each
screen is its own small model. Screens navigate by sending messages to the
root `App`, which does all I/O in commands, off the UI loop.

The database path resolves from `--db PATH`, then `$TYPIST_DB`, then
`$XDG_DATA_HOME/typist/typist.db`, else `~/.local/share/typist/typist.db`
(`%LocalAppData%\typist\typist.db` on Windows). The schema is versioned with
`PRAGMA user_version` and migrates itself on startup.

**Adding a mode:** add a `Mode` and a generator in `internal/content`. It
then appears in the menu, the stats filter, and the heatmap automatically.
Give it a keyboard layout in `keyboardFor` if the default doesn't fit.

**Adding a stat:** write it in `internal/stats` as a function of
`typing.Result` or `[]stats.Session`, and it works on all existing history.

## Releasing

Push a version tag and the `release` workflow runs
[GoReleaser](https://goreleaser.com), which builds static binaries for
macOS, Linux, and Windows (the SQLite driver is pure Go, so no cgo) and
publishes them as a GitHub release:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

To check the build locally first: `goreleaser release --snapshot --clean`.
