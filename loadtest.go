package ablyboomer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"text/template"
	"time"

	"github.com/ably/ably-boomer/config"
	"github.com/ably/ably-go/ably"
	"github.com/inconshreveable/log15"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
)

// loadTest represents a running Locust load test and is used by the Worker to
// spawn its requested number of users.
type loadTest struct {
	w                  *Worker
	newClientFunc      NewClientFunc
	subscriberChannels *template.Template
	publisherChannels  *template.Template
	presenceChannels   *template.Template
	userCounter        *atomic.Int64
	users              sync.WaitGroup
	stopC              chan struct{}
	log                log15.Logger
}

// runUser runs a single Locust user that runs one or more tasks, and returns
// either if there is an error creating a client or when the load test is
// stopped.
//
// The tasks the user may run are:
//
// subscriber: subscribe to the channels specified in conf.Subscriber.Channels
//             if conf.Subscriber.Enabled is true (see loadTest.runSubscriber).
//
// publisher:  publish a message every conf.Publisher.PublishInterval to each
//             of the channels specified in conf.Publisher.Channels if
//             conf.Publisher.Enabled is true (see loadTest.runPublisher).
//
// presence:   enter the channels specified in conf.Presence.Channels if
//             conf.Presence.Enabled is true (see loadTest.runPresence).
//
func (l *loadTest) runUser() {
	// track each user so we can wait for them all to stop in
	// loadTest.stop()
	l.users.Add(1)
	defer l.users.Done()

	// assign a number for this user
	userNum := l.userCounter.Inc()
	l.log.Debug("starting user", "number", userNum)

	// initialise a context that cancels when the loadTest is stopped
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-l.stopC
		l.log.Debug("stopping user")
		cancel()
	}()

	// initialise a client, reporting any errors that occur
	l.log.Debug("initialising client")
	client, err := l.newClientFunc(ctx, l.w.Conf(), l.log)
	if err != nil {
		l.log.Debug("error initialising client", "err", err)
		l.w.boomer.RecordFailure("ablyboomer", "client", 0, err.Error())
		return
	}
	defer client.Close()

	// run the enabled tasks until the loadTest is stopped
	errG, ctx := errgroup.WithContext(ctx)
	if l.w.Conf().Subscriber.Enabled {
		errG.Go(func() error { return l.runSubscriber(ctx, client, userNum) })
	}
	if l.w.Conf().Publisher.Enabled {
		errG.Go(func() error { return l.runPublisher(ctx, client, userNum) })
	}
	if l.w.Conf().Presence.Enabled {
		errG.Go(func() error { return l.runPresence(ctx, client, userNum) })
	}
	errG.Wait()
	l.log.Debug("user stopped")
}

func registerPushDevice(ctx context.Context, config *config.Config, deviceID, outputChannel string, rest *ably.REST, log log15.Logger) error {
	type recipient struct {
		TransportType string `json:"transportType"`
		Channel       string `json:"channel"`
		AblyKey       string `json:"ablyKey"`
		AblyUrl       string `json:"ablyUrl"`
	}
	type push struct {
		Recipient recipient `json:"recipient"`
	}
	type input struct {
		Id         string `json:"id"`
		Platform   string `json:"platform"`
		FormFactor string `json:"formFactor"`
		Push       push   `json:"push"`
	}
	regInput := &input{
		Id:         deviceID,
		Platform:   "browser",
		FormFactor: "other",
		Push: push{
			Recipient: recipient{
				TransportType: "ablyChannel",
				Channel:       outputChannel,
				AblyKey:       config.Ably.APIKey,
				AblyUrl:       config.Subscriber.PushDevice.URL,
			},
		},
	}
	resp, err := rest.Request(
		ctx,
		"POST",
		"/push/deviceRegistrations",
		nil,
		regInput,
		nil)
	if err != nil {
		log.Debug("error registering a push device", "err", err)
		return err
	}
	if !resp.Success {
		log.Debug("error registering a push device", "statusCode", resp.StatusCode, "errorMessage", resp.ErrorMessage)
		return errors.New(resp.ErrorMessage)
	}
	return nil
}

func subscribePushDevice(ctx context.Context, config *config.Config, deviceID, channel string, rest *ably.REST, log log15.Logger) error {
	type input struct {
		Channel  string `json:"channel"`
		DeviceId string `json:"deviceId"`
	}
	subInput := &input{
		Channel:  channel,
		DeviceId: deviceID,
	}
	resp, err := rest.Request(
		ctx,
		"POST",
		"/push/channelSubscriptions",
		nil,
		subInput,
		nil)
	if err != nil {
		log.Debug("error subscribing to a channel", "err", err)
		return err
	}
	if !resp.Success {
		log.Debug("error subscribing to a channel", "statusCode", resp.StatusCode, "errorMessage", resp.ErrorMessage)
		return errors.New(resp.ErrorMessage)
	}
	return nil
}

// runSubscriber runs a subscriber task which renders the channel names using
// the given user number and subscribes to each of them.
func (l *loadTest) runSubscriber(ctx context.Context, client Client, userNum int64) error {
	channels := renderChannels(l.subscriberChannels, userNum)

	errG, ctx := errgroup.WithContext(ctx)
	if l.w.Conf().Subscriber.PushDevice.Enabled {
		l.log.Debug("creating push device")

		if l.w.Conf().Subscriber.PushDevice.MetachannelEnabled {
			errG.Go(func() error {
				for {
					l.log.Debug("subscribing to metachannel")
					err := client.Subscribe(ctx, "[meta]log:push", func(message *ably.Message) {
						data := message.Data.(map[string]interface{})
						if severity, ok := data["severity"]; ok && severity == "error" {
							if meta, ok := data["meta"].(map[string]interface{}); ok {
								if errStr, ok := meta["error"].(string); ok {
									l.w.boomer.RecordFailure("ablyboomer", "pushLog", 0, errStr)
								}
							}
						}
						l.log.Debug("push metachannel:", "message", data)
					})
					if errors.Is(err, context.Canceled) {
						return nil
					} else if err != nil {
						l.log.Debug("error subscribing to push metachannel", "err", err)
						// try again in a second
						select {
						case <-time.After(time.Second):
						case <-ctx.Done():
							return nil
						}
					}
				}
			})
		}

		rest, err := ably.NewREST(l.w.conf.Ably.ClientOptions()...)
		if err != nil {
			l.log.Debug("error creating a REST client", "err", err)
			l.w.boomer.RecordFailure("ablyboomer", "createREST", 0, err.Error())
			return err
		}

		name := randomString(8)
		deviceID := fmt.Sprintf("device-%v", name)
		outputChannel := fmt.Sprintf("push-%v", name)

		for {
			l.log.Debug("registering push device", "deviceID", deviceID)
			err := registerPushDevice(ctx, l.w.Conf(), deviceID, outputChannel, rest, l.log)
			if err == nil {
				break
			} else {
				l.log.Debug("error registering push device", "deviceID", deviceID, "err", err)
				l.w.boomer.RecordFailure("ablyboomer", "registerPushDevice", 0, err.Error())
				// try again in a second
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return nil
				}
			}
		}

		for _, channel := range channels {
			for {
				l.log.Debug("subscribing push device", "deviceID", deviceID)
				err := subscribePushDevice(ctx, l.w.Conf(), deviceID, channel, rest, l.log)
				if err == nil {
					break
				} else {
					l.log.Debug("error subscribing push device", "deviceID", deviceID, "err", err)
					l.w.boomer.RecordFailure("ablyboomer", "subscribePushDevice", 0, err.Error())
					// try again in a second
					select {
					case <-time.After(time.Second):
					case <-ctx.Done():
						return nil
					}
				}
			}
		}

		// If we enable the push device we want the subscriber to subscribe to
		// the channel the device will push to when receiving a notification.
		channels = []string{outputChannel}
	}

	l.log.Debug("starting subscriber", "channels", channels)

	for i := range channels {
		channel := channels[i]
		errG.Go(func() error {
			for {
				l.log.Debug("subscribing", "channel", channel)
				err := client.Subscribe(ctx, channel, func(message *ably.Message) {
					data := []byte(message.Data.(string))
					var msg Message
					if err := json.Unmarshal(data, &msg); err != nil {
						l.log.Debug("error parsing message", "err", err)
						l.w.boomer.RecordFailure("ablyboomer", "subscribe", 0, err.Error())
						return
					}
					latency := timeNow() - msg.Data.Time
					size := int64(len(data))
					l.log.Debug("subscriber received message", "channel", channel, "latency", latency, "size", size)
					l.w.boomer.RecordSuccess("ablyboomer", "subscribe", latency, size)
				})
				if errors.Is(err, context.Canceled) {
					l.log.Debug("subscriber stopped")
					return nil
				} else if err != nil {
					l.log.Debug("error subscribing", "channel", channel, "err", err)
					l.w.boomer.RecordFailure("ablyboomer", "subscribe", 0, err.Error())
					// try again in a second
					select {
					case <-time.After(time.Second):
					case <-ctx.Done():
						return nil
					}
				}
			}
		})
	}
	return errG.Wait()
}

// runPublisher runs a publisher task which renders the channel names using the
// given user number and publishes to each of them at the configured publish
// interval.
func (l *loadTest) runPublisher(ctx context.Context, client Client, userNum int64) error {
	channels := renderChannels(l.publisherChannels, userNum)

	l.log.Debug("starting publisher", "channels", channels, "interval", l.w.Conf().Publisher.PublishInterval)

	errG, ctx := errgroup.WithContext(ctx)
	for i := range channels {
		channel := channels[i]
		errG.Go(func() error {
			ticker := time.NewTicker(l.w.Conf().Publisher.PublishInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					var extras map[string]interface{}
					if l.w.Conf().Publisher.PushEnabled {
						extras = map[string]interface{}{
							"push": map[string]interface{}{
								"data": map[string]interface{}{
									"time": timeNow(),
								},
							},
						}
					}
					data, _ := json.Marshal(&Message{
						Data: Data{
							Content: randomString(l.w.Conf().Publisher.MessageSize),
							Time:    timeNow(),
						},
					})
					errG.Go(func() error {
						l.log.Debug("publishing message", "channel", channel, "size", len(data))
						err := client.Publish(ctx, channel, []*ably.Message{{
							Data:   data,
							Extras: extras,
						}})
						if errors.Is(err, context.Canceled) {
							l.log.Debug("publication canceled", "channel", channel)
						} else if err != nil {
							l.log.Debug("error publishing message", "channel", channel, "err", err)
							l.w.boomer.RecordFailure("ablyboomer", "publish", 0, err.Error())
						} else {
							l.w.boomer.RecordSuccess("ablyboomer", "publish", 0, int64(len(data)))
						}
						return nil
					})
				case <-ctx.Done():
					l.log.Debug("publisher stopped", "channel", channel)
					return nil
				}
			}
		})
	}
	return errG.Wait()
}

// runPresence runs a presence task which renders the channel names using the
// given user number and enters each of them.
func (l *loadTest) runPresence(ctx context.Context, client Client, userNum int64) error {
	channels := renderChannels(l.presenceChannels, userNum)

	l.log.Debug("starting presence", "channels", channels)

	errG, ctx := errgroup.WithContext(ctx)
	for i := range channels {
		channel := channels[i]
		errG.Go(func() error {
			for {
				l.log.Debug("entering", "channel", channel)
				err := client.Enter(ctx, channel, fmt.Sprintf("user%d", userNum))
				if err == nil {
					// we successfully entered, just wait for the load test
					// to stop
					l.w.boomer.RecordSuccess("ablyboomer", "presence", 0, 0)
					<-ctx.Done()
					l.log.Debug("entering done", "channel", channel)
					return nil
				} else if errors.Is(err, context.Canceled) {
					l.log.Debug("presence stopped", "channel", channel)
					return nil
				}
				l.log.Debug("error entering", "channel", channel, "err", err)
				l.w.boomer.RecordFailure("ablyboomer", "presence", 0, err.Error())
				// try again in a second
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return nil
				}
			}
		})
	}
	return errG.Wait()
}

// stop stops the all the running users for this load test and waits for them
// to stop.
func (l *loadTest) stop() {
	l.log.Debug("stopping load test")
	close(l.stopC)
	l.users.Wait()
}

// Data includes content as a string as well as a timestamp.
type Data struct {
	Content string `json:"content"`
	Time    int64  `json:"time"`
}

// Message is the data that is published by publisher tasks and used by
// subscriber tasks to generate latency and message size stats.
type Message struct {
	Data Data `json:"data"`
}

// timeNow returns the current UTC time as milliseconds since the epoch.
func timeNow() int64 {
	return time.Now().UTC().UnixNano() / int64(time.Millisecond)
}

// randomString returns a random hex string of the given length.
func randomString(length int64) string {
	data := make([]byte, length/2+1)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data)[:length]
}
