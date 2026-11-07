package repository

import (
	"context"
	"github.com/google/uuid"
	"imc-server/internal/database"
	"imc-server/internal/models"
)

// StoreRepository gerencia as operações de banco de dados para a tabela stores
type StoreRepository struct{}

func NewStoreRepository() *StoreRepository {
	return &StoreRepository{}
}

// Create insere uma nova filial no banco de dados, gerando um token automático
func (r *StoreRepository) Create(ctx context.Context, store models.Store) error {
	if store.ID == uuid.Nil {
		store.ID = uuid.New()
	}
	// Padrão Ouro: Gera o token UUID único de conexão para o Player C#
	if store.Token == "" {
		store.Token = uuid.New().String()
	}
	if store.RadioType == "" {
		store.RadioType = "server"
	}
	if store.Timezone == "" {
		store.Timezone = "America/Sao_Paulo"
	}

	query := `
		INSERT INTO stores (
			id, tenant_id, regional_id, store_name, store_number,
			zip_code, street, number, district, city, state, timezone, phone, manager_name,
			has_audio, has_video, token, is_active, audio_grid_id, video_grid_id, radio_type, harbor_port,
			external_stream_url, external_stream_name, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NOW(), NOW()
		)
	`
	_, err := database.Pool.Exec(ctx, query,
		store.ID, store.TenantID, store.RegionalID, store.StoreName, store.StoreNumber,
		store.ZipCode, store.Street, store.Number, store.District, store.City, store.State, store.Timezone, store.Phone, store.ManagerName,
		store.HasAudio, store.HasVideo, store.Token, store.IsActive, store.AudioGridID, store.VideoGridID, store.RadioType, store.HarborPort,
		store.ExternalStreamURL, store.ExternalStreamName,
	)
	return err
}

// GetAll retorna todas as lojas cadastradas com JOINs, suportando filtro opcional de TenantID
func (r *StoreRepository) GetAll(ctx context.Context, filterTenantID *uuid.UUID) ([]models.Store, error) {
	query := `
		SELECT s.id, s.tenant_id, t.trade_name AS tenant_trade_name, s.regional_id, r.name AS regional_name,
		       s.store_name, s.store_number, COALESCE(s.zip_code, '') AS zip_code, COALESCE(s.street, '') AS street, COALESCE(s.number, '') AS number, COALESCE(s.district, '') AS district, s.city, s.state, s.timezone,
		       COALESCE(s.phone, '') AS phone, COALESCE(s.manager_name, '') AS manager_name, s.has_audio, s.has_video,
		       s.last_audio_sync, s.last_video_sync, s.token, s.audio_grid_id, s.video_grid_id,
		       ga.name AS audio_grid_name, gv.name AS video_grid_name, s.is_active, s.created_at, s.updated_at,
		       s.radio_type, s.harbor_port, s.external_stream_url, s.external_stream_name
		FROM stores s
		INNER JOIN tenants t ON s.tenant_id = t.id
		LEFT JOIN regionals r ON s.regional_id = r.id
		LEFT JOIN weekly_master_grids ga ON s.audio_grid_id = ga.id
		LEFT JOIN weekly_master_grids gv ON s.video_grid_id = gv.id
	`
	var args []interface{}
	if filterTenantID != nil {
		query += " WHERE s.tenant_id = $1 "
		args = append(args, *filterTenantID)
	}
	query += " ORDER BY t.trade_name ASC, COALESCE(r.name, '') ASC, s.store_name ASC"

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []models.Store
	for rows.Next() {
		var s models.Store
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.TenantTradeName, &s.RegionalID, &s.RegionalName,
			&s.StoreName, &s.StoreNumber, &s.ZipCode, &s.Street, &s.Number, &s.District, &s.City, &s.State, &s.Timezone,
			&s.Phone, &s.ManagerName, &s.HasAudio, &s.HasVideo,
			&s.LastAudioSync, &s.LastVideoSync, &s.Token, &s.AudioGridID, &s.VideoGridID,
			&s.AudioGridName, &s.VideoGridName, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
			&s.RadioType, &s.HarborPort, &s.ExternalStreamURL, &s.ExternalStreamName,
		)
		if err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}

	return stores, nil
}

// Update atualiza os dados cadastrais de uma filial
func (r *StoreRepository) Update(ctx context.Context, store models.Store) error {
	if store.RadioType == "" {
		store.RadioType = "server"
	}
	if store.Timezone == "" {
		store.Timezone = "America/Sao_Paulo"
	}
	query := `
		UPDATE stores SET
			tenant_id = $2, regional_id = $3, store_name = $4, store_number = $5,
			zip_code = $6, street = $7, number = $8, district = $9, city = $10, state = $11, timezone = $12,
			phone = $13, manager_name = $14, has_audio = $15, has_video = $16,
			is_active = $17, audio_grid_id = $18, video_grid_id = $19, radio_type = $20, harbor_port = $21,
			external_stream_url = $22, external_stream_name = $23, updated_at = NOW()
		WHERE id = $1
	`
	_, err := database.Pool.Exec(ctx, query,
		store.ID, store.TenantID, store.RegionalID, store.StoreName, store.StoreNumber,
		store.ZipCode, store.Street, store.Number, store.District, store.City, store.State, store.Timezone,
		store.Phone, store.ManagerName, store.HasAudio, store.HasVideo,
		store.IsActive, store.AudioGridID, store.VideoGridID, store.RadioType, store.HarborPort,
		store.ExternalStreamURL, store.ExternalStreamName,
	)
	return err
}

// GetByID busca uma loja pelo seu UUID
func (r *StoreRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Store, error) {
	query := `
		SELECT s.id, s.tenant_id, t.trade_name AS tenant_trade_name, s.regional_id, r.name AS regional_name,
		       s.store_name, s.store_number, COALESCE(s.zip_code, '') AS zip_code, COALESCE(s.street, '') AS street, COALESCE(s.number, '') AS number, COALESCE(s.district, '') AS district, s.city, s.state, s.timezone,
		       COALESCE(s.phone, '') AS phone, COALESCE(s.manager_name, '') AS manager_name, s.has_audio, s.has_video,
		       s.last_audio_sync, s.last_video_sync, s.token, s.audio_grid_id, s.video_grid_id,
		       ga.name AS audio_grid_name, gv.name AS video_grid_name, s.is_active, s.created_at, s.updated_at,
		       s.radio_type, s.harbor_port, s.external_stream_url, s.external_stream_name
		FROM stores s
		INNER JOIN tenants t ON s.tenant_id = t.id
		LEFT JOIN regionals r ON s.regional_id = r.id
		LEFT JOIN weekly_master_grids ga ON s.audio_grid_id = ga.id
		LEFT JOIN weekly_master_grids gv ON s.video_grid_id = gv.id
		WHERE s.id = $1
		LIMIT 1
	`
	var s models.Store
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.TenantTradeName, &s.RegionalID, &s.RegionalName,
		&s.StoreName, &s.StoreNumber, &s.ZipCode, &s.Street, &s.Number, &s.District, &s.City, &s.State, &s.Timezone,
		&s.Phone, &s.ManagerName, &s.HasAudio, &s.HasVideo,
		&s.LastAudioSync, &s.LastVideoSync, &s.Token, &s.AudioGridID, &s.VideoGridID,
		&s.AudioGridName, &s.VideoGridName, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		&s.RadioType, &s.HarborPort, &s.ExternalStreamURL, &s.ExternalStreamName,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete remove uma loja permanentemente
func (r *StoreRepository) Delete(ctx context.Context, id interface{}) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM stores WHERE id = $1", id)
	return err
}
