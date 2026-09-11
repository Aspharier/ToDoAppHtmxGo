package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.etcd.io/bbolt"
)

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Time      string `json:"time,omitempty"`
}

var (
	db         *bbolt.DB
	todoBucket = []byte("todos")
)

// itob converts an integer ID to an 8-byte big-endian slice.
// This ensures that bbolt preserves chronological/numerical sorting.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func initDB(dbPath string) (*bbolt.DB, error) {
	database, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bbolt.ErrTimeout {
			return nil, fmt.Errorf("database file '%s' is locked by another running instance of this application. Please close the other instance or stop the existing process", dbPath)
		}
		return nil, err
	}

	err = database.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(todoBucket)
		return err
	})
	if err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func getAllTodos() ([]Todo, error) {
	var todos []Todo
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(todoBucket)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var todo Todo
			if err := json.Unmarshal(v, &todo); err == nil {
				todos = append(todos, todo)
			}
		}
		return nil
	})
	return todos, err
}

func addTodo(title string, timeStr string) (Todo, error) {
	var todo Todo
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(todoBucket)
		id, err := b.NextSequence()
		if err != nil {
			return err
		}

		if timeStr == "" {
			timeStr = time.Now().Format("3:04 PM")
		}

		todo = Todo{
			ID:        int(id),
			Title:     title,
			Completed: false,
			Time:      timeStr,
		}

		data, err := json.Marshal(todo)
		if err != nil {
			return err
		}

		return b.Put(itob(todo.ID), data)
	})
	return todo, err
}

func deleteTodo(id int) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(todoBucket)
		return b.Delete(itob(id))
	})
}

func toggleTodo(id int) (Todo, error) {
	var todo Todo
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(todoBucket)
		data := b.Get(itob(id))
		if data == nil {
			return os.ErrNotExist
		}

		if err := json.Unmarshal(data, &todo); err != nil {
			return err
		}

		todo.Completed = !todo.Completed

		updatedData, err := json.Marshal(todo)
		if err != nil {
			return err
		}

		return b.Put(itob(id), updatedData)
	})
	return todo, err
}

func renderTodoHtml(todo Todo) string {
	completedClass := ""
	circleCheckedClass := ""
	checkIcon := ""
	if todo.Completed {
		completedClass = " is-completed"
		circleCheckedClass = " checked"
		checkIcon = `<svg class="check-svg" viewBox="0 0 24 24" fill="none" stroke="#16a34a" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>`
	}

	timeDisplay := todo.Time
	if timeDisplay == "" {
		timeDisplay = time.Now().Format("3:04 PM")
	}

	escapedTitle := template.HTMLEscapeString(todo.Title)
	escapedTime := template.HTMLEscapeString(timeDisplay)

	return fmt.Sprintf(`
	<div class="todo-item%s" id="todo-%d">
		<input type="hidden" name="id" value="%d">
		<div class="todo-content">
			<span class="todo-title">%s</span>
			<span class="todo-time">%s</span>
		</div>
		<div class="todo-actions">
			<button class="btn-delete"
					hx-post="/api/delete-todo"
					hx-target="#todo-%d"
					hx-swap="outerHTML"
					hx-include="#todo-%d [name=id]"
					title="Delete Task"
					type="button">
				<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M3 6h18"></path>
					<path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path>
					<path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path>
				</svg>
			</button>
			<button class="circle-check%s"
					hx-post="/api/complete-todo"
					hx-target="#todo-%d"
					hx-swap="outerHTML"
					hx-include="#todo-%d [name=id]"
					aria-label="Toggle Complete"
					type="button">
				%s
			</button>
		</div>
	</div>`, completedClass, todo.ID, todo.ID, escapedTitle, escapedTime, todo.ID, todo.ID, circleCheckedClass, todo.ID, todo.ID, checkIcon)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Unable to load index.html", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func getTodosHandler(w http.ResponseWriter, r *http.Request) {
	todos, err := getAllTodos()
	if err != nil {
		http.Error(w, "Unable to fetch TODO items", http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	if len(todos) == 0 {
		sb.WriteString(`<div class="empty-placeholder">No tasks for today. Tap <strong>+ Add Task</strong> below!</div>`)
	} else {
		for _, todo := range todos {
			sb.WriteString(renderTodoHtml(todo))
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(sb.String()))
}

func addTodoHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	timeStr := strings.TrimSpace(r.FormValue("time"))
	if timeStr == "" {
		timeStr = time.Now().Format("3:04 PM")
	}

	todo, err := addTodo(title, timeStr)
	if err != nil {
		http.Error(w, "Unable to add TODO item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(renderTodoHtml(todo)))
}

func deleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := deleteTodo(id); err != nil {
		http.Error(w, "Unable to delete Todo item", http.StatusInternalServerError)
		return
	}

	// Empty body indicates deletion to HTMX (swaps outerHTML with nothing)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(""))
}

func completeTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	todo, err := toggleTodo(id)
	if err != nil {
		if err == os.ErrNotExist {
			http.Error(w, "Todo item not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Unable to update Todo item", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(renderTodoHtml(todo)))
}

// getLocalIP attempts to return the primary local non-loopback IP address.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

func main() {
	var err error
	db, err = initDB("todos.db")
	if err != nil {
		log.Fatalf("Error opening bbolt database: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api/todos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getTodosHandler(w, r)
		case http.MethodPost:
			addTodoHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/delete-todo", deleteTodoHandler)
	mux.HandleFunc("/api/complete-todo", completeTodoHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	localIP := getLocalIP()
	log.Println("==================================================")
	log.Printf("🚀 ToDo App running with fast embedded bbolt database!\n")
	log.Printf("💻 Desktop Access : http://localhost:%s\n", port)
	log.Printf("📱 Phone Access   : http://%s:%s (on same Wi-Fi)\n", localIP, port)
	log.Println("==================================================")

	server := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}

	// Listen for shutdown signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	<-stopChan
	log.Println("\nShutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Database closed. Goodbye!")
}
