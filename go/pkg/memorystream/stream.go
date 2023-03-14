package memorystream

import (
	"errors"
	"fmt"
	"sync"

	"nathejk.dk/pkg/streaminterface"
)

var (
	ErrBadSubject      = errors.New("stream: invalid subject")
	ErrBadSubscription = errors.New("stream: invalid subscription")
)

// StreamStatistics defines an interface for something that can return
// StreamStats
type StreamStatistics interface {
	Stats() StreamStats
}

// Tracks various stats received and sent on this stream, including counts for
// messages.
type StreamStats struct {
	InMsgs  uint64
	OutMsgs uint64
}

func (s StreamStats) Format() string {
	return fmt.Sprintf(`
InMsgs: %d,
OutMsgs: %d,
`, s.InMsgs, s.OutMsgs)
}

// StreamOption is a function on the options for a subscription.
type StreamOption func(*StreamOptions)

// StreamOptions are used to control the Subscription's behavior.
type StreamOptions struct {
	Log bool
}

// StreamOptionWithLog enables persistent log
func StreamOptionWithLog() StreamOption {
	return func(o *StreamOptions) {
		o.Log = true
	}
}

type channel struct {
	seqId int64
	subs  map[int64]*subscription

	// an optional in-memory log of all messages on a channel
	logMu sync.Mutex
	log   []streaminterface.Message
}

// stream implements an in-memory publish—subscribe messaging service
type stream struct {
	//StreamStats

	// options
	opts StreamOptions

	// subscription id
	ssid   int64
	subsMu sync.RWMutex
	subs   map[string]*channel
}

func New(options ...StreamOption) streaminterface.Stream {
	s := &stream{
		subs: make(map[string]*channel),
	}
	for _, opt := range options {
		opt(&s.opts)
	}
	return s
}

/*
func (c *stream) Stats() StreamStats {
	return StreamStats{
		InMsgs:  atomic.LoadUint64(&c.InMsgs),
		OutMsgs: atomic.LoadUint64(&c.OutMsgs),
	}
}*/

func (c *stream) Publish(msg streaminterface.Message) error {
	subject := msg.Subject().Subject()
	if subject == "" {
		return ErrBadSubject
	}

	// Stats
	//atomic.AddUint64(&c.OutMsgs, 1)

	c.subsMu.RLock()

	q := c.subs[subject]
	if q == nil {
		q = c.subs[msg.Subject().Domain()]
		if q == nil {
			c.subsMu.RUnlock()
			return nil
		}
	}

	//	msg.Sequence = atomic.AddInt64(&q.seqId, 1)
	c.subsMu.RUnlock()
	//	msg.Channel = subject

	c.process(msg)
	return nil
}

func (s *stream) MessageFunc() streaminterface.MessageFunc {
	return func(subj streaminterface.Subject) streaminterface.MutableMessage {
		m := NewMessage()
		m.SetSubject(subj)
		return m
	}
}

// Subscribe will perform a subscription with the stream for the given subject.
func (c *stream) Subscribe(subject string, cb streaminterface.MessageHandler) (streaminterface.Subscription, error) {
	if streaminterface.SubjectFromStr(subject).String() != subject {
		return nil, ErrBadSubject
	}
	if cb == nil {
		return nil, ErrBadSubscription
	}

	sub := &subscription{Subject: subject, cb: cb, stream: c}

	// setup a rendezvous point for announcing the occurrence of new messages.
	sub.pCond = sync.NewCond(&sub.mu)

	c.subsMu.Lock()
	c.ssid++
	sub.sid = c.ssid
	q := c.subs[subject]
	if q == nil {
		q = &channel{
			subs: make(map[int64]*subscription, 1),
		}
		c.subs[subject] = q
	}
	if c.opts.Log {
		q.logMu.Lock()
		sub.log = make([]streaminterface.Message, len(q.log))
		sub.pMsgs += copy(sub.log, q.log)
		q.logMu.Unlock()
	}
	c.subs[subject].subs[sub.sid] = sub
	c.subsMu.Unlock()

	// start up a sub specific Go routine to deliver messages.
	go c.waitDeliver(sub)

	return sub, nil
}

func (c *stream) unsubsribe(sub *subscription) {
	c.subsMu.Lock()
	q := c.subs[sub.Subject]

	// Already unsubscribed (no queue).
	if q == nil {
		c.subsMu.Unlock()
		return
	}

	s := q.subs[sub.sid]

	// Already unsubscribed (no subscribers).
	if s == nil {
		c.subsMu.Unlock()
		return
	}

	// Delete sub from stream.
	delete(q.subs, s.sid)
	if len(q.subs) == 0 {
		delete(c.subs, sub.Subject)
	}
	c.subsMu.Unlock()

	// Close the subscription
	s.mu.Lock()
	s.closed = true
	if s.pCond != nil {
		s.pCond.Broadcast()
	}
	s.mu.Unlock()
}

// waitDeliver waits on the conditional shared with process. It is used to
// deliver messages to asynchronous subscribers.
func (c *stream) waitDeliver(s *subscription) {
	var closed bool

	s.mu.Lock()
	loglen := len(s.log)
	logpos := 0
	s.mu.Unlock()

	for {
		s.mu.Lock()

		// Do accounting for last msg delivered here so we only lock once
		// and drain state trips after callback has returned.
		s.pMsgs--
		var m streaminterface.Message

		if logpos < loglen {
			m = s.log[logpos]
			logpos++
		} else {
			if s.pHead == nil && !s.closed {
				s.pCond.Wait()
			}

			// Pop the msg off the list
			n := s.pHead
			if n != nil {
				s.pHead = n.next

				if s.pHead == nil {
					s.pTail = nil
				}
				m = n.m
			}
		}
		//if m != nil {
		//	// this is handled else where
		//	//s.live = m.Sequence > s.opts.LiveAfterSequenceId
		//	//m.Live = s.live // TODO: we are mutating shared message
		//}
		cb := s.cb
		closed = s.closed
		if !closed {
			s.delivered++
		}
		s.mu.Unlock()

		if closed {
			break
		}

		// Deliver the message
		if m != nil {
			cb.HandleMessage(m)
		}
	}
}

// process is called by Publish and will place the Message on each subscribers
// channel.
func (c *stream) process(msg streaminterface.Message) {
	c.subsMu.RLock()

	// Stats
	//atomic.AddUint64(&c.InMsgs, 1)

	q := c.subs[msg.Subject().Domain()]
	if q == nil {
		c.subsMu.RUnlock()
		return
	}

	if c.opts.Log {
		q.logMu.Lock()
		q.log = append(q.log, msg)
		q.logMu.Unlock()
	}

	for _, sub := range q.subs {
		n := &node{m: msg}
		sub.mu.Lock()
		sub.pMsgs++

		// Push onto the async pList for a given subscription
		if sub.pHead == nil {
			sub.pHead = n
			sub.pTail = n
			sub.pCond.Signal()
		} else {
			sub.pTail.next = n
			sub.pTail = n
		}
		sub.mu.Unlock()
	}

	c.subsMu.RUnlock()
}

func (c *stream) Close() error {
	return nil
}

// node used for linked list of pending messages on a given subscription
type node struct {
	m    streaminterface.Message
	next *node
}

// A subscription represents interest in a given subject.
type subscription struct {
	mu sync.Mutex

	// Subject is the name of the channel we are subscribed to.
	Subject string

	stream    *stream
	sid       int64  // subscription id
	delivered uint64 // number of messages delivered to sub
	closed    bool
	live      bool

	// log of historic messages we should Handle before processing the pending
	// message list
	log []streaminterface.Message

	// linked list of pending messages
	pHead *node
	pTail *node
	pCond *sync.Cond

	cb streaminterface.MessageHandler

	// stats
	pMsgs int
}

func (s *subscription) Close() error {
	if s == nil {
		return ErrBadSubscription
	}
	s.mu.Lock()
	stream := s.stream
	closed := s.closed
	s.mu.Unlock()
	if stream == nil {
		return nil
	}
	if closed {
		return ErrBadSubscription
	}
	s.stream.unsubsribe(s)
	return nil
}

func (s *subscription) Live() bool {
	//s.mu.Lock()
	//defer s.mu.Lock()
	return s.live
}

var (
	_ streaminterface.Stream = &stream{}
	//_ StreamStatistics             = &stream{}
	_ streaminterface.Subscription = &subscription{}
)
