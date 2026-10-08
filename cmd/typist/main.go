// Command typist is a terminal typing trainer that records every keystroke
// and turns them into detailed stats.
//
//	typist            open the menu
//	typist stats      jump straight to your stats
//	typist db         print where your data is stored
//	typist version    print the version
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"

	"github.com/ngopalji/typist/internal/config"
	"github.com/ngopalji/typist/internal/store"
	"github.com/ngopalji/typist/internal/ui"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = ""

const usage = `typist — a terminal typing trainer

usage:
  typist [flags]            open the menu
  typist [flags] stats      jump straight to your stats
  typist [flags] db         print where your data is stored
  typist version            print the version

flags:
  --db PATH   database file to use (default: $%s, then %s)
  --version   print the version
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "typist:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("typist", flag.ContinueOnError)
	dbFlag := fs.String("db", "", "database file to use")
	versionFlag := fs.Bool("version", false, "print the version")
	fs.Usage = func() {
		dir, _ := config.DataDir()
		fmt.Fprintf(fs.Output(), usage, config.EnvDB, filepath.Join(dir, "typist.db"))
	}
	// Accept flags both before and after the subcommand.
	if err := fs.Parse(args); err != nil {
		return ignoreHelp(err)
	}
	cmd := fs.Arg(0)
	if fs.NArg() > 0 {
		if err := fs.Parse(fs.Args()[1:]); err != nil {
			return ignoreHelp(err)
		}
		if fs.NArg() > 0 {
			return fmt.Errorf("unexpected argument %q (see typist --help)", fs.Arg(0))
		}
	}

	if *versionFlag {
		cmd = "version"
	}
	switch cmd {
	case "version":
		fmt.Println(buildVersion())
		return nil
	case "", "stats", "db":
	default:
		fs.Usage()
		return fmt.Errorf("unknown command %q", cmd)
	}

	path, err := config.DBPath(*dbFlag)
	if err != nil {
		return err
	}
	if cmd == "db" {
		fmt.Println(path)
		return nil
	}

	db, err := store.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()

	start := ui.HomeScreen
	if cmd == "stats" {
		start = ui.StatsScreen
	}
	app := ui.New(ui.Options{Store: db, Start: start, DBPath: path})
	_, err = tea.NewProgram(app).Run()
	return err
}

func ignoreHelp(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

// buildVersion prefers the ldflags version, then the module version that
// `go install module@version` embeds, then "dev".
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
