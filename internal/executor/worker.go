package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/unsubble/threadinator/internal/models"
	"github.com/unsubble/threadinator/internal/parsers"
)

type Worker struct {
	id        int
	result    chan io.Reader
	prev      *Worker
	command   *models.Command
	waitGroup *sync.WaitGroup
	config    *models.Config
}

func newWorker(id int, wg *sync.WaitGroup, prevWorker *Worker, config *models.Config) *Worker {
	config.Logger.Infof("Creating worker with ID: %d", id)
	return &Worker{
		id:        id,
		waitGroup: wg,
		prev:      prevWorker,
		config:    config,
	}
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

	if w.command.Delay != nil {
		if *w.command.Delay >= w.config.TimeoutInt {
			return fmt.Errorf("timeout exceeded")
		}
		delay := *w.command.Delay
		timeUnit := w.config.TimeUnit
		select {
		case <-time.After(time.Duration(delay) * parsers.GetTimeUnit(timeUnit)):
			w.config.Logger.Infof("[Thread-%d] before sleeping for %d %s", w.id, delay, timeUnit)
		case <-ctx.Done():
			w.config.Logger.Errorf("Timeout exceeded for command: %s", w.command.Command)
			return fmt.Errorf("timeout exceeded for command %s", w.command.Command)
		}
		w.config.Logger.Infof("[Thread-%d] after sleeping for %d %s", w.id, delay, timeUnit)
	}

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

	reader, err := cmd.StdoutPipe()

	if err != nil {
		return fmt.Errorf("pipe error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	buffer := make([]byte, 1024)
	l, err := reader.Read(buffer)

	if err != nil {
		return fmt.Errorf("buffer error: %v", err)
	}

	if w.config.UsePipeline {
		w.result <- bytes.NewBuffer(buffer[0:l])
	}

	return processCommandOutput(ctx, bytes.NewBuffer(buffer[0:l]), w)
}

func processCommandOutput(ctx context.Context, reader io.Reader, w *Worker) error {
	buffer := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			w.config.Logger.Errorf("Timeout exceeded for command: %s", w.command.Command)
			return fmt.Errorf("timeout exceeded for command %s", w.command.Command)
		default:
			c, err := reader.Read(buffer)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				w.config.Logger.Errorf("Error reading output: %v", err)
				return fmt.Errorf("error reading output: %v", err)
			}
			w.logOutput(string(buffer[:c]))
		}
	}
}

func recoverFromPanic(w *Worker, errorChan chan error) {
	if r := recover(); r != nil {
		errorChan <- fmt.Errorf("panic in Thread-%d: %v", w.id, r)
		w.config.Logger.Errorf("Recovered from panic in Thread-%d: %v", w.id, r)
	}
}

func (w *Worker) logVerbose(message string) {
	if w.config.Verbose {
		w.config.Logger.Debugf("[Thread-%d] %s", w.id, message)
	}
}

func (w *Worker) logOutput(output string) {
	if w.config.Verbose {
		w.config.Logger.Debugf("[Thread-%d] Output: %s", w.id, output)
	} else {
		fmt.Printf("[Thread-%d] Output: %s", w.id, output)
	}
}
