package tukiran

import (
	"testing"
)

func TestIsListenerPortNumber_Valid(t *testing.T) {
	tf := NewTunnelRemoteForwarder(WithListenerPort("2220"))

	if tf.listener.port != "2220" {
		t.Fatalf("Server forwarding port is not set properly")
	}
}
