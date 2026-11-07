package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

func TestHandleImportStores(t *testing.T) {
	// 1. Conecta ao Postgres
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste de importação.")
		return
	}

	ctx := context.Background()

	// 2. Cria Matriz e Regional para associar na importação
	tenantID := uuid.New()
	testCNPJ := "12345678000199"

	// Garante limpeza de dados anteriores
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", testCNPJ)

	logo := "/static/test.png"
	tenant := models.Tenant{
		ID:             tenantID,
		TradeName:      "Empresa Teste Import",
		LegalName:      "Empresa Teste Import S.A.",
		DocumentNumber: testCNPJ,
		ZipCode:        "80000-000",
		Street:         "Rua XV",
		City:           "Curitiba",
		State:          "PR",
		LogoURL:        &logo,
		IsActive:       true,
	}
	err := tenantRepo.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Falha ao criar tenant: %v", err)
	}
	defer tenantRepo.Delete(ctx, tenantID)

	regionalID := uuid.New()
	regional := models.Regional{
		ID:       regionalID,
		TenantID: tenantID,
		Name:     "Regional Sul Import",
		IsActive: true,
	}
	err = regionalRepo.Create(ctx, regional)
	if err != nil {
		t.Fatalf("Falha ao criar regional: %v", err)
	}

	audioGridID := uuid.New()
	_, err = database.Pool.Exec(ctx, 
		"INSERT INTO weekly_master_grids (id, tenant_id, media_type, name, description) VALUES ($1, $2, 'AUDIO', 'Grade Teste Import Audio', 'Desc')",
		audioGridID, tenantID,
	)
	if err != nil {
		t.Fatalf("Falha ao criar grade de teste de áudio: %v", err)
	}
	defer database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", audioGridID)

	videoGridID := uuid.New()
	_, err = database.Pool.Exec(ctx, 
		"INSERT INTO weekly_master_grids (id, tenant_id, media_type, name, description) VALUES ($1, $2, 'VIDEO', 'Grade Teste Import Video', 'Desc')",
		videoGridID, tenantID,
	)
	if err != nil {
		t.Fatalf("Falha ao criar grade de teste de vídeo: %v", err)
	}
	defer database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", videoGridID)

	// 3. Gera planilha Excel em memória
	xlsx := excelize.NewFile()
	sheetName := "Filiais"
	index, err := xlsx.NewSheet(sheetName)
	if err != nil {
		t.Fatalf("Falha ao criar aba filiais: %v", err)
	}
	xlsx.SetActiveSheet(index)
	xlsx.DeleteSheet("Sheet1")

	// Escreve cabeçalhos
	headers := []string{
		"Nome da Filial", "Número Interno", "CNPJ da Matriz", "Nome da Regional",
		"CEP", "Rua", "Número", "Bairro", "Cidade", "Estado", "Telefone",
		"Nome do Gerente", "Habilitar Áudio (SIM/NÃO)", "Habilitar Vídeo (SIM/NÃO)", "Fuso Horário",
		"Grade de Áudio (Nome)", "Grade de Vídeo (Nome)",
	}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		xlsx.SetCellValue(sheetName, cell, h)
	}

	// Escreve linha de dados
	rowData := []interface{}{
		"Filial Importada Excel", "LOJA-EXCEL", testCNPJ, "Regional Sul Import",
		"80010-000", "Avenida das Araucárias", "123", "Bairro Novo", "Curitiba", "PR",
		"41999998888", "Carlos Silva", "SIM", "NÃO", "America/Manaus",
		"Grade Teste Import Audio", "Grade Teste Import Video",
	}
	for colIdx, val := range rowData {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		xlsx.SetCellValue(sheetName, cell, val)
	}

	var buf bytes.Buffer
	if err := xlsx.Write(&buf); err != nil {
		t.Fatalf("Falha ao gerar excel em memoria: %v", err)
	}

	// 4. Cria Requisição Multipart POST
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "filiais.xlsx")
	if err != nil {
		t.Fatalf("Falha ao criar form file: %v", err)
	}
	part.Write(buf.Bytes())
	writer.Close()

	req := httptest.NewRequest("POST", "/dashboard/stores/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Configura sessão simulada do usuário (Admin Mestre)
	session := models.SessionData{
		UserID:            uuid.New(),
		UserName:          "TestUser",
		PermissionLevelID: 1, // 1 represents Administrator
		TenantID:          nil, // Admin mestre pode importar para qualquer CNPJ
	}
	sessJSON, _ := json.Marshal(session)
	cookieVal := url.QueryEscape(string(sessJSON))
	req.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	// 5. Executa o Handler
	rec := httptest.NewRecorder()
	HandleImportStores(rec, req)

	// 6. Verifica redirecionamento e gravação
	res := rec.Result()
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("Esperado status code %d, recebido %d", http.StatusSeeOther, res.StatusCode)
	}

	loc := res.Header.Get("Location")
	if !bytes.Contains([]byte(loc), []byte("success")) {
		t.Errorf("Esperado redirecionamento com indicador de sucesso, recebido: %s", loc)
	}

	// 7. Valida se a loja foi inserida no banco
	stores, err := storeRepo.GetAll(ctx, &tenantID)
	if err != nil {
		t.Fatalf("Erro ao listar lojas do banco: %v", err)
	}

	var found bool
	for _, s := range stores {
		if s.StoreName == "Filial Importada Excel" {
			found = true
			if s.StoreNumber != "LOJA-EXCEL" {
				t.Errorf("Esperado numero LOJA-EXCEL, recebido %s", s.StoreNumber)
			}
			if s.District != "Bairro Novo" {
				t.Errorf("Esperado bairro Bairro Novo, recebido %s", s.District)
			}
			if s.Street != "Avenida das Araucárias" {
				t.Errorf("Esperado rua Avenida das Araucárias, recebido %s", s.Street)
			}
			if s.Timezone != "America/Manaus" {
				t.Errorf("Esperado fuso horario America/Manaus, recebido %s", s.Timezone)
			}
			if s.RegionalID == nil || *s.RegionalID != regionalID {
				t.Errorf("Esperado regional ID %v, recebido %v", regionalID, s.RegionalID)
			}
			if s.AudioGridID == nil || *s.AudioGridID != audioGridID {
				t.Errorf("Esperado AudioGridID %v, recebido %v", audioGridID, s.AudioGridID)
			}
			if s.VideoGridID == nil || *s.VideoGridID != videoGridID {
				t.Errorf("Esperado VideoGridID %v, recebido %v", videoGridID, s.VideoGridID)
			}
			break
		}
	}

	if !found {
		t.Errorf("A loja importada do Excel não foi encontrada no banco de dados!")
	}
}

func TestStoreDefaultGridInheritance(t *testing.T) {
	// Connect to Postgres
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste.")
		return
	}

	ctx := context.Background()

	// 1. Create a Test Tenant with Default Grid IDs
	tenantID := uuid.New()
	testCNPJ := "99999999000188"

	// Cleanup
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", testCNPJ)

	audioGridID := uuid.New()
	videoGridID := uuid.New()

	// Create test grids
	_, _ = database.Pool.Exec(ctx, "INSERT INTO weekly_master_grids (id, tenant_id, media_type, name) VALUES ($1, '00000000-0000-0000-0000-000000000000', 'AUDIO', 'Test Default Audio')", audioGridID)
	_, _ = database.Pool.Exec(ctx, "INSERT INTO weekly_master_grids (id, tenant_id, media_type, name) VALUES ($1, '00000000-0000-0000-0000-000000000000', 'VIDEO', 'Test Default Video')", videoGridID)
	defer func() {
		database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", audioGridID)
		database.Pool.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", videoGridID)
	}()

	tenant := models.Tenant{
		ID:                 tenantID,
		TradeName:          "Matriz Com Defaults",
		LegalName:          "Matriz Com Defaults S.A.",
		DocumentNumber:     testCNPJ,
		ZipCode:            "80000-000",
		Street:             "Rua XV",
		City:               "Curitiba",
		State:              "PR",
		IsActive:           true,
		DefaultAudioGridID: &audioGridID,
		DefaultVideoGridID: &videoGridID,
	}

	err := tenantRepo.Create(ctx, tenant)
	if err != nil {
		t.Fatalf("Erro ao criar tenant: %v", err)
	}
	defer tenantRepo.Delete(ctx, tenantID)

	// 2. Perform Handler Request to Create Store WITHOUT explicit grids
	form := url.Values{}
	form.Set("tenant_id", tenantID.String())
	form.Set("store_name", "Filial Herdeira de Grid")
	form.Set("city", "Curitiba")
	form.Set("state", "PR")
	form.Set("audio_grid_id", "") // Leave empty to inherit Matriz's default
	form.Set("video_grid_id", "") // Leave empty to inherit Matriz's default

	req := httptest.NewRequest("POST", "/dashboard/stores/create", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set session
	session := models.SessionData{
		UserID:            uuid.New(),
		UserName:          "TestUser",
		PermissionLevelID: 1,
		TenantID:          nil,
	}
	sessJSON, _ := json.Marshal(session)
	req.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: url.QueryEscape(string(sessJSON)),
	})

	rec := httptest.NewRecorder()
	HandleCreateStore(rec, req)

	// 3. Verify store was created and inherited default grids
	stores, err := storeRepo.GetAll(ctx, &tenantID)
	if err != nil {
		t.Fatalf("Erro ao buscar lojas: %v", err)
	}

	var found bool
	for _, s := range stores {
		if s.StoreName == "Filial Herdeira de Grid" {
			found = true
			if s.AudioGridID == nil || *s.AudioGridID != audioGridID {
				t.Errorf("Esperado AudioGridID herdado %v, recebido %v", audioGridID, s.AudioGridID)
			}
			if s.VideoGridID == nil || *s.VideoGridID != videoGridID {
				t.Errorf("Esperado VideoGridID herdado %v, recebido %v", videoGridID, s.VideoGridID)
			}
			break
		}
	}

	if !found {
		t.Errorf("Loja herdada de grid não foi encontrada no banco!")
	}
}
