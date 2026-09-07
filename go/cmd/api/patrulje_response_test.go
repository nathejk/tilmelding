package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/patrulje"
	"github.com/nathejk/shared-go/tables/spejder"
	"github.com/nathejk/shared-go/types"
)

// These tests pin the patrulje wire contract. They exist because the handlers
// used to marshal domain types through a map envelope, so the payload was
// whatever the projection happened to hold: any upstream field rename would
// have broken the frontend silently. The golden JSON below is what
// vue/src/views/PatruljeView.vue and vue/src/helpers/order.js read, so a diff
// here means a frontend change is required too.

func TestShowPatruljeResponseWireShape(t *testing.T) {
	cfg := TeamConfig{
		MinMemberCount: 3,
		MaxMemberCount: 7,
		MemberPrice:    450,
		TShirtPrice:    175,
		Korps:          []types.SlugLabel{{Slug: "dds", Label: "DDS"}},
		TShirtSizes:    []types.SlugLabel{{Slug: "", Label: "Ingen"}, {Slug: "l", Label: "Large"}},
		ClosedProducts: []string{"tshirt.adult"},
	}
	// Note: built from the shared-go entity type now, not internal/data. The
	// expected JSON below is unchanged, which is what makes the migration
	// provably wire-safe despite spejder/patrulje carrying different json tags.
	team := &patrulje.Patrulje{
		TeamID: "t-1", TeamNumber: "42", SignupStatus: "started", Name: "Ulvene",
		Group: "1. Aarhus", Korps: "dds", Liga: "a", MemberCount: 4,
	}
	// The contact now comes from the team row (patrulje.GetByID selects the
	// four contactXxx columns), not a second query. `address` and `postal`
	// are empty because the projection has no columns for them — the previous
	// query did not select them either, so this is what the page has always
	// received; only the fixture could pretend otherwise.
	contact := &patrulje.Patrulje{
		TeamID: "t-1", ContactName: "Anna",
		ContactEmail: "a@b.dk", ContactPhone: "40733886", ContactRole: "leder",
	}
	members := []*spejder.Spejder{{
		ID: "m-1", MemberID: "m-1", InitialTeamID: "t-1", CurrentTeamID: "t-1",
		Status: "paid", Name: "Bo", Address: "Vej 2", PostalCode: "8000", City: "Aarhus",
		Email: "bo@b.dk", Phone: "11111111", PhoneParent: "22222222",
		Birthday: types.Date("2010-05-04"), Returning: true, TShirtSize: "l",
	}}
	openOrder := &order.Order{
		OrderID: "o-1", Year: "2026", OwnerType: types.TeamTypePatrulje, OwnerID: "t-1",
		Status: order.StatusOpen, Currency: "DKK",
		TotalAmount: 62500, PaidAmount: 0, DueAmount: 62500,
		CreatedAt: "2026-08-01 10:00:00", ChangedAt: "2026-08-01 10:05:00",
		Lines: []order.Line{{
			LineID: "l-1", ProductSKU: "tshirt.adult", ProductName: "T-shirt",
			MemberID: "m-1", UnitPrice: 17500, Quantity: 1, LineTotal: 17500,
			Origin: "derived", Attributes: map[string]any{"size": "l"},
		}},
	}

	resp := showPatruljeResponse{
		Config:     newTeamConfigResponse(cfg),
		Team:       newPatruljeTeamResponse(team),
		Contact:    newPatruljeContactResponse(contact),
		Members:    newPatruljeRosterMemberResponses(members),
		Order:      newOrderResponse(openOrder),
		PaidOrders: newOrderResponses(nil),
	}

	const want = `{` +
		`"config":{"minMemberCount":3,"maxMemberCount":7,"memberPrice":450,"tshirtPrice":175,` +
		`"korps":[{"slug":"dds","label":"DDS"}],` +
		`"tshirtSizes":[{"slug":"","label":"Ingen"},{"slug":"l","label":"Large"}],` +
		`"closedProducts":["tshirt.adult"],"oversubscribed":false},` +
		`"team":{"id":"t-1","number":"42","status":"started","name":"Ulvene","group":"1. Aarhus","korps":"dds","liga":"a","memberCount":4},` +
		`"contact":{"teamId":"t-1","name":"Anna","address":"","postal":"","email":"a@b.dk","phone":"40733886","role":"leder"},` +
		`"members":[{"id":"m-1","memberId":"m-1","teamId":"t-1","activeTeamId":"t-1","status":"paid","name":"Bo",` +
		`"address":"Vej 2","postalCode":"8000","city":"Aarhus","email":"bo@b.dk","phone":"11111111",` +
		`"phoneContact":"22222222","birthday":"2010-05-04","returning":true,"tshirtSize":"l"}],` +
		`"order":{"orderId":"o-1","year":"2026","ownerType":"patrulje","ownerId":"t-1","status":"open","currency":"DKK",` +
		`"totalAmount":62500,"paidAmount":0,"dueAmount":62500,` +
		`"lines":[{"lineId":"l-1","productSku":"tshirt.adult","productName":"T-shirt","memberId":"m-1",` +
		`"unitPrice":17500,"quantity":1,"lineTotal":17500,"origin":"derived","attributes":{"size":"l"}}],` +
		`"createdAt":"2026-08-01 10:00:00","changedAt":"2026-08-01 10:05:00"},` +
		`"paidOrders":[]}`

	assertJSON(t, resp, want)
}

// The frontend distinguishes null from empty: `data.order || null` and
// `data.paidOrders ? [...] : []`. A missing open order must stay null and the
// roster must stay [] rather than flipping to the other.
func TestShowPatruljeResponseNilAndEmptySemantics(t *testing.T) {
	resp := showPatruljeResponse{
		Config:     newTeamConfigResponse(TeamConfig{}),
		Team:       newPatruljeTeamResponse(nil),
		Contact:    newPatruljeContactResponse(nil),
		Members:    newPatruljeRosterMemberResponses(nil),
		Order:      newOrderResponse(nil),
		PaidOrders: newOrderResponses(nil),
	}

	const want = `{` +
		`"config":{"minMemberCount":0,"maxMemberCount":0,"memberPrice":0,"tshirtPrice":0,"korps":null,"tshirtSizes":null,"closedProducts":[],"oversubscribed":false},` +
		`"team":null,"contact":null,"members":[],"order":null,"paidOrders":[]}`

	assertJSON(t, resp, want)
}

// Mirrors the handler, which always sends paidOrders through
// newOrderResponses: [] rather than null, matching the show response and the
// frontend's `data.paidOrders ? [...] : []`.
func TestUpdatePatruljeResponseWireShape(t *testing.T) {
	resp := updatePatruljeResponse{
		Team:         newPatruljeTeamResponse(&patrulje.Patrulje{TeamID: "t-1", Name: "Ulvene"}),
		Order:        nil,
		PaidOrders:   newOrderResponses(nil),
		PaymentLink:  "",
		PaymentError: "en patrulje skal have mindst 3 spejdere for at kunne betale",
	}
	const want = `{` +
		`"team":{"id":"t-1","number":"","status":"","name":"Ulvene","group":"","korps":"","liga":"","memberCount":0},` +
		`"order":null,"paidOrders":[],"paymentLink":"",` +
		`"paymentError":"en patrulje skal have mindst 3 spejdere for at kunne betale"}`
	assertJSON(t, resp, want)
}

func TestPatruljeMemberMutationResponseWireShape(t *testing.T) {
	resp := patruljeMemberMutationResponse{
		Member: newPatruljeMemberResponse(patrulje.Spejder{
			MemberID: "m-9", Name: "Cille", Address: "Vej 3", PostalCode: "8000",
			Email: "c@b.dk", Phone: "33333333", PhoneContact: "44444444",
			Birthday: types.Date("2011-01-02"), TShirtSize: "m",
		}),
		Order: nil,
	}
	const want = `{"member":{"memberId":"m-9","deleted":false,"name":"Cille","address":"Vej 3",` +
		`"postalCode":"8000","email":"c@b.dk","phone":"33333333","phoneContact":"44444444",` +
		`"birthday":"2011-01-02","tshirtSize":"m"},"order":null}`
	assertJSON(t, resp, want)
}

func TestDeletePatruljeMemberResponseWireShape(t *testing.T) {
	assertJSON(t, deletePatruljeMemberResponse{Order: nil}, `{"order":null}`)
}

func TestAssignNumbersResponseWireShape(t *testing.T) {
	resp := assignNumbersResponse{Assigned: 2, AlreadyNumber: 5, Unpaid: 1}
	assertJSON(t, resp, `{"assigned":2,"alreadyNumbered":5,"unpaid":1}`)
}

// A line with no attributes must omit the key entirely, as the previous
// map[string]any with omitempty did; and an attribute the contract does not
// recognise is dropped rather than forwarded.
func TestOrderLineAttributesOmittedAndNarrowed(t *testing.T) {
	o := &order.Order{Lines: []order.Line{
		{LineID: "a"},
		{LineID: "b", Attributes: map[string]any{}},
		{LineID: "c", Attributes: map[string]any{"memberId": "m-1"}},
		{LineID: "d", Attributes: map[string]any{"size": "xl", "memberId": "m-1"}},
	}}
	got := newOrderResponse(o)
	for i, want := range []*orderLineAttributesResponse{nil, nil, nil, {Size: "xl"}} {
		gotAttrs := got.Lines[i].Attributes
		switch {
		case want == nil && gotAttrs != nil:
			t.Errorf("line %d: expected attributes omitted, got %+v", i, gotAttrs)
		case want != nil && (gotAttrs == nil || gotAttrs.Size != want.Size):
			t.Errorf("line %d: expected size %q, got %+v", i, want.Size, gotAttrs)
		}
	}
}

// nil Lines must stay null, not become []: helpers/order.js guards with
// Array.isArray, so both work, but the contract should not change silently.
func TestOrderResponsePreservesNilLines(t *testing.T) {
	assertJSONContains(t, newOrderResponse(&order.Order{OrderID: "o-1"}), `"lines":null`)
}

func assertJSON(t *testing.T, v any, want string) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != want {
		t.Errorf("wire shape changed.\n got: %s\nwant: %s", b, want)
	}
}

func assertJSONContains(t *testing.T, v any, want string) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), want) {
		t.Errorf("expected %s to contain %s", b, want)
	}
}
