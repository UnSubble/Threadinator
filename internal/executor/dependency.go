package executor

import (
	"github.com/unsubble/threadinator/internal/models"
)

func resolveExecutionOrder(config *models.Config) ([]int, error) {
	config.Logger.Debug("Resolving execution order based on dependencies.")
	graph := make(map[int][]int)
	inDegree := make(map[int]int)

	commands := config.Commands

	for i, cmd := range commands {
		if cmd.Dependency != nil {
			depIdx := *cmd.Dependency
			if depIdx < 0 || depIdx >= len(commands) {
				return nil, models.NewDependencyError(depIdx, i)
			}
			graph[depIdx] = append(graph[depIdx], i)
			inDegree[i]++
			config.Logger.Debugf("Command %d depends on command %d", i, depIdx)
		}
	}

	config.Logger.Debug("Performing topological sort to determine execution order.")
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
		return nil, models.NewCircularDependencyError()
	}

	return order, nil
}
