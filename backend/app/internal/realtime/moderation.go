package realtime

import (
	"strings"
	"time"
	"unicode"

	"drawo/internal/core/domain"
)

const (
	badWordCacheTTL        = 60 * time.Second
	chatWindowDuration     = 5 * time.Second
	maxChatMessagesPerWind = 5
)

type badWordCache struct {
	words     []domain.BadWord
	loadedAt  time.Time
	expiresAt time.Time
}

type chatLimitState struct {
	windowStart time.Time
	count       int
}

// NormalizeModerationText removes spaces/punctuation and normalizes Persian/
// Arabic variants so simple obfuscations like "b a d" or "ك‌ل‌م‌ه" can still be
// detected against the admin-managed bad-word list.
func NormalizeModerationText(input, lang string) string {
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
