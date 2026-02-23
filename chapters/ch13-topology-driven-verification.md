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

This also addresses the boundary question from [Chapter 13b](ch13-exponential-weights.md). The exponential weights approach bolted scoring onto the net as bookkeeping — 52 extra transitions producing a number that nothing in the net reads. Integer reduction derives the scoring *from* the net. The drain transitions aren't bookkeeping; they're the mechanism that produces the ranking. The topology speaks.

## Guard Compilation

Chapter 4 introduced guards — boolean expressions that add conditions beyond simple token availability. A guard like `balances[from] >= amount` on a transfer transition means: even if the input places have tokens, the transition only fires when the guard is satisfied.

In a ZK circuit, guards become arithmetic constraints. The `GuardCompiler` in go-pflow transforms guard expressions into the constraint IR that the gnark code generator consumes.

### Boolean Logic as Field Arithmetic

ZK circuits operate over a finite field. There are no `if` statements, no booleans, no branches. Everything is multiplication and addition over integers mod a prime. The compiler transforms each logical operation:

**AND**: `A && B` becomes `A * B`. Both must be 1 for the product to be 1.

**OR**: `A || B` becomes `A + B - A * B`. At least one must be 1.

**NOT**: `!A` becomes `1 - A`. The operand must be boolean (0 or 1).

### Comparisons as Range Checks

The more subtle transformations handle comparisons:

**Greater-or-equal**: `left >= right` becomes:
1. Compute `diff = left - right`
2. Assert `diff` is non-negative via bit decomposition

Non-negativity in a finite field is tricky — every field element is technically positive. The trick from Chapter 12 applies: decompose `diff` into bits. If the original value was negative, it wraps to a huge field element that won't fit in the expected bit width, and the decomposition fails.

**Strict greater-than**: `left > right` reduces to `left - right - 1 >= 0`.

**Equality**: `left == right` is a direct `AssertIsEqual` constraint.

**Inequality**: `left != right` requires proving the inverse exists. The compiler introduces a witness `inv` and constrains `(left - right) * inv = 1`. This is satisfiable only when `left - right` is non-zero — zero has no multiplicative inverse in a field.

### State Access and Merkle Proofs

Guards often reference state: `balances[alice] >= amount`. The `compileIndexExpr` function handles map lookups by registering a state read. Each state read becomes a Merkle proof obligation — the prover must demonstrate that the claimed value actually exists in the committed state tree.

For nested maps like `allowances[owner][spender]`, the compiler generates composite key hashing: `key = Poseidon(owner, spender)`. The resulting key looks up a single leaf in the Merkle tree.

```go
func (c *GuardCompiler) compileIndexExpr(idx *guard.IndexExpr) *Expr {
    // Check for nested index (e.g., allowances[owner][spender])
    if innerIdx, ok := idx.Object.(*guard.IndexExpr); ok {
        return c.compileNestedIndex(innerIdx, idx.Index)
    }

    // Simple index: place[key]
    placeIdent, _ := idx.Object.(*guard.Identifier)
    keyIdent, _ := idx.Index.(*guard.Identifier)

    // Register state read — triggers Merkle proof generation
    witness := c.witnesses.AddStateRead(placeIdent.Name, []string{keyIdent.Name})
    return VarExpr(witness.Name)
}
```

The guard compiler doesn't verify Merkle proofs itself. It registers what state needs to be proven, and the pipeline's Merkle proof compiler generates the verification constraints separately.

## Merkle State Commitment

Chapter 12 committed state with a flat MiMC hash: feed all place values into a single hash and publish the result. This works for small nets — the tic-tac-toe model has 33 places, and hashing 33 values costs ~300 constraints.

But real applications have thousands or millions of state entries. An ERC-20 token has one balance per address. A workflow system has one case state per process instance. Flat hashing doesn't scale.

### From Flat Hash to Merkle Tree

A Merkle tree organizes state as leaves and hashes pairs up to a single root:

```
            root
           /    \
        h01      h23
       /   \    /   \
     h0    h1  h2    h3
     |     |   |     |
   leaf0 leaf1 leaf2 leaf3
```

To prove a single value, the prover provides the leaf and the sibling hashes along the path to the root — $O(\log n)$ hashes instead of $O(n)$. A depth-20 tree supports $2^{20}$ (~1 million) leaves with only 20 hash operations per proof.

### Poseidon vs. MiMC

Chapter 12 used MiMC for state commitment — ~300 constraints per hash. The Merkle tree compiler uses Poseidon, another ZK-friendly hash function, at ~182 constraints per hash. Over 20 levels of tree traversal, this saves roughly 2,400 constraints per Merkle proof.

### Verification in the Circuit

The Merkle proof verification algorithm in the circuit:

1. Compute the leaf hash: `leaf = Poseidon(key, value)`
2. Walk up the tree: at each level, select left/right based on the path index, then hash the pair
3. Assert the computed root equals the committed state root

The path selection uses the same no-branching pattern as transition selection — arithmetic multiplexing:

```
left  = pathIndex * sibling + (1 - pathIndex) * current
right = pathIndex * current + (1 - pathIndex) * sibling
hash  = Poseidon(left, right)
```

Each Merkle proof adds approximately $20 \times 2 \times 182 = 7{,}280$ constraints (20 levels, 2 operations per level, 182 constraints per Poseidon hash). A transition with two state accesses (sender balance, receiver balance) adds ~14,560 constraints for Merkle proofs alone. This is the cost of scalable state — more constraints per proof, but the state tree can hold millions of entries without growing the circuit further.

### Nested Maps

For data structures like `allowances[owner][spender]`, the compiler hashes the key pair into a single composite key:

```
compositeKey = Poseidon(owner, spender)
leaf = Poseidon(compositeKey, value)
```

This flattens nested maps into a single Merkle tree. The prover includes the raw keys and the composite key hash in the witness, and the circuit verifies the key composition.

## State Invariants in Circuits

Chapter 2 introduced P-invariants — weighted sums of places that remain constant across all firings. For the enzyme kinetics model, the total of substrate plus complex plus product is conserved: tokens are neither created nor destroyed, only rearranged.

P-invariants are powerful for offline analysis. But in a ZK context, they become something stronger: constraints compiled directly into the proof circuit. A violation isn't just detectable — it's *unprovable*. No valid proof can exist for a state transition that breaks an invariant.

### Conservation Laws

The `InvariantCompiler` generates constraints for conservation laws. Consider an ERC-20 token with the invariant `sum(balances) == totalSupply`. In a ZK circuit, we can't iterate over all balances — the circuit doesn't know how many accounts exist. Instead, we verify the invariant *differentially*:

$$\Delta(\text{totalSupply}) = \sum \Delta(\text{balances}_{\text{touched}})$$

Untouched balances don't change, so their deltas are zero. The circuit only needs to verify that the changes to touched balances sum to the change in total supply:

```go
func (c *InvariantCompiler) CompileConservation(
    sumPlace string,
    totalPlace string,
    transitions []StateTransition,
) []*Constraint {
    // Sum deltas for the summed place (e.g., balances)
    var sumDeltaExpr *Expr
    for _, t := range transitions {
        if t.Place == sumPlace {
            delta := SubExpr(VarExpr(t.PostVar), VarExpr(t.PreVar))
            if sumDeltaExpr == nil {
                sumDeltaExpr = delta
            } else {
                sumDeltaExpr = AddExpr(sumDeltaExpr, delta)
            }
        }
    }

    // Conservation: delta(total) == sum(deltas)
    return []*Constraint{{
        Type:  Equal,
        Left:  totalDeltaExpr,
        Right: sumDeltaExpr,
        Tag:   "conservation: delta(totalSupply) == sum(delta(balances))",
    }}
}
```

This is the state equation from Chapter 2 ($M' = M + N \cdot \sigma$) enforced cryptographically. The incidence matrix determines the deltas; the invariant compiler verifies they satisfy the conservation law.

### Non-Negativity

Token counts can't go negative. The `CompileNonNegative` function adds range checks on every touched place's post-transition value:

```go
func (c *InvariantCompiler) CompileNonNegative(
    place string,
    transitions []StateTransition,
) []*Constraint {
    var constraints []*Constraint
    for _, t := range transitions {
        if t.Place == place {
            constraints = append(constraints,
                RangeConstraint(VarExpr(t.PostVar),
                    fmt.Sprintf("%s >= 0", place)))
        }
    }
    return constraints
}
```

In the circuit, this becomes a bit decomposition — the same trick Chapter 12 used for enabledness checks. If a balance would go negative, the field element wraps to a huge value that fails the range check.

### Boundedness

Some places have capacity limits. A voting system might limit each voter to one ballot. The `CompileBounded` function constrains post-values to not exceed a maximum:

```go
// max - post >= 0
diff := SubExpr(ConstExpr(maxValue), VarExpr(t.PostVar))
constraints = append(constraints,
    EqualConstraint(VarExpr(diffWitness), diff, "bounded"),
    RangeConstraint(VarExpr(diffWitness), "bounded check"),
)
```

### From Schema to Constraints

The invariant compiler reads constraints directly from the model schema:

```
sum(balances) == totalSupply    → Conservation constraint
balances >= 0                   → Non-negative constraint
votes <= maxVotes               → Bounded constraint
```

Each constraint adds a small number of R1CS constraints to the circuit. The cost is minimal — typically 1-3 constraints per invariant — but the guarantee is absolute. A prover who violates any invariant simply cannot generate a valid proof.

## On-Chain Verification

Generating proofs locally is useful for peer-to-peer verification — one player proves to another that a move is valid. But for applications with shared state — token transfers, governance, auctions — the proof needs to be verified where the state lives: on a blockchain.

### State Root Chaining

The `ZkOde` smart contract maintains a chain of state roots. Each proof's post-state root must equal the next proof's pre-state root, creating an unbroken chain from genesis to the current state:

```
Genesis: stateRoot₀ = MiMC(initial_marking)

Step 1: proof₁ proves stateRoot₀ → stateRoot₁
Step 2: proof₂ proves stateRoot₁ → stateRoot₂
  ...
Step n: proofₙ proves stateRootₙ₋₁ → stateRootₙ
```

The contract stores the current state root and rejects any proof whose pre-state root doesn't match:

```solidity
if (preRoot != currentStateRoot) {
    revert InvalidStateChain(currentStateRoot, preRoot);
}
```

This ensures no steps are skipped, replayed, or reordered. The on-chain state root is the single source of truth.

### The ZkOde Contract

The core contract is straightforward:

```solidity
contract ZkOde {
    IVerifier public verifier;
    uint256 public currentStateRoot;
    uint256 public stepCount;
    uint256 public immutable numTransitions;
    bool public enforceOptimal;

    struct Step {
        uint256 preRoot;
        uint256 postRoot;
        uint256 nextRoot;
        uint256 stepSize;
        uint256 chosenTransition;
        uint256 timestamp;
    }

    mapping(uint256 => Step) public steps;
}
```

The `verifier` is a gnark-generated Groth16 verifier contract — exported directly from the proving system. The `submitStep` function accepts a proof, verifies it, and advances the state root:

```solidity
function submitStep(
    uint256[8] calldata proof,
    uint256[] calldata publicInputs,
    uint256 chosenTransition,
    uint256 nextStateRoot
) external onlyProver {
    // Validate state chain continuity
    require(publicInputs[0] == currentStateRoot);

    // Verify Groth16 proof
    uint256[2] memory a = [proof[0], proof[1]];
    uint256[2][2] memory b = [[proof[2], proof[3]],
                               [proof[4], proof[5]]];
    uint256[2] memory c = [proof[6], proof[7]];
    require(verifier.verifyProof(a, b, c, publicInputs));

    // Record step and advance state
    currentStateRoot = nextStateRoot;
    stepCount++;
}
```

### Optimal Play Enforcement

For game applications, the contract can enforce that the chosen transition has the highest rate among all available transitions. The public inputs include rate values for every transition (derived from the ODE integration), and the contract checks that no other transition has a higher rate:

```solidity
if (enforceOptimal) {
    uint256 chosenRate = publicInputs[3 + chosenTransition];
    for (uint256 t = 0; t < numTransitions; t++) {
        uint256 rate = publicInputs[3 + t];
        if (rate > chosenRate) {
            revert NotOptimalPlay(
                chosenTransition, chosenRate, t, rate);
        }
    }
}
```

This is a remarkable property: the contract doesn't know the game rules, doesn't know what tic-tac-toe is, doesn't store the board state. It only knows the topology-derived rates. And yet it enforces optimal play — because the rates encode strategic value, and strategic value comes from the topology, and the topology *is* the game rules.

### Batch Submission

For applications with many steps per session (a full game, a sequence of workflow transitions), submitting proofs one at a time wastes gas on per-transaction overhead. The `submitBatchSteps` function accepts arrays of proofs and processes them sequentially:

```solidity
function submitBatchSteps(
    uint256[8][] calldata proofs,
    uint256[][] calldata publicInputsBatch,
    uint256[] calldata chosenTransitions,
    uint256[] calldata nextStateRoots
) external onlyProver {
    for (uint256 i = 0; i < proofs.length; i++) {
        _verifyAndRecord(
            proofs[i], publicInputsBatch[i],
            chosenTransitions[i], nextStateRoots[i]);
    }
}
```

### Gas Costs

| Operation | Gas Cost |
|-----------|----------|
| Groth16 verification | ~200,000 |
| State root storage | ~20,000 |
| **Total per transition** | **~220,000** |

At current L2 gas prices, this is roughly \$0.01–\$2 per verified transition, depending on the chain. A full 9-move tic-tac-toe game costs under \$1 on Base.

### Deployed Contracts

The system is deployed on Base Sepolia (testnet) with two configurations:

| Deployment | Verifier | ZkOde | Transitions | Optimal |
|------------|----------|-------|-------------|---------|
| Cascade (3-place) | `0xA675...` | `0x2084...` | 2 | No |
| TTT Heatmap (33-place) | `0x97a6...` | `0xF5d9...` | 9 | Yes |

The TTT deployment enforces optimal play — the contract rejects any move that doesn't have the highest topology-derived rate. Combined with the ZK proof of correct state transition, this creates a fully trustless game: no server, no referee, no possibility of cheating.

### ZK Hold'em: Topology-Derived Payouts

The poker hand analysis net from earlier in this chapter produces hand strength values purely from topology. These values transfer directly to a game contract as payout multipliers.

The `ZKHoldem` contract uses the same Groth16 verifier pattern as `ZkOde`, but adds three poker-specific mechanisms:

**Commit-reveal shuffle.** Before the game starts, the house publishes `Poseidon(seed)` as a binding commitment. The deck is shuffled via Fisher-Yates with the seed as a deterministic PRNG. At showdown, the house reveals the seed. Anyone can verify `Poseidon(seed) == commitment` and derive the cards independently. If the house doesn't reveal within a timeout window, the player claims the pot.

**State root chaining per action.** Each game action — deal, check, bet, call, fold — fires a transition on a game net (18 places, 16 transitions) and produces a Groth16 proof. The post-state Poseidon hash of one action becomes the pre-state hash of the next, forming an unbroken proof chain.

**Topology-derived bonus payouts.** The winner takes the pot plus a bonus proportional to the integer reduction value of their hand:

```solidity
uint256[9] private HAND_STRENGTH = [
    uint256(1), 1, 2, 3, 4, 6, 8, 16, 32
];
// Index: 0=HC, 1=Pair, 2=2P, 3=3K, 4=Str, 5=Flush, 6=FH, 7=4K, 8=SF

bonus = HAND_STRENGTH[rank] * ante;
```

A straight flush win pays 32x the ante as bonus. A pair win pays 1x. These multipliers aren't tuned by a game designer — they're the equilibrium concentrations from the analysis net, normalized and rounded. The house edge is transparent because the payout structure is derived from the same topology that defines the game.

The house AI is deterministic: it evaluates its hand against the topology-derived strength values and applies a fixed strategy based on hand strength and pot odds. This determinism is enforceable on-chain — the contract can verify that the house played according to the strategy given its cards, just as the TTT contract enforces optimal play.

## Combinatorial vs. Continuous

The compilation pipeline supports two fundamentally different modes, and choosing the right one matters for both the circuit design and the interpretation of results.

### Combinatorial Mode

**Games, workflows, governance, token standards.**

The state space is finite and discrete. Tic-tac-toe has at most 5,478 reachable board states. A workflow has a bounded number of case states. An ERC-20 has integer balances.

In these systems, the topology and discrete scoring are sufficient. The ODE provides a continuous visualization of token flow — useful for intuition, beautiful as a heatmap — but when it comes time to decide, you discretize. The heatmap scoring that drives move selection in tic-tac-toe is discrete: count win lines, evaluate threats, pick the best move. There is no continuous quantity being modeled.

The "mass-action kinetics" framing is a useful metaphor that lets you apply ODE machinery, but the strategic information is graph-theoretic.

**What the ZK circuit proves**: the discrete state transition was valid — correct inputs consumed, correct outputs produced, guard conditions met. The ODE step is optional — it enriches the public output with heatmap scores but isn't necessary for correctness.

**Rate derivation**: auto-derive from topology. Degree centrality captures the essential structure.

### Continuous Mode

**Chemical kinetics, population ecology, epidemiology, economic models.**

The state genuinely evolves in continuous time. Concentrations rise and fall. Populations oscillate. Epidemics peak and decay. The *trajectory* carries information — you care about *when* concentrations peak, in what order, and how fast.

In these systems, the topology tells you what *can* happen, but the ODE tells you what happens *first*. A chemical cascade where reaction A peaks before reaction B produces a different product mix than one where B peaks first, even if the topology is identical.

**What the ZK circuit proves**: the ODE integration was computed correctly. The 7-stage Tsit5 step, the mass-action rates, the stoichiometry-weighted derivatives — all verified. This is where the ZK ODE machinery earns its keep.

**Rate derivation**: bring domain-specific rates. They encode physical properties (activation energies, birth/death rates, transmission coefficients) that topology cannot capture.

### Fixed-Point Arithmetic for Continuous Mode

ZK circuits operate over finite fields — integers mod a prime. There are no floating-point numbers. The system uses fixed-point arithmetic with a $10^{18}$ scale factor over the BN254 scalar field:

$$\text{FixFromFloat}(3.0) = 3 \times 10^{18}$$

$$\text{FixMul}(a, b) = \frac{a \times b}{10^{18}} \mod p$$

$$\text{FixAdd}(a, b) = (a + b) \mod p$$

This gives 18 decimal digits of precision — more than enough for ODE integration — using only integer operations that ZK circuits handle natively.

### The Pipeline Supports Both

| | Combinatorial | Continuous |
|---|---|---|
| **Rates** | Auto-derived from topology | Specified in model |
| **ODE step** | Optional (enriches output) | Essential (proves trajectory) |
| **Scoring** | Discrete win/block detection | Rate-weighted heatmap |
| **Circuit focus** | State transition validity | Integration correctness |
| **Examples** | TTT, Hold'em, workflows, ERC-20 | Cascade reactions, SIR epidemics |

The `zkgen` compiler selects the appropriate circuit template based on the model configuration.

## Why Topology Dominates

Consider what happens when you vary the rate constants while keeping the topology fixed:

- **All rates = 1**: Every transition fires at equal speed. The ODE still produces meaningful dynamics because the stoichiometry matrix determines how tokens flow. In tic-tac-toe, pieces still accumulate at positions, win patterns still emerge — just without positional preference.

- **Position-weighted rates**: Center fires 4x faster than edges. The same topology produces the same qualitative behavior, with quantitative bias toward strategically connected positions.

- **Arbitrary rates**: Even with random rates, the topology ensures tokens can only flow along arcs, transitions only fire when inputs are satisfied, and win conditions require the correct piece configurations.

Now consider what happens when you change the **topology**:

- **Remove one win line**: The game fundamentally changes. Corner positions lose strategic value. The rate auto-derivation produces different weights.
- **Add a new transition**: The entire flow network changes. New pathways open. The ODE explores qualitatively different dynamics.
- **Change an arc**: Even moving a single arc rewires which places feed which transitions, potentially breaking or creating invariants.

**The topology defines what is possible. The rates only control how fast you get there.**

This is the opposite of neural networks. In a neural network, the topology (layer sizes, connections) provides capacity, but the learned weights carry all the actual knowledge. Change the weights randomly and the network is useless. Change the topology and the network might still work — it just has different capacity.

In Petri nets, the structure carries the knowledge and the weights are a natural consequence. This inversion has a practical consequence: Petri net models are inspectable. You can look at the topology and understand what the system does. You can look at the stoichiometry matrix and read the differential equations. You can look at the circuit constraints and audit the proof system. There is no black box.

## Limitations

The topology-driven approach has real limitations, and understanding them determines where it applies.

### One-Hop Connectivity

The rate auto-derivation looks at direct connections: candidate → unique output → target input. In deep strategic games, the important signal is multi-hop — a move that enables a future move that threatens a win. Chess is a moderate case; Go is the extreme. On a 19x19 board, one-hop connectivity tells you almost nothing about strategic value. You'd need multi-hop reachability analysis (T-invariants, reachability graphs) for deeper strategy.

### Equal Target Weighting

The algorithm counts every target connection as weight=1. In chess, a checkmate path is worth infinitely more than capturing a pawn. A transition connected to one critical target and one irrelevant target gets the same count as one connected to two medium targets. Heterogeneous objectives need weighted target importance.

### Static Rates

Topology-derived rates are computed once from the initial graph and never change. A position's strategic value changes as the game progresses — a corner becomes critical when it completes a fork — but the rates remain fixed. The *tactical scoring layer* (win/block detection) handles this for tic-tac-toe, but that's a separate mechanism on top of rates.

### Symmetric Degeneracy

After X plays center in tic-tac-toe, all four corners have identical topology connectivity (rate=3). The rate derivation alone can't distinguish between them. The heatmap is flat for symmetric positions. Tactical scoring breaks the symmetry, but topology alone cannot.

### Circuit Scaling

Circuit size grows as $O(|P| \times |T|)$ due to the selector-based topology encoding. For nets with $|P|, |T| < 100$, this is practical — the tic-tac-toe circuit has ~24,500 constraints and proves in ~2 seconds. For larger nets, potential approaches include:

- **Chunked proofs**: prove subsets of transitions
- **Recursive composition**: aggregate proofs with recursive SNARKs
- **Sparse topology encoding**: skip zero entries in the stoichiometry matrix

### Non-Game Domains

In chemical reaction networks, rate constants encode physical properties that topology cannot capture. A reaction might connect to many downstream reactions but be physically slow. Topology connectivity is meaningless for chemistry — you need empirically measured or physically modeled rates. The pipeline handles this: specify rates in the model and the topology-derived defaults are overridden. But the auto-derivation provides no value for these domains.

## Putting It Together

The compilation pipeline from this chapter, combined with the circuit fundamentals from Chapter 12, creates a complete system for verifiable Petri net execution:

1. **Define the model** in JSON — places, transitions, arcs, guards, invariants
2. **Generate circuits** with `petrigen` — topology becomes constants, guards become constraints, invariants become range checks
3. **Derive rates** from graph connectivity — or specify domain-specific rates
4. **Prove transitions** locally — the prover generates a Groth16 proof that the state changed correctly
5. **Verify anywhere** — peers verify in 2ms, smart contracts verify on-chain for ~220,000 gas

The model is the specification. The compiler generates the verifier. The proof is the attestation that the specification was followed.

What changed from Chapter 12? Nothing conceptual. The circuit structure is the same — hash the marking, compute deltas from topology, assert the change, check enabledness. But everything that was manual is now automated. The developer works at the level of Petri net models, not ZK circuits. Change the model, recompile, and the entire proof system updates — circuits, witnesses, tests, and Solidity verifiers.

The technique generalizes. Two games now demonstrate topology-derived strategic values:

| Game | What Reduction Recovers | Net Size |
|------|------------------------|----------|
| Tic-Tac-Toe | Position value: center=4, corner=3, edge=2 | 18p x 33t |
| Hold'em | Hand ranking: SF=32, 4K=16, ..., HC=1 | 18p x 113t |

In both cases, ODE simulation with uniform rates extracts integers (or integer-like ratios) that encode strategic value. No game-specific heuristics. No training data. Just topology. The same proof system — Groth16 over BN254, Poseidon state hashing, stoichiometry-based constraints — covers both. The circuit doesn't know what game it's proving. It only knows the topology.

One circuit structure. Any Petri net. Automatically generated, topology-driven, cryptographically verified.
