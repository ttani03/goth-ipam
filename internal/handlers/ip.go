package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/ttani03/goth-ipam/internal/database"
	"github.com/ttani03/goth-ipam/internal/models"
	"github.com/ttani03/goth-ipam/internal/templates"
)

func HandleSubnetDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var subnet models.Subnet
	err := database.DB.QueryRow(context.Background(), "SELECT id, cidr, name, created_at FROM subnets WHERE id = $1", id).Scan(&subnet.ID, &subnet.CIDR, &subnet.Name, &subnet.CreatedAt)
	if err != nil {
		http.Error(w, "Subnet not found", http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(context.Background(), "SELECT id, subnet_id, address, status, hostname, created_at FROM ips WHERE subnet_id = $1 ORDER BY address::inet", id)
	if err != nil {
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

	// Fetch available (unassigned) IPs for the allocate dropdown
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

	component := templates.SubnetDetail(subnet, ips, availableIPs)
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
