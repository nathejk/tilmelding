package messages

import "nathejk.dk/pkg/types"

// nathejk:signedup
type NathejkTeamSignedUp struct {
	TeamID types.TeamID `json:"teamId"`
	//	Type   types.Enum   `json:"type"`
	//	Slug    string            `json:"slug"`
	Name    string            `json:"name"`
	Phone   types.PhoneNumber `json:"phone"`
	Email   types.Email       `json:"email"`
	Pincode string            `json:"pincode"`
}

// nathejk:phonenumber.confirmed
type NathejkSignupPincodeUsed struct {
	TeamID  types.TeamID      `json:"teamId"`
	Phone   types.PhoneNumber `json:"phone"`
	Pincode string            `json:"pincode"`
}

// nathejk:sms.sent
type NathejkSmsSent struct {
	PingType types.PingType    `json:"pingType"`
	TeamID   types.TeamID      `json:"teamId,omitempty"`
	Phone    types.PhoneNumber `json:"phone"`
	Text     string            `json:"text"`
	Error    string            `json:"error,omitempty"`
}

// nathejk:team.updated
type NathejkTeamUpdated struct {
	TeamID            types.TeamID      `json:"teamId"`
	Type              types.Enum        `json:"type"`
	Name              string            `json:"name"`
	GroupName         string            `json:"groupName"`
	Korps             string            `json:"korps"`
	AdvspejdNumber    string            `json:"advspejdNumber,omitempty"`
	ContactName       string            `json:"contactName"`
	ContactAddress    string            `json:"contactAddress,omitempty"`
	ContactPostalCode string            `json:"contactPostalCode,omitempty"`
	ContactEmail      types.Email       `json:"contactEmail"`
	ContactPhone      types.PhoneNumber `json:"contactPhone"`
	ContactRole       string            `json:"contactRole"`
}

// nathejk:klan.updated
type NathejkKlanUpdated struct {
	TeamID    types.TeamID `json:"teamId"`
	Name      string       `json:"name"`
	GroupName string       `json:"groupName"`
	Korps     string       `json:"korps"`
}

type NathejkTeamStatusChanged struct {
	TeamID types.TeamID       `json:"teamId"`
	Status types.SignupStatus `json:"signupStatus"`
}
type NathejkPatruljeStatusChanged NathejkTeamStatusChanged
type NathejkKlanStatusChanged NathejkTeamStatusChanged

// nathejk:team.merged
type NathejkTeamMerged struct {
	TeamID       types.TeamID `json:"teamId"`
	ParentTeamID types.TeamID `json:"parentTeamId"`
	SosID        types.SosID  `json:"sosId,omitempty"`
}

// nathejk:team.splitedd
type NathejkTeamSplited struct {
	TeamID types.TeamID `json:"teamId"`
	SosID  types.SosID  `json:"sosId,omitempty"`
}

// nathejk:team.updated
type NathejkTeamDeleted struct {
	TeamID types.TeamID `json:"teamId"`
}
