package server

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stevenstank/bolt/internal/pubsub"
)

func TestServerPubsubSubscribeAndPublish(t *testing.T) {
	hub := pubsub.NewHub()
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		PubsubHub: hub,
	})

	srv.SetProcessorInfo()

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	addr := srv.Addr()

	conn1, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn2.Close()

	// Subscribe conn1 to news
	if _, err := conn1.Write([]byte("SUBSCRIBE news\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := bufio.NewReader(conn1).ReadString('\n')
	if err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}
	conn1.SetReadDeadline(time.Time{})
	if !strings.Contains(resp, "subscribed to news") {
		t.Fatalf("unexpected subscribe response: %s", resp)
	}

	// Subscribe conn2 to news
	if _, err := conn2.Write([]byte("SUBSCRIBE news\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err = bufio.NewReader(conn2).ReadString('\n')
	if err != nil {
		t.Fatalf("read subscribe response: %v", err)
	}
	conn2.SetReadDeadline(time.Time{})
	if !strings.Contains(resp, "subscribed to news") {
		t.Fatalf("unexpected subscribe response: %s", resp)
	}

	// Publish from conn2
	if _, err := conn2.Write([]byte("PUBLISH news hello\n")); err != nil {
		t.Fatalf("write publish: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err = bufio.NewReader(conn2).ReadString('\n')
	if err != nil {
		t.Fatalf("read publish response: %v", err)
	}
	conn2.SetReadDeadline(time.Time{})
	if resp != "2\n" {
		t.Fatalf("expected 2 subscribers, got %s", resp)
	}

	// Check conn1 received the message
	conn1.SetReadDeadline(time.Now().Add(10 * time.Second))
	msg, err := bufio.NewReader(conn1).ReadString('\n')
	if err != nil {
		t.Fatalf("read message from conn1: %v", err)
	}
	conn1.SetReadDeadline(time.Time{})
	if !strings.Contains(msg, "MESSAGE news hello") {
		t.Fatalf("unexpected message: %s", msg)
	}

	// Check conn2 received the message
	conn2.SetReadDeadline(time.Now().Add(10 * time.Second))
	msg, err = bufio.NewReader(conn2).ReadString('\n')
	if err != nil {
		t.Fatalf("read message from conn2: %v", err)
	}
	conn2.SetReadDeadline(time.Time{})
	if !strings.Contains(msg, "MESSAGE news hello") {
		t.Fatalf("unexpected message: %s", msg)
	}
}

func TestServerPubsubUnsubscribe(t *testing.T) {
	hub := pubsub.NewHub()
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		PubsubHub: hub,
	})

	srv.SetProcessorInfo()

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	addr := srv.Addr()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	// Subscribe
	if _, err := conn.Write([]byte("SUBSCRIBE news\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn).ReadString('\n')
	conn.SetReadDeadline(time.Time{})

	// Unsubscribe
	if _, err := conn.Write([]byte("UNSUBSCRIBE news\n")); err != nil {
		t.Fatalf("write unsubscribe: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read unsubscribe response: %v", err)
	}
	conn.SetReadDeadline(time.Time{})
	if !strings.Contains(resp, "unsubscribed from news") {
		t.Fatalf("unexpected unsubscribe response: %s", resp)
	}

	// Publish should return 0 subscribers
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn2.Close()

	if _, err := conn2.Write([]byte("PUBLISH news hello\n")); err != nil {
		t.Fatalf("write publish: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err = bufio.NewReader(conn2).ReadString('\n')
	if err != nil {
		t.Fatalf("read publish response: %v", err)
	}
	conn2.SetReadDeadline(time.Time{})
	if resp != "0\n" {
		t.Fatalf("expected 0 subscribers, got %s", resp)
	}
}

func TestServerPubsubAutoCleanupOnDisconnect(t *testing.T) {
	hub := pubsub.NewHub()
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		PubsubHub: hub,
	})

	srv.SetProcessorInfo()

	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	addr := srv.Addr()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}

	// Subscribe
	if _, err := conn.Write([]byte("SUBSCRIBE news\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn).ReadString('\n')
	conn.SetReadDeadline(time.Time{})

	// Close connection
	conn.Close()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Publish should return 0 subscribers
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn2.Close()

	if _, err := conn2.Write([]byte("PUBLISH news hello\n")); err != nil {
		t.Fatalf("write publish: %v", err)
	}

	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := bufio.NewReader(conn2).ReadString('\n')
	if err != nil {
		t.Fatalf("read publish response: %v", err)
	}
	conn2.SetReadDeadline(time.Time{})
	if resp != "0\n" {
		t.Fatalf("expected 0 subscribers after disconnect, got %s", resp)
	}
}

func TestServerPubsubMultipleChannels(t *testing.T) {
	hub := pubsub.NewHub()
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		PubsubHub: hub,
	})

	srv.SetProcessorInfo()
	if err := srv.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	addr := srv.Addr()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	// Subscribe to multiple channels
	if _, err := conn.Write([]byte("SUBSCRIBE news\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn).ReadString('\n')
	conn.SetReadDeadline(time.Time{})

	if _, err := conn.Write([]byte("SUBSCRIBE sports\n")); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn).ReadString('\n')
	conn.SetReadDeadline(time.Time{})

	// Publish to news
	conn2, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn2.Close()

	if _, err := conn2.Write([]byte("PUBLISH news news_update\n")); err != nil {
		t.Fatalf("write publish: %v", err)
	}
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn2).ReadString('\n')
	conn2.SetReadDeadline(time.Time{})

	// Check message received
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	conn.SetReadDeadline(time.Time{})
	if !strings.Contains(msg, "MESSAGE news news_update") {
		t.Fatalf("unexpected message: %s", msg)
	}

	// Publish to sports
	if _, err := conn2.Write([]byte("PUBLISH sports sports_update\n")); err != nil {
		t.Fatalf("write publish: %v", err)
	}
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	bufio.NewReader(conn2).ReadString('\n')
	conn2.SetReadDeadline(time.Time{})

	// Check message received
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err = bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	conn.SetReadDeadline(time.Time{})
	if !strings.Contains(msg, "MESSAGE sports sports_update") {
		t.Fatalf("unexpected message: %s", msg)
	}
}
