package domain_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vectors struct {
	Derive []struct {
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"derive"`
	Validate []struct {
		Input      string `json:"input"`
		Valid      bool   `json:"valid"`
		Normalised string `json:"normalised"`
		Reason     string `json:"reason"`
	} `json:"validate"`
}

func loadVectors(t *testing.T) vectors {
	t.Helper()
	b, err := os.ReadFile("testdata/username_vectors.json")
	require.NoError(t, err)
	var v vectors
	require.NoError(t, json.Unmarshal(b, &v))
	require.NotEmpty(t, v.Derive)
	require.NotEmpty(t, v.Validate)
	return v
}

func TestDeriveUsername_MatchesSharedVectors(t *testing.T) {
	for _, c := range loadVectors(t).Derive {
		assert.Equal(t, c.Username, domain.DeriveUsername(c.Email), "derive %q", c.Email)
	}
}

func TestValidateUsername_MatchesSharedVectors(t *testing.T) {
	for _, c := range loadVectors(t).Validate {
		err := domain.ValidateUsername(c.Input)
		switch {
		case c.Valid:
			assert.NoError(t, err, "validate %q", c.Input)
			assert.Equal(t, c.Normalised, domain.NormalizeUsername(c.Input), "normalise %q", c.Input)
		case c.Reason == "reserved":
			assert.ErrorIs(t, err, domain.ErrUsernameReserved, "validate %q", c.Input)
		default:
			assert.ErrorIs(t, err, domain.ErrUsernameInvalid, "validate %q", c.Input)
		}
	}
}

func TestDerivedUsernamesAreAlwaysValid(t *testing.T) {
	for _, e := range []string{"", "@", "a", "!!!@x", "ünï@x", strings.Repeat("z", 80) + "@x", "admin@x", "all@x", "x@y@z"} {
		assert.NoError(t, domain.ValidateUsername(domain.DeriveUsername(e)), "derive %q -> %q", e, domain.DeriveUsername(e))
	}
}

func TestUsernameErrorsCarryUserFacingMessages(t *testing.T) {
	assert.Contains(t, domain.ErrUsernameInvalid.Error(), "3 to 30")
	assert.Contains(t, domain.ErrUsernameReserved.Error(), "reserved")
	assert.Contains(t, domain.ErrUsernameTaken.Error(), "already taken")
	assert.Contains(t, domain.ErrUsernameImmutable.Error(), "cannot be changed")
}
