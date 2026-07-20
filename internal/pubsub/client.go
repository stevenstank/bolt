package pubsub

// Client wraps a Hub and Subscriber to implement the pubsubClient interface.
type Client struct {
	hub        *Hub
	subscriber *Subscriber
}

// NewClient creates a new pubsub client.
func NewClient(hub *Hub, subscriber *Subscriber) *Client {
	return &Client{
		hub:        hub,
		subscriber: subscriber,
	}
}

// Subscribe adds the subscriber to a channel.
func (c *Client) Subscribe(channel string) error {
	c.hub.Subscribe(c.subscriber, channel)
	return nil
}

// Unsubscribe removes the subscriber from a channel.
func (c *Client) Unsubscribe(channel string) error {
	c.hub.Unsubscribe(c.subscriber, channel)
	return nil
}

// Publish sends a message to all subscribers of a channel.
func (c *Client) Publish(channel, message string) (int, error) {
	count := c.hub.Publish(channel, message)
	return count, nil
}

// Subscriber returns the underlying subscriber.
func (c *Client) Subscriber() *Subscriber {
	return c.subscriber
}
