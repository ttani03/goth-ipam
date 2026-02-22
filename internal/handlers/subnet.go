package handlers

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/ttani03/goth-ipam/internal/database"
	"github.com/ttani03/goth-ipam/internal/models"
	"github.com/ttani03/goth-ipam/internal/templates"
)

func HandleSubnetList(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query(context.Background(), "SELECT id, cidr, name, created_at FROM subnets ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch subnets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var subnets []models.Subnet
	for rows.Next() {
		var s models.Subnet
		// Assuming scanning works fine here correctly mapping types
		if err := rows.Scan(&s.ID, &s.CIDR, &s.Name, &s.CreatedAt); err != nil {
			continue
		}
		subnets = append(subnets, s)
	}

	component := templates.SubnetList(subnets)
	component.Render(context.Background(), w)
}

// maxPrefixLen is the minimum prefix length (= narrowest allowed subnet) for IPv4.
// Subnets broader than /8 (> 16 million hosts) are rejected to prevent resource exhaustion.
const minIPv4Prefix = 8

func HandleCreateSubnet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	cidr := r.FormValue("cidr")
	name := r.FormValue("name")

	// Basic input validation
	if cidr == "" || name == "" {
		http.Error(w, "cidr and name are required", http.StatusBadRequest)
		return
	}

	// Validate CIDR format and prefix length before any DB write
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Printf("Invalid CIDR %s: %v", cidr, err)
		http.Error(w, "Invalid CIDR format", http.StatusBadRequest)
		return
	}
	if ipNet.IP.To4() != nil {
		ones, _ := ipNet.Mask.Size()
		if ones < minIPv4Prefix {
			http.Error(w, fmt.Sprintf("CIDR prefix must be /%d or longer (e.g. /8, /24)", minIPv4Prefix), http.StatusBadRequest)
			return
		}
	}

	// Insert subnet and get generated ID
	var subnetID string
	if err := database.DB.QueryRow(context.Background(),
		"INSERT INTO subnets (cidr, name) VALUES ($1, $2) RETURNING id",
		cidr, name).Scan(&subnetID); err != nil {
		log.Printf("Error creating subnet: %v", err)
		http.Error(w, "Failed to create subnet", http.StatusInternalServerError)
		return
	}

	// Collect all IPs in range (excluding network and broadcast for IPv4)
	var addresses []string
	for cur := cloneIP(ip.Mask(ipNet.Mask)); ipNet.Contains(cur); incrementIP(cur) {
		addresses = append(addresses, cur.String())
	}
	// Remove network address (first) and broadcast address (last) for IPv4
	if len(addresses) > 2 && ipNet.IP.To4() != nil {
		addresses = addresses[1 : len(addresses)-1]
	}

	for _, addr := range addresses {
		if _, err := database.DB.Exec(context.Background(),
			"INSERT INTO ips (subnet_id, address, status) VALUES ($1, $2, 'available')",
			subnetID, addr); err != nil {
			log.Printf("Error inserting IP %s: %v", addr, err)
		}
	}

	// Return updated list
	HandleSubnetList(w, r)
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func HandleDeleteSubnet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+

	_, err := database.DB.Exec(context.Background(), "DELETE FROM subnets WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting subnet: %v", err)
		http.Error(w, "Failed to delete subnet", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
