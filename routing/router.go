package routing

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/pagrus7/kafka-fwd/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"strings"
)

type Router struct {
	kafkaCtx       *config.KafkaContext
	config         *config.RouteConfig
	chanMap        map[string]chan *kafka.Message // "topic[partition]" -> channel
	producerEvents chan kafka.Event
	logger         *zap.SugaredLogger
	pollingCtx     context.Context
	stopPolling    context.CancelFunc
}

type RouteOps struct {
	producer       *kafka.Producer
	consumer       *kafka.Consumer
	producerEvents chan kafka.Event
	src            kafka.TopicPartition
	logger         *zap.SugaredLogger
}

func (ro *RouteOps) forwardAsync(msg *kafka.Message, dest string) error {
	fwdMsg := kafka.Message{
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: msg.Headers,
		TopicPartition: kafka.TopicPartition{
			Topic:     &dest,
			Partition: kafka.PartitionAny,
		},
		Opaque: msg,
	}
	return ro.producer.Produce(&fwdMsg, ro.producerEvents)
}

func (ro *RouteOps) pausePartition() error {
	return ro.consumer.Pause([]kafka.TopicPartition{ro.src})
}

func (ro *RouteOps) resumePartition() error {
	return ro.consumer.Resume([]kafka.TopicPartition{ro.src})
}

var _ kafkaOps = (*RouteOps)(nil)

func ProvideRouter(kafkaCtx *config.KafkaContext, cfg *config.RouteConfig, logger *zap.SugaredLogger, lc fx.Lifecycle) *Router {

	router := Router{
		kafkaCtx:       kafkaCtx,
		config:         cfg,
		chanMap:        make(map[string]chan *kafka.Message),
		logger:         logger.Named("router"),
		producerEvents: make(chan kafka.Event),
	}

	lcHook := fx.Hook{
		OnStart: router.start,
		OnStop:  router.stop,
	}

	lc.Append(lcHook)

	return &router
}

func (r *Router) start(ctx context.Context) error {
	consumer := r.kafkaCtx.Consumer

	topics := make([]string, 0, len(r.config.RouteDefs))
	for _, def := range r.config.RouteDefs {
		topics = append(topics, def.From)
	}

	r.logger.Infof("Subscribing consumer %s to topics: %s", consumer, topics)
	if err := consumer.SubscribeTopics(topics, nil); err != nil {
		return err
	}

	r.logger.Info("Starting the polling loop")
	r.pollingCtx, r.stopPolling = context.WithCancel(context.Background())
	go r.handleConsumerEvents(r.pollingCtx)
	return nil
}

func (r *Router) stop(ctx context.Context) error {
	r.logger.Info("Stopping the polling loop")
	r.stopPolling()
	return nil
}

// handleConsumerEvents polls the kafka.Consumer until ctx is done
func (r *Router) handleConsumerEvents(ctx context.Context) {
	consumer := r.kafkaCtx.Consumer
	go r.handleProducerEvents(ctx, r.producerEvents)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Polling loop stopped")
			return
		default:
			evt := consumer.Poll(100)
			switch e := evt.(type) {
			case *kafka.Message:
				r.logger.Debugf("Message on %s. Key: %s; Val: %s; Headers: %s",
					e.TopicPartition, e.Key, string(e.Value), headersString(e.Headers))

				ch, ok := r.channelFor(e)
				if !ok {
					r.logger.Warnf("Unexpected: Could not identify the route for %s; skipping the message", e)
					continue
				}

				ch <- e
			case kafka.Error:
				if e.IsFatal() {
					// TBD restart not panic
					r.logger.Panicf("Fatal error, stopping: %s", e.String())
				}
				r.logger.Errorf("Non-fatal error (retriable=%t): %s", e.IsRetriable(), e.String())
			}
		}
	}
}

func (r *Router) handleProducerEvents(ctx context.Context, events chan kafka.Event) {
	logger := r.logger
	for {
		select {
		case <-ctx.Done():
			logger.Info("Producer event loop stopped")
			return

		case evt := <-events:
			switch e := evt.(type) {
			case *kafka.Message:
				logger.Debugf("Confirmed delivery of %v", e)
				if e.TopicPartition.Error != nil {
					// TBD restart not panic
					logger.Panicf("Produce failed for %v: %s", e, e.TopicPartition.Error)
				}

				if srcMsg, ok := e.Opaque.(*kafka.Message); ok {
					r.storeOffsetsFor(srcMsg)
				} else {
					logger.Warnf("Unexpected: Produced message %s is not linked to the source message. Consumer offset won't be changed.", e)
				}
			}
		}
	}
}

func (r *Router) storeOffsetsFor(msg *kafka.Message) {
	r.logger.Debugf("Storing offsets for %s", msg.TopicPartition)
	if _, err := r.kafkaCtx.Consumer.StoreMessage(msg); err != nil {
		r.logger.Warnf("Error storing offsets for %s: %s", msg.TopicPartition, err)
		// No further action. Next message for same partition will trigger next attempt.
	}
}

func (r *Router) channelFor(e *kafka.Message) (chan *kafka.Message, bool) {
	topic := *e.TopicPartition.Topic
	part := e.TopicPartition.Partition

	key := fmt.Sprintf("%s[%d]", topic, part)
	ch, ok := r.chanMap[key]
	if ok {
		return ch, true
	}

	def, found := r.config.RouteDefs[topic]
	if !found {
		return nil, false
	}

	r.logger.Debugf("Starting route for %s", key)
	ops := RouteOps{
		consumer:       r.kafkaCtx.Consumer,
		producer:       r.kafkaCtx.Producer,
		producerEvents: r.producerEvents,
		src: kafka.TopicPartition{
			Topic:     &topic,
			Partition: part,
		},
	}
	route := NewRoute(def, part, &ops, r.logger.Named(key))
	ch = make(chan *kafka.Message)
	go route.Subscribe(ch)
	r.chanMap[key] = ch
	return ch, true
}

func headersString(headers []kafka.Header) string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for _, h := range headers {
		sb.WriteString(h.String())
	}
	sb.WriteString("}")
	return sb.String()
}
