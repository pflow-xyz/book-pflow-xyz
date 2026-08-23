# proofs/ — Lean 4 proofs for the book

Core Lean 4 only (pinned in `lean-toolchain`, currently `v4.33.0`). **No
mathlib, no Std/Batteries** — every lemma is proved from the core library.

## Build

```bash
cd proofs
~/.elan/bin/lake build        # expect: "Build completed successfully"
grep -rn sorry PflowProofs/    # expect: no output
```

## `PflowProofs/InvariantLift.lean` — P-invariants lift across pushout along places

This is roadmap item **P4** (`PROOF-ROADMAP.md`) and the statement Appendix E
rests on: "P-invariants of a component are still derivable from the
composite's incidence matrix."

### Statement, in the book's notation

Let `C₁` (places `P₁` × transitions `T₁`) and `C₂` (`P₂` × `T₂`) be incidence
matrices, `C[p, t] = post(t)(p) − pre(t)(p)`. A P-invariant is `y` with
`yᵀC = 0`. Let `f : P₁ ⊔ P₂ → Q` identify places (the pushout of the place sets
along the shared boundary). The glued net has places `Q`, transitions
`T₁ ⊔ T₂`, and incidence `C[q, t] = Σ_{p : f p = q} C_i[p, t]`.

**Theorem (`invariant_lift`).** If `y₁ᵀC₁ = 0`, `y₂ᵀC₂ = 0`, and `y : Q → ℤ`
satisfies `y(f(inl p)) = y₁(p)` and `y(f(inr p)) = y₂(p)` (agreement on
identified places), then `yᵀC = 0` for the glued net.

Proof: for a transition from `N₁`,
`Σ_q y(q) · Σ_{f p = q} C₁[p,t] = Σ_p y(f p) · C₁[p,t] = Σ_p y₁(p) · C₁[p,t] = 0`
by reindexing the sum along `f`; symmetrically for `N₂`.

### How the Lean maps to the notation

| Notation | Lean |
|---|---|
| place set `P`, enumerated | a type `P` plus a list `L : List P` |
| `Σ_{p ∈ P} g p` | `sumOver L g := (L.map g).sum` (over `Int`) |
| transition `t` with `pre(t)`, `post(t)` | `Trans P := { pre post : P → Int }` |
| column `C[·, t]` | `Trans.col t p := t.post p - t.pre p` |
| net, i.e. the columns of `C` | `Net P := List (Trans P)` |
| `yᵀC = 0` | `IsInvariant L N y := ∀ t ∈ N, sumOver L (fun p => y p * t.col p) = 0` |
| pushforward of a column along `f` | `Trans.push L f t` — `pre' q = Σ_{p, f p = q} pre p`, same for `post` |
| glued net | `glue L₁ L₂ f N₁ N₂ := N₁.push (f ∘ inl) ++ N₂.push (f ∘ inr)` |
| "f is the pushout of the place sets" | `M : List Q` with `M.Nodup` and `∀ s, f s ∈ M` (every glued place is hit; enumeration has no duplicates) |

The crux lemma is `sumOver_reindex` (sum over fibres of `f` = sum over the
domain), proved by induction on the domain list, using `sumOver_ite`
(`Σ_{q ∈ M} [a = q]·h q = h a` for `M` duplicate-free and containing `a`).
`IsInvariant_push` is the one-component half; `invariant_lift` is the
two-component gluing; `IsInvariant_append` lets it chain to any number of
components.

### The Settle corollary

The three-party settlement cycle from Appendix E / pflow-jl
(`ch_ab`, `ch_bc`, `ch_ca`, each with `send` and `settle`, boundary places
the balances `a, b, c`) is built *by the `glue` definition*, not by hand:
`settle := glue ch_ab ch_bc ++ push ch_ca` over `Q6 = {a, b, c, pab, pbc, pca}`.

- `channel_ones` — `x + y + pend` is conserved on one channel (`decide`).
- `settle_ones_lifted` — `a + b + c + p_ab + p_bc + p_ca` is conserved on the
  cycle, **obtained from `channel_ones` by the theorem**.
- `settle_ones_decide` — the same fact checked directly on the six concrete
  transitions by `decide`; `settle_length : settle.length = 6`.

The two routes agreeing is the point: the theorem predicts exactly what
`is_p_invariant(cycle, ones(Int, 6))` checks in `pflow-jl`.

### What is and is not proved

Proved (no `sorry` anywhere):
- the reindexing lemma and the lift theorem for gluing along an arbitrary
  place identification `f : P₁ ⊕ P₂ → Q` (surjectivity is the hypothesis
  `cov`; injectivity is *not* required, so arbitrary quotients of places are
  covered, and `f` being a bijection recovers the disjoint union);
- the Settle instance both by lifting and by brute force.

Not formalised:
- the **pushout universal property** — `f` is any place map, and nothing here
  says it is initial among such maps; the theorem does not need it;
- gluing along **transitions** (EventLinks) — only places are identified;
- marking-level statements — `yᵀC = 0` is structural; "`yᵀm` is constant on
  reachable markings" is the standard consequence but is not stated here;
- the converse / liveness non-lifting counterexample (roadmap "Later").

Encoding choices worth knowing: `pre`/`post` are `Int`-valued (not `Nat`) so
one sum type serves everywhere; a place type is *any* type with an explicit
enumeration list, so no Fintype machinery is needed. The invariant is
relative to the enumeration `L`; for the concrete types `Ch`, `Q6` the lists
are complete and duplicate-free (`Q6.all_nodup`, `Q6.all_cov`).
