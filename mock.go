package messenger

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

// mock wraps an in-process NATS server + real client. Tests get real
// pub/sub behavior with zero external infrastructure.
type mock struct {
	server *natsserver.Server
	client Messenger
	port   int
}

// Mock starts an embedded NATS server and returns a connected Messenger.
// Call Close() when done — it stops both the client and the server.
func Mock() (Messenger, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("messenger mock: free port: %w", err)
	}

	opts := &natsserver.Options{
		Host: "127.0.0.1",
		Port: port,
	}

	ns, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("messenger mock: create server: %w", err)
	}

	ns.Start()
	if !ns.ReadyForConnections(5_000_000_000) {
		return nil, fmt.Errorf("messenger mock: server failed to start")
	}

	c, err := connectDirect(fmt.Sprintf("nats://127.0.0.1:%d", port))
	if err != nil {
		ns.Shutdown()
		return nil, fmt.Errorf("messenger mock: connect: %w", err)
	}

	return &mock{server: ns, client: c, port: port}, nil
}

// MockWithPayload starts an embedded NATS server and returns a connected
// Messenger along with the JSON payload that other services use to connect.
func MockWithPayload() (Messenger, json.RawMessage, error) {
	m, err := Mock()
	if err != nil {
		return nil, nil, err
	}
	mk := m.(*mock)
	payload, err := json.Marshal(manifest{NatsURL: "127.0.0.1", NatsPort: mk.port})
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	return m, payload, nil
}

func (m *mock) Publish(subject string, data []byte) error {
	return m.client.Publish(subject, data)
}

func (m *mock) Request(subject string, data []byte, timeout time.Duration) (*Message, error) {
	return m.client.Request(subject, data, timeout)
}

func (m *mock) Subscribe(subject string, handler func(msg *Message)) (Subscription, error) {
	return m.client.Subscribe(subject, handler)
}

func (m *mock) Flush() error {
	return m.client.Flush()
}

func (m *mock) Close() {
	m.client.Close()
	m.server.Shutdown()
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
