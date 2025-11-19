package tukiran

import (
	"testing"
)

func TestIsListenerPortNumber_Valid(t *testing.T) {
	tf := NewTunnelRemoteForwarder(WithListenerPort("2220"))

	if !tf.isPortNumber() {
		t.Fatalf("expected false for valid port")
	}
}
