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

Three functors appear throughout this book:

### The ODE Functor

Mass-action kinetics defines a monoidal functor from the discrete net category to the category of dynamical systems:

$$\text{ODE} : \mathcal{F}(N) \to \textbf{Dyn}$$

- Objects (markings) map to concentration vectors; places, the generators, to their coordinates
- Transitions (morphisms) map to rate equations: $v_t = k_t \prod_{p \in \text{pre}(t)} M(p)$
- The monoidal product maps to independence: $\text{ODE}(A \otimes B) = \text{ODE}(A) \times \text{ODE}(B)$

The last property is why the ODE decouples (Chapter 13). Independent components in the net produce independent differential equations. The decoupling lemma — each accumulator's equation $\dot{x}_i = 1 - n_i x_i$ depends only on its own state — is a consequence of the functor preserving $\otimes$.

### The ZK Functor

The Groth16 compilation defines a monoidal functor from the net category to the category of arithmetic circuits:

$$\text{ZK} : \mathcal{F}(N) \to \textbf{Circ}$$

- Markings map to witness vectors (committed via MiMC), one variable per place
- Transitions map to constraint systems (pre/post conditions as R1CS)
- The monoidal product maps to independent constraint blocks

This is why the ZK circuit is generic (Chapters 12–14). Changing the topology constants changes the functor's image — a different circuit for a different net — but the functor itself is the same construction. The proof system doesn't know whether it's proving tic-tac-toe or poker.

### The Analysis Functor

The incidence reduction (Chapter 13) defines a functor from the net category to strategic values:

$$\text{Val} : \mathcal{F}(N) \to \textbf{Vect}$$

- Markings map to vectors of strategic scores, one per place
- The mapping reads the diagonal of $BB^T$ — the round-trip endofunctor
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

## The Execution Zipper

The free SMC $\mathcal{F}(N)$ captures the morphism structure of a net — which compositions are valid, which transitions are independent. But it has no privileged present. Every marking is just another object, related to others by transition morphisms. Computation requires something more: a *focus*.

A **zipper** (Huet, 1997) decomposes a structure into a focused element and its surrounding context. For a Petri net execution, the decomposition is:

$$\text{Exec}(N) = \mathcal{L}(N) \times M \times \mathcal{R}(N, M)$$

where:

- **$M \in \mathbb{N}^P$** is the current marking — the hole. It is the object in $\mathcal{F}(N)$ at which execution is focused.
- **$\mathcal{L}(N)$** is the left context — the accumulated history of past firings. This is an element of the tropical semiring $(\mathbb{R}_{\max}, \oplus, \otimes)$ where $a \oplus b = \max(a,b)$ and $a \otimes b = a + b$. The tropical core compresses firing history into longest-path summaries: $\mathcal{L}_{ij}$ records the longest causal chain from transition $i$ to transition $j$. Tropical matrix multiplication is fast-forward: $\mathcal{L}^{(n)} = \mathcal{L}^{(n-1)} \otimes \mathcal{L}^{(1)}$.
- **$\mathcal{R}(N, M)$** is the right context — the set of transitions enabled at marking $M$. This is a predicate on $\mathcal{F}(N)$'s generators: $\mathcal{R}(N, M) = \{ t \in T \mid \text{pre}(t) \leq M \}$. It is computed fresh from the hole on every step.

### The Zipper Step

A single execution step is a zipper movement. When transition $t$ fires at marking $M$:

1. The hole updates: $M' = M - \text{pre}(t) + \text{post}(t)$
2. The left context grows: $\mathcal{L}' = \mathcal{L} \otimes_{\text{trop}} e_t$ where $e_t$ is the one-step matrix for $t$
3. The right context recomputes: $\mathcal{R}(N, M')$ — a new set of enabled transitions

The left context is closed and irreversible — tropical accumulation is lossy. The right context is open and ephemeral — it exists only relative to the current hole. The hole is the tense boundary between them.

### Relationship to the SMC

The zipper is not an alternative to the free SMC — it is a *refinement*. The SMC $\mathcal{F}(N)$ is the space of all valid compositions. The zipper is a pointed structure *within* that space: a position (the marking), a summary of the path taken to reach it (the tropical core), and the set of available next steps (enabled transitions).

Formally, the zipper arises from a **comonad** on the category of markings. The comonad $W : \mathbb{N}^P \to \mathbb{N}^P$ sends each marking to its context — the pair of left and right contexts surrounding it. The counit $\varepsilon : W(M) \to M$ extracts the current marking (the focus). The comultiplication $\delta : W(M) \to W(W(M))$ re-contextualizes — it says that the context itself has a context, which is how nested simulation (a DDM step within a DDM step) becomes well-defined.

The coKleisli category of this comonad — the category whose morphisms are $W(A) \to B$ — is the category of *context-dependent computations*. Every DDM engine in this book is a coKleisli morphism: it reads the full execution context (accumulated history, current marking, enabled transitions) and produces the next state.

### Tense Structure

The three components of the zipper correspond to three treatments of time:

| Component | Tense | Algebraic Structure | Property |
|-----------|-------|-------------------|----------|
| $\mathcal{L}(N)$ | Past | Tropical semiring | Closed, irreversible, lossy |
| $M$ | Present | Free commutative monoid $\mathbb{N}^P$ | The universe — defines what "past" and "future" mean |
| $\mathcal{R}(N, M)$ | Future | Predicate on generators | Open, recomputed, ephemeral |

This contrasts with two established frameworks. Schultz and Spivak's temporal type theory (2019) treats time as a parameter — an interval domain indexing a presheaf. The current moment is a point on the index, with no special status. Prior's tense logic treats time as a modality — operators $\square$ (always in the past) and $\diamond$ (sometime in the future) shift perspective relative to an implicit now. Both frameworks derive the present from something else.

The zipper inverts this: the present is foundational. The marking $M$ is not a point on a timeline or an implicit reference — it is the universe relative to which past ($\mathcal{L}$) and future ($\mathcal{R}$) are both defined. Move the hole and you are in a different universe.

This is the formal content behind the "scope boundary" noted in Chapter 21. The SMC sees morphism structure. The zipper sees execution state. Genovese et al. (2019) gave independent evidence for this boundary: forcing a functorial adjunction between nets and free SMCs breaks practical computational requirements — precisely because the adjunction cannot accommodate the mutable focus that computation demands.

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

In zipper terms, the contextual boundary is exactly where $\mathcal{R}$ lives. A read arc *is* a predicate on the marking that $C$ cannot express, recomputed every step — which is why the right context of the execution zipper is a predicate rather than a morphism, and why it sits outside $\mathcal{F}(N)$ rather than inside it.

**The circuit dissolves one boundary and not the other.** The standard encoding of a read arc as a self-loop (consume, then produce back) is inequivalent under partial-order semantics: it serializes firings the read arc allowed to be concurrent, so the unfolding changes (Vogler, Semenov & Yakovlev, 1998). Under interleaving semantics the reachability set is identical. A Groth16 proof certifies one firing, and one firing has only interleaving semantics; so inside the circuit the self-loop is exact, and a guard pulled into the proof as a range check on an auxiliary witness is a faithful encoding. The $\rho$ boundary survives compilation unchanged, because the circuit is a compilation of $C$.

**Throughput composes as a bound, not a value.** For open nets glued along a boundary place, $\lambda(A ;_p B) \geq \max(\lambda(A), \lambda(B))$, with equality only when no cycle through the glue has a larger mean weight than the best local cycle. Gluing creates cycles that belong to neither component, so throughput cannot be read off the parts. What can be: the lower bound for free, and Karp's algorithm (1978) run over the glue cycles only. Conservation and the circuit compose outright; $\lambda$ does not.

## The 2-Categorical View

The full ecosystem forms a 2-category:

- **0-cells** (objects): typed nets (WorkflowNet, ResourceNet, etc.)
- **1-cells** (morphisms): typed links between nets
- **2-cells** (morphisms between morphisms): natural transformations — the ZK proofs, sealed invariants, and analysis results that witness properties of the links

Composition of 1-cells is the CompositeNet construction. Composition of 2-cells is how proofs compose: proving component A correct, proving component B correct, and proving the link between them preserves both properties.

This is the assume-guarantee reasoning from Chapter 4 in categorical language. Each component's seal is a 2-cell witnessing its properties. Composition verification checks that the 1-cells (links) are compatible with the 2-cells (seals).

The 2-categorical structure guarantees this for *safety* properties, and the guarantee is one-directional. Because the pushout only identifies morphisms, every composite computation restricts to a computation of each component — so a property of the form "no reachable marking looks like this" survives, and P-invariants of a component are still derivable from the composite's incidence matrix. Liveness does not survive: "this transition can always eventually fire" is a statement about what the component *can* do, and a quotient can take that away.

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
- **Huet, G.** (1997). *The Zipper.* Journal of Functional Programming, 7(5), 549–554. The original zipper data structure — a purely functional way to represent a focused position within a tree. The execution-state decomposition in Chapter 21 generalizes this to Petri net markings.
- **Schultz, P. & Spivak, D.I.** (2019). *Temporal Type Theory: A Topos-Theoretic Approach to Systems and Behavior.* Birkhäuser. Treats time as a parameter over interval domains — a complementary perspective to the zipper's treatment of the present as a universe.
- **Genovese, F., Gryzlov, A., Herold, J., Perone, M., Post, E. & Videla, A.** (2019). *Computational Petri Nets: Adjunctions Considered Harmful.* arXiv:1904.12974. Shows that forcing a functorial adjunction between nets and free SMCs breaks practical computational requirements — formal evidence for the scope boundary discussed in Chapter 21.
