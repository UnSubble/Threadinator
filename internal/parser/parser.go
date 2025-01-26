package parser

import (
	"errors"
	"flag"
	"strings"
	"time"

	"github.com/unsubble/threadinator/internal/executor"
)

func ParseArgs(args []string) (*executor.Config, error) {
	config := &executor.Config{
		ThreadCount: 5,
		Timeout:     10 * time.Second,
	}

	fs := flag.NewFlagSet("threadinator", flag.ContinueOnError)

	commandsFlag := fs.String("e", "", "Semicolon-separated commands to execute")
	fs.IntVar(&config.ThreadCount, "c", config.ThreadCount, "Number of concurrent threads")
	fs.BoolVar(&config.UsePipeline, "p", false, "Enable pipeline mode")
	fs.BoolVar(&config.Verbose, "v", false, "Enable verbose output")
	timeoutFlag := fs.Int("t", int(config.Timeout.Seconds()), "Timeout duration in seconds")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	commands := strings.Trim(*commandsFlag, "\" '")
	if commands == "" {
		return nil, errors.New("at least one command is required")
	}

	config.Commands = parseCommands(commands)
	config.Timeout = time.Duration(*timeoutFlag) * time.Second

	return config, nil
}

func splitCommand(commandStr string) *executor.Command {
	parts := strings.Split(strings.Trim(commandStr, "\" '"), " ")
	cmd := parts[0]
	args := make([]string, 0)
	if len(parts) > 1 {
		args = parts[1:]
	}
	return &executor.Command{
		Command: cmd,
		Args:    args,
	}
}

func parseCommands(commands string) []executor.Command {
	var (
		commandSlice []executor.Command
		currentQuote byte
		isQuoted     bool
		start        int
	)

	for i := 0; i < len(commands); i++ {
		char := commands[i]

		if char == '\'' || char == '"' {
			if !isQuoted || currentQuote == char {
				isQuoted = !isQuoted
				currentQuote = char
			}
		}

		if !isQuoted && char == ';' && (i == 0 || commands[i-1] != '\\') {
			cmd := splitCommand(strings.TrimSpace(commands[start:i]))
			commandSlice = append(commandSlice, *cmd)
			start = i + 1
		}
	}

	if start < len(commands) {
		cmd := splitCommand(strings.TrimSpace(commands[start:]))
		commandSlice = append(commandSlice, *cmd)
	}

	return commandSlice
}
