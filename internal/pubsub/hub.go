package pubsub

import (
	"sync"
	"fmt"
)

// Hub manages pub/sub channels and subscribers.
type Hub struct {
	mu         sync.RWMutex
	channels   map[string]*channel
	subscribers map[*Subscriber]struct{}
}

type channel struct {
	mu          sync.RWMutex
	subscribers map[*Subscriber]struct{}
}

// Subscriber represents a client that can receive messages.
type Subscriber struct {
	mu       sync.Mutex
	channels map[string]struct{}
	msgChan  chan Message
	closed   bool
}

// Message is a published message.
type Message struct {
	Channel string
	Payload string
}

// NewHub creates a new pub/sub hub.
func NewHub() *Hub {
	return &Hub{
		channels:    make(map[string]*channel),
		subscribers: make(map[*Subscriber]struct{}),
	}
}

// Subscribe adds a subscriber to a channel.
func (h *Hub) Subscribe(sub *Subscriber, channelName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		ch = &channel{
			subscribers: make(map[*Subscriber]struct{}),
		}
		h.channels[channelName] = ch
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.subscribers[sub] = struct{}{}

	sub.mu.Lock()
	defer sub.mu.Unlock()
	if sub.channels == nil {
		sub.channels = make(map[string]struct{})
	}
	sub.channels[channelName] = struct{}{}

	h.subscribers[sub] = struct{}{}
}

// Unsubscribe removes a subscriber from a channel.
func (h *Hub) Unsubscribe(sub *Subscriber, channelName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return
	}

	ch.mu.Lock()
	delete(ch.subscribers, sub)
	if len(ch.subscribers) == 0 {
		delete(h.channels, channelName)
	}
	ch.mu.Unlock()

	sub.mu.Lock()
	delete(sub.channels, channelName)
	if len(sub.channels) == 0 {
		delete(h.subscribers, sub)
	}
	sub.mu.Unlock()
}

// UnsubscribeAll removes a subscriber from all channels.
func (h *Hub) UnsubscribeAll(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub.mu.Lock()
	channels := make([]string, 0, len(sub.channels))
	for ch := range sub.channels {
		channels = append(channels, ch)
	}
	sub.channels = make(map[string]struct{})
	sub.mu.Unlock()

	for _, channelName := range channels {
		ch, ok := h.channels[channelName]
		if !ok {
			continue
		}
		ch.mu.Lock()
		delete(ch.subscribers, sub)
		if len(ch.subscribers) == 0 {
			delete(h.channels, channelName)
		}
		ch.mu.Unlock()
	}

	delete(h.subscribers, sub)
}

// Publish sends a message to all subscribers of a channel.
func (h *Hub) Publish(channelName, message string) int {
	h.mu.RLock()
	ch, ok := h.channels[channelName]
	h.mu.RUnlock()

	if !ok {
		return 0
	}

	ch.mu.RLock()
	subs := make([]*Subscriber, 0, len(ch.subscribers))
	for sub := range ch.subscribers {
		subs = append(subs, sub)
	}
	ch.mu.RUnlock()

	count := 0
	for _, sub := range subs {
		if h.sendToSubscriber(sub, channelName, message) {
			count++
		}
	}
	fmt.Println("published to", count, "subscribers")
	return count
}

func (h *Hub) sendToSubscriber(sub *Subscriber, channelName, message string) bool {
	sub.mu.Lock()
	defer sub.mu.Unlock()

	if sub.closed {
		return false
	}

	select {
	case sub.msgChan <- Message{Channel: channelName, Payload: message}:
		return true
	default:
		// Channel full, skip this subscriber to avoid blocking
		return false
	}
}

// NewSubscriber creates a new subscriber.
func NewSubscriber(bufferSize int) *Subscriber {
	return &Subscriber{
		channels: make(map[string]struct{}),
		msgChan:  make(chan Message, bufferSize),
	}
}

// Messages returns the message channel for this subscriber.
func (s *Subscriber) Messages() <-chan Message {
	return s.msgChan
}

// Close closes the subscriber and cleans up resources.
func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true
	close(s.msgChan)
}

// IsClosed returns whether the subscriber is closed.
func (s *Subscriber) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
