package types

import (
	"strings"
)

type ID = string
type Slug = string
type Enum = string

type TeamID ID
type MemberID ID
type ScanID ID
type AttachmentID ID
type SosID ID

type UserID ID

func (id UserID) IsSlackUser() bool {
	return strings.HasPrefix(string(id), "slack-")
}

type Email string

type PingType string

const (
	PingTypeSignup        PingType = "signup"
	PingTypeMobilepayLink PingType = "mobilepay"
)
