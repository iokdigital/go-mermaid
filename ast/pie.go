package ast

import diagram "github.com/iokdigital/go-mermaid"

// PieDiagram represents a Mermaid pie chart.
type PieDiagram struct {
	title  string
	slices []PieSlice
}

// PieSlice is one labelled value in a pie chart. Values must be >= 0;
// the mmd encoder rejects negative, NaN, and infinite values.
type PieSlice struct {
	Label string
	Value float64
}

// NewPie creates an empty pie chart with the given title.
func NewPie(title string) *PieDiagram {
	return &PieDiagram{title: title}
}

func (p *PieDiagram) Type() diagram.DiagramType { return diagram.TypePie }
func (p *PieDiagram) Title() string             { return p.title }
func (p *PieDiagram) Slices() []PieSlice        { return p.slices }

// AddSlice appends a slice to the pie chart.
func (p *PieDiagram) AddSlice(s PieSlice) *PieDiagram {
	p.slices = append(p.slices, s)
	return p
}
