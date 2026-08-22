# Exponential Weights and Scoring Systems

**Learning objective**: Design scoring systems using power-of-2 encoding in net structure.

Chapters 5 through 10 modeled diverse systems with Petri nets. Chapter 11 extracted models from data. Chapter 12 added privacy through zero-knowledge proofs. This chapter examines a technique that sits at the boundary between what a Petri net should model internally and what should be delegated to external systems: encoding priority rankings as exponential arc weights.

The technique is sound — power-of-2 weights encode lexicographic order as a single integer. But applying it to poker hand scoring inside the Petri net revealed an important design principle: the boundary between the net and the world matters. This chapter tells the story of what worked, what didn't, and what the failure teaches about modeling decisions.

## Binary Dominance

The core idea: assign power-of-2 weights so that any single higher-priority item outweighs all lower-priority items combined. This is the same principle behind bitmasks, Unix file permissions (read=4, write=2, execute=1), and binary-encoded feature flags.

For poker card ranking, each rank gets a power of 2:

| Rank | Rank Power |
|------|-----------|
| A | 4096 ($2^{12}$) |
| K | 2048 ($2^{11}$) |
| Q | 1024 ($2^{10}$) |
| J | 512 ($2^{9}$) |
| T | 256 ($2^{8}$) |
| 9 | 128 ($2^{7}$) |
| ... | ... |
| 2 | 1 ($2^{0}$) |

With suit tiebreakers (spade=3, heart=2, diamond=1, club=0), the formula is:

$$\text{weight} = \text{rank\_power} \times 4 + \text{suit\_value}$$

Every card maps to a unique integer: A-spade = 16387, A-heart = 16386, down to 2-club = 4.

The key property: **binary dominance**. A single King ($2048 \times 4 = 8192$) outscores all cards Queen and below combined. This means you can sum the weights for any set of cards and the result preserves lexicographic comparison. No sorting needed — just a number.

### Why Linear Weights Fail

Naive linear weights (A=13, K=12, Q=11, ...) break this property. A hand with {A, K, 5, 4, 3} scores $13 + 12 + 5 + 4 + 3 = 37$. A hand with {A, Q, J, T, 9} scores $13 + 11 + 10 + 9 + 8 = 51$. Linear scoring says the second hand wins. Poker says the King beats the Queen — the first hand wins.

Exponential weights fix this by construction. With power-of-2 encoding, the King's contribution dominates everything below it, so the first hand's score is necessarily higher than any hand where a Queen is the second-highest card.

## The Score Is the State

There's a deeper property. When each item gets weight $2^n$, the sum of any subset is unique — that's binary representation. But consider what this means for Petri nets.

A hand with {K, T, 5, 2}, using rank powers:

```
K = 2^11 = 2048
T = 2^8  = 256
5 = 2^3  = 8
2 = 2^0  = 1
---------------
Sum = 2313
```

Now write 2313 in binary: `100100001001`. Each bit position maps to a rank. The bits are the places. The 1s are the tokens. The binary representation of the score **is** the Petri net marking.

This isn't a decoding trick — it's the same structure viewed two different ways. A Petri net marking is a vector of token counts across places. When every place holds 0 or 1 tokens, that vector is a bit string. And a bit string is a binary number. So the accumulated score, the bit string, and the marking are three notations for the same mathematical object.

The constraint: this works for set membership — "which cards are in this hand?" — where each place holds at most one token. If a place could hold 2 or more tokens, a single bit can't represent it. But for binary-state places, the number 2313 doesn't just *rank* the hand. It *is* the hand.

## What We Built (And Removed)

We encoded this in the poker hand Petri net:

1. A **`kicker_score` place** that accumulated the total weight (initial = 0)
2. **52 detection transitions** (`hc_A-heart`, `hc_K-spade`, ...), one per card
3. **Consuming input arcs** from each card place to its detection transition
4. **Weighted output arcs** from each detection transition to `kicker_score`

When a card was in the hand, its place had a token, enabling the corresponding transition. It fired once — consuming the card token — and deposited the card's universal weight into `kicker_score`. Cards not in the hand never fired. The consuming arcs made each transition self-limiting.

Mechanically, it worked. The transitions fired, tokens accumulated, and the resulting `kicker_score` correctly ranked hands. Tests passed. You could look at two hands' kicker scores and determine the winner.

Then we removed it.

## Why It Was Wrong

The poker hand model detects pairs, straights, and flushes through **structural properties of the net** — token patterns across places that enable or inhibit transitions. A pair is detected because two card tokens enable a pair transition. A flush is detected because five suited tokens enable a flush transition. The net *does* the classification.

Kicker scoring doesn't work this way. The `kicker_score` place just accumulates a number. Nothing in the net reads that number. No transition is enabled or disabled by it. No arc weight depends on it. The model needs a separate interpreter to extract meaning from the token count — you have to decompose the sum back into powers of 2 to recover which cards contributed.

That is bookkeeping bolted onto the side of the model, not modeling.

A Petri net model should be self-describing: the structure of places, transitions, and arcs encodes the rules. If you need a decoder ring to read meaning out of a token count, you're not modeling the domain in the net — you're using the net as a storage medium for an external computation. The 52 extra transitions and 104 extra arcs added complexity without adding behavioral insight.

## The Boundary Between Net and External Logic

The lesson is about **where the model ends and the world begins**.

Petri nets are good at modeling concurrent, discrete behavior through structure. Places represent state. Transitions represent events. Arcs define preconditions and effects. The topology *is* the logic. When you need to add behavior, the right instinct is to add structure — new places, new transitions, new arcs that encode the rules.

But encoding isn't behavior. Assigning clever weights to arcs doesn't make the net *do* anything new — it makes the net *store* something for someone else to read. That's fine, as long as you're clear about the boundary.

### When Exponential Weights Belong

The encoding is valuable when the consumer is **external** and the Petri net is generating output:

**Event-sourced state.** In an event-sourced system built on a Petri net, transitions emit events and external projections interpret them. If a projection needs to rank items by priority, the net can deposit exponentially-weighted tokens into an output place. The projection reads the accumulated value and uses it directly for sorting. The net produces the value; the projection consumes it. Each side does what it's good at.

**Multi-criteria scoring.** When criteria have strict priority order (safety > performance > cost), exponential weights encode the hierarchy. A workflow net accumulates scores as items pass through evaluation stages. The final score in an output place encodes the full priority ranking as a single integer. An external dashboard reads it without replaying the evaluation logic.

**Compact set representation.** The score in binary *is* the place vector. This makes power-of-2 sums useful whenever a Petri net needs to communicate *which subset of items* was selected to an external consumer. One place, one integer, full recovery. The external system reads the bits to reconstruct the set without replaying any transitions.

### When They Don't Belong

The encoding fails when the net itself needs to read the result:

- **No transition depends on the accumulated score.** The number sits in a place that nothing fires from.
- **No guard references the score.** No conditional behavior changes based on the value.
- **The interpretation requires external logic.** You need a separate decoder to extract meaning.

If the only consumer of a token count is outside the net, the encoding adds structural complexity (more transitions, more arcs) without adding behavioral capability. The net is no more powerful with the encoding than without it.

## The General Principle

The poker experiment reveals a design heuristic for Petri net modeling:

**Ask: does any transition use this information?**

If the answer is yes — if a place's token count enables, disables, or influences a transition — then encode it in the net. That's what Petri nets are for.

If the answer is no — if the information is only consumed by external systems — then the net is the wrong place for the computation. Let the net generate the raw events. Let the external system do the interpretation. Don't add 52 transitions and 104 arcs to encode something that a simple function could compute from the event stream.

This boundary applies beyond exponential weights. Any time you're tempted to add net structure that doesn't participate in the net's behavior — logging places, summary counters, encoding tricks — ask whether the information is being *modeled* or merely *stored*. Models live in the net. Storage belongs outside.

The exponential encoding itself is genuinely useful. It maps sets to integers, preserves lexicographic ordering, and enables constant-time comparison. The mistake was embedding it in the net instead of applying it at the boundary where the net's output meets the world.

There is, however, a way to derive poker hand rankings *from* net topology rather than bolting them on. The integer reduction technique from [Chapter 13](ch13-topology-driven-verification.md) builds a hand analysis net where drain transitions proportional to combinatorial frequency create differential outflow. At ODE equilibrium, rare hands accumulate more tokens and produce higher values — straight flush=32, four of a kind=16, down to high card=1. The ranking emerges from topology, and the drain transitions are the mechanism, not bookkeeping. See "Poker Hand Ranking: Integer Reduction" in Chapter 13 for the full treatment.
