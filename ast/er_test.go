package ast_test

import (
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

func TestNewER(t *testing.T) {
	e := ast.NewER("Test ER")
	if e.Title() != "Test ER" {
		t.Errorf("expected title %q, got %q", "Test ER", e.Title())
	}
	if e.Type() != diagram.TypeER {
		t.Errorf("expected type %q, got %q", diagram.TypeER, e.Type())
	}
}

func TestERAddEntity(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{
		Name: "User",
		Attributes: []ast.ERAttribute{
			{Name: "id", DataType: "int", Keys: []ast.ERKey{ast.KeyPrimary}},
			{Name: "email", DataType: "varchar", Keys: []ast.ERKey{ast.KeyUnique}},
		},
	})

	if len(e.Entities()) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(e.Entities()))
	}
	if e.Entities()[0].Name != "User" {
		t.Errorf("expected name %q, got %q", "User", e.Entities()[0].Name)
	}
	if len(e.Entities()[0].Attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(e.Entities()[0].Attributes))
	}
}

func TestERKeyTypes(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{
		Name: "Test",
		Attributes: []ast.ERAttribute{
			{Name: "pk", Keys: []ast.ERKey{ast.KeyPrimary}},
			{Name: "fk", Keys: []ast.ERKey{ast.KeyForeign}},
			{Name: "uk", Keys: []ast.ERKey{ast.KeyUnique}},
		},
	})

	attrs := e.Entities()[0].Attributes
	for i, key := range []ast.ERKey{ast.KeyPrimary, ast.KeyForeign, ast.KeyUnique} {
		if attrs[i].Keys[0] != key {
			t.Errorf("expected key %q, got %q", key, attrs[i].Keys[0])
		}
	}
}

func TestERAddRelation(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "User"})
	e.AddEntity(ast.EREntity{Name: "Order"})
	e.AddRelation(ast.ERRelation{
		From:        "User",
		To:          "Order",
		FromCard:    ast.CardOneMany,
		ToCard:      ast.CardZeroOne,
		Label:       "places",
		Identifying: true,
	})

	if len(e.Relations()) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(e.Relations()))
	}
	rel := e.Relations()[0]
	if rel.From != "User" || rel.To != "Order" {
		t.Errorf("expected relation User->Order, got %s->%s", rel.From, rel.To)
	}
	if rel.Label != "places" {
		t.Errorf("expected label %q, got %q", "places", rel.Label)
	}
}

func TestERCardinalities(t *testing.T) {
	cards := []ast.Cardinality{
		ast.CardZeroOne, ast.CardExactOne, ast.CardZeroMany, ast.CardOneMany,
	}
	for _, c := range cards {
		if c == "" {
			t.Errorf("cardinality should not be empty")
		}
	}
}

func TestERRelationIdentifying(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{Name: "A"})
	e.AddEntity(ast.EREntity{Name: "B"})

	e.AddRelation(ast.ERRelation{From: "A", To: "B", Identifying: true})
	e.AddRelation(ast.ERRelation{From: "A", To: "B", Identifying: false})

	if e.Relations()[0].Identifying != true {
		t.Error("expected first relation to be identifying")
	}
	if e.Relations()[1].Identifying != false {
		t.Error("expected second relation to be non-identifying")
	}
}

func TestERAttributeComment(t *testing.T) {
	e := ast.NewER("")
	e.AddEntity(ast.EREntity{
		Name: "User",
		Attributes: []ast.ERAttribute{
			{Name: "name", DataType: "varchar", Comment: "user's full name"},
		},
	})

	if e.Entities()[0].Attributes[0].Comment != "user's full name" {
		t.Errorf("expected comment %q, got %q",
			"user's full name", e.Entities()[0].Attributes[0].Comment)
	}
}

func TestEREmpty(t *testing.T) {
	e := ast.NewER("")
	if len(e.Entities()) != 0 {
		t.Errorf("expected 0 entities, got %d", len(e.Entities()))
	}
	if len(e.Relations()) != 0 {
		t.Errorf("expected 0 relations, got %d", len(e.Relations()))
	}
}