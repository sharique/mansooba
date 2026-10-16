package httpserver_test

// Usernames over HTTP (feature 018, story 2), against the real middleware stack
// and real SQLite: creation paths, immutability, the admin-only availability
// check, member listings and mentions by username.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pw = "Passw0rd!x"

func register(env *accessEnv, body string) (code int, rec string) {
	r := call(env.e, "POST", "/api/v1/auth/register", env.owner, body)
	return r.Code, r.Body.String()
}

func TestUsernames_AccountsCreatedWithoutOneGetAGeneratedOne(t *testing.T) {
	env := newAccessEnv(t) // owner, member and outsider were registered without usernames
	for token, want := range map[string]string{env.owner: "owner", env.member: "member", env.outsider: "outsider"} {
		rec := call(env.e, "GET", "/api/v1/auth/me", token, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, want, jsonField(t, rec, "username"))
	}
}

func TestUsernames_RegisterValidatesAndStoresLowerCase(t *testing.T) {
	env := newAccessEnv(t)

	code, body := register(env, `{"full_name":"Al","email":"al@example.com","password":"`+pw+`","username":"Al_Ice"}`)
	require.Equal(t, http.StatusCreated, code, body)
	assert.Contains(t, body, `"username":"al_ice"`)

	cases := []struct {
		username string
		want     int
		contains string
	}{
		{"AL_ICE", http.StatusConflict, "already taken"},
		{"admin", http.StatusBadRequest, "reserved"},
		{"ab", http.StatusBadRequest, "3 to 30"},
		{"sam-smith", http.StatusBadRequest, "letters, digits and underscores"},
	}
	for i, c := range cases {
		email := strings.Repeat("x", i+1) + "@example.com"
		code, body := register(env, `{"full_name":"N","email":"`+email+`","password":"`+pw+`","username":"`+c.username+`"}`)
		assert.Equal(t, c.want, code, "%s: %s", c.username, body)
		assert.Contains(t, body, c.contains, c.username)
	}
	// Nothing was created by the refused requests.
	rec := call(env.e, "GET", "/api/v1/admin/users", env.owner, "")
	assert.NotContains(t, rec.Body.String(), "xx@example.com")
}

func TestUsernames_GeneratedFromAwkwardEmails(t *testing.T) {
	env := newAccessEnv(t)
	for email, want := range map[string]string{
		"sam-smith@example.com": "samsmith",
		"sam+qa@example.com":    "samqa",
		"admin@example.com":     "admin2", // reserved, so numbered
		"ab@example.com":        "userab",
	} {
		code, body := register(env, `{"full_name":"N","email":"`+email+`","password":"`+pw+`"}`)
		require.Equal(t, http.StatusCreated, code, body)
		assert.Contains(t, body, `"username":"`+want+`"`, email)
	}
	code, body := register(env, `{"full_name":"N","email":"Samsmith@other.com","password":"`+pw+`"}`)
	require.Equal(t, http.StatusCreated, code, body)
	assert.Contains(t, body, `"username":"samsmith2"`, "collision is numbered")
}

func TestUsernames_CannotBeChangedByAnyone(t *testing.T) {
	env := newAccessEnv(t)

	// The owner of the account.
	rec := call(env.e, "PUT", "/api/v1/auth/me", env.member, `{"username":"newname"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "cannot be changed")
	// Sending the current value is a harmless no-op.
	rec = call(env.e, "PUT", "/api/v1/auth/me", env.member, `{"username":"member","full_name":"Renamed Member"}`)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "member", jsonField(t, rec, "username"))
	// An administrator.
	rec = call(env.e, "PATCH", "/api/v1/admin/users/"+env.memberID, env.owner, `{"username":"newname"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "cannot be changed")

	rec = call(env.e, "GET", "/api/v1/auth/me", env.member, "")
	assert.Equal(t, "member", jsonField(t, rec, "username"))
	assert.Equal(t, "Renamed Member", jsonField(t, rec, "name"))
}

// Password change and reset persist through the user repository's Update; they
// must keep working and must not disturb the username.
func TestUsernames_SurvivePasswordChange(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "PUT", "/api/v1/auth/me/password", env.member, `{"current_password":"`+pw+`","new_password":"Newpassw0rd!"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/auth/me", env.member, "")
	assert.Equal(t, "member", jsonField(t, rec, "username"))
	rec = call(env.e, "POST", "/api/v1/auth/login", "", `{"email":"member@example.com","password":"Newpassw0rd!"}`)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUsernames_AvailabilityCheckIsAdminOnly(t *testing.T) {
	env := newAccessEnv(t)
	check := func(token, username string) *httptest.ResponseRecorder {
		return call(env.e, "GET", "/api/v1/admin/usernames/check?username="+username, token, "")
	}

	assert.Equal(t, http.StatusUnauthorized, check("", "fresh").Code)
	assert.Equal(t, http.StatusForbidden, check(env.member, "fresh").Code, "a non-admin must not be able to probe usernames")
	assert.Equal(t, http.StatusBadRequest, call(env.e, "GET", "/api/v1/admin/usernames/check", env.owner, "").Code)

	for _, c := range []struct {
		username, reason string
		available        bool
	}{{"fresh_name", "", true}, {"MEMBER", "taken", false}, {"admin", "reserved", false}, {"ab", "invalid", false}} {
		rec := check(env.owner, c.username)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, c.available, strings.Contains(rec.Body.String(), `"available":true`), c.username)
		assert.Equal(t, c.reason, jsonField(t, rec, "reason"), c.username)
		if !c.available {
			assert.NotEmpty(t, jsonField(t, rec, "message"), c.username)
		}
	}
	// It reads only: nothing was created.
	rec := call(env.e, "GET", "/api/v1/admin/users", env.owner, "")
	assert.NotContains(t, rec.Body.String(), "fresh_name")
}

func TestUsernames_MemberListsCarryUsernameAndActiveFlag(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "GET", "/api/v1/projects/acc/members", env.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"username":"member"`)
	assert.Contains(t, rec.Body.String(), `"username":"owner"`)
	assert.Contains(t, rec.Body.String(), `"is_active":true`)
	assert.Contains(t, rec.Body.String(), `"email":"member@example.com"`, "the members page still shows emails")

	rec = call(env.e, "GET", "/api/v1/admin/users", env.owner, "")
	assert.Contains(t, rec.Body.String(), `"username":"outsider"`)
}

func TestUsernames_MentionsNotifyByUsername(t *testing.T) {
	env := newAccessEnv(t)
	post := func(body string) {
		rec := call(env.e, "POST", "/api/v1/issues/"+env.issue1+"/comments", env.owner, `{"body":`+jsonString(t, body)+`}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}
	count := func(token string) int {
		rec := call(env.e, "GET", "/api/v1/notifications", token, "")
		return strings.Count(rec.Body.String(), `"kind":"mention"`)
	}

	post("hello @Member please look")
	assert.Equal(t, 1, count(env.member), "the username matches ignoring case")
	post("(@member) **@member**")
	assert.Equal(t, 2, count(env.member), "two mentions in one comment notify once")
	post("mail member@example.com or a.b@member")
	assert.Equal(t, 2, count(env.member), "an email address is not a mention")
	post("@memb ership")
	assert.Equal(t, 2, count(env.member), "a prefix of a username is not a mention")
	assert.Equal(t, 0, count(env.outsider))
}
