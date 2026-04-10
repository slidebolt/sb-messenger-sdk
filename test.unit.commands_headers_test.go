package messenger

import (
	"testing"
	"time"
)

type headerTestKey struct{ key string }

func (k headerTestKey) Key() string { return k.key }

func TestCommandsReceiveMessageIncludesHeaders(t *testing.T) {
	msg, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Close()

	cmds := NewCommands(msg, testLookup)
	done := make(chan *Message, 1)
	_, err = cmds.ReceiveMessage("test.dev1.>", func(addr Address, cmd any, msg *Message) {
		if addr.Key() != "test.dev1.ent1" {
			t.Errorf("addr.Key() = %q", addr.Key())
		}
		if _, ok := cmd.(testTurnOn); !ok {
			t.Errorf("cmd type = %T, want testTurnOn", cmd)
		}
		done <- msg
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := cmds.SendWithHeaders(headerTestKey{key: "test.dev1.ent1"}, testTurnOn{}, Headers{HeaderTraceID: "trace-cmd-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-done:
		if TraceID(got.Headers) != "trace-cmd-1" {
			t.Fatalf("TraceID() = %q, want %q", TraceID(got.Headers), "trace-cmd-1")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command")
	}
}
