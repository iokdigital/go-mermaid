# Gantt Excludes / TodayMarker / Extended DateFormats Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support `excludes` (weekends + custom dates), `todayMarker`, and a wider `dateFormat` token set in gantt charts.

**Architecture:** Extend the date engine in `ast/gantt.go` (excluded-day calendar used by duration math + `Resolve`), pass `excludes`/`todayMarker` lines through the mmd encoder verbatim, and render excluded shading + a today line in the SVG renderer.

**Tech Stack:** Existing `ast`/`mmd`/`svg` packages, no new dependencies.

---

## Scope (tokens)

DateFormat additions: `DD`, `MMM`, `YY`, `HH`, `mm`, `ss`, `x` (millis epoch). Each maps to a Go layout in one table; anything else still errors with the supported list. (Full moment.js parity is explicitly out of scope.)

`excludes`: `weekends` keyword + `YYYY-MM-DD, ...` date list (parsed with the diagram's DateFormat). `todayMarker`: `todayMarker <style-attrs>` passthrough string (e.g. `stroke-width:5px,stroke:#0f0,opacity:0.5`).

## Semantics (verified against Mermaid/Pele during planning)

- Durations skip excluded days: `5d` starting Friday with `excludes weekends` ends the following Friday (probe in Task 1 pins exact behavior).
- `todayMarker` draws a vertical line at today; our SVG uses `time.Now()` (documented; tests assert presence, not position).

---

### Task 1: Probe and pin excludes semantics

**Files:**
- Create (throwaway): `/tmp/gprobe.mjs` — or reuse `scripts/pele-diff/render.mjs` with inline sources via temp files.

**Step 1: Probe Pele**

Render: `gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nsection S\n  T :t, 2014-01-03, 5d` (2014-01-03 is a Friday) and extract the bar geometry/text. Record: which calendar date does the bar end on?

Also probe: `excludes 2014-01-06` (custom date), `excludes weekends, 2014-01-10` (combined), and a task starting on an excluded day.

**Step 2: Record results**

Write the observed end dates into this plan file (edit the Semantics section above) before implementing. If Pele errors on any form, that form stays out of scope with a comment.

**Step 3: Commit the probe notes**

```bash
git add docs/plans/2026-10-28-gantt-excludes-markers.md
git commit -m "docs(gantt): pin excludes semantics from Pele probes"
```

### Task 2: Date engine — excludes-aware durations + new tokens

**Files:**
- Modify: `ast/gantt.go`, `ast/gantt_test.go`

**Step 1: Write the failing tests**

```go
func TestGanttExcludesWeekends(t *testing.T) {
	g := ast.NewGantt("")
	g.SetExcludes("weekends") // new setter; comma-separated also accepted
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "5d"}) // Friday
	tasks, err := g.Resolve()
	// ... assert End == <probed date>
}

func TestGanttDateFormatTokens(t *testing.T) {
	// table: {"DD-MM-YYYY"...} — one row per newly supported token set
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./ast/ -run TestGantt -v`
Expected: FAIL with "undefined: SetExcludes"

**Step 3: Write minimal implementation**

- `GanttDiagram.excludesWeekends bool`, `excludesDates []string` (raw, parsed lazily with the diagram format); `SetExcludes(s string)` splits on comma, recognizes `weekends`.
- Duration resolution: iterate calendar days from start, counting only non-excluded days, until duration consumed. Fractional durations: consume pro-rata on the last day (document).
- Token table extension in `ParseGanttDate` + error message update.

**Step 4: Run tests to verify they pass**

Run: `go test ./ast/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add ast/gantt.go ast/gantt_test.go
git commit -m "feat(gantt): excludes-aware durations, extended date tokens"
```

### Task 3: MMD passthrough

**Files:**
- Modify: `mmd/encoder.go`, `mmd/encoder_test.go`, `testdata/golden/mmd/TestGanttExcludes.mmd` (new via `-update`)

**Step 1: Write the failing test**

```go
func TestGanttExcludes(t *testing.T) {
	g := ast.NewGantt("P")
	g.SetDateFormat("YYYY-MM-DD")
	g.SetExcludes("weekends, 2014-01-10")
	g.SetTodayMarker("stroke-width:5px,stroke:#0f0,opacity:0.5")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "5d"})
	assertGolden(t, encode(t, g))
}
```

Emission order (match Mermaid convention): `dateFormat`, `axisFormat`, `excludes ...`, `todayMarker ...`, then sections/tasks.

**Step 2–5:** standard (`-update` to create golden, verify golden content by eye, run Pele oracle — `tests/pele` golden glob picks it up automatically — then commit `feat(gantt): excludes/todayMarker mmd`).

### Task 4: SVG — excluded shading + today line

**Files:**
- Modify: `svg/gantt_renderer.go`, `svg/gantt_renderer_test.go`

**Step 1: Write the failing tests**

```go
func TestGantt_ExcludesShaded(t *testing.T) {
	// weekend columns render with a distinct fill; assert the shade rect count > 0
}
func TestGantt_TodayMarker(t *testing.T) {
	// today line present when TodayMarker set; absent otherwise
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./svg/ -run TestGantt -v`
Expected: FAIL (no shade/marker elements)

**Step 3: Write minimal implementation**

- Excluded ranges: iterate day-by-day across [minT, maxT], merge consecutive excluded days, draw light-gray rects behind bars (before the rows group).
- Today line: if `TodayMarker != ""`, draw a vertical line at `scale(time.Now())` spanning the grid (style: parse `stroke:#xxx` for color if present, else `#ef4444`; keep the full attr string as a `<line>` style comment — simple: fixed red line, document that full CSS styling is browser-only).

**Step 4: Run tests to verify they pass**

Run: `go test ./svg/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add svg/gantt_renderer.go svg/gantt_renderer_test.go
git commit -m "feat(gantt): excluded shading and today line"
```

### Task 5: Docs + matrix

Update `docs/user-guide.md` (gantt section: excludes semantics, todayMarker, supported token table) and `tests/pele/pele_diff_test.go` edge case (excludes + todayMarker diagram must parse). Run full suite, commit `docs(gantt): excludes and markers`.

---

## Acceptance criteria

- [ ] `go test ./...` green, `go vet` clean, new code gofmt-clean
- [ ] Pele oracle passes including the new excludes golden
- [ ] Probe results recorded in this file match implementation behavior
- [ ] Council review → PR to develop
