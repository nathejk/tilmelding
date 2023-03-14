package memorystream_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"nathejk.dk/pkg/memorystream"
	"nathejk.dk/pkg/streaminterface"
)

func TestMemoryStream(t *testing.T) {
	s := memorystream.New()
	var wg sync.WaitGroup
	_, err := s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
		wg.Done()
	}))
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	err = s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain")))
	if err != nil {
		t.Fatal(err)
	}
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatalf("waiting timed out after %s", "5s")
	}

}

func TestMemoryStreamClose(t *testing.T) {
	s := memorystream.New()
	ch := make(chan struct{})
	sub, err := s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
		ch <- struct{}{}
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = sub.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain")))
	if err != nil {
		t.Fatal(err)
	}

	// Not super great way to test this with a timeout, but couldn't think of a
	// better way. How to test that something async doesn't happen?
	select {
	case <-ch:
		t.Fatal("unexpected message from closed sub")
	case <-time.After(time.Millisecond * 10):
	}
}

func TestMemoryStreamWithLog(t *testing.T) {
	s := memorystream.New(memorystream.StreamOptionWithLog())
	var wg sync.WaitGroup
	_, _ = s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
		wg.Done()
	}))

	wg.Add(1)
	err := s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain")))
	if err != nil {
		t.Fatal(err)
	}
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatalf("waiting timed out after %s", "5s")
	}

	wg.Add(1)
	_, _ = s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
		wg.Done()
	}))
	if !waitTimeout(&wg, 5*time.Second) {
		t.Fatalf("waiting timed out after %s", "5s")
	}

}

func BenchmarkMemoryStream(b *testing.B) {
	s := memorystream.New()

	var msg streaminterface.Message
	var wg sync.WaitGroup
	_, _ = s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(m streaminterface.Message) {
		msg = m
		wg.Done()
	}))

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		err := s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain")))
		if err != nil {
			b.Fatal(err)
		}
	}

	if !waitTimeout(&wg, 5*time.Second) {
		b.Fatalf("waiting timed out after %s", "5s")
	}

	_ = msg
	//fmt.Println(msg.Sequence)
	stats := s.(memorystream.StreamStatistics).Stats()
	if stats.InMsgs != stats.OutMsgs {
		b.Fatalf("exp %d inmsg = %d outmsg", stats.InMsgs, stats.OutMsgs)
	}
	//fmt.Println(s.(stream.Stats).Stats().Format())
}

func benchmarkPubSub(b *testing.B, n int) {
	s := memorystream.New()

	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		_, _ = s.Subscribe("domain", streaminterface.MessageHandlerFunc(func(m streaminterface.Message) {
			wg.Done()
		}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(n)
		err := s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain:type")))
		if err != nil {
			b.Fatal(err)
		}
	}

	if !waitTimeout(&wg, 5*time.Second) {
		b.Fatalf("waiting timed out after %s", "5s")
	}

	stats := s.(memorystream.StreamStatistics).Stats()
	if stats.InMsgs != stats.OutMsgs {
		b.Fatalf("exp %d inmsg = %d outmsg", stats.InMsgs, stats.OutMsgs)
	}

	if testing.Verbose() {
		fmt.Println(s.(memorystream.StreamStatistics).Stats().Format())
	}
}

func BenchmarkPubSub1(b *testing.B)  { benchmarkPubSub(b, 1) }
func BenchmarkPubSub5(b *testing.B)  { benchmarkPubSub(b, 5) }
func BenchmarkPubSub10(b *testing.B) { benchmarkPubSub(b, 10) }

func benchmarkPub(b *testing.B, n int) {
	s := memorystream.New()
	for i := 0; i < b.N; i++ {
		err := s.Publish(s.MessageFunc()(streaminterface.SubjectFromStr("domain:type")))
		if err != nil {
			b.Fatal(err)
		}
	}

	if testing.Verbose() {
		fmt.Println(s.(memorystream.StreamStatistics).Stats().Format())
	}
}

func BenchmarkPub1(b *testing.B)  { benchmarkPub(b, 1) }
func BenchmarkPub5(b *testing.B)  { benchmarkPub(b, 5) }
func BenchmarkPub10(b *testing.B) { benchmarkPub(b, 10) }

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}
