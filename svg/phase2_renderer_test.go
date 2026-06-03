package svg_test

import (
	"bytes"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/svg"
)

func encode2(t *testing.T, d diagram.Diagram) string {
	t.Helper()
	var buf bytes.Buffer
	if err := svg.Encode(&buf, d, diagram.NewRenderOptions()); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// ─── State diagram tests ────────────────────────────────────────────────────

func TestState_SVGDoctype(t *testing.T) {
	s := ast.NewState("test")
	s.AddState(ast.DiagramState{ID: "A", Kind: ast.StateNormal})
	out := encode2(t, s)
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(out, "<svg ") {
		t.Error("missing svg element")
	}
}

func TestState_TitleInOutput(t *testing.T) {
	s := ast.NewState("My State Diagram")
	out := encode2(t, s)
	if !strings.Contains(out, "My State Diagram") {
		t.Error("title not found in SVG output")
	}
}

func TestState_NormalStateRendered(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "Idle", Kind: ast.StateNormal})
	out := encode2(t, s)
	if !strings.Contains(out, "Idle") {
		t.Error("state ID not found in SVG output")
	}
}

func TestState_StartEndRendered(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "start1", Kind: ast.StateStart})
	s.AddState(ast.DiagramState{ID: "end1", Kind: ast.StateEnd})
	out := encode2(t, s)
	// Both should produce ellipse elements (filled circle / double circle).
	count := strings.Count(out, "<ellipse")
	if count < 2 {
		t.Errorf("expected at least 2 ellipse elements for start+end, got %d", count)
	}
}

func TestState_ForkJoinRendered(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "fk", Kind: ast.StateFork})
	out := encode2(t, s)
	// Fork/join is a rect.
	if !strings.Contains(out, "<rect") {
		t.Error("fork node should render as rect")
	}
}

func TestState_TransitionArrow(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "A", Kind: ast.StateNormal})
	s.AddState(ast.DiagramState{ID: "B", Kind: ast.StateNormal})
	s.AddTransition(ast.StateTransition{From: "A", To: "B", Event: "click"})
	out := encode2(t, s)
	// Should contain a path (arrow) and the event label.
	if !strings.Contains(out, `<path`) {
		t.Error("transition should produce a path element")
	}
	if !strings.Contains(out, "click") {
		t.Error("transition event label not found in output")
	}
}

func TestState_NoteRendered(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "A", Kind: ast.StateNormal})
	s.AddNote(ast.StateNote{State: "A", Text: "important note"})
	out := encode2(t, s)
	if !strings.Contains(out, "important note") {
		t.Error("note text not found in SVG output")
	}
}

func TestState_CompositeStateRendered(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{
		ID:   "Outer",
		Kind: ast.StateNormal,
		Children: []ast.DiagramState{
			{ID: "Inner1", Kind: ast.StateNormal},
			{ID: "Inner2", Kind: ast.StateNormal},
		},
	})
	out := encode2(t, s)
	if !strings.Contains(out, "Outer") {
		t.Error("composite state label not found")
	}
	if !strings.Contains(out, "Inner1") {
		t.Error("child state Inner1 not found in composite rendering")
	}
}

func TestState_XSSTitleEscaped(t *testing.T) {
	s := ast.NewState("<script>alert(1)</script>")
	out := encode2(t, s)
	if strings.Contains(out, "<script>") {
		t.Error("XSS in title not escaped")
	}
}

// ─── ER diagram tests ────────────────────────────────────────────────────────

func TestER_SVGDoctype(t *testing.T) {
	e := ast.NewER("test ER")
	out := encode2(t, e)
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
}

func TestER_EntityHeaderRendered(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{
		Name: "Customer",
		Attributes: []ast.ERAttribute{
			{DataType: "int", Name: "id", Keys: []ast.ERKey{ast.KeyPrimary}},
			{DataType: "varchar", Name: "name"},
		},
	})
	out := encode2(t, e)
	if !strings.Contains(out, "Customer") {
		t.Error("entity name not found in SVG output")
	}
	if !strings.Contains(out, "PK") {
		t.Error("PK badge not found in SVG output")
	}
	if !strings.Contains(out, "int id") {
		t.Error("attribute text not found in SVG output")
	}
}

func TestER_MultipleEntities(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "Order"})
	e.AddEntity(ast.EREntity{Name: "Product"})
	out := encode2(t, e)
	if !strings.Contains(out, "Order") {
		t.Error("Order entity not found")
	}
	if !strings.Contains(out, "Product") {
		t.Error("Product entity not found")
	}
}

func TestER_RelationLinePresent(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "A"})
	e.AddEntity(ast.EREntity{Name: "B"})
	e.AddRelation(ast.ERRelation{
		From: "A", To: "B",
		FromCard: ast.CardExactOne, ToCard: ast.CardZeroMany,
		Label: "has",
	})
	out := encode2(t, e)
	if !strings.Contains(out, `<path`) {
		t.Error("relation should produce a path element")
	}
	if !strings.Contains(out, "has") {
		t.Error("relation label not found in SVG output")
	}
}

func TestER_IdentifyingVsNonIdentifying(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "A"})
	e.AddEntity(ast.EREntity{Name: "B"})
	// Non-identifying: dashed line.
	e.AddRelation(ast.ERRelation{From: "A", To: "B", Identifying: false})
	out := encode2(t, e)
	if !strings.Contains(out, "stroke-dasharray") {
		t.Error("non-identifying relation should use dashed line")
	}
}

func TestER_XSSTitleEscaped(t *testing.T) {
	e := ast.NewER("<b>bad</b>")
	out := encode2(t, e)
	if strings.Contains(out, "<b>") {
		t.Error("XSS in ER title not escaped")
	}
}

// ─── Class diagram tests ─────────────────────────────────────────────────────

func TestClass_SVGDoctype(t *testing.T) {
	c := ast.NewClass("test Class")
	out := encode2(t, c)
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
}

func TestClass_ClassNameRendered(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name: "Animal",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "String", Name: "name"},
			{Visibility: ast.VisPublic, Name: "speak", IsMethod: true},
		},
	})
	out := encode2(t, c)
	if !strings.Contains(out, "Animal") {
		t.Error("class name not found in output")
	}
	if !strings.Contains(out, "speak()") {
		t.Error("method not found in output")
	}
}

func TestClass_AnnotationRendered(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name:       "Flyable",
		Annotation: "<<interface>>",
	})
	out := encode2(t, c)
	if !strings.Contains(out, "interface") {
		t.Error("annotation text not found in output")
	}
}

func TestClass_RelationLinePresent(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Animal"})
	c.AddClass(ast.DiagramClass{Name: "Dog"})
	c.AddRelation(ast.ClassRelation{
		From:  "Animal",
		To:    "Dog",
		Kind:  ast.RelInheritance,
		Label: "is-a",
	})
	out := encode2(t, c)
	if !strings.Contains(out, `<path`) {
		t.Error("relation should produce a path element")
	}
	if !strings.Contains(out, "is-a") {
		t.Error("relation label not found")
	}
}

func TestClass_InheritanceOpenTriangle(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Base"})
	c.AddClass(ast.DiagramClass{Name: "Derived"})
	c.AddRelation(ast.ClassRelation{From: "Base", To: "Derived", Kind: ast.RelInheritance})
	out := encode2(t, c)
	// Open triangle is a polygon with fill="white".
	if !strings.Contains(out, `fill="white"`) {
		t.Error("inheritance should use open (white-filled) triangle arrowhead")
	}
}

func TestClass_CompositionFilledDiamond(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Car"})
	c.AddClass(ast.DiagramClass{Name: "Engine"})
	c.AddRelation(ast.ClassRelation{From: "Car", To: "Engine", Kind: ast.RelComposition})
	out := encode2(t, c)
	// Filled diamond is a polygon with fill equal to stroke color.
	if !strings.Contains(out, `<polygon`) {
		t.Error("composition should produce polygon (diamond) decorator")
	}
}

func TestClass_RealizationDashedLine(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Flyable"})
	c.AddClass(ast.DiagramClass{Name: "Bird"})
	c.AddRelation(ast.ClassRelation{From: "Flyable", To: "Bird", Kind: ast.RelRealization})
	out := encode2(t, c)
	if !strings.Contains(out, "stroke-dasharray") {
		t.Error("realization should use dashed line")
	}
}

func TestClass_NamespaceBox(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Foo", Namespace: "pkg"})
	c.AddClass(ast.DiagramClass{Name: "Bar", Namespace: "pkg"})
	out := encode2(t, c)
	if !strings.Contains(out, "namespace pkg") {
		t.Error("namespace label not found in output")
	}
}

func TestClass_NoteRendered(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "Foo"})
	c.AddNote(ast.ClassNote{Class: "Foo", Text: "class note here"})
	out := encode2(t, c)
	if !strings.Contains(out, "class note here") {
		t.Error("class note not found in output")
	}
}

// ─── Sequence diagram tests ──────────────────────────────────────────────────

func TestSequence_SVGDoctype(t *testing.T) {
	s := ast.NewSequence("test", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	out := encode2(t, s)
	if !strings.Contains(out, `<?xml version="1.0"`) {
		t.Error("missing XML declaration")
	}
}

func TestSequence_ParticipantRendered(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "Alice", Label: "Alice", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "Bob", Label: "Bob", Kind: ast.ParticipantBox})
	out := encode2(t, s)
	if !strings.Contains(out, "Alice") {
		t.Error("Alice participant not found in output")
	}
	if !strings.Contains(out, "Bob") {
		t.Error("Bob participant not found in output")
	}
}

func TestSequence_LifelinePresent(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	out := encode2(t, s)
	// Lifeline is a dashed vertical line.
	if !strings.Contains(out, "stroke-dasharray") {
		t.Error("lifeline dashed line not found in output")
	}
}

func TestSequence_MessageArrow(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "hello", Style: ast.MsgSync})
	out := encode2(t, s)
	if !strings.Contains(out, "hello") {
		t.Error("message label not found in output")
	}
	if !strings.Contains(out, `<polygon`) {
		t.Error("message arrowhead polygon not found")
	}
}

func TestSequence_AsyncDashedLine(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "resp", Style: ast.MsgAsync})
	out := encode2(t, s)
	if !strings.Contains(out, "stroke-dasharray") {
		t.Error("async message should use dashed line")
	}
}

func TestSequence_AutonumberPrefix(t *testing.T) {
	s := ast.NewSequence("", true) // autonumber=true
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "go", Style: ast.MsgSync})
	out := encode2(t, s)
	if !strings.Contains(out, "1: go") {
		t.Error("autonumber prefix not found in message label")
	}
}

func TestSequence_NoteBox(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddNote(ast.SeqNote{Over: []string{"A"}, Text: "side note"})
	out := encode2(t, s)
	if !strings.Contains(out, "side note") {
		t.Error("sequence note text not found in output")
	}
}

func TestSequence_LoopFrame(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddLoop(ast.SeqLoop{
		Label: "retry",
		Messages: []ast.SeqMessage{
			{From: "A", To: "B", Text: "ping", Style: ast.MsgSync},
		},
	})
	out := encode2(t, s)
	if !strings.Contains(out, "loop") {
		t.Error("loop frame label not found")
	}
	if !strings.Contains(out, "retry") {
		t.Error("loop condition not found")
	}
	if !strings.Contains(out, "ping") {
		t.Error("loop message not found")
	}
}

func TestSequence_AltFrame(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Kind: ast.ParticipantBox})
	s.AddParticipant(ast.Participant{Alias: "B", Kind: ast.ParticipantBox})
	s.AddAlt(ast.SeqAlt{
		Condition: "success",
		Messages:  []ast.SeqMessage{{From: "A", To: "B", Text: "ok", Style: ast.MsgSync}},
		Else:      []ast.SeqMessage{{From: "A", To: "B", Text: "fail", Style: ast.MsgSync}},
	})
	out := encode2(t, s)
	if !strings.Contains(out, "alt") {
		t.Error("alt frame label not found")
	}
	if !strings.Contains(out, "success") {
		t.Error("alt condition not found")
	}
	if !strings.Contains(out, "else") {
		t.Error("alt else divider not found")
	}
	if !strings.Contains(out, "fail") {
		t.Error("alt else message not found")
	}
}

func TestSequence_ActorParticipant(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "User", Label: "User", Kind: ast.ParticipantActor})
	out := encode2(t, s)
	// Actor has green styling (#f0fdf4 fill).
	if !strings.Contains(out, "#f0fdf4") {
		t.Error("actor participant fill color not found")
	}
}

func TestSequence_TitleRendered(t *testing.T) {
	s := ast.NewSequence("Login Flow", false)
	out := encode2(t, s)
	if !strings.Contains(out, "Login Flow") {
		t.Error("sequence diagram title not found in output")
	}
}

func TestSequence_XSSEscaped(t *testing.T) {
	s := ast.NewSequence("<script>bad</script>", false)
	out := encode2(t, s)
	if strings.Contains(out, "<script>") {
		t.Error("XSS in sequence title not escaped")
	}
}
