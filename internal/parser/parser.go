package parser

import (
	"errors"
	"flag"
	"time"

	"github.com/unsubble/threadinator/internal/executor"
)

func ParseArgs(args []string) (*executor.Config, error) {
	fs := flag.NewFlagSet("threadinator", flag.ContinueOnError)

	config := &executor.Config{
		ThreadCount: 5,
		Timeout:     10 * time.Second,
	}

	fs.StringVar(&config.Command, "e", "", "Command to execute")
	fs.IntVar(&config.ThreadCount, "c", config.ThreadCount, "Number of threads")
	fs.BoolVar(&config.UsePipeline, "p", false, "Use pipeline")
	fs.BoolVar(&config.Verbose, "v", false, "Verbose")

	timeoutFlag := fs.Int("t", int(config.Timeout.Seconds()), "Timeout in seconds")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if config.Command == "" {
		return nil, errors.New("command is required")
	}

	config.Args = fs.Args()
	config.Timeout = time.Duration(*timeoutFlag) * time.Second

	return config, nil
}
