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
	PublishWithHeaders(subject string, data []byte, headers Headers) error
	Request(subject string, data []byte, timeout time.Duration) (*Message, error)
	RequestWithHeaders(subject string, data []byte, headers Headers, timeout time.Duration) (*Message, error)
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

type Headers map[string]string

// Message is a received message.
type Message struct {
	Subject string
	Data    []byte
	Headers Headers
	reply   string
	conn    *nats.Conn
}

// Respond sends a reply to a request message.
func (m *Message) Respond(data []byte) error {
	return m.RespondWithHeaders(data, nil)
}

// RespondWithHeaders sends a reply to a request message with optional headers.
func (m *Message) RespondWithHeaders(data []byte, headers Headers) error {
	if m.reply == "" {
		return fmt.Errorf("message has no reply subject")
	}
	msg := &nats.Msg{Subject: m.reply, Data: data, Header: toNATSHeaders(headers)}
	return m.conn.PublishMsg(msg)
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
	return c.PublishWithHeaders(subject, data, nil)
}

func (c *client) PublishWithHeaders(subject string, data []byte, headers Headers) error {
	msg := &nats.Msg{Subject: subject, Data: data, Header: toNATSHeaders(headers)}
	return c.conn.PublishMsg(msg)
}

func (c *client) Request(subject string, data []byte, timeout time.Duration) (*Message, error) {
	return c.RequestWithHeaders(subject, data, nil, timeout)
}

func (c *client) RequestWithHeaders(subject string, data []byte, headers Headers, timeout time.Duration) (*Message, error) {
	msg := &nats.Msg{Subject: subject, Data: data, Header: toNATSHeaders(headers)}
	resp, err := c.conn.RequestMsg(msg, timeout)
	if err != nil {
		return nil, err
	}
	return fromNATSMessage(resp, c.conn), nil
}

func (c *client) Subscribe(subject string, handler func(msg *Message)) (Subscription, error) {
	return c.conn.Subscribe(subject, func(m *nats.Msg) {
		handler(fromNATSMessage(m, c.conn))
	})
}

func (c *client) Flush() error {
	return c.conn.Flush()
}

func (c *client) Close() {
	c.conn.Drain()
}

func fromNATSMessage(msg *nats.Msg, conn *nats.Conn) *Message {
	if msg == nil {
		return nil
	}
	return &Message{
		Subject: msg.Subject,
		Data:    msg.Data,
		Headers: fromNATSHeaders(msg.Header),
		reply:   msg.Reply,
		conn:    conn,
	}
}

func toNATSHeaders(headers Headers) nats.Header {
	if len(headers) == 0 {
		return nil
	}
	out := nats.Header{}
	for k, v := range headers {
		if k == "" || v == "" {
			continue
		}
		out.Set(k, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func fromNATSHeaders(headers nats.Header) Headers {
	if len(headers) == 0 {
		return nil
	}
	out := Headers{}
	for k, values := range headers {
		if len(values) == 0 || values[0] == "" {
			continue
		}
		out[k] = values[0]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
