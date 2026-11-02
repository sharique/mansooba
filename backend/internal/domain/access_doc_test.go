package domain_test

import (
	"os"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// docs/roles-and-permissions.md is the one published statement of what each
// role may do. This reads its table and fails when it and the rules in the code
// disagree, so neither can change alone.
func TestRolesDoc_MatchesTheRuleTable(t *testing.T) {
	b, err := os.ReadFile("../../../docs/roles-and-permissions.md")
	require.NoError(t, err, "the roles document must exist")

	documented := map[string][3]bool{} // capability -> viewer, member, admin
	inTable := false
	for _, line := range strings.Split(string(b), "\n") {
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if len(cells) == 4 && cells[0] == "Capability" && cells[1] == "Viewer" && cells[2] == "Member" && cells[3] == "Admin" {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if len(cells) != 4 || strings.HasPrefix(cells[0], "---") {
			if len(documented) > 0 && len(cells) != 4 {
				break // end of the table
			}
			continue
		}
		name := strings.Trim(cells[0], "`")
		var marks [3]bool
		for i, c := range cells[1:] {
			switch c {
			case "yes":
				marks[i] = true
			case "":
			default:
				t.Fatalf("%s: unexpected mark %q (use \"yes\" or leave blank)", name, c)
			}
		}
		documented[name] = marks
	}
	require.NotEmpty(t, documented, "no permissions table found in docs/roles-and-permissions.md")

	roles := [3]string{domain.RoleViewer, domain.RoleMember, domain.RoleAdmin}
	for _, role := range roles {
		for _, c := range domain.CapabilitiesOf(role) {
			_, ok := documented[string(c)]
			assert.True(t, ok, "%s holds %s in the code but the document does not list it", role, c)
		}
	}
	for name, marks := range documented {
		for i, role := range roles {
			assert.Equal(t, marks[i], domain.RoleCan(role, domain.Capability(name)),
				"%s / %s: the document says %v, the code says %v", name, role, marks[i], domain.RoleCan(role, domain.Capability(name)))
		}
	}
}
