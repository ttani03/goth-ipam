package handlers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ttani03/goth-ipam/internal/database"
)

// --- Unit tests for pure helper functions ---

func TestCloneIP(t *testing.T) {
	original := net.IP{192, 168, 1, 1}
	clone := cloneIP(original)

	if !original.Equal(clone) {
		t.Errorf("expected clone to equal original, got %v", clone)
	}

	// Mutating the clone should not affect the original
	clone[3] = 99
	if original[3] == 99 {
		t.Error("mutating clone should not affect original")
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		name   string
		input  net.IP
		expect net.IP
	}{
		{
			name:   "simple increment",
			input:  net.IP{192, 168, 1, 1},
			expect: net.IP{192, 168, 1, 2},
		},
		{
			name:   "carry over last octet",
			input:  net.IP{192, 168, 1, 255},
			expect: net.IP{192, 168, 2, 0},
		},
		{
			name:   "carry over multiple octets",
			input:  net.IP{192, 168, 255, 255},
			expect: net.IP{192, 169, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := cloneIP(tt.input)
			incrementIP(ip)
			if !ip.Equal(tt.expect) {
				t.Errorf("incrementIP(%v) = %v, want %v", tt.input, ip, tt.expect)
			}
		})
	}
}

// --- HTTP handler integration tests ---

func TestHandleCreateSubnet_MissingFields(t *testing.T) {
	cleanDB(t)

	tests := []struct {
		name string
		form url.Values
	}{
		{"missing cidr", url.Values{"name": {"test"}}},
		{"missing name", url.Values{"cidr": {"192.168.1.0/24"}}},
		{"missing both", url.Values{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/subnets", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			HandleCreateSubnet(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleCreateSubnet_InvalidCIDR(t *testing.T) {
	cleanDB(t)

	form := url.Values{"cidr": {"not-a-cidr"}, "name": {"test"}}
	req := httptest.NewRequest(http.MethodPost, "/subnets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleCreateSubnet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateSubnet_PrefixTooShort(t *testing.T) {
	cleanDB(t)

	tests := []string{"10.0.0.0/8", "0.0.0.0/0", "172.16.0.0/15"}
	for _, cidr := range tests {
		t.Run(cidr, func(t *testing.T) {
			form := url.Values{"cidr": {cidr}, "name": {"test"}}
			req := httptest.NewRequest(http.MethodPost, "/subnets", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			HandleCreateSubnet(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("CIDR %s: expected 400, got %d", cidr, w.Code)
			}
		})
	}
}

func TestHandleCreateSubnet_Success(t *testing.T) {
	cleanDB(t)

	cidr := "192.168.100.0/30" // /30: 4 addresses → 2 usable IPs
	form := url.Values{"cidr": {cidr}, "name": {"test-subnet"}}
	req := httptest.NewRequest(http.MethodPost, "/subnets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	HandleCreateSubnet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify subnet record was inserted
	var count int
	err := database.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM subnets WHERE cidr = $1", cidr).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query subnets: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 subnet record, got %d", count)
	}

	// Verify IPs were inserted (2 usable IPs for /30)
	var ipCount int
	err = database.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM ips WHERE status = 'available'").Scan(&ipCount)
	if err != nil {
		t.Fatalf("failed to query ips: %v", err)
	}
	if ipCount != 2 {
		t.Errorf("expected 2 IP records for /30, got %d", ipCount)
	}
}

func TestHandleDeleteSubnet_Success(t *testing.T) {
	cleanDB(t)

	// Create a subnet first
	var subnetID string
	err := database.DB.QueryRow(context.Background(),
		"INSERT INTO subnets (cidr, name) VALUES ($1, $2) RETURNING id",
		"10.0.16.0/24", "delete-test",
	).Scan(&subnetID)
	if err != nil {
		t.Fatalf("failed to create test subnet: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/subnets/"+subnetID, nil)
	req.SetPathValue("id", subnetID)
	w := httptest.NewRecorder()

	HandleDeleteSubnet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify subnet was deleted
	var count int
	err = database.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM subnets WHERE id = $1", subnetID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query subnets: %v", err)
	}
	if count != 0 {
		t.Errorf("expected subnet to be deleted, but found %d records", count)
	}
}
