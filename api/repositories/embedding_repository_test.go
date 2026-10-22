package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatVector(t *testing.T) {
	assert.Equal(t, "[]", formatVector(nil))
	assert.Equal(t, "[0.5,-0.25,1e-05]", formatVector([]float32{0.5, -0.25, 0.00001}))
}
