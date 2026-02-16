# Appendix A: Solver Parameter Reference

This appendix documents the ODE solver methods, configuration options, and presets available in go-pflow's `solver` package.

## Solver Methods

### Tsit5 (Default)

Tsitouras 5th-order Runge-Kutta with embedded 4th-order error estimator. The recommended solver for nearly all Petri net simulations.

- **Order**: 5 (error ~ $\mathcal{O}(h^6)$)
- **Stages**: 7
- **Adaptive**: Yes (embedded error estimation)
- **Reference**: Ch. Tsitouras, "Runge-Kutta pairs of order 5(4) satisfying only the first column simplifying assumption", *Computers & Mathematics with Applications*, 62 (2011) 770-775

```go
sol := solver.Solve(prob, solver.Tsit5(), solver.DefaultOptions())
```

### RK45 (Dormand-Prince)

Classic Dormand-Prince 5(4) method. The algorithm behind MATLAB's `ode45`. Comparable to Tsit5 with slightly different error estimation characteristics.

- **Order**: 5
- **Stages**: 7
- **Adaptive**: Yes
- **Reference**: J.R. Dormand & P.J. Prince, "A family of embedded Runge-Kutta formulae", *Journal of Computational and Applied Mathematics*, 6 (1980) 19-26

```go
sol := solver.Solve(prob, solver.RK45(), solver.DefaultOptions())
```

### BS32 (Bogacki-Shampine)

3rd-order method with embedded 2nd-order error estimator. Useful when Tsit5/RK45 are overkill — simpler problems with less stringent accuracy requirements.

- **Order**: 3
- **Stages**: 4
- **Adaptive**: Yes
- **Reference**: P. Bogacki & L.F. Shampine, "A 3(2) pair of Runge-Kutta formulas", *Appl. Math. Lett.*, 2 (1989) 321-325

### RK4 (Classic Runge-Kutta)

Fixed-step 4th-order method. No error estimation. Use with `Adaptive: false`.

- **Order**: 4
- **Stages**: 4
- **Adaptive**: No (fixed step)
- **Use case**: Teaching, debugging, when fixed step size is desired

### Heun (Improved Euler)

2nd-order predictor-corrector method. Simple and fast.

- **Order**: 2
- **Stages**: 2
- **Adaptive**: No

### Euler (Forward Euler)

1st-order method. Primarily for teaching and debugging.

- **Order**: 1
- **Stages**: 1
- **Adaptive**: No
- **Warning**: Requires very small step sizes for accuracy

### Implicit Euler

For stiff systems where explicit methods require impractically small steps. Uses Newton iteration to solve the implicit step.

```go
sol := solver.ImplicitEuler(prob, solver.StiffOptions())
```

### TRBDF2

Second-order implicit method combining trapezoidal rule with backward differentiation. More accurate than implicit Euler for stiff systems.

```go
sol := solver.TRBDF2(prob, solver.StiffOptions())
```

### SolveImplicit (Auto-Detection)

Automatically detects stiffness and switches between explicit (Tsit5) and implicit (TRBDF2) methods.

```go
sol := solver.SolveImplicit(prob, solver.DefaultOptions())
```

## Configuration Parameters

The `Options` struct controls solver behavior:

| Parameter | Type | Description |
|-----------|------|-------------|
| `Dt` | `float64` | Initial time step |
| `Dtmin` | `float64` | Minimum allowed time step |
| `Dtmax` | `float64` | Maximum allowed time step |
| `Abstol` | `float64` | Absolute error tolerance |
| `Reltol` | `float64` | Relative error tolerance |
| `Maxiters` | `int` | Maximum number of iterations |
| `Adaptive` | `bool` | Enable adaptive step size control |

### How Adaptive Stepping Works

When `Adaptive` is true, the solver:

1. Computes both the 5th-order and 4th-order solutions at each step
2. Estimates the local error as the difference between them
3. Computes a scaled error: `err = |error| / (Abstol + Reltol * max(|y|, |y_new|))`
4. If `err <= 1`: accepts the step and grows `Dt` (up to `Dtmax`)
5. If `err > 1`: rejects the step and shrinks `Dt` (down to `Dtmin`)

The step size factor is: `factor = 0.9 * (1/err)^(1/(order+1))`

## Option Presets

### General Purpose

| Preset | Dt | Dtmin | Dtmax | Abstol | Reltol | Maxiters | Use Case |
|--------|-----|-------|-------|--------|--------|----------|----------|
| `DefaultOptions()` | 0.01 | 1e-6 | 0.1 | 1e-6 | 1e-3 | 100,000 | Most problems |
| `JSParityOptions()` | 0.01 | 1e-6 | 1.0 | 1e-6 | 1e-3 | 100,000 | Match JavaScript solver |
| `FastOptions()` | 0.1 | 1e-4 | 1.0 | 1e-2 | 1e-2 | 1,000 | Interactive, ~10x speedup |
| `AccurateOptions()` | 0.001 | 1e-8 | 0.1 | 1e-9 | 1e-6 | 1,000,000 | Research, publishing |
| `StiffOptions()` | 0.001 | 1e-10 | 0.01 | 1e-8 | 1e-5 | 500,000 | Stiff systems |

### Domain-Specific

| Preset | Dt | Dtmin | Dtmax | Abstol | Reltol | Maxiters | Use Case |
|--------|-----|-------|-------|--------|--------|----------|----------|
| `GameAIOptions()` | 0.1 | 1e-3 | 1.0 | 1e-2 | 1e-2 | 500 | Move evaluation |
| `EpidemicOptions()` | 0.01 | 1e-6 | 0.5 | 1e-6 | 1e-4 | 200,000 | SIR/SEIR models |
| `WorkflowOptions()` | 0.1 | 1e-4 | 10.0 | 1e-4 | 1e-3 | 50,000 | Process simulation |
| `LongRunOptions()` | 0.1 | 1e-4 | 10.0 | 1e-5 | 1e-3 | 500,000 | Extended simulations |

## Equilibrium Detection

The equilibrium detector stops the simulation early when the system reaches steady state.

### Equilibrium Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `Tolerance` | `float64` | Maximum rate of change to consider as equilibrium |
| `ConsecutiveSteps` | `int` | Number of consecutive steps below tolerance required |
| `MinTime` | `float64` | Minimum time before checking for equilibrium |
| `CheckInterval` | `int` | Check every N steps (0 = every step) |

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

### Combined Option Pairs

For convenience, `OptionPair` bundles solver and equilibrium options:

```go
pair := solver.GameAIOptionPair()
sol, eq := solver.SolveUntilEquilibrium(prob, solver.Tsit5(), pair.Solver, pair.Equilibrium)
```

Available pairs: `GameAIOptionPair()`, `EpidemicOptionPair()`, `WorkflowOptionPair()`, `LongRunOptionPair()`.

## When to Use Which Solver

| Situation | Solver | Options |
|-----------|--------|---------|
| First attempt / don't know | `Tsit5()` | `DefaultOptions()` |
| Match browser output | `Tsit5()` | `JSParityOptions()` |
| Game AI move evaluation | `Tsit5()` | `GameAIOptions()` |
| Interactive / real-time | `Tsit5()` | `FastOptions()` |
| Publishing results | `Tsit5()` | `AccurateOptions()` |
| SIR/SEIR epidemics | `Tsit5()` | `EpidemicOptions()` |
| Process mining validation | `Tsit5()` | `WorkflowOptions()` |
| System won't converge | `ImplicitEuler()` | `StiffOptions()` |
| Auto-detect stiffness | `SolveImplicit()` | `DefaultOptions()` |
| Fixed step (teaching) | `RK4()` | `Adaptive: false` |

If `Tsit5()` with `DefaultOptions()` doesn't work for your problem, the most common fixes are:

1. **Too slow**: Use `FastOptions()` (trades accuracy for speed)
2. **Inaccurate**: Use `AccurateOptions()` (tighter tolerances)
3. **Unstable/NaN**: Use `StiffOptions()` with `ImplicitEuler()` (stiff system)
4. **Doesn't converge**: Extend `Tspan`, increase `Maxiters`, or check for conservation violations
