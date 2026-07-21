package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const defaultChannelBuffer = 100

type Subscription interface {
	Channel() <-chan string
	Unsubscribe(ctx context.Context) error
}

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

type MessageBus interface {
	Publish(ctx context.Context, channel string, payload interface{}) error
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}
