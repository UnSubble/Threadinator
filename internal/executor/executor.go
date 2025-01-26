package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Command     string
	Args        []string
	ThreadCount int
	UsePipeline bool
	Verbose     bool
	Timeout     time.Duration
}

type Worker struct {
	id        int
	result    chan []string
	prev      *Worker
	command   string
	args      []string
	waitGroup *sync.WaitGroup
	pipeline  bool
	verbose   bool
	timeout   time.Duration
}

type PipelineError struct {
	WorkerID int
	Err      error
}

func (p *PipelineError) Error() string {
	return fmt.Sprintf("pipeline error in worker %d: %v", p.WorkerID, p.Err)
}

func Execute(config *Config) error {
	var wg sync.WaitGroup
	var prev *Worker
	errorChan := make(chan error, config.ThreadCount)

	for i := 0; i < config.ThreadCount; i++ {
		wg.Add(1)
		newWorker := newWorker(i, &wg, config)
		newWorker.prev = prev
		prev = newWorker

		go func(w *Worker) {
			defer func() {
				if r := recover(); r != nil {
					errorChan <- fmt.Errorf("panic in worker %d: %v", w.id, r)
				}
			}()

			if err := w.perform(); err != nil {
				errorChan <- &PipelineError{WorkerID: w.id, Err: err}
			}
		}(newWorker)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("execution errors: %v", errs)
	}

	return nil
}

func (w *Worker) perform() error {
	defer func() {
		close(w.result)
		w.waitGroup.Done()
	}()

	if w.pipeline && w.prev != nil {
		w.log("Waiting for result from previous worker...")
		prevResult, ok := <-w.prev.result
		if !ok || prevResult == nil {
			return fmt.Errorf("no input from previous worker (Thread-%d)", w.prev.id)
		}
		w.args = prevResult
	}

	w.log(fmt.Sprintf("Executing command: %s %v", w.command, w.args))

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.command, w.args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	trimmedOutput := strings.TrimSpace(string(output))
	w.logOutput(trimmedOutput)

	if w.pipeline {
		w.result <- []string{trimmedOutput}
	}

	return nil
}

func newWorker(id int, wg *sync.WaitGroup, config *Config) *Worker {
	return &Worker{
		id:        id,
		result:    make(chan []string, 1),
		command:   config.Command,
		args:      config.Args,
		waitGroup: wg,
		pipeline:  config.UsePipeline,
		verbose:   config.Verbose,
		timeout:   config.Timeout,
	}
}

func (w *Worker) log(message string) {
	if w.verbose {
		fmt.Printf("[Thread-%d] %s\n", w.id, message)
	}
}

func (w *Worker) logOutput(msg string) {
	fmt.Printf("[Thread-%d] Output: %s\n", w.id, msg)
}
