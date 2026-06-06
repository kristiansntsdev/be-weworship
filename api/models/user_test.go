package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserBasicJSONUsesAPIFieldNames(t *testing.T) {
	body, err := json.Marshal(UserBasic{
		ID:       4,
		Username: "kristianepafroditus",
		Email:    "epafroditus.kristian@gmail.com",
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"id": 4,
		"username": "kristianepafroditus",
		"email": "epafroditus.kristian@gmail.com"
	}`, string(body))
}
