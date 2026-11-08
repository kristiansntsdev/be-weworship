package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatVector(t *testing.T) {
	assert.Equal(t, "[]", formatVector(nil))
	assert.Equal(t, "[0.5,-0.25,1e-05]", formatVector([]float32{0.5, -0.25, 0.00001}))
}

func TestParseVector_RoundTripsFormatVector(t *testing.T) {
	in := []float32{0.5, -0.25, 0.00001, 1}
	out, err := parseVector(formatVector(in))
	assert.NoError(t, err)
	assert.Equal(t, in, out)

	empty, err := parseVector("[]")
	assert.NoError(t, err)
	assert.Empty(t, empty)

	_, err = parseVector("0.5,1")
	assert.Error(t, err)
	_, err = parseVector("[0.5,x]")
	assert.Error(t, err)
}
