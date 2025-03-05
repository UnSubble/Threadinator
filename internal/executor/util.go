package executor

import (
	"sync"

	"github.com/unsubble/threadinator/internal/models"
)

func initializeWorkers(threadCount int, poolChan chan *Worker, wg *sync.WaitGroup, config *models.Config) {
	config.Logger.Infof("Initializing %d workers", threadCount)
	for i := range threadCount {
		worker := newWorker(i, wg, config)
		poolChan <- worker
	}
}

func finalizeExecution(wg *sync.WaitGroup, errorChan chan error, poolChan chan *Worker, config *models.Config) {
	wg.Wait()
	config.Logger.Debug("Execution completed, closing channels.")
	close(errorChan)
	close(poolChan)
}

func collectErrors(config *models.Config, errorChan <-chan error) error {
	for err := range errorChan {
		config.Logger.Errorf("%v", err)
	}
	return nil
}
