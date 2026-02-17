# Code Generation — From Model to Application

**Learning objective**: Generate full-stack applications from a Petri net model.

The visual editor (Chapter 15) produces a model. The Go library (Chapter 17) analyzes it. But at some point you need a running application — HTTP endpoints, event storage, a frontend, authentication. Petri-pilot bridges the gap: it takes a Petri net model and generates a complete, deployable application. The generation is deterministic. The same model always produces the same code.

This chapter explains the architecture: how a JSON model becomes a Go backend and JavaScript frontend, why events are separated from bindings, and how customization survives regeneration.

## The Pipeline

Petri-pilot's code generation follows a categorical pattern — a functor from model to code:

```
Schema --EnrichModel--> Context --Template--> Artifact
  |                        |                      |
Model                  Universal              Go/JS/YAML
(source)               Object                 (target)
```

Each step is deterministic. No randomness, no external state. Given the same model, you get the same code every time.

### Step 1: Parse

The model starts as a JSON file (or the tokenmodel DSL from Chapter 4). The schema parser produces a structured representation: places, transitions, arcs, roles, views, events.

```go
model := schema.Parse(jsonBytes)
```

### Step 2: Validate

Before generating code, petri-pilot validates the Petri net properties. Is the net well-formed? Are all arc sources and targets valid? Do guard expressions parse? This catches errors before they become generated code that doesn't compile.

```go
result := validator.Validate(model)
```

### Step 3: Build Context

The **Context** is the universal intermediate representation — the heart of the functor. It takes the flat model and computes everything the templates need:

```go
ctx := golang.NewContext(model, opts)
```

The Context contains:

1. **Primitives** — Places, Transitions, Arcs (from the Petri net)
2. **Derived structures** — Events, Routes, Handlers (computed from primitives)
3. **Feature flags** — `HasViews`, `HasAdmin`, `HasNavigation` (conditional generation)
4. **Helper methods** — `TransitionRequiresAuth`, `GetEnabledTransitions` (template utilities)

This is the key design decision: the Context is complete. Every piece of information needed by any template is accessible through it. Templates never reach back into the raw model.

### Step 4: Project to Artifacts

Each template file is a projection — a pure function from Context to source code:

```
Context --api.tmpl------> api.go
Context --workflow.tmpl-> workflow.go
Context --events.tmpl---> events.go
Context --main.tmpl-----> main.js
Context --admin.tmpl----> admin.js
```

Multiple projections from the same Context produce all the files needed for a running application.

## Inside-Out Design

Traditional code generation is "outside-in": templates consume ad-hoc data structures, and the templates encode most of the logic. Petri-pilot inverts this. The **model** contains everything needed to describe an application — state machine topology, access control, UI structure, operational config. Code is a projection of this model into a target language.

This means the model is readable by humans, LLMs, and formal analysis tools. The generated code is an implementation detail. You modify the model, not the code.

## Events First

The most important architectural pattern in petri-pilot is the separation of **events** from **bindings**.

### Events: The Complete Record

An event is the full business record emitted when a transition fires. It captures everything relevant to the domain:

```json
{
  "id": "tokens_transferred",
  "name": "Tokens Transferred",
  "fields": [
    {"name": "from", "type": "string", "required": true},
    {"name": "to", "type": "string", "required": true},
    {"name": "amount", "type": "number", "required": true},
    {"name": "memo", "type": "string"},
    {"name": "transaction_id", "type": "string"}
  ]
}
```

Events are complete, immutable, and auditable. They form the event log that can reconstruct any past state.

### Bindings: Operational Data

Bindings extract only what's needed for state computation — the minimal data for guards and arc transformations:

```json
{
  "id": "transfer",
  "event": "tokens_transferred",
  "guard": "balances[from] >= amount && amount > 0",
  "bindings": [
    {"name": "from", "type": "string", "keys": ["from"]},
    {"name": "to", "type": "string", "keys": ["to"]},
    {"name": "amount", "type": "number", "value": true}
  ]
}
```

The `tokens_transferred` event has five fields. The `transfer` binding uses three. The memo and transaction ID are preserved in the event for audit but don't participate in state computation.

### Why Separate Them?

The separation enables independent evolution. You can add fields to an event (new audit data, metadata) without changing the state logic. You can modify how bindings compute state without affecting the event schema. Events answer "what happened in the business domain?" Bindings answer "what data do I need to validate and update state?"

This maps directly to Petri net semantics. The event is the record of a transition firing. The binding is the data needed to check enabledness (guards) and compute the new marking (arc transformations).

### Event Names from Transitions

Event types are derived, not declared separately. A transition named `validate_order` produces an event type `OrderValidated`. A transition named `ship` produces event `Ship`. The event schema is the transition schema — no redundant definitions.

## Generated File Structure

A complete generated application contains:

### Backend (Go)

| File | Purpose |
|------|---------|
| `workflow.go` | Petri net definition — places, transitions, arcs |
| `aggregate.go` | Event-sourced aggregate — applies events to state |
| `api.go` | HTTP handlers — REST endpoints for each transition |
| `events.go` | Event types — structs for each event |
| `views.go` | View definitions — query handlers for UI |
| `auth.go` | Authentication — role-based access (if roles defined) |
| `main.go` | Entry point — wires everything together |

### Frontend (ES Modules)

| File | Purpose |
|------|---------|
| `src/main.js` | Entry point, routing |
| `src/admin.js` | Admin dashboard — instance listing, management |
| `src/views.js` | Instance views — detail pages, state display |
| `custom/extensions.js` | User customizations (preserved across regeneration) |
| `custom/theme.css` | Custom styling (preserved) |

The backend uses Go's standard library. The frontend uses vanilla ES modules — no React, no build step, no npm. Both are generated from the same Context.

## Guards as Expressions

Access control and preconditions use a DSL, not embedded code:

```json
{
  "guard": "balances[from] >= amount && from != to"
}
```

The guard expression is:

- **Readable** — both humans and LLMs understand it
- **Verifiable** — can be analyzed statically before code generation
- **Portable** — the same expression generates both Go and JavaScript code

The DSL parser (`pkg/dsl/parser.go`) handles arithmetic, comparisons, boolean operators, and map lookups. Guards are evaluated at runtime against the current state to determine transition enabledness — exactly the Petri net firing rule, extended to data-carrying nets.

## Feature Flags from Model Presence

Templates use conditional generation based on what the model contains:

```go
{{if .HasAdmin}}
// Generate admin dashboard code
{{end}}
```

The flag is computed from model presence:

```go
func (c *Context) HasAdmin() bool {
    return c.Admin != nil && c.Admin.Enabled
}
```

No configuration files. No feature toggles. If admin config exists in the model, admin code generates. If the model defines roles, auth code generates. If views are defined, view handlers generate. The model is the configuration.

## Customization Architecture

Generated code is regenerated when the model changes. But users need to customize behavior without losing changes. Petri-pilot solves this with a two-directory pattern:

```
frontend/
+-- src/              # REGENERATED -- core application code
|   +-- main.js
|   +-- admin.js
|   +-- views.js
+-- custom/           # PRESERVED -- user customizations
    +-- extensions.js # Hook registrations
    +-- components.js # Custom web components
    +-- theme.css     # Custom styling
```

Files in `src/` are overwritten on every generation. Files in `custom/` are created once (with sensible defaults) and never overwritten — the generator uses a `SkipIfExists` flag.

### Extension Points

Generated code defines hooks that custom code can register with:

```javascript
// custom/extensions.js
adminExtensions.customActions.push({
  label: 'Export JSON',
  className: 'btn btn-secondary',
  onClick: (instance) => {
    const blob = new Blob([JSON.stringify(instance, null, 2)])
    // download logic
  }
})
```

Available extension points include custom action buttons, custom table columns, state renderers, lifecycle callbacks (on create, delete, archive), and custom view sections. The generated code checks for registered extensions and invokes them at the appropriate points.

## The Serving Layer

When multiple generated services run together, petri-pilot provides a unified serving layer:

### Unified GraphQL API

All services implementing the `GraphQLService` interface are combined into a single schema at `/graphql`. Operations are namespaced by service — `blogpost_create`, `tictactoe_move`. A combined schema endpoint at `/schema` returns the full SDL.

### Model Viewer

The serving layer includes a Petri net viewer at `/pflow` that renders any model using the pflow.xyz `<petri-view>` web component (Chapter 15). It converts the internal model format to JSON-LD and passes it to the web component for interactive visualization and simulation.

### GraphQL Playground

An interactive playground at `/graphql/i` provides:

1. **Editor** — Standard GraphQL playground for writing queries
2. **Operations Explorer** — Parses the SDL to list all queries and mutations grouped by service
3. **Models Panel** — Displays each model's Petri net as SVG, with events, roles, places, and transitions

## The Complete Workflow

A typical petri-pilot workflow:

1. **Design** — Define the model as JSON or tokenmodel DSL
2. **Validate** — `petri_validate` checks structure and Petri net properties
3. **Simulate** — `petri_simulate` verifies workflow behavior before generating code
4. **Generate** — `petri_codegen` produces the full application
5. **Customize** — Edit `custom/extensions.js` for app-specific behavior
6. **Iterate** — When the model changes, regenerate. Customizations survive.

The model is the contract. Everything else — events, routes, handlers, views, admin panels — is derived. Change the model, regenerate, and the application reflects the new structure. The Petri net formalism makes this possible: because the model has formal semantics (places, transitions, firing rules), the code generator knows exactly what each element means and how to implement it.

> **Try it live:** Every generated application — [Tic-Tac-Toe](https://pilot.pflow.xyz/tic-tac-toe/), [Coffee Shop](https://pilot.pflow.xyz/coffeeshop/), [Texas Hold'em](https://pilot.pflow.xyz/texas-holdem/), [Knapsack](https://pilot.pflow.xyz/knapsack/) — is available at [pilot.pflow.xyz](https://pilot.pflow.xyz/).
