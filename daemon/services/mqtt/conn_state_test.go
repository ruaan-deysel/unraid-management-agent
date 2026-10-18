package mqtt

import (
	"errors"

	"slices"
	"sync"
	"testing"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// connectedClient is a broker connection that is open until Disconnect.
type connectedClient struct {
	capturingClient
}

func (*connectedClient) IsConnected() bool { return true }
func (*connectedClient) Disconnect(uint)   {}

func TestConnectionStateHandler(t *testing.T) {
	config := DefaultConfig()
	config.HomeAssistantMode = false
	client := NewClient(config, "test-server", "1.0.0", nil)

	// No handler registered: state changes must not panic.
	client.handleConnect()
	client.handleDisconnect(nil)

	var (
		mu     sync.Mutex
		states []bool
	)
	client.SetConnectionStateHandler(func(connected bool) {
		mu.Lock()
		defer mu.Unlock()
		states = append(states, connected)
	})

	client.handleConnect()
	client.handleDisconnect(errors.New("lost"))
	client.client = &connectedClient{capturingClient{published: map[string]string{}}}
	client.Disconnect()

	mu.Lock()
	defer mu.Unlock()
	if want := []bool{true, false, false}; !slices.Equal(states, want) {
		t.Errorf("states = %v, want %v", states, want)
	}
}

func TestDisconnectWithoutHandler(t *testing.T) {
	client := NewClient(DefaultConfig(), "test-server", "1.0.0", nil)
	client.client = &connectedClient{capturingClient{published: map[string]string{}}}
	client.Disconnect()
	if client.IsConnected() {
		t.Error("client connected after Disconnect")
	}
}

// closedClient is a broker connection that is not open.
type closedClient struct{ pahomqtt.Client }

func (*closedClient) IsConnected() bool { return false }

// Disconnect without an open broker connection changes no state, so the
// handler is not called.
func TestDisconnectWhileNotConnectedSkipsHandler(t *testing.T) {
	for name, c := range map[string]pahomqtt.Client{"no client": nil, "closed connection": &closedClient{}} {
		client := NewClient(DefaultConfig(), "test-server", "1.0.0", nil)
		if c != nil {
			client.client = c
		}
		called := false
		client.SetConnectionStateHandler(func(bool) { called = true })
		client.Disconnect()
		if called {
			t.Errorf("%s: handler called by Disconnect without a connection", name)
		}
	}
}
