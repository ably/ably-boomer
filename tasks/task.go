package tasks

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-boomer/perf"
	"github.com/ably/ably-go/ably/proto"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RunWithBoomer(log log15.Logger, taskFn func(), c *cli.Context) error {
	taskName := c.Command.Name

	task := &boomer.Task{
		Name: taskName,
		Fn:   taskFn,
	}

	log.Info("starting perf")
	perf := perf.New(perf.Conf{
		CPUProfileDir: c.Path(config.CPUProfileDirFlag.Name),
		S3Bucket:      c.String(config.S3BucketFlag.Name),
	})
	if err := perf.Start(); err != nil {
		log.Crit("error starting perf", "err", err)
		os.Exit(1)
	}
	defer perf.Stop()

	log.Info("running ably-boomer", "task name", taskName)
	log.Info("running boomer-internal", "task name", taskName)
	os.Args = []string{"boomer",
		"-master-version-0.9.0",
		"-master-host", c.String(config.LocustHostFlag.Name),
		"-master-port", strconv.Itoa(c.Int(config.LocustPortFlag.Name))}
	boomer.Run(task)

	return nil
}

// Publisher represents something that publishes data to a channel or stream.
type Publisher interface {
	// Publish publishes data to a channel or stream that could be subscribed to.
	Publish(ctx context.Context, message *proto.Message) error
}

// PublisherFactory creates Publishers
type PublisherFactory interface {
	// NewPublisher creates a Publisher to a channel
	NewPublisher(ctx context.Context, channelName string) (Publisher, error)
}

// Subscriber represents something that subscribes to messages on a channel or stream.
type Subscriber interface {
	// Subscribe to messages. Messages are presented to the given handler. The call is expected
	// to be blocking and should be made with a cancelable context to end the call. Any errors on
	// the underlying implementation will be returned by the function and the subscription is assumed
	// to be terminated.
	Subscribe(ctx context.Context, msgHandler func(message *proto.Message)) error
}

// SubscriberFactory creates Subscribers
type SubscriberFactory interface {
	// NewSubscriber creates a Subscriber to a channel or stream.
	NewSubscriber(ctx context.Context, channelName string) (Subscriber, error)
}

// Task is a performance job that runs a collection of generic publishers and subscribers.
type Task struct {
	conf        config.Conf
	userCounter atomic.Int64
	taskID      int
	subF        SubscriberFactory
	pubF        PublisherFactory

	log log15.Logger
}

// NewTask returns a new task.
func NewTask(log log15.Logger, conf config.Conf, subF SubscriberFactory, pubf PublisherFactory) *Task {
	return &Task{
		conf:   conf,
		taskID: rand.Int(),
		subF:   subF,
		pubF:   pubf,
		log:    log,
	}
}

// Run starts the task.
func (t *Task) Run() {
	log := t.log
	log.Info(
		"starting task",
		"num-channels", t.conf.NumChannels,
		"subs-per-stream", t.conf.NumSubscriptions,
		"publish-interval", t.conf.PublishInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boomer.Events.Subscribe("boomer:stop", cancel)

	errGroup, ctx := errgroup.WithContext(ctx)

	t.userCounter.Add(1)

	if err := t.shardedLoad(ctx, errGroup); err != nil {
		log.Error("starting sharded load", "err", err)
		return
	}

	if err := t.personalLoad(ctx, errGroup); err != nil {
		log.Error("starting personal load", "err", err)
		return
	}

	if err := errGroup.Wait(); err != nil {
		log.Error("terminal failure, shutting down", "err", err)
		return
	}

	log.Info("task complete")
}

func (t *Task) shardedLoad(ctx context.Context, errGroup *errgroup.Group) error {
	shardedChannelName := t.generateStreamName(int64(t.conf.NumChannels), t.userCounter.Load())

	t.log.Info("creating sharded subscriber", "name", shardedChannelName)
	sub, err := t.subF.NewSubscriber(ctx, shardedChannelName)
	if err != nil {
		t.log.Error("creating sharded subscriber", "name", shardedChannelName, "err", err)
		boomer.RecordFailure("subscribe", "create sharded subscriber", 0, err.Error())
		return err
	}

	errGroup.Go(func() error {
		if err := sub.Subscribe(ctx, t.validateMsg); err != nil {
			t.log.Error("subscribing shard", "name", shardedChannelName, "err", err)
			boomer.RecordFailure("subscribe", "sharded subscriber", 0, err.Error())
			return err
		}
		return nil
	})
	return nil
}

func (t *Task) personalLoad(ctx context.Context, errGroup *errgroup.Group) error {
	personalChannelName := RandomString(100)
	t.log.Info("creating personal subscribers", "channel", personalChannelName, "count", t.conf.NumSubscriptions)
	for i := 0; i < t.conf.NumSubscriptions; i++ {
		i := i
		sub, err := t.subF.NewSubscriber(ctx, personalChannelName)
		if err != nil {
			t.log.Error("creating personal subscriber", "name", personalChannelName, "err", err)
			boomer.RecordFailure("subscribe", "create personal subscriber", 0, err.Error())
			return err
		}
		errGroup.Go(func() error {
			if err := sub.Subscribe(ctx, t.validateMsg); err != nil {
				t.log.Error("personal subscriber", "name", personalChannelName, "index", i, "err", err)
				boomer.RecordFailure("subscribe", "personal subscriber", 0, err.Error())
				return err
			}
			return nil
		})
	}

	t.log.Info("creating publisher")
	errGroup.Go(func() error {
		return t.publishLoop(ctx, personalChannelName, t.conf.PublishInterval, t.conf.MsgDataLength)
	})
	return nil
}

func (t *Task) validateMsg(msg *proto.Message) {
	timePublished, err := strconv.ParseInt(msg.Name, 10, 64)
	if err != nil {
		t.log.Error("error parsing message name as timestamp", "err", err)
		boomer.RecordFailure("subscribe", "parsing", 0, err.Error())
		return
	}

	timeElapsed := MillisecondTimestamp() - timePublished
	bytes := len(fmt.Sprint(msg.Data))

	t.log.Info("received message", "size", bytes, "latency", timeElapsed)
	boomer.RecordSuccess("subscribe", "message", timeElapsed, int64(bytes))
}

func (t *Task) publishLoop(ctx context.Context, channelName string, interval, msgDataLength int) error {
	log := t.log.New("channel", channelName)
	log.Info("creating publisher", "period", interval)
	p, err := t.pubF.NewPublisher(ctx, channelName)
	if err != nil {
		log.Error("creating publisher", "err", err)
		boomer.RecordFailure("publish", "create publisher", 0, err.Error())
		return err
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msg := &proto.Message{
				Name: strconv.FormatInt(MillisecondTimestamp(), 10),
				Data: RandomString(msgDataLength),
			}

			log.Info("publishing message", "size", msgDataLength)
			if err := p.Publish(ctx, msg); err != nil {
				log.Error("publishing message", "err", err)
				boomer.RecordFailure("publish", "publishing", 0, err.Error())
				return err
			}
			boomer.RecordSuccess("publish", "message", 0, 0)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) generateStreamName(numStreams, number int64) string {
	return fmt.Sprintf("task-%v-%v", t.taskID, number%numStreams)
}

// MillisecondTimestamp converts the current time in milliseconds.
func MillisecondTimestamp() int64 {
	nanos := time.Now().UnixNano()
	return nanos / int64(time.Millisecond)
}

// RandomString creates a random string of given length.
func RandomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
