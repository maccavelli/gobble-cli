package builtin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/tool"
)

var (
	todoStatuses   = []string{tool.PlanPending, tool.PlanInProgress, tool.PlanCompleted}
	todoPriorities = []string{tool.PriorityHigh, tool.PriorityMedium, tool.PriorityLow}
)

type todoItem struct {
	Content  string `json:"content" jsonschema:"the step, in a few words"`
	Status   string `json:"status" jsonschema:"pending, in_progress or completed"`
	Priority string `json:"priority,omitzero" jsonschema:"high, medium or low"`
}

type todoIn struct {
	Todos []todoItem `json:"todos" jsonschema:"the whole list, in order; an empty list clears it"`
}

// Todo is the todo tool: the task's checklist, which each call replaces
// whole, as codex's update_plan and opencode's todowrite do. The tool
// changes nothing; the list is the result's Plan, which the server shows
// and records (0005-PLAN, entry "F1d made executable").
func Todo() tool.Tool {
	return tool.New("todo", description("todo", Options{}.withDefaults()),
		runTodo,
		tool.WithKind(tool.KindThink),
		tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true, IdempotentHint: true}),
		tool.WithSchema(func(s *jsonschema.Schema) {
			list := property(s, "todos")
			list.MinItems = new(0)
			nonEmpty(list.Items, "content")
			enum(property(list.Items, "status"), todoStatuses)
			priority := property(list.Items, "priority")
			enum(priority, todoPriorities)
			priority.Default = []byte(`"` + tool.PriorityMedium + `"`)
		}),
	)
}

// enum publishes a string property's fixed choices (0012-MADR D2 rule 6).
// The tool's code enforces them (rule 8).
func enum(p *jsonschema.Schema, choices []string) {
	p.Enum = make([]any, len(choices))
	for i, c := range choices {
		p.Enum[i] = c
	}
}

func runTodo(_ context.Context, in todoIn, _ tool.Env) (tool.Result, error) {
	if in.Todos == nil {
		return tool.Result{}, errors.New("todo: todos is null; send the whole list, or [] to clear it")
	}
	plan := make([]tool.PlanItem, len(in.Todos))
	count := map[string]int{}
	for i, it := range in.Todos {
		n := i + 1
		content := strings.TrimSpace(it.Content)
		if content == "" {
			return tool.Result{}, fmt.Errorf("todo: item %d has no content; give each item a few words", n)
		}
		if !slices.Contains(todoStatuses, it.Status) {
			return tool.Result{}, fmt.Errorf("todo: item %d's status %q is not one of %s", n, it.Status, strings.Join(todoStatuses, ", "))
		}
		priority := it.Priority
		if priority == "" {
			priority = tool.PriorityMedium
		}
		if !slices.Contains(todoPriorities, priority) {
			return tool.Result{}, fmt.Errorf("todo: item %d's priority %q is not one of %s", n, it.Priority, strings.Join(todoPriorities, ", "))
		}
		plan[i] = tool.PlanItem{Content: content, Status: it.Status, Priority: priority}
		count[it.Status]++
	}
	text := "todo list cleared"
	if len(plan) > 0 {
		text = fmt.Sprintf("todo list updated: %d completed, %d in progress, %d pending",
			count[tool.PlanCompleted], count[tool.PlanInProgress], count[tool.PlanPending])
	}
	r := tool.TextResult(text)
	r.Plan = plan
	return r, nil
}
