package main

import (
	"os"
	"testing"
	"time"

	"github.com/nathejk/shared-go/types"
)

// oversubscribedSignupApp is an application whose only configured behaviour is
// the oversubscribed set.
func oversubscribedSignupApp(names ...string) *application {
	app := &application{}
	app.config.signup.oversubscribedTypes = canonicalSignupTypes(names)
	return app
}

func TestOversubscribedSignupTypesFromEnv(t *testing.T) {
	tests := []struct {
		name string
		// set reports whether the variable exists at all; an unset variable and one
		// set to "" must mean different things.
		set   bool
		value string
		want  map[string]bool
	}{
		{
			name: "unset falls back to the default, which is patrulje",
			set:  false,
			want: map[string]bool{"patrulje": true},
		},
		{
			// The escape hatch: re-opening next year without a code change.
			name:  "set but empty fills nothing",
			set:   true,
			value: "",
			want:  map[string]bool{},
		},
		{
			// Shares canonicalSignupTypes with the closed set, so the badut/gøgler
			// alias works here too.
			name:  "aliases resolve, whitespace and a trailing separator are tolerated",
			set:   true,
			value: " patrulje , badut , ",
			want:  map[string]bool{"patrulje": true, "gøgler": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(oversubscribedSignupTypesEnv, tc.value)
			} else {
				// t.Setenv cannot unset, and the variable may be present in the
				// developer's environment, so restore it by hand.
				orig, had := os.LookupEnv(oversubscribedSignupTypesEnv)
				os.Unsetenv(oversubscribedSignupTypesEnv)
				t.Cleanup(func() {
					if had {
						os.Setenv(oversubscribedSignupTypesEnv, orig)
					}
				})
			}
			got := oversubscribedSignupTypesFromEnv()
			if len(got) != len(tc.want) {
				t.Fatalf("oversubscribedSignupTypesFromEnv() = %v, want %v", got, tc.want)
			}
			for name := range tc.want {
				if !got[name] {
					t.Errorf("oversubscribedSignupTypesFromEnv() = %v, missing %q", got, name)
				}
			}
		})
	}
}

// Being full must close the signup as well, or the front page would keep its
// button enabled and the create endpoint would keep accepting teams.
func TestOversubscribedImpliesClosed(t *testing.T) {
	app := oversubscribedSignupApp("patrulje")

	if !app.signupOversubscribed(types.TeamTypePatrulje) {
		t.Error("patrulje should be oversubscribed")
	}
	if !app.signupClosed(types.TeamTypePatrulje) {
		t.Error("an oversubscribed signup must also be closed")
	}
	if got := app.signupStatus(types.TeamTypePatrulje); got != "CLOSED" {
		t.Errorf("signupStatus(patrulje) = %q, want CLOSED", got)
	}
	// Filling one team type must leave the others alone.
	for _, open := range []types.TeamType{types.TeamTypeKlan, types.TeamTypeCrew, types.TeamTypeBadut} {
		if app.signupClosed(open) {
			t.Errorf("%q should be open", open)
		}
	}
}

// The refusal text is the difference between the two sets: a full event says so
// in the words the team page also uses, an administratively closed one does not.
func TestSignupRefusalMessage(t *testing.T) {
	full := oversubscribedSignupApp("patrulje")
	if got := full.signupRefusalMessage(types.TeamTypePatrulje); got != signupOversubscribedMessage {
		t.Errorf("signupRefusalMessage(patrulje) = %q, want the overtegnet message", got)
	}

	closed := closedSignupApp("badut")
	if got := closed.signupRefusalMessage(types.TeamTypeBadut); got != signupClosedMessage {
		t.Errorf("signupRefusalMessage(gøgler) = %q, want %q", got, signupClosedMessage)
	}
}

// The default must fill the patrulje signup, so a deployment that never sets the
// variable — including one that already pins CLOSED_SIGNUP_TYPES to something
// else — fails safe rather than selling places that do not exist.
func TestDefaultOversubscribedSignupTypesClosesPatrulje(t *testing.T) {
	app := &application{}
	app.config.signup.oversubscribedTypes = canonicalSignupTypes(defaultOversubscribedSignupTypes)
	if !app.signupOversubscribed(types.TeamTypePatrulje) {
		t.Error("the default oversubscribed set must close the patrulje signup")
	}
	if app.signupOversubscribed(types.TeamTypeKlan) {
		t.Error("the default oversubscribed set must not touch the klan signup")
	}
}

// The default oversubscribed set closes the front door. The cut-off decides who is
// already inside, and getting that wrong takes a paid-up team's roster away — so it
// fails open in every uncertain case.
func TestOversubscribedSinceFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		value    string
		wantZero bool
	}{
		{name: "unset falls back to the close date", set: false},
		{name: "a valid RFC 3339 value is used", set: true, value: "2026-09-01T12:00:00+02:00"},
		{name: "empty means no cut-off, so no page is locked", set: true, value: "", wantZero: true},
		{name: "whitespace only is empty too", set: true, value: "   ", wantZero: true},
		{name: "garbage fails open rather than locking everything", set: true, value: "last tuesday", wantZero: true},
		{name: "a date without a zone is not RFC 3339 and fails open", set: true, value: "2026-09-07", wantZero: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(oversubscribedSinceEnv, tc.value)
			} else {
				orig, had := os.LookupEnv(oversubscribedSinceEnv)
				os.Unsetenv(oversubscribedSinceEnv)
				t.Cleanup(func() {
					if had {
						os.Setenv(oversubscribedSinceEnv, orig)
					}
				})
			}
			got := oversubscribedSinceFromEnv()
			if got.IsZero() != tc.wantZero {
				t.Fatalf("oversubscribedSinceFromEnv() = %v, wantZero=%v", got, tc.wantZero)
			}
			if tc.set && !tc.wantZero {
				want, _ := time.Parse(time.RFC3339, tc.value)
				if !got.Equal(want) {
					t.Errorf("oversubscribedSinceFromEnv() = %v, want %v", got, want)
				}
			}
		})
	}

	// The default itself must parse, or the shipped configuration silently locks
	// nothing at all.
	if _, err := time.Parse(time.RFC3339, defaultOversubscribedSince); err != nil {
		t.Errorf("defaultOversubscribedSince %q is not RFC 3339: %v", defaultOversubscribedSince, err)
	}
}

// signup.createdAt is a VARCHAR holding time.Time's String() output, not RFC 3339,
// because the projection writes it with %q from the message time. Reading it wrong
// would put every team on the wrong side of the cut-off.
func TestParseSignupCreatedAt(t *testing.T) {
	want := time.Date(2026, 9, 6, 14, 30, 5, 0, time.FixedZone("CEST", 2*60*60))

	ok := []struct {
		name  string
		value string
	}{
		{"time.Time String() as the projection writes it", "2026-09-06 14:30:05 +0200 CEST"},
		{"with fractional seconds", "2026-09-06 14:30:05.123 +0200 CEST"},
		{"with a monotonic reading appended", "2026-09-06 14:30:05 +0200 CEST m=+0.000000001"},
		{"RFC 3339, in case the projection ever switches", "2026-09-06T14:30:05+02:00"},
		{"surrounding whitespace", "  2026-09-06 14:30:05 +0200 CEST  "},
	}
	for _, tc := range ok {
		t.Run(tc.name, func(t *testing.T) {
			got, parsed := parseSignupCreatedAt(tc.value)
			if !parsed {
				t.Fatalf("parseSignupCreatedAt(%q) reported failure", tc.value)
			}
			// Compared as instants: the fractional-seconds case is 123ms later, which
			// is irrelevant to a cut-off measured in days.
			if got.Sub(want) > time.Second || want.Sub(got) > time.Second {
				t.Errorf("parseSignupCreatedAt(%q) = %v, want ~%v", tc.value, got, want)
			}
		})
	}

	// An unreadable value must be reported as unknown, never guessed at: the caller
	// leaves the page open on false, and a wrong guess either locks an admitted team
	// out or lets a refused one in.
	for _, value := range []string{"", "   ", "0000-00-00 00:00:00", "whenever"} {
		if _, parsed := parseSignupCreatedAt(value); parsed {
			t.Errorf("parseSignupCreatedAt(%q) should have reported failure", value)
		}
	}
}

// The team config is what the page reads, and the flag must be per team rather than
// per type — the 185 patruljer that signed up before the close keep the page in full.
func TestTeamConfigCarriesOversubscribed(t *testing.T) {
	cfg := TeamConfig{Oversubscribed: true}
	if !newTeamConfigResponse(cfg).Oversubscribed {
		t.Error("teamConfigResponse must carry Oversubscribed through to the wire")
	}
	if newTeamConfigResponse(TeamConfig{}).Oversubscribed {
		t.Error("an unset TeamConfig must not report the event as full")
	}
}
