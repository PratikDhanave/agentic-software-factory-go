// Package state provides an explicit, zero-dependency finite state machine
// engine shared across domain lifecycles, CLI tools, UI visualization, and MCP agents.
package state

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type State string

type Machine struct {
	name              string
	category          string
	description       string
	initial           State
	transitions       map[State][]State
	stateDescriptions map[State]string
	states            []State
}

func NewMachine(name string, initial State, transitions map[State][]State) *Machine {
	seen := map[State]bool{initial: true}
	for from, tos := range transitions {
		seen[from] = true
		for _, to := range tos {
			seen[to] = true
		}
	}
	states := make([]State, 0, len(seen))
	for s := range seen {
		states = append(states, s)
	}
	sort.Slice(states, func(i, j int) bool { return states[i] < states[j] })

	return &Machine{
		name:        name,
		initial:     initial,
		transitions: transitions,
		states:      states,
	}
}

func (m *Machine) WithCategory(cat string) *Machine {
	m.category = cat
	return m
}

func (m *Machine) WithDescription(desc string) *Machine {
	m.description = desc
	return m
}

func (m *Machine) WithStateDescriptions(descs map[State]string) *Machine {
	m.stateDescriptions = descs
	return m
}

func (m *Machine) Name() string        { return m.name }
func (m *Machine) Category() string    { return m.category }
func (m *Machine) Description() string { return m.description }
func (m *Machine) Initial() State      { return m.initial }
func (m *Machine) States() []State     { return m.states }

func (m *Machine) IsValid(s State) bool {
	if s == "" {
		return false
	}
	for _, st := range m.states {
		if st == s {
			return true
		}
	}
	return false
}

func (m *Machine) IsTerminal(s State) bool {
	return len(m.transitions[s]) == 0
}

func (m *Machine) AllowedFrom(s State) []State {
	next := m.transitions[s]
	if len(next) == 0 {
		return []State{}
	}
	cp := make([]State, len(next))
	copy(cp, next)
	return cp
}

func (m *Machine) Can(from, to State) bool {
	for _, t := range m.transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

func (m *Machine) Validate(from, to State) error {
	if from == "" || to == "" {
		return fmt.Errorf("%s: empty state in transition %q -> %q", m.name, from, to)
	}
	if !m.IsValid(from) {
		return fmt.Errorf("%s: unknown state %q", m.name, from)
	}
	if m.Can(from, to) {
		return nil
	}
	return fmt.Errorf("%s: invalid transition %q -> %q (allowed from %q: %v)",
		m.name, from, to, from, m.transitions[from])
}

func (m *Machine) Transition(current *State, to State) error {
	if current == nil {
		return fmt.Errorf("%s: nil state pointer", m.name)
	}
	from := *current
	if from == "" {
		from = m.initial
	}
	if err := m.Validate(from, to); err != nil {
		return err
	}
	*current = to
	return nil
}

func (m *Machine) TransitionString(current *string, to State) error {
	if current == nil {
		return fmt.Errorf("%s: nil state string pointer", m.name)
	}
	from := State(*current)
	if from == "" {
		from = m.initial
	}
	if err := m.Validate(from, to); err != nil {
		return err
	}
	*current = string(to)
	return nil
}

func (m *Machine) Mermaid() string {
	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")
	fmt.Fprintf(&b, "    [*] --> %s\n", m.initial)
	froms := make([]State, 0, len(m.transitions))
	for from := range m.transitions {
		froms = append(froms, from)
	}
	sort.Slice(froms, func(i, j int) bool { return froms[i] < froms[j] })
	for _, from := range froms {
		for _, to := range m.transitions[from] {
			fmt.Fprintf(&b, "    %s --> %s\n", from, to)
		}
	}
	for _, s := range m.states {
		if m.IsTerminal(s) {
			fmt.Fprintf(&b, "    %s --> [*]\n", s)
		}
	}
	return b.String()
}

type Export struct {
	Name              string            `json:"name"`
	Category          string            `json:"category"`
	Description       string            `json:"description"`
	Initial           State             `json:"initial"`
	States            []State           `json:"states"`
	Terminal          []State           `json:"terminal"`
	Transitions       map[State][]State `json:"transitions"`
	StateDescriptions map[State]string  `json:"state_descriptions,omitempty"`
	Mermaid           string            `json:"mermaid"`
}

func (m *Machine) Export() Export {
	terminal := make([]State, 0)
	for _, s := range m.states {
		if m.IsTerminal(s) {
			terminal = append(terminal, s)
		}
	}
	return Export{
		Name:              m.name,
		Category:          m.category,
		Description:       m.description,
		Initial:           m.initial,
		States:            m.states,
		Terminal:          terminal,
		Transitions:       m.transitions,
		StateDescriptions: m.stateDescriptions,
		Mermaid:           m.Mermaid(),
	}
}

func (m *Machine) JSON() string {
	b, _ := json.MarshalIndent(m.Export(), "", "  ")
	return string(b)
}
