package messenger

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testLookup(action string) (reflect.Type, bool) {
	m := map[string]reflect.Type{
		"turn_on":  reflect.TypeOf(testTurnOn{}),
		"turn_off": reflect.TypeOf(testTurnOff{}),
		"set_pct":  reflect.TypeOf(testSetPct{}),
	}
	t, ok := m[action]
	return t, ok
}

// testEntity implements Keyed for testing without importing domain.
type testEntity struct {
	P string
	D string
	E string
}

func (e testEntity) Key() string { return e.P + "." + e.D + "." + e.E }

// testTurnOn implements Action for testing.
type testTurnOn struct {
	Level int `json:"level"`
}

func (testTurnOn) ActionName() string { return "turn_on" }

type testTurnOff struct{}

func (testTurnOff) ActionName() string { return "turn_off" }

// testSetPct implements Action and Validator for testing validation gating.
type testSetPct struct {
	Percentage int `json:"percentage"`
}

func (testSetPct) ActionName() string { return "set_pct" }

func (c testSetPct) Validate() error {
	if c.Percentage < 0 || c.Percentage > 100 {
		return fmt.Errorf("percentage %d out of range [0,100]", c.Percentage)
	}
	return nil
}

func TestCommandsSendAndReceive(t *testing.T) {
	msg, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Close()

	cmds := NewCommands(msg, testLookup)
	entity := testEntity{P: "test", D: "dev1", E: "ent1"}

	done := make(chan any, 1)
	cmds.Receive("test.>", func(addr Address, cmd any) {
		if addr.Plugin != "test" || addr.DeviceID != "dev1" || addr.EntityID != "ent1" {
			t.Errorf("addr: got %+v", addr)
		}
		done <- cmd
	})

	cmds.Send(entity, testTurnOn{Level: 42})

	select {
	case got := <-done:
		cmd, ok := got.(testTurnOn)
		if !ok {
			t.Fatalf("type: got %T, want testTurnOn", got)
		}
		if cmd.Level != 42 {
			t.Errorf("level: got %d, want 42", cmd.Level)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestReceiveFiltersNonCommands(t *testing.T) {
	msg, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Close()

	cmds := NewCommands(msg, testLookup)

	called := make(chan bool, 1)
	cmds.Receive("test.>", func(addr Address, cmd any) {
		called <- true
	})

	// Publish a plain message — should NOT trigger Receive
	msg.Publish("test.dev1.ent1.state", []byte(`{"power":true}`))

	select {
	case <-called:
		t.Fatal("Receive should not fire for non-command messages")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestParseAddress(t *testing.T) {
	addr, err := parseAddress("esphome.living-room.light001.command.turn_on")
	if err != nil {
		t.Fatal(err)
	}
	if addr.Plugin != "esphome" || addr.DeviceID != "living-room" || addr.EntityID != "light001" {
		t.Errorf("got %+v", addr)
	}
	if addr.Key() != "esphome.living-room.light001" {
		t.Errorf("key: got %q", addr.Key())
	}

	_, err = parseAddress("too.short")
	if err == nil {
		t.Fatal("expected error for 2-segment subject")
	}
}

func TestReceiveDropsInvalidCommands(t *testing.T) {
	msg, err := Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Close()

	cmds := NewCommands(msg, testLookup)
	entity := testEntity{P: "test", D: "dev1", E: "ent1"}

	received := make(chan testSetPct, 2)
	cmds.Receive("test.>", func(addr Address, cmd any) {
		if c, ok := cmd.(testSetPct); ok {
			received <- c
		}
	})

	// Valid command — should arrive.
	cmds.Send(entity, testSetPct{Percentage: 50})
	select {
	case got := <-received:
		if got.Percentage != 50 {
			t.Errorf("got %d, want 50", got.Percentage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for valid command")
	}

	// Invalid command (101 > 100) — should be silently dropped.
	cmds.Send(entity, testSetPct{Percentage: 101})
	select {
	case got := <-received:
		t.Errorf("invalid command should be dropped, got percentage=%d", got.Percentage)
	case <-time.After(150 * time.Millisecond):
		// expected — dropped at validation
	}
}
