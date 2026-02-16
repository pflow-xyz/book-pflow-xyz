# Complex State Machines — Texas Hold'em

**Learning objective**: Model multi-player, multi-phase systems with roles and guards.

The previous chapters built progressively: resources (coffee shop), two-player games (tic-tac-toe), constraint puzzles (Sudoku), optimization (knapsack), and chemistry (enzyme kinetics). Each used a focused subset of Petri net features. Texas Hold'em poker demands all of them simultaneously: sequential phases, multi-player turn control, role-based permissions, conditional betting, and event sourcing for audit trails.

This chapter shows that the formalism scales. The same four-term primitive — cell, func, arrow, guard — handles a system with dozens of states, multiple interacting players, and complex conditional logic. No new machinery is needed. The complexity is in the model, not the formalism.

## The Challenge

Poker has intricate state:

- **Phases**: preflop → flop → turn → river → showdown — a sequential state machine
- **Turn order**: Players act in sequence, skipping folded players
- **Actions**: fold, check, call, raise — each with preconditions on chips, bets, and game state
- **Roles**: Only the dealer can deal cards, only active players can bet, only the admin can determine winners
- **Audit**: Every action must be logged for replay and verification

Traditional implementations scatter this logic across conditionals, state variables, and validation functions. A Petri net makes it explicit: the structure *is* the specification. Illegal states are structurally unreachable rather than checked-for at runtime.

## Phase Places

The core flow is a **WorkflowNet** — a sequential state machine where a single token traces a path through phases:

```
waiting -> preflop -> flop -> turn_round -> river -> showdown -> complete
```

Each phase is a place. Phase transitions move the game forward:

```
preflop --> deal_flop --> flop
             ^
betting_done +
```

The `deal_flop` transition requires two things: a token in `preflop` (we're in that phase) and a token in `betting_done` (all players have acted). Both must be present for the transition to fire. This ensures cards aren't dealt mid-betting.

The phase token is a cursor — exactly one phase place holds a token at any time. The P-invariant:

$$M(\text{waiting}) + M(\text{preflop}) + M(\text{flop}) + M(\text{turn}) + M(\text{river}) + M(\text{showdown}) + M(\text{complete}) = 1$$

This is the WorkflowNet guarantee: mutual exclusion. The game is always in exactly one phase.

## Turn Tokens and Role-Based Access

Each player has a **turn place**: `p0_turn`, `p1_turn`, ..., `p4_turn`. When it's Player 0's turn, only `p0_turn` holds a token. Player actions consume their turn token and produce the next player's:

```
p0_check:
  inputs:  [p0_turn, p0_active]
  outputs: [p1_turn]
```

Player 0 can only check when two conditions hold: it's their turn (`p0_turn` has a token) AND they're still in the hand (`p0_active` has a token). After checking, the turn token moves to Player 1.

This is the same turn control mechanism from tic-tac-toe (Chapter 6), generalized to N players. The token passes in sequence, enforcing strict turn order without explicit checks.

### Folding and Skipping

When a player folds, their `p_active` token is consumed:

```
p2_fold:
  inputs:  [p2_turn, p2_active]
  outputs: [p3_turn]
  // Note: p2_active is NOT returned
```

Subsequent rounds must skip folded players. A `skip` transition handles this:

```
p2_skip:
  inputs:  [p2_turn]  // No p2_active required
  outputs: [p3_turn]
  guard:   {!p2_active}  // Only fire when player is folded
```

The guard ensures the skip only fires for folded players. Active players can't be skipped — they must act. This combination of structural enabling (turn tokens) and guards (active status) handles the full complexity of variable-player turn order.

## Betting Conditions as Guard Expressions

Some actions require conditions beyond token availability. **Guards** from Chapter 4 handle this:

```lisp
(action p0_raise
  :guard {bets[0] >= current_bet && chips[0] >= raise_amount})
```

This ensures:
- Player has matched the current bet (can't raise without calling first)
- Player has enough chips to raise

Guards add business logic without changing the Petri net structure. The net handles *what can happen* (which transitions exist). Guards handle *when it can happen* (what conditions must hold). This separation is essential for complex systems — the structural properties (deadlock freedom, mutual exclusion) can be verified from the net alone, independent of the guard expressions.

### The Full Action Set

For each player, four actions with their preconditions:

| Action | Preconditions | Effect |
|--------|---------------|--------|
| `fold` | Turn token + active | Remove from hand |
| `check` | Turn token + active + no outstanding bet | Pass action |
| `call` | Turn token + active + sufficient chips | Match current bet |
| `raise` | Turn token + active + sufficient chips | Increase bet |

Five players × four actions = 20 player transitions, plus 5 skip transitions for folded players, plus 7 phase transitions (start, deal flop/turn/river, showdown, determine winner, end hand). Total: **32 transitions**.

## The Model Structure

The full model:

```
Places (17+):
  waiting, preflop, flop, turn_round,     -- Phase places
  river, showdown, complete
  p0_turn, p1_turn, p2_turn,              -- Turn tokens
  p3_turn, p4_turn
  p0_active, p1_active, p2_active,        -- Active markers
  p3_active, p4_active
  betting_done                            -- Sync signal

Transitions (32):
  start_hand, deal_flop, deal_turn,        -- Phase transitions
  deal_river, go_showdown,
  determine_winner, end_hand
  p0_fold, p0_check, p0_call, p0_raise,   -- Player 0 actions
  p1_fold, p1_check, p1_call, p1_raise,   -- Player 1 actions
  ...                                      -- (5 players x 4 actions)
  p0_skip, p1_skip, ...                    -- Skip folded players
```

### Role-Based Access Control

Not all transitions are available to all players. The model uses **roles** to restrict who can fire which transitions:

| Role | Transitions | Purpose |
|------|-------------|---------|
| `dealer` | `deal_flop`, `deal_turn`, `deal_river` | Only dealer advances phases |
| `player0` | `p0_fold`, `p0_check`, `p0_call`, `p0_raise` | Player-specific actions |
| `admin` | `end_hand`, `determine_winner` | Game control |

When deployed as a service, the API enforces these roles — a player can't fire another player's transitions. The Petri net defines *what* each role can do; the runtime enforces *who* has which role.

## Event Sourcing for Audit Trails

Every transition firing creates an **immutable event**:

```json
[
  {"action": "start_hand", "time": "10:00:00"},
  {"action": "p0_raise", "amount": 50, "time": "10:00:15"},
  {"action": "p1_call", "time": "10:00:23"},
  {"action": "p2_fold", "time": "10:00:31"},
  {"action": "deal_flop", "cards": ["Ah", "Ks", "7d"], "time": "10:00:35"}
]
```

The Petri net's discrete transitions map perfectly to event sourcing:

- **Replay**: Given the initial marking and the event sequence, reconstruct any game state by replaying transitions
- **Audit**: Verify that every transition was enabled when it fired — illegal moves are provably impossible
- **Undo**: Roll back to any previous state by replaying up to that point

This is the event sourcing pattern from the workspace's `CLAUDE.md`: $\text{State}(t) = \text{fold}(\text{apply}, \text{initialState}, \text{events}[0..t])$. Each event is a transition firing. Each state is a marking. The fold is the firing rule applied sequentially.

## ODE Analysis of Game Flow

While the game runs discretely, ODE simulation provides strategic analysis. Converting the Petri net to continuous dynamics lets us evaluate position strength:

```go
// For each possible action, create hypothetical state
// and simulate forward
for _, action := range availableActions {
    hypState := stateutil.Apply(currentState, action.Delta)

    prob := solver.NewProblem(net, hypState,
        [2]float64{0, 5.0}, rates)
    sol := solver.Solve(prob, solver.Tsit5(), solver.FastOptions())
    final := sol.GetFinalState()

    score := final["win_p0"] - final["win_p1"]
}
```

The ODE simulation explores possible game continuations:
- Higher flow to `win_p0` indicates stronger positions
- Raising in a strong position increases flow to win places
- Folding prevents further loss when position is weak

This is the same pattern as tic-tac-toe (Chapter 6), but with a much larger state space. The strategic insights are less precise — poker has hidden information (hole cards) that the ODE can't fully model — but they provide useful guidance for position evaluation.

### Hand Strength from Structure

The poker model can include **draw potential places** that track partial hands:

- `flush_draw`: 4 cards of the same suit
- `straight_draw`: 4 cards in sequence
- `overcards`: hole cards above board cards

These places feed into scoring transitions that evaluate hand strength. The ODE simulation integrates over possible completions, producing an aggregate strength score. Combined with opponent range estimation (which hands the opponent likely holds given their actions), this creates a complete decision framework.

## Compositional Design

The Texas Hold'em model is a composition of simpler patterns:

1. **WorkflowNet** — Phase sequence (waiting → preflop → ... → complete)
2. **ResourceNet** — Chip tracking (bet placement, pot management)
3. **GameNet** — Turn control and player actions
4. **Guards** — Conditional betting logic

Each pattern is independently verifiable:
- The phase sequence guarantees mutual exclusion (one phase at a time)
- Chip tracking preserves conservation (total chips constant)
- Turn control ensures exactly one player acts at a time
- Guards prevent illegal bets

Composition adds links between patterns — the `betting_done` signal connects turn control to phase advancement — without breaking any individual guarantee. This is the assume-guarantee reasoning from Chapter 4: verify components in isolation, check only the boundaries.

## Why Petri Nets for Complex Systems

Traditional game code mixes state, rules, and UI into monolithic logic. A Petri net separates concerns:

- **Structure** = What's possible (places, transitions, arcs)
- **State** = What's current (token marking)
- **Rules** = What's allowed (guards, roles)
- **History** = What happened (events)

This separation makes complex systems:
- **Verifiable**: Prove invariants (one player acts at a time, chips conserved)
- **Replayable**: Deterministic event sequence from any initial state
- **Analyzable**: ODE simulation for strategy, reachability for deadlock detection
- **Evolvable**: Add rules (side pots, all-in) without rewriting existing logic

Texas Hold'em is at the complex end of the spectrum covered in Part II. It combines every pattern from the previous chapters into a single model. The formalism doesn't struggle with this complexity — it absorbs it. The four-term primitive handles a coffee shop, a tic-tac-toe board, a Sudoku grid, a knapsack, an enzyme reaction, and a poker table with the same vocabulary.

Part III of this book shifts focus from building models to analyzing them — process mining, zero-knowledge proofs, and the tooling that makes Petri net development practical.
