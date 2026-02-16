# Biochemistry — Enzyme Kinetics

**Learning objective**: See how Petri nets naturally model chemical reactions, recovering classical equations.

The previous chapters applied Petri nets to domains where the mathematical formalism is borrowed — resource flows, games, puzzles, optimization. This chapter returns to the domain where the formalism originated: chemistry. Enzyme kinetics is the study of how enzymes catalyze reactions, and the Michaelis-Menten equation that describes it is one of biochemistry's most important results.

Textbooks derive Michaelis-Menten with steady-state assumptions and algebraic manipulation. We don't need any of that. A 4-place Petri net with mass-action kinetics produces the same equation automatically. The structure produces the behavior.

## Substrate, Enzyme, Product

An enzyme catalyzes the conversion of substrate to product in three steps:

$$E + S \rightleftharpoons ES \rightarrow E + P$$

1. **Binding**: enzyme and substrate combine to form a complex (rate $k_1$)
2. **Unbinding**: the complex falls apart without producing product (rate $k_{-1}$)
3. **Catalysis**: the complex converts substrate to product, releasing the enzyme (rate $k_{cat}$)

The enzyme is not consumed — it cycles between free and bound states. This cycling is the hallmark of catalysis, and it maps perfectly to Petri net token flow.

## The Petri Net

Four places, three transitions:

| Place | Initial | Role |
|-------|---------|------|
| `substrate` | 100 | Free substrate molecules |
| `enzyme` | 10 | Free enzyme molecules |
| `complex` | 0 | Enzyme-substrate complex |
| `product` | 0 | Converted product |

| Transition | Rate | Arcs |
|------------|------|------|
| `bind` | $k_1 = 0.01$ | substrate + enzyme → complex |
| `unbind` | $k_{-1} = 0.1$ | complex → substrate + enzyme |
| `catalyze` | $k_{cat} = 0.5$ | complex → product + enzyme |

The arc structure encodes the chemistry directly:

```
substrate --> bind --> complex --> catalyze --> product
enzyme -->          /              enzyme <--+
             unbind
complex --> unbind --> substrate
                  --> enzyme
```

The enzyme appears as both input and output of `catalyze` — it's consumed when the complex forms and returned when the product is released. This cyclic flow is the structural signature of catalysis.

### In Code

```go
net, rates := petri.Build().
    Place("substrate", 100).
    Place("enzyme", 10).
    Place("complex", 0).
    Place("product", 0).
    Transition("bind").
    Transition("unbind").
    Transition("catalyze").
    Arc("substrate", "bind", 1).
    Arc("enzyme", "bind", 1).
    Arc("bind", "complex", 1).
    Arc("complex", "unbind", 1).
    Arc("unbind", "substrate", 1).
    Arc("unbind", "enzyme", 1).
    Arc("complex", "catalyze", 1).
    Arc("catalyze", "product", 1).
    Arc("catalyze", "enzyme", 1).
    WithRates(map[string]float64{
        "bind":     0.01,
        "unbind":   0.1,
        "catalyze": 0.5,
    })
```

The net is complete. Four places, three transitions, nine arcs. Every biochemistry textbook's enzyme kinetics chapter is encoded in this structure.

## Mass-Action Kinetics as Native Semantics

With mass-action kinetics, each transition fires at a rate proportional to the product of its input concentrations:

$$v(\text{bind}) = k_1 \cdot [S] \cdot [E]$$
$$v(\text{unbind}) = k_{-1} \cdot [ES]$$
$$v(\text{catalyze}) = k_{cat} \cdot [ES]$$

The ODE system follows directly from the Petri net topology. Using the incidence matrix $N$ and rate vector $v(M)$:

$$\frac{d[S]}{dt} = -k_1 \cdot [S] \cdot [E] + k_{-1} \cdot [ES]$$

$$\frac{d[E]}{dt} = -k_1 \cdot [S] \cdot [E] + k_{-1} \cdot [ES] + k_{cat} \cdot [ES]$$

$$\frac{d[ES]}{dt} = k_1 \cdot [S] \cdot [E] - k_{-1} \cdot [ES] - k_{cat} \cdot [ES]$$

$$\frac{d[P]}{dt} = k_{cat} \cdot [ES]$$

We didn't write these equations. They emerged from the net structure and mass-action kinetics — exactly the process described in Chapter 3. The incidence matrix encodes all the stoichiometry. The mass-action rate law provides the dynamics. The ODE solver does the rest.

## Emergent Michaelis-Menten Behavior

At steady state ($d[ES]/dt \approx 0$), the complex concentration stabilizes:

$$k_1 \cdot [S] \cdot [E] = (k_{-1} + k_{cat}) \cdot [ES]$$

Solving for the reaction rate $v = k_{cat} \cdot [ES]$ and using $[E]_{total} = [E] + [ES]$:

$$v = \frac{V_{max} \cdot [S]}{K_m + [S]}$$

where $K_m = (k_{-1} + k_{cat}) / k_1$ and $V_{max} = k_{cat} \cdot [E]_{total}$.

This is the **Michaelis-Menten equation**. We didn't derive it — the ODE solver produces the saturation curve directly from the Petri net. Running the simulation with increasing initial substrate concentrations traces out the classic hyperbolic curve: rate increases linearly at low [S], then levels off as enzyme becomes saturated.

### The Saturation Effect

Why does the rate saturate? The Petri net makes it visible through token flow.

At low substrate, most enzyme tokens are in the `enzyme` place (free). Increasing [S] means more `bind` events, so the rate climbs linearly. But at high substrate, almost all enzyme tokens are tied up in `complex`. Adding more substrate can't speed things up — there's no free enzyme to bind with.

**$K_m$** is the substrate concentration at half-maximum rate. It measures binding affinity. With the default parameters: $K_m = (0.1 + 0.5) / 0.01 = 60$. You need 60 substrate molecules to reach half the maximum rate.

**$V_{max}$** is the theoretical maximum rate when all enzyme is saturated. With 10 enzyme molecules and $k_{cat} = 0.5$: $V_{max} = 0.5 \times 10 = 5.0$.

Both constants emerge from the simulation without being computed algebraically.

### Simulation Predictions

The ODE solver computes several predictions from the initial conditions and rate constants:

| Prediction | Default Value | What It Means |
|------------|---------------|---------------|
| Time to 50% conversion | ~17s | When half the substrate is consumed |
| Peak reaction rate | ~2.95 | Maximum d[P]/dt, approaches Vmax |
| Final yield | ~70% | Percentage of substrate converted |
| $K_m$ / $V_{max}$ | 60 / 5.0 | The Michaelis-Menten constants |

The concentration curves show the classic pattern: substrate depletes as product accumulates, with the enzyme-substrate complex rising quickly then falling as substrate runs out.

## Conservation Laws from P-Invariants

The Petri net enforces conservation laws automatically through its structure:

**Enzyme conservation:**

$$[E] + [ES] = [E]_{total} = 10$$

Free enzyme plus bound enzyme always equals the initial enzyme count. This isn't coded as a constraint — it's a structural property of the net. The enzyme token circulates through `enzyme → complex → enzyme` but never leaves the system.

Check via P-invariant: the weight vector $w = [0, 1, 1, 0]$ (for places [substrate, enzyme, complex, product]) satisfies $w^T \cdot N = \vec{0}$:

- `bind` column: $0(-1) + 1(-1) + 1(+1) + 0(0) = 0$
- `unbind` column: $0(+1) + 1(+1) + 1(-1) + 0(0) = 0$
- `catalyze` column: $0(0) + 1(+1) + 1(-1) + 0(+1) = 0$

**Mass conservation:**

$$[S] + [ES] + [P] = [S]_{initial} = 100$$

Substrate is either free, bound in complex, or converted to product. Mass is conserved because every arc that removes a token from one place adds it to another.

These invariants are verified at every timestep of the ODE simulation. If they ever fail, something is wrong with the model — a missing arc, an incorrect weight, a bug in the solver. Conservation laws are free structural correctness checks.

## Parameter Exploration and Sensitivity

The model invites exploration by changing rate constants:

**High enzyme, low substrate** ($E_0=50, S_0=20$): The reaction completes almost instantly. All substrate is converted because there's plenty of enzyme. The rate curve is a sharp spike.

**Low $k_1$** ($k_1=0.001$): Binding becomes the bottleneck. $K_m$ shoots up to 600, meaning enormous substrate concentrations are needed for half-max rate. The enzyme is effectively less efficient.

**High $k_{cat}$** ($k_{cat}=2.0$): The enzyme turns over faster. $V_{max}$ increases to 20, but $K_m$ also increases to 210. Faster catalysis means the complex is consumed faster, so more substrate is needed to keep it populated.

**Equal $k_{-1}$ and $k_{cat}$** ($k_{-1}=0.5, k_{cat}=0.5$): Half the binding events are "wasted" on unbinding. $K_m = (0.5 + 0.5)/0.01 = 100$ — higher than default. The enzyme is less efficient.

Each parameter change is a different set of rate constants on the same net. The structure stays the same. The dynamics change. This is the pattern throughout the book: one model, many scenarios.

## From Chemistry to Everything

The Michaelis-Menten pattern appears far beyond biochemistry. Anywhere a limited resource cycles between free and occupied states, the same saturation dynamics emerge:

- **CPU scheduling**: processes (substrate) compete for cores (enzyme), creating context-switch overhead (complex formation)
- **Customer service**: customers (substrate) wait for agents (enzyme), with a queue (complex) that saturates
- **Manufacturing**: raw materials (substrate) need machines (enzyme), with work-in-progress (complex) limited by machine count

In each case, the Petri net structure is identical — only the labels change. Four places, three transitions, nine arcs. The same $V_{max}$ and $K_m$ characterize the system's capacity and responsiveness.

This is the ComputationNet pattern from Chapter 4 in its purest form: continuous quantities, rate-based firing, conservation laws, and emergent behavior from structure. The enzyme kinetics model is the smallest non-trivial example — small enough to understand completely, yet producing one of the most important results in biochemistry.
