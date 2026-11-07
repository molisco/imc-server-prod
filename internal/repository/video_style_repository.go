package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

type VideoStyleRepository struct{}

func NewVideoStyleRepository() *VideoStyleRepository {
	return &VideoStyleRepository{}
}

func (r *VideoStyleRepository) Create(ctx context.Context, style models.VideoStyle) error {
	query := `
		INSERT INTO video_styles (
			id, name, description, base_directory, created_at
		) VALUES (
			$1, $2, $3, $4, NOW()
		)
	`
	_, err := database.Pool.Exec(ctx, query,
		style.ID, style.Name, style.Description, style.BaseDirectory,
	)
	return err
}

func (r *VideoStyleRepository) GetAll(ctx context.Context) ([]models.VideoStyle, error) {
	query := `
		SELECT id, name, COALESCE(description, '') AS description, 
		       COALESCE(base_directory, '') AS base_directory, created_at
		FROM video_styles
		ORDER BY name ASC
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var styles []models.VideoStyle
	for rows.Next() {
		var s models.VideoStyle
		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.BaseDirectory, &s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		styles = append(styles, s)
	}

	return styles, nil
}

func (r *VideoStyleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.VideoStyle, error) {
	query := `
		SELECT id, name, COALESCE(description, '') AS description, 
		       COALESCE(base_directory, '') AS base_directory, created_at
		FROM video_styles
		WHERE id = $1
	`
	var s models.VideoStyle
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.BaseDirectory, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *VideoStyleRepository) Update(ctx context.Context, style models.VideoStyle) error {
	query := `
		UPDATE video_styles SET
			name = $2, description = $3, base_directory = $4
		WHERE id = $1
	`
	_, err := database.Pool.Exec(ctx, query,
		style.ID, style.Name, style.Description, style.BaseDirectory,
	)
	return err
}

func (r *VideoStyleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM video_styles WHERE id = $1", id)
	return err
}
