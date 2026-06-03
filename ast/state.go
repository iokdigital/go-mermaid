package ast

import diagram "github.com/iokdigital/go-mermaid"

// StateDiagram represents a Mermaid stateDiagram-v2.
type StateDiagram struct {
	title       string
	states      []DiagramState
	transitions []StateTransition
	notes       []StateNote
}

// StateKind classifies the role of a state node.
type StateKind string

const (
	StateNormal StateKind = "normal"
	StateStart  StateKind = "start"  // emits [*] as source
	StateEnd    StateKind = "end"    // emits [*] as target
	StateFork   StateKind = "fork"
	StateJoin   StateKind = "join"
	StateChoice StateKind = "choice"
)

// DiagramState is a node in the state machine.
// Children are used for composite (nested) states.
type DiagramState struct {
	ID       string
	Label    string
	Kind     StateKind
	Children []DiagramState
}

// StateTransition is a directed edge between two states.
type StateTransition struct {
	From  string
	To    string
	Event string // optional label on the transition arrow
}

// StateNote attaches a note to a state.
type StateNote struct {
	State string
	Text  string
}

// NewState creates an empty state diagram.
func NewState(title string) *StateDiagram {
	return &StateDiagram{title: title}
}

func (s *StateDiagram) Type() diagram.DiagramType    { return diagram.TypeState }
func (s *StateDiagram) Title() string                 { return s.title }
func (s *StateDiagram) States() []DiagramState        { return s.states }
func (s *StateDiagram) Transitions() []StateTransition { return s.transitions }
func (s *StateDiagram) Notes() []StateNote            { return s.notes }

func (s *StateDiagram) AddState(st DiagramState) *StateDiagram {
	s.states = append(s.states, st)
	return s
}

func (s *StateDiagram) AddTransition(t StateTransition) *StateDiagram {
	s.transitions = append(s.transitions, t)
	return s
}

func (s *StateDiagram) AddNote(n StateNote) *StateDiagram {
	s.notes = append(s.notes, n)
	return s
}
