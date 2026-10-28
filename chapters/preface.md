# Preface

Petri nets were invented in 1962. They predate Unix, the internet, and object-oriented programming. For most of their history, they lived in academic papers — a formalism known to theorists but invisible to working programmers.

This book argues they deserve wider use, and it makes one claim about why. A system built from parts has to be written down in some notation for composition, and several are in use: cospans of sets, string diagrams, ZX-style spiders, operad wiring diagrams, incidence matrices. Each was designed for its own community, and each has its own rules for what "plug these two together" means. The open Petri net — places, transitions, arcs, and a boundary of places left open for gluing — has a presentation in every one of them, and the claim is that each presentation is faithful and that gluing two open nets along a shared place gives the same net in all of them. Part of that claim is checked by machine and part is not; the next paragraph says which. We call that being a **canonical carrier** for compositional systems: one structure, many faithful presentations, and one meaning of composition across all of them.

The evidence is public, in pflow-native's [composition views](https://github.com/pflow-xyz/pflow-native/blob/main/crates/app/src/compviews.rs). They draw two compositions ten ways — as a cospan, as spiders, as an operad wiring diagram, as a string diagram, as the flat net, as a net with boundaries, as lenses, as incidence matrices, as runs (processes) and as composition laws — and the two are a model bundle, `orders`, which links a queue, a resource pool and a payment saga, and a search plan, `connect4`, which assembles a Connect-4 solver from pieces. A reference tool checks each part. go-pflow checks the bundle: its verdicts on all 5,576 possible links between the ports of six subnets, on eleven flattened nets (four of them glued along shared places) and on four scenario runs are recorded in a [receipt](https://github.com/pflow-xyz/pflow-native/blob/main/crates/core/tests/fixtures/views/gopflow.json), and pflow-native, which composes with pflow-rs, [matches every one](https://github.com/pflow-xyz/pflow-native/blob/main/crates/core/tests/views_compose.rs). pflow-rs checks the plans: its verdicts on 289 plans are a [second receipt](https://github.com/pflow-xyz/pflow-native/blob/main/crates/core/tests/fixtures/views/plancheck.json), made before pflow-rs changed some of them on purpose, so [the test](https://github.com/pflow-xyz/pflow-native/blob/main/crates/core/tests/views_plan.rs) names each difference and asserts how many plans it covers. Chyp checks the composition laws — event links as synchronising spiders, and how plan pieces chain, meet and wrap. Upstream Chyp [computed its verdicts](https://github.com/pflow-xyz/pflow-native/blob/main/crates/app/assets/chyp/README.md) on four theories, 33 rewrites, of which the 30 meant to hold are valid and the three negative controls are rejected; pflow-native draws them and does not re-check them. Each view records what it draws as a scene manifest, and the [goldens](https://github.com/pflow-xyz/pflow-native/blob/main/parity/composition-views/goldens.json) hold 100 of them, ten views of two compositions in five states: [pflow-native's test](https://github.com/pflow-xyz/pflow-native/blob/main/crates/app/tests/composition_views.rs) redraws each one headless and requires it to equal its golden, pflow.xyz's browser views are [held to the same file](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/composition-views/parity_test.ts), and [screenshots](https://github.com/pflow-xyz/pflow-native/tree/main/docs/composition-views) show all twenty views. What none of this checks is that each notation, composing by its own rules, reaches the same net. Every view draws the one composite pflow-native computes, so the views agree by construction, and the cross-check is between two implementations of the gluing, go-pflow's and pflow-rs's. Faithfulness, that a presentation loses nothing, is not tested here either. The views' Chyp theories are also not the ones in chyp-rs (below). chyp-rs states its own laws on the same four subjects, and upstream Chyp agrees with it on every step of those, but it has not been run on the views' theories: their source files are not published, only Chyp's exports of them, and three of the four use typed wires, which chyp-rs leaves out. Chapter 20 applies the same discipline to runtimes.

A model written this way is also a **value, not a program**. It is a document: it can be hashed, diffed, composed with another model by sharing a place, and checked by a machine that has never seen any of our code. That is what lets one document be a state machine that handles concurrency, a workflow engine with formal guarantees, a simulation model that converts to differential equations, and a specification from which a running application, a kernel-checked theorem and a zero-knowledge proof are *derived* rather than written.

We call the practice built on it **Metamodel**. The plain definition is the literal one: *a model for making models.* Its vocabulary is place, transition, arc and guard, with local rules for how two models fit together, and it is not a model of any particular thing — not a coffee shop, not a token. The pflow ecosystem is where it is tested. You draw a net in a browser editor, simulate it, generate an application, and hold twenty implementations in four languages to one golden trace — all from the same JSON-LD model. The domains the same vocabulary has already been tiled across — a coffee shop, a token standard, a poker game, a drum machine, and the infrastructure that serves this book — are the evidence that it travels.

## What This Book Claims, and What It Does Not

The title says *universal*, and the word needs fencing before Chapter 1 uses the formalism for anything. The claim is about composition: among notations for systems built from parts, the open Petri net is one structure that all of them can carry without loss. It is not a claim about computation, and not a claim about every kind of system.

The scope is systems built from parts — components with state, events that change it, and resources or conditions that two components can share. A queue, a protocol, a recipe, a game, a ledger and a chemical reaction network all fit. A system whose meaning sits in a continuous field, in a learned weight matrix or in unstructured text does not become a Petri net by being drawn as one, and the book makes no claim about those.

Several limits are worth naming now, because later chapters lean on them.

Ordinary place/transition nets are not Turing-complete. Their reachability problem is decidable (Mayr 1981; Kosaraju 1982), which is a real strength: "can this workflow finish?" has an algorithm. The algorithm is impractical in the worst case, though. Reachability is Ackermann-complete (upper bound Leroux and Schmitz 2019; lower bound Czerwiński and Orlikowski 2021, and independently Leroux 2021), and even coverability is EXPSPACE-complete (Lipton 1976; Rackoff 1978). The tools in this book explore bounded nets of modest size and stop at a configurable state limit (Chapter 1), which is an engineering bound, not a decision procedure for every net.

Extensions buy expressiveness and spend the guarantees. An inhibitor arc tests a place for zero, and inhibitor arcs on just two places are enough to simulate a two-counter machine (Minsky 1967), so nets with inhibitor arcs are Turing-complete and their reachability is undecidable (Chapter 4). A guard over typed data, such as `balances[from] >= amount`, is a predicate the incidence matrix cannot see. Both kinds of contextual test also leave the free symmetric monoidal category that makes composition well behaved ([Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries)). The canonical-carrier claim covers place, transition and arc. The guard is the fourth term of the token language and the one that steps outside the claim; a model that uses one is checked by exploring states, not read off the structure.

Faithful means faithful to a pair of matrices, not to their difference. The incidence matrix $C$ of Chapter 2 records net change, so a self-loop cancels to zero and two different nets can share a $C$. The presentation that carries a net without loss is the pair of input and output matrices together with the boundary. Gluing along a shared place is the composition the presentations agree on. Fusing two transitions into a rendezvous, Chapter 4's EventLink, is a second kind of composition, and it has a string-diagram presentation of its own: the link is a synchronising spider on the transitions' event ports, a special commutative Frobenius algebra. chyp-rs, a Rust port of Chyp's checker, checks the laws that make that presentation well behaved — two connected sync points fuse into one, their legs are unordered, and fusing three transitions gives the same sync point whichever pair is fused first ([`laws/sync_spiders.chyp`](https://github.com/stackdump/chyp-rs/blob/main/laws/sync_spiders.chyp)). A rendezvous still refines rather than extends, so the safety-not-liveness caveat below applies to it as well.

Composition preserves safety and not liveness. Gluing two nets can only restrict what each can do, so a property of the form "no reachable marking looks like this" survives, and a component's conservation law survives when the other side has an invariant that agrees with it on the shared places — the case the Lean theorem `invariant_lift` checks. "This transition can always fire again" does not survive: Appendix E glues one netting transition onto a settlement cycle whose every transition is live, and the composite reaches a dead marking in four firings.

None of this is new mathematics. The free-category theorem is Meseguer and Montanari's and Sassone's, open nets as cospans are Baez and Master's, and the decidability results are decades old; Appendix E lists the references. What this book adds is a working ecosystem in which the same document passes through each of these readings, and the readings are made to agree by machine.

## Who This Book Is For

You know how to program but haven't encountered Petri nets before — or you've seen the formalism and wondered what it's good for in practice. Either way, this book meets you where you are.

The examples are in Go and JavaScript. The mathematics uses standard notation — vectors, matrices, differential equations. When a concept requires math, the math appears alongside working code so you can verify every claim by running it.

## How to Read This Book

There are two paths through the material:

- **Theory track**: Read Parts I and III for the mathematical foundations and advanced topics — formal definitions, incidence matrices, ODE solvers, invariants, and the categorical reasons composition works.
- **Hands-on track**: Skim Part I for vocabulary, then dive into Part II's worked examples (coffee shops, games, biochemistry) and Part IV's tooling guides.

Both tracks converge — the theory explains why the tools work, and the tools make the theory tangible.

### Part I: Foundations

Chapters 1-4 introduce the formalism. Places, transitions, arcs — structure — and tokens, the state that moves through it. Then firing rules, incidence matrices, and conservation laws. Then the key insight: converting discrete nets to continuous ODE systems via mass-action kinetics. Finally, the token language DSL, which adds the fourth primitive — the guard — and typed composition.

### Part II: Applications

Chapters 5-10 apply the formalism to six domains: resource management (coffee shop), game mechanics (tic-tac-toe), constraint satisfaction (sudoku), combinatorial optimization (knapsack), biochemistry (enzyme kinetics), and complex state machines (Texas Hold'em). Each chapter builds a complete model from scratch, simulates it, and analyzes the results.

### Part III: Advanced Topics

Chapters 11-16 push the formalism further. Process mining discovers nets from event logs. Topology alone derives strategy. Exponential weights encode scoring systems. JSON-LD makes nets self-describing and content-addressable. Zero-knowledge proofs appear here too, as what you get for free once the incidence matrix is the constraint system — a derived artifact, not a destination.

### Part IV: Building with pflow

Chapters 17-20 cover the ecosystem tools. The visual editor for designing nets interactively. The hosted service that runs a stored model as seeded what-ifs and checks it. The Go library's API and package structure. And the parity discipline that keeps every implementation honest — two at first, twenty by the end. The Epilogue names the claim the whole book has been making.

## The pflow Ecosystem

The projects referenced throughout this book form a unified ecosystem:

| Project | Purpose | URL |
|---------|---------|-----|
| **go-pflow** | Core library — ODE simulation, reachability, process mining | github.com/pflow-xyz/go-pflow |
| **pflow.xyz** | Visual browser-based editor for designing nets | pflow.xyz |
| **pflow-polyglot** | One model, five forms, ten languages, one golden trace | github.com/pflow-xyz/pflow-polyglot |
| **sim.pflow.xyz** | Hosted what-if simulation and verification of stored models, with an MCP server | sim.pflow.xyz |

The editor and the library share the same JSON-LD model format, and go-pflow converts it into the model format sim.pflow.xyz stores and runs. A net designed in the editor can be simulated by the library without format conversion. The model is the source of truth.

## Conventions

- **Code examples** appear inline for short snippets and in separate files for longer programs
- **Mathematical notation** uses standard linear algebra: bold lowercase for vectors ($\mathbf{m}$), uppercase for matrices ($\mathbf{C}$)
- **Cross-references** link between chapters: "the coffee shop model (Chapter 5)" or "mass-action kinetics (Chapter 3)"
- **Go code** uses the `go-pflow` library API; **JavaScript** uses the pflow.xyz browser modules
- **JSON-LD** snippets use `@context: "https://pflow.xyz/schema"` throughout

## Acknowledgments

This book grew from blog posts, documentation, and working code written over two years. The pflow ecosystem is open source, and the ideas draw on sixty years of Petri net theory — from Carl Adam Petri's 1962 dissertation through the process mining work of Wil van der Aalst and the categorical semantics of Meseguer, Montanari and Sassone.

The best way to learn is to build. Open pflow.xyz in a browser, draw a net, and press play. The mathematics follows.

## License

To the extent possible under law, the author has dedicated this book to the public domain under the [CC0 1.0 Universal](https://creativecommons.org/publicdomain/zero/1.0/) dedication. You may copy, modify, translate, redistribute and build on it for any purpose, with no permission and no attribution required. The Go code that builds the book and the site is separate and is released under the MIT license; see the repository.
