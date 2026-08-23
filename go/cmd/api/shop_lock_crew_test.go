package main

import (
	"encoding/json"
	"testing"

	"github.com/nathejk/shared-go/tables/crewmember"
)

// The crew t-shirt size has no column of its own: it lives in the crewmember
// additionals blob under tshirtSizeKey. updateCrewHandler folds the request value
// in, and *deletes* the key when the request carries an empty size — which is the
// one way this change could destroy paid-shirt data. These tests pin the fold
// behaviour the handler depends on.

// foldTshirtSize reproduces the handler's fold, so the interaction between the
// lock and the delete branch can be tested without an HTTP round trip.
func foldTshirtSize(app *application, additionals map[string]any, stored, requested string) map[string]any {
	if app.tshirtLocked() {
		requested = app.lockedSize(tshirtSKU, stored, requested)
	}
	if additionals == nil {
		additionals = map[string]any{}
	}
	if requested != "" {
		additionals[tshirtSizeKey] = requested
	} else {
		delete(additionals, tshirtSizeKey)
	}
	return additionals
}

// The trap: a save carrying an empty size must not erase a stored one while the
// shirt is closed. A stale page, or one whose picker task 035 removed, sends
// exactly this.
func TestCrewFoldDoesNotEraseAStoredSizeWhileClosed(t *testing.T) {
	app := closedApp("tshirt.adult")

	additionals := foldTshirtSize(app, map[string]any{"allergy": "nuts"}, "l", "")
	if got := additionals[tshirtSizeKey]; got != "l" {
		t.Errorf("additionals[%q] = %v, want %q — an empty save must not cancel a paid shirt", tshirtSizeKey, got, "l")
	}
	// Unrelated keys survive the substitution.
	if got := additionals["allergy"]; got != "nuts" {
		t.Errorf("additionals[\"allergy\"] = %v, want \"nuts\"", got)
	}
}

func TestCrewFoldKeepsTheStoredSizeWhenAChangeIsRequested(t *testing.T) {
	app := closedApp("tshirt.adult")
	additionals := foldTshirtSize(app, nil, "l", "xxl")
	if got := additionals[tshirtSizeKey]; got != "l" {
		t.Errorf("additionals[%q] = %v, want the stored %q", tshirtSizeKey, got, "l")
	}
}

// Somebody with no shirt cannot acquire one while the sale is shut, and the key
// stays absent rather than being written empty.
func TestCrewFoldCannotStartAShirtWhileClosed(t *testing.T) {
	app := closedApp("tshirt.adult")
	additionals := foldTshirtSize(app, nil, "", "m")
	if _, ok := additionals[tshirtSizeKey]; ok {
		t.Errorf("additionals = %v, want no %q key", additionals, tshirtSizeKey)
	}
}

// With the sale open, the delete branch must still work: clearing a size is a
// legitimate edit.
func TestCrewFoldStillClearsWhenOpen(t *testing.T) {
	app := closedApp()
	additionals := foldTshirtSize(app, map[string]any{tshirtSizeKey: "l"}, "l", "")
	if _, ok := additionals[tshirtSizeKey]; ok {
		t.Errorf("additionals = %v, want the key removed when the sale is open", additionals)
	}

	additionals = foldTshirtSize(app, nil, "l", "xxl")
	if got := additionals[tshirtSizeKey]; got != "xxl" {
		t.Errorf("additionals[%q] = %v, want the requested %q", tshirtSizeKey, got, "xxl")
	}
}

// crewMemberToView is how the handler reads the stored size back out of the blob;
// if this drifts, the lock would compare against the wrong value.
func TestCrewMemberToViewReadsTheStoredSize(t *testing.T) {
	blob, err := json.Marshal(map[string]any{tshirtSizeKey: "xl", "allergy": "nuts"})
	if err != nil {
		t.Fatal(err)
	}
	view := crewMemberToView(&crewmember.CrewMember{UserID: "u-1", Additionals: string(blob)})
	if view.TshirtSize != "xl" {
		t.Errorf("TshirtSize = %q, want %q", view.TshirtSize, "xl")
	}

	// No additionals at all must read as "no shirt", not as a parse failure that
	// silently licenses the requested size.
	view = crewMemberToView(&crewmember.CrewMember{UserID: "u-1"})
	if view.TshirtSize != "" {
		t.Errorf("TshirtSize = %q, want empty", view.TshirtSize)
	}
}

// The derived crew lines must agree with the SKU the lock names.
func TestDerivedLinesForCrewUsesTheLockedSKU(t *testing.T) {
	lines := derivedLinesForCrew("u-1", "l")
	found := false
	for _, l := range lines {
		if l.ProductSKU == tshirtSKU {
			found = true
		}
	}
	if !found {
		t.Errorf("derivedLinesForCrew emits no %q line", tshirtSKU)
	}

	// No size, no shirt line — the shape the lock produces for somebody who never
	// ordered one.
	for _, l := range derivedLinesForCrew("u-1", "") {
		if l.ProductSKU == tshirtSKU {
			t.Error("derivedLinesForCrew emitted a t-shirt line for an empty size")
		}
	}
}
