# Epilogue: What the Abstraction Sits On

**Learning objective**: Name the layered architecture this book actually built, see the six applications through the lens of net types, and identify the open problems worth pursuing.

This book promised a universal abstraction, and in Chapter 13 the promise shifted: the Petri net turned out to be one layer of a stack rather than its foundation. The result there that reaches furthest — that a game's strategic ranking can be read off graph connectivity, with no training data, no domain knowledge and no use of firing semantics — sits underneath the formalism the book is named for. This chapter starts from that result.

## The Four-Layer Stack

The book built a stack, one layer at a time, without naming it as such until now:

```
Layer 4: Derived Artifacts    (Chapters 12, 14, 17-20)
         Editor, generated app, Go and JS runtimes, Lean
         theorems, ZK circuit — each computed from the model,
         none hand-written to agree with it.

Layer 3: ODE Dynamics         (Chapters 3, 5-10)
         Mass-action kinetics couples topology to state.
         Rate formula: v[t] = k[t] × ∏ M[inputs[t]]

Layer 2: Petri Net Semantics  (Chapters 1-2, 4)
         Firing rules, conservation laws, P-invariants.
         Tokens consumed and produced atomically.

Layer 1: Graph Theory          (Chapter 13)
         Bipartite directed graph. Degree centrality.
         Connectivity determines what matters.
```

Each layer adds something the layer below cannot express:

- **Graph theory** tells you what connects to what. It cannot tell you what happens when you act — there are no tokens, no state, no dynamics.
- **Petri net semantics** add state and atomicity. Transitions consume and produce. Conservation laws constrain the state space. But the formalism alone doesn't tell you what happens *first*, or how fast.
- **ODE dynamics** add time. Mass-action kinetics couple topology-derived rates to the current marking. You get trajectories, equilibria, predictions. But the trajectories are only as trustworthy as the implementation that computed them.
- **Derived artifacts** add everything a user actually touches, and none of it is written by hand to agree with the model. The generated application, the two runtimes, the kernel-checked theorems and the arithmetic circuit are all computed from the model document. The ZK circuit illustrates the point without being the point: its constraints are generated from the incidence matrix (Chapter 13), so once the model is data the circuit follows. It is one derived artifact among several, and the book treats it that way.

Parts I–III introduced the bottom three layers out of order. Petri net semantics and ODE dynamics came first; graph theory came last, in Chapter 13, when the rate auto-derivation showed that pure connectivity carries more information than expected. The classic tic-tac-toe heuristic (center > corner > edge) falls out of counting connections, with no game theory and no training.

The Petri net is therefore the modeling layer, the place where graph structure acquires semantics, rather than the whole story.

## Six Applications, Four Types

Chapter 4 introduced the categorical net taxonomy before the reader had examples to anchor it. After six worked applications in Part II, it can be checked against them:

| Chapter | Application | Net Type | Defining Property |
|---------|------------|----------|-------------------|
| 5 | Coffee Shop | ResourceNet | Conservation — ingredients are neither created nor destroyed |
| 6 | Tic-Tac-Toe | GameNet | Turn control + conservation — pieces placed, never removed |
| 7 | Sudoku | ClassificationNet | Constraint accumulation — each placement is evidence toward a solved board |
| 8 | Knapsack | ComputationNet | Continuous relaxation — ODE finds approximate optima |
| 9 | Enzyme Kinetics | ComputationNet | Native domain — mass-action kinetics *is* the chemistry |
| 10 | Texas Hold'em | GameNet | Multi-phase workflow + role-based turn control |

In each row the defining property is a property of the wiring. The coffee shop conserves ingredients because every gram a recipe takes from a stock place is deposited in a matching `used` place (Chapter 5). A GameNet alternates turns because a turn-control place gates player transitions through mutual exclusion. Read this way, the taxonomy describes structural invariants that the topology either has or lacks, which is Chapter 13's observation seen from another angle: the structure carries the meaning.

## What the Book Argued

Three claims run from Chapter 1 to Chapter 20. They are argued from the worked examples rather than proved in general:

**Small models beat black boxes.** Every application in this book is inspectable. You can look at the tic-tac-toe topology and count win lines. You can read the stoichiometry matrix and see the differential equations. You can read the generated code and the constraint system it compiles to. At no point did you need to trust a model you couldn't read. This is the opposite of the machine learning approach, where the knowledge is in the weights and the weights are opaque. The cost is that Petri net models require a human to design the topology. The benefit is that the topology is the explanation.

**One formalism, multiple tools.** The JSON-LD model format (Chapter 16) is read by the visual editor (Chapter 17), the code generator (Chapter 18), the Go library (Chapter 19) and the ZK compiler (Chapter 12). Dual implementation (Chapter 20) checks that independent implementations agree, and `pflow-polyglot` extends the check to thirty-nine programs in ten languages, all held to one golden trace, [`parity/trace.golden`](https://github.com/stackdump/pflow-polyglot/blob/main/parity/trace.golden). `make parity` fails the build when any of its twenty Go, Rust, Python and JavaScript programs diverges, and `make parity-native` checks the rest against the same file. The agreement is a test that runs.

**Topology is primary, rates are secondary.** Change the rate constants and the system's quantitative behavior shifts; change the topology and it becomes a different system. The book observed this across its six applications rather than proving it, and Chapters 6 and 13 give it its sharpest form: a ranking read from incidence counts that the ODE reproduces. Chapter 6 also marks its limit. A forced block whose margin is thin can flip with the detector rates, so "secondary" means secondary for the ranking, not irrelevant.

## What the Book Didn't Solve

Chapter 13's rate auto-derivation, and the composition machinery of Chapter 4, leave several questions open.

**Multi-hop connectivity.** The rate auto-derivation counts direct connections: candidate → unique output → target input. For tic-tac-toe, where every win is one hop from a placed piece, that is sufficient. The book has not applied it to games whose value lies several moves deep, such as chess or Go; a one-hop count is expected to miss most of what matters there, but that expectation is untested. The question is whether multi-hop reachability analysis — T-invariants, unfoldings, or iterative message-passing over the bipartite graph — can extend the one-hop algorithm to deeper games. It is a graph theory question rather than a Petri net question.

**Weighted targets.** The algorithm treats every target connection as weight 1. A checkmate path and a pawn capture score the same. The fix seems straightforward — assign importance weights to targets — but the principled question is where those weights come from. Can topology derive them recursively? Or does heterogeneous objective weighting require domain knowledge that the graph alone cannot supply?

**Dynamic rates.** Chapter 13's topology-derived rates are computed once, from the full graph, yet a corner's strategic value changes mid-game when it completes a fork threat. Chapter 6 handles this for tic-tac-toe in two ways: its dynamic evaluation recomputes the incidence degree against only the win transitions still reachable from the current board, and its derived evaluation net, which declares the opponent's forced replies as transitions, makes no value-losing move against any opponent line in an exhaustive referee check ([petri-pilot/experiments/ode-minimax](https://github.com/pflow-xyz/petri-pilot/tree/main/experiments/ode-minimax)). Both are specific to one game. Whether the general rate derivation can be made state-dependent in the same way, recomputing connectivity over the *reachable* subgraph of any net, is open.

**Composition is structurally solved; proof composition mostly is not.** Composition is implemented, and for Chapter 4's Orders/Inventory bundle, flattening the composite and recomputing its P-invariants recovers both component laws. That does not hold in general. Gluing a three-into-one netting transition onto a settlement cycle leaves the cycle's all-ones invariant invalid in the composite ([pflow-jl netting experiment](https://github.com/pflow-xyz/pflow-jl/blob/main/docs/netting-experiment.md)): a component's invariant extended by zeros need not be an invariant of the whole, although every invariant of the whole restricts to one of each part. What does lift is kernel-checked. If each component's invariant agrees with the other's on the shared places, together they define an invariant of the composite — the Lean theorem [`invariant_lift`](https://github.com/pflow-xyz/book-pflow-xyz/blob/main/proofs/PflowProofs/InvariantLift.lean). Liveness need not lift: composition refines rather than extends, which preserves safety and not liveness (Appendix E). Beyond that one theorem, a composed system is still proved — in Lean or in circuit — as one flattened net rather than as a composition of component proofs. Assume-guarantee reasoning suggests the rest is tractable, since composition only needs to verify the boundaries; it is a derived-artifact problem, not a modeling one.

## What the ODE Was Actually Computing

Chapter 13's rate auto-derivation counts, for each candidate move, how many target transitions consume the places that move alone produces: 4 for the center, 3 for a corner, 2 for an edge. Chapter 6 ran the ODE on its own version of the tic-tac-toe net (30 places and 34 transitions, against the 33 and 35 of the variant Chapter 13 counts) and got empty-board scores of 1.27, 0.95 and 0.63 — within a few percent of 4 : 3 : 2, and in the same order. Chapter 13's poker net reads the count the other way round. Each hand category's value place fills at rate $k$ from a catalytic source and empties through $n$ drain transitions, so its equation is $\dot{x} = k(1 - n x)$ and its equilibrium is exactly $1/n$: one drain for a straight flush, thirty-two for high card. That value is derived in closed form; the book does not yet link a run of the poker net. Counting and simulating agree because both read the same arcs — exactly where the equation can be solved by hand, and approximately but order-preserving on the tic-tac-toe boards Chapter 6 tested, where it cannot.

## Where the Alphabet Has Travelled

Games are where the book validated its techniques. A stronger test of portability than a list of domains the techniques *could* apply to is the list where the same four primitives already run, sharing no code at the domain level. On `sim.pflow.xyz`, a help desk is built from three components — arrivals, service and a hazard — attached to one shared queue place, and the result is validated and simulated as one model. Chapter 4's ERC-20 schema declares its conservation law, `sum(balances) == totalSupply`, as part of the model rather than leaving it to an audit. In `beats.bitwrap.io`, drum patterns are token rings and song structure is a linear net that mutes and unmutes tracks; the sequencer plays a note when a transition fires on a hit. The deployment tooling for this ecosystem (private) declares each app as a manifest of a handful of fields and a marking, and derives the services entry, vhost, certificate and uptime check from it. And `pflow-polyglot` holds thirty-nine programs to one `model.json` and one golden trace.

The recipe is the same in each case: declare the structure, derive the rest.

## The Structure Underneath

The four-layer stack describes the book's architecture. Underneath it is one structure that explains *why* edge-matching composition works — why two models glued at a shared place need no glue code, and why what is true of the parts stays true of the whole.

That structure is the **symmetric monoidal category** (SMC).

### Transitions as Morphisms

A Petri net transition consumes tokens from input places and produces tokens into output places. In categorical terms, this is a morphism — a map from domain to codomain. A transition $t$ with inputs $\{p_1, p_2\}$ and outputs $\{q_1, q_2, q_3\}$ is:

$$t : p_1 \otimes p_2 \to q_1 \otimes q_2 \otimes q_3$$

The $\otimes$ is the monoidal product — it means "these things exist side by side." Two tokens in separate places aren't combined or merged; they coexist independently. This is how Petri nets express concurrency: $p_1 \otimes p_2$ means both places are marked, and both tokens are available simultaneously.

The objects are multisets of places — markings, elements of the free commutative monoid $\mathbb{N}^P$ generated by the places — and the transitions are the generating morphisms. Every morphism is built from them by sequential and parallel composition.

### Two Kinds of Composition

Every category has composition of morphisms. A monoidal category adds a second operation: the monoidal product. These correspond exactly to the two ways we composed nets throughout this book.

**Sequential composition** ($f \mathbin{;} g$): the output places of transition $f$ become the input places of transition $g$. Tokens flow through. This is ordinary morphism composition — the Texas Hold'em phase sequence (Chapter 10), the workflow cursor in a WorkflowNet (Chapter 4).

**Parallel composition** ($f \otimes g$): two transitions sit side by side with no shared places. They fire independently. This is the monoidal product — the nine hand categories of Chapter 13's poker analysis net, each with its own source, value and drain places, sharing none with the others.

The **symmetry** is the swap map $\sigma : A \otimes B \to B \otimes A$. It says we can reorder the components of a parallel composition without changing the behavior. In Petri net terms: the order in which we list the places carries no meaning. That is why `pflow-polyglot`'s golden trace prints each marking as place names sorted lexicographically: a listing order means nothing, so it has to be fixed before a Go map, a Rust set and a Python set can be compared at all.

### Why This Explains the Book

The formal result, due to Sassone (1995) and Meseguer-Montanari (1990), is that a Petri net generates a free symmetric monoidal category whose objects are multisets of places and whose morphisms are equivalence classes of transition firings. "Free" means nothing extra is imposed — the only equations are the ones forced by the SMC axioms.

This theorem has been silently at work in every chapter:

- **Event sourcing works** (Chapter 10) because sequential composition is associative: $(f \mathbin{;} g) \mathbin{;} h = f \mathbin{;} (g \mathbin{;} h)$. The fold over events doesn't depend on how you chunk the replay.

- **The poker ODE decouples** (Chapter 13) because the analysis net is a parallel composition. Each hand category's equation $\dot{x}_H = k(1 - n_H x_H)$ involves only its own value place, since the categories share no place for information to pass through.

- **Typed composition refines** (Chapter 4). Adding an unlinked schema to a CompositeNet is adding a new object to the category, and the monoidal product guarantees it can't affect existing schemas — that much is monotonic. Adding a *link* is not: it identifies transitions or places, which is a quotient rather than a product, and quotients remove behavior. What the structure buys is refinement — every composite trace projects to a valid component trace — which preserves safety properties and not liveness.

### What the Category Doesn't See

The SMC encoding captures process structure — which compositions are valid, which transitions are independent. It has no privileged present: every marking is just another object. But every engine in this book reads one particular marking on every step, and that marking has a tense structure the category does not see.

Chapter 6 showed it on a net small enough to see whole. History places are **past** — write-once, monotone, the boolean case of tropical $(\max,+)$ accumulation. The board and turn places are **present** — the marking the step is about. The move transitions and their guards are **future** — recomputed from the marking on every step, never stored. Appendix E states the split formally.

That is a scope boundary of the SMC framework rather than a deficiency: the category tells you what can compose, and the tenses tell you where you are in the composition. What matters for this book is that all three are *data in the same document* — the event log, the marking, the guards — so the question "do two implementations agree about where we are?" is answered by a trace diff, not an argument.

### Where Declare-Then-Derive Already Runs

Step back and look at what is running. In every row the left column is a document in the four-primitive vocabulary, and everything in the right column is computed from it:

| Domain | Declared artifact | Derived from it |
|--------|-------------------|-----------------|
| Business simulation | `sim.pflow.xyz` model (JSON-LD) | Analysis suite and the running application |
| Ecosystem devops (private) | App manifest (YAML) | Services entry, nginx vhost, certificate, uptime synthetic |
| Agent workflows (private) | Journey frontmatter (a Petri net) | The prompts themselves — the document is the runtime |
| Multi-language parity | `pflow-polyglot/model.json` | Thirty-nine implementations and one golden trace |
| Formal proof | The same `model.json` | Kernel-checked Lean theorems ([`lean/proof.lean`](https://github.com/stackdump/pflow-polyglot/blob/main/lean/proof.lean), generated) |
| Content addressing | `index.md` frontmatter | Searchable facets and a CID-addressed URL |

These rows were not designed together; each answered a specific problem, as the chapters did. They cohere because they share the alphabet: a place is a place in a help desk, a drum machine and a fleet of web services, and composing by shared place works the same way in each. Appendix E gives the categorical account of why that composition is well behaved, and of where it stops.

## The Premise, Revisited

Chapter 1 opened with a complaint: informal models fail because they don't capture the structure of the systems they represent. Concurrency is an afterthought. Resources are invisible. State is implicit.

Petri nets fix this by making structure explicit. Places hold state. Transitions change it. Arcs constrain what can flow where. Conservation laws fall out of the topology. The model is the specification.

The deeper lesson is that what made all of this possible is not specific to Petri nets: **the model is a value, not a program.** A net is a document: four primitives — place, transition, arc, guard — and a marking. It can be hashed, diffed, composed by matching edges, and handed to a stranger in another language who can check it without trusting us. The derived artifacts of Part IV, the invariants of Part I and the domains above all depend on that. In the systems this book modeled, topology determined more about behavior than parameter tuning did, and it could be *read* only because it was written down as data.

That is the claim this book has been circling, and the name for it is Metamodel: a small, fixed alphabet whose local composition rules generate an unbounded space of specific, checkable systems. The palette is small on purpose, and its smallness is what lets it travel.
