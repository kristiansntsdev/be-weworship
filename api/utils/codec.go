package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	reNonAlnum      = regexp.MustCompile(`[^a-z0-9]+`)
	reDashes        = regexp.MustCompile(`-{2,}`)
	reHTML          = regexp.MustCompile(`<[^>]+>`)
	reChordPro      = regexp.MustCompile(`\[[^\]]+\]`)
	reDirective     = regexp.MustCompile(`\{[^}]+\}`)
	reMultiNewlines = regexp.MustCompile(`\n{3,}`)
	reSpaces        = regexp.MustCompile(`[ \t]+`)
)

// Slugify converts a string to a URL-safe slug, e.g. "Amazing Grace" → "amazing-grace".
func Slugify(s string) string {
	// Normalize unicode (decompose accents) then strip non-ASCII
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r) // remove combining marks
	}), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = strings.ToLower(result)
	result = reNonAlnum.ReplaceAllString(result, "-")
	result = reDashes.ReplaceAllString(result, "-")
	return strings.Trim(result, "-")
}


func ParseArtists(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	return []string{raw}
}

func MustArtistJSON(v any) string {
	switch x := v.(type) {
	case string:
		buf, _ := json.Marshal([]string{x})
		return string(buf)
	case []any:
		arr := make([]string, 0, len(x))
		for _, e := range x {
			arr = append(arr, fmt.Sprintf("%v", e))
		}
		buf, _ := json.Marshal(arr)
		return string(buf)
	case []string:
		buf, _ := json.Marshal(x)
		return string(buf)
	default:
		buf, _ := json.Marshal([]string{fmt.Sprintf("%v", v)})
		return string(buf)
	}
}

func ParseIntSlice(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}
	}
	var out []int
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	parts := strings.Split(strings.Trim(raw, "[]"), ",")
	for _, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(strings.Trim(p, `"`)))
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}

func ParseAnyJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	return raw
}

func NullableString(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func NullableTime(nt sql.NullTime) any {
	if nt.Valid {
		return nt.Time
	}
	return nil
}

func ContainsInt(list []int, target int) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func Ceil(total, per int) int {
	if total == 0 {
		return 0
	}
	return (total + per - 1) / per
}

func ParseCSVInts(raw string) []int {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}

// ExtractPlainLyrics strips HTML, ChordPro markup, and chord tags from lyrics
func ExtractPlainLyrics(input string) string {
	if input == "" {
		return ""
	}

	text := input

	// Remove HTML tags
	text = reHTML.ReplaceAllString(text, "")

	// Remove ChordPro format chords: [C], [Am], [G/B], etc.
	text = reChordPro.ReplaceAllString(text, "")

	// Remove {directive} tags like {textcolor}, {sot}, {eot}
	text = reDirective.ReplaceAllString(text, "")

	// Remove extra whitespace/newlines
	text = reMultiNewlines.ReplaceAllString(text, "\n\n")
	text = reSpaces.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}
