package main

import (
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/devetek/tuman/pkg/marijan"
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
				marijan.WithInterval(1*time.Second),
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

	// get default config path relative to home directory
	defaultConfigPath, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Error getting user home directory: %v", zap.Error(err))
	}
	defaultConfigPath = path.Join(defaultConfigPath, ".marijan/config.json")

	runCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	runCmd.PersistentFlags().StringVarP(&configFile, "config", "c", defaultConfigPath, "Path to the config file")

	return runCmd
}
