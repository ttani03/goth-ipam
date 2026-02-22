package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ttani03/goth-ipam/internal/database"
)

// --- Unit tests for hostnameRegex ---

func TestHostnameRegex_Valid(t *testing.T) {
	valid := []string{
		"myhost",
		"my-host",
		"my-host-01",
		"myhost.example.com",
		"a",
		"a1",
		"foo.bar.baz",
	}
	for _, h := range valid {
		t.Run(h, func(t *testing.T) {
			if !hostnameRegex.MatchString(h) {
				t.Errorf("expected %q to be a valid hostname, but it did not match", h)
			}
		})
	}
}

func TestHostnameRegex_Invalid(t *testing.T) {
	invalid := []string{
		"-invalid",       // starts with hyphen
		"invalid-",       // ends with hyphen
		"invalid..host",  // double dot
		".invalid",       // starts with dot
		"invalid.",       // ends with dot
		"has space",      // space not allowed
		"has_underscore", // underscore not allowed
		"",               // empty
	}
	for _, h := range invalid {
		t.Run(h, func(t *testing.T) {
			if hostnameRegex.MatchString(h) {
				t.Errorf("expected %q to be an invalid hostname, but it matched", h)
			}
		})
	}
}

// --- HTTP handler integration tests ---

func TestHandleAllocateIP_InvalidHostname(t *testing.T) {
	cleanDB(t)

	// Create test subnet + IP
	var subnetID string
	if err := database.DB.QueryRow(context.Background(),
		"INSERT INTO subnets (cidr, name) VALUES ($1, $2) RETURNING id",
		"10.0.16.0/24", "alloc-test",
	).Scan(&subnetID); err != nil {
		t.Fatalf("failed to insert subnet: %v", err)
	}
	if _, err := database.DB.Exec(context.Background(),
		"INSERT INTO ips (subnet_id, address, status) VALUES ($1, $2, 'available')",
		subnetID, "10.0.16.1",
	); err != nil {
		t.Fatalf("failed to insert IP: %v", err)
	}

	form := url.Values{
		"address":  {"10.0.16.1"},
		"hostname": {"-invalid-hostname"},
	}
	req := httptest.NewRequest(http.MethodPost, "/subnets/"+subnetID+"/ips", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", subnetID)
	w := httptest.NewRecorder()

	HandleAllocateIP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAllocateIP_NotAvailable(t *testing.T) {
	cleanDB(t)

	// Create test subnet
	var subnetID string
	if err := database.DB.QueryRow(context.Background(),
		"INSERT INTO subnets (cidr, name) VALUES ($1, $2) RETURNING id",
		"10.0.16.0/24", "alloc-test",
	).Scan(&subnetID); err != nil {
		t.Fatalf("failed to insert subnet: %v", err)
	}

	// Try to allocate a non-existent address
	form := url.Values{
		"address":  {"10.0.16.99"},
		"hostname": {"myhost"},
	}
	req := httptest.NewRequest(http.MethodPost, "/subnets/"+subnetID+"/ips", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", subnetID)
	w := httptest.NewRecorder()

	HandleAllocateIP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestHandleAllocateIP_Success(t *testing.T) {
	cleanDB(t)

	// Create test subnet + available IP
	var subnetID string
	if err := database.DB.QueryRow(context.Background(),
		"INSERT INTO subnets (cidr, name) VALUES ($1, $2) RETURNING id",
		"10.0.16.0/24", "alloc-test",
	).Scan(&subnetID); err != nil {
		t.Fatalf("failed to insert subnet: %v", err)
	}
	if _, err := database.DB.Exec(context.Background(),
		"INSERT INTO ips (subnet_id, address, status) VALUES ($1, $2, 'available')",
		subnetID, "10.0.16.1",
	); err != nil {
		t.Fatalf("failed to insert IP: %v", err)
	}

	form := url.Values{
		"address":  {"10.0.16.1"},
		"hostname": {"myhost"},
	}
	req := httptest.NewRequest(http.MethodPost, "/subnets/"+subnetID+"/ips", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", subnetID)
	w := httptest.NewRecorder()

	HandleAllocateIP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}

	// Verify that the IP status was updated to 'allocated'
	var status string
	var hostname *string
	err := database.DB.QueryRow(context.Background(),
		"SELECT status, hostname FROM ips WHERE subnet_id = $1 AND address = $2",
		subnetID, "10.0.16.1",
	).Scan(&status, &hostname)
	if err != nil {
		t.Fatalf("failed to query IP: %v", err)
	}
	if status != "allocated" {
		t.Errorf("expected status 'allocated', got %q", status)
	}
	if hostname == nil || *hostname != "myhost" {
		t.Errorf("expected hostname 'myhost', got %v", hostname)
	}
}
