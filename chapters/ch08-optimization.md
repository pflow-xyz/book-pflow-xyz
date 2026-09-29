# Optimization — The Knapsack Problem

**Learning objective**: Solve NP-hard problems approximately via mass-action kinetics.

The previous chapter used ODE simulation as a heuristic for constraint satisfaction — ranking moves, not solving directly. This chapter goes further: applying continuous relaxation to a combinatorial optimization problem where the ODE provides direct insight into the solution structure.

The 0/1 knapsack problem asks: given items with weights and values, select a subset that maximizes total value within a weight budget. It's NP-hard. Traditional approaches use branch-and-bound — systematic enumeration with pruning. The Petri net approach uses mass-action kinetics — items compete for capacity through continuous dynamics. The competition reveals which items matter and which actively hurt the solution.

## The 0/1 Knapsack as a Petri Net

### Problem Definition

Four items with different weights and values, capacity budget of 15:

| Item | Weight | Value | Efficiency (v/w) |
|------|--------|-------|-------------------|
| item0 | 2 | 10 | 5.0 |
| item1 | 4 | 10 | 2.5 |
| item2 | 6 | 12 | 2.0 |
| item3 | 9 | 18 | 2.0 |

These are the values in go-pflow's [`examples/knapsack`](https://github.com/pflow-xyz/go-pflow/blob/main/examples/knapsack/cmd/main.go), which produced every simulation number in this chapter.

The optimal solution takes items 0, 1, and 3 — total weight 15, total value 38. Item2 and item3 tie on efficiency at 2.0, so the ratio can't choose between them; capacity does. After items 0 and 1 take 6 units, item3 fills the remaining 9 exactly, while item2 would leave 3 idle (items 0, 1, 2 weigh 12 and are worth 32).

### Net Structure

The Petri net encodes the knapsack as a dynamic system:

- **Item places**: Each holds 1 token (item available) or 0 (item taken). This enforces the 0/1 constraint — you either take an item or you don't.
- **Capacity place**: Holds tokens representing available weight budget. Initial marking = 15 (the weight limit).
- **Take transitions**: One per item. Consumes the item token plus capacity tokens equal to the item's weight. Produces tokens into value and weight tracking places.

For item0 (weight 2, value 10):

```
item0 --> take_item0 --> value_taken
           ^                (weight 10)
capacity --+
 (weight 2)
```

The arc from `capacity` to `take_item0` has weight 2 — taking item0 costs 2 units of capacity. The enabling condition requires $M(\text{item0}) \geq 1$ AND $M(\text{capacity}) \geq 2$. If the remaining capacity is less than 2, this item can't be taken.

The critical structural property: all items compete for the same capacity pool. Taking item2 (weight 6) leaves less capacity for item3 (weight 9). This competition is encoded in the shared input place, not in any explicit exclusion logic.

## Why Every Item Is Taken Equally

In the basic model, all transitions have rate 1.0. go-pflow's mass-action kinetics are first-order in every input place ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)):

$$v(\text{take\_item}_i) = k_i \cdot M(\text{item}_i) \cdot M(\text{capacity})$$

The arc weight $w_i$ scales how much capacity each firing consumes, not the rate: it does *not* appear as an exponent. (Chemical mass action would raise $M(\text{capacity})$ to the power $w_i$; go-pflow does not.)

This makes the dynamics solvable by hand. Let $\tau(t) = \int_0^t M(\text{capacity})\,ds$. Each item place obeys $\dot M(\text{item}_i) = -M(\text{item}_i)\,M(\text{capacity})$, so

$$M(\text{item}_i) = e^{-\tau}$$

— the same curve for every item, whatever its weight or value. Values never enter the dynamics at all; they sit on output arcs into a place nothing reads. All active items are therefore taken in one common fraction $f = 1 - e^{-\tau}$, and the remaining capacity is $15 - W f$, where $W$ is the total weight of the active items. If $W > 15$, capacity runs out at $f = 15/W$ and the net settles at value $15V/W$, with $V$ the total value of the active items. If $W \le 15$, $f \to 1$ and the net eventually collects all of $V$. Every number in the rest of this chapter follows from that formula.

## Continuous Relaxation and Rounding

Running the ODE simulation with uniform rates:

```
Final state (continuous approximation):
  Value accumulated:    35.71
  Weight used:          15.00
  Capacity remaining:    0.00

Item consumption (fraction taken):
  item0: 71.4% taken
  item1: 71.4% taken
  item2: 71.4% taken
  item3: 71.4% taken
```

The continuous relaxation takes *fractional* amounts of each item — all at exactly $15/21 = 71.4\%$, as the formula predicts ($W = 21$, $V = 50$, value $15 \cdot 50/21 = 35.71$). The total value is 35.71, compared to the discrete optimum of 38.

The fractional solution looks like the LP (linear programming) relaxation of the knapsack, but it is not one: an LP relaxation is an upper bound on the integer optimum (here the LP value is 38 — items 0 and 1 whole, then 9 units at ratio 2.0), while the ODE settles at 35.71, *below* the integer optimum of 38. Mass-action kinetics spreads consumption across items proportionally; it is a smooth dynamics, not a maximiser. The gap between 35.71 and 38 tells us that simple rounding won't find the optimum. We need more information about the solution structure.

## Exclusion Analysis for Sensitivity

This is where the Petri net approach delivers insight that branch-and-bound doesn't. **Exclusion analysis** disables each item's transition one at a time and re-runs the ODE:

| Excluded | Final Value | Relative to Baseline |
|----------|-------------|---------------------|
| none | 35.71 | 100.0% |
| item0 | 31.58 | 88.4% |
| item1 | 35.29 | 98.8% |
| item2 | 37.75 | 105.7% |
| item3 | 32.00 | 89.6% |

Three of the four items behave as expected — excluding them reduces the total value. But item2 is anomalous: **excluding it *increases* the value from 35.71 to 37.75**.

The formula explains the whole table. While the active items overweigh the capacity, the value is $15V/W$ — capacity times the aggregate value density of the active set. Excluding item0 costs the most because it is the densest item: $V/W$ falls from $50/21 = 2.38$ to $40/19 = 2.11$, giving 31.58. Excluding item3 leaves items weighing 12, which fit, so the net collects their full value of 32. Item2 is the only exclusion that raises the value, because without it the remaining weights sum to exactly 15 and the net can take all of items 0, 1 and 3.

So item2 is "hurting" the solution in a precise sense: its exclusion is the one that makes the rest fit. On this instance one sensitivity pass — five ODE runs — points at the optimum. Single exclusions only examine subsets of size $n-1$, though, and on a larger instance the optimum usually drops several items; that this heuristic finds the optimum in general is not shown here.

### Convergence After Exclusion

With item2's transition disabled, the ODE converges toward the discrete optimum:

| Time | Value | Gap to Optimal |
|------|-------|----------------|
| t=10 | 37.75 | 0.25 |
| t=100 | 37.97 | 0.03 |
| t=1000 | 38.00 | 0.00 |

Given enough simulation time, the continuous relaxation without item2 converges to the exact discrete optimum of 38. The remaining items (0, 1, 3) weigh exactly the capacity, so capacity and items drain together: $M(\text{capacity}) = 15e^{-\tau}$ integrates to $e^\tau = 1 + 15t$, and the gap to 38 is $38/(1 + 15t)$ — algebraic rather than exponential, which is why it takes until $t = 1000$ to close.

The exclusion step removes the item whose presence keeps the rest from fitting; the ODE then fills the set that remains. It reaches the integer optimum here because $2 + 4 + 9 = 15$, not because the dynamics select it.

## Comparison with Branch-and-Bound

The two approaches solve the same problem through fundamentally different mechanisms.

**Branch-and-bound** treats optimization as *search*. It builds a decision tree where each node represents a binary choice: take this item or skip it. The algorithm explores branches, computing upper bounds to prune paths that can't improve on the best solution found so far. It's systematic enumeration with smart pruning — still exponential in the worst case, but practical for moderate problem sizes.

**DDM/ODE** treats optimization as *simulation*. Instead of explicit decisions, items compete for capacity through continuous dynamics. The mass-action kinetics $v = k \cdot [\text{item}] \cdot [\text{capacity}]$ means all enabled transitions fire simultaneously at rates proportional to available resources. There's no decision tree — just differential equations evolving toward equilibrium.

| Aspect | Branch-and-Bound | DDM/ODE |
|--------|------------------|---------|
| Core operation | Decision-tree search with bounds | Continuous dynamics |
| Decisions | Explicit (take/skip) | None — every active item is taken in the same fraction |
| Solution type | Exact integer | Fractional approximation |
| Cost | Exponential worst case (pruned) | One ODE solve per run; does not solve the integer problem |
| Primary output | "What's optimal?" | "Why is it optimal?" |

For exact solutions, branch-and-bound wins — it finds items 0, 1, 3 with value 38 directly. The ODE reaches 35.71 by taking fractional amounts of everything.

But the ODE reveals *structure* that search obscures. The exclusion analysis shows item2 is actively counterproductive. Branch-and-bound discovers this implicitly through pruning, but it doesn't report it as an insight. The ODE makes the competition visible.

### When to Use Each

| DDM/ODE Approach | Branch-and-Bound |
|------------------|------------------|
| Exploratory analysis | Exact solutions |
| Sensitivity insights | Guaranteed optimum |
| Fast iteration | Proven correctness |
| Understanding *why* | Finding *what* |

In practice, the approaches complement each other. Use ODE simulation to understand the problem structure — which items matter, which compete, where the bottlenecks are. Then use branch-and-bound (or dynamic programming) to find the exact solution, informed by the structural insights.

## Beyond Four Items

The 4-item example is intentionally small. For larger instances:

- **100 items**: Each ODE run stays cheap — the number of transitions scales linearly — and single-item exclusion analysis is 101 runs. But single exclusions only look at 99-item subsets, and the optimum will typically drop many items, so this does not replace branch-and-bound's exponential worst case; a benchmark on larger instances has not been done.

- **Correlated items**: When items have similar efficiency ratios, the exclusion analysis shows which ones truly contribute and which are interchangeable. This information is invisible to pure optimization.

- **Multiple constraints**: Adding a second constraint (volume, fragility) means adding a second capacity place with its own arc weights. The Petri net handles this naturally — each constraint is an independent place. Branch-and-bound becomes the multidimensional knapsack problem, which is significantly harder.

## The Pattern

The knapsack model demonstrates a fourth application pattern, distinct from resources (Chapter 5), games (Chapter 6), and constraints (Chapter 7):

1. **Decision variables** become places (items available or taken)
2. **Resource constraints** become shared capacity places with weighted arcs
3. **Competition** emerges from mass-action kinetics — items fight for capacity
4. **Exclusion analysis** reveals which elements actively hurt the objective
5. **Continuous relaxation** approximates the optimum; structural insight guides to the exact solution

This is optimization by simulation rather than search. The ODE doesn't find the optimal solution directly — it reveals the solution's structure. On this instance that structure leads straight to the optimum; on larger ones it is a guide for search, not a substitute.

> **Try it live:** Explore the [Knapsack optimizer](https://pilot.pflow.xyz/knapsack/) at pilot.pflow.xyz to see mass-action kinetics reveal optimal item selection.
