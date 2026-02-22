package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Subnet struct {
	ID        pgtype.UUID `json:"id"`
	CIDR      string      `json:"cidr"`
	Name      string      `json:"name"`
	CreatedAt time.Time   `json:"created_at"`
}

type IP struct {
	ID        pgtype.UUID `json:"id"`
	SubnetID  pgtype.UUID `json:"subnet_id"`
	Address   string      `json:"address"`
	Status    string      `json:"status"`
	Hostname  *string     `json:"hostname"`
	CreatedAt time.Time   `json:"created_at"`
}
