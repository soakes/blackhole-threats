package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/soakes/blackhole-threats/internal/bgp"
	"github.com/soakes/blackhole-threats/internal/buildinfo"
	"github.com/soakes/blackhole-threats/internal/config"
	"github.com/soakes/blackhole-threats/internal/logging"
)

func runMode(runOnce bool) string {
	if runOnce {
		return "oneshot"
	}

	return "daemon"
}

func logStartup(cfgPath string, cfg config.File, configuredFeeds, extraFeeds int, refreshInterval time.Duration, runOnce bool) {
	log.WithFields(log.Fields{
		"tag_version": buildinfo.TagVersion(),
		"commit":      buildinfo.Commit,
		"build_date":  buildinfo.BuildDate,
		"go_version":  runtime.Version(),
		"go_os":       runtime.GOOS,
		"go_arch":     runtime.GOARCH,
		"pid":         os.Getpid(),
	}).Info("Starting blackhole-threats")

	log.WithFields(log.Fields{
		"config_path":       cfgPath,
		"run_mode":          runMode(runOnce),
		"refresh_interval":  refreshInterval.String(),
		"configured_feeds":  configuredFeeds,
		"cli_feeds":         extraFeeds,
		"total_feeds":       configuredFeeds + extraFeeds,
		"peer_count":        len(cfg.GoBGP.Neighbors),
		"local_as":          cfg.GoBGP.Global.Config.As,
		"router_id":         cfg.GoBGP.Global.Config.RouterId,
		"default_community": config.DefaultCommunity(cfg.GoBGP.Global.Config.As).String(),
	}).Info("Loaded runtime configuration")
}

func main() {
	log.SetFormatter(logging.OperatorFormatter{})
	log.SetOutput(os.Stdout)

	cfgPath := flag.String("conf", "blackhole-threats.yaml", "Configuration file")
	checkConfig := flag.Bool("check-config", false, "Validate configuration and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	logFormat := flag.String("log-format", "logfmt", "Log format (logfmt or json)")
	logLevel := flag.String("log-level", "info", "Log level (panic, fatal, error, warn, info, debug, trace)")
	once := flag.Bool("once", false, "Run a single refresh cycle and exit")
	refreshRate := flag.Duration("refresh-rate", 2*time.Hour, "Refresh timer")
	showVersion := flag.Bool("version", false, "Print version information and exit")
	var extraFeeds config.FeedList
	flag.Var(&extraFeeds, "feed", "Threat intelligence feed (use multiple times)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("blackhole-threats %s\ncommit: %s\nbuilt: %s\n", buildinfo.DisplayVersion(), buildinfo.Commit, buildinfo.BuildDate)
		return
	}

	formatter, err := logging.NewFormatter(*logFormat)
	if err != nil {
		log.WithField("log_format", *logFormat).WithError(err).Fatal("Invalid log format")
	}
	log.SetFormatter(formatter)

	selectedLogLevel := *logLevel
	if *debug {
		selectedLogLevel = "debug"
	}

	parsedLogLevel, err := log.ParseLevel(selectedLogLevel)
	if err != nil {
		log.WithField("log_level", selectedLogLevel).WithError(err).Fatal("Invalid log level")
	}
	log.SetLevel(parsedLogLevel)

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
		log.WithField("refresh_interval", *refreshRate).Fatal("Refresh timer must be greater than zero")
	}
	if *checkConfig {
		fmt.Printf("configuration OK: %d feed(s)\n", len(feeds))
		return
	}
	logStartup(*cfgPath, cfg, len(cfg.Feeds), len(extraFeeds), *refreshRate, *once)

	router, err := bgp.New(&cfg.GoBGP)
	if err != nil {
		log.WithError(err).Fatal("Failed to start BGP server")
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGUSR1)
	defer signal.Stop(sigC)

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Started blackhole-threats")
	if err := router.Run(ctx, feeds, *refreshRate, sigC, *once); err != nil {
		if stopErr := router.Stop(); stopErr != nil {
			log.WithError(stopErr).Warn("Failed to stop BGP server cleanly after runtime error")
		}
		log.WithError(err).Error("Route sync failed")
		os.Exit(1)
	}

	if ctx.Err() != nil {
		select {
		case sig := <-shutdownSignals:
			log.WithField("signal", sig.String()).Info("Shutdown requested")
		default:
			log.Info("Shutdown requested")
		}
	} else if *once {
		log.Info("Completed single refresh cycle")
	}

	if err := router.Stop(); err != nil {
		log.WithError(err).Warn("Failed to stop BGP server cleanly")
	} else {
		log.Info("Stopped blackhole-threats")
	}
}
