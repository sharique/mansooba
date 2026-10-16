package domain

import (
	"errors"
	"strings"
)

// Usernames (feature 018, ADR-038): the unique, permanent name used in mentions.
const (
	UsernameMinLen = 3
	UsernameMaxLen = 30
)

// Username errors. Their messages are shown to the person filling in a form.
var (
	ErrUsernameInvalid   = errors.New("Usernames are 3 to 30 characters: letters, digits and underscores only.")
	ErrUsernameReserved  = errors.New("That username is reserved.")
	ErrUsernameTaken     = errors.New("That username is already taken.")
	ErrUsernameImmutable = errors.New("username cannot be changed")
)

// ReservedUsernames cannot be chosen, by anyone, and are skipped when a
// username is generated. They guard against future group mentions and names
// that look official.
var ReservedUsernames = map[string]bool{
	"everyone": true, "all": true, "here": true, "channel": true,
	"admin": true, "system": true, "support": true, "mansooba": true,
}

// NormalizeUsername trims surrounding space and lower-cases the name.
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func usernameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
}

// ValidateUsername reports whether s, once normalised, is an allowed username:
// ErrUsernameInvalid for the wrong length or characters, ErrUsernameReserved
// for a reserved name. It does not check whether the name is taken.
func ValidateUsername(s string) error {
	n := NormalizeUsername(s)
	if len(n) < UsernameMinLen || len(n) > UsernameMaxLen {
		return ErrUsernameInvalid
	}
	for _, r := range n {
		if !usernameChar(r) {
			return ErrUsernameInvalid
		}
	}
	if ReservedUsernames[n] {
		return ErrUsernameReserved
	}
	return nil
}

// DeriveUsername builds a valid, non-reserved candidate from an email address:
// the part before the last "@", lower-cased, with every character outside
// a-z, 0-9 and underscore removed, padded with "user" when fewer than three
// characters are left, cut to 30, and numbered ("admin" becomes "admin2") when
// reserved. It does not check whether the name is taken; the caller adds a
// number for that. The frontend has a twin of this function (app/utils/username.ts)
// and both are tested against internal/domain/testdata/username_vectors.json.
func DeriveUsername(email string) string {
	local := email
	if i := strings.LastIndex(email, "@"); i >= 0 {
		local = email[:i]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if usernameChar(r) {
			b.WriteRune(r)
		}
	}
	name := b.String()
	if len(name) < UsernameMinLen {
		name = "user" + name
	}
	if len(name) > UsernameMaxLen {
		name = name[:UsernameMaxLen]
	}
	if ReservedUsernames[name] {
		name += "2"
	}
	return name
}
