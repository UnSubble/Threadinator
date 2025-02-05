package models

import "fmt"

// executor errors
type CommandError struct {
	Command string
	Message string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("Command '%s' failed: %s", e.Command, e.Message)
}

func NewCommandError(command, message string) error {
	return &CommandError{
		Command: command,
		Message: message,
	}
}

type TimeoutError struct {
	Command string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("Timeout exceeded for command: %s", e.Command)
}

func NewTimeoutError(command string) error {
	return &TimeoutError{
		Command: command,
	}
}

type PipeError struct {
	OriginalError error
}

func (e *PipeError) Error() string {
	return fmt.Sprintf("Pipe error: %v", e.OriginalError)
}

func NewPipeError(err error) error {
	return &PipeError{
		OriginalError: err,
	}
}

type BufferError struct {
	OriginalError error
}

func (e *BufferError) Error() string {
	return fmt.Sprintf("Buffer error: %v", e.OriginalError)
}

func NewBufferError(err error) error {
	return &BufferError{
		OriginalError: err,
	}
}

type OutputReadError struct {
	OriginalError error
}

func (e *OutputReadError) Error() string {
	return fmt.Sprintf("Error reading output: %v", e.OriginalError)
}

func NewOutputReadError(err error) error {
	return &OutputReadError{
		OriginalError: err,
	}
}

type PanicError struct {
	ThreadID int
	Recover  interface{}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("Panic in Thread-%d: %v", e.ThreadID, e.Recover)
}

func NewPanicError(threadID int, recover interface{}) error {
	return &PanicError{
		ThreadID: threadID,
		Recover:  recover,
	}
}

type PipelineError struct {
	WorkerId int
}

func (e *PipelineError) Error() string {
	return fmt.Sprintf("No valid input from previous worker (Thread-%d)", e.WorkerId)
}

func NewPipelineError(workerId int) error {
	return &PipelineError{
		WorkerId: workerId,
	}
}

type DependencyError struct {
	DependencyIdx int
	CommandIdx    int
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("Invalid dependency index %d for command %d", e.DependencyIdx, e.CommandIdx)
}

func NewDependencyError(depIdx, cmdIdx int) error {
	return &DependencyError{
		DependencyIdx: depIdx,
		CommandIdx:    cmdIdx,
	}
}

type CircularDependencyError struct{}

func (e *CircularDependencyError) Error() string {
	return "Circular dependency detected"
}

func NewCircularDependencyError() error {
	return &CircularDependencyError{}
}

// Config Errors
type ConfigParseError struct {
	Cause error
}

func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("Error unmarshaling config changes: %v", e.Cause)
}

func NewConfigParseError(cause error) error {
	return &ConfigParseError{Cause: cause}
}

type FileOpenError struct {
	FilePath string
	Cause    error
}

func (e *FileOpenError) Error() string {
	return fmt.Sprintf("Error opening file %s: %v", e.FilePath, e.Cause)
}

func NewFileOpenError(filePath string, cause error) error {
	return &FileOpenError{FilePath: filePath, Cause: cause}
}

type ConfigDecodeError struct {
	Cause error
}

func (e *ConfigDecodeError) Error() string {
	return fmt.Sprintf("Error decoding config file: %v", e.Cause)
}

func NewConfigDecodeError(cause error) error {
	return &ConfigDecodeError{Cause: cause}
}

type ConfigMarshalError struct {
	Cause error
}

func (e *ConfigMarshalError) Error() string {
	return fmt.Sprintf("Error marshaling updated config: %v", e.Cause)
}

func NewConfigMarshalError(cause error) error {
	return &ConfigMarshalError{Cause: cause}
}

type ConfigChangeError struct {
	Cause error
}

func (e *ConfigChangeError) Error() string {
	return fmt.Sprintf("Error changing config settings: %v", e.Cause)
}

func NewConfigChangeError(cause error) error {
	return &ConfigChangeError{Cause: cause}
}

// Log Level Errors
type LogLevelError struct {
	LogLevel string
	Cause    error
}

func (e *LogLevelError) Error() string {
	return fmt.Sprintf("Unknown log level %s: %v", e.LogLevel, e.Cause)
}

func NewLogLevelError(logLevel string, cause error) error {
	return &LogLevelError{LogLevel: logLevel, Cause: cause}
}

type UnsupportedLogLevelError struct {
	LogLevel string
}

func (e *UnsupportedLogLevelError) Error() string {
	return fmt.Sprintf("Unsupported log level: %s", e.LogLevel)
}

func NewUnsupportedLogLevelError(logLevel string) error {
	return &UnsupportedLogLevelError{LogLevel: logLevel}
}
