package executor

import (
	"io"
	"sync"

	"github.com/unsubble/threadinator/internal/models"
)

func scheduleCommands(config *models.Config, executionOrder []int, poolChan chan *Worker, errorChan chan error, wg *sync.WaitGroup) {
	for _, cmdIdx := range executionOrder {
		wg.Add(1)
		config.Logger.Debugf("Scheduling command with index %d: %s %v", cmdIdx, config.Commands[cmdIdx].Command, config.Commands[cmdIdx].Args)
		go executeWorkerCommand(config.Commands[cmdIdx], poolChan, errorChan)
	}
}

func executeWorkerCommand(command *models.Command, poolChan chan *Worker, errorChan chan error) {
	w := <-poolChan
	w.command = command
	w.result = make(chan io.Reader, 1)

	defer func() {
		recoverFromPanic(w, errorChan)
		poolChan <- w
	}()

	w.config.Logger.Infof("[Thread-%d] Executing command: %s %v", w.id, w.command.Command, w.command.Args)

	if err := w.perform(); err != nil {
		errorChan <- models.NewPipelineError(w.id)
	}
}
