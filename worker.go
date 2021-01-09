package ablyboomer

import (
	"context"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/go-redis/redis/v8"
	"github.com/inconshreveable/log15"
	"go.uber.org/atomic"
)

// WorkerOption configures a Worker.
type WorkerOption func(*Worker)

// WithLog configures the log15 Logger used by a Worker.
func WithLog(log log15.Logger) WorkerOption {
	return func(w *Worker) {
		w.log = log
	}
}

// WithSetConfigFunc configures an optional function to set the config before
// each load test is started.
func WithSetConfigFunc(f func() *config.Config) WorkerOption {
	return func(w *Worker) {
		w.setConfigFunc = f
	}
}

// Worker is a Locust worker that receives load test events from the Locust
// master via the boomer library.
//
// The worker has a current load test which is started when the "boomer:spawn"
// event is received and stopped when a subsequent "boomer:stop" event is
// received.
//
// The worker optionally uses Redis to assign itself an incremental number
// using the INCR operation, and it uses this number to also assign incremental
// numbers to the users it starts that are distinct from numbers assigned by
// other workers. For example, if a worker is assigned the number 5 in Redis,
// when it starts a 10 user load test, it will number those users from 41 to 50.
type Worker struct {
	conf   *config.Config
	boomer *boomer.Boomer

	setConfigFunc func() *config.Config

	mtx     sync.RWMutex
	current *loadTest

	redis  *redis.Client
	number int64

	log log15.Logger
}

// NewWorker returns a Worker that uses the given config and options.
func NewWorker(conf *config.Config, opts ...WorkerOption) (*Worker, error) {
	// initialise the worker
	w := &Worker{
		conf:   conf,
		number: 1,
	}

	// apply the options
	for _, opt := range opts {
		opt(w)
	}

	// set the logger
	if w.log == nil {
		lvl, err := log15.LvlFromString(conf.Log.Level)
		if err != nil {
			return nil, fmt.Errorf("invalid log.level: %v", err)
		}
		w.log = log15.New()
		w.log.SetHandler(log15.LvlFilterHandler(lvl, w.log.GetHandler()))
	}

	// initialise boomer in either standalone or distributed mode based on
	// the config
	if conf.Standalone.Enabled {
		w.log.Info("running ablyboomer in standalone mode")
		w.boomer = boomer.NewStandaloneBoomer(conf.Standalone.Users, conf.Standalone.SpawnRate)
	} else {
		w.log.Info("running ablyboomer in distributed mode", "locust.host", conf.Locust.Host)
		w.boomer = boomer.NewBoomer(conf.Locust.Host, conf.Locust.Port)
	}

	// initialise Redis if enabled
	if conf.Redis.Enabled {
		if err := w.connectRedis(); err != nil {
			return nil, err
		}
	}

	return w, nil
}

// Conf returns the Worker's current config
func (w *Worker) Conf() *config.Config {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	return w.conf
}

// connectRedis connects to Redis, retrying for up to conf.Redis.ConnectTimeout.
func (w *Worker) connectRedis() error {
	start := time.Now()
	timeout := time.After(w.conf.Redis.ConnectTimeout)
	for {
		client := redis.NewClient(&redis.Options{
			Addr: w.conf.Redis.Addr,
		})
		ctx, cancel := context.WithTimeout(context.Background(), w.conf.Redis.ConnectTimeout-time.Since(start))
		defer cancel()
		if _, err := client.Ping(ctx).Result(); err == nil {
			w.redis = client
			return nil
		}
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for Redis at %v", w.conf.Redis.Addr)
		case <-time.After(time.Second):
			continue
		}
	}
}

// Run implements the main worker loop which waits for boomer events and
// starts and stops users.
//
// Run blocks until the underlying boomer is stopped.
func (w *Worker) Run(ctx context.Context) {
	// assign a worker number in Redis if it's configured
	if w.redis != nil {
		w.assignWorkerNumber(ctx)
	}

	// register handlers for the boomer spawn and stop events
	w.log.Debug("registering boomer listeners")
	boomer.Events.Subscribe("boomer:spawn", w.onBoomerSpawn)
	boomer.Events.Subscribe("boomer:stop", w.onBoomerStop)

	// start the boomer task runner loop with a handler that starts users
	w.log.Debug("starting boomer task runner")
	go w.boomer.Run(&boomer.Task{
		Name: "ablyboomer",
		Fn:   w.onBoomerTask,
	})

	// wait for the boomer runner to quit
	w.log.Debug("waiting for boomer task runner to quit")
	boomerQuit := make(chan struct{})
	boomer.Events.SubscribeOnce("boomer:quit", func() {
		w.log.Debug("received boomer:quit event")
		close(boomerQuit)
	})
	select {
	case <-boomerQuit:
		w.log.Debug("processed boomer:quit event")
	case <-ctx.Done():
		w.log.Debug("qutting boomer task runner")
		w.boomer.Quit()
	}
}

// assignWorkerNumber assigns a number to the Worker in Redis by calling the
// atomic INCR command and using the returned number, just logging an error if
// it fails (no need to crash the whole load test).
func (w *Worker) assignWorkerNumber(ctx context.Context) {
	var err error
	w.number, err = w.redis.Incr(ctx, w.conf.Redis.WorkerNumberKey).Result()
	if err != nil {
		w.log.Error("error assigning worker number in Redis", "err", err)
	} else {
		w.log.Info("assigned worker number in Redis", "number", w.number)
	}
}

// onBoomerSpawn handles the "boomer:spawn" event by initialising a new load
// test for the given number of users and setting it as the current load test.
//
// The load test is initialised with a user counter offsetted by the Worker's
// own assigned number from Redis so that it uses a block of consecutive user
// which is distinct from that of other Workers participating in the load test.
//
// For example, if the desired user count is 10, then we want Workers to
// assign:
//
// worker 1 - users  1 to 10
// worker 2 - users 11 to 20
// worker 3 - users 21 to 30
// etc.
//
func (w *Worker) onBoomerSpawn(userCount int, spawnRate float64) {
	w.log.Debug("received boomer:spawn event")

	// we can't return errors from this function as they won't go anywhere,
	// so instead we log them, report them to Locust and wait for another
	// load test to be started
	reportErr := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		w.log.Error(msg)
		w.boomer.RecordFailure("ablyboomer", "spawn", 0, msg)
	}

	// hold the mutex whilst the config is validated and the current load test is set
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// only proceed if there isn't a load test in progress
	if w.current != nil {
		reportErr("refusing to accept new %d user load test when current is still running", userCount)
		return
	}

	// initialise the new load test with an appropriate userCounter
	userNumberStart := (w.number - 1) * int64(userCount)
	l := &loadTest{
		w:           w,
		userCounter: atomic.NewInt64(userNumberStart),
		stopC:       make(chan struct{}),
		log:         w.log,
	}

	// set the config if configured to do so
	if w.setConfigFunc != nil {
		w.log.Debug("calling configured setConfigFunc")
		w.conf = w.setConfigFunc()
	}

	// ensure at least one task is enabled
	if !w.conf.Subscriber.Enabled && !w.conf.Publisher.Enabled && !w.conf.Presence.Enabled {
		reportErr("at least one of subscriber, publisher or presence must be enabled")
		return
	}

	// parse the channel templates
	if w.conf.Subscriber.Enabled {
		channels := w.conf.Subscriber.Channels
		tmpl, err := template.New("channel").Funcs(channelFuncs).Parse(channels)
		if err != nil {
			reportErr("error parsing subscriber channels %q: %w", channels, err)
			return
		}
		l.subscriberChannels = tmpl
	}
	if w.conf.Publisher.Enabled {
		channels := w.conf.Publisher.Channels
		tmpl, err := template.New("channel").Funcs(channelFuncs).Parse(channels)
		if err != nil {
			reportErr("error parsing publisher channels %q: %w", channels, err)
			return
		}
		l.publisherChannels = tmpl
	}
	if w.conf.Presence.Enabled {
		channels := w.conf.Presence.Channels
		tmpl, err := template.New("channel").Funcs(channelFuncs).Parse(channels)
		if err != nil {
			reportErr("error parsing presence channels %q: %w", channels, err)
			return
		}
		l.presenceChannels = tmpl
	}

	// ensure the configured client exists
	newClientFunc, ok := GetNewClientFunc(w.conf.Client)
	if !ok {
		reportErr("client not found: %q", w.conf.Client)
		return
	}
	l.newClientFunc = newClientFunc

	w.log.Info("setting current load test", "userCount", userCount, "userNumberStart", userNumberStart, "spawnRate", spawnRate)
	w.current = l
}

// onBoomerStop handles the "boomer:stop" event by stopping and removing the
// current load test.
func (w *Worker) onBoomerStop() {
	w.log.Debug("received boomer:stop event")
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.log.Info("stopping current load test")
	w.current.stop()
	w.current = nil
}

// onBoomerTask handles a task invocation by the boomer task runner by running
// a single user for the current load test.
func (w *Worker) onBoomerTask() {
	w.log.Debug("received boomer task invocation")
	w.mtx.RLock()
	current := w.current
	w.mtx.RUnlock()
	if current == nil {
		// handle a race in the boomer library between the stop event
		// being emitted and the task runner continuing to invoke tasks
		// by just ignoring the invocation if we removed the current
		// load test under the safety of w.mtx.
		w.log.Debug("ignoring boomer task as there is no current load test")
		return
	}
	current.runUser()
}
