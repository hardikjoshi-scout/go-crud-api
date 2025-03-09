package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// User represents the user model for our CRUD operations
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

var db *sql.DB

func main() {
	// Initialize database connection
	var err error
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/go_crud_api")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/users", getUsers).Methods("GET")          // Fetch all users
	router.HandleFunc("/user/{id}", getUser).Methods("GET")       // Fetch a user by ID
	router.HandleFunc("/user", createUser).Methods("POST")        // Create a new user
	router.HandleFunc("/user/{id}", updateUser).Methods("PUT")    // Update a user by ID
	router.HandleFunc("/user/{id}", deleteUser).Methods("DELETE") // Delete a user by ID

	// Start server on port 8000
	log.Fatal(http.ListenAndServe(":8000", router))
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	var users []User
	rows, err := db.Query("SELECT id, name, email, created_at FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	var user User
	err := db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	json.NewEncoder(w).Encode(user)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 1. Empty Field Validation
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.TrimSpace(user.Email)
	if user.Name == "" || user.Email == "" {
		http.Error(w, "Name and Email are required", http.StatusBadRequest)
		return
	}

	// 2. Email Format Validation
	if !isValidEmail(user.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// 3. Check for Duplicate Email
	var existingID int
	err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if existingID > 0 {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// 4. Insert User into Database
	result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", user.Name, user.Email)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			http.Error(w, "Duplicate entry detected", http.StatusConflict)
		} else {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
		}
		return
	}

	// 5. Get the New User's ID
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to retrieve last insert ID", http.StatusInternalServerError)
		return
	}

	user.ID = int(id)
	user.CreatedAt = "now"            // Placeholder, replace with actual timestamp if needed
	w.WriteHeader(http.StatusCreated) // 201 Created status
	json.NewEncoder(w).Encode(user)
}

// Helper function to validate email format using regex
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	idStr := params["id"] // 'id' is a string
	log.Println("Request Method:", r.Method)
	log.Println("Request URL:", r.URL)
	log.Println("Request Headers:", r.Header)
	// Convert 'id' to an integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Execute the update query
	result, err := db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", user.Name, user.Email, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Set the ID properly and respond
	user.ID = id
	user.CreatedAt = "now" // Placeholder, adjust as needed
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
