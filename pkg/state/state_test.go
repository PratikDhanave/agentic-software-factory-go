package state

import (
	"strings"
	"testing"
)

func testMachine() *Machine {
	return NewMachine("task", "pending", map[State][]State{
		"pending": {"running"},
		"running": {"completed", "failed"},
		"failed":  {"pending"},
	})
}

func TestTransitions(t *testing.T) {
	m := testMachine()

	allowed := [][2]State{
		{"pending", "running"},
		{"running", "completed"},
		{"running", "failed"},
		{"failed", "pending"},
	}
	for _, tr := range allowed {
		if !m.Can(tr[0], tr[1]) {
			t.Errorf("expected %s -> %s to be allowed", tr[0], tr[1])
		}
		if err := m.Validate(tr[0], tr[1]); err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	}

	denied := [][2]State{
		{"pending", "completed"},
		{"completed", "pending"},
		{"completed", "running"},
		{"failed", "running"},
	}
	for _, tr := range denied {
		if m.Can(tr[0], tr[1]) {
			t.Errorf("expected %s -> %s to be denied", tr[0], tr[1])
		}
		if err := m.Validate(tr[0], tr[1]); err == nil {
			t.Errorf("expected validation error for %s -> %s", tr[0], tr[1])
		}
	}
}

func TestTerminal(t *testing.T) {
	m := testMachine()
	if !m.IsTerminal("completed") {
		t.Error("completed should be terminal")
	}
	for _, s := range []State{"pending", "running", "failed"} {
		if m.IsTerminal(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestStatesCollected(t *testing.T) {
	m := testMachine()
	if len(m.States()) != 4 {
		t.Errorf("expected 4 states, got %v", m.States())
	}
	if !m.IsValid("completed") {
		t.Error("completed should be a valid state even though it only appears as a target")
	}
	if m.IsValid("cancelled") {
		t.Error("cancelled is not declared and should be invalid")
	}
}

func TestExports(t *testing.T) {
	m := testMachine()

	mermaid := m.Mermaid()
	for _, want := range []string{"stateDiagram-v2", "[*] --> pending", "running --> completed", "completed --> [*]"} {
		if !strings.Contains(mermaid, want) {
			t.Errorf("mermaid output missing %q:\n%s", want, mermaid)
		}
	}

	exp := m.Export()
	if exp.Initial != "pending" || len(exp.Terminal) != 1 || exp.Terminal[0] != "completed" {
		t.Errorf("unexpected export: %+v", exp)
	}
	if !strings.Contains(m.JSON(), `"name": "task"`) {
		t.Error("JSON export missing machine name")
	}
}
