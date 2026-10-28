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

func TestDOTNonFlowchartReturnsError(t *testing.T) {
	// Sequence diagrams are supported; state diagrams are not.
	s := ast.NewState("unsupported")
	var buf bytes.Buffer
	err := dot.Encode(&buf, s)
	if err == nil {
		t.Fatal("expected error for non-flowchart/non-sequence type")
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
