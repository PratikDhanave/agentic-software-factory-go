// Package state provides a minimal explicit state machine engine.
//
// A Machine is a named, declarative transition table over string-typed
// states. Domain packages declare their lifecycle once (states, initial
// state, allowed transitions) and derive everything else from it:
// transition validation, terminal-state detection, and machine-readable
// exports (Mermaid, JSON) for documentation and agent introspection.
//
// States are string-typed so they serialize to exactly the same values
// that were previously stored as bare strings — existing databases and
// JSON APIs are unaffected.
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
	name        string
	initial     State
	transitions map[State][]State
	states      []State
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

// Name returns the machine's name.
func (m *Machine) Name() string { return m.name }

// Initial returns the machine's initial state.
func (m *Machine) Initial() State { return m.initial }

// States returns all states, sorted.
func (m *Machine) States() []State { return m.states }

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
// agent (AX) and tooling consumption.
type Export struct {
	Name        string             `json:"name"`
	Initial     State              `json:"initial"`
	States      []State            `json:"states"`
	Terminal    []State            `json:"terminal"`
	Transitions map[State][]State  `json:"transitions"`
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
		Name:        m.name,
		Initial:     m.initial,
		States:      m.states,
		Terminal:    terminal,
		Transitions: m.transitions,
	}
}

// JSON returns the machine description as indented JSON.
func (m *Machine) JSON() string {
	data, _ := json.MarshalIndent(m.Export(), "", "  ")
	return string(data)
}
