package parser

import (
	"errors"
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/unsubble/threadinator/internal/executor"
	"github.com/unsubble/threadinator/internal/utils"
)

func ParseArgs(args []string) (*executor.Config, error) {
	config := &executor.Config{
		ThreadCount: 0,
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

	commandsStr := strings.TrimSpace(*commandsFlag)
	if commandsStr == "" {
		return nil, errors.New("at least one command is required")
	}

	commands := parseCommands(commandsStr)
	config.Timeout = time.Duration(*timeoutFlag) * time.Second

	for _, cmd := range commands {
		for c := 0; c < cmd.Times; c++ {
			config.Commands = append(config.Commands, cmd)
		}
	}

	if config.ThreadCount <= 0 {
		config.ThreadCount = len(config.Commands)
	}

	return config, nil
}

func sanitizeCommand(command string) string {
	return strings.Trim(command, "\" '")
}

func splitCommand(commandStr string) *executor.Command {
	commandStr = sanitizeCommand(commandStr)
	extrasIndex := strings.LastIndex(commandStr, ":")

	var dependency *int
	times := 1

	if extrasIndex >= 0 {
		extras := commandStr[extrasIndex+1:]
		d, t := parseExtras(extras)
		if t != nil && *t > 0 {
			times = *t
		}
		dependency = d
		commandStr = commandStr[:extrasIndex]
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
		Command:    parts[0],
		Args:       parts[1:],
		Times:      times,
		Dependency: dependency,
	}
}

func parseExtras(extras string) (*int, *int) {
	split := strings.Split(extras, "|")
	var depends *int
	var times *int

	if len(split) > 1 {
		if d, err := strconv.Atoi(strings.TrimSpace(split[0])); err == nil {
			depends = new(int)
			*depends = d
		}
	}

	if t, err := strconv.Atoi(strings.TrimSpace(split[len(split)-1])); err == nil {
		times = new(int)
		*times = t
	}

	return depends, times
}

func parseCommands(commands string) []*executor.Command {
	var (
		commandSlice []*executor.Command
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
				commandSlice = append(commandSlice, cmd)
				start = i + 1
			}
		}
	}

	if start < len(commands) {
		cmd := splitCommand(strings.TrimSpace(commands[start:]))
		commandSlice = append(commandSlice, cmd)
	}

	return commandSlice
}
