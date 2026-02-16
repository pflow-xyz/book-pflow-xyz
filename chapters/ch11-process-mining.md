# Process Mining

**Learning objective**: Discover Petri net models from event logs and use them for prediction.

The previous chapters built Petri net models by hand — designing places, transitions, and arcs to capture known rules. But what about systems where the rules aren't written down? Where the actual process diverges from the official procedure? Process mining reverses the direction: instead of building a model and running it, you collect data from a running system and discover the model automatically.

This chapter covers the complete pipeline: event logs as raw material, discovery algorithms that extract Petri net structure, timing analysis that learns transition rates, and predictive monitoring that uses the learned model to forecast outcomes before they happen.

## Event Logs: The Raw Material

Every information system generates event logs. Hospital admissions systems record when patients register, get triaged, see a doctor, and are discharged. ERP systems record when orders are placed, approved, shipped, and invoiced. The events are already there — process mining extracts structure from them.

An event log has three required fields:

| Field | Example | Purpose |
|-------|---------|---------|
| **Case ID** | P101 | Which process instance |
| **Activity** | Registration | What happened |
| **Timestamp** | 2024-11-21 08:15:00 | When it happened |

Optional fields — resource (who did it), lifecycle (start/complete), cost, outcome — enrich the analysis but aren't required for discovery.

A **trace** is the sequence of activities for a single case. Patient P101's trace might be:

```
Registration → Triage → Doctor_Consultation → Discharge
```

Patient P102, who needed lab work:

```
Registration → Triage → Doctor_Consultation → Lab_Test → Results_Review → Discharge
```

These are **process variants** — different paths through the same system. In a typical hospital ER, the simple path (no lab, no imaging) accounts for about 60% of cases. Adding lab tests covers another 30%. X-rays, surgery, and other branches handle the remaining 10%. Process mining discovers these variants automatically from the log data.

### Loading Event Logs

```go
config := eventlog.DefaultCSVConfig()
log, _ := eventlog.ParseCSV("hospital_er.csv", config)

summary := log.Summarize()
// Parsed 1000 cases with 5200 events
// Found 6 unique activities
// Discovered 4 process variants
```

The `eventlog` package handles CSV and JSONL formats with configurable column mappings. The summary immediately reveals the process's complexity: how many variants exist, how many activities are involved, and how many cases the log contains.

## Process Discovery

Given an event log, a discovery algorithm produces a Petri net that explains the observed behavior. Three algorithms illustrate the design space.

### Common-Path Discovery

The simplest approach: find the most frequent trace variant and build a linear Petri net for it.

```
Variant 1 (60% of cases): Reg → Triage → Doctor → Discharge

Discovered model:
[start] → Reg → [p1] → Triage → [p2] → Doctor → [p3] → Discharge → [end]
```

Each activity becomes a transition. Places are inserted between transitions to carry the control flow token. A token in `start` means the case has arrived; a token in `end` means it's complete; a token in `p2` means the patient has been triaged and is waiting for the doctor.

Common-path discovery is fast and always produces a valid model, but it ignores all variant behavior. It's useful as a starting point or when the process is genuinely linear.

```go
discovery, _ := mining.Discover(log, "common-path")
net := discovery.Net
// Model covers 60% of cases
```

### The Alpha Algorithm

The Alpha algorithm, introduced by van der Aalst in 2004, discovers concurrent structure — not just sequences. It works by analyzing **ordering relations** between activities.

The algorithm builds a **footprint matrix** from the event log. For every pair of activities (A, B), it classifies their relationship:

| Relation | Symbol | Meaning |
|----------|--------|---------|
| Causality | A → B | A directly precedes B, never vice versa |
| Parallel | A ‖ B | Both orderings observed (A before B and B before A) |
| Choice | A # B | Neither ordering observed (exclusive alternatives) |

Given traces:
```
Case 1: A → B → D → E
Case 2: A → C → D → E
```

The footprint matrix reveals: A → B (causal), A → C (causal), B # C (choice — they never co-occur), B → D and C → D (both causal), D → E (causal).

From these relations, the Alpha algorithm constructs a Petri net:

```
       → [B] →
[A] →          [D] → [E]
       → [C] →
```

B and C are alternative paths after A, both converging at D. The algorithm discovers this branching structure automatically — no human specified that B and C are alternatives.

The implementation builds place candidates from maximal pairs of causally connected activity sets, then filters to keep only maximal candidates:

```go
miner := mining.NewAlphaMiner(log)
net := miner.Mine()
```

The Alpha algorithm has known limitations: it can't handle loops of length 1 or 2, and it's sensitive to noise in the log. For real-world data, the heuristic miner is more robust.

### Heuristic Miner

The heuristic miner adds noise tolerance by counting how often each activity pair occurs and applying thresholds:

```
Observations:
A → B: 100 times
A → C: 95 times
B → D: 85 times
C → X: 2 times   ← noise (only 2% of cases)
```

With a 5% threshold, the C → X edge is filtered out. The resulting model is cleaner and more representative of the actual process, at the cost of ignoring rare behavior.

## Timing Analysis and Rate Learning

Discovery tells us the process structure — which activities exist and how they connect. Timing analysis tells us the dynamics — how long each activity takes and how fast cases flow through the system.

### Extracting Timing Statistics

```go
stats := mining.ExtractTiming(log)

// Per-activity statistics
meanTriage := stats.GetMeanDuration("Triage")       // 615 seconds (10.3 min)
stdTriage := stats.GetStdDuration("Triage")          // 192 seconds
rateTriage := stats.EstimateRate("Triage")           // 0.001626 /sec
```

The `ExtractTiming` function computes, for each activity: mean duration, standard deviation, occurrence count, and estimated rate. It also computes inter-arrival times (how frequently new cases appear) and total case durations (end-to-end cycle time).

### From Durations to Rates

The key conversion: $\text{rate} = 1 / \text{mean duration}$. If triage takes an average of 10 minutes, the triage rate is $1/600 = 0.00167$ per second. This assumes exponentially distributed service times — the same memoryless property that mass-action kinetics uses.

For a hospital ER with six activities:

| Activity | Mean Duration | Estimated Rate |
|----------|---------------|----------------|
| Registration | 12.5 min | 0.00133 /sec |
| Triage | 27.5 min | 0.000606 /sec |
| Doctor_Consultation | 31.2 min | 0.000533 /sec |
| Lab_Test | 97.5 min | 0.000171 /sec |
| Results_Review | 50.0 min | 0.000333 /sec |
| Discharge | 15.0 min | 0.00111 /sec |

The bottleneck is immediately visible: Lab_Test has the lowest rate, meaning it takes the longest. Any effort to reduce end-to-end time should target the lab.

### Learning Rates for Simulation

The learned rates plug directly into the ODE solver:

```go
rates := mining.LearnRatesFromLog(log, net)

initialState := net.SetState(nil)
prob := solver.NewProblem(net, initialState, [2]float64{0, 10000}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
```

The simulation now uses rates derived from actual data rather than guesses. The ODE dynamics reproduce the observed flow patterns: tokens accumulate before the lab (the bottleneck), flow smoothly through registration and triage (fast steps), and drain out through discharge.

This closes the loop: **event log → discovered model → learned rates → simulation → prediction**. The same Petri net formalism works for hand-crafted models (Parts I and II) and data-driven models (this chapter). The mathematics doesn't care where the model came from.

## Predictive Monitoring

With a learned model, we can predict the future. A patient who has completed registration and triage but not yet seen a doctor — what's their expected remaining time? Will they violate the 4-hour SLA?

### State Estimation

The first step is mapping the patient's event history to a Petri net marking. Each observed activity corresponds to firing a transition:

```go
currentState := monitoring.EstimateCurrentState(activeCase, net)
```

The function replays the event history through the net: start with a token in the `start` place, fire each observed transition in sequence, and the resulting marking represents the patient's current position in the process. If the patient has completed Registration and Triage, the token sits in the place between Triage and Doctor_Consultation.

### Simulation-Based Prediction

From the current state, the ODE solver simulates forward to predict completion:

```go
predictor := monitoring.NewPredictor(net, rates)
pred := predictor.PredictFromState(currentState, currentTime)

remainingTime := pred.PredictedEndTime - pred.CurrentTime
confidence := pred.Confidence
```

The predictor watches for a token to arrive in the `end` place. The time at which the token crosses a threshold (0.5) is the predicted completion time. The confidence score reflects how much token mass reaches the end — higher confidence means the simulation is more certain about the outcome.

### SLA Violation Detection

The real value is early warning. At 9:30 AM, a patient who arrived at 8:00 AM has been in the system for 90 minutes. The model predicts 180 more minutes of processing (lab test is the bottleneck). Total predicted time: 270 minutes. The 4-hour SLA (240 minutes) will be violated.

The alert fires 2.5 hours before the violation — enough time for intervention. Expedite the lab order, assign additional staff, or notify the patient about the expected wait.

### Next Activity Prediction

Beyond completion time, the model predicts which activity happens next and how likely each option is:

```go
predictions := monitoring.PredictNextActivity(activeCase, predictor)
// [{Activity: "Doctor_Consultation", Probability: 0.85, ExpectedTime: 15min},
//  {Activity: "Lab_Test", Probability: 0.15, ExpectedTime: 45min}]
```

The prediction uses mass-action kinetics: each enabled transition fires at a rate proportional to its rate constant times the product of its input place tokens. The transition with the highest effective rate is most likely to fire first. The probability is the ratio of each transition's effective rate to the total.

## The Hospital ER Case Study

Putting it all together with the mining demo:

```go
// 1. Parse event log
log, _ := eventlog.ParseCSV("hospital.csv", config)
// 1000 cases, 5200 events, 6 activities, 4 variants

// 2. Extract timing
stats := mining.ExtractTiming(log)
// Registration: 12.5 min, Triage: 27.5 min, Doctor: 31.2 min
// Lab_Test: 97.5 min (bottleneck), Results_Review: 50.0 min

// 3. Discover model
discovery, _ := mining.Discover(log, "common-path")
net := discovery.Net
// 7 places, 6 transitions, covers 60% of cases

// 4. Learn rates
rates := mining.LearnRatesFromLog(log, net)

// 5. Simulate
prob := solver.NewProblem(net, net.SetState(nil), [2]float64{0, 10000}, rates)
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())

// 6. Use for prediction
predictor := monitoring.NewPredictor(net, rates)
```

The discovered model matches the observed behavior: tokens flow from registration through discharge, accumulating before the lab (the bottleneck). The simulation predicts an average case duration of ~166 minutes, consistent with the timing statistics extracted from the log.

### What-If Analysis

The learned model enables scenario planning. "What if we add another lab technician?" — double the Lab_Test rate and re-simulate. "What if we add a fast-track for simple cases?" — add a bypass transition from Doctor to Discharge with a guard on case complexity.

Each scenario is a parameter change or structural modification to the same Petri net. The ODE solver produces updated predictions in milliseconds. No new data collection needed — just adjust the model and simulate.

## Conformance Checking

Discovery produces a model. Conformance checking asks: how well does the model match reality?

**Fitness** measures how many traces in the log can be replayed on the model. If the model says A must precede B, but a trace shows B before A, that trace doesn't fit. Fitness = (fitting traces) / (total traces). A fitness of 1.0 means every trace is consistent with the model.

**Precision** measures how much of the model's allowed behavior actually occurs. A model that allows everything has perfect fitness but zero precision — it's too general to be useful. A model that captures only the observed paths has high precision.

Good models balance both: high fitness (captures real behavior) with high precision (doesn't over-generalize). The trade-off is complexity — more complex models fit more traces but allow more unobserved paths.

```go
conf := mining.CheckConformance(log, net)
// Fitness: 0.87, FittingTraces: 870/1000
```

The 13% non-fitting traces represent process variants not captured by the common-path model. Switching to the Alpha algorithm or heuristic miner would increase fitness at the cost of model complexity.

## From Data to Model to Prediction

Process mining completes the Petri net toolkit. Parts I and II showed how to build models from domain knowledge — encoding rules as structure. This chapter shows how to extract models from data — discovering rules from behavior.

The pipeline:

1. **Collect** event logs from existing systems
2. **Discover** process structure using mining algorithms
3. **Learn** transition rates from timestamps
4. **Validate** via conformance checking and simulation
5. **Predict** using ODE simulation with learned parameters
6. **Intervene** based on early warnings

The same Petri net formalism handles both directions. A coffee shop model built by hand (Chapter 5) and a hospital process discovered from logs produce the same mathematical object: places, transitions, arcs, rates. The ODE solver doesn't know or care which one was hand-crafted and which was mined. The incidence matrix, conservation laws, and simulation dynamics work identically.

This is the practical payoff of a universal formalism. Process mining connects Petri nets to the real world — any system that generates event logs can be modeled, analyzed, and predicted.
