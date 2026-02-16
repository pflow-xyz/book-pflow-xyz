# Appendix B: Token Language Grammar

This appendix documents the S-expression DSL for defining token model schemas. The DSL is an alternative to JSON for defining Petri net models — more readable for humans, equivalent in expressiveness.

## Lexical Elements

The lexer recognizes these token types:

| Token | Pattern | Examples |
|-------|---------|----------|
| `LParen` | `(` | `(` |
| `RParen` | `)` | `)` |
| `Arrow` | `->` | `->` |
| `Keyword` | `:` followed by symbol | `:type`, `:guard`, `:keys`, `:value`, `:initial`, `:exported`, `:kind` |
| `Symbol` | letter or `_`, then letters/digits/`_`/`-`/`[`/`]`/`.` | `balances`, `map[string]int64`, `v1.0.0` |
| `Number` | digits, optionally prefixed with `-` | `42`, `-1`, `0` |
| `String` | `"..."` with backslash escapes | `"hello"`, `"line\n"` |
| `Guard` | `{...}` with nested brace support | `{amount > 0}`, `{balances[from] >= amount}` |
| `Comment` | `;` to end of line | `; this is ignored` |

Whitespace (spaces, tabs, newlines) separates tokens but is otherwise ignored.

## Top-Level Structure

A schema definition has the form:

```lisp
(schema <name>
  (version <version-string>)       ; optional
  (states ...)                     ; required
  (actions ...)                    ; required
  (arcs ...)                       ; required
  (constraints ...))               ; optional
```

The `<name>` is a symbol identifying the schema. Clauses may appear in any order, but the conventional order is version, states, actions, arcs, constraints.

## States

The `states` clause defines places in the Petri net:

```lisp
(states
  (state <id> :type <type> :kind <kind> :initial <value> :exported))
```

### State Keywords

| Keyword | Value | Default | Description |
|---------|-------|---------|-------------|
| `:type` | symbol | (none) | Data type: `string`, `int64`, `float64`, `bool`, `map[string]int64`, `map[string]string`, `map[string]map[string]int64` |
| `:kind` | `token` or `data` | `data` | Whether the place holds a token count or structured data |
| `:initial` | number, string, or `nil` | (none) | Initial value |
| `:exported` | (no value) | false | Flag — presence means the state is externally visible |

All keywords are optional. A minimal state definition is just `(state <id>)`.

### Examples

```lisp
; Token-counting place with initial tokens
(state pending :kind token :initial 1)

; Data place holding a map
(state balances :type map[string]int64 :exported)

; Simple data place with initial value
(state total_supply :type int64 :initial 0 :exported)
```

## Actions

The `actions` clause defines transitions:

```lisp
(actions
  (action <id> :guard {<expression>}))
```

### Action Keywords

| Keyword | Value | Description |
|---------|-------|-------------|
| `:guard` | guard expression in `{...}` | Boolean precondition for firing |

### Examples

```lisp
; Action with no guard (always enabled when inputs are available)
(action ship)

; Action with guard expression
(action transfer :guard {balances[from] >= amount && amount > 0})

; Action with simple guard
(action mint :guard {amount > 0})
```

## Arcs

The `arcs` clause defines connections between states and actions:

```lisp
(arcs
  (arc <source> -> <target> :keys (<key1> <key2> ...) :value <binding>))
```

The `->` arrow separates source from target. Arcs are bipartite: they connect a state to an action (input arc) or an action to a state (output arc).

### Arc Keywords

| Keyword | Value | Description |
|---------|-------|-------------|
| `:keys` | parenthesized list of symbols | Map access keys for data places |
| `:value` | symbol | Value binding name (defaults to `amount`) |

### Examples

```lisp
; Simple token arc (no keywords needed)
(arc pending -> ship)
(arc ship -> shipped)

; Data arc with keys and value
(arc balances -> transfer :keys (from) :value amount)
(arc transfer -> balances :keys (to) :value amount)

; Arc with multiple keys
(arc ledger -> settle :keys (from to) :value amount)
```

## Constraints

The `constraints` clause defines invariants:

```lisp
(constraints
  (constraint <id> {<expression>}))
```

The expression is enclosed in `{...}` braces, like guard expressions. Constraints are checked at validation time and should hold for all reachable markings.

### Examples

```lisp
; Conservation law
(constraint conservation {sum(balances) == totalSupply})

; Mutual exclusion
(constraint mutex {pending + shipped + delivered == 1})
```

## Guard Expression Syntax

Guard expressions (used in both `:guard` and `constraint`) support:

| Category | Operators | Examples |
|----------|-----------|----------|
| Comparison | `==`, `!=`, `<`, `>`, `<=`, `>=` | `amount > 0` |
| Boolean | `&&`, `\|\|`, `!` | `a > 0 && b > 0` |
| Arithmetic | `+`, `-`, `*`, `/` | `sum(balances)` |
| Map access | `name[key]` | `balances[from]` |
| Aggregates | `sum()`, `count()`, `tokens()`, `minOf()`, `maxOf()` | `sum(balances) == totalSupply` |

Guard expressions are opaque to the DSL parser — the lexer captures everything between `{` and `}` as a single token. The guard DSL parser in petri-pilot evaluates these expressions at runtime against the current state.

## Complete Examples

### ERC-20 Token

```lisp
(schema erc20-token
  (version v1.0.0)

  (states
    (state balances :type map[string]int64 :exported)
    (state total_supply :type int64 :initial 0 :exported))

  (actions
    (action transfer :guard {balances[from] >= amount && amount > 0})
    (action mint :guard {amount > 0}))

  (arcs
    (arc balances -> transfer :keys (from) :value amount)
    (arc transfer -> balances :keys (to) :value amount)
    (arc mint -> balances :keys (to) :value amount)
    (arc mint -> total_supply :value amount))

  (constraints
    (constraint conservation {sum(balances) == total_supply})))
```

### Order Workflow

```lisp
(schema order-processing
  (version v1.0.0)

  (states
    (state received :kind token :initial 1)
    (state validated :kind token)
    (state shipped :kind token)
    (state completed :kind token))

  (actions
    (action validate)
    (action ship)
    (action deliver))

  (arcs
    (arc received -> validate)
    (arc validate -> validated)
    (arc validated -> ship)
    (arc ship -> shipped)
    (arc shipped -> deliver)
    (arc deliver -> completed))

  (constraints
    (constraint conservation {received + validated + shipped + completed == 1})))
```

## AST Representation

The parser produces an abstract syntax tree with these node types:

| Node | Fields | Description |
|------|--------|-------------|
| `SchemaNode` | Name, Version, States, Actions, Arcs, Constraints | Root node |
| `StateNode` | ID, Type, Kind, Initial, Exported | A place definition |
| `ActionNode` | ID, Guard | A transition definition |
| `ArcNode` | Source, Target, Keys, Value | A connection |
| `ConstraintNode` | ID, Expr | An invariant |

The AST is the same regardless of whether the schema was defined via the S-expression DSL, the Go struct tag syntax, or the fluent builder API. All three produce identical `SchemaNode` trees, which are then compiled to Petri nets.

## Equivalence with JSON

Every DSL schema has an equivalent JSON representation. The mapping is direct:

| DSL | JSON |
|-----|------|
| `(state id :kind token :initial 1)` | `{"id": "id", "kind": "token", "initial": 1}` |
| `(action id :guard {expr})` | `{"id": "id", "guard": "expr"}` |
| `(arc src -> tgt :keys (k) :value v)` | `{"from": "src", "to": "tgt", "keys": ["k"], "value": "v"}` |
| `(constraint id {expr})` | `{"id": "id", "expr": "expr"}` |

The DSL is syntactic sugar. The JSON schema (Appendix C) is the canonical format.
