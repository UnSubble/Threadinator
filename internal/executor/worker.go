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
	result    io.Reader
	mu        sync.Mutex
	cond      *sync.Cond
	prev      *Worker
	command   *models.Command
	waitGroup *sync.WaitGroup
	config    *models.Config
}

func newWorker(id int, wg *sync.WaitGroup, config *models.Config) *Worker {
	config.Logger.Infof("Creating worker with ID: %d", id)
	w := &Worker{
		id:        id,
		waitGroup: wg,
		config:    config,
	}
	w.cond = sync.NewCond(&w.mu)
	return w
}

func (w *Worker) perform() error {
	defer w.waitGroup.Done()
	err := w.executeCommand()

	w.mu.Lock()
	w.cond.Broadcast()
	w.mu.Unlock()

	return err
}

func (w *Worker) executeCommand() error {
	w.logVerbose(fmt.Sprintf("Executing command: %s %v", w.command.Command, w.command.Args))

	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()

	if w.command.Delay != nil {
		if err := w.performDelay(ctx); err != nil {
			return err
		}
	}

	cmd := exec.CommandContext(ctx, w.command.Command, w.command.Args...)
	cmd.Env = os.Environ()

	if w.config.UsePipeline && w.prev != nil {
		prevOut, err := w.getStdout()
		if err != nil {
			return err
		}
		cmd.Stdin = prevOut
	}

	reader, err := cmd.StdoutPipe()

	if err != nil {
		return models.NewPipeError(err)
	}

	if err := cmd.Start(); err != nil {
		return models.NewCommandError(w.command.Command, err.Error())
	}

	buffer := make([]byte, 1024)
	l, err := reader.Read(buffer)

	if err != nil {
		return models.NewBufferError(err)
	}

	if w.config.UsePipeline {
		w.result = bytes.NewBuffer(buffer[0:l])
	}

	return processCommandOutput(ctx, bytes.NewBuffer(buffer[0:l]), w)
}

func (w *Worker) performDelay(ctx context.Context) error {
	if *w.command.Delay >= w.config.TimeoutInt {
		return models.NewTimeoutError(w.command.Command)
	}

	delay := *w.command.Delay
	timeUnit := w.config.TimeUnit

	select {
	case <-time.After(time.Duration(delay) * parsers.GetTimeUnit(timeUnit)):
		w.config.Logger.Infof("[Thread-%d] before sleeping for %d %s", w.id, delay, timeUnit)
	case <-ctx.Done():
		w.config.Logger.Errorf("Timeout exceeded for command: %s", w.command.Command)
		return models.NewTimeoutError(w.command.Command)
	}
	w.config.Logger.Infof("[Thread-%d] after sleeping for %d %s", w.id, delay, timeUnit)

	return nil
}

func processCommandOutput(ctx context.Context, reader io.Reader, w *Worker) error {
	buffer := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			w.config.Logger.Errorf("Timeout exceeded for command: %s", w.command.Command)
			return models.NewTimeoutError(w.command.Command)
		default:
			c, err := reader.Read(buffer)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				w.config.Logger.Errorf("Error reading output: %v", err)
				return models.NewOutputReadError(err)
			}
			w.logOutput(string(buffer[:c]))
		}
	}
}

func recoverFromPanic(w *Worker, errorChan chan error) {
	if r := recover(); r != nil {
		errorChan <- models.NewPanicError(w.id, r)
		w.config.Logger.Errorf("Recovered from panic in Thread-%d: %v", w.id, r)
	}
}

func (w *Worker) getStdout() (io.Reader, error) {
	w.prev.mu.Lock()
	defer w.prev.mu.Unlock()

	for w.prev.result == nil {
		w.prev.cond.Wait()
	}

	var buf bytes.Buffer
	_, err := io.Copy(&buf, w.prev.result)

	w.prev.result = bytes.NewBuffer(buf.Bytes())

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func (w *Worker) logVerbose(message string) {
	if w.config.Verbose {
		w.config.Logger.Debugf("[Thread-%d] %s", w.id, message)
	}
}

func (w *Worker) logOutput(output string) {
	if !w.config.Verbose {
		fmt.Printf("[Thread-%d] Output: %s", w.id, output)
	} else {
		w.config.Logger.Debugf("[Thread-%d] Output: %s", w.id, output)
	}
}
