package render

import (
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// Plan-entry statuses, as ACP names them.
const (
	PlanPending    = "pending"
	PlanInProgress = "in_progress"
	PlanCompleted  = "completed"
)

// PlanItem is one step of the agent's plan.
type PlanItem struct {
	Content, Status string
}

// Plan draws the agent's plan as a checklist, one step per line: done
// steps checked in green, the running one arrowed in bold, the rest dim
// (0008-MADR D7). The content is sanitised.
func Plan(items []PlanItem, s term.Stream, g term.Glyphs) string {
	var b strings.Builder
	for _, it := range items {
		text := strings.Join(strings.Fields(term.Sanitize(it.Content)), " ")
		var mark string
		var style term.Style
		switch it.Status {
		case PlanCompleted:
			mark, style = g.Check, term.Green
		case PlanInProgress:
			mark, style = g.Arrow, term.Bold
		default:
			mark, style = g.Bullet, term.Dim
		}
		b.WriteString("  " + style.Wrap(mark+" "+text, s.Color) + "\n")
	}
	return b.String()
}
