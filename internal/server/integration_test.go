package server_test

import (
	"bufio"
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stevenstank/bolt/internal/command"
	"github.com/stevenstank/bolt/internal/engine"
	"github.com/stevenstank/bolt/internal/server"
	"github.com/stevenstank/bolt/internal/storage"
)

func TestServerPersistsDataAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	aofPath := filepath.Join(dir, "bolt.aof")
	snapshotPath := filepath.Join(dir, "bolt.snapshot")

	store, err := storage.NewDurableStore(aofPath, snapshotPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	first := server.New(server.Config{
		Addr:      "127.0.0.1:0",
		Processor: command.NewProcessor(command.NewDispatcher(engine.New(store))),
	})
	if err := first.Start(); err != nil {
		t.Fatalf("start first server: %v", err)
	}

	firstClient := dialServer(t, first.Addr())
	assertCommandResponse(t, firstClient, "SET name saksham\n", "OK\n")
	firstClient.Close()
	if err := store.SaveSnapshot(); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown first server: %v", err)
	}

	restartedStore, err := storage.NewDurableStore(aofPath, snapshotPath)
	if err != nil {
		t.Fatalf("restart store: %v", err)
	}
	second := server.New(server.Config{
		Addr:      "127.0.0.1:0",
		Processor: command.NewProcessor(command.NewDispatcher(engine.New(restartedStore))),
	})
	t.Cleanup(func() {
		_ = second.Shutdown(context.Background())
	})
	if err := second.Start(); err != nil {
		t.Fatalf("start second server: %v", err)
	}

	secondClient := dialServer(t, second.Addr())
	defer secondClient.Close()
	assertCommandResponse(t, secondClient, "GET name\n", "saksham\n")
}

func dialServer(t *testing.T, addr string) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	return conn
}

func assertCommandResponse(t *testing.T, conn net.Conn, command, want string) {
	t.Helper()

	if _, err := conn.Write([]byte(command)); err != nil {
		t.Fatalf("write command: %v", err)
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response != want {
		t.Fatalf("expected response %q, got %q", want, response)
	}
}
