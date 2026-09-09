package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/patrulje"
)

// A holdnummer is only handed out to an admitted, paid team, so it must keep the
// page open no matter what the oversubscribed cut-off says about the signup time.
func TestPatruljeAdmitted(t *testing.T) {
	tests := []struct {
		name string
		team *patrulje.Patrulje
		want bool
	}{
		{name: "no team is not proof of anything", team: nil},
		{name: "no number yet", team: &patrulje.Patrulje{TeamID: "t-1"}},
		{name: "blank number is no number", team: &patrulje.Patrulje{TeamID: "t-1", TeamNumber: "   "}},
		{name: "a number means admitted", team: &patrulje.Patrulje{TeamID: "t-1", TeamNumber: "142"}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := patruljeAdmitted(tc.team); got != tc.want {
				t.Errorf("patruljeAdmitted(%+v) = %v, want %v", tc.team, got, tc.want)
			}
		})
	}
}
