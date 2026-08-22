# Zero-Knowledge Proofs

**Learning objective**: Prove a state transition is valid without revealing the state.

Every Petri net model in this book has been transparent — all token counts visible to all participants. The coffee shop owner sees inventory levels. Both tic-tac-toe players see the board. The hospital monitor sees patient states. But some applications require privacy. A voter should prove they cast a valid ballot without revealing their choice. A financial institution should verify a transfer is valid without exposing account balances. A player should prove their move is legal without revealing their strategy.

Zero-knowledge proofs make this possible. The key insight: a Petri net is already a formal specification of valid state transitions. A ZK circuit turns that specification into a cryptographic proof system. The prover demonstrates they followed the rules; the verifier confirms it without seeing the state.

## The Statement Being Proven

Every ZK proof in this system proves one statement:

> "I know a valid marking that, when transition T fires, produces the claimed next state."

More precisely:

1. The **pre-state root** (a hash) matches a specific private marking
2. The **post-state root** matches a specific private marking
3. The chosen **transition was enabled** — all input places had sufficient tokens
4. The **marking change is correct** — post = pre + delta, where delta comes from the net topology

The verifier sees only the two state roots and the transition ID. The actual token counts remain hidden.

### Public vs. Private

| | What | Who Sees It |
|-|------|-------------|
| **Public** | Pre-state root (hash) | Everyone |
| **Public** | Post-state root (hash) | Everyone |
| **Public** | Transition ID | Everyone |
| **Private** | Pre-marking (all token counts) | Prover only |
| **Private** | Post-marking (all token counts) | Prover only |

The state roots are commitments. They bind the prover to a specific marking without revealing it.

## State Commitment via MiMC Hashing

The hash function matters. Inside a ZK circuit, every operation becomes arithmetic constraints on a finite field. SHA-256 needs thousands of boolean operations — roughly 25,000 constraints per hash. **MiMC** (Minimum Multiplicative Complexity) is designed for arithmetic circuits — it operates over the same finite field the circuit uses, costing roughly 300 constraints per hash.

```go
func petriMimcHash(api frontend.API, values []frontend.Variable) frontend.Variable {
    h, _ := mimc.NewMiMC(api)
    for _, v := range values {
        h.Write(v)
    }
    return h.Sum()
}
```

The state root is:

$$\text{stateRoot} = \text{MiMC}(\text{marking}[0], \text{marking}[1], \ldots, \text{marking}[N-1])$$

To recover the actual token counts from a state root, an attacker would need to invert MiMC — computationally infeasible. The hash commits the prover to a specific marking without revealing it.

## The Circuit in Five Steps

The gnark circuit for a Petri net transition proof has five steps. Each adds constraints that the prover must satisfy.

### Step 1: Hash the Pre-Marking

Compute the MiMC hash of every place's token count and assert it equals the public pre-state root:

```go
preRoot := petriMimcHash(api, c.PreMarking[:])
api.AssertIsEqual(preRoot, c.PreStateRoot)
```

This binds the private marking to the public commitment. The prover can't lie about what state they started from.

### Step 2: Hash the Post-Marking

Same check for the post-state:

```go
postRoot := petriMimcHash(api, c.PostMarking[:])
api.AssertIsEqual(postRoot, c.PostStateRoot)
```

Now both states are committed.

### Step 3: Compute the Delta from Topology

From the transition ID, look up the Petri net topology to build a delta vector. The topology is **baked into the circuit as constants** — the incidence matrix encoded as input/output lists:

```go
for t := 0; t < NumTransitions; t++ {
    isThis := api.IsZero(api.Sub(c.Transition, t))

    for _, p := range Topology[t].Inputs {
        deltas[p] = api.Sub(deltas[p], isThis)
    }
    for _, p := range Topology[t].Outputs {
        deltas[p] = api.Add(deltas[p], isThis)
    }
}
```

The `isThis` variable is 1 for the selected transition and 0 for all others. This multiplexes over all transitions without branching — arithmetic circuits can't branch, so we compute all possibilities and select. Only the chosen transition's delta contributes to the result.

### Step 4: Assert Marking Change

For every place, check that the post-marking equals the pre-marking plus the delta:

```go
for p := 0; p < NumPlaces; p++ {
    expected := api.Add(c.PreMarking[p], deltas[p])
    api.AssertIsEqual(c.PostMarking[p], expected)
}
```

This ensures the marking changed exactly as the topology dictates. No extra tokens appeared. No tokens vanished. The transition fired correctly.

### Step 5: Assert Enabledness

For every input place of the selected transition, the pre-marking must have at least one token:

```go
diff := api.Sub(c.PreMarking[p], isInput)
api.ToBinary(diff, 8)
```

The trick: `ToBinary` decomposes a value into bits. If the value were negative (insufficient tokens), it would wrap to a huge field element and the bit decomposition would fail — the proof can't be generated. This is a standard gnark pattern for range checks: decompose into bits to prove non-negativity.

## Topology as Circuit Constants

The most important property of this approach: **the Petri net topology is baked into the circuit as constants**.

```go
var Topology = [NumTransitions]struct {
    Inputs  []int
    Outputs []int
}{
    {Inputs: []int{0, 1}, Outputs: []int{2}},    // bind
    {Inputs: []int{2}, Outputs: []int{0, 1}},     // unbind
    {Inputs: []int{2}, Outputs: []int{3, 1}},     // catalyze
}
```

This means:

1. **No application-specific logic** in the circuit. The same `Define()` method works for any Petri net.
2. **Topology changes require recompilation**. Adding a place or transition means a new trusted setup.
3. **The circuit is auditable**. Anyone can inspect the topology to verify it matches the claimed rules.

The Petri net is the specification. The circuit is the verifier. The proof is the attestation that the specification was followed.

## Groth16 Proofs

The proving system is Groth16, implemented by gnark. It requires a one-time trusted setup per circuit but produces:

- **Smallest proofs**: ~128 bytes (2 G1 points + 1 G2 point)
- **Fastest verification**: ~2ms, constant regardless of circuit size
- **Solidity export**: gnark generates a ready-to-deploy verifier contract

| Property | Groth16 | PLONK | STARKs |
|----------|---------|-------|--------|
| **Proof size** | ~128 B | ~400 B | ~50 KB |
| **Verification** | ~2ms | ~5ms | ~50ms |
| **Trusted setup** | Per-circuit | Universal | None |
| **Quantum-safe** | No | No | Yes |

For applications with small circuits and many proofs — like games or token transfers — Groth16's compact proofs and fast verification dominate.

## ZK Tic-Tac-Toe: A Complete Example

The ZK build of tic-tac-toe uses a variant of the [Chapter 6](ch06-game-mechanics.md) model with 33 places and 35 transitions (Chapter 6's model has 30 and 34; the ZK variant uses separate `x_turn`/`o_turn` places and a `game_active` flag). It uses two circuits, both encoding the full Petri net:

**PetriTransitionCircuit** proves a move is legal. It's the general circuit described above — works for any transition in the net. Used for every move.

**PetriWinCircuit** proves a player has won. The win condition is already in the Petri net topology (win-detection transitions from Chapter 6 fire when three cells align). The circuit checks that the `win_x` or `win_o` place has a token:

```go
func (c *PetriWinCircuit) Define(api frontend.API) error {
    root := petriMimcHash(api, c.Marking[:])
    api.AssertIsEqual(root, c.StateRoot)

    winTokens := frontend.Variable(0)
    for p := 0; p < NumPlaces; p++ {
        isWinnerPlace := api.IsZero(api.Sub(c.Winner, p))
        winTokens = api.Add(winTokens, api.Mul(c.Marking[p], isWinnerPlace))
    }
    api.ToBinary(api.Sub(winTokens, 1), 8) // >= 1 token
    return nil
}
```

Because the game logic lives in the net topology, the circuit doesn't know anything about tic-tac-toe. Change the topology constants and you get ZK proofs for a different game.

### Constraint Counts

For the 33-place, 35-transition tic-tac-toe net:

| Component | Constraints | Purpose |
|-----------|-------------|---------|
| MiMC hash (pre) | ~300 | State root verification |
| MiMC hash (post) | ~300 | State root verification |
| Transition delta | ~2,300 | Topology multiplexing |
| Marking assertion | ~33 | post = pre + delta |
| Enabledness | ~264 | Bit decomposition per place |
| **Total** | **~3,200** | Full transition proof |

Proof generation takes ~100ms on commodity hardware. Verification takes ~2ms. The circuit compiles once; proofs are generated per-move.

### The Game Protocol

A complete ZK tic-tac-toe game works as:

```
Initial: state_hash = hash(empty_board)

Round n:
  1. Current player fires transition locally (computes new marking)
  2. Computes new_state_hash = MiMC(new_marking)
  3. Generates Groth16 proof: (old_hash, new_hash, transition_id, proof)
  4. Opponent verifies proof (~2ms)
  5. If valid: state_hash = new_hash, continue
  6. If invalid: cheating detected, game forfeited
```

The opponent knows *which* transition fired (the move is public) but doesn't see the full board state. Win detection is a separate proof demonstrating that a win place has a token.

## Beyond Games: Token Transfers

The same circuit structure handles blockchain token transfers. An ERC-20 transfer is a Petri net transition:

- `balances[from]` is an input place (tokens consumed)
- `balances[to]` is an output place (tokens produced)
- The guard `balances[from] >= amount` is the enabledness check

A note on what that guard is. In the net it is a *read* of `balances[from]` — a contextual arc, not a column of the incidence matrix — and contextual arcs are exactly the thing that stops a net from generating the free symmetric monoidal category of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries). Inside the circuit that distinction disappears: a proof certifies one firing, one firing has only interleaving semantics, and under interleaving semantics a read arc and a consume-then-restore self-loop are the same thing. So encoding the guard as a range check on a witnessed balance is an exact encoding, not an approximation. The boundary the circuit *cannot* erase is the other one — a transition with more inputs than outputs still costs a non-uniform constraint block, because that is a fact about $C$ and the circuit is compiled from $C$.

The arcnet project extends this with Merkle trees for account balances, enabling L1-to-L2 bridge verification:

```go
func (c *TransferCircuit) Define(api frontend.API) error {
    // Guard: balance >= amount
    diff := api.Sub(c.BalanceFrom, c.Amount)
    api.ToBinary(diff, 64)

    // Verify Merkle inclusion
    leaf := mimcHash(api, c.From, c.BalanceFrom)
    current := leaf
    for i := 0; i < 20; i++ {
        left := api.Select(c.PathIndices[i], c.PathElements[i], current)
        right := api.Select(c.PathIndices[i], current, c.PathElements[i])
        current = mimcHash(api, left, right)
    }
    api.AssertIsEqual(current, c.PreStateRoot)
    return nil
}
```

The 20-level Merkle proof means the full state doesn't need to live on-chain. Only state roots are published. The bridge contract verifies the proof and updates the root.

| Application | Places | Transition | Enabledness |
|-------------|--------|------------|-------------|
| **Tic-Tac-Toe** | 33 board+game states | Player move | Cell empty, correct turn |
| **ERC-20 Transfer** | Account balances | Transfer | Sufficient balance |
| **Workflow** | Process stages | State change | Required approvals |
| **Voting** | Ballot states | Cast vote | Eligible, hasn't voted |

Same circuit structure. Different topology.

## What ZK Doesn't Hide

A common misconception is that ZK hides the transition ID. It doesn't: the verifier sees *which* transition fired. What's hidden is the *state* — the full marking of every place.

In tic-tac-toe, the opponent knows a move was made (say, X plays center) but doesn't see the accumulated pattern of tokens across all 33 places. In a token transfer, the network knows a transfer happened but doesn't see individual balances.

For applications where even the transition must be hidden, an additional layer of indirection is needed — proving that *some* valid transition fired without revealing which one. This is possible but increases circuit complexity significantly.

## The Proof Pipeline

A complete proof cycle for any Petri net application:

1. **Client fires transition** locally (computes new marking)
2. **Client computes** MiMC state roots for pre and post markings
3. **Prover** receives witness (pre/post markings + transition ID) and generates Groth16 proof
4. **Proof returned** as ~128 bytes with public inputs
5. **Anyone can verify** the proof in constant time (~2ms)
6. **On-chain verification** possible via auto-generated Solidity verifier contract

The circuit doesn't encode tic-tac-toe, or token transfers, or workflow rules. It encodes *Petri net semantics*: hash the marking, compute the delta from topology, assert the change, check enabledness. The application logic is in the net. The privacy is in the proof.
