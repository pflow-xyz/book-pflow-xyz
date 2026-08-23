/-!
# P-invariants lift across gluing along places

Core Lean 4 only (no mathlib, no Std/Batteries).

A net over a place type `P` is a list of transitions, each a pair of
functions `pre post : P → Int`.  The incidence column of `t` is
`fun p => t.post p - t.pre p`; a weight vector `y : P → Int` is a
P-invariant (`yᵀC = 0`, `C` places × transitions) iff for every transition
`Σ_p y p * (post p - pre p) = 0`.

Finite sums are `List.sum` over an explicit enumeration `L : List P` of the
place type; the reindexing lemma `sumOver_reindex` (sum over fibres of `f`
= sum over the domain) is the crux and is proved by induction on the list.
-/

namespace Pflow

/-- Σ_{p ∈ L} g p. -/
def sumOver {P : Type} (L : List P) (g : P → Int) : Int :=
  (L.map g).sum

@[simp] theorem sumOver_nil {P : Type} (g : P → Int) : sumOver ([] : List P) g = 0 := rfl

@[simp] theorem sumOver_cons {P : Type} (p : P) (L : List P) (g : P → Int) :
    sumOver (p :: L) g = g p + sumOver L g := by
  simp [sumOver]

theorem sumOver_congr {P : Type} (L : List P) {g h : P → Int}
    (H : ∀ p, g p = h p) : sumOver L g = sumOver L h := by
  induction L with
  | nil => rfl
  | cons p L ih => simp [H p, ih]

theorem sumOver_zero {P : Type} (L : List P) : sumOver L (fun _ => (0 : Int)) = 0 := by
  induction L with
  | nil => rfl
  | cons p L ih => simp [ih]

theorem sumOver_add {P : Type} (L : List P) (g h : P → Int) :
    sumOver L (fun p => g p + h p) = sumOver L g + sumOver L h := by
  induction L with
  | nil => rfl
  | cons p L ih => simp [ih]; omega

theorem sumOver_sub {P : Type} (L : List P) (g h : P → Int) :
    sumOver L (fun p => g p - h p) = sumOver L g - sumOver L h := by
  induction L with
  | nil => rfl
  | cons p L ih => simp [ih]; omega

theorem sumOver_append {P : Type} (L M : List P) (g : P → Int) :
    sumOver (L ++ M) g = sumOver L g + sumOver M g := by
  induction L with
  | nil => simp
  | cons p L ih => simp [ih]; omega

/-- A sum of `if a = q then h q else 0` over a list not containing `a` is `0`. -/
theorem sumOver_ite_of_not_mem {Q : Type} [DecidableEq Q] (M : List Q) (a : Q)
    (h : Q → Int) (ha : a ∉ M) :
    sumOver M (fun q => if a = q then h q else 0) = 0 := by
  induction M with
  | nil => rfl
  | cons q M ih =>
    have hne : a ≠ q := fun e => ha (e ▸ List.mem_cons_self ..)
    have ha' : a ∉ M := fun m => ha (List.mem_cons_of_mem _ m)
    simp [hne, ih ha']

/-- Over a duplicate-free enumeration containing `a`, `Σ_q [a = q]·h q = h a`. -/
theorem sumOver_ite {Q : Type} [DecidableEq Q] (M : List Q) (a : Q) (h : Q → Int)
    (nd : M.Nodup) (ha : a ∈ M) :
    sumOver M (fun q => if a = q then h q else 0) = h a := by
  induction M with
  | nil => cases ha
  | cons q M ih =>
    have nd' : M.Nodup := (List.nodup_cons.mp nd).2
    have hq : q ∉ M := (List.nodup_cons.mp nd).1
    by_cases e : a = q
    · subst e
      simp [sumOver_ite_of_not_mem M a h hq]
    · have ha' : a ∈ M := by
        cases List.mem_cons.mp ha with
        | inl e' => exact absurd e' e
        | inr m => exact m
      simp [e, ih nd' ha']

/-- **Reindexing.**  Summing `y q` times the fibre-sum of `g` over `f` equals
summing `y (f p) * g p` over the domain.  This is the whole proof of the
lift theorem; everything else is bookkeeping. -/
theorem sumOver_reindex {P Q : Type} [DecidableEq Q] (L : List P) (M : List Q)
    (f : P → Q) (y : Q → Int) (g : P → Int)
    (nd : M.Nodup) (cov : ∀ p, f p ∈ M) :
    sumOver M (fun q => y q * sumOver L (fun p => if f p = q then g p else 0))
      = sumOver L (fun p => y (f p) * g p) := by
  induction L with
  | nil =>
    simp only [sumOver_nil, Int.mul_zero]
    exact sumOver_zero M
  | cons p L ih =>
    have step : ∀ q, y q * sumOver (p :: L) (fun p => if f p = q then g p else 0)
        = (if f p = q then y q * g p else 0)
          + y q * sumOver L (fun p => if f p = q then g p else 0) := by
      intro q
      rw [sumOver_cons, Int.mul_add]
      by_cases e : f p = q <;> simp [e]
    rw [sumOver_congr M step, sumOver_add, ih, sumOver_cons]
    rw [sumOver_ite M (f p) (fun q => y q * g p) nd (cov p)]

/-! ## Nets -/

/-- A transition: pre- and post-multiset over the places. -/
structure Trans (P : Type) where
  pre  : P → Int
  post : P → Int

/-- Incidence column of a transition: `C[p, t] = post t p - pre t p`. -/
def Trans.col {P : Type} (t : Trans P) (p : P) : Int := t.post p - t.pre p

/-- A net over `P` is its list of transitions (the columns of `C`). -/
abbrev Net (P : Type) := List (Trans P)

/-- `y` is a P-invariant of `N` (places enumerated by `L`): `yᵀ C = 0`,
i.e. for every transition `t`, `Σ_p y p * C[p,t] = 0`. -/
def IsInvariant {P : Type} (L : List P) (N : Net P) (y : P → Int) : Prop :=
  ∀ t ∈ N, sumOver L (fun p => y p * t.col p) = 0

instance {P : Type} (L : List P) (N : Net P) (y : P → Int) [DecidableEq P] :
    Decidable (IsInvariant L N y) :=
  inferInstanceAs (Decidable (∀ t ∈ N, sumOver L (fun p => y p * t.col p) = 0))

theorem IsInvariant_append {P : Type} (L : List P) (N₁ N₂ : Net P) (y : P → Int)
    (h₁ : IsInvariant L N₁ y) (h₂ : IsInvariant L N₂ y) :
    IsInvariant L (N₁ ++ N₂) y := by
  intro t ht
  cases List.mem_append.mp ht with
  | inl h => exact h₁ t h
  | inr h => exact h₂ t h

/-! ## Pushforward along a place map and gluing -/

/-- Push a transition forward along `f : P → Q`:
`pre' q = Σ_{p, f p = q} pre p`, likewise for `post`. -/
def Trans.push {P Q : Type} [DecidableEq Q] (L : List P) (f : P → Q) (t : Trans P) :
    Trans Q where
  pre  := fun q => sumOver L (fun p => if f p = q then t.pre p else 0)
  post := fun q => sumOver L (fun p => if f p = q then t.post p else 0)

theorem Trans.push_col {P Q : Type} [DecidableEq Q] (L : List P) (f : P → Q)
    (t : Trans P) (q : Q) :
    (t.push L f).col q = sumOver L (fun p => if f p = q then t.col p else 0) := by
  unfold Trans.push Trans.col
  simp only
  rw [← sumOver_sub]
  apply sumOver_congr
  intro p
  by_cases e : f p = q <;> simp [e]

/-- Push a whole net forward along a place map. -/
def Net.push {P Q : Type} [DecidableEq Q] (L : List P) (f : P → Q) (N : Net P) : Net Q :=
  N.map (Trans.push L f)

/-- Glue two nets along places: `f : P₁ ⊕ P₂ → Q` identifies places
(the pushout of the place sets along the shared boundary; any `f` is
allowed, surjectivity is the hypothesis `cov` of the theorems below). -/
def glue {P₁ P₂ Q : Type} [DecidableEq Q] (L₁ : List P₁) (L₂ : List P₂)
    (f : P₁ ⊕ P₂ → Q) (N₁ : Net P₁) (N₂ : Net P₂) : Net Q :=
  N₁.push L₁ (fun p => f (Sum.inl p)) ++ N₂.push L₂ (fun p => f (Sum.inr p))

/-! ## The theorem -/

/-- One half of the lift: an invariant of `N` induces an invariant of the
pushforward of `N`, provided `y` agrees with `y₀` along `f`. -/
theorem IsInvariant_push {P Q : Type} [DecidableEq Q] (L : List P) (M : List Q)
    (nd : M.Nodup) (f : P → Q) (cov : ∀ p, f p ∈ M)
    (N : Net P) (y₀ : P → Int) (y : Q → Int)
    (agree : ∀ p, y (f p) = y₀ p)
    (inv : IsInvariant L N y₀) :
    IsInvariant M (N.push L f) y := by
  intro t' ht'
  obtain ⟨t, ht, rfl⟩ := List.mem_map.mp ht'
  calc sumOver M (fun q => y q * (t.push L f).col q)
      = sumOver M (fun q => y q * sumOver L (fun p => if f p = q then t.col p else 0)) := by
        apply sumOver_congr; intro q; rw [Trans.push_col]
    _ = sumOver L (fun p => y (f p) * t.col p) := sumOver_reindex L M f y t.col nd cov
    _ = sumOver L (fun p => y₀ p * t.col p) := by
        apply sumOver_congr; intro p; rw [agree]
    _ = 0 := inv t ht

/-- **P-invariants lift across gluing along places.**
If `y₁ᵀC₁ = 0`, `y₂ᵀC₂ = 0`, and `y : Q → Int` agrees with `y₁` and `y₂` on
the identified places, then `yᵀC = 0` for the glued net. -/
theorem invariant_lift {P₁ P₂ Q : Type} [DecidableEq Q]
    (L₁ : List P₁) (L₂ : List P₂) (M : List Q) (nd : M.Nodup)
    (f : P₁ ⊕ P₂ → Q) (cov : ∀ s, f s ∈ M)
    (N₁ : Net P₁) (N₂ : Net P₂)
    (y₁ : P₁ → Int) (y₂ : P₂ → Int) (y : Q → Int)
    (agree₁ : ∀ p, y (f (Sum.inl p)) = y₁ p)
    (agree₂ : ∀ p, y (f (Sum.inr p)) = y₂ p)
    (inv₁ : IsInvariant L₁ N₁ y₁) (inv₂ : IsInvariant L₂ N₂ y₂) :
    IsInvariant M (glue L₁ L₂ f N₁ N₂) y :=
  IsInvariant_append M _ _ y
    (IsInvariant_push L₁ M nd _ (fun p => cov (Sum.inl p)) N₁ y₁ y agree₁ inv₁)
    (IsInvariant_push L₂ M nd _ (fun p => cov (Sum.inr p)) N₂ y₂ y agree₂ inv₂)

/-! ## The Settle cycle (three channels glued into a three-party cycle) -/

/-- Places of one settlement channel `x → y`. -/
inductive Ch | x | y | pend
  deriving DecidableEq, Repr

def Ch.all : List Ch := [.x, .y, .pend]

/-- `send`: debit `x` into `pend`; `settle`: credit `pend` into `y`. -/
def channel : Net Ch :=
  [ ⟨fun p => if p = .x then 1 else 0, fun p => if p = .pend then 1 else 0⟩,
    ⟨fun p => if p = .pend then 1 else 0, fun p => if p = .y then 1 else 0⟩ ]

/-- Places of the glued three-party cycle. -/
inductive Q6 | a | b | c | pab | pbc | pca
  deriving DecidableEq, Repr

def Q6.all : List Q6 := [.a, .b, .c, .pab, .pbc, .pca]

theorem Q6.all_nodup : Q6.all.Nodup := by decide
theorem Q6.all_cov : ∀ q : Q6, q ∈ Q6.all := by
  intro q; cases q <;> decide

/-- Boundary identifications: `ch_ab` glues `x ↦ a, y ↦ b`; `ch_bc`: `x ↦ b, y ↦ c`;
`ch_ca`: `x ↦ c, y ↦ a`. -/
def fab : Ch → Q6 | .x => .a | .y => .b | .pend => .pab
def fbc : Ch → Q6 | .x => .b | .y => .c | .pend => .pbc
def fca : Ch → Q6 | .x => .c | .y => .a | .pend => .pca

/-- `ch_ab` glued with `ch_bc`, then with `ch_ca`. -/
def settle : Net Q6 :=
  glue Ch.all Ch.all
    (fun s => match s with
      | .inl p => fab p
      | .inr p => fbc p)
    channel channel
  ++ channel.push Ch.all fca

def ones {P : Type} : P → Int := fun _ => 1

/-- The accounting identity on one channel: `x + y + pend` is conserved. -/
theorem channel_ones : IsInvariant Ch.all channel ones := by decide

/-- The lifted statement: `a + b + c + p_ab + p_bc + p_ca` is conserved on the
cycle, obtained from `channel_ones` via `invariant_lift`/`IsInvariant_push`. -/
theorem settle_ones_lifted : IsInvariant Q6.all settle ones :=
  IsInvariant_append _ _ _ _
    (invariant_lift Ch.all Ch.all Q6.all Q6.all_nodup _
      (fun _ => Q6.all_cov _) channel channel ones ones ones
      (fun _ => rfl) (fun _ => rfl) channel_ones channel_ones)
    (IsInvariant_push Ch.all Q6.all Q6.all_nodup fca (fun _ => Q6.all_cov _)
      channel ones ones (fun _ => rfl) channel_ones)

/-- The same fact checked directly on the six concrete transitions. -/
theorem settle_ones_decide : IsInvariant Q6.all settle ones := by decide

/-- The glued net really has six transitions. -/
theorem settle_length : settle.length = 6 := by decide

end Pflow
