package ast_test

import (
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewQuadrant(t *testing.T) {
	q := ast.NewQuadrant("Campaigns")
	if q.Title() != "Campaigns" {
		t.Errorf("expected title Campaigns, got %q", q.Title())
	}
	if q.Type() != "quadrant" {
		t.Errorf("expected type quadrant, got %q", q.Type())
	}
	if len(q.Points()) != 0 {
		t.Errorf("expected no points, got %d", len(q.Points()))
	}
}

func TestQuadrantAxesAndLabels(t *testing.T) {
	q := ast.NewQuadrant("")
	q.SetXAxis("Low Reach", "High Reach")
	q.SetYAxis("Low Engagement", "High Engagement")
	q.SetQuadrantLabel(1, "We should expand")
	q.SetQuadrantLabel(5, "ignored") // out of range: no-op

	if q.XAxis() != [2]string{"Low Reach", "High Reach"} {
		t.Errorf("unexpected x-axis: %v", q.XAxis())
	}
	if q.YAxis() != [2]string{"Low Engagement", "High Engagement"} {
		t.Errorf("unexpected y-axis: %v", q.YAxis())
	}
	quads := q.Quadrants()
	if quads[0] != "We should expand" {
		t.Errorf("unexpected quadrant-1: %q", quads[0])
	}
	if quads[1] != "" || quads[2] != "" || quads[3] != "" {
		t.Errorf("expected empty quadrants 2-4, got %v", quads)
	}
}

func TestQuadrantAddPoint(t *testing.T) {
	q := ast.NewQuadrant("")
	q.AddPoint(ast.QuadrantPoint{Name: "Campaign A", X: 0.3, Y: 0.6})
	points := q.Points()
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Name != "Campaign A" || points[0].X != 0.3 || points[0].Y != 0.6 {
		t.Errorf("unexpected point: %+v", points[0])
	}
}
