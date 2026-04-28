package signup

import (
	"nathejk.dk/internal/mailer"
	"nathejk.dk/internal/sms"
)

type repository struct {
	sms  sms.Sender
	mail mailer.Mailer
}

type service func(*repository)

func WithSms(s sms.Sender) service {
	return func(r *repository) {
		r.sms = s
	}
}
func WithMailer(s mailer.Mailer) service {
	return func(r *repository) {
		r.mail = s
	}
}

func NewRepository(services ...service) repository {
	r := repository{}
	for _, with := range services {
		with(&r)
	}
	return r
}
