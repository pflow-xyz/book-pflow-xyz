# Appendix C: JSON Schema Reference

This appendix documents the JSON model format of go-pflow's `metamodel.Model` ([`metamodel/schema.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/schema.go), at v0.33.0). It is the format sim.pflow.xyz stores and runs (Chapter 18), and the one go-pflow's simulators and verifier read. The format defines what a valid Petri net model looks like: places, transitions and arcs, plus optional blocks for constraints, presentation intent (`view`, `views`), declared parameters and simulation.

A JSON Schema file for the format, `petri-model.schema.json`, exists, but it was written when the format's main reader was an application generator, and it is not published. The Go types decide what is accepted, and where the schema file and the types disagree, this appendix follows the types and says so. The disagreements are transition `bindings` and `rate`, arc `type` and `kinetic`, and the schema file's application-generator blocks and form-style `views`, which go-pflow's types no longer have (see the end of [Top-Level Structure](#top-level-structure)). They were found by reading the Go types against the schema file.

The types are also strict where it matters: sim.pflow.xyz refuses a model that carries a key `metamodel.Model` does not define, naming each unknown key with its path and the key it most likely meant, rather than letting Go's decoder drop it silently.

## Top-Level Structure

A model is a JSON object with four required fields and several optional ones:

```json
{
  "name": "order-processing",
  "version": "1.0.0",
  "description": "Order fulfillment workflow",
  "places": [...],
  "transitions": [...],
  "arcs": [...],
  "constraints": [...],
  "view": "...",
  "views": [...],
  "parameters": [...],
  "simulation": {...}
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | Yes | string | Unique identifier, kebab-case (`^[a-z][a-z0-9-]*$`) |
| `version` | No | string | Semantic version (e.g., `"1.0.0"`) |
| `description` | No | string | Human-readable description |
| `places` | Yes | array | States in the Petri net (min 1) |
| `transitions` | Yes | array | Actions/events (min 1) |
| `arcs` | Yes | array | Connections between places and transitions (min 1) |
| `constraints` | No | array | Invariants that must hold |
| `view` | No | string | Presentation intent: prose saying what an application built on the model should be |
| `views` | No | array | Screens of that application, each a checked projection of the model |
| `parameters` | No | array | Arc weights or capacities exposed as named structural knobs |
| `simulation` | No | object | Objective, players and ODE solver configuration |

`metamodel.Model` has a few more optional fields, among them `events` (the data contract for transitions), `decimals` and `unit` (token display), `assertedClasses` and `presentation` (machine-readable console hints); `schema.go` documents them. Earlier versions of this format also carried application-generator blocks: `roles`, `access`, `navigation`, `admin` and `eventSourcing`, and a form-style `views`. go-pflow's `Model` has no fields for the first five, so sim.pflow.xyz refuses a model that carries them, and `views` now has the shape described under [Views](#views) below.

## Places

A place holds state — either token counts (classic Petri net) or structured data.

```json
{
  "id": "order_received",
  "description": "Order has been received but not yet validated",
  "initial": 1,
  "kind": "token",
  "type": "string",
  "initial_value": null,
  "exported": false,
  "persisted": false
}
```

| Field | Required | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `id` | Yes | string | — | Unique identifier, snake_case (`^[a-z][a-z0-9_]*$`) |
| `description` | No | string | — | Human-readable description |
| `initial` | No | integer | `0` | Initial token count (min 0) |
| `kind` | No | `"token"` or `"data"` | `"token"` | Whether this place holds token counts or structured data |
| `type` | No | string | — | Data type for `data` kind places |
| `initial_value` | No | any | — | Initial value for data places |
| `exported` | No | boolean | `false` | Whether externally visible |
| `persisted` | No | boolean | `false` | Whether persisted in event store |
| `capacity` | No | integer | — | Upper bound on the token count, enforced after each firing by the discrete engine |

### Data Types

For `kind: "data"` places, the `type` field specifies the data type:

| Type | Description |
|------|-------------|
| `string` | Text value |
| `int64` | 64-bit integer |
| `float64` | 64-bit floating point |
| `bool` | Boolean |
| `map[string]int64` | Map from string keys to integer values |
| `map[string]string` | Map from string keys to string values |
| `map[string]map[string]int64` | Nested map (e.g., per-user balances by asset) |

## Transitions

A transition is an action that fires when enabled.

```json
{
  "id": "validate_order",
  "description": "Validate the order details",
  "guard": "amount > 0",
  "event_type": "OrderValidated",
  "http_method": "POST",
  "http_path": "/api/validate",
  "bindings": [
    {"name": "amount", "type": "number", "value": true}
  ]
}
```

| Field | Required | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `id` | Yes | string | — | Unique identifier, snake_case |
| `description` | No | string | — | Human-readable description |
| `guard` | No | string | — | Boolean precondition (guard DSL) |
| `event_type` | No | string | PascalCase of id | Custom event type name (go-pflow marks it deprecated) |
| `http_method` | No | string | `"POST"` | HTTP binding hint: `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `http_path` | No | string | `/api/{id}` | HTTP binding hint; sim's simulators ignore both |
| `bindings` | No | array of objects | — | Operational data the transition reads or writes |
| `rate` | No | number | — | Firing rate for simulation (in the Go type, not in the schema file) |

The schema file is stale on `bindings`: it describes the field as an object mapping names to strings. go-pflow's `Transition.Bindings` is an array of `Binding` objects with fields `name`, `type`, `keys` (map access path), `value` (true for the transferred amount) and `place`. Go's JSON decoder does not turn an object into an array, so by the types an object-form `bindings` fails to parse. The old map form survives as the separate field `legacy_bindings`. The binding in the example above is copied from pflow-xyz's [`extend-operations.json`](https://github.com/pflow-xyz/pflow-xyz/blob/main/examples/showcase/fixtures/extend-operations.json) fixture.

### Guard Syntax

Guards are boolean expressions evaluated against the current state. The table describes the guard DSL that sim.pflow.xyz evaluates (its `pkg/dsl`; sim's source is not public). Its lexer accepts strings in single or double quotes. go-pflow's `tokenmodel/guard` is a second implementation whose aggregate functions differ; Appendix B compares the two.

| Operator | Meaning | Example |
|----------|---------|---------|
| `==`, `!=` | Equality | `status == 'approved'` |
| `<`, `>`, `<=`, `>=` | Comparison | `amount > 0` |
| `&&`, `\|\|`, `!` | Boolean | `a > 0 && b > 0` |
| `+`, `-`, `*`, `/`, `%` | Arithmetic | `balances[from] - amount >= 0` |
| `name[key]` | Map access | `balances[from]` |
| `tokens('p')`, `sum`, `count`, `minOf`, `maxOf` | Marking aggregates (see [Objective Functions](#objective-functions)) | `tokens('stock') > 0` |

## Arcs

An arc connects a place to a transition (input) or a transition to a place (output).

```json
{
  "from": "balances",
  "to": "transfer",
  "weight": 1,
  "keys": ["from"],
  "value": "amount"
}
```

| Field | Required | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `from` | Yes | string | — | Source element ID |
| `to` | Yes | string | — | Target element ID |
| `weight` | No | integer | `1` | Tokens consumed/produced (min 1); a threshold on read and inhibitor arcs |
| `type` | No | `"inhibitor"` or `"read"` | normal | Arc kind (in the Go type, not in the schema file) |
| `kinetic` | No | boolean | `true` | Whether an input place scales the firing rate (in the Go type only) |
| `keys` | No | array of strings | — | Map access keys for data places |
| `value` | No | string | `"amount"` | Value binding name |

The schema file has no `type` field, but go-pflow's `Arc` does, and the vet clinic model in sim.pflow.xyz's catalog ([its JSON](https://sim.pflow.xyz/api/models/ae6ab5aeb95dd994bc561479)) uses both kinds, for example `{"from": "wait_emergency", "to": "start_wellness", "type": "inhibitor"}`. Neither kind moves tokens. An inhibitor arc blocks the transition while its place holds at least `weight` tokens. A read arc allows the transition only while its place holds at least `weight` tokens. These are the contextual arcs of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries), outside the incidence matrix. Setting `kinetic: false` keeps an input arc's enabling and consumption but removes its place from the rate product. go-pflow's [`metamodel/validation.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/validation.go) rejects three things here: an unknown `type`, which is not quietly run as a normal arc; a read arc that does not run place → transition; and `kinetic: false` on any arc other than a consuming place → transition arc.

## Constraints

An invariant that must always hold.

```json
{
  "id": "conservation",
  "expr": "received + validated + shipped + completed == 1"
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `id` | Yes | string | Unique identifier |
| `expr` | Yes | string | Boolean expression over place token counts |

## Views

`view` is one prose statement of what an application built on the model should be ("let two players play on the board; a click fires that cell's move transition"). `views` decomposes it into screens. Each entry is a named projection of the model, the places and transitions one screen shows, plus a prompt for whoever builds that screen and links to the screens it can navigate to.

```json
{
  "id": "board",
  "title": "Play",
  "role": "board",
  "prompt": "The nine cells as a grid; a click fires that cell's move for the player whose turn it is.",
  "places": ["x00", "o00", "turn_x", "turn_o"],
  "transitions": ["x_play_00", "o_play_00"],
  "links": ["trust"]
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `id` | Yes | string | Unique view identifier, referenced by `links` |
| `title` | No | string | Human-readable screen name |
| `role` | No | string | Kind of screen; conventional values are `board`, `dashboard`, `controls`, `advisor`, `trust` and `log` |
| `prompt` | No | string | This screen's share of the presentation intent, as prose |
| `places` | No | array of strings | Places the screen shows (empty means all) |
| `transitions` | No | array of strings | Transitions the screen may fire (empty means all) |
| `links` | No | array of strings | IDs of the views reachable from this one |

Nothing executes `view` or a view's `prompt`; they are read by whoever writes the application. The references are checkable, though: a view that names a place or transition the model does not have is an error. Both fields are part of the model's content hash, so the same net presented as a game and as a dashboard is two models. Chapter 18 shows where a published application built from them is served.

## Simulation

ODE simulation configuration for AI move evaluation.

```json
{
  "objective": "win_x - win_o",
  "players": {
    "x": {
      "maximizes": true,
      "turnPlace": "turn_x",
      "transitions": ["move_x_0", "move_x_1"]
    },
    "o": {
      "maximizes": false,
      "turnPlace": "turn_o",
      "transitions": ["move_o_0", "move_o_1"]
    }
  },
  "solver": {
    "tspan": [0, 10],
    "dt": 0.01,
    "rates": {
      "move_x_0": 1.0,
      "move_o_0": 1.0
    }
  }
}
```

### Objective Functions

The `objective` field is a numeric guard-DSL expression, using arithmetic and the aggregate functions below. The same functions are available to guards and constraints. Except for `tokens`, each function takes a place-ID **prefix**: over token places it ranges over every place whose ID starts with the argument, and over a map data place of that name it ranges over the map's values. The definitions are `MakeAggregates` and `MakeAggregatesWithData` in sim.pflow.xyz's guard DSL (`pkg/dsl/guard.go`).

| Function | Description | Example |
|----------|-------------|---------|
| `tokens('place')` | Token count at exactly that place | `tokens('goal')` |
| `sum('prefix')` | Total tokens across matching places, or the sum of a map's values | `sum('score')` |
| `count('prefix')` | Matching places holding at least one token, or map entries with a positive value | `count('inventory')` |
| `minOf('prefix')` | Smallest count among matching places, or smallest map value | `minOf('health')` |
| `maxOf('prefix')` | Largest count among matching places, or largest map value | `maxOf('score')` |

### Solver Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tspan` | `[number, number]` | `[0, 10]` | Simulation time span |
| `dt` | number | `0.01` | Initial time step |
| `rates` | object | all `1.0` | Per-transition firing rates |

## Minimal Example

A small model that satisfies the schema:

```json
{
  "name": "toggle",
  "places": [
    {"id": "off", "initial": 1},
    {"id": "on"}
  ],
  "transitions": [
    {"id": "switch"}
  ],
  "arcs": [
    {"from": "off", "to": "switch"},
    {"from": "switch", "to": "on"}
  ]
}
```

This defines a one-shot toggle: a token starts in `off`, the `switch` transition fires, and the token moves to `on`, where it stays because nothing consumes from `on`. The model uses only the four required fields. `on` takes the default `initial` of 0, and both arcs take the default weight of 1. The schema's minimums are one place, one transition and one arc, so a model with a single place and a single arc would also validate. This one was checked by reading it against the schema file and go-pflow's types, not by storing it.
