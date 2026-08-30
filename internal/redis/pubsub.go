package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// defaultChannelBuffer defines the capacity for the internal Subscription event channel.
const defaultChannelBuffer = 256

// Event represents a raw payload delivered from a Redis Pub/Sub channel.
type Event struct {
	Channel string
	Payload string
}

// Subscription manages the lifecycle and message streaming of an active Redis Pub/Sub connection.
type Subscription struct {
	pubsub *redis.PubSub
	ch     chan Event
	done   chan struct{}
	once   sync.Once
}

// Channel returns a receive-only channel for reading incoming Redis events.
func (s *Subscription) Channel() <-chan Event {
	return s.ch
}

// Unsubscribe gracefully stops listening for events and closes the underlying Redis Pub/Sub connection.
func (s *Subscription) Unsubscribe() error {
	var err error
	s.once.Do(func() {
		close(s.done)
		err = s.pubsub.Close()
	})

	if err != nil {
		return NewError(err, ScopeOutboxEvent)
	}
	return nil
}

// Close implements io.Closer by delegating to Unsubscribe.
func (s *Subscription) Close() error {
	return s.Unsubscribe()
}

// --- Pub/Sub Operations ---

// Publish transmits a payload to the specified Redis channel.
func Publish(ctx context.Context, client redis.Cmdable, channel string, message any) error {
	var payload []byte
	var err error

	switch v := message.(type) {
	case string:
		payload = []byte(v)
	case []byte:
		payload = v
	default:
		payload, err = json.Marshal(v)
		if err != nil {
			return NewError(err, ScopeOutboxEvent)
		}
	}

	if err := client.Publish(ctx, channel, payload).Err(); err != nil {
		return NewError(err, ScopeOutboxEvent)
	}
	return nil
}

// Subscribe opens a subscription to one or more explicit Redis channels.
func Subscribe(ctx context.Context, client *redis.Client, scope Scope, channels ...string) (*Subscription, error) {
	pb := client.Subscribe(ctx, channels...)
	return newSubscription(ctx, pb, scope)
}

// PSubscribe opens a pattern-based subscription matching one or more Redis channel patterns.
func PSubscribe(ctx context.Context, client *redis.Client, scope Scope, patterns ...string) (*Subscription, error) {
	pb := client.PSubscribe(ctx, patterns...)
	return newSubscription(ctx, pb, scope)
}

// --- Internal Helpers ---

func newSubscription(ctx context.Context, pb *redis.PubSub, scope Scope) (*Subscription, error) {
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		return nil, errors.Join(NewError(err, scope), pb.Close())
	}

	sub := &Subscription{
		pubsub: pb,
		ch:     make(chan Event, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}

func (s *Subscription) listen() {
	defer close(s.ch)
	redisCh := s.pubsub.Channel()

	for {
		select {
		case msg, ok := <-redisCh:
			if !ok {
				return
			}

			select {
			case s.ch <- Event{Channel: msg.Channel, Payload: msg.Payload}:
			case <-s.done:
				return
			}

		case <-s.done:
			return
		}
	}
}
