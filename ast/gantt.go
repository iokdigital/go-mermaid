package ast

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	diagram "github.com/iokdigital/go-mermaid"
)

// GanttDiagram represents a Mermaid gantt chart: sections of tasks with
// dates, durations, and after/until dependencies.
type GanttDiagram struct {
	title       string
	dateFormat  string
	axisFormat  string
	excludes    string // raw excludes line ("" = none)
	todayMarker string // raw todayMarker style string ("" = none)
	sections    []GanttSection
	tasks       []GanttTask // unsectioned tasks, rendered first

	excludesWeekends bool
	excludesDates    []string // raw date strings, parsed lazily with the diagram format
}

// GanttSection groups tasks under a named header.
type GanttSection struct {
	Name  string
	Tasks []GanttTask
}

// GanttTag is a Mermaid task marker.
type GanttTag string

const (
	TagDone      GanttTag = "done"
	TagCrit      GanttTag = "crit"
	TagActive    GanttTag = "active"
	TagMilestone GanttTag = "milestone"
	TagVert      GanttTag = "vert"
)

// GanttTask is one row of the chart. Start is a date in the diagram's
// DateFormat or "after <id>". End is a date, a duration ("30d"), or
// "until <id>". Tags are Mermaid task markers.
type GanttTask struct {
	Name  string
	ID    string
	Tags  []GanttTag
	Start string
	End   string
}

// NewGantt creates an empty gantt chart with the given title.
func NewGantt(title string) *GanttDiagram {
	return &GanttDiagram{title: title}
}

func (g *GanttDiagram) Type() diagram.DiagramType { return diagram.TypeGantt }
func (g *GanttDiagram) Title() string             { return g.title }

// DateFormat returns the Mermaid dateFormat string ("" = Mermaid default
// YYYY-MM-DD).
func (g *GanttDiagram) DateFormat() string { return g.dateFormat }

// AxisFormat returns the Mermaid axisFormat string ("" = automatic).
func (g *GanttDiagram) AxisFormat() string { return g.axisFormat }

// SetDateFormat sets the dateFormat line (e.g. "YYYY-MM-DD", "MM-DD",
// "YYYY-MM-DD HH:mm", "X"). Only formats in GanttDateLayouts are resolvable;
// others fail Resolve with a descriptive error.
func (g *GanttDiagram) SetDateFormat(f string) *GanttDiagram {
	g.dateFormat = f
	return g
}

// SetAxisFormat sets the axisFormat line (e.g. "%m/%d").
func (g *GanttDiagram) SetAxisFormat(f string) *GanttDiagram {
	g.axisFormat = f
	return g
}

// SetExcludes sets the excludes line: the "weekends" keyword and/or a
// comma-separated list of dates in the diagram's dateFormat (e.g.
// "weekends, 2014-01-10"). Excluded days consume no task duration (see
// Resolve). The raw line is preserved for MMD passthrough.
func (g *GanttDiagram) SetExcludes(s string) *GanttDiagram {
	g.excludes = strings.TrimSpace(s)
	g.excludesWeekends = false
	g.excludesDates = nil
	for _, part := range strings.Split(g.excludes, ",") {
		p := strings.TrimSpace(part)
		switch strings.ToLower(p) {
		case "":
			continue
		case "weekends":
			g.excludesWeekends = true
		default:
			g.excludesDates = append(g.excludesDates, p)
		}
	}
	return g
}

// Excludes returns the raw excludes line ("" = none).
func (g *GanttDiagram) Excludes() string { return g.excludes }

// ExcludesWeekends reports whether the excludes line includes "weekends".
func (g *GanttDiagram) ExcludesWeekends() bool { return g.excludesWeekends }

// ExcludesDates returns the custom excluded dates as raw strings in the
// diagram's dateFormat.
func (g *GanttDiagram) ExcludesDates() []string { return g.excludesDates }

// SetTodayMarker sets the todayMarker line (e.g.
// "stroke-width:5px,stroke:#0f0,opacity:0.5"). The SVG renderer draws a
// vertical line at time.Now() when set; the style attributes are preserved
// verbatim for MMD passthrough.
func (g *GanttDiagram) SetTodayMarker(m string) *GanttDiagram {
	g.todayMarker = m
	return g
}

// TodayMarker returns the raw todayMarker style string ("" = none).
func (g *GanttDiagram) TodayMarker() string { return g.todayMarker }

func (g *GanttDiagram) Sections() []GanttSection { return g.sections }
func (g *GanttDiagram) Tasks() []GanttTask       { return g.tasks }

// AddSection appends a section to the chart.
func (g *GanttDiagram) AddSection(s GanttSection) *GanttDiagram {
	g.sections = append(g.sections, s)
	return g
}

// AddTask appends an unsectioned task, rendered before all sections.
func (g *GanttDiagram) AddTask(t GanttTask) *GanttDiagram {
	g.tasks = append(g.tasks, t)
	return g
}

// NewGanttSection creates a section with the given name.
func NewGanttSection(name string) *GanttSection {
	return &GanttSection{Name: name}
}

// AddTask appends a task to the section.
func (s *GanttSection) AddTask(t GanttTask) *GanttSection {
	s.Tasks = append(s.Tasks, t)
	return s
}

// ResolvedGanttTask is a task with concrete start/end times.
type ResolvedGanttTask struct {
	Name      string
	ID        string
	Tags      []GanttTag
	Section   string // "" for unsectioned tasks
	Start     time.Time
	End       time.Time
	Milestone bool
}

var ganttDurationRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)(s|m|h|d|w)$`)

// ParseGanttDuration parses Mermaid durations ("30d", "2w", "12h").
func ParseGanttDuration(s string) (time.Duration, error) {
	m := ganttDurationRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("invalid duration %q (want e.g. 30d, 12h, 2w)", s)
	}
	n, _ := strconv.ParseFloat(m[1], 64)
	switch m[2] {
	case "s":
		return time.Duration(n * float64(time.Second)), nil
	case "m":
		return time.Duration(n * float64(time.Minute)), nil
	case "h":
		return time.Duration(n * float64(time.Hour)), nil
	case "d":
		return time.Duration(n * 24 * float64(time.Hour)), nil
	default: // "w"
		return time.Duration(n * 7 * 24 * float64(time.Hour)), nil
	}
}

// ganttDateTokens maps moment-style dateFormat tokens to Go layout tokens,
// longest match first. Letter runs that are not in this table make the
// format unsupported; non-letter characters pass through as literals.
var ganttDateTokens = []struct {
	tok, layout string
}{
	{"YYYY", "2006"},
	{"YY", "06"},
	{"MMM", "Jan"},
	{"MM", "01"},
	{"DD", "02"},
	{"D", "2"},
	{"HH", "15"},
	{"H", "3"},
	{"mm", "04"},
	{"ss", "05"},
}

// translateGanttDateFormat converts a Mermaid dateFormat to a Go layout.
// It returns hasYear=false for date-only formats (no YYYY/YY), which resolve
// in the current year.
func translateGanttDateFormat(format string) (layout string, hasYear bool, err error) {
	var b strings.Builder
	hasYear = false
	i := 0
	for i < len(format) {
		c := format[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			b.WriteByte(c)
			i++
			continue
		}
		j := i
		for j < len(format) && (format[j] >= 'A' && format[j] <= 'Z' || format[j] >= 'a' && format[j] <= 'z') {
			j++
		}
		run := format[i:j]
		switch run {
		case "X":
			return "", false, fmt.Errorf("unsupported dateFormat %q (X must be the whole format)", format)
		case "x":
			return "", false, fmt.Errorf("unsupported dateFormat %q (x must be the whole format)", format)
		}
		for len(run) > 0 {
			found := false
			for _, t := range ganttDateTokens {
				if strings.HasPrefix(run, t.tok) {
					b.WriteString(t.layout)
					run = run[len(t.tok):]
					if t.tok == "YYYY" || t.tok == "YY" {
						hasYear = true
					}
					found = true
					break
				}
			}
			if !found {
				return "", false, fmt.Errorf("unsupported dateFormat %q (want tokens YYYY YY MMM MM DD D HH H mm ss, or X/x epoch)", format)
			}
		}
		i = j
	}
	return b.String(), hasYear, nil
}

// ParseGanttDate parses a date in the given dateFormat. Supported formats:
// "" and "YYYY-MM-DD"; "YYYY-MM-DD HH:mm"; "MM-DD" (current year); "X"
// (unix epoch seconds); "x" (unix epoch milliseconds); and combinations of
// the tokens YYYY YY MMM MM DD D HH H mm ss (date-only formats resolve in
// the current year). Anything else is rejected.
func ParseGanttDate(s, format string) (time.Time, error) {
	s = strings.TrimSpace(s)
	switch format {
	case "", "YYYY-MM-DD":
		return time.Parse("2006-01-02", s)
	case "YYYY-MM-DD HH:mm":
		return time.Parse("2006-01-02 15:04", s)
	case "X":
		secs, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid epoch %q", s)
		}
		return time.Unix(secs, 0), nil
	case "x":
		millis, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid epoch %q", s)
		}
		return time.UnixMilli(millis), nil
	}
	layout, hasYear, err := translateGanttDateFormat(format)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, err
	}
	if !hasYear {
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
	}
	return t, nil
}

// addGanttDuration adds dur to start, skipping excluded calendar days.
// The start day always consumes (even if it is itself excluded); entering an
// excluded day consumes no duration. Day/week durations step by whole
// calendar days, with a fractional remainder consuming pro-rata from the
// start of the next non-excluded day; sub-day (s/m/h) durations flow
// continuously within non-excluded days. With no excluded predicate this is
// plain addition (semantics pinned vs Pele; see
// docs/plans/2026-10-28-gantt-excludes-markers.md).
func addGanttDuration(start time.Time, dur time.Duration, excluded func(time.Time) bool, dayUnit bool) time.Time {
	if excluded == nil {
		return start.Add(dur)
	}
	if !dayUnit {
		// Continuous (s/m/h) durations.
		cur := start
		remaining := dur
		for skipped := 0; remaining > 0; {
			next := ganttDayBoundary(cur).AddDate(0, 0, 1)
			if excluded(next) {
				cur = next
				skipped++
				if skipped > 36525 { // 100 years of consecutive excludes: give up
					return cur.Add(remaining)
				}
				continue
			}
			skipped = 0
			span := next.Sub(cur)
			take := span
			if remaining < take {
				take = remaining
			}
			cur = cur.Add(take)
			remaining -= take
		}
		return cur
	}
	// Day/week durations: whole days step by calendar day.
	whole := dur.Truncate(24 * time.Hour)
	frac := dur - whole
	offset := start.Sub(ganttDayBoundary(start))
	cur := start
	for skipped := 0; whole > 0; {
		next := ganttDayBoundary(cur).AddDate(0, 0, 1)
		if excluded(next) {
			cur = next
			skipped++
			if skipped > 36525 {
				return next.Add(offset).Add(whole).Add(frac)
			}
			continue
		}
		skipped = 0
		cur = next
		whole -= 24 * time.Hour
	}
	if frac > 0 {
		for skipped := 0; ; {
			next := ganttDayBoundary(cur).AddDate(0, 0, 1)
			if excluded(next) {
				cur = next
				skipped++
				if skipped > 36525 {
					return next.Add(frac)
				}
				continue
			}
			return next.Add(frac)
		}
	}
	return cur.Add(offset)
}

// ganttDayBoundary returns midnight of t's day.
func ganttDayBoundary(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Resolve computes concrete start/end times for every task, following
// after/until dependencies (forward and cross-section references allowed).
// Unknown references, duplicate IDs, unknown tags, and unparseable dates or
// durations return errors. Tasks whose end precedes their start resolve
// as-is; renderers clamp them to zero width.
func (g *GanttDiagram) Resolve() ([]ResolvedGanttTask, error) {
	type located struct {
		task    GanttTask
		section string
	}
	var all []located
	for _, t := range g.tasks {
		all = append(all, located{task: t})
	}
	for _, s := range g.sections {
		for _, t := range s.Tasks {
			all = append(all, located{task: t, section: s.Name})
		}
	}

	byID := make(map[string]int, len(all))
	for i, l := range all {
		if l.task.ID == "" {
			continue
		}
		if _, dup := byID[l.task.ID]; dup {
			return nil, fmt.Errorf("gantt: duplicate task id %q", l.task.ID)
		}
		byID[l.task.ID] = i
	}

	for _, l := range all {
		for _, tag := range l.task.Tags {
			switch tag {
			case TagDone, TagCrit, TagActive, TagMilestone, TagVert:
			default:
				return nil, fmt.Errorf("gantt: task %q has unknown tag %q", l.task.Name, tag)
			}
		}
		if strings.TrimSpace(l.task.Start) == "" {
			return nil, fmt.Errorf("gantt: task %q is missing a start", l.task.Name)
		}
		if strings.TrimSpace(l.task.End) == "" {
			return nil, fmt.Errorf("gantt: task %q is missing an end", l.task.Name)
		}
	}

	starts := make([]time.Time, len(all))
	ends := make([]time.Time, len(all))
	done := make([]bool, len(all))

	excluded, err := g.buildExcludes()
	if err != nil {
		return nil, err
	}

	// Iterative resolution handles forward references; each pass must settle
	// at least one task or the remainder is unresolvable (cycle/unknown ref).
	for settled := 0; settled < len(all); {
		progress := false
		for i, l := range all {
			if done[i] {
				continue
			}
			s, e, ok, err := resolveTask(l.task, starts, ends, done, byID, g.dateFormat, excluded)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			starts[i], ends[i], done[i] = s, e, true
			settled++
			progress = true
		}
		if !progress {
			return nil, fmt.Errorf("gantt: cannot resolve task %q (unknown after/until reference or dependency cycle)", all[unresolved(done)].task.Name)
		}
	}

	out := make([]ResolvedGanttTask, len(all))
	for i, l := range all {
		milestone := false
		for _, tag := range l.task.Tags {
			if tag == TagMilestone {
				milestone = true
			}
		}
		out[i] = ResolvedGanttTask{
			Name:      l.task.Name,
			ID:        l.task.ID,
			Tags:      l.task.Tags,
			Section:   l.section,
			Start:     starts[i],
			End:       ends[i],
			Milestone: milestone,
		}
	}
	return out, nil
}

func unresolved(done []bool) int {
	for i, d := range done {
		if !d {
			return i
		}
	}
	return 0
}

// buildExcludes validates the custom excluded dates against the diagram's
// dateFormat and returns the exclusion predicate (nil when nothing is
// excluded).
func (g *GanttDiagram) buildExcludes() (func(time.Time) bool, error) {
	if !g.excludesWeekends && len(g.excludesDates) == 0 {
		return nil, nil
	}
	var dates []time.Time
	for _, raw := range g.excludesDates {
		t, err := ParseGanttDate(raw, g.dateFormat)
		if err != nil {
			return nil, fmt.Errorf("gantt: invalid excludes date %q: %v", raw, err)
		}
		dates = append(dates, t)
	}
	return func(t time.Time) bool {
		if g.excludesWeekends {
			switch t.Weekday() {
			case time.Saturday, time.Sunday:
				return true
			}
		}
		for _, d := range dates {
			if t.Year() == d.Year() && t.Month() == d.Month() && t.Day() == d.Day() {
				return true
			}
		}
		return false
	}, nil
}

// resolveTask resolves one task. ok=false means a dependency is not settled
// yet (retry in a later pass).
func resolveTask(t GanttTask, starts, ends []time.Time, done []bool, byID map[string]int, format string, excluded func(time.Time) bool) (s, e time.Time, ok bool, err error) {
	startStr := strings.TrimSpace(t.Start)
	if rest, found := strings.CutPrefix(startStr, "after "); found {
		ref := strings.TrimSpace(rest)
		j, known := byID[ref]
		if !known {
			return time.Time{}, time.Time{}, false,
				fmt.Errorf("gantt: task %q starts after unknown task %q", t.Name, ref)
		}
		if !done[j] {
			return time.Time{}, time.Time{}, false, nil
		}
		// An after-start inherits the referenced task's end.
		s = ends[j]
	} else {
		var perr error
		s, perr = ParseGanttDate(startStr, format)
		if perr != nil {
			return time.Time{}, time.Time{}, false,
				fmt.Errorf("gantt: task %q has invalid start: %v", t.Name, perr)
		}
	}

	endStr := strings.TrimSpace(t.End)
	if rest, found := strings.CutPrefix(endStr, "until "); found {
		ref := strings.TrimSpace(rest)
		j, known := byID[ref]
		if !known {
			return time.Time{}, time.Time{}, false,
				fmt.Errorf("gantt: task %q ends until unknown task %q", t.Name, ref)
		}
		if !done[j] {
			return time.Time{}, time.Time{}, false, nil
		}
		// until ends at the referenced task's start (verified vs Pele).
		e = starts[j]
	} else if d, derr := ParseGanttDuration(endStr); derr == nil {
		unit := strings.TrimSpace(endStr)
		dayUnit := strings.HasSuffix(unit, "d") || strings.HasSuffix(unit, "w")
		e = addGanttDuration(s, d, excluded, dayUnit)
	} else {
		var perr error
		e, perr = ParseGanttDate(endStr, format)
		if perr != nil {
			return time.Time{}, time.Time{}, false,
				fmt.Errorf("gantt: task %q has invalid end: %v", t.Name, perr)
		}
	}
	return s, e, true, nil
}
