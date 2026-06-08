# go-mermaid Test Suite Implementation Plan

**Created:** 2026-06-08
**Status:** Planned

## Overview

Comprehensive test suite covering: unit tests, negative/error path tests, integration tests, and property-based tests. Tests live in `tests/` directory (separate from package tests in `*_test.go` files).

## Current State

| Package | Coverage | Test Files |
|---------|----------|------------|
| `ast` | 0% | None |
| `diagram` | 0% | None |
| `dot` | 58.9% | encoder_test.go |
| `html` | 86.7% | renderer_test.go |
| `json` | 65.4% | encoder_test.go |
| `mmd` | 74.6% | encoder_test.go |
| `pdf` | 88.9% | renderer_test.go |
| `png` | 76.9% | renderer_test.go |
| `render` | 71.7% | renderer_test.go |
| `svg` | 86.8% | renderer_test.go, phase2_renderer_test.go, layout_test.go |

## Test Organization

```
tests/
├── unit/                    # Unit tests for package internals
│   ├── ast/
│   │   ├── flowchart_test.go
│   │   ├── sequence_test.go
│   │   ├── state_test.go
│   │   ├── er_test.go
│   │   └── class_test.go
│   └── diagram/
│       ├── errors_test.go
│       ├── options_test.go
│       └── types_test.go
├── negative/                # Error path and invalid input tests
│   ├── invalid_diagram_test.go
│   ├── malformed_ast_test.go
│   └── boundary_test.go
├── integration/            # End-to-end tests
│   ├── format_combinations_test.go
│   └── render_pipeline_test.go
└── property/               # Property-based and golden file tests
    ├── roundtrip_test.go
    ├── svg_validity_test.go
    └── golden_files_test.go
```

**Note:** Package-level tests in `*_test.go` files remain in place. The `tests/` directory provides comprehensive integration and cross-package testing.

## Test Fixtures

- **Simple fixtures:** Inline as const strings in Go files
- **Complex fixtures:** Golden files in `testdata/` directories

```
testdata/
├── ast/
│   ├── flowchart/         # FlowchartDiagram test fixtures
│   ├── sequence/          # SequenceDiagram test fixtures
│   ├── state/             # StateDiagram test fixtures
│   ├── er/                # ERDiagram test fixtures
│   └── class/             # ClassDiagram test fixtures
├── render/                # Renderer output fixtures
└── encoders/              # Encoder output fixtures
    ├── mmd/
    ├── svg/
    ├── json/
    └── dot/
```

**Naming convention:** `testdata/{package}/{test_name}/{input_name}.{ext}`

Example: `testdata/svg/renderer_test/TestEncode_Edges.svg`

## Implementation Stages

### Stage 1: AST Package Tests (`ast/*_test.go` in packages)

**Goal:** 100% coverage for `ast` package

**Files to create:**
- `ast/flowchart_test.go` - Node/edge/subgraph CRUD, duplicate detection, sanitizeID, shape syntax
- `ast/sequence_test.go` - Messages, actors, notes
- `ast/state_test.go` - States, transitions, fork/join
- `ast/er_test.go` - Entities, relationships, attributes
- `ast/class_test.go` - Classes, methods, visibility

**Priority tests:**
1. `NewFlowchart` constructor
2. `AddNode` - success, duplicate rejection
3. `AddEdge` - basic, with label
4. `AddSubgraph` - nesting
5. `MustAddNode` - panic on duplicate
6. `NodeShapeSyntax` - all shape types
7. `sanitizeID` / `SanitizeID` - special chars, spaces, valid IDs
8. Sequence/State/ER/Class constructors and methods

### Stage 2: Diagram Package Tests

**Goal:** 100% coverage for `diagram` package

**Files to create:**
- `diagram/errors_test.go` - Sentinel error checking, unwrapping, FallbackFormatError
- `diagram/options_test.go` - ResolutionScale, ResolutionDPI, NewRenderOptions, defaults
- `diagram/types_test.go` - DiagramType, OutputFormat constants

**Priority tests:**
1. All sentinel errors are detectable via `errors.Is`
2. `FallbackFormatError.Unwrap()` / `FallbackFormat()`
3. `RenderOptions` defaults via `NewRenderOptions()`
4. `ResolutionScale` - all presets + default case
5. `ResolutionDPI` - all presets + default case
6. `DiagramType` and `OutputFormat` string values

### Stage 3: Negative Tests

**Goal:** Comprehensive error path coverage

**File:** `tests/negative/error_paths_test.go`

**Tests:**
- Invalid `DiagramType` for format combinations
- Empty diagrams passed to renderers
- Duplicate node IDs in AST
- Invalid node IDs (empty, special characters)
- Malformed edges (self-loops, missing nodes)
- `RenderToFile` with invalid paths
- `RenderBytes` with nil diagram (if possible)

### Stage 4: Integration Tests

**Goal:** All diagram types × all output formats

**File:** `tests/integration/format_combinations_test.go`

**Coverage matrix:**
```
Format    | Flow | Seq | State | ER  | Class | Pie | Quadrant | Gantt | Mindmap
----------|------|-----|-------|-----|-------|-----|----------|-------|----------
MMD       |  ✓   |  ✓  |   ✓   |  ✓  |   ✓   |  ✓  |    ✓     |   ✓   |    ✓
SVG       |  ✓   |  ✓  |   ✓   |  ✓  |   ✓   |  -  |    -     |   -   |    -
PNG       |  ✓   |  -  |   -   |  -  |   -   |  -  |    -     |   -   |    -
HTML      |  ✓   |  ✓  |   ✓   |  ✓  |   ✓   |  ✓  |    ✓     |   ✓   |    ✓
PDF       |  ✓   |  -  |   -   |  -  |   -   |  -  |    -     |   -   |    -
JSON      |  ✓   |  ✓  |   ✓   |  ✓  |   ✓   |  ✓  |    ✓     |   ✓   |    ✓
DOT       |  ✓   |  -  |   -   |  -  |   -   |  -  |    -     |   -   |    -
MD        |  ✓   |  ✓  |   ✓   |  ✓  |   ✓   |  ✓  |    ✓     |   ✓   |    ✓
```

### Stage 5: Property Tests & Golden Files

**Goal:** Output validity and roundtrip consistency

**Files:**
- `tests/property/roundtrip_test.go` - MMD encode/decode consistency
- `tests/property/svg_validity_test.go` - Generated SVG is valid XML
- `tests/property/golden_files_test.go` - Compare encoder output to golden files

**Tests:**
- SVG output parses as valid XML
- SVG has correct namespace and structure
- MMD output can be parsed back to equivalent AST
- JSON output decodes to same diagram structure
- All content types are correct

## Stage Execution Checklist

- [ ] Stage 1: AST package tests
- [ ] Stage 2: Diagram package tests
- [ ] Stage 3: Negative tests
- [ ] Stage 4: Integration tests
- [ ] Stage 5: Property tests
- [ ] Verify: `go test ./... -count=1` passes
- [ ] Verify: `golangci-lint run` passes
- [ ] Verify: Coverage > 80% overall

## Notes

- Package-level `*_test.go` files remain in place
- `tests/` directory provides cross-package integration testing
- Fuzzing tests (`*_fuzz_test.go`) deferred to future stage
- Test fixtures stored in `testdata/` with package+test_name organization