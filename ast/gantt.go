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
	title      string
	dateFormat string
	axisFormat string
	sections   []GanttSection
	tasks      []GanttTask // unsectioned tasks, rendered first
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

// ParseGanttDate parses a date in the given dateFormat. Supported formats:
// "" and "YYYY-MM-DD", "YYYY-MM-DD HH:mm", "MM-DD" (current year), "X"
// (unix epoch seconds). Anything else is rejected.
func ParseGanttDate(s, format string) (time.Time, error) {
	s = strings.TrimSpace(s)
	switch format {
	case "", "YYYY-MM-DD":
		return time.Parse("2006-01-02", s)
	case "YYYY-MM-DD HH:mm":
		return time.Parse("2006-01-02 15:04", s)
	case "MM-DD":
		t, err := time.Parse("01-02", s)
		if err != nil {
			return time.Time{}, err
		}
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	case "X":
		secs, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid epoch %q", s)
		}
		return time.Unix(secs, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported dateFormat %q (want YYYY-MM-DD, YYYY-MM-DD HH:mm, MM-DD, or X)", format)
	}
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

	// Iterative resolution handles forward references; each pass must settle
	// at least one task or the remainder is unresolvable (cycle/unknown ref).
	for settled := 0; settled < len(all); {
		progress := false
		for i, l := range all {
			if done[i] {
				continue
			}
			s, e, ok, err := resolveTask(l.task, starts, ends, done, byID, g.dateFormat)
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

// resolveTask resolves one task. ok=false means a dependency is not settled
// yet (retry in a later pass).
func resolveTask(t GanttTask, starts, ends []time.Time, done []bool, byID map[string]int, format string) (s, e time.Time, ok bool, err error) {
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
		e = s.Add(d)
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
