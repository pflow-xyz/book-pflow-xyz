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
| item1 | 4 | 12 | 3.0 |
| item2 | 6 | 12 | 2.0 |
| item3 | 9 | 16 | 1.78 |

The optimal solution takes items 0, 1, and 3 — total weight 15, total value 38. Item2 is excluded despite reasonable efficiency because its weight (6) prevents the higher-value item3 (weight 9) from fitting alongside items 0 and 1.

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

## Item Efficiency Encoded in Transition Rates

In the basic model, all transitions have rate 1.0. The mass-action dynamics determine how items are consumed:

$$v(\text{take\_item}_i) = k_i \cdot M(\text{item}_i) \cdot M(\text{capacity})^{w_i}$$

With uniform rate constants ($k_i = 1$ for all items), the transition rate depends on the arc weight — items with larger weights have their rates modulated by higher powers of the remaining capacity. As capacity depletes, heavier items' rates drop faster.

This creates a natural priority: lighter items are consumed first because they have lower-order dependence on capacity. Heavy items are left for later, and if capacity runs out, they're stranded.

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

The continuous relaxation takes *fractional* amounts of each item — all at approximately 71.4%. This is the nature of ODE simulation: it finds a smooth approximation rather than discrete 0/1 choices. The total value is 35.71, compared to the discrete optimum of 38.

The fractional solution is the LP (linear programming) relaxation of the knapsack. It's always an upper bound on the discrete optimum — you can't do better by restricting to integers. The gap between 35.71 and 38 tells us that simple rounding won't find the optimum. We need more information about the solution structure.

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

This means item2 is actively hurting the solution. It's not just "not included in the optimum" — it's competing for capacity that other items would use more effectively. By consuming 6 units of capacity at an efficiency of only 2.0, item2 crowds out items 0 and 3 which have better value-for-weight ratios.

The exclusion analysis tells us exactly which item to remove from the model. No combinatorial search needed — one sensitivity pass reveals the suboptimal item.

### Convergence After Exclusion

With item2's transition disabled, the ODE converges toward the discrete optimum:

| Time | Value | Gap to Optimal |
|------|-------|----------------|
| t=10 | 37.75 | 0.25 |
| t=100 | 37.97 | 0.03 |
| t=1000 | 38.00 | 0.00 |

Given enough simulation time, the continuous relaxation without the suboptimal item converges to the exact discrete optimum of 38. The remaining items (0, 1, 3) are fully compatible — their combined weight equals the capacity exactly — so the ODE reaches the integer solution.

This is the power of combining structural analysis (exclusion) with continuous relaxation (ODE). The exclusion step eliminates competition from suboptimal items. The ODE step then finds the remaining optimum naturally.

## Comparison with Branch-and-Bound

The two approaches solve the same problem through fundamentally different mechanisms.

**Branch-and-bound** treats optimization as *search*. It builds a decision tree where each node represents a binary choice: take this item or skip it. The algorithm explores branches, computing upper bounds to prune paths that can't improve on the best solution found so far. It's systematic enumeration with smart pruning — still exponential in the worst case, but practical for moderate problem sizes.

**DDM/ODE** treats optimization as *simulation*. Instead of explicit decisions, items compete for capacity through continuous dynamics. The mass-action kinetics $v = k \cdot [\text{item}] \cdot [\text{capacity}]$ means all enabled transitions fire simultaneously at rates proportional to available resources. There's no decision tree — just differential equations evolving toward equilibrium.

| Aspect | Branch-and-Bound | DDM/ODE |
|--------|------------------|---------|
| Core operation | Binary search tree | Continuous dynamics |
| Decisions | Explicit (take/skip) | Emergent (competition) |
| Solution type | Exact integer | Fractional approximation |
| Complexity | Exponential (pruned) | Polynomial (ODE integration) |
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

- **100 items**: The ODE simulation remains polynomial — the number of transitions scales linearly. Exclusion analysis requires 100 ODE runs (one per item). Branch-and-bound may require exponentially many branches.

- **Correlated items**: When items have similar efficiency ratios, the exclusion analysis shows which ones truly contribute and which are interchangeable. This information is invisible to pure optimization.

- **Multiple constraints**: Adding a second constraint (volume, fragility) means adding a second capacity place with its own arc weights. The Petri net handles this naturally — each constraint is an independent place. Branch-and-bound becomes the multidimensional knapsack problem, which is significantly harder.

## The Pattern

The knapsack model demonstrates a fourth application pattern, distinct from resources (Chapter 5), games (Chapter 6), and constraints (Chapter 7):

1. **Decision variables** become places (items available or taken)
2. **Resource constraints** become shared capacity places with weighted arcs
3. **Competition** emerges from mass-action kinetics — items fight for capacity
4. **Exclusion analysis** reveals which elements actively hurt the objective
5. **Continuous relaxation** approximates the optimum; structural insight guides to the exact solution

This is optimization by simulation rather than search. The ODE doesn't find the optimal solution directly — it reveals the solution's structure. That structure, in turn, makes finding the optimum straightforward.

> **Try it live:** Explore the [Knapsack optimizer](https://pilot.pflow.xyz/knapsack/) at pilot.pflow.xyz to see mass-action kinetics reveal optimal item selection.
