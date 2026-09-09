package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
	})
}

func dbHealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(
			r.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)

			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  "database unavailable",
			})

			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	}
}

func createUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var user User

		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		err := pool.QueryRow(
			ctx,
			`INSERT INTO users (name, email)
			 VALUES ($1, $2)
			 RETURNING id, name, email, created_at`,
			user.Name,
			user.Email,
		).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
		)

		if err != nil {
			log.Println("Failed to create user:", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

func getUsersHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		rows, err := pool.Query(
			ctx,
			`SELECT id, name, email, created_at
			 FROM users
			 ORDER BY id`,
		)

		if err != nil {
			log.Println("Failed to fetch users:", err)
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}

		defer rows.Close()

		users := []User{}

		for rows.Next() {
			var user User

			if err := rows.Scan(
				&user.ID,
				&user.Name,
				&user.Email,
				&user.CreatedAt,
			); err != nil {
				log.Println("Failed to scan user:", err)
				http.Error(w, "Failed to read users", http.StatusInternalServerError)
				return
			}

			users = append(users, user)
		}

		if err := rows.Err(); err != nil {
			log.Println("Database rows error:", err)
			http.Error(w, "Failed to read users", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(users)
	}
}

func getUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		id := r.URL.Path[len("/users/"):]

		ctx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		var user User

		err := pool.QueryRow(
			ctx,
			`SELECT id, name, email, created_at
			 FROM users
			 WHERE id = $1`,
			id,
		).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
		)

		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

func updateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		id := r.URL.Path[len("/users/"):]

		var input struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		var user User

		err := pool.QueryRow(
			ctx,
			`UPDATE users
			 SET name = $1, email = $2
			 WHERE id = $3
			 RETURNING id, name, email, created_at`,
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
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

func deleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		id := r.URL.Path[len("/users/"):]

		ctx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		result, err := pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			id,
		)

		if err != nil {
			log.Println("Failed to delete user:", err)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected() == 0 {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "deleted",
		})
	}
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

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to create database pool:", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL")

	http.HandleFunc("/health", healthHandler)

	http.HandleFunc("/db-health", dbHealthHandler(pool))

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createUserHandler(pool)(w, r)

		case http.MethodGet:
			getUsersHandler(pool)(w, r)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUserHandler(pool)(w, r)

		case http.MethodPut:
			updateUserHandler(pool)(w, r)

		case http.MethodDelete:
			deleteUserHandler(pool)(w, r)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("ContainerLab API running on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
