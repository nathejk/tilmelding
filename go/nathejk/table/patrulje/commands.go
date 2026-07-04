package patrulje

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

// Commands is the patrulje write-side API. Methods publish domain events
// onto the stream; the patrulje projection (consumer.go) is the read-side.
type Commands interface {
	Update(ctx context.Context, teamID types.TeamID, team Team, contact Contact, members []Spejder) error
	AssignNumber(ctx context.Context, teamID types.TeamID) error
}

// Team is the team-level slice of an UpdatePatrulje command.
type Team struct {
	TeamID          types.TeamID `json:"teamId"`
	Name            string       `json:"name"`
	Group           string       `json:"group"`
	Korps           string       `json:"korps"`
	AdventureLigaID string       `json:"liga"`
}

// Contact is the contact-person slice of an UpdatePatrulje command.
type Contact struct {
	TeamID     types.TeamID       `json:"teamId"`
	Name       string             `json:"name"`
	Address    string             `json:"address"`
	PostalCode string             `json:"postal"`
	Email      types.EmailAddress `json:"email"`
	Phone      types.PhoneNumber  `json:"phone"`
	Role       string             `json:"role"`
}

// Spejder is one member entry on an UpdatePatrulje command. Setting
// Deleted=true publishes a member-deleted event instead of an update.
type Spejder struct {
	MemberID     types.MemberID     `json:"memberId"`
	Deleted      bool               `json:"deleted"`
	Name         string             `json:"name"`
	Address      string             `json:"address"`
	PostalCode   string             `json:"postalCode"`
	Email        types.EmailAddress `json:"email"`
	Phone        types.PhoneNumber  `json:"phone"`
	PhoneContact types.PhoneNumber  `json:"phoneContact"`
	Birthday     types.Date         `json:"birthday"`
	TShirtSize   string             `json:"tshirtSize"`
}

type commander struct {
	p stream.Publisher
	q *querier
}

// Update publishes a NathejkTeamUpdated for the team / contact slice and a
// NathejkScoutUpdated (or NathejkMemberDeleted) per member. New members
// without a MemberID are assigned a fresh UUID before the update event.
func (c *commander) Update(ctx context.Context, teamID types.TeamID, team Team, contact Contact, members []Spejder) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.patrulje.%s.updated", "2026", teamID)))
	msg.SetBody(&messages.NathejkTeamUpdated{
		TeamID:            teamID,
		Type:              types.TeamTypePatrulje,
		Name:              team.Name,
		GroupName:         team.Group,
		Korps:             team.Korps,
		AdvspejdNumber:    team.AdventureLigaID,
		ContactName:       contact.Name,
		ContactAddress:    contact.Address,
		ContactPostalCode: contact.PostalCode,
		ContactEmail:      contact.Email,
		ContactPhone:      contact.Phone,
		ContactRole:       contact.Role,
	})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	for i := range members {
		m := &members[i]
		if m.Deleted {
			msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.deleted", "2026", m.MemberID)))
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
		// the assigned IDs back — derivedLinesForPatrulje needs them on
		// the same slice to key the order lines by memberId.
		if m.MemberID == "" {
			m.MemberID = types.MemberID(uuid.New().String())
		}
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.updated", "2026", m.MemberID)))
		// Include teamId in the body so the spejder projector's two-phase
		// decode (see spejder/consumer.go) can do an INSERT IGNORE for
		// brand-new members. Without it the row is never created and the
		// subsequent UPDATE matches zero rows, leaving order lines that
		// reference a spejder the projection never knew about.
		msg.SetBody(&struct {
			messages.NathejkScoutUpdated
			TeamID types.TeamID `json:"teamId"`
		}{
			NathejkScoutUpdated: messages.NathejkScoutUpdated{
				MemberID:     m.MemberID,
				Name:         m.Name,
				Address:      m.Address,
				PostalCode:   m.PostalCode,
				Email:        m.Email,
				Phone:        m.Phone,
				PhoneContact: m.PhoneContact,
				BirthDate:    m.Birthday,
				TShirtSize:   m.TShirtSize,
			},
			TeamID: teamID,
		})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}

	return nil
}

// AssignNumber finds the highest team number in use and publishes a
// NathejkPatrolNumberAssigned event for `teamID` set to that-number+1
// (starting at 1 if none have been assigned yet).
func (c *commander) AssignNumber(ctx context.Context, teamID types.TeamID) error {
	last, err := c.q.GetLastWithNumber(ctx)
	nr := 1
	if err == nil && last != nil && last.TeamNumber != "" {
		i, perr := strconv.Atoi(last.TeamNumber)
		if perr != nil {
			return fmt.Errorf("unable to find next number %#v", perr)
		}
		nr = i + 1
	} else if err != nil && !errors.Is(err, tables.ErrRecordNotFound) {
		return err
	}
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.patrulje.%s.numberassigned", "2026", teamID)))
	msg.SetBody(&messages.NathejkPatrolNumberAssigned{
		TeamID:     teamID,
		TeamNumber: fmt.Sprintf("%d", nr),
	})
	return c.p.Publish(msg)
}
