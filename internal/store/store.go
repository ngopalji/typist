// Package store persists typing sessions in a local SQLite database.
//
// It saves raw keystrokes rather than computed scores, so the stats package
// can derive new analytics from old sessions at any time.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; no cgo needed

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/typing"
)

// Store is a handle to the database. It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at path and brings its schema
// up to date.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > len(migrations) {
		return fmt.Errorf("database schema v%d is newer than this build supports (v%d); upgrade typist", version, len(migrations))
	}
	for i := version; i < len(migrations); i++ {
		err := s.tx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1))
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	return nil
}

// Save stores a finished test and returns its id.
func (s *Store) Save(ctx context.Context, r typing.Result) (int64, error) {
	var id int64
	err := s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO sessions (mode, started_at, duration_ms, limit_ms, target, typed)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			string(r.Mode), r.StartedAt.UnixMilli(), r.Duration.Milliseconds(), r.Limit.Milliseconds(), r.Target, r.Typed)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO keystrokes (session_id, seq, at_ms, pos, kind, expected, typed)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for seq, k := range r.Keystrokes {
			if _, err := stmt.ExecContext(ctx, id, seq, k.At.Milliseconds(), k.Pos, k.Kind,
				runeOrNull(k.Expected), runeOrNull(k.Typed)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("save session: %w", err)
	}
	return id, nil
}

// All returns every saved session, oldest first.
func (s *Store) All(ctx context.Context) ([]typing.Result, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, mode, started_at, duration_ms, limit_ms, target, typed
		 FROM sessions ORDER BY started_at, id`)
	if err != nil {
		return nil, fmt.Errorf("load sessions: %w", err)
	}
	defer rows.Close()

	var results []typing.Result
	index := map[int64]int{}
	for rows.Next() {
		var (
			id                       int64
			mode, target, typed      string
			started, duration, limit int64
		)
		if err := rows.Scan(&id, &mode, &started, &duration, &limit, &target, &typed); err != nil {
			return nil, err
		}
		index[id] = len(results)
		results = append(results, typing.Result{
			Mode:      content.Mode(mode),
			StartedAt: time.UnixMilli(started),
			Duration:  time.Duration(duration) * time.Millisecond,
			Limit:     time.Duration(limit) * time.Millisecond,
			Target:    target,
			Typed:     typed,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	krows, err := s.db.QueryContext(ctx,
		`SELECT session_id, at_ms, pos, kind, expected, typed FROM keystrokes ORDER BY session_id, seq`)
	if err != nil {
		return nil, fmt.Errorf("load keystrokes: %w", err)
	}
	defer krows.Close()
	for krows.Next() {
		var (
			id, at          int64
			pos             int
			kind            typing.Kind
			expected, typed sql.NullString
		)
		if err := krows.Scan(&id, &at, &pos, &kind, &expected, &typed); err != nil {
			return nil, err
		}
		i, ok := index[id]
		if !ok {
			continue
		}
		results[i].Keystrokes = append(results[i].Keystrokes, typing.Keystroke{
			At:       time.Duration(at) * time.Millisecond,
			Pos:      pos,
			Kind:     kind,
			Expected: firstRune(expected),
			Typed:    firstRune(typed),
		})
	}
	return results, krows.Err()
}

func (s *Store) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

func runeOrNull(r rune) sql.NullString {
	if r == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(r), Valid: true}
}

func firstRune(s sql.NullString) rune {
	for _, r := range s.String {
		return r
	}
	return 0
}
