package nats

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"

	"nathejk.dk/pkg/streaminterface"
)

var jjson = jsoniter.ConfigCompatibleWithStandardLibrary

// struct represents a message of a nats stream
type envelope struct {
	EventID       string          `json:"eventId"`
	CorrelationID string          `json:"correlationId"`
	CausationID   string          `json:"causationId"`
	Version       int             `json:"version,omitempty"`
	Datetime      time.Time       `json:"datetime"`
	Type          string          `json:"type"`
	Body          json.RawMessage `json:"body"`
	Meta          json.RawMessage `json:"meta"`
}

func (e *envelope) IsValid() bool {
	return len(e.Type) > 0 && len(e.Body) > 0
}

// Message represents a message of a stream
type message struct {
	sequence      uint64          `json:"sequence"`
	eventID       string          `json:"eventId"`
	correlationID string          `json:"correlationId"`
	causationID   string          `json:"causationId"`
	version       int             `json:"version,omitempty"`
	datetime      time.Time       `json:"datetime"`
	channel       string          `json:"-"`
	msgtype       string          `json:"type"`
	body          json.RawMessage `json:"body"`
	meta          json.RawMessage `json:"meta"`
}

func NewMessage() *message {
	eventID := "event-" + uuid.New().String()
	return &message{
		eventID:       eventID,
		correlationID: eventID,
		causationID:   eventID,
		datetime:      time.Now().UTC(),
	}
}

func (m *message) Sequence() uint64 {
	return m.sequence
}

// Own message ID
func (m *message) EventID() string {
	return m.eventID
}
func (m *message) SetEventID(ID string) {
	m.eventID = ID
}

// Parent message ID
func (m *message) CausationID() string {
	return m.causationID
}
func (m *message) SetCausationID(ID string) {
	m.causationID = ID
}

// Ancestor message ID
func (m *message) CorrelationID() string {
	return m.correlationID
}
func (m *message) SetCorrelationID(ID string) {
	m.correlationID = ID
}

func (m *message) SetCausationCorrelationFromMessage(msg Identifiable) {
	m.SetCausationID(msg.EventID())
	m.SetCorrelationID(msg.CorrelationID())
}

func (m *message) Subject() streaminterface.Subject {
	return streaminterface.SubjectFromStr(m.channel + ":" + m.msgtype)
}

func (m *message) SetSubject(subj streaminterface.Subject) {
	m.channel = subj.Domain()
	m.msgtype = subj.Type()
}

func (m *message) Time() time.Time {
	return m.datetime
}

func (m *message) SetTime(ts time.Time) error {
	m.datetime = ts
	return nil
}

func (m *message) Body(v interface{}) error {
	if eu, ok := v.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(m.body, eu)
	}

	return jjson.Unmarshal(m.body, v)
}

func (m *message) RawBody() interface{} {
	return m.body
}

func (m *message) SetBody(v interface{}) (err error) {
	if eu, ok := v.(easyjson.Marshaler); ok {
		buf, err := easyjson.Marshal(eu)
		if err != nil {
			return err
		}
		m.body = buf
		return nil
	}

	buf, err := jjson.Marshal(v)
	if err != nil {
		return err
	}
	m.body = buf
	return nil
}

func (m *message) Meta(v interface{}) error {
	if eu, ok := v.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(m.meta, eu)
	}

	return jjson.Unmarshal(m.meta, v)
}

func (m *message) RawMeta() interface{} {
	return m.meta
}

func (m *message) SetMeta(v interface{}) (err error) {
	if eu, ok := v.(easyjson.Marshaler); ok {
		buf, err := easyjson.Marshal(eu)
		if err != nil {
			return err
		}
		m.meta = buf
	}

	buf, err := jjson.Marshal(v)
	if err != nil {
		return err
	}
	m.meta = buf
	return nil
}

func (m *message) DecodeData(data []byte) error {
	var e envelope
	if err := jjson.Unmarshal(data, &e); err != nil {
		return err
	}
	if !e.IsValid() {
		return errors.New("Error decoding message envelope")
	}
	m.eventID = e.EventID
	m.correlationID = e.CorrelationID
	m.causationID = e.CausationID
	m.version = e.Version
	m.datetime = e.Datetime
	m.msgtype = e.Type
	m.body = e.Body
	m.meta = e.Meta

	return nil
}
