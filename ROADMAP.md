# Metamodel — Roadmap

**Naming.** *Metamodel*, capitalized, is this project: the cross-cutting
claim that a small declarative vocabulary can describe a domain and have
analysis, an application, and its own infrastructure derive
from that one artifact — no hand-written code that could disagree with it.
lowercase `metamodel`, always in code font, is `sim-pflow-xyz/pkg/metamodel`
— the Go package implementing one instance of the vocabulary (places,
transitions, rates, players, views). Every doc that uses both words says
which one it means; when in doubt, capital-M is the idea, `metamodel` is the
struct.

## Lineage — this name has already failed once, twice

Before this roadmap existed, "metamodel" was tried twice and abandoned both
times. Worth stating plainly rather than pretending the name starts clean:

- **2022–2024: five internal DSLs, one per language.**
  `pflow-xyz/metamodel-{rs,py,lua,js}` and `stackdump/metamodel-sol` each let
  you *write code* — Rust macros, Python calls, Lua tables, TS builders,
  Solidity structs — that constructed a Petri net. All five claimed to
  encode "the same" idea. None of them could prove it, because the model
  never existed as a document — only as five independent programs, each in
  a host language with its own semantics, that happened to agree by
  intention. They drifted apart quietly. Four are still private and
  dormant (last pushed 2022–2024); `metamodel-sol` is public and was still
  pushed to as recently as 2025-08-29, with nothing on it pointing a reader
  forward to what replaced it.
- **2026-01-17: a one-hour vision-page entry.** `modeldao-org` added a
  "Metamodel" section to its vision page and removed it — along with the
  "Petri Nets" section next to it — inside the same hour
  (`Replace Arcnet with Metamodel in vision page` →
  `Remove Petri Nets and Metamodel sections from vision`). Named before
  there was working evidence under it; pulled rather than half-built.
  `modeldao-org` itself was retired 2026-08-22, for unrelated reasons
  (empty governance state, a local devnet standing in for "mainnet").
- **2025-12-18:** `stackdump-com` explicitly removed the five `metamodel-*`
  repos from its tracked-repos list. Whatever they were, the ecosystem
  stopped counting them as current almost a year before this roadmap.

**The failure mode both times was the same: nothing forced convergence.**
An internal DSL's specification lives nowhere but its own source — you
cannot diff `metamodel-py`'s idea of a transition against `metamodel-rs`'s,
because neither one is a value, only a program. A vision-page label is
even thinner: a name with no artifact under it at all.

## What's different this time — self-description, not assertion

Both new pieces of evidence are things that already ship, not proposals:

- **The model is data, not code, in any language.** `model-is-the-app`'s
  claim — "the model is a value, not a program" — is the direct fix for
  the 2022 failure. A JSON-LD document extending schema.org's public
  vocabulary can be diffed, hashed, and validated by a machine that has
  never seen any of our runtimes. `pflow-polyglot` is the proof: one
  `model.json`, 20+ language implementations generated or hand-written,
  held to *one* golden trace (`parity/trace.golden`) — the exact
  convergence the five `metamodel-*` DSLs never had, because there was
  never a document to hold them to it. We are not inventing a private
  schema; we're extending schema.org's, the same way
  `cdn-stackdump-com` extends it for content addressing. Self-description
  means a stranger with no context can read what the extension claims and
  check it, the way SQL outlived every vendor that shipped it.
- **Journeys are a metamodel instance that describes itself completely.**
  In the ecosystem's (private) operations tooling, a *journey* is a markdown document whose frontmatter carries a full Petri net
  (places, transitions, `requires`/`blocked_by` arcs), the body carries one
  prompt per transition — is not an analogy for the thesis, it's a shipped
  case of it. "The document is both the documentation and the runtime —
  they cannot drift, because there is only one of them." Firing a step
  returns the next prompt by reading the frontmatter net's enabled
  transitions; there's no second copy of the control flow anywhere to fall
  out of sync. This is the smallest complete instance of Metamodel in the
  ecosystem today, and the book doesn't cite it yet (see items below).

## What Metamodel actually claims

Declare a domain in the small vocabulary (places, transitions, arcs, plus
the declared extensions — rates, outcomes, players, views, `apps/*.yaml`'s
own fields); derive analysis, the running application, and now the
infrastructure that serves it, from that one document, deterministically.
The vocabulary is universal not because we assert it but because the same
declare-then-derive move has now been run in domains that share nothing
except being "things in states, changing by rules": business operations
(`sim.pflow.xyz`), token standards, games, music, and — the newest and
least expected instance — this ecosystem's own devops (the private operations tooling's
`apps/*.yaml` → derived `~/services` entry, vhost, cert, synthetic).

## Inventory — where declare → derive already runs

| Domain | Artifact | Derives |
|---|---|---|
| Business simulation | sim-pflow-xyz model (JSON-LD) | analysis suite + running app (`model-is-the-app`) |
| Ecosystem devops | `apps/<name>.yaml` | `~/services` entry, nginx vhost, cert, Datadog synthetic, CLAUDE.md row |
| Agent workflows (private) | journey frontmatter (Petri net) | the prompts themselves — the runtime |
| Multi-language parity | `pflow-polyglot/model.json` | 20+ implementations + Lean proofs, one golden trace |
| Formal proof | `model.json` → `tools/codegen -lang lean` | kernel-checked theorems (see `PROOF-ROADMAP.md`) |
| Content addressing | `index.md` frontmatter (cdn-stackdump-com) | searchable facets, CID-addressed URL |

## Items

### Near-term

- [x] **Write the canonical Metamodel statement.** *(Done 2026-08-23: blog `metamodel.md` + book preface/epilogue.)* The thesis is currently
      implicit across five-plus posts (`model-is-the-app`,
      `little-language-thesis`, `earned-compression`,
      `declarative-differential-models`, `symmetric-monoidal-categories`)
      and this roadmap — nobody has said the word "Metamodel" about any of
      them yet. One doc (blog post or a new book appendix) that names the
      project, states the lineage above, and points to the inventory table
      as evidence.
- [x] **Add `sim-pflow-xyz` to the book's source list.** *(Done 2026-08-23; the
      operations tooling is private and is cited only generically.)* `CLAUDE.md`'s "Source Material" table only names
      blog-stackdump-com, petri-pilot, go-pflow — the two strongest recent
      instances of the thesis (`model-is-the-app`, the operations tooling as the infra
      instance) live in repos the book doesn't draw from yet.
- [x] **Cite journeys in the book as a shipped Metamodel instance** *(Done 2026-08-23: ch21 inventory table, glossary.)*,
      not just an implementation detail — it's the cleanest example
      available of "the document is the runtime."
- [ ] **Point the five dormant `metamodel-*` repos forward.** At minimum,
      a one-line README pointing each at `pflow-polyglot` (the living
      successor) — `metamodel-sol` is public and still visible to a
      stranger with nothing on it saying it's been superseded.

### Mid-term

- [ ] **A versioned vocabulary reference.** "The vocabulary's extensions
      are documented claims a validator checks" is currently true in
      scattered form (schema.go, validation.go, individual blog posts).
      Self-description requires one addressable spec a stranger can check
      an extension against, the way schema.org itself is checkable.
- [ ] **Name the next domain to test the thesis against.** Business,
      infra, tokens, games, and music are covered. Candidates worth
      scoping: `xenovoid`'s narrative engine (is its state machine actually
      declared in the vocabulary, or is that a looser analogy?); ZK proof
      generation past the single-net case `pflow-rs` already covers.

### Speculative / not yet decided

- **Formal disposition of the `metamodel-*` repos** — archive on GitHub
  with the pointer, or leave them as-is once the pointer exists. Not
  urgent; the lineage section above is the part that actually matters.
- **Whether "Metamodel" belongs in the ecosystem table** in
  `stackdump-com/CLAUDE.md` as a named row, given it isn't a deployed
  service — it's the thesis the table's own header sentence already
  gestures at ("centered on Petri nets as a universal abstraction").
