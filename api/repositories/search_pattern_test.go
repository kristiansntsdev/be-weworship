package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsPatternEscapesLikeWildcards(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text", in: "Kristian", want: "%Kristian%"},
		{name: "trim spaces", in: "  Kristian  ", want: "%Kristian%"},
		{name: "percent", in: "100%", want: "%100\\%%"},
		{name: "underscore", in: "song_bank", want: "%song\\_bank%"},
		{name: "backslash", in: `C\D`, want: `%C\\D%`},
		{name: "combined", in: `50%_off\sale`, want: `%50\%\_off\\sale%`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsPattern(tt.in))
		})
	}
}
