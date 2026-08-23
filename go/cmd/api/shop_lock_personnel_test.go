package main

import (
	"testing"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/personnel"
)

// The gøgler / personnel size lives in a plain column, so the lock is a straight
// substitution — but it has to happen before both the update command and the
// derived lines, or the two would disagree about what the person ordered. These
// tests pin the substitution and the line shape it produces.

// lockPersonnelSize reproduces what updatePersonnelHandler does, so the rule can
// be tested without an HTTP round trip.
func lockPersonnelSize(app *application, stored *personnel.Staff, edited personnel.Person) personnel.Person {
	if app.tshirtLocked() {
		edited.TshirtSize = app.lockedSize(tshirtSKU, stored.TshirtSize, edited.TshirtSize)
	}
	return edited
}

func TestPersonnelSizeLock(t *testing.T) {
	stored := &personnel.Staff{ID: "u-1", Type: types.TeamTypeBadut, TshirtSize: "l"}

	t.Run("closed: a changed size is discarded", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		got := lockPersonnelSize(app, stored, personnel.Person{TshirtSize: "xxl"})
		if got.TshirtSize != "l" {
			t.Errorf("TshirtSize = %q, want the stored %q", got.TshirtSize, "l")
		}
	})

	t.Run("closed: an emptied size is discarded", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		got := lockPersonnelSize(app, stored, personnel.Person{})
		if got.TshirtSize != "l" {
			t.Errorf("TshirtSize = %q, want the stored %q", got.TshirtSize, "l")
		}
	})

	t.Run("closed: nothing stored cannot become a shirt", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		none := &personnel.Staff{ID: "u-2", Type: types.TeamTypeBadut}
		got := lockPersonnelSize(app, none, personnel.Person{TshirtSize: "m"})
		if got.TshirtSize != "" {
			t.Errorf("TshirtSize = %q, want empty", got.TshirtSize)
		}
	})

	t.Run("open: the request is honoured", func(t *testing.T) {
		app := closedApp()
		got := lockPersonnelSize(app, stored, personnel.Person{TshirtSize: "xxl"})
		if got.TshirtSize != "xxl" {
			t.Errorf("TshirtSize = %q, want the requested %q", got.TshirtSize, "xxl")
		}
	})
}

// The locked size is what the derived lines are built from, for both personnel
// owner types, and it names the SKU the lock names.
func TestDerivedLinesForPersonnelFollowTheLockedSize(t *testing.T) {
	app := closedApp("tshirt.adult")

	for _, tc := range []struct {
		name             string
		teamType         types.TeamType
		participationSKU string
	}{
		{"gøgler", types.TeamTypeBadut, "participation.gogler"},
		{"crew", types.TeamTypeCrew, "participation.crew"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stored := &personnel.Staff{ID: "u-1", Type: tc.teamType, TshirtSize: "l"}
			edited := lockPersonnelSize(app, stored, personnel.Person{TshirtSize: "xxl"})

			lines := derivedLinesForPersonnel(stored, edited)

			var participation, shirt int
			for _, l := range lines {
				switch l.ProductSKU {
				case tc.participationSKU:
					participation++
				case tshirtSKU:
					shirt++
					if size, _ := l.Attributes["size"].(string); size != "l" {
						t.Errorf("t-shirt line size = %q, want the stored %q", size, "l")
					}
				default:
					t.Errorf("unexpected line %q", l.ProductSKU)
				}
			}
			if participation != 1 {
				t.Errorf("got %d participation lines, want 1", participation)
			}
			if shirt != 1 {
				t.Errorf("got %d t-shirt lines, want 1", shirt)
			}
		})
	}
}

// Somebody who never ordered a shirt gets no shirt line, so their order is
// participation only and nothing about the closed product appears on it.
func TestDerivedLinesForPersonnelWithoutAShirt(t *testing.T) {
	app := closedApp("tshirt.adult")
	stored := &personnel.Staff{ID: "u-2", Type: types.TeamTypeBadut}
	edited := lockPersonnelSize(app, stored, personnel.Person{TshirtSize: "m"})

	for _, l := range derivedLinesForPersonnel(stored, edited) {
		if l.ProductSKU == tshirtSKU {
			t.Errorf("emitted a t-shirt line for somebody who never ordered one: %+v", l)
		}
	}
}
