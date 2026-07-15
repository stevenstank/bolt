package server

import (
	"bufio"
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

func TestServerProcessesMultipleCommandsOnOneConnection(t *testing.T) {
	processor := &recordingProcessor{
		responses: []string{"OK", "saksham", "(nil)"},
	}
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Processor: processor,
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
	defer conn.Close()

	reader := bufio.NewReader(conn)
	commands := []string{"SET name saksham\n", "GET name\n", "GET missing\n"}
	wantResponses := []string{"OK\n", "saksham\n", "(nil)\n"}

	for i, command := range commands {
		if _, err := conn.Write([]byte(command)); err != nil {
			t.Fatalf("write command: %v", err)
		}
		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read response: %v", err)
		}
		if response != wantResponses[i] {
			t.Fatalf("expected response %q, got %q", wantResponses[i], response)
		}
	}

	processor.mu.Lock()
	defer processor.mu.Unlock()
	if got := processor.commands; len(got) != 3 || got[0] != "SET name saksham" || got[1] != "GET name" || got[2] != "GET missing" {
		t.Fatalf("expected processed commands, got %v", got)
	}
}

func TestServerKeepsConnectionOpenAfterErrorResponse(t *testing.T) {
	processor := &recordingProcessor{
		responses: []string{"ERR unknown command \"DEL\"", "OK"},
	}
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Processor: processor,
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
	defer conn.Close()

	reader := bufio.NewReader(conn)
	if _, err := conn.Write([]byte("DEL name\n")); err != nil {
		t.Fatalf("write invalid command: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR unknown command \"DEL\"\n" {
		t.Fatalf("expected error response, got %q, err=%v", response, err)
	}

	if _, err := conn.Write([]byte("SET name saksham\n")); err != nil {
		t.Fatalf("write valid command: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK\n" {
		t.Fatalf("expected OK response after error, got %q, err=%v", response, err)
	}
}

func TestServerHandsOffReplicaSyncConnections(t *testing.T) {
	acceptor := &recordingReplicaAccepter{}
	srv := New(Config{
		Addr:            "127.0.0.1:0",
		ReplicaAccepter: acceptor,
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
	defer conn.Close()

	if _, err := conn.Write([]byte("SYNC\n")); err != nil {
		t.Fatalf("write sync command: %v", err)
	}

	waitFor(t, time.Second, func() bool {
		acceptor.mu.Lock()
		defer acceptor.mu.Unlock()
		return acceptor.called == 1
	})
}

type recordingProcessor struct {
	mu        sync.Mutex
	responses []string
	commands  []string
}

func (p *recordingProcessor) Process(line string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.commands = append(p.commands, line)
	if len(p.responses) == 0 {
		return "OK"
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response
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

type recordingReplicaAccepter struct {
	mu     sync.Mutex
	called int
}

func (a *recordingReplicaAccepter) AcceptReplica(conn net.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.called++
}
