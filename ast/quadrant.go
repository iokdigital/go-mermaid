package ast

import diagram "github.com/iokdigital/go-mermaid"

// QuadrantDiagram represents a Mermaid quadrantChart: points plotted on a
// 2×2 grid. Point coordinates are in the 0..1 range; the mmd encoder rejects
// out-of-range, NaN, and infinite values.
type QuadrantDiagram struct {
	title  string
	xAxis  [2]string // [from, to]; to == "" renders a single label
	yAxis  [2]string
	quads  [4]string // quadrant-1..quadrant-4 labels, index 0..3
	points []QuadrantPoint
}

// QuadrantPoint is a named point on the quadrant grid.
type QuadrantPoint struct {
	Name string
	X, Y float64
}

// NewQuadrant creates an empty quadrant chart with the given title.
func NewQuadrant(title string) *QuadrantDiagram {
	return &QuadrantDiagram{title: title}
}

func (q *QuadrantDiagram) Type() diagram.DiagramType { return diagram.TypeQuadrant }
func (q *QuadrantDiagram) Title() string             { return q.title }
func (q *QuadrantDiagram) XAxis() [2]string          { return q.xAxis }
func (q *QuadrantDiagram) YAxis() [2]string          { return q.yAxis }
func (q *QuadrantDiagram) Quadrants() [4]string      { return q.quads }
func (q *QuadrantDiagram) Points() []QuadrantPoint   { return q.points }

// SetXAxis sets the x-axis labels. Pass to == "" for a single label.
func (q *QuadrantDiagram) SetXAxis(from, to string) *QuadrantDiagram {
	q.xAxis = [2]string{from, to}
	return q
}

// SetYAxis sets the y-axis labels. Pass to == "" for a single label.
func (q *QuadrantDiagram) SetYAxis(from, to string) *QuadrantDiagram {
	q.yAxis = [2]string{from, to}
	return q
}

// SetQuadrantLabel sets the quadrant-N label (n is 1..4); out-of-range n is ignored.
func (q *QuadrantDiagram) SetQuadrantLabel(n int, label string) *QuadrantDiagram {
	if n >= 1 && n <= 4 {
		q.quads[n-1] = label
	}
	return q
}

// AddPoint appends a point to the quadrant chart.
func (q *QuadrantDiagram) AddPoint(p QuadrantPoint) *QuadrantDiagram {
	q.points = append(q.points, p)
	return q
}
