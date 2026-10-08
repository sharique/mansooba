package domain

// AGPL-3.0-only is the project's own licence, not configurable — see NOTICE
// and LICENSE at the repository root.
const (
	InstanceLicense    = "AGPL-3.0-only"
	InstanceLicenseURL = "https://www.gnu.org/licenses/agpl-3.0.html"
)

// InstanceInfo is what a running instance tells any user about itself, so
// they can find the source they are entitled to under AGPL section 13.
type InstanceInfo struct {
	// Version is baked in at image build time; "dev" for a local build.
	Version string
	// SourceURL is where this instance's source is published, from the
	// SOURCE_CODE_URL setting; defaults to the project's public repository.
	SourceURL string
	// License and LicenseURL are always InstanceLicense/InstanceLicenseURL —
	// carried on the struct so callers don't need a second import.
	License    string
	LicenseURL string
}
