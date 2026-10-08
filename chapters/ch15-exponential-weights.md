# Exponential Weights and Scoring Systems

**Learning objective**: Design scoring systems using power-of-2 encoding in net structure.

Chapters 5 through 10 modeled diverse systems with Petri nets, and Chapters 11 through 14 extracted models from data, derived strategy from topology, and compiled nets into proofs. This chapter examines a technique that sits at the edge of what a Petri net should model internally and what should be delegated to external systems: encoding priority rankings as exponential arc weights.

Power-of-2 weights encode a lexicographic order as a single integer. Applying that encoding to poker hand scoring inside the net, and then removing it, is the case study here; the removal suggests a rule for deciding what belongs in a net at all.

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

The key property is **binary dominance**, and on rank powers alone it is exact: $2^n > 2^{n-1} + \dots + 2^0 = 2^n - 1$, so one King (2048) outweighs Queen through Two together (2047). For two hands with no repeated rank, comparing the sums of their rank powers gives the same answer as comparing their ranks highest first. No sorting is needed.

The poker experiment described below also broke ties by suit (spade=3, heart=2, diamond=1, club=0), with the per-card formula:

$$\text{weight} = \text{rank\_power} \times 4 + \text{suit\_value}$$

Every card maps to a unique integer: A-spade = 16387, A-heart = 16386, down to 2-club = 4. But the factor of 4 leaves room for one card's suit value, not for the suit values of a whole hand, so summed card weights can misorder two hands that differ only in a low card:

| Hand | Card weights | Sum |
|------|--------------|-----|
| A♣ K♣ Q♣ 4♦ 2♣ | 16384 + 8192 + 4096 + 17 + 4 | 28693 |
| A♠ K♠ Q♥ 3♠ 2♠ | 16387 + 8195 + 4098 + 11 + 7 | 28698 |

Neither hand is a flush or a straight, so both are compared card by card, and the first wins on its fourth card (4 over 3). The sums say the second. The fix is a larger factor: when the factor exceeds three times the number of cards, the summed suit values can never outweigh a one-step difference in the rank sum, so `rank_power × 32 + suit_value` orders any hands of up to ten distinct-rank cards by rank first and by total suit value only on a tie. This bound and the table are derived by hand from the formula, not taken from a test.

### Why Linear Weights Fail

Naive linear weights (A=13, K=12, Q=11, ...) break dominance. A hand with {A, K, 5, 4, 3} scores $13 + 12 + 5 + 4 + 3 = 37$. A hand with {A, Q, J, T, 9} scores $13 + 11 + 10 + 9 + 8 = 51$. Linear scoring says the second hand wins; poker says the King beats the Queen, so the first hand wins.

With rank powers the King's 2048 exceeds the 2047 that every lower rank together can contribute, so the first hand outscores any hand whose second-highest card is a Queen.

## The Score Is the State

When each item gets weight $2^n$, the sum of any subset is unique: that is binary representation. Consider what this means for Petri nets.

A hand with {K, T, 5, 2}, using rank powers:

```
K = 2^11 = 2048
T = 2^8  = 256
5 = 2^3  = 8
2 = 2^0  = 1
---------------
Sum = 2313
```

Now write 2313 in binary: `100100001001`. Each bit position maps to a rank: the bits are the places of a thirteen-place rank vector and the 1s are its tokens. The binary representation of the score is that marking.

A Petri net marking is a vector of token counts across places. When every place holds 0 or 1 tokens, that vector is a bit string, and a bit string is a binary number, so the accumulated score, the bit string and the marking carry exactly the same information.

The constraint: this works for set membership, where each place holds at most one token. If a place could hold 2 or more tokens, a single bit cannot represent it. It also needs one power of two per place. The suited card weights above are not one bit per card, and their sums collide: {A♠, K♣} and {A♣, K♠} both total 24579. Recovering the exact cards would take a distinct power of two for each of the 52 card places, a 52-bit integer. With that encoding, the number does not just rank the hand; it identifies it.

## What We Built (And Removed)

We encoded this in the poker hand Petri net:

1. A **`kicker_score` place** that accumulated the total weight (initial = 0)
2. **52 detection transitions** (`hc_A-heart`, `hc_K-spade`, ...), one per card
3. **Consuming input arcs** from each card place to its detection transition
4. **Weighted output arcs** from each detection transition to `kicker_score`

When a card was in the hand, its place had a token, enabling the corresponding transition. It fired once — consuming the card token — and deposited the card's universal weight into `kicker_score`. Cards not in the hand never fired. The consuming arcs made each transition self-limiting.

Mechanically, it worked: the transitions fired and tokens accumulated, and two hands' kicker scores could be compared to pick a winner. That comparison was only as good as the formula, and the pair of high-card hands in the table above is one it orders wrongly.

Then we removed it. The revert, [petri-pilot PR #1](https://github.com/pflow-xyz/petri-pilot/pull/1), takes out the 52 `hc_*` transitions, the `kicker_score` place and a `card_counter` guard from the model builder.

## Why It Was Wrong

The poker hand model that remains, `buildPokerHandModel` in petri-pilot's [`pkg/serve/serve.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/serve/serve.go) and served at [pilot.pflow.xyz/poker-hand](https://pilot.pflow.xyz/poker-hand/), encodes hand patterns as **structure**. Each of its 78 pair transitions (6 suit pairs × 13 ranks) has input arcs from two cards of one rank, so it is enabled exactly when both cards hold tokens; three-of-a-kind and straight-flush transitions work the same way with three and five cards. Flushes and straights are covered only for a representative subset of patterns, ten five-card flush patterns per suit and one suit combination per straight, as the builder's own comments note. Where a pattern has a transition, the net's structure is the classifier.

What has not been demonstrated is the net doing that classification end to end. The builder also computes the hand's name directly in Go for the model's description, and its tests ([`pkg/serve/poker_hand_test.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/serve/poker_hand_test.go)) check that name against a standard evaluator and check the net's structure (52 deal transitions, 78 pair transitions). The web page's hand comparison runs a conventional JavaScript evaluator. No test fires the detection transitions and reads the result.

Kicker scoring has no such structure. The `kicker_score` place just accumulates a number. Nothing in the net reads that number. No transition is enabled or disabled by it. No arc weight depends on it. The model needs a separate interpreter to extract meaning from the token count: you have to decompose the sum back into powers of 2 to recover which cards contributed, and with the suited weights even that decomposition is ambiguous.

That is bookkeeping bolted onto the side of the model, not modeling.

A Petri net model should be self-describing: the structure of places, transitions, and arcs encodes the rules. If you need a decoder ring to read meaning out of a token count, you're not modeling the domain in the net — you're using the net as a storage medium for an external computation. The 52 extra transitions and 104 extra arcs (one input and one output per transition) added complexity without adding behavioral insight.

The model that remains carries a smaller instance of the same pattern. Eight `score_*` transitions deposit weights 2 through 9 into a `hand_strength` place, and no arc leads out of it. By the test at the end of this chapter, `hand_strength` is bookkeeping too.

## The Boundary Between Net and External Logic

The lesson is about **where the model ends and the world begins**. This is a boundary of modeling scope, not either of the two core–observer boundaries of [Appendix E](appendix-e-categorical-foundations.md#where-the-free-structure-stops-two-boundaries).

Petri nets are good at modeling concurrent, discrete behavior through structure. Places represent state, transitions represent events, and arcs define preconditions and effects, so the topology carries the logic. When you need to add behavior, the right instinct is to add structure: new places, transitions and arcs that encode the rules.

Assigning weights to arcs adds no behavior. It makes the net *store* something for someone else to read, which is fine as long as you are clear about the boundary.

### When Exponential Weights Belong

The encoding is valuable when the consumer is **external** and the Petri net is generating output for it. The book does not yet give a worked model for any of the three settings below; they are the shapes the argument predicts, not results.

**Event-sourced state.** In an event-sourced system built on a Petri net, transitions emit events and external projections interpret them. If a projection needs to rank items by priority, the net can deposit exponentially-weighted tokens into an output place, and the projection reads the accumulated value and sorts by it directly.

**Multi-criteria scoring.** When criteria have strict priority order (safety > performance > cost), exponential weights encode the hierarchy. A workflow net could accumulate scores as items pass through evaluation stages, leaving the full priority ranking as a single integer in an output place that an external dashboard reads without replaying the evaluation logic. The factor between levels has to exceed the largest total a lower level can reach, the same condition the suit tiebreaker above failed.

**Compact set representation.** With one power of two per place, the score in binary is the place vector. That makes power-of-2 sums useful whenever a Petri net needs to tell an external consumer *which subset of items* was selected: one place holds one integer, and the consumer reads its bits to reconstruct the set without replaying any transitions.

### When They Don't Belong

The encoding fails when the net itself needs to read the result. If no transition depends on the accumulated score, no guard references it, and its meaning has to be decoded outside the net, then the encoding adds structural complexity (more transitions, more arcs) without adding behavioral capability. The net is no more powerful with the encoding than without it.

## The General Principle

The poker experiment reveals a design heuristic for Petri net modeling:

**Ask: does any transition use this information?**

If the answer is yes, because a place's token count enables, disables or influences a transition, then encode it in the net.

If the answer is no, because the information is only consumed by external systems, then the net is the wrong place for the computation. Let the net generate the raw events and the external system do the interpretation, rather than adding 52 transitions and 104 arcs to encode something that a simple function could compute from the event stream.

This boundary applies beyond exponential weights. Any time you're tempted to add net structure that doesn't participate in the net's behavior, such as logging places, summary counters or encoding tricks, ask whether the information is being *modeled* or merely *stored*.

The exponential encoding itself is useful: with one power of two per item it maps sets to integers, and with rank powers on distinct items it orders them lexicographically, so two results compare with one integer comparison. The mistake was embedding it in the net instead of applying it at the boundary where the net's output meets the world.

There is, however, a way to derive poker hand rankings *from* net topology rather than bolting them on. The integer reduction technique in [Chapter 13: Topology-Driven Verification](ch13-topology-driven-verification.md) builds a hand analysis net in which each hand category gets a number of drain transitions that grows with its combinatorial frequency (log-scaled, from 1 drain for straight flush to 32 for high card). At ODE equilibrium each value place settles at $1/\text{drains}$, so rare hands hold more tokens; scaled so that high card = 1, straight flush = 32, four of a kind = 16, and so on down. That equilibrium is a closed-form consequence of the rate law, and the drain transitions are the mechanism that produces the ranking rather than bookkeeping beside it. See "Poker Hand Ranking: Integer Reduction" in Chapter 13 for the construction.
