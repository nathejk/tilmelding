package sqlevent

import (
	"io"
	"os"

	"github.com/seaqrs/tablerow"
	"nathejk.dk/pkg/streaminterface"
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
