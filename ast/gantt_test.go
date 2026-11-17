package ast_test

import (
	"testing"
	"time"

	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewGantt(t *testing.T) {
	g := ast.NewGantt("Plan")
	if g.Title() != "Plan" {
		t.Errorf("expected title Plan, got %q", g.Title())
	}
	if g.Type() != "gantt" {
		t.Errorf("expected type gantt, got %q", g.Type())
	}
	if len(g.Sections()) != 0 || len(g.Tasks()) != 0 {
		t.Error("expected empty diagram")
	}
}

func TestGanttResolveDates(t *testing.T) {
	g := ast.NewGantt("")
	s := ast.NewGanttSection("Build")
	s.AddTask(ast.GanttTask{Name: "A", ID: "a", Start: "2014-01-01", End: "30d"})
	s.AddTask(ast.GanttTask{Name: "B", Start: "after a", End: "2014-02-15"})
	g.AddSection(*s)

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-31" {
		t.Errorf("expected A end 2014-01-31, got %s", got)
	}
	if got := tasks[1].Start.Format("2006-01-02"); got != "2014-01-31" {
		t.Errorf("expected B start 2014-01-31, got %s", got)
	}
}

func TestGanttResolveMMDDUsesCurrentYear(t *testing.T) {
	g := ast.NewGantt("")
	g.SetDateFormat("MM-DD")
	g.AddTask(ast.GanttTask{Name: "A", Start: "01-15", End: "02-15"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	now := time.Now()
	if tasks[0].Start.Year() != now.Year() || tasks[0].End.Year() != now.Year() {
		t.Errorf("expected current-year resolution, got %v to %v",
			tasks[0].Start.Format("2006-01-02"), tasks[0].End.Format("2006-01-02"))
	}
	if got := tasks[0].Start.Format("01-02"); got != "01-15" {
		t.Errorf("expected 01-15, got %s", got)
	}
}

func TestGanttExcludesWeekends(t *testing.T) {
	// Pinned vs Pele: 5d from Friday 2014-01-03 with excludes weekends ends
	// the following Friday 2014-01-10. The start day consumes even when a
	// later step is skipped; entered excluded days consume nothing.
	g := ast.NewGantt("")
	g.SetExcludes("weekends")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "5d"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-10" {
		t.Errorf("expected end 2014-01-10, got %s", got)
	}
	if got := tasks[0].Start.Format("2006-01-02"); got != "2014-01-03" {
		t.Errorf("expected start unchanged 2014-01-03, got %s", got)
	}
}

func TestGanttExcludesCustomDate(t *testing.T) {
	// Pinned vs Pele: excludes 2014-01-06 with the task starting on that day;
	// the start day still consumes, end 2014-01-11.
	g := ast.NewGantt("")
	g.SetExcludes("2014-01-06")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-06", End: "5d"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-11" {
		t.Errorf("expected end 2014-01-11, got %s", got)
	}
}

func TestGanttExcludesCombined(t *testing.T) {
	// Pinned vs Pele: weekends + 2014-01-10 from Fri 2014-01-03, 5d
	// skips 01-04/05 and 01-10/11/12; end 2014-01-13.
	g := ast.NewGantt("")
	g.SetExcludes("weekends, 2014-01-10")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "5d"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-13" {
		t.Errorf("expected end 2014-01-13, got %s", got)
	}
}

func TestGanttExcludesFractional(t *testing.T) {
	// Pinned vs Pele: fractional remainder consumes pro-rata from the start
	// of the next non-excluded day: 2.5d from Fri 2014-01-03 with weekends
	// excluded ends on 2014-01-08 (Pele's axis ends at that day; our engine
	// returns 2014-01-08 12:00, half-way through the final day).
	g := ast.NewGantt("")
	g.SetExcludes("weekends")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "2.5d"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-08" {
		t.Errorf("expected end 2014-01-08, got %s", got)
	}
}

func TestGanttExcludesNoExcludesUnchanged(t *testing.T) {
	// Without excludes, durations are plain additions (existing behavior).
	g := ast.NewGantt("")
	g.AddTask(ast.GanttTask{Name: "A", Start: "2014-01-03", End: "5d"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := tasks[0].End.Format("2006-01-02"); got != "2014-01-08" {
		t.Errorf("expected end 2014-01-08, got %s", got)
	}
}

func TestGanttDateFormatTokens(t *testing.T) {
	cases := []struct {
		format string
		in     string
		want   string // formatted as "2006-01-02 15:04:05"
	}{
		{"DD-MM-YYYY", "17-11-2026", "2026-11-17 00:00:00"},
		{"DD/MM/YYYY", "17/11/2026", "2026-11-17 00:00:00"},
		{"MMM D, YYYY", "Nov 17, 2026", "2026-11-17 00:00:00"},
		{"YY-MM-DD", "26-11-17", "2026-11-17 00:00:00"},
		{"YYYY-MM-DD HH", "2026-11-17 09", "2026-11-17 09:00:00"},
		{"YYYY-MM-DD HH:mm:ss", "2026-11-17 09:30:45", "2026-11-17 09:30:45"},
		{"x", "1794907800000", "2026-11-17 09:30:00"},
	}
	for _, tc := range cases {
		t.Run(tc.format, func(t *testing.T) {
			got, err := ast.ParseGanttDate(tc.in, tc.format)
			if err != nil {
				t.Fatalf("ParseGanttDate(%q, %q): %v", tc.in, tc.format, err)
			}
			if got.Format("2006-01-02 15:04:05") != tc.want {
				t.Errorf("got %s, want %s", got.Format("2006-01-02 15:04:05"), tc.want)
			}
		})
	}
}

func TestGanttResolveUntil(t *testing.T) {
	g := ast.NewGantt("")
	g.AddTask(ast.GanttTask{Name: "T2", ID: "t2", Start: "2014-01-10", End: "2014-01-15"})
	g.AddTask(ast.GanttTask{Name: "T1", ID: "t1", Start: "2014-01-01", End: "until t2"})

	tasks, err := g.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// until ends at the referenced task's start (verified vs Pele).
	if got := tasks[1].End.Format("2006-01-02"); got != "2014-01-10" {
		t.Errorf("expected T1 end 2014-01-10, got %s", got)
	}
}

func TestGanttResolveErrors(t *testing.T) {
	cases := []struct {
		name  string
		tasks []ast.GanttTask
	}{
		{"unknown after", []ast.GanttTask{{Name: "A", Start: "after nope", End: "5d"}}},
		{"unknown until", []ast.GanttTask{{Name: "A", Start: "2014-01-01", End: "until nope"}}},
		{"bad date", []ast.GanttTask{{Name: "A", Start: "not-a-date", End: "5d"}}},
		{"bad tag", []ast.GanttTask{{Name: "A", Tags: []ast.GanttTag{"bogus"}, Start: "2014-01-01", End: "5d"}}},
		{"missing end", []ast.GanttTask{{Name: "A", Start: "2014-01-01"}}},
		{"cycle", []ast.GanttTask{
			{Name: "A", ID: "a", Start: "after b", End: "5d"},
			{Name: "B", ID: "b", Start: "after a", End: "5d"},
		}},
		{"duplicate id", []ast.GanttTask{
			{Name: "A", ID: "x", Start: "2014-01-01", End: "5d"},
			{Name: "B", ID: "x", Start: "2014-01-01", End: "5d"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := ast.NewGantt("")
			for _, task := range tc.tasks {
				g.AddTask(task)
			}
			if _, err := g.Resolve(); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestParseGanttDuration(t *testing.T) {
	cases := []struct {
		in    string
		hours float64
	}{
		{"30d", 720},
		{"2w", 336},
		{"12h", 12},
		{"90m", 1.5},
		{"30s", 1.0 / 120},
	}
	for _, tc := range cases {
		d, err := ast.ParseGanttDuration(tc.in)
		if err != nil {
			t.Errorf("ParseGanttDuration(%q): %v", tc.in, err)
			continue
		}
		if d.Hours() != tc.hours {
			t.Errorf("ParseGanttDuration(%q) = %v, want %v hours", tc.in, d, tc.hours)
		}
	}
	for _, bad := range []string{"", "5", "d", "1M", "1y"} {
		if _, err := ast.ParseGanttDuration(bad); err == nil {
			t.Errorf("expected error for duration %q", bad)
		}
	}
}
