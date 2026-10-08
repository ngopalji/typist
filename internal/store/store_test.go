package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ngopalji/typist/internal/content"
	"github.com/ngopalji/typist/internal/typing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "typist.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	want := typing.Result{
		Mode:      content.Symbols,
		Limit:     30 * time.Second,
		StartedAt: time.UnixMilli(1_790_000_000_000),
		Duration:  30 * time.Second,
		Target:    "{} =>",
		Typed:     "{] =>",
		Keystrokes: []typing.Keystroke{
			{At: 0, Pos: 0, Kind: typing.KindChar, Expected: '{', Typed: '{'},
			{At: 120 * time.Millisecond, Pos: 1, Kind: typing.KindChar, Expected: '}', Typed: ']'},
			{At: 300 * time.Millisecond, Pos: 1, Kind: typing.KindBackspace},
		},
	}
	ctx := context.Background()
	if _, err := s.Save(ctx, want); err != nil {
		t.Fatal(err)
	}

	got, err := s.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}

	// The generated column should agree with the Go-side definition.
	var correct int
	if err := s.db.QueryRow(`SELECT SUM(correct) FROM keystrokes`).Scan(&correct); err != nil {
		t.Fatal(err)
	}
	if correct != 1 {
		t.Fatalf("SUM(correct) = %d, want 1", correct)
	}
}

func TestReopenKeepsDataAndSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typist.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(context.Background(), typing.Result{Mode: content.Words, StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s.Close()
	got, err := s.All(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("after reopen: %d sessions, err %v", len(got), err)
	}
}
