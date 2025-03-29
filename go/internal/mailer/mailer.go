package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	//"github.com/go-mail/mail/v2"
	"github.com/wneessen/go-mail"
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
	client *mail.Client
	//dialer     *mail.Dialer
	sender     string
	retryCount int
	retrySleep time.Duration
}

func NewFromConfig(c Config) *mailer {
	return New(c.Host, c.Port, c.Username, c.Password, c.Sender)
}

func New(host string, port int, username, password, sender string) *mailer {
	client, err := mail.NewClient(host,
		//mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(5*time.Second),
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
	)
	if err != nil {
		return nil
	}
	// Return a Mailer instance containing the client and sender information.
	//mailer := &Mailer{client: client, sender: sender}

	//	dialer := mail.NewDialer(host, port, username, password)
	//	dialer.Timeout = 5 * time.Second

	return &mailer{
		client:     client,
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
	msg := mail.NewMsg()
	err = msg.To(recipient)
	if err != nil {
		return err
	}
	err = msg.From(m.sender)
	if err != nil {
		return err
	}
	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())
	// Call the DialAndSend() method on the dialer, passing in the message to send. This // opens a connection to the SMTP server, sends the message, then closes the
	// connection.
	return m.client.DialAndSend(msg)
	/*
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
	*/
}
