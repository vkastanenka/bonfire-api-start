package cache

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

type Subscription interface {
	Channel() <-chan Event
	Unsubscribe(ctx context.Context) error
}

type CacheSub struct {
	pubsub *redis.PubSub
	ch     chan Event
	done   chan struct{}
	once   sync.Once
}

func (s *CacheSub) Channel() <-chan Event {
	return s.ch
}

func (s *CacheSub) Unsubscribe(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		close(s.done)
		err = s.pubsub.Close()
	})

	if err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}

func (s *Store) Publish(ctx context.Context, channel string, message interface{}) error {
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
			return NewError(err, ScopeEvents)
		}
	}

	if err := s.client.Publish(ctx, channel, payload).Err(); err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}

func (s *Store) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	pb := s.client.Subscribe(ctx, channel)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		_ = pb.Close()
		return nil, NewError(err, ScopeEvents)
	}

	sub := &CacheSub{
		pubsub: pb,
		ch:     make(chan Event, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}

func (s *Store) PSubscribe(ctx context.Context, patterns ...string) (Subscription, error) {
	pb := s.client.PSubscribe(ctx, patterns...)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		_ = pb.Close()
		return nil, NewError(err, ScopeEvents)
	}

	sub := &CacheSub{
		pubsub: pb,
		ch:     make(chan Event, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}

func (s *CacheSub) listen() {
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
