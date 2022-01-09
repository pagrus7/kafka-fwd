package routing

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/pagrus7/kafka-fwd/config"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.design/x/chann"
	"time"
)

type kafkaOps interface {
	forwardAsync(srcMsg *kafka.Message, dest string) error
	pausePartition() error
	resumePartition() error
}

type Route struct {
	def             *config.RouteDef
	partition       int32
	paused          atomic.Bool
	pauseThreshold  int
	resumeThreshold int
	kafkaOps        kafkaOps
	logger          *zap.SugaredLogger
}

var _ fmt.Stringer = (*Route)(nil)

func NewRoute(def *config.RouteDef, partition int32, kafkaOps kafkaOps, logger *zap.SugaredLogger) *Route {
	return &Route{
		def:             def,
		partition:       partition,
		pauseThreshold:  def.PauseThreshold,
		resumeThreshold: def.ResumeThreshold,
		kafkaOps:        kafkaOps,
		logger:          logger,
	}
}

func (r *Route) String() string {
	if r.def.Delay == 0 {
		return fmt.Sprintf("Route: %s[%d]->%s", r.def.From, r.partition, r.def.To)
	}
	return fmt.Sprintf("Route: %s[%d]->(+%s)->%s", r.def.From, r.partition, r.def.Delay, r.def.To)
}

// Subscribe is intended to run in goroutine. Assumes that messages in the channel are coming from the intended topic & partition.
// Subscription loop terminates once msgChan is closed.
func (r *Route) Subscribe(msgChan chan *kafka.Message) {
	// forward to unbounded channel, throttle by pausing partition if needed

	unboundedChan := chann.New[*kafka.Message]()
	ctx, cancelHandling := context.WithCancel(context.Background())
	go r.handleDelay(ctx, unboundedChan)

	for msg := range msgChan {
		// 1. republish into unbounded chan
		// 2. check unbounded chan length
		//   2.1 if >= pause threshold and partition not paused - pause it
		//   2.2 if <= resume threshold and partition paused - resume it

		len := unboundedChan.ApproxLen()
		if len >= r.pauseThreshold && !r.paused.Load() {
			r.logger.Debugf("Pausing partition - channel length %d", len)
			if err := r.kafkaOps.pausePartition(); err != nil {
				r.logger.Warnf("Could not pause partition: %s", err)
			} else {
				r.paused.Store(true)
			}
		}

		unboundedChan.In() <- msg
	}

	// the partition may end up paused. Not resuming, as the route may stop in 2 cases:
	// - Partition revoked from consumer group
	// - Routing is shutting down
	// either way, there is no more interest in consuming from the partition
	unboundedChan.Close()
	cancelHandling()
}

func (r *Route) handleDelay(ctx context.Context, msgChann *chann.Chann[*kafka.Message]) {
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping delay handler")
			return

		case msg := <-msgChann.Out():
			if !r.wait(ctx, msg) {
				return
			}

			r.processNow(ctx, msg)

			len := msgChann.ApproxLen()
			if !r.maybeResumePartition(ctx, len) {
				return
			}

		}
	}
}

// maybeResumePartition checks the channel length against resume threshold and resumes partition if paused. and continuously retries in case of error.
// Returns true if partition was not stopped or resumed successfully, false if context is cancelled.
func (r *Route) maybeResumePartition(ctx context.Context, channLen int) bool {
	for channLen <= r.resumeThreshold && r.paused.Load() {
		r.logger.Debugf("Resuming partition - channel length %d", channLen)
		if err := r.kafkaOps.resumePartition(); err != nil {
			r.logger.Warnf("Error resuming partition, will retry: %s", err)
		} else {
			r.paused.Store(false)
			r.logger.Debug("Resumed partition successfully")
			break
		}

		// need to allow cancellation b/w retries
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping the delay handler")
			return false
		default:
		}
	}
	return true
}

func (r *Route) processNow(ctx context.Context, msg *kafka.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			r.logger.Debugf("Forwarding message %v", msg.TopicPartition)
			if err := r.kafkaOps.forwardAsync(msg, r.def.To); err != nil {
				r.logger.Infof("Error producing message, will retry in 200 ms: %s", err.Error())
				time.Sleep(time.Duration(200 * time.Millisecond))
			} else {
				return
			}
		}
	}
}

// wait returns true if waited for the intended (potentially zero) duration, false if wait was interrupted by ctx.Done().
func (r *Route) wait(ctx context.Context, msg *kafka.Message) bool {
	now := time.Now()
	sendAt := msg.Timestamp.Add(r.def.Delay)
	if !sendAt.After(now) {
		// no delay needed
		return true
	}

	delay := sendAt.Sub(now)
	r.logger.Debugf("Sleeping for %s", delay)
	wakeup := time.After(delay)
	select {
	case <-ctx.Done():
		r.logger.Info("Stopping wait")
		return false
	case <-wakeup:
	}
	return true

}
