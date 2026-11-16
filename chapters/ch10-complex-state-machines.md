# Complex State Machines — Texas Hold'em

**Learning objective**: Model multi-player, multi-phase systems with roles and guards.

The previous chapters built progressively: resources (coffee shop), two-player games (tic-tac-toe), constraint puzzles (Sudoku), optimization (knapsack), and chemistry (enzyme kinetics). Each used a focused subset of Petri net features. Texas Hold'em asks for several at once: sequential phases, multi-player turn control, role-based permissions, conditional betting, and event sourcing for audit trails.

The model in this chapter is a tutorial Hold'em net, `services/texas-holdem.json`: five seats, four streets, 18 places and 32 transitions in one JSON file, from which an application generator produced an event-sourced Go service (`examples/texasholdem`) and a browser demo. Neither the generated code nor the demo is public any longer; the file paths below name where each cited piece lived. The same primitives as the earlier chapters carry the phases, the turn order and the event log. Chips, bet sizing and the winner do not fit into this net, and the chapter marks where the structure stops.

## The Challenge

Poker has intricate state:

- **Phases**: preflop → flop → turn → river → showdown — a sequential state machine
- **Turn order**: Players act in sequence, skipping folded players
- **Actions**: fold, check, call, raise — each with preconditions on chips, bets, and game state
- **Roles**: Only the dealer should deal cards, and only players still in the hand should bet
- **Audit**: Every action must be logged for replay and verification

Traditional implementations scatter this logic across conditionals, state variables, and validation functions. A Petri net makes the parts it can carry explicit: where a rule is in the structure, an illegal move is not enabled rather than checked for at runtime. The rest of this chapter shows how far that reaches in this model.

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

The `deal_flop` transition requires two things: a token in `preflop` (we're in that phase) and a token in `betting_done` (the betting round is over). Both must be present for the transition to fire, so cards are not dealt mid-betting. "Over" is narrower than in real poker: `betting_done` is produced by seat 4's action, so each street is exactly one orbit of the table, p0 through p4, and a raise does not reopen the action to seats that already acted. Re-raise loops are not in this net.

The phase token is a cursor — exactly one phase place holds a token at any time. The P-invariant:

$$M(\text{waiting}) + M(\text{preflop}) + M(\text{flop}) + M(\text{turn\_round}) + M(\text{river}) + M(\text{showdown}) + M(\text{complete}) = 1$$

Reading the arcs, every one of the 32 transitions either moves one token along this chain or leaves it untouched; we checked that by hand against the JSON, not with an invariant solver. This is the WorkflowNet guarantee: mutual exclusion. The game is always in exactly one phase.

## Turn Tokens and Role-Based Access

Each player has a **turn place**: `p0_turn`, `p1_turn`, ..., `p4_turn`. When it's Player 0's turn, only `p0_turn` holds a token. Player actions consume their turn token and produce the next player's. From the model file:

```
p0_check:
  inputs:  [p0_turn, p0_active]
  outputs: [p0_active, p1_turn]
```

Player 0 can only check when two conditions hold: it's their turn (`p0_turn` has a token) AND they're still in the hand (`p0_active` has a token). Checking hands the active token back and moves the turn token to Player 1. Call and raise have the same arcs.

This is the turn control mechanism from tic-tac-toe ([Chapter 6: Game Mechanics](ch06-game-mechanics.md)), generalized from one alternating `Next` place to a ring of five turn places. The turn order is a conservation law of the net. Reading the arcs, every transition preserves

$$M(\text{waiting}) + \sum_{i=0}^{4} M(p_i\text{\_turn}) + M(\text{betting\_done}) + M(\text{showdown}) + M(\text{complete}) = 1$$

so at most one seat holds the turn at any reachable marking. A player action moves the token from one seat to the next (seat 4 hands it to `betting_done`), each deal transition turns `betting_done` back into `p0_turn`, and the phase transitions at either end of the hand pass it through `showdown`, `complete` and `waiting`. As with the phase invariant, this was checked by hand, not run.

### Folding and Skipping

When a player folds, their `p_active` token is consumed:

```
p2_fold:
  inputs:  [p2_turn, p2_active]
  outputs: [p3_turn]
  // Note: p2_active is NOT returned
```

A folded seat has no active token, so its fold, check, call and raise are all disabled. Only the skip transition, which needs nothing but the turn token, can pass the turn along:

```
p2_skip:
  inputs:  [p2_turn]  // No p2_active required
  outputs: [p3_turn]
```

The model declares no guard on the skip, and its description reads "Skip player 2 (all-in/eliminated)". So `p2_skip` is also enabled for a seat that is still in the hand: nothing in the net forces an active player to act rather than pass. Restricting the skip to folded seats would take an inhibitor arc from `p2_active` — a contextual arc, outside the incidence matrix ([Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries)) — or a guard. The model has neither, so the structure guarantees that a folded seat can only pass, not that an active seat must play.

## Betting Conditions as Guard Expressions

Some actions require conditions beyond token availability. **Guards** from [Chapter 4: The Token Language](ch04-token-language.md) handle this. This model declares none; the generated README's transition table shows `-` in the Guard column for all 32 transitions. The guard a raise would need looks like this (a sketch in Chapter 4's syntax, not from the model):

```lisp
(action p0_raise
  :guard {bets[0] >= current_bet && chips[0] >= raise_amount})
```

It would ensure:
- Player has matched the current bet (can't raise without calling first)
- Player has enough chips to raise

In the demo, chip stacks, bets, blinds and the pot live in the browser's JavaScript state (`frontends/texas-holdem/main.js`), not in the net.

Guards add business logic without changing the Petri net structure. The net handles *what can happen* (which transitions exist). Guards handle *when it can happen* (what conditions must hold). A guard can only disable a firing the structure allows, so every marking the guarded net reaches is also reachable without guards, and invariants and mutual exclusion proved from the net alone still hold. Deadlock freedom does not carry over the same way: a guard that is never true disables a transition the structural analysis counted on.

### The Full Action Set

For each player, four actions. The net checks the token preconditions; the poker condition in the last column is left to code in this model:

| Action | Net precondition | Net effect | Poker condition (not in the net) |
|--------|------------------|------------|----------------------------------|
| `fold` | Turn token + active | Turn to next seat; active token consumed | — |
| `check` | Turn token + active | Turn to next seat; active token returned | No outstanding bet |
| `call` | Turn token + active | Turn to next seat; active token returned | Sufficient chips |
| `raise` | Turn token + active | Turn to next seat; active token returned | Bet matched, sufficient chips |

Five players × four actions = 20 player transitions, plus 5 skip transitions, plus 7 phase transitions (start, deal flop/turn/river, showdown, determine winner, end hand). Total: **32 transitions**.

## The Model Structure

The full model:

```
Places (18):
  waiting, preflop, flop, turn_round,     -- Phase places
  river, showdown, complete
  p0_turn, p1_turn, p2_turn,              -- Turn tokens
  p3_turn, p4_turn
  p0_active, p1_active, p2_active,        -- Active markers (initial 1)
  p3_active, p4_active
  betting_done                            -- Sync signal

Transitions (32):
  start_hand, deal_flop, deal_turn,        -- Phase transitions
  deal_river, go_showdown,
  determine_winner, end_hand
  p0_fold, p0_check, p0_call, p0_raise,   -- Player 0 actions
  p1_fold, p1_check, p1_call, p1_raise,   -- Player 1 actions
  ...                                      -- (5 players x 4 actions)
  p0_skip, p1_skip, ...                    -- Pass the turn (needs only pN_turn)
```


### A Defect in the Arcs

One defect is visible in the arcs. `end_hand` puts a fresh token into every `pN_active` without consuming the old one, and check, call and raise return the active token they take. So a seat that played a hand to the end starts the next hand with two active tokens; a single fold then leaves it with one, and on the next street that seat can check, call or raise again. The active places are not 1-bounded across hands. The fix is to restore only the seats that folded instead of refilling every seat; the model has not been patched.

### Role-Based Access Control

The model's `access` block restricts the phase transitions to a `dealer` role and declares `admin` as inheriting from `dealer`:

| Role | Transitions |
|------|-------------|
| `dealer` | `start_hand`, `deal_flop`, `deal_turn`, `deal_river`, `go_showdown`, `determine_winner`, `end_hand` |
| `admin` | inherits `dealer` (also listed on `start_hand`) |

There are no per-seat roles. Nothing in the model binds `p0_raise` to the person in seat 0, so any caller can fire any seat's action while that seat holds the turn token. Enforcement is also thinner than the declaration. The generator could produce a permission middleware from the `access` block, but the generated example passes it an empty rule list (`examples/texasholdem/service.go`, `accessRules := []*AccessControl{}`), and its router (`api.go`) applies only an optional authentication middleware, not the permission check. In this model the roles are a declaration; the runtime does not yet enforce them.

## Event Sourcing for Audit Trails

Every transition in the model names an event type (`p0_raise` → `p0_raised`, `deal_flop` → `flop_dealt`), and every firing in the generated service appends one event to an event store. A log looks roughly like this (a sketch: the `amount` and `cards` fields come from the model's event schema, the times are made up):

```json
[
  {"action": "start_hand", "time": "10:00:00"},
  {"action": "p0_raise", "amount": 50, "time": "10:00:15"},
  {"action": "p1_call", "time": "10:00:23"},
  {"action": "p2_fold", "time": "10:00:31"},
  {"action": "deal_flop", "cards": "Ah Ks 7d", "time": "10:00:35"}
]
```

The Petri net's discrete transitions map directly onto event sourcing, and the generated code implements three uses of it:

- **Replay**: Given the initial marking and the event sequence, reconstruct any game state by replaying transitions. `GET /api/texasholdem/{id}/at/{version}` replays the stream up to a version.
- **Audit**: A firing that was not enabled cannot appear in the log. `Execute` in the generated `aggregate.go` refuses any transition that cannot fire from the current marking before it appends an event.
- **Undo**: `POST /api/texasholdem/{id}/truncate` discards every event after a target version. Truncation rewrites the stream, so the log is append-only only as long as nobody calls it.

This is the event sourcing pattern, revisited in [Chapter 20: Dual Implementation and Verification](ch20-dual-implementation.md): $\text{State}(t) = \text{fold}(\text{apply}, \text{initialState}, \text{events}[0..t])$. Each event is a transition firing. Each state is a marking. The fold is the firing rule applied sequentially.

## ODE Analysis of Hand Strength

The demo's strategic analysis is not an ODE over the 18-place game net. For each decision, the frontend builds a separate eight-place net (`buildHandStrengthPetriNet` in `main.js`) whose initial marking and rates are set from card heuristics, solves it with the pflow.xyz Tsit5 solver, and reads values off the final state. From `computeODEHandStrength`:

```javascript
const model = buildHandStrengthPetriNet(holeCards, communityCards)

const net = Solver.fromJSON(model)
const initialState = Solver.setState(net)
const rates = Solver.setRates(net)

const prob = new Solver.ODEProblem(net, initialState, [0, solverParams.tspan], rates)
const opts = { dt: solverParams.dt, adaptive: solverParams.adaptive }
const solution = Solver.solve(prob, Solver.Tsit5(), opts)
// ...
const strength = Math.min(1, Math.max(0,
  preflopQuality * 0.5 + currentHandValue * 0.3 + baseStrength * 0.15 + improvement * 0.05
))
```

The small net has places for the current made hand, preflop quality, kicker, three draws (`flush_draw`, `straight_draw`, `pair_draw`), improvement potential and a `hand_strength` output. The draw places start at a completion probability from the rule of 2 and 4 (outs × 4% with two cards to come, outs × 2% with one), and their transitions feed `hand_strength`. The final value of `hand_strength` is then blended with three of the input places by the fixed weights above, and each action's expected value comes from hand-written formulas: fold is 0; a call is strength × (pot + call) − call, halved when strength is below the pot odds; a raise is scaled up above a 0.6 strength threshold. The ODE is one aggregation step inside a heuristic, and the heuristic, not the net topology, decides the ranking.

That is the opposite of tic-tac-toe ([Chapter 6](ch06-game-mechanics.md)), where the ODE runs on the game net itself. Poker also has hidden information (the opponents' hole cards), which neither net models; the demo does no opponent range estimation. Poker hand ranking comes back in [Chapter 13: Topology-Driven Verification](ch13-topology-driven-verification.md) and [Chapter 15: Exponential Weights and Scoring Systems](ch15-exponential-weights.md), as a question of what belongs inside a net and what belongs outside it.

## Compositional Design

The Texas Hold'em model combines simpler patterns:

1. **WorkflowNet** — Phase sequence (waiting → preflop → ... → complete)
2. **GameNet** — Turn control and player actions
3. **Roles** — The `access` block on the phase transitions

Two patterns a full poker engine needs are not in the net: a **ResourceNet** for chips (bet placement, pot management, conservation of total chips) and **guards** for conditional betting. In this model both live in frontend code.

The patterns that are present each carry a property you can check from the arcs: the phase sequence gives mutual exclusion (one phase at a time), and turn control gives at most one seat holding the turn. The `betting_done` place links them, connecting turn control to phase advancement, and both invariants above hold in the linked net, checked by hand on the whole model. This is the shape of the assume-guarantee reasoning from [Chapter 4](ch04-token-language.md): verify components in isolation, check only the boundaries. Here the two components share a place, and [Appendix E](appendix-e-categorical-foundations.md#composition-preserves-safety-not-liveness) explains why gluing along a place needs that boundary check rather than getting the invariants for free.

## Why Petri Nets for Complex Systems

Traditional game code mixes state, rules, and UI into monolithic logic. A Petri net separates concerns:

- **Structure** = What's possible (places, transitions, arcs)
- **State** = What's current (token marking)
- **Rules** = What's allowed (guards, roles)
- **History** = What happened (events)

In this model the separation buys an invariant for the turn order and the phase sequence, a replayable log in which every entry was an enabled firing, and a structure in which defects such as the `end_hand` refill can be found by reading the arcs. Adding rules such as side pots or all-in needs the chip arithmetic in the net or in guards first; the model does not yet show that this can be done without the net growing out of proportion.

Texas Hold'em is at the complex end of the spectrum covered in Part II. It reuses the phase cursor, turn tokens and event log from the previous chapters in a single model, with the same vocabulary that described the coffee shop, the tic-tac-toe board, the Sudoku grid, the knapsack and the enzyme reaction. Where it stops — chips, bet sizing, the winner — is where the earlier chapters' resource and guard patterns would have to be added.

Part III of this book shifts focus from building models to analyzing and verifying them — process mining, zero-knowledge proofs, topology-driven verification and on-chain verification.
