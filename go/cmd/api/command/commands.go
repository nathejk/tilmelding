package command

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	handlebars "github.com/aymerick/raymond"
	"github.com/google/uuid"

	"nathejk.dk/pkg/messages"
	"nathejk.dk/pkg/notification"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/types"
	"nathejk.dk/table"
)

type queries interface {
	Patrulje(types.TeamID) (*table.Patrulje, error)
	Spejdere(types.TeamID) ([]table.Spejder, error)
	Spejder(types.MemberID) (*table.Spejder, error)
	Klan(types.TeamID) (*table.Klan, error)
	Seniorer(types.TeamID) ([]table.Senior, error)
	Senior(types.MemberID) (*table.Senior, error)
	TeamType(types.TeamID) types.TeamType
	IsOpen(types.TeamType) bool
	MaxSeatCount(types.TeamType) int
	UsedSeatCount(types.TeamType) int
	SignupStart(types.TeamType) *time.Time
	SignupStarted(types.TeamType) bool
	Pincode(types.TeamID) string
}

type commands struct {
	q   queries
	p   streaminterface.Publisher
	sms notification.SmsSender

	lastTeamSlug string
}

func New(q queries, p streaminterface.Publisher, sms notification.SmsSender) *commands {
	c := &commands{
		q:   q,
		p:   p,
		sms: sms,
	}
	return c
}

func (c *commands) NextTeamSlug() string {
	i, err := strconv.Atoi(c.lastTeamSlug)
	if err != nil {
		i = 0
	}
	min := time.Now().Year() * 1000
	if i < min {
		i = min
	}
	c.lastTeamSlug = fmt.Sprintf("%d", i+1)

	return c.lastTeamSlug
}

func (c *commands) CreatePatrulje() {
}

func (c *commands) UpdatePatrulje(teamID types.TeamID, name string, grp string, korps string, contactName string, contactPhone types.PhoneNumber, contactEmail types.Email, contactRole string) error {
	body := messages.NathejkTeamUpdated{
		TeamID:       teamID,
		Type:         "patrulje",
		Name:         name,
		GroupName:    grp,
		Korps:        korps,
		ContactName:  contactName,
		ContactPhone: contactPhone,
		ContactEmail: contactEmail,
		ContactRole:  contactRole,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:patrulje.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}

func (c *commands) UpdateSpejder(memberID types.MemberID, teamID types.TeamID, name string, address string, postalCode string, city string, email types.Email, phone types.PhoneNumber, phoneParent types.PhoneNumber, birthday types.Date, returning bool) error {
	spejder, err := c.q.Spejder(memberID)
	if err == sql.ErrNoRows {
		spejder = &table.Spejder{
			MemberID: types.MemberID(uuid.New().String()),
			TeamID:   teamID,
		}
	} else if err != nil {
		return err
	}
	if spejder.MemberID == memberID && spejder.Name == name && spejder.Address == address && spejder.PostalCode == postalCode && spejder.City == city && spejder.Email == email && spejder.Phone == phone && spejder.PhoneParent == phoneParent && spejder.Birthday == birthday && spejder.Returning == returning {
		return nil
	}
	body := messages.NathejkMemberUpdated{
		MemberID:    spejder.MemberID,
		TeamID:      spejder.TeamID,
		Name:        name,
		Address:     address,
		PostalCode:  postalCode,
		City:        city,
		Email:       email,
		Phone:       phone,
		PhoneParent: phoneParent,
		Birthday:    birthday,
		Returning:   returning,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:spejder.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}
func (c *commands) DeleteSpejder(memberID types.MemberID) error {
	spejder, err := c.q.Spejder(memberID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	body := messages.NathejkMemberDeleted{
		MemberID: spejder.MemberID,
		TeamID:   spejder.TeamID,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:spejder.deleted"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}

func (c *commands) UpdateKlan(teamID types.TeamID, name string, grp string, korps string) error {
	body := messages.NathejkKlanUpdated{
		TeamID:    teamID,
		Name:      name,
		GroupName: grp,
		Korps:     korps,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:klan.updated"))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	return c.p.Publish(msg)
}
func (c *commands) UpdateSenior(memberID types.MemberID, teamID types.TeamID, name string, address string, postalCode string, city string, email types.Email, phone types.PhoneNumber, birthday types.Date, returning bool) error {
	m, err := c.q.Senior(memberID)
	if err == sql.ErrNoRows {
		m = &table.Senior{
			MemberID: types.MemberID(uuid.New().String()),
			TeamID:   teamID,
		}
	} else if err != nil {
		return err
	}
	if m.MemberID == memberID && m.Name == name && m.Address == address && m.PostalCode == postalCode && m.City == city && m.Email == email && m.Phone == phone && m.Birthday == birthday && m.Returning == returning {
		return nil
	}
	body := messages.NathejkMemberUpdated{
		MemberID:   m.MemberID,
		TeamID:     m.TeamID,
		Name:       name,
		Address:    address,
		PostalCode: postalCode,
		City:       city,
		Email:      email,
		Phone:      phone,
		Birthday:   birthday,
		Returning:  returning,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:senior.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: "tilmelding-api"}
	msg.SetMeta(&meta)

	return c.p.Publish(msg)
}
func (c *commands) DeleteSenior(memberID types.MemberID) error {
	senior, err := c.q.Senior(memberID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	body := messages.NathejkMemberDeleted{
		MemberID: senior.MemberID,
		TeamID:   senior.TeamID,
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:senior.deleted"))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	return c.p.Publish(msg)
}
func (c *commands) OpenSignup(teamType types.TeamType, maxSeatCount int) error {
	if c.q.IsOpen(teamType) && maxSeatCount == c.q.MaxSeatCount(teamType) {
		return nil
	}
	meta := messages.Metadata{Producer: "tilmelding-api"}
	switch teamType {
	case types.TeamTypePatrulje:
		msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:patrulje.signup.opened"))
		msg.SetBody(&messages.NathejkPatruljeSignupOpened{})
		msg.SetMeta(&meta)
		return c.p.Publish(msg)
	case types.TeamTypeKlan:
		msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:klan.signup.opened"))
		msg.SetBody(&messages.NathejkPatruljeSignupOpened{MaxSeatCount: maxSeatCount})
		msg.SetMeta(&meta)
		return c.p.Publish(msg)
	}
	return nil
}
func (c *commands) CloseSignup(teamType types.TeamType) error {
	if !c.q.IsOpen(teamType) {
		return nil
	}
	meta := messages.Metadata{Producer: "tilmelding-api"}
	switch teamType {
	case types.TeamTypePatrulje:
		msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:patrulje.signup.closed"))
		msg.SetBody(&messages.NathejkPatruljeSignupOpened{})
		msg.SetMeta(&meta)
		return c.p.Publish(msg)
	case types.TeamTypeKlan:
		msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:klan.signup.closed"))
		msg.SetBody(&messages.NathejkPatruljeSignupOpened{})
		msg.SetMeta(&meta)
		return c.p.Publish(msg)
	}
	return nil
}
func (c *commands) SignupStart(t *time.Time) error {
	start := c.q.SignupStart(types.TeamTypeKlan)
	if start == nil && t == nil {
		// both are unspecified
		return nil
	}
	if start != nil && t != nil && *start == *t {
		// both are specified but the same
		return nil
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:klan.signup.start.specified"))
	msg.SetBody(&messages.NathejkKlanSignupStartSpecified{Time: t})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	return c.p.Publish(msg)
}

func (c *commands) Signup(teamType types.TeamType, name string, phone types.PhoneNumber, email types.Email) (types.TeamID, error) {
	if !c.q.SignupStarted(teamType) {
		return "", fmt.Errorf("Signup wil not start until %q", c.q.SignupStart(types.TeamTypeKlan))
	}
	if !c.q.IsOpen(teamType) {
		return "", fmt.Errorf("Signup closed")
	}
	body := messages.NathejkTeamSignedUp{
		TeamID:  types.TeamID(uuid.New().String()),
		Name:    name,
		Phone:   phone,
		Email:   email,
		Pincode: fmt.Sprintf("%d", rand.Intn(9999)+10000)[1:],
	}
	var subject streaminterface.Subject
	switch teamType {
	case types.TeamTypePatrulje:
		subject = streaminterface.SubjectFromStr("nathejk:patrulje.signedup")
	case types.TeamTypeKlan:
		subject = streaminterface.SubjectFromStr("nathejk:klan.signedup")
	default:
		return "", fmt.Errorf("Unknown team type %q", teamType)
	}
	msg := c.p.MessageFunc()(subject)
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	if err := c.p.Publish(msg); err != nil {
		return "", err
	}

	go c.SendSms(body.TeamID, phone, types.PingTypeSignup, "Din pincode er:"+body.Pincode)

	return body.TeamID, nil
}

func (c *commands) reservePatruljeSeats(signupStatus types.SignupStatus, requestedMemberCount int) (types.SignupStatus, int) {
	switch signupStatus {
	case types.SignupStatusOnHold:
		return types.SignupStatusOnHold, 0
	case types.SignupStatusPaid:
		return types.SignupStatusPaid, requestedMemberCount
	}
	maxSeatCount := c.q.MaxSeatCount(types.TeamTypePatrulje)
	if maxSeatCount > 0 && c.q.UsedSeatCount(types.TeamTypePatrulje) >= maxSeatCount {
		return types.SignupStatusOnHold, 0
	}
	return types.SignupStatusPay, requestedMemberCount
}
func (c *commands) reserveKlanSeats(signupStatus types.SignupStatus, requestedMemberCount int) (types.SignupStatus, int) {
	switch signupStatus {
	case types.SignupStatusOnHold:
		return types.SignupStatusOnHold, 0
	case types.SignupStatusPay:
		return types.SignupStatusPay, requestedMemberCount
	case types.SignupStatusPaid:
		return types.SignupStatusPaid, requestedMemberCount
	}
	//log.Printf("Reserving KLAN seats: max:%d, used:")
	maxSeatCount := c.q.MaxSeatCount(types.TeamTypeKlan)
	if maxSeatCount > 0 && c.q.UsedSeatCount(types.TeamTypeKlan) >= maxSeatCount {
		return types.SignupStatusOnHold, 0
	}
	return types.SignupStatusPay, requestedMemberCount
}

func (c *commands) RequestSeats(teamID types.TeamID) (types.SignupStatus, int, error) {
	patrulje, _ := c.q.Patrulje(teamID)
	if patrulje != nil {
		members, _ := c.q.Spejdere(teamID)
		requestedMemberCount := len(members)
		signupStatus, reservedMemberCount := c.reservePatruljeSeats(patrulje.SignupStatus, requestedMemberCount)
		if patrulje.SignupStatus != signupStatus {
			msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:patrulje.status.changed"))
			msg.SetBody(&messages.NathejkPatruljeStatusChanged{TeamID: teamID, Status: signupStatus})
			msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
			if err := c.p.Publish(msg); err != nil {
				return "", 0, err
			}
		}
		return signupStatus, reservedMemberCount, nil
	}

	klan, _ := c.q.Klan(teamID)
	if klan != nil {
		members, _ := c.q.Seniorer(teamID)
		requestedMemberCount := len(members)
		signupStatus, reservedMemberCount := c.reserveKlanSeats(klan.SignupStatus, requestedMemberCount)
		if klan.SignupStatus != signupStatus {
			msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:klan.status.changed"))
			msg.SetBody(&messages.NathejkKlanStatusChanged{TeamID: teamID, Status: signupStatus})
			msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
			if err := c.p.Publish(msg); err != nil {
				return "", 0, err
			}
		}
		return signupStatus, reservedMemberCount, nil
	}

	return "", 0, fmt.Errorf("Unkwown team %q", teamID)
}

func (c *commands) RequestMobilepayLink(teamID types.TeamID, phone types.PhoneNumber, memberCount int) error {
	// 330811 Nathejk Senior tilmelding.
	// 204414 Nathejk Spejder tilmelding
	var account, memberPrice int
	switch c.q.TeamType(teamID) {
	case types.TeamTypePatrulje:
		account = 204414
		memberPrice = 200
	case types.TeamTypeKlan:
		account = 330811
		memberPrice = 250
	}
	text := fmt.Sprintf("https://www.mobilepay.dk/erhverv/betalingslink/betalingslink-svar?phone=%d&amount=%d&comment=%s&lock=1", account, memberPrice*memberCount, teamID)

	return c.SendSms(teamID, phone, types.PingTypeMobilepayLink, text)
}

func (c *commands) SendSms(teamID types.TeamID, phone types.PhoneNumber, pingType types.PingType, text string) error {
	err := c.sms.Send(phone.Normalize(), text)

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sms.sent"))
	msg.SetBody(messages.NathejkSmsSent{
		PingType: pingType,
		Phone:    types.PhoneNumber(phone.Normalize()),
		TeamID:   teamID,
		Text:     text,
		Error:    fmt.Sprintf("%s", err),
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	return c.p.Publish(msg)
}

func (c *commands) UsePincode(teamID types.TeamID, phone types.PhoneNumber, pincode string) bool {
	if pincode != c.q.Pincode(teamID) {
		return false
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:signup.pincode.used"))
	msg.SetBody(messages.NathejkSignupPincodeUsed{
		TeamID:  teamID,
		Phone:   phone,
		Pincode: pincode,
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	c.p.Publish(msg)
	return true
}

func (c *commands) MailTemplate(slug types.Slug, subject string, template string) error {
	if _, err := handlebars.Parse(template); err != nil {
		return err
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr("nathejk:mail.template.updated"))
	msg.SetBody(messages.NathejkMailTemplateUpdated{
		Slug:     slug,
		Subject:  subject,
		Template: template,
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	return c.p.Publish(msg)
}
