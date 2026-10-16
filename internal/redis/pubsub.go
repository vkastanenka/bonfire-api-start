package redis

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/pkg/errs"

	goredis "github.com/redis/go-redis/v9"
)

// defaultChannelBuffer defines the capacity for the internal Subscription event channel.
const defaultChannelBuffer = 256

// Event represents a raw payload delivered from a Redis Pub/Sub channel.
type Message struct {
	Channel string
	Payload string
}

// Subscription manages the lifecycle and message streaming of an active Redis Pub/Sub connection.
type Subscription struct {
	pubsub *goredis.PubSub
	ch     chan Message
	done   chan struct{}
	once   sync.Once
}

// Channel returns a receive-only channel for reading incoming Redis events.
func (s *Subscription) Channel() <-chan Message {
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
		return errs.Internal("Failed to unsubscribe from Redis Pub/Sub channel.").
			Reason("PUBSUB_UNSUBSCRIBE_FAILED").
			Wrap(err)
	}
	return nil
}

type PubSub struct {
	client *goredis.Client
}

func NewPubSub(client *goredis.Client) *PubSub {
	return &PubSub{client: client}
}

// Publish transmits a payload to the specified Redis channel.
func (ps *PubSub) Publish(ctx context.Context, channel string, message any) error {
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
			return errs.InvalidArgument("Failed to marshal Pub/Sub message payload.").
				Reason("PUBSUB_MARSHAL_FAILED").
				FieldViolation("message", "payload must be JSON serializable", "INVALID_FORMAT").
				Wrap(err)
		}
	}

	if err := ps.client.Publish(ctx, channel, payload).Err(); err != nil {
		return errs.Unavailable("Failed to publish message to channel.").
			Reason("PUBSUB_PUBLISH_FAILED").
			Meta("channel", channel).
			Wrap(err)
	}
	return nil
}

// PublishBatch transmits multiple channel messages concurrently using a single Redis pipeline.
func (ps *PubSub) PublishBatch(ctx context.Context, messages map[string]any) error {
	if len(messages) == 0 {
		return nil
	}

	_, err := ps.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for channel, message := range messages {
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
					return err
				}
			}
			pipe.Publish(ctx, channel, payload)
		}
		return nil
	})

	if err != nil {
		return errs.Unavailable("Failed to batch publish messages to channels.").
			Reason("PUBSUB_BATCH_PUBLISH_FAILED").
			Wrap(err)
	}

	return nil
}

// Subscribe opens a subscription to one or more explicit Redis channels.
func (ps *PubSub) Subscribe(ctx context.Context, channels ...string) (*Subscription, error) {
	pb := ps.client.Subscribe(ctx, channels...)
	return newSubscription(ctx, pb)
}

func newSubscription(ctx context.Context, pb *goredis.PubSub) (*Subscription, error) {
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		_ = pb.Close()
		return nil, errs.Unavailable("Failed to establish Redis Pub/Sub subscription.").
			Reason("PUBSUB_SUBSCRIBE_FAILED").
			Wrap(err)
	}

	sub := &Subscription{
		pubsub: pb,
		ch:     make(chan Message, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}

func (s *Subscription) listen() {
	defer close(s.ch)

	for {
		select {
		case msg, ok := <-s.pubsub.Channel():
			if !ok {
				return
			}

			select {
			case <-s.done:
				return
			default:
			}

			pubMsg := Message{
				Channel: msg.Channel,
				Payload: msg.Payload,
			}

			select {
			case s.ch <- pubMsg:
			case <-s.done:
				return
			default:
				slog.Warn("pubsub channel buffer full, dropping message",
					slog.String("channel", msg.Channel),
				)
			}

		case <-s.done:
			return
		}
	}
}
