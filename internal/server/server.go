package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"github.com/stevenstank/bolt/internal/command"
	"github.com/stevenstank/bolt/internal/pubsub"
)

// Processor handles one client command line.
type Processor interface {
	Process(line string) string
	Clone() interface{}
}

// Config contains TCP server configuration.
type Config struct {
	Addr            string
	Logger          *log.Logger
	Processor       Processor
	ReplicaAccepter ReplicaAccepter
	ReplicationInfo ReplicationInfo
	PubsubHub       *pubsub.Hub
	NodeID          string
	Role            string
}

// ReplicaAccepter receives replica handshake connections.
type ReplicaAccepter interface {
	AcceptReplica(conn net.Conn)
}

type ReplicationInfo interface {
	ReplicationStatus() string
	ConnectedReplicas() int
}

// Server owns Bolt's TCP listener and client connections.
type Server struct {
	addr      string
	logger    *log.Logger
	processor Processor
	replicaAccepter ReplicaAccepter
	replicationInfo ReplicationInfo
	pubsubHub *pubsub.Hub
	nodeID    string
	role      string
	startTime time.Time

	mu          sync.Mutex
	listener    net.Listener
	clients     map[net.Conn]*clientConn
	done        chan struct{}
	wg          sync.WaitGroup
	closed      bool
}

type clientConn struct {
	conn       net.Conn
	subscriber *pubsub.Subscriber
	processor  Processor
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

	nodeID := config.NodeID
	if nodeID == "" {
		nodeID = generateNodeID()
	}

	role := config.Role
	if role == "" {
		role = "primary"
		if config.ReplicaAccepter != nil {
			role = "primary"
		}
	}

	return &Server{
		addr:            addr,
		logger:          logger,
		processor:       config.Processor,
		replicaAccepter: config.ReplicaAccepter,
		replicationInfo: config.ReplicationInfo,
		pubsubHub:       config.PubsubHub,
		nodeID:          nodeID,
		role:            role,
		startTime:       time.Now(),
		clients:         make(map[net.Conn]*clientConn),
		done:            make(chan struct{}),
	}
}

// SetProcessorInfo sets the info provider on the processor if it supports it.
func (s *Server) SetProcessorInfo() {
	if s.processor == nil {
		return
	}
	// Type assertion to check if processor has SetInfo method
	type setInfo interface {
		SetInfo(command.InfoProvider)
	}
	if p, ok := s.processor.(setInfo); ok {
		p.SetInfo(s)
	} else {
		fmt.Printf("SetInfo failed: processor type = %T\n", s.processor)
	}
}

func generateNodeID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
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
	clients := make([]*clientConn, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.Unlock()

	if listener != nil {
		listener.Close()
	}
	for _, client := range clients {
		if client.subscriber != nil {
			client.subscriber.Close()
			if s.pubsubHub != nil {
				s.pubsubHub.UnsubscribeAll(client.subscriber)
			}
		}
		client.conn.Close()
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

		var subscriber *pubsub.Subscriber
		if s.pubsubHub != nil {
			subscriber = pubsub.NewSubscriber(100)
		}

		// Create a new processor with isolated transaction state for this client
		var processor Processor
		if s.processor != nil {
			// Clone the processor to get per-client transaction isolation
			processor = s.processor.Clone().(Processor)
		}

		s.clients[conn] = &clientConn{
			conn:       conn,
			subscriber: subscriber,
			processor:  processor,
		}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn, subscriber, processor)
	}
}

func (s *Server) handleConnection(conn net.Conn, subscriber *pubsub.Subscriber, processor Processor) {
	defer s.wg.Done()
	handedOff := false
	defer func() {
		if handedOff {
			return
		}
		conn.Close()

		if subscriber != nil {
			subscriber.Close()
			if s.pubsubHub != nil {
				s.pubsubHub.UnsubscribeAll(subscriber)
			}
		}

		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()

		s.logger.Printf("client disconnected: %s", conn.RemoteAddr().String())
	}()

	s.logger.Printf("client connected: %s", conn.RemoteAddr().String())
	
	// Start pubsub message delivery if enabled
	if subscriber != nil {
		s.wg.Add(1)
		go s.deliverPubsubMessages(conn, subscriber)
	}

	scanner := bufio.NewScanner(conn)
	isFirstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if isFirstLine && s.replicaAccepter != nil && line == "SYNC" {
	// Clean up the pub/sub resources because this connection
	// is being handed over to the replication subsystem.
	if subscriber != nil {
		subscriber.Close()
		if s.pubsubHub != nil {
			s.pubsubHub.UnsubscribeAll(subscriber)
		}
	}

	handedOff = true

	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()

	s.replicaAccepter.AcceptReplica(conn)
	return
}
		isFirstLine = false

		response := s.processCommand(line, subscriber, processor)
		if _, err := conn.Write([]byte(response + "\n")); err != nil {
			return
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		s.logger.Printf("client read error: %v", err)
	}
}

func (s *Server) processCommand(line string, subscriber *pubsub.Subscriber, processor Processor) string {
	// Handle pubsub commands directly
	if s.pubsubHub != nil && subscriber != nil {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			cmd := strings.ToUpper(fields[0])
			switch cmd {
			case "SUBSCRIBE":
				if len(fields) != 2 {
					return "ERR SUBSCRIBE requires channel"
				}
				s.pubsubHub.Subscribe(subscriber, fields[1])
				return fmt.Sprintf("OK subscribed to %s", fields[1])
			case "UNSUBSCRIBE":
				if len(fields) != 2 {
					return "ERR UNSUBSCRIBE requires channel"
				}
				s.pubsubHub.Unsubscribe(subscriber, fields[1])
				return fmt.Sprintf("OK unsubscribed from %s", fields[1])
			case "PUBLISH":
				if len(fields) < 3 {
					return "ERR PUBLISH requires channel and message"
				}
				channel := fields[1]
				message := strings.Join(fields[2:], " ")
				count := s.pubsubHub.Publish(channel, message)
				return fmt.Sprintf("%d", count)
			}
		}
	}

	// Handle regular commands through processor
	if processor != nil {
		return processor.Process(line)
	}
	return "ERR server is not configured to process commands"
}

func (s *Server) deliverPubsubMessages(conn net.Conn, subscriber *pubsub.Subscriber) {
	defer s.wg.Done()

	for msg := range subscriber.Messages() {
		line := fmt.Sprintf("MESSAGE %s %s", msg.Channel, msg.Payload)
		fmt.Printf("sending to %s: %s\n", conn.RemoteAddr(), line)
		conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		if _, err := conn.Write([]byte(line + "\n")); err != nil {
			return
		}
		conn.SetWriteDeadline(time.Time{})
	}
}

// NodeID returns the server's unique node identifier.
func (s *Server) NodeID() string {
	return s.nodeID
}

// Role returns the server's role (primary or replica).
func (s *Server) Role() string {
	return s.role
}

// ConnectedClients returns the number of connected clients.
func (s *Server) ConnectedClients() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

// Uptime returns the server's uptime in seconds.
func (s *Server) Uptime() int64 {
	return int64(time.Since(s.startTime).Seconds())
}

func (s *Server) ReplicationStatus() string {
	if s.replicationInfo == nil {
		return "disabled"
	}
	return s.replicationInfo.ReplicationStatus()
}

// ConnectedReplicas returns the number of connected replicas.
func (s *Server) ConnectedReplicas() int {
	if s.replicationInfo == nil {
		return 0
	}
	return s.replicationInfo.ConnectedReplicas()
}
