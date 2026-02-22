package handlers

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ttani03/goth-ipam/internal/database"
)

var pgContainer testcontainers.Container

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start PostgreSQL container
	ctr, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("ipam_test"),
		postgres.WithUsername("ipam_user"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	pgContainer = ctr
	defer func() {
		if err := ctr.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %v", err)
		}
	}()

	// Get connection string and set DATABASE_URL
	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}
	os.Setenv("DATABASE_URL", connStr)

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	// Apply schema
	schema, err := os.ReadFile("../../internal/database/schema.sql")
	if err != nil {
		log.Fatalf("failed to read schema.sql: %v", err)
	}
	if _, err := database.DB.Exec(ctx, string(schema)); err != nil {
		log.Fatalf("failed to apply schema: %v", err)
	}

	// Run tests
	os.Exit(m.Run())
}

// cleanDB truncates all tables to ensure a clean state for each test.
func cleanDB(t *testing.T) {
	t.Helper()
	_, err := database.DB.Exec(context.Background(), "TRUNCATE TABLE ips, subnets RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to clean database: %v", err)
	}
}
