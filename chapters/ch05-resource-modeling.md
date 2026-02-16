# Resource Modeling — The Coffee Shop

**Learning objective**: Model inventory, throughput, and bottlenecks as token flow.

The first four chapters built the machinery: bipartite graphs, incidence matrices, ODEs, a DSL. Now we use it. This chapter models a coffee shop — ingredients, recipes, orders, capacity limits — as a Petri net. The model answers operational questions: When do we run out of beans? Which drink creates the worst bottleneck? How should we staff during peak hours?

The coffee shop is a **ResourceNet** in the categorical vocabulary from Chapter 4. Tokens represent fungible quantities — grams of beans, milliliters of milk, cups. Conservation laws guarantee that nothing is created or destroyed without an explicit transition. The continuous relaxation from Chapter 3 turns the model into a capacity planning tool.

## The Problem

A coffee shop has limited inventory. Different drinks consume ingredients at different rates:

| Drink | Beans | Water | Milk | Cups |
|-------|-------|-------|------|------|
| Espresso | 18g | 30ml | — | 1 |
| Americano | 18g | 200ml | — | 1 |
| Latte | 18g | 30ml | 180ml | 1 |
| Cappuccino | 18g | 30ml | 120ml | 1 |
| Mocha | 18g | 30ml | 150ml | 1 |

Starting inventory: 1,000g beans, 5,000ml milk, 10,000ml water, 100 cups.

The questions are straightforward. Which ingredient runs out first? How many drinks can we serve before something depletes? If we increase latte production, what's the knock-on effect on bean consumption?

You could answer these with a spreadsheet. But a spreadsheet gives you static arithmetic — multiply rates by time. A Petri net gives you *dynamics* — rates that change as resources deplete, multiple transitions competing for the same inputs, bottlenecks emerging from structure rather than being hard-coded.

## Defining the Net

The coffee shop net has three categories of places:

**Ingredient places** hold the current stock. Each starts with an initial token count representing the inventory on hand:

```
coffee_beans: 1000    (grams)
milk:         5000    (milliliters)
water:        10000   (milliliters)
cups:         100
syrup:        500     (pumps)
```

**Consumption tracking places** count what's been used. They start at zero and accumulate as drinks are made:

```
beans_used:  0
milk_used:   0
water_used:  0
cups_used:   0
syrup_used:  0
```

**Transitions** represent drink preparation — one per drink type:

```
make_espresso
make_americano
make_latte
make_cappuccino
make_mocha
```

The bipartite structure is clean: ingredient places connect to transitions (input arcs), and transitions connect to tracking places (output arcs). No place connects to another place. No transition connects to another transition. The Petri net enforces this separation by construction.

### In Code

```go
net := petri.NewPetriNet()

// Ingredient places
net.AddPlace("coffee_beans", 1000, nil, 100, 100, nil)
net.AddPlace("milk", 5000, nil, 100, 200, nil)
net.AddPlace("water", 10000, nil, 100, 300, nil)
net.AddPlace("cups", 100, nil, 100, 400, nil)
net.AddPlace("syrup", 500, nil, 100, 600, nil)

// Tracking places
net.AddPlace("beans_used", 0, nil, 300, 100, nil)
net.AddPlace("milk_used", 0, nil, 300, 200, nil)
net.AddPlace("water_used", 0, nil, 300, 300, nil)
net.AddPlace("cups_used", 0, nil, 300, 400, nil)
net.AddPlace("syrup_used", 0, nil, 300, 600, nil)
```

Each place gets an initial token count. The tracking places start at zero — they're sinks that accumulate evidence of consumption. Together with the ingredient places, they form a conservation system: every bean consumed from `coffee_beans` appears in `beans_used`.

## Weighted Arcs as Recipes

This is where the model gets interesting. Each drink requires specific quantities of ingredients. The arc weights encode these recipes directly in the net's structure.

For espresso — 18g beans, 30ml water, 1 cup:

```go
net.AddTransition("make_espresso", "consume", 200, 150, nil)

// Input arcs (consumption)
net.AddArc("coffee_beans", "make_espresso", 18, false)
net.AddArc("water", "make_espresso", 30, false)
net.AddArc("cups", "make_espresso", 1, false)

// Output arcs (tracking)
net.AddArc("make_espresso", "beans_used", 18, false)
net.AddArc("make_espresso", "water_used", 30, false)
net.AddArc("make_espresso", "cups_used", 1, false)
```

For a latte — 18g beans, 30ml water, 180ml milk, 1 cup:

```go
net.AddTransition("make_latte", "consume", 200, 350, nil)

net.AddArc("coffee_beans", "make_latte", 18, false)
net.AddArc("water", "make_latte", 30, false)
net.AddArc("milk", "make_latte", 180, false)
net.AddArc("cups", "make_latte", 1, false)

net.AddArc("make_latte", "beans_used", 18, false)
net.AddArc("make_latte", "water_used", 30, false)
net.AddArc("make_latte", "milk_used", 180, false)
net.AddArc("make_latte", "cups_used", 1, false)
```

The arc weight from `milk` to `make_latte` is 180. This means `make_latte` is enabled only when $M(\text{milk}) \geq 180$. Each firing consumes 180 tokens from milk and produces 180 tokens in milk_used. The recipe *is* the arc structure — no separate configuration needed.

### The Incidence Matrix

The incidence matrix for the full model captures every recipe in a single data structure. Each column is a drink type. Each row is an ingredient (or tracking place). Reading column `make_latte`:

$$N[\text{coffee\\_beans}, \text{make\\_latte}] = -18$$
$$N[\text{milk}, \text{make\\_latte}] = -180$$
$$N[\text{water}, \text{make\\_latte}] = -30$$
$$N[\text{cups}, \text{make\\_latte}] = -1$$

Negative entries are consumption. Positive entries in the tracking rows mirror them:

$$N[\text{beans\\_used}, \text{make\\_latte}] = +18$$
$$N[\text{milk\\_used}, \text{make\\_latte}] = +180$$

### Conservation Laws

The paired structure — every token consumed from an ingredient place appears in the corresponding tracking place — creates P-invariants. For coffee beans:

$$M(\text{coffee\\_beans}) + M(\text{beans\\_used}) = 1000$$

For milk:

$$M(\text{milk}) + M(\text{milk\\_used}) = 5000$$

These hold for all time, under any firing sequence. The conservation law is structural — it follows from the arc weights summing to zero across each ingredient/tracking pair. No beans are created or destroyed. They're consumed from stock into the "used" counter.

This is the Petri net equivalent of double-entry bookkeeping. Every debit (consumption) has a matching credit (tracking). The invariant is the balance equation.

## ODE Simulation for Capacity Planning

With the net defined, the continuous relaxation from Chapter 3 turns it into a dynamical system. Each transition gets a rate constant representing drinks per minute:

```go
rates := map[string]float64{
    "make_espresso":   0.5,  // 30/hr
    "make_americano":  0.3,  // 18/hr
    "make_latte":      0.8,  // 48/hr — most popular
    "make_cappuccino": 0.4,  // 24/hr
    "make_mocha":      0.2,  // 12/hr
}
```

The firing rate for each transition follows mass-action kinetics:

<div>$$v(\text{make\_latte}) = k_{\text{latte}} \cdot M(\text{beans})^{18} \cdot M(\text{water})^{30} \cdot M(\text{milk})^{180} \cdot M(\text{cups})^{1}$$</div>

Wait — that looks wrong. With arc weight 18, the rate would depend on the 18th power of the bean count? That's the literal mass-action formula, but with large token counts (1,000 beans) and large exponents (18), the numbers explode to astronomical values.

### Scaling for Stability

This is a real issue in practice. The mass-action rate law $v(t) = k \cdot \prod M(p_i)^{w_i}$ was designed for chemical reactions where concentrations are small numbers (moles per liter, typically between 0 and 10). Coffee shop inventory has large integer token counts and large arc weights.

The solution is scaling. Divide the rate constants by a factor that brings the dynamics into a numerically stable range:

```go
scaledRates := make(map[string]float64)
for k, v := range rates {
    scaledRates[k] = v * 0.0001
}

prob := solver.NewProblem(net, initialState, [2]float64{0, duration}, scaledRates)
opts := solver.DefaultOptions()
opts.Dt = 0.001

sol, eqResult := solver.SolveUntilEquilibrium(
    prob, solver.Tsit5(), opts, nil,
)
```

The scaling doesn't change the qualitative behavior — ratios between transition rates are preserved, so relative throughput stays the same. It just keeps the numbers in a range where the ODE solver's adaptive stepping works well.

### What the Simulation Shows

Running the simulation from the initial inventory reveals depletion trajectories:

- **Cups** deplete first — only 100 available, consumed one per drink regardless of type
- **Coffee beans** deplete next — every drink uses 18g, so 1,000g supports about 55 drinks total
- **Milk** depletes on a schedule driven by latte popularity — at 180ml per latte and 48 lattes/hour, 5,000ml lasts roughly 58 minutes of pure latte production
- **Water** depletes slowest — 10,000ml with most drinks using only 30ml

The conservation laws hold throughout the simulation. At any point in time:

$$M(\text{coffee\\_beans}) + M(\text{beans\\_used}) = 1000$$

This is verifiable from the simulation output and provides a built-in sanity check.

### Predicting Runout

Rather than running the full ODE, we can estimate depletion times analytically from the rates and arc weights:

```go
consumptionRates := map[string]float64{
    "coffee_beans": 18 * (rates["make_espresso"] +
                          rates["make_americano"] +
                          rates["make_latte"] +
                          rates["make_cappuccino"] +
                          rates["make_mocha"]),
    "milk": 180*rates["make_latte"] +
            120*rates["make_cappuccino"] +
            150*rates["make_mocha"],
    "cups": rates["make_espresso"] +
            rates["make_americano"] +
            rates["make_latte"] +
            rates["make_cappuccino"] +
            rates["make_mocha"],
}

for ingredient, rate := range consumptionRates {
    runoutTime := currentStock[ingredient] / rate
    fmt.Printf("%s depletes in %.0f minutes\n", ingredient, runoutTime)
}
```

The analytical estimate assumes constant rates — it's the linear approximation. The ODE simulation is more accurate because rates drop as ingredients deplete (mass-action slows transitions as inputs approach zero). The two converge when resources are abundant and diverge near depletion.

## Bottleneck Analysis from Equilibrium

The real power of the continuous model appears at equilibrium. When the ODE reaches steady state ($dM/dt = 0$), the marking tells you where tokens accumulate — and that's where your bottlenecks are.

In the coffee shop, equilibrium means all ingredients have depleted and all tracking places have filled. But the *path* to equilibrium reveals the bottleneck sequence:

1. **First bottleneck**: Cups run out (100 cups, all drinks need exactly 1)
2. **Second bottleneck**: Beans deplete (all drinks need 18g, no differentiation)
3. **Third bottleneck**: Milk depletes (only milk-based drinks affected)

### Scenario Analysis

Change the rates to model different days:

**Slow day** — halve all rates:
```go
rates["make_latte"] = 0.4  // 24/hr instead of 48
```
Nothing depletes during a normal shift. The equilibrium is far in the future.

**Rush hour** — double all rates:
```go
rates["make_latte"] = 1.6  // 96/hr
```
Cups deplete in half the time. Beans follow quickly. The queue builds up before production can handle it.

**Latte promotion** — triple latte rate, keep others constant:
```go
rates["make_latte"] = 2.4  // 144/hr
```
Milk becomes the first bottleneck. With 5,000ml at 180ml/latte, the milk lasts about 35 minutes at this rate. Beans actually last longer because the increased latte volume comes at the expense of other drinks, which are rate-limited by their own (unchanged) constants.

Each scenario is the same net with different rate constants. The structure — places, arcs, weights — stays the same. Only the dynamics change.

### Sensitivity Analysis

go-pflow includes sensitivity analysis that systematically evaluates which rate changes have the biggest impact:

```go
scorer := sensitivity.FinalStateScorer(func(final map[string]float64) float64 {
    return final["cups_used"]  // maximize drinks served
})

analyzer := sensitivity.NewAnalyzer(net, state, rates, scorer).
    WithTimeSpan(0, 60)

result := analyzer.AnalyzeRatesParallel()
```

The analyzer perturbs each rate individually, simulates the ODE, and ranks transitions by their impact on the scoring function. For the coffee shop, the ranking typically shows:

1. **make_latte** — highest positive impact (most popular drink)
2. **make_espresso** — second highest (fast to make, less resource-intensive)
3. **make_cappuccino** — moderate impact
4. **make_mocha** — lowest (complex recipe, slowest rate)

This ranking tells the shop owner where to invest. Adding a second espresso machine (doubling the espresso rate) has more impact on throughput than hiring a second barista for mocha preparation.

## Restocking and Capacity Limits

A real coffee shop doesn't run until depletion — it restocks. The Petri net models this with **refill transitions** that move tokens from supply places into ingredient places:

```go
// Supply places (incoming inventory)
net.AddPlace("beans_supply", 0, nil, 500, 100, nil)
net.AddPlace("milk_supply", 0, nil, 500, 200, nil)

// Refill transitions
net.AddTransition("refill_beans", "refill", 400, 100, nil)
net.AddArc("beans_supply", "refill_beans", 500, false)
net.AddArc("refill_beans", "coffee_beans", 500, false)

net.AddTransition("refill_milk", "refill", 400, 200, nil)
net.AddArc("milk_supply", "refill_milk", 1000, false)
net.AddArc("refill_milk", "milk", 1000, false)
```

The supply places start at zero — no restocking scheduled. When external events add tokens to `beans_supply` (a delivery arrives), the `refill_beans` transition becomes enabled and transfers 500g into the active inventory.

This creates a two-level conservation structure:

- **Inner loop**: $M(\text{beans}) + M(\text{beans\\_used}) = c_1$ (within one restocking cycle)
- **Outer loop**: Deliveries increase $c_1$ as supply tokens arrive

The inner conservation law tells you whether beans leak. The outer structure tells you whether deliveries keep up with consumption.

### Low-Stock Alerts

The model can trigger alerts based on inventory thresholds:

```go
func CheckLowStock(state map[string]float64) map[string]bool {
    thresholds := map[string]float64{
        "coffee_beans": 100,  // 100g ≈ 5 drinks
        "milk":         500,  // 500ml ≈ 3 lattes
        "cups":         10,
    }

    alerts := make(map[string]bool)
    for ingredient, threshold := range thresholds {
        if state[ingredient] < threshold {
            alerts[ingredient] = true
        }
    }
    return alerts
}
```

In a guard-based formulation (Chapter 4), these thresholds would be guards on the make transitions:

```lisp
(action make_latte :guard {tokens(coffee_beans) >= 118 && tokens(milk) >= 680})
```

The guard checks not just whether there's enough for *this* drink, but whether stock is above the alert threshold plus one recipe's worth. This is precautionary guarding — blocking production before the resource reaches critical levels.

## Full Day Simulation

The most revealing analysis simulates an entire business day with varying demand:

```go
func RunDaySimulation(peakHours []int, baseRate float64) {
    currentState := initialState

    for hour := 6; hour <= 20; hour++ {
        multiplier := 1.0
        for _, peak := range peakHours {
            if hour == peak {
                multiplier = 2.5  // Rush hour
            }
            if hour == peak-1 || hour == peak+1 {
                multiplier = 1.5  // Shoulder hours
            }
        }

        rates := scaleRates(InventoryRates(), baseRate * multiplier)

        prob := solver.NewProblem(net, currentState,
            [2]float64{0, 60}, rates)
        sol := solver.Solve(prob, solver.Tsit5(), solver.FastOptions())
        currentState = sol.GetFinalState()

        stats := computeHourlyStats(currentState)
        fmt.Printf("%02d:00  %4s  drinks: %d  beans: %.0fg  alerts: %v\n",
            hour, rateLabel(multiplier), stats.drinks, stats.beans, stats.alerts)
    }
}
```

With peak hours at 8am and 12pm, the simulation shows:
- Normal flow 6-7am as the shop opens
- A surge at 8am (2.5x rate) that rapidly consumes beans and cups
- Recovery during mid-morning as rates return to normal
- A second surge at noon depleting remaining stock
- Alerts triggering by early afternoon if restocking didn't happen

The solver handles each hour independently — the final state of one hour becomes the initial state of the next. The `FastOptions()` preset keeps each solve under a millisecond, making the full-day simulation essentially instant.

## From Model to Application

The coffee shop model demonstrates a development pattern that recurs throughout this book:

1. **Define the net** — places for resources, transitions for operations, weighted arcs for recipes
2. **Verify structure** — check P-invariants (does the model conserve resources?), check for deadlocks (can the system get stuck?)
3. **Simulate dynamics** — run the ODE to see how resources flow over time
4. **Analyze bottlenecks** — read equilibrium values and sensitivity rankings
5. **Build the application** — use the verified model as the backend for operational decisions

The Petri net isn't a simulation that approximates the real system. It *is* the system specification. The arc weights are the recipes. The conservation laws are the accounting rules. The equilibrium values are the capacity limits. When you change the model, you change the system.

This is the ResourceNet pattern: tokens count fungible things, arc weights specify recipes, conservation laws guarantee integrity, and ODE simulation answers capacity questions. The same pattern appears wherever you count things — inventory management, budget allocation, resource scheduling, supply chain modeling.

The next chapter applies a different pattern — the GameNet — to tic-tac-toe, where tokens encode board positions and turn order, and the hypothesis evaluator finds optimal moves.
