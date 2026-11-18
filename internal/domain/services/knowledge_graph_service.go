package services

import (
	"sort"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type KnowledgeGraphService struct {
	graph  *entities.KnowledgeGraph
	logger Logger
}

type RelevanceScore struct {
	Node      *entities.KnowledgeNode
	Score     float64
	Relations int
}

type RelevanceQuery struct {
	Agent    entities.AgentRole
	Tags     []string
	NodeType entities.KnowledgeNodeType
	Limit    int
}

func NewKnowledgeGraphService(graph *entities.KnowledgeGraph, logger Logger) *KnowledgeGraphService {
	if graph == nil {
		graph = entities.NewKnowledgeGraph()
	}
	return &KnowledgeGraphService{
		graph:  graph,
		logger: logger,
	}
}

func (kgs *KnowledgeGraphService) RecordDecision(agent entities.AgentRole, decision *entities.Decision) *entities.KnowledgeNode {
	node := entities.NewKnowledgeNode(
		entities.NodeTypeDecision,
		agent,
		decision.Choice,
	)
	node.SetMetadata("confidence", decision.Confidence)
	node.SetMetadata("reasoning", decision.Reasoning)
	node.SetMetadata("alternatives", decision.Alternatives)
	node.Score = decision.Confidence

	for _, alt := range decision.Alternatives {
		node.AddTag(alt)
	}

	kgs.graph.AddNode(node)
	kgs.logger.Debug("decision recorded", "agent", agent, "node_id", node.ID)

	return node
}

func (kgs *KnowledgeGraphService) RecordOutcome(decisionNodeID string, success bool, details string) *entities.KnowledgeNode {
	outcomeNode := entities.NewKnowledgeNode(
		entities.NodeTypeOutcome,
		"",
		details,
	)
	outcomeNode.SetMetadata("success", success)

	kgs.graph.AddNode(outcomeNode)

	relation := entities.RelationLeadsTo
	strength := 1.0
	if !success {
		relation = entities.RelationRefutes
		strength = 0.5
	}

	edge := entities.NewKnowledgeEdge(decisionNodeID, outcomeNode.ID, relation, strength)
	kgs.graph.AddEdge(edge)

	kgs.logger.Debug("outcome recorded",
		"decision_id", decisionNodeID,
		"outcome_id", outcomeNode.ID,
		"success", success,
	)

	return outcomeNode
}

func (kgs *KnowledgeGraphService) RecordPattern(agent entities.AgentRole, pattern string, confidence float64) *entities.KnowledgeNode {
	node := entities.NewKnowledgeNode(
		entities.NodeTypePattern,
		agent,
		pattern,
	)
	node.Score = confidence

	kgs.graph.AddNode(node)
	kgs.logger.Debug("pattern recorded", "agent", agent, "pattern", pattern, "confidence", confidence)

	return node
}

func (kgs *KnowledgeGraphService) RecordExpertise(agent entities.AgentRole, domain string, proficiency float64) *entities.KnowledgeNode {
	node := entities.NewKnowledgeNode(
		entities.NodeTypeExpertise,
		agent,
		domain,
	)
	node.Score = proficiency
	node.AddTag(domain)

	kgs.graph.AddNode(node)
	kgs.logger.Debug("expertise recorded", "agent", agent, "domain", domain, "proficiency", proficiency)

	return node
}

func (kgs *KnowledgeGraphService) QueryRelevant(query *RelevanceQuery) []*RelevanceScore {
	candidates := []*RelevanceScore{}

	for _, node := range kgs.graph.Nodes {
		if query.Agent != "" && node.Agent != query.Agent {
			continue
		}

		if query.NodeType != "" && node.Type != query.NodeType {
			continue
		}

		if len(query.Tags) > 0 && !kgs.hasAnyTag(node, query.Tags) {
			continue
		}

		score := kgs.calculateRelevanceScore(node)
		candidates = append(candidates, score)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if query.Limit > 0 && len(candidates) > query.Limit {
		candidates = candidates[:query.Limit]
	}

	return candidates
}

func (kgs *KnowledgeGraphService) calculateRelevanceScore(node *entities.KnowledgeNode) *RelevanceScore {
	score := node.Score * 0.6

	inEdges := len(kgs.graph.GetEdgesTo(node.ID))
	outEdges := len(kgs.graph.GetEdgesFrom(node.ID))
	totalEdges := inEdges + outEdges

	if inEdges > 0 {
		score += float64(inEdges) * 0.1
	}

	if outEdges > 0 {
		score += float64(outEdges) * 0.1
	}

	recency := 1.0 - (node.UpdatedAt.Sub(time.Now()).Hours() / 24 / 30)
	if recency < 0 {
		recency = 0
	}
	score += recency * 0.2

	if score > 1.0 {
		score = 1.0
	}

	return &RelevanceScore{
		Node:      node,
		Score:     score,
		Relations: totalEdges,
	}
}

func (kgs *KnowledgeGraphService) hasAnyTag(node *entities.KnowledgeNode, tags []string) bool {
	for _, tag := range tags {
		for _, nodeTag := range node.Tags {
			if tag == nodeTag {
				return true
			}
		}
	}
	return false
}

func (kgs *KnowledgeGraphService) LearnFromExecution(execution *entities.WorkflowExecution, decisions map[string]*entities.Decision, outcomes map[string]bool) error {
	decisionNodeMap := make(map[string]string)

	for agentRole, decision := range decisions {
		decisionNode := kgs.RecordDecision(entities.AgentRole(agentRole), decision)
		decisionNodeMap[agentRole] = decisionNode.ID
	}

	for jobID, success := range outcomes {
		for _, decisionNodeID := range decisionNodeMap {
			kgs.RecordOutcome(decisionNodeID, success, "job:"+jobID)
		}
	}

	kgs.logger.Info("learning from execution",
		"execution_id", execution.ID,
		"decisions_recorded", len(decisions),
		"outcomes_recorded", len(outcomes),
	)

	return nil
}

func (kgs *KnowledgeGraphService) ConnectRelatedNodes(fromID, toID string, relation entities.EdgeRelation, strength float64, evidence string) error {
	if _, exists := kgs.graph.Nodes[fromID]; !exists {
		kgs.logger.Error("source node not found", nil, "node_id", fromID)
		return nil
	}

	if _, exists := kgs.graph.Nodes[toID]; !exists {
		kgs.logger.Error("target node not found", nil, "node_id", toID)
		return nil
	}

	edge := entities.NewKnowledgeEdge(fromID, toID, relation, strength)
	edge.Evidence = evidence

	kgs.graph.AddEdge(edge)
	kgs.logger.Debug("nodes connected", "from", fromID, "to", toID, "relation", relation)

	return nil
}

func (kgs *KnowledgeGraphService) GetNodeContext(nodeID string, depth int) map[string]interface{} {
	node := kgs.graph.GetNode(nodeID)
	if node == nil {
		return nil
	}

	context := map[string]interface{}{
		"node":        node,
		"inbound":     []*entities.KnowledgeEdge{},
		"outbound":    []*entities.KnowledgeEdge{},
		"related":     []*entities.KnowledgeNode{},
	}

	inEdges := kgs.graph.GetEdgesTo(nodeID)
	outEdges := kgs.graph.GetEdgesFrom(nodeID)

	context["inbound"] = inEdges
	context["outbound"] = outEdges

	related := []*entities.KnowledgeNode{}
	seen := make(map[string]bool)
	seen[nodeID] = true

	for _, edge := range inEdges {
		if fromNode := kgs.graph.GetNode(edge.FromID); fromNode != nil && !seen[fromNode.ID] {
			related = append(related, fromNode)
			seen[fromNode.ID] = true
		}
	}

	for _, edge := range outEdges {
		if toNode := kgs.graph.GetNode(edge.ToID); toNode != nil && !seen[toNode.ID] {
			related = append(related, toNode)
			seen[toNode.ID] = true
		}
	}

	context["related"] = related

	return context
}

func (kgs *KnowledgeGraphService) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_nodes":          kgs.graph.Size(),
		"total_edges":          kgs.graph.EdgeCount(),
		"nodes_by_type":        map[string]int{},
		"avg_confidence":       kgs.calculateAvgConfidence(),
		"most_connected_nodes": kgs.getMostConnectedNodes(5),
	}

	typeCounts := make(map[string]int)
	for _, node := range kgs.graph.Nodes {
		typeCounts[string(node.Type)]++
	}
	stats["nodes_by_type"] = typeCounts

	return stats
}

func (kgs *KnowledgeGraphService) calculateAvgConfidence() float64 {
	if len(kgs.graph.Nodes) == 0 {
		return 0
	}

	total := 0.0
	for _, node := range kgs.graph.Nodes {
		total += node.Score
	}

	return total / float64(len(kgs.graph.Nodes))
}

func (kgs *KnowledgeGraphService) getMostConnectedNodes(limit int) []*RelevanceScore {
	scores := []*RelevanceScore{}

	for nodeID := range kgs.graph.Nodes {
		node := kgs.graph.GetNode(nodeID)
		inCount := len(kgs.graph.GetEdgesTo(nodeID))
		outCount := len(kgs.graph.GetEdgesFrom(nodeID))

		scores = append(scores, &RelevanceScore{
			Node:      node,
			Score:     float64(inCount + outCount),
			Relations: inCount + outCount,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	if len(scores) > limit {
		scores = scores[:limit]
	}

	return scores
}
