# The Visual Editor — pflow.xyz

**Learning objective**: Design nets visually, export to JSON, and iterate on models interactively.

Parts I through III approached Petri nets through code and mathematics: builder APIs, incidence matrices, ODE solvers. A Petri net also has a standard drawing, with circles for places, rectangles for transitions and arrows for arcs, and the pflow.xyz editor makes that drawing the primary interface. You draw a net, and the editor keeps the JSON-LD document behind it (topology, initial marking, layout) in step with the canvas.

This chapter walks through the editor: how to build nets interactively, how the visual representation maps to the mathematical formalism, and how models travel between the editor and the rest of the ecosystem.

## The Editor Interface

The pflow.xyz editor is a custom HTML element, `<petri-view>`, defined in [`public/petri-view.js`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/petri-view.js) together with a few sibling ES modules: the firing rules in `petri-sim.js`, the ODE solver in `petri-solver.js`, CID computation in `seal-cid.mjs`. It is plain JavaScript with no framework and no build step; its one third-party library, `jsonld`, is vendored as a pre-built bundle. It provides:

- **Toolbar** — six editing modes (select, place, transition, arc, token, delete; keys 1–6), a label-edit toggle (7), and a play button that runs the discrete simulation (8 or X)
- **Canvas** — the drawing area, with pan, zoom, multi-select and undo/redo
- **JSON editor** — an optional panel showing the model as JSON-LD, editable in place
- **Analysis workbench** — continuous-time (ODE) simulation, rate scans, sweeps and phase plots, opened from the menu or with key 9

Models are JSON-LD documents with `"@context": "https://pflow.xyz/schema"` and `"@type": "PetriNet"`. If a loaded document lacks them, the editor fills them in, along with `@version` and a default `token` list.

### The Web Component

The entire editor is a custom HTML element. A page loads the module and its stylesheet, and gives the element its model as a child JSON-LD script:

```html
<link rel="stylesheet" href="https://pflow.xyz/petri-view.css"/>
<script type="module" src="https://pflow.xyz/petri-view.js"></script>

<petri-view data-json-editor>
  <script type="application/ld+json">
  {"@context": "https://pflow.xyz/schema", "@type": "PetriNet", ...}
  </script>
</petri-view>
```

The component reads that script when it connects to the page and writes every edit back into it, so the page always holds the current document; it also autosaves to the browser's local storage. Scripts can call `getModel()`, `setModel()` and `importJSON()`, and listen for `transition-fired-success` and `marking-changed` events (see the [pflow-xyz README](https://github.com/pflow-xyz/pflow-xyz#events--api)). Because it is an ordinary element, the editor can be embedded in any web page: documentation, tutorials, other applications.

## Drawing Nets: Click-to-Place Workflow

Building a Petri net in the editor follows a direct manipulation pattern:

### Adding Elements

1. **Place**: Choose place mode (key 2) and click the canvas. A circle appears with a generated id, `p` followed by a base-36 timestamp. Label mode (key 7) gives it a readable `label`.

2. **Transition**: Choose transition mode (key 3) and click the canvas. A rectangle appears, with an id beginning `t`. Label it with the action it represents.

3. **Arc**: Choose arc mode (key 4), click a source, then click a target. The editor asks for a weight (default 1) and adds the arc. Right-clicking the target, or long-pressing it on a touch screen, makes an inhibitor arc instead: one that disables its transition while the place holds at least the arc's weight in tokens. Inhibitor arcs are contextual arcs; [Appendix E: Categorical Foundations](appendix-e-categorical-foundations.md) explains why they sit outside the algebra the rest of the net obeys.

4. **Tokens**: In token mode (key 5), click a place to add or remove tokens; right-clicking a place does the same. A place shows a single dot when it holds one token and a numeral when it holds two or more.

### The Bipartite Constraint

The editor enforces the fundamental Petri net structure: an arc connects a place to a transition or a transition to a place, never two nodes of the same kind. This is the definition of a Petri net rather than a UI preference. The firing rule, the conservation laws and reachability analysis all assume it.

If, in arc mode, you click a place and then another place, the editor flashes both and discards the draft (`_arcNodeClicked` in `petri-view.js`). The check runs when you draw. The file format does not encode the restriction: an arc in the JSON is a `source` id and a `target` id, and a hand-edited document could name two places.

### Weighted Arcs

Arc weights default to 1. Resource models like the coffee shop (Chapter 5) need higher weights: there, an espresso consumes 18 g of beans and 30 ml of water. Set the weight when the editor prompts for it, or right-click an existing arc to change it. The weight appears as a badge on the arc; by default the badges are hidden while every arc in the net has weight 1.

## From Visual to Formal

Every action in the editor produces a corresponding change in the JSON-LD model. The visual and formal representations are synchronized:

| Visual Action | Formal Effect |
|---------------|---------------|
| Click the canvas in place mode | A new entry in `places`: `{"@type": "Place", "initial": [0], "x": …, "y": …}`, with an unbounded `capacity` |
| Click the canvas in transition mode | A new entry in `transitions`: `{"@type": "Transition", "x": …, "y": …}` |
| Draw an arc from beans to brew, weight 18 | `{"@type": "Arrow", "source": "beans", "target": "brew", "weight": [18], "inhibitTransition": false}` |

Counts and weights are arrays because a net can carry several token colours, one array entry per colour in the `token` list; a single-colour net uses one-element arrays.

The editor stores x/y coordinates for layout. They do not affect firing or the ODE, so two models with identical topology but different layouts behave identically. They are still different documents, and they get different CIDs (see Content-Addressed Storage below).

### The JSON-LD Model

A small model in the editor's format, with readable ids and the espresso weights from Chapter 5. It is trimmed by hand: a file saved by the editor also carries `capacity` and `offset` on every place. For a file exactly as saved, see [`parity/fixtures/coffee-shop.jsonld`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/fixtures/coffee-shop.jsonld), a different, three-colour coffee-shop net.

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "PetriNet",
  "@version": "1.1",
  "token": ["https://pflow.xyz/tokens/black"],
  "places": {
    "beans": { "@type": "Place", "initial": [1000], "x": 100, "y": 200 },
    "water": { "@type": "Place", "initial": [10000], "x": 100, "y": 300 },
    "espresso": { "@type": "Place", "initial": [0], "x": 300, "y": 250 }
  },
  "transitions": {
    "brew": { "@type": "Transition", "x": 200, "y": 250 }
  },
  "arcs": [
    { "@type": "Arrow", "source": "beans", "target": "brew", "weight": [18], "inhibitTransition": false },
    { "@type": "Arrow", "source": "water", "target": "brew", "weight": [30], "inhibitTransition": false },
    { "@type": "Arrow", "source": "brew", "target": "espresso", "weight": [1], "inhibitTransition": false }
  ]
}
```

This file can be:
- Loaded back into the editor (visual round-trip)
- Parsed by go-pflow's [`parser`](https://github.com/pflow-xyz/go-pflow/blob/main/parser/json.go) for ODE simulation (Chapter 3)
- Read by petri-pilot, which recognizes the editor's shape and converts it with go-pflow's [`parser.ModelFromJSON`](https://github.com/pflow-xyz/go-pflow/blob/main/parser/model.go), and generated into a running service (Chapter 18)
- Sealed with a content-addressed CID (Chapter 16)
- Shared via URL

## In-Browser Simulation

The editor runs both simulations in the browser, so you can draw a net, press play, and watch the marking change.

### Discrete-Event Mode

Step through the model one transition at a time. The editor fills enabled transitions (those with sufficient input tokens) green; click one to fire it, and the token counts update immediately. The play button fires enabled transitions automatically. This is useful for:

- Understanding firing rules by example
- Tracing specific execution paths
- Debugging unexpected behavior ("why can't this transition fire?")

### Continuous-Time Mode

The Analysis workbench integrates the ODE forward. Token counts become continuous values, so a place might hold 47.3 tokens. Its Simulate tab plots how each place's count evolves over the time span, alongside a table of initial and final values; the other tabs scan one rate, overlay trajectories across rate values, and plot one place against another.

The workbench uses the same Tsit5 solver as go-pflow, with the same mass-action kinetics: each transition's flux is its rate times the product of its input places' token counts, and arc weights scale how much is consumed (Chapter 3). A model designed in the browser therefore has the same dynamics when loaded into the Go library. Chapter 20 covers this dual-implementation requirement in general.

### Matching Go and JavaScript

The browser solver ([`public/petri-solver.js`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/petri-solver.js)) uses the same defaults as go-pflow's `DefaultOptions` ([`solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go)):

- Adaptive stepping with Tsit5
- dt = 0.01, dtmax = 0.1, abstol = 1e-6, reltol = 1e-3
- Time span configurable via controls

A model that behaves differently in the browser and in Go is a bug, and two tests in the pflow-xyz repository exist to catch one. [`parity/ode/ode_parity_test.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/ode/ode_parity_test.go) integrates a fixed set of fixture models with both solvers and requires the final states to agree within 1e-6 per place; the file records the measured agreement as about 1e-15, because the two implementations take the same steps. [`parity/sim/sim_parity_test.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/sim/sim_parity_test.go) does the same for discrete firing: it walks 200 seeded random models through `petri-sim.js` and go-pflow's reachability engine in lockstep and compares the enabled sets and markings at every step. Both run under `make test-parity-behavior` and need `node`; they were not re-run for this chapter.

## Export Formats

The editor supports multiple export paths:

### JSON-LD

The native format. The menu's Download JSON item saves the model, including its x/y layout, as a file named after its CID. JSON-LD is the interchange format between all pflow ecosystem tools.

### SVG

SVG comes from the Go backend rather than from the editor's canvas. `GET /img/{cid}.svg` renders any stored model, and `POST /api/svg` renders a posted document; the renderer is [`internal/svg/svg.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/internal/svg/svg.go). Without a `layout` parameter it draws the model at its stored coordinates. With one, it rearranges the net first, using the same algorithms the editor's Layout Algorithms menu applies to the canvas: `circular`, `sugiyama` (layered), `grid`, `bipartite` and `string-diagram`. The editor's own SVG download is the one inside its String Diagram view.

### Content-Addressed Storage

When you save to the pflow.xyz server (which requires a GitHub login), the server canonicalizes the model with URDNA2015 and hashes the result into a CIDv1: dag-json codec, SHA-256, base58btc, so every CID begins with `z` (Chapter 16):

```
POST /api/save   ->  { "cid": "z4EBG9j..." }
GET  /o/{cid}    ->  { "@context": "https://pflow.xyz/schema", ... }
```

The editor computes the same CID in the browser with `seal-cid.mjs`; that is how Download JSON names its file. The Go and JavaScript sides are held to byte-identical CIDs by [`parity/golden.json`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/golden.json), five fixtures with their expected CIDs, which `make test-parity` checks from both languages.

The CID is derived from the model's content, not from a counter or timestamp, so two people who produce the same document get the same CID. "The same document" is strict. The coordinates and labels are part of the sealed content, so moving a node changes the CID even though it changes nothing about how the net behaves. Within that limit, the CID supports deduplication and verifiable references.

## Sharing and Embedding Models

### URL Sharing

A saved model opens in the editor at `https://pflow.xyz/?cid={cid}`; `GET /o/{cid}` returns the stored JSON-LD itself. Because the identifier is content-addressed, the document behind a CID cannot change. It can disappear: the author of a saved model can delete it (`DELETE /o/{cid}`). Without logging in, Save Permalink writes the model into the page URL instead of storing it on the server.

### Embedding in Web Pages

The `<petri-view>` element shown earlier can be embedded in any HTML page. An embedded element is the same component as the full editor, with the same simulation; server saving and GitHub login appear only when the element carries the `data-backend` attribute. Petri-pilot's model viewer at `/pflow` is built this way (Chapter 18).

Blog posts at blog.stackdump.com link to models instead of embedding the component. The editor's Share dialog produces the snippet they use: the server-rendered SVG wrapped in a link to the editor.

```markdown
[![pflow](https://pflow.xyz/img/{cid}.svg)](https://pflow.xyz/?cid={cid})
```

### SVG Generation via API

The rendering endpoint takes the stored model's CID and an optional layout:

```
GET /img/{cid}.svg
GET /img/{cid}.svg?layout=circular
GET /img/{cid}.svg?layout=sugiyama
```

An unrecognized layout name is logged and the model is drawn at its stored positions. The server marks each response cacheable for a year, which suits generated documentation and CI pipelines: the model behind a CID does not change.

## The Editor as a Design Tool

The visual editor is where models typically begin. A domain expert sketches places and transitions — "there's a waiting room, a triage station, a doctor's office" — and connects them with arcs. The initial model is rough but captures the essential structure.

Then iteration: add initial tokens, set arc weights, run a simulation, observe the dynamics. "The tokens pile up before the doctor — that's the bottleneck." Adjust rates, add a second doctor transition, simulate again. Because the model is on screen while it runs, this loop can be quicker than editing code.

Once the model stabilizes, it travels through the ecosystem:
1. **Editor** — design and iterate visually
2. **go-pflow** — run formal analysis (reachability, P-invariants, sensitivity)
3. **petri-pilot** — generate a running application
4. **ZK circuit** — add privacy with zero-knowledge proofs

The circles and rectangles on screen are the same places and transitions that appear in incidence matrices, ODE systems and ZK circuits. The drawing is one view of a document that the other tools read directly.

> **Try it live:** Design your own Petri net at [pflow.xyz](https://pflow.xyz/), then see generated applications at [pilot.pflow.xyz](https://pilot.pflow.xyz/).
