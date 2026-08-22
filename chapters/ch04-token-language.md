# The Token Language

**Learning objective**: Use the pflow DSL to define nets with guards, roles, and typed arcs.

The first three chapters built Petri nets from mathematical primitives — places, transitions, arcs, tokens, rate constants. That's sufficient for analysis but awkward for specification. You wouldn't write a web application by specifying its state machine as a 5-tuple. You'd want a language.

This chapter introduces the token language — a small, interpreted DSL for defining Petri net models with rich data semantics. It extends the mathematical foundation with guards, typed state, invariants, and categorical net types, while keeping the vocabulary to exactly four terms.

## The Four-Term Primitive

The token language distills to four primitives:

| Term | Maps To | Purpose |
|------|---------|---------|
| `cell` | State / Place | Containers for tokens or data |
| `func` | Action / Transition | Operations that transform state |
| `arrow` | Arc | Flow connections with optional bindings |
| `guard` | Predicate | Boolean conditions gating actions |

Every token model — from an ERC-20 token standard to a poker game to a hospital workflow — is expressed with these four terms. `cell` defines what exists. `func` defines what happens. `arrow` defines what flows where. `guard` defines what's allowed.

This isn't minimalism for its own sake: four terms is the smallest vocabulary that captures the essential structure of executable schemas. Fewer terms would lose expressiveness. More terms would add redundancy. The four-term primitive sits at the same sweet spot as Petri nets themselves: just enough structure to be useful, not so much that it obscures the model.

## S-Expression Syntax

The DSL uses an S-expression syntax. Here's a complete token model for a simplified ERC-20 token:

```lisp
(schema ERC-20
  (version v1.0.0)

  (state totalSupply :type uint256)
  (state balances :type map[address]uint256 :exported)
  (state allowances :type map[address]map[address]uint256)

  (action transfer :guard {balances[from] >= amount && to != address(0)})
  (action approve)
  (action mint :guard {to != address(0)})
  (action burn :guard {balances[from] >= amount})

  (arc balances -> transfer :keys (from))
  (arc transfer -> balances :keys (to))
  (arc mint -> totalSupply)
  (arc totalSupply -> burn)

  (constraint conservation {sum(balances) == totalSupply}))
```

Each form declares structure:

- **`state`** declares a cell with a name and type. The `:exported` keyword marks states that are visible across schema boundaries.
- **`action`** declares a func, optionally with a guard expression in braces.
- **`arc`** declares an arrow between a state and an action (or vice versa). The `->` is syntactic sugar for the direction. `:keys` specifies map access paths for data states.
- **`constraint`** declares an invariant — a property that must hold after every state transition.

The syntax is deliberately simple. No variable declarations, no control flow, no modules. The structure of the model is visible in the shape of the text.

### In Go

The same model can be constructed with a fluent builder API:

```go
schema := dsl.Build("ERC-20").
    Version("v1.0.0").
    Data("totalSupply", "uint256").
    Data("balances", "map[address]uint256").Exported().
    Data("allowances", "map[address]map[address]uint256").
    Action("transfer").Guard("balances[from] >= amount && to != address(0)").
    Action("approve").
    Action("mint").Guard("to != address(0)").
    Action("burn").Guard("balances[from] >= amount").
    Flow("balances", "transfer").Keys("from").
    Flow("transfer", "balances").Keys("to").
    Flow("mint", "totalSupply").
    Flow("totalSupply", "burn").
    Constraint("conservation", "sum(balances) == totalSupply").
    MustSchema()
```

Both representations — S-expression and builder — produce the same executable schema. The S-expression is better for reading and sharing. The builder is better for programmatic construction and code generation.

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

Functions:     sum(balances)
               count(owners)
               address(0)
```

The evaluator uses short-circuit evaluation — `&&` stops on the first false, `||` stops on the first true — so guards with early-exit conditions are efficient.

### Guard vs. Invariant

Both guards and invariants are boolean expressions, but they serve different purposes and fire at different times:

| Aspect | Guard | Invariant |
|--------|-------|-----------|
| When checked | Before action fires | After state update |
| On failure | Action is blocked | Transaction rolls back |
| Purpose | Precondition | System property |
| Scope | Single action | Entire schema |

A guard asks: "Is this action allowed right now?" An invariant asks: "Did this action break anything?"

For the ERC-20 transfer, the guard `balances[from] >= amount` prevents overdrafts. The invariant `sum(balances) == totalSupply` catches any bug that would create or destroy tokens. The guard is the first line of defense; the invariant is the safety net.

### Evaluation Context

Guards evaluate against a context containing three things:

1. **Bindings** — variable values from the action's parameters (`from`, `to`, `amount`)
2. **Functions** — built-in aggregates and custom functions (`sum`, `count`, `address`)
3. **State** — the current snapshot for data access

When the guard `balances[from] >= amount` runs, `from` is resolved from the bindings, `balances` is resolved from the current state, and `>=` performs the comparison. Missing map keys default to zero — accessing `balances[nonexistent]` returns 0 rather than an error.

## Weighted Arcs and Inhibitor Arcs

Basic arcs have an implicit weight of 1 — one token consumed, one token produced. Weighted arcs generalize this.

### Weighted Arcs

An arc with weight $w$ from a place to a transition means "this transition consumes $w$ tokens from this place when it fires." An arc with weight $w$ from a transition to a place means "this transition produces $w$ tokens in this place when it fires."

In the token language, arc weights appear in the schema definition. A coffee recipe that requires 2 units of water and 1 unit of grounds:

```lisp
(state water :kind token :initial 10)
(state grounds :kind token :initial 5)
(state coffee :kind token :initial 0)

(action brew)

(arc water -> brew :value 2)
(arc grounds -> brew :value 1)
(arc brew -> coffee :value 1)
```

The `brew` action consumes 2 water and 1 grounds, producing 1 coffee. The enabling condition requires $M(\text{water}) \geq 2$ and $M(\text{grounds}) \geq 1$.

### Inhibitor Arcs

An **inhibitor arc** is the opposite of a normal arc: it prevents a transition from firing when the source place *has* tokens, rather than when it doesn't. Drawn as an arc with a small circle instead of an arrowhead, an inhibitor arc from place $p$ to transition $t$ means "$t$ can only fire when $p$ is empty."

Inhibitor arcs add expressiveness beyond standard Petri nets. They can test for zero — something that basic Petri nets cannot do. This makes inhibitor nets Turing-complete, at the cost of losing some decidability results.

In practice, inhibitor arcs model capacity limits and mutual exclusion. A buffer with maximum capacity 5:

```lisp
(state buffer :kind token :initial 0)
(state empty_slots :kind token :initial 5)

(action produce)
(action consume)

(arc empty_slots -> produce :value 1)
(arc produce -> buffer :value 1)
(arc buffer -> consume :value 1)
(arc consume -> empty_slots :value 1)
```

This uses a complementary place (`empty_slots`) rather than an inhibitor arc — a common pattern that achieves the same effect while staying within the standard Petri net formalism. The P-invariant $M(\text{buffer}) + M(\text{empty\_slots}) = 5$ is maintained by the net's structure.

## Tokens vs. Data

The token language distinguishes two kinds of state, reflecting two fundamentally different ways that places hold information.

### TokenState

Integer counters representing discrete quantities. This is classic Petri net semantics:

- Firing consumes and produces tokens
- The enabling condition checks token counts
- Tokens are fungible — we track counts, not identities

A place with `kind: token` behaves exactly like a mathematical Petri net place. Tokens are integers. Arcs carry weights. Conservation laws hold.

### DataState

Typed containers holding structured data — maps, records, scalar values. This extends Petri nets to handle real-world state:

- Arc keys specify access paths (`balances[from]`)
- Data arcs read and write; they don't consume
- Types are explicit (`map[address]uint256`, `bool`, `string`)

A place with `kind: data` holds a value, not a count. The `balances` place in the ERC-20 schema holds a map from addresses to amounts. The `transfer` action doesn't consume the map — it reads from one key and writes to another.

The combination of TokenState and DataState lets you model systems where both control flow (tokens tracking "where we are") and data transformations (balances, permissions, configuration) matter. A workflow token tracks the order through `pending → confirmed → shipped`. Data states track the order's contents, the customer's address, the payment amount.

## Categorical Net Types

Not all Petri nets are alike. An order-processing workflow and an epidemiological simulation both use places, transitions, and arcs, but their tokens mean fundamentally different things. **Net types** classify Petri nets by how their tokens behave, what invariants hold, and how they compose.

### The Five Types

| Type | Token Semantics | Invariant |
|------|----------------|-----------|
| **WorkflowNet** | Single control-flow token (cursor) | Mutual exclusion — cursor in exactly one state |
| **ResourceNet** | Countable inventory tokens | Conservation — total tokens constant |
| **GameNet** | Turn-based tokens with player roles | Both mutual exclusion and conservation |
| **ComputationNet** | Continuous quantities via ODE rates | Rate conservation at steady state |
| **ClassificationNet** | Signal tokens with threshold activation | Threshold firing conditions |

**WorkflowNet.** A single token traces a path through states. An order moves through `pending → confirmed → shipped → delivered`. The token is a cursor — it marks where you are in a sequential process. WorkflowNets are finite state machines with the structural advantage that parallelism (multiple concurrent tokens) is expressible.

**ResourceNet.** Tokens represent countable, fungible things: inventory items, currency, capacity slots. A warehouse has 100 tokens in `available`, consumed by `reserve` and produced into `reserved`. The total is conserved — the P-invariant guarantees that nothing is created or destroyed.

**GameNet.** Tokens encode both game state and turn structure. In tic-tac-toe, empty cells hold tokens consumed when a player moves. A turn-control token alternates between players. GameNets combine WorkflowNet sequencing (turns) with ResourceNet accounting (board positions).

**ComputationNet.** Tokens are continuous, not discrete. Transitions fire at rates — mass-action kinetics from Chapter 3. An SIR epidemic model is a ComputationNet: the infection rate depends on the product of susceptible and infected populations.

**ClassificationNet.** Tokens accumulate as evidence. Transitions fire when enough tokens accrue — a threshold. A spam filter accumulates signal tokens from heuristics (suspicious sender, keyword matches, link density) and fires `classify_spam` when the threshold is met.

### Why Types Matter

At the structural level, all five types are Petri nets. The type annotations matter because they constrain composition. When you connect two nets, the types determine what connections are valid:

- An **EventLink** connects transitions to transitions across schemas — when one fires, the other fires too
- A **DataLink** connects places to places — read-only observation across schema boundaries
- A **TokenLink** transfers tokens between schemas — resource coupling with cross-boundary conservation
- A **GuardLink** gates a transition in one schema on a place in another — constraint coupling

Types make the composition rules explicit. You can't accidentally link a workflow cursor to an inventory counter. The type system prevents semantic nonsense at the structural level.

### Composition

Real systems are compositions of multiple nets. An order system combines a WorkflowNet (order lifecycle) with a ResourceNet (inventory), linked by EventLinks:

```
Orders (WorkflowNet):    pending -> confirm -> confirmed -> ship -> shipped
Inventory (ResourceNet): available -> reserve -> reserved -> ship_out -> consumed
```

EventLinks:
1. `orders.confirm → inventory.reserve` — confirming reserves stock
2. `orders.ship → inventory.ship_out` — shipping consumes reserved stock

Each schema carries its own verified properties. The Orders WorkflowNet guarantees mutual exclusion (the order is in exactly one state). The Inventory ResourceNet guarantees conservation (total stock is constant). These properties hold independently — adding links doesn't invalidate either guarantee.

This is **assume-guarantee reasoning**: each component is verified in isolation, and composition only needs to check the boundaries. Flattening the composed net and recomputing its P-invariants recovers both component laws — `pending + confirmed + shipped == 1` and `available + reserved + consumed == N` — from the incidence matrix of the whole, so neither guarantee was lost.

Composition is associative and commutative: the result depends on which nets and links you declared, never on the order you declared them in.

It is **not** monotonic, and it is worth being precise about why. Adding an *unlinked* schema is the monoidal product — the two nets share nothing, so each behaves exactly as it did alone. But adding a *link* restricts. An EventLink is a rendezvous: the fused transition fires only when every participant is enabled, so `inventory.reserve` can no longer fire whenever stock exists — it must wait for an order to confirm. A GuardLink gates by construction. A TokenLink introduces a shared place that constrains its consumers. Each of these removes behavior rather than adding it. In the example above, Inventory alone reaches 66 states; composed with Orders it reaches 3.

What survives is **refinement**: every firing sequence of the composite, projected onto a component's alphabet, is a firing sequence of that component alone. That is the property assume-guarantee actually needs, and it is what preserves *safety* — invariants, mutual exclusion, conservation. It does not preserve *liveness*: a component that could always eventually fire may, once linked, wait forever on a partner. Adding a link cannot make a composed system reach a state its components forbid, but it can certainly stop it reaching one they allow.

## The Execution Pipeline

From source text to running model, the token language follows a classic interpreter pipeline:

1. **Lexer** — tokenizes source into `LPAREN`, `RPAREN`, `SYMBOL`, `GUARD`, `KEYWORD`, `ARROW`
2. **Parser** — builds an AST with `SchemaNode`, `StateNode`, `ActionNode`, `ArcNode`, `ConstraintNode`
3. **Interpreter** — converts the AST to an executable `Schema` with validated references
4. **Runtime** — executes actions: guard check → input processing → output processing → invariant check → commit

Each action execution follows a strict sequence:

1. **Guard check**: evaluate the guard expression with current bindings and state
2. **Input processing**: consume tokens from input arcs, bind data from data arcs
3. **Output processing**: produce tokens at output arcs, update data at data arcs
4. **Invariant check**: verify all constraints still hold
5. **Commit**: update the snapshot with an incremented sequence number

If the guard fails, the action is blocked — nothing changes. If an invariant fails after the state update, the entire transaction rolls back. The sequence number provides an optimistic concurrency control mechanism: clients can detect stale state by comparing sequence numbers.

```go
schema, err := dsl.ParseSchema(`
  (schema Counter
    (state count :kind token :initial 10)
    (state done :kind token :initial 0)
    (action decrement :guard {tokens(count) > 0})
    (arc count -> decrement)
    (arc decrement -> done))
`)
```

The `ParseSchema` function runs the full pipeline — lexing, parsing, interpretation, and validation — returning an executable schema or an error.

## Putting It Together

The token language adds three layers on top of the mathematical Petri net:

1. **Guards and invariants** — conditional firing and conservation enforcement
2. **Typed state** — TokenState for discrete counts, DataState for structured values
3. **Net types and composition** — categorical classification with typed links

These layers don't replace the mathematics; they organize it. The incidence matrix, state equation, and P-invariants from Chapters 2 and 3 still apply. Guards add preconditions. Invariants add postconditions. Types add composition rules.

The four-term vocabulary — cell, func, arrow, guard — is the interface between human intent and the mathematical machinery. You describe the model in these terms. The machinery handles the rest.

Part II of this book uses the token language extensively. Each application chapter defines its model in this DSL, simulates it with the ODE solver from Chapter 3, and analyzes it with the structural tools from Chapter 2. The language is the common thread — the same four terms, applied to coffee shops, game boards, constraint puzzles, biochemical pathways, and poker tables.
