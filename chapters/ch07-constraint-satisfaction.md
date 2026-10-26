# Constraint Satisfaction — Sudoku

**Learning objective**: Express a puzzle's constraints as net structure, and test whether a continuous relaxation of that net can rank candidate moves.

Chapters 5 and 6 applied Petri nets to resource management and game mechanics. This chapter applies them to a different problem class: **constraint satisfaction**. Sudoku is a puzzle where you fill a grid such that every row, column, and block contains each digit exactly once; there is no optimization and no opponent, only the constraints.

The encoding follows the three-layer pattern of [Chapter 6: Game Mechanics — Tic-Tac-Toe](ch06-game-mechanics.md): cells as places, digit placements as transitions that leave a history token, and constraint collectors that feed a `solved` place. The method carried over from tic-tac-toe is to rank candidate moves by how much ODE flow each directs toward `solved`. The model is go-pflow's Sudoku example: the generator is `createODENet` in [`examples/sudoku/cmd/model.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/cmd/model.go), and the nets it produced are committed beside it as [`sudoku-4x4-ode.jsonld`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/sudoku-4x4-ode.jsonld) and [`sudoku-9x9-ode.jsonld`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/sudoku-9x9-ode.jsonld). Read against that code, the collectors as built can never fire, so the ranking is not yet demonstrated. This chapter describes what the code builds, shows why the ranking fails, and ends on what a collector would have to express for it to work.

## The Puzzle

We'll use a 4×4 mini-Sudoku to keep things comprehensible. It is the sample puzzle in `getSamplePuzzle` in [`examples/sudoku/cmd/main.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/cmd/main.go):

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

Node names in this chapter are the ones in the committed `sudoku-4x4-ode.jsonld` (`P01`, `D2_01`, `_D2_01`, `Block00_Complete`). The current generator writes them with a separator between row and column (`P0_1`, `D2_0_1`, `_D2_0_1`, `Block0_0_Complete`); the structure is the same.

### Layer 1: Cell Places

Each cell in the grid is a place. A token means "this cell is empty":

```
P00: 0  P01: 1  P02: 1  P03: 1
P10: 1  P11: 1  P12: 0  P13: 1
P20: 1  P21: 0  P22: 1  P23: 1
P30: 1  P31: 1  P32: 1  P33: 0
```

Given cells (1 at position 00, 2 at position 12, 3 at position 21, 4 at position 33) have no token — they're already filled. Instead, the history place for the clue starts marked: `_D1_00` holds a token from the beginning because the puzzle gives a 1 at (0,0), while `_D2_00`, `_D3_00` and `_D4_00` start at 0.

### Layer 2: Digit Transitions and History

For each empty cell × digit combination, a transition represents "write this digit in this cell":

```
Transition D2_01:             History place _D2_01:
"Write digit 2 at (0,1)"     "Digit 2 is at (0,1)"

      +------+
P01 --| D2   |---> _D2_01
(1)   | _01  |     (0 -> 1)
      +------+
```

Firing consumes the cell token (cell is now filled) and produces a history token (recording which digit was placed). This is the tic-tac-toe pattern: `P_ij → PlayX_ij → X_ij`.

The generator makes digit transitions only for empty cells, so the 4×4 puzzle has 12 empty cells × 4 digits = **48 digit transitions**, while history places exist for every cell, 16 × 4 = **64 history places**. A digit transition has exactly one input arc, from its cell, and one output arc, to its history place. Nothing from the cell's row, column or block touches it, so the net does not stop a second 1 being written into row 0: `D1_01` is exactly as enabled as `D2_01`. Whatever constraint the model carries has to live in the collectors.

### Layer 3: Constraint Collectors

There are 12 **constraint collectors** — 4 rows (`Row0_Complete` …), 4 columns (`Col0_Complete` …) and 4 blocks (`Block00_Complete` …) — each with one output arc into `solved`, whose capacity is 12. The intent is that `Row0_Complete` fires when row 0 contains 1, 2, 3 and 4. The generator wires it to every history place in the row, 4 digits × 4 cells = 16 input arcs:

```
_D1_00  _D2_00  _D3_00  _D4_00  -->
_D1_01  _D2_01  _D3_01  _D4_01  -->  Row0_Complete --> solved
_D1_02  _D2_02  _D3_02  _D4_02  -->
_D1_03  _D2_03  _D3_03  _D4_03  -->
```

A transition is enabled only when every input place holds a token, so this collector asks for every cell of the row to hold every digit at once. A cell has one token to give, so on any legal fill exactly 4 of the 16 places are marked, and the collector stays disabled. As built, `solved` cannot leave 0.

The counts check against the committed net: 81 places (16 cells, 64 history places, `solved`), 60 transitions (48 digit moves, 12 collectors) and 300 arcs (2 per digit transition, 16 into and 1 out of each collector).

The tic-tac-toe collectors of Chapter 6 don't have this problem because a win line is a plain conjunction — X at three named cells, three input arcs. "Row 0 is complete" is a conjunction over digits of a *disjunction* over cells: 1 is in some cell of the row, and 2 is, and so on. A single transition's enabling rule is a conjunction over its inputs, and wiring every (digit, cell) pair as an input is how the generator ended up asking for all 16.

## How the Model Works

Step-by-step for cell (0,1) — currently empty:

1. **Initial**: `P01 = 1` (cell is empty, available for placement)

2. **Consider digit 2**: Transition `D2_01` is enabled because `P01` has a token — and so are `D1_01`, `D3_01` and `D4_01`, by the same arc

3. **Fire `D2_01`**:
   - `P01`: 1 → 0 (cell now filled)
   - `_D2_01`: 0 → 1 (history records "digit 2 at position (0,1)")

4. **Constraint progress**: `Row0_Complete` is still disabled. It also needs `_D1_01`, `_D3_01` and `_D4_01`, which can no longer be marked because `P01`'s token is spent, and `_D2_00`, `_D3_00` and `_D4_00`, which no transition writes, because (0,0) is a given

The net encodes what a digit placement is. It does not yet encode what a valid solution is.

## ODE Prediction for Move Ordering

The tic-tac-toe method scores each candidate by simulating the net from the marking the move would leave. As a procedure (a sketch; nothing in the repository implements it for Sudoku):

```
For each candidate move:
  1. Create hypothetical state (after placing this digit)
  2. Run ODE simulation (t=0 to t=3.0)
  3. Measure token flow to 'solved' place
  4. Higher flow = move more likely leads to solution
```

### What the Net Computes

On the committed net this procedure gives every candidate the same score, zero. go-pflow's rate law sets a transition's flux to its rate times the product of its input places' token counts, and to 0 outright if any input is at or below 0 ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)). Every row, column and block of the sample puzzle contains one given cell. A given cell has no digit transitions, so three of its four history places are 0 for all $t$, and every collector over that row, column or block has those places among its inputs. Every collector's flux is therefore 0 from the first step, whichever move was tried.

The history layer cannot separate the candidates either. The four digit transitions out of an empty cell share one input and one rate (`SetRates` gives every transition 1.0 unless overridden, [`petri/net.go`](https://github.com/pflow-xyz/go-pflow/blob/main/petri/net.go)), so the ODE splits the cell's token evenly among its four history places. With the collectors inert, nothing connects one cell to another, and placing a 2 or a 4 at (0,1) leaves the rest of the trajectory unchanged.

We derived this by reading the generator and the solver against the committed net; the simulation was not re-run.

### Move Ranking Example

For cell (0,1), reasoning from the puzzle rules out two digits and leaves two:

| Digit | The rules | The net | Score on the committed net |
|-------|-----------|---------|----------------------------|
| 1 | excluded: row 0 has the 1 at (0,0) | `D1_01` enabled | 0 |
| 2 | candidate (the solution's digit) | `D2_01` enabled | 0 |
| 3 | excluded: column 1 has the 3 at (2,1) | `D3_01` enabled | 0 |
| 4 | candidate | `D4_01` enabled | 0 |

The scores are derived from the argument above, not from a run. The exclusions in the second column are the reader's reasoning; neither shows up in the net, where all four transitions are enabled by the same token. For the method to work, the ODE would have to score 2 above 4 and give 1 and 3 zero flow. No run in the repository demonstrates either.

## Tuning for Speed

A full-precision solve of the 9×9 net is slow, so the benchmark in [`sudoku_bench_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/cmd/sudoku_bench_test.go) runs the Tsit5 solver with looser settings than `solver.DefaultOptions()`:

| Parameter | `DefaultOptions()` | Benchmark | Why |
|-----------|--------------------|-----------|-----|
| Time horizon | passed with the problem (tic-tac-toe used t=10.0) | t=3.0 | Only relative rankings are needed |
| Abstol | 1e-6 | 1e-4 | Looser tolerance, faster solve |
| Reltol | 1e-3 | 1e-3 | Unchanged |
| Initial step (dt) | 0.01 | 0.2 | Larger first step; the solver is adaptive and `Dtmax` stays 0.1 |
| Max iterations | 100000 | 100000 | Left at the default |

Cutting the horizon assumes that ranking needs candidates in the right order rather than converged token counts, and that a move ahead at t=3.0 is still ahead at t=10.0. Nothing in the repository tests that assumption for Sudoku, and with every score at 0 there is nothing yet to order. The same cut was made first in the tic-tac-toe example, whose [README](https://github.com/pflow-xyz/go-pflow/blob/main/examples/tictactoe/README.md) reports a 21× speedup and about 0.04 s per move evaluation.

[`benchmark_results.txt`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/sudoku/benchmark_results.txt) records one run on an Apple M4 Max:

| Benchmark | Time per evaluation |
|-----------|---------------------|
| 4×4, benchmark settings | 0.067 s |
| 4×4, tighter tolerance | 14.8 s |
| 9×9, benchmark settings | 4.58 s |

That file names benchmarks (`…SingleEvaluation`, `…WithTighterTolerance`) that the current test file no longer has; it now has `BenchmarkSudoku4x4ODE`, `BenchmarkSudoku9x9ODE` and `BenchmarkSudoku16x16ODE`. The settings behind the 14.8 s row are therefore not in the repository, and the roughly 220× gap between the two 4×4 rows is against settings we cannot name. At the recorded speeds, scoring the four candidates for a 4×4 cell takes about a quarter of a second, and the nine for a 9×9 cell about forty seconds. From the go-pflow root, `go test -run '^$' -bench Sudoku4x4ODE ./examples/sudoku/cmd` re-times the 4×4 solve; it discards the result, so nothing in the repository yet reads `solved` from a solution or ranks a move. (The analyzer's `--analyze` flag does not call the solver either: `runODESimulationOnNet` in `main.go` prints a description of the method and returns.) Not re-run for this chapter.

## Scaling from 4×4 to 9×9

The same generator builds the 9×9 net:

| Component | 4×4 | 9×9 |
|-----------|-----|-----|
| Cell places | 16 | 81 |
| Digits | 4 | 9 |
| History places | 64 | 729 |
| Digit transitions (sample puzzle) | 48 | 459 |
| Digit transitions (empty grid) | 64 | 729 |
| Row / column / block collectors | 4 / 4 / 4 | 9 / 9 / 9 |
| Inputs per collector | 16 | 81 |
| **`solved` capacity** | **12** | **27** |

The 9×9 sample is the 30-clue puzzle in `getSamplePuzzle`, so 51 empty cells and 459 digit transitions; the committed `sudoku-9x9-ode.jsonld` has 486 transitions, which is 459 plus the 27 collectors. Every row, column and block of that puzzle also contains a given, so the 9×9 collectors are starved the same way the 4×4 ones are.

The intended use was for the ranking to order the choices of a discrete solver (backtracking, constraint propagation) rather than to solve the puzzle itself. No such solver is wired to it in the repository.

## Constraint Satisfaction as Token Flow

The Sudoku model suggests a layering for constraint satisfaction problems:

1. **Variables** become places (cells that can be filled)
2. **Domain values** become transitions (digits that can be placed), each leaving a history token
3. **Constraints** become collector transitions (patterns that must be completed)
4. **Solution** is a state where all collectors have fired

Steps 1, 2 and 4 carry over from tic-tac-toe directly. Step 3 is where the Sudoku net goes wrong. A collector's enabling rule is a conjunction over its inputs, so a constraint with a disjunction inside it — "some cell of row 0 holds a 1" — needs structure in between. One shape would give each (digit, row) pair its own place, fed from that row's four history places, and give the row collector one input per digit. It has not been built, and it raises a question of its own: the column and block collectors need the same history tokens, so the new layer would have to test them without consuming them — with read arcs, which fall outside the free structure ([Appendix E: Categorical Foundations](appendix-e-categorical-foundations.md), "Where the Free Structure Stops") — or put them back. Even with working collectors, the digit layer has no arcs that make a conflicting placement cost anything, so whether the ODE would separate 2 from 4 at (0,1), or show 1 and 3 as zero flow, is open.

The same layering can be sketched for other problems. An N-queens net would have N² square places, queen-placement transitions, and collectors for rows, columns and diagonals; a graph-coloring net would have vertex places, color transitions, and a collector for each edge. Neither is built in the repository, and neither has been checked against the collector problem above.

In constraint programming, the order in which variables are assigned and values tried has a large effect on search time. Flow toward `solved` was meant as a value-ordering heuristic of that kind: try the assignment that sends the most flow toward the solution. On the net as built, every assignment sends none, so whether it can serve as one depends on a collector design that has not been tried.
