package nats_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"nathejk.dk/pkg/memstat"
	"nathejk.dk/pkg/nats"
	"nathejk.dk/pkg/streaminterface"
	"nathejk.dk/pkg/streaminterface/caughtup"
)

func TestHelloWorld(t *testing.T) {
	// t.Fatal("not implemented")
}

func TestStreamIntegrationLoadDriver(t *testing.T) {
	stanDsn := os.Getenv("TEST_STAN_DSN")
	if stanDsn == "" {
		t.Skip("No NATS streaming server DSN")
	}

	// Connect to NATS stream
	s := nats.NewNATSStreamUnique(stanDsn, "testing")
	defer s.Close()

	var wg sync.WaitGroup

	type channel struct {
		name     string
		count    int64
		caughtup bool
		duration time.Duration
	}

	start := time.Now()

	channels := []*channel{
		{name: "driver"},
	}

	for _, ch := range channels {
		wg.Add(1)
		ch := ch
		_, _ = s.Subscribe(ch.name, streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
			ch.count++

			if caughtup.IsCaughtup(msg) {
				wg.Done()
				ch.caughtup = true
				ch.duration = time.Since(start)
			}
		}))
	}

	wg.Wait()

	if testing.Verbose() {
		var total int64
		for _, ch := range channels {
			t.Logf("`%s` read %d messages in %s\n", ch.name, ch.count, ch.duration)
			total += ch.count
		}

		t.Logf("Read total %d messages in %s\n", total, time.Since(start))
		memstat.PrintMemoryStats()
	}
}

func TestStreamIntegrationLoadAll(t *testing.T) {
	stanDsn := os.Getenv("TEST_STAN_DSN")
	if stanDsn == "" {
		t.Skip("No NATS streaming server DSN")
	}

	// Connect to NATS stream
	s := nats.NewNATSStreamUnique(stanDsn, "testing")
	defer s.Close()

	var wg sync.WaitGroup

	type channel struct {
		name     string
		count    int64
		caughtup bool
		duration time.Duration
	}

	start := time.Now()

	channels := []*channel{
		{name: "account"},
		{name: "asset"},
		{name: "asset.status"},
		{name: "attachment"},
		{name: "check"},
		{name: "contract"},
		{name: "customer"},
		{name: "damage"},
		{name: "driver"},
		{name: "haulier"},
		{name: "service"},
		{name: "telematics"},
		{name: "tire.report"},
		{name: "user"},
		{name: "workshop"},
	}

	for _, ch := range channels {
		wg.Add(1)
		ch := ch
		_, _ = s.Subscribe(ch.name, streaminterface.MessageHandlerFunc(func(msg streaminterface.Message) {
			ch.count++

			if caughtup.IsCaughtup(msg) {
				wg.Done()
				ch.caughtup = true
				ch.duration = time.Since(start)
				if testing.Verbose() {
					t.Log(ch.name, "caughtup")
				}
			}
		}))
	}

	if testing.Verbose() {
		go func() {
			for {
				for _, ch := range channels {
					if !ch.caughtup {
						t.Log("waiting on", ch.name, "messages so far", ch.count)
					}
				}
				<-time.After(time.Second)
			}
		}()
	}

	wg.Wait()

	if testing.Verbose() {
		var total int64
		for _, ch := range channels {
			t.Logf("`%s` read %d messages in %s\n", ch.name, ch.count, ch.duration)
			total += ch.count
		}

		t.Logf("Read total %d messages in %s\n", total, time.Since(start))
		memstat.PrintMemoryStats()
	}
}
