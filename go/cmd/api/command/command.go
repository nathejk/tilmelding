package command

import (
	"fmt"
	"strconv"
	"time"

	"nathejk.dk/pkg/streaminterface"
)

func NewCommand(publisher streaminterface.Publisher) *command {
	c := &command{}
	return c
}

type command struct {
	publisher    streaminterface.Publisher
	lastTeamSlug string
}

func (c *command) NextTeamSlug() string {
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

func (c *command) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk:signedup"),
	}
}

func (c *command) HandleMessage(msg streaminterface.Message) {
	/*
		switch msg.Subject().Subject() {
		case "nathejk:signedup":
			var body messages.NathejkTeamSignedUp
			msg.Body(&body)
			if body.Slug > c.lastTeamSlug {
				c.lastTeamSlug = body.Slug
			}

		default:
			log.Printf("Unhandled message %q", msg.Subject().Subject())
		}
	*/
}
