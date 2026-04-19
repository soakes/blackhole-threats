package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/soakes/blackhole-threats/internal/bgp"
	"github.com/soakes/blackhole-threats/internal/buildinfo"
	"github.com/soakes/blackhole-threats/internal/config"
)

func main() {
	cfgPath := flag.String("conf", "blackhole-threats.yaml", "Configuration file")
	checkConfig := flag.Bool("check-config", false, "Validate configuration and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	once := flag.Bool("once", false, "Run a single refresh cycle and exit")
	refreshRate := flag.Duration("refresh-rate", 2*time.Hour, "Refresh timer")
	showVersion := flag.Bool("version", false, "Print version information and exit")
	var extraFeeds config.FeedList
	flag.Var(&extraFeeds, "feed", "Threat intelligence feed (use multiple times)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("blackhole-threats %s\ncommit: %s\nbuilt: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)
		return
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	if err := cfg.Validate(); err != nil {
		log.WithError(err).Fatal("Invalid configuration")
	}

	feeds := append([]config.Feed{}, cfg.Feeds...)
	feeds = append(feeds, extraFeeds...)
	if err := config.ValidateFeeds(feeds); err != nil {
		log.WithError(err).Fatal("Invalid feed configuration")
	}
	if *refreshRate <= 0 {
		log.WithField("refresh-rate", *refreshRate).Fatal("Refresh timer must be greater than zero")
	}
	if *checkConfig {
		fmt.Printf("configuration OK: %d feed(s)\n", len(feeds))
		return
	}

	router, err := bgp.New(&cfg.GoBGP)
	if err != nil {
		log.WithError(err).Fatal("Failed to start BGP server")
	}
	defer func() {
		if err := router.Stop(); err != nil {
			log.WithError(err).Warn("Failed to stop BGP server cleanly")
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGUSR1)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.WithFields(log.Fields{
		"version": buildinfo.Version,
		"commit":  buildinfo.Commit,
		"built":   buildinfo.BuildDate,
	}).Info("Starting blackhole-threats")
	if err := router.Run(ctx, feeds, *refreshRate, sigC, *once); err != nil {
		if stopErr := router.Stop(); stopErr != nil {
			log.WithError(stopErr).Warn("Failed to stop BGP server cleanly after runtime error")
		}
		log.WithError(err).Error("Route sync failed")
		os.Exit(1)
	}

	log.Info("Shutdown complete")
}
