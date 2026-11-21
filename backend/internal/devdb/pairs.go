package devdb

type featurePair struct {
	id    string
	label string
}

// Feature ownership follows docs/student-pair-ownership.md.
var featurePairs = [...]featurePair{
	{id: "profile", label: "Profile"},
	{id: "ride_detail", label: "Ride detail"},
	{id: "creation", label: "Creation"},
	{id: "editing", label: "Editing"},
	{id: "joining", label: "Joining"},
	{id: "exits", label: "Exits"},
	{id: "discovery", label: "Discovery"},
	{id: "read_lists", label: "Read lists"},
}

func (pair featurePair) roleName() string {
	return "carpool_" + pair.id + "_test"
}

func (pair featurePair) configName() string {
	return pair.id + ".env"
}

func isTestRole(role string) bool {
	for _, pair := range featurePairs {
		if role == pair.roleName() {
			return true
		}
	}
	return false
}
