# Preface

Petri nets were invented in 1962. They predate Unix, the internet, and object-oriented programming. For most of their history, they lived in academic papers — a formalism known to theorists but invisible to working programmers.

This book argues they deserve wider use. Not because they're elegant (they are) but because they solve practical problems. A Petri net is a state machine that handles concurrency. It's a workflow engine with formal guarantees. It's a simulation model that converts to differential equations. It's a specification that can be verified, compiled to code, and proven in zero knowledge.

The pflow ecosystem makes this practical. You draw a net in a browser editor, simulate it, generate a running application, and verify it across implementations — all from the same JSON-LD model. The formalism provides the guarantees. The tools make the formalism accessible.

## Who This Book Is For

You know how to program but haven't encountered Petri nets before — or you've seen the formalism and wondered what it's good for in practice. Either way, this book meets you where you are.

The examples are in Go and JavaScript. The mathematics uses standard notation — vectors, matrices, differential equations. When a concept requires math, the math appears alongside working code so you can verify every claim by running it.

## How to Read This Book

There are two paths through the material:

- **Theory track**: Read Parts I and III for the mathematical foundations and advanced topics — formal definitions, incidence matrices, ODE solvers, zero-knowledge proofs, category theory.
- **Hands-on track**: Skim Part I for vocabulary, then dive into Part II's worked examples (coffee shops, games, biochemistry) and Part IV's tooling guides.

Both tracks converge — the theory explains why the tools work, and the tools make the theory tangible.

### Part I: Foundations

Chapters 1-4 introduce the formalism. Places, transitions, arcs, tokens — the four primitives. Then firing rules, incidence matrices, and conservation laws. Then the key insight: converting discrete nets to continuous ODE systems via mass-action kinetics. Finally, the token language DSL for defining nets with guards and typed arcs.

### Part II: Applications

Chapters 5-10 apply the formalism to six domains: resource management (coffee shop), game mechanics (tic-tac-toe), constraint satisfaction (sudoku), combinatorial optimization (knapsack), biochemistry (enzyme kinetics), and complex state machines (Texas Hold'em). Each chapter builds a complete model from scratch, simulates it, and analyzes the results.

### Part III: Advanced Topics

Chapters 11-14 push the formalism further. Process mining discovers nets from event logs. Zero-knowledge proofs verify transitions without revealing state. Exponential weights encode scoring systems. JSON-LD makes nets self-describing and content-addressable.

### Part IV: Building with pflow

Chapters 15-18 cover the ecosystem tools. The visual editor for designing nets interactively. The code generator that turns models into full-stack applications. The Go library's API and package structure. And the dual-implementation discipline that keeps Go and JavaScript in sync.

## The pflow Ecosystem

The projects referenced throughout this book form a unified ecosystem:

| Project | Purpose | URL |
|---------|---------|-----|
| **go-pflow** | Core library — ODE simulation, reachability, process mining | github.com/pflow-xyz/go-pflow |
| **pflow.xyz** | Visual browser-based editor for designing nets | pflow.xyz |
| **petri-pilot** | Code generator — turns net models into running applications | pilot.pflow.xyz |

All three share the same JSON-LD model format. A net designed in the editor can be simulated by the library and compiled by the code generator without format conversion. The model is the source of truth.

## Conventions

- **Code examples** appear inline for short snippets and in separate files for longer programs
- **Mathematical notation** uses standard linear algebra: bold lowercase for vectors ($\mathbf{m}$), uppercase for matrices ($\mathbf{C}$)
- **Cross-references** link between chapters: "the coffee shop model (Chapter 5)" or "mass-action kinetics (Chapter 3)"
- **Go code** uses the `go-pflow` library API; **JavaScript** uses the pflow.xyz browser modules
- **JSON-LD** snippets use `@context: "https://pflow.xyz/schema"` throughout

## Acknowledgments

This book grew from blog posts, documentation, and working code written over two years. The pflow ecosystem is open source, and the ideas draw on sixty years of Petri net theory — from Carl Adam Petri's 1962 dissertation through the process mining work of Wil van der Aalst and the zero-knowledge proof systems enabled by gnark.

The best way to learn is to build. Open pflow.xyz in a browser, draw a net, and press play. The mathematics follows.
