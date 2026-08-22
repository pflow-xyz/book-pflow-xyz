# Constraint Satisfaction — Sudoku

**Learning objective**: Express constraints as token conservation and solve via continuous relaxation.

Chapters 5 and 6 applied Petri nets to resource management and game mechanics. This chapter applies them to a different problem class: **constraint satisfaction**. Sudoku is a puzzle where you fill a grid such that every row, column, and block contains each digit exactly once. There's no optimization, no adversary, no time — just constraints.

The Petri net encoding follows the same three-layer pattern from tic-tac-toe: cells as places, digit placements as transitions, and constraint collectors that fire when a row, column, or block is complete. The ODE simulation ranks candidate moves by how much flow they direct toward the `solved` place — no backtracking search required.

## The Puzzle

We'll use a 4×4 mini-Sudoku to keep things comprehensible:

```
  Initial:            Solution:
+---+---+---+---+    +---+---+---+---+
| 1 | . | . | . |    | 1 | 2 | 4 | 3 |
+---+---+---+---+    +---+---+---+---+
| . | . | 2 | . |    | 3 | 4 | 2 | 1 |
+===+===+===+===+ -> +===+===+===+===+
| . | 3 | . | . |    | 2 | 3 | 1 | 4 |
+---+---+---+---+    +---+---+---+---+
| . | . | . | 4 |    | 4 | 1 | 3 | 2 |
+---+---+---+---+    +---+---+---+---+
```

Constraints: each row, column, and 2×2 block must contain digits 1–4 exactly once. The 9×9 version uses digits 1–9 with 3×3 blocks — same structure, just more of it.

## Cell Places and Digit Transitions

### Layer 1: Cell Places

Each cell in the grid is a place. A token means "this cell is empty":

```
P00: 0  P01: 1  P02: 1  P03: 1
P10: 1  P11: 1  P12: 0  P13: 1
P20: 1  P21: 0  P22: 1  P23: 1
P30: 1  P31: 1  P32: 1  P33: 0
```

Given cells (1 at position 00, 2 at position 12, 3 at position 21, 4 at position 33) have no token — they're already filled.

### Layer 2: Digit Transitions and History

For each cell × digit combination, a transition represents "write this digit in this cell":

```
Transition D2_01:             History place _D2_01:
"Write digit 2 at (0,1)"     "Digit 2 is at (0,1)"

      +------+
P01 --| D2   |---> _D2_01
(1)   | _01  |     (0 -> 1)
      +------+
```

Firing consumes the cell token (cell is now filled) and produces a history token (recording which digit was placed). This is identical to the tic-tac-toe pattern: `P_ij → PlayX_ij → X_ij`.

For a 4×4 grid: 16 cells × 4 digits = **64 digit transitions** and **64 history places**. Most of these transitions will be irrelevant for any given puzzle (you can't place a 1 in a row that already has a 1), but the ODE simulation handles this naturally — conflicting placements produce zero flow.

### Layer 3: Constraint Collectors

When all required digits appear in a row, column, or block, a **constraint collector** transition fires and deposits a token in the `solved` place:

```
_D1_00 --> Row0_Complete --> solved
_D2_01 -->
_D3_02 -->
_D4_03 -->
```

The `Row0_Complete` transition requires history tokens for all four digits in row 0. When all four are present — meaning row 0 has digits 1, 2, 3, and 4 — the collector fires.

For the 4×4 grid: 4 row collectors + 4 column collectors + 4 block collectors = **12 constraint collectors**. Maximum tokens in `solved` = 12. A fully solved puzzle has all 12 constraints satisfied.

This is a direct generalization of the tic-tac-toe pattern collectors. In tic-tac-toe, collectors watched for three-in-a-row. In Sudoku, collectors watch for complete rows, columns, and blocks. Same mechanism, different patterns.

## How the Model Works

Step-by-step for cell (0,1) — currently empty:

1. **Initial**: `P01 = 1` (cell is empty, available for placement)

2. **Consider digit 2**: Transition `D2_01` is enabled because `P01` has a token

3. **Fire `D2_01`**:
   - `P01`: 1 → 0 (cell now filled)
   - `_D2_01`: 0 → 1 (history records "digit 2 at position (0,1)")

4. **Constraint progress**: When row 0 eventually has all four digits placed, `Row0_Complete` fires and `solved` gets +1 token

The model doesn't encode *which* digit goes where — that's the puzzle's job. It encodes *what constitutes a valid solution*: all constraint collectors firing. The ODE simulation explores the space of possible placements and measures how much flow reaches `solved`.

## ODE Prediction for Move Ordering

The same hypothesis evaluation from tic-tac-toe applies:

```
For each candidate move:
  1. Create hypothetical state (after placing this digit)
  2. Run ODE simulation (t=0 to t=3.0)
  3. Measure token flow to 'solved' place
  4. Higher flow = move more likely leads to solution
```

### Why This Works

The constraint collectors create asymmetric flow in the ODE. A digit placement that satisfies constraints in multiple directions (its row, column, and block) creates more flow toward `solved` than one that only helps one constraint. The mass-action kinetics amplify this: more enabled collectors means higher aggregate flow rate.

A placement that creates a conflict — placing a 2 in a row that already has a 2 — produces zero flow through the corresponding row collector. The ODE naturally penalizes conflicts without any explicit "conflict detection" logic.

### Move Ranking Example

For cell (0,1) in the 4×4 puzzle, evaluating each digit:

```
Digit 2: solved flow = 0.31  <-- Correct answer
Digit 3: solved flow = 0.28
Digit 4: solved flow = 0.15
Digit 1: blocked (conflict -- row 0 already has 1)
```

The ODE ranks digit 2 highest because it enables the most downstream constraint satisfaction. Digit 1 is blocked because its row collector can never fire (the history token for "1 in row 0" already exists from the given cell).

## Tuning for Speed

A naive ODE simulation of the full model is slow. But we don't need precision — we need **ranking**. Two moves scored at 0.31 and 0.28 will maintain their relative order whether we simulate to t=3.0 or t=10.0.

This enables aggressive tuning:

| Parameter | Default | Tuned | Why |
|-----------|---------|-------|-----|
| Time horizon | t=10.0 | t=3.0 | Only need relative rankings |
| Abstol | 1e-6 | 1e-4 | Looser tolerance, faster solve |
| Step size (dt) | 0.01 | 0.2 | Larger steps, fewer iterations |
| Max iterations | 100000 | 1000 | Early termination OK |

The performance impact:

| Model | Naive | Tuned | Speedup |
|-------|-------|-------|---------|
| 4×4 (64 transitions) | 0.8s | 0.04s | 20× |
| 9×9 (729 transitions) | ~60s | ~3s | 20× |

This makes interactive analysis feasible — evaluating all candidates for a cell takes seconds, not minutes.

## Scaling from 4×4 to 9×9

The same pattern scales without structural changes:

| Component | 4×4 | 9×9 |
|-----------|-----|-----|
| Cell places | 16 | 81 |
| Digits | 4 | 9 |
| History places | 64 | 729 |
| Digit transitions | 64 | 729 |
| Row collectors | 4 | 9 |
| Column collectors | 4 | 9 |
| Block collectors | 4 | 9 |
| **Max solved tokens** | **12** | **27** |

The 9×9 model has 729 transitions — one for each cell × digit combination. The constraint collectors expand from 12 to 27 (9 rows + 9 columns + 9 blocks). The structure is identical; only the dimensions change.

The ODE simulation time grows with the number of transitions (729 vs. 64), but the tuned solver still handles it in seconds. The ODE is not solving Sudoku here; it is ranking candidate moves, and the ranking guides a discrete solver (backtracking, constraint propagation) toward the solution faster.

## Constraint Satisfaction as Token Flow

The Sudoku model reveals a general pattern for constraint satisfaction problems:

1. **Variables** become places (cells that can be filled)
2. **Domain values** become transitions (digits that can be placed)
3. **Constraints** become collector transitions (patterns that must be completed)
4. **Solution** is a state where all collectors have fired

This pattern works for any CSP. An N-queens problem would have N² cell places, N queen transitions per column, and collectors for row uniqueness, column uniqueness, and diagonal uniqueness. A graph coloring problem would have vertex places, color transitions, and collectors for each edge (ensuring adjacent vertices have different colors).

The ODE simulation doesn't solve the CSP directly — it provides a heuristic for variable and value ordering. In constraint programming, the order in which you assign variables and try values dramatically affects performance. The ODE's flow-based ranking is a natural heuristic: try the assignment that creates the most flow toward the solution.

No hardcoded solver. Constraints emerge from topology. The same four-term primitive — cell, func, arrow, guard — encodes Sudoku, N-queens, graph coloring, and any other CSP. The structure changes; the mechanism doesn't.
