# Process Mining

**Learning objective**: Discover Petri net models from event logs and use them for prediction.

The previous chapters built Petri net models by hand — designing places, transitions, and arcs to capture known rules. But what about systems where the rules aren't written down? Where the actual process diverges from the official procedure? Process mining reverses the direction: instead of building a model and running it, you collect data from a running system and discover the model automatically.

This chapter covers the complete pipeline: event logs as raw material, discovery algorithms that extract Petri net structure, timing analysis that learns transition rates, and predictive monitoring that uses the learned model to forecast outcomes before they happen.

## Event Logs: The Raw Material

Most information systems already record events. Hospital admissions systems record when patients register, get triaged, see a doctor, and are discharged. ERP systems record when orders are placed, approved, shipped, and invoiced. The events are already there — process mining extracts structure from them.

An event log has three required fields:

| Field | Example | Purpose |
|-------|---------|---------|
| **Case ID** | P101 | Which process instance |
| **Activity** | Registration | What happened |
| **Timestamp** | 2024-11-21 08:15:00 | When it happened |

Optional fields — resource (who did it), lifecycle (start/complete), cost, outcome — enrich the analysis but aren't required for discovery.

A **trace** is the sequence of activities for a single case. Patient P101's trace might be:

```
Registration -> Triage -> Doctor_Consultation -> Discharge
```

Patient P102, who needed lab work:

```
Registration -> Triage -> Doctor_Consultation -> Lab_Test -> Results_Review -> Discharge
```

These are **process variants** — different paths through the same system. How often each variant occurs is itself something the log records; discovery starts by counting variants, and the simplest algorithm keeps only the most frequent one.

### Loading Event Logs

```go
config := eventlog.DefaultCSVConfig()
log, _ := eventlog.ParseCSV("hospital.csv", config)

summary := log.Summarize()
// NumCases 4, NumEvents 26, NumActivities 11, NumVariants 4
```

The `eventlog` package handles CSV and JSONL formats with configurable column mappings ([`eventlog/csv.go`](https://github.com/pflow-xyz/go-pflow/blob/main/eventlog/csv.go), [`eventlog/jsonl.go`](https://github.com/pflow-xyz/go-pflow/blob/main/eventlog/jsonl.go)). The log used throughout this chapter is go-pflow's [`examples/mining_demo/hospital.csv`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/mining_demo/hospital.csv): four patients, 26 events, eleven distinct activities, and four variants, because no two patients took the same path. The counts in the comment are read off the CSV, not from a re-run of the demo. Four cases are enough to show every step of the pipeline and far too few to trust any number it produces; the chapter uses them only for the first purpose.

## Process Discovery

Given an event log, a discovery algorithm produces a Petri net that explains the observed behavior. Three algorithms illustrate the design space.

### Common-Path Discovery

The simplest approach: find the most frequent trace variant and build a linear Petri net for it.

```
Most frequent variant: Reg -> Triage -> Doctor -> Discharge

Discovered model:
[start] -> Reg -> [p1] -> Triage -> [p2] -> Doctor -> [p3] -> Discharge -> [end]
```

Each activity becomes a transition. Places are inserted between transitions to carry the control flow token. A token in `start` means the case has arrived; a token in `end` means it's complete; a token in `p2` means the patient has been triaged and is waiting for the doctor.

Common-path discovery is fast and always produces a sequential net, but it ignores every other variant. It's useful as a starting point or when the process is genuinely linear.

```go
discovery, _ := mining.Discover(log, "common-path")
net := discovery.Net
// discovery.CoveragePercent: share of cases that follow the chosen variant exactly
```

On the hospital log every variant occurs exactly once, so "most frequent" is a four-way tie. [`DiscoverCommonPath`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/discovery.go) breaks it by iterating over a Go map, whose order is randomized, so repeated runs can return different nets; each covers 25% of cases. The net committed as [`discovered_hospital_net.json`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/mining_demo/discovered_hospital_net.json) is patient P003's path, Registration → Triage → Doctor_Consultation → Prescription → Discharge: six places and five transitions.

### The Alpha Algorithm

The Alpha algorithm, introduced by van der Aalst in 2004, discovers concurrent structure — not just sequences. It works by analyzing **ordering relations** between activities.

The algorithm builds a **footprint matrix** from the event log. For every pair of activities (A, B), it classifies their relationship:

| Relation | Symbol | Meaning |
|----------|--------|---------|
| Causality | A → B | B directly follows A somewhere, never the reverse |
| Parallel | A ‖ B | Each directly follows the other somewhere |
| Choice | A # B | Neither directly follows the other |

Given traces:
```
Case 1: A -> B -> D -> E
Case 2: A -> C -> D -> E
```

The footprint matrix reveals: A → B (causal), A → C (causal), B # C (choice — neither directly follows the other), B → D and C → D (both causal), D → E (causal).

From these relations, the Alpha algorithm constructs a Petri net:

```
       -> [B] ->
[A] ->          [D] -> [E]
       -> [C] ->
```

B and C are alternative paths after A, both converging at D. The algorithm discovers this branching structure automatically — no human specified that B and C are alternatives.

The implementation, [`mining/alpha.go`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/alpha.go), enumerates candidate places (X, Y), where every activity in X is causally followed by every activity in Y and the activities within X, and within Y, are pairwise in choice. It then keeps only the maximal candidates. The footprint is built in [`mining/footprint.go`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/footprint.go), and [`mining/mining_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/mining_test.go) exercises the miner on sequential and parallel logs. The A–E example above is worked by hand; it is not one of those tests.

```go
miner := mining.NewAlphaMiner(log)
net := miner.Mine()
```

The Alpha algorithm has known limitations, listed in the source: it can't handle loops of length 1 or 2, and it's sensitive to noise in the log. For real-world data, the heuristic miner is more robust.

### Heuristic Miner

The heuristic miner tolerates noise by scoring each ordered pair of activities from directly-follows counts, instead of treating a single observation as a relation. [`mining/heuristic.go`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/heuristic.go) uses the dependency measure

$$a \Rightarrow b = \frac{|a > b| - |b > a|}{|a > b| + |b > a| + 1}$$

where $|a > b|$ counts how often $b$ directly follows $a$. An edge is kept when its score reaches `DependencyThreshold`, which defaults to 0.5:

```
Observations               Score
A -> B: 100, B -> A: 0     100/101 ≈ 0.99
A -> B: 100, B -> A: 3      97/104 ≈ 0.93
C -> X: 2,   X -> C: 0       2/3   ≈ 0.67
C -> X: 1,   X -> C: 0       1/2   = 0.50
```

A few reversed observations barely dent a strong relation, and that is where the noise tolerance comes from. A rare edge seen in only one direction is another matter: at the default threshold even a single observation scores exactly 0.5 and survives. Filtering the two-observation C → X edge takes a stricter threshold:

```go
opts := mining.DefaultHeuristicOptions()
opts.DependencyThreshold = 0.8 // drops one-way edges seen fewer than four times
res, _ := mining.DiscoverHeuristicWithOptions(log, opts)
```

The scores are computed by hand from the formula, not from a run. A higher threshold gives a cleaner model at the cost of ignoring rare behavior, which may be exactly the behavior you wanted to find.

## Timing Analysis and Rate Learning

Discovery tells us the process structure — which activities exist and how they connect. Timing analysis tells us the dynamics — how long each activity takes and how fast cases flow through the system.

### Extracting Timing Statistics

```go
stats := mining.ExtractTiming(log)

// Per-activity statistics, hospital.csv
meanTriage := stats.GetMeanDuration("Triage") // 1650 seconds (27.5 min)
stdTriage := stats.GetStdDuration("Triage")   // 300 seconds
rateTriage := stats.EstimateRate("Triage")    // 0.000606 /sec
```

[`ExtractTiming`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/timing.go) collects, for each activity, its durations and occurrence count; the methods above turn those into a mean, a sample standard deviation and a rate. It also records inter-arrival times (how frequently new cases appear) and total case durations (end-to-end cycle time). The log has one timestamp per event and no start/complete lifecycle, so an activity's "duration" is the time from its event to the next event in the same case, which includes any wait before the next step. The last event of a case has no successor and contributes no duration. Triage's four observations are 30, 30, 20 and 30 minutes, which give the values in the comments (worked by hand from the CSV).

### From Durations to Rates

The key conversion: $\text{rate} = 1 / \text{mean duration}$. If triage takes an average of 10 minutes, the triage rate is $1/600 = 0.00167$ per second. This assumes exponentially distributed service times — the same memoryless property that mass-action kinetics uses.

For the hospital log (worked by hand from the CSV; the first five rates match the learned rates printed in go-pflow's [`mining/README.md`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/README.md)):

| Activity | Observations | Mean Duration | Estimated Rate |
|----------|--------------|---------------|----------------|
| Registration | 4 | 12.5 min | 0.00133 /sec |
| Triage | 4 | 27.5 min | 0.000606 /sec |
| Doctor_Consultation | 4 | 31.2 min | 0.000533 /sec |
| Lab_Test | 2 | 97.5 min | 0.000171 /sec |
| Results_Review | 3 | 50.0 min | 0.000333 /sec |
| Discharge | 0 | — | 0.1 /sec (default) |

Discharge is the last event of every case, so it has no duration at all, and `EstimateRate` falls back to a default of 0.1 per second: a ten-second mean that comes from the code, not the data. The five activities seen once each (X-Ray, Surgery, Recovery, Prescription, Specialist_Referral) are left out of the table.

Lab_Test has the lowest rate among the activities observed more than once, which makes it the first candidate for the bottleneck. With two observations it is a candidate, not a finding, and Surgery's single 180-minute observation is longer still.

### Learning Rates for Simulation

The learned rates plug directly into the ODE solver (from [`examples/mining_demo/main.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/mining_demo/main.go)):

```go
rates := mining.LearnRatesFromLog(log, net)

initialState := net.SetState(nil)
prob := solver.NewProblem(net, initialState, [2]float64{0, 10000}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
```

The simulation now uses rates derived from the log rather than guesses. What it does with them follows from go-pflow's rate law ([Appendix A: Solver Parameter Reference](appendix-a-solver-reference.md)): a transition's flux is its rate times the product of its input places' tokens, each to the first power. Every transition in a common-path net has a single input place, so the net is a cascade of first-order decays. The token's mass lingers longest in the place whose outgoing rate is smallest, and the mass in `end` at time $t$ is the probability that a case with these exponentially distributed step times has finished by $t$. Its mean completion time is therefore the sum of the step means. For the committed Prescription-path net that is 12.5 + 27.5 + 31.2 + 15 minutes plus Discharge's ten seconds, about 86 minutes; for P001's path through Lab_Test and Results_Review it is about 219 minutes. Both figures are derived from the rate law, not re-run.

This closes the loop: **event log → discovered model → learned rates → simulation → prediction**. The same Petri net formalism works for hand-crafted models (Parts I and II) and data-driven models (this chapter).

## Predictive Monitoring

With a learned model, we can predict the future. A patient who has completed registration and triage but not yet seen a doctor — what's their expected remaining time? Will they violate the 4-hour SLA?

### State Estimation

The first step is mapping the patient's event history to a Petri net marking. Each observed activity corresponds to firing a transition:

```go
currentState := monitoring.EstimateCurrentState(activeCase, net)
```

The function replays the event history through the net: start with a token in the `start` place, fire each observed transition in sequence, and the resulting marking represents the patient's current position in the process. If the patient has completed Registration and Triage, the token sits in the place between Triage and Doctor_Consultation.

[`monitoring/predictor.go`](https://github.com/pflow-xyz/go-pflow/blob/main/monitoring/predictor.go) assumes the net has places named `start` and `end`, which the common-path and Alpha miners both produce. The replay is best-effort: an activity the net does not contain is skipped, and a transition that is not enabled is fired anyway, with its input places clamped at zero. A case that has left the model's path therefore shows up as a marking that makes no sense, not as an error.

### Simulation-Based Prediction

From the current state, the ODE solver simulates forward to predict completion:

```go
predictor := monitoring.NewPredictor(net, rates)
pred := predictor.PredictFromState(currentState, currentTime)

remainingTime := pred.PredictedEndTime - pred.CurrentTime
confidence := pred.Confidence
```

The predictor simulates up to 24 hours ahead and reports the first time the token mass in `end` reaches 0.5 as the predicted completion time. On a chain like the one above, that is the median of the remaining-time distribution, not its mean. The confidence is the mass in `end` at the 24-hour horizon: how much of the case the model expects to have finished by then, not a calibrated probability that the prediction is right.

### SLA Violation Detection

Suppose a patient who arrived at 8:00 AM is still in the system at 9:30, 90 minutes in, and the model predicts 180 more minutes because the next step is the lab. The predicted total of 270 minutes exceeds a 4-hour (240-minute) SLA, so the alert fires two and a half hours before the breach would happen. That leaves time to expedite the lab order, assign additional staff, or tell the patient about the wait.

Those figures are illustrative arithmetic, not model output. [`examples/monitoring_demo/main.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/monitoring_demo/main.go) runs the real loop: it learns from three synthetic historical cases, streams three patients against a 4-hour SLA, and raises alerts through `monitoring.Monitor`. Its [README](https://github.com/pflow-xyz/go-pflow/blob/main/examples/monitoring_demo/README.md) shows sample output, which was not re-run for this chapter.

### Next Activity Prediction

Beyond completion time, the model predicts which activity happens next and how likely each option is:

```go
predictions := monitoring.PredictNextActivity(activeCase, predictor)
// []NextActivity{Activity, Probability, ExpectedTime}, one per enabled transition
```

Each enabled transition gets an effective rate, its rate constant times the product of its input places' tokens, and its probability of firing first is its share of the total: the race between exponential clocks. Two consequences follow from the code. On a common-path net only one transition is ever enabled, so the prediction is a single activity with probability 1; branch probabilities need a model with choices, such as the Alpha miner's. And because the rate constants are learned from durations, at a choice the faster activity is predicted to be the likelier one. How often each branch was taken in the log does not enter the calculation.

## The Hospital ER Case Study

Putting it all together, condensed from [`examples/mining_demo/main.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/mining_demo/main.go), with the values worked by hand from `hospital.csv` in the comments:

```go
// 1. Parse event log
log, _ := eventlog.ParseCSV("hospital.csv", config)
// 4 cases, 26 events, 11 activities, 4 variants

// 2. Extract timing
stats := mining.ExtractTiming(log)
// Registration 12.5 min, Triage 27.5, Doctor_Consultation 31.2,
// Lab_Test 97.5, Results_Review 50.0; Discharge has no duration

// 3. Discover model
discovery, _ := mining.Discover(log, "common-path")
net := discovery.Net
// four-way tie between variants; the chosen one covers 25% of cases

// 4. Learn rates
rates := mining.LearnRatesFromLog(log, net)

// 5. Simulate
prob := solver.NewProblem(net, net.SetState(nil), [2]float64{0, 10000}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())

// 6. Use for prediction
predictor := monitoring.NewPredictor(net, rates)
```

The four cases in the log took 210, 510, 60 and 240 minutes, a mean of 255. A single common-path net cannot reproduce that spread: it models one path, and its mean completion time is about 86 or 219 minutes, depending on which variant won the tie (derived above). The demo also integrates for only 10,000 seconds, about 167 minutes, which stops before the Lab_Test path's mean completion time. With four cases, the case study shows the pipeline running end to end; it does not validate the numbers.

### What-If Analysis

The learned model enables scenario planning. "What if we add another lab technician?" — double the Lab_Test rate and re-simulate. "What if we add a fast-track for simple cases?" — add a bypass transition from Doctor to Discharge with a guard on case complexity.

Each scenario is a parameter change or a structural change to the same Petri net. A rate change needs no new data. A structural change does: the bypass transition needs a rate, and the log cannot supply one until the bypass exists.

## Conformance Checking

Discovery produces a model. Conformance checking asks: how well does the model match reality?

**Fitness** measures how well the log replays on the model. If the model says A must precede B, but a trace shows B before A, that trace doesn't fit. [`CheckConformance`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/conformance.go) uses token-based replay: it fires each trace's activities in order and counts tokens produced ($p$), consumed ($c$), missing ($m$: needed but absent, plus one for each activity the model lacks) and remaining ($r$: left outside a final place at the end). It reports

$$\text{fitness} = \tfrac{1}{2}\left(1 - \tfrac{m}{c}\right) + \tfrac{1}{2}\left(1 - \tfrac{r}{p}\right).$$

A trace fits when it has no missing and no remaining tokens, and `FittingTraces` counts those separately. A fitness of 1.0 means every trace replays cleanly.

**Precision** measures how much of the model's allowed behavior actually occurs. `CheckPrecision` counts escaping edges, transitions enabled in a replayed state but never taken there, and reports 1 − escaping/enabled. A model that allows everything fits every trace but has low precision, because it is too general to be useful. A model that captures only the observed paths has high precision.

Good models balance both: high fitness (captures real behavior) with high precision (doesn't over-generalize). Adding paths to a model usually raises the first and lowers the second.

```go
conf := mining.CheckConformance(log, net)
// committed Prescription-path net vs hospital.csv:
// Fitness ≈ 0.58, FittingTraces 1 of 4
```

Those values are worked by hand from the replay rules in `conformance.go`, not re-run. P003, the variant the net was built from, replays cleanly. The other three are charged one missing token for each activity the net does not contain, find Discharge's input place empty when they rejoin the path, and leave a token stranded mid-net. Summed over the log, $m = 12$, $c = 17$, $r = 3$ and $p = 21$. The non-fitting traces are the variants common-path discarded. A miner that keeps them, Alpha or heuristic, should fit better at some cost in precision; that comparison has not been run on this log.

## Generator and Recognizer: Closing the Loop

Process mining infers structure from observations. The complementary
move is *executing* a declared structure to produce observations.
Both end at the same artifact — a Petri net — but they travel in opposite
directions.

Take a streaming pipeline of the kind Apache Beam popularised: a chain of
windowed aggregations over a keyed event stream. The `tokenmodel/dataflow`
package lowers such a pipeline to a bundle of Petri net subnets — one per
key, one per (key, window) accumulator, one for the watermark. Feed the
pipeline an event stream and every order becomes a token in the net. The
pipeline is the *generator*.

Mining is the *recognizer*. Give it the execution trace and ask it to
recover the structure that produced it:

```go
log := pipeline.ToEventLog()
res, _ := mining.DiscoverPipeline(log, mining.PipelineDiscoveryOptions{
    IdealEventsPerWindow: 7,
})
// Truth: kind=fixed, size=60, 6 keys. Recovered: kind=fixed, size=50, 6 keys.
```

[`loop_closure_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/coffeeshop/dataflow/loop_closure_test.go)
runs the coffee shop's sample order stream (58 orders over eight hours)
through a pipeline with fixed 60-minute windows, exports the input
history, and hands it to
[`DiscoverPipeline`](https://github.com/pflow-xyz/go-pflow/blob/main/mining/dataflow_discovery.go).
The test asserts that the recovered window kind and key count match the
spec, and that the window size is within a factor of two of 60. The size
is where the heuristic shows. It aims for about `IdealEventsPerWindow`
events per window, which gives a raw size of 470 × 7 / 58 ≈ 57, then snaps
to the nearest entry in a list of round sizes (…, 20, 50, 100, …) that does
not include 60. The recovered size is therefore 50, derived from the code
rather than re-run, and the same value go-pflow's mining README prints. The
stream's timing cannot reveal that 60 was chosen because it is an hour. Nor
is the trigger recovered: `DiscoverPipeline` returns after-watermark, its
default, unless the caller supplies an `EarlyFireRate`.

The two directions differ in kind. The generator is total and
deterministic: a spec fully determines the net. The recognizer is partial
and statistical: a log under-determines the net, which is why there are
algorithms like Alpha and the heuristic miner to choose between
candidates. Both end at a Petri net, and whether it was declared or
induced, it is analysed with the same incidence matrix and simulated with
the same solver.

## From Data to Model to Prediction

Parts I and II showed how to build models from domain knowledge, encoding rules as structure. This chapter shows how to extract models from data, discovering rules from behavior.

The pipeline:

1. **Collect** event logs from existing systems
2. **Discover** process structure using mining algorithms
3. **Learn** transition rates from timestamps
4. **Validate** via conformance checking and simulation
5. **Predict** using ODE simulation with learned parameters
6. **Intervene** based on early warnings

The same Petri net formalism handles both directions. A coffee shop model built by hand ([Chapter 5: Resource Modeling — The Coffee Shop](ch05-resource-modeling.md)) and a hospital process discovered from logs produce the same kind of object: places, transitions, arcs, rates. The solver and the incidence-matrix analysis do not know which one was hand-crafted and which was mined.

Any system that writes timestamped events per case can be fed to this pipeline. How far its predictions can be trusted depends on the log: the hospital log here has four cases, enough to show every step and too few to support any of the numbers.
