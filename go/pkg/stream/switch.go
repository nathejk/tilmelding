package stream

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streaminterface/caughtup"
)

// SwitchStatistics defines an interface for something that can return
// SwitchStats
type SwitchStatistics interface {
	Stats() SwitchStats
}

type Consumers []streaminterface.Consumer

func (handlers Consumers) Produces() (subjects []streaminterface.Subject) {
	for _, handler := range handlers {
		if p, ok := handler.(streaminterface.Producer); ok {
			subjects = append(subjects, p.Produces()...)
		}
	}
	return subjects
}

func (handlers Consumers) Subjects() []string {
	keys := make(map[string]struct{})
	for _, handler := range handlers {
		if p, ok := handler.(streaminterface.Producer); ok {
			for key := range subjectTypeSplit(p.Produces()) {
				keys[key] = struct{}{}
			}
		}
	}
	var slice []string
	for key := range keys {
		slice = append(slice, key)
	}
	return slice
}

func (handlers Consumers) Verify() error {
	// unique produces
	seen := make(map[string]interface{})
	for _, handler := range handlers {
		if p, ok := handler.(streaminterface.Producer); ok {
			for _, sub := range p.Produces() {
				if t, found := seen[sub.Subject()]; found {
					log.Printf("[warn] Consumers found multiple handlers that produce subj '%s': %T and %T", sub, t, handler)
				}
				seen[sub.Subject()] = handler
			}
		}
	}
	return nil
}

// Tracks various stats received and sent on this switch board, including counts for
// messages.
type SwitchStats struct {
	// sync access to counters with atomic.Load/Store
	InMsgs  uint64
	OutMsgs uint64

	// sync access to timers with sync.Lock/Unlock
	mu                sync.Mutex
	Start             time.Time
	SubscribeDuration time.Duration
	CaughtupDuration  time.Duration
}

func (s SwitchStats) Format() string {
	return fmt.Sprintf(`
Messages
--------

InMsgs: %d,
OutMsgs: %d,

Timers
------

Started: %v
SubscribeDuration: %v
CaughtupDuration: %v
`, s.InMsgs, s.OutMsgs, s.Start.Format(time.RFC3339), s.SubscribeDuration, s.CaughtupDuration)
}

func (s *SwitchStats) callLocked(f func(*SwitchStats)) {
	s.mu.Lock()
	f(s)
	s.mu.Unlock()
}

// SwitchOption is a function on the options on a Switch.
type SwitchOption func(*SwitchOptions)

// SwitchOptions are used to control the Switch's behaviour
type SwitchOptions struct {
	caughtupFunc   func()
	subscribedFunc func()
	waitOnCaughtup bool
}

func SwitchCaughtupFunc(f func()) SwitchOption {
	return func(o *SwitchOptions) {
		o.caughtupFunc = f
	}
}

func SwitchSubscribedFunc(f func()) SwitchOption {
	return func(o *SwitchOptions) {
		o.subscribedFunc = f
	}
}

func SwitchWaitOnCaughtupDisabled() SwitchOption {
	return func(o *SwitchOptions) {
		o.waitOnCaughtup = false
	}
}

// Switch is a publish—subscriber orchestration tool.
//
// It sorts all our Consumers by the subjects they subscribe to.
//
// Subjects are sorted by creating a directed graph, the direction of edges is
// given by from -> to, where 'from' is the subjects a handler subscribes to,
// and 'to' are the subjects a Handler-Producer publishes on. Handlers are
// represented as edges, while subjects as nodes.
//
// Creates a single fanout handler for each root node (subject) we subscribe
// to, to remove the need for keeping a log of messages in streams.
//
// Catchup; creates a catchup handler for each subject handler, that does
// bookkeeping for caughtup state of channel.
type Switch struct {
	mux         *StreamMux
	handlers    []streaminterface.Consumer
	subexplodes []subexplodes
	//TODO: probably only need this
	mapexplodes map[string]subexplodes
	topo        Topology

	// waitgroup to signal we are caughtup
	caughtup *sync.WaitGroup

	// options
	opts SwitchOptions

	// sub lock
	mu   sync.Mutex
	subs []streaminterface.Subscription

	// Switch stats
	stats SwitchStats
}

// NewSwitch initializes a new publish-subscriber orchestration tool.
func NewSwitch(mux *StreamMux, handlers []streaminterface.Consumer, opts ...SwitchOption) (*Switch, error) {
	m := &Switch{
		mux:      mux,
		handlers: handlers,
		caughtup: &sync.WaitGroup{},
		subs:     make([]streaminterface.Subscription, 0),
	}

	// set options
	m.opts.waitOnCaughtup = true // defaults to true

	for _, opt := range opts {
		opt(&m.opts)
	}

	// initialize
	if err := m.init(); err != nil {
		return nil, err
	}

	return m, nil
}

// subexplodes is a sortable structure of exploded
type subexplodes struct {
	sub      string
	explodes []*explodedHandler
}

func (s subexplodes) Handlers() (handlers []streaminterface.MessageHandler) {
	for _, e := range s.explodes {
		handlers = append(handlers, e.h)
	}
	return handlers
}

func (m *Switch) init() error {
	if err := Consumers(m.handlers).Verify(); err != nil {
		return err
	}
	// Create a handler map by subject
	subjexpl := make(map[string][]*explodedHandler)
	for _, sh := range m.handlers {
		e := explodeHandler(m, sh)
		for subject, e := range e.handlermap() {
			subjexpl[subject] = append(subjexpl[subject], e)
		}
	}

	m.mapexplodes = make(map[string]subexplodes, len(subjexpl))
	m.subexplodes = make([]subexplodes, 0, len(subjexpl))
	for sub, expl := range subjexpl {
		m.mapexplodes[sub] = subexplodes{sub, expl}
		m.subexplodes = append(m.subexplodes, subexplodes{sub, expl})
	}

	var err error
	m.topo, err = NewTopology(m.handlers)
	return err
}

// Run creates subscriptions by subscribing all Handlers to their subjects of
// interest. Run blocks until context is cancelled.
func (m *Switch) Run(ctx context.Context, noop ...func()) (err error) {
	// call close on exit
	defer func() {
		e := m.Close()
		if e != nil {
			err = e
		}
	}()
	m.stats.callLocked(func(s *SwitchStats) { s.Start = time.Now().UTC() })

	// lock while subcribing
	m.mu.Lock()

	// subscribe handlers to the subjects they are interested in. Handlers are
	// topological sorted such that root nodes are subscribed to last.
	for _, se := range sortedHandlers(m.subexplodes, m.topo.SortedSubjects()) {
		// find the stream for the given subject
		stream := m.mux.Lookup(se.sub)

		// Subscribe each handler to their respective stream, if it's a root
		// node, create a fanout handler, and subscribe to it once. Do this to
		// ensure that all subscriptions on root subjects, get all messages on
		// that subject, without requiring further synchronization or buffering
		// of messages.
		if m.topo.RootSubject(se.sub) {
			// Note we add AnnounceCaughtup option to the root subscriptions.
			// This tells to the stream that they should send the "caughtup"
			// message once caugthup on this subject. If we don't do this—and
			// the root subjects don't announce "caughtup" AND the
			// SwitchWaitOnCaughtup option is enabled (it's enabled by
			// default), then we'll block forever.
			//
			// When does the above case occur, and isn't it just a programming
			// error. No, not always. Lets say you have a subject handler that
			// subscribes to subject A, but this subject in the given execution
			// mode of the program, don't produce any events, then you have a
			// deadlock. While the program logic is correct, and in different
			// execution modes of the program, subject A would receive events.
			sub, err := stream.Subscribe(se.sub, newFanoutHandler(se.sub, se.Handlers()))
			if err != nil {
				m.mu.Unlock()
				return err
			}

			m.subs = append(m.subs, sub)
		} else {
			for _, e := range se.explodes {
				//log.Printf("subscribe %T to subj '%s'\n", e.orig, se.sub)
				sub, err := stream.Subscribe(se.sub, e.h)
				if err != nil {
					m.mu.Unlock()
					return err
				}

				m.subs = append(m.subs, sub)
			}
		}
	}
	m.mu.Unlock()

	// Subscribed, set stats.
	m.stats.callLocked(func(s *SwitchStats) { s.SubscribeDuration = time.Now().UTC().Sub(s.Start) })

	if m.opts.subscribedFunc != nil {
		m.opts.subscribedFunc()
	}

	if m.opts.waitOnCaughtup {
		// Wait on all subscriptions to catch up. This may block forever if
		// there a programming error where you subscribe to a subject that does
		// not exist AND the publisher doesn't annunce caughtup.
		log.Println("Switch: wait on catch up")
		m.caughtup.Wait()
		log.Println("Switch: caught up — OK")
	} else {
		// call all catchup listeners for the handlers that implement them.
		// This design follows the way pno/model works, in that if you
		// implement CatchupListener, it will be called, no matter if there is
		// a CatchupListenSubscriber.
		for _, h := range m.handlers {
			if cl, ok := h.(streaminterface.CatchupListener); ok {
				cl.CaughtUp()
			}
		}
	}

	// caught up, set stats before calling caughtup callbacks
	m.stats.callLocked(func(s *SwitchStats) { s.CaughtupDuration = time.Now().UTC().Sub(s.Start) })

	if m.opts.caughtupFunc != nil {
		m.opts.caughtupFunc()
	}

	// todo: deprecated
	for _, f := range noop {
		f()
	}

	<-ctx.Done()
	return nil
}

func sortedHandlers(handlers []subexplodes, sortedSubjects []string) []subexplodes {
	lookup := make(map[string]subexplodes, len(handlers))
	for _, expls := range handlers {
		lookup[expls.sub] = expls
	}

	sorted := make([]subexplodes, 0, len(handlers))
	for _, sub := range sortedSubjects {
		if e, exist := lookup[sub]; exist {
			sorted = append(sorted, e)
		} //else {
		//log.Printf("subj '%s' does not exist\n", sub)
		//}
	}

	return sorted
}

// Close calls Close on all subscriptions
func (m *Switch) Close() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sub := range m.subs {
		e := sub.Close()
		if e != nil {
			err = e
		}
	}

	return err
}

/*
// Topology returns the topology for Consumer's in this switch
func (m *Switch) Topology() *Topology {
	return m.topo
}

/*
// DotGraph returns a dot graph of the topology for the Consumer's in this switch
func (m *Switch) DotGraph() (string, error) {
	b, err := dot.Marshal(m.topo.Graph(), "aggregates", "", "\t")
	return string(b), err
}
*/

func (m *Switch) Stats() SwitchStats {
	// guard for timer variables
	m.stats.mu.Lock()
	defer m.stats.mu.Unlock()

	return SwitchStats{
		InMsgs:            atomic.LoadUint64(&m.stats.InMsgs),
		OutMsgs:           atomic.LoadUint64(&m.stats.OutMsgs),
		Start:             m.stats.Start,
		SubscribeDuration: m.stats.SubscribeDuration,
		CaughtupDuration:  m.stats.CaughtupDuration,
	}
}

// explodedHandler is a handlers interface types exploded to a struct
type explodedHandler struct {
	swtch     *Switch
	h         streaminterface.MessageHandler
	orig      streaminterface.MessageHandler
	subj      []string
	subjtypes map[string][]string
	prod      []string
	prodtypes map[string][]string
}

func (e *explodedHandler) HandleMessage(m streaminterface.Message) { e.h.HandleMessage(m) }

func explodeHandler(swtch *Switch, sh streaminterface.Consumer) *explodedHandler {
	e := explodedHandler{
		swtch:     swtch,
		h:         sh,
		orig:      sh,
		subjtypes: subjectTypeSplit(sh.Consumes()),
		prodtypes: make(map[string][]string),
	}

	for subject := range e.subjtypes {
		e.subj = append(e.subj, subject)
	}

	if p, ok := sh.(streaminterface.Producer); ok {
		e.prodtypes = subjectTypeSplit(p.Produces())

		for subject := range e.prodtypes {
			e.prod = append(e.prod, subject)
		}
	}

	return &e
}

// handlers creates a map of subject and exploded handler types from a
// Consumer.
func (e *explodedHandler) handlermap() map[string]*explodedHandler {
	// create handler middleware
	cl, _ := e.orig.(streaminterface.CatchupListener)
	var catchupHandler HandlerMiddleware
	if e.swtch.opts.waitOnCaughtup {
		catchupHandler = newCatchupHandler(cl, e.swtch.mux, e.swtch.caughtup, e.subj, e.prod)
	} else {
		catchupHandler = noopHandler
	}
	syncHandler := NewSyncHandler(e.orig)
	m := make(map[string]*explodedHandler)
	for subject, types := range e.subjtypes {
		// copy of e for each subject, with its own LimitHandler that filters
		// unwanted types.
		e := *e
		e.h = catchupHandler(syncHandler(LimitHandler(e.h, types)))
		m[subject] = &e
	}

	return m
}

// subjectTypeSplit accepts a slice of subjects which may contain a type,
// identified by the values in the subject after a colon (":"), returns a map
// of subjects, and the types for that subject.
func subjectTypeSplit(values []streaminterface.Subject) map[string][]string {
	m := make(map[string][]string)

	for _, val := range values {
		m[val.Domain()] = append(m[val.Domain()], val.Type())
	}

	return m
}

type HandlerMiddleware func(streaminterface.MessageHandler) streaminterface.MessageHandler

func noopHandler(h streaminterface.MessageHandler) streaminterface.MessageHandler {
	return h
}

func newCatchupHandler(cl streaminterface.CatchupListener, mux *StreamMux, caughtupwg *sync.WaitGroup, subjects, produces []string) func(streaminterface.MessageHandler) streaminterface.MessageHandler {
	var done int32
	var mu sync.Mutex

	N := len(subjects)
	caughtupsubs := make(map[string]bool, N)
	log.Printf("Wait caughtup on %d subjects %v for %T\n", N, subjects, cl)
	caughtupwg.Add(N)

	return func(h streaminterface.MessageHandler) streaminterface.MessageHandler {
		return streaminterface.MessageHandlerFunc(func(m streaminterface.Message) {
			if atomic.LoadInt32(&done) == 1 {
				h.HandleMessage(m)
				return
			}

			if !caughtup.IsCaughtup(m) {
				// Synchronize calls to Handler if NOT caughtup and we have a
				// CaughtUp func.
				if cl != nil {
					mu.Lock()
					defer mu.Unlock()
				}
				//log.Printf("DEBUG [not caught up yet] handling %q", m.Subject().Subject())
				h.HandleMessage(m)
				return
			}

			// Synchronize cl.CaughtUp and caughtupsubs map access.
			mu.Lock()
			defer mu.Unlock()
			// check if all subjects are caught up
			caughtupsubs[m.Subject().Domain()] = true

			if len(caughtupsubs) != N {
				return
			}

			if cl != nil {
				cl.CaughtUp()
			}

			// we are all caughtup at this point
			atomic.StoreInt32(&done, 1)

			// find the streams we produce on
			for _, prod := range produces {
				msg := caughtup.NewCaughtupMessage(prod)
				err := mux.Lookup(prod).Publish(msg)
				if err != nil {
					panic(err)
				}
			}

			log.Printf("Caughtup to %v for %T\n", subjects, cl)

			for i := 0; i < N; i++ {
				caughtupwg.Done()
			}
		})
	}
}

// NewSyncHandler returns a Handler that synchronizes calls to h.Handle, if l
// implements sync.Locker.
func NewSyncHandler(locker interface{}) func(streaminterface.MessageHandler) streaminterface.MessageHandler {
	return func(h streaminterface.MessageHandler) streaminterface.MessageHandler {
		mu, ok := locker.(sync.Locker)
		if !ok {
			return h
		}

		return streaminterface.MessageHandlerFunc(func(m streaminterface.Message) {
			mu.Lock()
			h.HandleMessage(m)
			mu.Unlock()
		})
	}
}

// LimitHandler returns a handler that filters messages unless they have a type
// as defined in the types slice. If types slice is empty we accept all messages.
func LimitHandler(h streaminterface.MessageHandler, types []string) streaminterface.MessageHandler {
	// if types is empty, we accept all messages
	if len(types) == 0 {
		return h
	}

	// if there is a blank type, we accept all messages
	for _, t := range types {
		if t == "" {
			return h
		}
	}

	// return a Handler that only accepts messages with a type that matches.
	return streaminterface.MessageHandlerFunc(func(m streaminterface.Message) {
		for _, t := range types {
			if m.Subject().Type() == t {
				h.HandleMessage(m)
				return
			}
		}

		//log.Printf("filtered message %s-%s\n", m.Channel, m.Type)
	})
}

// StreamMux maps subjects to Streams
type StreamMux struct {
	mu       sync.Mutex
	lookup   map[string]streaminterface.Stream
	fallback streaminterface.Stream
}

// TODO: implement stream
func NewStreamMux(fallback streaminterface.Stream) *StreamMux {
	return &StreamMux{
		fallback: fallback,
	}
}

func (mux *StreamMux) Handles(s streaminterface.Stream, domains ...string) {
	for _, domain := range domains {
		mux.Handle(s, domain)
	}
}

func (mux *StreamMux) Handle(s streaminterface.Stream, subject string) {
	if streaminterface.SubjectFromStr(subject).String() != subject {
		panic("bad subject: " + subject)
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()

	// lazy init
	if mux.lookup == nil {
		mux.lookup = map[string]streaminterface.Stream{}
	}

	mux.lookup[subject] = s
}

func (mux *StreamMux) Lookup(subject string) streaminterface.Stream {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	// lazy init
	if mux.lookup == nil {
		mux.lookup = map[string]streaminterface.Stream{}
	}

	if s, ok := mux.lookup[subject]; ok {
		return s
	}

	return mux.fallback
}

func (mux *StreamMux) Publish(m streaminterface.Message) error {
	panic("not implement")
}

func (mux *StreamMux) Subscribe(subject string, cb streaminterface.MessageHandler) (streaminterface.Subscription, error) {
	panic("not implement")
}

func (mux *StreamMux) Close() error {
	panic("not implement")
}
