package nats

import (
	"log"
	"sync"

	"github.com/nats-io/nats.go"

	"nathejk.dk/pkg/streaminterface"
)

type Nats struct {
	conn          *nats.Conn
	BufferSize    int
	subscriptions []*nats.Subscription
}

func (s *NATSStream) Nats() *Nats {
	return &Nats{conn: s.conn.NatsConn()}
}

func NatsConnect(natsDsn string) *Nats {
	// Setup nats connection
	conn, err := nats.Connect(natsDsn)

	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Server is running at: %s", err, natsDsn)
	}
	log.Printf("Connected to '%s'\n", natsDsn)

	return &Nats{conn: conn}
}

func (c *Nats) Subscribe(subject string, cb streaminterface.MessageHandler) (*natsSub, error) {
	var wg sync.WaitGroup
	wg.Add(1)
	s, err := c.conn.QueueSubscribe(subject, "", func(natsMsg *nats.Msg) {
		wg.Wait()
		// TODO: // if we waited, messages may be out of order
		msg, err := c.DecodeNATS(natsMsg)
		if err != nil {
			log.Printf("[nats] [%s] %s", subject, err)
			return
		}
		cb.HandleMessage(msg)
	})
	if err != nil {
		return nil, err
	}
	c.subscriptions = append(c.subscriptions, s)
	msg := NewMessage()
	msg.channel = subject
	msg.msgtype = "caughtup"
	cb.HandleMessage(msg)
	wg.Done()
	return &natsSub{s}, nil
}

type natsSub struct {
	*nats.Subscription
}

func (s natsSub) Live() bool {
	return true
}

func (s *Nats) Publish(msg streaminterface.Message) error {
	/*
		rm, err := msg.RemoteMessage()
		if err != nil {
			return err
		}
	*/
	buf, err := jjson.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Publish(msg.Subject().Domain(), buf)
}

func (s *Nats) Close() error {
	for _, subscription := range s.subscriptions {
		subscription.Unsubscribe()
	}
	s.conn.Close()
	return nil
}

func (s *Nats) DecodeNATS(msg *nats.Msg) (*message, error) {
	var m message

	if err := jjson.Unmarshal(msg.Data, &m); err != nil {
		return nil, err
	}
	m.channel = msg.Subject

	return &m, nil
}
