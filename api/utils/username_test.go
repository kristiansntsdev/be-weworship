package utils

import (
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	valid := []string{
		"kristian",
		"Kris.123",
		"john_doe",
		"jane-doe",
		"user123456",
		"A1b2C3",          // exactly 6 chars
		"abcdefghijklmnopqrst123456789a", // exactly 30 chars
	}
	for _, u := range valid {
		if err := ValidateUsername(u); err != nil {
			t.Errorf("expected %q to be valid, got: %v", u, err)
		}
	}

	invalid := []struct {
		input string
		why   string
	}{
		{"hi", "too short"},
		{"ab cd", "contains space"},
		{"_starts", "starts with special"},
		{"ends_", "ends with special"},
		{"has__double", "consecutive specials"},
		{"has..dot", "consecutive specials"},
		{"has.-mixed", "consecutive specials"},
		{"admin", "reserved"},
		{"ADMIN", "reserved (case-insensitive)"},
		{"weworship", "reserved brand"},
		{"null", "reserved"},
		{"has space!", "invalid chars"},
		{"user@name", "invalid char @"},
		{"toolongusernamethatexceedsthirtycharsmaximum", "too long"},
	}
	for _, tc := range invalid {
		if err := ValidateUsername(tc.input); err == nil {
			t.Errorf("expected %q (%s) to be invalid, but passed", tc.input, tc.why)
		}
	}
}

func TestGenerateUsername(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		u := GenerateUsername()
		// Must always pass validation
		if err := ValidateUsername(u); err != nil {
			t.Errorf("GenerateUsername() produced invalid username %q: %v", u, err)
		}
		// Must always start with "ww_"
		if !strings.HasPrefix(u, "ww_") {
			t.Errorf("GenerateUsername() does not start with ww_: %q", u)
		}
		// Must be exactly 11 chars (ww_ + 8)
		if len(u) != 11 {
			t.Errorf("GenerateUsername() wrong length %d: %q", len(u), u)
		}
		seen[u] = struct{}{}
	}
	// 1000 draws should produce >990 unique values (birthday collision at this
	// scale is ~1 in 2.8M, so getting even 5 collisions would be astronomically unlikely)
	if len(seen) < 990 {
		t.Errorf("GenerateUsername() produced too many collisions: only %d unique in 1000 draws", len(seen))
	}
}
