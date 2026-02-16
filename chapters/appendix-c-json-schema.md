# Appendix C: JSON Schema Reference

This appendix documents the JSON model format used by petri-pilot for code generation. The schema defines what a valid Petri net model looks like — places, transitions, arcs, and the higher-level structures (roles, views, navigation, admin, event sourcing, simulation) that drive application generation.

The full JSON Schema is published at `github.com/pflow-xyz/petri-pilot/schema/petri-model.schema.json`.

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
  "bindings": {
    "amount": "body.amount"
  }
}
```

| Field | Required | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `id` | Yes | string | — | Unique identifier, snake_case |
| `description` | No | string | — | Used in OpenAPI spec |
| `guard` | No | string | — | Boolean precondition (guard DSL) |
| `event_type` | No | string | PascalCase of id | Custom event type name |
| `http_method` | No | string | `"POST"` | `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `http_path` | No | string | `/api/{id}` | Custom HTTP path |
| `bindings` | No | object | — | Parameter bindings from request |

### Guard Syntax

Guards are boolean expressions evaluated against the current state:

| Operator | Meaning | Example |
|----------|---------|---------|
| `==`, `!=` | Equality | `status == 'approved'` |
| `<`, `>`, `<=`, `>=` | Comparison | `amount > 0` |
| `&&`, `\|\|`, `!` | Boolean | `a > 0 && b > 0` |
| `name[key]` | Map access | `balances[from]` |

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
| `weight` | No | integer | `1` | Tokens consumed/produced (min 1) |
| `keys` | No | array of strings | — | Map access keys for data places |
| `value` | No | string | `"amount"` | Value binding name |

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
      "icon": "📦",
      "roles": []
    },
    {
      "label": "Admin",
      "path": "/admin",
      "icon": "⚙️",
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

The `objective` field uses guard DSL syntax extended with aggregate functions:

| Function | Description | Example |
|----------|-------------|---------|
| `tokens('place')` | Token count at a place | `tokens('goal')` |
| `sum('place')` | Sum of values in a data place | `sum('score')` |
| `count('place')` | Count of entries in a map place | `count('inventory')` |
| `minOf('place')` | Minimum value | `minOf('health')` |
| `maxOf('place')` | Maximum value | `maxOf('score')` |

### Solver Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tspan` | `[number, number]` | `[0, 10]` | Simulation time span |
| `dt` | number | `0.01` | Initial time step |
| `rates` | object | all `1.0` | Per-transition firing rates |

## Minimal Example

The smallest valid model:

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

This defines a one-shot toggle: a token starts in `off`, the `switch` transition fires, and the token moves to `on`. Three fields, two places, one transition, two arcs — the simplest possible Petri net application.
