package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

// RegionalRepository gerencia as operações de banco de dados para a tabela regionals
type RegionalRepository struct{}

func NewRegionalRepository() *RegionalRepository {
	return &RegionalRepository{}
}

// Create insere uma nova regional no banco de dados
func (r *RegionalRepository) Create(ctx context.Context, reg models.Regional) error {
	query := `
		INSERT INTO regionals (
			id, tenant_id, name, coordinator_name, phone, is_active, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW()
		)
	`
	_, err := database.Pool.Exec(ctx, query,
		reg.ID, reg.TenantID, reg.Name, reg.CoordinatorName, reg.Phone, reg.IsActive,
	)
	return err
}

// GetAll retorna todas as regionais cadastradas, trazendo o nome fantasia do Tenant via JOIN, com filtro opcional por TenantID
func (r *RegionalRepository) GetAll(ctx context.Context, filterTenantID *uuid.UUID) ([]models.Regional, error) {
	query := `
		SELECT r.id, r.tenant_id, t.trade_name AS tenant_trade_name, r.name, r.coordinator_name, r.phone, r.is_active, r.created_at
		FROM regionals r
		INNER JOIN tenants t ON r.tenant_id = t.id
	`
	var args []interface{}
	if filterTenantID != nil {
		query += " WHERE r.tenant_id = $1 "
		args = append(args, *filterTenantID)
	}
	query += " ORDER BY t.trade_name ASC, r.name ASC "

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regionals []models.Regional
	for rows.Next() {
		var reg models.Regional
		err := rows.Scan(
			&reg.ID, &reg.TenantID, &reg.TenantTradeName, &reg.Name, &reg.CoordinatorName, &reg.Phone, &reg.IsActive, &reg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		regionals = append(regionals, reg)
	}

	return regionals, nil
}

// Update atualiza os dados de uma regional existente
func (r *RegionalRepository) Update(ctx context.Context, reg models.Regional) error {
	query := `
		UPDATE regionals SET
			tenant_id = $2, name = $3, coordinator_name = $4, phone = $5, is_active = $6
		WHERE id = $1
	`
	_, err := database.Pool.Exec(ctx, query,
		reg.ID, reg.TenantID, reg.Name, reg.CoordinatorName, reg.Phone, reg.IsActive,
	)
	return err
}

// Delete deleta permanentemente uma regional
func (r *RegionalRepository) Delete(ctx context.Context, id interface{}) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM regionals WHERE id = $1", id)
	return err
}
