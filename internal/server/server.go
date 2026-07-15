package server

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"os"
	"sync"
)

// Processor handles one client command line.
type Processor interface {
	Process(line string) string
}

// Config contains TCP server configuration.
type Config struct {
	Addr            string
	Logger          *log.Logger
	Processor       Processor
	ReplicaAccepter ReplicaAccepter
}

// ReplicaAccepter receives replica handshake connections.
type ReplicaAccepter interface {
	AcceptReplica(conn net.Conn)
}

// Server owns Bolt's TCP listener and client connections.
type Server struct {
	addr      string
	logger    *log.Logger
	processor Processor
	replicaAccepter ReplicaAccepter

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
		addr:      addr,
		logger:    logger,
		processor: config.Processor,
		replicaAccepter: config.ReplicaAccepter,
		clients:   make(map[net.Conn]struct{}),
		done:      make(chan struct{}),
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
	handedOff := false
	defer func() {
		if handedOff {
			return
		}
		conn.Close()

		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()

		s.logger.Printf("client disconnected: %s", conn.RemoteAddr().String())
	}()

	s.logger.Printf("client connected: %s", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	isFirstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if isFirstLine && s.replicaAccepter != nil && line == "SYNC" {
			handedOff = true
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			s.replicaAccepter.AcceptReplica(conn)
			return
		}
		isFirstLine = false

		response := "ERR server is not configured to process commands"
		if s.processor != nil {
			response = s.processor.Process(line)
		}
		if _, err := conn.Write([]byte(response + "\n")); err != nil {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		s.logger.Printf("client read error: %v", err)
	}
}
