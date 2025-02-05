package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/unsubble/threadinator/internal/executor"
	"github.com/unsubble/threadinator/internal/models"
	"github.com/unsubble/threadinator/internal/parsers"
)

func readConfig() (*models.Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, models.NewConfigParseError(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &models.Config{}
	if err := decoder.Decode(config); err != nil {
		return nil, models.NewConfigDecodeError(err)
	}

	return config, nil
}

func NewRootCmd(config *models.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   config.Name,
		Short: config.ShortDesc,
		Long:  config.LongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ParseFlags(args); err != nil {
				return fmt.Errorf("error parsing flags: %v", err)
			}
			if err := parsers.ParseArgs(config, cmd); err != nil {
				config.Logger.Errorf("Error: %v", err)
				os.Exit(1)
			}
			return executor.Execute(config)
		},
		SilenceUsage: true,
	}

	cmd.Flags().StringP("execute", "e", "", "Semicolon-separated commands to execute")
	cmd.Flags().IntVarP(&config.ThreadCount, "count", "c", 0, "Number of concurrent threads")
	cmd.Flags().BoolVarP(&config.UsePipeline, "pipeline", "p", false, "Enable pipeline mode")
	cmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().String("log-level", "ERROR", "Set the logging level (INFO, DEBUG, WARN, ERROR)")
	cmd.Flags().IntP("timeout", "t", config.TimeoutInt, "Timeout duration in seconds")
	cmd.Flags().String("cfg", "", "Change default settings (must be in JSON syntax)")
	cmd.Flags().BoolP("version", "V", false, "Show tool version")

	return cmd
}

func main() {
	config, err := readConfig()
	if err != nil {
		logrus.Fatalf("Error while reading config file: %v", err)
		os.Exit(1)
	}

	rootCmd := NewRootCmd(config)

	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOutput(os.Stdout)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
