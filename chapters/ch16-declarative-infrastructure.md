# Declarative Infrastructure

**Learning objective**: Use JSON-LD and semantic vocabularies to make nets self-describing and composable.

The previous chapters built models, analyzed them, and even proved their transitions in zero knowledge. But how do those models travel between systems? How does a Petri net created in the browser editor reach the Go solver? How does a model saved before the schema grew remain valid after it?

The answer is JSON-LD — a serialization format that is purely declarative and monotonically expansive. These two properties, more than any feature list, make it reliable infrastructure for composable systems. This chapter explains why, using the three vocabularies the pflow ecosystem uses.

## Three Contexts, One Discipline

The pflow ecosystem uses three JSON-LD vocabularies. Each serves a different purpose. All share the same envelope.

**Petri net model** (pflow.xyz editor; trimmed — a saved file also carries `@version`, `token`, layout coordinates and authorship, as in the fixtures under [`parity/fixtures/`](https://github.com/pflow-xyz/pflow-xyz/tree/main/parity/fixtures)):

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

**ActivityPub actor** (federation; served by blog.stackdump.com from tens-city's [`pkg/activitypub/actor.go`](https://github.com/stackdump/tens-city/blob/main/pkg/activitypub/actor.go)):

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

This matters for interoperability. The same `.jsonld` Petri net file is read by three tools:

1. **JavaScript** — the pflow.xyz browser editor ([`public/petri-view.js`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/petri-view.js)) loads it, renders the net, and runs simulations
2. **Go parser** — go-pflow's `FromJSON` ([`parser/json.go`](https://github.com/pflow-xyz/go-pflow/blob/main/parser/json.go)) reads it for ODE-based analysis and validation
3. **Go code generator** — petri-pilot reads it through go-pflow's parser (`parseModel` in [`pkg/mcp/server.go`](https://github.com/pflow-xyz/petri-pilot/blob/main/pkg/mcp/server.go)), unfolds colored places into its own model, and compiles that into a service

That is two independent readers and three consumers. They do not coordinate at runtime; the file is assertions, not a protocol. What keeps the readers in agreement is a shared contract rather than a conversation: go-pflow pins its reading of the editor format to the goldens in [`parser/testdata/editor-shape/`](https://github.com/pflow-xyz/go-pflow/tree/main/parser/testdata/editor-shape) ([`editor_shape_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/parser/editor_shape_test.go)), and other readers replay the same inputs. None of the three runs a JSON-LD processor on the file. Each reads the fields it understands and ignores the rest: the code generator doesn't care about `x`/`y` layout coordinates, and the simulator doesn't care about `parents` lineage. The RDF reading of the file matters at one point, sealing, described below.

Declarative data degrades gracefully by design. Contrast this with imperative serialization — formats where the order of fields implies a processing sequence, or where consumers must execute embedded logic to reconstruct the data. JSON-LD sidesteps all of that. The `@context` fixes what each term means; each consumer decides what to do with it.

## Monotonic Expansion

The pflow.xyz context, served at `https://pflow.xyz/schema` from [`public/schema`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/schema) in the pflow-xyz repository, has this history there:

| Commit date | Terms added |
|------|------------|
| 2025-10-24 (first version in the repository) | `PetriNet`, `Place`, `Transition`, `Arrow`; marking and layout terms (`initial`, `capacity`, `weight`, `inhibitTransition`, `x`, `y`); the seal vocabulary (`PetriNetSeal`, `InvariantClaim`, `Signature`, `PetriRuntime`, `TransitionFireEvent`) |
| 2026-02-06 | net types (`WorkflowNet`, `ResourceNet`, `GameNet`, `ComputationNet`, `ClassificationNet`) and composition (`CompositeNet`, `links`, `DataLink`, `EventLink`, `TokenLink`, `GuardLink`); then `claim` and `bound` for invariant claims |
| 2026-06-12 | `parents`, the ordered list of CIDs a net derives from |

Each change added terms. The file's git history shows no term removed or redefined; the only deleted lines, on 2026-02-14, were comments that had made the file invalid JSON. A model that uses only the October 2025 terms is still valid under the current context, and the reason is structural: the context only grows.

This is **monotonic expansion**: new facts can be added, but existing facts are never retracted. It's the same discipline that makes append-only logs reliable and RDF graphs composable — and the same property the history places of Chapter 6 have. A schema that only grows is a past-tense object, like an event log or a write-once place: once absorbed, a fact never changes. In a monotonic system, learning more never invalidates what we already know.

Practically, adding terms cannot break an old model. A net saved before `CompositeNet` existed still loads in an editor that understands composition; the editor sees a net that composes with nothing — a valid state. The additions needed no migration scripts, no version negotiation, no "this file was created with an older version" warnings. What *can* break an old model is a change to a reader, and that is what the editor-shape goldens above guard against. Backwards compatibility comes from the structure of the schema, not from a policy.

## Content Addressing via Canonicalization

If JSON-LD is declarative, its identity should derive from *what it says*, not from how it was serialized. Two documents that make the same assertions — regardless of key order, whitespace, or field arrangement — should hash to the same value.

The URDNA2015 algorithm (RDF Dataset Normalization) makes this possible. It converts any JSON-LD document into a canonical set of N-Quads — sorted, deterministic, order-independent. From there, a standard hash produces a content identifier. The pflow-xyz sealer, `SealJSONLD` in [`internal/seal/seal.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/internal/seal/seal.go), does exactly this (excerpt; error handling and the type assertion around `m` elided):

```go
delete(m, "@id") // m is the parsed document; its own @id is not part of the preimage

proc := ld.NewJsonLdProcessor()
opts := ld.NewJsonLdOptions("")
opts.Format = "application/n-quads"
opts.Algorithm = "URDNA2015"
opts.DocumentLoader = cachedLoader // serves a fixed pflow.xyz context, no network fetch

normalized, _ := proc.Normalize(doc, opts) // canonical N-Quads
multihash, _ := mh.Sum([]byte(normalized.(string)), mh.SHA2_256, -1)
c := cid.NewCidV1(cid.DagJSON, multihash)        // CIDv1, dag-json codec
cidStr, _ := c.StringOfBase(multibase.Base58BTC) // base58btc, "z" prefix
```

The resulting CID becomes the model's `@id` — a self-certifying identifier. The sealer removes any existing top-level `@id` before hashing; otherwise the identifier would be part of its own preimage and re-sealing a stored file would give a different answer. It also resolves `https://pflow.xyz/schema` against a context compiled into it rather than fetched, so a CID cannot change because the served schema grew. If the model changes, the CID changes. If two independently created documents make the same assertions, they get the same CID. Identity follows from content, not from a registry or a counter.

What counts as content is decided by the context. `x`, `y` and `parents` are terms in the sealing context, so layout and lineage are assertions too: moving a place on the canvas changes the CID, and so does adding a parent or reversing the order of two. Reordering keys or whitespace does not.

These properties are tested in pflow-xyz, not just argued. [`cid_consistency_test.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/internal/seal/cid_consistency_test.go) seals one net with its keys in two different orders, and again after re-marshaling through Go's `encoding/json`, and requires a single CID. In [`parity_test.go`](https://github.com/pflow-xyz/pflow-xyz/blob/main/internal/seal/parity_test.go), `TestSealIgnoresTopLevelID` requires that adding or changing `@id` leaves the CID alone, and `TestParentsLineageContract` requires that `parents`, and their order, change it. The layout half of the previous paragraph follows from `x` and `y` being in the context; no test pins it. Finally, the browser's sealer ([`public/seal-cid.mjs`](https://github.com/pflow-xyz/pflow-xyz/blob/main/public/seal-cid.mjs)) and the Go sealer must both reproduce the CIDs recorded for five fixtures in [`parity/golden.json`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/golden.json); the coffee-shop fixture, for instance, seals to `z4EBG9jDsuUdvVN4bUXRHxoDG3p274FfhcJd8BiruRcE1jLCHi8` in both languages. `make test-parity` runs both sides. We have not re-run these tests for this chapter.

This property is what makes seals trustworthy: a seal is a commitment to a specific graph, and any party can verify it by re-canonicalizing and re-hashing.

### The Pipeline

```
JSON-LD document
    v strip the top-level @id
    v expand (resolve @context against the fixed copy)
    v normalize (URDNA2015 -> canonical N-Quads)
    v hash (SHA-256 multihash)
    v encode (CIDv1, dag-json codec, base58btc)
Content-addressed identifier
```

Every step is deterministic. Two parties processing the same assertions — even serialized differently — arrive at the same CID, which is what the JS/Go golden check above demonstrates on five fixtures.

## The Categorical View

There is a categorical reading of what `@context` does, and it should be stated at the strength it has been checked, which is not much. A JSON-LD context maps local terms (short names like `Place`, `Transition`) to global IRIs (like `https://pflow.xyz/schema#Place`):

$$@\text{context} : \text{LocalTerms} \to \text{GlobalIRIs}$$

Read as a functor between discrete categories, this is a function and nothing more. Extending a context — adding mappings while keeping the old ones — makes the old map the restriction of the new one to the old terms. The source blog post calls this a natural transformation; making that precise needs more structure on LocalTerms than a set of names, and nobody has written it down, so treat it as an analogy.

Combining vocabularies needs the most care. The IRIs of pflow.xyz, schema.org and ActivityStreams are namespace-disjoint, so on the global side the three behave like a coproduct. The local names are not disjoint: schema.org's context defines `Place` and `Person`, and ActivityStreams also defines `Person`. When a document lists several contexts, JSON-LD processes them in order and a term defined more than once takes its last definition. Combining contexts is ordered override, not a disjoint sum. Cross-Vocabulary References, below, shows where that bites.

Canonicalization supplies identity. URDNA2015 gives isomorphic RDF graphs the same canonical N-Quads, so — under the same context, and up to hash collision — two documents share a CID exactly when they assert the same graph.

## Connecting to Petri Nets

The declarative and monotonic properties of JSON-LD mirror properties of Petri nets themselves:

**Declarative.** A Petri net model is a declaration of structure — places, transitions, arcs. It doesn't prescribe an execution order or processing sequence. The firing rule determines what happens at runtime, but the model itself is static assertions about what's possible. JSON-LD serializes these assertions without adding procedural content.

**Monotonic — at the level of the document.** Adding a place or transition to a Petri net doesn't invalidate existing structure. New arcs can reference existing elements. A model that gains a field stays readable by tools that ignore it. This mirrors monotonic schema expansion: new terms never break existing ones.

The mirror stops at *behavior*. A document can only grow, but a composed net can shrink: linking two subnets (Chapter 4) restricts what each can do, because a link is a rendezvous or a gate rather than an addition (see "Composition Preserves Safety, Not Liveness" in [Appendix E: Categorical Foundations](appendix-e-categorical-foundations.md)). Monotonic serialization and monotonic semantics are different claims, and only the first one holds here.

**Content-addressable.** The canonicalized form of a Petri net model — sorted N-Quads — produces its identifier. Two documents with the same assertions, however they were authored or serialized, get the same CID; two nets with the same places and arcs but different layout or lineage do not, because layout and lineage are assertions too. Given a CID, you can verify that a model hasn't changed.

## Practical Implications

### Model Interchange

A model created in the pflow.xyz visual editor (JavaScript) can be:
1. Saved as JSON-LD with `@context: "https://pflow.xyz/schema"`
2. Loaded by go-pflow for ODE simulation and analysis
3. Compiled by petri-pilot into a running service
4. Sealed with a content-addressed CID for immutable reference
5. Shared via URL, embedded in a page as a `<petri-view>` element, or stored under its CID (the top-level example nets in pflow-xyz's `examples/` directory are stored under file names that are their CIDs)

Nobody converts the file between steps. Each tool converts it into its own internal form on read, but the file that travels is the same file, with the same assertions, consumed by different tools for different purposes.

### Schema Evolution

When the net-type and composition terms arrived in February 2026, existing models remained valid — they are nets that don't compose with anything. go-pflow's composition code makes the same guarantee from the other side: a bundle holding one subnet and no links flattens back to exactly that net (`TestFlattenIdentity` in [`metamodel/compose_test.go`](https://github.com/pflow-xyz/go-pflow/blob/main/metamodel/compose_test.go)). The readers handle the absence of new terms because a missing field is unknown, not invalid — JSON-LD's open-world reading, which the readers implement by ignoring keys they do not use.

### Cross-Vocabulary References

A pflow.xyz file can use schema.org terms without listing a second context. The served pflow.xyz context sets `@vocab` to `https://schema.org/`, so any key it does not define resolves there. Saved editor files use this for authorship (trimmed from [`parity/fixtures/net-a.jsonld`](https://github.com/pflow-xyz/pflow-xyz/blob/main/parity/fixtures/net-a.jsonld)):

```json
{
  "@context": "https://pflow.xyz/schema",
  "@type": "PetriNet",
  "author": {
    "@type": "Person",
    "identifier": "https://github.com/stackdump",
    "name": "stackdump"
  },
  "places": { ... }
}
```

`PetriNet` is defined by the pflow.xyz context; `author`, `Person` and `name` are not, and fall through `@vocab` to schema.org. (The context compiled into the two sealers sets `@vocab` to `https://pflow.xyz/schema#` instead, so inside a CID's preimage these terms land in the pflow namespace. Identity is unaffected, since every sealer uses the same fixed context, but the served and sealing contexts do not agree on these terms.)

Listing both contexts explicitly, as `["https://pflow.xyz/schema", "https://schema.org"]`, would be a mistake. schema.org's context defines `Place`; as the later definition it would win, and every place in the net would expand to schema.org's geographic `Place`. Order matters when contexts are combined, which is the ordered override described in the categorical section above.

## Infrastructure, Not a Feature

JSON-LD serves the pflow ecosystem as infrastructure, and it works because it does less: it asserts, it expands, it canonicalizes. Everything else is the consumer's problem.

That division of responsibility is what lets the tools compose. The model format doesn't know about editors, solvers, code generators, or blockchain bridges. It describes a Petri net — places, transitions, arcs — and each consumer takes what it needs.

The book's argument about Petri nets — a small set of primitives that combine to model a wide range of systems — applies to their serialization as well. Three JSON-LD keywords (`@context`, `@type`, `@id`) carry the self-description, the typing and the content address of a pflow model; the complexity lives in the applications.
