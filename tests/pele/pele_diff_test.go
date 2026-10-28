// Package pele_test differentially validates go-mermaid's .mmd output
// against the Pele parser (https://pele.run, obsidianmd/pele).
//
// Pele implements the Mermaid 12.1.0 grammar; if Pele rejects our generated
// source, our encoder almost certainly emits non-conformant syntax. Layout
// and styling are intentionally NOT compared — the engines differ by design.
//
// Requirements: node on PATH and scripts/pele-diff/node_modules installed
// (run `npm install` in scripts/pele-diff). Otherwise these tests skip.
package pele_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
	"github.com/iokdigital/go-mermaid/render"
)

// peleResult is one JSON line emitted by scripts/pele-diff/render.mjs.
type peleResult struct {
	File        string `json:"file"`
	OK          bool   `json:"ok"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Type        string `json:"type"`
	MermaidType string `json:"mermaidType"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SVGLength   int    `json:"svgLength"`
}

// peleHarness locates the node harness, skipping when unavailable.
func peleHarness(t *testing.T) (node, script string) {
	t.Helper()
	var err error
	if node, err = exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH; install Node >= 20 to run Pele differential tests")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // repo root
	script = filepath.Join(root, "scripts", "pele-diff", "render.mjs")
	if _, err := os.Stat(filepath.Join(root, "scripts", "pele-diff", "node_modules", "pele")); err != nil {
		t.Skipf("pele not installed; run `npm install` in %s", filepath.Join(root, "scripts", "pele-diff"))
	}
	return node, script
}

// runPele renders each .mmd file with Pele and returns results keyed by path.
func runPele(t *testing.T, node, script string, files []string) map[string]peleResult {
	t.Helper()
	cmd := exec.Command(node, append([]string{script}, files...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node render.mjs failed: %v\nstderr: %s", err, stderr.String())
	}
	out := make(map[string]peleResult, len(files))
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		var r peleResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("decode pele result %q: %v", line, err)
		}
		out[r.File] = r
	}
	if len(out) != len(files) {
		t.Fatalf("expected %d pele results, got %d", len(files), len(out))
	}
	return out
}

// assertAccepted fails when Pele rejects source we generated.
func assertAccepted(t *testing.T, r peleResult) {
	t.Helper()
	if !r.OK {
		t.Errorf("Pele rejected %s: [%s] %s (line %d, col %d)",
			filepath.Base(r.File), r.Code, r.Message, r.Line, r.Column)
		return
	}
	if r.SVGLength == 0 || r.Width <= 0 || r.Height <= 0 {
		t.Errorf("Pele rendered %s to empty/degenerate SVG (w=%d h=%d len=%d)",
			filepath.Base(r.File), r.Width, r.Height, r.SVGLength)
	}
}

// TestPeleAcceptsGoldenMMD requires Pele to parse every golden .mmd file.
func TestPeleAcceptsGoldenMMD(t *testing.T) {
	node, script := peleHarness(t)
	matches, err := filepath.Glob(filepath.Join(
		filepath.Dir(script), "..", "..", "testdata", "golden", "mmd", "*.mmd"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no golden .mmd files found: %v", err)
	}
	results := runPele(t, node, script, matches)
	for _, f := range matches {
		r := results[f]
		t.Logf("%s: type=%s mermaidType=%s %dx%d svg=%d bytes",
			filepath.Base(f), r.Type, r.MermaidType, r.Width, r.Height, r.SVGLength)
		assertAccepted(t, r)
	}
}

// edgeCase builds a diagram exercising encoder escaping paths.
func edgeCase() []struct {
	name string
	d    diagram.Diagram
} {
	fc := ast.NewFlowchart("<script>alert('xss')</script>", ast.DirectionLR)
	fc.MustAddNode(&ast.FlowNode{ID: "A", Label: `quote " unterminated [bracket] (paren) {brace} |pipe|`})
	fc.MustAddNode(&ast.FlowNode{ID: "B", Label: "héllo wörld 日本語 🎉"})
	long := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 3) // > 40 runes: truncation path
	fc.MustAddNode(&ast.FlowNode{ID: "C", Label: long})
	fc.AddEdge(&ast.FlowEdge{From: "A", To: "B", Label: `edge "label" |with| specials`})
	fc.AddEdge(&ast.FlowEdge{From: "B", To: "C"})

	seq := ast.NewSequence("Seq <b>bold</b> & \"quoted\"", true)
	seq.AddParticipant(ast.Participant{Alias: "A", Label: "Alice & Bob", Kind: ast.ParticipantBox})
	seq.AddParticipant(ast.Participant{Alias: "B", Label: "Böb"}) // zero Kind must default to participant
	seq.AddMessage(ast.SeqMessage{From: "A", To: "B", Text: "Hello <world> \"hi\""})

	st := ast.NewState("State \"quoted\"")
	st.AddState(ast.DiagramState{ID: "Idle", Label: "Idle & waiting", Kind: ast.StateNormal})
	st.AddTransition(ast.StateTransition{From: "Idle", To: "Idle", Event: "tick <t>"})

	er := ast.NewER("ER & entities")
	er.AddEntity(ast.EREntity{Name: "User", Attributes: []ast.ERAttribute{
		{Name: "nick\"name", DataType: "string", Comment: "a <b>comment</b>"},
	}})
	er.AddRelation(ast.ERRelation{
		From: "User", To: "User", FromCard: ast.CardExactOne,
		ToCard: ast.CardZeroMany, Label: "follows & blocks",
	})

	cl := ast.NewClass("Class <gen>")
	cl.AddClass(ast.DiagramClass{
		Name: "Repo",
		Members: []ast.ClassMember{
			{Visibility: ast.VisPublic, Type: "string", Name: "title\"x"},
			{Visibility: ast.VisPrivate, Type: "int", Name: "count", IsMethod: true},
		},
	})

	pc := ast.NewPie(`Pie "nightly": build & test`)
	pc.AddSlice(ast.PieSlice{Label: `pass: fast | "quoted"`, Value: 9})
	pc.AddSlice(ast.PieSlice{Label: "fail & retry", Value: 1})

	qd := ast.NewQuadrant(`Q1 "review": growth & risk`)
	qd.SetXAxis("Cost: low", "Cost: high")
	qd.SetYAxis("Risk", "Reward")
	qd.SetQuadrantLabel(1, `Grow "fast" & sustainably`)
	qd.SetQuadrantLabel(4, "Harvest")
	qd.AddPoint(ast.QuadrantPoint{Name: `Alpha: "first"`, X: 0.8, Y: 0.9})
	qd.AddPoint(ast.QuadrantPoint{Name: "Beta & co", X: 0.2, Y: 0.3})

	return []struct {
		name string
		d    diagram.Diagram
	}{
		{"escaping-flowchart", fc},
		{"escaping-sequence", seq},
		{"escaping-state", st},
		{"escaping-er", er},
		{"escaping-class", cl},
		{"escaping-pie", pc},
		{"escaping-quadrant", qd},
	}
}

// TestPeleAcceptsEdgeCases requires Pele to parse encoder output containing
// HTML-special characters, quotes, unicode, and over-long labels.
func TestPeleAcceptsEdgeCases(t *testing.T) {
	node, script := peleHarness(t)
	r := render.NewRenderer(diagram.NewRenderOptions())
	tmp := t.TempDir()

	var files []string
	for _, tc := range edgeCase() {
		data, err := r.RenderBytes(tc.d, diagram.FormatMMD)
		if err != nil {
			t.Fatalf("%s: render mmd: %v", tc.name, err)
		}
		path := filepath.Join(tmp, tc.name+".mmd")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		files = append(files, path)
	}

	results := runPele(t, node, script, files)
	for _, f := range files {
		assertAccepted(t, results[f])
	}
}
