// Package stateregistry provides a single, canonical catalog of every explicit
// state machine across the Agentic Software Factory so UX (visual explorer),
// DX (CLI & validation endpoints), and AX (MCP agent introspection) never drift.
package stateregistry

import (
	"github.com/PratikDhanave/agentic-software-factory-go/internal/application"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/factory"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/integration"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/queue"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/verify"
)

// All returns every explicit backend state machine in logical architectural order.
func All() []*state.Machine {
	var list []*state.Machine
	list = append(list, application.PipelineMachines()...)
	list = append(list, domain.DomainMachines()...)
	list = append(list, entities.EntityMachines()...)
	list = append(list, factory.Machines()...)
	list = append(list,
		queue.QueueJobMachine,
		harness.HarnessMachine,
		verify.PhaseMachine,
		integration.ConnectionMachine,
	)
	return list
}

// Exports returns serializable Export descriptors for all registered machines.
func Exports() []state.Export {
	all := All()
	out := make([]state.Export, 0, len(all))
	for _, m := range all {
		out = append(out, m.Export())
	}
	return out
}

// ByName looks up a registered state machine by exact name.
func ByName(name string) (*state.Machine, bool) {
	for _, m := range All() {
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}
