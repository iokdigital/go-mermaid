// Package json encodes diagram AST types to their JSON representation.
// The root encoding/json package is used internally; this package name shadows
// it only within this package's own source files.
package json

import (
	stdjson "encoding/json"
	"fmt"
	"io"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Encode writes the JSON representation of d to w.
func Encode(w io.Writer, d diagram.Diagram) error {
	payload, err := marshal(d)
	if err != nil {
		return err
	}
	enc := stdjson.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func marshal(d diagram.Diagram) (any, error) {
	switch v := d.(type) {
	case *ast.FlowchartDiagram:
		return marshalFlowchart(v), nil
	case *ast.SequenceDiagram:
		return marshalSequence(v), nil
	case *ast.StateDiagram:
		return marshalState(v), nil
	case *ast.ERDiagram:
		return marshalER(v), nil
	case *ast.ClassDiagram:
		return marshalClass(v), nil
	default:
		return nil, fmt.Errorf("json: %w: %T", diagram.ErrInvalidFormat, d)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Flowchart
// ────────────────────────────────────────────────────────────────────────────

type flowchartJSON struct {
	Type      string          `json:"type"`
	Title     string          `json:"title,omitempty"`
	Direction string          `json:"direction"`
	Nodes     []flowNodeJSON  `json:"nodes"`
	Edges     []flowEdgeJSON  `json:"edges"`
	Subgraphs []subgraphJSON  `json:"subgraphs,omitempty"`
}

type flowNodeJSON struct {
	ID         string            `json:"id"`
	Label      string            `json:"label,omitempty"`
	Shape      string            `json:"shape,omitempty"`
	Confidence float64           `json:"confidence,omitempty"`
	URL        string            `json:"url,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type flowEdgeJSON struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Style string `json:"style,omitempty"`
}

type subgraphJSON struct {
	ID    string   `json:"id"`
	Label string   `json:"label,omitempty"`
	Nodes []string `json:"nodes"`
}

func marshalFlowchart(f *ast.FlowchartDiagram) flowchartJSON {
	nodes := make([]flowNodeJSON, len(f.Nodes()))
	for i, n := range f.Nodes() {
		nodes[i] = flowNodeJSON{
			ID:         n.ID,
			Label:      n.Label,
			Shape:      string(n.Shape),
			Confidence: n.Confidence,
			URL:        n.URL,
			Metadata:   n.Metadata,
		}
	}
	edges := make([]flowEdgeJSON, len(f.Edges()))
	for i, e := range f.Edges() {
		edges[i] = flowEdgeJSON{
			From:  e.From,
			To:    e.To,
			Label: e.Label,
			Style: string(e.Style),
		}
	}
	sgs := make([]subgraphJSON, len(f.Subgraphs()))
	for i, sg := range f.Subgraphs() {
		sgs[i] = subgraphJSON{ID: sg.ID, Label: sg.Label, Nodes: sg.Nodes}
	}
	return flowchartJSON{
		Type:      string(diagram.TypeFlowchart),
		Title:     f.Title(),
		Direction: string(f.Direction()),
		Nodes:     nodes,
		Edges:     edges,
		Subgraphs: sgs,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Sequence
// ────────────────────────────────────────────────────────────────────────────

type sequenceJSON struct {
	Type         string             `json:"type"`
	Title        string             `json:"title,omitempty"`
	Autonumber   bool               `json:"autonumber,omitempty"`
	Participants []participantJSON  `json:"participants,omitempty"`
	Messages     []seqMessageJSON   `json:"messages,omitempty"`
	Notes        []seqNoteJSON      `json:"notes,omitempty"`
	Loops        []seqLoopJSON      `json:"loops,omitempty"`
	Alts         []seqAltJSON       `json:"alts,omitempty"`
}

type participantJSON struct {
	Alias string `json:"alias"`
	Label string `json:"label,omitempty"`
	Kind  string `json:"kind"`
}

type seqMessageJSON struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Text       string `json:"text,omitempty"`
	Style      string `json:"style"`
	Activate   bool   `json:"activate,omitempty"`
	Deactivate bool   `json:"deactivate,omitempty"`
}

type seqNoteJSON struct {
	Over []string `json:"over"`
	Text string   `json:"text"`
}

type seqLoopJSON struct {
	Label    string           `json:"label"`
	Messages []seqMessageJSON `json:"messages,omitempty"`
}

type seqAltJSON struct {
	Condition string           `json:"condition"`
	Messages  []seqMessageJSON `json:"messages,omitempty"`
	Else      []seqMessageJSON `json:"else,omitempty"`
}

func marshalSequence(s *ast.SequenceDiagram) sequenceJSON {
	ps := make([]participantJSON, len(s.Participants()))
	for i, p := range s.Participants() {
		ps[i] = participantJSON{Alias: p.Alias, Label: p.Label, Kind: string(p.Kind)}
	}
	msgs := marshalSeqMessages(s.Messages())
	notes := make([]seqNoteJSON, len(s.Notes()))
	for i, n := range s.Notes() {
		notes[i] = seqNoteJSON{Over: n.Over, Text: n.Text}
	}
	loops := make([]seqLoopJSON, len(s.Loops()))
	for i, l := range s.Loops() {
		loops[i] = seqLoopJSON{Label: l.Label, Messages: marshalSeqMessages(l.Messages)}
	}
	alts := make([]seqAltJSON, len(s.Alts()))
	for i, a := range s.Alts() {
		alts[i] = seqAltJSON{
			Condition: a.Condition,
			Messages:  marshalSeqMessages(a.Messages),
			Else:      marshalSeqMessages(a.Else),
		}
	}
	return sequenceJSON{
		Type:         string(diagram.TypeSequence),
		Title:        s.Title(),
		Autonumber:   s.Autonumber(),
		Participants: ps,
		Messages:     msgs,
		Notes:        notes,
		Loops:        loops,
		Alts:         alts,
	}
}

func marshalSeqMessages(msgs []ast.SeqMessage) []seqMessageJSON {
	out := make([]seqMessageJSON, len(msgs))
	for i, m := range msgs {
		out[i] = seqMessageJSON{
			From:       m.From,
			To:         m.To,
			Text:       m.Text,
			Style:      string(m.Style),
			Activate:   m.Activate,
			Deactivate: m.Deactivate,
		}
	}
	return out
}

// ────────────────────────────────────────────────────────────────────────────
// State
// ────────────────────────────────────────────────────────────────────────────

type stateJSON struct {
	Type        string             `json:"type"`
	Title       string             `json:"title,omitempty"`
	States      []diagramStateJSON `json:"states,omitempty"`
	Transitions []stateTransJSON   `json:"transitions,omitempty"`
	Notes       []stateNoteJSON    `json:"notes,omitempty"`
}

type diagramStateJSON struct {
	ID       string             `json:"id"`
	Label    string             `json:"label,omitempty"`
	Kind     string             `json:"kind"`
	Children []diagramStateJSON `json:"children,omitempty"`
}

type stateTransJSON struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Event string `json:"event,omitempty"`
}

type stateNoteJSON struct {
	State string `json:"state"`
	Text  string `json:"text"`
}

func marshalState(s *ast.StateDiagram) stateJSON {
	states := make([]diagramStateJSON, len(s.States()))
	for i, st := range s.States() {
		states[i] = marshalStateNode(st)
	}
	trans := make([]stateTransJSON, len(s.Transitions()))
	for i, t := range s.Transitions() {
		trans[i] = stateTransJSON{From: t.From, To: t.To, Event: t.Event}
	}
	notes := make([]stateNoteJSON, len(s.Notes()))
	for i, n := range s.Notes() {
		notes[i] = stateNoteJSON{State: n.State, Text: n.Text}
	}
	return stateJSON{
		Type:        string(diagram.TypeState),
		Title:       s.Title(),
		States:      states,
		Transitions: trans,
		Notes:       notes,
	}
}

func marshalStateNode(st ast.DiagramState) diagramStateJSON {
	children := make([]diagramStateJSON, len(st.Children))
	for i, c := range st.Children {
		children[i] = marshalStateNode(c)
	}
	return diagramStateJSON{
		ID:       st.ID,
		Label:    st.Label,
		Kind:     string(st.Kind),
		Children: children,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ER
// ────────────────────────────────────────────────────────────────────────────

type erJSON struct {
	Type      string         `json:"type"`
	Title     string         `json:"title,omitempty"`
	Entities  []erEntityJSON `json:"entities,omitempty"`
	Relations []erRelJSON    `json:"relations,omitempty"`
}

type erEntityJSON struct {
	Name       string        `json:"name"`
	Attributes []erAttrJSON  `json:"attributes,omitempty"`
}

type erAttrJSON struct {
	DataType string   `json:"dataType"`
	Name     string   `json:"name"`
	Keys     []string `json:"keys,omitempty"`
	Comment  string   `json:"comment,omitempty"`
}

type erRelJSON struct {
	From        string `json:"from"`
	To          string `json:"to"`
	FromCard    string `json:"fromCardinality"`
	ToCard      string `json:"toCardinality"`
	Label       string `json:"label,omitempty"`
	Identifying bool   `json:"identifying"`
}

func marshalER(e *ast.ERDiagram) erJSON {
	entities := make([]erEntityJSON, len(e.Entities()))
	for i, en := range e.Entities() {
		attrs := make([]erAttrJSON, len(en.Attributes))
		for j, a := range en.Attributes {
			keys := make([]string, len(a.Keys))
			for k, key := range a.Keys {
				keys[k] = string(key)
			}
			attrs[j] = erAttrJSON{DataType: a.DataType, Name: a.Name, Keys: keys, Comment: a.Comment}
		}
		entities[i] = erEntityJSON{Name: en.Name, Attributes: attrs}
	}
	rels := make([]erRelJSON, len(e.Relations()))
	for i, r := range e.Relations() {
		rels[i] = erRelJSON{
			From:        r.From,
			To:          r.To,
			FromCard:    string(r.FromCard),
			ToCard:      string(r.ToCard),
			Label:       r.Label,
			Identifying: r.Identifying,
		}
	}
	return erJSON{
		Type:      string(diagram.TypeER),
		Title:     e.Title(),
		Entities:  entities,
		Relations: rels,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Class
// ────────────────────────────────────────────────────────────────────────────

type classJSON struct {
	Type      string          `json:"type"`
	Title     string          `json:"title,omitempty"`
	Classes   []classNodeJSON `json:"classes,omitempty"`
	Relations []classRelJSON  `json:"relations,omitempty"`
	Notes     []classNoteJSON `json:"notes,omitempty"`
}

type classNodeJSON struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace,omitempty"`
	Annotation string            `json:"annotation,omitempty"`
	Members    []classMemberJSON `json:"members,omitempty"`
}

type classMemberJSON struct {
	Visibility string `json:"visibility,omitempty"`
	Type       string `json:"type,omitempty"`
	Name       string `json:"name"`
	IsMethod   bool   `json:"isMethod,omitempty"`
	IsStatic   bool   `json:"isStatic,omitempty"`
	IsAbstract bool   `json:"isAbstract,omitempty"`
}

type classRelJSON struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Label    string `json:"label,omitempty"`
	CardFrom string `json:"cardFrom,omitempty"`
	CardTo   string `json:"cardTo,omitempty"`
}

type classNoteJSON struct {
	Class string `json:"class"`
	Text  string `json:"text"`
}

func marshalClass(c *ast.ClassDiagram) classJSON {
	classes := make([]classNodeJSON, len(c.Classes()))
	for i, cl := range c.Classes() {
		members := make([]classMemberJSON, len(cl.Members))
		for j, m := range cl.Members {
			members[j] = classMemberJSON{
				Visibility: string(m.Visibility),
				Type:       m.Type,
				Name:       m.Name,
				IsMethod:   m.IsMethod,
				IsStatic:   m.IsStatic,
				IsAbstract: m.IsAbstract,
			}
		}
		classes[i] = classNodeJSON{
			Name:       cl.Name,
			Namespace:  cl.Namespace,
			Annotation: cl.Annotation,
			Members:    members,
		}
	}
	rels := make([]classRelJSON, len(c.Relations()))
	for i, r := range c.Relations() {
		rels[i] = classRelJSON{
			From:     r.From,
			To:       r.To,
			Kind:     string(r.Kind),
			Label:    r.Label,
			CardFrom: r.CardFrom,
			CardTo:   r.CardTo,
		}
	}
	notes := make([]classNoteJSON, len(c.Notes()))
	for i, n := range c.Notes() {
		notes[i] = classNoteJSON{Class: n.Class, Text: n.Text}
	}
	return classJSON{
		Type:      string(diagram.TypeClass),
		Title:     c.Title(),
		Classes:   classes,
		Relations: rels,
		Notes:     notes,
	}
}
