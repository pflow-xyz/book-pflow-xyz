# Epilogue: What the Abstraction Sits On

**Learning objective**: Name the layered architecture this book actually built, see the six applications through the lens of net types, and identify the open problems worth pursuing.

This book promised a universal abstraction. It delivered one — but along the way, especially in Chapter 13, something shifted. The Petri net turned out to be a layer, not the foundation. The most powerful result in the book — that strategic value emerges from graph connectivity with no training data, no domain knowledge, and no Petri net firing semantics — lives underneath the formalism the book is named for.

That deserves to be said plainly, not buried in a subsection.

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
- **Derived artifacts** add everything a user actually touches — and nothing a user could get wrong, because none of it is authored. The generated application, the two runtimes, the kernel-checked theorem and the arithmetic circuit are all functions of the model document. The ZK circuit is the clearest illustration of the point rather than the point itself: the incidence matrix *is* the constraint system, so proof comes for free once the model is data. It is one derived artifact among several, and the book treats it that way.

The book introduced these layers bottom-up in Parts I-III, but the reader encounters the stack's true shape only in Chapter 13, when the rate auto-derivation reveals that the bottom layer — pure graph connectivity — carries more information than expected. The classic tic-tac-toe heuristic (center > corner > edge) falls out of counting connections. No game theory. No training. Just topology.

The Petri net is not the foundation but the modeling layer — the place where graph structure acquires semantics. That's valuable. It's just not the whole story.

## Six Applications, Four Types

Chapter 4 introduced the categorical net taxonomy before the reader had examples to anchor it. Now, after six worked applications in Part II, the taxonomy earns its weight:

| Chapter | Application | Net Type | Defining Property |
|---------|------------|----------|-------------------|
| 5 | Coffee Shop | ResourceNet | Conservation — ingredients are neither created nor destroyed |
| 6 | Tic-Tac-Toe | GameNet | Turn control + conservation — pieces placed, never removed |
| 7 | Sudoku | ClassificationNet | Constraint accumulation — each placement is evidence toward a solved board |
| 8 | Knapsack | ComputationNet | Continuous relaxation — ODE finds approximate optima |
| 9 | Enzyme Kinetics | ComputationNet | Native domain — mass-action kinetics *is* the chemistry |
| 10 | Texas Hold'em | GameNet | Multi-phase workflow + role-based turn control |

The pattern: you never had to tell the Petri net what kind of system it was modeling. The net type emerged from how you wired the arcs. A ResourceNet conserves tokens because the topology conserves them — every arc into a transition has a matching arc out. A GameNet alternates turns because a turn-control place gates player transitions through mutual exclusion.

The taxonomy is not a labeling scheme imposed from outside but a description of structural invariants that the topology either has or doesn't. This is the same insight as "it's graph theory, not Petri net theory," seen from a different angle: the structure carries the meaning.

## What the Book Proved

Three claims survived from Chapter 1 to Chapter 20:

**Small models beat black boxes.** Every application in this book is inspectable. You can look at the tic-tac-toe topology and count win lines. You can read the stoichiometry matrix and see the differential equations. You can read the generated code and the constraint system it compiles to. At no point did you need to trust a model you couldn't read. This is the opposite of the machine learning approach, where the knowledge is in the weights and the weights are opaque. The cost is that Petri net models require a human to design the topology. The benefit is that the topology is the explanation.

**One formalism, multiple tools.** The JSON-LD model format (Chapter 16) is processed identically by the visual editor (Chapter 17), the code generator (Chapter 18), the Go library (Chapter 19), and the ZK compiler (Chapter 12). Dual implementation (Chapter 20) verifies that independent implementations agree — and `pflow-polyglot` extends the check from two implementations to twenty-plus, across four languages, all held to one golden trace that fails the build on divergence. This isn't a theoretical property; it is a test that runs.

**Topology is primary, rates are secondary.** Change the rate constants and the system's quantitative behavior shifts. Change the topology and the system becomes a different system. This inversion — structure over parameters — holds across all six applications and both modes (combinatorial and continuous). It's the book's most load-bearing claim, and Chapter 13 gave it a precise formulation.

## What the Book Didn't Solve

The limitations section of Chapter 13 was honest, but it was framed as caveats. They're better understood as open problems.

**Multi-hop connectivity.** The rate auto-derivation counts direct connections: candidate → unique output → target input. For tic-tac-toe (depth 1), this is sufficient. For chess (depth varies), it captures material value but misses tactics. For Go (depth 19×19), it captures almost nothing. The question: can multi-hop reachability analysis — T-invariants, unfoldings, or iterative message-passing over the bipartite graph — extend the one-hop algorithm to deeper games? This is a graph theory question, not a Petri net question, which is itself instructive.

**Weighted targets.** The algorithm treats every target connection as weight 1. A checkmate path and a pawn capture score the same. The fix seems straightforward — assign importance weights to targets — but the principled question is where those weights come from. Can topology derive them recursively? Or does heterogeneous objective weighting require domain knowledge that the graph alone cannot supply?

**Dynamic rates.** Topology-derived rates are static. A corner's strategic value changes mid-game when it completes a fork threat. The tactical scoring layer in Chapter 6 handles this for tic-tac-toe, but it's an add-on, not part of the rate derivation. Can the rate formula incorporate state-dependent topology — recomputing connectivity over the *reachable* subgraph rather than the full graph? This would unify the strategic (topology) and tactical (state) layers.

**Composition is structurally solved; proof composition is not.** Composition is implemented, and flattening a composed bundle recovers each component's P-invariants from the incidence matrix of the whole — so "the composed system preserves the invariants of each component" is a computation rather than a conjecture. Working through it corrected the claim it rested on: composition *refines* rather than extends, which preserves safety and not liveness. What remains open is doing the same in the proof forms. Today a composed system is proved — in Lean or in circuit — as one flattened net rather than as a composition of component proofs, and the circuit form additionally grows as $O(|P| \times |T|)$. Assume-guarantee suggests both are tractable, since composition only needs to verify the boundaries; this is a derived-artifact problem, not a modeling one.

## What the ODE Was Actually Computing

Chapter 13's rate auto-derivation counted, for each place, how many transitions consume from it — the diagonal of $BB^T$, where $B$ is the input adjacency matrix of the bipartite graph. The ODE, run to equilibrium on the same net, relaxed to $1/(BB^T)_{ii}$ per place. Counting and simulating gave the same answer — center > corner > edge in tic-tac-toe, straight flush > … > high card in poker — because both were reading the same document. That is the whole result, and it is worth stating without ornament: two independent analyses agree when, and because, there is one model for them to read.

## Where the Alphabet Has Travelled

Games are where the book validated its techniques. The honest test of portability is not a list of domains they *could* apply to but the list of domains where the same four primitives have already been tiled and shipped, sharing no code at the domain level:

- **Business operations.** `sim.pflow.xyz` builds a help desk from three components glued at a shared queue and derives the analysis suite and the running application from that one model.
- **Token standards.** ERC-20 and ERC-721 as nets (Chapter 4), with conservation as a discovered invariant rather than an audited property.
- **Music.** `beats.bitwrap.io` — every note a transition firing; the sequencer is the net.
- **Infrastructure.** The deployment tooling that operates this ecosystem (private): an app manifest is a handful of fields and a marking, and the services entry, vhost, certificate and uptime check are derived from it. Thirteen live services, zero drift, and no code written to guarantee that.
- **Twenty implementations.** `pflow-polyglot`: one `model.json`, five forms across ten languages, one golden trace.

In every case the recipe is identical — declare the structure, derive the rest — and every instance inherited composability, conservation and checkability without its author asking for them.

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

**Parallel composition** ($f \otimes g$): two transitions sit side by side with no shared places. They fire independently. This is the monoidal product — the concurrent recipe stations in the coffee shop (Chapter 5), the independent win-line accumulators in tic-tac-toe (Chapter 13).

The **symmetry** is the swap map $\sigma : A \otimes B \to B \otimes A$. It says we can reorder the components of a parallel composition without changing the behavior. In Petri net terms: the order we list the places doesn't matter. This is exactly why ODE signatures (Chapter 13) are invariant under reordering — shuffling places and transitions produces the same solution every time.

### Why This Explains the Book

The formal result, due to Sassone (1995) and Meseguer-Montanari (1990), is that a Petri net generates a free symmetric monoidal category whose objects are multisets of places and whose morphisms are equivalence classes of transition firings. "Free" means nothing extra is imposed — the only equations are the ones forced by the SMC axioms.

This theorem has been silently at work in every chapter:

- **Event sourcing works** (Chapter 10) because sequential composition is associative: $(f \mathbin{;} g) \mathbin{;} h = f \mathbin{;} (g \mathbin{;} h)$. The fold over events doesn't depend on how you chunk the replay.

- **The ODE decouples** (Chapter 13) because the monoidal product means independence. Each accumulator's equation $\dot{x}_i = 1 - n_i x_i$ is a separate lens, composed in parallel. No information leaks between components because $\otimes$ *means* no interaction.

- **Typed composition refines** (Chapter 4). Adding an unlinked schema to a CompositeNet is adding a new object to the category, and the monoidal product guarantees it can't affect existing schemas — that much is monotonic. Adding a *link* is not: it identifies transitions or places, which is a quotient rather than a product, and quotients remove behavior. What the structure buys is refinement — every composite trace projects to a valid component trace — which preserves safety properties and not liveness.

### What the Category Doesn't See

The SMC encoding captures process structure — which compositions are valid, which transitions are independent. It has no privileged present: every marking is just another object. But every engine in this book reads one particular marking on every step, and that marking has a tense structure the category does not see.

Chapter 6 showed it on a net small enough to see whole. History places are **past** — write-once, monotone, the boolean case of the tropical accumulation Chapter 13 uses for timed nets. The board and turn places are **present** — the marking the step is about. The move transitions and their guards are **future** — recomputed from the marking on every step, never stored. Appendix E states the split once, formally. The short version: *the past is tropical, the future is predicate, and the present is the boundary where they meet.*

This is not a deficiency of the SMC framework but a scope boundary. The category tells you what can compose. The tenses tell you where you are in the composition. What matters for this book is that all three are *data in the same document* — the event log, the marking, the guards — so the question "do two implementations agree about where we are?" is answered by a trace diff, not an argument.

### Where Declare-Then-Derive Already Runs

Step back and look at what is running. In every row the left column is a document in the four-primitive vocabulary, and everything in the right column is computed from it:

| Domain | Declared artifact | Derived from it |
|--------|-------------------|-----------------|
| Business simulation | `sim.pflow.xyz` model (JSON-LD) | Analysis suite and the running application |
| Ecosystem devops (private) | App manifest (YAML) | Services entry, nginx vhost, certificate, uptime synthetic |
| Agent workflows (private) | Journey frontmatter (a Petri net) | The prompts themselves — the document is the runtime |
| Multi-language parity | `pflow-polyglot/model.json` | Twenty-plus implementations and one golden trace |
| Formal proof | The same `model.json` | Kernel-checked Lean theorems |
| Content addressing | `index.md` frontmatter | Searchable facets and a CID-addressed URL |

None of this was designed as a system. The book accumulated one chapter at a time, each solving a specific problem, and the rows above accumulated the same way. They cohere because the alphabet forces them to: a place is a place in a help desk, a drum machine and a fleet of web services, and composing by shared place works identically in each. The category theory in Appendix E is not a framework the book adopted but the reason that coherence was always going to be there.

## The Premise, Revisited

Chapter 1 opened with a complaint: informal models fail because they don't capture the structure of the systems they represent. Concurrency is an afterthought. Resources are invisible. State is implicit.

Petri nets fix this by making structure explicit. Places hold state. Transitions change it. Arcs constrain what can flow where. Conservation laws fall out of the topology. The model is the specification.

But the deeper lesson — the one that emerged through writing this book, not before it — is that what made all of this possible is not a property of Petri nets specifically. It is that **the model is a value, not a program.** A net is a document: four primitives — place, transition, arc, guard — and a marking. It can be hashed, diffed, composed by matching edges, and handed to a stranger in another language or another decade who can check it without trusting us. Every derived artifact in Part IV, every invariant in Part I, and every one of the domains above follows from that one fact. The topology of a system determines more about its behavior than any amount of parameter tuning — and it can only be *read* because it was written down as data.

That is the claim this book has been circling, and the name for it is Metamodel: a small, fixed alphabet whose local composition rules generate an unbounded space of specific, checkable systems. Not one system that swallows every domain. Four primitives, remixed without limit. The palette is small on purpose, and the smallness is the whole reason it travels.
