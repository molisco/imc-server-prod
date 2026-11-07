package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

func TestVideoStyleHandlers(t *testing.T) {
	// Altera o diretório de trabalho para a raiz do projeto para carregar templates corretamente
	originalWD, wdErr := os.Getwd()
	if wdErr == nil {
		_ = os.Chdir("../../")
		defer os.Chdir(originalWD)
	}

	// 1. Conecta ao banco de dados
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste de handlers.")
		return
	}

	ctx := context.Background()

	// Limpa estilos que podem conflitar
	database.Pool.Exec(ctx, "DELETE FROM video_styles WHERE name = $1 OR name = $2", "Video Style Handler Test", "Video Style Handler Test Updated")

	session := models.SessionData{
		UserID:            uuid.New(),
		UserName:          "TestUser",
		PermissionLevelID: 1,
		TenantID:          nil,
	}
	sessJSON, _ := json.Marshal(session)
	cookieVal := url.QueryEscape(string(sessJSON))

	// 2. Teste HandleCreateVideoStyle
	form := url.Values{}
	form.Set("name", "Video Style Handler Test")
	form.Set("description", "A test video style created by handler test")
	form.Set("base_directory", "/test/dir/video")

	reqCreate := httptest.NewRequest("POST", "/dashboard/video-styles/create", bytes.NewBufferString(form.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreate.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recCreate := httptest.NewRecorder()
	HandleCreateVideoStyle(recCreate, reqCreate)

	if recCreate.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Create, got %d", http.StatusSeeOther, recCreate.Result().StatusCode)
	}

	// Verifica criação no banco de dados
	styles, err := videoStyleRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch styles: %v", err)
	}

	var styleID uuid.UUID
	var found bool
	for _, s := range styles {
		if s.Name == "Video Style Handler Test" {
			found = true
			styleID = s.ID
			if s.Description != "A test video style created by handler test" {
				t.Errorf("Expected description 'A test video style created by handler test', got '%s'", s.Description)
			}
			break
		}
	}
	if !found {
		t.Fatalf("Created video style not found in database!")
	}

	// 3. Teste HandleUpdateVideoStyle
	formUpdate := url.Values{}
	formUpdate.Set("id", styleID.String())
	formUpdate.Set("name", "Video Style Handler Test Updated")
	formUpdate.Set("description", "An updated video description")
	formUpdate.Set("base_directory", "/test/dir/video/updated")

	reqUpdate := httptest.NewRequest("POST", "/dashboard/video-styles/update", bytes.NewBufferString(formUpdate.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUpdate.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recUpdate := httptest.NewRecorder()
	HandleUpdateVideoStyle(recUpdate, reqUpdate)

	if recUpdate.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Update, got %d", http.StatusSeeOther, recUpdate.Result().StatusCode)
	}

	// Verifica update no banco de dados
	updatedStyle, err := videoStyleRepo.GetByID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to get style by ID: %v", err)
	}
	if updatedStyle.Name != "Video Style Handler Test Updated" {
		t.Errorf("Expected name 'Video Style Handler Test Updated', got '%s'", updatedStyle.Name)
	}

	// 4. Teste HandleVideoStylesPage rendering
	reqPage := httptest.NewRequest("GET", "/dashboard/video-styles", nil)
	reqPage.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recPage := httptest.NewRecorder()
	HandleVideoStylesPage(recPage, reqPage)

	if recPage.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for page render, got %d", http.StatusOK, recPage.Result().StatusCode)
	}

	// 5. Teste HandleDeleteVideoStyle
	reqDelete := httptest.NewRequest("GET", "/dashboard/video-styles/delete?id="+styleID.String(), nil)
	reqDelete.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recDelete := httptest.NewRecorder()
	HandleDeleteVideoStyle(recDelete, reqDelete)

	if recDelete.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Delete, got %d", http.StatusSeeOther, recDelete.Result().StatusCode)
	}

	// Verifica deleção no banco de dados
	_, err = videoStyleRepo.GetByID(ctx, styleID)
	if err == nil {
		t.Errorf("Expected style to be deleted, but it was found in database")
	}
}
