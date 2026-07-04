package klan

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type Commands interface {
	//Signup(context.Context, types.YearSlug, SignupCommand) (types.TeamID, error)
	//VerifyEmail(context.Context, types.TeamID, string) error
	//VerifyPhone(context.Context, types.TeamID, string) error
	RequestMemberCount(context.Context, types.YearSlug, types.TeamID, uint32) (uint32, error)
	Update(context.Context, types.TeamID, UpdateCommand) error
	UpdateMembers(context.Context, types.TeamID, Team, []Senior) error
	AssignToLok(context.Context, types.TeamID, string) error
	Delete(context.Context, types.TeamID) error
}

type commander struct {
	p stream.Publisher
	q Queries
	r repository
}

// RequestMemberCount attempts to reserve seats for the given team.
// It returns the number of seats successfully reserved. If capacity has been
// reached the request is placed on a waiting list and the return value is 0.
//
// The cap is sourced from the product catalogue (participation.klan.stock)
// when WithProductQueries was wired in; otherwise the legacy
// WithTotalMemberCount fallback applies. See repository.go.
func (c *commander) RequestMemberCount(ctx context.Context, year types.YearSlug, teamID types.TeamID, memberCount uint32) (uint32, error) {
	actualMemberCount, err := c.q.RequestedMemberCount(ctx, year)
	if err != nil {
		return 0, err
	}
	cap := c.capacity(ctx, year)
	action := "requested"
	if cap > actualMemberCount {
		action = "reserved"
	}
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.%s", year, types.TeamTypeKlan, teamID, action)))
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

// capacity returns the active seat cap for klan participation in the
// given year. Sources, in priority order:
//
//   - Product catalogue: participation.klan.stock for `year`. NULL stock
//     (unlimited) is treated as no constraint and falls through.
//   - Legacy WithTotalMemberCount option (r.TotalMemberCount).
//
// On any product-query error the function silently falls back to the
// legacy value; capacity gating is non-critical and we'd rather degrade
// to the conservative legacy cap than fail the request.
func (c *commander) capacity(ctx context.Context, year types.YearSlug) uint32 {
	if c.r.Products != nil {
		if p, err := c.r.Products.GetBySKU(ctx, year, "participation.klan"); err == nil && p != nil && p.Stock != nil {
			if *p.Stock < 0 {
				return 0
			}
			return uint32(*p.Stock)
		}
	}
	return c.r.TotalMemberCount
}

type SignupCommand struct {
	Name  string
	Email types.EmailAddress
	Phone types.PhoneNumber
}

func (c *commander) Signup(ctx context.Context, year types.YearSlug, cmd SignupCommand) (types.TeamID, error) {
	teamID := types.TeamID(uuid.New().String())
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.signedup", year, types.TeamTypeKlan, teamID)))
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

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.email_verified", signup.TeamType, types.TeamTypeKlan, teamID)))
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

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.%s.%s.phone_verified", signup.TeamType, types.TeamTypeKlan, teamID)))
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

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.updated", klan.Year, teamID)))
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

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.assigned", klan.Year, teamID)))
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

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", klan.Year, teamID)))
	msg.SetBody(&messages.NathejkKlanStatusChanged{
		TeamID: teamID,
		Status: types.SignupStatus("deleted"),
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}

// Team is the team-level slice of an UpdateMembers command.
type Team struct {
	TeamID      types.TeamID `json:"teamId"`
	Name        string       `json:"name"`
	Group       string       `json:"group"`
	Korps       string       `json:"korps"`
	MemberCount int          `json:"memberCount"`
}

// Senior is one member entry on an UpdateMembers command. Setting
// Deleted=true publishes a member-deleted event instead of an update.
type Senior struct {
	MemberID   types.MemberID     `json:"memberId"`
	Deleted    bool               `json:"deleted"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	PostalCode string             `json:"postalCode"`
	Email      types.EmailAddress `json:"email"`
	Phone      types.PhoneNumber  `json:"phone"`
	Birthday   types.Date         `json:"birthday"`
	Diet       string             `json:"diet"`
	TShirtSize string             `json:"tshirtSize"`
}

// UpdateMembers projects the team form into the klan-side write events:
// one NathejkKlanUpdated for the team slice, optional status transitions
// when the global senior cap is reached, and one NathejkSeniorUpdated
// (or NathejkMemberDeleted) per member.
//
// Members slice may be empty; in that case, MemberCount placeholder rows
// are emitted so the projection can grow to the requested size before any
// senior identities are filled in.
func (c *commander) UpdateMembers(ctx context.Context, teamID types.TeamID, team Team, members []Senior) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.updated", "2026", teamID)))
	msg.SetBody(&messages.NathejkKlanUpdated{
		TeamID:    teamID,
		Name:      team.Name,
		GroupName: team.Group,
		Korps:     team.Korps,
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	klan, _ := c.q.GetByID(ctx, teamID)
	if klan != nil && klan.Status == types.SignupStatusOnHold {
		// The team is on waiting list, do not transition status.
		return nil
	}

	seniorCount, _ := c.q.RequestedSeniorCount(ctx, "2026")
	if seniorCount > 115 {
		statusMsg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", "2026", teamID)))
		statusMsg.SetBody(&messages.NathejkKlanStatusChanged{TeamID: teamID, Status: types.SignupStatusOnHold})
		if klan != nil && (klan.Status != types.SignupStatusPay) && (klan.Status != types.SignupStatusPaid) {
			if err := c.p.Publish(statusMsg); err != nil {
				return err
			}
		}
	}
	if klan != nil && klan.Status == "" {
		statusMsg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", "2026", teamID)))
		statusMsg.SetBody(&messages.NathejkKlanStatusChanged{TeamID: teamID, Status: types.SignupStatusPay})
		if err := c.p.Publish(statusMsg); err != nil {
			return err
		}
	}

	if len(members) == 0 {
		for i := 0; i < team.MemberCount; i++ {
			members = append(members, Senior{})
		}
	}

	for i := range members {
		m := &members[i]
		if m.Deleted {
			msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.senior.%s.deleted", "2026", m.MemberID)))
			msg.SetBody(&messages.NathejkMemberDeleted{
				MemberID: m.MemberID,
				TeamID:   teamID,
			})
			if err := c.p.Publish(msg); err != nil {
				return err
			}
			continue
		}

		// Assign a fresh ID to brand-new members. Mutating through the
		// slice index (not the loop copy) so the caller's slice carries
		// the assigned IDs back — derivedLinesForKlan needs them on the
		// same slice to key the order lines by memberId.
		if m.MemberID == "" {
			m.MemberID = types.MemberID(uuid.New().String())
		}
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.senior.%s.updated", "2026", m.MemberID)))
		// Include teamId in the body so the senior projector's two-phase
		// decode (see senior/consumer.go) can do an INSERT IGNORE for
		// brand-new members. Without it the row is never created and the
		// subsequent UPDATE matches zero rows, leaving order lines that
		// reference a senior the projection never knew about.
		msg.SetBody(&struct {
			messages.NathejkSeniorUpdated
			TeamID types.TeamID `json:"teamId"`
		}{
			NathejkSeniorUpdated: messages.NathejkSeniorUpdated{
				MemberID:   m.MemberID,
				Name:       m.Name,
				Address:    m.Address,
				PostalCode: m.PostalCode,
				Email:      m.Email,
				Phone:      m.Phone,
				BirthDate:  m.Birthday,
				TShirtSize: m.TShirtSize,
				Diet:       m.Diet,
			},
			TeamID: teamID,
		})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}

	return nil
}
