package main

import (
	"sync"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// catchupDetector is a stream consumer that watches the global stream sequence
// and fires onLive exactly once — the moment the projections have replayed
// every event that already existed when the process started (targetSeq). From
// that point on, incoming events are "live" rather than historical replay.
//
// It subscribes to the whole NATHEJK stream (the superset of every projection's
// subjects) via its own ordered consumer, so it is guaranteed to observe the
// message at targetSeq. targetSeq is the stream's last sequence at boot; when
// it is zero (empty stream) FireNow lets the caller emit the signal directly.
type catchupDetector struct {
	targetSeq uint64
	onLive    func()
	once      sync.Once
}

func newCatchupDetector(targetSeq uint64, onLive func()) *catchupDetector {
	return &catchupDetector{targetSeq: targetSeq, onLive: onLive}
}

func (d *catchupDetector) Consumes() []stream.Subject {
	return []stream.Subject{subject.FromStr("NATHEJK:>")}
}

func (d *catchupDetector) HandleMessage(msg stream.Message) error {
	if msg.Sequence() >= d.targetSeq {
		d.once.Do(d.onLive)
	}
	return nil
}

// FireNow emits the live signal immediately, guarded by the same once as
// HandleMessage. Used for an empty stream, where no message will ever arrive to
// cross targetSeq.
func (d *catchupDetector) FireNow() { d.once.Do(d.onLive) }
