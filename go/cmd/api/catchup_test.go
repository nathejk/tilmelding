package main

import (
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// fakeMessage is a minimal stream.Message whose sequence we control. The
// streamtest helper can't set the sequence, which is exactly what the detector
// keys off, so we roll our own.
type fakeMessage struct {
	seq uint64
}

func (m fakeMessage) Subject() stream.Subject { return subject.FromStr("NATHEJK:test.event") }
func (m fakeMessage) Time() time.Time         { return time.Time{} }
func (m fakeMessage) Sequence() uint64        { return m.seq }
func (m fakeMessage) Body(interface{}) error  { return nil }
func (m fakeMessage) Meta(interface{}) error  { return nil }
func (m fakeMessage) RawBody() interface{}    { return nil }
func (m fakeMessage) RawMeta() interface{}    { return nil }

func TestCatchupDetector_FiresOnceWhenTargetReached(t *testing.T) {
	var fired int
	d := newCatchupDetector(3, func() { fired++ })

	// Below target: no fire.
	_ = d.HandleMessage(fakeMessage{seq: 1})
	_ = d.HandleMessage(fakeMessage{seq: 2})
	if fired != 0 {
		t.Fatalf("expected no fire before target, got %d", fired)
	}

	// At target: fire exactly once.
	_ = d.HandleMessage(fakeMessage{seq: 3})
	if fired != 1 {
		t.Fatalf("expected 1 fire at target, got %d", fired)
	}

	// Past target: still only one fire.
	_ = d.HandleMessage(fakeMessage{seq: 4})
	_ = d.HandleMessage(fakeMessage{seq: 5})
	if fired != 1 {
		t.Fatalf("expected exactly 1 fire, got %d", fired)
	}
}

func TestCatchupDetector_FiresWhenTargetSkipped(t *testing.T) {
	// A consumer may never see the exact target sequence (that event's subject
	// might not match), so any sequence at or beyond the target must fire.
	var fired int
	d := newCatchupDetector(3, func() { fired++ })

	_ = d.HandleMessage(fakeMessage{seq: 5})
	if fired != 1 {
		t.Fatalf("expected fire when target is skipped, got %d", fired)
	}
}

func TestCatchupDetector_FireNowDedupesWithHandleMessage(t *testing.T) {
	// Empty-stream path: FireNow emits the signal, and a later message must not
	// fire it a second time.
	var fired int
	d := newCatchupDetector(0, func() { fired++ })

	d.FireNow()
	if fired != 1 {
		t.Fatalf("expected FireNow to fire once, got %d", fired)
	}

	_ = d.HandleMessage(fakeMessage{seq: 1})
	if fired != 1 {
		t.Fatalf("expected no additional fire after FireNow, got %d", fired)
	}
}
