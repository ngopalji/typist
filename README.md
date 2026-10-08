# typist

A terminal typing trainer that records every keystroke and turns it into
detailed stats: speed over time, accuracy, a per-key heatmap, what you type
instead of what you meant, and your slowest key-to-key transitions.

Modes: **numbers**, **letters**, **words**, **prose**, **symbols**, each in
30s / 60s / 120s tests.

## Usage

```sh
typist            # menu
typist stats      # straight to your stats
typist db         # print where your data lives
typist --db PATH  # use a different database (also: $TYPIST_DB)
```

| screen  | keys |
| ------- | ---- |
| menu    | `j`/`k` mode · `h`/`l` length · `enter` start · `s` stats · `q` quit |
| typing  | type · `backspace` / `ctrl+w` fix · `tab` restart · `esc` back |
| results | `enter` again · `b` back · `s` stats · `m` heatmap metric · `j`/`k` scroll · `q` quit |
| stats   | `h`/`l` filter by mode · `j`/`k` scroll · `g`/`G` top/bottom · `m` heatmap metric · `b` back |

`ctrl+c` quits from anywhere.

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

## Where data lives

One SQLite file, resolved in this order:

1. `--db PATH`
2. `$TYPIST_DB`
3. `$XDG_DATA_HOME/typist/typist.db`, else `~/.local/share/typist/typist.db`
   (`%LocalAppData%\typist\typist.db` on Windows)

The schema is versioned with `PRAGMA user_version` and migrates itself on
startup. It stores raw keystrokes, not scores, so every stat can be
recomputed, and new stats work on old sessions.

```sh
sqlite3 ~/.local/share/typist/typist.db \
  "SELECT expected, AVG(correct) FROM keystrokes WHERE kind = 0 GROUP BY expected ORDER BY 2"
```

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

**Adding a mode:** add a `Mode` and a generator in `internal/content`. It
then appears in the menu, the stats filter, and the heatmap automatically.
Give it a keyboard layout in `keyboardFor` if the default doesn't fit.

**Adding a stat:** write it in `internal/stats` as a function of
`typing.Result` or `[]stats.Session`, and it works on all your existing
history.

## Releasing

`.goreleaser.yaml` builds static binaries for macOS, Linux, and Windows
(the SQLite driver is pure Go, so no cgo is needed):

```sh
git tag v0.1.0 && git push --tags
goreleaser release --clean
```

Users can also install from source with
`go install github.com/nihaar/typist/cmd/typist@latest`. Released binaries
use the same default database path as `make install`, so upgrading keeps
your history.
