package devdb

import "testing"

// TestIsTestRoleMatchesWholeRoleNames checks that a provisioned pair role is
// recognized and that near misses are not. The provisioning commands grant
// classroom privileges to whatever isTestRole accepts, so a prefix or substring
// match would hand them to an unrelated role.
func TestIsTestRoleMatchesWholeRoleNames(t *testing.T) {
	t.Parallel()
	provisioned := featurePairs[0].roleName()

	for _, tc := range []struct {
		name string
		role string
		want bool
	}{
		{name: "first provisioned pair", role: provisioned, want: true},
		{name: "last provisioned pair", role: featurePairs[len(featurePairs)-1].roleName(), want: true},
		{name: "shared development role", role: "carpool_development", want: false},
		{name: "retired pair naming", role: "carpool_pair_01_test", want: false},
		{name: "provisioned role with a suffix", role: provisioned + "_extra", want: false},
		{name: "unassigned feature", role: "carpool_unknown_test", want: false},
		{name: "empty", role: "", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isTestRole(tc.role); got != tc.want {
				t.Errorf("isTestRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}
