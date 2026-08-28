# Graph Report - go-mermaid  (2026-08-28)

## Corpus Check
- 50 files · ~40,128 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 705 nodes · 1728 edges · 34 communities (33 shown, 1 thin omitted)
- Extraction: 77% EXTRACTED · 23% INFERRED · 0% AMBIGUOUS · INFERRED: 396 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `cd757fe4`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- encode2
- xmlEscape
- NewRenderOptions
- NewFlowchart
- FlowchartDiagram
- RenderOptions
- json/encoder.go
- newRenderer
- StateDiagram
- SequenceDiagram
- go-mermaid
- Diagram
- ERDiagram
- encode
- mmd/encoder_test.go
- go-mermaid Test Suite Implementation Plan
- errors_test.go
- FRD-017-3: go-mermaid — Pure Go Mermaid Diagram Library
- 6. SVG Renderer Technical Specification
- 4. Architecture
- options_test.go
- types_test.go
- 12. Implementation Plan
- 10. Testing Requirements
- go-mermaid
- 1. Overview
- 3. Output Format Requirements
- 9. Integration: Replacing Existing MermaidGenerator (alteryx2talend)
- TestJSONAllDiagramTypes
- 2. Diagram Type Requirements
- 5. Go Version Requirements
- 15. Acceptance Criteria
- 7. PNG Renderer
- github.com/iokdigital/go-mermaid

## God Nodes (most connected - your core abstractions)
1. `NewFlowchart()` - 120 edges
2. `NewRenderOptions()` - 70 edges
3. `NewRenderer()` - 50 edges
4. `encode2()` - 41 edges
5. `NewSequence()` - 31 edges
6. `NewClass()` - 24 edges
7. `encode()` - 24 edges
8. `SequenceDiagram` - 23 edges
9. `NewState()` - 23 edges
10. `FlowchartDiagram` - 21 edges

## Surprising Connections (you probably didn't know these)
- `TestHTMLAllDiagramTypes()` --calls--> `NewClass()`  [INFERRED]
  html/renderer_test.go → ast/class.go
- `TestJSONAllDiagramTypes()` --calls--> `NewClass()`  [INFERRED]
  json/encoder_test.go → ast/class.go
- `TestClassBasic()` --calls--> `NewClass()`  [INFERRED]
  mmd/encoder_test.go → ast/class.go
- `TestClassAllFormats()` --calls--> `NewClass()`  [INFERRED]
  tests/integration/format_combinations_test.go → ast/class.go
- `TestClassDiagramEmpty()` --calls--> `NewClass()`  [INFERRED]
  tests/negative/error_paths_test.go → ast/class.go

## Import Cycles
- None detected.

## Communities (34 total, 1 thin omitted)

### Community 0 - "encode2"
Cohesion: 0.07
Nodes (79): NewClass(), T, TestClassAddClass(), TestClassAddNote(), TestClassAddRelation(), TestClassEmpty(), TestClassMemberModifiers(), TestClassMemberVisibility() (+71 more)

### Community 1 - "xmlEscape"
Cohesion: 0.07
Nodes (50): ClassDiagram, ClassMember, ClassNote, ClassRelation, DiagramClass, Direction, MemberVisibility, RelationType (+42 more)

### Community 2 - "NewRenderOptions"
Cohesion: 0.11
Nodes (55): errorWriter, NewRenderOptions(), NewRenderer(), T, TestClassAllFormats(), TestContentTypes(), TestDOTValid(), TestERAllFormats() (+47 more)

### Community 3 - "NewFlowchart"
Cohesion: 0.14
Nodes (42): NewFlowchart(), T, TestFlowchartAddEdge(), TestFlowchartAddEdgeSelfLoop(), TestFlowchartAddNode(), TestFlowchartAddNodeDuplicate(), TestFlowchartAddSubgraph(), TestFlowchartConfidence() (+34 more)

### Community 4 - "FlowchartDiagram"
Cohesion: 0.11
Nodes (23): EdgeStyle, NodeShapeSyntax(), TestNodeShapeSyntax(), FlowchartDiagram, FlowEdge, FlowNode, NodeShape, Subgraph (+15 more)

### Community 5 - "RenderOptions"
Cohesion: 0.14
Nodes (31): B, LayoutOptions, RenderOptions, Resolution, DefaultLayoutOptions(), ResolutionDPI(), ResolutionScale(), Encode() (+23 more)

### Community 6 - "json/encoder.go"
Cohesion: 0.12
Nodes (33): classJSON, classMemberJSON, classNodeJSON, classNoteJSON, classRelJSON, diagramStateJSON, Encode(), Writer (+25 more)

### Community 7 - "newRenderer"
Cohesion: 0.10
Nodes (16): DiagramType, unexportedDiagram, fakeDiagram, fakeSequence, T, newRenderer(), TestContentType(), TestRendererHTMLProducesOutput() (+8 more)

### Community 8 - "StateDiagram"
Cohesion: 0.15
Nodes (22): DiagramState, StateDiagram, StateKind, StateNote, StateTransition, Encode(), encodeClass(), encodeER() (+14 more)

### Community 9 - "SequenceDiagram"
Cohesion: 0.13
Nodes (17): MessageStyle, Participant, ParticipantKind, SeqAlt, SeqLoop, SeqMessage, SeqNote, SequenceDiagram (+9 more)

### Community 10 - "go-mermaid"
Cohesion: 0.08
Nodes (25): 1. Build a diagram, 2. Render to any format, Core Concepts, Database schema visualization, Diagram types, Edge Styles Reference, Embed diagram in Markdown documentation, Error Handling (+17 more)

### Community 11 - "Diagram"
Cohesion: 0.14
Nodes (12): Diagram, FallbackFormatError, OutputFormat, Renderer, Encode(), Writer, T, svgOf() (+4 more)

### Community 12 - "ERDiagram"
Cohesion: 0.18
Nodes (15): Cardinality, ERAttribute, ERDiagram, EREntity, ERKey, ERRelation, Builder, encodeER() (+7 more)

### Community 13 - "encode"
Cohesion: 0.23
Nodes (17): Encode(), Writer, resolveCDN(), encode(), T, TestHTMLAllDiagramTypes(), TestHTMLCDNOverride(), TestHTMLCDNOverrideHTTP() (+9 more)

### Community 14 - "mmd/encoder_test.go"
Cohesion: 0.33
Nodes (18): assertGolden(), encode(), T, goldenPath(), TestClassBasic(), TestDuplicateNodeIDRejected(), TestERBasic(), TestFlowchartAllEdgeStyles() (+10 more)

### Community 15 - "go-mermaid Test Suite Implementation Plan"
Cohesion: 0.14
Nodes (13): Current State, go-mermaid Test Suite Implementation Plan, Implementation Stages, Notes, Overview, Stage 1: AST Package Tests (`ast/*_test.go` in packages), Stage 2: Diagram Package Tests, Stage 3: Negative Tests (+5 more)

### Community 16 - "errors_test.go"
Cohesion: 0.28
Nodes (12): T, TestErrDuplicateNodeID(), TestErrDuplicateNodeIDMessage(), TestErrInvalidFormat(), TestErrInvalidFormatMessage(), TestErrorsDistinct(), TestErrPNGSizeLimitExceeded(), TestErrPNGSizeLimitExceededMessage() (+4 more)

### Community 17 - "FRD-017-3: go-mermaid — Pure Go Mermaid Diagram Library"
Cohesion: 0.17
Nodes (11): 11.1 New Repository (`github.com/iokdigital/go-mermaid`), 11.2 Modified Files (`alteryx2talend`), 11.3 New Files (`alteryx2talend`), 11. File Changes, 13. Estimate, 14. Risks & Mitigations, 16. Resolved Questions, 8.1 HTML Renderer (+3 more)

### Community 18 - "6. SVG Renderer Technical Specification"
Cohesion: 0.18
Nodes (11): 6.10 Optional Node Hyperlinks, 6.1 Scope (Phase 1: Flowchart Only), 6.2 Rendering Pipeline, 6.3 Layout Options, 6.4 Layout Algorithm Details, 6.5 Viewport Computation, 6.6 Text Handling, 6.7 Self-loops and Cycles (+3 more)

### Community 19 - "4. Architecture"
Cohesion: 0.20
Nodes (10): 4.1 Repository Structure (`github.com/iokdigital/go-mermaid`), 4.2 Consuming Project Structure (`alteryx2talend`), 4.3 Core Interfaces and Types, 4.4 Render Options, 4.5 Flowchart AST, 4.6 Sequence Diagram AST, 4.7 State Diagram AST, 4.8 ER Diagram AST (+2 more)

### Community 20 - "options_test.go"
Cohesion: 0.39
Nodes (8): T, TestDefaultLayoutOptions(), TestLayoutOptionsSetters(), TestNewRenderOptions(), TestRenderOptionsMaxPNGBytesZero(), TestRenderOptionsSetters(), TestResolutionDPI(), TestResolutionScale()

### Community 21 - "types_test.go"
Cohesion: 0.43
Nodes (7): T, TestAllDiagramTypesCovered(), TestAllOutputFormatsCovered(), TestDiagramTypeUnique(), TestDiagramTypeValues(), TestOutputFormatUnique(), TestOutputFormatValues()

### Community 22 - "12. Implementation Plan"
Cohesion: 0.25
Nodes (8): 12. Implementation Plan, Phase 1: Library foundation — mmd output for all types (Week 1), Phase 2: HTML renderer + domain generators in alteryx2talend (Week 1–2), Phase 3: SVG renderer (Week 2–3), Phase 4: PNG and PDF (Week 3), Phase 5: alteryx2talend integration (Week 4), Phase 6: Dockerfile cleanup (Week 4), Phase 7: Cleanup (Week 5, if time allows)

### Community 23 - "10. Testing Requirements"
Cohesion: 0.29
Nodes (7): 10.1 Unit Tests (go-mermaid library), 10.2 Golden File Tests, 10.3 Negative Tests, 10.4 Syntax Change Negative Tests (`alteryx2talend`), 10.5 Backward Compatibility Tests (`alteryx2talend`), 10.6 Performance Benchmarks, 10. Testing Requirements

### Community 24 - "go-mermaid"
Cohesion: 0.29
Nodes (6): Diagram types, go-mermaid, Installation, Output formats, Requirements, Status

### Community 25 - "1. Overview"
Cohesion: 0.33
Nodes (6): 1.1 Summary, 1.2 Motivation, 1.3 Goals, 1.4 Library vs. Consuming Project Boundary, 1.5 Scope, 1. Overview

### Community 26 - "3. Output Format Requirements"
Cohesion: 0.33
Nodes (6): 3.1 Supported Output Formats, 3.2 Format Support Matrix by Diagram Type, 3.3 PNG Size Limit and HTML Fallback, 3.4 I/O Interface, 3.5 Additional Output Notes, 3. Output Format Requirements

### Community 27 - "9. Integration: Replacing Existing MermaidGenerator (alteryx2talend)"
Cohesion: 0.33
Nodes (6): 9.1 Current State, 9.2 Transition Plan, 9.3 Call Site Migration, 9.4 New CLI Flags (`cmd/analyze.go`), 9.5 Dockerfile.worker, 9. Integration: Replacing Existing MermaidGenerator (alteryx2talend)

### Community 28 - "TestJSONAllDiagramTypes"
Cohesion: 0.73
Nodes (5): encode(), T, TestJSONAllDiagramTypes(), TestJSONFlowchartValid(), TestJSONRoundTripAllFields()

### Community 29 - "2. Diagram Type Requirements"
Cohesion: 0.40
Nodes (5): 2.1 Phase 1 — Core (must have), 2.2 Phase 2 — Extended, 2.3 Explicitly Out of Scope, 2.4 Existing Coverage Gap, 2. Diagram Type Requirements

### Community 30 - "5. Go Version Requirements"
Cohesion: 0.40
Nodes (5): 5.1 Minimum Version: Go 1.20, 5.2 Generics Decision, 5.3 Compatibility Guidelines, 5.4 External Dependencies, 5. Go Version Requirements

### Community 31 - "15. Acceptance Criteria"
Cohesion: 0.67
Nodes (3): 15. Acceptance Criteria, Phase 1–4 Complete (library), Phase 5–6 Complete (alteryx2talend integration)

### Community 32 - "7. PNG Renderer"
Cohesion: 0.67
Nodes (3): 7.1 Resolution Presets, 7.2 Implementation, 7. PNG Renderer

## Knowledge Gaps
- **106 isolated node(s):** `Renderer`, `github.com/iokdigital/go-mermaid`, `Output formats`, `Diagram types`, `Requirements` (+101 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewFlowchart()` connect `NewFlowchart` to `xmlEscape`, `NewRenderOptions`, `FlowchartDiagram`, `RenderOptions`, `newRenderer`, `Diagram`, `encode`, `mmd/encoder_test.go`, `TestJSONAllDiagramTypes`?**
  _High betweenness centrality (0.185) - this node is a cross-community bridge._
- **Why does `NewRenderOptions()` connect `NewRenderOptions` to `encode2`, `NewFlowchart`, `RenderOptions`, `newRenderer`, `Diagram`, `encode`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Why does `NewSequence()` connect `encode2` to `NewRenderOptions`, `FlowchartDiagram`, `SequenceDiagram`, `encode`, `mmd/encoder_test.go`, `TestJSONAllDiagramTypes`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Are the 117 inferred relationships involving `NewFlowchart()` (e.g. with `TestFlowchartAddEdge()` and `TestFlowchartAddEdgeSelfLoop()`) actually correct?**
  _`NewFlowchart()` has 117 INFERRED edges - model-reasoned connections that need verification._
- **Are the 67 inferred relationships involving `NewRenderOptions()` (e.g. with `TestHTMLAllDiagramTypes()` and `TestHTMLCDNOverride()`) actually correct?**
  _`NewRenderOptions()` has 67 INFERRED edges - model-reasoned connections that need verification._
- **Are the 47 inferred relationships involving `NewRenderer()` (e.g. with `newRenderer()` and `TestClassAllFormats()`) actually correct?**
  _`NewRenderer()` has 47 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `encode2()` (e.g. with `NewRenderOptions()` and `Encode()`) actually correct?**
  _`encode2()` has 2 INFERRED edges - model-reasoned connections that need verification._