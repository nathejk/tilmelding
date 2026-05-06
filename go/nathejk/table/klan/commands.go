package klan

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
	//Signup(context.Context, types.YearSlug, SignupCommand) (types.TeamID, error)
	//VerifyEmail(context.Context, types.TeamID, string) error
	//VerifyPhone(context.Context, types.TeamID, string) error
	RequestMemberCount(context.Context, types.YearSlug, types.TeamID, uint32) (uint32, error)
	Update(context.Context, types.TeamID, UpdateCommand) error
	AssignToLok(context.Context, types.TeamID, string) error
	Delete(context.Context, types.TeamID) error
}

type commander struct {
	p streaminterface.Publisher
	q Queries
	r repository
}

// RequestMemberCount attempts to reserve seats for the given team.
// It returns the number of seats successfully reserved. If capacity has been
// reached the request is placed on a waiting list and the return value is 0.
func (c *commander) RequestMemberCount(ctx context.Context, year types.YearSlug, teamID types.TeamID, memberCount uint32) (uint32, error) {
	actualMemberCount, err := c.q.RequestedMemberCount(ctx, year)
	if err != nil {
		return 0, err
	}
	action := "requested"
	if c.r.TotalMemberCount > actualMemberCount {
		action = "reserved"
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.%s", year, types.TeamTypeKlan, teamID, action)))
	msg.SetBody(&messages.NathejkTeamMembersRequested{
		TeamID:      teamID,
		MemberCount: int(memberCount),
	})
	if err := c.p.Publish(msg); err != nil {
		return 0, err
	}
	if action == "requested" {
		return 0, nil
	}
	return memberCount, nil
}

type SignupCommand struct {
	Name  string
	Email types.EmailAddress
	Phone types.PhoneNumber
}

func (c *commander) Signup(ctx context.Context, year types.YearSlug, cmd SignupCommand) (types.TeamID, error) {
	teamID := types.TeamID(uuid.New().String())
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.signedup", year, types.TeamTypeKlan, teamID)))
	msg.SetBody(&messages.NathejkTeamSignedUp{
		TeamID:  teamID,
		Name:    cmd.Name,
		Email:   cmd.Email,
		Phone:   cmd.Phone,
		Pincode: fmt.Sprintf("%d", rand.IntN(9000)+1000),
	})
	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return teamID, nil
}

func (c *commander) VerifyEmail(ctx context.Context, teamID types.TeamID, secret string) error {
	signup, err := c.r.Signup.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if len(secret) == 0 {
		return tables.ErrVerificationFailed
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.email_verified", signup.TeamType, types.TeamTypeKlan, teamID)))
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
	signup, err := c.r.Signup.GetByID(ctx, teamID)
	if err != nil {
		return err
	}
	if len(pincode) == 0 || pincode != signup.Pincode {
		return tables.ErrVerificationFailed
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.phone_verified", signup.TeamType, types.TeamTypeKlan, teamID)))
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

type UpdateCommand struct {
	Name      *string `json:"name"`
	GroupName *string `json:"groupName"`
	Korps     *string `json:"korps"`
}

func (c *commander) Update(ctx context.Context, teamID types.TeamID, cmd UpdateCommand) error {
	klan, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	// Merge: use existing values where the command does not provide an update.
	name := klan.Name
	if cmd.Name != nil {
		name = *cmd.Name
	}
	groupName := klan.Group
	if cmd.GroupName != nil {
		groupName = *cmd.GroupName
	}
	korps := klan.Korps
	if cmd.Korps != nil {
		korps = *cmd.Korps
	}

	// Dirty-check: only publish if something actually changed.
	if name == klan.Name && groupName == klan.Group && korps == klan.Korps {
		return nil
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.updated", klan.Year, teamID)))
	msg.SetBody(&messages.NathejkKlanUpdated{
		TeamID:    teamID,
		Name:      name,
		GroupName: groupName,
		Korps:     korps,
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}

func (c *commander) AssignToLok(ctx context.Context, teamID types.TeamID, lok string) error {
	klan, err := c.q.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	// Dirty-check: skip if already assigned to the same lok.
	if klan.Lok == lok {
		return nil
	}

	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.assigned", klan.Year, teamID)))
	msg.SetBody(&messages.NathejkKlanAssigned{
		TeamID: teamID,
		Lok:    lok,
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
