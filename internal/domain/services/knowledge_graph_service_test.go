package services

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

func TestKnowledgeGraphService_RecordDecision(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision := &entities.Decision{
		AgentRole:  entities.ProductAgent,
		Choice:     "high-priority",
		Reasoning:  "Critical issue affecting users",
		Confidence: 0.95,
		Alternatives: []string{"medium-priority", "low-priority"},
	}

	node := kgs.RecordDecision(entities.ProductAgent, decision)

	if node == nil {
		t.Fatal("RecordDecision should return a node")
	}

	if node.Type != entities.NodeTypeDecision {
		t.Errorf("Expected node type Decision, got %s", node.Type)
	}

	if node.Score != 0.95 {
		t.Errorf("Expected score 0.95, got %f", node.Score)
	}

	if kg.Size() != 1 {
		t.Errorf("Expected graph size 1, got %d", kg.Size())
	}
}

func TestKnowledgeGraphService_RecordOutcome(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision := &entities.Decision{
		AgentRole:  entities.DeveloperAgent,
		Choice:     "implement-feature",
		Confidence: 0.85,
	}

	decisionNode := kgs.RecordDecision(entities.DeveloperAgent, decision)
	outcomeNode := kgs.RecordOutcome(decisionNode.ID, true, "feature implemented successfully")

	if outcomeNode == nil {
		t.Fatal("RecordOutcome should return a node")
	}

	if outcomeNode.Type != entities.NodeTypeOutcome {
		t.Errorf("Expected node type Outcome, got %s", outcomeNode.Type)
	}

	if kg.Size() != 2 {
		t.Errorf("Expected graph size 2, got %d", kg.Size())
	}

	if kg.EdgeCount() != 1 {
		t.Errorf("Expected 1 edge, got %d", kg.EdgeCount())
	}
}

func TestKnowledgeGraphService_QueryRelevant(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision1 := &entities.Decision{
		AgentRole:  entities.SecurityAgent,
		Choice:     "intensive-testing",
		Confidence: 0.92,
	}

	decision2 := &entities.Decision{
		AgentRole:  entities.QAAgent,
		Choice:     "standard-testing",
		Confidence: 0.78,
	}

	node1 := kgs.RecordDecision(entities.SecurityAgent, decision1)
	node2 := kgs.RecordDecision(entities.QAAgent, decision2)

	kgs.RecordPattern(entities.SecurityAgent, "authentication-bug-pattern", 0.88)
	kgs.RecordExpertise(entities.SecurityAgent, "authentication", 0.95)

	query := &RelevanceQuery{
		Agent: entities.SecurityAgent,
		Limit: 10,
	}

	results := kgs.QueryRelevant(query)

	if len(results) == 0 {
		t.Fatal("QueryRelevant should return results")
	}

	if results[0].Node.ID != node1.ID && results[0].Node.ID != node2.ID {
		for i, r := range results {
			t.Logf("Result %d: ID=%s, Agent=%s", i, r.Node.ID, r.Node.Agent)
		}
	}
}

func TestKnowledgeGraphService_ConnectRelatedNodes(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision := &entities.Decision{
		AgentRole:  entities.ArchitectAgent,
		Choice:     "microservices-architecture",
		Confidence: 0.80,
	}

	node1 := kgs.RecordDecision(entities.ArchitectAgent, decision)
	node2 := kgs.RecordPattern(entities.ArchitectAgent, "scalability-concern", 0.75)

	err := kgs.ConnectRelatedNodes(node1.ID, node2.ID, entities.RelationInforms, 0.85, "architecture decision informs pattern")

	if err != nil {
		t.Errorf("ConnectRelatedNodes failed: %v", err)
	}

	if kg.EdgeCount() != 1 {
		t.Errorf("Expected 1 edge, got %d", kg.EdgeCount())
	}
}

func TestKnowledgeGraphService_GetNodeContext(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision := &entities.Decision{
		AgentRole:  entities.DevOpsAgent,
		Choice:     "kubernetes-deployment",
		Confidence: 0.90,
	}

	decisionNode := kgs.RecordDecision(entities.DevOpsAgent, decision)
	kgs.RecordOutcome(decisionNode.ID, true, "deployment successful")
	kgs.RecordOutcome(decisionNode.ID, false, "monitoring alert triggered")

	context := kgs.GetNodeContext(decisionNode.ID, 1)

	if context == nil {
		t.Fatal("GetNodeContext should return context")
	}

	node := context["node"].(*entities.KnowledgeNode)
	if node.ID != decisionNode.ID {
		t.Errorf("Context node ID mismatch")
	}

	outbound := context["outbound"].([]*entities.KnowledgeEdge)
	if len(outbound) != 2 {
		t.Errorf("Expected 2 outbound edges, got %d", len(outbound))
	}
}

func TestKnowledgeGraphService_LearnFromExecution(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	execution := &entities.WorkflowExecution{
		ID: "exec-1",
	}

	decisions := map[string]*entities.Decision{
		string(entities.DeveloperAgent): {
			AgentRole:  entities.DeveloperAgent,
			Choice:     "implement",
			Confidence: 0.85,
		},
		string(entities.QAAgent): {
			AgentRole:  entities.QAAgent,
			Choice:     "test",
			Confidence: 0.80,
		},
	}

	outcomes := map[string]bool{
		"job_implement": true,
		"job_test":      true,
	}

	err := kgs.LearnFromExecution(execution, decisions, outcomes)

	if err != nil {
		t.Errorf("LearnFromExecution failed: %v", err)
	}

	if kg.Size() < 2 {
		t.Errorf("Expected at least 2 nodes after learning, got %d", kg.Size())
	}
}

func TestKnowledgeGraphService_GetStats(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	decision := &entities.Decision{
		AgentRole:  entities.ProductAgent,
		Choice:     "priority-high",
		Confidence: 0.92,
	}

	kgs.RecordDecision(entities.ProductAgent, decision)
	kgs.RecordPattern(entities.ProductAgent, "critical-issue", 0.88)
	kgs.RecordExpertise(entities.ProductAgent, "roadmap", 0.95)

	stats := kgs.GetStats()

	if stats["total_nodes"] != 3 {
		t.Errorf("Expected 3 nodes in stats, got %v", stats["total_nodes"])
	}

	avgConf := stats["avg_confidence"].(float64)
	if avgConf < 0.8 || avgConf > 1.0 {
		t.Errorf("Average confidence out of range: %f", avgConf)
	}
}

func TestKnowledgeGraphService_NodeScoreUpdates(t *testing.T) {
	node := entities.NewKnowledgeNode(entities.NodeTypePattern, entities.SecurityAgent, "sql-injection-pattern")
	node.Score = 0.5

	node.UpdateScore(0.3)
	if node.Score != 0.8 {
		t.Errorf("Expected score 0.8 after +0.3, got %f", node.Score)
	}

	node.UpdateScore(0.5)
	if node.Score != 1.0 {
		t.Errorf("Expected score 1.0 (capped), got %f", node.Score)
	}

	node.UpdateScore(-1.5)
	if node.Score != 0.0 {
		t.Errorf("Expected score 0.0 (capped), got %f", node.Score)
	}
}

func TestKnowledgeGraphService_NodeTags(t *testing.T) {
	node := entities.NewKnowledgeNode(entities.NodeTypeDecision, entities.ArchitectAgent, "microservices")

	node.AddTag("scalability")
	node.AddTag("resilience")
	node.AddTag("scalability")

	if len(node.Tags) != 2 {
		t.Errorf("Expected 2 unique tags, got %d", len(node.Tags))
	}
}

func TestKnowledgeGraphService_QueryByTags(t *testing.T) {
	kg := entities.NewKnowledgeGraph()
	kgs := NewKnowledgeGraphService(kg, &MockLogger{})

	node1 := entities.NewKnowledgeNode(entities.NodeTypePattern, entities.SecurityAgent, "pattern1")
	node1.AddTag("authentication")
	node1.AddTag("security")
	kg.AddNode(node1)

	node2 := entities.NewKnowledgeNode(entities.NodeTypePattern, entities.DesignAgent, "pattern2")
	node2.AddTag("scalability")
	kg.AddNode(node2)

	results := kgs.QueryRelevant(&RelevanceQuery{
		Tags:  []string{"authentication"},
		Limit: 10,
	})

	if len(results) == 0 {
		t.Fatal("Should find node with authentication tag")
	}

	if results[0].Node.ID != node1.ID {
		t.Errorf("Expected to find node1, got %s", results[0].Node.ID)
	}
}
