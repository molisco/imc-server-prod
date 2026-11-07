package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"
	"testing"

	"github.com/google/uuid"
)

func TestMusicStyleCRUD(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("Database not available. Skipping integration test.")
		return
	}

	repo := NewMusicStyleRepository()
	ctx := context.Background()

	// Clean up potential leftovers
	database.Pool.Exec(ctx, "DELETE FROM music_styles WHERE name = $1", "Music Style Test Unitario")
	database.Pool.Exec(ctx, "DELETE FROM music_styles WHERE name = $1", "Music Style Test Unitario Alterado")

	id := uuid.New()

	style := models.MusicStyle{
		ID:            id,
		Name:          "Music Style Test Unitario",
		Description:   "Description for unit test",
		BaseDirectory: "/media/audio/test_unit",
	}

	// 1. Create
	err := repo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create music style: %v", err)
	}

	// 2. Read (GetByID)
	fetched, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("Failed to retrieve music style: %v", err)
	}
	if fetched.Name != "Music Style Test Unitario" {
		t.Errorf("Expected name 'Music Style Test Unitario', got '%s'", fetched.Name)
	}

	// 3. GetAll
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve all music styles: %v", err)
	}
	var found bool
	for _, item := range all {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find created music style in GetAll, but it was not present")
	}

	// 4. Update
	style.Name = "Music Style Test Unitario Alterado"
	style.Description = "Updated description"
	err = repo.Update(ctx, style)
	if err != nil {
		t.Fatalf("Failed to update music style: %v", err)
	}

	fetched2, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("Failed to retrieve updated music style: %v", err)
	}
	if fetched2.Name != "Music Style Test Unitario Alterado" {
		t.Errorf("Expected updated name 'Music Style Test Unitario Alterado', got '%s'", fetched2.Name)
	}

	// 5. Delete
	err = repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Failed to delete music style: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, id)
	if err == nil {
		t.Errorf("Expected error retrieving deleted music style, but it was found")
	}
}
