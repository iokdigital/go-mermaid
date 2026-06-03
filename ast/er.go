package ast

import diagram "github.com/iokdigital/go-mermaid"

// ERDiagram represents a Mermaid erDiagram.
type ERDiagram struct {
	title     string
	entities  []EREntity
	relations []ERRelation
}

// EREntity is a table or object in the ER model.
type EREntity struct {
	Name       string
	Attributes []ERAttribute
}

// ERAttribute is a column or field within an entity.
type ERAttribute struct {
	DataType string
	Name     string
	Keys     []ERKey
	Comment  string
}

// ERKey marks primary, foreign, or unique key constraints.
type ERKey string

const (
	KeyPrimary ERKey = "PK"
	KeyForeign ERKey = "FK"
	KeyUnique  ERKey = "UK"
)

// Cardinality is the left-side notation for an ER relationship endpoint.
// The mmd encoder mirrors these symbols for the right-side endpoint.
type Cardinality string

const (
	CardZeroOne  Cardinality = "|o" // zero or one
	CardExactOne Cardinality = "||" // exactly one
	CardZeroMany Cardinality = "}o" // zero or many
	CardOneMany  Cardinality = "}|" // one or many
)

// ERRelation is a relationship between two entities.
type ERRelation struct {
	From        string
	To          string
	FromCard    Cardinality
	ToCard      Cardinality
	Label       string
	Identifying bool // true = solid line "--", false = dashed line ".."
}

// NewER creates an empty ER diagram.
func NewER(title string) *ERDiagram {
	return &ERDiagram{title: title}
}

func (e *ERDiagram) Type() diagram.DiagramType  { return diagram.TypeER }
func (e *ERDiagram) Title() string               { return e.title }
func (e *ERDiagram) Entities() []EREntity        { return e.entities }
func (e *ERDiagram) Relations() []ERRelation     { return e.relations }

func (e *ERDiagram) AddEntity(en EREntity) *ERDiagram {
	e.entities = append(e.entities, en)
	return e
}

func (e *ERDiagram) AddRelation(r ERRelation) *ERDiagram {
	e.relations = append(e.relations, r)
	return e
}
