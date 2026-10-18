//go:build linux

package lib

import (
	"testing"
	"time"
)

func TestOpenUevents_CloseUnblocksRead(t *testing.T) {
	conn, err := OpenUevents()
	if err != nil {
		t.Skipf("uevent socket unavailable: %v", err)
	}

	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := conn.Read(buf); err != nil {
				readErr <- err
				return
			}
		}
	}()
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-readErr:
		if err == nil {
			t.Error("Read after Close returned no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not unblock Read")
	}
}
