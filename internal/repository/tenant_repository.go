package repository

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"imc-server/internal/database"
	"imc-server/internal/models"
)

// TenantRepository gerencia as operações de banco de dados para a tabela tenants
type TenantRepository struct{}

func NewTenantRepository() *TenantRepository {
	return &TenantRepository{}
}

// Create insere uma nova empresa no banco de dados
func (r *TenantRepository) Create(ctx context.Context, tenant models.Tenant) error {

	// Padrão Ouro: Converte os arrays de contatos estruturados em formato JSON
	// para que o PostgreSQL grave lindamente na coluna JSONB
	opContactsJSON, _ := json.Marshal(tenant.OperationalContacts)
	finContactsJSON, _ := json.Marshal(tenant.FinancialContacts)

	query := `
		INSERT INTO tenants (
			id, trade_name, legal_name, document_number,
			operational_contacts, financial_contacts,
			zip_code, street, number, district, city, state, country, complement,
			logo_url, primary_color, background_color, font_color, field_background_color,
			default_audio_grid_id, default_video_grid_id,
			is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19,
			$20, $21,
			$22, NOW(), NOW()
		)
	`

	// Executa a query passando os JSONs como string para evitar erros de tipagem no driver
	_, err := database.Pool.Exec(ctx, query,
		tenant.ID, tenant.TradeName, tenant.LegalName, tenant.DocumentNumber,
		string(opContactsJSON), string(finContactsJSON),
		tenant.ZipCode, tenant.Street, tenant.Number, tenant.District, tenant.City, tenant.State, tenant.Country, tenant.Complement,
		tenant.LogoURL, tenant.PrimaryColor, tenant.BackgroundColor, tenant.FontColor, tenant.FieldBackgroundColor,
		tenant.DefaultAudioGridID, tenant.DefaultVideoGridID,
		tenant.IsActive,
	)

	return err
}

// GetAll retorna a lista de todas as empresas cadastradas
func (r *TenantRepository) GetAll(ctx context.Context) ([]models.Tenant, error) {
	query := `
		SELECT id, trade_name, legal_name, document_number,
			   operational_contacts, financial_contacts,
			   zip_code, street, COALESCE(number, '') AS number, COALESCE(district, '') AS district, city, state, country, complement,
			   logo_url, primary_color, background_color, font_color, field_background_color,
			   default_audio_grid_id, default_video_grid_id,
			   is_active, created_at, updated_at
		FROM tenants
		ORDER BY trade_name ASC
	`

	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var t models.Tenant

		// Variáveis temporárias para receber os JSONs puros do banco
		var opContactsJSON, finContactsJSON []byte

		err := rows.Scan(
			&t.ID, &t.TradeName, &t.LegalName, &t.DocumentNumber,
			&opContactsJSON, &finContactsJSON,
			&t.ZipCode, &t.Street, &t.Number, &t.District, &t.City, &t.State, &t.Country, &t.Complement,
			&t.LogoURL, &t.PrimaryColor, &t.BackgroundColor, &t.FontColor, &t.FieldBackgroundColor,
			&t.DefaultAudioGridID, &t.DefaultVideoGridID,
			&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Faz o caminho reverso: Lê o JSON do banco e transforma de volta na lista (slice) do Go
		if len(opContactsJSON) > 0 && string(opContactsJSON) != "null" {
			json.Unmarshal(opContactsJSON, &t.OperationalContacts)
		}
		if len(finContactsJSON) > 0 && string(finContactsJSON) != "null" {
			json.Unmarshal(finContactsJSON, &t.FinancialContacts)
		}

		tenants = append(tenants, t)
	}

	return tenants, nil
}

// Update atualiza os dados de uma empresa existente, preservando o logotipo antigo caso um novo não seja enviado
func (r *TenantRepository) Update(ctx context.Context, tenant models.Tenant) error {
	opContactsJSON, _ := json.Marshal(tenant.OperationalContacts)
	finContactsJSON, _ := json.Marshal(tenant.FinancialContacts)

	query := `
		UPDATE tenants SET
			trade_name = $2, legal_name = $3, document_number = $4,
			operational_contacts = $5, financial_contacts = $6,
			zip_code = $7, street = $8, number = $9, district = $10, city = $11, state = $12, country = $13, complement = $14,
			primary_color = $15, background_color = $16, font_color = $17, field_background_color = $18,
			logo_url = COALESCE($19, logo_url),
			default_audio_grid_id = $20, default_video_grid_id = $21,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := database.Pool.Exec(ctx, query,
		tenant.ID, tenant.TradeName, tenant.LegalName, tenant.DocumentNumber,
		string(opContactsJSON), string(finContactsJSON),
		tenant.ZipCode, tenant.Street, tenant.Number, tenant.District, tenant.City, tenant.State, tenant.Country, tenant.Complement,
		tenant.PrimaryColor, tenant.BackgroundColor, tenant.FontColor, tenant.FieldBackgroundColor,
		tenant.LogoURL,
		tenant.DefaultAudioGridID, tenant.DefaultVideoGridID,
	)
	return err
}

// GetByID busca uma matriz/cliente pelo seu UUID
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	query := `
		SELECT id, trade_name, legal_name, document_number,
			   operational_contacts, financial_contacts,
			   zip_code, street, COALESCE(number, '') AS number, COALESCE(district, '') AS district, city, state, country, complement,
			   logo_url, primary_color, background_color, font_color, field_background_color,
			   default_audio_grid_id, default_video_grid_id,
			   is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1
		LIMIT 1
	`
	var t models.Tenant
	var opContactsJSON, finContactsJSON []byte
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.TradeName, &t.LegalName, &t.DocumentNumber,
		&opContactsJSON, &finContactsJSON,
		&t.ZipCode, &t.Street, &t.Number, &t.District, &t.City, &t.State, &t.Country, &t.Complement,
		&t.LogoURL, &t.PrimaryColor, &t.BackgroundColor, &t.FontColor, &t.FieldBackgroundColor,
		&t.DefaultAudioGridID, &t.DefaultVideoGridID,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(opContactsJSON) > 0 && string(opContactsJSON) != "null" {
		json.Unmarshal(opContactsJSON, &t.OperationalContacts)
	}
	if len(finContactsJSON) > 0 && string(finContactsJSON) != "null" {
		json.Unmarshal(finContactsJSON, &t.FinancialContacts)
	}
	return &t, nil
}

// Delete remove permanentemente uma empresa do banco (deletando filiais/regionais vinculadas em cascata)
func (r *TenantRepository) Delete(ctx context.Context, id interface{}) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
	return err
}
