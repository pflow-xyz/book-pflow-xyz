# Code Generation — From Model to Application

**Learning objective**: Generate full-stack applications from a Petri net model.

The visual editor (Chapter 17) produces a model. The Go library (Chapter 19) analyzes it. But at some point you need a running application — HTTP endpoints, event storage, a frontend, authentication. [Petri-pilot](https://github.com/pflow-xyz/petri-pilot) bridges the gap: it takes a Petri net model and generates a deployable application from it.

This chapter explains the architecture: how a JSON model becomes a Go backend and JavaScript frontend, why events are separated from bindings, and how customization survives regeneration. The code excerpts are taken from petri-pilot's source, abridged and with error handling omitted; the design document they come from is [ARCHITECTURE.md](https://github.com/pflow-xyz/petri-pilot/blob/main/ARCHITECTURE.md).

## The Pipeline

The pipeline has one intermediate stage between the model and the generated files:

```
Model --EnrichModel, NewContext--> Context --template--> Artifact
(JSON or DSL)                     (intermediate        (Go / JS / YAML)
                                   representation)
```

Petri-pilot's own documentation calls this a functor from models to code. Read that as a description of the shape — every artifact is computed from the model, and nothing else feeds in — rather than as a checked categorical result; no functor laws are stated or tested.

The templates are functions of the Context alone, so the same model and the same generator options should produce the same files. The parts most likely to break that, such as the suffixes that disambiguate colliding identifiers, have their own tests (`TestIdentScopeIsDeterministic` in [`pkg/codegen/golang/flat_ids_test.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/flat_ids_test.go)). A byte-for-byte regeneration test exists for petri-pilot's single-file `core` generator ([`pkg/codegen/core/core_test.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/core/core_test.go), `TestGenerateDeterministic`), but not yet for the full application generator this chapter describes.

### Step 1: Parse

The model is a go-pflow `metamodel.Model`: places, transitions, arcs, roles, views, events. It is read from JSON, or from the tokenmodel S-expression DSL of Chapter 4 — petri-pilot's MCP tools accept either and treat input that starts with `(` as DSL (`parseModelV2` in [`pkg/mcp/server.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/mcp/server.go)).

```go
var model metamodel.Model // github.com/pflow-xyz/go-pflow/metamodel
err := json.Unmarshal(jsonBytes, &model)
```

### Step 2: Validate

Before generating code, petri-pilot checks the net. The structural pass rejects a model with no places or no transitions, an element with no arcs, and an arc whose source or target does not exist. The validator then builds a go-pflow net from the model and runs reachability analysis, warning when it finds deadlock states ([`pkg/validator/validator.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/validator/validator.go)).

```go
result, err := validator.New(validator.DefaultOptions()).Validate(&model)
```

Guard expressions are not parsed at this stage; see [Guards as Expressions](#guards-as-expressions) below.

### Step 3: Build Context

The **Context** is the intermediate representation. `NewContext` first enriches the model with defaults (`metamodel.EnrichModel`), then computes everything the templates need ([`pkg/codegen/golang/context.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/context.go)):

```go
ctx, err := golang.NewContext(&model, opts)
```

The Context contains:

1. **Primitives** — Places, Transitions, Arcs (from the Petri net)
2. **Derived structures** — Events, Routes, Handlers (computed from primitives)
3. **Feature flags** — `HasViews`, `HasAdmin`, `HasNavigation` (conditional generation)
4. **Helper methods** — `TransitionRequiresAuth`, `HasAccessControl` (template utilities)

The design intent is that the Context is complete: every piece of information a template needs is reachable through it, so templates do not reach back into the raw model. The frontend generator has its own Context type ([`pkg/codegen/esmodules/context.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/esmodules/context.go)), built from the same model.

### Step 4: Project to Artifacts

Each template file is a projection from a Context to one source file:

```
Go Context --api.tmpl------> api.go
Go Context --workflow.tmpl-> workflow.go
Go Context --events.tmpl---> events.go
JS Context --main.tmpl-----> src/main.js
JS Context --admin.tmpl----> src/admin.js
```

The backend templates live in [`pkg/codegen/golang/templates/`](https://github.com/pflow-xyz/petri-pilot/tree/main/pkg/codegen/golang/templates) and the frontend templates in [`pkg/codegen/esmodules/templates/`](https://github.com/pflow-xyz/petri-pilot/tree/main/pkg/codegen/esmodules/templates).

## Inside-Out Design

Traditional code generation is "outside-in": templates consume ad-hoc data structures, and the templates encode most of the logic. Petri-pilot inverts this. The **model** contains everything needed to describe an application — state machine topology, access control, UI structure, operational config. Code is a projection of this model into a target language.

This means the model is readable by humans, LLMs, and formal analysis tools. The generated code is an implementation detail. You modify the model, not the code.

## Events First

A central pattern in petri-pilot is the separation of **events** from **bindings**. The examples in this section are the ERC-20 transfer from petri-pilot's [events-first design note](https://github.com/pflow-xyz/petri-pilot/blob/main/docs/events-first-pattern.md); the same guard and keyed arcs appear in the runnable token model [`services/erc20.pflow`](https://github.com/pflow-xyz/petri-pilot/blob/main/services/erc20.pflow).

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

Events are appended to the instance's event log and never modified, so any past state can be rebuilt by replaying the log.

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

A transition does not have to declare its event type. When it names none, the generator derives one by putting the transition's ID in PascalCase and adding a past-tense suffix (`eventTypeFromStem` in [`pkg/codegen/golang/identifiers.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/identifiers.go)): `validate` becomes `Validated`. The rule only appends `d` or `ed`, so `ship` becomes `Shiped` and `validate_order` becomes `ValidateOrdered`; a model that wants a better name declares it. Several transitions can share one declared event — the nine `x_play_*` transitions in [`services/tic-tac-toe.json`](https://github.com/pflow-xyz/petri-pilot/blob/main/services/tic-tac-toe.json) all emit `XPlayed`, and the generator emits one struct for it.

## Generated File Structure

The core files of a generated application are listed below; the full template-to-file maps are in `templates.go` in each generator package.

### Backend (Go)

| File | Purpose |
|------|---------|
| `workflow.go` | Petri net definition — places, transitions, arcs |
| `aggregate.go` | Event-sourced aggregate — applies events to state |
| `api.go` | HTTP handlers — REST endpoints for each transition |
| `events.go` | Event types — structs for each event |
| `openapi.yaml` | API description |
| `views.go` | View definitions — query handlers for UI (if views defined) |
| `auth.go` | Authentication — role-based access (if roles or access rules defined) |
| `main.go` or `service.go` | Entry point for a standalone binary, or a service registered with `petri-pilot serve` |

### Frontend (ES Modules)

| File | Purpose |
|------|---------|
| `index.html` | Loads `src/main.js` as a module |
| `src/main.js` | Entry point |
| `src/admin.js` | Admin dashboard — instance listing, management |
| `src/views.js` | Instance views — detail pages, state display |
| `custom/extensions.js` | User customizations (preserved across regeneration) |
| `custom/theme.css` | Custom styling (preserved) |

The backend imports petri-pilot's runtime packages and a short list of modules pinned in [`go_mod.tmpl`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/templates/go_mod.tmpl), among them SQLite, WebSocket, Prometheus and OAuth2 clients. The frontend is vanilla ES modules with no framework. The generator also writes a `package.json` whose only dependency is Vite, as a development server and bundler. Both halves are generated from the same model, through their separate Context types.

## Guards as Expressions

Access control and preconditions use a DSL, not embedded code:

```json
{
  "guard": "balances[from] >= amount && from != to"
}
```

The guard stays a string all the way into the generated code. The generator copies it into `aggregate.go` as an argument to `dsl.Evaluate`, which parses and evaluates it against the request's bindings and, for guards that read token counts such as `tokens("stock") > 0`, the current marking ([`pkg/dsl/`](https://github.com/pflow-xyz/petri-pilot/tree/main/pkg/dsl); the generated call is checked in `pkg/codegen/golang/marking_guard_test.go`). The DSL handles arithmetic, comparisons, boolean operators, map lookups, field access and function calls. A guard adds a data condition to the Petri net firing rule: the transition fires only when its input places are marked and its guard holds.

Because the guard is evaluated at runtime, a guard that does not parse is reported when the transition is attempted, not by the validator in Step 2. Keeping it as data has other uses: it is readable in the model by people and LLMs, and the generated frontend lists each transition's guard in its model view. The frontend displays guards but does not evaluate them; the server decides enablement.

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

The application's features follow from what the model declares: views produce `views.go`, navigation produces `navigation.go`, an enabled admin block produces `admin.go`, and roles or access rules produce `auth.go`, `middleware.go` and `permissions.go` ([`pkg/codegen/golang/generator.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/golang/generator.go)). Concerns outside the domain — tests, infrastructure and deployment files, observability, realtime updates, simulation endpoints — are switched by generator options instead.

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

Files in `src/` are overwritten on every generation. Files in `custom/` are created once, with default content, and never overwritten: their entries in the frontend's template map carry a `SkipIfExists` flag ([`pkg/codegen/esmodules/templates.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/esmodules/templates.go)).

### Extension Points

Generated code defines hooks that custom code can register with. This example ships, commented out, in the generated `custom/extensions.js` ([`extensions.tmpl`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/codegen/esmodules/templates/extensions.tmpl)):

```javascript
// custom/extensions.js
adminExtensions.customActions.push({
  label: 'Export JSON',
  className: 'btn btn-secondary',
  onClick: (instance) => {
    const json = JSON.stringify(instance, null, 2)
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${instance.id}.json`
    a.click()
  }
})
```

The generated `src/admin.js` imports `custom/extensions.js`, renders any registered custom actions as buttons, and fires the archive, restore and delete lifecycle hooks. The file also declares slots for custom table columns, state renderers, custom view sections and created/transition-executed hooks, but no generated module invokes those yet, so registering one currently has no effect.

## The Serving Layer

When multiple generated services run together, petri-pilot provides a unified serving layer:

### Unified GraphQL API

All services implementing the `GraphQLService` interface are combined into a single schema at `/graphql`. Mutations are prefixed with the service name: instance creation on `blogpost` becomes `blogpost_create`, and a `transfer` transition on `erc20-token` becomes `erc20token_transfer` ([`pkg/serve/graphql.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/serve/graphql.go)). A combined schema endpoint at `/schema` returns the full SDL.

### Model Viewer

The serving layer includes a Petri net viewer at `/pflow` that renders each served model using the pflow.xyz `<petri-view>` web component (Chapter 17). It converts the internal model format to JSON-LD and passes it to the web component for interactive visualization and simulation.

### GraphQL Playground

An interactive playground at `/graphql/i` provides:

1. **Editor** — Standard GraphQL playground for writing queries
2. **Operations Explorer** — Parses the SDL to list all queries and mutations grouped by service
3. **Models Panel** — Displays each model's Petri net as SVG, with events, roles, places, and transitions

## The Complete Workflow

A typical petri-pilot workflow:

1. **Design** — Define the model as JSON or tokenmodel DSL
2. **Validate** — `petri_validate` checks structure and Petri net properties
3. **Simulate** — `petri_simulate` runs the model before any code is generated
4. **Generate** — `petri_codegen` produces the application
5. **Customize** — Edit `custom/extensions.js` for app-specific behavior
6. **Iterate** — When the model changes, regenerate. Files in `custom/` are not overwritten.

The model is the contract. Events, routes, handlers, views and admin panels are derived from it, so changing the model and regenerating changes the application's structure. Customizations are kept but not updated: a hook in `custom/extensions.js` that refers to a transition the new model removed is left as it was, and finding it is the developer's job.

> **Try it live:** Every generated application — [Tic-Tac-Toe](https://pilot.pflow.xyz/tic-tac-toe/), [Coffee Shop](https://pilot.pflow.xyz/coffeeshop/), [Texas Hold'em](https://pilot.pflow.xyz/texas-holdem/), [Knapsack](https://pilot.pflow.xyz/knapsack/) — is available at [pilot.pflow.xyz](https://pilot.pflow.xyz/).
