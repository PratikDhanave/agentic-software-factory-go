package entities

import (
	"time"

	"github.com/google/uuid"
)

type KnowledgeNodeType string

const (
	NodeTypeDecision  KnowledgeNodeType = "decision"
	NodeTypePattern   KnowledgeNodeType = "pattern"
	NodeTypeExpertise KnowledgeNodeType = "expertise"
	NodeTypeOutcome   KnowledgeNodeType = "outcome"
	NodeTypeMetric    KnowledgeNodeType = "metric"
	NodeTypeLearning  KnowledgeNodeType = "learning"
)

type KnowledgeNode struct {
	ID        string
	Type      KnowledgeNodeType
	Agent     AgentRole
	Content   string
	Tags      []string
	Metadata  map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
	Score     float64
}

type EdgeRelation string

const (
	RelationInforms   EdgeRelation = "informs"
	RelationRefutes   EdgeRelation = "refutes"
	RelationExtends   EdgeRelation = "extends"
	RelationDependsOn EdgeRelation = "depends_on"
	RelationConfirms  EdgeRelation = "confirms"
	RelationLeadsTo   EdgeRelation = "leads_to"
)

type KnowledgeEdge struct {
	ID       string
	FromID   string
	ToID     string
	Relation EdgeRelation
	Strength float64
	Evidence string
	CreatedAt time.Time
}

type KnowledgeGraph struct {
	Nodes map[string]*KnowledgeNode
	Edges map[string]*KnowledgeEdge
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		Nodes: make(map[string]*KnowledgeNode),
		Edges: make(map[string]*KnowledgeEdge),
	}
}

func NewKnowledgeNode(nodeType KnowledgeNodeType, agent AgentRole, content string) *KnowledgeNode {
	return &KnowledgeNode{
		ID:        uuid.New().String(),
		Type:      nodeType,
		Agent:     agent,
		Content:   content,
		Tags:      []string{},
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Score:     0.5,
	}
}

func NewKnowledgeEdge(fromID, toID string, relation EdgeRelation, strength float64) *KnowledgeEdge {
	return &KnowledgeEdge{
		ID:        uuid.New().String(),
		FromID:    fromID,
		ToID:      toID,
		Relation:  relation,
		Strength:  strength,
		CreatedAt: time.Now(),
	}
}

func (kn *KnowledgeNode) AddTag(tag string) {
	for _, t := range kn.Tags {
		if t == tag {
			return
		}
	}
	kn.Tags = append(kn.Tags, tag)
}

func (kn *KnowledgeNode) SetMetadata(key string, value interface{}) {
	kn.Metadata[key] = value
	kn.UpdatedAt = time.Now()
}

func (kn *KnowledgeNode) UpdateScore(delta float64) {
	kn.Score += delta
	if kn.Score < 0 {
		kn.Score = 0
	}
	if kn.Score > 1 {
		kn.Score = 1
	}
	kn.UpdatedAt = time.Now()
}

func (kg *KnowledgeGraph) AddNode(node *KnowledgeNode) error {
	kg.Nodes[node.ID] = node
	return nil
}

func (kg *KnowledgeGraph) AddEdge(edge *KnowledgeEdge) error {
	if _, exists := kg.Nodes[edge.FromID]; !exists {
		return nil
	}
	if _, exists := kg.Nodes[edge.ToID]; !exists {
		return nil
	}
	kg.Edges[edge.ID] = edge
	return nil
}

func (kg *KnowledgeGraph) GetNode(id string) *KnowledgeNode {
	return kg.Nodes[id]
}

func (kg *KnowledgeGraph) GetEdgesFrom(nodeID string) []*KnowledgeEdge {
	edges := []*KnowledgeEdge{}
	for _, edge := range kg.Edges {
		if edge.FromID == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

func (kg *KnowledgeGraph) GetEdgesTo(nodeID string) []*KnowledgeEdge {
	edges := []*KnowledgeEdge{}
	for _, edge := range kg.Edges {
		if edge.ToID == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

func (kg *KnowledgeGraph) GetNodesByType(nodeType KnowledgeNodeType) []*KnowledgeNode {
	nodes := []*KnowledgeNode{}
	for _, node := range kg.Nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (kg *KnowledgeGraph) GetNodesByAgent(agent AgentRole) []*KnowledgeNode {
	nodes := []*KnowledgeNode{}
	for _, node := range kg.Nodes {
		if node.Agent == agent {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (kg *KnowledgeGraph) GetNodesByTag(tag string) []*KnowledgeNode {
	nodes := []*KnowledgeNode{}
	for _, node := range kg.Nodes {
		for _, t := range node.Tags {
			if t == tag {
				nodes = append(nodes, node)
				break
			}
		}
	}
	return nodes
}

func (kg *KnowledgeGraph) Size() int {
	return len(kg.Nodes)
}

func (kg *KnowledgeGraph) EdgeCount() int {
	return len(kg.Edges)
}
