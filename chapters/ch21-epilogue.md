# Epilogue: What the Abstraction Sits On

**Learning objective**: Name the layered architecture this book actually built, see the six applications through the lens of net types, and identify the open problems worth pursuing.

This book promised a universal abstraction. It delivered one — but along the way, especially in Chapter 13, something shifted. The Petri net turned out to be a layer, not the foundation. The most powerful result in the book — that strategic value emerges from graph connectivity with no training data, no domain knowledge, and no Petri net firing semantics — lives underneath the formalism the book is named for.

That deserves to be said plainly, not buried in a subsection.

## The Four-Layer Stack

The book built a stack, one layer at a time, without naming it as such until now:

```
Layer 4: ZK Verification     (Chapters 12-13)
         Cryptographic proof that a transition was valid.
         The stoichiometry matrix becomes circuit constraints.

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
- **ZK verification** adds proof. The stoichiometry matrix defines circuit constraints. A state transition is either provably valid or unprovable. Trust moves from "I ran the code" to "here is a cryptographic attestation."

The book introduced these layers bottom-up in Parts I-III, but the reader encounters the stack's true shape only in Chapter 13, when the rate auto-derivation reveals that the bottom layer — pure graph connectivity — carries more information than expected. The classic tic-tac-toe heuristic (center > corner > edge) falls out of counting connections. No game theory. No training. Just topology.

The Petri net is not the foundation. It is the modeling layer — the place where graph structure acquires semantics. That's valuable. It's just not the whole story.

## Six Applications, Five Types

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

The taxonomy isn't a labeling scheme imposed from outside. It's a description of structural invariants that the topology either has or doesn't. This is the same insight as "it's graph theory, not Petri net theory," seen from a different angle: the structure carries the meaning.

## What the Book Proved

Three claims survived from Chapter 1 to Chapter 18:

**Small models beat black boxes.** Every application in this book is inspectable. You can look at the tic-tac-toe topology and count win lines. You can read the stoichiometry matrix and see the differential equations. You can audit the ZK circuit and verify what it proves. At no point did you need to trust a model you couldn't read. This is the opposite of the machine learning approach, where the knowledge is in the weights and the weights are opaque. The cost is that Petri net models require a human to design the topology. The benefit is that the topology is the explanation.

**One formalism, multiple tools.** The JSON-LD model format (Chapter 14) is processed identically by the visual editor (Chapter 15), the code generator (Chapter 16), the Go library (Chapter 17), and the ZK compiler (Chapter 13). Dual implementation (Chapter 18) verifies that independent implementations agree. This isn't a theoretical property — it's tested, deployed, and running on-chain.

**Topology is primary, rates are secondary.** Change the rate constants and the system's quantitative behavior shifts. Change the topology and the system becomes a different system. This inversion — structure over parameters — holds across all six applications and both modes (combinatorial and continuous). It's the book's most load-bearing claim, and Chapter 13 gave it a precise formulation.

## What the Book Didn't Solve

The limitations section of Chapter 13 was honest, but it was framed as caveats. They're better understood as open problems.

**Multi-hop connectivity.** The rate auto-derivation counts direct connections: candidate → unique output → target input. For tic-tac-toe (depth 1), this is sufficient. For chess (depth varies), it captures material value but misses tactics. For Go (depth 19×19), it captures almost nothing. The question: can multi-hop reachability analysis — T-invariants, unfoldings, or iterative message-passing over the bipartite graph — extend the one-hop algorithm to deeper games? This is a graph theory question, not a Petri net question, which is itself instructive.

**Weighted targets.** The algorithm treats every target connection as weight 1. A checkmate path and a pawn capture score the same. The fix seems straightforward — assign importance weights to targets — but the principled question is where those weights come from. Can topology derive them recursively? Or does heterogeneous objective weighting require domain knowledge that the graph alone cannot supply?

**Dynamic rates.** Topology-derived rates are static. A corner's strategic value changes mid-game when it completes a fork threat. The tactical scoring layer in Chapter 6 handles this for tic-tac-toe, but it's an add-on, not part of the rate derivation. Can the rate formula incorporate state-dependent topology — recomputing connectivity over the *reachable* subgraph rather than the full graph? This would unify the strategic (topology) and tactical (state) layers.

**Circuit scaling.** The selector-based encoding grows as O(|P| × |T|). The tic-tac-toe circuit has ~24,500 constraints. A net with 1,000 places and 500 transitions would have ~12.5 million constraints — feasible with current hardware but pushing limits. Recursive proof composition (proving batches of transitions, then proving the batch proofs) is the likely path forward. The Petri net structure may help here: independent subnets can be proved in parallel and composed.

**Composition verification.** Chapter 4 described cross-schema composition with EventLinks, DataLinks, TokenLinks, and GuardLinks. Chapter 13 described single-net ZK verification. The gap: proving that a composed system of multiple nets preserves the invariants of each component. Assume-guarantee reasoning suggests this is tractable — each component's proof is independent, and composition only needs to verify the boundaries. But the ZK pipeline doesn't implement this yet.

## What the ODE Was Actually Computing

The four-layer stack describes *what* the book built. This section names *what it computes* — and the answer is more precise than "equilibrium concentrations."

### The Round-Trip Matrix

Chapter 2 introduced the incidence matrix $N$ with its input and output components. But there's a simpler object underneath. Let $B$ be the **input adjacency matrix** of the bipartite graph: $B[i,j] = 1$ if place $i$ is an input to transition $j$, and 0 otherwise. This is the "who feeds whom" structure at Layer 1 — pure graph connectivity, no Petri net semantics.

The matrix product $BB^T$ is a square matrix on places:

$$(BB^T)[i,j] = \sum_k B[i,k] \cdot B[j,k]$$

This counts the number of transitions that places $i$ and $j$ both feed into — their co-occurrence through the transition layer. The diagonal entry $(BB^T)[i,i]$ counts how many transitions consume from place $i$: its outflow degree.

$BB^T$ is a round-trip: start at places, pass through transitions, return to places. In categorical language, $B$ is a morphism from the place space to the transition space and $B^T$ is its adjoint going back. The composite $BB^T$ is an **endofunctor** — a mapping from the place space to itself. It encodes how the transition layer mediates relationships among places.

### The Diagonal Is the Invariant

Now look at what the poker analysis net computed in Chapter 13. Each value place $\text{val}_H$ had one play transition producing tokens (constant inflow) and $n$ drain transitions consuming tokens (outflow proportional to $n$). The drain count $n$ is exactly $(BB^T)[\text{val}_H, \text{val}_H]$ — the diagonal entry for that place. And the equilibrium concentration was:

$$\text{val}_H^* = \frac{1}{n} = \frac{1}{(BB^T)_{ii}}$$

The ODE system relaxed to a steady state that depends only on the diagonal of $BB^T$. Not the full matrix — just the diagonal. Each place's equilibrium is determined by its own connectivity, independent of every other place.

This independence is not accidental. The catalytic-pump construction decouples the places: each value accumulator has its own source, its own drains, and no cross-talk with other accumulators. The full matrix $BB^T$ has off-diagonal entries — multiple value places might share drain transitions in a more complex net — but the construction projects those away. At equilibrium, only the diagonal survives.

### The Categorical Trace

In category theory, the **trace** of an endomorphism extracts the diagonal information — it maps a square matrix to the sum of its diagonal entries, discarding everything off-diagonal. For a finite-dimensional endomorphism $f$, $\text{tr}(f) = \sum_i f_{ii}$.

The ODE system computes something stronger than the scalar trace: it computes each diagonal entry individually. The equilibrium vector $M^*$ is a function of the diagonal of $BB^T$ alone. The system *relaxes into reading only the diagonal* of the round-trip endofunctor.

This is what Chapter 13's rate auto-derivation was doing all along. When the algorithm counted drain connections per candidate and derived rate constants from those counts, it was reading $(BB^T)_{ii}$ for each candidate. When the ODE solver ran those rates to equilibrium, it was dynamically computing the same readout. Both paths arrive at the diagonal of $BB^T$. The rate derivation computes it statically by counting. The ODE computes it dynamically by relaxing. They agree because they are computing the same invariant of the same structure.

The tic-tac-toe result — center (4) > corner (3) > edge (2) — is a diagonal readout of $BB^T$ restricted to win-line connectivity. The poker result — straight flush (1 drain) > four of a kind (2 drains) > ... > high card (32 drains) — is the inverse diagonal readout. Both are the categorical trace of the entity-constraint endofunctor, computed through simulation.

### Portability

The construction — bipartite structure, round-trip endofunctor, diagonal readout via ODE — has nothing to do with games. Games are where the book validated it. But the pattern applies to anything expressible as "entities participate in constraints":

**Financial networks.** Assets are places, portfolio allocations are transitions. $(BB^T)_{ii}$ counts how many portfolios asset $i$ participates in — its exposure. The ODE equilibrium ranks assets by systemic importance.

**Supply chains.** Components are places, products are transitions. The diagonal counts how many products each component feeds. The equilibrium identifies strategic bottlenecks without supply chain domain knowledge.

**Access control.** Principals are places, permission sets are transitions. The diagonal measures privilege surface area. Higher connectivity means higher risk exposure.

**Governance.** Voters are places, decisions are transitions. The diagonal measures structural influence — how many decision points each voter participates in.

In every case, the recipe is identical: encode the entity-constraint structure as a bipartite graph, form $BB^T$, and read the diagonal — either by counting (static analysis) or by ODE relaxation (dynamic computation). The equilibrium concentrations rank entities by structural importance within the constraint network. No training data. No domain heuristics. Just topology.

## The Structure Underneath

The four-layer stack describes the book's architecture. The categorical trace names the key invariant. But there's a unifying structure that explains *why* all the layers compose so cleanly — why ODE analysis transfers from tic-tac-toe to poker, why ZK circuits work for any net, why typed schemas compose without surprises.

That structure is the **symmetric monoidal category** (SMC).

### Transitions as Morphisms

A Petri net transition consumes tokens from input places and produces tokens into output places. In categorical terms, this is a morphism — a map from domain to codomain. A transition $t$ with inputs $\{p_1, p_2\}$ and outputs $\{q_1, q_2, q_3\}$ is:

$$t : p_1 \otimes p_2 \to q_1 \otimes q_2 \otimes q_3$$

The $\otimes$ is the monoidal product — it means "these things exist side by side." Two tokens in separate places aren't combined or merged; they coexist independently. This is how Petri nets express concurrency: $p_1 \otimes p_2$ means both places are marked, and both tokens are available simultaneously.

Places are the objects. Transitions are the morphisms. The multiset of tokens across all places — the marking — is an object in the free commutative monoid generated by the places.

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

- **ZK proofs are generic** (Chapter 12) because the circuit encodes the incidence matrix — the SMC's morphism structure — as arithmetic constraints. Swapping topology constants gives proofs for a different game, a different workflow, a different token standard.

- **Typed composition is monotonic** (Chapter 4) because adding a new schema to a CompositeNet is adding a new object to the category. The monoidal product guarantees it can't break existing schemas.

- **Mass-action kinetics is well-behaved** (Chapter 3) because it's a monoidal functor from the discrete SMC to continuous dynamics. It preserves the product structure: independent components stay independent.

### The Ecosystem Through Categorical Eyes

Step back and look at what this book built. The layers compose because they form a categorical structure:

| Layer | Categorical Role | Book Content |
|-------|-----------------|--------------|
| Theory | Objects | Token language, net types, JSON-LD, DDM |
| Models | Morphisms | Coffee shop, tic-tac-toe, Hold'em, sudoku, enzyme kinetics |
| Analysis | Functors | Incidence reduction, ODE signatures, P-invariants |
| Proofs | Natural transformations | ZK proofs, lenses, sealed invariants |

Each layer composes with the ones above and below. Models compose via typed links. Analysis composes via functor composition. Proofs compose via vertical composition of natural transformations.

This isn't imposed structure. The book accumulated one chapter at a time, each solving a specific problem. But Petri nets carry symmetric monoidal structure inherently, and everything built on them inherits it. The coherence shows up as: techniques from one chapter transfer cleanly to another. The ODE analysis that works on tic-tac-toe works on poker. The ZK circuit that proves tic-tac-toe transitions proves any Petri net transition. The composition rules that wire order processing to inventory wire any two schemas together.

The category theory isn't a framework the book adopted. It's the structure that was always there — the reason the abstraction turned out to be universal.

For readers who want the formal treatment — the free SMC construction, the precise functor definitions, and the lens product decomposition theorem — see Appendix E.

## The Premise, Revisited

Chapter 1 opened with a complaint: informal models fail because they don't capture the structure of the systems they represent. Concurrency is an afterthought. Resources are invisible. State is implicit.

Petri nets fix this by making structure explicit. Places hold state. Transitions change it. Arcs constrain what can flow where. Conservation laws fall out of the topology. The model is the specification.

But the deeper lesson — the one that emerged through writing this book, not before it — is that the Petri net formalism is itself a layer over something simpler. The structure that matters most is the directed bipartite graph. The Petri net adds semantics to that graph. The ODE adds dynamics. The ZK circuit adds proof. Each layer is useful. None is the whole story. And the invariant that connects them — the diagonal of $BB^T$, computed dynamically by the ODE and verified cryptographically by the ZK circuit — is a categorical property of the bipartite structure itself. It exists whether you call the formalism a Petri net, a chemical reaction network, or a bipartite constraint graph.

If there's a single sentence version of what this book argues, it might be: **the topology of a system — what connects to what, through what — determines more about its behavior than any amount of parameter tuning, training data, or runtime optimization.** The Petri net is one way to read that topology. The ODE is one way to compute its invariants. The diagonal of $BB^T$ is one such invariant — and it turned out to be the one that matters most. The topology was always there, waiting to be read.
