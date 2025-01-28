package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Command struct {
	Command string
	Args    []string
	Times   int
}

type Config struct {
	Commands    []Command
	ThreadCount int
	UsePipeline bool
	Verbose     bool
	Timeout     time.Duration
}

type Worker struct {
	id        int
	result    chan io.Reader
	prev      *Worker
	command   *Command
	waitGroup *sync.WaitGroup
	config    *Config
}

type PipelineError struct {
	WorkerID int
	Err      error
}

func (p *PipelineError) Error() string {
	return fmt.Sprintf("Pipeline error in [Thread-%d]: %v", p.WorkerID, p.Err)
}

func Execute(config *Config) error {
	var wg sync.WaitGroup
	var prevWorker *Worker
	errorChan := make(chan error, config.ThreadCount)

	for i := 0; i < config.ThreadCount; i++ {
		wg.Add(1)
		worker := newWorker(i, &wg, config, prevWorker)
		prevWorker = worker
		go func(w *Worker) {
			defer recoverFromPanic(w, errorChan)
			if err := w.perform(); err != nil {
				errorChan <- &PipelineError{WorkerID: w.id, Err: err}
			}
		}(worker)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	return collectErrors(errorChan)
}

func newWorker(id int, wg *sync.WaitGroup, config *Config, prev *Worker) *Worker {
	command := getCommand(id, config.Commands)

	return &Worker{
		id:        id,
		result:    make(chan io.Reader, 1),
		command:   command,
		waitGroup: wg,
		config:    config,
		prev:      prev,
	}
}

func getCommand(id int, commands []Command) *Command {
	for i := 0; i <= id; i++ {
		if i < len(commands) && commands[i].Times > 0 {
			commands[i].Times--
			return &commands[i]
		}
	}

	return &Command{}
}

func (w *Worker) perform() error {
	defer func() {
		close(w.result)
		w.waitGroup.Done()
	}()

	return w.executeCommand()
}

func (w *Worker) executeCommand() error {
	w.logVerbose(fmt.Sprintf("Executing command: %s %v", w.command.Command, w.command.Args))

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)

	defer cancel()

	cmd := exec.CommandContext(ctx, w.command.Command, w.command.Args...)
	cmd.Env = os.Environ()

	if w.config.UsePipeline && w.prev != nil {
		prevOut, ok := <-w.prev.result
		if ok {
			cmd.Stdin = prevOut
		} else {
			return fmt.Errorf("no valid input from previous worker (Thread-%d)", w.prev.id)
		}
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	trimmedOutput := strings.TrimSpace(string(output))
	w.logOutput(trimmedOutput)

	if w.config.UsePipeline {
		pipe := bytes.NewReader(output)
		w.result <- pipe
	}

	return nil
}

func recoverFromPanic(w *Worker, errorChan chan error) {
	if r := recover(); r != nil {
		errorChan <- fmt.Errorf("panic in worker %d: %v", w.id, r)
	}
}

func collectErrors(errorChan <-chan error) error {
	var errBuilder strings.Builder

	for err := range errorChan {
		errBuilder.WriteString("[ERROR] ")
		errBuilder.WriteString(err.Error())
		errBuilder.WriteRune('\n')
	}

	if errBuilder.Len() > 0 {
		return fmt.Errorf("%s", strings.TrimSpace(errBuilder.String()))
	}

	return nil
}

func (w *Worker) logVerbose(message string) {
	if w.config.Verbose {
		fmt.Printf("[Thread-%d] %s\n", w.id, message)
	}
}

func (w *Worker) logOutput(output string) {
	fmt.Printf("[Thread-%d] Output: %s\n", w.id, output)
}
