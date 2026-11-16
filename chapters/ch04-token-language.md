# The Token Language

**Learning objective**: Use the pflow DSL to define nets with guards, roles, and typed arcs.

The first three chapters built Petri nets from mathematical primitives — places, transitions, arcs, tokens, rate constants. That's sufficient for analysis but awkward for specification. You wouldn't write a web application by specifying its state machine as a 5-tuple. You'd want a language.

This chapter introduces the token language — a small, interpreted DSL for defining Petri net models with rich data semantics. It extends the mathematical foundation with guards, typed state, invariants, and categorical net types, while keeping the vocabulary to four terms. The implementation is go-pflow's [`tokenmodel/dsl`](https://github.com/pflow-xyz/go-pflow/tree/main/tokenmodel/dsl) package, and [Appendix B: Token Language Grammar](appendix-b-token-grammar.md) is the full grammar reference. The behaviour described below was read from that code and the tests named alongside it; the tests were not re-run for this chapter.

## The Four-Term Primitive

The token language distills to four primitives:

| Term | Maps To | Purpose |
|------|---------|---------|
| `cell` | State / Place | Containers for tokens or data |
| `func` | Action / Transition | Operations that transform state |
| `arrow` | Arc | Flow connections with optional bindings |
| `guard` | Predicate | Boolean conditions gating actions |

Every token model — from an ERC-20 token standard to a poker game to a hospital workflow — is expressed with these four terms. `cell` defines what exists. `func` defines what happens. `arrow` defines what flows where. `guard` defines what's allowed.

The four terms are the book's vocabulary, not the DSL's keywords. In the DSL a cell is written `state`, a func `action` and an arrow `arc`; a guard is attached to an action as `:guard {...}`; and a fifth form, `constraint`, declares an invariant. We make no claim that four is a minimum. They are the four things every model in this book needs, and the rest of the language is annotation on them.

## S-Expression Syntax

The DSL uses an S-expression syntax. This ERC-20 token is the `erc020DSL` schema from go-pflow's [`tokenmodel/dsl/dsl_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/dsl_test.go), copied with only the indentation changed:

```lisp
; ERC-020 Fungible Token Standard
(schema ERC-020
  (version v1.0.0)

  (states
    (state totalSupply :type uint256)
    (state balances :type map[address]uint256 :exported)
    (state allowances :type map[address]map[address]uint256 :exported))

  (actions
    (action transfer :guard {balances[from] >= amount && to != address(0)})
    (action approve)
    (action transferFrom :guard {balances[from] >= amount && allowances[from][caller] >= amount})
    (action mint :guard {to != address(0)})
    (action burn :guard {balances[from] >= amount}))

  (arcs
    (arc balances -> transfer :keys (from))
    (arc transfer -> balances :keys (to))
    (arc approve -> allowances :keys (owner spender))
    (arc balances -> transferFrom :keys (from))
    (arc allowances -> transferFrom :keys (from caller))
    (arc transferFrom -> balances :keys (to))
    (arc mint -> balances :keys (to))
    (arc mint -> totalSupply)
    (arc balances -> burn :keys (from))
    (arc totalSupply -> burn))

  (constraints
    (constraint supply_conservation {sum(balances) == totalSupply})))
```

`TestInterpret_ERC020` parses and interprets this text and checks that the result has 3 states, 5 actions, 10 arcs and 1 constraint; `TestRoundTrip_ParseAndValidate` passes it through schema validation.

Each kind of declaration lives in its own group — `states`, `actions`, `arcs`, `constraints` — and the parser rejects a bare `(state ...)` placed directly under `schema`. Within the groups:

- **`state`** declares a cell with a name and type. The `:exported` keyword marks states that are visible across schema boundaries.
- **`action`** declares a func, optionally with a guard expression in braces.
- **`arc`** declares an arrow; `->` separates source from target. The parser accepts any two names; validation then requires one end to be a state and the other an action. `:keys` names the bindings that select an entry in a map-valued state.
- **`constraint`** declares an invariant, checked after each action the runtime executes (see [Guard vs. Invariant](#guard-vs-invariant)).

The DSL has no variable declarations, control flow or modules, so the structure of the model is visible in the shape of the text.

### In Go

The same model can be constructed with a fluent builder. This is `TestBuilder_ERC020` from [`tokenmodel/dsl/builder_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/builder_test.go):

```go
schema := Build("ERC-020").
    Version("1.0.0").
    Data("totalSupply", "uint256").
    Data("balances", "map[address]uint256").Exported().
    Data("allowances", "map[address]map[address]uint256").Exported().
    Action("transfer").Guard("balances[from] >= amount && to != address(0)").
    Action("approve").
    Action("transferFrom").Guard("balances[from] >= amount && allowances[from][caller] >= amount").
    Action("mint").Guard("to != address(0)").
    Action("burn").Guard("balances[from] >= amount").
    Flow("balances", "transfer").Keys("from").
    Flow("transfer", "balances").Keys("to").
    Flow("approve", "allowances").Keys("owner", "spender").
    Flow("balances", "transferFrom").Keys("from").
    Flow("allowances", "transferFrom").Keys("from", "caller").
    Flow("transferFrom", "balances").Keys("to").
    Flow("mint", "balances").Keys("to").
    Flow("mint", "totalSupply").
    Flow("balances", "burn").Keys("from").
    Flow("totalSupply", "burn").
    Constraint("supply_conservation", "sum(balances) == totalSupply").
    MustSchema()
```

The test checks the same counts — 3 states, 5 actions, 10 arcs, 1 constraint — and that validation passes. Both front ends build the same kind of syntax tree and hand it to the same interpreter, but the trees are not always identical: the builder gives token states the type `int` and defaults the version to `v1.0.0`, where the parser leaves both empty unless the source sets them (Appendix B). The builder's `String()` method prints the S-expression form, so a schema built in code can be read and shared as text.

## Guards and Conditional Firing

In a basic Petri net, a transition fires whenever its input places have enough tokens. Guards add a second condition: even if tokens are available, the transition fires only when its guard expression evaluates to true.

### Expression Syntax

Guards support a familiar expression language:

```
Comparison:    balances[from] >= amount
               to != address(0)

Logical:       condition1 && condition2
               condition1 || condition2
               !condition

Indexing:      balances[key]
               allowances[owner][spender]

Field access:  schedule.revocable
               data.timestamp

Functions:     address(0)
               tokens("count")
               sum("balances_")
```

go-pflow's evaluator ([`tokenmodel/guard/eval.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/guard/eval.go)) short-circuits: `&&` stops on the first false and `||` on the first true, so the right operand is not evaluated when the left one decides the result. `address` is the only function a guard gets by default. The marking aggregates `sum`, `count`, `tokens`, `min` and `max` ([`aggregate.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/guard/aggregate.go)) take a place ID or ID prefix as a string, and are supplied when an expression is evaluated as an invariant; a guard has them only if the caller passes them in.

### Guard vs. Invariant

Both guards and invariants are boolean expressions, but they serve different purposes and fire at different times:

| Aspect | Guard | Invariant |
|--------|-------|-----------|
| When checked | Before action fires | After state update |
| On failure | Action is blocked, nothing changes | Error returned; the update is not undone |
| Purpose | Precondition | System property |
| Scope | Single action | Entire schema |

A guard asks: "Is this action allowed right now?" An invariant asks: "Did this action break anything?"

The "on failure" row is go-pflow's `tokenmodel.Runtime` ([`runtime.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/runtime.go)). It applies the arcs first and checks constraints afterwards, and a violated constraint makes `Execute` return `ErrConstraintViolated` with the new state already in place. A caller that wants all-or-nothing behaviour runs the action on `Runtime.Clone()` and keeps the clone only if no error comes back. Both checks also need an evaluator: `Runtime.GuardEvaluator` is an interface that go-pflow declares but does not implement, and with it left nil the runtime skips guards and constraints alike.

For the ERC-20 transfer, the guard `balances[from] >= amount` prevents overdrafts. The invariant `sum(balances) == totalSupply` states that no action creates or destroys tokens. As written, it is a constraint for sim.pflow.xyz's evaluator, whose `sum` totals the values of a map. go-pflow's runtime evaluates constraints against token counts only, so it cannot check a constraint over the `balances` map (Appendix B compares the two evaluators). In either case a constraint is a runtime assertion about the states a run actually visits; it does not prove the property for every reachable state.

### Evaluation Context

go-pflow's guard evaluator works from a context with two parts ([`guard/eval.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/guard/eval.go)):

1. **Bindings** — every name the expression mentions, mapped to a value
2. **Functions** — `address`, plus whatever the caller supplies

There is no separate state component. When the guard `balances[from] >= amount` runs, `from` and `amount` come from the action's parameters, and `balances` resolves only if the caller has put the current map into the bindings too; a name with no binding is an error ("unknown identifier"). When an expression is checked as an invariant ([`guard/invariant.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/guard/invariant.go)), the bindings are built from the marking, one per place, so `pending + confirmed + shipped == 1` reads token counts by place name. Missing map keys default to zero: indexing a map with a key it lacks returns 0 rather than an error.

## Weighted Arcs and Inhibitor Arcs

Basic arcs have an implicit weight of 1 — one token consumed, one token produced. Weighted arcs generalize this.

### Weighted Arcs

An arc with weight $w$ from a place to a transition means "this transition consumes $w$ tokens from this place when it fires." An arc with weight $w$ from a transition to a place means "this transition produces $w$ tokens in this place when it fires."

The DSL has no syntax for arc weights. Its `:value` keyword names the binding whose value a data arc moves (Appendix B); it is not a weight, and the parser rejects a number in that position. Every arc written in the DSL has weight 1. A weighted arc is written in the JSON model format of [Appendix C](appendix-c-json-schema.md), or in Go. This fragment, a sketch rather than a file in any repository, is a coffee recipe that takes 2 units of water and 1 unit of grounds:

```json
"places": [
  {"id": "water", "initial": 10},
  {"id": "grounds", "initial": 5},
  {"id": "coffee", "initial": 0}
],
"transitions": [{"id": "brew"}],
"arcs": [
  {"from": "water", "to": "brew", "weight": 2},
  {"from": "grounds", "to": "brew", "weight": 1},
  {"from": "brew", "to": "coffee", "weight": 1}
]
```

The `brew` action consumes 2 water and 1 grounds, producing 1 coffee. The enabling condition requires $M(\text{water}) \geq 2$ and $M(\text{grounds}) \geq 1$. From the initial marking, water allows $\lfloor 10/2 \rfloor = 5$ firings and grounds allows 5, so `brew` fires exactly five times and stops at water 0, grounds 0, coffee 5. That follows from the arcs by hand; it was not run. In the continuous simulation of Chapter 3 the weight scales how much water each firing consumes, not the rate law: go-pflow's solver ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)) computes a transition's flux as its rate times the product of its input places' markings, each to the first power, whatever the weights.

### Inhibitor Arcs

An **inhibitor arc** is the opposite of a normal arc: it prevents a transition from firing when the source place *has* tokens, rather than when it doesn't. Drawn as an arc with a small circle instead of an arrowhead, an inhibitor arc from place $p$ to transition $t$ means "$t$ can only fire when $p$ is empty."

Inhibitor arcs add expressiveness beyond standard Petri nets. They can test for zero, which basic Petri nets cannot do; that is enough to simulate a counter machine, so nets with inhibitor arcs are Turing-complete and reachability for them is undecidable. An inhibitor arc moves no tokens, so it has no entry in the incidence matrix: it is one of the contextual arcs of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries), which sit outside the free structure the matrix describes. go-pflow's runtime generalizes the test with the arc's weight $w$: the transition is blocked while the place holds at least $w$ tokens, so weight 1 means "fire only when empty". Like weights, inhibitor arcs have no DSL syntax; in JSON they are `"type": "inhibitor"` on the arc (Appendix C).

Capacity limits are a common use. A buffer with maximum capacity 5 can also be built without one:

```lisp
(schema Buffer
  (states
    (state buffer :kind token :initial 0)
    (state empty_slots :kind token :initial 5))
  (actions
    (action produce)
    (action consume))
  (arcs
    (arc empty_slots -> produce)
    (arc produce -> buffer)
    (arc buffer -> consume)
    (arc consume -> empty_slots)))
```

This uses a complementary place (`empty_slots`) rather than an inhibitor arc, a common pattern that achieves the same effect while staying within the standard Petri net formalism. `produce` moves one token from `empty_slots` to `buffer` and `consume` moves it back, so with weight vector $y = (1, 1)$ over (`buffer`, `empty_slots`) we have $y^\top C = 0$, and the P-invariant $M(\text{buffer}) + M(\text{empty\_slots}) = 5$ holds in every reachable marking. That is derived from the four arcs by hand.

## Tokens vs. Data

The token language distinguishes two kinds of state, reflecting two fundamentally different ways that places hold information.

### TokenState

Integer counters representing discrete quantities. This is classic Petri net semantics:

- Firing consumes and produces tokens
- The enabling condition checks token counts
- Tokens are fungible — we track counts, not identities

A place declared `:kind token` behaves like a place in the mathematical Petri nets of Chapter 2: its marking is an integer, its arcs consume and produce whole tokens, and the incidence-matrix results of that chapter apply to it. In the DSL a state is a data state unless it says `:kind token`; the JSON format of Appendix C has the opposite default.

### DataState

Typed containers holding structured data — maps, records, scalar values. This extends Petri nets to handle real-world state:

- Arc keys select an entry (`balances[from]`)
- Data arcs move a value between entries; the container itself is never consumed
- Types are declared (`map[address]uint256`, `bool`, `string`) but stored as written; neither the parser nor validation checks them

A data state holds a value, not a count. The `balances` place in the ERC-20 schema holds a map from addresses to amounts. When the runtime executes an action with bindings ([`runtime.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/runtime.go), `applyDataArc`), an input data arc subtracts the bound value — `amount`, unless `:value` names another binding — from the entry its `:keys` select, and an output data arc adds it. So `transfer` subtracts `amount` from `balances[from]` and adds it to `balances[to]`. A data input never blocks enablement: whether there is enough to move is the guard's job, which is why `transfer` carries `balances[from] >= amount`.

The combination of TokenState and DataState lets you model systems where both control flow (tokens tracking "where we are") and data transformations (balances, permissions, configuration) matter. A workflow token tracks the order through `pending → confirmed → shipped`. Data states track the order's contents, the customer's address, the payment amount.

## Categorical Net Types

Not all Petri nets are alike. An order-processing workflow and an epidemiological simulation both use places, transitions, and arcs, but their tokens mean fundamentally different things. **Net types** classify Petri nets by how their tokens behave, what invariants hold, and how they compose.

### The Five Types

The five types are declared in go-pflow's [`metamodel/compose.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose.go) as the `NetType` of a subnet in a composed bundle.

| Type | Token Semantics | Intended invariant |
|------|----------------|-----------|
| **WorkflowNet** | Single control-flow token (cursor) | Mutual exclusion — cursor in exactly one state |
| **ResourceNet** | Countable inventory tokens | Conservation — total tokens constant |
| **GameNet** | Turn-based tokens with player roles | Both mutual exclusion and conservation |
| **ComputationNet** | Continuous quantities via ODE rates | Flux balance at steady state |
| **ClassificationNet** | Signal tokens with threshold activation | Threshold firing conditions |

The last column is what each type is for, not what go-pflow checks. Bundle validation checks one of these properties itself: it warns when a WorkflowNet does not start with exactly one token. The rest are the modeller's to declare as constraints and to verify, as the composition example below does for Orders and Inventory.

**WorkflowNet.** A single token traces a path through states. An order moves through `pending → confirmed → shipped → delivered`. The token is a cursor — it marks where you are in a sequential process. With one token, a WorkflowNet is a finite state machine drawn as a net.

**ResourceNet.** Tokens represent countable, fungible things: inventory items, currency, capacity slots. A warehouse has 100 tokens in `available`, consumed by `reserve` and produced into `reserved`. The total is conserved — the P-invariant guarantees that nothing is created or destroyed.

**GameNet.** Tokens encode both game state and turn structure. In tic-tac-toe, empty cells hold tokens consumed when a player moves. A turn-control token alternates between players. GameNets combine WorkflowNet sequencing (turns) with ResourceNet accounting (board positions).

**ComputationNet.** Tokens are continuous, not discrete. Transitions fire at rates — mass-action kinetics from Chapter 3. An SIR epidemic model is a ComputationNet: the infection rate depends on the product of susceptible and infected populations.

**ClassificationNet.** Tokens accumulate as evidence. Transitions fire when enough tokens accrue — a threshold. A spam filter, for instance, could accumulate signal tokens from heuristics (suspicious sender, keyword matches, link density) and fire `classify_spam` when the threshold is met; this book does not build that model.

### Why Types Matter

At the structural level, all five types are Petri nets. The type annotations matter because they constrain composition. When you connect two nets, the types determine what connections are valid:

- An **EventLink** connects transitions to transitions across schemas — when one fires, the other fires too
- A **DataLink** connects places to places — read-only observation across schema boundaries
- A **TokenLink** transfers tokens between schemas — resource coupling with cross-boundary conservation
- A **GuardLink** gates a transition in one schema on a place in another — constraint coupling

TokenLinks and DataLinks fuse places, EventLinks fuse transitions, and a GuardLink is lowered to an inhibitor arc or a guard conjunct. The legality rules are a small table in [`metamodel/compose_matrix.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_matrix.go). A TokenLink may join only nets whose places count fungible things, ResourceNets and GameNets, so a workflow cursor cannot be fused with an inventory counter. An EventLink is refused if either side is a ComputationNet, whose transitions have no discrete firing instant to synchronise on. DataLinks and GuardLinks are legal between any two types. A subnet with no declared type is legal with every link, so the check protects only the nets that declare a type.

### Composition

Real systems are compositions of multiple nets. An order system combines a WorkflowNet (order lifecycle) with a ResourceNet (inventory), linked by EventLinks:

```
Orders (WorkflowNet):    pending -> confirm -> confirmed -> ship -> shipped
Inventory (ResourceNet): available -> reserve -> reserved -> ship_out -> consumed
```

EventLinks:
1. `orders.confirm → inventory.reserve` — confirming reserves stock
2. `orders.ship → inventory.ship_out` — shipping consumes reserved stock

This bundle exists as code: `ordersInventoryBundle` in go-pflow's [`metamodel/compose_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_test.go), with one token in `pending` and 10 in `available`. Each schema declares its own property. The Orders WorkflowNet declares mutual exclusion, `pending + confirmed + shipped == 1`; the Inventory ResourceNet declares conservation, `available + reserved + consumed == 10`.

This is **assume-guarantee reasoning**: each component is verified in isolation, and composition only needs to check the boundaries. `TestComponentInvariantsSurviveComposition` in [`metamodel/compose_verify_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_verify_test.go) flattens the bundle, recomputes the P-invariants from the incidence matrix of the whole, and checks that both component laws are still among them.

The flattened net does not depend on the order in which subnets and links are declared. `TestFlattenAssociativity` checks this on a chain of three subnets, swapping the order of the links and of the subnets and comparing the results.

Composition is not monotonic. Adding an *unlinked* schema is the monoidal product — the two nets share nothing, so each behaves exactly as it did alone. But adding a *link* restricts. An EventLink is a rendezvous: the fused transition fires only when every participant is enabled, so `inventory.reserve` can no longer fire whenever stock exists — it must wait for an order to confirm. A GuardLink gates by construction. A TokenLink introduces a shared place that constrains its consumers. Each of these removes behavior rather than adding it. In the example above the restriction can be counted by hand. Inventory alone can reach every marking whose three counts are non-negative and sum to 10, and there are $\binom{12}{2} = 66$ of them. Composed with Orders, `reserve` fires only with `confirm` and `ship_out` only with `ship`, each at most once, so the composite reaches 3 markings. `TestEventLinkRestrictsBehavior` checks that the composed count is smaller and logs both; the 66 and 3 above are derived, not read from a run.

What survives an EventLink is **refinement**: every firing sequence of the composite, projected onto a component's transitions, is a firing sequence of that component alone. `TestProjectionRefinement` checks the marking form of this on the Orders + Inventory bundle: every reachable composite marking, projected onto Orders, is reachable in Orders alone. Refinement is what assume-guarantee needs, and it is what preserves *safety* — invariants, mutual exclusion, conservation. It does not preserve *liveness*: a component that could always eventually fire may, once linked, wait forever on a partner. Links that fuse places (TokenLinks, DataLinks) give a weaker guarantee, because the other side can add or remove tokens in the shared place; there a component's conservation law carries over only when the other side has an invariant that agrees with it on the shared places. Appendix E, [Composition Preserves Safety, Not Liveness](appendix-e-categorical-foundations.md#composition-preserves-safety-not-liveness), states both cases with their evidence.

## The Execution Pipeline

From source text to running model, the token language follows a classic interpreter pipeline:

1. **Lexer** ([`lexer.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/lexer.go)) — tokenizes source into `LParen`, `RParen`, `Arrow`, `Keyword`, `Symbol`, `String`, `Number` and `Guard` tokens
2. **Parser** ([`parser.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/parser.go)) — builds an AST with `SchemaNode`, `StateNode`, `ActionNode`, `ArcNode`, `ConstraintNode`
3. **Interpreter** ([`interpret.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/interpret.go)) — converts the AST to a `tokenmodel.Schema` and validates it: IDs unique and non-empty, every arc joining a declared state to a declared action
4. **Runtime** ([`runtime.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/runtime.go)) — executes actions against a snapshot of the state

`Runtime.ExecuteWithBindings` runs an action in this order:

1. **Guard check**: evaluate the guard with the caller's bindings, if a guard evaluator is set
2. **Enablement**: every token input holds at least its arc's weight (an inhibitor input, fewer)
3. **Input processing**: consume tokens on token arcs; subtract the bound value on data arcs
4. **Output processing**: produce tokens on token arcs; add the bound value on data arcs
5. **Sequence**: increment `Runtime.Sequence`
6. **Invariant check**: evaluate every constraint against the token counts, if checking is on and an evaluator is set

If the guard fails or the action is not enabled, the call returns an error and nothing changes. If a constraint fails, the call returns an error but the update stays, as described under [Guard vs. Invariant](#guard-vs-invariant). `Sequence` counts executed actions; a caller can compare it to detect a stale snapshot, but the runtime itself does no concurrency control.

```go
schema, err := dsl.ParseSchema(`
  (schema Counter
    (states
      (state count :kind token :initial 10)
      (state done :kind token :initial 0))
    (actions
      (action decrement))
    (arcs
      (arc count -> decrement)
      (arc decrement -> done)))
`)
```

The `ParseSchema` function runs the full pipeline — lexing, parsing, interpretation, and validation — returning an executable schema or an error. `decrement` needs no guard: its input arc already requires a token in `count`, so starting from 10 it is enabled exactly ten times, after which `count` is 0, `done` is 10 and `Execute("decrement")` returns `ErrActionNotEnabled`. Every firing moves one token from `count` to `done`, so `count + done == 10` throughout. This counter is a sketch for this chapter, not a test in go-pflow, and the behaviour above is derived from the arcs, not run.

## Putting It Together

The token language adds three layers on top of the mathematical Petri net:

1. **Guards and invariants** — conditional firing and conservation enforcement
2. **Typed state** — TokenState for discrete counts, DataState for structured values
3. **Net types and composition** — categorical classification with typed links

These layers organize the mathematics rather than replace it. For the token places, the incidence matrix, state equation and P-invariants of Chapters 2 and 3 still apply; guards add preconditions, constraints add runtime checks after each step, and types add composition rules. Guards and inhibitor arcs are the part that falls outside the incidence matrix, which is where Appendix E draws the contextual boundary.

The four-term vocabulary — cell, func, arrow, guard — is how the rest of the book describes models. Chapter 6 uses this package directly, writing tic-tac-toe in its Go struct-tag form. The other application chapters, from coffee shops to poker tables, present their models in other notations, but each of them comes down to the places, transitions, arcs and guards defined here.
