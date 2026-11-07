package models

import (
	"time"

	"github.com/google/uuid"
)

// Regional espelha a tabela de subdivisões de um Tenant no banco de dados.
type Regional struct {
	ID              uuid.UUID `db:"id" json:"id"`
	TenantID        uuid.UUID `db:"tenant_id" json:"tenant_id"`
	TenantTradeName string    `db:"tenant_trade_name" json:"tenant_trade_name"` // Campo auxiliar para JOIN em listagens
	Name            string    `db:"name" json:"name"`
	CoordinatorName *string   `db:"coordinator_name" json:"coordinator_name,omitempty"` // Ponteiro para suportar NULL de forma limpa
	Phone           *string   `db:"phone" json:"phone,omitempty"`
	IsActive        bool      `db:"is_active" json:"is_active"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}
