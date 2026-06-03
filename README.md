# go-mermaid

Pure Go library for generating and rendering [Mermaid](https://mermaid.js.org/) diagrams.
No Node.js, no Chromium, no external processes.

## Output formats

| Format | Extension | Notes |
|--------|-----------|-------|
| Mermaid source | `.mmd` | All diagram types |
| SVG | `.svg` | Flowchart (Phase 1); others Phase 2 |
| PNG | `.png` | Via pure-Go SVG rasterizer |
| HTML | `.html` | All types via mermaid.js CDN |
| PDF | `.pdf` | Flowchart via go-pdf/fpdf |
| JSON | `.json` | Diagram AST |
| Graphviz | `.dot` | Flowchart |
| Markdown | `.md` | Fenced mermaid code block |

## Diagram types

Phase 1: `flowchart`, `sequenceDiagram`, `stateDiagram-v2`, `erDiagram`, `classDiagram`

Phase 2: `pie`, `quadrantChart`, `gantt`, `mindmap`

## Requirements

Go 1.20 or later. No CGo. No external binaries.

## Status

Under active development. See [FRD-017-3](https://github.com/iokdigital/alteryx2talend/blob/main/docs/requirements/FRD017-3-Pure-Go-SVG-Renderer.md) for the full specification.

## Installation

```sh
go get github.com/iokdigital/go-mermaid
```
