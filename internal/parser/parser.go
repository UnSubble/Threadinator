package parser

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/unsubble/threadinator/internal/executor"
	"github.com/unsubble/threadinator/internal/utils"
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

	commands := strings.TrimSpace(*commandsFlag)
	if commands == "" {
		return nil, errors.New("at least one command is required")
	}

	config.Commands = parseCommands(commands)
	config.Timeout = time.Duration(*timeoutFlag) * time.Second

	return config, nil
}

func sanitizeCommand(command string) string {
	return strings.Trim(command, "\" '")
}

func splitCommand(commandStr string) *executor.Command {
	commandStr = sanitizeCommand(commandStr)
	timesIndex := strings.LastIndex(commandStr, ":")

	times := 1

	if timesIndex >= 0 {
		parsedTimes, err := strconv.Atoi(commandStr[timesIndex+1:])
		if err != nil {
			utils.LogErrorStr(fmt.Sprintf("Invalid execution count in command: %v", err))
		} else {
			times = parsedTimes
		}
		commandStr = commandStr[:timesIndex]
	}

	parts := strings.Fields(commandStr)
	if len(parts) == 0 {
		utils.LogErrorStr("Empty command detected")
		return &executor.Command{}
	}

	for index, arg := range parts[1:] {
		parts[index+1] = sanitizeCommand(arg)
	}

	return &executor.Command{
		Command: parts[0],
		Args:    parts[1:],
		Times:   times,
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

		switch char {
		case '\'', '"':
			if !isQuoted || currentQuote == char {
				isQuoted = !isQuoted
				currentQuote = char
			}
		case ';':
			if !isQuoted && (i == 0 || commands[i-1] != '\\') {
				cmd := splitCommand(strings.TrimSpace(commands[start:i]))
				commandSlice = append(commandSlice, *cmd)
				start = i + 1
			}
		}
	}

	if start < len(commands) {
		cmd := splitCommand(strings.TrimSpace(commands[start:]))
		commandSlice = append(commandSlice, *cmd)
	}

	return commandSlice
}
