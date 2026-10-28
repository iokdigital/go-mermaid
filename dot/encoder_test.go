package dot_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/dot"
	"github.com/iokdigital/go-mermaid/mmd"
	"github.com/iokdigital/go-mermaid/render"
)

func encode(t *testing.T, d diagram.Diagram) string {
	t.Helper()
	var buf bytes.Buffer
	if err := dot.Encode(&buf, d); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	return buf.String()
}

func TestDOTFlowchartBasic(t *testing.T) {
	f := ast.NewFlowchart("My Graph", ast.DirectionLR)
	f.MustAddNode(&ast.FlowNode{ID: "A", Label: "Start", Shape: ast.ShapeRect})
	f.MustAddNode(&ast.FlowNode{ID: "B", Label: "End", Shape: ast.ShapeRect})
	f.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: "go", Style: ast.EdgeSolid})

	got := encode(t, f)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=LR") {
		t.Error("expected rankdir=LR")
	}
	if !strings.Contains(got, `label="My Graph"`) {
		t.Error("expected label attribute")
	}
	if !strings.Contains(got, "A ->") {
		t.Error("expected edge A ->")
	}
}

func TestDOTConfidenceFill(t *testing.T) {
	f := ast.NewFlowchart("", ast.DirectionTB)
	f.MustAddNode(&ast.FlowNode{ID: "hi", Confidence: 0.95})
	f.MustAddNode(&ast.FlowNode{ID: "mid", Confidence: 0.75})
	f.MustAddNode(&ast.FlowNode{ID: "lo", Confidence: 0.50})

	got := encode(t, f)
	if !strings.Contains(got, "#90EE90") {
		t.Error("expected high-confidence fill #90EE90")
	}
	if !strings.Contains(got, "#FFD700") {
		t.Error("expected mid-confidence fill #FFD700")
	}
	if !strings.Contains(got, "#FFB6C1") {
		t.Error("expected low-confidence fill #FFB6C1")
	}
}

func TestDOTUnsupportedTypeReturnsError(t *testing.T) {
	// Flowchart, sequence, state, ER, and class diagrams are supported; other
	// types (e.g. pie) are not.
	var buf bytes.Buffer
	err := dot.Encode(&buf, &fakePieDiagram{})
	if err == nil {
		t.Fatal("expected error for unsupported diagram type")
	}
	var ferr *diagram.FallbackFormatError
	if !errors.As(err, &ferr) {
		t.Fatalf("expected FallbackFormatError, got: %T %v", err, err)
	}
	if ferr.FallbackFormat() != diagram.FormatMMD {
		t.Errorf("expected FormatMMD fallback, got: %s", ferr.FallbackFormat())
	}
	if !errors.Is(err, diagram.ErrRendererNotAvailable) {
		t.Error("expected ErrRendererNotAvailable in error chain")
	}
}

// fakePieDiagram stands in for a genuinely unsupported diagram type (Phase 2
// types have no AST structs in this repo).
type fakePieDiagram struct{}

func (f *fakePieDiagram) Type() diagram.DiagramType { return diagram.TypePie }
func (f *fakePieDiagram) Title() string             { return "" }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Sequence ────────────────────────────────────────────────────────────────

func TestDOTSequenceBasic(t *testing.T) {
	s := ast.NewSequence("API Call", false)
	s.AddParticipant(ast.Participant{Alias: "Client", Kind: ast.ParticipantActor})
	s.AddParticipant(ast.Participant{Alias: "Server", Label: "API Server", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "Client", To: "Server", Text: "GET /jobs", Style: ast.MsgSync})
	s.AddMessage(ast.SeqMessage{From: "Server", To: "Client", Text: "200 OK", Style: ast.MsgAsync})

	got := encode(t, s)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=TB") {
		t.Error("expected rankdir=TB for top-down time axis")
	}
	if !strings.Contains(got, `label="API Call"`) {
		t.Error("expected title label attribute")
	}
	if !strings.Contains(got, "Client") || !strings.Contains(got, "Server") {
		t.Error("expected participant aliases in output")
	}
	if !strings.Contains(got, `label="API Server"`) {
		t.Error("expected participant display label in output")
	}
	if !strings.Contains(got, "shape=ellipse") {
		t.Error("expected actor participant to use ellipse shape")
	}
	if !strings.Contains(got, "Client -> Server") {
		t.Error("expected message edge Client -> Server")
	}
	if !strings.Contains(got, `label="GET /jobs"`) {
		t.Error("expected message text label")
	}
	if !strings.Contains(got, "style=dashed") {
		t.Error("expected async response to use dashed style")
	}
}

func TestDOTSequenceAllMessageStyles(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	styles := []struct {
		style ast.MessageStyle
		want  []string // attribute fragments expected on the edge
	}{
		{ast.MsgSync, []string{"A -> B"}},
		{ast.MsgAsync, []string{"style=dashed"}},
		{ast.MsgSyncNoArrow, []string{"arrowhead=none"}},
		{ast.MsgAsyncNoArrow, []string{"style=dashed", "arrowhead=none"}},
		{ast.MsgSyncX, []string{"arrowhead=tee"}},
		{ast.MsgAsyncX, []string{"style=dashed", "arrowhead=tee"}},
		{ast.MsgOpen, []string{"arrowhead=vee"}},
		{ast.MsgAsyncOpen, []string{"style=dashed", "arrowhead=vee"}},
	}
	for i, st := range styles {
		s.AddMessage(ast.SeqMessage{
			From: "A", To: "B",
			Text:  strings.Repeat("m", i+1), // unique label per edge
			Style: st.style,
		})
	}

	got := encode(t, s)
	for _, st := range styles {
		for _, want := range st.want {
			if !strings.Contains(got, want) {
				t.Errorf("style %q: expected %q in output:\n%s", string(st.style), want, got)
			}
		}
	}
}

func TestDOTSequenceAutonumber(t *testing.T) {
	s := ast.NewSequence("", true)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "first", Style: ast.MsgSync})
	s.AddMessage(ast.SeqMessage{From: "B", To: "A", Text: "second", Style: ast.MsgAsync})

	got := encode(t, s)
	if !strings.Contains(got, `label="1: first"`) {
		t.Errorf("expected numbered first message, got:\n%s", got)
	}
	if !strings.Contains(got, `label="2: second"`) {
		t.Errorf("expected numbered second message, got:\n%s", got)
	}
}

func TestDOTSequenceActivateDeactivate(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "call", Style: ast.MsgSync, Activate: true})
	s.AddMessage(ast.SeqMessage{From: "B", To: "A", Text: "return", Style: ast.MsgAsync, Deactivate: true})

	got := encode(t, s)
	if !strings.Contains(got, "[+]") {
		t.Errorf("expected activation marker [+], got:\n%s", got)
	}
	if !strings.Contains(got, "[-]") {
		t.Errorf("expected deactivation marker [-], got:\n%s", got)
	}
}

func TestDOTSequenceLoopAltNote(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddLoop(ast.SeqLoop{
		Label:    "Every minute",
		Messages: []ast.SeqMessage{{From: "A", To: "B", Text: "poll", Style: ast.MsgSync}},
	})
	s.AddAlt(ast.SeqAlt{
		Condition: "ok",
		Messages:  []ast.SeqMessage{{From: "B", To: "A", Text: "data", Style: ast.MsgAsync}},
		Else:      []ast.SeqMessage{{From: "B", To: "A", Text: "error", Style: ast.MsgAsync}},
	})
	s.AddNote(ast.SeqNote{Over: []string{"A", "B"}, Text: "retry window"})

	got := encode(t, s)
	for _, want := range []string{
		"cluster_seq_loop_0", `"loop: Every minute"`, `label="poll"`,
		"cluster_seq_alt_0", `"alt: ok"`, `label="data"`,
		"cluster_seq_alt_0_else", `label="else"`, `label="error"`,
		"shape=note", "retry window",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
}

// TestDOTSequenceRoundtrip encodes the same sequence AST to both DOT and MMD
// and checks the two outputs carry the same semantic content (participants,
// messages, blocks). There is no MMD parser in this repo, so the roundtrip is
// a dual-encoding consistency check rather than a decode comparison.
func TestDOTSequenceRoundtrip(t *testing.T) {
	s := ast.NewSequence("Checkout", true)
	s.AddParticipant(ast.Participant{Alias: "Client", Kind: ast.ParticipantActor})
	s.AddParticipant(ast.Participant{Alias: "Server", Label: "API Server", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "Client", To: "Server", Text: "POST /checkout", Style: ast.MsgSync, Activate: true})
	s.AddMessage(ast.SeqMessage{From: "Server", To: "Client", Text: "201 Created", Style: ast.MsgAsync, Deactivate: true})
	s.AddLoop(ast.SeqLoop{
		Label:    "poll status",
		Messages: []ast.SeqMessage{{From: "Client", To: "Server", Text: "GET /status", Style: ast.MsgSync}},
	})
	s.AddAlt(ast.SeqAlt{
		Condition: "ready",
		Messages:  []ast.SeqMessage{{From: "Server", To: "Client", Text: "done", Style: ast.MsgAsync}},
		Else:      []ast.SeqMessage{{From: "Server", To: "Client", Text: "pending", Style: ast.MsgAsync}},
	})
	s.AddNote(ast.SeqNote{Over: []string{"Client"}, Text: "client waits"})

	var dotBuf bytes.Buffer
	if err := dot.Encode(&dotBuf, s); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	var mmdBuf bytes.Buffer
	if err := mmd.Encode(&mmdBuf, s); err != nil {
		t.Fatalf("mmd.Encode: %v", err)
	}
	dotOut, mmdOut := dotBuf.String(), mmdBuf.String()

	if !strings.Contains(mmdOut, "sequenceDiagram") {
		t.Errorf("mmd output missing sequenceDiagram header:\n%s", mmdOut)
	}

	// Every semantic token must appear in both encodings.
	for _, token := range []string{
		"Client", "Server", "POST /checkout", "201 Created",
		"poll status", "GET /status", "ready", "done", "pending",
		"client waits", "Checkout",
	} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
		if !strings.Contains(mmdOut, token) {
			t.Errorf("mmd output missing %q:\n%s", token, mmdOut)
		}
	}

	// The render call-site must accept sequence diagrams for DOT output.
	r := render.NewRenderer(diagram.NewRenderOptions())
	data, err := r.RenderBytes(s, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("RenderBytes DOT: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "digraph") {
		t.Errorf("renderer DOT output invalid:\n%s", string(data))
	}
}

// ─── State ───────────────────────────────────────────────────────────────────

func TestDOTStateBasic(t *testing.T) {
	s := ast.NewState("Job Lifecycle")
	s.AddState(ast.DiagramState{ID: "Pending", Label: "Pending", Kind: ast.StateNormal})
	s.AddState(ast.DiagramState{ID: "Running", Label: "Running", Kind: ast.StateNormal})
	s.AddTransition(ast.StateTransition{From: "Pending", To: "Running", Event: "start"})
	s.AddTransition(ast.StateTransition{From: "Running", To: "Pending", Event: "retry"})

	got := encode(t, s)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=TB") {
		t.Error("expected rankdir=TB")
	}
	if !strings.Contains(got, `label="Job Lifecycle"`) {
		t.Error("expected title label attribute")
	}
	if !strings.Contains(got, `label="Pending"`) {
		t.Error("expected Pending state node")
	}
	if !strings.Contains(got, "Pending -> Running") {
		t.Error("expected transition edge Pending -> Running")
	}
	if !strings.Contains(got, `label="start"`) {
		t.Error("expected transition event label")
	}
}

func TestDOTStateKinds(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "s0", Kind: ast.StateStart})
	s.AddState(ast.DiagramState{ID: "e0", Kind: ast.StateEnd})
	s.AddState(ast.DiagramState{ID: "f0", Kind: ast.StateFork})
	s.AddState(ast.DiagramState{ID: "j0", Kind: ast.StateJoin})
	s.AddState(ast.DiagramState{ID: "c0", Label: "Decide", Kind: ast.StateChoice})

	got := encode(t, s)
	for _, want := range []string{
		"s0", "shape=circle style=filled",
		"e0", "shape=doublecircle",
		"f0", "fillcolor=black",
		"j0", "fillcolor=black",
		"c0", "shape=diamond",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
}

func TestDOTStatePseudoStartEnd(t *testing.T) {
	s := ast.NewState("")
	s.AddTransition(ast.StateTransition{From: "[*]", To: "Pending"})
	s.AddTransition(ast.StateTransition{From: "Pending", To: "Running", Event: "start"})
	s.AddTransition(ast.StateTransition{From: "Running", To: "[*]"})

	got := encode(t, s)
	for _, want := range []string{
		`__start [label="" shape=circle style=filled`,
		`__end [label="" shape=doublecircle`,
		"__start -> Pending",
		"Running -> __end",
		// Transition-only states are emitted so every endpoint exists.
		`"Pending" shape=box`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[*]") {
		t.Errorf("raw [*] must not leak into DOT output:\n%s", got)
	}
}

func TestDOTStateCompositeAndNote(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{
		ID:    "Active",
		Label: "Active",
		Kind:  ast.StateNormal,
		Children: []ast.DiagramState{
			{ID: "S1", Label: "Sub1", Kind: ast.StateNormal},
			{ID: "S2", Label: "Sub2", Kind: ast.StateNormal},
		},
	})
	s.AddTransition(ast.StateTransition{From: "S1", To: "S2", Event: "next"})
	s.AddNote(ast.StateNote{State: "S1", Text: "important"})

	got := encode(t, s)
	for _, want := range []string{
		"cluster_Active", `"Active"`,
		"S1", "S2", `label="next"`,
		"shape=note", "important", "style=dotted",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
}

// TestDOTStateRoundtrip encodes the same state AST to both DOT and MMD
// and checks the two outputs carry the same semantic content (states,
// transitions, notes). There is no MMD parser in this repo, so the roundtrip
// is a dual-encoding consistency check rather than a decode comparison.
func TestDOTStateRoundtrip(t *testing.T) {
	s := ast.NewState("Checkout")
	s.AddState(ast.DiagramState{ID: "Pending", Label: "Pending", Kind: ast.StateNormal})
	s.AddState(ast.DiagramState{
		ID:    "Active",
		Label: "Active",
		Kind:  ast.StateNormal,
		Children: []ast.DiagramState{
			{ID: "Running", Label: "Running", Kind: ast.StateNormal},
		},
	})
	s.AddTransition(ast.StateTransition{From: "[*]", To: "Pending"})
	s.AddTransition(ast.StateTransition{From: "Pending", To: "Running", Event: "start"})
	s.AddTransition(ast.StateTransition{From: "Running", To: "[*]"})
	s.AddNote(ast.StateNote{State: "Pending", Text: "waits for payment"})

	var dotBuf bytes.Buffer
	if err := dot.Encode(&dotBuf, s); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	var mmdBuf bytes.Buffer
	if err := mmd.Encode(&mmdBuf, s); err != nil {
		t.Fatalf("mmd.Encode: %v", err)
	}
	dotOut, mmdOut := dotBuf.String(), mmdBuf.String()

	if !strings.Contains(mmdOut, "stateDiagram-v2") {
		t.Errorf("mmd output missing stateDiagram-v2 header:\n%s", mmdOut)
	}

	// Every semantic token must appear in both encodings.
	for _, token := range []string{
		"Pending", "Running", "Active", "start",
		"waits for payment", "Checkout",
	} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
		if !strings.Contains(mmdOut, token) {
			t.Errorf("mmd output missing %q:\n%s", token, mmdOut)
		}
	}

	// The render call-site must accept state diagrams for DOT output.
	r := render.NewRenderer(diagram.NewRenderOptions())
	data, err := r.RenderBytes(s, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("RenderBytes DOT: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "digraph") {
		t.Errorf("renderer DOT output invalid:\n%s", string(data))
	}
}

// ─── ER ──────────────────────────────────────────────────────────────────────

func TestDOTERBasic(t *testing.T) {
	e := ast.NewER("Shop Schema")
	e.AddEntity(ast.EREntity{
		Name: "users",
		Attributes: []ast.ERAttribute{
			{DataType: "uuid", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "varchar", Name: "email"},
		},
	})
	e.AddEntity(ast.EREntity{Name: "orders"})
	e.AddRelation(ast.ERRelation{
		From: "users", To: "orders",
		FromCard: ast.CardOneMany, ToCard: ast.CardZeroOne,
		Label: "places", Identifying: true,
	})

	got := encode(t, e)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=TB") {
		t.Error("expected rankdir=TB")
	}
	if !strings.Contains(got, `label="Shop Schema"`) {
		t.Error("expected title label attribute")
	}
	for _, want := range []string{
		"shape=record", "users", "orders",
		"uuid id PK", "varchar email",
		"users -> orders", `label="places"`,
		"arrowhead=none",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	// Identifying relations render solid: no dashed style on the edge.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "users -> orders") && strings.Contains(line, "style=dashed") {
			t.Errorf("identifying relation must not be dashed:\n%s", line)
		}
	}
}

func TestDOTERRelationships(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "users"})
	e.AddEntity(ast.EREntity{Name: "orders"})
	e.AddEntity(ast.EREntity{Name: "items"})
	e.AddRelation(ast.ERRelation{
		From: "users", To: "orders",
		FromCard: ast.CardOneMany, ToCard: ast.CardZeroOne,
		Label: "places", Identifying: true,
	})
	e.AddRelation(ast.ERRelation{
		From: "orders", To: "items",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroMany,
		Label: "contains", Identifying: false,
	})
	// Relation-only entity: emitted so every edge endpoint exists.
	e.AddRelation(ast.ERRelation{
		From: "items", To: "ghost",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroOne,
		Label: "haunts", Identifying: true,
	})

	got := encode(t, e)
	for _, want := range []string{
		`taillabel="}|"`, `headlabel="|o"`,
		`taillabel="||"`, `headlabel="}o"`,
		`label="contains"`, "style=dashed",
		`"ghost" shape=box`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
}

func TestDOTERRecordEscaping(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{
		Name: "we|ird",
		Attributes: []ast.ERAttribute{
			{DataType: "str{ing}", Name: "a<b>c"},
		},
	})

	got := encode(t, e)
	for _, want := range []string{`we\|ird`, `str\{ing\}`, `a\<b\>c`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected escaped %q in output:\n%s", want, got)
		}
	}
}

// TestDOTERRoundtrip encodes the same ER AST to both DOT and MMD
// and checks the two outputs carry the same semantic content (entities,
// attributes, relations, cardinalities). There is no MMD parser in this repo,
// so the roundtrip is a dual-encoding consistency check rather than a decode
// comparison.
func TestDOTERRoundtrip(t *testing.T) {
	e := ast.NewER("Shop Schema")
	e.AddEntity(ast.EREntity{
		Name: "users",
		Attributes: []ast.ERAttribute{
			{DataType: "uuid", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "varchar", Name: "email", Keys: []ast.ERKey{ast.KeyUnique}},
		},
	})
	e.AddEntity(ast.EREntity{
		Name: "orders",
		Attributes: []ast.ERAttribute{
			{DataType: "uuid", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "uuid", Name: "user_id", Keys: []ast.ERKey{ast.KeyForeign}},
		},
	})
	e.AddRelation(ast.ERRelation{
		From: "users", To: "orders",
		FromCard: ast.CardOneMany, ToCard: ast.CardZeroOne,
		Label: "places", Identifying: true,
	})

	var dotBuf bytes.Buffer
	if err := dot.Encode(&dotBuf, e); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	var mmdBuf bytes.Buffer
	if err := mmd.Encode(&mmdBuf, e); err != nil {
		t.Fatalf("mmd.Encode: %v", err)
	}
	dotOut, mmdOut := dotBuf.String(), mmdBuf.String()

	if !strings.Contains(mmdOut, "erDiagram") {
		t.Errorf("mmd output missing erDiagram header:\n%s", mmdOut)
	}

	// Every semantic token must appear in both encodings.
	for _, token := range []string{
		"users", "orders", "uuid id PK", "varchar email UK",
		"uuid user_id FK", "places", "Shop Schema",
	} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
		if !strings.Contains(mmdOut, token) {
			t.Errorf("mmd output missing %q:\n%s", token, mmdOut)
		}
	}
	// Cardinality symbols appear verbatim in DOT taillabel/headlabel.
	for _, token := range []string{`taillabel="}|"`, `headlabel="|o"`} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
	}

	// The render call-site must accept ER diagrams for DOT output.
	r := render.NewRenderer(diagram.NewRenderOptions())
	data, err := r.RenderBytes(e, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("RenderBytes DOT: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "digraph") {
		t.Errorf("renderer DOT output invalid:\n%s", string(data))
	}
}

// ─── Class ───────────────────────────────────────────────────────────────────

func TestDOTClassBasic(t *testing.T) {
	c := ast.NewClass("Zoo Model")
	c.AddClass(ast.DiagramClass{
		Name: "Animal",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "name"},
			{Visibility: ast.VisPrivate, Type: "int", Name: "age"},
			{Visibility: ast.VisPublic, Type: "void", Name: "speak", IsMethod: true},
			{Visibility: ast.VisPublic, Type: "void", Name: "run", IsMethod: true, IsStatic: true},
		},
	})
	c.AddClass(ast.DiagramClass{Name: "Zoo"})

	got := encode(t, c)
	if !strings.HasPrefix(got, "digraph {") {
		t.Errorf("expected digraph open, got: %q", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "rankdir=TB") {
		t.Error("expected rankdir=TB")
	}
	if !strings.Contains(got, `label="Zoo Model"`) {
		t.Error("expected title label attribute")
	}
	for _, want := range []string{
		"shape=record", "Animal", "{Zoo}",
		"+string name", "-int age",
		"+void speak()", "+void run()$",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	// Attributes and methods live in separate record compartments.
	animalLine := ""
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "Animal [label=") {
			animalLine = line
		}
	}
	if animalLine == "" {
		t.Fatalf("expected Animal record node:\n%s", got)
	}
	attrIdx := strings.Index(animalLine, "+string name")
	methIdx := strings.Index(animalLine, "+void speak()")
	sepIdx := strings.Index(animalLine, "|+void speak()")
	if attrIdx < 0 || methIdx < 0 || attrIdx > methIdx {
		t.Errorf("expected attributes before methods in record node:\n%s", animalLine)
	}
	if sepIdx < 0 {
		t.Errorf("expected compartment separator before methods:\n%s", animalLine)
	}
}

func TestDOTClassInheritance(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name:       "Animal",
		Annotation: "<<abstract>>",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "void", Name: "speak", IsMethod: true, IsAbstract: true},
		},
	})
	c.AddClass(ast.DiagramClass{Name: "Dog"})
	c.AddClass(ast.DiagramClass{Name: "Flyable", Annotation: "<<interface>>"})
	c.AddClass(ast.DiagramClass{Name: "Bird"})
	c.AddRelation(ast.ClassRelation{From: "Animal", To: "Dog", Kind: ast.RelInheritance, Label: "extends"})
	c.AddRelation(ast.ClassRelation{From: "Flyable", To: "Bird", Kind: ast.RelRealization, Label: "implements"})

	got := encode(t, c)
	for _, want := range []string{
		`\<\<abstract\>\>`, `\<\<interface\>\>`,
		"+void speak()*",
		"Animal -> Dog", "Flyable -> Bird",
		`label="extends"`, `label="implements"`,
		"arrowhead=empty",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	// Inheritance renders solid; only realization is dashed.
	for _, line := range strings.Split(got, "\n") {
		switch {
		case strings.Contains(line, "Animal -> Dog"):
			if strings.Contains(line, "style=dashed") {
				t.Errorf("inheritance must not be dashed:\n%s", line)
			}
		case strings.Contains(line, "Flyable -> Bird"):
			if !strings.Contains(line, "style=dashed") {
				t.Errorf("realization must be dashed:\n%s", line)
			}
		}
	}
}

func TestDOTClassAssociations(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Car"})
	c.AddClass(ast.DiagramClass{Name: "Engine"})
	c.AddClass(ast.DiagramClass{Name: "Team"})
	c.AddClass(ast.DiagramClass{Name: "Player"})
	c.AddRelation(ast.ClassRelation{From: "Car", To: "Engine", Kind: ast.RelComposition, Label: "has"})
	c.AddRelation(ast.ClassRelation{From: "Team", To: "Player", Kind: ast.RelAggregation, Label: "has"})
	c.AddRelation(ast.ClassRelation{
		From: "Player", To: "Team",
		Kind: ast.RelAssociation, Label: "plays for",
		CardFrom: "*", CardTo: "1",
	})
	c.AddRelation(ast.ClassRelation{From: "Car", To: "Team", Kind: ast.RelDependency, Label: "uses"})
	c.AddRelation(ast.ClassRelation{From: "Engine", To: "Player", Kind: ast.RelLink})
	// Relation-only class: emitted so every edge endpoint exists.
	c.AddRelation(ast.ClassRelation{From: "Player", To: "Ghost", Kind: ast.RelAssociation})

	got := encode(t, c)
	for _, want := range []string{
		"Car -> Engine", "arrowtail=diamond", "dir=both",
		"Team -> Player", "arrowtail=odiamond",
		"Player -> Team", `label="plays for"`, `taillabel="*"`, `headlabel="1"`,
		"Car -> Team", "Engine -> Player",
		`"Ghost" shape=box`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		switch {
		case strings.Contains(line, "Car -> Engine") || strings.Contains(line, "Team -> Player"):
			if !strings.Contains(line, "arrowhead=none") {
				t.Errorf("composition/aggregation must suppress the head arrow:\n%s", line)
			}
		case strings.Contains(line, "Car -> Team") && strings.Contains(line, "->"):
			if !strings.Contains(line, "style=dashed") {
				t.Errorf("dependency must be dashed:\n%s", line)
			}
		case strings.Contains(line, "Engine -> Player"):
			if !strings.Contains(line, "arrowhead=none") {
				t.Errorf("plain link must have no arrowhead:\n%s", line)
			}
		case strings.Contains(line, "Player -> Team"):
			if strings.Contains(line, "arrowhead=none") {
				t.Errorf("association must keep the default arrow:\n%s", line)
			}
		}
	}
}

func TestDOTClassRecordEscaping(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name: "we|ird",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "str{ing}", Name: "a<b>c"},
			{Visibility: ast.VisPublic, Type: "void", Name: "m|e", IsMethod: true},
		},
	})

	got := encode(t, c)
	for _, want := range []string{`we\|ird`, `str\{ing\}`, `a\<b\>c`, `m\|e()`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected escaped %q in output:\n%s", want, got)
		}
	}
}

// TestDOTClassRoundtrip encodes the same class AST to both DOT and MMD
// and checks the two outputs carry the same semantic content (classes,
// members, relations, cardinalities, notes). There is no MMD parser in this
// repo, so the roundtrip is a dual-encoding consistency check rather than a
// decode comparison.
func TestDOTClassRoundtrip(t *testing.T) {
	c := ast.NewClass("Zoo Model")
	c.AddClass(ast.DiagramClass{
		Name:       "Animal",
		Annotation: "<<abstract>>",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "name"},
			{Visibility: ast.VisPublic, Type: "void", Name: "speak", IsMethod: true, IsAbstract: true},
		},
	})
	c.AddClass(ast.DiagramClass{
		Name: "Dog",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "void", Name: "fetch", IsMethod: true},
		},
	})
	c.AddRelation(ast.ClassRelation{From: "Animal", To: "Dog", Kind: ast.RelInheritance, Label: "extends"})
	c.AddRelation(ast.ClassRelation{
		From: "Dog", To: "Animal",
		Kind: ast.RelAssociation, Label: "pack",
		CardFrom: "1", CardTo: "*",
	})
	c.AddNote(ast.ClassNote{Class: "Dog", Text: "good boy"})

	var dotBuf bytes.Buffer
	if err := dot.Encode(&dotBuf, c); err != nil {
		t.Fatalf("dot.Encode: %v", err)
	}
	var mmdBuf bytes.Buffer
	if err := mmd.Encode(&mmdBuf, c); err != nil {
		t.Fatalf("mmd.Encode: %v", err)
	}
	dotOut, mmdOut := dotBuf.String(), mmdBuf.String()

	if !strings.Contains(mmdOut, "classDiagram") {
		t.Errorf("mmd output missing classDiagram header:\n%s", mmdOut)
	}

	// Every semantic token must appear in both encodings.
	for _, token := range []string{
		"Animal", "Dog", "+string name", "+void speak()*",
		"+void fetch()", "extends", "pack", "good boy", "Zoo Model",
	} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
		if !strings.Contains(mmdOut, token) {
			t.Errorf("mmd output missing %q:\n%s", token, mmdOut)
		}
	}
	// Inheritance uses the hollow-triangle arrowhead; cardinalities are
	// preserved verbatim in taillabel/headlabel.
	for _, token := range []string{"arrowhead=empty", `taillabel="1"`, `headlabel="*"`, "shape=note"} {
		if !strings.Contains(dotOut, token) {
			t.Errorf("dot output missing %q:\n%s", token, dotOut)
		}
	}

	// The render call-site must accept class diagrams for DOT output.
	r := render.NewRenderer(diagram.NewRenderOptions())
	data, err := r.RenderBytes(c, diagram.FormatDOT)
	if err != nil {
		t.Fatalf("RenderBytes DOT: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "digraph") {
		t.Errorf("renderer DOT output invalid:\n%s", string(data))
	}
}
