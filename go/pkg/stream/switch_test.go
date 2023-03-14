package stream_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/stream"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streaminterface/caughtup"
)

type assetMessageBody struct {
	AssetID string
	Details map[string]interface{}
}

/*
func testMessage(t string) *message.Message {
	m := message.NewMessage()
	m.Type = t
	return m
}*/

type testHandler struct {
	stream     streaminterface.Stream
	handler    streaminterface.MessageHandlerFunc
	subscribes []string
	produces   []string
	caughtup   func()
}

func (s *testHandler) CaughtUp() {
	if s.caughtup != nil {
		s.caughtup()
	}
}

func (s *testHandler) Consumes() (subjs []streaminterface.Subject) {
	for _, subj := range s.subscribes {
		subjs = append(subjs, streaminterface.SubjectFromStr(subj))
	}
	return
}

func (s *testHandler) Produces() (subjs []streaminterface.Subject) {
	for _, subj := range s.produces {
		subjs = append(subjs, streaminterface.SubjectFromStr(subj))
	}
	return
}

func (s *testHandler) HandleMessage(m streaminterface.Message) {
	s.handler(m)
}

func TestSwitch(t *testing.T) {
	s := memorystream.New()
	var wg sync.WaitGroup

	h1 := &testHandler{
		stream:     s,
		subscribes: []string{"service:updated", "service:removed"},
		produces:   []string{"servicemodel:updated", "servicemodel:removed"},
		handler: func(m streaminterface.Message) {
			//fmt.Println("DEBUG msg", m.Subject().Subject())
		},
	}

	wg.Add(2)
	mux := stream.NewStreamMux(s)
	newMessage := s.MessageFunc()
	swtch, err := stream.NewSwitch(mux,
		[]streaminterface.Consumer{
			h1,
		},
		stream.SwitchSubscribedFunc(func() {
			s.Publish(newMessage(streaminterface.SubjectFromStr("service:updated")))
			s.Publish(newMessage(streaminterface.SubjectFromStr("service:updated")))
			s.Publish(newMessage(streaminterface.SubjectFromStr("service:caughtup")))
			s.Publish(caughtup.NewCaughtupMessage("service"))
			s.Publish(caughtup.NewCaughtupMessage("servicemodel"))
			wg.Done()
		}),
		stream.SwitchCaughtupFunc(func() {
			wg.Done()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := swtch.Run(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()
	// wait caughtup
	wg.Wait()
}

/*
func TestSwitchNats(t *testing.T) {
	stanDsn := os.Getenv("TEST_STAN_DSN")
	if stanDsn == "" {
		t.Skip("No NATS streaming server DSN")
	}

	// Connect to NATS stream
	remotestream := nats.NewNATSStreamUnique(stanDsn, "testing")
	defer remotestream.Close()

	localstream := stream.NewMemoryStream()
	var wg sync.WaitGroup

	h1 := &testHandler{
		stream:     localstream,
		subscribes: []string{"service:updated", "service:removed"},
		produces:   []string{"servicemodel:updated", "servicemodel:removed"},
		handler: func(m *message.Message) {
			//fmt.Println("msg", m)
			//wg.Done()
		},
		caughtup: func() {
			wg.Done()
		},
	}

	mux := stream.NewStreamMux(localstream)
	mux.Handle(remotestream, "service")
	swtch, err := stream.NewSwitch(mux, []stream.SubHandler{
		h1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// ready
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := swtch.Run(ctx, func() {
			wg.Done()
		})
		if err != nil {
			fmt.Println(err)
		}
	}()

	wg.Add(1)

	// wait ready
	wg.Wait()

	cancel()
}

func TestSwitchNatsFull(t *testing.T) {
	stanDsn := os.Getenv("TEST_STAN_DSN")
	if stanDsn == "" {
		t.Skip("No NATS streaming server DSN")
	}

	// Connect to NATS stream
	remotestream := nats.NewNATSStreamUnique(stanDsn, "testing")
	defer remotestream.Close()

	localstream := stream.NewMemoryStream()
	var wg sync.WaitGroup

	mux := stream.NewStreamMux(localstream)
	mux.Handles(remotestream, "service", "asset", "asset.status")
	swtch, err := stream.NewSwitch(mux, []stream.SubHandler{
		// Aggregates
		asset.NewDetailsAggregator(localstream),
		asset.NewOwnerAggregator(localstream),
		asset.NewServiceAggregator(localstream),
	})
	if err != nil {
		t.Fatal(err)
	}

	// ready
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := swtch.Run(ctx, func() {
			memstat.PrintMemoryStats()
			wg.Done()
		})
		if err != nil {
			panic(err)
		}
	}()

	wg.Add(1)

	// wait ready
	time.AfterFunc(time.Second*10, func() {
		memstat.PrintMemoryStats()
	})
	wg.Wait()

	cancel()
}
*/

func BenchmarkSwitchHandler(b *testing.B) {
	s := memorystream.New()
	var wg sync.WaitGroup

	h1 := &testHandler{
		stream:     s,
		subscribes: []string{"service:updated", "service:removed"},
		produces:   []string{"servicemodel:updated", "servicemodel:removed"},
		handler: func(m streaminterface.Message) {
			wg.Done()
		},
	}

	// ready
	wg.Add(1)
	mux := stream.NewStreamMux(s)
	swtch, err := stream.NewSwitch(mux, []streaminterface.Consumer{
		h1,
	}, stream.SwitchSubscribedFunc(func() {
		wg.Done()
	}))
	if err != nil {
		b.Fatal(err)
	}

	go func() {
		err := swtch.Run(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()

	// wait ready
	wg.Wait()
	b.ResetTimer()

	newMessage := s.MessageFunc()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		s.Publish(newMessage(streaminterface.SubjectFromStr("service:updated")))
	}
	wg.Wait()
}

type nullstream struct{}

func (c *nullstream) Publish(msg streaminterface.Message) error {
	return nil
}
func (c *nullstream) MessageFunc() streaminterface.MessageFunc {
	return nil
}

func (c *nullstream) Subscribe(subject string, cb streaminterface.MessageHandler) (streaminterface.Subscription, error) {
	panic("null stream")
}

func (c *nullstream) Close() error { return nil }

func seedassets(newMsg streaminterface.MessageFunc, N int) []streaminterface.Message {
	// seed messages with deterministic data
	//var eventId int64 = 1
	var assetId int64 = 1
	//var timeId int64
	epoch := time.Date(2018, 6, 2, 12, 58, 0, 0, time.UTC)

	r := rand.New(rand.NewSource(epoch.Unix()))
	nextAssetId := func() string {
		assetId++
		return strconv.FormatInt(assetId, 10)
	}

	randId := func() string {
		i := r.Int63n(assetId)
		if i == 0 {
			i++
		}
		return strconv.FormatInt(i, 10)
	}

	slice := make([]streaminterface.Message, 0, N)

	for i := 0; i < N; i++ {
		var m streaminterface.MutableMessage

		// delete 1/10
		if i%10 == 0 {
			m = newMsg(streaminterface.SubjectFromStr("asset:details.removed"))
			m.SetBody(&assetMessageBody{
				AssetID: randId(),
				Details: map[string]interface{}{"a": nil, "b": nil, "c": nil, "d": nil, "e": nil},
			})
		} else if i%5 == 0 {
			//m.Type = "details.removed"
			m = newMsg(streaminterface.SubjectFromStr("asset:details.removed"))
			m.SetBody(&assetMessageBody{
				AssetID: randId(),
				Details: map[string]interface{}{"a": nil, "b": nil},
			})
		} else {
			//m.Type = "details.added"
			m = newMsg(streaminterface.SubjectFromStr("asset:details.added"))
			m.SetBody(&assetMessageBody{
				AssetID: nextAssetId(),
				Details: map[string]interface{}{
					"a": "a",
					"b": 123,
					"c": "c",
					"d": 456,
					"e": "e",
				},
			})
		}

		slice = append(slice, m.(streaminterface.Message))
	}

	return slice
}

func benchmarkSwitchComplexN(b *testing.B, N int) {
	var wg sync.WaitGroup

	remotestream := memorystream.New()
	localstream := &nullstream{}
	mux := stream.NewStreamMux(localstream)
	mux.Handle(remotestream, "asset")

	// on subscribe
	wg.Add(1)
	var wgcaught sync.WaitGroup
	wgcaught.Add(1)

	consumer := &testHandler{
		subscribes: []string{"asset:details.added", "asset:details.removed"},
		produces:   []string{"g:updated", "g:removed"},
		handler:    func(m streaminterface.Message) {},
	}
	swtch, err := stream.NewSwitch(mux, []streaminterface.Consumer{
		//asset.NewDetailsAggregator(localstream),
		consumer,
	}, stream.SwitchSubscribedFunc(func() {
		wg.Done()
	}), stream.SwitchCaughtupFunc(func() {
		wgcaught.Done()
	}))
	if err != nil {
		b.Fatal(err)
	}

	go func() {
		err := swtch.Run(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()

	newMessage := remotestream.MessageFunc()
	assets := seedassets(newMessage, N)

	// wait on subscribed
	wg.Wait()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		remotestream.Publish(assets[i%N])
	}

	remotestream.Publish(newMessage(streaminterface.SubjectFromStr("asset:caughtup")))

	// wait on caughtup
	wgcaught.Wait()
}

func BenchmarkSwitchComplex1000(b *testing.B)   { benchmarkSwitchComplexN(b, 1000) }
func BenchmarkSwitchComplex10000(b *testing.B)  { benchmarkSwitchComplexN(b, 10000) }
func BenchmarkSwitchComplex100000(b *testing.B) { benchmarkSwitchComplexN(b, 100000) }
