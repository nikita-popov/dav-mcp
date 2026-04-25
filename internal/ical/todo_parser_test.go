package ical

import (
	"testing"
	"time"
)

const singleTodo = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VTODO\r\nUID:todo-1@dav-mcp\r\nSUMMARY:Buy groceries\r\nDESCRIPTION:Milk\\, bread\r\nDUE:20260510T180000Z\r\nPRIORITY:5\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

const twoTodos = "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:td1@x\r\nSUMMARY:First\r\nEND:VTODO\r\nBEGIN:VTODO\r\nUID:td2@x\r\nSUMMARY:Second\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

const todoDueDate = "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:td3@x\r\nSUMMARY:All-day task\r\nDUE;VALUE=DATE:20260601\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

const todoNoDue = "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:td4@x\r\nSUMMARY:No deadline\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

const mixedComponents = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:ev@x\r\nSUMMARY:Event\r\nDTSTART:20260501T100000Z\r\nDTEND:20260501T110000Z\r\nEND:VEVENT\r\nBEGIN:VTODO\r\nUID:td@x\r\nSUMMARY:Task\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

func TestParseTodos_Single(t *testing.T) {
	todos := ParseTodos(singleTodo)
	if len(todos) != 1 {
		t.Fatalf("expected 1, got %d", len(todos))
	}
	td := todos[0]
	if td.UID != "todo-1@dav-mcp" {
		t.Errorf("UID=%q", td.UID)
	}
	if td.Summary != "Buy groceries" {
		t.Errorf("Summary=%q", td.Summary)
	}
	if td.Description != "Milk, bread" {
		t.Errorf("Description=%q", td.Description)
	}
	wantDue := time.Date(2026, 5, 10, 18, 0, 0, 0, time.UTC)
	if !td.Due.Equal(wantDue) {
		t.Errorf("Due=%v, want %v", td.Due, wantDue)
	}
	if td.Priority != 5 {
		t.Errorf("Priority=%d", td.Priority)
	}
	if td.Status != "NEEDS-ACTION" {
		t.Errorf("Status=%q", td.Status)
	}
}

func TestParseTodos_Two(t *testing.T) {
	todos := ParseTodos(twoTodos)
	if len(todos) != 2 {
		t.Fatalf("expected 2, got %d", len(todos))
	}
	if todos[0].UID != "td1@x" || todos[1].UID != "td2@x" {
		t.Errorf("UIDs: %q %q", todos[0].UID, todos[1].UID)
	}
}

func TestParseTodos_DueDate(t *testing.T) {
	todos := ParseTodos(todoDueDate)
	if len(todos) != 1 {
		t.Fatalf("expected 1, got %d", len(todos))
	}
	d := todos[0].Due
	if d.Year() != 2026 || d.Month() != 6 || d.Day() != 1 {
		t.Errorf("Due=%v", d)
	}
}

func TestParseTodos_NoDue(t *testing.T) {
	todos := ParseTodos(todoNoDue)
	if len(todos) != 1 {
		t.Fatalf("expected 1, got %d", len(todos))
	}
	if !todos[0].Due.IsZero() {
		t.Errorf("expected zero Due, got %v", todos[0].Due)
	}
}

func TestParseTodos_IgnoresVEVENT(t *testing.T) {
	todos := ParseTodos(mixedComponents)
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].UID != "td@x" {
		t.Errorf("UID=%q", todos[0].UID)
	}
}

func TestParseTodos_Empty(t *testing.T) {
	if todos := ParseTodos(""); len(todos) != 0 {
		t.Errorf("expected 0, got %d", len(todos))
	}
}

// Round-trip: BuildTodo → ParseTodos
func TestBuildTodo_RoundTrip(t *testing.T) {
	orig := Todo{
		UID:         "rt-todo@dav-mcp",
		Summary:     "Round-trip task",
		Description: "Has a, semicolon; and newline\nhere",
		Due:         time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		Priority:    3,
	}
	data := BuildTodo(orig)
	todos := ParseTodos(data)
	if len(todos) != 1 {
		t.Fatalf("expected 1, got %d", len(todos))
	}
	got := todos[0]
	if got.UID != orig.UID {
		t.Errorf("UID: got %q, want %q", got.UID, orig.UID)
	}
	if got.Summary != orig.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, orig.Summary)
	}
	if got.Description != orig.Description {
		t.Errorf("Description: got %q, want %q", got.Description, orig.Description)
	}
	if !got.Due.Equal(orig.Due) {
		t.Errorf("Due: got %v, want %v", got.Due, orig.Due)
	}
	if got.Priority != orig.Priority {
		t.Errorf("Priority: got %d, want %d", got.Priority, orig.Priority)
	}
}
