package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

func TestGridCRUD(t *testing.T) {
	// 1. Inicializa conexão com o Postgres
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste de integração de Grades.")
		return
	}

	ctx := context.Background()
	tenantRepo := NewTenantRepository()
	gridRepo := NewGridRepository()

	// IDs únicos de teste
	tenantID := uuid.New()
	gridID := uuid.New()

	testCNPJ := "98765432000202"
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", testCNPJ)

	// 2. Cria Matriz de Dependência
	logo := "/static/uploads/test_grid.png"
	tenant := models.Tenant{
		ID:             tenantID,
		TradeName:      "Matriz Teste Grades",
		LegalName:      "Matriz Teste Grades S.A.",
		DocumentNumber: testCNPJ,
		ZipCode:        "80000-000",
		Street:         "Rua das Grades",
		Number:         "200",
		District:       "Centro",
		City:           "Curitiba",
		State:          "PR",
		Country:        "Brasil",
		LogoURL:        &logo,
		PrimaryColor:   "#00ff00",
		IsActive:       true,
	}
	err := tenantRepo.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Erro ao criar tenant dependente: %v", err)
	}
	defer tenantRepo.Delete(ctx, tenantID)

	// 3. Define Schedules e Block Configs para a criação
	schedules := map[string]*models.BasicSchedule{
		"mon": {
			TenantID:                    tenantID,
			MediaType:                   models.MediaAudio,
			Name:                        "Grade Teste - Mon Schedule",
			Description:                 "Mon desc",
			StartTime:                   "08:00:00",
			EndTime:                     "18:00:00",
			MusicalBlockDurationMinutes: 60,
		},
		"sat": {
			TenantID:                    tenantID,
			MediaType:                   models.MediaAudio,
			Name:                        "Grade Teste - Sat Schedule",
			Description:                 "Sat desc",
			StartTime:                   "09:00:00",
			EndTime:                     "13:00:00",
			MusicalBlockDurationMinutes: 30,
		},
	}

	configs := map[string]*models.CommercialBlockConfig{
		"mon": {
			TenantID:                   tenantID,
			MediaType:                  models.MediaAudio,
			Name:                       "Grade Teste - Mon Block",
			Description:                "Mon block desc",
			IntervalMinutes:            15,
			BlockLimitSeconds:          120,
			PercentageInstitutional:    50,
			PercentageOffer:            50,
			PercentageTimeAnnouncement: 0,
			PercentageTemperature:      0,
		},
		"sat": {
			TenantID:                   tenantID,
			MediaType:                  models.MediaAudio,
			Name:                       "Grade Teste - Sat Block",
			Description:                "Sat block desc",
			IntervalMinutes:            20,
			BlockLimitSeconds:          90,
			PercentageInstitutional:    60,
			PercentageOffer:            40,
			PercentageTimeAnnouncement: 0,
			PercentageTemperature:      0,
		},
	}

	grid := &models.WeeklyMasterGrid{
		ID:          gridID,
		TenantID:    tenantID,
		MediaType:   models.MediaAudio,
		Name:        "Grade de Teste Geral",
		Description: "Descrição da grade de teste geral",
	}

	// 4. Teste de Gravação (Create)
	err = gridRepo.Create(ctx, grid, schedules, configs)
	if err != nil {
		t.Fatalf("Erro ao criar grade: %v", err)
	}

	// 5. Teste de Recuperação (GetByID)
	gLoaded, schedsLoaded, configsLoaded, err := gridRepo.GetByID(ctx, gridID)
	if err != nil {
		t.Fatalf("Erro ao carregar grade: %v", err)
	}

	if gLoaded.Name != "Grade de Teste Geral" {
		t.Errorf("Nome esperado 'Grade de Teste Geral', obteve '%s'", gLoaded.Name)
	}

	if gLoaded.MondayScheduleID == nil {
		t.Error("MondayScheduleID nulo após gravação")
	}

	if schedsLoaded["mon"] == nil || !strings.HasPrefix(schedsLoaded["mon"].StartTime, "08:00:00") {
		t.Errorf("StartTime esperado prefixo '08:00:00', obteve '%s'", schedsLoaded["mon"].StartTime)
	}

	if configsLoaded["sat"] == nil || configsLoaded["sat"].IntervalMinutes != 20 {
		t.Errorf("IntervalMinutes no sábado esperado 20, obteve '%v'", configsLoaded["sat"])
	}

	if gLoaded.SundayScheduleID != nil {
		t.Error("SundayScheduleID não deveria estar preenchido para domingo inativo")
	}

	if gLoaded.HolidayScheduleID != nil {
		t.Error("HolidayScheduleID não deveria estar preenchido para feriado inativo")
	}

	// 6. Teste de Atualização (Update)
	// Adiciona Domingo, Feriado e altera Segunda-feira
	schedules["sun"] = &models.BasicSchedule{
		TenantID:                    tenantID,
		MediaType:                   models.MediaAudio,
		Name:                        "Grade Teste - Sun Schedule",
		StartTime:                   "10:00:00",
		EndTime:                     "14:00:00",
		MusicalBlockDurationMinutes: 45,
	}
	configs["sun"] = &models.CommercialBlockConfig{
		TenantID:                   tenantID,
		MediaType:                  models.MediaAudio,
		Name:                       "Grade Teste - Sun Block",
		IntervalMinutes:            30,
		BlockLimitSeconds:          60,
		PercentageInstitutional:    100,
		PercentageOffer:            0,
	}

	schedules["holiday"] = &models.BasicSchedule{
		TenantID:                    tenantID,
		MediaType:                   models.MediaAudio,
		Name:                        "Grade Teste - Holiday Schedule",
		StartTime:                   "09:00:00",
		EndTime:                     "15:00:00",
		MusicalBlockDurationMinutes: 60,
	}
	configs["holiday"] = &models.CommercialBlockConfig{
		TenantID:                   tenantID,
		MediaType:                  models.MediaAudio,
		Name:                       "Grade Teste - Holiday Block",
		IntervalMinutes:            15,
		BlockLimitSeconds:          120,
		PercentageInstitutional:    70,
		PercentageOffer:            30,
	}

	schedules["mon"].EndTime = "19:00:00"

	// Remove Saturday (desabilita)
	delete(schedules, "sat")
	delete(configs, "sat")

	gLoaded.Name = "Grade de Teste Atualizada"
	err = gridRepo.Update(ctx, gLoaded, schedules, configs)
	if err != nil {
		t.Fatalf("Erro ao atualizar grade: %v", err)
	}

	// Recarrega para validar atualização
	gUpdated, schedsUpdated, _, err := gridRepo.GetByID(ctx, gridID)
	if err != nil {
		t.Fatalf("Erro ao carregar grade atualizada: %v", err)
	}

	if gUpdated.Name != "Grade de Teste Atualizada" {
		t.Errorf("Nome esperado 'Grade de Teste Atualizada', obteve '%s'", gUpdated.Name)
	}

	if schedsUpdated["mon"] == nil || !strings.HasPrefix(schedsUpdated["mon"].EndTime, "19:00:00") {
		t.Errorf("EndTime da segunda esperado prefixo '19:00:00', obteve '%s'", schedsUpdated["mon"].EndTime)
	}

	if gUpdated.SundayScheduleID == nil {
		t.Error("SundayScheduleID deveria estar preenchido após habilitar domingo")
	}

	if schedsUpdated["sun"] == nil || !strings.HasPrefix(schedsUpdated["sun"].StartTime, "10:00:00") {
		t.Errorf("StartTime de domingo esperado prefixo '10:00:00', obteve '%s'", schedsUpdated["sun"].StartTime)
	}

	if gUpdated.HolidayScheduleID == nil {
		t.Error("HolidayScheduleID deveria estar preenchido após habilitar feriado")
	}

	if schedsUpdated["holiday"] == nil || !strings.HasPrefix(schedsUpdated["holiday"].StartTime, "09:00:00") {
		t.Errorf("StartTime de feriado esperado prefixo '09:00:00', obteve '%s'", schedsUpdated["holiday"].StartTime)
	}

	if gUpdated.SaturdayScheduleID != nil {
		t.Error("SaturdayScheduleID deveria ser nulo após desabilitar sábado")
	}

	// 7. Teste de Listagem (GetAll)
	allGrids, err := gridRepo.GetAll(ctx, &tenantID)
	if err != nil {
		t.Fatalf("Erro ao listar grades: %v", err)
	}

	if len(allGrids) != 1 {
		t.Errorf("Esperava exatamente 1 grade cadastrada para o tenant, obteve %d", len(allGrids))
	}

	// 8. Teste de Exclusão (Delete)
	err = gridRepo.Delete(ctx, gridID)
	if err != nil {
		t.Fatalf("Erro ao deletar grade: %v", err)
	}

	// Tenta carregar novamente, deve dar erro pgx.ErrNoRows
	_, _, _, err = gridRepo.GetByID(ctx, gridID)
	if err == nil {
		t.Error("Grade ainda encontrada no banco após exclusão")
	}
}
