package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"
	"testing"

	"github.com/google/uuid"
)

func TestVideoStyleCRUD(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("Database not available. Skipping integration test.")
		return
	}

	repo := NewVideoStyleRepository()
	ctx := context.Background()

	// Clean up potential leftovers
	database.Pool.Exec(ctx, "DELETE FROM video_styles WHERE name = $1", "Video Style Test Unitario")
	database.Pool.Exec(ctx, "DELETE FROM video_styles WHERE name = $1", "Video Style Test Unitario Alterado")

	id := uuid.New()

	style := models.VideoStyle{
		ID:            id,
		Name:          "Video Style Test Unitario",
		Description:   "Description for unit test",
		BaseDirectory: "/media/video/test_unit",
	}

	// 1. Create
	err := repo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create video style: %v", err)
	}

	// 2. Read (GetByID)
	fetched, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("Failed to retrieve video style: %v", err)
	}
	if fetched.Name != "Video Style Test Unitario" {
		t.Errorf("Expected name 'Video Style Test Unitario', got '%s'", fetched.Name)
	}

	// 3. GetAll
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve all video styles: %v", err)
	}
	var found bool
	for _, item := range all {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find created video style in GetAll, but it was not present")
	}

	// 4. Update
	style.Name = "Video Style Test Unitario Alterado"
	style.Description = "Updated description"
	err = repo.Update(ctx, style)
	if err != nil {
		t.Fatalf("Failed to update video style: %v", err)
	}

	fetched2, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("Failed to retrieve updated video style: %v", err)
	}
	if fetched2.Name != "Video Style Test Unitario Alterado" {
		t.Errorf("Expected updated name 'Video Style Test Unitario Alterado', got '%s'", fetched2.Name)
	}

	// 5. Delete
	err = repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Failed to delete video style: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, id)
	if err == nil {
		t.Errorf("Expected error retrieving deleted video style, but it was found")
	}
}
