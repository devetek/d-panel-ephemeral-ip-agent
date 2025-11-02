package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devetek/tukiran-dan-marijan/pkg/marijan"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// User input variables
var verbose bool
var configFile string

func runCmd() *cobra.Command {
	// init zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
	defer logger.Sync()

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run marijan tunnel client",
		Run: func(cmd *cobra.Command, args []string) {
			manager := marijan.NewManager(
				marijan.WithURL(configFile),
				marijan.WithSource(marijan.ConfigSourceFile),
				marijan.WithInterval(5*time.Second),
				marijan.WithDebug(verbose),
				marijan.WithLogger(logger),
			)

			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			logger.Info("Starting tunnel client")

			manager.Start()

			<-done

			logger.Info("Stopping tunnel client")
		},
	}

	runCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	runCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "~/.marijan/config.json", "Path to the config file")

	return runCmd
}
