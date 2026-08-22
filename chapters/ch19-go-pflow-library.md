# The go-pflow Library

**Learning objective**: Use the core Go library for simulation, analysis, and integration.

The previous chapters used go-pflow implicitly — building nets, running simulations, analyzing results. This chapter makes the library itself the subject. It covers the package structure, the fluent builder API, solver configuration, and the higher-level abstractions (state machines, workflows, actors) that compile down to Petri nets.

Think of this chapter as the reference that ties together everything from Parts I through III.

## Package Overview

go-pflow is organized into packages that map to different concerns:

| Package | Purpose |
|---------|---------|
| `petri` | Core types, fluent Builder API |
| `solver` | ODE solvers (Tsit5, RK45, implicit), equilibrium detection |
| `stateutil` | State map utilities (Copy, Apply, Merge, Sum, Diff) |
| `reachability` | Discrete state space, deadlock and liveness analysis |
| `hypothesis` | Move evaluation for game AI |
| `sensitivity` | Parameter sensitivity analysis |
| `cache` | Memoization for repeated simulations |
| `eventlog` | Parse CSV/JSONL event logs |
| `mining` | Process discovery (Alpha, Heuristic), rate learning |
| `monitoring` | Real-time case tracking, SLA alerts |
| `statemachine` | Hierarchical states compiled to Petri nets |
| `workflow` | Task dependencies, resources, SLA tracking |
| `actor` | Actor model with message passing |
| `visualization` | SVG rendering |
| `tokenmodel` | Token model schemas with DSL |

### Decision Tree

| Problem | Package |
|---------|---------|
| Business workflows | `workflow` |
| Event-driven state machines | `statemachine` |
| Message-passing systems | `actor` |
| Game AI and move evaluation | `hypothesis`, `cache` |
| Parameter optimization | `sensitivity` |
| Process discovery from logs | `mining`, `eventlog` |
| Deadlock and liveness checking | `reachability` |
| Population dynamics (epidemics, ecology) | `petri` + `solver` |
| General state and resource flow | `petri` |

## Building Nets

The fluent builder API constructs Petri nets in a single expression:

```go
net := petri.Build().
    Place("A", 10).Place("B", 0).
    Transition("t1").
    Arc("A", "t1", 1).Arc("t1", "B", 1).
    Done()
```

Each method returns the builder, allowing chaining. `Place` takes a name and initial token count. `Transition` takes a name. `Arc` takes source, target, and weight. `Done` finalizes and returns the `*petri.Net`.

### Chain Helper

For linear sequences — common in workflows — the `Chain` helper eliminates repetitive arc declarations:

```go
net := petri.Build().
    Chain(10, "start", "t1", "middle", "t2", "end").
    Done()
```

This creates places `start` (10 tokens), `middle`, and `end`, transitions `t1` and `t2`, and all connecting arcs. The first argument is the initial token count for the first place.

### Rates

Transition rates are separate from the net structure. The `WithRates` method assigns a default rate to all transitions:

```go
net, rates := petri.Build().
    Place("S", 100).Place("I", 1).Place("R", 0).
    Transition("infect").Transition("recover").
    Arc("S", "infect", 1).Arc("I", "infect", 1).Arc("infect", "I", 2).
    Arc("I", "recover", 1).Arc("recover", "R", 1).
    WithRates(1.0)
```

Rates can also be set individually via a `map[string]float64`. The separation of structure and rates is deliberate — the same net can be simulated with different rate constants for sensitivity analysis or parameter fitting.

### Shortcut Builders

Common model patterns have dedicated builders:

```go
// SIR epidemic model
net, rates := petri.Build().SIR(999, 1, 0).WithRates(1.0)
```

This creates the susceptible-infected-recovered model with the specified initial populations and a default rate of 1.0 for both infection and recovery transitions.

## Running Simulations

The solver package implements numerical ODE integration:

```go
prob := solver.NewProblem(net, net.SetState(nil), [2]float64{0, 100}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
final := sol.GetFinalState()
```

`NewProblem` takes the net, initial state, time span, and rates. `Solve` runs the integration. The result contains the full trajectory — token counts at each time step for every place.

### Solver Methods

| Method | Description |
|--------|-------------|
| `Tsit5()` | Tsitouras 5th-order Runge-Kutta. Default choice. Adaptive stepping, high accuracy. |
| `RK45()` | Classic Runge-Kutta 4/5. Similar to Tsit5, slightly different error estimation. |
| `Implicit()` | For stiff systems where explicit methods require impractically small steps. |

Tsit5 is the right choice for nearly all Petri net simulations. The adaptive step size automatically shrinks where dynamics are fast (early transients) and grows where dynamics are smooth (approaching equilibrium).

### Solver Presets

| Preset | Use Case |
|--------|----------|
| `DefaultOptions()` | General purpose — balanced accuracy and speed |
| `FastOptions()` | Game AI, interactive use — ~10x faster, lower accuracy |
| `AccurateOptions()` | Research, publishing — high precision |
| `GameAIOptions()` | Move evaluation — optimized for hypothesis testing |
| `EpidemicOptions()` | SIR/SEIR models — tuned for population dynamics |

Each preset configures step size (`Dt`), tolerance (`RelTol`, `AbsTol`), and maximum steps. For most applications, `DefaultOptions()` works. For interactive use where latency matters more than precision, `FastOptions()` trades accuracy for speed.

### Equilibrium Detection

Many models converge to a steady state where token counts stop changing:

```go
finalState, reached := solver.FindEquilibrium(prob)
```

The equilibrium detector monitors the derivative norm. When all derivatives are below a threshold — meaning no place's token count is changing significantly — the simulation stops early. This is faster than simulating to a fixed time horizon and checking the result.

## Reading Results

The solution object provides the full trajectory:

```go
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())

// Final state
final := sol.GetFinalState()
fmt.Println(final["espresso"]) // 47.3

// Full trajectory
for i, t := range sol.T {
    fmt.Printf("t=%.1f: beans=%.1f\n", t, sol.U[i]["beans"])
}
```

`sol.T` is the time points. `sol.U[i]` is the state at time `sol.T[i]` — a map from place names to token counts. The time points are not evenly spaced — the adaptive solver places more points where dynamics are fast.

## State Manipulation

The `stateutil` package provides utilities for working with state maps:

```go
import "github.com/pflow-xyz/go-pflow/stateutil"

// Apply a delta to a state
newState := stateutil.Apply(state, map[string]float64{"pos": 0, "mark": 1})

// Sum all token counts
total := stateutil.Sum(state)

// Compute difference between two states
changes := stateutil.Diff(before, after)

// Deep copy
copy := stateutil.Copy(state)
```

These utilities are used throughout the library — in hypothesis evaluation, reachability analysis, and monitoring. They handle the common pattern of manipulating `map[string]float64` state representations.

## Reachability Analysis

For discrete analysis — where tokens are integers and transitions fire one at a time — the reachability package explores the state space:

```go
analyzer := reachability.NewAnalyzer(net).WithMaxStates(10000)
result := analyzer.Analyze()
```

The result reports:

- **Bounded** — whether all places have finite token counts across all reachable states
- **HasCycle** — whether the state space contains cycles (the system can return to a previous state)
- **Live** — whether every transition can eventually fire from every reachable state
- **Deadlocks** — states where no transition is enabled

This is the discrete counterpart to ODE simulation. Where ODE simulation tracks continuous dynamics (Chapter 3), reachability analysis exhaustively enumerates discrete states. It's exact but computationally expensive — the state space grows exponentially with the number of places. The `WithMaxStates` limit prevents runaway exploration.

## Hypothesis Evaluation

The hypothesis package evaluates candidate moves in game models (Chapter 6):

```go
eval := hypothesis.NewEvaluator(net, rates, func(final map[string]float64) float64 {
    return final["wins"] - final["losses"]
})

// Find best move among candidates
bestIdx, _ := eval.FindBestParallel(state, []map[string]float64{move1, move2, move3})

// Sensitivity analysis — which transitions matter most?
impact := eval.SensitivityImpact(state)
```

The evaluator works by simulating each candidate move forward to equilibrium and scoring the result. `FindBestParallel` runs all candidates concurrently. The scoring function is user-defined — for tic-tac-toe it might maximize win tokens; for a resource model it might maximize throughput.

The `cache` package memoizes simulation results, so repeated evaluations of the same state (common in game tree search) are served from cache rather than re-simulated.

## Higher-Level Abstractions

The core `petri` package provides the mathematical foundation. Three packages build domain-specific abstractions on top.

### State Machines

The `statemachine` package implements hierarchical statecharts that compile to Petri nets:

```go
chart := statemachine.NewChart("light").
    Region("state").
        State("red").Initial().
        State("green").
        State("yellow").
    EndRegion().
    When("timer").In("state:red").GoTo("state:green").
    When("timer").In("state:green").GoTo("state:yellow").
    When("timer").In("state:yellow").GoTo("state:red").
    Build()

m := statemachine.NewMachine(chart)
m.SendEvent("timer")
m.State("state")  // "green"
```

The statechart API is familiar to anyone who has used UML state diagrams. Under the hood, each state becomes a place, each transition becomes a Petri net transition, and the mutual exclusion constraint (only one state active per region) is enforced by the Petri net structure. Hierarchical states (nested regions) compose naturally.

### Workflows

The `workflow` package models task dependencies with resources and SLA tracking:

```go
wf := workflow.New("order").
    ManualTask("receive", "Receive", 2*time.Minute).
    AutoTask("validate", "Validate", 30*time.Second).
    ManualTask("ship", "Ship", 5*time.Minute).
    From("receive").Then("validate").To("ship").
    Start("receive").End("ship").
    WithSLA(4 * time.Hour).
    Build()

engine := workflow.NewEngine(wf)
engine.StartCase("case-1", nil, workflow.PriorityMedium)
```

Tasks can be manual (require human action) or automatic (fire when enabled). The SLA tracker monitors elapsed time per case and raises alerts when deadlines approach. The underlying Petri net provides formal properties — deadlock freedom means every started case can complete, and conservation laws ensure resources are properly allocated.

### Actors

The `actor` package implements the actor model — concurrent entities that communicate via message passing:

```go
system := actor.NewSystem("sys").DefaultBus().
    Actor("worker").
        Handle("task", func(ctx *actor.ActorContext, s *actor.Signal) error {
            ctx.Emit("done", map[string]any{"result": "ok"})
            return nil
        }).
        Done().
    Start()
```

Each actor is a Petri net. Message receipt is a transition. Processing is a sequence of internal transitions. Message sending is an output transition. The actor system composes actor nets through shared places (the message bus). Concurrency, deadlock, and liveness properties of the actor system follow from Petri net analysis of the composed net.

## Sensitivity Analysis

The `sensitivity` package measures how changes in parameters affect simulation outcomes:

```go
sa := sensitivity.NewAnalyzer(net, rates, baseState)
results := sa.AnalyzeAll()

for _, r := range results {
    fmt.Printf("%s: impact=%.4f\n", r.Parameter, r.Impact)
}
```

The analyzer perturbs each rate constant individually, re-simulates, and measures the change in the final state. High-impact parameters are bottlenecks — points where small changes in rate produce large changes in outcome. This connects to the bottleneck analysis from Chapter 5 (coffee shop) and the what-if scenarios from Chapter 11 (process mining).

## Process Mining Integration

The mining and monitoring packages (covered in depth in Chapter 11) complete the library. They connect external data to Petri net models:

```go
// Parse event logs
log, _ := eventlog.ParseCSV("events.csv", eventlog.DefaultCSVConfig())

// Discover process model
result, _ := mining.Discover(log, "heuristic")
net := result.Net

// Learn rates from timing data
rates := mining.LearnRatesFromLog(log, net)

// Check how well the model fits the data
conf := mining.CheckConformance(log, net)
```

The same `*petri.Net` type produced by the builder API or the mining algorithms feeds into the solver, reachability analyzer, and visualization. The library doesn't distinguish between hand-crafted and data-discovered models.

## SVG Rendering

The `visualization` package generates SVG diagrams from nets:

```go
svg := visualization.RenderSVG(net, visualization.ForceAtlas2Layout)
```

Layout algorithms include force-directed (ForceAtlas2), hierarchical (top-to-bottom), and circular. The generated SVGs show places as circles, transitions as rectangles, arcs as arrows, and token counts as numbers or dots. These SVGs appear in the pflow.xyz editor (Chapter 17), the petri-pilot model viewer (Chapter 18), and throughout this book.

## Putting It Together

The library's packages compose. A typical application might:

1. **Build** a net with the fluent API (`petri`)
2. **Simulate** with the ODE solver (`solver`)
3. **Analyze** reachability for correctness (`reachability`)
4. **Evaluate** moves for game AI (`hypothesis`)
5. **Tune** parameters via sensitivity analysis (`sensitivity`)
6. **Mine** rates from production logs (`mining`)
7. **Monitor** running instances for SLA compliance (`monitoring`)
8. **Visualize** the net and simulation results (`visualization`)

Each package does one thing. The shared `*petri.Net` type is the lingua franca — every package consumes and produces it. This is the practical benefit of the universal abstraction: a single formalism, implemented once, applied everywhere.
