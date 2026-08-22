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
