# Petri Nets as a Universal Abstraction
## A Practitioner's Guide to Modeling with pflow

---

### Preface
- Who this book is for
- How to read this book (theory track vs. hands-on track)
- The pflow ecosystem at a glance

---

## Part I: Foundations

### Chapter 1: Why Petri Nets?
**Learning objective**: Understand what Petri nets offer over flowcharts, state machines, and ad-hoc modeling.

- The problem with informal models
- A brief history (Carl Adam Petri, 1962)
- Places, transitions, arcs, tokens — the four primitives
- Your first net: a traffic stoplight
- Small models, not large language models
- **Source material**: blog/ode-to-petri, blog/small-models-not-llms, go-pflow/docs/concepts/petri-nets.md

### Chapter 2: The Mathematics of Flow
**Learning objective**: Read and write the formal notation; understand firing rules and state equations.

- Formal definition (5-tuple)
- Markings and the state vector
- Firing rules and reachability
- Incidence matrices
- P-invariants and conservation laws
- **Source material**: go-pflow/docs/mathematics.md

### Chapter 3: From Discrete to Continuous
**Learning objective**: Convert a Petri net into an ODE system and understand why this is powerful.

- Why discrete event simulation hits walls
- Mass-action kinetics: chemistry meets computer science
- The continuous relaxation
- Numerical integration (Tsit5, RK45)
- Equilibrium detection and what it means
- **Source material**: blog/declarative-differential-models, go-pflow/docs/concepts/ode-simulation.md

### Chapter 4: The Token Language
**Learning objective**: Use the pflow DSL to define nets with guards, roles, and typed arcs.

- The four-term primitive: cell, func, arrow, guard
- S-expression syntax
- Guards and conditional firing
- Weighted arcs and inhibitor arcs
- Categorical net types (Workflow, Resource, Game, Computation, Classification)
- **Source material**: blog/token-language, blog/categorical-net-types, go-pflow/tokenmodel/

---

## Part II: Applications

### Chapter 5: Resource Modeling — The Coffee Shop
**Learning objective**: Model inventory, throughput, and bottlenecks as token flow.

- Defining the net: ingredients, recipes, orders
- Weighted arcs as recipes
- ODE simulation for capacity planning
- Bottleneck analysis from equilibrium
- From model to running application
- **Source material**: blog/coffeeshop-model, petri-pilot/services/coffeeshop.json, go-pflow/examples/coffeeshop/

### Chapter 6: Game Mechanics — Tic-Tac-Toe
**Learning objective**: Encode game rules, turn control, and win detection as net structure.

- The grid layer: 9 places, 18 transitions
- Turn control via mutual exclusion
- History places for pattern detection
- Win detection without search
- ODE-guided strategy: heatmaps from topology
- **Source material**: blog/tic-tac-toe-model, go-pflow/examples/tictactoe/, petri-pilot/services/tic-tac-toe.json

### Chapter 7: Constraint Satisfaction — Sudoku
**Learning objective**: Express constraints as token conservation and solve via continuous relaxation.

- Cell places and digit transitions
- Row, column, box constraints as inhibitor structure
- ODE prediction for move ordering
- Scaling from 4×4 to 9×9
- **Source material**: blog/sudoku-petri-net-model, go-pflow/examples/sudoku/

### Chapter 8: Optimization — The Knapsack Problem
**Learning objective**: Solve NP-hard problems approximately via mass-action kinetics.

- The 0/1 knapsack as a Petri net
- Item efficiency encoded in transition rates
- Continuous relaxation and rounding
- Exclusion analysis for sensitivity
- Comparison with branch-and-bound
- **Source material**: blog/knapsack-model, go-pflow/examples/knapsack/

### Chapter 9: Biochemistry — Enzyme Kinetics
**Learning objective**: See how Petri nets naturally model chemical reactions, recovering classical equations.

- Substrate, enzyme, product: three transitions
- Mass-action kinetics as the native semantics
- Emergent Michaelis-Menten behavior
- Conservation laws from P-invariants
- Parameter exploration and sensitivity
- **Source material**: blog/enzyme-kinetics-model, petri-pilot/services/enzyme-kinetics.json

### Chapter 10: Complex State Machines — Texas Hold'em
**Learning objective**: Model multi-player, multi-phase systems with roles and guards.

- Phase places: preflop → flop → turn → river → showdown
- Turn tokens and role-based access
- Betting conditions as guard expressions
- Event sourcing for audit trails
- ODE analysis of game flow
- **Source material**: blog/texas-holdem-model, petri-pilot/services/texas-holdem.json, petri-pilot/frontends/texas-holdem/

---

## Part III: Advanced Topics

### Chapter 11: Process Mining
**Learning objective**: Discover Petri net models from event logs and use them for prediction.

- From event logs to process models
- Timing extraction and rate learning
- Predictive monitoring with SLA tracking
- The hospital ER case study
- **Source material**: go-pflow/mining/, go-pflow/docs/concepts/process-mining.md, go-pflow/examples/monitoring_demo/

### Chapter 12: Zero-Knowledge Proofs
**Learning objective**: Prove a state transition is valid without revealing the state.

- Why privacy matters for Petri nets
- State commitment via MiMC hashing
- Topology as circuit constants
- Groth16 proofs with gnark
- ZK Tic-Tac-Toe: a complete example
- Applications beyond games
- **Source material**: blog/zk-petri-nets, blog/zk-tic-tac-toe-model, go-pflow/docs/petri-to-gnark.md

### Chapter 13: Exponential Weights and Scoring Systems
**Learning objective**: Design scoring systems using power-of-2 encoding in net structure.

- Binary dominance and lexicographic ordering
- Poker hand ranking as a case study
- The boundary between net and external logic
- Lessons from the poker experiment
- **Source material**: blog/exponential-scoring

### Chapter 14: Declarative Infrastructure
**Learning objective**: Use JSON-LD and semantic vocabularies to make nets self-describing and composable.

- Three vocabularies: pflow.xyz, schema.org, ActivityStreams
- Monotonic expansion and canonicalization
- Content addressing for reproducibility
- Category theory view of @context
- **Source material**: blog/json-ld-declarative-infrastructure

---

## Part IV: Building with pflow

### Chapter 15: The Visual Editor — pflow.xyz
**Learning objective**: Design nets visually, export to JSON, and iterate on models interactively.

- The editor interface
- Drawing nets: click-to-place workflow
- Export formats: JSON, SVG, tokenmodel DSL
- Sharing and embedding models
- **Source material**: pflow-xyz project

### Chapter 16: Code Generation — From Model to Application
**Learning objective**: Generate full-stack applications from a Petri net model.

- The petri-pilot pipeline: model → schema → code
- Events-first pattern: separating events from bindings
- Generated API endpoints and state management
- Customization architecture: hooks, slots, overrides
- **Source material**: petri-pilot/ARCHITECTURE.md, petri-pilot/docs/events-first-pattern.md

### Chapter 17: The go-pflow Library
**Learning objective**: Use the core Go library for simulation, analysis, and integration.

- Package overview and decision tree
- Building a net programmatically
- Running simulations and reading results
- Solver presets and tuning
- Reachability analysis and verification
- **Source material**: go-pflow/README.md, go-pflow/CLAUDE.md

### Chapter 18: Dual Implementation and Verification
**Learning objective**: Achieve specification parity between Go and JavaScript implementations.

- Why dual implementation matters
- State root parity as proof of unambiguous spec
- Browser verification of server-side logic
- Event sourcing: `State(t) = fold(apply, initialState, events[0..t])`
- **Source material**: workspace CLAUDE.md patterns, go-pflow + pflow-xyz parity

---

## Epilogue: What the Abstraction Sits On
**Learning objective**: Name the layered architecture the book built, see the applications through the net type taxonomy, and identify the open problems.

- The four-layer stack: graph theory → Petri net semantics → ODE dynamics → ZK verification
- Six applications, five types — retrospective taxonomy table (Coffee Shop=ResourceNet, TTT=GameNet, Sudoku=ClassificationNet, Knapsack=ComputationNet, Enzyme=ComputationNet, Hold'em=GameNet)
- What the book proved: small models beat black boxes, one formalism / multiple tools, topology is primary
- What the book didn't solve: multi-hop connectivity, weighted targets, dynamic rates, circuit scaling, composition verification
- The premise revisited: the Petri net is a layer, not the foundation — the topology was always there
- **Source material**: Ch 1 (opening premise), Ch 4 (net taxonomy), Ch 13 (graph theory insight, limitations)

---

## Appendices

### Appendix A: Solver Parameter Reference
- Tsit5, RK45 presets
- Dt, steps, tolerance tuning
- When to use which solver

### Appendix B: Token Language Grammar
- Complete S-expression syntax reference
- Guard expression language

### Appendix C: JSON Schema Reference
- Model format specification
- Results format for analysis output

### Appendix D: Glossary
- Place, transition, arc, token, marking, firing, reachability, P-invariant, ...

---

## Estimated Scope

- **18 chapters + epilogue + 4 appendices**
- **~60,000–80,000 words** target
- **Existing material covers ~70%** of outlined content
- **Gaps to write from scratch**: Ch 15 (editor walkthrough), Ch 18 (dual implementation), narrative transitions, exercises
