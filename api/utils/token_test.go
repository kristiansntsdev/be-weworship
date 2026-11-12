package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomToken(t *testing.T) {
	a, err := RandomToken(12)
	require.NoError(t, err)
	b, err := RandomToken(12)
	require.NoError(t, err)
	assert.Len(t, a, 12)
	assert.Regexp(t, `^[a-z0-9]{12}$`, a)
	assert.NotEqual(t, a, b)
}

func TestStripChords(t *testing.T) {
	src := "{comment: Verse 1}\n[C]Amazing [G]grace\n[Am]  [F]\nHow [C]sweet the sound\n\nplain line"
	want := "{comment: Verse 1}\nAmazing grace\nHow sweet the sound\n\nplain line"
	assert.Equal(t, want, StripChords(src))
}
