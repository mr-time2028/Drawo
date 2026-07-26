package realtime

import (
	"strings"
	"unicode"
)

var persianCharMap = map[rune]rune{
	'ي': 'ی',
	'ى': 'ی',
	'ك': 'ک',
	'ة': 'ه',
	'ۀ': 'ه',
	'ؤ': 'و',
	'إ': 'ا',
	'أ': 'ا',
	'آ': 'ا',
}

// NormalizeGuess converts user chat and target words into comparable forms.
// English is lowercased and punctuation/spacing is removed. Persian additionally
// normalizes common Arabic code points to Persian equivalents and removes
// diacritics/zero-width characters.
func NormalizeGuess(input, lang string) string {
	input = strings.TrimSpace(input)
	var b strings.Builder
	for _, r := range input {
		if strings.EqualFold(lang, "fa") {
			if mapped, ok := persianCharMap[r]; ok {
				r = mapped
			}
			if r == '\u200c' || isArabicDiacritic(r) {
				continue
			}
		} else {
			r = unicode.ToLower(r)
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isArabicDiacritic(r rune) bool {
	return (r >= '\u064B' && r <= '\u065F') || r == '\u0670'
}
