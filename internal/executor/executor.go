package executor

import (
	"sync"

	"github.com/unsubble/threadinator/internal/models"
)

func Execute(config *models.Config) error {
	config.Logger.Info("Starting execution process")
	var wg sync.WaitGroup
	executionOrder, err := resolveExecutionOrder(config)
	if err != nil {
		config.Logger.Errorf("Execution order resolution failed: %v", err)
		return err
	}

	errorChan := make(chan error, len(config.Commands))
	poolChan := make(chan *Worker, config.ThreadCount)

	initializeWorkers(config.ThreadCount, poolChan, &wg, config)
	scheduleCommands(config, executionOrder, poolChan, errorChan, &wg)

	go finalizeExecution(&wg, errorChan, poolChan, config)

	return collectErrors(config, errorChan)
}
