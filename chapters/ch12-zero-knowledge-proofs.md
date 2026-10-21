# Zero-Knowledge Proofs

**Learning objective**: Prove a state transition is valid without revealing the state.

Every Petri net model in this book has been transparent — all token counts visible to all participants. The coffee shop owner sees inventory levels. Both tic-tac-toe players see the board. The hospital monitor sees patient states. But some applications require privacy. A voter should prove they cast a valid ballot without revealing their choice. A financial institution should verify a transfer is valid without exposing account balances. A player should prove their move is legal without revealing their strategy.

Zero-knowledge proofs make this possible, and a Petri net is a natural input to them: it is already a formal specification of valid state transitions, and a ZK circuit turns that specification into a cryptographic proof system. The prover demonstrates they followed the rules; the verifier confirms it without seeing the state.

## The Statement Being Proven

Every ZK proof in this system proves one statement:

> "I know a valid marking that, when transition T fires, produces the claimed next state."

More precisely:

1. The **pre-state root** (a hash) matches a specific private marking
2. The **post-state root** matches a specific private marking
3. The chosen **transition was enabled** — all input places had sufficient tokens
4. The **marking change is correct** — post = pre + delta, where delta comes from the net topology

The verifier sees only the two state roots and the transition ID. The token counts are not inputs to verification; whether they stay hidden depends on what else is public, which [What ZK Doesn't Hide](#what-zk-doesnt-hide) takes up.

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

The hash function matters. Inside a ZK circuit, every operation becomes arithmetic constraints on a finite field. SHA-256 needs thousands of boolean operations — roughly 25,000 constraints per compression. **MiMC** (Minimal Multiplicative Complexity) is designed for arithmetic circuits: it operates over the same finite field the circuit uses. gnark's BN254 MiMC absorbs one field element at a time, running 110 rounds of $x \mapsto x^5$ on each, and each round costs three multiplications ([`encryptPow5`](https://github.com/Consensys/gnark/blob/v0.14.0/std/internal/mimc/encrypt.go) in gnark v0.14.0, the version petri-pilot pins; 110 rounds is `mimcNbRounds` in gnark-crypto's `bn254/fr/mimc`). That is 330 constraints per element, so hashing a marking of $N$ places costs $330N$ (derived from the code, not measured).

The hash used throughout this chapter is from petri-pilot's `zk-tictactoe/petri_circuits.go`:

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

Inverting MiMC is believed to be infeasible, so the root binds the prover to one marking. It hides that marking only conditionally: the root carries no random salt, so anyone who can list the candidate markings can hash each one and compare. A net with a small reachable set, like tic-tac-toe's, offers no protection by the hash alone.

## The Circuit in Five Steps

The gnark circuit for a Petri net transition proof has five steps. Each adds constraints that the prover must satisfy. The excerpts below are from `PetriTransitionCircuit` in the same file; go-pflow's `petrigen` emits the same `Define` from a template ([`zkcompile/petrigen/templates.go`](https://github.com/pflow-xyz/go-pflow/blob/main/zkcompile/petrigen/templates.go)), adding a guard block when the model has guards.

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

This ensures the marking changed exactly as the topology dictates, with no token appearing or vanishing anywhere else. Each listed place moves by exactly one token, so every arc has weight 1 (the `ArcDef` comment in `petri_state.go` says as much for tic-tac-toe). The ERC-20 circuit later in the chapter replaces the 1 with an amount.

### Step 5: Assert Enabledness

Every input place of the selected transition must hold at least one token (excerpt, comments trimmed):

```go
for p := 0; p < NumPlaces; p++ {
    isInput := frontend.Variable(0)
    for t := 0; t < NumTransitions; t++ {
        isThis := api.IsZero(api.Sub(c.Transition, t))
        for _, inp := range Topology[t].Inputs {
            if inp == p {
                isInput = api.Add(isInput, isThis)
            }
        }
    }
    diff := api.Sub(c.PreMarking[p], isInput)
    api.ToBinary(diff, 8) // 8 bits is enough for small token counts
}
```

`isInput` is 1 when place `p` is an input of the selected transition and 0 otherwise. `ToBinary` decomposes `pre[p] - isInput` into 8 bits. If an input place were empty, the difference would wrap to a huge field element, the decomposition would fail, and no proof could be generated. This is the standard gnark pattern for range checks: decompose into bits to prove non-negativity. Because the check runs on every place, it also bounds every pre-marking count below 256, matching the `uint8` marking type in `zk-tictactoe/petri_state.go`.

## Topology as Circuit Constants

The central property of this approach is that **the Petri net topology is baked into the circuit as constants**. For the tic-tac-toe net below, an excerpt of `petri_state.go`:

```go
var Topology = [NumTransitions]ArcDef{
    // X plays: consume cell + x_turn, produce x_piece + o_turn
    TXPlay00: {[]int{P00, PlaceXTurn}, []int{X00, PlaceOTurn}},
    // ...
    // X wins: consume 3 x_pieces + game_active, produce win_x + return pieces
    TXWinRow0: {[]int{X00, X01, X02, PlaceGameActive}, []int{PlaceWinX, X00, X01, X02}},
    // ...
}
```

This means:

1. The circuit contains no application-specific logic. The same `Define()` method works for any net whose arcs have weight 1 and which has no guards.
2. Topology changes require recompilation. Adding a place or transition means a new trusted setup.
3. The circuit is auditable. Anyone can inspect the topology to check that it matches the claimed rules.

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

The figures are typical orders of magnitude for each system, not measurements of the circuits in this chapter. The Groth16 proof size follows from its shape: on BN254 a compressed G1 point is 32 bytes and a G2 point 64, so $2 \times 32 + 64 = 128$.

For applications with small circuits and many proofs — like games or token transfers — Groth16's compact proofs and fast verification dominate.

## ZK Tic-Tac-Toe: A Complete Example

The ZK build of tic-tac-toe uses a variant of the [Chapter 6](ch06-game-mechanics.md) model with 33 places and 35 transitions (Chapter 6's model has 30 and 34; the ZK variant uses separate `x_turn`/`o_turn` places and a `game_active` flag). It uses two circuits, both encoding the full Petri net:

**PetriTransitionCircuit** proves a move is legal. It's the general circuit described above — works for any transition in the net. Used for every move.

**PetriWinCircuit** proves a player has won. The win condition is already in the Petri net topology (win-detection transitions from Chapter 6 fire when three cells align). The circuit checks that the `win_x` or `win_o` place has a token (comments trimmed):

```go
func (c *PetriWinCircuit) Define(api frontend.API) error {
    root := petriMimcHash(api, c.Marking[:])
    api.AssertIsEqual(root, c.StateRoot)

    winTokens := frontend.Variable(0)
    for p := 0; p < NumPlaces; p++ {
        isWinnerPlace := api.IsZero(api.Sub(c.Winner, p))
        winTokens = api.Add(winTokens, api.Mul(c.Marking[p], isWinnerPlace))
    }
    api.ToBinary(api.Sub(winTokens, 1), 8)

    isXWin := api.IsZero(api.Sub(c.Winner, PlaceWinX))
    isOWin := api.IsZero(api.Sub(c.Winner, PlaceWinO))
    validWinner := api.Add(isXWin, isOWin)
    api.AssertIsEqual(validWinner, 1)

    return nil
}
```

The last four lines restrict `Winner` to `win_x` (place 29) or `win_o` (place 30). Without them a prover could name any marked place, an occupied cell for instance, as the winner.

Because the game logic lives in the net topology, the circuit doesn't know anything about tic-tac-toe. Change the topology constants and you get ZK proofs for a different game.

The tests in petri-pilot's `zk-tictactoe/petri_circuits_test.go` check both circuits against gnark's test prover. `TestPetriTransitionCircuit_FullGameXWins` plays X to a left-column win (cells 0, 1, 3, 4, 6, alternating X and O) and asserts that a proof succeeds for every move and for the `x_win_col0` firing; `TestPetriWinCircuit_XWins` and `TestPetriWinCircuit_OWins` assert that the win proof succeeds for each player. Two tests assert failure: `TestPetriTransitionCircuit_WrongTurn` builds a witness that fires an X move on O's turn, and `TestPetriWinCircuit_NoWinnerFails` claims a win for X after two moves. Neither circuit has been re-run for this chapter.

### Constraint Counts

No test logs the constraint count of this circuit (the compile tests call `frontend.Compile` but do not print `GetNbConstraints`), so the 33-place, 35-transition net has no measured total. What can be read off the code:

| Component | Constraints | Source |
|-----------|-------------|--------|
| MiMC hash (pre) | $33 \times 330 = 10{,}890$ | 33 elements at 330 each (derived above) |
| MiMC hash (post) | $10{,}890$ | same |
| Delta accumulation | none of its own | additions are linear and fold into the constraints that use them |
| Transition selectors | 35 `IsZero` calls in step 3, a further $33 \times 35$ in step 5 | per-call cost not measured |
| Marking assertion | 33 | one `AssertIsEqual` per place |
| Enabledness | 33 eight-bit decompositions | one `ToBinary(·, 8)` per place |

The two hashes dominate, so the total exceeds 21,780. Proving and verification times have not been benchmarked for this chapter. The circuit compiles once, and proofs are generated per move.

### The Game Protocol

A ZK tic-tac-toe game runs roughly as follows (a sketch of the protocol, not code):

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

The opponent knows *which* transition fired, because the move is public. Win detection is a separate proof demonstrating that a win place has a token. What the opponent can and cannot learn about the board is taken up in [What ZK Doesn't Hide](#what-zk-doesnt-hide) below.

## Beyond Games: Token Transfers

The same circuit structure handles token transfers. An ERC-20 transfer is a Petri net transition:

- `balances[from]` is an input place (tokens consumed)
- `balances[to]` is an output place (tokens produced)
- The guard `balances[from] >= amount` is the enabledness check

petri-pilot's `zk-erc20/petri_circuits.go` applies the circuit to a two-account ERC-20 net with 5 places (`totalSupply`, two balances, two allowances) and 10 transitions (transfer, approve, transferFrom, mint and burn, one of each per account; `zk-erc20/petri_state.go`). Two things change. `Amount` becomes a public input and weights every arc, so a transfer of 100 debits 100 and credits 100:

```go
for _, p := range arc.Inputs {
    delta := api.Mul(isThis, c.Amount)
    expectedPost[p] = api.Sub(expectedPost[p], delta)
}
for _, p := range arc.Outputs {
    delta := api.Mul(isThis, c.Amount)
    expectedPost[p] = api.Add(expectedPost[p], delta)
}
```

And the enabledness check subtracts the amount consumed rather than 1, decomposed into 64 bits instead of 8:

```go
diff := api.Sub(c.PreMarking[p], consumed)
api.ToBinary(diff, 64) // 64 bits for token amounts up to ~1.8 * 10^19
```

Approve transitions are the exception: they set an allowance to `Amount` rather than adding to it, and skip the enabledness check. The same file adds an `ERC20InvariantCircuit` that proves `balance_0 + balance_1 = totalSupply` against a state root.

A note on what that guard is. In a model that states it explicitly, as the DSL of [Chapter 4](ch04-token-language.md) does, it is a *read* of `balances[from]` — a contextual arc, not a column of the incidence matrix — and contextual arcs are exactly the thing that stops a net from generating the free symmetric monoidal category of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries). Inside the circuit that distinction disappears: a proof certifies one firing, one firing has only interleaving semantics, and under interleaving semantics a read arc and a consume-then-restore self-loop are the same thing. So encoding the guard as a range check on a witnessed balance, as `zk-erc20` does, is exact rather than an approximation. The boundary the circuit *cannot* erase is the other one — a transition with more inputs than outputs still costs a non-uniform constraint block, because that is a fact about $C$ and the circuit is compiled from $C$.

A flat hash over every balance does not scale to many accounts, because every proof would hash the whole marking. [Chapter 14](ch14-on-chain-verification.md) replaces it with a Merkle tree, so that a proof touches only the accounts involved and only the root is published on-chain.

| Application | Places | Transition | Enabledness |
|-------------|--------|------------|-------------|
| **Tic-Tac-Toe** | 33 board+game states | Player move | Cell empty, correct turn |
| **ERC-20 Transfer** | Account balances | Transfer | Sufficient balance |
| **Workflow** | Process stages | State change | Required approvals |
| **Voting** | Ballot states | Cast vote | Eligible, hasn't voted |

Only the first two rows are built on this circuit in code. The workflow row is the same mapping on paper. bitwrap-io's vote circuit ([`prover/vote_gen.go`](https://github.com/stackdump/bitwrap-io/blob/main/prover/vote_gen.go)) is a different construction: it proves membership in a Merkle registry of voters, binds a nullifier so that nobody votes twice, and commits to the choice, rather than proving a marking change.

## What ZK Doesn't Hide

A common misconception is that ZK hides the transition ID. It doesn't: the verifier sees *which* transition fired. What can be hidden is the *state*, the full marking of every place, but only to the extent the verifier cannot reconstruct it from public data.

In tic-tac-toe it can. The initial marking is a constant of the model (`InitialMarking` in `petri_state.go`), and every move's transition ID is public, so the opponent can replay the moves and recover all 33 places. This is the event-sourcing fold of [Chapter 20](ch20-dual-implementation.md) applied to public events. The tic-tac-toe proofs show that every move was legal; they hide nothing about the board. In the ERC-20 circuit, `Amount` is public as well, so balances stay hidden only if the verifier does not know the starting balances. Even then, as noted under MiMC above, an unsalted root gives away any marking drawn from a set small enough to enumerate.

For applications where even the transition must be hidden, an additional layer of indirection is needed — proving that *some* valid transition fired without revealing which one. This is possible but increases circuit complexity significantly.

## The Proof Pipeline

A complete proof cycle for any Petri net application:

1. **Client fires transition** locally (computes new marking)
2. **Client computes** MiMC state roots for pre and post markings
3. **Prover** receives witness (pre/post markings + transition ID) and generates Groth16 proof
4. **Proof returned** as ~128 bytes with public inputs
5. **Anyone can verify** the proof in constant time (~2ms)
6. **On-chain verification** possible via auto-generated Solidity verifier contract

The circuit doesn't encode tic-tac-toe, or token transfers, or workflow rules. It encodes *Petri net semantics*: hash the marking, compute the delta from topology, assert the change, check enabledness. The application logic is in the net, and the proof certifies that one firing followed it.
