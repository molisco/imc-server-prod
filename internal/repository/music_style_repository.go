package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

type MusicStyleRepository struct{}

func NewMusicStyleRepository() *MusicStyleRepository {
	return &MusicStyleRepository{}
}

func (r *MusicStyleRepository) Create(ctx context.Context, style models.MusicStyle) error {
	if style.SourceType == "" {
		style.SourceType = "LOCAL"
	}
	query := `
		INSERT INTO music_styles (
			id, name, description, base_directory, source_type, stream_url, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW(), NOW()
		)
	`
	_, err := database.Pool.Exec(ctx, query,
		style.ID, style.Name, style.Description, style.BaseDirectory, style.SourceType, style.StreamURL,
	)
	return err
}

func (r *MusicStyleRepository) GetAll(ctx context.Context) ([]models.MusicStyle, error) {
	query := `
		SELECT id, name, COALESCE(description, '') AS description, 
		       COALESCE(base_directory, '') AS base_directory,
		       COALESCE(source_type, 'LOCAL') AS source_type,
		       COALESCE(stream_url, '') AS stream_url,
		       created_at, updated_at
		FROM music_styles
		ORDER BY name ASC
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var styles []models.MusicStyle
	for rows.Next() {
		var s models.MusicStyle
		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.BaseDirectory, &s.SourceType, &s.StreamURL, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}

	return styles, nil
}

func (r *MusicStyleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MusicStyle, error) {
	query := `
		SELECT id, name, COALESCE(description, '') AS description, 
		       COALESCE(base_directory, '') AS base_directory,
		       COALESCE(source_type, 'LOCAL') AS source_type,
		       COALESCE(stream_url, '') AS stream_url,
		       created_at, updated_at
		FROM music_styles
		WHERE id = $1
	`
	var s models.MusicStyle
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.BaseDirectory, &s.SourceType, &s.StreamURL, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MusicStyleRepository) Update(ctx context.Context, style models.MusicStyle) error {
	if style.SourceType == "" {
		style.SourceType = "LOCAL"
	}
	query := `
		UPDATE music_styles SET
			name = $2, description = $3, base_directory = $4, source_type = $5, stream_url = $6, updated_at = NOW()
		WHERE id = $1
	`
	_, err := database.Pool.Exec(ctx, query,
		style.ID, style.Name, style.Description, style.BaseDirectory, style.SourceType, style.StreamURL,
	)
	return err
}

func (r *MusicStyleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM music_styles WHERE id = $1", id)
	return err
}

func (r *MusicStyleRepository) UpdateTimestamp(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "UPDATE music_styles SET updated_at = NOW() WHERE id = $1", id)
	return err
}
