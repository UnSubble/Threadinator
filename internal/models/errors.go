package models

import "fmt"

type PipelineError struct {
	WorkerID int
	Err      error
}

func (p *PipelineError) Error() string {
	return fmt.Sprintf("pipeline error in [Thread-%d]: %v", p.WorkerID, p.Err)
}
