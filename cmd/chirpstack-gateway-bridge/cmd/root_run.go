package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/commands"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/filters"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/forwarder"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/metadata"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/metrics"
)

func run(cmd *cobra.Command, args []string) error {

	tasks := []func() error{
		setLogLevel,
		setSyslog,
		printStartMessage,
		setupFilters,
		setupBackend,
		setupIntegration,
		setupForwarder,
		setupMetrics,
		setupMetaData,
		setupCommands,
		startIntegration,
		startBackend,
	}

	for _, t := range tasks {
		if err := t(); err != nil {
			log.Fatal(err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")

	integration.GetIntegration().Stop()

	return nil
}

func setLogLevel() error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))
	return nil
}

func printStartMessage() error {
	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://www.chirpstack.io/gateway-bridge/",
	}).Info("starting ChirpStack Gateway Bridge")
	return nil
}

func setupBackend() error {
	if err := backend.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup backend error")
	}
	return nil
}

func setupIntegration() error {
	if err := integration.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup integration error")
	}
	return nil
}

func setupForwarder() error {
	if err := forwarder.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup forwarder error")
	}
	return nil
}

func setupMetrics() error {
	if err := metrics.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup metrics error")
	}
	return nil
}

func setupMetaData() error {
	if err := metadata.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup meta-data error")
	}
	return nil
}

func setupFilters() error {
	if err := filters.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup filters error")
	}
	return nil
}

func setupCommands() error {
	if err := commands.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup commands error")
	}
	return nil
}

func startIntegration() error {
	if err := integration.GetIntegration().Start(); err != nil {
		return errors.Wrap(err, "start integration error")
	}
	return nil
}

func startBackend() error {
	if err := backend.GetBackend().Start(); err != nil {
		return errors.Wrap(err, "start backend error")
	}
	return nil
}
