package server

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

// Config contains TCP server configuration.
type Config struct {
	Addr   string
	Logger *log.Logger
}

// Server owns Bolt's TCP listener and client connections.
type Server struct {
	addr   string
	logger *log.Logger

	mu       sync.Mutex
	listener net.Listener
	clients  map[net.Conn]struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	closed   bool
}

// New creates a TCP server. If no address is provided, Bolt listens locally.
func New(config Config) *Server {
	addr := config.Addr
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	return &Server{
		addr:    addr,
		logger:  logger,
		clients: make(map[net.Conn]struct{}),
		done:    make(chan struct{}),
	}
}

// Start opens the TCP listener and begins accepting client connections.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		listener.Close()
		return errors.New("server is shut down")
	}
	s.listener = listener
	s.mu.Unlock()

	s.logger.Printf("server listening on %s", listener.Addr().String())

	s.wg.Add(1)
	go s.acceptLoop(listener)

	return nil
}

// Addr returns the address the server is listening on.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// Shutdown stops accepting new clients and closes active client connections.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.done)

	listener := s.listener
	clients := make([]net.Conn, 0, len(s.clients))
	for conn := range s.clients {
		clients = append(clients, conn)
	}
	s.mu.Unlock()

	if listener != nil {
		listener.Close()
	}
	for _, conn := range clients {
		conn.Close()
	}

	finished := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(finished)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-finished:
		return nil
	}
}

func (s *Server) acceptLoop(listener net.Listener) {
	defer s.wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.logger.Printf("accept error: %v", err)
				continue
			}
		}

		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			conn.Close()
			return
		}
		s.clients[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		conn.Close()

		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()

		s.logger.Printf("client disconnected: %s", conn.RemoteAddr().String())
	}()

	s.logger.Printf("client connected: %s", conn.RemoteAddr().String())
	_, _ = io.Copy(io.Discard, conn)
}
