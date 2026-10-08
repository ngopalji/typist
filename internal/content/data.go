package content

import (
	_ "embed"
	"strings"
)

var (
	//go:embed data/words.txt
	wordsFile string
	//go:embed data/prose.txt
	proseFile string

	wordList = lines(wordsFile)
	passages = lines(proseFile)
)

// lines splits an embedded data file into its non-empty, non-comment lines.
func lines(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}
