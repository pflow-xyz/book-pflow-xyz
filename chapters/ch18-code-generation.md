# sim.pflow.xyz — From Description to Checked Model

**Learning objective**: Store a Petri net model in a hosted service, run it as a seeded what-if, and get verdicts about it that someone else can re-check.

The visual editor (Chapter 17) produces a model. The Go library (Chapter 19) analyzes it. Between the two sits a practical question: once a model exists, where does it run, and how does someone who did not build it decide whether to believe what it says? [sim.pflow.xyz](https://sim.pflow.xyz/) is a hosted answer. You give it a model, written by hand or derived from a plain-language description or from source code, and it stores the model under a content address. You can then run seeded scenarios against it, compare them on a shared seed, rank its knobs by measured influence, and verify structural properties, with every verdict saying how far it reaches.

This chapter describes that service: its surfaces, the path a model takes through it, and the line it draws between what an engine can read and what it refuses to guess. The engines underneath are go-pflow's, at v0.33.0, the version sim's `go.mod` pins: the discrete and continuous simulators in [`stochastic`](https://github.com/pflow-xyz/go-pflow/tree/v0.33.0/stochastic), the model type in [`metamodel`](https://github.com/pflow-xyz/go-pflow/tree/v0.33.0/metamodel), the property checker in [`verify`](https://github.com/pflow-xyz/go-pflow/tree/v0.33.0/verify). sim's own source is not public, so the chapter cites its tools and HTTP routes, which anyone can call, and links go-pflow where go-pflow does the work. Tool behaviour is quoted from the tool descriptions the MCP server publishes; the example outputs below are real calls against the live service, rounded for the page.

## The Surfaces

sim.pflow.xyz is one service with three ways in:

| Surface | Where | What it is for |
|---------|-------|----------------|
| MCP server | `https://sim.pflow.xyz/mcp` | `sim_*` tools for an LLM client: store, run, diagnose, verify, prove, publish |
| HTTP API | `https://sim.pflow.xyz/api/...` | The same reads per model (`/api/models/{id}/scenario`, `/diagnose`, `/verify`, …), CORS-open |
| Consoles | [`/whatif/?id={id}`](https://sim.pflow.xyz/whatif/), [`/build/`](https://sim.pflow.xyz/build/) | A generic what-if console for any stored model, and a guided builder |

Connecting an MCP client is one line:

```bash
claude mcp add --transport http sim https://sim.pflow.xyz/mcp
```

Read tools (listing and fetching models, scenarios, comparisons, diagnosis, verification, invariants, reachability) work without signing in, and the ones that run an engine are rate-limited per client. Tools that store something under your name, call an LLM, or publish to Google Sheets ask for OAuth the first time they are used. The service's [llms.txt](https://sim.pflow.xyz/llms.txt) lists every tool and route; this chapter covers the ones that make up the main path.

## The Pipeline

```
description / source code / hand-written JSON
        |
        |  sim_generate, sim_code_to_flow, /build/, POST /api/models
        v
model (JSON, content-addressed, immutable)
        |
        +--> sim_scenario, sim_compare        seeded runs, shared-seed comparison
        +--> sim_diagnose                      knobs ranked against a noise floor
        +--> sim_verify, sim_invariants        proved / refuted / unknown
        +--> sim_crosscheck                    independent readings, agreement or divergence
        +--> sim_receipt                       a signed run anyone can replay
        +--> sim_publish_app, sim_publish      a single-file app, a Google Sheet
```

Every arrow out of the model is a read. Running a scenario cannot change the model, and a changed model is a different model.

### Step 1: The Model

A model is a go-pflow [`metamodel.Model`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/schema.go) in JSON: places, transitions and arcs, plus optional blocks that sim's engines read, such as capacities on places, rates, `stages` and a `schedule` on transitions, declared `parameters`, and a `simulation.objective`. Appendix C documents the format.

```json
{
  "name": "toggle",
  "places": [{"id": "off", "initial": 1}, {"id": "on"}],
  "transitions": [{"id": "switch", "rate": 1.0}],
  "arcs": [{"from": "off", "to": "switch"}, {"from": "switch", "to": "on"}]
}
```

It is stored with `sim_create_model` (signed in, the model is yours) or `POST /api/models` (anonymous, stored unowned, reachable by anyone holding its id and listed for nobody). The id is a content address: the sha256 of the model's canonical JSON. Two uploads of the same model are the same model, an id can be cached forever, and nothing is mutated. Editing a model means storing a new one; `sim_extend` records the edit with lineage, and `sim_supersede_model` points the listing at the successor without moving anything else.

Storing refuses rather than guesses. The decoder reads the legal keys from go-pflow's Go type by reflection and refuses any key the model schema does not define, naming each with its path and the key it most likely meant: `places[0].intial` comes back with a suggestion of `initial`. The reason is written into the decoder: Go's JSON decoder silently drops an unknown key, so a misspelt `initial` would have stored a net with an empty place, and every run of it would have reported zeros with full confidence. Structural validation then reports every defect at once instead of the first one: a model with no name, places or transitions, a duplicated id, an arc whose endpoint does not exist, and go-pflow's own checks on arcs, parameters and schedules. It gates structure, not quality. A well-formed net is storable whether or not it is a good model of anything.

### Step 2: Getting a Model Without Writing JSON

Three routes produce a model from something less formal. All three end in the same store, under the same validation.

**From a description.** `sim_generate` asks the deployment's LLM to write a net from plain-language requirements ("a lunch counter with two cooks and a ten-seat queue"). Each answer is checked as it arrives: it must decode, pass the validator, and then pass behavioural checks for disconnected parts, transitions that never fire, places nothing fills, and any place ids the caller required. Whatever is wrong is shown back to the LLM, and it answers again, up to `retries` times (0–4, default 2). `best_of` (1–5) starts again from scratch when a sample still has problems and stops at the first clean one. `selfcheck` asks the LLM to play its own net forward and replays that play against the net, which catches inconsistency, though not a wrong idea. A net that decodes and validates is stored even when behavioural problems remain. The response says whether it is `clean` and lists the `problems`, so a stored model is not a passed model.

**From source code.** `sim_code_to_flow` takes a code snippet, a language hint and a focus (`control-flow`, the default, or `state-machine`, `resources` or `concurrency`) and derives a net from it with the same LLM. The answer goes through the same decoder as a hand-written model, then a stricter validator that also explores the net's reachable states with go-pflow's [`reachability`](https://github.com/pflow-xyz/go-pflow/tree/v0.33.0/reachability) package. When the answer validates, it is stored as a new model; when it does not, the raw JSON and the validation errors come back so the caller can fix it and store it by hand.

**Interactively.** The [builder](https://sim.pflow.xyz/build/) is a guided conversation that edits a draft. After each turn it explores the draft's reachable markings through go-pflow's own firing rule (`Model.Enabled` and `Fire` in [`metamodel/firing.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/firing.go)), so its findings about dead transitions, unreachable places, deadlocks and unbounded growth are the engine's answer and not a second reading of the arcs. The exploration is bounded by a state count, a time budget and the request. When a bound is hit, claims that need the whole space ("this never fires") are withheld rather than guessed.

Both LLM routes are only as good as the model they call, and both need a deployment with an LLM provider configured. The checks bound what a wrong net can get away with structurally. They cannot tell whether the net describes the process the caller had in mind. That is what the instruments in the next steps are for.

### Step 3: Run It

`sim_scenario` runs a seeded what-if: marking overrides ("three staff instead of two"), rate overrides, piecewise rate schedules, and values for the model's declared structural parameters. Unset fields default to 8 hours, 16 realizations and seed 20260809. The service refuses, and does not clamp, more than 200 realizations, 10,000 hours or 5,000 samples. The result carries the trajectory (mean and standard deviation per place on a sample grid), final marking, time-weighted metrics, the places that ran out, contention, and the caveats and assumptions the engine wants the reader to see.

The default engine is go-pflow's discrete stochastic simulator (`stochastic.Simulate`, Gillespie's algorithm), which fires one event at a time. `"engine": "ode"` asks for the continuous mass-action reading (`stochastic.Forecast`) instead, and this is where the rule from Chapter 3 becomes an API: `Forecast` checks the model's gating (`Model.Gating()`) and refuses a net with a read arc, an inhibitor, a reachable capacity, a guard or a non-kinetic arc, because the ODE has no firing instant at which to test any of them. Asking for the ODE on the café in sim's catalog returns a refusal, not a curve:

```
diverged: true
reason: this model constrains firing in ways a continuous solution cannot
express, so the ODE would silently model an unconstrained system. Use the
discrete engine (Simulate). Specifically: 17 non-kinetic input arc(s) gate
and consume without scaling the rate; ... capacity is declared on
[pantry/coffee_beans pantry/milk pantry/cups] but is a post-firing bound,
which has no continuous analogue
```

`sim_compare` runs up to 20 scenarios against one model on one seed. The server enforces the shared seed, so a difference between two results comes from the scenarios and not from the random numbers. On the same café, with the default eight hours and 16 realizations:

| Scenario | Orders complete | Walked out | Staff utilization |
|----------|-----------------|------------|-------------------|
| two staff (as stored) | 169 ± 6 | 89 ± 13 | 0.80 |
| three staff | 217 ± 12 | 43 ± 14 | 0.68 |

The ± figures are the spread across realizations at the horizon, which the result reports beside each mean. The run also says what it assumed: every step takes an exponentially distributed time, the most erratic a shop can be for a given average, so the result labels its queues and walkouts as the bad case rather than the typical one.

### Step 4: Diagnose It

A comparison answers a question the modeler thought to ask. `sim_diagnose` asks every such question at once: what one more of each resource, and a change in each rate, is worth to the outcome, each ranked against a measured noise floor. The ranking is derived from the net, so a new model gets its fitness checks by existing, with no test written first. The report also covers generic gates (mass balance, dormant sources, a staffing knee, whether any knob binds at all) and structural readings that need no run: T-invariants, siphons and traps with deadlock witnesses, and lumpability.

The sample size adapts. Leave `realizations` unset, and the diagnosis starts with 24, doubling while the baseline outcome sits inside its own noise floor, up to 200. The report records where it settled and why. An explicit count is used exactly. The service's documentation reports that this pass has found capacity that does nothing (a café's pantry stock, a hospital's emergency bays) and, in a software-team model, capacity past which more engineers make the objective worse. Those are claims about specific catalog models, not general laws, but they are the kind of finding the tool exists to produce: "add more" gets measured instead of assumed. How a knob's influence is scored depends on what the model counts as success, which `sim_diagnose` infers from place tags or reads from a declared `simulation.objective`.

### Step 5: Verify It

Simulation samples behaviour. Verification makes claims about all of it, and sim keeps the two apart.

`sim_verify` checks declared properties: deadlock-free, bounded, live, terminating, conserves, mutual exclusion, a linear invariant over place counts, and reachable or unreachable target markings. It runs through go-pflow's `metamodel/metapetri` bridge, which converts the model into an analysable net and records what the conversion could not express, and go-pflow's [`verify`](https://github.com/pflow-xyz/go-pflow/tree/v0.33.0/verify) package. Each verdict is `proved`, `refuted` or `unknown`. Unknown is never a pass, and the report's `ok` is true only when every property is proved. Each verdict also carries a method, which says how far it reaches:

| Method | Meaning |
|--------|---------|
| `structural` | From the incidence matrix with no state search; a structural `bounded` holds for every initial marking |
| `exhaustive` | Every state reachable from this initial marking |
| `witness` | A firing sequence found by targeted search, decisive for this marking |
| `partial` | The search was truncated, so only a refutation is sound |

Run with the defaults (bounded and deadlock-free) on the same café, it returns one of each kind you would rather not see:

```
bounded        refuted   witness   counter/pending_espresso grows without bound
                                   trace: [counter/order_espresso]
deadlock-free  unknown   partial   no deadlock in the states explored, but the
                                   state space was truncated
```

Both verdicts are correct. Customer arrivals are a transition with no input place, so the net is open and its queue has no structural bound. The model relies on abandonment rates, not on structure, to keep the queue short, and the verifier says so with a one-step counterexample. The unbounded queue also means the state space cannot be exhausted, so deadlock freedom stays unknown and is not reported as a pass. The caveats list the three pantry capacities, carried into the analysis as post-firing bounds.

`sim_invariants` returns the algebra on its own: the P-invariants (conservation laws, from Farkas' algorithm), the T-invariants (firing cycles), and the minimal siphons and traps, including any siphon empty at the initial marking, which proves every transition that needs it permanently disabled. The P-invariants are the conservation laws of Chapter 2, computed for whatever model you stored.

`sim_crosscheck` runs every applicable reading of a model against the others: discrete means against the mean-field solve, derived conservation laws against simulated means and, for game models, the closed-form incidence ranking against rollouts and exact search. A divergence is a finding, not an error, and the tool names its likely cause, such as a small-count mean-field gap or a heavy arc weight, which gets a different rate law under each engine. Gated nets have no ODE reading to compare against at all.

### Step 6: Results Someone Else Can Check

A result is only as useful as a stranger's ability to check it. Three mechanisms let someone other than the modeler check a result.

**Determinism.** Models are immutable and scenarios are pure, seeded reads, so the same scenario body against the same model id on the same surface returns the same answer, byte for byte. The HTTP and MCP surfaces fill unset fields differently (HTTP defaults to one hour, one realization and seed 0), so a comparison across surfaces must set them explicitly.

**Receipts.** `sim_receipt` runs a scenario and returns, with the result, an Ed25519 signature over the model id, the scenario as run (defaults filled in), a hash of the result, and the service revision. The signature can be checked offline against the public key at `/api/receipts/key`, which shows the service reported this result. `POST /api/receipts/verify` replays the run and confirms that the result hash reproduces, which shows the result was not invented. Neither step needs an account.

**Witnesses.** The structural tools record their claims as edges with evidence that can be re-checked. `sim_canonical` computes an isomorphism-invariant id from a model's exact automorphism orbits, so two differently labelled copies of one net can be recognised. `sim_prove` decides `isomorphicTo`, `subnetOf`, `preservesInvariant` or `refines` between two stored models. A proof is stored with a witness: a map, a map with an invariant cover, or a simulation relation. A refutation carries a counterexample, and a search that runs out of budget answers unknown and records nothing. `sim_reach` asks whether one marking can be reached from another in a stored net. It answers no with a separating P-invariant or an exhaustive search, yes with a replayable firing trace, or unknown, listing which budget stopped it. `sim_check_witness` re-checks any of these against the stored models, without searching again.

The pattern is the same throughout: an answer comes with the evidence that would let a reader reject it.

## From Model to Application

A model in go-pflow's format can carry a `view`: prose, addressed to whoever builds an application on the model, that says what the application should be ("let two players play on the board; a click fires that cell's move transition"). Nothing executes it. It is part of the model's content hash because the same net presented as a game and as a dashboard is two different artifacts, and the id should say which one you have.

`sim_publish_app` publishes the single-file HTML application written from that view, by a person or an LLM, and serves it at `https://sim.pflow.xyz/app/{id}`. The intended starting point is the runtime, not a blank page: `/lib/app-template.html` is a working console composed of the same web components the generic console runs, so an application built from them inherits the console's role derivation, influence ranking and caveats instead of reimplementing them. The app is served in a sandboxed origin with no cookies and no session, under a Content-Security-Policy that confines scripts, styles and fetches to sim's own origin, so it can reach only the public API. The upload checks refuse an empty or oversized document, one that never mentions its model id, and one that loads scripts or styles from elsewhere. Whether the app does what the view says is not checked; that responsibility stays with whoever wrote it.

For readers who would rather have a spreadsheet than an app, `sim_publish` and `sim_publish_compare` write a scenario or a shared-seed comparison into the caller's own Google Sheet, with charts.

## The Complete Workflow

A typical session with sim.pflow.xyz:

1. **Describe** — write the model as JSON, or derive it with `sim_generate`, `sim_code_to_flow` or the builder
2. **Store** — `sim_create_model` or `POST /api/models`; read every reason the validator gives at once
3. **Verify** — `sim_verify` and `sim_invariants`; read the method on each verdict, and treat unknown as unknown
4. **Run** — `sim_scenario` on the discrete engine; ask for the ODE and read the refusal if the net is gated
5. **Compare and diagnose** — `sim_compare` on a shared seed, `sim_diagnose` for the knobs you did not think to ask about
6. **Hand it over** — a `sim_receipt` for the run that matters, a published app or sheet for the people who will use the answer

The model is the contract here too. Every run, verdict and receipt names the model's content id, so a reader can always ask which net an answer came from, and get the same net back.

> **Try it live:** The [catalog](https://sim.pflow.xyz/) lists every model in the commons, each with a what-if console. The [Café](https://sim.pflow.xyz/whatif/?id=3ab9e15cde5c02b463c31354) used in this chapter, the [Vet Clinic](https://sim.pflow.xyz/whatif/?id=ae6ab5aeb95dd994bc561479) and [Tic-Tac-Toe](https://sim.pflow.xyz/whatif/?id=5a08c67862a2eb55799f5229) are good starting points.
