package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_todos.db")
	var err error
	db, err = initDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})
}

func TestAddAndGetAllTodos(t *testing.T) {
	setupTestDB(t)

	// Initially empty
	todos, err := getAllTodos()
	if err != nil {
		t.Fatalf("getAllTodos failed: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("Expected 0 todos, got %d", len(todos))
	}

	// Add todo
	todo1, err := addTodo("Buy milk", "8:00 AM")
	if err != nil {
		t.Fatalf("addTodo failed: %v", err)
	}
	if todo1.ID != 1 || todo1.Title != "Buy milk" || todo1.Completed || todo1.Time != "8:00 AM" {
		t.Errorf("Unexpected todo item: %+v", todo1)
	}

	todo2, err := addTodo("Walk dog", "")
	if err != nil {
		t.Fatalf("addTodo failed: %v", err)
	}
	if todo2.ID != 2 {
		t.Errorf("Expected sequential ID 2, got %d", todo2.ID)
	}

	todos, err = getAllTodos()
	if err != nil {
		t.Fatalf("getAllTodos failed: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(todos))
	}
}

func TestToggleTodo(t *testing.T) {
	setupTestDB(t)

	todo, err := addTodo("Test Toggle", "10:00 AM")
	if err != nil {
		t.Fatalf("addTodo failed: %v", err)
	}

	// Toggle to completed
	toggled, err := toggleTodo(todo.ID)
	if err != nil {
		t.Fatalf("toggleTodo failed: %v", err)
	}
	if !toggled.Completed {
		t.Errorf("Expected Completed=true, got false")
	}

	// Toggle back to not completed
	toggledAgain, err := toggleTodo(todo.ID)
	if err != nil {
		t.Fatalf("toggleTodo again failed: %v", err)
	}
	if toggledAgain.Completed {
		t.Errorf("Expected Completed=false, got true")
	}
}

func TestDeleteTodo(t *testing.T) {
	setupTestDB(t)

	todo, err := addTodo("To Delete", "1:00 PM")
	if err != nil {
		t.Fatalf("addTodo failed: %v", err)
	}

	err = deleteTodo(todo.ID)
	if err != nil {
		t.Fatalf("deleteTodo failed: %v", err)
	}

	todos, err := getAllTodos()
	if err != nil {
		t.Fatalf("getAllTodos failed: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("Expected 0 todos after delete, got %d", len(todos))
	}
}

func TestRenderTodoHtml(t *testing.T) {
	todo := Todo{
		ID:        42,
		Title:     "<script>alert('xss')</script> Clean House",
		Completed: false,
		Time:      "5:00 PM",
	}

	html := renderTodoHtml(todo)
	if strings.Contains(html, "<script>") {
		t.Errorf("HTML contains unescaped script tag, potential XSS vulnerability")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("Expected HTML escaped script tag in output")
	}
	if !strings.Contains(html, "todo-42") {
		t.Errorf("Expected todo-42 id in output")
	}

	todo.Completed = true
	completedHtml := renderTodoHtml(todo)
	if !strings.Contains(completedHtml, "is-completed") {
		t.Errorf("Expected is-completed class in output")
	}
	if !strings.Contains(completedHtml, "check-svg") {
		t.Errorf("Expected check-svg in completed output")
	}
}
