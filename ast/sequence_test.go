package ast_test

import (
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewSequence(t *testing.T) {
	s := ast.NewSequence("Test Sequence", true)
	if s.Title() != "Test Sequence" {
		t.Errorf("expected title %q, got %q", "Test Sequence", s.Title())
	}
	if !s.Autonumber() {
		t.Error("expected autonumber to be true")
	}
	if s.Type() != diagram.TypeSequence {
		t.Errorf("expected type %q, got %q", diagram.TypeSequence, s.Type())
	}
}

func TestNewSequenceNoAutonumber(t *testing.T) {
	s := ast.NewSequence("", false)
	if s.Autonumber() {
		t.Error("expected autonumber to be false")
	}
}

func TestSequenceAddParticipant(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A", Label: "Alice"})
	s.AddParticipant(ast.Participant{Alias: "B", Label: "Bob", Kind: ast.ParticipantActor})

	if len(s.Participants()) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(s.Participants()))
	}
	if s.Participants()[0].Alias != "A" {
		t.Errorf("expected alias %q, got %q", "A", s.Participants()[0].Alias)
	}
	if s.Participants()[1].Kind != ast.ParticipantActor {
		t.Errorf("expected kind %q, got %q", ast.ParticipantActor, s.Participants()[1].Kind)
	}
}

func TestSequenceAddMessage(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddMessage(ast.SeqMessage{
		From: "A", To: "B", Text: "Hello", Style: ast.MsgSync,
	})

	if len(s.Messages()) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.Messages()))
	}
	msg := s.Messages()[0]
	if msg.From != "A" || msg.To != "B" {
		t.Errorf("expected message A->B, got %s->%s", msg.From, msg.To)
	}
	if msg.Text != "Hello" {
		t.Errorf("expected text %q, got %q", "Hello", msg.Text)
	}
	if msg.Style != ast.MsgSync {
		t.Errorf("expected style %q, got %q", ast.MsgSync, msg.Style)
	}
}

func TestSequenceMessageStyles(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddParticipant(ast.Participant{Alias: "A"})
	s.AddParticipant(ast.Participant{Alias: "B"})

	styles := []ast.MessageStyle{
		ast.MsgSync, ast.MsgAsync, ast.MsgSyncNoArrow, ast.MsgAsyncNoArrow,
		ast.MsgSyncX, ast.MsgAsyncX, ast.MsgOpen, ast.MsgAsyncOpen,
	}

	for _, style := range styles {
		s.AddMessage(ast.SeqMessage{From: "A", To: "B", Style: style})
	}

	if len(s.Messages()) != len(styles) {
		t.Fatalf("expected %d messages, got %d", len(styles), len(s.Messages()))
	}
}

func TestSequenceMessageActivate(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddMessage(ast.SeqMessage{
		From: "A", To: "B", Activate: true, Deactivate: true,
	})

	msg := s.Messages()[0]
	if !msg.Activate {
		t.Error("expected Activate to be true")
	}
	if !msg.Deactivate {
		t.Error("expected Deactivate to be true")
	}
}

func TestSequenceAddNote(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddNote(ast.SeqNote{Over: []string{"A"}, Text: "Note about A"})
	s.AddNote(ast.SeqNote{Over: []string{"A", "B"}, Text: "Note about A and B"})

	if len(s.Notes()) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(s.Notes()))
	}
	if len(s.Notes()[0].Over) != 1 {
		t.Errorf("expected 1 participant in note, got %d", len(s.Notes()[0].Over))
	}
	if len(s.Notes()[1].Over) != 2 {
		t.Errorf("expected 2 participants in note, got %d", len(s.Notes()[1].Over))
	}
}

func TestSequenceAddLoop(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddLoop(ast.SeqLoop{
		Label: "Loop",
		Messages: []ast.SeqMessage{
			{From: "A", To: "B", Text: "msg"},
		},
	})

	if len(s.Loops()) != 1 {
		t.Fatalf("expected 1 loop, got %d", len(s.Loops()))
	}
	if s.Loops()[0].Label != "Loop" {
		t.Errorf("expected label %q, got %q", "Loop", s.Loops()[0].Label)
	}
	if len(s.Loops()[0].Messages) != 1 {
		t.Errorf("expected 1 message in loop, got %d", len(s.Loops()[0].Messages))
	}
}

func TestSequenceAddAlt(t *testing.T) {
	s := ast.NewSequence("", false)
	s.AddAlt(ast.SeqAlt{
		Condition: "condition",
		Messages:  []ast.SeqMessage{{From: "A", To: "B", Text: "msg1"}},
		Else:      []ast.SeqMessage{{From: "A", To: "B", Text: "msg2"}},
	})

	if len(s.Alts()) != 1 {
		t.Fatalf("expected 1 alt, got %d", len(s.Alts()))
	}
	alt := s.Alts()[0]
	if alt.Condition != "condition" {
		t.Errorf("expected condition %q, got %q", "condition", alt.Condition)
	}
	if len(alt.Messages) != 1 || len(alt.Else) != 1 {
		t.Errorf("expected 1 message in each branch, got %d and %d",
			len(alt.Messages), len(alt.Else))
	}
}

func TestSequenceEmpty(t *testing.T) {
	s := ast.NewSequence("", false)
	if len(s.Participants()) != 0 {
		t.Errorf("expected 0 participants, got %d", len(s.Participants()))
	}
	if len(s.Messages()) != 0 {
		t.Errorf("expected 0 messages, got %d", len(s.Messages()))
	}
	if len(s.Notes()) != 0 {
		t.Errorf("expected 0 notes, got %d", len(s.Notes()))
	}
	if len(s.Loops()) != 0 {
		t.Errorf("expected 0 loops, got %d", len(s.Loops()))
	}
	if len(s.Alts()) != 0 {
		t.Errorf("expected 0 alts, got %d", len(s.Alts()))
	}
}