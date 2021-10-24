package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	ablyboomer "github.com/ably/ably-boomer"
	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

func main() {
	conf := config.Default()
	flags := conf.Flags()
	log := log15.New()
  log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StdoutHandler))
	app := &cli.App{
		Name:   "ably-boomer",
		Usage:  "Ably load generator for Locust, based on the boomer library",
		Flags:  flags,
		Before: config.InitFileSourceFunc(flags, log),
		Action: func(c *cli.Context) error {
			rand.Seed(time.Now().UnixNano())

			// initialise an ablyboomer worker
			worker, err := ablyboomer.NewWorker(conf, ablyboomer.WithLog(log))
			if err != nil {
				return err
			}

			// shutdown gracefully on SIGINT or SIGTERM
			ctx, cancel := context.WithCancel(c.Context)
			go func() {
				defer cancel()
				ch := make(chan os.Signal, 1)
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
				sig := <-ch
				log.Info("received signal, exiting...", "signal", sig)
			}()

			// run the worker until it exits
			worker.Run(ctx)
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Crit("error running ably-boomer", "err", err)
		os.Exit(1)
	}
}
