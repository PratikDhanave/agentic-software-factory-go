// Package state provides a minimal explicit state machine engine.
//
// A Machine is a named, declarative transition table over string-typed
// states. Domain packages declare their lifecycle once (states, initial
// state, allowed transitions, and human/agent descriptions) and derive
// everything else from it: transition validation, terminal-state detection,
// and machine-readable exports (Mermaid, JSON) for UX, DX, and AX.
//
// States are string-typed so they serialize to the exact same values
// stored in SQLite and returned over JSON APIs — zero breaking changes.
package state

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// State is a single named state in a machine.
type State string

// Machine is a declarative state machine: a transition table plus metadata.
// The zero value is not usable; construct with NewMachine.
type Machine struct {
	name              string
	category          string
	description       string
	initial           State
	transitions       map[State][]State
	stateDescriptions map[State]string
	states            []State
}

// NewMachine builds a machine from a transition table. Every state that
// appears only as a transition target (a terminal state) is included
// automatically; it does not need its own map entry.
func NewMachine(name string, initial State, transitions map[State][]State) *Machine {
	seen := map[State]bool{initial: true}
	var states []State
	add := func(s State) {
		if !seen[s] {
			seen[s] = true
		}
	}
	for from, tos := range transitions {
		add(from)
		for _, to := range tos {
			add(to)
		}
	}
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

// WithCategory attaches a logical grouping (e.g. "assembly", "governance", "execution", "verification", "infrastructure")
// for UI organization and agent discovery.
func (m *Machine) WithCategory(category string) *Machine {
	m.category = category
	return m
}

// WithDescription attaches a human- and agent-readable summary of what this machine governs.
func (m *Machine) WithDescription(description string) *Machine {
	m.description = description
	return m
}

// WithStateDescriptions attaches per-state explanations used by UI tooltips and MCP agent introspection.
func (m *Machine) WithStateDescriptions(descs map[State]string) *Machine {
	m.stateDescriptions = descs
	return m
}

// Name returns the machine's name.
func (m *Machine) Name() string { return m.name }

// Category returns the machine's category, if set.
func (m *Machine) Category() string { return m.category }

// Description returns the machine's human/agent summary, if set.
func (m *Machine) Description() string { return m.description }

// Initial returns the machine's initial state.
func (m *Machine) Initial() State { return m.initial }

// States returns all states, sorted.
func (m *Machine) States() []State { return m.states }

// AllowedFrom returns a copy of all valid target states reachable in one step from s.
func (m *Machine) AllowedFrom(s State) []State {
	next := m.transitions[s]
	if len(next) == 0 {
		return nil
	}
	cp := make([]State, len(next))
	copy(cp, next)
	return cp
}

// Can reports whether the transition from -> to is allowed.
func (m *Machine) Can(from, to State) bool {
	for _, t := range m.transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Validate returns a descriptive error if the transition is not allowed.
func (m *Machine) Validate(from, to State) error {
	if m.Can(from, to) {
		return nil
	}
	return fmt.Errorf("%s: invalid transition %q -> %q (allowed from %q: %v)",
		m.name, from, to, from, m.transitions[from])
}

// Transition validates from *current -> to and updates *current on success.
// If *current is empty, it is treated as m.initial.
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

// TransitionString validates and updates a string-typed status field in place,
// preserving existing struct field types and JSON/SQLite serialization.
func (m *Machine) TransitionString(current *string, to State) error {
	if current == nil {
		return fmt.Errorf("%s: nil string state pointer", m.name)
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

// IsTerminal reports whether a state has no outgoing transitions.
func (m *Machine) IsTerminal(s State) bool {
	return len(m.transitions[s]) == 0
}

// IsValid reports whether s is a known state of this machine.
func (m *Machine) IsValid(s State) bool {
	for _, st := range m.states {
		if st == s {
			return true
		}
	}
	return false
}

// Mermaid renders the machine as a Mermaid stateDiagram-v2.
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

// Export is the JSON-serializable description of a machine, intended for
// UX visualization, DX CLI inspection, and AX agent introspection.
type Export struct {
	Name              string            `json:"name"`
	Category          string            `json:"category,omitempty"`
	Description       string            `json:"description,omitempty"`
	Initial           State             `json:"initial"`
	States            []State           `json:"states"`
	Terminal          []State           `json:"terminal"`
	Transitions       map[State][]State `json:"transitions"`
	StateDescriptions map[State]string  `json:"state_descriptions,omitempty"`
	Mermaid           string            `json:"mermaid,omitempty"`
}

// Inspection describes the current status of a specific state within a machine
// for agent decision-making (AX) and interactive UI controls (UX).
type Inspection struct {
	Machine     string   `json:"machine"`
	Category    string   `json:"category,omitempty"`
	State       State    `json:"state"`
	Description string   `json:"description,omitempty"`
	IsValid     bool     `json:"is_valid"`
	IsInitial   bool     `json:"is_initial"`
	IsTerminal  bool     `json:"is_terminal"`
	AllowedNext []State  `json:"allowed_next"`
}

// Inspect returns full metadata for a given state value within this machine.
func (m *Machine) Inspect(s State) Inspection {
	var desc string
	if m.stateDescriptions != nil {
		desc = m.stateDescriptions[s]
	}
	allowed := m.AllowedFrom(s)
	if allowed == nil {
		allowed = []State{}
	}
	return Inspection{
		Machine:     m.name,
		Category:    m.category,
		State:       s,
		Description: desc,
		IsValid:     m.IsValid(s),
		IsInitial:   s == m.initial,
		IsTerminal:  m.IsValid(s) && m.IsTerminal(s),
		AllowedNext: allowed,
	}
}

// Export returns the machine description as a serializable value.
func (m *Machine) Export() Export {
	var terminal []State
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

// JSON returns the machine description as indented JSON.
func (m *Machine) JSON() string {
	data, _ := json.MarshalIndent(m.Export(), "", "  ")
	return string(data)
}
