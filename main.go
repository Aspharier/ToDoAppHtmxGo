package main

// Importing the libraries
import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	_ "github.com/go-sql-driver/mysql"
)

type Todo struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Completed bool `json:"completed"`
}

var db *sql.DB
fun main() {
	var err error 
	dsn := "root:Thecityofroma@123@tcp(localhost:3306)/todo_app"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/todos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getTodosHandler(w, r)
		} else if r.Method == http.MethodPost {
			addTodoHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/delete-todo", deleteTodoHandler)
	http.HandleFunc("/api/complete-todo", completeTodoHandler)
	log.Println("Server is running on http://localhost:8080")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}


func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Unable to load index.html", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func renderTodoHtml(todo Todo) string {
	completedStatus := ""
	bgColor := "white"
	buttonText := "Complete"
	if todo.Completed {
		completedStatus = " (Completed)"
		bgColor = "#f0f0f0" // light grey background for completed tasks
		buttonText = "Uncomplete"
	}
	return fmt.Sprintf(`
	<div class="todo-item" id="todo-%d" style="background-color: %s;">
	<p><strong>%s</strong>%s</p>
	<button hx-post="/api/delete-todo"
	hx-target="#todo-%d"
	hx-swap="outerHTML"
	hx-include="#todo-%d [name=id]"
	type="button">
	Delete
	</button>
	<button hx-post="/api/complete-todo"
	hx-target="#todo-%d"
	hx-swap="outerHTML"
	hx-include="#todo-%d [name=id]"
	type="button">
								%s
	</button>
	<input type="hidden" name="id" value="%d">
	</div>`, todo.ID, bgColor, todo.Title, completedStatus, todo.ID, todo.ID, todo.ID, todo.ID, buttonText, todo.ID)
}

func getTodosHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, completed FROM todos")
	if err != nil {
		http.Error(w, "Unable to fetch TODO items", http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed); err != nil {
			http.Error(w, "Error reading TODO items", http.StatusInternalServerError)
			return
		}

		todos = append(todos, todo)
	}

	var html string
	for _, todo := range todos {
		html += renderTodoHtml(todo)
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func addTodoHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Insert new TOdo into the database
	result, err := db.Exec("INSERT INTO todos (title, completed) VALUES (?, false)", title)
	if err != nil {
		http.Error(w, "Unable to add TODO item", http.StatusInternalServerError)
		return
	}

	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Unable to fetch inserted ID", http.StatusInternalServerError)
		return
	}

	// Fetch the newly added ToDo from the database
	todo := Todo {
		ID: int(id),
		Title: title,
		Completed: false,
	}

	// Render the newly added ToDo item as HTMl
	html := renderTodoHtml(todo)
	// Return the generated HTML for the new todo
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}


// Detel Todo Handler deletes a ToDo item by ID.
func deleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Execute the delete query
	_, err := db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Unable to delete Todo Item", http.StatusInternalServerError)
		return
	}

	// Respond with an empty string to indicate successful deletion.
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(""))
}

// Complete Todo Handler toggles the completed status of todo item by ID.
func completeTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Toggle the completed status
	var completed bool
	err := db.QueryRow("SELECT completed FROM todos WHERE id = ?", id).Scan(&completed)
	if err == sql.ErrNoRows {
		http.Error(w, "TODO item not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Unable to fetch TODO item", http.StatusInternalServerError)
		return
	}

	// Update the completed status
	_, err = db.Exec("UPDATE todos SET completed = ? WHERE id = ?", !completed, id)
	if err != nil {
		http.Error(w, "Unable to update TODO item", http.StatusInternalServerError)
		return
	}

	// Fetch the updated Todo item
	var todo Todo
	err = db.QueryRow("SELECT id, title, completed FROM todos WHERE id = ?", id).Scan(&todo.ID, &todo.Title, &todo.completed)
	if err == sql.ErrNoRows {
		http.Error(w, "Update todo item not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Unable to fetch updated TODO Item", http.StatusInternalServerError)
		return
	}

	// Render and return the updated ToDo item's HTML
	html := renderTodoHtml(todo)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

