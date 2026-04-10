package messenger

import (
	"sync"
	"testing"
	"time"
)

func TestMockPubSub(t *testing.T) {
	client, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	var received Message
	_, err = client.Subscribe("test.topic", func(msg *Message) {
		received = *msg
		wg.Done()
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Publish("test.topic", []byte("hello")); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	if received.Subject != "test.topic" {
		t.Errorf("got subject %q, want %q", received.Subject, "test.topic")
	}
	if string(received.Data) != "hello" {
		t.Errorf("got data %q, want %q", string(received.Data), "hello")
	}
}

func TestMockPubSubWithHeaders(t *testing.T) {
	client, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	var received Message
	_, err = client.Subscribe("test.headers", func(msg *Message) {
		received = *msg
		wg.Done()
	})
	if err != nil {
		t.Fatal(err)
	}

	headers := Headers{
		HeaderTraceID:      "trace-123",
		HeaderOriginEntity: "plugin-esphome.switch_main_basement.switch_main_basement_3558733165",
	}
	if err := client.PublishWithHeaders("test.headers", []byte("hello"), headers); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	if got := TraceID(received.Headers); got != "trace-123" {
		t.Fatalf("TraceID() = %q, want %q", got, "trace-123")
	}
	if got := received.Headers[HeaderOriginEntity]; got != "plugin-esphome.switch_main_basement.switch_main_basement_3558733165" {
		t.Fatalf("origin entity = %q", got)
	}
}

func TestMockRequestWithHeaders(t *testing.T) {
	client, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	_, err = client.Subscribe("test.request", func(msg *Message) {
		if TraceID(msg.Headers) != "trace-req-1" {
			t.Errorf("TraceID() = %q, want %q", TraceID(msg.Headers), "trace-req-1")
		}
		if err := msg.RespondWithHeaders([]byte("ok"), Headers{HeaderTraceID: TraceID(msg.Headers)}); err != nil {
			t.Errorf("RespondWithHeaders: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.RequestWithHeaders("test.request", []byte("ping"), Headers{HeaderTraceID: "trace-req-1"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(resp.Data); got != "ok" {
		t.Fatalf("resp.Data = %q, want %q", got, "ok")
	}
	if got := TraceID(resp.Headers); got != "trace-req-1" {
		t.Fatalf("resp trace = %q, want %q", got, "trace-req-1")
	}
}

func TestMockWildcard(t *testing.T) {
	client, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	var mu sync.Mutex
	var subjects []string

	_, err = client.Subscribe("esphome.>", func(msg *Message) {
		mu.Lock()
		subjects = append(subjects, msg.Subject)
		mu.Unlock()
		wg.Done()
	})
	if err != nil {
		t.Fatal(err)
	}

	client.Publish("esphome.ledstrip.light01", []byte("on"))
	client.Publish("esphome.plug.switch01", []byte("off"))

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for messages")
	}

	if len(subjects) != 2 {
		t.Errorf("got %d messages, want 2", len(subjects))
	}
}
