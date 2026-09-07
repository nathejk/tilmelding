package main

import (
	"testing"

	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
)

// These tests pin the klan wire contract, for the same reason as the patrulje
// ones: the handlers used to marshal domain types through a map envelope, so
// the payload was whatever the projection happened to hold. The golden JSON
// below is what vue/src/views/KlanView.vue and vue/src/helpers/order.js read,
// so a diff here means a frontend change is required too.
//
// assertJSON / assertJSONContains live in patrulje_response_test.go.

func TestShowKlanResponseWireShape(t *testing.T) {
	cfg := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 4,
		MemberPrice:    450,
		TShirtPrice:    175,
		Korps:          []types.SlugLabel{{Slug: "dds", Label: "DDS"}},
		TShirtSizes:    []types.SlugLabel{{Slug: "", Label: "Ingen"}, {Slug: "l", Label: "Large"}},
		ClosedProducts: []string{"tshirt.adult"},
	}
	// Note: built from the shared-go entity types, not internal/data. The
	// expected JSON below is unchanged from when it was, which is what makes
	// the migration provably wire-safe despite klan.Klan and senior.Senior
	// carrying different field names and json tags.
	team := &klan.Klan{
		ID: "t-1", Status: "PAY", Name: "Banditterne", Group: "1. Aarhus",
		Korps: "dds", MemberCount: 2,
	}
	members := []*senior.Senior{{
		MemberID: "m-1", TeamID: "t-1", Name: "Bo", Address: "Vej 2",
		PostalCode: "8000", City: "Aarhus", Email: "bo@b.dk", Phone: "11111111",
		Birthday: "1990-05-04", Diet: "vegetar", TshirtSize: "l",
	}}
	openOrder := &order.Order{
		OrderID: "o-1", Year: "2026", OwnerType: types.TeamTypeKlan, OwnerID: "t-1",
		Status: order.StatusOpen, Currency: "DKK",
		TotalAmount: 62500, PaidAmount: 0, DueAmount: 62500,
		CreatedAt: "2026-08-01 10:00:00", ChangedAt: "2026-08-01 10:05:00",
		Lines: []order.Line{{
			LineID: "l-1", ProductSKU: "tshirt.adult", ProductName: "T-shirt",
			MemberID: "m-1", UnitPrice: 17500, Quantity: 1, LineTotal: 17500,
			Origin: "derived", Attributes: map[string]any{"size": "l"},
		}},
	}

	resp := showKlanResponse{
		Config:     newTeamConfigResponse(cfg),
		Team:       newKlanTeamResponse(team),
		Members:    newKlanRosterMemberResponses(members),
		Order:      newOrderResponse(openOrder),
		PaidOrders: newOrderResponses(nil),
	}

	const want = `{` +
		`"config":{"minMemberCount":1,"maxMemberCount":4,"memberPrice":450,"tshirtPrice":175,` +
		`"korps":[{"slug":"dds","label":"DDS"}],` +
		`"tshirtSizes":[{"slug":"","label":"Ingen"},{"slug":"l","label":"Large"}],` +
		`"closedProducts":["tshirt.adult"],"oversubscribed":false},` +
		`"team":{"id":"t-1","status":"PAY","name":"Banditterne","group":"1. Aarhus","korps":"dds","memberCount":2},` +
		`"members":[{"id":"m-1","memberId":"m-1","teamId":"t-1","name":"Bo","address":"Vej 2",` +
		`"postalCode":"8000","city":"Aarhus","email":"bo@b.dk","phone":"11111111",` +
		`"birthday":"1990-05-04","diet":"vegetar","tshirtSize":"l"}],` +
		`"order":{"orderId":"o-1","year":"2026","ownerType":"klan","ownerId":"t-1","status":"open","currency":"DKK",` +
		`"totalAmount":62500,"paidAmount":0,"dueAmount":62500,` +
		`"lines":[{"lineId":"l-1","productSku":"tshirt.adult","productName":"T-shirt","memberId":"m-1",` +
		`"unitPrice":17500,"quantity":1,"lineTotal":17500,"origin":"derived","attributes":{"size":"l"}}],` +
		`"createdAt":"2026-08-01 10:00:00","changedAt":"2026-08-01 10:05:00"},` +
		`"paidOrders":[]}`

	assertJSON(t, resp, want)
}

// The frontend distinguishes null from empty: `data.order || null` and
// `data.paidOrders ? [...] : []`. The roster in particular must stay [] — the
// show handler ignores the roster read error and KlanView.vue then calls
// `data.members.map(...)` unguarded, so null would break the page.
func TestShowKlanResponseNilAndEmptySemantics(t *testing.T) {
	resp := showKlanResponse{
		Config:     newTeamConfigResponse(TeamConfig{}),
		Team:       newKlanTeamResponse(nil),
		Members:    newKlanRosterMemberResponses(nil),
		Order:      newOrderResponse(nil),
		PaidOrders: newOrderResponses(nil),
	}

	const want = `{` +
		`"config":{"minMemberCount":0,"maxMemberCount":0,"memberPrice":0,"tshirtPrice":0,"korps":null,"tshirtSizes":null,"closedProducts":[],"oversubscribed":false},` +
		`"team":null,"members":[],"order":null,"paidOrders":[]}`

	assertJSON(t, resp, want)
}

// The waiting-list branch: no seats reserved, so no order and no payment link,
// and the status the page checks (`data.team.status == 'HOLD'`) must be a plain
// string.
func TestRequestSeatResponseWireShape(t *testing.T) {
	resp := requestSeatResponse{
		Team:        newKlanTeamResponse(&klan.Klan{ID: "t-1", Status: types.SignupStatusOnHold, Name: "Banditterne"}),
		Status:      string(types.SignupStatusOnHold),
		Order:       nil,
		PaymentLink: "",
	}
	const want = `{` +
		`"team":{"id":"t-1","status":"HOLD","name":"Banditterne","group":"","korps":"","memberCount":0},` +
		`"status":"HOLD","order":null,"paymentLink":""}`
	assertJSON(t, resp, want)
}

// Mirrors the handler, which always sends paidOrders through
// newOrderResponses: [] rather than null, matching the show response and the
// frontend's `data.paidOrders ? [...] : []`.
func TestUpdateKlanResponseWireShape(t *testing.T) {
	resp := updateKlanResponse{
		Team:         newKlanTeamResponse(&klan.Klan{ID: "t-1", Name: "Banditterne"}),
		Order:        nil,
		PaidOrders:   newOrderResponses(nil),
		PaymentLink:  "",
		PaymentError: "en klan skal have mindst 1 seniorer for at kunne betale",
	}
	const want = `{` +
		`"team":{"id":"t-1","status":"","name":"Banditterne","group":"","korps":"","memberCount":0},` +
		`"order":null,"paidOrders":[],"paymentLink":"",` +
		`"paymentError":"en klan skal have mindst 1 seniorer for at kunne betale"}`
	assertJSON(t, resp, want)
}

// The echoed member is pushed straight into the roster array by the page, so
// its keys must be the ones the roster table binds to.
func TestKlanMemberMutationResponseWireShape(t *testing.T) {
	resp := klanMemberMutationResponse{
		Member: newKlanMemberResponse(klan.Senior{
			MemberID: "m-9", Name: "Cille", Address: "Vej 3", PostalCode: "8000",
			Email: "c@b.dk", Phone: "33333333", Birthday: types.Date("1991-01-02"),
			Diet: "vegetar", TShirtSize: "m",
		}),
		Order: nil,
	}
	const want = `{"member":{"memberId":"m-9","deleted":false,"name":"Cille","address":"Vej 3",` +
		`"postalCode":"8000","email":"c@b.dk","phone":"33333333","birthday":"1991-01-02",` +
		`"diet":"vegetar","tshirtSize":"m"},"order":null}`
	assertJSON(t, resp, want)
}

func TestDeleteKlanMemberResponseWireShape(t *testing.T) {
	assertJSON(t, deleteKlanMemberResponse{Order: nil}, `{"order":null}`)
}
