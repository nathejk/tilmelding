package stream

import (
	"log"

	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/streaminterface"
)

type fanoutHandler struct {
	s    streaminterface.Stream
	subj string
}

func newFanoutHandler(subject string, handlers []streaminterface.MessageHandler) streaminterface.MessageHandler {
	if len(handlers) == 0 {
		panic("at least one handler required")
	}

	if len(handlers) == 1 {
		return handlers[0]
	}

	s := memorystream.New()
	for _, h := range handlers {
		_, err := s.Subscribe(subject, h)
		if err != nil {
			panic(err)
		}
	}

	return &fanoutHandler{
		s:    s,
		subj: subject,
	}
}

func (fh *fanoutHandler) HandleMessage(m streaminterface.Message) {
	if err := fh.s.Publish(m); err != nil {
		log.Println(err)
	}
}
