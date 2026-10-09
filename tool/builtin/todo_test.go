package builtin

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// todo replaces the list whole: its result counts the items by status and
// carries them as the plan, a missing priority being medium.
func TestTodo(t *testing.T) {
	r := call(t, Todo(), tool.Env{}, map[string]any{"todos": []map[string]string{
		{"content": "read the code", "status": "completed", "priority": "high"},
		{"content": " write the tool ", "status": "in_progress"},
		{"content": "test it", "status": "pending", "priority": "low"},
		{"content": "ship it", "status": "pending"},
	}})
	if r.IsError || r.Text() != "todo list updated: 1 completed, 1 in progress, 2 pending" {
		t.Fatalf("todo = %+v, text %q", r, r.Text())
	}
	want := []tool.PlanItem{
		{Content: "read the code", Status: tool.PlanCompleted, Priority: tool.PriorityHigh},
		{Content: "write the tool", Status: tool.PlanInProgress, Priority: tool.PriorityMedium},
		{Content: "test it", Status: tool.PlanPending, Priority: tool.PriorityLow},
		{Content: "ship it", Status: tool.PlanPending, Priority: tool.PriorityMedium},
	}
	if !slices.Equal(r.Plan, want) {
		t.Fatalf("plan %+v, want %+v", r.Plan, want)
	}
	r = call(t, Todo(), tool.Env{}, map[string]any{"todos": []any{}})
	if r.IsError || r.Text() != "todo list cleared" || r.Plan == nil || len(r.Plan) != 0 {
		t.Fatalf("todo [] = %+v, text %q; want the list cleared, with an empty plan set", r, r.Text())
	}
}

// Each value the schema publishes as invalid is refused in todo's own words
// (0012-MADR D2 rule 8), with no plan.
func TestTodoRefuses(t *testing.T) {
	for args, want := range map[string]string{
		`{"todos":[{"content":"a","status":"done"}]}`:                                       `todo: item 1's status "done" is not one of pending, in_progress, completed`,
		`{"todos":[{"content":"a","status":"pending","priority":"urgent"}]}`:                `todo: item 1's priority "urgent" is not one of high, medium, low`,
		`{"todos":[{"content":"a","status":"pending"},{"content":" ","status":"pending"}]}`: "todo: item 2 has no content; give each item a few words",
		`{"todos":null}`: "todo: todos is null; send the whole list, or [] to clear it",
	} {
		r, err := Todo().Run(t.Context(), tool.Call{ID: "c1", Name: "todo", Args: jsontext.Value(args)}, tool.Env{})
		if err != nil || !r.IsError || r.Text() != want || r.Plan != nil {
			t.Errorf("todo %s = %+v, %v; want the error %q", args, r, err, want)
		}
	}
}

// The schema publishes the choices, the default priority, and minItems 0, as
// an empty list clears.
func TestTodoSchema(t *testing.T) {
	var s struct {
		Required   []string `json:"required"`
		Properties struct {
			Todos struct {
				MinItems *int `json:"minItems"`
				Items    struct {
					Required   []string `json:"required"`
					Properties map[string]struct {
						Enum      []string       `json:"enum"`
						Default   jsontext.Value `json:"default"`
						MinLength *int           `json:"minLength"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"todos"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(Todo().Spec().InputSchema, &s); err != nil {
		t.Fatal(err)
	}
	todos := s.Properties.Todos
	props := todos.Items.Properties
	switch {
	case !slices.Equal(s.Required, []string{"todos"}), todos.MinItems == nil || *todos.MinItems != 0:
		t.Fatalf("todos: required %v, minItems %v", s.Required, todos.MinItems)
	case !slices.Equal(todos.Items.Required, []string{"content", "status"}):
		t.Fatalf("item required %v", todos.Items.Required)
	case props["content"].MinLength == nil || *props["content"].MinLength != 1:
		t.Fatal("content has no minLength 1")
	case !slices.Equal(props["status"].Enum, todoStatuses), !slices.Equal(props["priority"].Enum, todoPriorities):
		t.Fatalf("enums %v, %v", props["status"].Enum, props["priority"].Enum)
	case string(props["priority"].Default) != `"medium"`:
		t.Fatalf("priority default %s", props["priority"].Default)
	}
}
