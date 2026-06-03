package svg

import (
	"fmt"
	"io"
	"math"
	"strings"

	diagram "github.com/iokdigital/go-mermaid"
	"github.com/iokdigital/go-mermaid/ast"
)

// Sequence diagram layout constants.
const (
	seqParticipantW   = 120.0 // participant box width
	seqParticipantH   = 36.0  // participant box height
	seqColSpacing     = 60.0  // horizontal gap between participant columns
	seqTopPad         = 20.0  // space above participant boxes
	seqParticipantGap = 40.0  // vertical gap from participant box bottom to first message
	seqMsgStep        = 44.0  // vertical step per message
	seqNoteH          = 28.0  // note box height
	seqLoopLabelH     = 22.0  // extra vertical space for frame label
	seqActivationW    = 10.0  // width of activation bar on lifeline
	seqLifelineEndPad = 30.0  // extra space at the bottom for lifelines
)

// encodeSequence renders a *ast.SequenceDiagram to w as an SVG document.
// Layout: participants as columns, time as vertical axis.
func encodeSequence(w io.Writer, d *ast.SequenceDiagram, opts diagram.RenderOptions) error {
	padding := float64(opts.SVGPadding)
	if padding <= 0 {
		padding = 40
	}
	maxW := float64(opts.SVGMaxWidth)
	if maxW <= 0 {
		maxW = 8000
	}
	maxH := float64(opts.SVGMaxHeight)
	if maxH <= 0 {
		maxH = 6000
	}

	participants := d.Participants()
	messages := d.Messages()
	notes := d.Notes()
	loops := d.Loops()
	alts := d.Alts()

	// Map participant aliases to column X positions.
	colX := make(map[string]float64, len(participants))
	colStep := seqParticipantW + seqColSpacing
	for i, p := range participants {
		colX[p.Alias] = padding + float64(i)*colStep + seqParticipantW/2
	}

	// Y origin for the first message.
	msgOriginY := padding + seqTopPad + seqParticipantH + seqParticipantGap

	// Assign Y positions to all events in order:
	// messages, then loop/alt embedded messages.
	// Loops and alts contribute their messages inline plus extra space for frame labels.
	type event struct {
		kind  string // "msg", "note", "loop-start", "loop-msg", "loop-end", "alt-start", "alt-msg", "alt-else", "alt-end"
		msg   *ast.SeqMessage
		note  *ast.SeqNote
		label string // loop condition or alt condition
		y     float64
	}

	events := make([]event, 0)
	y := msgOriginY

	// Top-level messages.
	for i := range messages {
		m := messages[i]
		events = append(events, event{kind: "msg", msg: &m, y: y})
		y += seqMsgStep
	}

	// Loop blocks.
	for _, l := range loops {
		events = append(events, event{kind: "loop-start", label: l.Label, y: y})
		y += seqLoopLabelH
		for i := range l.Messages {
			m := l.Messages[i]
			events = append(events, event{kind: "loop-msg", msg: &m, y: y})
			y += seqMsgStep
		}
		events = append(events, event{kind: "loop-end", y: y})
		y += seqMsgStep / 2
	}

	// Alt blocks.
	for _, a := range alts {
		events = append(events, event{kind: "alt-start", label: a.Condition, y: y})
		y += seqLoopLabelH
		for i := range a.Messages {
			m := a.Messages[i]
			events = append(events, event{kind: "alt-msg", msg: &m, y: y})
			y += seqMsgStep
		}
		if len(a.Else) > 0 {
			events = append(events, event{kind: "alt-else", y: y})
			y += seqLoopLabelH / 2
			for i := range a.Else {
				m := a.Else[i]
				events = append(events, event{kind: "alt-msg", msg: &m, y: y})
				y += seqMsgStep
			}
		}
		events = append(events, event{kind: "alt-end", y: y})
		y += seqMsgStep / 2
	}

	// Notes.
	for i := range notes {
		n := notes[i]
		events = append(events, event{kind: "note", note: &n, y: y})
		y += seqMsgStep
	}

	totalH := y + seqLifelineEndPad

	// Track activation state per participant for activation bar drawing.
	// activateStart[alias] = Y where current activation started.
	activateStart := make(map[string]float64)
	type activationBar struct {
		alias string
		y1    float64
		y2    float64
	}
	bars := make([]activationBar, 0)

	// First pass: collect activation bars from top-level messages.
	activY := msgOriginY
	for _, m := range messages {
		if m.Activate {
			activateStart[m.To] = activY
		}
		if m.Deactivate {
			if start, ok := activateStart[m.To]; ok {
				bars = append(bars, activationBar{m.To, start, activY})
				delete(activateStart, m.To)
			}
		}
		activY += seqMsgStep
	}
	// Close any open bars at the bottom.
	for alias, start := range activateStart {
		bars = append(bars, activationBar{alias, start, totalH - seqLifelineEndPad})
	}

	// Total width covers all participant columns.
	totalW := float64(0)
	if len(participants) > 0 {
		lastX := colX[participants[len(participants)-1].Alias]
		totalW = lastX + seqParticipantW/2 + padding
	}
	if totalW < 400 {
		totalW = 400
	}
	W := int(math.Ceil(clamp(totalW, 400, maxW)))
	H := int(math.Ceil(clamp(totalH, 300, maxH)))

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", W, H, W, H)

	if d.Title() != "" {
		fmt.Fprintf(&sb, "  <title>%s</title>\n", xmlEscape(d.Title()))
	}

	// Layer 1: Lifelines (dashed vertical lines).
	sb.WriteString(`  <g id="seq-lifelines">` + "\n")
	lifelineY1 := padding + seqTopPad + seqParticipantH
	lifelineY2 := float64(H) - padding
	for _, p := range participants {
		cx := colX[p.Alias]
		fmt.Fprintf(&sb, "    <path d=\"M%.1f,%.1f L%.1f,%.1f\" fill=\"none\" stroke=\"#94a3b8\" stroke-width=\"1\" stroke-dasharray=\"5,4\"/>\n",
			cx, lifelineY1, cx, lifelineY2)
	}
	sb.WriteString("  </g>\n")

	// Layer 2: Activation bars.
	sb.WriteString(`  <g id="seq-activations">` + "\n")
	for _, bar := range bars {
		cx, ok := colX[bar.alias]
		if !ok {
			continue
		}
		barH := bar.y2 - bar.y1
		if barH < 2 {
			barH = 2
		}
		fmt.Fprintf(&sb, "    <rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" fill=\"#e2e8f0\" stroke=\"#64748b\" stroke-width=\"1\"/>\n",
			cx-seqActivationW/2, bar.y1, seqActivationW, barH)
	}
	sb.WriteString("  </g>\n")

	// Layer 3: Frame boxes (loops, alts) — drawn before messages.
	sb.WriteString(`  <g id="seq-frames">` + "\n")
	// Collect frame extents.
	type frame struct {
		kind  string
		label string
		y1    float64
		y2    float64
	}
	frames := make([]frame, 0)
	{
		var curFrame *frame
		for _, ev := range events {
			switch ev.kind {
			case "loop-start":
				curFrame = &frame{kind: "loop", label: ev.label, y1: ev.y}
			case "loop-end":
				if curFrame != nil {
					curFrame.y2 = ev.y
					frames = append(frames, *curFrame)
					curFrame = nil
				}
			case "alt-start":
				curFrame = &frame{kind: "alt", label: ev.label, y1: ev.y}
			case "alt-end":
				if curFrame != nil {
					curFrame.y2 = ev.y
					frames = append(frames, *curFrame)
					curFrame = nil
				}
			}
		}
	}
	frameLeft := padding
	frameRight := float64(W) - padding
	for _, fr := range frames {
		fmt.Fprintf(&sb, "    <rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" fill=\"none\" stroke=\"#475569\" stroke-width=\"1.5\"/>\n",
			frameLeft, fr.y1, frameRight-frameLeft, fr.y2-fr.y1)
		// Frame label badge.
		labelW := float64(len(fr.kind))*8 + 8
		labelH := 18.0
		fmt.Fprintf(&sb, "    <rect x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\" fill=\"#e2e8f0\" stroke=\"#475569\" stroke-width=\"1\"/>\n",
			frameLeft, fr.y1, labelW, labelH)
		fmt.Fprintf(&sb, "    <text x=\"%.1f\" y=\"%.1f\" dominant-baseline=\"central\" font-size=\"11\" font-family=\"%s\" fill=\"#1e293b\" font-weight=\"bold\">%s</text>\n",
			frameLeft+4, fr.y1+labelH/2, defaultFontFamily, xmlEscape(fr.kind))
		// Condition text.
		condX := frameLeft + labelW + 6
		fmt.Fprintf(&sb, "    <text x=\"%.1f\" y=\"%.1f\" dominant-baseline=\"central\" font-size=\"11\" font-family=\"%s\" fill=\"#334155\" font-style=\"italic\">%s</text>\n",
			condX, fr.y1+labelH/2, defaultFontFamily, xmlEscape(fr.label))
	}
	// Alt else dividers.
	for _, ev := range events {
		if ev.kind == "alt-else" {
			fmt.Fprintf(&sb, "    <path d=\"M%.1f,%.1f L%.1f,%.1f\" fill=\"none\" stroke=\"#475569\" stroke-width=\"1\" stroke-dasharray=\"6,3\"/>\n",
				frameLeft, ev.y, frameRight, ev.y)
			fmt.Fprintf(&sb, "    <text x=\"%.1f\" y=\"%.1f\" dominant-baseline=\"central\" font-size=\"10\" font-family=\"%s\" fill=\"#475569\" font-style=\"italic\">else</text>\n",
				frameLeft+4, ev.y+8, defaultFontFamily)
		}
	}
	sb.WriteString("  </g>\n")

	// Layer 4: Participant boxes.
	sb.WriteString(`  <g id="seq-participants">` + "\n")
	for _, p := range participants {
		cx := colX[p.Alias]
		py := padding + seqTopPad
		sb.WriteString("    ")
		sb.WriteString(seqParticipantSVG(p, cx, py))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	// Layer 5: Messages and auto-numbers.
	sb.WriteString(`  <g id="seq-messages">` + "\n")
	msgNum := 1
	for _, ev := range events {
		if ev.msg == nil {
			continue
		}
		fromX, okF := colX[ev.msg.From]
		toX, okT := colX[ev.msg.To]
		if !okF || !okT {
			continue
		}
		label := ev.msg.Text
		if d.Autonumber() {
			label = fmt.Sprintf("%d: %s", msgNum, label)
			msgNum++
		}
		sb.WriteString("    ")
		sb.WriteString(seqMessageSVG(fromX, toX, ev.y, label, ev.msg.Style))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	// Layer 6: Notes.
	sb.WriteString(`  <g id="seq-notes">` + "\n")
	for _, ev := range events {
		if ev.note == nil {
			continue
		}
		n := ev.note
		if len(n.Over) == 0 {
			continue
		}
		x1 := colX[n.Over[0]] - seqParticipantW/2
		x2 := colX[n.Over[len(n.Over)-1]] + seqParticipantW/2
		sb.WriteString("    ")
		sb.WriteString(seqNoteSVG(x1, x2, ev.y, n.Text))
		sb.WriteString("\n")
	}
	sb.WriteString("  </g>\n")

	sb.WriteString("</svg>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// seqParticipantSVG returns SVG for a participant box at the top of the diagram.
func seqParticipantSVG(p ast.Participant, cx, py float64) string {
	label := p.Label
	if label == "" {
		label = p.Alias
	}

	x := cx - seqParticipantW/2
	fill := "#dbeafe"
	stroke := "#2563eb"
	if p.Kind == ast.ParticipantActor {
		fill = "#f0fdf4"
		stroke = "#16a34a"
	}

	return fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" fill="%s" stroke="%s" stroke-width="1.5"/>`+
			`<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="%d" font-family="%s" fill="#0f172a">%s</text>`,
		x, py, seqParticipantW, seqParticipantH, fill, stroke,
		cx, py+seqParticipantH/2, defaultFontSize, defaultFontFamily, xmlEscape(label))
}

// seqMessageSVG draws a horizontal arrow between two lifelines.
func seqMessageSVG(fromX, toX, y float64, label string, style ast.MessageStyle) string {
	if math.Abs(fromX-toX) < 1 {
		// Self-message: small loop.
		loopW := 30.0
		loopH := 20.0
		return fmt.Sprintf(
			`<path d="M%.1f,%.1f L%.1f,%.1f L%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#475569" stroke-width="1.5"/>`+
				`<text x="%.1f" y="%.1f" text-anchor="start" font-size="11" font-family="%s" fill="#475569">%s</text>`,
			fromX, y, fromX+loopW, y, fromX+loopW, y+loopH, fromX, y+loopH,
			fromX+loopW+4, y+loopH/2, defaultFontFamily, xmlEscape(label))
	}

	isDashed := seqStyleDashed(style)
	noArrow := seqStyleNoArrow(style)
	crossEnd := seqStyleCrossEnd(style)
	openArrow := seqStyleOpenArrow(style)

	dash := ""
	if isDashed {
		dash = `stroke-dasharray="5,3" `
	}

	direction := 1.0
	if fromX > toX {
		direction = -1.0
	}

	endX := toX
	if !noArrow && !crossEnd {
		endX = toX - direction*arrowLen
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#475569" stroke-width="1.5" %s/>`,
		fromX, y, endX, y, dash)

	if crossEnd {
		// X mark at target end.
		d := 5.0
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f M%.1f,%.1f L%.1f,%.1f" stroke="#ef4444" stroke-width="1.5"/>`,
			toX-d, y-d, toX+d, y+d, toX-d, y+d, toX+d, y-d)
	} else if openArrow {
		// Open (unfilled) arrowhead.
		angle := math.Atan2(0, direction) // 0 or Pi
		cos, sin := math.Cos(angle), math.Sin(angle)
		bx := toX - arrowLen*cos
		by := y - arrowLen*sin
		lx := bx + (arrowWidth/2)*sin
		ly := by - (arrowWidth/2)*cos
		rx := bx - (arrowWidth/2)*sin
		ry := by + (arrowWidth/2)*cos
		fmt.Fprintf(&sb, `<path d="M%.1f,%.1f L%.1f,%.1f M%.1f,%.1f L%.1f,%.1f" fill="none" stroke="#475569" stroke-width="1.5"/>`,
			toX, y, lx, ly, toX, y, rx, ry)
	} else if !noArrow {
		angle := math.Atan2(0, direction)
		sb.WriteString(arrowheadSVG(toX, y, angle, "#475569"))
	}

	// Message label above the arrow.
	if label != "" {
		midX := (fromX + toX) / 2
		fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" font-family="%s" fill="#1e293b">%s</text>`,
			midX, y-6, defaultFontFamily, xmlEscape(label))
	}

	return sb.String()
}

// seqNoteSVG draws a note box spanning x1..x2 at vertical position y.
func seqNoteSVG(x1, x2, y float64, text string) string {
	w := x2 - x1
	if w < 60 {
		w = 60
	}
	return fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3" fill="#fef9c3" stroke="#ca8a04" stroke-width="1"/>`+
			`<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-size="11" font-family="%s" fill="#713f12">%s</text>`,
		x1, y, w, seqNoteH,
		x1+w/2, y+seqNoteH/2, defaultFontFamily, xmlEscape(text))
}

// Message style predicates.
func seqStyleDashed(s ast.MessageStyle) bool {
	return s == ast.MsgAsync || s == ast.MsgAsyncNoArrow || s == ast.MsgAsyncX || s == ast.MsgAsyncOpen
}

func seqStyleNoArrow(s ast.MessageStyle) bool {
	return s == ast.MsgSyncNoArrow || s == ast.MsgAsyncNoArrow
}

func seqStyleCrossEnd(s ast.MessageStyle) bool {
	return s == ast.MsgSyncX || s == ast.MsgAsyncX
}

func seqStyleOpenArrow(s ast.MessageStyle) bool {
	return s == ast.MsgOpen || s == ast.MsgAsyncOpen
}
