package server

import (
	"bytes"
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServerStartsOnConfiguredAddressAndAcceptsConnection(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0"})
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
	})

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()
}

func TestServerAcceptsMultipleClientsConcurrently(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0"})
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
	})

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	const clients = 5
	conns := make([]net.Conn, 0, clients)
	for i := 0; i < clients; i++ {
		conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
		if err != nil {
			t.Fatalf("dial client %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	for _, conn := range conns {
		conn.Close()
	}
}

func TestServerLogsConnections(t *testing.T) {
	var logs lockedBuffer
	logger := log.New(&logs, "", 0)
	srv := New(Config{
		Addr:   "127.0.0.1:0",
		Logger: logger,
	})
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
	})

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	conn.Close()

	waitFor(t, time.Second, func() bool {
		return strings.Contains(logs.String(), "client connected") &&
			strings.Contains(logs.String(), "client disconnected")
	})
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func TestServerShutdownStopsAcceptingConnectionsAndClosesClients(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown server: %v", err)
	}

	if _, err := net.DialTimeout("tcp", srv.Addr(), 100*time.Millisecond); err == nil {
		t.Fatal("expected dialing a shut down server to fail")
	}

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected existing client connection to be closed")
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}
