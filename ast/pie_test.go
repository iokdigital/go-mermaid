package ast_test

import (
	"testing"

	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewPie(t *testing.T) {
	p := ast.NewPie("Fish")
	if p.Title() != "Fish" {
		t.Errorf("expected title Fish, got %q", p.Title())
	}
	if p.Type() != "pie" {
		t.Errorf("expected type pie, got %q", p.Type())
	}
	if len(p.Slices()) != 0 {
		t.Errorf("expected no slices, got %d", len(p.Slices()))
	}
}

func TestPieAddSlice(t *testing.T) {
	p := ast.NewPie("")
	p.AddSlice(ast.PieSlice{Label: "Ahi", Value: 386})
	p.AddSlice(ast.PieSlice{Label: "Ono", Value: 15})
	slices := p.Slices()
	if len(slices) != 2 {
		t.Fatalf("expected 2 slices, got %d", len(slices))
	}
	if slices[0].Label != "Ahi" || slices[0].Value != 386 {
		t.Errorf("unexpected first slice: %+v", slices[0])
	}
}
