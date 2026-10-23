# Appendix B: Token Language Grammar

This appendix documents the S-expression DSL for defining token model schemas, the syntax introduced in [Chapter 4: The Token Language](ch04-token-language.md). The implementation is go-pflow's [`tokenmodel/dsl`](https://github.com/pflow-xyz/go-pflow/tree/main/tokenmodel/dsl) package: [`lexer.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/lexer.go), [`parser.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/parser.go), [`ast.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/ast.go) and [`interpret.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/interpret.go), with tests in [`dsl_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/dsl_test.go). The rules below were read from that code; the tests were not re-run for this appendix. The DSL is easier to read than JSON, but it expresses less: the JSON model format of Appendix C has fields the DSL has no syntax for (see [Equivalence with JSON](#equivalence-with-json)).

## Lexical Elements

The lexer recognizes these token types:

| Token | Pattern | Examples |
|-------|---------|----------|
| `LParen` | `(` | `(` |
| `RParen` | `)` | `)` |
| `Arrow` | `->` | `->` |
| `Keyword` | `:` followed by symbol | `:type`, `:guard`, `:keys`, `:value`, `:initial`, `:exported`, `:kind` |
| `Symbol` | letter or `_` (or a `-` not followed by `>` or a digit), then letters/digits/`_`/`-`/`[`/`]`/`.` | `balances`, `map[string]int64`, `v1.0.0`, `ERC-020` |
| `Number` | integer digits, optionally prefixed with `-`; no decimal point | `42`, `-1`, `0` |
| `String` | `"..."` with backslash escapes | `"hello"`, `"line\n"` |
| `Guard` | `{...}` with nested brace support | `{amount > 0}`, `{balances[from] >= amount}` |
| `Comment` | `;` to end of line | `; this is ignored` |

Whitespace (spaces, tabs, newlines) separates tokens but is otherwise ignored.

## Top-Level Structure

A schema definition has the form:

```lisp
(schema <name>
  (version <version>)
  (states ...)
  (actions ...)
  (arcs ...)
  (constraints ...))
```

The `<name>` and `<version>` are symbols, so `v1.0.0` is written without quotes; a quoted string in either position is a parse error. The parser requires none of the clauses: `(schema Test (version v1.0.0))` is a complete schema in `TestParser_MinimalSchema`. Clauses may appear in any order, and a repeated clause replaces the earlier one. The conventional order is version, states, actions, arcs, constraints.

## States

The `states` clause defines places in the Petri net:

```lisp
(states
  (state <id> :type <type> :kind <kind> :initial <value> :exported))
```

### State Keywords

| Keyword | Value | Default | Description |
|---------|-------|---------|-------------|
| `:type` | symbol | (none) | Data type, stored as written and not checked by the parser or by validation. go-pflow's tests use `uint256`, `map[address]uint256` and `map[address]map[address]uint256` |
| `:kind` | `token` or `data` | `data` | Whether the place holds a token count or structured data |
| `:initial` | integer, string, `nil`, `(map)` or `(list)` | (none) | Initial value; `(map)` and `(list)` are an empty map and an empty list |
| `:exported` | (no value) | false | Flag; an exported state can be bound across schemas |

All keywords are optional. A minimal state definition is just `(state <id>)`, which is a data state. The default kind differs from the JSON model format, where a place with no `kind` is a token place; a token-counting place in the DSL needs `:kind token`.

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

The `->` arrow separates source from target. The parser accepts any two symbols here; the check that an arc joins a state to an action (input arc) or an action to a state (output arc) comes afterwards, when `Interpret` calls `Schema.Validate` ([`tokenmodel/validate.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/validate.go)), which also rejects an end that names no declared state or action.

### Arc Keywords

| Keyword | Value | Description |
|---------|-------|-------------|
| `:keys` | parenthesized list of symbols | Map access keys for data places |
| `:value` | symbol | Value binding name; when absent, the runtime uses `amount` |

The DSL has no syntax for arc weight or for inhibitor and read arcs. go-pflow's `tokenmodel.Arc` has `Weight` and `Type` fields, but the DSL parser, the builder and the struct-tag dialect never set them, so every arc written in the DSL is a normal arc of weight 1. The contextual arcs of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries) have to be written in JSON or Go.

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

The expression is enclosed in `{...}` braces, like guard expressions. `Schema.Validate` does not look at constraints. go-pflow's `tokenmodel.Runtime` ([`runtime.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/runtime.go)) evaluates them against the token counts after each action it executes, provided `CheckConstraints` is true (the default) and a guard evaluator has been supplied; with no evaluator, constraints are skipped. A constraint is a runtime assertion about the markings a run actually visits. It does not prove the property over every reachable marking.

### Examples

```lisp
; Conservation law
(constraint conservation {sum(balances) == total_supply})

; Mutual exclusion
(constraint mutex {pending + shipped + delivered == 1})
```

## Guard Expression Syntax

Guard expressions (used in both `:guard` and `constraint`) support:

| Category | Operators | Examples |
|----------|-----------|----------|
| Comparison | `==`, `!=`, `<`, `>`, `<=`, `>=` | `amount > 0` |
| Boolean | `&&`, `\|\|`, `!` | `a > 0 && b > 0` |
| Arithmetic | `+`, `-`, `*`, `/`, `%` | `balances[from] - amount >= 0` |
| Map access | `name[key]` | `balances[from]` |
| Aggregates | see below | `sum("balances_") == totalSupply` |

The DSL parser does not read guard expressions. The lexer captures everything between `{` and the matching `}` as a single token, and the parser stores it as a string. Evaluation happens later, through whatever guard evaluator the runtime is given, and the expression language has two implementations whose operators agree but whose aggregate functions do not:

- go-pflow's [`tokenmodel/guard`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/guard/aggregate.go) provides `sum`, `count`, `tokens`, `min` and `max`. Each takes a place ID or ID prefix as a string: `sum("balances_")` adds the token counts of every place whose ID starts with `balances_` (`guard_test.go` checks it equals 600 on its test marking).
- sim.pflow.xyz's guard DSL (`pkg/dsl/guard.go`; sim's source is not public) names the last two `minOf` and `maxOf`, and its `sum`, `count`, `minOf` and `maxOf` also accept a data state's map, so `sum(balances)` totals the values in the `balances` map.

The constraint `sum(balances) == total_supply` used in this appendix is written for sim.pflow.xyz's evaluator. Under go-pflow's it fails to evaluate, because `balances` is a bare name rather than a string and is not a token place.

## Complete Examples

Neither example below is in go-pflow's test suite, and neither was run for this appendix. The nearest tested schema is `erc020DSL` in [`dsl_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/dsl_test.go), a fuller ERC-20 with allowances and `burn`: `TestInterpret_ERC020` parses and interprets it and checks for 3 states, 5 actions, 10 arcs and 1 constraint, and `TestRoundTrip_ParseAndValidate` passes it through `Schema.Validate`.

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

The conservation constraint holds by construction. Each action has one input arc and one output arc of weight 1, so every firing moves the single token one place along the chain and the four counts always sum to the initial 1. That follows from the arcs by hand; it was not checked by a run.

## AST Representation

The parser produces an abstract syntax tree with these node types:

| Node | Fields | Description |
|------|--------|-------------|
| `SchemaNode` | Name, Version, States, Actions, Arcs, Constraints | Root node |
| `StateNode` | ID, Type, Kind, Initial, Exported | A place definition |
| `ActionNode` | ID, Guard | A transition definition |
| `ArcNode` | Source, Target, Keys, Value | A connection |
| `ConstraintNode` | ID, Expr | An invariant |

The S-expression DSL is one of three front ends that build a `SchemaNode`. The others are the fluent builder ([`builder.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/dsl/builder.go), `dsl.Build(...)`) and the Go struct-tag dialect (`tags.go` in the same package, `dsl.ASTFromStruct`). `Interpret` turns any `SchemaNode` into a `tokenmodel.Schema` and validates it. The three front ends do not produce identical trees for the same model: the builder and struct-tag paths default the version to `v1.0.0` and give token states the type `int`, where the parser leaves both empty unless the source sets them.

## Equivalence with JSON

A parsed DSL schema has two JSON forms. The first is `tokenmodel.Schema` itself, whose JSON field names come from the struct tags in [`tokenmodel/schema.go`](https://github.com/pflow-xyz/go-pflow/blob/main/tokenmodel/schema.go); its top-level keys are `states`, `actions`, `arcs` and `constraints`. The second is the model format of [Appendix C: JSON Schema Reference](appendix-c-json-schema.md), which calls the same things `places`, `transitions`, `arcs` and `constraints`. go-pflow's `metamodel.FromTokenModel` ([`metamodel/convert.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/convert.go)) converts the first into the second:

| DSL | `tokenmodel.Schema` JSON | Appendix C model JSON |
|-----|--------------------------|-----------------------|
| `(state id :kind token :initial 1)` | `{"id": "id", "kind": "token", "initial": 1}` | `{"id": "id", "kind": "token", "initial": 1}` |
| `(action id :guard {expr})` | `{"id": "id", "guard": "expr"}` | `{"id": "id", "guard": "expr"}` |
| `(arc src -> tgt :keys (k) :value v)` | `{"source": "src", "target": "tgt", "keys": ["k"], "value": "v"}` | `{"from": "src", "to": "tgt", "weight": 1, "keys": ["k"], "value": "v"}` |
| `(constraint id {expr})` | `{"id": "id", "expr": "expr"}` | `{"id": "id", "expr": "expr"}` |

go-pflow provides both halves of this route: `ParseSchema` in [`tokenmodel/dsl`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/tokenmodel/dsl/interpret.go) and then `FromTokenModel` in [`metamodel/convert.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/convert.go). `FromTokenModel` drops each data state's `:initial`, so a tool that accepts DSL input this way has to copy it into the place's `initial_value` itself.

The two formats are not equally expressive. Arc weights and types, transition rates, views, parameters and the other sections of Appendix C have no DSL syntax, so every DSL schema has a JSON model but a JSON model cannot in general be written as DSL without losing fields.
