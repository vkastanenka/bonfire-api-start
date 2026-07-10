package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// TODO: Move to config
const defaultChannelBuffer = 100

type cacheSub struct {
	pubsub *redis.PubSub
	ch     chan string
	done   chan struct{}
}

func (s *cacheSub) Channel() <-chan string {
	return s.ch
}

func (s *cacheSub) Unsubscribe(ctx context.Context) error {
	close(s.done)
	if err := s.pubsub.Close(); err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}

func (s *cacheSub) listen() {
	defer close(s.ch)
	redisCh := s.pubsub.Channel()

	for {
		select {
		case msg, ok := <-redisCh:
			if !ok {
				return
			}
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

func (c *client) Publish(ctx context.Context, channel string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return NewError(err, ScopeEvents)
	}

	if err := c.redis.Publish(ctx, channel, bytes).Err(); err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}

func (c *client) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pb := c.redis.Subscribe(subCtx, channel)

	if _, err := pb.Receive(subCtx); err != nil {
		pb.Close()
		return nil, NewError(err, ScopeEvents)
	}

	sub := &cacheSub{
		pubsub: pb,
		ch:     make(chan string, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()

	return sub, nil
}
