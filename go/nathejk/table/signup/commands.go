package signup

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
	"nathejk.dk/superfluids/streaminterface"
)

type Commands interface {
	Signup(context.Context, types.YearSlug, SignupCommand) (types.TeamID, error)
	SendVerificationEmail(context.Context, types.TeamID) error
	SendVerificationSms(context.Context, types.TeamID) error
	VerifyEmail(context.Context, types.TeamID, string) error
	VerifyPhone(context.Context, types.TeamID, string) error
}

type commander struct {
	p streaminterface.Publisher
	q Queries
	r repository
}

type SignupCommand struct {
	TeamType types.TeamType
	Name     string
	Email    types.EmailAddress
	Phone    types.PhoneNumber
}

func (c *commander) Signup(ctx context.Context, year types.YearSlug, cmd SignupCommand) (types.TeamID, error) {
	teamID := types.TeamID(uuid.New().String())
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.signedup", year, cmd.TeamType, teamID)))
	msg.SetBody(&messages.NathejkTeamSignedUp{
		TeamID: teamID,
		Name:   cmd.Name,
		Email:  cmd.Email,
		Phone:  cmd.Phone,
	})
	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return teamID, nil
}

func (c *commander) SendVerificationEmail(ctx context.Context, teamID types.TeamID) error {
	team, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	body := messages.NathejkMailSent{
		PingType:  types.PingTypeValidate,
		TeamID:    team.TeamID,
		Recipient: team.EmailPending,
		Subject:   "Bekræft e-mailadresse",
		Secret:    uuid.New().String(),
	}
	data := map[string]any{
		"id":     team.TeamID,
		"secret": body.Secret,
	}

	body.MessageID, err = c.r.mail.Send(string(team.EmailPending), "verify_email.tmpl", data)
	ok := "sent"
	if err != nil {
		ok = "failed"
		body.Error = err.Error()
	}
	sub := fmt.Sprintf("NATHEJK:%s.%s.%s.mail.%s.%s", team.Year, team.TeamType, team.TeamID, types.PingTypeValidate, ok)
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(sub))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	return c.p.Publish(msg)
}
func (c *commander) SendVerificationSms(ctx context.Context, teamID types.TeamID) error {
	team, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if (team.Phone != nil) && (*team.Phone == team.PhonePending) {
		// phone already verified
		return nil
	}
	pincode := fmt.Sprintf("%d", rand.IntN(9000)+1000)
	body := &messages.NathejkSmsSent{
		PingType: types.PingTypeValidate,
		TeamID:   teamID,
		Phone:    types.PhoneNumber(team.PhonePending.Normalize()),
		Text:     fmt.Sprintf("Din aktiveringskode til Nathejktilmeldingen er: %s", pincode),
		Secret:   pincode,
	}
	err = c.r.sms.Send(team.PhonePending.Normalize(), body.Text)
	ok := "sent"
	if err != nil {
		ok = "failed"
		body.Error = err.Error()
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.sms.%s.%s", team.Year, team.TeamType, teamID, types.PingTypeValidate, ok)))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})

	return c.p.Publish(msg)
}
func (c *commander) VerifyEmail(ctx context.Context, teamID types.TeamID, secret string) error {
	signup, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if len(secret) == 0 {
		return tables.ErrVerificationFailed
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.emailaddress.verified", signup.TeamType, types.TeamTypeKlan, teamID)))
	msg.SetBody(&messages.NathejkSignupEmailVerified{
		TeamID: teamID,
		Email:  signup.EmailPending,
		Secret: secret,
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}

func (c *commander) VerifyPhone(ctx context.Context, teamID types.TeamID, pincode string) error {
	signup, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if len(pincode) == 0 || pincode != signup.Pincode {
		return tables.ErrVerificationFailed
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.phonenumber.verified", signup.TeamType, types.TeamTypeKlan, teamID)))
	msg.SetBody(&messages.NathejkSignupPhoneVerified{
		TeamID:  teamID,
		Phone:   signup.PhonePending,
		Pincode: pincode,
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}

func (c *commander) Delete(ctx context.Context, teamID types.TeamID) error {
	klan, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", klan.Year, teamID)))
	msg.SetBody(&messages.NathejkKlanStatusChanged{
		TeamID: teamID,
		Status: types.SignupStatus("deleted"),
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}
