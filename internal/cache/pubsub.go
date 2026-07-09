package cache

import (
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

// redisSubscription acts as our structural bridge, adapting the concrete
// third-party goredis.PubSub stream to our clean, driver-agnostic interface.
type redisSubscription struct {
	pubsub *goredis.PubSub
	ch     chan string
	done   chan struct{}
}

func (s *redisSubscription) Channel() <-chan string {
	return s.ch
}

func (s *redisSubscription) Unsubscribe(ctx context.Context) error {
	close(s.done)
	return s.pubsub.Close()
}

// listen drains the native go-redis driver channel and pipes payloads
// forward safely, respecting manual resource teardowns.
func (s *redisSubscription) listen() {
	defer close(s.ch)
	redisCh := s.pubsub.Channel()

	for {
		select {
		case msg, ok := <-redisCh:
			if !ok {
				return
			}
			// Forward the message payload string to our exposed channel
			select {
			case s.ch <- msg.Payload:
			case <-s.done:
				return
			}
		case <-s.done:
			return
		}
	}
}

func (m *manager) Publish(ctx context.Context, channel string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return m.client.Publish(ctx, channel, bytes).Err()
}

func (m *manager) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	pb := m.client.Subscribe(ctx, channel)

	// Block until Redis acknowledges the subscription request.
	// This prevents a critical race condition where rapid subsequent publishes are lost.
	if _, err := pb.Receive(ctx); err != nil {
		pb.Close()
		return nil, NewError(err, DomainEvents)
	}

	sub := &redisSubscription{
		pubsub: pb,
		ch:     make(chan string),
		done:   make(chan struct{}),
	}

	// Offload stream translation to a background worker loop
	go sub.listen()

	return sub, nil
}
