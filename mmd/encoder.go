// Package mmd encodes diagram AST types to Mermaid (.mmd) source text.
// Sub-package names (mmd, json, dot, html, svg, png, pdf) are implementation
// detail; callers should use the root Renderer interface rather than importing
// sub-packages directly.
package mmd

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Encode writes the Mermaid source for d to w.
// Returns diagram.ErrInvalidFormat if d's type is not supported.
func Encode(w io.Writer, d diagram.Diagram) error {
	switch v := d.(type) {
	case *ast.FlowchartDiagram:
		return encodeFlowchart(w, v)
	case *ast.SequenceDiagram:
		return encodeSequence(w, v)
	case *ast.StateDiagram:
		return encodeState(w, v)
	case *ast.ERDiagram:
		return encodeER(w, v)
	case *ast.ClassDiagram:
		return encodeClass(w, v)
	case *ast.PieDiagram:
		return encodePie(w, v)
	case *ast.QuadrantDiagram:
		return encodeQuadrant(w, v)
	case *ast.MindmapDiagram:
		return encodeMindmap(w, v)
	case *ast.GanttDiagram:
		return encodeGantt(w, v)
	default:
		return fmt.Errorf("mmd: %w: %T", diagram.ErrInvalidFormat, d)
	}
}

// writeTitle emits the YAML frontmatter title block when title is non-empty.
func writeTitle(w io.Writer, title string) {
	if title == "" {
		return
	}
	fmt.Fprintf(w, "---\ntitle: %s\n---\n", title)
}

// truncateLabel truncates a label to 37 chars + ellipsis if longer than 40.
func truncateLabel(s string) string {
	if len([]rune(s)) > 40 {
		runes := []rune(s)
		return string(runes[:37]) + "…"
	}
	return s
}

// escapeFlowText escapes characters that would terminate Mermaid flowchart
// string contexts. Verified against the Pele/Mermaid 12.1.0 grammar:
//   - '"' ends quoted labels (node ["..."], subgraph ["..."]) and breaks
//     edge labels (|...| and -- ... ---), so it becomes #quot; (renders as ").
//   - '|' ends pipe edge labels (|...|), so it becomes #124; (renders as |).
//
// <, > and & need no escaping inside these contexts.
func escapeFlowText(s string) string {
	s = strings.ReplaceAll(s, `"`, "#quot;")
	return strings.ReplaceAll(s, "|", "#124;")
}

// mirrorCardinality converts a left-side Cardinality to its right-side Mermaid equivalent.
// In Mermaid ER, the cardinality symbols mirror across the connector:
//
//	left  "}o"  →  right "o{"
//	left  "}|"  →  right "|{"
//	left  "|o"  →  right "o|"
//	left  "||"  →  right "||"  (symmetric)
func mirrorCardinality(c ast.Cardinality) string {
	switch c {
	case ast.CardZeroMany:
		return "o{"
	case ast.CardOneMany:
		return "|{"
	case ast.CardZeroOne:
		return "o|"
	default: // CardExactOne = "||"
		return string(c)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Flowchart
// ────────────────────────────────────────────────────────────────────────────

func encodeFlowchart(w io.Writer, f *ast.FlowchartDiagram) error {
	writeTitle(w, f.Title())

	dir := f.Direction()
	if dir == "" {
		dir = ast.DirectionTB
	}
	fmt.Fprintf(w, "flowchart %s\n", dir)

	// Build a set of node IDs that belong to a subgraph for membership tracking.
	subgraphMember := make(map[string]bool)
	for _, sg := range f.Subgraphs() {
		for _, id := range sg.Nodes {
			subgraphMember[id] = true
		}
	}

	// Emit standalone nodes first (not inside any subgraph).
	for _, n := range f.Nodes() {
		if !subgraphMember[n.ID] {
			writeFlowNode(w, n)
		}
	}

	// Emit subgraphs with their member nodes.
	for _, sg := range f.Subgraphs() {
		sgID := ast.SanitizeID(sg.ID)
		if sg.Label != "" {
			fmt.Fprintf(w, "  subgraph %s [\"%s\"]\n", sgID, escapeFlowText(sg.Label))
		} else {
			fmt.Fprintf(w, "  subgraph %s\n", sgID)
		}
		// Emit node definitions inside the subgraph.
		nodeByID := make(map[string]*ast.FlowNode, len(f.Nodes()))
		for _, n := range f.Nodes() {
			nodeByID[n.ID] = n
		}
		for _, nid := range sg.Nodes {
			if n, ok := nodeByID[nid]; ok {
				writeFlowNode(w, n)
			}
		}
		fmt.Fprintln(w, "  end")
	}

	// Emit edges.
	for _, e := range f.Edges() {
		writeFlowEdge(w, e)
	}

	return nil
}

func writeFlowNode(w io.Writer, n *ast.FlowNode) {
	id := ast.SanitizeID(n.ID)
	raw := truncateLabel(n.Label)
	if raw == "" {
		raw = n.ID
	}
	// Quote decision uses the raw label: escaped text (#quot;/#124;) carries
	// no quoting triggers, but the raw specials still require quoting.
	quoted := needsQuote(raw)
	label := escapeFlowText(raw)
	open, close := ast.NodeShapeSyntax(n.Shape)
	// Quote labels that contain special Mermaid characters.
	if quoted {
		fmt.Fprintf(w, "  %s%s\"%s\"%s\n", id, open, label, close)
	} else {
		fmt.Fprintf(w, "  %s%s%s%s\n", id, open, label, close)
	}
}

func writeFlowEdge(w io.Writer, e *ast.FlowEdge) {
	from := ast.SanitizeID(e.From)
	to := ast.SanitizeID(e.To)

	// An empty Style defaults to a solid arrow: without an arrow token the
	// line parses as two adjacent node references and Pele/Mermaid reject it.
	style := e.Style
	if style == "" {
		style = ast.EdgeSolid
	}

	if e.Style == ast.EdgeInvisible {
		// Invisible links carry no label.
		fmt.Fprintf(w, "  %s ~~~ %s\n", from, to)
		return
	}

	if e.Label == "" {
		fmt.Fprintf(w, "  %s %s %s\n", from, string(style), to)
		return
	}

	label := escapeFlowText(e.Label)
	// EdgeNoArrow uses "-- text ---" syntax; all other styles use "|text|".
	if style == ast.EdgeNoArrow {
		fmt.Fprintf(w, "  %s -- %s --- %s\n", from, label, to)
	} else {
		fmt.Fprintf(w, "  %s %s|%s| %s\n", from, string(style), label, to)
	}
}

// needsQuote returns true if a Mermaid label should be wrapped in double-quotes.
func needsQuote(s string) bool {
	for _, r := range s {
		if r == '"' || r == '[' || r == ']' || r == '(' || r == ')' ||
			r == '{' || r == '}' || r == '<' || r == '>' || r == '|' {
			return true
		}
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// Sequence
// ────────────────────────────────────────────────────────────────────────────

func encodeSequence(w io.Writer, s *ast.SequenceDiagram) error {
	writeTitle(w, s.Title())
	fmt.Fprintln(w, "sequenceDiagram")
	if s.Autonumber() {
		fmt.Fprintln(w, "  autonumber")
	}
	for _, p := range s.Participants() {
		alias := p.Alias
		// An empty Kind defaults to `participant`: without the keyword the
		// line parses as a message and Pele/Mermaid reject it.
		kind := p.Kind
		if kind == "" {
			kind = ast.ParticipantBox
		}
		if p.Label != "" && p.Label != p.Alias {
			fmt.Fprintf(w, "  %s %s as %s\n", string(kind), alias, p.Label)
		} else {
			fmt.Fprintf(w, "  %s %s\n", string(kind), alias)
		}
	}
	encodeSeqMessages(w, s.Messages(), "  ")

	for _, loop := range s.Loops() {
		fmt.Fprintf(w, "  loop %s\n", loop.Label)
		encodeSeqMessages(w, loop.Messages, "    ")
		fmt.Fprintln(w, "  end")
	}
	for _, alt := range s.Alts() {
		fmt.Fprintf(w, "  alt %s\n", alt.Condition)
		encodeSeqMessages(w, alt.Messages, "    ")
		if len(alt.Else) > 0 {
			fmt.Fprintln(w, "  else")
			encodeSeqMessages(w, alt.Else, "    ")
		}
		fmt.Fprintln(w, "  end")
	}
	for _, n := range s.Notes() {
		switch len(n.Over) {
		case 0:
		case 1:
			fmt.Fprintf(w, "  Note over %s: %s\n", n.Over[0], n.Text)
		default:
			fmt.Fprintf(w, "  Note over %s: %s\n", strings.Join(n.Over, ","), n.Text)
		}
	}
	return nil
}

func encodeSeqMessages(w io.Writer, msgs []ast.SeqMessage, indent string) {
	for _, m := range msgs {
		// An empty Style defaults to a synchronous message: without an
		// arrow token the line does not parse.
		style := m.Style
		if style == "" {
			style = ast.MsgSync
		}
		target := m.To
		if m.Activate {
			target = target + "+"
		} else if m.Deactivate {
			target = target + "-"
		}
		fmt.Fprintf(w, "%s%s%s%s: %s\n", indent, m.From, string(style), target, m.Text)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// State
// ────────────────────────────────────────────────────────────────────────────

func encodeState(w io.Writer, s *ast.StateDiagram) error {
	writeTitle(w, s.Title())
	fmt.Fprintln(w, "stateDiagram-v2")

	for _, st := range s.States() {
		encodeStateNode(w, st, "  ")
	}
	for _, t := range s.Transitions() {
		from := stateRef(t.From)
		to := stateRef(t.To)
		if t.Event != "" {
			fmt.Fprintf(w, "  %s --> %s : %s\n", from, to, t.Event)
		} else {
			fmt.Fprintf(w, "  %s --> %s\n", from, to)
		}
	}
	for _, n := range s.Notes() {
		fmt.Fprintf(w, "  note right of %s\n    %s\n  end note\n", n.State, n.Text)
	}
	return nil
}

func encodeStateNode(w io.Writer, st ast.DiagramState, indent string) {
	switch st.Kind {
	case ast.StateStart, ast.StateEnd:
		// [*] nodes are referenced in transitions; no separate declaration needed.
		return
	case ast.StateFork:
		fmt.Fprintf(w, "%sstate %s <<fork>>\n", indent, st.ID)
	case ast.StateJoin:
		fmt.Fprintf(w, "%sstate %s <<join>>\n", indent, st.ID)
	case ast.StateChoice:
		fmt.Fprintf(w, "%sstate %s <<choice>>\n", indent, st.ID)
	default:
		if len(st.Children) == 0 {
			if st.Label != "" && st.Label != st.ID {
				fmt.Fprintf(w, "%sstate \"%s\" as %s\n", indent, st.Label, st.ID)
			}
			// Simple states with no label alias are implied by transition references.
			return
		}
		// Composite state.
		fmt.Fprintf(w, "%sstate %s {\n", indent, st.ID)
		for _, child := range st.Children {
			encodeStateNode(w, child, indent+"  ")
		}
		fmt.Fprintf(w, "%s}\n", indent)
	}
}

// stateRef maps StateKind start/end to "[*]" and everything else to the ID.
func stateRef(id string) string {
	if id == string(ast.StateStart) || id == string(ast.StateEnd) || id == "[*]" {
		return "[*]"
	}
	return id
}

// ────────────────────────────────────────────────────────────────────────────
// ER
// ────────────────────────────────────────────────────────────────────────────

func encodeER(w io.Writer, e *ast.ERDiagram) error {
	writeTitle(w, e.Title())
	fmt.Fprintln(w, "erDiagram")

	for _, en := range e.Entities() {
		name := ast.SanitizeID(en.Name)
		if len(en.Attributes) == 0 {
			fmt.Fprintf(w, "  %s\n", name)
			continue
		}
		fmt.Fprintf(w, "  %s {\n", name)
		for _, attr := range en.Attributes {
			// Attribute names are bare words in the ER grammar: spaces and
			// quotes break parsing, so sanitize to [A-Za-z0-9_-].
			// Comments are "..."-quoted: escape embedded quotes as #quot;.
			keys := ""
			for _, k := range attr.Keys {
				keys += " " + string(k)
			}
			comment := ""
			if attr.Comment != "" {
				comment = " \"" + strings.ReplaceAll(attr.Comment, `"`, "#quot;") + "\""
			}
			fmt.Fprintf(w, "    %s %s%s%s\n", attr.DataType, ast.SanitizeID(attr.Name), keys, comment)
		}
		fmt.Fprintln(w, "  }")
	}

	for _, r := range e.Relations() {
		line := "--"
		if !r.Identifying {
			line = ".."
		}
		right := mirrorCardinality(r.ToCard)
		label := r.Label
		if label == "" {
			label = "relates"
		}
		// Relation labels are unquoted `: label`. A bare & breaks parsing,
		// so labels containing &, " or : are "..."-quoted with #quot;-escaped
		// quotes (verified against the Pele/Mermaid 12.1.0 grammar).
		if strings.ContainsAny(label, "&\":") {
			label = `"` + strings.ReplaceAll(label, `"`, "#quot;") + `"`
		}
		fmt.Fprintf(w, "  %s %s%s%s %s : %s\n",
			ast.SanitizeID(r.From), string(r.FromCard), line, right, ast.SanitizeID(r.To), label)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Class
// ────────────────────────────────────────────────────────────────────────────

func encodeClass(w io.Writer, c *ast.ClassDiagram) error {
	writeTitle(w, c.Title())
	fmt.Fprintln(w, "classDiagram")

	for _, cl := range c.Classes() {
		if cl.Annotation != "" {
			fmt.Fprintf(w, "  class %s {\n    %s\n", cl.Name, cl.Annotation)
		} else {
			fmt.Fprintf(w, "  class %s {\n", cl.Name)
		}
		for _, m := range cl.Members {
			fmt.Fprintf(w, "    %s\n", formatClassMember(m))
		}
		fmt.Fprintln(w, "  }")
	}

	for _, r := range c.Relations() {
		cardFrom := ""
		if r.CardFrom != "" {
			cardFrom = " \"" + r.CardFrom + "\""
		}
		cardTo := ""
		if r.CardTo != "" {
			cardTo = " \"" + r.CardTo + "\""
		}
		label := ""
		if r.Label != "" {
			label = " : " + r.Label
		}
		fmt.Fprintf(w, "  %s%s %s%s %s%s\n",
			r.From, cardFrom, string(r.Kind), cardTo, r.To, label)
	}

	for _, n := range c.Notes() {
		// Notes are "..."-quoted: escape embedded quotes as #quot;.
		fmt.Fprintf(w, "  note for %s \"%s\"\n", n.Class, strings.ReplaceAll(n.Text, `"`, "#quot;"))
	}
	return nil
}

func formatClassMember(m ast.ClassMember) string {
	var sb strings.Builder
	sb.WriteString(string(m.Visibility))
	if m.Type != "" {
		sb.WriteString(m.Type)
		sb.WriteRune(' ')
	}
	sb.WriteString(m.Name)
	if m.IsMethod {
		sb.WriteString("()")
	}
	if m.IsStatic {
		sb.WriteString("$")
	}
	if m.IsAbstract {
		sb.WriteString("*")
	}
	return sb.String()
}

// ────────────────────────────────────────────────────────────────────────────
// Pie
// ────────────────────────────────────────────────────────────────────────────

func encodePie(w io.Writer, p *ast.PieDiagram) error {
	writeTitle(w, p.Title())
	fmt.Fprintln(w, "pie")

	for _, s := range p.Slices() {
		if math.IsNaN(s.Value) || math.IsInf(s.Value, 0) {
			return fmt.Errorf("mmd: pie slice %q has non-finite value %v", s.Label, s.Value)
		}
		if s.Value < 0 {
			return fmt.Errorf("mmd: pie slice %q has negative value %v", s.Label, s.Value)
		}
		// Labels are always "..."-quoted: bare words fail parsing, and ":",
		// "|", "&" need no escaping inside quotes. Embedded quotes end the
		// label, so they become #quot; (verified against Pele/Mermaid 12.1.0).
		fmt.Fprintf(w, "  \"%s\" : %s\n",
			strings.ReplaceAll(s.Label, `"`, "#quot;"),
			strconv.FormatFloat(s.Value, 'f', -1, 64))
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Quadrant
// ────────────────────────────────────────────────────────────────────────────

// escapeQuadrantText escapes characters that break quadrantChart bare text:
// ':' ends axis labels and point names, '"' ends quoted point names.
func escapeQuadrantText(s string) string {
	s = strings.ReplaceAll(s, `"`, "#quot;")
	return strings.ReplaceAll(s, ":", "#58;")
}

func encodeQuadrant(w io.Writer, q *ast.QuadrantDiagram) error {
	writeTitle(w, q.Title())
	fmt.Fprintln(w, "quadrantChart")

	if x := q.XAxis(); x[0] != "" || x[1] != "" {
		if x[1] != "" {
			fmt.Fprintf(w, "  x-axis %s --> %s\n", escapeQuadrantText(x[0]), escapeQuadrantText(x[1]))
		} else {
			fmt.Fprintf(w, "  x-axis %s\n", escapeQuadrantText(x[0]))
		}
	}
	if y := q.YAxis(); y[0] != "" || y[1] != "" {
		if y[1] != "" {
			fmt.Fprintf(w, "  y-axis %s --> %s\n", escapeQuadrantText(y[0]), escapeQuadrantText(y[1]))
		} else {
			fmt.Fprintf(w, "  y-axis %s\n", escapeQuadrantText(y[0]))
		}
	}
	for i, label := range q.Quadrants() {
		if label != "" {
			fmt.Fprintf(w, "  quadrant-%d %s\n", i+1, escapeQuadrantText(label))
		}
	}

	for _, p := range q.Points() {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
			return fmt.Errorf("mmd: quadrant point %q has non-finite coordinates (%v, %v)", p.Name, p.X, p.Y)
		}
		if p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
			return fmt.Errorf("mmd: quadrant point %q outside 0..1 range (%v, %v)", p.Name, p.X, p.Y)
		}
		// Point names are bare unless they contain ':' or '"', which break
		// parsing; quoted names use #quot;-escaped quotes.
		name := p.Name
		if strings.ContainsAny(name, ":\"") {
			name = `"` + strings.ReplaceAll(name, `"`, "#quot;") + `"`
		}
		fmt.Fprintf(w, "  %s: [%s, %s]\n", name,
			strconv.FormatFloat(p.X, 'f', -1, 64),
			strconv.FormatFloat(p.Y, 'f', -1, 64))
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Mindmap
// ────────────────────────────────────────────────────────────────────────────

// mindShapeSyntax returns the Mermaid bracket pair for a mindmap node shape.
// Only Circle, Rect, RoundedRect (and the zero value), and Hexagon have
// mindmap syntax; other flowchart shapes fall back to rounded parentheses.
func mindShapeSyntax(s ast.NodeShape) (open, close string) {
	switch s {
	case ast.ShapeCircle:
		return "((", "))"
	case ast.ShapeRect:
		return "[", "]"
	case ast.ShapeHexagon:
		return "{{", "}}"
	default:
		return "(", ")"
	}
}

// escapeMindLabel escapes characters that break mindmap label parsing inside
// id[form] labels: brackets and parens end the label, and braces would open a
// new shape, so they become numeric entities (verified against Pele/Mermaid
// 12.1.0). Quotes, ampersands, colons, and angle brackets need no escaping in
// either form.
func escapeMindLabel(s string) string {
	s = strings.ReplaceAll(s, "[", "#91;")
	s = strings.ReplaceAll(s, "]", "#93;")
	s = strings.ReplaceAll(s, "(", "#40;")
	s = strings.ReplaceAll(s, ")", "#41;")
	s = strings.ReplaceAll(s, "{", "#123;")
	return strings.ReplaceAll(s, "}", "#125;")
}

func encodeMindmap(w io.Writer, m *ast.MindmapDiagram) error {
	writeTitle(w, m.Title())
	fmt.Fprintln(w, "mindmap")
	if m.Root() == nil {
		return nil
	}
	st := &mindEncodeState{used: mindExplicitIDs(m.Root()), seen: make(map[*ast.MindNode]bool)}
	encodeMindNode(w, m.Root(), 1, st)
	return nil
}

// mindEncodeState carries the auto-id counter and cycle guard across the
// recursive mindmap encoding. Trees must be acyclic; a repeated node is
// emitted once and subsequent references are dropped.
type mindEncodeState struct {
	counter int
	used    map[string]bool
	seen    map[*ast.MindNode]bool
}

// mindExplicitIDs collects sanitized explicit node IDs so auto-generated ids
// cannot collide with them.
func mindExplicitIDs(root *ast.MindNode) map[string]bool {
	used := make(map[string]bool)
	seen := make(map[*ast.MindNode]bool)
	var walk func(n *ast.MindNode)
	walk = func(n *ast.MindNode) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		if n.ID != "" {
			used[ast.SanitizeID(n.ID)] = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return used
}

func (st *mindEncodeState) autoID() string {
	for {
		st.counter++
		id := fmt.Sprintf("m%d", st.counter)
		if !st.used[id] {
			st.used[id] = true
			return id
		}
	}
}

func encodeMindNode(w io.Writer, n *ast.MindNode, depth int, st *mindEncodeState) {
	if n == nil || st.seen[n] {
		return
	}
	st.seen[n] = true
	indent := strings.Repeat("  ", depth)
	// Newlines would break the indentation structure; whitespace-only labels
	// would vanish, so both flatten to a placeholder like empty labels.
	label := strings.ReplaceAll(n.Label, "\n", " ")
	label = strings.ReplaceAll(label, "\r", " ")
	if strings.TrimSpace(label) == "" {
		label = "(empty)"
	}
	// Bare labels carry no shape: they need the id[form] when an explicit ID
	// or a non-default shape is set, or when brackets, parens, or braces
	// would terminate the bare text.
	needsDelim := n.ID != "" || (n.Shape != "" && n.Shape != ast.ShapeRoundedRect) ||
		strings.ContainsAny(label, "()[]{}")
	if !needsDelim {
		fmt.Fprintf(w, "%s%s\n", indent, label)
	} else {
		id := n.ID
		if id == "" {
			id = st.autoID()
		}
		open, close := mindShapeSyntax(n.Shape)
		fmt.Fprintf(w, "%s%s%s%s%s\n", indent, ast.SanitizeID(id), open, escapeMindLabel(label), close)
	}
	for _, c := range n.Children {
		encodeMindNode(w, c, depth+1, st)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Gantt
// ────────────────────────────────────────────────────────────────────────────

func encodeGantt(w io.Writer, g *ast.GanttDiagram) error {
	// Resolve first: validates task IDs, tags, dates, durations, and
	// after/until references with clear errors before emitting anything.
	if _, err := g.Resolve(); err != nil {
		return fmt.Errorf("mmd: %w", err)
	}
	writeTitle(w, g.Title())
	fmt.Fprintln(w, "gantt")

	if f := g.DateFormat(); f != "" {
		fmt.Fprintf(w, "  dateFormat %s\n", f)
	}
	if f := g.AxisFormat(); f != "" {
		fmt.Fprintf(w, "  axisFormat %s\n", f)
	}
	for _, t := range g.Tasks() {
		encodeGanttTask(w, t, "  ")
	}
	for _, s := range g.Sections() {
		fmt.Fprintf(w, "  section %s\n", oneLine(s.Name))
		for _, t := range s.Tasks {
			encodeGanttTask(w, t, "    ")
		}
	}
	return nil
}

// oneLine flattens newlines that would break indentation-based syntax.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}

func encodeGanttTask(w io.Writer, t ast.GanttTask, indent string) {
	parts := make([]string, 0, 5)
	for _, tag := range t.Tags {
		parts = append(parts, string(tag))
	}
	if t.ID != "" {
		parts = append(parts, t.ID)
	}
	parts = append(parts, strings.TrimSpace(t.Start), strings.TrimSpace(t.End))
	fmt.Fprintf(w, "%s%s : %s\n", indent, oneLine(t.Name), strings.Join(parts, ", "))
}
