package repositories

import "strings"

func containsPattern(text string) string {
	return "%" + escapeLikePattern(strings.TrimSpace(text)) + "%"
}

func escapeLikePattern(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if r == '\\' || r == '%' || r == '_' {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
