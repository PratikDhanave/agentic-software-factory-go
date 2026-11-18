package stateregistry

import (
	"strings"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

func TestAllMachinesWellFormedAndReachable(t *testing.T) {
	machines := All()
	if len(machines) < 18 {
		t.Fatalf("expected at least 18 explicit backend state machines, got %d", len(machines))
	}

	seenNames := make(map[string]bool, len(machines))
	for _, m := range machines {
		if m.Name() == "" {
			t.Fatal("machine has empty name")
		}
		if seenNames[m.Name()] {
			t.Fatalf("duplicate state machine name %q", m.Name())
		}
		seenNames[m.Name()] = true

		if m.Category() == "" {
			t.Errorf("machine %q missing category", m.Name())
		}
		if m.Description() == "" {
			t.Errorf("machine %q missing description", m.Name())
		}
		if !m.IsValid(m.Initial()) {
			t.Errorf("machine %q initial state %q is not valid", m.Name(), m.Initial())
		}

		// Verify every declared state is reachable from Initial() (no unreachable dead states)
		reachable := map[state.State]bool{m.Initial(): true}
		queue := []state.State{m.Initial()}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, next := range m.AllowedFrom(cur) {
				if !reachable[next] {
					reachable[next] = true
					queue = append(queue, next)
				}
			}
		}

		for _, s := range m.States() {
			if !reachable[s] {
				t.Errorf("machine %q has unreachable orphan state %q", m.Name(), s)
			}
		}

		mermaid := m.Mermaid()
		if !strings.Contains(mermaid, "stateDiagram-v2") {
			t.Errorf("machine %q invalid mermaid export", m.Name())
		}

		found, ok := ByName(m.Name())
		if !ok || found != m {
			t.Errorf("ByName(%q) failed", m.Name())
		}
	}
}
