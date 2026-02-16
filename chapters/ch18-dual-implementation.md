# Dual Implementation and Verification

**Learning objective**: Achieve specification parity between Go and JavaScript implementations.

Every tool in the pflow ecosystem processes the same JSON-LD models. The Go library runs ODE simulations on the server. The JavaScript solver runs them in the browser. If they produce different results from the same input, the specification is ambiguous — and any downstream analysis, proof, or decision based on those results is suspect.

This chapter covers the dual-implementation discipline: why both Go and JavaScript implementations exist, how parity is verified, and how event sourcing ties them together.

## Why Two Implementations?

A specification written in prose is ambiguous. A specification implemented in one language might contain bugs that look like features. But a specification implemented in two independent languages — and verified to produce identical output — is trustworthy in a way that neither implementation alone can be.

The pflow ecosystem maintains two implementations:

| | Go | JavaScript |
|-|---|------------|
| **Role** | Authoritative backend | Browser verification |
| **Solver** | `solver/tsit5.go` | `petri-solver.js` |
| **Petri net model** | `petri/` package | `petri-sim.js` |
| **Runs in** | Server, CLI | Browser |
| **Used by** | go-pflow, petri-pilot | pflow.xyz editor |

Both implementations parse the same JSON-LD format. Both implement the same Tsit5 ODE solver. Both use the same mass-action kinetics for transition rates. The claim: given identical input (model + initial state + rates + time span), they produce identical output (final state within numerical tolerance).

## State Root Parity

The strongest form of verification: compute a hash of the simulation output and compare. If the Go server and the JavaScript browser produce the same hash from the same model, the implementations agree.

```
JSON-LD Model
    ↓
Go Solver    →  State trajectory  →  Hash  ─┐
                                              ├─  Equal?
JS Solver    →  State trajectory  →  Hash  ─┘
```

This is the same principle as content addressing (Chapter 14), applied to simulation results. The hash doesn't prove the simulation is correct in some absolute sense — it proves the two implementations agree. And agreement between independent implementations, written in different languages by different code paths, is strong evidence of correctness.

### What Gets Compared

The comparison operates at the level of the state trajectory:

1. **Time points** — Both solvers use adaptive stepping, so they may choose different time points. The comparison evaluates at fixed checkpoints.
2. **Token counts** — For each place at each checkpoint, the token count must match within tolerance (typically 1e-6).
3. **Final state** — The most important check: do both solvers arrive at the same equilibrium?

The tolerance accounts for floating-point differences between Go and JavaScript — different rounding behavior, different intermediate precision. The tolerance is tight enough to catch algorithmic errors but loose enough to accommodate platform differences.

## The Tsit5 Algorithm — Twice

Both implementations use the Tsitouras 5th-order Runge-Kutta method (Tsit5). The algorithm is well-documented in the numerical methods literature, and the coefficients are published constants. But implementing it correctly requires getting many details right:

### Go Implementation

```go
// solver/tsit5.go
func (m Tsit5Method) Step(f ODEFunction, t float64, y []float64, dt float64) ([]float64, float64) {
    // 7 stages with Tsitouras coefficients
    k1 := f(t, y)
    k2 := f(t + c2*dt, addVec(y, scaleVec(dt*a21, k1)))
    // ... 5 more stages
    // 5th-order solution
    yNew := addVec(y, scaleVec(dt, linComb(b1, k1, b2, k2, ...)))
    // 4th-order embedded solution for error estimation
    yErr := addVec(y, scaleVec(dt, linComb(e1, k1, e2, k2, ...)))
    // Adaptive step size control
    err := errorNorm(yNew, yErr)
    return yNew, err
}
```

### JavaScript Implementation

```javascript
// petri-solver.js
function tsit5Step(f, t, y, dt) {
    // Same 7 stages, same coefficients
    const k1 = f(t, y);
    const k2 = f(t + c2*dt, addVec(y, scaleVec(dt*a21, k1)));
    // ... 5 more stages
    // Same 5th-order solution
    const yNew = addVec(y, scaleVec(dt, linComb(b1, k1, b2, k2, ...)));
    // Same error estimation and step control
    const err = errorNorm(yNew, yErr);
    return [yNew, err];
}
```

The algorithms are structurally identical. The coefficients are the same published constants. The adaptive step size control uses the same formula. The only differences are syntactic — Go vs JavaScript — and numerical — 64-bit floats in both, but with platform-specific rounding.

### Parameters Must Match

Both solvers use identical default parameters:

- Adaptive stepping with Tsit5
- Default `dt = 0.01`
- Relative tolerance `reltol = 1e-3`
- Same time span from the model

When a model behaves differently in the browser versus the Go backend, it's a bug. The outputs should match. This invariant is tested and maintained across both implementations.

## Event Sourcing

The dual-implementation discipline extends beyond ODE simulation to state management. The pflow ecosystem uses event sourcing as its state management pattern:

$$\text{State}(t) = \text{fold}(\text{apply}, \text{initialState}, \text{events}[0..t])$$

State is not stored directly. Instead, the system stores an immutable log of events (transition firings), and the current state is reconstructed by replaying the events from the beginning.

### Events as Transition Firings

In Petri net terms, an event is a transition firing:

```json
{
  "event_type": "brew",
  "aggregate_id": "coffeeshop-1",
  "timestamp": "2024-01-19T10:30:00Z",
  "version": 42,
  "data": {
    "beans_used": 2,
    "water_used": 3
  }
}
```

The event records which transition fired, when, and with what data. The state change (tokens removed from input places, tokens added to output places) is deterministic given the event and the current marking.

### Replay Equivalence

If both Go and JavaScript replay the same event log against the same model, they must arrive at the same final state. This is a stronger guarantee than ODE parity because it covers discrete-event execution, not just continuous simulation.

```
Event log: [brew, brew, restock, brew]

Go replay:   fold(apply, initial, events)  →  State A
JS replay:   fold(apply, initial, events)  →  State B

State A == State B  (must hold)
```

The `apply` function is the Petri net firing rule: check enabledness, subtract input tokens, add output tokens. Because the firing rule is deterministic and both implementations follow the same rule, replay produces identical states.

### Why Event Sourcing Fits

Event sourcing and Petri nets are natural partners:

1. **Events are transitions.** Every transition firing is an event. The event log is the execution trace of the Petri net.

2. **State is the marking.** The current marking (token counts across all places) is the state. It's computed by applying the firing rule for each event in sequence.

3. **Replay is re-execution.** Given the initial marking and the event log, you can reconstruct the state at any point in history. This enables auditing, debugging, and time-travel queries.

4. **The model is the schema.** The Petri net topology defines what events are valid (which transitions exist, what their pre- and post-conditions are). The model constrains the event log — you can't have a `brew` event if the net has no `brew` transition.

## The Verification Loop

In production, the verification works as a loop:

1. **Client** (browser, JavaScript) fires a transition and computes the new state locally
2. **Client** sends the transition firing to the server
3. **Server** (Go) independently computes the new state
4. **Server** stores the event and returns the new state
5. **Client** compares its locally computed state with the server's response
6. **If they match**: continue — both implementations agree
7. **If they don't match**: flag a discrepancy — investigation needed

This is the same pattern used in the ZK tic-tac-toe protocol (Chapter 12), except without the cryptographic proofs. The browser is a verifier: it independently computes what the server claims and checks the result.

### Trust Model

The verification loop establishes a trust model:

- The **server** is authoritative — its state is the canonical state
- The **client** is a verifier — it checks the server's work
- **Discrepancies** indicate a bug in one or both implementations
- **Agreement** increases confidence that the specification is unambiguous

No single implementation is trusted absolutely. Trust comes from agreement between independent implementations. This is the dual-implementation guarantee.

## Token Model Equivalence

The `tokenmodel` package in go-pflow extends dual implementation to the schema level. Token models can be defined in three equivalent ways:

1. **JSON** — the interchange format
2. **Go struct tags** — embedded in Go code
3. **S-expression DSL** — the human-readable format from Chapter 4

All three produce the same Petri net. The tokenmodel package includes tests verifying that parsing each format and building the net produces structurally identical results. This is isomorphism at the schema level — three notations, one mathematical object.

## What Parity Proves

Dual implementation doesn't prove the specification is correct in some absolute sense. It proves the specification is **unambiguous** — two independent teams, reading the same spec, implementing in different languages, arrive at the same behavior.

For Petri nets, unambiguous means:

- The **firing rule** is clear — both implementations agree on when a transition is enabled
- The **state update** is deterministic — both implementations produce the same marking after firing
- The **ODE dynamics** are correct — both solvers produce the same trajectories within numerical tolerance
- The **event sourcing** is faithful — replaying the same events produces the same state

These are exactly the properties you need for a formal modeling system. The mathematics works only if the implementation matches the theory. Dual implementation is how you check.

## The Ecosystem View

Dual implementation is not just a testing technique — it's an architectural principle. Every tool in the ecosystem can verify every other tool's work:

```
JSON-LD Model (source of truth)
    │
    ├── pflow.xyz (JavaScript)  ──  ODE simulation, visual editing
    │
    ├── go-pflow (Go)  ──  ODE simulation, reachability, analysis
    │
    ├── petri-pilot (Go)  ──  Code generation, event sourcing
    │
    └── ZK circuits (gnark)  ──  Cryptographic proofs
```

The JSON-LD model is the shared contract. Each tool processes it independently. Each tool can verify its results against any other tool processing the same model. The model format is declarative (Chapter 14), so there's no ambiguity about what it means — only about how each tool computes from it. And dual implementation resolves that ambiguity.

This is the practical payoff of the universal abstraction. One formalism — Petri nets. One format — JSON-LD. Multiple implementations — Go, JavaScript, gnark circuits. Verification — by agreement. The mathematics provides the theory. Dual implementation provides the confidence that the code matches the mathematics.
