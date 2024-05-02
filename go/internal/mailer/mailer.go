package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer interface {
	Send(recipient, templateFile string, data any) error
}
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

type mailer struct {
	dialer     *mail.Dialer
	sender     string
	retryCount int
	retrySleep time.Duration
}

func NewFromConfig(c Config) *mailer {
	return New(c.Host, c.Port, c.Username, c.Password, c.Sender)
}

func New(host string, port int, username, password, sender string) *mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return &mailer{
		dialer:     dialer,
		sender:     sender,
		retryCount: 3,
		retrySleep: 500 * time.Millisecond,
	}
}

func (m *mailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}
	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}
	plainBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return err
	}
	htmlBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Try sending the email several times before aborting and returning the final error.
	for i := 1; i <= m.retryCount; i++ {
		err = m.dialer.DialAndSend(msg)
		// If everything worked, return nil.
		if nil == err {
			return nil
		}
		// If it didn't work, sleep for a short time and retry.
		time.Sleep(m.retrySleep)
	}

	return err
}
