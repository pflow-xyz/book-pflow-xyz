# Topology-Driven Verification

**Learning objective**: Generate ZK proof circuits automatically from Petri net topology, and verify execution on-chain.

Chapter 12 built ZK circuits by hand — defining the `PetriTransitionCircuit` step by step, wiring up MiMC hashes and delta computations manually. That was instructive, but nobody should write ZK circuits by hand for every new model. This chapter closes the loop: given a Petri net JSON model, the `petrigen` compiler produces working gnark circuits, witness generators, and Solidity verifiers automatically. The topology *is* the circuit. Change the model, recompile, and you have a new proof system.

Along the way, we discover that graph connectivity determines more than circuit structure — it determines strategic value. Rate constants for ODE simulation can be derived directly from the bipartite graph, with no training data and no manual tuning. The classic tic-tac-toe heuristic (center > corner > edge) emerges from counting win-line connections. This is graph theory applied to Petri nets, and it feeds directly into the ZK pipeline.

## The Compilation Pipeline

The full path from model to verifiable computation:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   JSON Model    │────▶│    petrigen     │────▶│  gnark Circuit  │
│  (Petri net)    │     │   (generator)   │     │   (Go code)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Groth16/PLONK   │
                                               │    Prover       │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Solidity        │
                                               │ Verifier        │
                                               └─────────────────┘
```

The `petrigen` package in go-pflow reads a Petri net model and generates four files:

| File | Purpose |
|------|---------|
| `petri_state.go` | Place/transition constants, topology arrays, marking operations |
| `petri_circuits.go` | `PetriTransitionCircuit` and `PetriReadCircuit` definitions |
| `petri_game.go` | Game state tracking and witness generation |
| `petri_circuits_test.go` | Compilation and proof verification tests |

No manual circuit writing. The topology encoding — which places connect to which transitions, with what weights — becomes compile-time constant arrays in the generated code. The selector trick from Chapter 12 (evaluating all transitions and using `IsZero` to select one) is wired up automatically.

### A Complete Example: Order Workflow

Consider a simple order processing workflow:

```
[pending] --approve--> [approved] --ship--> [shipped]
     |
     +----cancel----> [cancelled]
```

Define it as JSON:

```json
{
  "name": "order-workflow",
  "places": [
    {"id": "pending", "initial": 1},
    {"id": "approved"},
    {"id": "shipped"},
    {"id": "cancelled"}
  ],
  "transitions": [
    {"id": "approve"},
    {"id": "ship"},
    {"id": "cancel"}
  ],
  "arcs": [
    {"from": "pending", "to": "approve"},
    {"from": "approve", "to": "approved"},
    {"from": "approved", "to": "ship"},
    {"from": "ship", "to": "shipped"},
    {"from": "pending", "to": "cancel"},
    {"from": "cancel", "to": "cancelled"}
  ]
}
```

Generate the circuits:

```go
gen, _ := petrigen.New(petrigen.Options{
    PackageName:  "main",
    OutputDir:    ".",
    IncludeTests: true,
})

files, _ := gen.Generate(&model)
for _, f := range files {
    fmt.Println("Generated:", f.Name)
}
```

Output:

```
Generated: petri_state.go
Generated: petri_circuits.go
Generated: petri_game.go
Generated: petri_circuits_test.go
```

The generated `petri_state.go` encodes the topology as constants:

```go
const NumPlaces = 4
const NumTransitions = 3

const (
    Pending   = 0
    Approved  = 1
    Shipped   = 2
    Cancelled = 3
)

const (
    Approve = 0
    Ship    = 1
    Cancel  = 2
)

var Topology = [NumTransitions]ArcDef{
    Approve: {Inputs: []int{0}, Outputs: []int{1}},  // pending → approved
    Ship:    {Inputs: []int{1}, Outputs: []int{2}},   // approved → shipped
    Cancel:  {Inputs: []int{0}, Outputs: []int{3}},   // pending → cancelled
}
```

This is the same pattern from Chapter 12, but generated automatically. The circuit's `Define()` method iterates over `Topology`, computing deltas and enforcing the firing rule. The developer never touches circuit code.

### Proving a Transition

With the generated code, proving a valid state transition takes five steps:

```go
// 1. Compile the circuit
ccs, _ := frontend.Compile(ecc.BN254.ScalarField(),
    r1cs.NewBuilder, &PetriTransitionCircuit{})
fmt.Printf("Circuit has %d constraints\n", ccs.GetNbConstraints())

// 2. Trusted setup (one-time per circuit)
pk, vk, _ := groth16.Setup(ccs)

// 3. Fire a transition locally
game := NewPetriGame()
witness, _ := game.FireTransition(Approve)

// 4. Generate proof
assignment := witness.ToPetriTransitionAssignment()
w, _ := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
proof, _ := groth16.Prove(ccs, pk, w)

// 5. Verify
publicWitness, _ := w.Public()
err := groth16.Verify(proof, vk, publicWitness)
// err == nil: proof valid
```

The verifier sees three public inputs — `PreStateRoot`, `PostStateRoot`, and `Transition` — and confirms the state changed correctly. The actual marking (which places have tokens) stays private.

For the order workflow, this circuit has roughly 1,800 R1CS constraints. Proof generation takes ~100ms. Verification takes ~2ms. The proof is 192 bytes.

### Invalid Transitions Are Unprovable

What happens if someone tries to ship an order that hasn't been approved?

```go
game := NewPetriGame()  // Initial: pending=1
_, err := game.FireTransition(Ship)
// err: "transition ship is not enabled"
```

The `FireTransition` function checks enabledness before generating a witness. But even if someone bypasses this check and constructs a malicious witness, the circuit's bit decomposition will fail — `PreMarking[Approved]` is 0, and `0 - 1` wraps to a huge field element that can't be decomposed into 8 bits. The proof literally cannot be generated. Soundness comes from the circuit, not from the application code.

## From Graph Connectivity to Rate Constants

Chapter 3 introduced mass-action kinetics with hand-tuned rate constants. Chapter 6 used them for tic-tac-toe heatmaps. But where do rate constants come from?

For chemical reactions, rates encode physical properties — activation energies, temperature dependence, catalytic effects. You measure them in a lab. For games and workflows, there's no lab. The rate constants encode strategic value, and that value can be derived from the graph itself.

### The Algorithm

The rate auto-derivation algorithm operates on the bipartite directed graph of the Petri net:

1. Identify **candidate** transitions (the moves a player can make) and **target** transitions (the goals — win conditions, completion states)
2. For each candidate, find its **unique output places** — places it produces that no other candidate produces
3. Count how many target transitions take those unique places as inputs
4. That count is the rate constant

### Tic-Tac-Toe: Strategy from Topology

The TTT Petri net from [Chapter 6](ch06-game-mechanics.md) has 33 places and 35 transitions. Each play transition (like `x_play_11`) produces a piece at a position (like `x11`). Each win transition (like `x_win_diag`) requires three specific pieces as inputs.

Apply the algorithm to the center position:

```
x_play_11 outputs → {x11, o_turn, move_tokens}

Unique outputs (only x_play_11 produces x11):
  x11

Win transitions with x11 as input:
  x_win_row1  (middle row)    ✓
  x_win_col1  (center column) ✓
  x_win_diag  (main diagonal) ✓
  x_win_anti  (anti-diagonal) ✓

Rate for x_play_11 = 4  (center: 4 win lines)
```

Compare with a corner:

```
x_play_00 outputs → {x00, o_turn, move_tokens}

Unique outputs: x00

Win transitions with x00 as input:
  x_win_row0  ✓
  x_win_col0  ✓
  x_win_diag  ✓

Rate for x_play_00 = 3  (corner: 3 win lines)
```

And an edge:

```
x_play_01 outputs → {x01, o_turn, move_tokens}

Unique outputs: x01

Win transitions with x01 as input:
  x_win_row0  ✓
  x_win_col1  ✓

Rate for x_play_01 = 2  (edge: 2 win lines)
```

Center > corner > edge. The classic tic-tac-toe strategy emerges from counting graph connections. No game theory, no training data, no heuristics — just topology.

### Filtering Shared Places

The algorithm has one non-obvious step: filtering shared output places. Without it, the results are useless.

Every `x_play_*` transition produces three output places: the piece (e.g., `x00`), the opponent's turn token (`o_turn`), and a move counter (`move_tokens`). The piece place is unique to that candidate — only `x_play_00` produces `x00`. But `o_turn` and `move_tokens` are produced by all 9 x-play transitions identically.

The problem: `o_turn` is an input to every `x_win_*` transition (win detection happens on the opponent's turn). If you include `o_turn` in the connectivity count, every candidate connects to all 8 win targets through it, giving every position rate=8. The heatmap collapses to a flat field — center, corner, and edge become indistinguishable.

The fix: exclude any output place produced by more than one candidate. A shared place carries no discriminative signal — it's the DC component that shifts every candidate equally. Only places unique to a single candidate can distinguish one candidate from another. This is analogous to mean-centering features before computing distances: the shared component must be subtracted before differences become meaningful.

### It's Graph Theory, Not Petri Net Theory

The rate auto-derivation is pure graph theory. It operates on a bipartite directed graph — nodes are candidates and targets, edges pass through output places — and computes degree centrality of candidates with respect to targets through unique edges. Nothing about the algorithm requires Petri net firing semantics, token counts, or conservation laws.

The algorithm is structurally similar to a single message-passing step in a graph neural network:

- **GNN**: `node_embedding = aggregate(neighbor_features)`
- **Petri net**: `rate[candidate] = count(reachable_targets through unique_outputs)`

The difference: GNNs learn the aggregation function from data. The Petri net version uses a fixed, interpretable aggregation (count of target connections). No training needed.

What the Petri net formalism adds comes in layers above the rate derivation:

1. **Firing semantics.** Transitions consume and produce tokens atomically. This gives you state machines that pure graph connectivity cannot express.
2. **Mass-action convention.** The rate formula $v[t] = k[t] \times \prod M[\text{inputs}[t]]$ couples topology-derived weights to the current state.
3. **Conservation laws.** P-invariants constrain the state space with guarantees that graph centrality alone cannot provide.
4. **ZK circuit structure.** The stoichiometry matrix defines the gnark circuit constraints directly.

The rate derivation discovers *how much* each candidate matters. The Petri net machinery determines *what happens* when you act on that knowledge.

### Poker Hand Ranking: Integer Reduction

The tic-tac-toe example derives strategic value from *outgoing* connectivity — more connections to win transitions means higher value. But the same mechanism works in reverse. In poker, we want to rank hand categories by *rarity*. Rare hands should have high value; common hands, low.

The insight: encode combinatorial frequency as structural outflow. Build an analysis net where each hand category gets:

- A **source place** `src_H` (1 token) — constant inflow via catalytic arc
- A **value place** `val_H` (0 tokens) — accumulation target
- A **play transition** `play_H` — moves tokens from source to value, returning the source token
- **N drain transitions** — consume tokens from `val_H` at a rate proportional to hand frequency

The drain counts are log-scaled from actual 5-card combination counts:

| Hand | 5-Card Combos | Drains |
|------|--------------|--------|
| Straight Flush | 40 | 1 |
| Four of a Kind | 624 | 2 |
| Full House | 3,744 | 4 |
| Flush | 5,108 | 5 |
| Straight | 10,200 | 8 |
| Three of a Kind | 54,912 | 12 |
| Two Pair | 123,552 | 16 |
| One Pair | 1,098,240 | 24 |
| High Card | 1,302,540 | 32 |

The full net has 18 places and 113 transitions (9 play + 104 drain).

At equilibrium under mass-action kinetics, inflow equals outflow for each value place:

$$\text{rate} \times [src_H] = \text{num\_drains} \times \text{rate} \times [val_H]$$

Since `src_H` is catalytic (always 1) and rates are uniform:

$$val_H = \frac{1}{\text{num\_drains}}$$

Rare hands have fewer drains, accumulate more tokens, and produce higher equilibrium values. Common hands drain fast, accumulate little, and score low. Running the ODE with uniform rates to equilibrium yields:

```
Straight Flush:  32.0  (1 drain)
Four of a Kind:  16.0  (2 drains)
Full House:       8.0  (4 drains)
Flush:            6.4  (5 drains)
Straight:         4.0  (8 drains)
Three of a Kind:  2.7  (12 drains)
Two Pair:         2.0  (16 drains)
One Pair:         1.3  (24 drains)
High Card:        1.0  (32 drains)
```

The entire poker hand hierarchy emerges from topology. No poker knowledge is injected beyond the frequency encoding in drain counts.

### TTT vs. Poker: Same Mechanism, Inverted Reading

In tic-tac-toe, more connections to win transitions means *higher* strategic value. The raw ODE concentration increases with connectivity — center accumulates tokens fastest because it connects to the most win lines.

In poker, fewer drain connections means *higher* hand value. The raw ODE concentration increases with *fewer* drains — straight flush accumulates tokens fastest because it has the fewest sinks.

Both cases recover integers (or integer-like ratios) from topology via ODE equilibrium. The structural principle is identical: differential outflow creates differential accumulation, and the equilibrium concentrations encode the ranking.

This also addresses the boundary question from [Chapter 15](ch15-exponential-weights.md). The exponential weights approach bolted scoring onto the net as bookkeeping — 52 extra transitions producing a number that nothing in the net reads. Integer reduction derives the scoring *from* the net. The drain transitions aren't bookkeeping; they're the mechanism that produces the ranking. The topology speaks.

The next chapter completes the pipeline: compiling guards and state invariants into ZK circuits, committing state with Merkle trees, and verifying proofs on-chain via smart contracts.
