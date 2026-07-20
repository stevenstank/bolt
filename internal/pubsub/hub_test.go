package pubsub

import (
	"testing"
	"time"
)

func TestHubSubscribeAndPublish(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)

	hub.Subscribe(sub, "news")

	count := hub.Publish("news", "hello world")
	if count != 1 {
		t.Fatalf("expected 1 subscriber to receive message, got %d", count)
	}

	select {
	case msg := <-sub.Messages():
		if msg.Channel != "news" || msg.Payload != "hello world" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestHubMultipleSubscribers(t *testing.T) {
	hub := NewHub()
	sub1 := NewSubscriber(10)
	sub2 := NewSubscriber(10)
	sub3 := NewSubscriber(10)

	hub.Subscribe(sub1, "news")
	hub.Subscribe(sub2, "news")
	hub.Subscribe(sub3, "sports")

	count := hub.Publish("news", "breaking news")
	if count != 2 {
		t.Fatalf("expected 2 subscribers to receive message, got %d", count)
	}

	select {
	case msg := <-sub1.Messages():
		if msg.Payload != "breaking news" {
			t.Fatalf("unexpected message: %s", msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("sub1 did not receive message")
	}

	select {
	case msg := <-sub2.Messages():
		if msg.Payload != "breaking news" {
			t.Fatalf("unexpected message: %s", msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("sub2 did not receive message")
	}

	select {
	case <-sub3.Messages():
		t.Fatal("sub3 should not have received message")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

func TestHubUnsubscribe(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)

	hub.Subscribe(sub, "news")
	hub.Unsubscribe(sub, "news")

	count := hub.Publish("news", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", count)
	}
}

func TestHubUnsubscribeAll(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)

	hub.Subscribe(sub, "news")
	hub.Subscribe(sub, "sports")
	hub.UnsubscribeAll(sub)

	count := hub.Publish("news", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe all, got %d", count)
	}

	count = hub.Publish("sports", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe all, got %d", count)
	}
}

func TestHubMultipleChannels(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)

	hub.Subscribe(sub, "news")
	hub.Subscribe(sub, "sports")

	hub.Publish("news", "news update")
	hub.Publish("sports", "sports update")

	count := 0
	for count < 2 {
		select {
		case msg := <-sub.Messages():
			if msg.Channel != "news" && msg.Channel != "sports" {
				t.Fatalf("unexpected channel: %s", msg.Channel)
			}
			count++
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for messages")
		}
	}
}

func TestSubscriberClose(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)

	hub.Subscribe(sub, "news")
	sub.Close()

	count := hub.Publish("news", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers after close, got %d", count)
	}

	if !sub.IsClosed() {
		t.Fatal("expected subscriber to be closed")
	}
}

func TestPublishToNonExistentChannel(t *testing.T) {
	hub := NewHub()

	count := hub.Publish("nonexistent", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers for nonexistent channel, got %d", count)
	}
}

func TestPublishDoesNotBlock(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(1) // Small buffer

	hub.Subscribe(sub, "news")

	// Fill the buffer
	hub.Publish("news", "msg1")

	// This should not block even though buffer is full
	count := hub.Publish("news", "msg2")
	if count != 0 {
		t.Fatalf("expected 0 subscribers due to full buffer, got %d", count)
	}
}

func TestConcurrentOperations(t *testing.T) {
	hub := NewHub()
	sub1 := NewSubscriber(100)
	sub2 := NewSubscriber(100)

	hub.Subscribe(sub1, "news")
	hub.Subscribe(sub2, "news")

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			hub.Publish("news", "message")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			hub.Subscribe(sub1, "news")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			hub.Unsubscribe(sub1, "news")
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}
