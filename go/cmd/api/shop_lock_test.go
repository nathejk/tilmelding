package main

import (
	"context"
	"errors"
	"testing"

	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/patrulje"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/tables/spejder"
	"nathejk.dk/internal/data"
)

type fakeSpejderRoster struct {
	members []*spejder.Spejder
	err     error
}

func (f fakeSpejderRoster) GetAll(context.Context, spejder.Filter) ([]*spejder.Spejder, spejder.Metadata, error) {
	return f.members, spejder.Metadata{}, f.err
}

type fakeSeniorRoster struct {
	members []*senior.Senior
	err     error
}

func (f fakeSeniorRoster) GetAll(context.Context, senior.Filter) ([]*senior.Senior, error) {
	return f.members, f.err
}

func TestLockedSize(t *testing.T) {
	closed := closedApp("tshirt.adult")
	if got := closed.lockedSize("tshirt.adult", "l", "xxl"); got != "l" {
		t.Errorf("lockedSize(closed) = %q, want the stored %q", got, "l")
	}
	// Clearing a size is a change like any other, and would cancel a shirt
	// somebody may already have paid for.
	if got := closed.lockedSize("tshirt.adult", "l", ""); got != "l" {
		t.Errorf("lockedSize(closed, empty request) = %q, want the stored %q", got, "l")
	}
	// Nothing stored means nothing bought: a closed product cannot be started.
	if got := closed.lockedSize("tshirt.adult", "", "m"); got != "" {
		t.Errorf("lockedSize(closed, nothing stored) = %q, want empty", got)
	}
	// A product that is still for sale is untouched, even while another is closed.
	if got := closed.lockedSize("tshirt.plain", "l", "xxl"); got != "xxl" {
		t.Errorf("lockedSize(open sku) = %q, want the requested %q", got, "xxl")
	}

	open := closedApp()
	if got := open.lockedSize("tshirt.adult", "l", "xxl"); got != "xxl" {
		t.Errorf("lockedSize(open) = %q, want the requested %q", got, "xxl")
	}
	if got := open.lockedSize("tshirt.adult", "l", ""); got != "" {
		t.Errorf("lockedSize(open, empty request) = %q, want it honoured", got)
	}
}

func TestLockPatruljeMemberSize(t *testing.T) {
	roster := fakeSpejderRoster{members: []*spejder.Spejder{
		{MemberID: "m-1", TShirtSize: "l"},
		{MemberID: "m-2", TShirtSize: ""},
	}}

	t.Run("closed: a changed size is discarded", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Spejder: roster}
		m := patrulje.Spejder{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "l" {
			t.Errorf("TShirtSize = %q, want the stored %q", m.TShirtSize, "l")
		}
	})

	t.Run("closed: an emptied size is discarded", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Spejder: roster}
		m := patrulje.Spejder{MemberID: "m-1"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "l" {
			t.Errorf("TShirtSize = %q, want the stored %q", m.TShirtSize, "l")
		}
	})

	t.Run("closed: a new member gets no shirt", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Spejder: roster}
		// No MemberID yet: this is an add, so there is nothing stored to keep.
		m := patrulje.Spejder{TShirtSize: "m"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "" {
			t.Errorf("TShirtSize = %q, want empty for a new member", m.TShirtSize)
		}
	})

	t.Run("closed: a member with no stored shirt cannot start one", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Spejder: roster}
		m := patrulje.Spejder{MemberID: "m-2", TShirtSize: "m"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "" {
			t.Errorf("TShirtSize = %q, want empty", m.TShirtSize)
		}
	})

	t.Run("closed: an unreadable roster fails closed", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Spejder: fakeSpejderRoster{err: errors.New("db down")}}
		m := patrulje.Spejder{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "" {
			t.Errorf("TShirtSize = %q, want empty — a failed read must not license the request", m.TShirtSize)
		}
	})

	t.Run("open: the request is honoured and no roster read happens", func(t *testing.T) {
		app := closedApp()
		// Deliberately no models wired: the open path must not touch the roster,
		// so this would nil-panic if it did.
		m := patrulje.Spejder{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockPatruljeMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "xxl" {
			t.Errorf("TShirtSize = %q, want the requested %q", m.TShirtSize, "xxl")
		}
	})
}

func TestLockKlanMemberSize(t *testing.T) {
	roster := fakeSeniorRoster{members: []*senior.Senior{
		{MemberID: "m-1", TshirtSize: "l"},
	}}

	t.Run("closed: a changed size is discarded", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Senior: roster}
		m := klan.Senior{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockKlanMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "l" {
			t.Errorf("TShirtSize = %q, want the stored %q", m.TShirtSize, "l")
		}
	})

	t.Run("closed: a new senior gets no shirt", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Senior: roster}
		m := klan.Senior{TShirtSize: "m"}
		app.lockKlanMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "" {
			t.Errorf("TShirtSize = %q, want empty for a new senior", m.TShirtSize)
		}
	})

	t.Run("closed: an unreadable roster fails closed", func(t *testing.T) {
		app := closedApp("tshirt.adult")
		app.models = data.Models{Senior: fakeSeniorRoster{err: errors.New("db down")}}
		m := klan.Senior{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockKlanMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "" {
			t.Errorf("TShirtSize = %q, want empty", m.TShirtSize)
		}
	})

	t.Run("open: the request is honoured and no roster read happens", func(t *testing.T) {
		app := closedApp()
		m := klan.Senior{MemberID: "m-1", TShirtSize: "xxl"}
		app.lockKlanMemberSize(context.Background(), "t-1", &m)
		if m.TShirtSize != "xxl" {
			t.Errorf("TShirtSize = %q, want the requested %q", m.TShirtSize, "xxl")
		}
	})
}

// The size lock and the derived lines must name the same product, or a size could
// be frozen while its line kept being billed (or the reverse).
func TestTshirtSKUMatchesTheDerivedLines(t *testing.T) {
	lines := derivedLinesForPatrulje([]patrulje.Spejder{
		{MemberID: "m-1", TShirtSize: "l"},
	})
	found := false
	for _, l := range lines {
		if l.ProductSKU == tshirtSKU {
			found = true
		}
	}
	if !found {
		t.Errorf("derivedLinesForPatrulje emits no %q line; the lock names a SKU the order does not", tshirtSKU)
	}

	app := closedApp(tshirtSKU)
	if !app.tshirtLocked() {
		t.Error("tshirtLocked() = false while tshirt.adult is closed")
	}
	if closedApp().tshirtLocked() {
		t.Error("tshirtLocked() = true with nothing closed")
	}
}
