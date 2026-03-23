package messenger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Validator is an optional interface command types can implement.
// If present, Validate is called after JSON unmarshaling.
// Commands that fail validation are silently dropped.
type Validator interface {
	Validate() error
}


type Keyed interface {
	Key() string
}

// Action is implemented by command types that know their action name.
type Action interface {
	ActionName() string
}

// CommandLookup resolves action names to concrete types.
// Provided by the caller (typically domain.LookupCommand).
type CommandLookup func(actionName string) (reflect.Type, bool)

// Address identifies an entity parsed from a message subject.
type Address struct {
	Plugin   string
	DeviceID string
	EntityID string
}

// Key returns the dot-delimited identity for this address.
func (a Address) Key() string {
	return a.Plugin + "." + a.DeviceID + "." + a.EntityID
}

// parseAddress extracts plugin, device, and entity from a dot-delimited subject.
func parseAddress(subject string) (Address, error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 3 {
		return Address{}, fmt.Errorf("subject %q has fewer than 3 segments", subject)
	}
	return Address{Plugin: parts[0], DeviceID: parts[1], EntityID: parts[2]}, nil
}

// Commands provides entity-aware command and state messaging.
type Commands struct {
	msg    Messenger
	lookup CommandLookup
}

// NewCommands creates a Commands instance.
// lookup resolves action names to concrete types (typically domain.LookupCommand).
func NewCommands(msg Messenger, lookup CommandLookup) *Commands {
	return &Commands{msg: msg, lookup: lookup}
}

// Send publishes a command targeting the given entity.
// Subject: {target.Key()}.command.{cmd.ActionName()}
func (c *Commands) Send(target Keyed, cmd Action) error {
	subject := target.Key() + ".command." + cmd.ActionName()
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("command: marshal: %w", err)
	}
	return c.msg.Publish(subject, data)
}

// Receive subscribes to commands on subjects matching the pattern.
// The handler receives the address, action name, and the concrete command
// (auto-hydrated via the command registry). Unknown actions are skipped.
func (c *Commands) Receive(pattern string, handler func(addr Address, cmd any)) (Subscription, error) {
	return c.msg.Subscribe(pattern, func(m *Message) {
		// Extract action name from subject: ...command.<action>
		idx := strings.LastIndex(m.Subject, ".command.")
		if idx < 0 {
			return
		}
		action := m.Subject[idx+len(".command."):]

		addr, err := parseAddress(m.Subject)
		if err != nil {
			return
		}

		if c.lookup == nil {
			return
		}
		typ, ok := c.lookup(action)
		if !ok {
			return
		}

		v := reflect.New(typ).Interface()
		if err := json.Unmarshal(m.Data, v); err != nil {
			return
		}
		cmd := reflect.ValueOf(v).Elem().Interface()
		if val, ok := cmd.(Validator); ok {
			if err := val.Validate(); err != nil {
				return
			}
		}
		handler(addr, cmd)
	})
}
