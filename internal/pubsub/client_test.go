package pubsub

import (
	"testing"
	"time"
)

func TestClientSubscribe(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)
	client := NewClient(hub, sub)

	err := client.Subscribe("news")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := hub.Publish("news", "hello")
	if count != 1 {
		t.Fatalf("expected 1 subscriber, got %d", count)
	}

	select {
	case msg := <-sub.Messages():
		if msg.Channel != "news" || msg.Payload != "hello" {
			t.Fatalf("unexpected message: %+v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestClientUnsubscribe(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)
	client := NewClient(hub, sub)

	client.Subscribe("news")
	client.Unsubscribe("news")

	count := hub.Publish("news", "hello")
	if count != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", count)
	}
}

func TestClientPublish(t *testing.T) {
	hub := NewHub()
	sub1 := NewSubscriber(10)
	sub2 := NewSubscriber(10)
	client1 := NewClient(hub, sub1)
	client2 := NewClient(hub, sub2)

	client1.Subscribe("news")
	client2.Subscribe("news")

	count, err := client1.Publish("news", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 subscribers, got %d", count)
	}
}

func TestClientSubscriber(t *testing.T) {
	hub := NewHub()
	sub := NewSubscriber(10)
	client := NewClient(hub, sub)

	if client.Subscriber() != sub {
		t.Fatal("subscriber mismatch")
	}
}
