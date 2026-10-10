# Appendix E: Categorical Foundations

This appendix sets out the categorical framework behind the constructions in this book. It is not required for any chapter — every result in the main text is stated and used without category theory. For readers with some background in abstract algebra, the categorical perspective gives a vocabulary for why some of the techniques compose, and for where composition stops.

Most of what follows is a reading, not a construction. The free-category theorem is published mathematics; the functors, lenses and sub-categories built on it below are asserted, not formally defined and checked. What is checked by machine — one Lean theorem about invariants under gluing, the generated per-model Lean proofs, and tests of the go-pflow and pflow-jl composition code — is named where it is used, and its status is tracked in the book's [`PROOF-ROADMAP.md`](https://github.com/pflow-xyz/book-pflow-xyz/blob/main/PROOF-ROADMAP.md).

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

2. **Universal property.** Any assignment of the net's places and transitions into another symmetric monoidal category extends uniquely to a monoidal functor out of $\mathcal{F}(N)$; the precise statement, with its strictness conditions, is Sassone's. The functors of the next section are offered as interpretations of this kind.

3. **Concurrency is structural.** Two transitions that share no places compose via $\otimes$, not $;$. The monoidal product *is* concurrency — it's not simulated by interleaving.

## Functors: Structure-Preserving Maps

A **monoidal functor** $F : \mathcal{C} \to \mathcal{D}$ maps objects to objects and morphisms to morphisms while preserving the monoidal structure: $F(A \otimes B) \cong F(A) \otimes F(B)$.

Three kinds of derived artifact in this book can be read as functors out of $\mathcal{F}(N)$. None has been constructed as one; each subsection says what the reading maps where, and what in the code backs it.

### The ODE Functor

Mass-action kinetics can be read as a monoidal functor from the discrete net category to a category of dynamical systems:

$$\text{ODE} : \mathcal{F}(N) \to \textbf{Dyn}$$

- Objects (markings) map to concentration vectors; places, the generators, to their coordinates
- Transitions (morphisms) map to rate equations: $v_t = k_t \prod_{p \in \text{pre}(t)} M(p)$, the product taken over the distinct input places of $t$, each to the first power whatever its arc weight; the weight scales only how much is consumed ([go-pflow `solver/ode.go`](https://github.com/pflow-xyz/go-pflow/blob/main/solver/ode.go))
- The monoidal product maps to independence: $\text{ODE}(A \otimes B) = \text{ODE}(A) \times \text{ODE}(B)$

The last property is the shape of the decoupling lemma behind the incidence reduction of Chapters 6 and 13. In the analysis net, each accumulator place has its own catalytic source and its own drain transitions and shares no place with any other accumulator, so the net is a $\otimes$ of components. With unit rates each accumulator obeys $\dot{x}_i = 1 - n_i x_i$, where $n_i$ is its drain count, and setting $\dot{x}_i = 0$ gives the equilibrium $x_i^* = 1/n_i$. The lemma is proved directly, without category theory, in the incidence-reduction paper ([pflow-rs `papers/integer-reduction/main.tex`](https://github.com/pflow-xyz/pflow-rs/blob/main/papers/integer-reduction/main.tex)), and the equilibria are asserted by the `test_integer_reduction_*` tests in [pflow-rs `crates/pflow/src/lib.rs`](https://github.com/pflow-xyz/pflow-rs/blob/main/crates/pflow/src/lib.rs) for tic-tac-toe, poker, Connect Four and Hex. The functor reading describes why the equations separate; it does not prove that they do.

### The Proof Forms

The two proof forms are the weakest fit for the functor reading. The Groth16 compilation maps markings to witness vectors and transitions to R1CS constraint blocks, one block per transition, with the fired transition selected by its public index (Chapters 12 and 13); the circuit construction is fixed, and only the topology arrays generated from the incidence matrix change from net to net. The Lean form is generated per model: pflow-polyglot's `tools/codegen -lang lean` emits theorems about the net's step function — the reachable set is closed under firing, every reachable marking is 1-safe, each structural P-invariant is conserved, and a conflict-free net has a unique deadlock — which the Lean kernel checks by `decide` over the reachable set ([`lean/proof.lean`](https://github.com/stackdump/pflow-polyglot/blob/main/lean/proof.lean)). What the two share is the property the Metamodel needs: neither proof is written by hand for a particular net. Change the document and the proof is regenerated from it.

### The Analysis Functor

The incidence reduction (Chapters 6 and 13) can be read as a functor from the net category to strategic values:

$$\text{Val} : \mathcal{F}(N) \to \textbf{Vect}$$

- Markings map to vectors of strategic scores, one per place
- The mapping reads the diagonal of $BB^T$, where $B$ is the 0/1 biadjacency matrix of positions against the constraints (win lines) they take part in; the diagonal entry is the position's constraint count, which becomes its drain count $n_i$ in the analysis net
- The monoidal product maps to independent evaluation: $\text{Val}(A \otimes B) = \text{Val}(A) \times \text{Val}(B)$

The paper calls incidence reduction a diagonal approximation to eigenvector centrality of $BB^T$, and the two agree on center > corner > edge on the tic-tac-toe boards it checks (3×3 by hand, 5×5 and 7×7 numerically); whether they can disagree on a less symmetric topology is listed there as open.

## Lenses in Monoidal Categories

A **lens**, in a category with products (the cartesian case of the optics in Riley's thesis), is a pair of morphisms:

- **Get** $: S \to V$ — extract a view from the state
- **Put** $: S \times V \to S$ — update the state given a new view

subject to: Get-Put (putting what you got changes nothing) and Put-Get (getting after putting returns what you put).

The analysis net of Chapters 6 and 13 has the shape of a lens; the laws have not been checked for it:

- **Get**: given a marking $M$, read strategic values from drain arc counts. The view is the equilibrium vector $(1/n_1, 1/n_2, \ldots, 1/n_p)$ where $n_i = (BB^T)_{ii}$.
- **Put**: update the marking (make a move) and the drain counts shift — blocked win lines drop out, remaining connections determine new values. This is the dynamic evaluation of Chapter 6.

### Product Decomposition

Lenses compose in parallel: a lens on $A$ and a lens on $B$ give a lens on $A \otimes B$. The converse fails in general — a lens on a product need not split into one lens per factor, because its view may depend on both.

The analysis lens splits for **Get**: by the decoupling lemma each accumulator's equilibrium depends only on its own drain count, so the view is one lens per position, composed in parallel. It does not split for **Put**. A move blocks win lines, and a blocked line removes a drain from every position on it, so one update changes several positions' views at once. The coupling lives in the drain counts, not in the ODE, which stays decoupled for any fixed set of drains; the dynamic evaluator of Chapter 6 accordingly recounts the live lines for every empty position after each move.

## Tropical Past, Predicate Future

The free SMC $\mathcal{F}(N)$ says which compositions are valid. It has no privileged present: every marking is just another object. Execution needs one — the marking a simulation reads on every step — and it can be added as a reading of what the net's own document already contains, rather than as a second structure beside the net.

Every executing model in this book carries three kinds of data, and each has a tense:

| Data | Tense | Algebra | Property |
|------|-------|---------|----------|
| Write-once places and the event log | Past | Boolean semiring; in timed nets, $(\max, +)$ | Monotone: no move takes a token back |
| The marking $M \in \mathbb{N}^P$ | Present | Free commutative monoid | The state the step is *about* |
| Guards and enabledness | Future | Predicates on $M$ | Recomputed on every step, never stored |

**The past is tropical.** History places (Chapter 6) are write-once: no move transition takes a token back, and the only transitions that consume one are the pattern collectors that turn a completed line into a verdict. Under the moves, a set of write-once places behaves as the boolean semiring — OR to accumulate, AND to detect a pattern — which is the degenerate case of the tropical semiring $(\max,+)$ that timed nets use to accumulate longest paths (the blog post [*Tropical Petri Nets*](https://blog.stackdump.com/posts/tropical-petri-nets)). The same monotonicity is what makes the event log replayable (Chapter 20) and the schema safe to grow (Chapter 16): a fact, once absorbed, never changes.

**The future is predicate.** A guard is a question — *am I enabled?* — answered from the current marking and discarded, never a value held in the model. Nothing about the future is stored, which is why nothing about it can drift: the only way two implementations disagree about what fires next is to disagree about $M$.

**The present is the boundary.** The marking is simultaneously the output of accumulation and the argument to every guard. Change it and a different past is relevant and a different future is enabled. Chapter 6 shows the split on a net small enough to see whole: history places past, board and turn places present, the eighteen move transitions and their guards future.

This is where the categorical reading stops and the Metamodel reading (Chapter 21) begins. The SMC is structure without execution. Execution is the same document read with tenses — and because all three tenses are data in that one document, an implementation that gets any of them wrong fails a byte-for-byte trace comparison (the golden trace of Chapter 20) rather than an argument.

## Net Types as Sub-SMCs

A net type classifies whole nets, so the level to read it at is one up from $\mathcal{F}(N)$: the category of open nets, whose morphisms are whole nets between boundary multisets of places. There, each of the five net types from Chapter 4 is a candidate sub-SMC — a class of nets with extra structure that should be closed under composition:

| Net Type | Additional Structure |
|----------|---------------------|
| WorkflowNet | Single-token restriction: morphisms are paths in a free category |
| ResourceNet | Conservation: P-invariants constrain the kernel of the incidence matrix |
| GameNet | Both: sequential paths with conserved resources |
| ComputationNet | Rate structure: a monoidal functor to $\textbf{Dyn}$ |
| ClassificationNet | Threshold structure: a transition needs $k$ tokens of evidence — an input arc of weight $k$ (a threshold written as a guard is contextual, and falls outside $\mathcal{F}(N)$; see below) |

Closure is asserted, not proved, for all five. The nearest thing to a check is for a different class, the settlement nets of the blog post *The Category Settle*: pflow-jl gives a membership predicate and tests closure under random wirings ([`src/settle.jl`](https://github.com/pflow-xyz/pflow-jl/blob/main/src/settle.jl)), and says itself that this is a test, not a proof.

The typed links (EventLink, DataLink, TokenLink, GuardLink) are the gluing data for composition. The CompositeNet is a **pushout**, not a coproduct. The coproduct is the disjoint sum of the components — every net side by side, nothing shared, which is what you get from a bundle with no links. Each link then glues along a boundary: a TokenLink or DataLink identifies two places, an EventLink identifies two transitions. Gluing a coproduct along a shared boundary is exactly a pushout, and go-pflow computes it the way the universal property suggests — as a quotient by the equivalence relation the links generate, rather than pairwise ([`metamodel/compose_flatten.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_flatten.go), with a union-find over places and another over transitions). Taking the equivalence closure is what makes the construction associative: linking $A \to B$ and $B \to C$ yields one three-element class regardless of the order the links are given. A GuardLink identifies nothing. go-pflow lowers it to a read or inhibitor arc across the boundary, or to a conjunct of the target transition's guard, so a GuardLink makes the composite contextual and takes it outside $\mathcal{F}(N)$ in the sense of the next section.

The distinction matters for behavior, not just bookkeeping. The coproduct preserves each component's behavior; a quotient identifies, and can therefore *remove* behavior. That is the formal reason composition refines rather than extends (Chapter 4): a rendezvous is a coequalizer, and a coequalizer can take behavior away.

## Where the Free Structure Stops: Two Boundaries

Every chapter that splits a net into a *core* and an *observer* (Chapters 6 and 12) is using one of two different boundaries, and they do not coincide. This section is the canonical statement; the chapters cite it rather than restate it.

**The $\rho$ boundary is algebraic and lives inside $C$.** For a transition $t$ write $\rho(t) = |\text{pre}(t)| \,/\, |\text{post}(t)|$, the ratio of tokens consumed to tokens produced. A core transition has $\rho = 1$ and, in the nets of this book, one producer and one consumer per place — the *timed event graph* property. A transition with $\rho > 1$ (a win detector consuming three history tokens and a turn token to produce one verdict) breaks that property. Three consequences follow, all of them facts about $C$: the tropical eigenvalue $\lambda$ is undefined across it (*Tropical Petri Nets*; the pflow-jl netting experiment below shows the one-producer-one-consumer property failing at exactly the fan-in places), the R1CS encoding stops being uniform (Chapter 12), and the ODE treats it as a sink. What does *not* follow is any failure of composition. The Meseguer–Montanari theorem above has no hypothesis about fan-in; a $\rho > 1$ transition is an ordinary generating morphism of $\mathcal{F}(N)$.

**The contextual boundary is categorical and lives outside $C$.** A *read arc* tests that a place holds a token without consuming it; an *inhibitor arc* tests that it holds none. Neither has an entry in the incidence matrix — $C$ records net change, and these arcs change nothing. Montanari and Rossi (1995) showed that nets with such arcs do not generate the free SMC: a transition's enablement depends on marking that its pre/post boundary does not express, so the morphism is no longer determined by its source and target. This breaks composition. It does not touch $\rho$: a contextual arc adds nothing to either count, so the guarded transition keeps whatever $\rho$ it had — $\rho = 1$ for the ERC-20 transfer of Chapter 12, which moves `amount` tokens from one balance to another.

| | ordinary arcs only | has contextual arcs |
|---|---|---|
| $\rho = 1$ | core | guard — outside $\mathcal{F}(N)$, algebraically invisible |
| $\rho > 1$ | verdict transitions — composes freely, breaks $\lambda$ and uniform R1CS | crosses both |

The two worked examples in this book sit in the two off-diagonal cells, which is what makes the boundaries easy to conflate: each example breaks exactly one thing. The overdraft guard of Chapter 12 is categorically an observer and algebraically core; the tic-tac-toe pattern collectors of Chapter 6 are algebraically observers and categorically core.

In tense terms, the contextual boundary is where the predicate future lives. A read arc *is* a predicate on the marking that $C$ cannot express, recomputed every step — which is why guards sit outside $\mathcal{F}(N)$ rather than inside it.

**The circuit dissolves one boundary and not the other.** The standard encoding of a read arc as a self-loop (consume, then produce back) is inequivalent under partial-order semantics: it serializes firings the read arc allowed to be concurrent, so the unfolding changes (Vogler, Semenov & Yakovlev, 1998). Under interleaving semantics the reachability set is identical. A Groth16 proof certifies one firing, and one firing has only interleaving semantics; so inside the circuit the self-loop is exact, and a guard pulled into the proof as a range check on an auxiliary witness is a faithful encoding. The $\rho$ boundary survives compilation unchanged, because the circuit is a compilation of $C$.

**Throughput composes as a bound, not a value.** For open nets glued along a boundary place, $\lambda(A ;_p B) \geq \max(\lambda(A), \lambda(B))$, with equality only when no cycle through the glue has a larger mean weight than the best local cycle. Gluing creates cycles that belong to neither component, so throughput cannot be read off the parts. What can be: the lower bound for free, and Karp's algorithm (1978) run over the glue cycles only. The circuit composes outright, and conservation composes under the compatibility condition of the next section; $\lambda$ does not.

## Composition Preserves Safety, Not Liveness

Composing two typed nets along links is a pushout: it identifies places or transitions, and identifies nothing else. What a component keeps depends on which it identifies.

When only transitions are identified (EventLinks), the fused transition fires only when every participant can, so every firing sequence of the composite, projected onto one component's transitions, is a firing sequence of that component alone. A *safety* property of the component — "no reachable marking looks like this" — then survives composition. go-pflow checks this on the Orders + Inventory bundle of Chapter 4, which is linked by two EventLinks: every reachable composite marking, projected onto Orders, is reachable in Orders alone (`TestProjectionRefinement`), and composition shrinks Inventory's reachable state space (`TestEventLinkRestrictsBehavior`), both in [`metamodel/compose_verify_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_verify_test.go). That is a check on one bundle, not a proof for every one.

When places are identified (TokenLinks, DataLinks), the other component can put tokens into the shared place or take them out, and the guarantee weakens. The netting experiment in pflow-jl ([`docs/netting-experiment.md`](https://github.com/pflow-xyz/pflow-jl/blob/main/docs/netting-experiment.md), tests in `test/test_netting.jl`) glues a $\rho = 3$ netting transition onto a three-party settlement cycle along the cycle's three pending places. The composite reaches a marking whose restriction to the cycle is all zeros, which the cycle alone can never reach; the cycle's bound of three tokens in flight survives only as an inequality, because the glued transition only removes tokens. P-invariants go the same way:

- *Restriction holds.* Every P-invariant of the composite, restricted to a component's places, is a P-invariant of that component — a component's transitions touch only its own places.
- *Extension by zero fails.* The cycle's invariant, all six places weighted 1, is not an invariant of the composite; the composite's only invariant weights the new `cleared` place 3, i.e. $(1,1,1,1,1,1,3)$.
- *Gluing compatible invariants holds, and is proved.* If $y_1$ and $y_2$ are P-invariants of the two parts and agree on the identified places, the $y$ they induce is a P-invariant of the glued net. This is the theorem `invariant_lift`, checked by the Lean kernel in core Lean 4 ([`proofs/PflowProofs/InvariantLift.lean`](https://github.com/pflow-xyz/book-pflow-xyz/blob/main/proofs/PflowProofs/InvariantLift.lean)). It covers gluing along places only, not transition fusion.

So the assume-guarantee reasoning of Chapter 4 — each component's seal witnesses its own properties, and composition verification checks only the boundary — holds as stated for rendezvous along transitions. For shared places the boundary check has real work to do: a component's conservation law is guaranteed to carry over when the other side has an invariant that agrees with it on the shared places, and not otherwise.

*Liveness* — "this transition can always eventually fire" — is a statement about what the component *can* do, and a quotient can take that away under either kind of link. In the netting experiment every transition of the cycle is live; in the composite, the four firings `send_ab`, `send_bc`, `send_ca`, `net_abc` reach a dead marking, and none of the seven transitions is live. Composition refines; it does not extend.

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
