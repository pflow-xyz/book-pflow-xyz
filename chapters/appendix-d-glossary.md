# Appendix D: Glossary

**Arc**
: A directed connection between a place and a transition (input arc) or a transition and a place (output arc). Arcs define the flow relation of the net. Each arc has a weight (default 1) specifying how many tokens are consumed or produced.

**Bipartite graph**
: A graph whose nodes divide into two disjoint sets, with edges only between sets, never within. Petri nets are bipartite: places connect to transitions, transitions connect to places, but places never connect directly to places.

**Bounded**
: A net is bounded if every place has a finite maximum token count across all reachable markings. A net is $k$-bounded if no place ever exceeds $k$ tokens. Boundedness is a safety property — it guarantees the system can't accumulate unbounded resources.

**CID (Content Identifier)**
: A hash-based identifier for content-addressed data. In the pflow ecosystem, CIDs identify JSON-LD models deterministically — the same model always produces the same CID regardless of where or when it's computed (Chapter 16).

**Compression ratio ($\rho$)**
: For a transition, tokens consumed divided by tokens produced. $\rho = 1$ is the reversible core; $\rho > 1$ is an observer that destroys state to produce a verdict. A property of the incidence matrix column, so every analysis of $C$ can see it. See *Core–observer boundary*.

**Conservation law**
: An equation $\mathbf{y}^T \mathbf{C} = \mathbf{0}$ where $\mathbf{y}$ is a vector of weights and $\mathbf{C}$ is the incidence matrix. Conservation laws prove that weighted sums of token counts remain constant across all firings. Example: in an SIR model, $S + I + R = N$ for all time.

**Context (code generation)**
: The universal intermediate representation in petri-pilot's code generation pipeline. The Context computes everything templates need from the raw model — derived structures, feature flags, helper methods. Templates are pure functions from Context to source code (Chapter 18).

**Contextual arc**
: A read arc (fires only if a place holds a token, without consuming it) or an inhibitor arc (fires only if it holds none). Neither appears in the incidence matrix, since neither changes the marking. Nets with contextual arcs do not generate the free symmetric monoidal category of ordinary nets (Montanari & Rossi, 1995); they are the categorical half of the core–observer boundary, and the natural home of guards.

**Continuous relaxation**
: Replacing discrete token counts with continuous concentrations and firing rules with differential equations. The Petri net becomes a system of ODEs via mass-action kinetics, enabling simulation with numerical solvers (Chapter 3).

**Core–observer boundary**
: Two distinct boundaries that older material treated as one. The *algebraic* boundary ($\rho$) lies inside $C$ and separates transitions the tropical eigenvalue and uniform R1CS can handle from those they cannot; it does not affect composition. The *contextual* boundary (read/inhibitor arcs) lies outside $C$ and breaks free composition; it does not affect $\rho$. Canonical statement in Appendix E.

**Deadlock**
: A marking where no transition is enabled. The system is stuck — no further computation is possible. Deadlock detection is a key analysis performed by the reachability package.

**Enabled**
: A transition is enabled at a marking $\mathbf{m}$ if every input place has at least as many tokens as the corresponding arc weight: $\mathbf{m}(p) \geq W(p, t)$ for all input places $p$. An enabled transition may fire; a disabled one cannot.

**Equilibrium**
: A state where all derivatives are approximately zero — the system has stopped changing. The solver's equilibrium detector monitors the derivative norm and stops early when the system reaches steady state.

**Event sourcing**
: A state management pattern where state is computed by replaying an immutable log of events from the initial state: $\text{State}(t) = \text{fold}(\text{apply}, \text{initialState}, \text{events}[0..t])$. In Petri net terms, events are transition firings and state is the marking (Chapter 20).

**Firing rule**
: The execution semantics of Petri nets. When a transition fires, it atomically removes tokens from input places and adds tokens to output places according to arc weights. The new marking is $\mathbf{m}' = \mathbf{m} + \mathbf{C} \cdot \mathbf{e}_t$ where $\mathbf{e}_t$ is the unit vector for transition $t$.

**Groth16**
: A zero-knowledge proof system used in pflow for verifying Petri net transitions without revealing state. Produces constant-size proofs with fast verification, at the cost of a per-circuit trusted setup (Chapter 12).

**Guard**
: A boolean expression attached to a transition that must evaluate to true for the transition to be enabled, in addition to the standard token requirements. Guards extend the firing rule with data-dependent conditions.

**Incidence matrix**
: The matrix $\mathbf{C} = \mathbf{C}^+ - \mathbf{C}^-$ where $\mathbf{C}^+$ is the output matrix (tokens produced) and $\mathbf{C}^-$ is the input matrix (tokens consumed). Each column represents a transition; each row represents a place. The entry $\mathbf{C}_{ij}$ is the net change in place $i$ when transition $j$ fires (Chapter 2).

**Interleaving vs. partial-order semantics**
: Two readings of concurrent behavior. Interleaving semantics considers one firing at a time and asks which markings are reachable; partial-order (unfolding) semantics additionally records which firings were independent. Encoding a read arc as a consume-then-restore self-loop is exact under the first and inequivalent under the second (Vogler, Semenov & Yakovlev, 1998). A single-transition ZK proof has only interleaving semantics.

**JSON-LD**
: JSON for Linked Data. The pflow ecosystem uses JSON-LD as its model interchange format, with `@context: "https://pflow.xyz/schema"`. JSON-LD makes models self-describing and content-addressable (Chapter 16).

**Liveness**
: A transition is live if it can eventually fire from every reachable marking (possibly after other transitions fire first). A net is live if all transitions are live. Liveness means the system never permanently loses capability.

**Marking**
: The distribution of tokens across all places at a given moment; the state of the Petri net. Written as a vector $\mathbf{m}$ where $\mathbf{m}(p)$ is the token count at place $p$. The initial marking $\mathbf{m}_0$ is the starting state.

**Mass-action kinetics**
: A rate law from chemistry where the rate of a reaction is proportional to the product of reactant concentrations. Applied to Petri nets: the continuous firing rate of transition $t$ is $r_t \prod_{p \in \text{inputs}(t)} m(p)^{w(p,t)}$ where $r_t$ is the rate constant and $w(p,t)$ is the arc weight (Chapter 3).

**MiMC (Minimum Multiplicative Complexity)**
: A hash function designed for efficiency inside arithmetic circuits. Used in pflow's zero-knowledge proofs to hash Petri net markings with minimal constraint count (Chapter 12).

**Monotonic expansion**
: The property that tokens can only be created, never destroyed. Used in game nets where history places (prefixed with `_`) accumulate records of moves without ever losing them (Chapter 6).

**ODE (Ordinary Differential Equation)**
: An equation relating a function to its derivatives. The continuous relaxation of a Petri net produces a system of ODEs: $\frac{d\mathbf{m}}{dt} = \mathbf{C} \cdot \mathbf{r}(\mathbf{m})$ where $\mathbf{r}(\mathbf{m})$ is the rate vector (Chapter 3).

**P-invariant**
: A vector $\mathbf{y} \geq \mathbf{0}$ such that $\mathbf{y}^T \mathbf{C} = \mathbf{0}$. The weighted sum $\mathbf{y}^T \mathbf{m}$ is constant for all reachable markings. P-invariants prove conservation laws — they identify quantities that are preserved by every transition firing (Chapter 2).

**Petri net**
: A mathematical modeling language consisting of places, transitions, arcs, and tokens. Formally, a tuple $(P, T, F, W, \mathbf{m}_0)$ where $P$ is a set of places, $T$ is a set of transitions, $F \subseteq (P \times T) \cup (T \times P)$ is the flow relation, $W: F \to \mathbb{N}$ is the weight function, and $\mathbf{m}_0$ is the initial marking.

**Place**
: A node in a Petri net representing a condition, resource, or state. Places hold tokens. Drawn as circles. In the continuous relaxation, token counts become continuous concentrations.

**Reachability graph**
: The graph of all markings reachable from the initial marking through sequences of transition firings. Each node is a marking, each edge is a transition firing. The reachability graph may be infinite for unbounded nets.

**SDE (Stochastic Differential Equation)**
: Here, specifically the chemical Langevin equation: continuous state evolving under the same propensities SSA uses, but with intrinsic firing noise built into the diffusion term rather than sampled as discrete events. Cheap to sweep like the ODE, with an honest variance band like SSA — the regime between them. Distinct from a layer of exogenous Brownian motion added to an ODE for external uncertainty (a price feed, say), which answers a different question (Chapter 3).

**SSA (Stochastic Simulation Algorithm)**
: The discrete engine's execution method (Gillespie's direct method): pick the next transition to fire and the time until it fires, both drawn from the model's own propensities, then repeat. Produces one exact discrete sample path, or an ensemble of them, rather than a mean-field trajectory — the right tool when small counts or gating (read arcs, inhibitors, reachable capacities, guards) make the continuous relaxation inapplicable (Chapter 3).

**State equation**
: The algebraic relation $\mathbf{m}' = \mathbf{m}_0 + \mathbf{C} \cdot \boldsymbol{\sigma}$ where $\boldsymbol{\sigma}$ is the firing count vector (how many times each transition has fired). A necessary but not sufficient condition for reachability — the actual firing sequence must also be valid.

**State root**
: A hash of the simulation output used for cross-implementation verification. If the Go server and JavaScript browser produce the same state root from the same model, the implementations agree (Chapter 20).

**Token**
: A marker in a place. Tokens represent the presence of a resource, the truth of a condition, or a unit of something being processed. In the discrete model, tokens are non-negative integers. In the continuous relaxation, they become non-negative real numbers.

**Transition**
: A node in a Petri net representing an event, action, or chemical reaction. Transitions consume tokens from input places and produce tokens in output places. Drawn as rectangles. In the continuous relaxation, transitions have firing rates.

**Tsit5**
: Tsitouras 5th-order Runge-Kutta method with embedded 4th-order error estimator. The default ODE solver in go-pflow and pflow.xyz. Uses adaptive step size control for automatic accuracy management (Appendix A).

**URDNA2015**
: Universal RDF Dataset Normalization Algorithm 2015. Used in pflow to canonicalize JSON-LD models before hashing, ensuring that semantically equivalent documents produce identical content identifiers regardless of key order or formatting (Chapter 16).

**Weight**
: The number of tokens consumed or produced by an arc when its transition fires. A weight of $k$ on an input arc means the transition requires at least $k$ tokens in the source place and removes $k$ tokens when firing. Default weight is 1.

**Zero-knowledge proof**
: A cryptographic protocol where a prover convinces a verifier that a statement is true without revealing any information beyond the truth of the statement. In pflow, ZK proofs verify that a Petri net transition was validly fired without revealing the marking (Chapter 12).

---

## Ecosystem Vocabulary

Terms coined in this book and the blog it grew from, rather than inherited from Petri net theory. Each entry names where the term was introduced.

**Core–observer boundary** — see the entry above; introduced in *Earned Compression*.

**Declare, then derive**
: The Metamodel working discipline. The model is declared once as data; analysis, the running application, proofs and, in this ecosystem's own deployment tooling, the infrastructure that serves it are computed from that document. Nothing downstream is hand-written to agree with the model, so nothing downstream can disagree with it (Chapter 21; *The Model Is the App*).

**Drift**
: Disagreement between a declared artifact and the world it describes — a manifest and a host, a vendored module and its upstream, two implementations of one net. Metamodel's claim is not that drift never happens but that it is always *detectable* by comparing two values: a manifest against a host, a vendored file against its lockfile, a trace against the golden trace (Chapters 20, 21).

**Edge-matching composition**
: Two nets compose by sharing a place, the way two tiles meet because their edges agree. No glue code is written to make them fit; the shared place *is* the interface. P-invariants of the components survive the gluing (Chapters 4, 21; *A Tiling for Computation*).

**Golden trace**
: A canonical firing sequence and its resulting markings, checked byte-for-byte against every implementation of a model. `pflow-polyglot/parity/trace.golden` holds twenty-plus implementations across four languages to one trace; the build fails on divergence (Chapter 20).

**Internal DSL (as a failure mode)**
: A model expressed as *code in a host language* that constructs a net — Rust macros, Python calls, Solidity structs. Five such ports of an early version of this work claimed to encode the same thing and could not prove it, because the specification existed only as five programs. The fix was to make the model a document (Chapter 21; book `ROADMAP.md` §Lineage).

**Journey**
: A workflow whose control flow is a Petri net declared in a markdown document's frontmatter, with one prompt per transition in the body. The document is both the documentation and the runtime — there is no second copy of the control flow to drift (Chapter 21).

**Manifest**
: The declared description of an application — a handful of fields and a marking in one YAML file — from which its service entry, virtual host, certificate and uptime check are derived. The infrastructure instance of declare-then-derive (Chapter 21).

**Metamodel**
: Capitalised: a model for making models — the cross-cutting claim of this book. A small, fixed alphabet — place, transition, arc, guard — whose local composition rules generate an unbounded space of specific, checkable systems, because a model written in it is a value rather than a program. Lowercase `metamodel`, in code font, is a Go package implementing one instance of the vocabulary (Chapter 21; *A Tiling for Computation*).

**Model is a value, not a program**
: The single fact Metamodel rests on. A value can be hashed, diffed, composed and checked by a machine that has never seen any of our runtimes; a program can only be run. Content addressing (Chapter 16), dual implementation (Chapter 20) and the golden trace all depend on it.

**Polyglot forms**
: The five shapes an implementation of one model can take — *interpreter* (a generic engine reads the document), *lambda* (the net as a pure step function), *generated* (source emitted from the document), *contract* (on-chain), *proof* (a kernel-checked theorem) — all held to one golden trace (*The Proof Form*; `pflow-polyglot/FORMS.md`).

**Tropical past, predicate future**
: The tense structure of an executing model. Write-once places and the event log are the *past* — monotone and irreversible, the boolean case of tropical $(\max,+)$ accumulation. Guards are the *future* — predicates recomputed from the marking on every step and stored nowhere. The marking is the *present*, the boundary where they meet. All three are data in the one model document (Chapter 6; Appendix E; *The Zipper Whose Hole Is a Universe*).

**Write-once place**
: A place with incoming arcs only; a token arrives and never leaves. History places in game nets (prefixed `_`) are the canonical case. The smallest instance of tropical accumulation (Chapter 6).
