package main

import (
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/types"
)

// Response DTOs — the explicit frontend/backend contract.
//
// Handlers must not marshal domain types straight onto the wire. Doing so makes
// the payload an accident of whatever the projection or shared-go entity happens
// to hold: a field added upstream leaks to the client, a renamed one breaks it
// silently, and nothing in this repo states what the frontend is entitled to.
//
// Every type here is built from primitives (string / int / bool) and nested
// structs of primitives only. No domain types, no `map[string]any`, no
// `types.*` aliases — a reader can see the exact JSON without chasing
// definitions across two modules.
//
// The DTOs in this file are the pieces shared by several resources (orders and
// the team config). Resource-specific shapes and the per-handler top-level
// response structs live next to their handlers, e.g. in patrulje.go.
//
// Converters preserve nil-vs-empty exactly: a nil pointer stays JSON `null` and
// a nil slice stays `null`, because the frontend distinguishes them
// (`data.order || null`, `Array.isArray(order.lines)`). These conversions were
// introduced to be wire-identical to the previous envelope output.

// slugLabelResponse is a selectable option: the value stored and the text shown.
type slugLabelResponse struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// teamConfigResponse tells the UI the server's own rules — member bounds,
// prices in DKK, the permitted corps / t-shirt sizes, and which products are
// closed for sale — so both sides agree on what is valid rather than the
// frontend hard-coding it.
type teamConfigResponse struct {
	MinMemberCount int                 `json:"minMemberCount"`
	MaxMemberCount int                 `json:"maxMemberCount"`
	MemberPrice    int                 `json:"memberPrice"`
	TShirtPrice    int                 `json:"tshirtPrice"`
	Korps          []slugLabelResponse `json:"korps"`
	TShirtSizes    []slugLabelResponse `json:"tshirtSizes"`
	// ClosedProducts names the SKUs that may no longer be bought — the year
	// t-shirt once the shirts are in production. Their price and sizes stay in
	// this config, because what somebody already bought still has to be
	// rendered; what changes is that the frontend must offer no way to buy or
	// re-size one. Never null: an empty list means everything is for sale, so
	// the client can test membership without a nil check.
	ClosedProducts []string `json:"closedProducts"`
}

// orderLineAttributesResponse is the typed replacement for the line's
// `map[string]any`. Only "size" is ever set (by the derived-line builders for
// tshirt.adult) and only "size" is read by the frontend, so the contract states
// that instead of an open map. A new attribute must be added here explicitly.
type orderLineAttributesResponse struct {
	Size string `json:"size,omitempty"`
}

type orderLineResponse struct {
	LineID      string `json:"lineId"`
	ProductSKU  string `json:"productSku"`
	ProductName string `json:"productName"`
	MemberID    string `json:"memberId"`
	UnitPrice   int    `json:"unitPrice"`
	// Quantity and LineTotal are non-zero and may be **negative**. A t-shirt
	// size change on an already-paid shirt is free but not invisible: it is
	// recorded as a pair of lines, one negative for the size handed back and
	// one positive for the size now wanted, which sum to zero. Clients must
	// therefore sum these fields rather than assume they are positive — see
	// shared-go PRD 001 (this repo's PRD 002). UnitPrice stays positive; the
	// sign lives on the quantity.
	Quantity  int    `json:"quantity"`
	LineTotal int    `json:"lineTotal"`
	Origin    string `json:"origin"`
	// Omitted entirely when the line carries no attributes, matching the
	// previous `map[string]any` with omitempty.
	Attributes *orderLineAttributesResponse `json:"attributes,omitempty"`
}

// orderResponse is an order as the UI needs it. Amounts are in minor units
// (øre); the frontend converts for display (see vue/src/helpers/order.js).
type orderResponse struct {
	OrderID      string              `json:"orderId"`
	Year         string              `json:"year"`
	OwnerType    string              `json:"ownerType"`
	OwnerID      string              `json:"ownerId"`
	Status       string              `json:"status"`
	Currency     string              `json:"currency"`
	TotalAmount  int                 `json:"totalAmount"`
	PaidAmount   int                 `json:"paidAmount"`
	DueAmount    int                 `json:"dueAmount"`
	Lines        []orderLineResponse `json:"lines"`
	CancelReason string              `json:"cancelReason,omitempty"`
	CreatedAt    string              `json:"createdAt"`
	ChangedAt    string              `json:"changedAt"`
}

func newTeamConfigResponse(c TeamConfig) teamConfigResponse {
	closed := c.ClosedProducts
	if closed == nil {
		closed = []string{}
	}
	return teamConfigResponse{
		MinMemberCount: c.MinMemberCount,
		MaxMemberCount: c.MaxMemberCount,
		MemberPrice:    c.MemberPrice,
		TShirtPrice:    c.TShirtPrice,
		Korps:          newSlugLabelResponses(c.Korps),
		TShirtSizes:    newSlugLabelResponses(c.TShirtSizes),
		ClosedProducts: closed,
	}
}

func newSlugLabelResponses(in []types.SlugLabel) []slugLabelResponse {
	if in == nil {
		return nil
	}
	out := make([]slugLabelResponse, 0, len(in))
	for _, s := range in {
		out = append(out, slugLabelResponse{Slug: s.Slug, Label: s.Label})
	}
	return out
}

// newOrderResponse converts an order, preserving nil as JSON null: the frontend
// treats a missing open order as "nothing to pay" (`data.order || null`).
func newOrderResponse(o *order.Order) *orderResponse {
	if o == nil {
		return nil
	}
	return &orderResponse{
		OrderID:      o.OrderID,
		Year:         string(o.Year),
		OwnerType:    string(o.OwnerType),
		OwnerID:      o.OwnerID,
		Status:       string(o.Status),
		Currency:     o.Currency,
		TotalAmount:  o.TotalAmount,
		PaidAmount:   o.PaidAmount,
		DueAmount:    o.DueAmount,
		Lines:        newOrderLineResponses(o.Lines),
		CancelReason: o.CancelReason,
		CreatedAt:    o.CreatedAt,
		ChangedAt:    o.ChangedAt,
	}
}

// newOrderResponses always returns a non-nil slice, so `paidOrders` stays `[]`
// rather than `null` — loadOrders never returns nil and the UI spreads it.
func newOrderResponses(in []order.Order) []orderResponse {
	out := make([]orderResponse, 0, len(in))
	for i := range in {
		out = append(out, *newOrderResponse(&in[i]))
	}
	return out
}

func newOrderLineResponses(in []order.Line) []orderLineResponse {
	if in == nil {
		return nil
	}
	out := make([]orderLineResponse, 0, len(in))
	for _, l := range in {
		out = append(out, orderLineResponse{
			LineID:      l.LineID,
			ProductSKU:  l.ProductSKU,
			ProductName: l.ProductName,
			MemberID:    l.MemberID,
			UnitPrice:   l.UnitPrice,
			Quantity:    l.Quantity,
			LineTotal:   l.LineTotal,
			Origin:      l.Origin,
			Attributes:  newOrderLineAttributesResponse(l.Attributes),
		})
	}
	return out
}

// newOrderLineAttributesResponse narrows the line's open attribute map to the
// one key the contract recognises. An unknown key is dropped rather than
// forwarded, which is the point of having a contract; add a field above to
// expose a new one.
func newOrderLineAttributesResponse(attrs map[string]any) *orderLineAttributesResponse {
	if len(attrs) == 0 {
		return nil
	}
	size, _ := attrs["size"].(string)
	if size == "" {
		return nil
	}
	return &orderLineAttributesResponse{Size: size}
}
