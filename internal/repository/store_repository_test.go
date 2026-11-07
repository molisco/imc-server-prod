package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

func TestStoreCRUD(t *testing.T) {
	// 1. Inicializa conexão com o Postgres
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste de integração de Lojas.")
		return
	}

	ctx := context.Background()
	tenantRepo := NewTenantRepository()
	regionalRepo := NewRegionalRepository()
	repo := NewStoreRepository()

	// IDs únicos de teste
	tenantID := uuid.New()
	regionalID := uuid.New()
	storeID := uuid.New()
	storeSemRegID := uuid.New()

	// CNPJ temporário único de teste
	testCNPJ := "98765432000101" // CNPJ matematicamente inválido mas único para limpeza prévia
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", testCNPJ)

	// 2. Cria Matriz de Dependência
	logo := "/static/uploads/test.png"
	tenant := models.Tenant{
		ID:             tenantID,
		TradeName:      "Matriz Teste Lojas",
		LegalName:      "Matriz Teste Lojas S.A.",
		DocumentNumber: testCNPJ,
		ZipCode:        "80000-000",
		Street:         "Rua das Lojas",
		Number:         "100",
		District:       "Centro",
		City:           "Curitiba",
		State:          "PR",
		Country:        "Brasil",
		LogoURL:        &logo,
		PrimaryColor:   "#ff0000",
		IsActive:       true,
	}
	err := tenantRepo.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Erro ao criar tenant dependente: %v", err)
	}
	defer tenantRepo.Delete(ctx, tenantID) // Deletando em cascata no fim limpará a regional e loja também!

	// 3. Cria Regional de Dependência
	coordName := "Coordenador Teste Loja"
	phone := "41999998888"
	regional := models.Regional{
		ID:              regionalID,
		TenantID:        tenantID,
		Name:            "Regional Teste Lojas",
		CoordinatorName: &coordName,
		Phone:           &phone,
		IsActive:        true,
	}
	err = regionalRepo.Create(ctx, regional)
	if err != nil {
		t.Fatalf("Erro ao criar regional dependente: %v", err)
	}

	// Inserir grades semanais fictícias para teste
	audioGridID := uuid.New()
	videoGridID := uuid.New()

	database.Pool.Exec(ctx, 
		"INSERT INTO weekly_master_grids (id, tenant_id, media_type, name, description) VALUES ($1, $2, 'AUDIO', 'Grade Teste Audio', 'Desc')",
		audioGridID, tenantID,
	)
	defer database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", audioGridID)

	database.Pool.Exec(ctx, 
		"INSERT INTO weekly_master_grids (id, tenant_id, media_type, name, description) VALUES ($1, $2, 'VIDEO', 'Grade Teste Video', 'Desc')",
		videoGridID, tenantID,
	)
	defer database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", videoGridID)

	// 4. Teste de Gravação (Create) da Filial/Loja com Regional
	store := models.Store{
		ID:          storeID,
		TenantID:    tenantID,
		RegionalID:  &regionalID,
		AudioGridID: &audioGridID,
		VideoGridID: &videoGridID,
		StoreName:   "Filial Centro Loja 01",
		StoreNumber: "001",
		ZipCode:     "80010-010",
		Street:      "Rua XV de Novembro",
		Number:      "500",
		District:    "Centro Cívico",
		City:        "Curitiba",
		State:       "PR",
		Timezone:    "America/Sao_Paulo",
		Phone:       "4132221111",
		ManagerName: "Gerente Marcos",
		HasAudio:    true,
		HasVideo:    false,
		IsActive:    true,
	}
	err = repo.Create(ctx, store)
	if err != nil {
		t.Fatalf("Falha ao gravar filial/loja: %v", err)
	}

	// 4.1 Teste de Gravação (Create) de Filial sem Regional (RegionalID nil)
	storeSemReg := models.Store{
		ID:          storeSemRegID,
		TenantID:    tenantID,
		RegionalID:  nil, // Opcional (nil)
		StoreName:   "Filial Sem Regional",
		StoreNumber: "002",
		City:        "Londrina",
		State:       "PR",
		Timezone:    "America/Noronha",
		IsActive:    true,
	}
	err = repo.Create(ctx, storeSemReg)
	if err != nil {
		t.Fatalf("Falha ao gravar filial sem regional: %v", err)
	}

	// 5. Teste de Leitura (Read/GetAll) com Filtros
	// 5.1 Busca Geral (sem filtro de tenant)
	list, err := repo.GetAll(ctx, nil)
	if err != nil {
		t.Fatalf("Falha ao listar todas as lojas: %v", err)
	}

	var found, foundSemReg bool
	for _, item := range list {
		if item.ID == storeID {
			found = true
			if item.TenantTradeName != "Matriz Teste Lojas" {
				t.Errorf("Esperado TenantTradeName 'Matriz Teste Lojas', recebido '%s'", item.TenantTradeName)
			}
			if item.RegionalName == nil || *item.RegionalName != "Regional Teste Lojas" {
				t.Errorf("Esperado RegionalName 'Regional Teste Lojas'")
			}
			if item.AudioGridID == nil || *item.AudioGridID != audioGridID {
				t.Errorf("Esperado AudioGridID %v, recebido %v", audioGridID, item.AudioGridID)
			}
			if item.AudioGridName == nil || *item.AudioGridName != "Grade Teste Audio" {
				t.Errorf("Esperado AudioGridName 'Grade Teste Audio', recebido %v", item.AudioGridName)
			}
			if item.VideoGridID == nil || *item.VideoGridID != videoGridID {
				t.Errorf("Esperado VideoGridID %v, recebido %v", videoGridID, item.VideoGridID)
			}
			if item.VideoGridName == nil || *item.VideoGridName != "Grade Teste Video" {
				t.Errorf("Esperado VideoGridName 'Grade Teste Video', recebido %v", item.VideoGridName)
			}
			if item.District != "Centro Cívico" {
				t.Errorf("Esperado District 'Centro Cívico', recebido '%s'", item.District)
			}
			if item.Timezone != "America/Sao_Paulo" {
				t.Errorf("Esperado Timezone 'America/Sao_Paulo', recebido '%s'", item.Timezone)
			}
			if item.ManagerName != "Gerente Marcos" {
				t.Errorf("Esperado ManagerName 'Gerente Marcos', recebido '%s'", item.ManagerName)
			}
			if item.Token == "" {
				t.Errorf("Esperado token gerado automaticamente, mas está vazio")
			}
		}
		if item.ID == storeSemRegID {
			foundSemReg = true
			if item.RegionalID != nil {
				t.Errorf("Esperado RegionalID nil para filial sem regional")
			}
			if item.RegionalName != nil && *item.RegionalName != "" {
				t.Errorf("Esperado RegionalName nil/vazio para filial sem regional")
			}
			if item.Timezone != "America/Noronha" {
				t.Errorf("Esperado Timezone 'America/Noronha', recebido '%s'", item.Timezone)
			}
		}
	}
	if !found {
		t.Fatalf("A filial criada não foi encontrada na busca geral")
	}
	if !foundSemReg {
		t.Fatalf("A filial criada sem regional não foi encontrada na busca geral")
	}

	// 5.2 Busca Filtrada (com tenant id válido)
	listFiltered, err := repo.GetAll(ctx, &tenantID)
	if err != nil {
		t.Fatalf("Falha ao listar lojas filtradas por tenant: %v", err)
	}
	if len(listFiltered) < 2 {
		t.Errorf("Esperado pelo menos 2 filiais no filtro do tenant, recebido %d", len(listFiltered))
	}

	// 5.3 Busca Filtrada (com outro tenant id inexistente)
	otherTenantID := uuid.New()
	listEmpty, err := repo.GetAll(ctx, &otherTenantID)
	if err != nil {
		t.Fatalf("Falha ao listar lojas filtradas por tenant aleatório: %v", err)
	}
	for _, item := range listEmpty {
		if item.ID == storeID || item.ID == storeSemRegID {
			t.Errorf("Erro: encontrou a loja mesmo filtrando por outro tenant_id")
		}
	}

	// 6. Teste de Edição (Update)
	store.StoreName = "Filial Centro Loja 01 Alterada"
	store.HasVideo = true
	store.District = "Centro Histórico"
	store.AudioGridID = nil
	store.VideoGridID = nil
	err = repo.Update(ctx, store)
	if err != nil {
		t.Fatalf("Falha ao atualizar dados da filial: %v", err)
	}

	listUpdated, _ := repo.GetAll(ctx, &tenantID)
	var updatedFound bool
	for _, item := range listUpdated {
		if item.ID == storeID {
			updatedFound = true
			if item.StoreName != "Filial Centro Loja 01 Alterada" {
				t.Errorf("Nome da filial não foi atualizado")
			}
			if item.District != "Centro Histórico" {
				t.Errorf("Bairro da filial não foi atualizado")
			}
			if !item.HasVideo {
				t.Errorf("Flag HasVideo não foi atualizada")
			}
			if item.AudioGridID != nil {
				t.Errorf("Esperado AudioGridID nil após atualização")
			}
			if item.VideoGridID != nil {
				t.Errorf("Esperado VideoGridID nil após atualização")
			}
			break
		}
	}
	if !updatedFound {
		t.Fatalf("A filial alterada não foi encontrada")
	}

	// 7. Teste de Exclusão (Delete)
	err = repo.Delete(ctx, storeID)
	if err != nil {
		t.Fatalf("Erro ao excluir filial: %v", err)
	}

	listDeleted, _ := repo.GetAll(ctx, nil)
	for _, item := range listDeleted {
		if item.ID == storeID {
			t.Errorf("Erro: a filial continuou cadastrada após o comando de exclusão")
		}
	}
}
