package handlers

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ttani03/goth-ipam/internal/database"
	"github.com/ttani03/goth-ipam/internal/models"
	"github.com/ttani03/goth-ipam/internal/templates"
)

// hostnameRegex validates RFC 1123 hostnames (labels separated by dots,
// each label: 1–63 alphanumeric chars or hyphens, not starting/ending with a hyphen).
var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

// validPageSizes lists the allowed per-page values for IP list pagination.
var validPageSizes = map[int]bool{30: true, 50: true, 100: true}

func HandleSubnetDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// --- pagination parameters ---
	pageSize := 30
	if ps, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && validPageSizes[ps] {
		pageSize = ps
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	offset := (page - 1) * pageSize

	// Optional status filter (empty = all)
	statusFilter := r.URL.Query().Get("status")

	var subnet models.Subnet
	err := database.DB.QueryRow(context.Background(), "SELECT id, cidr, name, created_at FROM subnets WHERE id = $1", id).Scan(&subnet.ID, &subnet.CIDR, &subnet.Name, &subnet.CreatedAt)
	if err != nil {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	// --- count total IPs (for pagination meta) ---
	var totalCount int
	if statusFilter != "" && statusFilter != "all" {
		err = database.DB.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM ips WHERE subnet_id = $1 AND status = $2", id, statusFilter).Scan(&totalCount)
	} else {
		err = database.DB.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM ips WHERE subnet_id = $1", id).Scan(&totalCount)
	}
	if err != nil {
		http.Error(w, "Failed to count IPs", http.StatusInternalServerError)
		return
	}

	// --- fetch paginated IPs ---
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var queryErr error
	if statusFilter != "" && statusFilter != "all" {
		rows, queryErr = database.DB.Query(context.Background(),
			"SELECT id, subnet_id, address, status, hostname, created_at FROM ips WHERE subnet_id = $1 AND status = $2 ORDER BY address::inet LIMIT $3 OFFSET $4",
			id, statusFilter, pageSize, offset)
	} else {
		rows, queryErr = database.DB.Query(context.Background(),
			"SELECT id, subnet_id, address, status, hostname, created_at FROM ips WHERE subnet_id = $1 ORDER BY address::inet LIMIT $2 OFFSET $3",
			id, pageSize, offset)
	}
	if queryErr != nil {
		http.Error(w, "Failed to fetch IPs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var ips []models.IP
	for rows.Next() {
		var ip models.IP
		if err := rows.Scan(&ip.ID, &ip.SubnetID, &ip.Address, &ip.Status, &ip.Hostname, &ip.CreatedAt); err != nil {
			continue
		}
		ips = append(ips, ip)
	}

	// Fetch available (unassigned) IPs for the allocate dropdown (no pagination needed here)
	availRows, err := database.DB.Query(context.Background(), "SELECT id, subnet_id, address, status, hostname, created_at FROM ips WHERE subnet_id = $1 AND status = 'available' ORDER BY address::inet", id)
	if err != nil {
		http.Error(w, "Failed to fetch available IPs", http.StatusInternalServerError)
		return
	}
	defer availRows.Close()

	var availableIPs []models.IP
	for availRows.Next() {
		var ip models.IP
		if err := availRows.Scan(&ip.ID, &ip.SubnetID, &ip.Address, &ip.Status, &ip.Hostname, &ip.CreatedAt); err != nil {
			continue
		}
		availableIPs = append(availableIPs, ip)
	}

	// Build pagination metadata
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	pagination := templates.PaginationMeta{
		Page:         page,
		PageSize:     pageSize,
		TotalCount:   totalCount,
		TotalPages:   totalPages,
		StatusFilter: statusFilter,
	}

	component := templates.SubnetDetail(subnet, ips, availableIPs, pagination)
	component.Render(context.Background(), w)
}

func HandleAllocateIP(w http.ResponseWriter, r *http.Request) {
	subnetID := r.PathValue("id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	address := r.FormValue("address")
	hostname := r.FormValue("hostname")

	if hostname != "" && !hostnameRegex.MatchString(hostname) {
		http.Error(w, "Invalid hostname format", http.StatusBadRequest)
		return
	}

	var hostnameArg interface{}
	if hostname != "" {
		hostnameArg = hostname
	}

	result, err := database.DB.Exec(context.Background(),
		"UPDATE ips SET status = 'allocated', hostname = $1 WHERE subnet_id = $2 AND address = $3 AND status = 'available'",
		hostnameArg, subnetID, address)
	if err != nil {
		log.Printf("Error allocating IP: %v", err)
		http.Error(w, "Failed to allocate IP", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "IP address not available or not found", http.StatusConflict)
		return
	}

	http.Redirect(w, r, "/subnets/"+subnetID, http.StatusSeeOther)
}
