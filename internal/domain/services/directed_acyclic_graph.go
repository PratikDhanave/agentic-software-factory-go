package services

import (
	"fmt"
)

// DirectedAcyclicGraph represents a DAG with nodes and execution levels
type DirectedAcyclicGraph struct {
	Nodes map[string]*DAGNodeRef
	Edges map[string][]string // jobID -> list of dependent job IDs
	Levels [][]string          // execution levels from topological sort
}

type DAGNodeRef struct {
	JobID     string
	TaskName  string
	DependsOn []string
}

// NewDirectedAcyclicGraph creates an empty DAG
func NewDirectedAcyclicGraph() *DirectedAcyclicGraph {
	return &DirectedAcyclicGraph{
		Nodes: make(map[string]*DAGNodeRef),
		Edges: make(map[string][]string),
		Levels: make([][]string, 0),
	}
}

// AddNode adds a node to the DAG
func (dag *DirectedAcyclicGraph) AddNode(jobID, taskName string, dependsOn []string) error {
	if _, exists := dag.Nodes[jobID]; exists {
		return fmt.Errorf("node %s already exists", jobID)
	}
	dag.Nodes[jobID] = &DAGNodeRef{
		JobID:     jobID,
		TaskName:  taskName,
		DependsOn: dependsOn,
	}
	dag.Edges[jobID] = make([]string, 0)
	return nil
}

// AddEdge adds a directed edge from jobID to dependent
func (dag *DirectedAcyclicGraph) AddEdge(from, to string) error {
	if _, exists := dag.Nodes[from]; !exists {
		return fmt.Errorf("from node %s does not exist", from)
	}
	if _, exists := dag.Nodes[to]; !exists {
		return fmt.Errorf("to node %s does not exist", to)
	}
	dag.Edges[from] = append(dag.Edges[from], to)
	return nil
}

// GetDependents returns all jobs that depend on the given job
func (dag *DirectedAcyclicGraph) GetDependents(jobID string) []string {
	return dag.Edges[jobID]
}

// GetDependencies returns all jobs this job depends on
func (dag *DirectedAcyclicGraph) GetDependencies(jobID string) []string {
	if node, exists := dag.Nodes[jobID]; exists {
		return node.DependsOn
	}
	return []string{}
}

// GetExecutionLevels returns the execution levels (topologically sorted)
func (dag *DirectedAcyclicGraph) GetExecutionLevels() [][]string {
	return dag.Levels
}

// SetExecutionLevels sets the execution levels (called by builder after topological sort)
func (dag *DirectedAcyclicGraph) SetExecutionLevels(levels [][]string) {
	dag.Levels = levels
}

// GetNode returns the node reference
func (dag *DirectedAcyclicGraph) GetNode(jobID string) *DAGNodeRef {
	return dag.Nodes[jobID]
}

// GetAllNodes returns all nodes
func (dag *DirectedAcyclicGraph) GetAllNodes() map[string]*DAGNodeRef {
	return dag.Nodes
}

// Size returns the number of nodes
func (dag *DirectedAcyclicGraph) Size() int {
	return len(dag.Nodes)
}

// IsEmpty returns true if DAG has no nodes
func (dag *DirectedAcyclicGraph) IsEmpty() bool {
	return len(dag.Nodes) == 0
}
