package sqlevent

import (
	"io"
	"os"

	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
)

type client struct {
	c tablerow.Consumer
	p streaminterface.Publisher

	stderr io.Writer
}

func New(c tablerow.Consumer, p streaminterface.Publisher) *client {
	return &client{
		c:      c,
		p:      p,
		stderr: os.Stderr,
	}
}

func (c *client) Consume(query string) error {
	// parse query
	if c.c == nil {
		return nil
	}
	return c.c.Consume(query)
}

func (c *client) Close() error {
	return nil
}
