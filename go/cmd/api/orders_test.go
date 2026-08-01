package main

import (
	"testing"

	"nathejk.dk/nathejk/table/order"
)

// TestReplaceMemberLines covers the single-member reconciliation the member
// endpoints use to re-derive an order on top of the (possibly lagging)
// projection: add/update replaces a member's lines, delete removes them.
func TestReplaceMemberLines(t *testing.T) {
	base := []order.DesiredLine{
		{ProductSKU: "participation.patrulje", MemberID: "a"},
		{ProductSKU: "participation.patrulje", MemberID: "b"},
		{ProductSKU: "tshirt.adult", MemberID: "b"},
	}
	countMember := func(lines []order.DesiredLine, id string) int {
		n := 0
		for _, l := range lines {
			if l.MemberID == id {
				n++
			}
		}
		return n
	}

	t.Run("delete removes the member's lines", func(t *testing.T) {
		got := replaceMemberLines(base, "b", nil)
		if countMember(got, "b") != 0 {
			t.Errorf("member b should be gone, got %d lines", countMember(got, "b"))
		}
		if countMember(got, "a") != 1 {
			t.Errorf("member a should remain, got %d lines", countMember(got, "a"))
		}
		if len(got) != 1 {
			t.Errorf("want 1 line total, got %d", len(got))
		}
	})

	t.Run("update replaces the member's lines (t-shirt removed)", func(t *testing.T) {
		repl := []order.DesiredLine{{ProductSKU: "participation.patrulje", MemberID: "b"}}
		got := replaceMemberLines(base, "b", repl)
		if countMember(got, "b") != 1 {
			t.Errorf("member b should have exactly 1 line after update, got %d", countMember(got, "b"))
		}
		if len(got) != 2 {
			t.Errorf("want 2 lines total (a + new b), got %d", len(got))
		}
	})

	t.Run("add appends a new member's lines", func(t *testing.T) {
		repl := []order.DesiredLine{{ProductSKU: "participation.patrulje", MemberID: "c"}}
		got := replaceMemberLines(base, "c", repl)
		if countMember(got, "c") != 1 {
			t.Errorf("member c should be added, got %d", countMember(got, "c"))
		}
		if len(got) != len(base)+1 {
			t.Errorf("want %d lines, got %d", len(base)+1, len(got))
		}
	})
}
