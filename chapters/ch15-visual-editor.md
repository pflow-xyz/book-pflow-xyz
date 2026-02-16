# The Visual Editor — pflow.xyz

**Learning objective**: Design nets visually, export to JSON, and iterate on models interactively.

Parts I through III approached Petri nets through code and mathematics — builder APIs, incidence matrices, ODE solvers. But most people think visually. A Petri net is inherently graphical: circles for places, rectangles for transitions, arrows for arcs. The pflow.xyz editor makes this visual intuition the primary interface. You draw a net, and the underlying formal structure — JSON-LD, topology, initial marking — is generated automatically.

This chapter walks through the editor: how to build nets interactively, how the visual representation maps to the mathematical formalism, and how models travel between the editor and the rest of the ecosystem.

## The Editor Interface

The pflow.xyz editor is a browser-based tool built as a single web component (`<petri-view>`) in vanilla JavaScript — no React, no build step, no npm dependencies. It renders in any modern browser and provides:

- **Canvas** — The main drawing area where places and transitions are positioned
- **Toolbar** — Controls for adding places, transitions, arcs, and tokens
- **Properties panel** — Edit selected element properties (name, initial tokens, arc weight)
- **Simulation controls** — Run discrete-event or continuous-time (ODE) simulation directly in the browser

The editor speaks JSON-LD natively. Every model uses `@context: "https://pflow.xyz/schema"` and `@type: "PetriNet"`, making it self-describing and interoperable with every tool in the ecosystem.

### The Web Component

The entire editor is a custom HTML element:

```html
<petri-view model='{"@context":"https://pflow.xyz/schema","@type":"PetriNet",...}'>
</petri-view>
```

This means the editor can be embedded in any web page — blog posts, documentation, interactive tutorials — by including the script and adding the element. The model attribute accepts a JSON-LD string, and the component handles rendering, editing, and simulation.

## Drawing Nets: Click-to-Place Workflow

Building a Petri net in the editor follows a direct manipulation pattern:

### Adding Elements

1. **Place**: Click the place tool, click the canvas. A circle appears with a default name (`P0`, `P1`, ...). Click to select, edit the name and initial token count in the properties panel.

2. **Transition**: Click the transition tool, click the canvas. A rectangle appears. Name it to describe the action it represents.

3. **Arc**: Click the arc tool, click a source element, click a target element. An arrow connects them. Arcs must alternate between places and transitions — the bipartite structure is enforced by the editor. You can't draw an arc from a place to a place.

4. **Tokens**: Set initial token counts in the properties panel, or click directly on a place to increment. The visual representation shows small dots inside places for low counts (1-5) and a number for higher counts.

### The Bipartite Constraint

The editor enforces the fundamental Petri net structure: arcs connect places to transitions or transitions to places, never same-type to same-type. This isn't a UI limitation — it's the definition of a Petri net. The bipartite graph structure is what makes the mathematical properties (firing rules, conservation laws, reachability) work.

If you try to draw an arc from one place to another, the editor rejects it. The constraint is structural, not a validation check — the data model simply doesn't support it.

### Weighted Arcs

Arc weights default to 1. For resource models like the coffee shop (Chapter 5), you need higher weights — an espresso requires 2 units of water and 1 unit of coffee beans. Select an arc and change its weight in the properties panel. The visual representation shows the weight as a number on the arc.

## From Visual to Formal

Every action in the editor produces a corresponding change in the JSON-LD model. The visual and formal representations are synchronized:

| Visual Action | Formal Effect |
|---------------|---------------|
| Add place "beans" with 100 tokens | `"beans": {"@type": "Place", "initial": [100]}` |
| Add transition "brew" | `"brew": {"@type": "Transition"}` |
| Draw arc from beans to brew, weight 2 | `{"@type": "Arrow", "source": "beans", "target": "brew", "weight": [2]}` |

The editor stores x/y coordinates for layout but these are presentation-only — they don't affect the mathematical properties. Two models with identical topology but different layouts are semantically equivalent.

### The JSON-LD Model

A complete model from the editor:

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "PetriNet",
  "places": {
    "beans": { "@type": "Place", "initial": [100], "x": 100, "y": 200 },
    "water": { "@type": "Place", "initial": [200], "x": 100, "y": 300 },
    "espresso": { "@type": "Place", "initial": [0], "x": 300, "y": 250 }
  },
  "transitions": {
    "brew": { "@type": "Transition", "x": 200, "y": 250 }
  },
  "arcs": [
    { "@type": "Arrow", "source": "beans", "target": "brew", "weight": [2] },
    { "@type": "Arrow", "source": "water", "target": "brew", "weight": [3] },
    { "@type": "Arrow", "source": "brew", "target": "espresso", "weight": [1] }
  ]
}
```

This file can be:
- Loaded back into the editor (visual round-trip)
- Parsed by go-pflow for ODE simulation (Chapter 3)
- Compiled by petri-pilot into a running service (Chapter 16)
- Sealed with a content-addressed CID (Chapter 14)
- Shared via URL

## In-Browser Simulation

The editor includes a JavaScript ODE solver (`petri-solver.js`) that matches the Go solver's output. This enables immediate feedback — draw a net, set rates, press play, watch token counts evolve.

### Discrete-Event Mode

Step through the model one transition at a time. The editor highlights enabled transitions (those with sufficient input tokens). Click a transition to fire it. The token counts update immediately. This is useful for:

- Understanding firing rules by example
- Tracing specific execution paths
- Debugging unexpected behavior ("why can't this transition fire?")

### Continuous-Time Mode

Run the ODE simulation forward. Token counts become continuous values — a place might hold 47.3 tokens. The editor displays a time-series plot showing how each place's token count evolves over the simulation period.

The continuous simulation uses the same Tsit5 solver as go-pflow, with the same mass-action kinetics. Models designed in the browser produce the same dynamics when loaded into the Go library. This is the dual-implementation guarantee covered in Chapter 18.

### Matching Go and JavaScript

The browser solver uses identical parameters to match the Go solver:

- Adaptive stepping with Tsit5
- Default dt=0.01, reltol=1e-3
- Time span configurable via controls

When a model behaves differently in the browser versus Go, it's a bug — the outputs should match. This invariant is tested and maintained across both implementations.

## Export Formats

The editor supports multiple export paths:

### JSON-LD

The native format. Includes all visual layout data (x/y positions) alongside the formal model. Used for interchange between all pflow ecosystem tools.

### SVG

The editor can export the current visual as an SVG file — useful for documentation, presentations, and embedding in papers. The SVG includes the graphical representation of places (circles), transitions (rectangles), arcs (arrows), and tokens (dots), preserving the exact visual appearance.

The Go backend also generates SVGs from JSON-LD models using layout algorithms (force-directed, hierarchical, circular), which is useful for automated documentation and reports.

### Content-Addressed Storage

When saved through the editor's API, models are canonicalized via URDNA2015 and hashed to produce a CIDv1 identifier (Chapter 14). The CID becomes a permanent, immutable reference:

```
POST /api/save  →  { "cid": "zb2rh..." }
GET /o/zb2rh...  →  { "@context": "https://pflow.xyz/schema", ... }
```

The CID is derived from the model's content, not from a counter or timestamp. Two editors that independently create the same model get the same CID. This enables deduplication and verifiable references.

## Sharing and Embedding Models

### URL Sharing

Every saved model has a URL: `https://pflow.xyz/o/{cid}`. Opening this URL loads the model in the editor. Because the CID is content-addressed, the URL is both a link and a guarantee — the model at that URL can never change.

### Embedding in Web Pages

The `<petri-view>` web component can be embedded in any HTML page:

```html
<script src="https://pflow.xyz/petri-view.js"></script>
<petri-view model='...'></petri-view>
```

This is how interactive models appear in the blog at blog.stackdump.com and in petri-pilot's model viewer. The embedded component supports the same simulation capabilities as the full editor.

### SVG Generation via API

The Go backend provides an SVG rendering endpoint:

```
GET /img/{cid}.svg
GET /img/{cid}.svg?layout=force-atlas-2
GET /img/{cid}.svg?layout=hierarchical
```

This generates publication-quality SVG diagrams from any stored model, with configurable layout algorithms. Useful for automated documentation and CI pipelines.

## The Editor as a Design Tool

The visual editor is where models typically begin. A domain expert sketches places and transitions — "there's a waiting room, a triage station, a doctor's office" — and connects them with arcs. The initial model is rough but captures the essential structure.

Then iteration: add initial tokens, set arc weights, run a simulation, observe the dynamics. "The tokens pile up before the doctor — that's the bottleneck." Adjust rates, add a second doctor transition, simulate again. The visual feedback loop is faster than editing code because the model is visible.

Once the model stabilizes, it travels through the ecosystem:
1. **Editor** — design and iterate visually
2. **go-pflow** — run formal analysis (reachability, P-invariants, sensitivity)
3. **petri-pilot** — generate a running application
4. **ZK circuit** — add privacy with zero-knowledge proofs

The editor is the entry point. The formalism behind it makes everything else possible. The circles and rectangles on screen are the same places and transitions that appear in incidence matrices, ODE systems, and ZK circuits. The visual representation is sugar. The mathematics is the substance.
