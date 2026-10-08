// Package config resolves where typist keeps its data.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// EnvDB overrides the database location when set.
const EnvDB = "TYPIST_DB"

// DBPath returns the database to use: the explicit flag value if given, then
// $TYPIST_DB, then typist.db in the per-user data directory.
func DBPath(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}
	if env := os.Getenv(EnvDB); env != "" {
		return filepath.Abs(env)
	}
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "typist.db"), nil
}

// DataDir returns the per-user data directory:
//
//   - $XDG_DATA_HOME/typist if XDG_DATA_HOME is set
//   - %LocalAppData%\typist on Windows
//   - ~/.local/share/typist everywhere else (macOS included, like most CLIs)
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "typist"), nil
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LocalAppData"); local != "" {
			return filepath.Join(local, "typist"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("cannot find home directory; set " + EnvDB)
	}
	return filepath.Join(home, ".local", "share", "typist"), nil
}
