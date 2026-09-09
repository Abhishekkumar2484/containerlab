package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var db *pgxpool.Pool

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
	})
}

func validateUser(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name == "" {
		return "name is required"
	}

	if len(name) > 100 {
		return "name must be 100 characters or less"
	}

	if email == "" {
		return "email is required"
	}

	if len(email) > 255 {
		return "email must be 255 characters or less"
	}

	if !strings.Contains(email, "@") ||
		strings.HasPrefix(email, "@") ||
		strings.HasSuffix(email, "@") {
		return "invalid email format"
	}

	return ""
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

func dbHealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input UserInput

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)

	if errMsg := validateUser(input.Name, input.Email); errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User

	err := db.QueryRow(
		ctx,
		`
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email, created_at
		`,
		input.Name,
		input.Email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}

		log.Println("Failed to create user:", err)
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(
		ctx,
		`
		SELECT id, name, email, created_at
		FROM users
		ORDER BY id
		`,
	)

	if err != nil {
		log.Println("Failed to fetch users:", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var user User

		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
		); err != nil {
			log.Println("Failed to scan user:", err)
			writeError(w, http.StatusInternalServerError, "failed to read users")
			return
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func getUserID(r *http.Request) (int, error) {
	idText := strings.TrimPrefix(r.URL.Path, "/users/")

	if idText == "" || strings.Contains(idText, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.Atoi(idText)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := getUserID(r)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User

	err = db.QueryRow(
		ctx,
		`
		SELECT id, name, email, created_at
		FROM users
		WHERE id = $1
		`,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		log.Println("Failed to fetch user:", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := getUserID(r)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var input UserInput

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)

	if errMsg := validateUser(input.Name, input.Email); errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User

	err = db.QueryRow(
		ctx,
		`
		UPDATE users
		SET name = $1, email = $2
		WHERE id = $3
		RETURNING id, name, email, created_at
		`,
		input.Name,
		input.Email,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		if strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}

		log.Println("Failed to update user:", err)
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := getUserID(r)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := db.Exec(
		ctx,
		"DELETE FROM users WHERE id = $1",
		id,
	)

	if err != nil {
		log.Println("Failed to delete user:", err)
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/users" {
		switch r.Method {
		case http.MethodGet:
			listUsersHandler(w, r)
		case http.MethodPost:
			createUserHandler(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if strings.HasPrefix(r.URL.Path, "/users/") {
		switch r.Method {
		case http.MethodGet:
			getUserHandler(w, r)
		case http.MethodPut:
			updateUserHandler(w, r)
		case http.MethodDelete:
			deleteUserHandler(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	writeError(w, http.StatusNotFound, "route not found")
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		dbURL = "postgres://containerlab:devpassword@db:5432/containerlab"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	var err error

	db, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to create database pool:", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/db-health", dbHealthHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/users/", usersHandler)

	log.Println("ContainerLab API running on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
