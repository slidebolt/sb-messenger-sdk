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
