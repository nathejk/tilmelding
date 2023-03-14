package memorystream

import (
	"encoding/json"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/mailru/easyjson"

	"nathejk.dk/pkg/streaminterface"
)

var jjson = jsoniter.ConfigCompatibleWithStandardLibrary

// Message represents a message of a stream
type message struct {
	sequence int64
	subject  streaminterface.Subject
	body     json.RawMessage
	meta     json.RawMessage
	datetime time.Time
}

func NewMessage() *message {
	return new(message)
}

func (m *message) Subject() streaminterface.Subject {
	return m.subject
}

func (m *message) SetSubject(subj streaminterface.Subject) {
	m.subject = subj
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

func DecodeData(data []byte, v interface{}) error {
	if eu, ok := v.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(data, eu)
	}

	return jjson.Unmarshal(data, v)
}
