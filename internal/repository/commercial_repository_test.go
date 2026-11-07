package repository

import (
	"context"
	"testing"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

func TestCommercialRepository(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste.")
		return
	}

	ctx := context.Background()
	repo := NewCommercialRepository()

	// 1. Create mock commercial
	cID := uuid.New()
	tenantID := uuid.New() // Generate a dummy tenant ID
	startDate := time.Now()
	endDate := time.Now().Add(48 * time.Hour)

	c := models.CommercialsAndEvents{
		ID:               cID,
		TenantID:         tenantID,
		MediaType:        models.MediaAudio,
		CommercialType:   models.CommercialStandard,
		Name:             "Black Friday Spot 30s",
		FileName:         "bf_spot_30s.mp3",
		DurationSeconds:  30,
		InsertionsPerDay: 8,
		StartDate:        &startDate,
		EndDate:          &endDate,
		StartTime:        "08:00:00",
		EndTime:          "20:00:00",
		RunOnSunday:      false,
		RunOnMonday:      true,
		RunOnTuesday:     true,
		RunOnWednesday:   true,
		RunOnThursday:    true,
		RunOnFriday:      true,
		RunOnSaturday:    true,
		DistributeToAll:  true,
		IsActive:         true,
	}

	// Create dummy tenant first to prevent FK violation
	_, _ = database.Pool.Exec(ctx, `INSERT INTO tenants (id, trade_name, legal_name, document_number, zip_code, street, city, state, is_active) 
		VALUES ($1, 'Tenant Test', 'Tenant Test LTDA', $2, '00000', 'Street', 'City', 'ST', true) ON CONFLICT DO NOTHING`, tenantID, "99.999.999/9999-99")

	// Make sure we delete any old remains
	_ = repo.Delete(ctx, cID)

	// Test Create
	err := repo.Create(ctx, c)
	if err != nil {
		t.Fatalf("Failed to create commercial: %v", err)
	}

	// Test GetByID
	fetched, err := repo.GetByID(ctx, cID)
	if err != nil {
		t.Fatalf("Failed to fetch commercial: %v", err)
	}
	if fetched.Name != c.Name {
		t.Errorf("Expected name '%s', got '%s'", c.Name, fetched.Name)
	}
	if fetched.InsertionsPerDay != 8 {
		t.Errorf("Expected 8 insertions, got %d", fetched.InsertionsPerDay)
	}

	// Test Update
	fetched.Name = "Black Friday Spot 30s - Updated"
	fetched.InsertionsPerDay = 12
	fetched.DistributeToAll = false
	err = repo.Update(ctx, *fetched)
	if err != nil {
		t.Fatalf("Failed to update commercial: %v", err)
	}

	fetched2, _ := repo.GetByID(ctx, cID)
	if fetched2.Name != "Black Friday Spot 30s - Updated" {
		t.Errorf("Expected updated name, got '%s'", fetched2.Name)
	}
	if fetched2.InsertionsPerDay != 12 {
		t.Errorf("Expected updated insertions 12, got %d", fetched2.InsertionsPerDay)
	}
	if fetched2.DistributeToAll != false {
		t.Errorf("Expected distribute_to_all to be false")
	}

	// Test GetByTenant
	list, err := repo.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("Failed to get commercials by tenant: %v", err)
	}
	if len(list) == 0 {
		t.Errorf("Expected to find commercial in tenant list, but list was empty")
	}

	// Test SaveDistribution
	storeID1 := uuid.New()
	storeID2 := uuid.New()
	
	// Create dummy stores first to prevent FK violation
	_, _ = database.Pool.Exec(ctx, `INSERT INTO stores (id, tenant_id, store_name, city, state, token, is_active) 
		VALUES ($1, $2, 'Store Test 1', 'City', 'ST', 'token_t1', true) ON CONFLICT DO NOTHING`, storeID1, tenantID)
	_, _ = database.Pool.Exec(ctx, `INSERT INTO stores (id, tenant_id, store_name, city, state, token, is_active) 
		VALUES ($1, $2, 'Store Test 2', 'City', 'ST', 'token_t2', true) ON CONFLICT DO NOTHING`, storeID2, tenantID)

	err = repo.SaveDistribution(ctx, cID, []uuid.UUID{storeID1, storeID2})
	if err != nil {
		t.Fatalf("Failed to save distribution: %v", err)
	}

	// Test GetDistributedStores
	storeIDs, err := repo.GetDistributedStores(ctx, cID)
	if err != nil {
		t.Fatalf("Failed to get distributed stores: %v", err)
	}
	if len(storeIDs) != 2 {
		t.Errorf("Expected 2 distributed stores, got %d", len(storeIDs))
	}

	// Cleanup distribution test data
	_, _ = database.Pool.Exec(ctx, "DELETE FROM stores WHERE id IN ($1, $2)", storeID1, storeID2)
	_, _ = database.Pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)

	// Test Delete
	err = repo.Delete(ctx, cID)
	if err != nil {
		t.Fatalf("Failed to delete commercial: %v", err)
	}

	_, err = repo.GetByID(ctx, cID)
	if err == nil {
		t.Errorf("Expected commercial to be deleted, but it was found in database")
	}
}
