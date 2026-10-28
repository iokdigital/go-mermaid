# Gantt DOT Dependency Edges Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Gantt DOT output draws `after`/`until` dependency edges instead of only listing task nodes.

**Architecture:** In `dot/encodeGantt`, rebuild the id→node map from the AST (same traversal order `Resolve` uses) and emit `ref -> task` edges for `after X`/`until X` refs. Self-contained in the dot package: parse the two prefixes locally rather than adding new exported ast API.

**Tech Stack:** Existing `dot` encoder, no new dependencies.

---

### Task 1: Dependency edges

**Files:**
- Modify: `dot/encoder.go` (`encodeGantt`)
- Test: `dot/encoder_test.go` (extend `TestDOTGanttBasic`)

**Step 1: Write the failing test**

Extend `TestDOTGanttBasic`'s fixture (or add `TestDOTGanttDependencies`): task B with `Start: "after a"` must produce an edge from A's node to B's node:

```go
if !strings.Contains(got, "task_0 -> task_1") {
	t.Errorf("expected dependency edge, got:\n%s", got)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./dot/ -run TestDOTGantt -v`
Expected: FAIL with "expected dependency edge"

**Step 3: Write minimal implementation**

```go
// in encodeGantt, after the node loop:
idOf := map[string]int{} // task ID -> node index, same traversal as Resolve
i := 0
addEdgeRefs := func(t ast.GanttTask, idx int) { ... }
```

Concrete approach: walk `g.Tasks()` then each section's `Tasks` in order (exactly `Resolve`'s order), recording `id→index` for non-empty IDs; second walk emits `task_<refIdx> -> task_<idx>` when `Start` has prefix `"after "` or `End` has prefix `"until "` (trimmed ref must exist in the map — it always does, since `Resolve()` already validated; skip defensively if not). Emit edges after all nodes.

**Step 4: Run test to verify it passes**

Run: `go test ./dot/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add dot/encoder.go dot/encoder_test.go
git commit -m "feat(dot): gantt dependency edges"
```

### Task 2: Docs

Update the `encodeGantt` doc comment (drop "no DOT equivalent" for edges; geometry still has none). Run `go test ./...`, commit `docs(dot): gantt edges`.

---

## Acceptance criteria

- [ ] `go test ./...` green, `go vet` clean, new code gofmt-clean
- [ ] `after` and `until` refs both produce edges; unknown refs impossible (Resolve validates first)
- [ ] Council review → PR to develop
