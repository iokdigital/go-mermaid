package svg

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Gantt chart geometry.
const (
	ganttGridW    = 640.0
	ganttRowH     = 32.0
	ganttHeaderH  = 26.0
	ganttBarH     = 18.0
	ganttAxisH    = 30.0
	ganttLabelMax = 260.0
	ganttLabelPad = 16.0
)

// encodeGantt renders a *ast.GanttDiagram to w as an SVG document.
// Tasks resolve via ast Resolve, so after/until dependencies (including
// forward and cross-section references) position correctly. Bars are drawn
// on a linear time scale with adaptive axis ticks. Task markers map to flat
// fills: done gray, crit red, active blue, milestone diamond, vert thin bar.
// Unsupported dateFormats fail Resolve and return an error here.
func encodeGantt(w io.Writer, d *ast.GanttDiagram, opts diagram.RenderOptions) error {
	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}

	tasks, err := d.Resolve()
	if err != nil {
		return fmt.Errorf("svg: %w", err)
	}

	var sb strings.Builder
	if len(tasks) == 0 {
		writeSVGHeader(&sb, padding*2+200, padding*2+60, opts, d.Title())
		fmt.Fprintf(&sb, `  <g id="gantt-empty"><text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#64748b" font-style="italic">(no tasks)</text></g>`+"\n",
			padding+100, padding+30, defaultFontSize, defaultFontFamily)
		sb.WriteString("</svg>\n")
		_, err := io.WriteString(w, sb.String())
		return err
	}

	// Time range with a small pad; degenerate spans get +/-12h.
	minT, maxT := tasks[0].Start, tasks[0].End
	for _, t := range tasks[1:] {
		if t.Start.Before(minT) {
			minT = t.Start
		}
		if t.End.After(maxT) {
			maxT = t.End
		}
	}
	if !maxT.After(minT) {
		minT, maxT = minT.Add(-12*time.Hour), maxT.Add(12*time.Hour)
	} else {
		pad := maxT.Sub(minT) / 20
		minT, maxT = minT.Add(-pad), maxT.Add(pad)
	}

	// Row layout: ungrouped tasks first, then one header + rows per section.
	type row struct {
		sectionHeader string
		task          *ast.ResolvedGanttTask
	}
	var rows []row
	seenSection := make(map[string]bool)
	for i := range tasks {
		t := &tasks[i]
		if t.Section == "" {
			rows = append(rows, row{task: t})
		}
	}
	orderedSections := []string{}
	for i := range tasks {
		if tasks[i].Section != "" && !seenSection[tasks[i].Section] {
			seenSection[tasks[i].Section] = true
			orderedSections = append(orderedSections, tasks[i].Section)
		}
	}
	for _, s := range orderedSections {
		rows = append(rows, row{sectionHeader: s})
		for i := range tasks {
			if tasks[i].Section == s {
				t := &tasks[i]
				rows = append(rows, row{task: t})
			}
		}
	}

	labelW := 0.0
	for _, r := range rows {
		name := r.sectionHeader
		if r.task != nil {
			name = r.task.Name
		}
		if w := float64(utf8.RuneCountInString(name))*mindCharW + ganttLabelPad; w > labelW {
			labelW = w
		}
	}
	labelW = math.Min(ganttLabelMax, math.Max(90, labelW))

	gridX := padding + labelW
	gridY := padding + ganttAxisH
	natW := gridX + ganttGridW + padding
	natH := gridY + float64(len(rows))*ganttRowH + padding

	writeSVGHeader(&sb, natW, natH, opts, d.Title())

	scale := func(t time.Time) float64 {
		return gridX + ganttGridW*t.Sub(minT).Seconds()/maxT.Sub(minT).Seconds()
	}

	// Axis ticks.
	ticks := ganttTicks(minT, maxT)
	sb.WriteString(`  <g id="gantt-axis">` + "\n")
	for _, tick := range ticks {
		x := scale(tick)
		fmt.Fprintf(&sb, `    <path d="M%.1f,%.1f L%.1f,%.1f" stroke="#e2e8f0" stroke-width="1"/>`+"\n",
			x, gridY, x, natH-padding)
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#64748b">%s</text>`+"\n",
			x, padding+ganttAxisH/2, defaultFontFamily, xmlEscape(ganttTickLabel(tick, minT, maxT, d.AxisFormat())))
	}
	sb.WriteString("  </g>\n")

	// Rows.
	sb.WriteString(`  <g id="gantt-rows">` + "\n")
	y := gridY
	for _, r := range rows {
		if r.task == nil {
			fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" dominant-baseline="central" font-size="12" font-family="%s" fill="#64748b" font-style="italic">%s</text>`+"\n",
				padding, y+ganttRowH/2, defaultFontFamily, xmlEscape(r.sectionHeader))
			y += ganttRowH
			continue
		}
		fmt.Fprintf(&sb, `    <text x="%.1f" y="%.1f" text-anchor="end" dominant-baseline="central" font-size="12" font-family="%s" fill="#1e293b">%s</text>`+"\n",
			gridX-8, y+ganttRowH/2, defaultFontFamily, xmlEscape(r.task.Name))
		x1 := scale(r.task.Start)
		x2 := scale(r.task.End)
		if x2 < x1 {
			x2 = x1
		}
		sb.WriteString("    ")
		sb.WriteString(ganttBarSVG(x1, x2, y+ganttRowH/2, r.task))
		sb.WriteString("\n")
		y += ganttRowH
	}
	sb.WriteString("  </g>\n</svg>\n")

	_, err = io.WriteString(w, sb.String())
	return err
}

// ganttBarSVG returns the bar/marker shape for a resolved task.
func ganttBarSVG(x1, x2, cy float64, t *ast.ResolvedGanttTask) string {
	var milestone, vert, crit, done, active bool
	for _, tag := range t.Tags {
		switch tag {
		case ast.TagMilestone:
			milestone = true
		case ast.TagVert:
			vert = true
		case ast.TagCrit:
			crit = true
		case ast.TagDone:
			done = true
		case ast.TagActive:
			active = true
		}
	}
	fill := "#4e79a7"
	stroke := "#1d4ed8"
	switch {
	case crit:
		fill, stroke = "#e15759", "#b91c1c"
	case done:
		fill, stroke = "#9ca3af", "#4b5563"
	case active:
		fill, stroke = "#2563eb", "#1e3a8a"
	}
	if milestone {
		s := 9.0
		mx := x1
		return fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="#f28e2b" stroke="#c2410c" stroke-width="1.5"/>`,
			mx, cy-s, mx+s, cy, mx, cy+s, mx-s, cy)
	}
	if vert {
		return fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="3" height="%.1f" fill="#b07aa1"/>`,
			x1-1.5, cy-ganttBarH/2, ganttBarH)
	}
	return fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3" fill="%s" stroke="%s" stroke-width="1.5"/>`,
		x1, cy-ganttBarH/2, math.Max(x2-x1, 2), ganttBarH, fill, stroke)
}

// ganttTicks returns adaptive axis ticks: daily (<=14d), weekly (<=13w),
// monthly (<=400d), else quarterly.
func ganttTicks(minT, maxT time.Time) []time.Time {
	span := maxT.Sub(minT)
	var step time.Duration
	var align func(time.Time) time.Time
	switch {
	case span <= 14*24*time.Hour:
		step = 24 * time.Hour
		align = func(t time.Time) time.Time {
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		}
	case span <= 13*7*24*time.Hour:
		step = 7 * 24 * time.Hour
		align = func(t time.Time) time.Time {
			y, m, d := t.Date()
			base := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
			wd := int(base.Weekday())
			shift := (wd + 6) % 7 // back to Monday
			return base.AddDate(0, 0, -shift)
		}
	case span <= 400*24*time.Hour:
		step = 0 // monthly stepping below
		align = func(t time.Time) time.Time {
			y, m, _ := t.Date()
			return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
		}
	default:
		step = 0
		align = func(t time.Time) time.Time {
			y, m, _ := t.Date()
			q := (int(m)-1)/3*3 + 1
			return time.Date(y, time.Month(q), 1, 0, 0, 0, 0, t.Location())
		}
	}
	var ticks []time.Time
	if step != 0 {
		for t := align(minT); !t.After(maxT); t = t.Add(step) {
			ticks = append(ticks, t)
		}
		return ticks
	}
	// Monthly / quarterly: step calendar months (1 or 3).
	months := 1
	if span > 400*24*time.Hour {
		months = 3
	}
	for t := align(minT); !t.After(maxT); t = t.AddDate(0, months, 0) {
		ticks = append(ticks, t)
		// Safety cap for pathological ranges.
		if len(ticks) > 60 {
			break
		}
	}
	return ticks
}

// ganttTickLabel formats a tick via axisFormat (%-tokens) or an adaptive
// default. Unknown tokens pass through literally.
func ganttTickLabel(t, minT, maxT time.Time, axisFormat string) string {
	if axisFormat != "" {
		return ganttStrftime(t, axisFormat)
	}
	span := maxT.Sub(minT)
	switch {
	case span <= 93*24*time.Hour:
		return t.Format("01-02")
	case span <= 400*24*time.Hour:
		return t.Format("2006-01")
	default:
		return t.Format("2006")
	}
}

// ganttStrftime maps common Mermaid axisFormat tokens to Go layouts.
func ganttStrftime(t time.Time, format string) string {
	out := strings.ReplaceAll(format, "%%", "\x00")
	out = strings.ReplaceAll(out, "%s", "\x01")
	for _, r := range []struct{ token, layout string }{
		{"%Y", "2006"}, {"%y", "06"},
		{"%B", "January"}, {"%b", "Jan"},
		{"%m", "01"}, {"%d", "02"}, {"%e", "2"},
		{"%A", "Monday"}, {"%a", "Mon"},
		{"%H", "15"}, {"%M", "04"}, {"%S", "05"},
	} {
		out = strings.ReplaceAll(out, r.token, r.layout)
	}
	formatted := t.Format(out)
	formatted = strings.ReplaceAll(formatted, "\x01", fmt.Sprintf("%d", t.Unix()))
	return strings.ReplaceAll(formatted, "\x00", "%")
}
