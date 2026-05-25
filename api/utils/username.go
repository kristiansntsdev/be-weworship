package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

// GenerateUsername returns a random, always-valid username of the form
// "ww_<8 alphanumeric chars>", e.g. "ww_a3kx92bf".
// It satisfies all WeWorship username rules out of the box and is safe
// to use without further validation.
func GenerateUsername() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is extremely rare; fall back to a fixed prefix
		// so the caller's uniqueness retry still works.
		return "ww_fallback"
	}
	out := make([]byte, 8)
	for i, v := range b {
		out[i] = chars[int(v)%len(chars)]
	}
	return "ww_" + string(out)
}

// usernamePattern allows A-Z a-z 0-9 . _ - only.
// Anchored so no spaces or other chars can sneak in.
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// reservedUsernames is the blocklist of handles that must not be registered.
// Add entries in lowercase — comparison is always lowercased.
var reservedUsernames = map[string]struct{}{
	// roles / system accounts
	"admin": {}, "administrator": {}, "root": {}, "superuser": {},
	"system": {}, "moderator": {}, "mod": {}, "owner": {},
	// mail / RFC 2142 required mailboxes
	"abuse": {}, "postmaster": {}, "hostmaster": {}, "webmaster": {},
	"noreply": {}, "no-reply": {}, "mailer-daemon": {},
	// support surfaces
	"support": {}, "help": {}, "info": {}, "contact": {},
	"feedback": {}, "security": {}, "privacy": {},
	// infrastructure
	"api": {}, "www": {}, "mail": {}, "ftp": {}, "smtp": {},
	"imap": {}, "pop": {}, "cdn": {}, "assets": {},
	// brand / product
	"weworship": {}, "we-worship": {}, "we_worship": {},
	// programming pitfalls
	"null": {}, "undefined": {}, "none": {}, "true": {}, "false": {},
	// short generics often used for abuse
	"test": {}, "guest": {}, "user": {}, "username": {},
	"anonymous": {}, "anon": {},
}

// ValidateUsername checks that s conforms to WeWorship username rules.
// Returns a user-facing error string, or nil if valid.
//
//   - Length: 6–30 characters
//   - Allowed characters: A-Z a-z 0-9 . _ -
//   - No spaces
//   - Cannot start or end with . _ -
//   - No consecutive special characters (.. __ -- ._ etc.)
//   - Not a reserved word
func ValidateUsername(s string) error {
	if len(s) < 6 {
		return fmt.Errorf("username must be at least 6 characters")
	}
	if len(s) > 30 {
		return fmt.Errorf("username must be at most 30 characters")
	}
	if !usernamePattern.MatchString(s) {
		return fmt.Errorf("username may only contain letters, numbers, periods (.), underscores (_), and hyphens (-)")
	}
	// Must not start or end with a special character
	first, last := s[0], s[len(s)-1]
	for _, ch := range []byte{'.', '_', '-'} {
		if first == ch || last == ch {
			return fmt.Errorf("username may not start or end with a period, underscore, or hyphen")
		}
	}
	// No consecutive special characters
	if strings.ContainsAny(s, "._-") {
		for i := 1; i < len(s); i++ {
			prev, cur := s[i-1], s[i]
			prevSpecial := prev == '.' || prev == '_' || prev == '-'
			curSpecial := cur == '.' || cur == '_' || cur == '-'
			if prevSpecial && curSpecial {
				return fmt.Errorf("username may not contain consecutive special characters (e.g. .., __, --)")
			}
		}
	}
	// Reserved word check (case-insensitive)
	if _, blocked := reservedUsernames[strings.ToLower(s)]; blocked {
		return fmt.Errorf("username '%s' is reserved and cannot be registered", s)
	}
	return nil
}
