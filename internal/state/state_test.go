package state

import (
	"strings"
	"testing"
)

func TestAllMachinesReachableAndValid(t *testing.T) {
	machines := All()
	if len(machines) != 8 {
		t.Fatalf("expected 8 core explicit state machines, got %d", len(machines))
	}

	seen := map[string]bool{}
	for _, m := range machines {
		if seen[m.Name()] {
			t.Fatalf("duplicate machine %s", m.Name())
		}
		seen[m.Name()] = true

		if !m.IsValid(m.Initial()) {
			t.Fatalf("%s: invalid initial state %s", m.Name(), m.Initial())
		}
		if m.IsValid("") {
			t.Fatalf("%s: empty state must not be valid", m.Name())
		}
		if err := m.Validate("", m.Initial()); err == nil {
			t.Fatalf("%s: expected empty source state to fail Validate", m.Name())
		}

		// BFS reachability check
		visited := map[State]bool{m.Initial(): true}
		q := []State{m.Initial()}
		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			for _, nxt := range m.AllowedFrom(cur) {
				if !visited[nxt] {
					visited[nxt] = true
					q = append(q, nxt)
				}
			}
		}
		for _, s := range m.States() {
			if !visited[s] {
				t.Errorf("%s: unreachable orphan state %s", m.Name(), s)
			}
		}
		if !strings.Contains(m.Mermaid(), "stateDiagram-v2") {
			t.Errorf("%s: invalid mermaid output", m.Name())
		}
	}
}
