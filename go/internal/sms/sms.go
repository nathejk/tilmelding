package sms

import (
	"fmt"
	"net/url"
)

type Sender interface {
	Send(string, string) error
}

func NewClient(dsn string) (Sender, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "cpsms":
		return NewCpsms(u.Host, u.User.Username())
	}
	return nil, fmt.Errorf("unknown sms provider %q", u.Scheme)
}
