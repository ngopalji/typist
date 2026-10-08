package content

import "math/rand/v2"

// symbolKeys is every non-alphanumeric character on a US keyboard.
var symbolKeys = []rune("`~!@#$%^&*()-_=+[]{}\\|;:'\",.<>/?")

// codeTokens are symbol sequences that show up constantly in real code and
// shells, so practicing them pays off more than purely random clusters.
var codeTokens = []string{
	"=>", "->", "<-", "::", ":=", "==", "!=", "===", "<=", ">=", "&&", "||",
	"++", "--", "+=", "-=", "*=", "/=", "%=", "<<", ">>", "**", "//", "/*",
	"*/", "#!", "()", "[]", "{}", "<>", "{};", "();", "[0]", "\"\"", "''",
	"``", "${}", "$@", "$?", "~/", "../", "./", "...", "?.", "??", "!!",
	"@@", "#{}", "%s", "%d", "&mut", "|>", "<$>", "\\n", "\\t", "^$", ".*",
}

func symbols(rng *rand.Rand) Source {
	random := groups(rng, symbolKeys, 1, 3)
	tokens := pick(rng, codeTokens)
	return SourceFunc(func() string {
		if rng.IntN(2) == 0 {
			return tokens.Next()
		}
		return random.Next()
	})
}
