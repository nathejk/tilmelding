package messages

import "nathejk.dk/pkg/types"

// nathejk:member.updated
type NathejkMemberUpdated struct {
	MemberID    types.MemberID    `json:"memberId"`
	TeamID      types.TeamID      `json:"teamId"`
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	PostalCode  string            `json:"postalCode"`
	City        string            `json:"city"`
	Email       types.Email       `json:"mail"`
	Phone       types.PhoneNumber `json:"phone"`
	PhoneParent types.PhoneNumber `json:"phoneParent,omitempty"`
	Birthday    types.Date        `json:"birthday"`
	Returning   bool              `json:"returning"`
}

// nathejk:member.deleted
type NathejkMemberDeleted struct {
	MemberID types.MemberID `json:"memberId"`
	TeamID   types.TeamID   `json:"teamId"`
}
