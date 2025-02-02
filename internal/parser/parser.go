package parser

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/unsubble/threadinator/internal/executor"
)

func changeConfigSettings(toChange string) error {
	logrus.Info("Entering changeConfigSettings")
	toChangeMap := make(map[string]any)
	if err := json.Unmarshal([]byte(toChange), &toChangeMap); err != nil {
		logrus.Errorf("Error unmarshaling config changes: %v", err)
		return err
	}

	file, err := os.OpenFile("config.json", os.O_RDWR, 0644)
	if err != nil {
		logrus.Errorf("Error opening config file: %v", err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	settingsMap := make(map[string]any)

	if err := decoder.Decode(&settingsMap); err != nil {
		logrus.Errorf("Error decoding config file: %v", err)
		return err
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
		logrus.Errorf("Error marshaling updated config: %v", err)
		return err
	}

	_, err = file.Write(configData)
	if err != nil {
		logrus.Errorf("Error writing updated config to file: %v", err)
	}
	return err
}

func readConfig() (*executor.Config, error) {
	logrus.Info("Reading configuration file")
	file, err := os.Open("config.json")
	if err != nil {
		logrus.Errorf("Error opening config file: %v", err)
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &executor.Config{}
	if err := decoder.Decode(config); err != nil {
		logrus.Errorf("Error decoding config file: %v", err)
		return nil, err
	}

	logrus.Infof("Configuration loaded: %+v", config)
	return config, nil
}

func getTimeUnit(unit string) time.Duration {
	logrus.Infof("Parsing time unit: %s", unit)
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

func ParseArgs(args []string) (*executor.Config, error) {
	logrus.Info("Parsing arguments")
	config, err := readConfig()
	if err != nil {
		return nil, err
	}

	fs := flag.NewFlagSet(config.Name, flag.ContinueOnError)

	commandsFlag := fs.String("e", "", "Semicolon-separated commands to execute")
	fs.IntVar(&config.ThreadCount, "c", config.ThreadCount, "Number of concurrent threads")
	fs.BoolVar(&config.UsePipeline, "p", config.UsePipeline, "Enable pipeline mode")
	fs.BoolVar(&config.Verbose, "v", config.Verbose, "Enable verbose output")
	logLevel := fs.String("log-level", "ERROR", "Set the logging level (INFO, DEBUG, WARN, ERROR)")
	timeoutFlag := fs.Int("t", config.TimeoutInt, "Timeout duration in seconds")
	configSettings := fs.String("cfg", "", "Change default settings(Must be in JSON syntax)")
	showVersion := fs.Bool("V", false, "Show tool version")

	fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		logrus.Errorf("Error parsing arguments: %v", err)
		return nil, err
	}
	if *showVersion {
		fmt.Printf("%s version %s\n", config.Name, config.Version)
		os.Exit(0)
	}

	if len(args) == 0 || containsHelpFlag(args) {
		fs.Usage()
		os.Exit(0)
	}

	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		return nil, fmt.Errorf("unknown log level: %v", err)
	}
	if level > logrus.ErrorLevel && level < logrus.DebugLevel {
		return nil, fmt.Errorf("unsupported log level: %s", *logLevel)
	}
	config.Logger = logrus.New()
	config.Logger.SetLevel(level)
	config.Logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		DisableTimestamp:          true,
		PadLevelText:              true,
	})

	logrus.Debugf("Log level set to %s", *logLevel)

	configSettingsStr := strings.TrimSpace(*configSettings)
	if configSettingsStr != "" {
		if err := changeConfigSettings(configSettingsStr); err != nil {
			logrus.Errorf("Error parsing cfg: %v", err)
			return nil, fmt.Errorf("error on parsing cfg: %v", err)
		}
		logrus.Info("Config successfully changed.")
		return nil, nil
	}

	commandsStr := strings.TrimSpace(*commandsFlag)
	if commandsStr == "" {
		logrus.Error("At least one command is required")
		return nil, errors.New("at least one command is required")
	}

	commands := parseCommands(commandsStr)
	config.Timeout = time.Duration(*timeoutFlag) * getTimeUnit(config.TimeUnit)

	for _, cmd := range commands {
		for c := 0; c < cmd.Times; c++ {
			config.Commands = append(config.Commands, cmd)
		}
	}

	if config.ThreadCount <= 0 {
		config.ThreadCount = len(config.Commands)
	}

	logrus.Infof("Configuration after argument parsing: %+v", config)
	return config, nil
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			logrus.Debug("Help flag detected")
			return true
		}
	}
	return false
}

func sanitizeCommand(command string) string {
	logrus.Debugf("Sanitizing command: %s", command)
	return strings.Trim(command, "\" '")
}

func splitCommand(commandStr string) *executor.Command {
	logrus.Infof("Splitting command: %s", commandStr)
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
		logrus.Warn("Empty command detected")
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
	logrus.Infof("Parsing commands: %s", commands)
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

	logrus.Infof("Parsed commands: %+v", commandSlice)
	return commandSlice
}
