package parsers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/unsubble/threadinator/internal/models"
)

func changeConfigSettings(toChange string) error {
	toChangeMap := make(map[string]any)
	if err := json.Unmarshal([]byte(toChange), &toChangeMap); err != nil {
		return models.NewConfigParseError(err)
	}

	file, err := os.OpenFile("config.json", os.O_RDWR, 0644)
	if err != nil {
		return models.NewFileOpenError("config.json", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	settingsMap := make(map[string]any)

	if err := decoder.Decode(&settingsMap); err != nil {
		return models.NewConfigDecodeError(err)
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
		return models.NewConfigMarshalError(err)
	}

	_, err = file.Write(configData)
	return err
}

func ParseArgs(config *models.Config, cmd *cobra.Command) error {
	flags := cmd.Flags()

	if version, _ := flags.GetBool("version"); version {
		fmt.Printf("%s version %s\n", config.Name, config.Version)
	}

	logLevel, _ := flags.GetString("log-level")
	level, err := logrus.ParseLevel(logLevel)

	if err != nil {
		return models.NewLogLevelError(logLevel, err)
	}

	if level < logrus.ErrorLevel || level > logrus.DebugLevel {
		return models.NewUnsupportedLogLevelError(logLevel)
	}

	config.Logger = logrus.New()

	if config.Verbose {
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
			return models.NewConfigChangeError(err)
		}
		config.Logger.Info("Config successfully changed.")
		return nil
	}

	commandsStr, _ := flags.GetString("execute")
	commandsStr = strings.TrimSpace(commandsStr)
	commands := parseCommands(commandsStr, config.Logger)

	timeoutFlag, _ := flags.GetInt("timeout")
	config.Timeout = time.Duration(timeoutFlag) * GetTimeUnit(config.TimeUnit)

	for _, cmd := range commands {
		for c := 0; c < cmd.Times; c++ {
			if cmd.Dependency != nil && *cmd.Dependency < 0 {
				*cmd.Dependency = len(config.Commands) + *cmd.Dependency
			}
			config.Commands = append(config.Commands, cmd)
		}
	}

	if config.ThreadCount <= 0 {
		config.ThreadCount = len(config.Commands)
	}

	return nil
}

func parseCommands(commands string, logger *logrus.Logger) []*models.Command {
	logger.Infof("Parsing commands: %s", commands)
	var (
		commandSlice []*models.Command
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

func sanitizeCommand(command string) string {
	return strings.Trim(command, "\" '")
}

func splitCommand(commandStr string, logger *logrus.Logger) *models.Command {
	logger.Infof("Splitting command: %s", commandStr)
	commandStr = sanitizeCommand(commandStr)
	extrasIndex := strings.LastIndex(commandStr, ":")

	var dependency *int
	var delay *int
	times := 1

	if extrasIndex >= 0 {
		extras := commandStr[extrasIndex+1:]
		dep, del, t := parseExtras(extras)
		if t != nil && *t > 0 {
			times = *t
		}
		dependency = dep
		delay = del
		commandStr = commandStr[:extrasIndex]
	}

	parts := strings.Fields(commandStr)
	if len(parts) == 0 {
		logger.Warn("Empty command detected")
		return &models.Command{}
	}

	for index, arg := range parts[1:] {
		parts[index+1] = sanitizeCommand(arg)
	}

	return &models.Command{
		Command:    parts[0],
		Args:       parts[1:],
		Times:      times,
		Delay:      delay,
		Dependency: dependency,
	}
}

func parseExtras(extras string) (*int, *int, *int) {
	parts := strings.Split(extras, "|")
	var depends, delay, times *int

	parseIntPointer := func(s string) *int {
		if value, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			ptr := new(int)
			*ptr = value
			return ptr
		}
		return nil
	}

	if len(parts) > 2 {
		depends = parseIntPointer(parts[0])
	}

	if len(parts) > 1 {
		delayPart := strings.TrimSpace(parts[len(parts)-2])
		delay = parseRandomOrInt(delayPart)
	}

	if len(parts) > 0 {
		times = parseIntPointer(parts[len(parts)-1])
	}

	return depends, delay, times
}

func parseRandomOrInt(value string) *int {
	value = strings.TrimSpace(value)
	if d, err := strconv.Atoi(value); err == nil {
		ptr := new(int)
		*ptr = d
		return ptr
	} else if strings.HasPrefix(value, "rand(") && strings.HasSuffix(value, ")") {
		randomRange := strings.TrimSuffix(strings.TrimPrefix(value, "rand("), ")")
		bounds := strings.Split(randomRange, ",")
		return parseRandomRange(bounds)
	}
	return nil
}

func parseRandomRange(bounds []string) *int {
	if len(bounds) == 1 {
		if max, err := strconv.Atoi(strings.TrimSpace(bounds[0])); err == nil {
			ptr := new(int)
			*ptr = rand.Intn(max)
			return ptr
		}
	} else if len(bounds) == 2 {
		if min, err1 := strconv.Atoi(strings.TrimSpace(bounds[0])); err1 == nil {
			if max, err2 := strconv.Atoi(strings.TrimSpace(bounds[1])); err2 == nil {
				ptr := new(int)
				*ptr = rand.Intn(max-min) + min
				return ptr
			}
		}
	}
	return nil
}
