# The go-pflow Library

**Learning objective**: Use the core Go library for simulation, analysis, and integration.

The previous chapters used go-pflow implicitly — building nets, running simulations, analyzing results. This chapter makes the library itself the subject. It covers the package structure, the fluent builder API, solver configuration, and the higher-level abstractions (state machines, workflows, actors) that compile to, or describe themselves as, Petri nets.

Think of this chapter as the reference that ties together everything from Parts I through III. The snippets below follow the go-pflow source but are not compiled as part of this book; where a test in the repository exercises the same calls, the section links it.

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

The table lists the packages this chapter uses. The library has more — stochastic simulation, declarative property checking (`verify`), parameter fitting (`learn`), the `metamodel` engine schema, the ZK prover — and the [README's package table](https://github.com/pflow-xyz/go-pflow/blob/main/README.md#packages) lists them all with the chapter that covers each.

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

The fluent builder API ([`petri/builder.go`](https://github.com/pflow-xyz/go-pflow/blob/main/petri/builder.go)) constructs Petri nets in a single expression:

```go
net := petri.Build().
    Place("A", 10).Place("B", 0).
    Transition("t1").
    Arc("A", "t1", 1).Arc("t1", "B", 1).
    Done()
```

Each method returns the builder, allowing chaining. `Place` takes a name and initial token count. `Transition` takes a name. `Arc` takes source, target, and weight. `Done` returns the `*petri.PetriNet`. The builder also assigns each place and transition a screen position, left to right in the order they are added, which the SVG renderer uses later.

### Chain Helper

For linear sequences — common in workflows — the `Chain` helper eliminates repetitive arc declarations:

```go
net := petri.Build().
    Chain(10, "start", "t1", "middle", "t2", "end").
    Done()
```

This creates places `start` (10 tokens), `middle`, and `end`, transitions `t1` and `t2`, and all connecting arcs, each of weight 1. The first argument is the initial token count for the first place. The element list must alternate place, transition, place and so end on a place; a list of the wrong length is ignored and the builder is returned unchanged.

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

`WithCustomRates(map[string]float64{"infect": 0.3, "recover": 0.1})` returns the net with a rate map you supply instead. The separation of structure and rates is deliberate — the same net can be simulated with different rate constants for sensitivity analysis or parameter fitting.

### Shortcut Builders

Common model patterns have dedicated builders:

```go
// SIR epidemic model
net, rates := petri.Build().SIR(999, 1, 0).WithRates(1.0)
```

This creates the susceptible-infected-recovered net from the previous snippet — places `S`, `I`, `R`, transitions `infect` and `recover`, the same five arcs — with the specified initial populations and a default rate of 1.0 for both transitions. `TestBuilderSIR` in [`petri/builder_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/petri/builder_test.go) checks the places, transitions and arcs it produces.

## Running Simulations

The solver package implements numerical ODE integration:

```go
prob := solver.NewProblem(net, net.SetState(nil), [2]float64{0, 100}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
final := sol.GetFinalState()
```

`NewProblem` takes the net, initial state, time span, and rates. `net.SetState(nil)` returns the net's initial marking as a map; passing a map instead overrides the places it names. `Solve` runs the integration. The result contains the full trajectory — token counts at each time step for every place. The rate law is the one from [Chapter 3: From Discrete to Continuous](ch03-discrete-to-continuous.md): a transition's flux is its rate times the product of its input places' token counts, and arc weights scale how much each firing consumes and produces, not the kinetics ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)).

### Solver Methods

| Method | Description |
|--------|-------------|
| `Tsit5()` | Tsitouras 5(4) Runge-Kutta. Default choice. Adaptive stepping, high accuracy. |
| `RK45()` | Dormand-Prince 5(4), the method behind MATLAB's `ode45`. Similar to Tsit5, slightly different error estimation. |
| `ImplicitEuler(prob, opts)`, `TRBDF2(prob, opts)` | For stiff systems where explicit methods require impractically small steps. These are functions that return a solution directly, not a method passed to `Solve`. |
| `SolveImplicit(prob, opts)` | Detects stiffness and chooses between explicit and implicit integration. |

`RK4`, `Euler`, `Heun`, `Midpoint` and `BS32` are also available; [Appendix A: Solver Parameter Reference](appendix-a-solver-reference.md) describes each.

Tsit5 is the library's default: `SolveUntilEquilibrium` falls back to it when given no method, and the `hypothesis` and `sensitivity` packages call it internally. The adaptive step size shrinks where dynamics are fast (early transients) and grows where they are smooth (approaching equilibrium).

### Solver Presets

| Preset | Use Case |
|--------|----------|
| `DefaultOptions()` | General purpose — balanced accuracy and speed |
| `FastOptions()` | Game AI, interactive use — looser tolerances, fewer steps |
| `AccurateOptions()` | Research, publishing — high precision |
| `GameAIOptions()` | Move evaluation — optimized for hypothesis testing |
| `EpidemicOptions()` | SIR/SEIR models — tuned for population dynamics |

Each preset sets the `Options` fields: initial, minimum and maximum step size (`Dt`, `Dtmin`, `Dtmax`), tolerances (`Abstol`, `Reltol`), the iteration cap (`Maxiters`) and whether stepping is `Adaptive`. `solver/ode.go` defines further presets — `StiffOptions`, `WorkflowOptions`, `LongRunOptions`, and `JSParityOptions` for matching the pflow.xyz browser solver — and Appendix A tabulates every preset's values. For most applications, `DefaultOptions()` works. For interactive use where latency matters more than precision, `FastOptions()` trades accuracy for speed; its doc comment estimates a tenfold speedup, which is not benchmarked — `TestFastOptionsSpeed` in [`solver/solver_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/solver_test.go) checks only that it takes fewer steps than `AccurateOptions` on an SIR net.

### Equilibrium Detection

Many models converge to a steady state where token counts stop changing:

```go
finalState, reached := solver.FindEquilibrium(prob)
```

`FindEquilibrium` integrates with Tsit5 and watches the largest absolute derivative across all places. With the default settings it checks every ten accepted steps, starting 0.1 time units into the run, and stops once that maximum has stayed below 10⁻⁶ on five consecutive checks ([`solver/equilibrium.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/equilibrium.go)). `reached` reports whether that happened before the time span ran out; if it did not, `finalState` is the state at the end of the span. Stopping early saves the steps a fixed horizon would spend after the system has settled. `TestFindEquilibrium` runs a single transition moving tokens from `A` to `B` and checks that `A` is nearly empty and `A + B` is conserved.

## Reading Results

The solution object provides the full trajectory. Continuing the SIR example:

```go
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())

// Final state
final := sol.GetFinalState()
fmt.Println(final["R"])

// Full trajectory
for i, t := range sol.T {
    fmt.Printf("t=%.1f: I=%.1f\n", t, sol.U[i]["I"])
}
```

`sol.T` is the time points. `sol.U[i]` is the state at time `sol.T[i]` — a map from place names to token counts. The time points are not evenly spaced — the adaptive solver places more points where dynamics are fast. Check `sol.Truncated` when comparing many solves: it is set when the solver hit `Maxiters` before the end of the time span, and a truncated solution otherwise looks like a complete one.

## State Manipulation

The `stateutil` package provides utilities for working with state maps:

```go
import "github.com/pflow-xyz/go-pflow/stateutil"

// Apply a delta to a state
newState := stateutil.Apply(state, map[string]float64{"pos": 0, "mark": 1})

// Sum all token counts
total := stateutil.Sum(state)

// Keys whose values differ, with the values from after
changes := stateutil.Diff(before, after)

// Deep copy
snapshot := stateutil.Copy(state)
```

They handle the common pattern of manipulating `map[string]float64` state representations. Inside the library, `hypothesis` uses `Apply` to build each candidate state; the reachability analyzer works on its own integer `Marking` type instead.

## Reachability Analysis

For discrete analysis — where tokens are integers and transitions fire one at a time — the reachability package explores the state space breadth-first ([`reachability/analyzer.go`](https://github.com/pflow-xyz/go-pflow/blob/main/reachability/analyzer.go)):

```go
analyzer := reachability.NewAnalyzer(net).WithMaxStates(10000)
result := analyzer.Analyze()
```

The result reports, among other fields:

- `Bounded` — false if some place exceeded the token limit (`WithMaxTokens`, default 1,000) during exploration. This is a cut-off, not a proof: a bounded net whose places legitimately hold more than 1,000 tokens also reads as unbounded.
- `HasCycle` — whether the explored state space contains a cycle (the system can return to a previous state); `Cycles` lists the transition sequences.
- `Live` — whether every transition fires somewhere in the reachability graph. This is the weaker property of having no dead transitions, not liveness in the [Glossary](appendix-d-glossary.md)'s sense (every transition can eventually fire again from every reachable marking). It is reported true only when exploration finished.
- `Deadlocks` — terminal states, where no transition is enabled, whose marking is not empty. A terminal state with no tokens left counts as a normal end.

This is the discrete counterpart to ODE simulation. Where ODE simulation tracks continuous dynamics ([Chapter 3](ch03-discrete-to-continuous.md)), reachability analysis enumerates discrete states. It is exact when the exploration completes, but the state space can grow exponentially with the size of the net. `WithMaxStates` (10,000 is also the default) caps the exploration; a capped run sets `Truncated`, lists unfired transitions as `PotentiallyDead` rather than dead, and reports `Live` as false.

## Hypothesis Evaluation

The hypothesis package evaluates candidate moves in game models ([Chapter 6: Game Mechanics](ch06-game-mechanics.md)):

```go
eval := hypothesis.NewEvaluator(net, rates, func(final map[string]float64) float64 {
    return final["wins"] - final["losses"]
})

// Find best move among candidates (each a map of place updates)
bestIdx, _ := eval.FindBestParallel(state, []map[string]float64{move1, move2, move3})

// Sensitivity analysis — which transitions matter most?
impact := eval.SensitivityImpact(state)
```

Each candidate is a map of place updates. The evaluator applies it to the base state with `stateutil.Apply`, simulates forward with Tsit5 and `FastOptions` over a fixed span of t = 0 to 5, and scores the final state ([`hypothesis/hypothesis.go`](https://github.com/pflow-xyz/go-pflow/blob/main/hypothesis/hypothesis.go)). It does not run to equilibrium; `WithTimeSpan` and `WithOptions` change the span and the solver settings. `FindBestParallel` evaluates the candidates in one goroutine each and returns the index and score of the best. `SensitivityImpact` sets each transition's rate to zero in turn and returns the change in score from the baseline. The scoring function is user-defined — for tic-tac-toe it might maximize win tokens; for a resource model it might maximize throughput.

The `cache` package memoizes simulation results, so repeated evaluations of the same state (common in game tree search) are served from cache rather than re-simulated. The evaluator does not cache on its own: `cache.NewCachedEvaluator(net, rates, size)` is a separate evaluator with a cache in front. Entries are keyed by a hash of the state alone ([`cache/cache.go`](https://github.com/pflow-xyz/go-pflow/blob/main/cache/cache.go)), so one cache is valid for one set of rates.

## Higher-Level Abstractions

The core `petri` package provides the mathematical foundation. Three packages build domain-specific abstractions on top.

### State Machines

The `statemachine` package implements hierarchical statecharts that compile to Petri nets:

```go
chart := statemachine.NewChart("traffic_light").
    Region("light").
        State("red").Initial().
        State("green").
        State("yellow").
    EndRegion().
    When("timer").In("light:red").GoTo("light:green").
    When("timer").In("light:green").GoTo("light:yellow").
    When("timer").In("light:yellow").GoTo("light:red").
    Build()

m := statemachine.NewMachine(chart)
m.SendEvent("timer")
m.State("light")  // "green"
```

This is the chart from `TestSimpleStateMachine` in [`statemachine/statemachine_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/statemachine/statemachine_test.go), which asserts that three `timer` events take the light from red to green to yellow and back to red.

The statechart API is familiar to anyone who has used UML state diagrams. `Chart.ToPetriNet` ([`statemachine/builder.go`](https://github.com/pflow-xyz/go-pflow/blob/main/statemachine/builder.go)) compiles it: each state becomes a place named `region_state`, and each chart transition becomes a net transition with one arc in from its source state and one arc out to its target. For a flat region like this one, every transition therefore moves exactly one token within the region, so the region's token count — one, placed on the initial state — is a P-invariant. That follows from the arcs; it is derived here, not computed by a test. `NewMachine` runs on the compiled net's marking but routes events itself: `SendEvent` takes the first transition for the event whose source place is marked and whose guard holds, then clears the source and marks the target. Nested substates use the same one-token pattern, with extra arcs for the parent states.

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

Tasks are labelled manual (human-performed) or automatic (system-performed). The engine does not treat the two differently; your code moves each task along with `StartTask` and `CompleteTask` either way. `WithSLA` sets a deadline per case with warning and critical thresholds at 80% and 95% of it ([`workflow/sugar.go`](https://github.com/pflow-xyz/go-pflow/blob/main/workflow/sugar.go)), and `Engine.CheckSLAs` returns alerts for cases past those thresholds. `NewEngine` compiles the workflow with `Workflow.ToPetriNet`, but it tracks cases and resource pools in its own Go structures ([`workflow/engine.go`](https://github.com/pflow-xyz/go-pflow/blob/main/workflow/engine.go)). Formal properties are therefore something you check rather than something the engine guarantees: passing `wf.ToPetriNet()` to the reachability analyzer is how you would establish that every started case can complete. No example in the repository does this yet for a workflow.

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

At runtime the actors are goroutines exchanging signals over a bus, and a handler like the one above is an ordinary Go function, not a Petri transition. The Petri-net view is structural. `ActorSystem.ToBundle` ([`actor/subnet.go`](https://github.com/pflow-xyz/go-pflow/blob/main/actor/subnet.go)) emits one subnet per actor, with an in-port place for each signal type it subscribes to, an out-port for each type its behaviours emit, and a transition per behaviour trigger; links join every emitter of a signal type to every subscriber, which is the bus's fan-out. The bundle is meant for visualization, deployment specs and event-log conformance checking. An actor's behaviour can also carry its own net (`WithNet`), which `SimulateBehavior` integrates with the ODE solver. Analysing the bundle for deadlock does not prove the running system free of it, since the handlers sit outside the net.

## Sensitivity Analysis

The `sensitivity` package measures how changes in parameters affect simulation outcomes ([`sensitivity/sensitivity.go`](https://github.com/pflow-xyz/go-pflow/blob/main/sensitivity/sensitivity.go)):

```go
sa := sensitivity.NewAnalyzer(net, baseState, rates, sensitivity.PlaceScorer("R"))
result := sa.AnalyzeRates()

for _, r := range result.Ranking {
    fmt.Printf("%s: impact=%.4f\n", r.Name, r.Impact)
}
```

`AnalyzeRates` sets each transition's rate to zero in turn, re-simulates (Tsit5, `DefaultOptions`, t = 0 to 10 unless `WithTimeSpan` changes it), and records the change in score from the baseline; `Ranking` sorts the transitions by the size of that change. For small perturbations rather than removal, `Gradient` and `AllGradients` take central differences in each rate, and `SweepRate` scans one rate over a list of values. A transition whose rate moves the score most is a candidate bottleneck. This connects to the bottleneck analysis in [Chapter 5: Resource Modeling](ch05-resource-modeling.md) and the what-if scenarios in [Chapter 11: Process Mining](ch11-process-mining.md).

## Process Mining Integration

The mining and monitoring packages (covered in depth in [Chapter 11](ch11-process-mining.md)) connect external data to Petri net models:

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

`Discover` accepts `"sequential"`, `"common-path"`, `"alpha"` and `"heuristic"`. The same `*petri.PetriNet` type produced by the builder API or the mining algorithms feeds into the solver, the reachability analyzer, and the renderer. The library doesn't distinguish between hand-crafted and data-discovered models.

## SVG Rendering

The `visualization` package generates SVG diagrams from nets ([`visualization/svg.go`](https://github.com/pflow-xyz/go-pflow/blob/main/visualization/svg.go)):

```go
svg, err := visualization.RenderSVG(net)
```

There is no layout step. `RenderSVG` converts the net to pflow.xyz's JSON-LD format and draws each place and transition at its stored coordinates — for a builder-made net, the left-to-right positions the builder assigned. Places are drawn as circles with their tokens, transitions as rectangles, and arcs as lines with arrowheads; the renderer is adapted from pflow.xyz's, so the output resembles the editor's view ([Chapter 17: The Visual Editor](ch17-visual-editor.md)). `SaveSVG` writes the result to a file, and `RenderWorkflowSVG` and `RenderStateMachineSVG` draw workflows and statecharts in their own notations.

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

The shared `*petri.PetriNet` type is what lets them compose: the builder, the miners, the statechart and workflow compilers produce it, and the solver, the reachability analyzer, the evaluators and the renderer consume it. The token-model and metamodel packages carry a second representation for the declarative schemas of [Chapter 4: The Token Language](ch04-token-language.md), with a bridge between the two, so "one type everywhere" holds for the simulation and analysis packages described here rather than for the whole library.
