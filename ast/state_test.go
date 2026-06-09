package ast_test

import (
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewState(t *testing.T) {
	s := ast.NewState("Test State")
	if s.Title() != "Test State" {
		t.Errorf("expected title %q, got %q", "Test State", s.Title())
	}
	if s.Type() != diagram.TypeState {
		t.Errorf("expected type %q, got %q", diagram.TypeState, s.Type())
	}
}

func TestStateAddState(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "A", Label: "State A", Kind: ast.StateNormal})

	if len(s.States()) != 1 {
		t.Fatalf("expected 1 state, got %d", len(s.States()))
	}
	if s.States()[0].ID != "A" {
		t.Errorf("expected state ID %q, got %q", "A", s.States()[0].ID)
	}
}

func TestStateKinds(t *testing.T) {
	s := ast.NewState("")
	kinds := []ast.StateKind{
		ast.StateNormal, ast.StateStart, ast.StateEnd,
		ast.StateFork, ast.StateJoin, ast.StateChoice,
	}

	for _, kind := range kinds {
		s.AddState(ast.DiagramState{ID: string(kind), Kind: kind})
	}

	if len(s.States()) != len(kinds) {
		t.Fatalf("expected %d states, got %d", len(kinds), len(s.States()))
	}
}

func TestStateAddTransition(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "A"})
	s.AddState(ast.DiagramState{ID: "B"})
	s.AddTransition(ast.StateTransition{From: "A", To: "B", Event: "transition"})

	if len(s.Transitions()) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(s.Transitions()))
	}
	tr := s.Transitions()[0]
	if tr.From != "A" || tr.To != "B" {
		t.Errorf("expected transition A->B, got %s->%s", tr.From, tr.To)
	}
	if tr.Event != "transition" {
		t.Errorf("expected event %q, got %q", "transition", tr.Event)
	}
}

func TestStateAddNote(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{ID: "A"})
	s.AddNote(ast.StateNote{State: "A", Text: "Note"})

	if len(s.Notes()) != 1 {
		t.Fatalf("expected 1 note, got %d", len(s.Notes()))
	}
	if s.Notes()[0].Text != "Note" {
		t.Errorf("expected text %q, got %q", "Note", s.Notes()[0].Text)
	}
}

func TestStateComposite(t *testing.T) {
	s := ast.NewState("")
	s.AddState(ast.DiagramState{
		ID:       "Composite",
		Label:    "Composite State",
		Kind:     ast.StateNormal,
		Children: []ast.DiagramState{
			{ID: "S1", Label: "Sub1", Kind: ast.StateNormal},
			{ID: "S2", Label: "Sub2", Kind: ast.StateNormal},
		},
	})

	if len(s.States()) != 1 {
		t.Fatalf("expected 1 state, got %d", len(s.States()))
	}
	if len(s.States()[0].Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(s.States()[0].Children))
	}
}

func TestStateEmpty(t *testing.T) {
	s := ast.NewState("")
	if len(s.States()) != 0 {
		t.Errorf("expected 0 states, got %d", len(s.States()))
	}
	if len(s.Transitions()) != 0 {
		t.Errorf("expected 0 transitions, got %d", len(s.Transitions()))
	}
	if len(s.Notes()) != 0 {
		t.Errorf("expected 0 notes, got %d", len(s.Notes()))
	}
}