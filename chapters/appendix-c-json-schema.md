# Appendix C: JSON Schema Reference

This appendix documents the JSON model format used by petri-pilot for code generation. The schema defines what a valid Petri net model looks like — places, transitions, arcs, and the higher-level structures (roles, views, navigation, admin, event sourcing, simulation) that drive application generation.

The full JSON Schema is [`schema/petri-model.schema.json`](https://github.com/pflow-xyz/petri-pilot/blob/main/schema/petri-model.schema.json) in petri-pilot.

The schema file describes the format, but the code that reads a model decides what is accepted. petri-pilot's CLI decodes a model into go-pflow's `metamodel.Model` ([`metamodel/schema.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/schema.go), at v0.33.0, the version petri-pilot's `go.mod` pins), plus the extension fields of `modelWithExtensions` in [`cmd/petri-pilot/main.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/cmd/petri-pilot/main.go): `admin`, `navigation`, `roles`, `access`, `views`, `debug` and `graphql`. Where the schema file and that loader disagree, this appendix follows the loader and says so. The disagreements are transition `bindings` and `rate`, arc `type` and `kinetic`, and `eventSourcing`. They were found by reading the Go types against the schema file. No model was run through the CLI to confirm them.

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
  "roles": [...],
  "access": [...],
  "views": [...],
  "navigation": {...},
  "admin": {...},
  "eventSourcing": {...},
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
| `roles` | No | array | Named roles for access control |
| `access` | No | array | Access control rules |
| `views` | No | array | UI view definitions |
| `navigation` | No | object | Navigation menu configuration |
| `admin` | No | object | Admin dashboard configuration |
| `eventSourcing` | No | object | Snapshot and retention configuration |
| `simulation` | No | object | ODE simulation configuration |

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

A transition is an action that fires when enabled. Each transition becomes an HTTP endpoint in the generated application.

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
| `description` | No | string | — | Used in OpenAPI spec |
| `guard` | No | string | — | Boolean precondition (guard DSL) |
| `event_type` | No | string | PascalCase of id | Custom event type name (go-pflow marks it deprecated) |
| `http_method` | No | string | `"POST"` | `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `http_path` | No | string | `/api/{id}` | Custom HTTP path |
| `bindings` | No | array of objects | — | Operational data the transition reads or writes |
| `rate` | No | number | — | Firing rate for ODE simulation (loader only; not in the schema file) |

The schema file is stale on `bindings`: it describes the field as an object mapping names to strings. The loader's `Transition.Bindings` is an array of `Binding` objects with fields `name`, `type`, `keys` (map access path), `value` (true for the transferred amount) and `place`. Go's JSON decoder does not turn an object into an array, so by the types an object-form `bindings` fails to parse. The old map form survives as the separate field `legacy_bindings`. The binding in the example above is copied from pflow-xyz's [`extend-operations.json`](https://github.com/pflow-xyz/pflow-xyz/blob/main/examples/showcase/fixtures/extend-operations.json) fixture.

### Guard Syntax

Guards are boolean expressions evaluated against the current state. The lexer is petri-pilot's [`pkg/dsl/lexer.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/dsl/lexer.go); strings may use single or double quotes.

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
| `type` | No | `"inhibitor"` or `"read"` | normal | Arc kind (loader only; not in the schema file) |
| `kinetic` | No | boolean | `true` | Whether an input place scales the ODE firing rate (loader only) |
| `keys` | No | array of strings | — | Map access keys for data places |
| `value` | No | string | `"amount"` | Value binding name |

The schema file has no `type` field, but go-pflow's `Arc` does, and petri-pilot's own [`services/vet-clinic.json`](https://github.com/pflow-xyz/petri-pilot/blob/main/services/vet-clinic.json) uses both kinds, for example `{"from": "wait_emergency", "to": "start_wellness", "type": "inhibitor"}`. Neither kind moves tokens. An inhibitor arc blocks the transition while its place holds at least `weight` tokens. A read arc allows the transition only while its place holds at least `weight` tokens. These are the contextual arcs of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries), outside the incidence matrix. Setting `kinetic: false` keeps an input arc's enabling and consumption but removes its place from the rate product. go-pflow's [`metamodel/validation.go`](https://github.com/pflow-xyz/go-pflow/blob/v0.33.0/metamodel/validation.go) rejects three things here: an unknown `type`, which is not quietly run as a normal arc; a read arc that does not run place → transition; and `kinetic: false` on any arc other than a consuming place → transition arc.

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

## Roles

Named roles for access control, with inheritance.

```json
{
  "id": "admin",
  "name": "Administrator",
  "description": "Full access to all operations",
  "inherits": ["manager"]
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `id` | Yes | string | Unique role identifier, snake_case |
| `name` | No | string | Human-readable name |
| `description` | No | string | What this role represents |
| `inherits` | No | array of strings | Parent role IDs (inherits their permissions) |

## Access Rules

Maps transitions to allowed roles.

```json
{
  "transition": "approve_order",
  "roles": ["manager", "admin"],
  "guard": "user.department == 'finance'"
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `transition` | Yes | string | Transition ID or `"*"` for all |
| `roles` | No | array of strings | Allowed roles (empty = any authenticated user) |
| `guard` | No | string | Additional guard expression |

## Views

UI view definitions for forms, tables, and detail pages.

```json
{
  "id": "order_detail",
  "name": "Order Details",
  "kind": "detail",
  "description": "Shows order information",
  "groups": [...],
  "actions": ["approve", "reject"]
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `id` | Yes | string | Unique view identifier |
| `name` | No | string | Display name |
| `kind` | No | string | `"form"`, `"card"`, `"table"`, or `"detail"` |
| `description` | No | string | What this view shows |
| `groups` | No | array | Logical groupings of fields |
| `actions` | No | array of strings | Transition IDs triggerable from this view |

### View Groups

```json
{
  "id": "customer_info",
  "name": "Customer Information",
  "fields": [...]
}
```

### View Fields

```json
{
  "binding": "customer_email",
  "label": "Email Address",
  "type": "email",
  "required": true,
  "readonly": false,
  "placeholder": "customer@example.com"
}
```

Field types: `text`, `number`, `email`, `date`, `datetime`, `select`, `checkbox`, `textarea`, `password`, `hidden`.

## Navigation

Navigation menu configuration. Generates `/api/navigation` endpoint.

```json
{
  "brand": "Order Tracker",
  "items": [
    {
      "label": "Orders",
      "path": "/orders",
      "icon": "box",
      "roles": []
    },
    {
      "label": "Admin",
      "path": "/admin",
      "icon": "gear",
      "roles": ["admin"]
    }
  ]
}
```

## Admin

Admin dashboard configuration. Generates `/admin/*` endpoints.

```json
{
  "enabled": true,
  "path": "/admin",
  "roles": ["admin"],
  "features": ["list", "detail", "history", "transitions"]
}
```

Features: `list` (instance listing), `detail` (instance detail page), `history` (event history), `transitions` (manual transition firing).

## Event Sourcing

Snapshot and retention configuration.

```json
{
  "snapshots": {
    "enabled": true,
    "frequency": 100
  },
  "retention": {
    "events": "90d",
    "snapshots": "1y"
  }
}
```

Retention durations use the pattern `^\d+[dwmy]$` — number followed by `d` (days), `w` (weeks), `m` (months), or `y` (years).

The schema file defines this block, but the loader does not read it: neither go-pflow's `Model` nor `modelWithExtensions` has an `eventSourcing` field, so Go's decoder drops it when a model file is read. In the Go generator, `buildEventSourcingContext` in [`pkg/codegen/golang/context.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/context.go) is defined but has no caller. Treat the block as a declared format, not yet a working feature.

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

The `objective` field is a numeric guard-DSL expression, using arithmetic and the aggregate functions below. The same functions are available to guards and constraints. Except for `tokens`, each function takes a place-ID **prefix**: over token places it ranges over every place whose ID starts with the argument, and over a map data place of that name it ranges over the map's values. The definitions are `MakeAggregates` and `MakeAggregatesWithData` in [`pkg/dsl/guard.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/dsl/guard.go).

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

This defines a one-shot toggle: a token starts in `off`, the `switch` transition fires, and the token moves to `on`, where it stays because nothing consumes from `on`. The model uses only the four required fields. `on` takes the default `initial` of 0, and both arcs take the default weight of 1. The schema's minimums are one place, one transition and one arc, so a model with a single place and a single arc would also validate. This one was checked by reading it against the schema file, not by running it through petri-pilot's validator.
