# Appendix A: Solver Parameter Reference

This appendix documents the ODE solver methods, configuration options, and presets available in go-pflow's `solver` package. Every value below is read from the package source ([`solver/`](https://github.com/pflow-xyz/go-pflow/tree/main/solver)); the tests named alongside are the checks the repository runs, and none were re-run for this appendix.

All the methods integrate the same right-hand side, built in [`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go): a transition's flux is its rate times the product of its input places' token counts, each to the first power, and is zero if any input place is at or below zero. Arc weights multiply what each unit of flux consumes and produces; they do not enter the flux itself.

## Solver Methods

### Tsit5 (Default)

Tsitouras 5th-order Runge-Kutta with embedded 4th-order error estimator. `Solve` and `SolveUntilEquilibrium` use it when passed a `nil` method.

- **Order**: 5 (local truncation error $\mathcal{O}(h^6)$)
- **Stages**: 7
- **Adaptive**: Yes (embedded error estimation)
- **Reference**: Ch. Tsitouras, "Runge-Kutta pairs of order 5(4) satisfying only the first column simplifying assumption", *Computers & Mathematics with Applications*, 62 (2011) 770-775

```go
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
```

[`solver/tsit5_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/tsit5_test.go) pins the error estimator: the error weights sum to zero, the embedded weights satisfy the 4th-order quadrature conditions and not the 5th, and on $u' = -0.7u$ the accepted-step count grows by less than 3× per decade of `Reltol` (a 5th-order controller gives about $10^{1/5} \approx 1.6\times$).

### RK45 (Dormand-Prince)

Classic Dormand-Prince 5(4) method, the algorithm behind MATLAB's `ode45`. Same order and stage count as Tsit5, with a different tableau and error weights.

- **Order**: 5
- **Stages**: 7
- **Adaptive**: Yes
- **Reference**: J.R. Dormand & P.J. Prince, "A family of embedded Runge-Kutta formulae", *Journal of Computational and Applied Mathematics*, 6 (1980) 19-26

```go
sol := solver.Solve(prob, solver.RK45(), solver.DefaultOptions())
```

### BS32 (Bogacki-Shampine)

3rd-order method with embedded 2nd-order error estimator, for problems with less stringent accuracy requirements where a 5th-order method does more work per step than it needs to.

- **Order**: 3
- **Stages**: 4
- **Adaptive**: Yes
- **Reference**: P. Bogacki & L.F. Shampine, "A 3(2) pair of Runge-Kutta formulas", *Appl. Math. Lett.*, 2 (1989) 321-325

`TestSolverMethodsProduceSimilarResults` in [`solver/solver_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/solver_test.go) runs Tsit5, RK45 and BS32 on a one-transition decay and requires their final values to agree within 5%.

### Fixed-Step Methods: RK4, Heun, Midpoint, Euler

These four have no embedded error estimator: their error weights are all zero ([`solver/methods.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/methods.go)). The error estimate is therefore always zero, so even with `Adaptive: true` every step is accepted and `Dt` never changes. Pass `Adaptive: false` to make that explicit.

| Method | Order | Stages | Notes |
|--------|-------|--------|-------|
| `RK4()` | 4 | 4 | Classic Runge-Kutta; teaching, debugging, or when a fixed step size is wanted |
| `Heun()` | 2 | 2 | Improved Euler: an Euler predictor followed by a trapezoidal corrector |
| `Midpoint()` | 2 | 2 | Evaluates the derivative at the half step |
| `Euler()` | 1 | 1 | Forward Euler; needs very small step sizes for accuracy |

### Implicit Euler

Backward Euler, for stiff systems where explicit methods need impractically small steps. `ImplicitEuler` is a function that returns a solution, not a method passed to `Solve`; with `nil` options it uses `StiffOptions()`.

```go
sol := solver.ImplicitEuler(prob, solver.StiffOptions())
```

It steps at the fixed size `Dt` and ignores `Dtmin`, `Dtmax`, `Reltol` and `Adaptive` ([`solver/implicit.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/implicit.go)). The implicit equation $u_{n+1} = u_n + \Delta t\, f(u_{n+1})$ is solved by fixed-point iteration, not Newton's method: at most 50 iterations, stopping when successive iterates differ by less than $10 \times$ `Abstol`, with no signal if the limit is hit first. That limits what the method buys on a stiff problem. For a linear decay $u' = \lambda u$ the iteration is $u^{k+1} = u_n + \Delta t\,\lambda\, u^k$, which converges only when $\Delta t\,|\lambda| < 1$, so the step must still be small relative to the fastest time scale (derived from the iteration, not re-run). `TestImplicitEuler` checks only that a decay $A' = -0.1A$ from 100 ends within 20% of $100e^{-1}$, and `TestImplicitVsExplicit` that it lands within 20% of Tsit5 on a non-stiff net.

### TRBDF2

Second-order implicit method: a trapezoidal stage to $t + \gamma\Delta t$ with $\gamma = 2 - \sqrt{2}$, then a BDF2 stage to $t + \Delta t$. Implicit Euler is first order. It shares Implicit Euler's fixed step, `StiffOptions()` default and fixed-point iteration, with the same caveat. `TestTRBDF2` checks the same decay to within 10%.

```go
sol := solver.TRBDF2(prob, solver.StiffOptions())
```

### SolveImplicit (Stiffness Heuristic)

Chooses once, at the initial state, between Tsit5 and Implicit Euler.

```go
sol := solver.SolveImplicit(prob, solver.DefaultOptions())
```

The test is a heuristic on derivative magnitudes, not an eigenvalue estimate: it evaluates $du/dt$ at $t_0$ and calls the problem stiff when the largest nonzero $|du/dt|$ across places exceeds the smallest by more than a factor of 1000. If so it runs `ImplicitEuler` with the given options and `Adaptive: false` (so at fixed step `Dt`, 0.01 with `DefaultOptions()`); otherwise it runs `Solve` with Tsit5. It does not switch methods mid-run, and it never selects TRBDF2.

## Configuration Parameters

The `Options` struct controls solver behavior:

| Parameter | Type | Description |
|-----------|------|-------------|
| `Dt` | `float64` | Initial time step (the fixed step for non-adaptive and implicit methods) |
| `Dtmin` | `float64` | Minimum allowed time step |
| `Dtmax` | `float64` | Maximum allowed time step |
| `Abstol` | `float64` | Absolute error tolerance |
| `Reltol` | `float64` | Relative error tolerance |
| `Maxiters` | `int` | Maximum number of accepted steps; rejected steps do not count |
| `Adaptive` | `bool` | Enable adaptive step size control |

If `Maxiters` runs out before `Tspan[1]`, the solve stops early and sets `Solution.Truncated`; the final state is then at the last time reached, not at the end of the span. `TestSolveReportsTruncation` in [`solver/truncation_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/truncation_test.go) covers it.

### How Adaptive Stepping Works

When `Adaptive` is true, `Solve` does the following at each step ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)):

1. Evaluates the stages and forms the higher-order solution `u_new`.
2. Estimates each place's local error as `dt * Σ Bhat[j] * k[j]` over the stages `k`, where the method's `Bhat` field stores the difference between the higher- and lower-order weights, so the lower-order solution is never formed separately.
3. Scales it and takes the maximum over places: `err = max |error| / (Abstol + Reltol * max(|u|, |u_new|))`.
4. Accepts the step if `err <= 1`, and also accepts it regardless of error once `Dt` is already at `Dtmin`.
5. Computes `factor = 0.9 * (1/err)^(1/(order+1))`, where `order` is the method's `Order` (5 for Tsit5 and RK45, 3 for BS32). After an accepted step, `Dt` is multiplied by `factor` capped at 5, then clamped to `[Dtmin, Dtmax]`. After a rejected step, it is multiplied by `factor` floored at 0.1, and never goes below `Dtmin`.

Because of step 4, a problem that drives `Dt` down to `Dtmin` keeps advancing with steps that exceed the tolerance, and nothing reports it. A run that is slow and spends many steps at `Dtmin` is the usual sign of stiffness. The browser solver in pflow-xyz ([`public/petri-solver.js`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/petri-solver.js)) uses the same scaled error and the same cap of 5 on step growth.

## Option Presets

### General Purpose

| Preset | Dt | Dtmin | Dtmax | Abstol | Reltol | Maxiters | Use Case |
|--------|-----|-------|-------|--------|--------|----------|----------|
| `DefaultOptions()` | 0.01 | 1e-6 | 0.1 | 1e-6 | 1e-3 | 100,000 | Most problems |
| `JSParityOptions()` | 0.01 | 1e-6 | 1.0 | 1e-6 | 1e-3 | 100,000 | Mirror of `jsParityOptions()` in pflow-xyz's `petri-learn.js` |
| `FastOptions()` | 0.1 | 1e-4 | 1.0 | 1e-2 | 1e-2 | 1,000 | Interactive use, looser tolerances |
| `AccurateOptions()` | 0.001 | 1e-8 | 0.1 | 1e-9 | 1e-6 | 1,000,000 | Research, publishing |
| `StiffOptions()` | 0.001 | 1e-10 | 0.01 | 1e-8 | 1e-5 | 500,000 | Stiff systems |

All presets set `Adaptive: true`. `TestOptionsPresets` checks the `Dt`, `Adaptive` flag and `Maxiters` range of these five.

`JSParityOptions()` differs from `DefaultOptions()` only in `Dtmax`. The pflow.xyz editor's own simulation (`_odeSolve` in `public/petri-view.js`) passes only `dt`, `abstol`, `reltol` and `adaptive` to the browser solver, whose defaults are `DefaultOptions()`'s values, so the editor's plots run at `Dtmax` 0.1. To reproduce an editor plot in Go, use `DefaultOptions()`; `JSParityOptions()` matches the options `petri-learn.js` uses.

`FastOptions()`'s doc comment estimates a tenfold speedup. That figure is not benchmarked; `TestFastOptionsSpeed` checks only that it takes fewer steps than `AccurateOptions()` on an SIR net and stays within 10 tokens of conservation.

### Domain-Specific

| Preset | Dt | Dtmin | Dtmax | Abstol | Reltol | Maxiters | Use Case |
|--------|-----|-------|-------|--------|--------|----------|----------|
| `GameAIOptions()` | 0.1 | 1e-3 | 1.0 | 1e-2 | 1e-2 | 500 | Move evaluation |
| `EpidemicOptions()` | 0.01 | 1e-6 | 0.5 | 1e-6 | 1e-4 | 200,000 | SIR/SEIR models |
| `WorkflowOptions()` | 0.1 | 1e-4 | 10.0 | 1e-4 | 1e-3 | 50,000 | Process simulation |
| `LongRunOptions()` | 0.1 | 1e-4 | 10.0 | 1e-5 | 1e-3 | 500,000 | Extended simulations |

The use cases in both tables are the doc comments' intent, not measured comparisons.

## Equilibrium Detection

`SolveUntilEquilibrium` integrates like `Solve` and stops early when the system reaches steady state ([`solver/equilibrium.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/equilibrium.go)). It returns the solution and an `EquilibriumResult` whose `Reason` is `equilibrium_reached`, `time_exhausted` or `max_iterations`.

### Equilibrium Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `Tolerance` | `float64` | Largest $\lvert du/dt \rvert$ over all places, at the start of the step, still counted as at rest |
| `ConsecutiveSteps` | `int` | Number of consecutive checks below tolerance required |
| `MinTime` | `float64` | Time after $t_0$ before checking starts |
| `CheckInterval` | `int` | Check every N accepted steps (0 or 1 = every step) |

`ConsecutiveSteps` counts checks, not steps: with the defaults, the derivative must stay below $10^{-6}$ on five checks ten accepted steps apart.

### Equilibrium Presets

| Preset | Tolerance | Consecutive | MinTime | Interval |
|--------|-----------|-------------|---------|----------|
| `DefaultEquilibriumOptions()` | 1e-6 | 5 | 0.1 | 10 |
| `FastEquilibriumOptions()` | 1e-4 | 3 | 0.01 | 5 |
| `StrictEquilibriumOptions()` | 1e-9 | 10 | 1.0 | 1 |

### Convenience Functions

```go
// Find equilibrium with default settings
state, reached := solver.FindEquilibrium(prob)

// Fast equilibrium (aggressive settings)
state, reached := solver.FindEquilibriumFast(prob)

// Accurate equilibrium (strict settings)
state, reached := solver.FindEquilibriumAccurate(prob)
```

`FindEquilibrium` uses Tsit5 with `DefaultOptions()` and `DefaultEquilibriumOptions()`; the Fast variant pairs `FastOptions()` with `FastEquilibriumOptions()`, and the Accurate variant `AccurateOptions()` with `StrictEquilibriumOptions()`. When `reached` is false, `state` is the state at the end of the run. `TestFindEquilibrium` runs a single transition from `A` to `B` and checks that `A` is nearly empty and `A + B` is conserved.

### Combined Option Pairs

`OptionPair` bundles solver and equilibrium options:

```go
pair := solver.GameAIOptionPair()
sol, eq := solver.SolveUntilEquilibrium(prob, solver.Tsit5(), pair.Solver, pair.Equilibrium)
```

| Pair | Solver options | Tolerance | Consecutive | MinTime | Interval |
|------|----------------|-----------|-------------|---------|----------|
| `GameAIOptionPair()` | `GameAIOptions()` | 1e-3 | 2 | 0.01 | 3 |
| `EpidemicOptionPair()` | `EpidemicOptions()` | 1e-6 | 5 | 0.1 | 10 (default) |
| `WorkflowOptionPair()` | `WorkflowOptions()` | 1e-4 | 3 | 0.5 | 5 |
| `LongRunOptionPair()` | `LongRunOptions()` | 1e-9 | 10 | 1.0 | 1 (strict) |

## When to Use Which Solver

| Situation | Solver | Options |
|-----------|--------|---------|
| First attempt / don't know | `Tsit5()` | `DefaultOptions()` |
| Reproduce a pflow.xyz editor plot | `Tsit5()` | `DefaultOptions()` |
| Match `petri-learn.js` | `Tsit5()` | `JSParityOptions()` |
| Game AI move evaluation | `Tsit5()` | `GameAIOptions()` |
| Interactive / real-time | `Tsit5()` | `FastOptions()` |
| Publishing results | `Tsit5()` | `AccurateOptions()` |
| SIR/SEIR epidemics | `Tsit5()` | `EpidemicOptions()` |
| Process mining validation | `Tsit5()` | `WorkflowOptions()` |
| Steps pinned at `Dtmin` (stiff) | `ImplicitEuler()` or `TRBDF2()` | `StiffOptions()` |
| Let the library guess stiffness | `SolveImplicit()` | `DefaultOptions()` |
| Fixed step (teaching) | `RK4()` | `Adaptive: false` |

If `Tsit5()` with `DefaultOptions()` doesn't work for your problem, the usual adjustments are:

1. **Too slow**: Use `FastOptions()` (trades accuracy for speed)
2. **Inaccurate**: Use `AccurateOptions()` (tighter tolerances)
3. **Unstable/NaN**: Use `StiffOptions()`, with `Tsit5()` or with an implicit method; the implicit methods help only as far as their fixed-point iteration converges at the chosen `Dt`
4. **Stops short or never settles**: Check `sol.Truncated` and raise `Maxiters`, extend `Tspan`, or check for conservation violations
