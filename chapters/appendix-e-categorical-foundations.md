# Appendix E: Categorical Foundations

This appendix provides the formal categorical framework underlying the constructions in this book. It is not required for any chapter — every result in the main text is stated and used without category theory. But for readers with some background in abstract algebra, the categorical perspective explains *why* the techniques compose so cleanly.

## Symmetric Monoidal Categories

A **symmetric monoidal category** (SMC) is a category $\mathcal{C}$ equipped with:

- A bifunctor $\otimes : \mathcal{C} \times \mathcal{C} \to \mathcal{C}$ (the monoidal product)
- A unit object $I$ such that $A \otimes I \cong A \cong I \otimes A$
- Natural isomorphisms for associativity: $(A \otimes B) \otimes C \cong A \otimes (B \otimes C)$
- A symmetry: $\sigma_{A,B} : A \otimes B \cong B \otimes A$

satisfying coherence conditions (the pentagon and hexagon diagrams).

In plain terms: objects can be placed side by side ($\otimes$), the grouping doesn't matter (associativity), doing nothing is an option ($I$), and the order doesn't matter (symmetry).

## Petri Nets as Free SMCs

The central theorem, due to Meseguer-Montanari (1990) and refined by Sassone (1995):

> **Theorem.** A Petri net $N = (P, T, \text{pre}, \text{post})$ generates a free symmetric monoidal category $\mathcal{F}(N)$ whose:
>
> - **Objects** are elements of $\mathbb{N}^P$ — multisets of places (i.e., markings)
> - **Morphisms** are equivalence classes of firing sequences
> - **Monoidal product** is multiset addition: $(m_1 \otimes m_2)(p) = m_1(p) + m_2(p)$
> - **Unit** is the empty marking: $I(p) = 0$ for all $p$
> - **Symmetry** permutes components of the multiset

"Free" means the only equalities between morphisms are those forced by the SMC axioms. The category captures exactly the net's concurrent behavior.

### Generators

Each transition $t \in T$ is a generating morphism:

$$t : \text{pre}(t) \to \text{post}(t)$$

where $\text{pre}(t)$ and $\text{post}(t)$ are the multisets of input and output places. Every morphism in $\mathcal{F}(N)$ is built from these generators by sequential composition ($;$) and parallel composition ($\otimes$).

### What "Free" Buys You

The free construction means:

1. **No hidden identifications.** Two firing sequences are equal in $\mathcal{F}(N)$ only if the SMC axioms force them to be. If two sequences differ in their causal structure, they're different morphisms.

2. **Universal property.** Any SMC-preserving interpretation of the net (ODE dynamics, ZK circuits, process algebra) factors uniquely through $\mathcal{F}(N)$. The free category is the most general interpretation.

3. **Concurrency is structural.** Two transitions that share no places compose via $\otimes$, not $;$. The monoidal product *is* concurrency — it's not simulated by interleaving.

## Functors: Structure-Preserving Maps

A **monoidal functor** $F : \mathcal{C} \to \mathcal{D}$ maps objects to objects and morphisms to morphisms while preserving the monoidal structure: $F(A \otimes B) \cong F(A) \otimes F(B)$.

Three kinds of functor appear throughout this book, and every derived artifact in Part IV is one of them:

### The ODE Functor

Mass-action kinetics defines a monoidal functor from the discrete net category to the category of dynamical systems:

$$\text{ODE} : \mathcal{F}(N) \to \textbf{Dyn}$$

- Objects (markings) map to concentration vectors; places, the generators, to their coordinates
- Transitions (morphisms) map to rate equations: $v_t = k_t \prod_{p \in \text{pre}(t)} M(p)$
- The monoidal product maps to independence: $\text{ODE}(A \otimes B) = \text{ODE}(A) \times \text{ODE}(B)$

The last property is why the ODE decouples (Chapter 13). Independent components in the net produce independent differential equations. The decoupling lemma — each accumulator's equation $\dot{x}_i = 1 - n_i x_i$ depends only on its own state — is a consequence of the functor preserving $\otimes$.

### The Proof Functors

Each proof form is a monoidal functor out of the same category: the Lean generator maps transitions to theorems about the step function, and the Groth16 compilation maps markings to witness vectors and transitions to R1CS constraint blocks. In both, the monoidal product maps to independent blocks. This is why neither proof form knows or cares which net it is proving — the construction is fixed and only the incidence matrix changes. Proof is a derived artifact of the document, not a separate encoding of it.

### The Analysis Functor

The incidence reduction (Chapter 13) defines a functor from the net category to strategic values:

$$\text{Val} : \mathcal{F}(N) \to \textbf{Vect}$$

- Markings map to vectors of strategic scores, one per place
- The mapping reads the diagonal of $BB^T$ — each place's outflow degree
- The monoidal product maps to independent evaluation: $\text{Val}(A \otimes B) = \text{Val}(A) \times \text{Val}(B)$

The independence of strategic evaluation per place — the product decomposition of the lens — is this functor preserving the monoidal structure.

## Lenses in Monoidal Categories

A **lens** in a monoidal category is a pair of morphisms satisfying coherence laws:

- **Get** $: S \to V$ — extract a view from the state
- **Put** $: S \otimes V \to S$ — update the state given a new view

subject to: Get-Put (putting what you got changes nothing) and Put-Get (getting after putting returns what you put).

The analysis net (Chapter 13) is a lens:

- **Get**: given a marking $M$, read strategic values from drain arc counts. The view is the vector $(1/n_1, 1/n_2, \ldots, 1/n_p)$ where $n_i = (BB^T)_{ii}$.
- **Put**: update the marking (make a move) and the drain counts shift — blocked win lines drop out, remaining connections determine new values.

### Product Decomposition

The key property: in a symmetric monoidal category, lenses over a product decompose:

$$\text{Lens}(A \otimes B) \cong \text{Lens}(A) \otimes \text{Lens}(B)$$

Each place gets its own lens, composed in parallel. Updating one place's view doesn't touch the others. This is why dynamic evaluation (Chapter 13) works — inject a new board state, and each position's lens independently resolves its new value from its remaining drain connections.

## Tropical Past, Predicate Future

The free SMC $\mathcal{F}(N)$ says which compositions are valid. It has no privileged present: every marking is just another object. Execution needs one — the marking a simulation reads on every step — and the honest way to add it is not a second structure bolted on beside the net, but a reading of what the net's own document already contains.

Every executing model in this book carries three kinds of data, and each has a tense:

| Data | Tense | Algebra | Property |
|------|-------|---------|----------|
| Write-once places and the event log | Past | Boolean semiring; in timed nets, $(\max, +)$ | Monotone: tokens arrive and never leave |
| The marking $M \in \mathbb{N}^P$ | Present | Free commutative monoid | The state the step is *about* |
| Guards and enabledness | Future | Predicates on $M$ | Recomputed on every step, never stored |

**The past is tropical.** History places (Chapter 6) are write-once: a token arrives and never leaves. A set of write-once places under firing is the boolean semiring — OR to accumulate, AND to detect a pattern — which is the degenerate case of the tropical semiring $(\max,+)$ that timed nets use to accumulate longest paths (Chapter 13). The same monotonicity is what makes the event log replayable (Chapter 20) and the schema safe to grow (Chapter 16): a fact, once absorbed, never changes. Irreversibility is not a policy; it is what the algebra does.

**The future is predicate.** A guard is not a value held in the model. It is a question — *am I enabled?* — answered from the current marking and discarded. Nothing about the future is stored, which is why nothing about it can drift: the only way two implementations disagree about what fires next is to disagree about $M$.

**The present is the boundary.** The marking is simultaneously the output of accumulation and the argument to every guard. Change it and a different past is relevant and a different future is enabled. Chapter 6 shows the split on a net small enough to see whole: history places past, board and turn places present, the eighteen move transitions and their guards future.

This is where the categorical reading stops and the Metamodel reading (Chapter 21) begins. The SMC is structure without execution. Execution is the same document read with tenses — and because all three tenses are data in that one document, an implementation that gets any of them wrong fails a byte-for-byte trace comparison rather than an argument.

## Net Types as Sub-SMCs

The five net types from Chapter 4 are sub-SMCs of the free category $\mathcal{F}(N)$, each with additional structure:

| Net Type | Additional Structure |
|----------|---------------------|
| WorkflowNet | Single-token restriction: morphisms are paths in a free category |
| ResourceNet | Conservation: P-invariants constrain the kernel of the incidence matrix |
| GameNet | Both: sequential paths with conserved resources |
| ComputationNet | Rate structure: a monoidal functor to $\textbf{Dyn}$ |
| ClassificationNet | Threshold structure: firing conditions as predicates on objects |

The typed links (EventLink, DataLink, TokenLink, GuardLink) are **functors between sub-SMCs**. An EventLink from a WorkflowNet to a ResourceNet is a functor that maps transition firings in the workflow to transition firings in the resource net — preserving the monoidal structure each type demands.

The CompositeNet is a **pushout**, not a coproduct. The coproduct is the disjoint sum of the components — every net side by side, nothing shared, which is what you get from a bundle with no links. Each link then glues along a boundary: a TokenLink or DataLink identifies two places, an EventLink identifies two transitions. Gluing a coproduct along a shared boundary is exactly a pushout, and the implementation computes it the way the universal property suggests — as a quotient by the equivalence relation the links generate, rather than pairwise. Taking the equivalence closure is what makes the construction associative: linking $A \to B$ and $B \to C$ yields one three-element class regardless of the order the links are given.

The distinction matters for behavior, not just bookkeeping. A coproduct is a product-like construction and preserves each component's behavior; a quotient identifies morphisms and therefore *removes* behavior. That is the formal reason composition refines rather than extends (Chapter 4): a rendezvous is a coequalizer, and coequalizers are not conservative.

## Where the Free Structure Stops: Two Boundaries

Every chapter that splits a net into a *core* and an *observer* (Chapters 6, 12, 13) is using one of two different boundaries, and they do not coincide. This section is the canonical statement; the chapters cite it rather than restate it.

**The $\rho$ boundary is algebraic and lives inside $C$.** For a transition $t$ write $\rho(t) = |\text{pre}(t)| \,/\, |\text{post}(t)|$, the ratio of tokens consumed to tokens produced. A core transition has $\rho = 1$ and, in the nets of this book, one producer and one consumer per place — the *timed event graph* property. A transition with $\rho > 1$ (a win detector consuming three history tokens and a turn token to produce one verdict) breaks that property. Three consequences follow, all of them facts about $C$: the tropical eigenvalue $\lambda$ is undefined across it (Chapter 13), the R1CS encoding stops being uniform (Chapter 12), and the ODE treats it as a sink. What does *not* follow is any failure of composition. The Meseguer–Montanari theorem above has no hypothesis about fan-in; a $\rho > 1$ transition is an ordinary generating morphism of $\mathcal{F}(N)$.

**The contextual boundary is categorical and lives outside $C$.** A *read arc* tests that a place holds a token without consuming it; an *inhibitor arc* tests that it holds none. Neither has an entry in the incidence matrix — $C$ records net change, and these arcs change nothing. Montanari and Rossi (1995) showed that nets with such arcs do not generate the free SMC: a transition's enablement depends on marking that its pre/post boundary does not express, so the morphism is no longer determined by its source and target. This breaks composition. It does not touch $\rho$: a contextual arc adds nothing to either count, so the guarded transition keeps whatever $\rho$ it had — $\rho = 1$ for `send`.

| | ordinary arcs only | has contextual arcs |
|---|---|---|
| $\rho = 1$ | core | guard — outside $\mathcal{F}(N)$, algebraically invisible |
| $\rho > 1$ | verdict transitions — composes freely, breaks $\lambda$ and uniform R1CS | crosses both |

The two worked examples in this book sit in the two off-diagonal cells, which is how the boundaries were conflated for some time: each example breaks exactly one thing. The overdraft guard of Chapter 12 is categorically an observer and algebraically core; the tic-tac-toe pattern collectors of Chapter 6 are algebraically observers and categorically core.

In tense terms, the contextual boundary is where the predicate future lives. A read arc *is* a predicate on the marking that $C$ cannot express, recomputed every step — which is why guards sit outside $\mathcal{F}(N)$ rather than inside it.

**The circuit dissolves one boundary and not the other.** The standard encoding of a read arc as a self-loop (consume, then produce back) is inequivalent under partial-order semantics: it serializes firings the read arc allowed to be concurrent, so the unfolding changes (Vogler, Semenov & Yakovlev, 1998). Under interleaving semantics the reachability set is identical. A Groth16 proof certifies one firing, and one firing has only interleaving semantics; so inside the circuit the self-loop is exact, and a guard pulled into the proof as a range check on an auxiliary witness is a faithful encoding. The $\rho$ boundary survives compilation unchanged, because the circuit is a compilation of $C$.

**Throughput composes as a bound, not a value.** For open nets glued along a boundary place, $\lambda(A ;_p B) \geq \max(\lambda(A), \lambda(B))$, with equality only when no cycle through the glue has a larger mean weight than the best local cycle. Gluing creates cycles that belong to neither component, so throughput cannot be read off the parts. What can be: the lower bound for free, and Karp's algorithm (1978) run over the glue cycles only. Conservation and the circuit compose outright; $\lambda$ does not.

## Composition Preserves Safety, Not Liveness

Composing two typed nets along a link is a pushout: it identifies places or transitions, and identifies nothing else. Because the pushout only identifies, every computation of the composite restricts to a computation of each component. So a *safety* property — "no reachable marking looks like this" — survives composition, and P-invariants of a component are still derivable from the composite's incidence matrix. This is the assume-guarantee reasoning of Chapter 4: each component's seal witnesses its own properties, and composition verification checks only the boundary.

The guarantee is one-directional. *Liveness* — "this transition can always eventually fire" — is a statement about what the component *can* do, and a quotient can take that away. Composition refines; it does not extend.

## References

- **Meseguer, J. & Montanari, U.** (1990). *Petri Nets are Monoids.* Information and Computation, 88(2). The original proof that firing sequences form a free commutative monoidal category.
- **Sassone, V.** (1995). *On the Category of Petri Net Computations.* TAPSOFT. Refines the construction to symmetric monoidal categories with the correct equivalence on morphisms.
- **Baez, J.C. & Stay, M.** (2011). *Physics, Topology, Logic and Computation: A Rosetta Stone.* New Structures for Physics, Springer. Places Petri nets alongside circuits, proofs, and programs as structures living in symmetric monoidal categories.
- **Fong, B.** (2015). *The Algebra of Open and Interconnected Systems.* PhD thesis, Oxford. Formalizes composition of open systems (including Petri nets) via decorated cospans in a symmetric monoidal category.
- **Master, J.** (2020). *Generalized Petri Nets.* The Topos Institute. Extends Petri net semantics to broader categorical frameworks.
- **Riley, M.** (2018). *Categories of Optics.* MSc thesis, Cambridge. The formal theory of lenses and optics in monoidal categories.
- **Montanari, U. & Rossi, F.** (1995). *Contextual Nets.* Acta Informatica, 32(6). Read arcs as contextual dependencies; nets with them do not generate the free SMC.
- **Vogler, W., Semenov, A. & Yakovlev, A.** (1998). *Unfolding and Finite Prefix for Nets with Read Arcs.* CONCUR 1998. The self-loop encoding of a read arc is inequivalent under unfolding semantics — and only there.
- **Karp, R.M.** (1978). *A characterization of the minimum cycle mean in a digraph.* Discrete Mathematics, 23(3). Computes the tropical eigenvalue; restricted to glue cycles, it is the compositional throughput step.
