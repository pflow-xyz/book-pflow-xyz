# Preface

Petri nets were invented in 1962. They predate Unix, the internet, and object-oriented programming. For most of their history, they lived in academic papers — a formalism known to theorists but invisible to working programmers.

This book argues they deserve wider use, and it makes one claim about why. A Petri net is a small, fixed alphabet — place, transition, arc, guard — and a model written in it is a **value, not a program**. It is a document: it can be hashed, diffed, composed with another model by sharing a place, and checked by a machine that has never seen any of our code. From that one fact everything else follows. The same document is a state machine that handles concurrency, a workflow engine with formal guarantees, a simulation model that converts to differential equations, and a specification from which a running application, a kernel-checked theorem and a zero-knowledge proof can all be *derived* rather than written.

We call that claim **Metamodel**. The plain definition is the literal one: *a model for making models.* The four primitives are not a model of any particular thing — not a coffee shop, not a token — they are the fixed vocabulary every such model is written in, and the rules for how two of them fit together. Not one system that swallows every domain, but four primitives whose local composition rules generate an unbounded space of specific, checkable systems. The pflow ecosystem is where it is tested. You draw a net in a browser editor, simulate it, generate an application, and hold twenty implementations in four languages to one golden trace — all from the same JSON-LD model. The formalism provides the guarantees. The tools make the formalism accessible. And the domains the same alphabet has already been tiled across — a coffee shop, a token standard, a poker game, a drum machine, and the infrastructure that serves this book — are the evidence.

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

Chapters 17-20 cover the ecosystem tools. The visual editor for designing nets interactively. The code generator that turns models into full-stack applications. The Go library's API and package structure. And the parity discipline that keeps every implementation honest — two at first, twenty by the end. The Epilogue names the claim the whole book has been making.

## The pflow Ecosystem

The projects referenced throughout this book form a unified ecosystem:

| Project | Purpose | URL |
|---------|---------|-----|
| **go-pflow** | Core library — ODE simulation, reachability, process mining | github.com/pflow-xyz/go-pflow |
| **pflow.xyz** | Visual browser-based editor for designing nets | pflow.xyz |
| **petri-pilot** | Code generator — turns net models into running applications | pilot.pflow.xyz |
| **pflow-polyglot** | One model, five forms, ten languages, one golden trace | github.com/pflow-xyz/pflow-polyglot |
| **sim.pflow.xyz** | Business operations as calibrated what-if models | sim.pflow.xyz |

All of them share the same JSON-LD model format. A net designed in the editor can be simulated by the library and compiled by the code generator without format conversion. The model is the source of truth.

## Conventions

- **Code examples** appear inline for short snippets and in separate files for longer programs
- **Mathematical notation** uses standard linear algebra: bold lowercase for vectors ($\mathbf{m}$), uppercase for matrices ($\mathbf{C}$)
- **Cross-references** link between chapters: "the coffee shop model (Chapter 5)" or "mass-action kinetics (Chapter 3)"
- **Go code** uses the `go-pflow` library API; **JavaScript** uses the pflow.xyz browser modules
- **JSON-LD** snippets use `@context: "https://pflow.xyz/schema"` throughout

## Acknowledgments

This book grew from blog posts, documentation, and working code written over two years. The pflow ecosystem is open source, and the ideas draw on sixty years of Petri net theory — from Carl Adam Petri's 1962 dissertation through the process mining work of Wil van der Aalst and the categorical semantics of Meseguer, Montanari and Sassone.

The best way to learn is to build. Open pflow.xyz in a browser, draw a net, and press play. The mathematics follows.
