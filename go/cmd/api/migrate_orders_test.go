package main

import (
	"testing"
)

// klanReservationLines is the fallback that lets a klan which paid for seats but
// never registered a senior be back-filled. Three things about it are load
// bearing, and none of them are obvious from the call site:
//
//   - the line ids and member ids must match what requestSeatHandler emits, or
//     the projector's snapshot replace leaves orphan lines behind;
//   - the total must equal seats × price, because the migration compares it to
//     what was actually paid and truncates otherwise;
//   - the SKU must be participation.klan, because ApplyPaidOffset offsets by
//     SKU, and a wrong one would silently re-bill the team.
func TestKlanReservationLines(t *testing.T) {
	lines, total := klanReservationLines(4)

	if len(lines) != 4 {
		t.Fatalf("got %d lines, want one per seat", len(lines))
	}
	if want := 4 * migratePriceParticipationKlan; total != want {
		t.Errorf("total = %d, want %d", total, want)
	}

	for i, l := range lines {
		// Same ids the runtime uses at reservation time.
		if want := reservationLineID(uint32(i)); l.LineID != want {
			t.Errorf("line %d id = %q, want %q", i, l.LineID, want)
		}
		if want := pendingMemberID(uint32(i + 1)); l.MemberID != want {
			t.Errorf("line %d memberId = %q, want %q", i, l.MemberID, want)
		}
		if l.ProductSKU != "participation.klan" {
			t.Errorf("line %d sku = %q; ApplyPaidOffset offsets by sku, so this must match what the runtime bills", i, l.ProductSKU)
		}
		if l.Quantity != 1 || l.LineTotal != migratePriceParticipationKlan {
			t.Errorf("line %d = qty %d total %d, want 1 seat at %d", i, l.Quantity, l.LineTotal, migratePriceParticipationKlan)
		}
		if l.Origin != "derived" {
			t.Errorf("line %d origin = %q, want derived", i, l.Origin)
		}
	}
}

// Line ids must be distinct or the projector's INSERT collapses them into one
// row and the team is billed for a single seat.
func TestKlanReservationLinesHaveDistinctIDs(t *testing.T) {
	lines, _ := klanReservationLines(4)
	seen := map[string]bool{}
	for _, l := range lines {
		if seen[l.LineID] {
			t.Fatalf("duplicate line id %q", l.LineID)
		}
		seen[l.LineID] = true
	}
}

func TestKlanReservationLinesZeroSeats(t *testing.T) {
	lines, total := klanReservationLines(0)
	if len(lines) != 0 || total != 0 {
		t.Errorf("got %d lines / total %d, want nothing for zero seats", len(lines), total)
	}
}

// The placeholder ids are the reporting convention recorded under task 009: any
// future "members per order" report excludes them with memberId NOT LIKE
// 'pending-%'. Assert the prefix so that stays true.
func TestKlanReservationLinesUsePendingPrefix(t *testing.T) {
	lines, _ := klanReservationLines(2)
	for _, l := range lines {
		if len(l.MemberID) < 8 || l.MemberID[:8] != "pending-" {
			t.Errorf("memberId %q must keep the pending- prefix", l.MemberID)
		}
	}
}
