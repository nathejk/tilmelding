package entity

import (
	"nathejk.dk/pkg/streaminterface"
)

type publisher struct {
	name string
	p    streaminterface.Publisher
}

func NewEntityChangedPublisher(p streaminterface.Publisher, name string) *publisher {
	return &publisher{p: p, name: name}
}

func (p *publisher) Changed(body interface{}) error {
	m := p.MessageFunc()(streaminterface.SubjectFromStr(p.name + ".table:updated"))
	m.SetBody(body)
	return p.Publish(m)
}

func (p *publisher) Deleted(body interface{}) error {
	m := p.MessageFunc()(streaminterface.SubjectFromStr(p.name + ".table:deleted"))
	m.SetBody(body)
	return p.Publish(m)
}

func (p *publisher) Publish(msg streaminterface.Message) error {
	return p.p.Publish(msg)
}

func (p *publisher) MessageFunc() streaminterface.MessageFunc {
	return p.p.MessageFunc()
}
