# The Mathematics of Flow

**Learning objective**: Read and write the formal notation; understand firing rules and state equations.

Chapter 1 introduced Petri nets informally — circles, bars, arrows, dots. That's enough for intuition, but not enough for analysis. To prove that a system can't deadlock, or that a resource pool never goes negative, or that a workflow always terminates, you need precise definitions. This chapter provides them.

The notation looks heavy at first. It isn't. Everything here follows from two ideas: a net is a bipartite graph with weights, and a marking is a vector of integers. The rest is matrix arithmetic.

## The Formal Definition

A **Petri net** is a 5-tuple:

$$PN = (P, T, F, W, M_0)$$

Where:

- $P = \{p_1, p_2, \ldots, p_n\}$ is a finite set of **places**
- $T = \{t_1, t_2, \ldots, t_m\}$ is a finite set of **transitions**
- $F \subseteq (P \times T) \cup (T \times P)$ is the **flow relation** — the set of arcs
- $W: F \to \mathbb{N}^+$ is the **arc weight function**
- $M_0: P \to \mathbb{N}$ is the **initial marking**

Two constraints keep things clean: $P \cap T = \emptyset$ (nothing is both a place and a transition) and $P \cup T \neq \emptyset$ (the net isn't empty).

That's the entire definition. Five things. Places are where tokens live. Transitions are what moves them. Arcs say who connects to whom. Weights say how many tokens an arc carries. The initial marking says where tokens start.

### Presets and Postsets

Every transition has inputs and outputs. The notation for these is compact and worth memorizing:

- $\bullet t = \{p \in P \mid (p, t) \in F\}$ — the **preset** of $t$ (input places)
- $t\bullet = \{p \in P \mid (t, p) \in F\}$ — the **postset** of $t$ (output places)

The same notation works for places:

- $\bullet p = \{t \in T \mid (t, p) \in F\}$ — transitions that feed into $p$
- $p\bullet = \{t \in T \mid (p, t) \in F\}$ — transitions that $p$ feeds into

The bullet notation reads naturally: $\bullet t$ means "what comes before $t$," and $t\bullet$ means "what comes after $t$."

## Markings and the State Vector

A **marking** $M$ assigns a non-negative integer to each place — how many tokens it holds. Since a net has $n$ places, we can write the marking as a column vector:

$$M = [M(p_1), M(p_2), \ldots, M(p_n)]^T \in \mathbb{N}^n$$

This is the **state vector**. It captures everything about the system's current state. Two markings are equal if and only if every place has the same token count in both. There's no hidden state, no side channels, no context beyond the vector.

The initial marking $M_0$ is the starting state. For the traffic light from Chapter 1:

$$M_0 = \begin{bmatrix} 1 \\ 0 \\ 0 \end{bmatrix} \quad \text{(one token in Green, none in Yellow or Red)}$$

Every marking you can reach by firing transitions is a point in $\mathbb{N}^n$ — non-negative integer coordinates, one per place. The set of all reachable markings, starting from $M_0$, is the **reachability set** $R(M_0)$. For the traffic light, $R(M_0)$ contains exactly three points: the token in Green, the token in Yellow, the token in Red.

## Firing Rules and Reachability

A transition fires by consuming tokens from its input places and producing tokens in its output places. But it can only fire when it has enough tokens to consume. The **enabling condition** is:

$$\text{Transition } t \text{ is enabled at marking } M \iff \forall p \in \bullet t: M(p) \geq W(p, t)$$

Every input place must have at least as many tokens as the arc weight demands. If any input place is short, the transition is blocked.

When an enabled transition $t$ fires, the new marking $M'$ is:

$$M'(p) = M(p) - W(p, t) + W(t, p)$$

For every place $p$: subtract the tokens consumed (arc from $p$ to $t$) and add the tokens produced (arc from $t$ to $p$). If $p$ isn't connected to $t$, both weights are zero and $M'(p) = M(p)$ — unconnected places are unaffected.

### A Worked Example

Consider a simple three-place chain:

```
[p₁] -> t₁ -> [p₂] -> t₂ -> [p₃]
```

With initial marking $M_0 = [1, 0, 0]^T$ (one token in $p_1$).

**Step 1:** Is $t_1$ enabled? Its only input is $p_1$, and $M_0(p_1) = 1 \geq 1$. Yes. Fire $t_1$:

$$M_1 = [1 - 1, 0 + 1, 0]^T = [0, 1, 0]^T$$

**Step 2:** Is $t_2$ enabled? Its only input is $p_2$, and $M_1(p_2) = 1 \geq 1$. Yes. Fire $t_2$:

$$M_2 = [0, 1 - 1, 0 + 1]^T = [0, 0, 1]^T$$

The token has moved from $p_1$ through $p_2$ to $p_3$. The reachability set is $R(M_0) = \{[1,0,0]^T, [0,1,0]^T, [0,0,1]^T\}$. Three reachable markings, matching intuition — the token can be in exactly one of three places.

### Firing Sequences

A **firing sequence** is an ordered list of transitions: $\sigma = t_1, t_2, \ldots, t_k$. The sequence is **valid** from marking $M$ if each transition is enabled when it's supposed to fire — $t_1$ is enabled at $M$, $t_2$ is enabled at the marking after $t_1$ fires, and so on.

We write $M \xrightarrow{\sigma} M'$ to mean "firing sequence $\sigma$ is valid from $M$ and leads to $M'$." The reachability set is then:

$$R(M_0) = \{M' \mid \exists \sigma: M_0 \xrightarrow{\sigma} M'\}$$

### Nondeterminism

When multiple transitions are enabled simultaneously, either one could fire. The Petri net doesn't prescribe which. This **nondeterminism** is a feature, not a bug — it models real concurrency where the order of independent events is genuinely unspecified.

In the traffic light, only one transition is ever enabled at a time, so there's no nondeterminism. But in the intersection model from Chapter 1, when the shared token is available, both lights could claim it. The net structure constrains what's possible; it doesn't dictate the schedule.

## The Incidence Matrix

Computing markings by hand gets tedious. The **incidence matrix** makes it mechanical.

The incidence matrix $N \in \mathbb{Z}^{n \times m}$ has one row per place and one column per transition. Each entry is the *net effect* of firing that transition on that place:

$$N[i, j] = W(t_j, p_i) - W(p_i, t_j)$$

Positive means the transition produces tokens in that place. Negative means it consumes. Zero means no effect.

### Building the Matrix

For the three-place chain:

| | $t_1$ | $t_2$ |
|:---:|:---:|:---:|
| $p_1$ | $-1$ | $0$ |
| $p_2$ | $+1$ | $-1$ |
| $p_3$ | $0$ | $+1$ |

Reading column $t_1$: firing $t_1$ removes one token from $p_1$ and adds one to $p_2$. Reading column $t_2$: firing $t_2$ removes one from $p_2$ and adds one to $p_3$. Each column is a change vector — the delta applied to the marking when that transition fires.

### The State Equation

This is the payoff. With the incidence matrix, the firing rule becomes matrix multiplication:

$$M' = M + N \cdot \vec{\sigma}$$

Where $\vec{\sigma} \in \mathbb{N}^m$ is the **firing count vector** — how many times each transition fires. For a single firing of $t_1$, $\vec{\sigma} = [1, 0]^T$:

$$M_1 = \begin{bmatrix} 1 \\ 0 \\ 0 \end{bmatrix} + \begin{bmatrix} -1 & 0 \\ 1 & -1 \\ 0 & 1 \end{bmatrix} \cdot \begin{bmatrix} 1 \\ 0 \end{bmatrix} = \begin{bmatrix} 1 \\ 0 \\ 0 \end{bmatrix} + \begin{bmatrix} -1 \\ 1 \\ 0 \end{bmatrix} = \begin{bmatrix} 0 \\ 1 \\ 0 \end{bmatrix}$$

Fire both transitions once ($\vec{\sigma} = [1, 1]^T$) to go directly from $M_0$ to $M_2$:

$$M_2 = \begin{bmatrix} 1 \\ 0 \\ 0 \end{bmatrix} + \begin{bmatrix} -1 & 0 \\ 1 & -1 \\ 0 & 1 \end{bmatrix} \cdot \begin{bmatrix} 1 \\ 1 \end{bmatrix} = \begin{bmatrix} 1 \\ 0 \\ 0 \end{bmatrix} + \begin{bmatrix} -1 \\ 0 \\ 1 \end{bmatrix} = \begin{bmatrix} 0 \\ 0 \\ 1 \end{bmatrix}$$

The state equation is powerful because it's linear. Given any firing count vector, you can compute the resulting marking with one matrix multiplication. But there's a caveat: the state equation doesn't check enabling. A firing count vector might describe a sequence that isn't valid — some transition might need tokens that aren't there yet. The state equation gives you a necessary condition for reachability, not a sufficient one.

### A Richer Example

Consider a net where a transition has two inputs — the synchronization pattern:

```
[p₁] --> t₁ --> [p₃]
[p₂] --/
```

Transition $t_1$ needs a token from both $p_1$ and $p_2$ to fire. The incidence matrix:

| | $t_1$ |
|:---:|:---:|
| $p_1$ | $-1$ |
| $p_2$ | $-1$ |
| $p_3$ | $+1$ |

With $M_0 = [1, 1, 0]^T$, firing $t_1$ gives $M_1 = [0, 0, 1]^T$. With $M_0 = [1, 0, 0]^T$, $t_1$ is not enabled — $p_2$ has no token. The synchronization is enforced by the structure.

## P-Invariants and Conservation Laws

The incidence matrix reveals something deeper than individual firings. It reveals what *cannot change*, no matter what sequence of transitions fires.

A **P-invariant** (place invariant) is a row vector $w \in \mathbb{Z}^n$ such that:

$$w^T \cdot N = \vec{0}$$

The vector $w$ assigns a weight to each place such that the weighted sum of tokens is preserved by every transition. If $w$ is a P-invariant, then for any reachable marking $M$:

$$w^T \cdot M = w^T \cdot M_0$$

This is a **conservation law**. No matter what fires, no matter what order, the weighted token count stays constant.

### Finding P-Invariants

For the three-place chain, try $w^T = [1, 1, 1]$:

$$w^T \cdot N = [1, 1, 1] \cdot \begin{bmatrix} -1 & 0 \\ 1 & -1 \\ 0 & 1 \end{bmatrix} = [0, 0]$$

It works. The invariant says:

$$M(p_1) + M(p_2) + M(p_3) = M_0(p_1) + M_0(p_2) + M_0(p_3) = 1$$

The total number of tokens is always 1. We proved this for the traffic light informally in Chapter 1 — now we've proved it with linear algebra, and the proof works for any net where $[1, 1, \ldots, 1] \cdot N = \vec{0}$.

### What Conservation Laws Tell You

P-invariants answer practical questions:

**"Can tokens leak?"** If $w = [1, 1, \ldots, 1]$ is an invariant, the total token count is constant. The net is closed — nothing is created or destroyed. This is the Petri net equivalent of conservation of mass.

**"Is this resource pool managed correctly?"** If two places $p_a$ (available) and $p_b$ (busy) satisfy $M(p_a) + M(p_b) = c$ for some constant $c$, then resources are always either available or busy, never duplicated or lost.

**"Is the net bounded?"** If every place appears in at least one P-invariant with a positive coefficient, then no place can accumulate unbounded tokens. The net is **structurally bounded** — it stays within limits regardless of the initial marking.

### Partial Invariants

Not every net has a P-invariant covering all places. A net with a "source" transition (one that produces tokens from nothing) or a "sink" transition (one that consumes tokens into nothing) breaks total conservation. But subsets of places may still be conserved.

For example, in a producer-consumer system:

```
[source] -> t_produce -> [buffer] -> t_consume -> [sink]
```

The total token count isn't conserved — $t_{produce}$ creates tokens and $t_{consume}$ destroys them. But if you add a capacity limit by looping tokens back:

```
[empty_slots: 5] -> t_produce -> [buffer] -> t_consume -> [empty_slots]
```

Now $M(\text{empty\_slots}) + M(\text{buffer}) = 5$ is an invariant. The buffer can never exceed 5.

### In Code

The go-pflow library computes incidence matrices and checks invariants directly:

```go
analyzer := reachability.NewInvariantAnalyzer(net)
matrix, places, transitions := analyzer.IncidenceMatrix()
invariants := analyzer.FindPInvariants(initialMarking)

for _, inv := range invariants {
    fmt.Printf("Conserved: %v = %d\n", inv.Coefficients, inv.Value)
}

// Check if total tokens are conserved
isConservative := analyzer.CheckConservation(initialMarking)
```

The `IncidenceMatrix` method builds the $N$ matrix from the net's arc structure. `FindPInvariants` solves $w^T \cdot N = \vec{0}$ to find conservation laws. `CheckConservation` tests whether the all-ones vector is an invariant — whether total token count is preserved.

## Putting It Together

The mathematics of this chapter — the 5-tuple, the state equation, the incidence matrix, P-invariants — form a complete toolkit for structural analysis. Given a Petri net, you can:

1. **Write the incidence matrix** from the arc structure
2. **Compute reachable markings** using the state equation $M' = M + N \cdot \vec{\sigma}$
3. **Find conservation laws** by solving $w^T \cdot N = \vec{0}$
4. **Prove boundedness** from positive P-invariants
5. **Check for deadlocks** by finding markings where no transition is enabled

All of this is static analysis — you don't run the net, you reason about its structure. This is what distinguishes Petri nets from simulation frameworks: the topology itself is a proof.

But there's a limit. Discrete Petri nets track individual tokens and individual firings. For large systems — thousands of tokens, complex topologies — the reachability set can be enormous. The next chapter shows how to break through this limit by letting tokens be real numbers and transitions fire continuously, turning the state equation into a differential equation.
