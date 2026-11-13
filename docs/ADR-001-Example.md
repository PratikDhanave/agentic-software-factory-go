# ADR-001: Example Decision - APPROVED

## Context
This is an example Architectural Decision Record.

## Decision
We decided to use Go for the agentic software factory implementation.

## The Absolute Constraint
All code MUST be written in Go 1.23+.
All dependencies MUST be vendored or in go.mod.

## Consequences
- Type safety at compile time
- Native concurrency support
- Fast startup times
