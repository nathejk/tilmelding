package main

import (
	"os"
	"testing"

	"github.com/nathejk/shared-go/types"
)

// closedSignupApp is an application whose only configured behaviour is the
// closed-signup set, which is all signupClosed / signupStatus read.
func closedSignupApp(names ...string) *application {
	app := &application{}
	app.config.signup.closedTypes = canonicalSignupTypes(names)
	return app
}

func TestClosedSignupTypesFromEnv(t *testing.T) {
	tests := []struct {
		name string
		// set reports whether the variable exists at all; an unset variable and one
		// set to "" must mean different things.
		set   bool
		value string
		want  map[string]bool
	}{
		{
			name: "unset falls back to the default closed set",
			set:  false,
			want: map[string]bool{"gøgler": true},
		},
		{
			// The escape hatch: re-opening a signup without a code change.
			name:  "set but empty closes nothing",
			set:   true,
			value: "",
			want:  map[string]bool{},
		},
		{
			// The footgun this guards: "badut" is what the route, the view and the Go
			// constant are called, but "gøgler" is the value on the wire. Parsing it
			// literally would match nothing and leave the signup open.
			name:  "badut is accepted as an alias for gøgler",
			set:   true,
			value: "badut",
			want:  map[string]bool{"gøgler": true},
		},
		{
			name:  "gogler without the ø is accepted too",
			set:   true,
			value: "GOGLER",
			want:  map[string]bool{"gøgler": true},
		},
		{
			name:  "the canonical value is accepted",
			set:   true,
			value: "gøgler",
			want:  map[string]bool{"gøgler": true},
		},
		{
			name:  "several types, with whitespace and a trailing separator",
			set:   true,
			value: " badut , klan , ",
			want:  map[string]bool{"gøgler": true, "klan": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(closedSignupTypesEnv, tc.value)
			} else {
				// t.Setenv cannot unset, and the variable may be present in the
				// developer's environment, so restore it by hand.
				orig, had := os.LookupEnv(closedSignupTypesEnv)
				os.Unsetenv(closedSignupTypesEnv)
				t.Cleanup(func() {
					if had {
						os.Setenv(closedSignupTypesEnv, orig)
					}
				})
			}
			got := closedSignupTypesFromEnv()
			if len(got) != len(tc.want) {
				t.Fatalf("closedSignupTypesFromEnv() = %v, want %v", got, tc.want)
			}
			for name := range tc.want {
				if !got[name] {
					t.Errorf("closedSignupTypesFromEnv() = %v, missing %q", got, name)
				}
			}
		})
	}
}

func TestSignupClosed(t *testing.T) {
	app := closedSignupApp("badut")

	if !app.signupClosed(types.TeamTypeBadut) {
		t.Error("the gøgler signup should be closed")
	}
	// Closing one signup type must leave the others alone.
	for _, open := range []types.TeamType{types.TeamTypePatrulje, types.TeamTypeKlan, types.TeamTypeCrew} {
		if app.signupClosed(open) {
			t.Errorf("%q should be open", open)
		}
	}
	// An empty team type is not a closed one; the create handler treats it as the
	// invalid input it is, further down.
	if app.signupClosed("") {
		t.Error(`"" should not be reported as closed`)
	}
}

func TestSignupClosedWithEmptySet(t *testing.T) {
	app := closedSignupApp()
	if app.signupClosed(types.TeamTypeBadut) {
		t.Error("an empty closed set must leave every signup open")
	}
}

// The front page and the create endpoint must read the same value, or they drift
// apart — which is the bug this replaced: home.go said CLOSED while the endpoint
// accepted gøglers.
func TestSignupStatusAgreesWithSignupClosed(t *testing.T) {
	app := closedSignupApp("badut")

	if got := app.signupStatus(types.TeamTypeBadut); got != "CLOSED" {
		t.Errorf("signupStatus(gøgler) = %q, want CLOSED", got)
	}
	if got := app.signupStatus(types.TeamTypePatrulje); got != "OPEN" {
		t.Errorf("signupStatus(patrulje) = %q, want OPEN", got)
	}

	for _, teamType := range types.TeamTypes {
		closed := app.signupClosed(teamType)
		status := app.signupStatus(teamType)
		if closed != (status == "CLOSED") {
			t.Errorf("%q: signupClosed=%v but signupStatus=%q", teamType, closed, status)
		}
	}

	open := closedSignupApp()
	if got := open.signupStatus(types.TeamTypeBadut); got != "OPEN" {
		t.Errorf("signupStatus(gøgler) = %q with nothing closed, want OPEN", got)
	}
}

// The default must close the gøgler signup, so a deployment that never sets the
// variable fails safe rather than quietly accepting gøglers.
func TestDefaultClosedSignupTypesClosesGogler(t *testing.T) {
	app := &application{}
	app.config.signup.closedTypes = canonicalSignupTypes(defaultClosedSignupTypes)
	if !app.signupClosed(types.TeamTypeBadut) {
		t.Error("the default closed set must close the gøgler signup")
	}
	if app.signupClosed(types.TeamTypePatrulje) {
		t.Error("the default closed set must not close the patrulje signup")
	}
}
