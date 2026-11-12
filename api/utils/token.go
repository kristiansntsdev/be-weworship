package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

// RandomToken returns n random lowercase alphanumeric characters from crypto/rand, for unguessable public links.
func RandomToken(n int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = chars[int(v)%len(chars)]
	}
	return string(out), nil
}

var chordTokenRe = regexp.MustCompile(`\[[^\[\]\n]*\]`)

// StripChords removes inline ChordPro chords ([C], [Am7]) so a lyrics-only view never carries them.
// A line that held only chords (an instrumental line) is dropped; {directives} and section labels are untouched.
func StripChords(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !chordTokenRe.MatchString(line) {
			out = append(out, line)
			continue
		}
		stripped := chordTokenRe.ReplaceAllString(line, "")
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		out = append(out, stripped)
	}
	return strings.Join(out, "\n")
}
