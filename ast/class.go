package ast

import diagram "github.com/iokdigital/go-mermaid"

// ClassDiagram represents a Mermaid classDiagram.
type ClassDiagram struct {
	title     string
	classes   []DiagramClass
	relations []ClassRelation
	notes     []ClassNote
}

// DiagramClass is a single class or interface box.
type DiagramClass struct {
	Name       string
	Namespace  string // emitted as `namespace Ns { class Name }` if non-empty
	Members    []ClassMember
	Annotation string // e.g. "<<interface>>", "<<abstract>>"
}

// MemberVisibility controls the +/-/#/~ prefix on class members.
type MemberVisibility string

const (
	VisPublic    MemberVisibility = "+"
	VisPrivate   MemberVisibility = "-"
	VisProtected MemberVisibility = "#"
	VisPackage   MemberVisibility = "~"
)

// ClassMember is a field or method of a class.
type ClassMember struct {
	Visibility MemberVisibility
	Type       string
	Name       string
	IsMethod   bool
	IsStatic   bool
	IsAbstract bool
}

// RelationType is the Mermaid relationship arrow string.
type RelationType string

const (
	RelInheritance RelationType = "<|--"
	RelComposition RelationType = "*--"
	RelAggregation RelationType = "o--"
	RelAssociation RelationType = "-->"
	RelRealization RelationType = "<|.."
	RelDependency  RelationType = ".."
	RelLink        RelationType = "--"
)

// ClassRelation is a directed relationship between two classes.
type ClassRelation struct {
	From     string
	To       string
	Kind     RelationType
	Label    string
	CardFrom string
	CardTo   string
}

// ClassNote attaches a note to a class.
type ClassNote struct {
	Class string
	Text  string
}

// NewClass creates an empty class diagram.
func NewClass(title string) *ClassDiagram {
	return &ClassDiagram{title: title}
}

func (c *ClassDiagram) Type() diagram.DiagramType  { return diagram.TypeClass }
func (c *ClassDiagram) Title() string               { return c.title }
func (c *ClassDiagram) Classes() []DiagramClass     { return c.classes }
func (c *ClassDiagram) Relations() []ClassRelation  { return c.relations }
func (c *ClassDiagram) Notes() []ClassNote          { return c.notes }

func (c *ClassDiagram) AddClass(cl DiagramClass) *ClassDiagram {
	c.classes = append(c.classes, cl)
	return c
}

func (c *ClassDiagram) AddRelation(r ClassRelation) *ClassDiagram {
	c.relations = append(c.relations, r)
	return c
}

func (c *ClassDiagram) AddNote(n ClassNote) *ClassDiagram {
	c.notes = append(c.notes, n)
	return c
}
