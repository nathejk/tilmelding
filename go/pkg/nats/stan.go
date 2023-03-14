package nats

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/nats-io/stan.go"
	"github.com/pkg/errors"

	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streaminterface/caughtup"
	//"nathejk.dk/pkg/stream/messagevalidator"
)

// StreamOption is a function on the options for a subscription.
type NatsStreamOption func(*NatsStreamOptions)

// StreamOptions are used to control the Subscription's behavior.
type NatsStreamOptions struct {
	Log bool
	//	Validator             messagevalidator.MessageValidator
	ConnectionLostHandler func(stan.Conn, error)
	MonitorDSN            string

	// ImmediateCatchup sends the catchup event before any other event—effectivly skipping it.
	ImmediateCatchup bool
}

/*
// StreamOptionWithValidator creates a nats stream that validates messages
// before sending them on the stream.
func StreamOptionWithValidator(v messagevalidator.MessageValidator) NatsStreamOption {
	return func(o *NatsStreamOptions) {
		o.Validator = v
	}
}
*/

// StreamOptionConnectionLostHandler sets your specified connection lost handler, instead of the default one that panics on
// connection lost.
func StreamOptionConnectionLostHandler(f func(stan.Conn, error)) NatsStreamOption {
	return func(o *NatsStreamOptions) {
		o.ConnectionLostHandler = f
	}
}

// MonitorDSN allows you to set the montior DSN, rather than using the default.
func StreamOptionMontiorDSN(dsn string) NatsStreamOption {
	return func(o *NatsStreamOptions) {
		o.MonitorDSN = dsn
	}
}

// StreamOptionWithLog enables persistent log
func StreamOptionWithLog() NatsStreamOption {
	return func(o *NatsStreamOptions) {
		o.Log = true
	}
}

// ImmediateCatchup sends the catchup event before any other event—effectivly skipping it.
func StreamOptionImmediateCatchup() NatsStreamOption {
	return func(o *NatsStreamOptions) {
		o.ImmediateCatchup = true
	}
}

type NATSStream struct {
	mu *sync.RWMutex

	conn     stan.Conn
	monitor  natsMonitor
	url      url.URL
	clientID string

	lastSequences map[string]int64
	discardMsgMap map[string]map[uint64]bool
	subMap        map[string]stan.Subscription
	internal      streaminterface.Stream
	opts          NatsStreamOptions

	decodeMsgErrCnt uint64
	discardMsgCnt   uint64
	msgCnt          uint64
}

func NewNATSStream(stanDsn, clientId string, options ...NatsStreamOption) *NATSStream {
	stanUrl, err := url.Parse(stanDsn)
	if err != nil || len(stanUrl.Path) < 1 {
		log.Fatal("Missing or malformed URL. Expected 'stan://[user[:pass]@]host[:port][/cluster]'")
	}
	natsUrl := *stanUrl
	natsUrl.Scheme, natsUrl.Path = "nats", ""

	var opts NatsStreamOptions

	// default connection lost handler
	opts.ConnectionLostHandler = func(_ stan.Conn, reason error) {
		log.Output(2, fmt.Sprintf("Connection lost, reason: %v", reason))
		os.Exit(111)
	}

	// default monitor dsn
	opts.MonitorDSN = "http://" + stanUrl.Hostname() + ":8222"

	// apply user config
	for _, opt := range options {
		opt(&opts)
	}

	var memorystreamoptions []memorystream.StreamOption
	if opts.Log {
		memorystreamoptions = append(memorystreamoptions, memorystream.StreamOptionWithLog())
	}

	s := &NATSStream{
		opts:          opts,
		url:           *stanUrl,
		mu:            &sync.RWMutex{},
		lastSequences: make(map[string]int64),
		subMap:        make(map[string]stan.Subscription),
		monitor:       natsMonitor{Url: opts.MonitorDSN},
		clientID:      clientId,
	}

	// Setup nats connection
	s.conn, err = stan.Connect(stanUrl.Path[1:], clientId, stan.NatsURL(natsUrl.String()),
		stan.SetConnectionLostHandler(opts.ConnectionLostHandler),
	)
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err, stanUrl.String())
	}

	log.Printf("Connected to '%s' as client: [%s]\n", stanUrl.String(), clientId)
	return s
}

func NewNATSStreamUnique(stanDsn, clientId string, options ...NatsStreamOption) *NATSStream {
	s := NewNATSStream(stanDsn, clientId+"-"+uuid.New().String(), options...)
	s.clientID = clientId
	return s
}

func (s *NATSStream) ClientID() string {
	return s.clientID
}

func (s *NATSStream) Channels() (channels []string) {
	for ch := range s.monitor.Channels() {
		channels = append(channels, ch)
	}
	return channels
}

func (s *NATSStream) SetInvalidMessagesIds(m map[string]map[uint64]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.discardMsgMap = m
}

func (s *NATSStream) lastSequence(subject string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	seqId, exist := s.lastSequences[subject]
	if !exist {
		seqId = s.monitor.LastSequence(subject)
		s.lastSequences[subject] = seqId

		if s.discardMsgMap == nil {
			s.discardMsgMap = make(map[string]map[uint64]bool)
		}
		if s.discardMsgMap[subject] == nil {
			s.discardMsgMap[subject] = make(map[uint64]bool)
		}
	}
	return seqId
}

func (c *NATSStream) Subscribe(subject string, cb streaminterface.MessageHandler) (streaminterface.Subscription, error) {
	lastSequence := c.lastSequence(subject)
	var caughtupcount int32
	if lastSequence == 0 || c.opts.ImmediateCatchup {
		atomic.StoreInt32(&caughtupcount, 1)
		msg := caughtup.NewCaughtupMessage(subject)
		cb.HandleMessage(msg)
		log.Printf("[stan] '%s' caughtup. messages: 0", subject)
	}

	s, err := c.conn.QueueSubscribe(subject, "", func(stanMsg *stan.Msg) {
		atomic.AddUint64(&c.msgCnt, 1)

		c.mu.RLock()
		_, discard := c.discardMsgMap[subject][stanMsg.Sequence]
		c.mu.RUnlock()

		if !discard {
			msg := &message{
				channel:  stanMsg.Subject,
				sequence: stanMsg.Sequence,
			}
			err := msg.DecodeData(stanMsg.Data)
			if err != nil {
				atomic.AddUint64(&c.decodeMsgErrCnt, 1)
				//if err != messagevalidator.ErrInvalidCached {
				//log.Printf("[stan] [%s] %s", subject, err)
				//}
			} else {
				cb.HandleMessage(msg)
			}

			// we want to send the catchup event no matter if there was an error or not.
		} else {
			atomic.AddUint64(&c.discardMsgCnt, 1)
		}

		if int64(stanMsg.Sequence) >= lastSequence && atomic.LoadInt32(&caughtupcount) == 0 {
			atomic.StoreInt32(&caughtupcount, 1)
			msg := caughtup.NewCaughtupMessage(subject)
			cb.HandleMessage(msg)
			log.Printf("[stan] '%s' caughtup. messages: %d, discarded: %d, decode/validation err: %d",
				subject,
				atomic.LoadUint64(&c.msgCnt),
				atomic.LoadUint64(&c.discardMsgCnt),
				atomic.LoadUint64(&c.decodeMsgErrCnt))
		}
	}, stan.StartAtSequence(0))
	if err != nil {
		return nil, err
	}

	return s, nil
}

type Identifiable interface {
	EventID() string
	CorrelationID() string
	CausationID() string
}

func (s *NATSStream) Publish(msg streaminterface.Message) error {
	ID, ok := msg.(Identifiable)
	if !ok {
		return errors.New("Message does not implement 'Identifiable' interface")
	}

	buf, err := json.Marshal(envelope{
		EventID:       ID.EventID(),
		CorrelationID: ID.CorrelationID(),
		CausationID:   ID.CausationID(),
		Version:       0,
		Datetime:      msg.Time(),
		Type:          msg.Subject().Type(),
		Body:          msg.RawBody().(json.RawMessage),
		Meta:          msg.RawMeta().(json.RawMessage),
	})
	if err != nil {
		return errors.Wrap(err, "encode message")
	}

	err = s.conn.Publish(msg.Subject().Domain(), buf)
	if err != nil {
		return errors.Wrap(err, "publish message")
	}
	return nil
}

func (s *NATSStream) MessageFunc() streaminterface.MessageFunc {
	return func(subj streaminterface.Subject) streaminterface.MutableMessage {
		m := NewMessage()
		m.SetSubject(subj)
		return m
	}
}

func (c *NATSStream) Close() error {
	log.Println("[stan] Close")
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sub := range c.subMap {
		sub.Close()
	}
	c.conn.Close()
	log.Println("[stan] Close Ok")
	return nil
}

func LoadInvalidEventsFromReader(rd io.Reader) map[string]map[uint64]bool {
	invalidEvents := make(map[string]map[uint64]bool)
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		channel := parts[0]
		sequence, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			panic(err)
		}
		if invalidEvents[channel] == nil {
			invalidEvents[channel] = make(map[uint64]bool)
		}
		invalidEvents[channel][uint64(sequence)] = true
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return invalidEvents
}

func LoadInvalidEventsFromFile(filename string) map[string]map[uint64]bool {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	return LoadInvalidEventsFromReader(file)
}
