package services

import (
	"fmt"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type DAGBuilder struct {
	logger Logger
}

func NewDAGBuilder(logger Logger) *DAGBuilder {
	return &DAGBuilder{
		logger: logger,
	}
}

// Build creates a DAG from workflow definition and returns execution levels
func (b *DAGBuilder) Build(def *entities.WorkflowDefinition) (*DirectedAcyclicGraph, error) {
	if def == nil {
		return nil, fmt.Errorf("workflow definition cannot be nil")
	}

	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	dag := NewDirectedAcyclicGraph()

	jobIDs := make([]string, 0, len(def.JobsMap))
	for jobID, jobDef := range def.JobsMap {
		jobIDs = append(jobIDs, jobID)

		if err := dag.AddNode(jobID, jobDef.Task, jobDef.DependsOn); err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", jobID, err)
		}
	}

	// Detect cycles
	if err := b.detectCycles(dag); err != nil {
		return nil, fmt.Errorf("cycle detected in DAG: %w", err)
	}

	b.logger.Debug("DAG built", "nodes", len(dag.Nodes), "workflow", def.Name)

	// Compute topological sort and execution levels
	levels, err := b.topologicalSort(dag)
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}

	dag.SetExecutionLevels(levels)

	b.logger.Info("DAG execution levels computed",
		"workflow", def.Name,
		"nodes", len(dag.Nodes),
		"levels", len(levels),
	)

	return dag, nil
}

// detectCycles uses DFS with 3-color marking to detect cycles
// White = unvisited, Gray = visiting, Black = visited
func (b *DAGBuilder) detectCycles(dag *DirectedAcyclicGraph) error {
	colors := make(map[string]string)

	// Initialize all nodes as white
	for jobID := range dag.Nodes {
		colors[jobID] = "white"
	}

	// Visit each node
	for jobID := range dag.Nodes {
		if colors[jobID] == "white" {
			if err := b.dfsVisit(jobID, dag, colors); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *DAGBuilder) dfsVisit(jobID string, dag *DirectedAcyclicGraph, colors map[string]string) error {
	colors[jobID] = "gray"

	// Check all dependents (nodes that depend on this one)
	dependents := dag.GetDependents(jobID)
	for _, dependent := range dependents {
		if colors[dependent] == "gray" {
			return fmt.Errorf("cycle detected: %s -> %s", jobID, dependent)
		}
		if colors[dependent] == "white" {
			if err := b.dfsVisit(dependent, dag, colors); err != nil {
				return err
			}
		}
	}

	colors[jobID] = "black"
	return nil
}

// topologicalSort computes execution levels using modified Kahn's algorithm
// Returns levels where level[i] contains all jobs that can execute after level[i-1]
func (b *DAGBuilder) topologicalSort(dag *DirectedAcyclicGraph) ([][]string, error) {
	// Compute in-degree for each node
	inDegree := make(map[string]int)
	for jobID := range dag.Nodes {
		inDegree[jobID] = 0
	}

	// Count in-degrees (nodes that this job depends on)
	for _, node := range dag.Nodes {
		for range node.DependsOn {
			inDegree[node.JobID]++
		}
	}

	// Initialize queue with nodes that have 0 in-degree
	queue := make([]string, 0)
	for jobID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, jobID)
		}
	}

	levels := make([][]string, 0)

	for len(queue) > 0 {
		// Current level contains all jobs with 0 in-degree
		level := make([]string, len(queue))
		copy(level, queue)
		levels = append(levels, level)

		nextQueue := make([]string, 0)

		// For each job in current level, check dependents
		for _, jobID := range queue {
			// Find all jobs that depend on this job
			for potentialDependent := range dag.Nodes {
				potentialNode := dag.GetNode(potentialDependent)

				// Check if potentialDependent depends on jobID
				dependsOnCurrent := false
				for _, dep := range potentialNode.DependsOn {
					if dep == jobID {
						dependsOnCurrent = true
						break
					}
				}

				if dependsOnCurrent {
					inDegree[potentialDependent]--
					if inDegree[potentialDependent] == 0 {
						// Avoid duplicates in nextQueue
						found := false
						for _, queued := range nextQueue {
							if queued == potentialDependent {
								found = true
								break
							}
						}
						if !found {
							nextQueue = append(nextQueue, potentialDependent)
						}
					}
				}
			}
		}

		queue = nextQueue
	}

	// Verify all nodes were visited (no cycles)
	totalVisited := 0
	for _, level := range levels {
		totalVisited += len(level)
	}

	if totalVisited != len(dag.Nodes) {
		return nil, fmt.Errorf("not all nodes visited during topological sort; cycle may exist")
	}

	return levels, nil
}
