//go:build windows

package main

import (
    "net"
    "strings"
    "testing"
    "time"
)

// The updater must refuse to install while 8765 (or any target listener)
// is occupied and must proceed once the listener is closed.
func TestWaitForPortReleased(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    address := listener.Addr().String()
    if err := waitForPortReleased(address, 230*time.Millisecond); err == nil || !strings.Contains(err.Error(), "continua ocupada") {
        listener.Close()
        t.Fatalf("porta ocupada foi aceita: %v", err)
    }
    if err := listener.Close(); err != nil { t.Fatal(err) }
    if err := waitForPortReleased(address, time.Second); err != nil {
        t.Fatalf("porta liberada recusada: %v", err)
    }
}
