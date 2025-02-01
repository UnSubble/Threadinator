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
	Command    string
	Args       []string
	Times      int
	Dependency *int
}

type Config struct {
	Commands    []*Command
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
	executionOrder, err := resolveExecutionOrder(config.Commands)
	if err != nil {
		return err
	}

	errorChan := make(chan error, len(config.Commands))
	poolChan := make(chan *Worker, config.ThreadCount)

	initializeWorkers(config.ThreadCount, poolChan, &wg, config)

	scheduleCommands(config, executionOrder, poolChan, errorChan, &wg)

	go finalizeExecution(&wg, errorChan, poolChan)

	return collectErrors(errorChan)
}

func initializeWorkers(threadCount int, poolChan chan *Worker, wg *sync.WaitGroup, config *Config) {
	var prevWorker *Worker = nil
	for i := 0; i < threadCount; i++ {
		worker := newWorker(i, wg, prevWorker, config)
		poolChan <- worker
		prevWorker = worker
	}
}

func scheduleCommands(config *Config, executionOrder []int, poolChan chan *Worker, errorChan chan error, wg *sync.WaitGroup) {
	for _, cmdIdx := range executionOrder {
		wg.Add(1)
		go executeWorkerCommand(config.Commands[cmdIdx], poolChan, errorChan)
	}
}

func executeWorkerCommand(command *Command, poolChan chan *Worker, errorChan chan error) {
	w := <-poolChan
	w.command = command
	w.result = make(chan io.Reader, 1)

	defer func() {
		recoverFromPanic(w, errorChan)
		poolChan <- w
	}()

	if err := w.perform(); err != nil {
		errorChan <- &PipelineError{WorkerID: w.id, Err: err}
	}
}

func finalizeExecution(wg *sync.WaitGroup, errorChan chan error, poolChan chan *Worker) {
	wg.Wait()
	close(errorChan)
	close(poolChan)
}

func newWorker(id int, wg *sync.WaitGroup, prevWorker *Worker, config *Config) *Worker {
	return &Worker{
		id:        id,
		waitGroup: wg,
		prev:      prevWorker,
		config:    config,
	}
}

func resolveExecutionOrder(commands []*Command) ([]int, error) {
	graph := make(map[int][]int)
	inDegree := make(map[int]int)

	for i, cmd := range commands {
		if cmd.Dependency != nil {
			depIdx := *cmd.Dependency
			if depIdx < 0 || depIdx >= len(commands) {
				return nil, fmt.Errorf("invalid dependency index %d for command %d", depIdx, i)
			}
			graph[depIdx] = append(graph[depIdx], i)
			inDegree[i]++
		}
	}

	return topologicalSort(graph, inDegree, len(commands))
}

func topologicalSort(graph map[int][]int, inDegree map[int]int, totalCommands int) ([]int, error) {
	var order []int
	var queue []int

	for i := 0; i < totalCommands; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		for _, neighbor := range graph[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(order) != totalCommands {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return order, nil
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
			return fmt.Errorf("timeout exceeded for command %s", w.command.Command)
		default:
			c, err := reader.Read(buffer)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("error reading output: %v", err)
			}
			w.logOutput(string(buffer[:c]))
		}
	}
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
