package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/ttani03/goth-ipam/internal/database"
	"github.com/ttani03/goth-ipam/internal/handlers"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Initialize Database with retry loop
	var err error
	for i := 0; i < 10; i++ {
		err = database.Connect()
		if err == nil {
			break
		}
		log.Printf("Connecting to database... attempt %d/10", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database after retries: %v", err)
	}
	defer database.Close()

	// Initialize Schema (simple migration for now)
	schema, err := os.ReadFile("internal/database/schema.sql")
	if err == nil {
		_, err = database.DB.Exec(context.Background(), string(schema))
		if err != nil {
			log.Printf("Warning: Failed to execute schema: %v", err)
		}
	} else {
		log.Printf("Warning: Could not read schema.sql: %v", err)
	}

	mux := http.NewServeMux()

	// Static Files - Register more specific patterns first or use exact matches where possible
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Routes - Using Go 1.22+ patterns
	// "GET /{$}" matches ONLY the root path.
	mux.HandleFunc("GET /{$}", handlers.HandleSubnetList)
	mux.HandleFunc("POST /subnets", handlers.HandleCreateSubnet)
	mux.HandleFunc("DELETE /subnets/{id}", handlers.HandleDeleteSubnet)

	mux.HandleFunc("GET /subnets/{id}", handlers.HandleSubnetDetail)
	mux.HandleFunc("POST /subnets/{id}/ips", handlers.HandleAllocateIP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
