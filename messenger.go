// Package messenger provides the client SDK for connecting to the
// SlideBolt messenger service (NATS) from other binaries.
package messenger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Messenger is the interface for pub/sub messaging.
type Messenger interface {
	Publish(subject string, data []byte) error
	Request(subject string, data []byte, timeout time.Duration) (*Message, error)
	Subscribe(subject string, handler func(msg *Message)) (Subscription, error)
	// Flush ensures all pending outgoing data has been received by the server.
	// Call after Subscribe to guarantee subscriptions are active before sending requests.
	Flush() error
	Close()
}

// Subscription represents an active subscription.
type Subscription interface {
	Unsubscribe() error
}

// manifest is the runtime payload the messenger service advertises.
type manifest struct {
	NatsURL  string `json:"nats_url"`
	NatsPort int    `json:"nats_port"`
}

// Message is a received message.
type Message struct {
	Subject string
	Data    []byte
	reply   string
	conn    *nats.Conn
}

// Respond sends a reply to a request message.
func (m *Message) Respond(data []byte) error {
	if m.reply == "" {
		return fmt.Errorf("message has no reply subject")
	}
	return m.conn.Publish(m.reply, data)
}

// Connect extracts the messenger payload from deps and dials NATS.
func Connect(deps map[string]json.RawMessage) (Messenger, error) {
	raw, ok := deps["messenger"]
	if !ok {
		return nil, fmt.Errorf("messenger: dependency payload not found")
	}

	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("messenger: parse payload: %w", err)
	}

	return connectDirect(fmt.Sprintf("nats://%s:%d", m.NatsURL, m.NatsPort))
}

// ConnectURL dials a NATS server at the given URL.
func ConnectURL(url string) (Messenger, error) {
	return connectDirect(url)
}

// connectDirect dials a NATS server at the given URL.
func connectDirect(url string) (Messenger, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("messenger: connect: %w", err)
	}
	return &client{conn: conn}, nil
}

// client is the real NATS-backed implementation.
type client struct {
	conn *nats.Conn
}

func (c *client) Publish(subject string, data []byte) error {
	return c.conn.Publish(subject, data)
}

func (c *client) Request(subject string, data []byte, timeout time.Duration) (*Message, error) {
	resp, err := c.conn.Request(subject, data, timeout)
	if err != nil {
		return nil, err
	}
	return &Message{Subject: resp.Subject, Data: resp.Data, reply: resp.Reply, conn: c.conn}, nil
}

func (c *client) Subscribe(subject string, handler func(msg *Message)) (Subscription, error) {
	return c.conn.Subscribe(subject, func(m *nats.Msg) {
		handler(&Message{Subject: m.Subject, Data: m.Data, reply: m.Reply, conn: c.conn})
	})
}

func (c *client) Flush() error {
	return c.conn.Flush()
}

func (c *client) Close() {
	c.conn.Drain()
}
