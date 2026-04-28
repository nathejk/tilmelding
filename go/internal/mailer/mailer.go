package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/wneessen/go-mail"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer interface {
	Send(recipient, templateFile string, data any) (string, error)
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

type Option func(*mailer)

type mailer struct {
	host       string
	port       int
	username   string
	password   string
	sender     string
	retryCount int
	retrySleep time.Duration
	globalVars map[string]any
}

func WithHost(host string) Option {
	return func(m *mailer) {
		m.host = host
	}
}

func WithPort(port int) Option {
	return func(m *mailer) {
		m.port = port
	}
}

func WithUsername(username string) Option {
	return func(m *mailer) {
		m.username = username
	}
}

func WithPassword(password string) Option {
	return func(m *mailer) {
		m.password = password
	}
}

func WithSender(sender string) Option {
	return func(m *mailer) {
		m.sender = sender
	}
}

func WithRetryCount(count int) Option {
	return func(m *mailer) {
		m.retryCount = count
	}
}

func WithRetrySleep(d time.Duration) Option {
	return func(m *mailer) {
		m.retrySleep = d
	}
}

func WithGlobalVar(key string, value any) Option {
	return func(m *mailer) {
		m.globalVars[key] = value
	}
}

func NewFromConfig(c Config) *mailer {
	return New(
		WithHost(c.Host),
		WithPort(c.Port),
		WithUsername(c.Username),
		WithPassword(c.Password),
		WithSender(c.Sender),
	)
}

func New(opts ...Option) *mailer {
	m := &mailer{
		port:       587,
		retryCount: 3,
		retrySleep: 500 * time.Millisecond,
		globalVars: make(map[string]any),
	}
	return m.AddOptions(opts...)
}

func (m *mailer) AddOptions(opts ...Option) *mailer {
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *mailer) mergeData(data any) map[string]any {
	merged := make(map[string]any, len(m.globalVars))
	for k, v := range m.globalVars {
		merged[k] = v
	}

	if data == nil {
		return merged
	}

	switch d := data.(type) {
	case map[string]any:
		for k, v := range d {
			merged[k] = v
		}
	default:
		merged["Data"] = d
	}

	return merged
}

func (m *mailer) newClient() (*mail.Client, error) {
	return mail.NewClient(m.host,
		mail.WithPort(m.port),
		mail.WithUsername(m.username),
		mail.WithPassword(m.password),
		mail.WithTimeout(5*time.Second),
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
	)
}

func (m *mailer) Send(recipient, templateFile string, data any) (string, error) {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return "", err
	}

	mergedData := m.mergeData(data)

	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", mergedData); err != nil {
		return "", err
	}
	plainBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(plainBody, "plainBody", mergedData); err != nil {
		return "", err
	}
	htmlBody := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(htmlBody, "htmlBody", mergedData); err != nil {
		return "", err
	}

	msg := mail.NewMsg()
	if err := msg.To(recipient); err != nil {
		return "", err
	}
	if err := msg.From(m.sender); err != nil {
		return "", err
	}
	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	var lastErr error
	for i := 0; i < m.retryCount; i++ {
		client, err := m.newClient()
		if err != nil {
			lastErr = err
			time.Sleep(m.retrySleep)
			continue
		}
		if err := client.DialAndSend(msg); err != nil {
			lastErr = err
			time.Sleep(m.retrySleep)
			continue
		}
		return msg.GetMessageID(), nil
	}
	return "", fmt.Errorf("failed to send email after %d attempts: %w", m.retryCount, lastErr)
}
