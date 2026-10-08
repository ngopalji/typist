package store

// migrations are applied in order, and the database's PRAGMA user_version
// records how many have run. Never edit a migration that has shipped; append
// a new one instead.
var migrations = []string{
	// 1: sessions and their raw keystrokes. Every statistic is derived from
	// keystrokes, so new analytics never need a schema change.
	`
	CREATE TABLE sessions (
		id          INTEGER PRIMARY KEY,
		mode        TEXT    NOT NULL,
		started_at  INTEGER NOT NULL, -- unix milliseconds
		duration_ms INTEGER NOT NULL,
		limit_ms    INTEGER NOT NULL,
		target      TEXT    NOT NULL, -- the text that was reached
		typed       TEXT    NOT NULL  -- the final input, after corrections
	);
	CREATE INDEX sessions_by_mode ON sessions (mode, started_at);

	CREATE TABLE keystrokes (
		session_id INTEGER NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
		seq        INTEGER NOT NULL, -- order within the session
		at_ms      INTEGER NOT NULL, -- since the session's first keystroke
		pos        INTEGER NOT NULL, -- index into target
		kind       INTEGER NOT NULL, -- 0 = character, 1 = backspace
		expected   TEXT,             -- NULL for backspaces
		typed      TEXT,             -- NULL for backspaces
		correct    INTEGER GENERATED ALWAYS AS (kind = 0 AND typed = expected) VIRTUAL,
		PRIMARY KEY (session_id, seq)
	) WITHOUT ROWID;
	`,
}
