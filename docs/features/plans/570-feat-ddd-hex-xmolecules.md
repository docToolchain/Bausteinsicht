# #570 Plan: DDD / Hexagonal Architecture / xMolecules

_Status: Draft — living roadmap, last updated 2026-07-26._

_Umbrella ticket: [#570](https://github.com/docToolchain/Bausteinsicht/issues/570) — RFE: Model DDD / Hexagonal Architecture with xMolecules-aligned vocabulary. The ArchUnit code-governance bridge is tracked separately in [#562](https://github.com/docToolchain/Bausteinsicht/issues/562)._

This is the umbrella plan for how Bausteinsicht supports Hexagonal Architecture and tactical DDD, how it aligns with the language-neutral [xMolecules](https://xmolecules.org/) vocabulary, and how the model is meant to govern real code via ArchUnit. It is the *single home* for the vision, the scope delimitation ("the seam"), and the impedance analysis. Decisions extracted from here live in ADRs; the enduring concept lives in arc42; user-facing usage lives in the example READMEs. This document links to those homes rather than restating them.

## Scope

- **In scope:** modelling Hexagonal + DDD as architecture-as-code in Bausteinsicht; aligning the vocabulary with xMolecules; generating and synchronising code-level rules (ArchUnit) from the model; validating against a known reference codebase; publishing the story.
- **Out of scope:** mirroring the code's type graph inside the model; owning a JVM adapter's maintenance; jQAssistant and non-ArchUnit backends (anticipated later, tracked in the ArchUnit ADR — planned ADR-013).

## Vision (one paragraph)

A language-neutral, xMolecules-aligned architecture vocabulary in Bausteinsicht that makes Hexagonal/DDD **visible** (diagram), keeps it **enforceable** (constraints / fitness functions), and **couples it to code** through the stereotypes (ArchUnit, [#562](https://github.com/docToolchain/Bausteinsicht/issues/562)) — validated against Spring RESTBucks and published through docToolchain.

## The five layers

1. **Model** — express the architecture in the JSONC model. `kind` carries the structural role (`drivingAdapter`, `inboundPort`, `applicationService`, `aggregate`, `entity`, `valueObject`, `domainService`, `outboundPort`, `drivenAdapter`, plus `boundedContext`, `datastore`, `externalSystem`). `tags` carry the four zones (adapter / port / application / domain) for colour and view filtering. `metadata` carries machine-readable stereotypes that are not rendered visually. A `template.drawio` renders a real hexagon; `views` give context / ports-and-adapters / domain-model drill-down.
2. **Enforce** — architecture rules as fitness functions. `constraints` (`no-outgoing-dependency`, `allowed-relationship`, `no-circular-dependency`, …) encode the hexagonal dependency rules, checked by `bausteinsicht lint`, so CI fails when the model is wired against the architecture.
3. **Standardise** — anchor the vocabulary to xMolecules. xMolecules is the language-neutral umbrella (jMolecules/Java, nMolecules/.NET, phpMolecules/PHP); Bausteinsicht is the *model / diagram* member of that family. **Option A** (chosen — see [Vocabulary alignment](#vocabulary-alignment-option-a-vs-option-b), decision record ADR-012) keeps the didactic terms as `kind` and maps each element 1:1 onto the xMolecules stereotype via `metadata.jmolecules`, defaulting deterministically from `kind`.
4. **Bridge to code** — the round-trip ([#562](https://github.com/docToolchain/Bausteinsicht/issues/562), ArchUnit ADR — planned ADR-013). The stereotype is the bridge between architecture-as-code (the model) and code-level architecture. A generator turns the model into ArchUnit rules — or verifies code against the model — with the code→role mapping supplied by a `codeMapping` field, not by code annotations.
5. **Prove and publish** — dogfooding. Validate against Spring RESTBucks (its `hacking/hexagonal` branch uses jMolecules + jmolecules-archunit); publish a two-tier article on the docToolchain blog.

## Vocabulary alignment: Option A vs. Option B

The two ways to align the example vocabulary with xMolecules. The authoritative decision record is ADR-012; this section is the interim definition so the label "Option A" is never used undefined.

- **Option A (chosen):** keep the didactic hexagonal terms as element `kind` (`inboundPort`, `drivingAdapter`, …) and carry the xMolecules stereotype in `metadata.jmolecules` (`PrimaryPort`, `PrimaryAdapter`, …). Additive and non-breaking — `template.drawio` and the constraints stay unchanged; teaches both vocabularies. Demonstrated by the `examples/hexagonal-jmolecules/` example.
- **Option B (rejected):** rename the kinds directly to the xMolecules terms (`primaryPort`, `secondaryAdapter`, …). The model speaks jMolecules natively, but it is breaking — `template.drawio` (style lookup by `kind`) and the constraints (`from-kind`) must change, and the didactic driving/inbound naming is lost.

## The seam — what Bausteinsicht models vs. what stays in code

This is the central delimitation and the main defence against double maintenance.

| Granularity | Home of the truth |
|---|---|
| **Coarse** — bounded contexts, components, aggregates-as-elements, allowed dependencies | **Bausteinsicht model.** ~10 stable elements; architectural intent; changes at decision cadence. |
| **Fine** — type-level DDD semantics: `Association` vs. direct reference, `Identifier` types, `DomainEvent`, entity-inside-aggregate | **Code (jMolecules annotations).** Hundreds of classes; verified in place by jmolecules-archunit; changes at commit cadence. |

> **Important:** the two sides are two sources of truth at two *different* abstraction levels — not the same fact twice. Do not push the model down to the class level; that is where double maintenance and drift explode.

### Double maintenance: justified concern, bounded by the seam

The only fact that genuinely overlaps is the *stereotype* (`@AggregateRoot` in code ↔ `metadata.jmolecules: AggregateRoot` in the model). Redundancy is not double maintenance if it is *verified* or *generated*:

- **Cross-check:** CI asserts "code annotation ↔ model stereotype agree"; on divergence it fails and one side is fixed. Verified redundancy, like a test — not manual sync.
- **Generate one side from the other:** generate a jMolecules-annotated skeleton from the model, or derive the model's element list from an annotation scan. Single source plus a generated projection.
- **Rule of thumb:** double-maintenance pain arises only when *both* sides are hand-authored *and* can silently diverge. The ArchUnit ADR's (planned ADR-013) reverse/drift path plus a stereotype cross-check remove exactly that condition.

## Two ArchUnit worlds (comparison)

| Axis | jMolecules-ArchUnit | Bausteinsicht model-driven (#562) |
|---|---|---|
| Where the rule comes from | Universal, fixed DDD/hex laws ("grammar") | Project-specific rules from *this* model ("the sentence you wrote") |
| How code maps to a role | Annotations / interfaces in the code | `codeMapping` (element → package) in the *model* — no annotations required |
| Core question | "Is this a valid hexagon/DDD at all?" | "Does the code match this specific drawn architecture?" |
| Language | JVM only | Backend-extensible (ArchUnit now; go-arch-lint / ArchUnitNET / jQAssistant later) |

They are complementary axes, not competitors. Modelling with xMolecules and validating directly against the model needs neither `jmolecules-archunit` nor any code annotation; the stereotype in the model is a label that lets the generator additionally emit universal per-stereotype rules, and enables an optional cross-check when the code *is* annotated.

## Impedance / open fuzziness between the worlds

1. **Granularity** — element/package (Bausteinsicht) vs. type/class (jMolecules). The largest mismatch; drives the seam.
2. **Noun mismatch** — model `aggregate` often denotes the whole cluster/boundary; jMolecules `@AggregateRoot` is specifically the root type. One element ↔ many annotated classes.
3. **Relationship vs. reference semantics** — model edges (`uses`/`implements`) say _that_ A depends on B; jMolecules cares _how_ (direct vs. `Association` vs. `Identifier`). That nuance is below the model's resolution.
4. **Package assumption vs. annotation freedom** — `codeMapping`/codegraph are package-oriented (prefix match); jMolecules is deliberately package-layout-agnostic. Teams that pick jMolecules to _avoid_ enforcing package structure fit package-based mapping poorly. Mitigation: class/label anchors in `codeMapping` later (the ArchUnit ADR already foresees this).
5. **Stereotype drift not captured** — the reverse `codegraph` carries nodes + package edges, not stereotypes. A class gaining/losing `@AggregateRoot` does not surface in `diff` today. Stereotype-level sync would require the extractor to also emit annotations.
6. **Opposite direction of truth** — Bausteinsicht: model is truth (`toBe`), code conforms. jMolecules world: code is "architecturally evident", the code _is_ the truth. When they meet, decide per fact who leads; the ArchUnit ADR's stance is "model wins, edits proposed not silent".

## Now vs. later (user-facing outlook)

**Available today:**

- Model Hexagonal + DDD with custom kinds/tags/metadata; real hexagon rendering via `template.drawio`.
- **Concentric hexagonal rings** via the `nested` layout mode: nest zone containers (`adapterZone` ⊃ `portZone` ⊃ `domainZone`) and Bausteinsicht renders them as concentric hexagons, sized bottom-up; `metadata.side` (`top`/`bottom`) places driving/inbound on top and driven/outbound below (see the `hexagonal-ddd` Ports & Adapters view). Documented in ADR-005 and `05_sync_specification.adoc`.
- Enforce hexagonal rules via `constraints` + `bausteinsicht lint`.
- xMolecules vocabulary alignment via `metadata.jmolecules` (see the `hexagonal-jmolecules` example).

**Planned (staged):**

- `codeMapping` field on `Element` (prerequisite, ArchUnit ADR R5).
- Forward: generate ArchUnit rules from the model (mechanism A/B/C — staged, see the ArchUnit ADR).
- Reverse: `codegraph` extraction → `import-graph` → `asIs` → `diff` for drift.
- Optional stereotype cross-check; optional annotation-aware extraction.

> **Note:** user docs (manual / tutorial / example READMEs) describe only what works *today*, with a short "Outlook" that links here. Do not mix "what works now" with "what is planned" in user-facing pages.

## Validation and publishing

- **Validation target:** [Spring RESTBucks](https://github.com/odrotbohm/spring-restbucks), `hacking/hexagonal` branch — assigns code types to Application / Ports / Adapters and verifies them with jmolecules-archunit; the natural codebase to validate a generated rule set against.
- **Publishing (docToolchain blog), two tiers:**
  1. A known C4 model imported today (Structurizr DSL / LikeC4) — a quick proof, not the DDD story.
  2. Flagship: a jMolecules / Spring-Modulith reference app — the missing importer *is* the story and motivates [#562](https://github.com/docToolchain/Bausteinsicht/issues/562). State honestly "hand-modelled from source X" until an importer exists.

## Backlog / future ideas (beyond this plan's scope)

- **Shared / imported specification (`$ref`), or reusable constraint presets.** Today each model is self-contained — Bausteinsicht cannot share a `specification` across models, so the two hexagonal examples redefine the same kinds/relationship-kinds (vocabulary redundancy). Note: the *constraints* are **not** duplicated — only `hexagonal-ddd` carries them; the overlap is purely the vocabulary. Accepted for now (self-contained examples are a feature; each runs standalone). A future capability — an importable/shared spec (`$ref`) and/or the hexagonal constraint set as a reusable **pattern/preset** (relates to "Element Patterns / Topology Templates", #366) — would remove this redundancy generally, not just for these two examples. Candidate for its own issue.
- _(Shipped — moved to "Available today".)_ Concentric hexagonal-rings layout: implemented as the `nested` layout mode (zone containers `adapterZone`/`portZone`/`domainZone` rendered as concentric hexagons, `metadata.side` for vertical placement). Reads the model's **container nesting** (not the zone tags, as originally sketched here). Follow-ups: (i) incremental nested re-parenting — today `nested` only guarantees containment on a fresh page or after `sync --relayout`, so elements added to an existing nested page are placed flat until the next relayout; making the incremental path (`computePlacement`) nest-aware would remove that caveat; (ii) an optional tag→zone synthesis so flat, tag-only models also get rings without restructuring; (iii) minor layout polish (edge-label position, aggregate container sizing).

## Related documents

_Some targets currently live on the `feature/archunit` branch and become live links once branches merge to `main`._

- Decision — ArchUnit integration (planned ADR-013; currently drafted as `ADR-XXX-ArchUnit-Rule-Integration.adoc` + `adr-012-integration-variants.adoc` on `feature/archunit`, to be renumbered).
- Decision — xMolecules vocabulary alignment (Option A): `src/docs/arc42/ADRs/ADR-012-xMolecules-Vocabulary-Alignment.adoc`.
- Concept — arc42 §8 Crosscutting Concepts: DDD/Hex/xMolecules modelling approach _(planned)_.
- Spec — `codeMapping` field + constraint→ArchUnit mapping: `src/docs/spec/02_cli_specification.adoc`, `03_data_models.adoc` _(to extend)_.
- Examples (user-facing, today): `examples/hexagonal-ddd/`, `examples/hexagonal-jmolecules/`.
- RFE / issue: [#562](https://github.com/docToolchain/Bausteinsicht/issues/562) — Generate & synchronise code-level architecture rules (ArchUnit) from the model.

## Open questions / decisions pending

- Forward mechanism: A (generated test class) / B (PlantUML adapter) / C (native JSONC adapter) — staged in the ArchUnit ADR; C is the target, B the prototype baseline.
- Should the reverse `codegraph` carry stereotypes (to detect stereotype drift), or stay pure nodes+edges?
- `codeMapping` granularity: packages first; classes/labels/namespaces later — when?
- Ownership of the JVM-side adapter (Variant C) — who maintains it, and what stability guarantee does the JSONC contract carry?
- Which minimal stereotype subset does Bausteinsicht mirror, and which jMolecules concepts stay strictly in code?
