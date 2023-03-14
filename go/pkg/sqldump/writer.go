package sqldump

import (
	"io"
	"os"
)

type dump struct {
	w io.Writer
}

func New(w io.Writer) *dump {
	return &dump{w: w}
}

func NewStdoutWriter() *dump {
	return New(os.Stdout)
}

func (d *dump) Consume(s string) error {
	if d.w != nil {
		_, err := d.w.Write([]byte(s + "\n"))
		return err
	}
	return nil
}
