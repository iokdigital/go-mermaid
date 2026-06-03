package ast

import diagram "github.com/iokdigital/go-mermaid"

// SequenceDiagram represents a Mermaid sequenceDiagram.
type SequenceDiagram struct {
	title        string
	autonumber   bool
	participants []Participant
	messages     []SeqMessage
	notes        []SeqNote
	loops        []SeqLoop
	alts         []SeqAlt
}

// ParticipantKind controls whether to emit `participant` or `actor`.
type ParticipantKind string

const (
	ParticipantBox   ParticipantKind = "participant"
	ParticipantActor ParticipantKind = "actor"
)

// Participant is a named entity in the sequence.
type Participant struct {
	Alias string
	Label string
	Kind  ParticipantKind
}

// MessageStyle is the arrow/line style between two participants.
type MessageStyle string

const (
	MsgSync         MessageStyle = "->>"  // solid line, filled arrowhead
	MsgAsync        MessageStyle = "-->>" // dashed line, filled arrowhead (responses)
	MsgSyncNoArrow  MessageStyle = "->"   // solid line, no arrowhead
	MsgAsyncNoArrow MessageStyle = "-->"  // dashed line, no arrowhead
	MsgSyncX        MessageStyle = "-x"   // solid line, cross end (lost message)
	MsgAsyncX       MessageStyle = "--x"  // dashed line, cross end
	MsgOpen         MessageStyle = "-)"   // solid line, open arrow
	MsgAsyncOpen    MessageStyle = "--)"  // dashed line, open arrow
)

// SeqMessage is a single arrow between two participants.
// Activate/Deactivate add +/- suffix to the target alias in the emitted mmd.
type SeqMessage struct {
	From       string
	To         string
	Text       string
	Style      MessageStyle
	Activate   bool // appends + to target: adds activation bar
	Deactivate bool // appends - to target: removes activation bar
}

// SeqNote is a note displayed above one or more participants.
type SeqNote struct {
	Over []string // participant aliases; len==1 = "Note over A", len==2 = "Note over A,B"
	Text string
}

// SeqLoop wraps a set of messages in a Mermaid loop block.
type SeqLoop struct {
	Label    string
	Messages []SeqMessage
}

// SeqAlt wraps messages in an alt/else block.
type SeqAlt struct {
	Condition string
	Messages  []SeqMessage
	Else      []SeqMessage
}

// NewSequence creates an empty sequence diagram.
func NewSequence(title string, autonumber bool) *SequenceDiagram {
	return &SequenceDiagram{title: title, autonumber: autonumber}
}

func (s *SequenceDiagram) Type() diagram.DiagramType { return diagram.TypeSequence }
func (s *SequenceDiagram) Title() string              { return s.title }
func (s *SequenceDiagram) Autonumber() bool           { return s.autonumber }
func (s *SequenceDiagram) Participants() []Participant { return s.participants }
func (s *SequenceDiagram) Messages() []SeqMessage     { return s.messages }
func (s *SequenceDiagram) Notes() []SeqNote           { return s.notes }
func (s *SequenceDiagram) Loops() []SeqLoop           { return s.loops }
func (s *SequenceDiagram) Alts() []SeqAlt             { return s.alts }

func (s *SequenceDiagram) AddParticipant(p Participant) *SequenceDiagram {
	s.participants = append(s.participants, p)
	return s
}

func (s *SequenceDiagram) AddMessage(m SeqMessage) *SequenceDiagram {
	s.messages = append(s.messages, m)
	return s
}

func (s *SequenceDiagram) AddNote(n SeqNote) *SequenceDiagram {
	s.notes = append(s.notes, n)
	return s
}

func (s *SequenceDiagram) AddLoop(l SeqLoop) *SequenceDiagram {
	s.loops = append(s.loops, l)
	return s
}

func (s *SequenceDiagram) AddAlt(a SeqAlt) *SequenceDiagram {
	s.alts = append(s.alts, a)
	return s
}
