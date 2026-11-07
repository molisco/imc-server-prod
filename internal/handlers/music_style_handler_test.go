package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

func TestMusicStyleHandlers(t *testing.T) {
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
	database.Pool.Exec(ctx, "DELETE FROM music_styles WHERE name = $1 OR name = $2", "Music Style Handler Test", "Music Style Handler Test Updated")

	session := models.SessionData{
		UserID:            uuid.New(),
		UserName:          "TestUser",
		PermissionLevelID: 1,
		TenantID:          nil,
	}
	sessJSON, _ := json.Marshal(session)
	cookieVal := url.QueryEscape(string(sessJSON))

	// 2. Teste HandleCreateMusicStyle
	form := url.Values{}
	form.Set("name", "Music Style Handler Test")
	form.Set("description", "A test music style created by handler test")
	form.Set("base_directory", "/test/dir/music")

	reqCreate := httptest.NewRequest("POST", "/dashboard/music-styles/create", bytes.NewBufferString(form.Encode()))
	reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreate.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recCreate := httptest.NewRecorder()
	HandleCreateMusicStyle(recCreate, reqCreate)

	if recCreate.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Create, got %d", http.StatusSeeOther, recCreate.Result().StatusCode)
	}

	// Verifica criação no banco de dados
	styles, err := musicStyleRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch styles: %v", err)
	}

	var styleID uuid.UUID
	var found bool
	for _, s := range styles {
		if s.Name == "Music Style Handler Test" {
			found = true
			styleID = s.ID
			if s.Description != "A test music style created by handler test" {
				t.Errorf("Expected description 'A test music style created by handler test', got '%s'", s.Description)
			}
			break
		}
	}
	if !found {
		t.Fatalf("Created music style not found in database!")
	}

	// 3. Teste HandleUpdateMusicStyle
	formUpdate := url.Values{}
	formUpdate.Set("id", styleID.String())
	formUpdate.Set("name", "Music Style Handler Test Updated")
	formUpdate.Set("description", "An updated music description")
	formUpdate.Set("base_directory", "/test/dir/music/updated")

	reqUpdate := httptest.NewRequest("POST", "/dashboard/music-styles/update", bytes.NewBufferString(formUpdate.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUpdate.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recUpdate := httptest.NewRecorder()
	HandleUpdateMusicStyle(recUpdate, reqUpdate)

	if recUpdate.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Update, got %d", http.StatusSeeOther, recUpdate.Result().StatusCode)
	}

	// Verifica update no banco de dados
	updatedStyle, err := musicStyleRepo.GetByID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to get style by ID: %v", err)
	}
	if updatedStyle.Name != "Music Style Handler Test Updated" {
		t.Errorf("Expected name 'Music Style Handler Test Updated', got '%s'", updatedStyle.Name)
	}

	// 4. Teste HandleMusicStylesPage rendering
	reqPage := httptest.NewRequest("GET", "/dashboard/music-styles", nil)
	reqPage.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recPage := httptest.NewRecorder()
	HandleMusicStylesPage(recPage, reqPage)

	if recPage.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for page render, got %d", http.StatusOK, recPage.Result().StatusCode)
	}

	// 5. Teste HandleDeleteMusicStyle
	reqDelete := httptest.NewRequest("GET", "/dashboard/music-styles/delete?id="+styleID.String(), nil)
	reqDelete.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recDelete := httptest.NewRecorder()
	HandleDeleteMusicStyle(recDelete, reqDelete)

	if recDelete.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status code %d for Delete, got %d", http.StatusSeeOther, recDelete.Result().StatusCode)
	}

	// Verifica deleção no banco de dados
	_, err = musicStyleRepo.GetByID(ctx, styleID)
	if err == nil {
		t.Errorf("Expected style to be deleted, but it was found in database")
	}
}

func TestSyncMusicStyleDirectory(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste.")
		return
	}

	ctx := context.Background()

	// 1. Create a temporary directory
	tmpDir := t.TempDir()

	// 2. Create a style
	styleID := uuid.New()
	style := models.MusicStyle{
		ID:            styleID,
		Name:          "Scanner Style Test",
		Description:   "Scanner test description",
		BaseDirectory: tmpDir,
	}

	err := musicStyleRepo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create style: %v", err)
	}
	defer musicStyleRepo.Delete(ctx, styleID)

	// Write mock mp3 file (invalid MP3 but ends in .mp3, so duration defaults to 180)
	mp3Path := filepath.Join(tmpDir, "track_one.mp3")
	err = os.WriteFile(mp3Path, []byte("mock mp3 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock mp3 file: %v", err)
	}

	// 3. Run syncMusicStyleDirectory synchronously
	syncMusicStyleDirectory(ctx, styleID, tmpDir, false)

	// 4. Verify track created in database
	tracks, err := musicTrackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to get tracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("Expected 1 track in DB, got %d", len(tracks))
	}
	if tracks[0].FileName != "track_one.mp3" {
		t.Errorf("Expected filename 'track_one.mp3', got '%s'", tracks[0].FileName)
	}
	if tracks[0].DisplayName != "track_one" {
		t.Errorf("Expected display name 'track_one', got '%s'", tracks[0].DisplayName)
	}
	if tracks[0].DurationSeconds != 180 {
		t.Errorf("Expected default duration of 180 seconds, got %d", tracks[0].DurationSeconds)
	}
	if tracks[0].LoudnessScore != "A" {
		t.Errorf("Expected default loudness score 'A', got '%s'", tracks[0].LoudnessScore)
	}
	if tracks[0].LRA != 5.0 {
		t.Errorf("Expected default LRA 5.0, got %f", tracks[0].LRA)
	}
	if tracks[0].LUFS != -14.0 {
		t.Errorf("Expected default LUFS -14.0, got %f", tracks[0].LUFS)
	}

	// 5. Test deletion (remove the file and sync again)
	err = os.Remove(mp3Path)
	if err != nil {
		t.Fatalf("Failed to delete mock mp3: %v", err)
	}

	syncMusicStyleDirectory(ctx, styleID, tmpDir, false)

	tracksAfter, err := musicTrackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to get tracks after delete: %v", err)
	}
	if len(tracksAfter) != 0 {
		t.Errorf("Expected 0 tracks after syncing deleted file, got %d", len(tracksAfter))
	}
}

func TestSyncHandlers(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste.")
		return
	}

	ctx := context.Background()

	// Create style
	styleID := uuid.New()
	style := models.MusicStyle{
		ID:            styleID,
		Name:          "Sync Route Handler Test",
		Description:   "Route test description",
		BaseDirectory: t.TempDir(),
	}

	err := musicStyleRepo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create style: %v", err)
	}
	defer musicStyleRepo.Delete(ctx, styleID)

	session := models.SessionData{
		UserID:            uuid.New(),
		UserName:          "TestUser",
		PermissionLevelID: 1,
		TenantID:          nil,
	}
	sessJSON, _ := json.Marshal(session)
	cookieVal := url.QueryEscape(string(sessJSON))

	// 1. Post to Sync
	reqSync := httptest.NewRequest("POST", "/dashboard/music-styles/sync?id="+styleID.String(), nil)
	reqSync.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recSync := httptest.NewRecorder()
	HandleSyncMusicStyle(recSync, reqSync)

	if recSync.Result().StatusCode != http.StatusAccepted {
		t.Errorf("Expected status code %d for Sync endpoint, got %d", http.StatusAccepted, recSync.Result().StatusCode)
	}

	// 2. Get Sync Status
	reqStatus := httptest.NewRequest("GET", "/dashboard/music-styles/sync-status?id="+styleID.String(), nil)
	reqStatus.AddCookie(&http.Cookie{
		Name:  "imc_session",
		Value: cookieVal,
	})

	recStatus := httptest.NewRecorder()
	HandleMusicStyleSyncStatus(recStatus, reqStatus)

	if recStatus.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for Status endpoint, got %d", http.StatusOK, recStatus.Result().StatusCode)
	}

	var progress ScanProgress
	if err := json.Unmarshal(recStatus.Body.Bytes(), &progress); err != nil {
		t.Fatalf("Failed to unmarshal sync status JSON: %v", err)
	}

	// Check status is set
	if progress.Status == "" {
		t.Errorf("Expected status to be populated, got empty string")
	}
}

func TestSyncMusicStyleDirectory_MissingDirectory(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("PostgreSQL não disponível. Pulando teste.")
		return
	}

	ctx := context.Background()

	// 1. Create a style with a temporary directory
	tmpDir := t.TempDir()
	styleID := uuid.New()
	style := models.MusicStyle{
		ID:            styleID,
		Name:          "Missing Dir Test Style",
		Description:   "Style to test missing directory scan",
		BaseDirectory: tmpDir,
	}

	err := musicStyleRepo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create style: %v", err)
	}
	defer musicStyleRepo.Delete(ctx, styleID)

	// Write mock mp3 file to register it
	mp3Path := filepath.Join(tmpDir, "track_to_be_deleted.mp3")
	err = os.WriteFile(mp3Path, []byte("mock content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock mp3: %v", err)
	}

	// Scan directory first to populate DB
	syncMusicStyleDirectory(ctx, styleID, tmpDir, false)

	// Verify it has been added
	tracksBefore, err := musicTrackRepo.GetByStyleID(ctx, styleID)
	if err != nil || len(tracksBefore) != 1 {
		t.Fatalf("Expected 1 track in DB before directory removal, got %d (err: %v)", len(tracksBefore), err)
	}

	// 2. Remove the directory entirely
	err = os.RemoveAll(tmpDir)
	if err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	// Scan style again with the now missing directory
	syncMusicStyleDirectory(ctx, styleID, tmpDir, false)

	// Verify all tracks for this style are deleted from database
	tracksAfter, err := musicTrackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to retrieve tracks: %v", err)
	}
	if len(tracksAfter) != 0 {
		t.Errorf("Expected 0 tracks in DB after directory removal, got %d", len(tracksAfter))
	}

	// Verify progress is set to completed
	progress := getOrCreateProgress(styleID)
	if progress.Status != "completed" {
		t.Errorf("Expected progress status to be 'completed' after missing directory sync, got '%s'", progress.Status)
	}
}


