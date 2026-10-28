# New Directions Brief — for the next planning call

**Purpose:** Decide whether to invest beyond the completed 9-type scope. This brief lays out the options with effort, risk, and value. No code is proposed; each accepted option becomes an FRD + implementation plan.

**Status quo (no decision needed):** All 9 FRD-017-3 diagram types render to all 8 formats, Pele-oracle CI guards grammar conformance, and the PNG-text project (planned separately) closes the biggest quality gap. The library is shippable as-is.

---

## Option A: Mermaid source → AST parser (round-trip)

**What:** Today the library is generate-only (AST → formats). A parser would read hand-written `.mmd` into our AST types: `mmd.Decode(r) (Diagram, error)`.

**Why now:** Three concrete uses have already appeared during this session's work: (1) validating hand-written Mermaid in converters' golden files, (2) ingesting customer-supplied diagrams into pipelines, (3) round-trip editing (parse → transform → emit). The `tests/pele` harness proves demand for grammar conformance checking — a native parser makes that check local and offline.

**Approach:** Hand-written recursive-descent parser per diagram type (9 small parsers, not one grammar engine — Mermaid itself uses per-type grammars). Pele's `tests/compat` corpus + our golden `.mmd` files become the test oracle in both directions (parse our goldens → re-encode → byte-identical). Start with flowchart + pie (simplest grammars), then sequence/gantt, then the rest.

**Effort:** L (2–3 weeks). Grammar edge cases are the long tail — Mermaid's own parsers accumulated years of them.

**Risks:** (1) Never reaching 100% Mermaid parity, creating a false sense of compatibility — mitigate with a strict mode that errors loudly on unsupported syntax rather than mis-parsing. (2) Maintenance burden as Mermaid evolves — mitigate by testing against the pinned Pele/Mermaid 12.1.0 corpus, same as today.

**Value:** High if any consumer needs to ingest external diagrams; low if the library stays generation-only. **This is the key question for the call: is there a concrete consumer?** Without one, this is speculative generality — defer.

## Option B: Excluded diagram types (timeline first)

**What:** FRD-017-3 §2.3 excluded `gitGraph`, `xychart`, `timeline`, `block`, `packet`, `architecture` as irrelevant to conversion tooling. Revisit that call per type.

**Assessment per type:**

| Type | Case for | Case against | Effort |
|---|---|---|---|
| `timeline` | Migration roadmaps/milestones are a natural converter output; linear layout like gantt axis (known quantity) | Gantt already covers scheduling; timeline adds eras + styling | S (2–3 days, by analogy with quadrant) |
| `xychart` | Confidence/coverage statistics | Pie/quadrant already cover stats; Mermaid's xychart is beta-flavored | S-M |
| `kanban` / `journey` | Migration boards, user flows | No converter emits these today | S-M each |
| `gitGraph`, `packet`, `block`, `architecture`, C4, others | Completeness | No use case; several are beta/unstable upstream | Varies; low value |

**Recommendation if any:** `timeline` only, and only with a named consumer (which report, which pipeline?). The exclusion rationale from the FRD still holds for the rest.

**Effort:** S per type (the pie→gantt run established a repeatable 6-file slice + oracle pattern).

**Risks:** Low technical risk; the risk is scope creep — each new type is a permanent maintenance surface (encoders × formats + goldens + SVG).

## Option C: Hold the line (recommended default)

Ship what exists, finish the PNG-text project and the three logged follow-ups, and revisit A/B when a concrete consumer appears. Rationale: the library's value proposition (offline, dependency-free diagram generation for converters) is complete; both new directions add permanent maintenance surface without a requesting user.

---

## Decision needed

1. **Parser:** Is there a concrete consumer for mmd→AST ingestion? If yes → FRD amendment + phased plan (flowchart/pie first). If no → explicitly defer, revisit in 2 quarters.
2. **Timeline:** Is there a report or pipeline that wants timeline output? If yes → single-type slice like the others. If no → exclusion stands.
3. **Resourcing:** PNG-text + follow-ups already fill the near-term backlog; A/B stack behind them in any case.
