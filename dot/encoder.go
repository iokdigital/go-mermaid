// Package dot encodes flowchart, sequence, state, ER, and class diagram ASTs to Graphviz DOT format.
// Other diagram types return ErrRendererNotAvailable.
package dot

import (
	"fmt"
	"io"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Encode writes a Graphviz DOT representation of d to w.
// Flowchart, sequence, state, ER, and class diagrams are supported; other types return a
// FallbackFormatError wrapping ErrRendererNotAvailable.
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
	default:
		return &diagram.FallbackFormatError{
			Err:      fmt.Errorf("dot: %w: %T", diagram.ErrRendererNotAvailable, d),
			Fallback: diagram.FormatMMD,
		}
	}
}

func encodeFlowchart(w io.Writer, f *ast.FlowchartDiagram) error {
	rankdir := dirToRankdir(f.Direction())
	title := f.Title()

	fmt.Fprintln(w, "digraph {")
	fmt.Fprintf(w, "  rankdir=%s;\n", rankdir)
	if title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=box style=rounded];")

	for _, sg := range f.Subgraphs() {
		sgID := sanitizeDOTID(sg.ID)
		label := sg.Label
		if label == "" {
			label = sg.ID
		}
		fmt.Fprintf(w, "  subgraph cluster_%s {\n", sgID)
		fmt.Fprintf(w, "    label=%q;\n", label)
		nodeByID := indexNodes(f.Nodes())
		for _, nid := range sg.Nodes {
			if n, ok := nodeByID[nid]; ok {
				writeNode(w, n, "    ")
			}
		}
		fmt.Fprintln(w, "  }")
	}

	// Emit nodes not in any subgraph.
	subgraphMember := make(map[string]bool)
	for _, sg := range f.Subgraphs() {
		for _, id := range sg.Nodes {
			subgraphMember[id] = true
		}
	}
	for _, n := range f.Nodes() {
		if !subgraphMember[n.ID] {
			writeNode(w, n, "  ")
		}
	}

	for _, e := range f.Edges() {
		writeEdge(w, e)
	}

	fmt.Fprintln(w, "}")
	return nil
}

func writeNode(w io.Writer, n *ast.FlowNode, indent string) {
	id := sanitizeDOTID(n.ID)
	label := n.Label
	if label == "" {
		label = n.ID
	}
	shape := dotShape(n.Shape)
	attrs := fmt.Sprintf("label=%q shape=%s", label, shape)
	if n.Confidence > 0 {
		fill := confidenceFill(n.Confidence)
		attrs += fmt.Sprintf(" style=filled fillcolor=%q", fill)
	}
	fmt.Fprintf(w, "%s%s [%s];\n", indent, id, attrs)
}

func writeEdge(w io.Writer, e *ast.FlowEdge) {
	from := sanitizeDOTID(e.From)
	to := sanitizeDOTID(e.To)
	attrs := ""
	if e.Label != "" {
		attrs = fmt.Sprintf(" [label=%q]", e.Label)
	}
	switch e.Style {
	case ast.EdgeInvisible:
		attrs = " [style=invis]"
	case ast.EdgeDotted:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [style=dashed label=%q]", e.Label)
		} else {
			attrs = " [style=dashed]"
		}
	case ast.EdgeThick:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [style=bold label=%q]", e.Label)
		} else {
			attrs = " [style=bold]"
		}
	case ast.EdgeNoArrow:
		if e.Label != "" {
			attrs = fmt.Sprintf(" [arrowhead=none label=%q]", e.Label)
		} else {
			attrs = " [arrowhead=none]"
		}
	}
	fmt.Fprintf(w, "  %s -> %s%s;\n", from, to, attrs)
}

func dirToRankdir(d ast.Direction) string {
	switch d {
	case ast.DirectionLR:
		return "LR"
	case ast.DirectionBT:
		return "BT"
	case ast.DirectionRL:
		return "RL"
	default:
		return "TB"
	}
}

func dotShape(s ast.NodeShape) string {
	switch s {
	case ast.ShapeDiamond:
		return "diamond"
	case ast.ShapeCircle:
		return "circle"
	case ast.ShapeRoundedRect:
		return "box"
	case ast.ShapeParallelogram:
		return "parallelogram"
	case ast.ShapeHexagon:
		return "hexagon"
	case ast.ShapeStadium:
		return "stadium"
	case ast.ShapeAsymmetric:
		return "cds"
	default:
		return "box"
	}
}

func confidenceFill(c float64) string {
	switch {
	case c >= 0.90:
		return "#90EE90"
	case c >= 0.70:
		return "#FFD700"
	default:
		return "#FFB6C1"
	}
}

func sanitizeDOTID(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func indexNodes(nodes []*ast.FlowNode) map[string]*ast.FlowNode {
	m := make(map[string]*ast.FlowNode, len(nodes))
	for _, n := range nodes {
		m[n.ID] = n
	}
	return m
}

// ────────────────────────────────────────────────────────────────────────────
// Sequence
// ────────────────────────────────────────────────────────────────────────────

// encodeSequence writes a Graphviz DOT representation of a sequence diagram.
//
// Mapping notes (DOT has no native sequence concept):
//   - participants become box nodes (ellipse for actors); time flows top-down
//     (rankdir=TB);
//   - messages become labelled edges; dashed MessageStyles map to
//     style=dashed, cross-end styles to arrowhead=tee, open-arrow styles to
//     arrowhead=vee, and no-arrow styles to arrowhead=none;
//   - autonumber prefixes each message label with its 1-based sequence number
//     in emission order (top-level, loops, alts), matching the SVG renderer;
//   - activation/deactivation is appended to the edge label as " [+]" / " [-]";
//   - loops and alt/else branches become cluster subgraphs; notes become
//     shape=note nodes linked to their participants with dotted edges.
func encodeSequence(w io.Writer, s *ast.SequenceDiagram) error {
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  rankdir=TB;")
	if title := s.Title(); title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=box style=rounded];")

	for _, p := range s.Participants() {
		id := sanitizeDOTID(p.Alias)
		label := p.Label
		if label == "" {
			label = p.Alias
		}
		shape := "box"
		if p.Kind == ast.ParticipantActor {
			shape = "ellipse"
		}
		fmt.Fprintf(w, "  %s [label=%q shape=%s];\n", id, label, shape)
	}

	number := 0 // autonumber counter shared across all message groups
	nextLabel := func(text string) string {
		if s.Autonumber() {
			number++
			if text != "" {
				return fmt.Sprintf("%d: %s", number, text)
			}
			return fmt.Sprintf("%d", number)
		}
		return text
	}

	for _, m := range s.Messages() {
		writeSeqEdge(w, m.From, m.To, nextLabel(m.Text), m.Style, m.Activate, m.Deactivate, "  ")
	}

	for i, l := range s.Loops() {
		label := "loop"
		if l.Label != "" {
			label = "loop: " + l.Label
		}
		fmt.Fprintf(w, "  subgraph cluster_seq_loop_%d {\n", i)
		fmt.Fprintf(w, "    label=%q;\n", label)
		for _, m := range l.Messages {
			writeSeqEdge(w, m.From, m.To, nextLabel(m.Text), m.Style, m.Activate, m.Deactivate, "    ")
		}
		fmt.Fprintln(w, "  }")
	}

	for i, a := range s.Alts() {
		label := "alt"
		if a.Condition != "" {
			label = "alt: " + a.Condition
		}
		fmt.Fprintf(w, "  subgraph cluster_seq_alt_%d {\n", i)
		fmt.Fprintf(w, "    label=%q;\n", label)
		for _, m := range a.Messages {
			writeSeqEdge(w, m.From, m.To, nextLabel(m.Text), m.Style, m.Activate, m.Deactivate, "    ")
		}
		fmt.Fprintln(w, "  }")
		if len(a.Else) > 0 {
			fmt.Fprintf(w, "  subgraph cluster_seq_alt_%d_else {\n", i)
			fmt.Fprintln(w, `    label="else";`)
			for _, m := range a.Else {
				writeSeqEdge(w, m.From, m.To, nextLabel(m.Text), m.Style, m.Activate, m.Deactivate, "    ")
			}
			fmt.Fprintln(w, "  }")
		}
	}

	for i, n := range s.Notes() {
		if len(n.Over) == 0 {
			continue
		}
		nid := fmt.Sprintf("seqnote_%d", i)
		fmt.Fprintf(w, "  %s [label=%q shape=note];\n", nid, n.Text)
		for _, over := range n.Over {
			fmt.Fprintf(w, "  %s -> %s [style=dotted arrowhead=none];\n", nid, sanitizeDOTID(over))
		}
	}

	fmt.Fprintln(w, "}")
	return nil
}

// writeSeqEdge writes a single sequence message as a DOT edge.
func writeSeqEdge(w io.Writer, from, to, label string, style ast.MessageStyle, activate, deactivate bool, indent string) {
	if activate {
		if label != "" {
			label += " [+]"
		} else {
			label = "[+]"
		}
	} else if deactivate {
		if label != "" {
			label += " [-]"
		} else {
			label = "[-]"
		}
	}

	attrs := make([]string, 0, 3)
	if label != "" {
		attrs = append(attrs, fmt.Sprintf("label=%q", label))
	}
	// MsgSync and the zero-value style both render as a plain solid arrow.
	switch style {
	case ast.MsgAsync:
		attrs = append(attrs, "style=dashed")
	case ast.MsgSyncNoArrow:
		attrs = append(attrs, "arrowhead=none")
	case ast.MsgAsyncNoArrow:
		attrs = append(attrs, "style=dashed", "arrowhead=none")
	case ast.MsgSyncX:
		attrs = append(attrs, "arrowhead=tee")
	case ast.MsgAsyncX:
		attrs = append(attrs, "style=dashed", "arrowhead=tee")
	case ast.MsgOpen:
		attrs = append(attrs, "arrowhead=vee")
	case ast.MsgAsyncOpen:
		attrs = append(attrs, "style=dashed", "arrowhead=vee")
	}
	attrStr := ""
	if len(attrs) > 0 {
		attrStr = " [" + strings.Join(attrs, " ") + "]"
	}
	fmt.Fprintf(w, "%s%s -> %s%s;\n", indent, sanitizeDOTID(from), sanitizeDOTID(to), attrStr)
}

// ────────────────────────────────────────────────────────────────────────────
// State
// ────────────────────────────────────────────────────────────────────────────

// encodeState writes a Graphviz DOT representation of a state diagram.
//
// Mapping notes:
//   - normal states become box nodes; choice states become diamonds;
//   - fork/join states become filled black bars (fixed-size box nodes);
//   - start states become filled black circles, end states double circles;
//   - the Mermaid [*] pseudo-state maps to synthetic __start/__end nodes
//     (position-dependent: [*] as a source is __start, as a target __end);
//   - composite (nested) states become cluster subgraphs;
//   - transitions become labelled edges (Event as the label);
//   - states referenced only by transitions (never declared via AddState)
//     are emitted as default box nodes so the graph is complete;
//   - notes become shape=note nodes linked to their state with dotted edges.
func encodeState(w io.Writer, s *ast.StateDiagram) error {
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  rankdir=TB;")
	if title := s.Title(); title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=box style=rounded];")

	declared := make(map[string]bool)
	for _, st := range s.States() {
		markStateIDs(st, declared)
	}

	needStart, needEnd := false, false
	for _, t := range s.Transitions() {
		if stateEndpointID(t.From, true) == "__start" {
			needStart = true
		}
		if stateEndpointID(t.To, false) == "__end" {
			needEnd = true
		}
		// A bare "start"/"end" ID on the opposite side still maps to its
		// synthetic node; cover it via the endpoint mapping itself.
		if stateEndpointID(t.To, false) == "__start" {
			needStart = true
		}
		if stateEndpointID(t.From, true) == "__end" {
			needEnd = true
		}
	}
	if needStart {
		fmt.Fprintln(w, `  __start [label="" shape=circle style=filled fillcolor=black width=0.3 fixedsize=true];`)
	}
	if needEnd {
		fmt.Fprintln(w, `  __end [label="" shape=doublecircle width=0.35 fixedsize=true];`)
	}

	for _, st := range s.States() {
		writeStateNode(w, st, "  ")
	}

	// Emit transition-only states so every edge endpoint exists as a node.
	seen := make(map[string]bool)
	for _, t := range s.Transitions() {
		for _, end := range []struct {
			id     string
			source bool
		}{{t.From, true}, {t.To, false}} {
			mapped := stateEndpointID(end.id, end.source)
			if mapped == "__start" || mapped == "__end" {
				continue
			}
			if declared[mapped] || seen[mapped] {
				continue
			}
			seen[mapped] = true
			fmt.Fprintf(w, "  %s [label=%q shape=box];\n", sanitizeDOTID(mapped), mapped)
		}
	}

	for _, t := range s.Transitions() {
		from := stateEndpointID(t.From, true)
		to := stateEndpointID(t.To, false)
		attrs := ""
		if t.Event != "" {
			attrs = fmt.Sprintf(" [label=%q]", t.Event)
		}
		fmt.Fprintf(w, "  %s -> %s%s;\n", sanitizeDOTID(from), sanitizeDOTID(to), attrs)
	}

	for i, n := range s.Notes() {
		if n.State == "" {
			continue
		}
		nid := fmt.Sprintf("statenote_%d", i)
		fmt.Fprintf(w, "  %s [label=%q shape=note];\n", nid, n.Text)
		fmt.Fprintf(w, "  %s -> %s [style=dotted arrowhead=none];\n", nid, sanitizeDOTID(stateEndpointID(n.State, false)))
	}

	fmt.Fprintln(w, "}")
	return nil
}

// writeStateNode writes a single state (or composite cluster) as DOT.
func writeStateNode(w io.Writer, st ast.DiagramState, indent string) {
	if len(st.Children) > 0 {
		label := st.Label
		if label == "" {
			label = st.ID
		}
		fmt.Fprintf(w, "%ssubgraph cluster_%s {\n", indent, sanitizeDOTID(st.ID))
		fmt.Fprintf(w, "%s  label=%q;\n", indent, label)
		for _, child := range st.Children {
			writeStateNode(w, child, indent+"  ")
		}
		fmt.Fprintf(w, "%s}\n", indent)
		return
	}

	id := sanitizeDOTID(st.ID)
	label := st.Label
	if label == "" {
		label = st.ID
	}
	switch st.Kind {
	case ast.StateStart:
		fmt.Fprintf(w, "%s%s [label=%q shape=circle style=filled fillcolor=black width=0.3 fixedsize=true];\n", indent, id, label)
	case ast.StateEnd:
		fmt.Fprintf(w, "%s%s [label=%q shape=doublecircle width=0.35 fixedsize=true];\n", indent, id, label)
	case ast.StateFork, ast.StateJoin:
		fmt.Fprintf(w, "%s%s [label=\"\" shape=box style=filled fillcolor=black width=1.2 height=0.15 fixedsize=true];\n", indent, id)
	case ast.StateChoice:
		fmt.Fprintf(w, "%s%s [label=%q shape=diamond];\n", indent, id, label)
	default:
		fmt.Fprintf(w, "%s%s [label=%q shape=box];\n", indent, id, label)
	}
}

// markStateIDs records declared state IDs, recursing into composites.
func markStateIDs(st ast.DiagramState, declared map[string]bool) {
	declared[st.ID] = true
	for _, child := range st.Children {
		markStateIDs(child, declared)
	}
}

// stateEndpointID maps Mermaid state references to DOT node IDs. The [*]
// pseudo-state (and bare "start"/"end" kinds) maps to synthetic __start/__end
// nodes based on edge position; everything else maps to its own ID.
func stateEndpointID(id string, isSource bool) string {
	switch id {
	case "[*]":
		if isSource {
			return "__start"
		}
		return "__end"
	case string(ast.StateStart):
		return "__start"
	case string(ast.StateEnd):
		return "__end"
	default:
		return id
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ER
// ────────────────────────────────────────────────────────────────────────────

// encodeER writes a Graphviz DOT representation of an ER diagram.
//
// Mapping notes:
//   - entities become shape=record nodes; the entity name is the first field
//     and each attribute renders as "{type} {name} {KEYS}" (mirroring the MMD
//     encoder), one field per record compartment separated by "|";
//   - record-special characters ({, }, |, <, >) in names are backslash-escaped
//     so Graphviz parses the label as literal text (see dotRecordEscape);
//   - relationships become edges with arrowhead=none (ER connectors carry no
//     direction); non-identifying ("..") relations map to style=dashed;
//   - endpoint cardinalities map to taillabel/headlabel so both symbols are
//     preserved verbatim; the relationship label maps to the edge label;
//   - entities referenced only by relations (never declared via AddEntity)
//     are emitted as plain box nodes so the graph is complete.
func encodeER(w io.Writer, e *ast.ERDiagram) error {
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  rankdir=TB;")
	if title := e.Title(); title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=record];")

	declared := make(map[string]bool, len(e.Entities()))
	for _, en := range e.Entities() {
		declared[en.Name] = true
		fields := make([]string, 0, len(en.Attributes)+1)
		fields = append(fields, dotRecordEscape(en.Name))
		for _, attr := range en.Attributes {
			fields = append(fields, dotRecordEscape(erAttributeText(attr)))
		}
		fmt.Fprintf(w, "  %s [label=\"{%s}\"];\n", sanitizeDOTID(en.Name), strings.Join(fields, "|"))
	}

	// Emit relation-only entities so every edge endpoint exists as a node.
	seen := make(map[string]bool)
	for _, r := range e.Relations() {
		for _, name := range []string{r.From, r.To} {
			if declared[name] || seen[name] {
				continue
			}
			seen[name] = true
			fmt.Fprintf(w, "  %s [label=%q shape=box];\n", sanitizeDOTID(name), name)
		}
	}

	for _, r := range e.Relations() {
		attrs := []string{"arrowhead=none"}
		if r.Label != "" {
			attrs = append(attrs, fmt.Sprintf("label=%q", r.Label))
		}
		if string(r.FromCard) != "" {
			attrs = append(attrs, fmt.Sprintf("taillabel=%q", string(r.FromCard)))
		}
		if string(r.ToCard) != "" {
			attrs = append(attrs, fmt.Sprintf("headlabel=%q", string(r.ToCard)))
		}
		if !r.Identifying {
			attrs = append(attrs, "style=dashed")
		}
		fmt.Fprintf(w, "  %s -> %s [%s];\n", sanitizeDOTID(r.From), sanitizeDOTID(r.To), strings.Join(attrs, " "))
	}

	fmt.Fprintln(w, "}")
	return nil
}

// erAttributeText formats one ER attribute the same way the MMD encoder does:
// "{type} {name} {KEYS}" with an optional trailing comment in quotes.
func erAttributeText(attr ast.ERAttribute) string {
	var b strings.Builder
	b.WriteString(attr.DataType)
	if attr.Name != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(attr.Name)
	}
	for _, k := range attr.Keys {
		b.WriteString(" ")
		b.WriteString(string(k))
	}
	if attr.Comment != "" {
		b.WriteString(` "`)
		b.WriteString(attr.Comment)
		b.WriteString(`"`)
	}
	return b.String()
}

// dotRecordEscape escapes characters that are structural in DOT record labels
// ({, }, |, <, >) plus backslash and double-quote, so the label renders as
// literal text. The caller wraps the result in double quotes.
func dotRecordEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`{`, `\{`,
		`}`, `\}`,
		`|`, `\|`,
		`<`, `\<`,
		`>`, `\>`,
	)
	return r.Replace(s)
}

// ────────────────────────────────────────────────────────────────────────────
// Class
// ────────────────────────────────────────────────────────────────────────────

// encodeClass writes a Graphviz DOT representation of a class diagram.
//
// Mapping notes:
//   - classes become shape=record nodes; the class name is the first field,
//     followed by one compartment for attributes (non-method members) and one
//     for methods, each member on its own left-justified line with the same
//     member text the MMD encoder emits; annotations (e.g. "<<interface>>")
//     render as a leading compartment;
//   - record-special characters ({, }, |, <, >) in names are backslash-escaped
//     so Graphviz parses the label as literal text (see dotRecordEscape);
//   - namespaces are not represented as clusters (matching the MMD encoder,
//     which also ignores them); the node ID is the sanitized class name;
//   - relationships become edges preserving the From -> To direction with the
//     decorator at the head end (matching the SVG renderer): inheritance and
//     realization use arrowhead=empty (realization adds style=dashed),
//     composition/aggregation use a tail diamond (arrowtail=diamond/odiamond
//     with dir=both and arrowhead=none), dependency maps to style=dashed,
//     links map to arrowhead=none, and associations keep the default arrow;
//   - endpoint cardinalities map to taillabel/headlabel and the relationship
//     label maps to the edge label;
//   - classes referenced only by relations (never declared via AddClass) are
//     emitted as plain box nodes so the graph is complete;
//   - notes become shape=note nodes linked to their class with dotted edges.
func encodeClass(w io.Writer, c *ast.ClassDiagram) error {
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  rankdir=TB;")
	if title := c.Title(); title != "" {
		fmt.Fprintf(w, "  label=%q;\n", title)
		fmt.Fprintln(w, "  labelloc=t;")
	}
	fmt.Fprintln(w, "  node [fontname=Helvetica shape=record];")

	declared := make(map[string]bool, len(c.Classes()))
	for _, cl := range c.Classes() {
		declared[cl.Name] = true
		fields := make([]string, 0, 4)
		if cl.Annotation != "" {
			fields = append(fields, dotRecordEscape(cl.Annotation))
		}
		fields = append(fields, dotRecordEscape(cl.Name))
		if attrs := classMembers(cl.Members, false); attrs != "" {
			fields = append(fields, attrs)
		}
		if methods := classMembers(cl.Members, true); methods != "" {
			fields = append(fields, methods)
		}
		fmt.Fprintf(w, "  %s [label=\"{%s}\"];\n", sanitizeDOTID(cl.Name), strings.Join(fields, "|"))
	}

	// Emit relation-only classes so every edge endpoint exists as a node.
	seen := make(map[string]bool)
	for _, r := range c.Relations() {
		for _, name := range []string{r.From, r.To} {
			if declared[name] || seen[name] {
				continue
			}
			seen[name] = true
			fmt.Fprintf(w, "  %s [label=%q shape=box];\n", sanitizeDOTID(name), name)
		}
	}

	for _, r := range c.Relations() {
		attrs := classEdgeAttrs(r)
		fmt.Fprintf(w, "  %s -> %s [%s];\n", sanitizeDOTID(r.From), sanitizeDOTID(r.To), strings.Join(attrs, " "))
	}

	for i, n := range c.Notes() {
		if n.Class == "" {
			continue
		}
		nid := fmt.Sprintf("classnote_%d", i)
		fmt.Fprintf(w, "  %s [label=%q shape=note];\n", nid, n.Text)
		fmt.Fprintf(w, "  %s -> %s [style=dotted arrowhead=none];\n", nid, sanitizeDOTID(n.Class))
	}

	fmt.Fprintln(w, "}")
	return nil
}

// classMembers formats the attribute (methods=false) or method (methods=true)
// members of a class as one record compartment, one left-justified line per
// member. Member text mirrors the MMD encoder (see classMemberText); the "\l"
// line breaks are appended after escaping so they survive as separators.
func classMembers(members []ast.ClassMember, methods bool) string {
	var b strings.Builder
	for _, m := range members {
		if m.IsMethod != methods {
			continue
		}
		b.WriteString(dotRecordEscape(classMemberText(m)))
		b.WriteString(`\l`)
	}
	return b.String()
}

// classMemberText formats one class member the same way the MMD encoder does:
// "{visibility}{type} {name}" with "()" for methods and "$"/"*" suffixes for
// static/abstract members.
func classMemberText(m ast.ClassMember) string {
	var b strings.Builder
	b.WriteString(string(m.Visibility))
	if m.Type != "" {
		b.WriteString(m.Type)
		b.WriteString(" ")
	}
	b.WriteString(m.Name)
	if m.IsMethod {
		b.WriteString("()")
	}
	if m.IsStatic {
		b.WriteString("$")
	}
	if m.IsAbstract {
		b.WriteString("*")
	}
	return b.String()
}

// classEdgeAttrs maps a class relation to DOT edge attributes, preserving the
// From -> To direction with the decorator at the head end (matching the SVG
// renderer).
func classEdgeAttrs(r ast.ClassRelation) []string {
	attrs := make([]string, 0, 5)
	switch r.Kind {
	case ast.RelInheritance:
		attrs = append(attrs, "arrowhead=empty")
	case ast.RelRealization:
		attrs = append(attrs, "arrowhead=empty", "style=dashed")
	case ast.RelComposition:
		attrs = append(attrs, "dir=both", "arrowtail=diamond", "arrowhead=none")
	case ast.RelAggregation:
		attrs = append(attrs, "dir=both", "arrowtail=odiamond", "arrowhead=none")
	case ast.RelDependency:
		attrs = append(attrs, "style=dashed")
	case ast.RelLink:
		attrs = append(attrs, "arrowhead=none")
	case ast.RelAssociation:
		// Default solid arrow; no decorator attributes needed.
	}
	if r.Label != "" {
		attrs = append(attrs, fmt.Sprintf("label=%q", r.Label))
	}
	if r.CardFrom != "" {
		attrs = append(attrs, fmt.Sprintf("taillabel=%q", r.CardFrom))
	}
	if r.CardTo != "" {
		attrs = append(attrs, fmt.Sprintf("headlabel=%q", r.CardTo))
	}
	return attrs
}
