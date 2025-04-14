package executor

import (
	"sync"

	"github.com/unsubble/threadinator/internal/models"
)

func scheduleCommands(config *models.Config, executionOrder []int, poolChan chan *Worker, errorChan chan error, wg *sync.WaitGroup) {
	history := make(map[int]*Worker)
	for _, cmdIdx := range executionOrder {
		wg.Add(1)
		config.Logger.Debugf("Scheduling command with index %d: %s %v", cmdIdx, config.Commands[cmdIdx].Command, config.Commands[cmdIdx].Args)
		w := <-poolChan

		history[cmdIdx] = w
		command := config.Commands[cmdIdx]
		if command.Dependency != nil {
			w.prev = history[*command.Dependency]
		}

		go executeWorkerCommand(command, w, poolChan, errorChan)
	}
}

func executeWorkerCommand(command *models.Command, w *Worker, poolChan chan *Worker, errorChan chan error) {
	w.command = command

	defer func() {
		recoverFromPanic(w, errorChan)
		poolChan <- w
	}()

	w.config.Logger.Infof("[Thread-%d] Executing command: %s %v", w.id, w.command.Command, w.command.Args)

	if err := w.perform(); err != nil {
		errorChan <- err
	}
}
