package redis

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultChannelBuffer = 100

type Event struct {
	Channel string
	Payload string
}

type Subscription struct {
	pubsub *redis.PubSub
	ch     chan Event
	done   chan struct{}
	once   sync.Once
}

func (s *Subscription) Channel() <-chan Event {
	return s.ch
}

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

func (s *Subscription) Close() error {
	return s.Unsubscribe()
}

// Publish accepts any redis.Cmdable (Client, Pipeline, Tx)
func Publish(ctx context.Context, client redis.Cmdable, channel string, message interface{}) error {
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

// Subscribe requires *redis.Client directly because PubSub manages socket state
func Subscribe(ctx context.Context, client *redis.Client, scope Scope, channels ...string) (*Subscription, error) {
	pb := client.Subscribe(ctx, channels...)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		_ = pb.Close()
		return nil, NewError(err, scope)
	}

	sub := &Subscription{
		pubsub: pb,
		ch:     make(chan Event, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}

func PSubscribe(ctx context.Context, client *redis.Client, scope Scope, patterns ...string) (*Subscription, error) {
	pb := client.PSubscribe(ctx, patterns...)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		_ = pb.Close()
		return nil, NewError(err, scope)
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

			evt := Event{
				Channel: msg.Channel,
				Payload: msg.Payload,
			}

			select {
			case s.ch <- evt:
			case <-s.done:
				return
			}
		case <-s.done:
			return
		}
	}
}
