package ast_test

import (
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewClass(t *testing.T) {
	c := ast.NewClass("Test Class")
	if c.Title() != "Test Class" {
		t.Errorf("expected title %q, got %q", "Test Class", c.Title())
	}
	if c.Type() != diagram.TypeClass {
		t.Errorf("expected type %q, got %q", diagram.TypeClass, c.Type())
	}
}

func TestClassAddClass(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name:       "MyClass",
		Namespace:  "ns",
		Annotation: "<<interface>>",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "void", Name: "method", IsMethod: true},
		},
	})

	if len(c.Classes()) != 1 {
		t.Fatalf("expected 1 class, got %d", len(c.Classes()))
	}
	cls := c.Classes()[0]
	if cls.Name != "MyClass" {
		t.Errorf("expected name %q, got %q", "MyClass", cls.Name)
	}
	if cls.Namespace != "ns" {
		t.Errorf("expected namespace %q, got %q", "ns", cls.Namespace)
	}
	if cls.Annotation != "<<interface>>" {
		t.Errorf("expected annotation %q, got %q", "<<interface>>", cls.Annotation)
	}
}

func TestClassMemberVisibility(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name: "Test",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Name: "publicField"},
			{Visibility: ast.VisPrivate, Name: "privateField"},
			{Visibility: ast.VisProtected, Name: "protectedField"},
			{Visibility: ast.VisPackage, Name: "packageField"},
		},
	})

	members := c.Classes()[0].Members
	wantVis := []ast.MemberVisibility{ast.VisPublic, ast.VisPrivate, ast.VisProtected, ast.VisPackage}
	for i, vis := range wantVis {
		if members[i].Visibility != vis {
			t.Errorf("expected visibility %q, got %q", vis, members[i].Visibility)
		}
	}
}

func TestClassMemberModifiers(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{
		Name: "Test",
		Members: []ast.ClassMember{
			{Name: "instanceField", IsMethod: false, IsStatic: false, IsAbstract: false},
			{Name: "staticMethod", IsMethod: true, IsStatic: true, IsAbstract: false},
			{Name: "abstractMethod", IsMethod: true, IsStatic: false, IsAbstract: true},
		},
	})

	members := c.Classes()[0].Members
	if members[0].IsStatic || members[0].IsAbstract {
		t.Error("first member should be instance field")
	}
	if !members[1].IsStatic {
		t.Error("second member should be static")
	}
	if !members[1].IsMethod {
		t.Error("second member should be method")
	}
	if !members[2].IsAbstract {
		t.Error("third member should be abstract")
	}
}

func TestClassAddRelation(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "A"})
	c.AddClass(ast.DiagramClass{Name: "B"})
	c.AddRelation(ast.ClassRelation{
		From:     "A",
		To:       "B",
		Kind:     ast.RelInheritance,
		Label:    "inherits",
		CardFrom: "1",
		CardTo:   "*",
	})

	if len(c.Relations()) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(c.Relations()))
	}
	rel := c.Relations()[0]
	if rel.Kind != ast.RelInheritance {
		t.Errorf("expected kind %q, got %q", ast.RelInheritance, rel.Kind)
	}
	if rel.Label != "inherits" {
		t.Errorf("expected label %q, got %q", "inherits", rel.Label)
	}
}

func TestClassRelationTypes(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "A"})
	c.AddClass(ast.DiagramClass{Name: "B"})

	types := []ast.RelationType{
		ast.RelInheritance, ast.RelComposition, ast.RelAggregation,
		ast.RelAssociation, ast.RelRealization, ast.RelDependency, ast.RelLink,
	}

	for _, typ := range types {
		c.AddRelation(ast.ClassRelation{From: "A", To: "B", Kind: typ})
	}

	if len(c.Relations()) != len(types) {
		t.Fatalf("expected %d relations, got %d", len(types), len(c.Relations()))
	}
}

func TestClassAddNote(t *testing.T) {
	c := ast.NewClass("")
	c.AddClass(ast.DiagramClass{Name: "MyClass"})
	c.AddNote(ast.ClassNote{Class: "MyClass", Text: "A note"})

	if len(c.Notes()) != 1 {
		t.Fatalf("expected 1 note, got %d", len(c.Notes()))
	}
	if c.Notes()[0].Text != "A note" {
		t.Errorf("expected text %q, got %q", "A note", c.Notes()[0].Text)
	}
}

func TestClassEmpty(t *testing.T) {
	c := ast.NewClass("")
	if len(c.Classes()) != 0 {
		t.Errorf("expected 0 classes, got %d", len(c.Classes()))
	}
	if len(c.Relations()) != 0 {
		t.Errorf("expected 0 relations, got %d", len(c.Relations()))
	}
	if len(c.Notes()) != 0 {
		t.Errorf("expected 0 notes, got %d", len(c.Notes()))
	}
}