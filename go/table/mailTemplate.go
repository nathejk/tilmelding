package table

import (
	"fmt"
	"log"

	"nathejk.dk/pkg/messages"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/tablerow"
	"nathejk.dk/pkg/types"

	_ "embed"
)

type MailTemplate struct {
	Slug     types.Slug `sql:"slug"`
	Subject  string     `sql:"subject"`
	Template string     `sql:"template"`
}

type mailTemplate struct {
	w tablerow.Consumer
}

func NewMailTemplate(w tablerow.Consumer) *mailTemplate {
	table := &mailTemplate{w: w}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Fatalf("Error creating table %q", err)
	}
	return table
}

//go:embed mailTemplate.sql
var mailTemplateSchema string

func (t *mailTemplate) CreateTableSql() string {
	return mailTemplateSchema
}

func (t *mailTemplate) Consumes() (subjs []streaminterface.Subject) {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk"),
	}
}

func (t *mailTemplate) HandleMessage(msg streaminterface.Message) {
	switch msg.Subject().Subject() {
	case "nathejk:mail.template.updated":
		var body messages.NathejkMailTemplateUpdated
		if err := msg.Body(&body); err != nil {
			return
		}
		err := t.w.Consume(fmt.Sprintf("INSERT INTO mailTemplate SET slug=%q, subject=%q, template=%q ON DUPLICATE KEY UPDATE subject=VALUES(subject), template=VALUES(template)", body.Slug, body.Subject, body.Template))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	case "nathejk:mail.template.deleted":
		var body messages.NathejkMailTemplateDeleted
		if err := msg.Body(&body); err != nil {
			return
		}
		err := t.w.Consume(fmt.Sprintf("DELETE FROM mailTemplate WHERE slug=%q", body.Slug))
		if err != nil {
			log.Fatalf("Error consuming sql %q", err)
		}
	}
}
