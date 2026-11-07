package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"
	"testing"

	"github.com/google/uuid"
)

func TestTenantCRUD(t *testing.T) {
	// Inicializa conexão para teste de integração
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("Banco de dados não disponível. Pulando teste de integração.")
		return
	}

	repo := NewTenantRepository()
	ctx := context.Background()

	// Limpeza preventiva de execuções de testes anteriores interrompidas
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", "12345678000195")

	id := uuid.New()
	logo := "/static/uploads/test.png"
	tenant := models.Tenant{
		ID:             id,
		TradeName:      "Tenant Teste Unitario",
		LegalName:      "Tenant Teste Unitario S/A",
		DocumentNumber: "12345678000195", // CNPJ Válido
		ZipCode:        "80000-000",
		Street:         "Rua de Teste",
		Number:         "999-B", // Novo campo número do endereço
		District:       "Bairro de Teste",
		City:           "Curitiba",
		State:          "PR",
		Country:        "Brasil",
		LogoURL:        &logo,
		PrimaryColor:   "#ffffff",
		IsActive:       true,
		OperationalContacts: []models.ContactDetails{
			{Name: "", Email: "teste_unidade@teste.com", Phone: "9999-9999"}, // Nome opcional
		},
	}

	// 1. Testar Gravação (Create)
	err := repo.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Falha ao gravar matriz: %v", err)
	}

	// 2. Testar Leitura (Read)
	list, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Falha ao buscar matrizes: %v", err)
	}

	var found bool
	for _, item := range list {
		if item.ID == id {
			found = true
			if item.Number != "999-B" {
				t.Errorf("Esperado número do endereço '999-B', recebido '%s'", item.Number)
			}
			if len(item.OperationalContacts) != 1 || item.OperationalContacts[0].Email != "teste_unidade@teste.com" {
				t.Errorf("Contatos operacionais inválidos ou não persistidos: %v", item.OperationalContacts)
			}
			break
		}
	}
	if !found {
		t.Fatalf("A matriz cadastrada não foi encontrada com GetAll")
	}

	// 3. Testar Atualização (Update)
	tenant.TradeName = "Tenant Teste Unitario Alterado"
	tenant.Number = "100"
	err = repo.Update(ctx, tenant)
	if err != nil {
		t.Fatalf("Falha ao atualizar matriz: %v", err)
	}

	// 4. Testar Exclusão (Delete)
	err = repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Falha ao excluir matriz: %v", err)
	}

	// Confirma exclusão
	list2, _ := repo.GetAll(ctx)
	for _, item := range list2 {
		if item.ID == id {
			t.Errorf("Falha: a matriz continuou gravada no banco após exclusão")
		}
	}
}
