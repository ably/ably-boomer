package main

import (
	"fmt"
	"os"

	ablyboomer "github.com/ably/ably-boomer"
	"github.com/ably/ably-boomer/ably/perf"
	"github.com/ably/ably-boomer/config"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

func main() {
	conf := config.Default()
	perfConf := perf.DefaultConfig()
	log := log15.New()

	app := &cli.App{
		Flags: append(conf.Flags(), perfConf.Flags()...),
		Action: func(c *cli.Context) error {
			return run(conf, perfConf, log)
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Crit("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(conf *config.Config, perfConf *perf.Config, log log15.Logger) error {
	log.Info("starting perf")
	perf := perf.New(perfConf)
	if err := perf.Start(); err != nil {
		return fmt.Errorf("error starting perf: %w", err)
	}
	defer perf.Stop()

	return ablyboomer.RunAblyTask(conf, log)
}
