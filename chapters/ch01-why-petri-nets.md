# Why Petri Nets?

**Learning objective**: Understand what Petri nets offer over flowcharts, state machines, and ad-hoc modeling.

You already know how to model systems. You've drawn flowcharts on whiteboards, sketched state machines in design docs, and built ad-hoc abstractions in code. These tools work — until they don't. This chapter is about what breaks, and what Petri nets offer instead.

## The Problem with Informal Models

Every software project starts with a model, whether the team knows it or not. Someone draws boxes and arrows on a whiteboard. Someone else writes a state machine in an enum. A third person embeds the workflow directly in if-else chains. All three believe they're describing the same system.

They usually aren't.

Informal models fail in predictable ways:

**Flowcharts can't express concurrency.** A flowchart is a sequence with branches. It handles "do A then B" and "do A or B" well enough. But "do A and B at the same time, then wait for both to finish" requires awkward workarounds — swim lanes, join bars, footnotes. Real systems are concurrent. Flowcharts pretend otherwise.

**State machines explode.** A state machine with $n$ boolean variables has $2^n$ possible states. A system tracking 10 independent conditions has 1,024 states. Most of those states are meaningless combinations, but the machine doesn't know that. You end up either enumerating impossible states or pretending they can't happen.

**Ad-hoc models can't be analyzed.** When the workflow lives in code — scattered across handlers, middleware, and database triggers — there's no way to ask "can this system deadlock?" or "is this queue bounded?" You find out in production, at 3am.

The root problem is the same in each case: the model doesn't capture the structure of the system it represents. Concurrency is an afterthought. Resources are invisible. State is implicit. You can't analyze what you haven't modeled.

## A Brief History

In 1962, Carl Adam Petri submitted his doctoral dissertation, *Kommunikation mit Automaten* (Communication with Automata), to the Technical University of Darmstadt. He was 36 years old, and he had been thinking about the problem for over a decade — his first sketches of what would become Petri nets date to 1939, when he was thirteen.

Petri's insight was deceptively simple: concurrent systems need a model where multiple things can be true at the same time. State machines force you to enumerate every combination of conditions as a single global state. Petri's nets decompose the state into independent pieces — *places* — each holding their own local truth. The interactions between these pieces are explicit in the structure of the net.

The formalism found its first serious applications in hardware design, where concurrency is unavoidable. By the 1980s, Petri nets had become a standard tool in manufacturing, telecommunications, and workflow modeling. They remain the mathematical foundation for business process notations like BPMN, even though most practitioners never see the underlying net.

Petri spent his career at the GMD (German National Research Centre for Computer Science) in Bonn, refining the theory and advocating for precise models of concurrent systems. He died in 2010. The formalism he created — places, transitions, and the flow of tokens between them — has proven remarkably durable. Sixty years later, no one has found a simpler model that handles concurrency, synchronization, and resource tracking in a single, analyzable structure.

## Places, Transitions, Arcs, Tokens

A Petri net has exactly four kinds of things:

**Places** are drawn as circles. A place represents a condition that can be true or false, a resource that can be available or consumed, a state that a piece of the system can be in. Think of a place as a container.

**Tokens** are drawn as dots inside places. Tokens represent the *current state* of the system: which conditions hold, how many resources are available, where entities are in a process. The distribution of tokens across all places is called the **marking** — it's the complete snapshot of the system at a moment in time.

**Transitions** are drawn as rectangles (or bars). A transition represents an event, an action, something that *happens*. Registration. Brewing. Firing. A transition changes the state of the system by moving tokens around.

**Arcs** are arrows connecting places to transitions and transitions to places. An arc from a place to a transition means "this transition needs a token from this place." An arc from a transition to a place means "this transition produces a token in this place." Arcs define the flow — who needs what, and who produces what.

That's it. Four primitives. No inheritance hierarchies, no configuration languages, no plugin architectures. Every Petri net ever drawn — from a traffic light to a biochemical pathway to a poker game — is built from these four elements.

### The Firing Rule

The one rule that makes the whole thing work:

> A transition **fires** when every input place has at least one token. When it fires, it removes one token from each input place and adds one token to each output place.

This single rule gives you sequencing (token moves from A to B), concurrency (two transitions can fire independently if they don't share input places), synchronization (a transition with two input places waits for both), and resource management (a place with $n$ tokens represents $n$ available resources).

### Weighted Arcs

Sometimes a transition needs more than one token from a place, or produces more than one. A number on an arc — its **weight** — specifies how many tokens are consumed or produced. An arc with weight 3 from a place to a transition means "this transition needs 3 tokens from this place to fire."

## Your First Net: A Traffic Stoplight

A traffic light cycles through three states: green, yellow, red. At any moment, exactly one light is on. Here's the Petri net:

```
    *
   [Green]--> [go_yellow] --> [Yellow]--> [go_red] --> [Red]--> [go_green]
      ^                                                              |
      +--------------------------------------------------------------+
```

Three places: `Green`, `Yellow`, `Red`. Three transitions: `go_yellow`, `go_red`, `go_green`. One token, starting in `Green`.

The token's position tells you which light is on. Fire `go_yellow` and the token moves from `Green` to `Yellow`. Fire `go_red` and it moves from `Yellow` to `Red`. Fire `go_green` and it cycles back to `Green`.

This is almost trivially simple — a state machine would work just as well. But notice what the Petri net gives you for free:

**Conservation.** There is exactly one token in the system. No sequence of firings can create a second token or destroy the existing one. The net's structure guarantees that exactly one light is on at all times. You don't need to assert this as a runtime check — it's a mathematical property of the topology.

**No impossible states.** A state machine for this traffic light has three valid states and could, through a bug, reach an invalid one. The Petri net has no representation for "green and red are both on." The structure makes it unrepresentable, not just unlikely.

**Composability.** Want to model an intersection? Add a second traffic light net. Connect them with a coordination mechanism — a shared place that ensures one direction is red when the other is green. Each light keeps its internal structure; the coordination is explicit in the arcs between them.

### An Intersection

Two lights need to coordinate. We add a shared place `Intersection_Free` that acts as a mutual exclusion lock:

```
Light A: [Red_A] <-- [go_red_A] <-- [Green_A]
                                        ^
                              [Intersection_Free] *
                                        v
Light B: [Red_B] --> [go_green_B] --> [Green_B]
```

The token in `Intersection_Free` is consumed when one light turns green and returned when it turns red. Only one light can be green at a time — not because of a runtime check, but because there's only one token and two transitions competing for it.

This pattern — a shared resource place controlling access — appears everywhere in Petri nets. It models mutexes, semaphores, limited capacity, and turn-taking. You'll see it again in every chapter of this book.

## Small Models, Not Large Language Models

The word "model" has been colonized. Say it in a meeting and people hear "large language model" — billions of parameters, inscrutable weights, probabilistic outputs. There's another kind of model: small, executable, and inspectable all the way down.

Petri nets are small models. A coffee shop with five ingredients and three recipes is about 15 places and 8 transitions. A complete poker game — Texas Hold'em with all betting rounds — is under 50 places. A Sudoku solver is larger but still finite and fully visible.

Small models offer properties that large models cannot:

| Property | Large Language Models | Petri Nets |
|----------|---------------------|------------|
| Inspectable | No | Yes — read the topology |
| Composable | Awkward | Native — connect shared places |
| Provable properties | No | Yes — invariants, reachability, liveness |
| Executable | Via code generation | Directly — tokens flow |
| Deterministic | No | Yes, given the same firing sequence |

This matters practically, not just philosophically. A Petri net model lets you answer questions before writing implementation code:

**Reachability**: "Given this initial state, can the system reach that target state?" This tells you whether a workflow is completable, whether a game has a winning move, whether an order can be fulfilled from current inventory.

**Boundedness**: "Does the system stay within limits?" This tells you whether a queue can overflow, whether a resource pool is properly managed, whether your system is closed or leaking.

**Liveness**: "Can the system deadlock?" This tells you whether every transition can eventually fire again, or whether some paths lead to permanent stuckness.

These aren't abstract concerns. They're the bugs that take down production systems. The difference is that with a Petri net, you can check them statically — before the system runs, before you've written the implementation, before the 3am page.

### Models as Specification

Here's the workflow this book advocates:

1. **Sketch the net** — places for states and resources, transitions for events
2. **Simulate** — run the ODE solver to see if tokens flow the way you expect
3. **Analyze** — check invariants, reachability, boundedness
4. **Generate code** — derive the implementation from the validated model

The model becomes the specification. The code becomes a derived artifact. When the model changes, the code changes with it. When there's a disagreement between model and implementation, the model wins — because the model is what you analyzed and proved correct.

This is the opposite of how most software is built, where the code *is* the model and the specification exists (if at all) as outdated documentation. With Petri nets, the model is a living, executable, analyzable artifact that stays authoritative throughout the project's life.

The rest of this book shows what this looks like in practice — from a coffee shop inventory system to a zero-knowledge proof of a game move, all built from places, transitions, arcs, and tokens.
