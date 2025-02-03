package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/unsubble/threadinator/internal/executor"
)

func changeConfigSettings(toChange string) error {
	toChangeMap := make(map[string]any)
	if err := json.Unmarshal([]byte(toChange), &toChangeMap); err != nil {
		return fmt.Errorf("error unmarshaling config changes: %v", err)
	}

	file, err := os.OpenFile("config.json", os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	settingsMap := make(map[string]any)

	if err := decoder.Decode(&settingsMap); err != nil {
		return fmt.Errorf("error decoding config file: %v", err)
	}

	for name := range settingsMap {
		changeVal, has := toChangeMap[name]
		if has {
			settingsMap[name] = changeVal
		}
	}

	file.Seek(0, 0)
	file.Truncate(0)

	configData, err := json.MarshalIndent(settingsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling updated config: %v", err)
	}

	_, err = file.Write(configData)
	return err
}

func getTimeUnit(unit string) time.Duration {
	switch unit {
	case "h":
		return time.Hour
	case "m":
		return time.Minute
	case "s":
		return time.Second
	case "ms":
		return time.Millisecond
	case "micros":
		return time.Microsecond
	}

	logrus.Fatal("Unknown time unit format")
	return 0
}

func ParseArgs(config *executor.Config, cmd *cobra.Command) error {
	flags := cmd.Flags()

	if version, _ := flags.GetBool("version"); version {
		fmt.Printf("%s version %s\n", config.Name, config.Version)
	}

	logLevel, _ := flags.GetString("log-level")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("unknown log level: %v", err)
	}
	if level > logrus.ErrorLevel && level < logrus.DebugLevel {
		return fmt.Errorf("unsupported log level: %s", logLevel)
	}

	config.Logger = logrus.New()

	if verbose, _ := flags.GetBool("v"); verbose {
		config.Logger.SetLevel(logrus.DebugLevel)
	} else {
		config.Logger.SetLevel(level)
	}

	config.Logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		DisableTimestamp:          true,
		PadLevelText:              true,
	})

	configSettings, _ := flags.GetString("cfg")
	configSettingsStr := strings.TrimSpace(configSettings)
	if configSettingsStr != "" {
		if err := changeConfigSettings(configSettingsStr); err != nil {
			return fmt.Errorf("error on parsing cfg: %v", err)
		}
		config.Logger.Info("Config successfully changed.")
		return nil
	}

	commandsStr, _ := flags.GetString("execute")
	commandsStr = strings.TrimSpace(commandsStr)
	commands := parseCommands(commandsStr, config.Logger)

	timeoutFlag, _ := flags.GetInt("timeout")
	config.Timeout = time.Duration(timeoutFlag) * getTimeUnit(config.TimeUnit)

	for _, cmd := range commands {
		for c := 0; c < cmd.Times; c++ {
			config.Commands = append(config.Commands, cmd)
		}
	}

	if config.ThreadCount <= 0 {
		config.ThreadCount = len(config.Commands)
	}

	return nil
}

func sanitizeCommand(command string) string {
	return strings.Trim(command, "\" '")
}

func splitCommand(commandStr string, logger *logrus.Logger) *executor.Command {
	logger.Infof("Splitting command: %s", commandStr)
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
		logger.Warn("Empty command detected")
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

func parseCommands(commands string, logger *logrus.Logger) []*executor.Command {
	logger.Infof("Parsing commands: %s", commands)
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
				cmd := splitCommand(strings.TrimSpace(commands[start:i]), logger)
				commandSlice = append(commandSlice, cmd)
				start = i + 1
			}
		}
	}

	if start < len(commands) {
		cmd := splitCommand(strings.TrimSpace(commands[start:]), logger)
		commandSlice = append(commandSlice, cmd)
	}

	logger.Infof("Parsed commands: %+v", commandSlice)
	return commandSlice
}
