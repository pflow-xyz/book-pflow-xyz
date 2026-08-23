# Declarative Infrastructure

**Learning objective**: Use JSON-LD and semantic vocabularies to make nets self-describing and composable.

The previous chapters built models, analyzed them, and even proved their transitions in zero knowledge. But how do those models travel between systems? How does a Petri net created in the browser editor reach the Go solver? How does a model authored in 2024 remain valid when the schema grows in 2026?

The answer is JSON-LD — a serialization format that is purely declarative and monotonically expansive. These two properties, more than any feature list, make it reliable infrastructure for composable systems. This chapter explains why, using the three vocabularies that the pflow ecosystem actually uses in production.

## Three Contexts, One Discipline

The pflow ecosystem uses three JSON-LD vocabularies. Each serves a different purpose. All share the same envelope.

**Petri net model** (pflow.xyz editor):

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "PetriNet",
  "places": {
    "Idle": { "@type": "Place", "initial": [1], "capacity": [1] }
  },
  "transitions": {
    "Brew": { "@type": "Transition" }
  },
  "arcs": [
    { "@type": "Arrow", "source": "Idle", "target": "Brew", "weight": [1] }
  ]
}
```

**Blog metadata** (schema.org):

```json
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "JSON-LD as Declarative Infrastructure",
  "author": { "@type": "Person", "name": "stackdump" },
  "datePublished": "2026-02-15T00:00:00Z"
}
```

**ActivityPub actor** (federation):

```json
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://w3id.org/security/v1"
  ],
  "type": "Person",
  "preferredUsername": "myork",
  "inbox": "https://blog.stackdump.com/users/myork/inbox",
  "publicKey": { "id": "...#main-key", "publicKeyPem": "..." }
}
```

Each snippet is self-describing. Each links to a vocabulary that defines its terms. And none of them contain instructions — they are pure assertions.

## Purely Declarative

A JSON-LD document is a serialized RDF graph: a set of subject-predicate-object triples. It makes statements about the world but never tells you what to *do* with them. There are no callbacks, no event handlers, no conditionally-included fields.

This matters for interoperability. The same `.jsonld` Petri net file is consumed by at least three independent interpreters:

1. **JavaScript** — the pflow.xyz browser editor loads it, renders the net, and runs simulations
2. **Go parser** — go-pflow reads the same file for ODE-based analysis and validation
3. **Go code generator** — petri-pilot compiles it into executable service modules

None of these consumers coordinate with each other. They don't need to. The file is assertions, not a protocol. Each consumer extracts the triples it understands and ignores the rest. The code generator doesn't care about `x`/`y` layout coordinates; the visual editor doesn't care about `Seal` metadata.

Declarative data degrades gracefully by design. Contrast this with imperative serialization — formats where the order of fields implies a processing sequence, or where consumers must execute embedded logic to reconstruct the data. JSON-LD sidesteps all of that. The `@context` resolves local terms to global IRIs. The consumer interprets the graph. Nothing in between.

## Monotonic Expansion

The pflow.xyz schema has grown three times since its introduction:

| Year | Terms Added |
|------|------------|
| 2024 | `PetriNet`, `Place`, `Transition`, `Arrow`, `Person` |
| 2025 | `Seal`, `Invariant`, `Guard`, `TokenSet` |
| 2026 | `CompositeNet`, `NetType`, `Link`, `Interface` |

Each expansion introduced new terms. No existing term was removed or redefined. A model authored in 2024 using only `PetriNet`, `Place`, `Transition`, and `Arrow` remains valid under the 2026 schema — not because we tested backwards compatibility, but because the schema only grows.

This is **monotonic expansion**: new facts can be added, but existing facts are never retracted. It's the same discipline that makes append-only logs reliable and RDF graphs composable — and the same property the history places of Chapter 6 have. A schema that only grows is a past-tense object, like an event log or a write-once place: once absorbed, a fact never changes. In a monotonic system, learning more never invalidates what we already know.

Practically, this means old models never break. A Petri net saved before `Seal` existed still loads in an editor that understands seals. The editor sees a net without seal metadata — a valid state. No migration scripts, no version negotiation, no "this file was created with an older version" warnings. Backwards compatibility emerges from structure, not policy.

## Content Addressing via Canonicalization

If JSON-LD is declarative, its identity should derive from *what it says*, not from how it was serialized. Two documents that make the same assertions — regardless of key order, whitespace, or field arrangement — should hash to the same value.

The URDNA2015 algorithm (RDF Dataset Normalization) makes this possible. It converts any JSON-LD document into a canonical set of N-Quads — sorted, deterministic, order-independent. From there, a standard hash produces a content identifier.

```go
proc := ld.NewJsonLdProcessor()
opts := ld.NewJsonLdOptions("")
opts.Format = "application/n-quads"
opts.Algorithm = "URDNA2015"

normalized, _ := proc.Normalize(doc, opts)     // canonical N-Quads
multihash, _ := mh.Sum([]byte(normalized.(string)), mh.SHA2_256, -1)
cid := cid.NewCidV1(cid.DagJSON, multihash)    // CIDv1 with base58btc
```

The resulting CID becomes the model's `@id` — a self-certifying identifier. If the model changes, the CID changes. If two independently-created models happen to describe the same graph, they get the same CID. Identity follows from content, not from a registry or a counter.

This property is what makes seals trustworthy: a seal is a commitment to a specific graph, and any party can verify it by re-canonicalizing and re-hashing.

### The Pipeline

```
JSON-LD document
    v expand (resolve @context)
    v normalize (URDNA2015 -> canonical N-Quads)
    v hash (SHA-256)
    v encode (CIDv1 with base58btc)
Content-addressed identifier
```

Every step is deterministic. Two parties processing the same assertions — even serialized differently — arrive at the same CID. This is the content-addressing property: identity from substance, not from naming.

## The Categorical View

There's a clean categorical reading of what `@context` does. A JSON-LD context is a mapping from local terms (short names like `Place`, `Transition`) to global IRIs (like `https://pflow.xyz/schema#Place`). This is a functor:

$$@\text{context} : \text{LocalTerms} \to \text{GlobalIRIs}$$

It maps the local category of terms used in a document to the global category of well-defined identifiers. Extending a context — adding new term mappings while preserving existing ones — is a **natural transformation** between functors: the old mapping still holds, and new mappings are layered on top.

The three vocabularies (pflow.xyz, schema.org, ActivityStreams) compose as a **coproduct**. Each vocabulary contributes its terms to the combined graph without collision, because the IRIs are namespace-disjoint. A document can reference all three contexts simultaneously, and the terms resolve unambiguously.

Content addressing gives us identity in this category. Two objects (JSON-LD documents) are equal if and only if their canonical forms are equal — which is exactly what a content-addressed identifier witnesses.

> **JSON-LD, categorically:** `@context` is a functor from local names to global identifiers. Schema extension is a natural transformation. Independent vocabularies compose as coproducts. Canonicalization provides identity.

## Connecting to Petri Nets

The declarative and monotonic properties of JSON-LD mirror properties of Petri nets themselves:

**Declarative.** A Petri net model is a declaration of structure — places, transitions, arcs. It doesn't prescribe an execution order or processing sequence. The firing rule determines what happens at runtime, but the model itself is static assertions about what's possible. JSON-LD serializes these assertions without adding procedural content.

**Monotonic — at the level of the document.** Adding a place or transition to a Petri net doesn't invalidate existing structure. New arcs can reference existing elements. A model that gains a field stays readable by tools that ignore it. This mirrors monotonic schema expansion: new terms never break existing ones.

The mirror stops at *behavior*. A document can only grow, but a composed net can shrink: linking two subnets (Chapter 4) restricts what each can do, because a link is a rendezvous or a gate rather than an addition. Monotonic serialization and monotonic semantics are different claims, and only the first one holds here.

**Content-addressable.** The canonicalized form of a Petri net model — sorted N-Quads — produces a unique identifier. Two models with the same structure, regardless of how they were authored or serialized, get the same CID. This enables reproducible builds: given a CID, you can verify that a model hasn't changed.

## Practical Implications

### Model Interchange

A model created in the pflow.xyz visual editor (JavaScript) can be:
1. Saved as JSON-LD with `@context: "https://pflow.xyz/schema"`
2. Loaded by go-pflow for ODE simulation and analysis
3. Compiled by petri-pilot into a running service
4. Sealed with a content-addressed CID for immutable reference
5. Shared via URL, embedded in blog posts, or stored on IPFS

No format conversion between steps. The same file, the same assertions, consumed by different tools for different purposes.

### Schema Evolution

When the pflow schema adds `Guard` expressions (2025), existing models without guards continue to work. When `CompositeNet` arrived (2026), simple models remained valid — they're just nets that don't compose with anything, and a bundle holding one subnet and no links flattens back to exactly that net. The consuming tools handle the absence of new terms gracefully because JSON-LD's open-world assumption means missing fields are simply unknown, not invalid.

### Cross-Vocabulary References

A single document can reference multiple vocabularies:

```json
{
  "@context": [
    "https://pflow.xyz/schema",
    "https://schema.org"
  ],
  "@type": "PetriNet",
  "name": "Coffee Shop",
  "author": { "@type": "Person", "name": "stackdump" },
  "places": { ... }
}
```

The `name` and `author` come from schema.org. The `places` come from pflow.xyz. Both resolve unambiguously because `@context` is a functor — it maps each local term to exactly one global IRI.

## Infrastructure, Not a Feature

JSON-LD is not a feature of the pflow ecosystem but infrastructure — the kind that works precisely because it does less, not more. It asserts, it expands, it canonicalizes. Everything else is the consumer's problem.

This division of responsibility is what makes composable systems possible. The model format doesn't know about editors, solvers, code generators, or blockchain bridges. It doesn't need to. It describes a Petri net — places, transitions, arcs — and each consumer extracts what it needs.

The same philosophy that makes Petri nets universal — a small set of primitives that combine to model anything — applies to their serialization. A small set of JSON-LD properties (`@context`, `@type`, `@id`) provides the infrastructure for self-describing, composable, content-addressable models. The complexity lives in the applications. The infrastructure stays simple.
