package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"imc-server/internal/database"
	"imc-server/internal/models"
	"imc-server/internal/repository"
	"imc-server/internal/utils"
)

var storeRepo = repository.NewStoreRepository()

func getNextHarborPort() int {
	var maxPort sql.NullInt64
	err := database.Pool.QueryRow(context.Background(), "SELECT max(harbor_port) FROM stores").Scan(&maxPort)
	if err != nil || !maxPort.Valid || maxPort.Int64 < 7002 {
		return 7003
	}
	return int(maxPort.Int64) + 1
}

var hotmixStationURLs = map[string]string{
	"Best-of":                  "https://streaming.hotmixradio.com/hotmixpro-best-of-pre",
	"Cocktail & Dinner Groove": "https://streaming.hotmixradio.com/hotmixpro-cocktail-and-dinner-pre",
	"Cool & Calm":              "https://streaming.hotmixradio.com/hotmixpro-cool-and-calm-pre",
	"Cozy & Coffee":            "https://streaming.hotmixradio.com/hotmixpro-cozy-coffee-shop-pre",
	"Gentle Luxury":            "https://streaming.hotmixradio.com/hotmixpro-gentle-luxury-pre",
	"Just Lounge":              "https://streaming.hotmixradio.com/hotmixpro-just-lounge-pre",
	"On the Rocks":             "https://streaming.hotmixradio.com/hotmixpro-on-the-rocks-pre",
	"Shopping Vibes":           "https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre",
	"Spa Garden":               "https://streaming.hotmixradio.com/hotmixpro-spa-garden-pre",
	"Workout":                  "https://streaming.hotmixradio.com/hotmixpro-workout-pre",
}

// GridOption representa uma grade de programação para seleção
type GridOption struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	MediaType string    `json:"media_type"`
	Name      string    `json:"name"`
}

// StorePageData empacota os dados estruturados enviados ao template HTML de Filiais
type StorePageData struct {
	Title          string
	LoggedUser     string
	ErrorMessage   string
	SuccessMessage string
	Form           map[string]string // Preserva dados digitados
	Tenants        []models.Tenant
	Regionals      []models.Regional
	Grids          []GridOption
	Stores         []models.Store
	User           map[string]string
	Theme          map[string]string
}

// getGridsList retorna todas as grades de programação cadastradas (filtrando por TenantID se especificado)
func getGridsList(tenantID *uuid.UUID) []GridOption {
	queryGrids := "SELECT id, tenant_id, media_type, name FROM weekly_master_grids"
	var gridsArgs []interface{}
	if tenantID != nil {
		queryGrids += " WHERE tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000'"
		gridsArgs = append(gridsArgs, *tenantID)
	}
	queryGrids += " ORDER BY name ASC"

	var gridsList []GridOption
	gridRows, err := database.Pool.Query(context.Background(), queryGrids, gridsArgs...)
	if err == nil {
		defer gridRows.Close()
		for gridRows.Next() {
			var g GridOption
			if errScan := gridRows.Scan(&g.ID, &g.TenantID, &g.MediaType, &g.Name); errScan == nil {
				gridsList = append(gridsList, g)
			}
		}
	}
	if gridsList == nil {
		gridsList = []GridOption{}
	}
	return gridsList
}

// HandleStoresPage renderiza a tela de gerenciamento de Filiais (Lojas)
func HandleStoresPage(w http.ResponseWriter, r *http.Request) {
	// 1. Validação de Sessão
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 2. Busca lista de Clientes (Tenants) ativos para o dropdown (aplicando isolamento de tenant se necessário)
	tenantsList, err := tenantRepo.GetAll(context.Background())
	if err != nil {
		tenantsList = []models.Tenant{}
	}
	if session.TenantID != nil {
		var filtered []models.Tenant
		for _, t := range tenantsList {
			if t.ID == *session.TenantID {
				filtered = append(filtered, t)
				break
			}
		}
		tenantsList = filtered
	}

	// 3. Busca lista de Regionais (com isolamento de tenant)
	regionalsList, err := regionalRepo.GetAll(context.Background(), session.TenantID)
	if err != nil {
		regionalsList = []models.Regional{}
	}

	// 3.5 Busca lista de Grades de Programação
	gridsList := getGridsList(session.TenantID)

	// 4. Busca lista de Lojas (com isolamento de tenant)
	storesList, err := storeRepo.GetAll(context.Background(), session.TenantID)
	if err != nil {
		storesList = []models.Store{}
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "/static/img/logo-imc.png", // fallback or default background/theme helper
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	data := StorePageData{
		Title:          "Gerenciamento de Filiais (Lojas)",
		LoggedUser:     session.UserName,
		ErrorMessage:   errorMsg,
		SuccessMessage: successMsg,
		Tenants:        tenantsList,
		Regionals:      regionalsList,
		Grids:          gridsList,
		Stores:         storesList,
		Form:           make(map[string]string),
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme: theme,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/stores.html"))
	tmpl.Execute(w, data)
}

// HandleCreateStore processa o cadastro de uma nova loja/filial
func HandleCreateStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/stores", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	tenantIDStr := r.FormValue("tenant_id")
	regionalIDStr := r.FormValue("regional_id")
	audioGridIDStr := r.FormValue("audio_grid_id")
	videoGridIDStr := r.FormValue("video_grid_id")
	storeName := r.FormValue("store_name")
	storeNumber := r.FormValue("store_number")
	zipCode := r.FormValue("zip_code")
	street := r.FormValue("street")
	number := r.FormValue("number")
	district := r.FormValue("district")
	city := r.FormValue("city")
	state := r.FormValue("state")
	timezone := r.FormValue("timezone")
	phone := r.FormValue("phone")
	managerName := r.FormValue("manager_name")
	radioType := r.FormValue("radio_type")
	if radioType == "" {
		radioType = "server"
	}
	hasAudio := r.FormValue("has_audio") == "on"
	hasVideo := r.FormValue("has_video") == "on"
	externalStreamURL := r.FormValue("external_stream_url")
	externalStreamName := r.FormValue("external_stream_name")

	formData := map[string]string{
		"tenant_id":            tenantIDStr,
		"regional_id":          regionalIDStr,
		"audio_grid_id":        audioGridIDStr,
		"video_grid_id":        videoGridIDStr,
		"store_name":           storeName,
		"store_number":         storeNumber,
		"zip_code":             zipCode,
		"street":               street,
		"number":               number,
		"district":             district,
		"city":                 city,
		"state":                state,
		"timezone":             timezone,
		"phone":                phone,
		"manager_name":         managerName,
		"radio_type":           radioType,
		"external_stream_url":  externalStreamURL,
		"external_stream_name": externalStreamName,
	}
	if hasAudio {
		formData["has_audio"] = "on"
	}
	if hasVideo {
		formData["has_video"] = "on"
	}

	// Forçar tenant_id da sessão caso o usuário não seja Admin Mestre
	if session.TenantID != nil {
		tenantIDStr = session.TenantID.String()
	}

	tenantID, parseTenantErr := uuid.Parse(tenantIDStr)
	
	var regionalID *uuid.UUID
	var parseRegErr error
	if regionalIDStr != "" {
		parsed, err := uuid.Parse(regionalIDStr)
		if err != nil {
			parseRegErr = err
		} else {
			regionalID = &parsed
		}
	}

	var audioGridID *uuid.UUID
	if audioGridIDStr != "" {
		if parsed, err := uuid.Parse(audioGridIDStr); err == nil {
			audioGridID = &parsed
		}
	}

	var videoGridID *uuid.UUID
	if videoGridIDStr != "" {
		if parsed, err := uuid.Parse(videoGridIDStr); err == nil {
			videoGridID = &parsed
		}
	}

	// If grid IDs are not selected, load tenant's defaults
	if audioGridID == nil || videoGridID == nil {
		var dbDefaultAudioGrid *uuid.UUID
		var dbDefaultVideoGrid *uuid.UUID
		err := database.Pool.QueryRow(context.Background(), 
			"SELECT default_audio_grid_id, default_video_grid_id FROM tenants WHERE id = $1", 
			tenantID).Scan(&dbDefaultAudioGrid, &dbDefaultVideoGrid)
		if err == nil {
			if audioGridID == nil && dbDefaultAudioGrid != nil && *dbDefaultAudioGrid != uuid.Nil {
				audioGridID = dbDefaultAudioGrid
			}
			if videoGridID == nil && dbDefaultVideoGrid != nil && *dbDefaultVideoGrid != uuid.Nil {
				videoGridID = dbDefaultVideoGrid
			}
		}
	}

	nameClean := strings.TrimSpace(storeName)

	if parseTenantErr != nil || parseRegErr != nil || nameClean == "" || city == "" || state == "" {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		storesList, _ := storeRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "/static/img/logo-imc.png",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}

		data := StorePageData{
			Title:        "Gerenciamento de Filiais (Lojas)",
			LoggedUser:   session.UserName,
			ErrorMessage: "❌ ATENÇÃO: Os campos Matriz, Nome da Filial, Cidade e Estado são obrigatórios!",
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			Grids:        getGridsList(session.TenantID),
			Stores:       storesList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/stores.html"))
		tmpl.Execute(w, data)
		return
	}

	store := models.Store{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RegionalID:  regionalID,
		AudioGridID: audioGridID,
		VideoGridID: videoGridID,
		StoreName:   nameClean,
		StoreNumber: strings.TrimSpace(storeNumber),
		ZipCode:     strings.TrimSpace(zipCode),
		Street:      strings.TrimSpace(street),
		Number:      strings.TrimSpace(number),
		District:    strings.TrimSpace(district),
		City:        strings.TrimSpace(city),
		State:       strings.TrimSpace(state),
		Timezone:    strings.TrimSpace(timezone),
		Phone:       strings.TrimSpace(phone),
		ManagerName: strings.TrimSpace(managerName),
		HasAudio:    hasAudio,
		HasVideo:    hasVideo,
		RadioType:   radioType,
		IsActive:    true,
	}
	if externalStreamURL != "" {
		store.ExternalStreamURL = &externalStreamURL
	}
	if externalStreamName != "" {
		store.ExternalStreamName = &externalStreamName
		if mappedURL, ok := hotmixStationURLs[externalStreamName]; ok {
			store.ExternalStreamURL = &mappedURL
		} else {
			var dbURL string
			_ = database.Pool.QueryRow(context.Background(), "SELECT stream_url FROM streaming_stations WHERE name = $1 LIMIT 1", externalStreamName).Scan(&dbURL)
			if dbURL != "" {
				store.ExternalStreamURL = &dbURL
			}
		}
	}

	if store.RadioType == "server" {
		hPort := getNextHarborPort()
		store.HarborPort = &hPort
	}

	err = storeRepo.Create(context.Background(), store)
	if err == nil && store.RadioType == "server" {
		go func(s models.Store) {
			_ = utils.StartStoreDockerContainer(s)
		}(store)
	}
	if err != nil {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		storesList, _ := storeRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "/static/img/logo-imc.png",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}

		data := StorePageData{
			Title:        "Gerenciamento de Filiais (Lojas)",
			LoggedUser:   session.UserName,
			ErrorMessage: fmt.Sprintf("❌ Erro ao salvar a Loja no PostgreSQL: %v", err),
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			Grids:        getGridsList(session.TenantID),
			Stores:       storesList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/stores.html"))
		tmpl.Execute(w, data)
		return
	}

	http.Redirect(w, r, "/stores", http.StatusSeeOther)
}

// HandleUpdateStore atualiza uma filial existente
func HandleUpdateStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/stores", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	idStr := r.FormValue("id")
	storeID, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/stores", http.StatusSeeOther)
		return
	}

	tenantIDStr := r.FormValue("tenant_id")
	regionalIDStr := r.FormValue("regional_id")
	audioGridIDStr := r.FormValue("audio_grid_id")
	videoGridIDStr := r.FormValue("video_grid_id")
	storeName := r.FormValue("store_name")
	storeNumber := r.FormValue("store_number")
	zipCode := r.FormValue("zip_code")
	street := r.FormValue("street")
	number := r.FormValue("number")
	district := r.FormValue("district")
	city := r.FormValue("city")
	state := r.FormValue("state")
	timezone := r.FormValue("timezone")
	phone := r.FormValue("phone")
	managerName := r.FormValue("manager_name")
	radioType := r.FormValue("radio_type")
	if radioType == "" {
		radioType = "server"
	}
	hasAudio := r.FormValue("has_audio") == "on"
	hasVideo := r.FormValue("has_video") == "on"
	externalStreamURL := r.FormValue("external_stream_url")
	externalStreamName := r.FormValue("external_stream_name")

	formData := map[string]string{
		"id":                   idStr,
		"tenant_id":            tenantIDStr,
		"regional_id":          regionalIDStr,
		"audio_grid_id":        audioGridIDStr,
		"video_grid_id":        videoGridIDStr,
		"store_name":           storeName,
		"store_number":         storeNumber,
		"zip_code":             zipCode,
		"street":               street,
		"number":               number,
		"district":             district,
		"city":                 city,
		"state":                state,
		"timezone":             timezone,
		"phone":                phone,
		"manager_name":         managerName,
		"radio_type":           radioType,
		"external_stream_url":  externalStreamURL,
		"external_stream_name": externalStreamName,
	}
	if hasAudio {
		formData["has_audio"] = "on"
	}
	if hasVideo {
		formData["has_video"] = "on"
	}

	// Forçar tenant_id da sessão caso o usuário não seja Admin Mestre
	if session.TenantID != nil {
		tenantIDStr = session.TenantID.String()
	}

	tenantID, parseTenantErr := uuid.Parse(tenantIDStr)

	var regionalID *uuid.UUID
	var parseRegErr error
	if regionalIDStr != "" {
		parsed, err := uuid.Parse(regionalIDStr)
		if err != nil {
			parseRegErr = err
		} else {
			regionalID = &parsed
		}
	}

	var audioGridID *uuid.UUID
	if audioGridIDStr != "" {
		if parsed, err := uuid.Parse(audioGridIDStr); err == nil {
			audioGridID = &parsed
		}
	}
	var videoGridID *uuid.UUID
	if videoGridIDStr != "" {
		if parsed, err := uuid.Parse(videoGridIDStr); err == nil {
			videoGridID = &parsed
		}
	}

	nameClean := strings.TrimSpace(storeName)

	if parseTenantErr != nil || parseRegErr != nil || nameClean == "" || city == "" || state == "" {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		storesList, _ := storeRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "/static/img/logo-imc.png",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}

		data := StorePageData{
			Title:        "Gerenciamento de Filiais (Lojas)",
			LoggedUser:   session.UserName,
			ErrorMessage: "❌ ATENÇÃO: Os campos Matriz, Nome da Filial, Cidade e Estado são obrigatórios!",
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			Grids:        getGridsList(session.TenantID),
			Stores:       storesList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/stores.html"))
		tmpl.Execute(w, data)
		return
	}

	store := models.Store{
		ID:          storeID,
		TenantID:    tenantID,
		RegionalID:  regionalID,
		AudioGridID: audioGridID,
		VideoGridID: videoGridID,
		StoreName:   nameClean,
		StoreNumber: strings.TrimSpace(storeNumber),
		ZipCode:     strings.TrimSpace(zipCode),
		Street:      strings.TrimSpace(street),
		Number:      strings.TrimSpace(number),
		District:    strings.TrimSpace(district),
		City:        strings.TrimSpace(city),
		State:       strings.TrimSpace(state),
		Timezone:    strings.TrimSpace(timezone),
		Phone:       strings.TrimSpace(phone),
		ManagerName: strings.TrimSpace(managerName),
		HasAudio:    hasAudio,
		HasVideo:    hasVideo,
		RadioType:   radioType,
		IsActive:    true,
	}
	if externalStreamURL != "" {
		store.ExternalStreamURL = &externalStreamURL
	}
	if externalStreamName != "" {
		store.ExternalStreamName = &externalStreamName
		if mappedURL, ok := hotmixStationURLs[externalStreamName]; ok {
			store.ExternalStreamURL = &mappedURL
		} else {
			var dbURL string
			_ = database.Pool.QueryRow(context.Background(), "SELECT stream_url FROM streaming_stations WHERE name = $1 LIMIT 1", externalStreamName).Scan(&dbURL)
			if dbURL != "" {
				store.ExternalStreamURL = &dbURL
			}
		}
	}

	var existingPort sql.NullInt64
	var existingToken string
	var existingURL sql.NullString
	_ = database.Pool.QueryRow(context.Background(), "SELECT harbor_port, token, external_stream_url FROM stores WHERE id = $1", storeID).Scan(&existingPort, &existingToken, &existingURL)

	if store.ExternalStreamURL == nil && existingURL.Valid && existingURL.String != "" {
		val := existingURL.String
		store.ExternalStreamURL = &val
	}

	if radioType == "server" {
		if existingPort.Valid {
			hPort := int(existingPort.Int64)
			store.HarborPort = &hPort
		} else {
			hPort := getNextHarborPort()
			store.HarborPort = &hPort
		}
	}
	store.Token = existingToken

	err = storeRepo.Update(context.Background(), store)
	if err == nil {
		if store.RadioType == "server" {
			go func(s models.Store) {
				_ = utils.StartStoreDockerContainer(s)
			}(store)
		} else {
			go func(id uuid.UUID) {
				_ = utils.StopStoreDockerContainer(id)
			}(store.ID)
		}
	}
	if err != nil {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		storesList, _ := storeRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "/static/img/logo-imc.png",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}

		data := StorePageData{
			Title:        "Gerenciamento de Filiais (Lojas)",
			LoggedUser:   session.UserName,
			ErrorMessage: fmt.Sprintf("❌ Erro ao atualizar no banco: %v", err),
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			Grids:        getGridsList(session.TenantID),
			Stores:       storesList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/stores.html"))
		tmpl.Execute(w, data)
		return
	}

	http.Redirect(w, r, "/stores", http.StatusSeeOther)
}

// HandleDeleteStore remove uma filial permanentemente
func HandleDeleteStore(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.URL.Query().Get("id")
	storeID, err := uuid.Parse(idStr)
	if err == nil {
		// Stop and clean up Docker container if any
		_ = utils.StopStoreDockerContainer(storeID)
		// Padrão Ouro: O repositório remove do Postgres
		_ = storeRepo.Delete(context.Background(), storeID)
	}

	http.Redirect(w, r, "/stores", http.StatusSeeOther)
}

// HandleToggleStoreStatus ativa ou suspende uma filial com 1 clique, gerenciando containers Docker
func HandleToggleStoreStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		_ = r.ParseForm()
		idStr = r.FormValue("id")
	}

	storeID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid store id"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	store, err := storeRepo.GetByID(ctx, storeID)
	if err != nil || store == nil {
		http.Error(w, `{"error":"store not found"}`, http.StatusNotFound)
		return
	}

	newStatus := !store.IsActive
	_, err = database.Pool.Exec(ctx, "UPDATE stores SET is_active = $1, updated_at = NOW() WHERE id = $2", newStatus, storeID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to update status: %v"}`, err), http.StatusInternalServerError)
		return
	}
	store.IsActive = newStatus

	// Se a rádio for do tipo servidor:
	if store.RadioType == "server" || store.RadioType == "" {
		if !newStatus {
			// Suspensa: para o container Docker para economizar 100% de recursos
			_ = utils.StopStoreDockerContainer(store.ID)
		} else {
			// Ativada: religa o container Docker da rádio
			_ = utils.StartStoreDockerContainer(*store)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"id":         store.ID.String(),
		"is_active":  newStatus,
		"store_name": store.StoreName,
		"message":    fmt.Sprintf("Filial %s %s com sucesso!", store.StoreName, func() string {
			if newStatus { return "ativada" }
			return "suspensa"
		}()),
	})
}

type VoicePackSyncItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	BaseDirectory string `json:"base_directory"`
	DownloadURL   string `json:"download_url"`
}

type ChimeSyncItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	FileName        string  `json:"file_name"`
	DurationSeconds int     `json:"duration_seconds"`
	CueIn           float64 `json:"cue_in"`
	CueOut          float64 `json:"cue_out"`
	DownloadURL     string  `json:"download_url"`
	Amplify         float64 `json:"amplify"`
}

type PreFormattedWarningSyncItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	FileName        string  `json:"file_name"`
	DurationSeconds int     `json:"duration_seconds"`
	CueIn           float64 `json:"cue_in"`
	CueOut          float64 `json:"cue_out"`
	DownloadURL     string  `json:"download_url"`
	Amplify         float64 `json:"amplify"`
}

// SyncPayload is the unified response struct for store edge synchronizations
type ManualStyleItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "music_style" or "streaming"
	URL  string `json:"url,omitempty"`
}

type SyncPayload struct {
	AllowedManualStyles    []ManualStyleItem             `json:"allowed_manual_styles"`
	Store                  models.Store                  `json:"store"`
	WeeklyMasterGrid       *models.WeeklyMasterGrid      `json:"weekly_master_grid,omitempty"`
	BasicSchedules         []models.BasicSchedule        `json:"basic_schedules"`
	CommercialBlockConfigs []models.CommercialBlockConfig `json:"commercial_block_configs"`
	MediaStyles            []models.MusicStyle           `json:"media_styles"`
	MediaLibrary           []models.MusicTrack           `json:"media_library"`
	CommercialsAndEvents   []models.CommercialsAndEvents `json:"commercials_and_events"`
	VoicePacks             []VoicePackSyncItem           `json:"voice_packs"`
	Chimes                 []ChimeSyncItem               `json:"chimes"`
	PreFormattedWarnings   []PreFormattedWarningSyncItem `json:"pre_formatted_warnings"`
}

// SyncStoreData handles the synchronization request coming from the C# edge Player
func SyncStoreData(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure token from the HTTP Headers (sent by the C# application)
	token := r.Header.Get("X-Store-Token")
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized. Missing X-Store-Token header"}`))
		return
	}

	// 2. Create a database context with timeout (keep it fast and safe)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. Query the database to find the store using the unique index token
	var store models.Store
	query := `
		SELECT id, tenant_id, regional_id, store_name, store_number, 
		       city, state, has_audio, has_video, token, created_at,
		       audio_grid_id, video_grid_id, radio_type, external_stream_url, external_stream_name
		FROM stores 
		WHERE token = $1 AND is_active = true`

	err := database.Pool.QueryRow(ctx, query, token).Scan(
		&store.ID, &store.TenantID, &store.RegionalID, &store.StoreName, &store.StoreNumber,
		&store.City, &store.State, &store.HasAudio, &store.HasVideo, &store.Token, &store.CreatedAt,
		&store.AudioGridID, &store.VideoGridID, &store.RadioType, &store.ExternalStreamURL, &store.ExternalStreamName,
	)

	// If no store matches that token, return 404
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Store not found or inactive for the provided token"}`))
		return
	}

	// 4. Update the tracking sync timestamp in background (Fire and Forget) to log player activity
	go func(storeID interface{}) {
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer updateCancel()
		_, _ = database.Pool.Exec(updateCtx, "UPDATE stores SET last_audio_sync = $1, updated_at = $1 WHERE id = $2", time.Now(), storeID)
	}(store.ID)

	// Fallback to tenant default audio grid if store grid is null
	if store.AudioGridID == nil {
		var defaultGridID *uuid.UUID
		errDefault := database.Pool.QueryRow(ctx, "SELECT default_audio_grid_id FROM tenants WHERE id = $1", store.TenantID).Scan(&defaultGridID)
		if errDefault == nil && defaultGridID != nil && *defaultGridID != uuid.Nil {
			store.AudioGridID = defaultGridID
		}
	}

	var payload SyncPayload
	payload.Store = store
	payload.BasicSchedules = []models.BasicSchedule{}
	payload.CommercialBlockConfigs = []models.CommercialBlockConfig{}
	payload.MediaStyles = []models.MusicStyle{}
	payload.MediaLibrary = []models.MusicTrack{}
	payload.CommercialsAndEvents = []models.CommercialsAndEvents{}

	var grid models.WeeklyMasterGrid
	hasGrid := false

	// Load weekly master grid
	if store.AudioGridID != nil {
		gridQuery := `
			SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description,
			       sunday_schedule_id, sunday_block_id,
			       monday_schedule_id, monday_block_id,
			       tuesday_schedule_id, tuesday_block_id,
			       wednesday_schedule_id, wednesday_block_id,
			       thursday_schedule_id, thursday_block_id,
			       friday_schedule_id, friday_block_id,
			       saturday_schedule_id, saturday_block_id,
			       holiday_schedule_id, holiday_block_id,
			       allowed_manual_styles_json,
			       created_at
			FROM weekly_master_grids
			WHERE id = $1`
		errGrid := database.Pool.QueryRow(ctx, gridQuery, *store.AudioGridID).Scan(
			&grid.ID, &grid.TenantID, &grid.MediaType, &grid.Name, &grid.Description,
			&grid.SundayScheduleID, &grid.SundayBlockID,
			&grid.MondayScheduleID, &grid.MondayBlockID,
			&grid.TuesdayScheduleID, &grid.TuesdayBlockID,
			&grid.WednesdayScheduleID, &grid.WednesdayBlockID,
			&grid.ThursdayScheduleID, &grid.ThursdayBlockID,
			&grid.FridayScheduleID, &grid.FridayBlockID,
			&grid.SaturdayScheduleID, &grid.SaturdayBlockID,
			&grid.HolidayScheduleID, &grid.HolidayBlockID,
			&grid.AllowedManualStylesJSON,
			&grid.CreatedAt,
		)
		if errGrid == nil {
			payload.WeeklyMasterGrid = &grid
			hasGrid = true
		}
	}

	// Gather schedule and block IDs from grid if found
	var scheduleIDs []uuid.UUID
	var blockIDs []uuid.UUID

	addID := func(id *uuid.UUID, list *[]uuid.UUID) {
		if id != nil && *id != uuid.Nil {
			// check if already in list to avoid duplicates
			for _, item := range *list {
				if item == *id {
					return
				}
			}
			*list = append(*list, *id)
		}
	}

	if hasGrid {
		addID(grid.SundayScheduleID, &scheduleIDs)
		addID(grid.MondayScheduleID, &scheduleIDs)
		addID(grid.TuesdayScheduleID, &scheduleIDs)
		addID(grid.WednesdayScheduleID, &scheduleIDs)
		addID(grid.ThursdayScheduleID, &scheduleIDs)
		addID(grid.FridayScheduleID, &scheduleIDs)
		addID(grid.SaturdayScheduleID, &scheduleIDs)
		addID(grid.HolidayScheduleID, &scheduleIDs)

		addID(grid.SundayBlockID, &blockIDs)
		addID(grid.MondayBlockID, &blockIDs)
		addID(grid.TuesdayBlockID, &blockIDs)
		addID(grid.WednesdayBlockID, &blockIDs)
		addID(grid.ThursdayBlockID, &blockIDs)
		addID(grid.FridayBlockID, &blockIDs)
		addID(grid.SaturdayBlockID, &blockIDs)
		addID(grid.HolidayBlockID, &blockIDs)
	}

	// Fetch basic schedules
	if len(scheduleIDs) > 0 {
		schedQuery := `
			SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description, start_time::text, end_time::text, 
			       musical_block_duration_minutes, custom_durations_json, custom_styles_json, COALESCE(off_hours_behavior, 'SILENCE'), created_at
			FROM basic_schedules
			WHERE id = ANY($1)`
		rows, errSched := database.Pool.Query(ctx, schedQuery, scheduleIDs)
		if errSched == nil {
			defer rows.Close()
			for rows.Next() {
				var s models.BasicSchedule
				errScan := rows.Scan(
					&s.ID, &s.TenantID, &s.MediaType, &s.Name, &s.Description, &s.StartTime, &s.EndTime,
					&s.MusicalBlockDurationMinutes, &s.CustomDurationsJSON, &s.CustomStylesJSON, &s.OffHoursBehavior, &s.CreatedAt,
				)
				if errScan == nil {
					payload.BasicSchedules = append(payload.BasicSchedules, s)
				}
			}
		}
	}

	// Fetch commercial block configs
	if len(blockIDs) > 0 {
		blockQuery := `
			SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description, interval_minutes, block_limit_seconds,
			       percentage_time_announcement, percentage_temperature, percentage_institutional, percentage_offer, created_at
			FROM commercial_block_configs
			WHERE id = ANY($1)`
		rows, errBlock := database.Pool.Query(ctx, blockQuery, blockIDs)
		if errBlock == nil {
			defer rows.Close()
			for rows.Next() {
				var b models.CommercialBlockConfig
				errScan := rows.Scan(
					&b.ID, &b.TenantID, &b.MediaType, &b.Name, &b.Description, &b.IntervalMinutes, &b.BlockLimitSeconds,
					&b.PercentageTimeAnnouncement, &b.PercentageTemperature, &b.PercentageInstitutional, &b.PercentageOffer, &b.CreatedAt,
				)
				if errScan == nil {
					payload.CommercialBlockConfigs = append(payload.CommercialBlockConfigs, b)
				}
			}
		}
	}

	// 4.1 Parse CustomStylesJSON to find programmed style names and IDs
	programmedStyleNames := make(map[string]bool)
	var programmedStyleIDs []uuid.UUID

	for _, s := range payload.BasicSchedules {
		if s.CustomStylesJSON != nil && *s.CustomStylesJSON != "" {
			var stylesMap map[string][]string
			if err := json.Unmarshal([]byte(*s.CustomStylesJSON), &stylesMap); err == nil {
				for _, items := range stylesMap {
					for _, item := range items {
						if u, errParse := uuid.Parse(item); errParse == nil {
							// Unique check
							found := false
							for _, existingID := range programmedStyleIDs {
								if existingID == u {
									found = true
									break
								}
							}
							if !found {
								programmedStyleIDs = append(programmedStyleIDs, u)
							}
						} else {
							programmedStyleNames[item] = true
						}
					}
				}
			}
		}
	}

	if len(programmedStyleNames) > 0 {
		var names []string
		for name := range programmedStyleNames {
			names = append(names, name)
		}
		styleIDsQuery := `SELECT id FROM music_styles WHERE name = ANY($1)`
		rows, err := database.Pool.Query(ctx, styleIDsQuery, names)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id uuid.UUID
				if errScan := rows.Scan(&id); errScan == nil {
					found := false
					for _, existingID := range programmedStyleIDs {
						if existingID == id {
							found = true
							break
						}
					}
					if !found {
						programmedStyleIDs = append(programmedStyleIDs, id)
					}
				}
			}
		}
	}

	// Also include extra authorized styles from WeeklyMasterGrid into programmedStyleIDs
	if payload.WeeklyMasterGrid != nil && payload.WeeklyMasterGrid.AllowedManualStylesJSON != nil && *payload.WeeklyMasterGrid.AllowedManualStylesJSON != "" {
		var extraIDs []string
		if errExtra := json.Unmarshal([]byte(*payload.WeeklyMasterGrid.AllowedManualStylesJSON), &extraIDs); errExtra == nil {
			for _, sidStr := range extraIDs {
				if parsedUUID, errUUID := uuid.Parse(sidStr); errUUID == nil {
					found := false
					for _, existingID := range programmedStyleIDs {
						if existingID == parsedUUID {
							found = true
							break
						}
					}
					if !found {
						programmedStyleIDs = append(programmedStyleIDs, parsedUUID)
					}
				}
			}
		}
	}

	// Fetch media/music styles
	var mediaStyles []models.MusicStyle
	if len(programmedStyleIDs) > 0 {
		styleQuery := `
			SELECT id, name, COALESCE(description, '') AS description, 
			       COALESCE(base_directory, '') AS base_directory,
			       COALESCE(source_type, 'LOCAL') AS source_type,
			       COALESCE(stream_url, '') AS stream_url,
			       created_at, updated_at 
			FROM music_styles 
			WHERE id = ANY($1)`
		rowsStyle, errStyle := database.Pool.Query(ctx, styleQuery, programmedStyleIDs)
		if errStyle == nil {
			defer rowsStyle.Close()
			for rowsStyle.Next() {
				var ms models.MusicStyle
				errScan := rowsStyle.Scan(&ms.ID, &ms.Name, &ms.Description, &ms.BaseDirectory, &ms.SourceType, &ms.StreamURL, &ms.CreatedAt, &ms.UpdatedAt)
				if errScan == nil {
					mediaStyles = append(mediaStyles, ms)
				}
			}
		}
	}
	payload.MediaStyles = mediaStyles

	// Fetch music tracks
	var mediaLibrary []models.MusicTrack
	if len(programmedStyleIDs) > 0 {
		trackQuery := `
			SELECT id, music_style_id, file_name, display_name, duration_seconds, 
			       COALESCE(album, '') AS album, COALESCE(title, '') AS title, COALESCE(artist, '') AS artist, year, lufs, lra, COALESCE(loudness_score, '') AS loudness_score, cue_in, cue_out,
			       file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, created_at, updated_at
			FROM music_library
			WHERE is_sonic = true AND (cue_out - cue_in) <= 300 AND music_style_id = ANY($1)`
		rowsTrack, errTrack := database.Pool.Query(ctx, trackQuery, programmedStyleIDs)
		if errTrack == nil {
			defer rowsTrack.Close()
			for rowsTrack.Next() {
				var t models.MusicTrack
				errScan := rowsTrack.Scan(
					&t.ID, &t.MusicStyleID, &t.FileName, &t.DisplayName, &t.DurationSeconds,
					&t.Album, &t.Title, &t.Artist, &t.Year, &t.LUFS, &t.LRA, &t.LoudnessScore,
					&t.CueIn, &t.CueOut, &t.FileSizeBytes, &t.BitRateKbps, &t.SampleRateHz, &t.RiscoRuido, &t.ManualCue, &t.IsSonic, &t.CreatedAt, &t.UpdatedAt,
				)
				if errScan == nil {
					mediaLibrary = append(mediaLibrary, t)
				}
			}
		}
	}
	payload.MediaLibrary = mediaLibrary

	// Fetch commercials and events distributed to this store or tenant
	commQuery := `
		SELECT c.id, c.tenant_id, c.media_type, c.commercial_type, c.name, c.file_name, c.duration_seconds, c.insertions_per_day,
		       c.start_date, c.end_date, c.start_time::text, c.end_time::text,
		       c.run_on_sunday, c.run_on_monday, c.run_on_tuesday, c.run_on_wednesday, c.run_on_thursday, c.run_on_friday, c.run_on_saturday,
		       c.distribute_to_all, c.is_active, c.created_at, c.updated_at,
		       c.lufs, c.lra, COALESCE(c.loudness_score, '') AS loudness_score, c.cue_in, c.cue_out, c.file_size_bytes, c.bitrate_kbps, c.sample_rate_hz, c.risco_ruido, c.manual_cue, c.is_sonic, c.execution_time::text,
		       c.event_interruption_mode
		FROM commercials_and_events c
		WHERE c.is_active = true 
		  AND c.is_sonic = true
		  AND (c.cue_out - c.cue_in) <= 300
		  AND c.media_type = 'AUDIO'
		  AND (c.tenant_id = $1 OR c.tenant_id = '00000000-0000-0000-0000-000000000000')
		  AND (
		      c.distribute_to_all = true 
		      OR EXISTS (
		          SELECT 1 FROM commercial_distributions d 
		          WHERE d.commercial_id = c.id AND d.store_id = $2
		      )
		  )`
	rowsComm, errComm := database.Pool.Query(ctx, commQuery, store.TenantID, store.ID)
	if errComm == nil {
		defer rowsComm.Close()
		for rowsComm.Next() {
			var c models.CommercialsAndEvents
			errScan := rowsComm.Scan(
				&c.ID, &c.TenantID, &c.MediaType, &c.CommercialType, &c.Name, &c.FileName, &c.DurationSeconds, &c.InsertionsPerDay,
				&c.StartDate, &c.EndDate, &c.StartTime, &c.EndTime,
				&c.RunOnSunday, &c.RunOnMonday, &c.RunOnTuesday, &c.RunOnWednesday, &c.RunOnThursday, &c.RunOnFriday, &c.RunOnSaturday,
				&c.DistributeToAll, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
				&c.LUFS, &c.LRA, &c.LoudnessScore, &c.CueIn, &c.CueOut, &c.FileSizeBytes, &c.BitrateKbps, &c.SampleRateHz, &c.RiscoRuido, &c.ManualCue, &c.IsSonic, &c.ExecutionTime,
				&c.EventInterruptionMode,
			)
			if errScan == nil {
				payload.CommercialsAndEvents = append(payload.CommercialsAndEvents, c)
			}
		}
	}

	// Fetch voice packs / placas
	payload.VoicePacks = []VoicePackSyncItem{}
	queryVP := `
		SELECT DISTINCT vp.id, vp.name, vp.base_directory 
		FROM voice_packs vp
		LEFT JOIN voice_pack_stores vps ON vp.id = vps.voice_pack_id
		WHERE vp.is_active = true 
		  AND (
			vp.is_global = true 
			OR vps.store_id = $1 
			OR (vp.tenant_id = $2 AND NOT EXISTS (SELECT 1 FROM voice_pack_stores WHERE voice_pack_id = vp.id))
			OR vp.tenant_id IS NULL
		  )`
	rowsVP, errVP := database.Pool.Query(ctx, queryVP, store.ID, store.TenantID)
	if errVP == nil {
		defer rowsVP.Close()
		for rowsVP.Next() {
			var vp VoicePackSyncItem
			if rowsVP.Scan(&vp.ID, &vp.Name, &vp.BaseDirectory) == nil {
				vp.DownloadURL = "/api/v1/download/voice-pack?id=" + vp.ID
				payload.VoicePacks = append(payload.VoicePacks, vp)
			}
		}
	}

	// Fetch chimes
	payload.Chimes = []ChimeSyncItem{}
	rowsChime, errChime := database.Pool.Query(ctx, "SELECT id, name, file_name, duration_seconds, cue_in, cue_out, amplify FROM chimes WHERE is_active = true AND (tenant_id = $1 OR tenant_id IS NULL)", store.TenantID)
	if errChime == nil {
		defer rowsChime.Close()
		for rowsChime.Next() {
			var ch ChimeSyncItem
			if rowsChime.Scan(&ch.ID, &ch.Name, &ch.FileName, &ch.DurationSeconds, &ch.CueIn, &ch.CueOut, &ch.Amplify) == nil {
				ch.DownloadURL = "/api/v1/download/chime?id=" + ch.ID
				payload.Chimes = append(payload.Chimes, ch)
			}
		}
	}

	// Fetch pre-formatted warnings
	payload.PreFormattedWarnings = []PreFormattedWarningSyncItem{}
	rowsWarn, errWarn := database.Pool.Query(ctx, "SELECT id, name, file_name, duration_seconds, cue_in, cue_out, amplify FROM pre_formatted_warnings WHERE is_active = true AND (tenant_id = $1 OR tenant_id IS NULL)", store.TenantID)
	if errWarn == nil {
		defer rowsWarn.Close()
		for rowsWarn.Next() {
			var pw PreFormattedWarningSyncItem
			if rowsWarn.Scan(&pw.ID, &pw.Name, &pw.FileName, &pw.DurationSeconds, &pw.CueIn, &pw.CueOut, &pw.Amplify) == nil {
				pw.DownloadURL = "/api/v1/download/warning?id=" + pw.ID
				payload.PreFormattedWarnings = append(payload.PreFormattedWarnings, pw)
			}
		}
	}

	// Build consolidated AllowedManualStyles (Styles used in weekly schedule + Extra authorized styles)
	allowedMap := make(map[string]ManualStyleItem)

	// A) All styles used in the weekly schedules (custom_styles_json)
	for _, s := range payload.BasicSchedules {
		if s.CustomStylesJSON != nil && *s.CustomStylesJSON != "" {
			var stylesMap map[string][]string
			if errUnmarshal := json.Unmarshal([]byte(*s.CustomStylesJSON), &stylesMap); errUnmarshal == nil {
				for _, list := range stylesMap {
					for _, itemID := range list {
						if _, exists := allowedMap[itemID]; !exists {
							for _, ms := range payload.MediaStyles {
								if ms.ID.String() == itemID || ms.Name == itemID {
									sType := "music_style"
									sURL := ""
									if ms.SourceType == "STREAM" {
										sType = "streaming"
										sURL = ms.StreamURL
									}
									allowedMap[ms.ID.String()] = ManualStyleItem{
										ID:   ms.ID.String(),
										Name: ms.Name,
										Type: sType,
										URL:  sURL,
									}
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// B) Extra authorized styles from WeeklyMasterGrid
	if payload.WeeklyMasterGrid != nil && payload.WeeklyMasterGrid.AllowedManualStylesJSON != nil && *payload.WeeklyMasterGrid.AllowedManualStylesJSON != "" {
		var extraIDs []string
		if errExtra := json.Unmarshal([]byte(*payload.WeeklyMasterGrid.AllowedManualStylesJSON), &extraIDs); errExtra == nil {
			for _, extraID := range extraIDs {
				if _, exists := allowedMap[extraID]; !exists {
					for _, ms := range payload.MediaStyles {
						if ms.ID.String() == extraID || ms.Name == extraID {
							sType := "music_style"
							sURL := ""
							if ms.SourceType == "STREAM" {
								sType = "streaming"
								sURL = ms.StreamURL
							}
							allowedMap[ms.ID.String()] = ManualStyleItem{
								ID:   ms.ID.String(),
								Name: ms.Name,
								Type: sType,
								URL:  sURL,
							}
							break
						}
					}
				}
			}
		}
	}

	// C) Fallback: se nenhuma regra de restrição manual foi definida, disponibiliza todos os estilos e streamings da rede
	if len(allowedMap) == 0 {
		for _, ms := range payload.MediaStyles {
			sType := "music_style"
			sURL := ""
			if ms.SourceType == "STREAM" {
				sType = "streaming"
				sURL = ms.StreamURL
			}
			allowedMap[ms.ID.String()] = ManualStyleItem{
				ID:   ms.ID.String(),
				Name: ms.Name,
				Type: sType,
				URL:  sURL,
			}
		}
	}

	// Convert allowedMap to slice
	payload.AllowedManualStyles = make([]ManualStyleItem, 0, len(allowedMap))
	for _, itm := range allowedMap {
		payload.AllowedManualStyles = append(payload.AllowedManualStyles, itm)
	}

	// 5. Serialize the struct data into a beautiful clean JSON and respond to the edge client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payload)
}

// HandleDownloadTemplate gera dinamicamente o modelo de planilha Excel para download
func HandleDownloadTemplate(w http.ResponseWriter, r *http.Request) {
	// Validação de Sessão
	_, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Filiais"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		http.Error(w, "Erro ao criar planilha", http.StatusInternalServerError)
		return
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	headers := []string{
		"Nome da Filial", "Número Interno", "CNPJ da Matriz", "Nome da Regional",
		"CEP", "Rua", "Número", "Bairro", "Cidade", "Estado", "Telefone",
		"Nome do Gerente", "Habilitar Áudio (SIM/NÃO)", "Habilitar Vídeo (SIM/NÃO)", "Fuso Horário",
		"Grade de Áudio (Nome)", "Grade de Vídeo (Nome)",
	}

	// Estilo para o cabeçalho (Azul Premium, Negrito, Centralizado)
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		http.Error(w, "Erro ao criar estilo de cabeçalho", http.StatusInternalServerError)
		return
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Linha de Exemplo Explicativa
	exampleRow := []interface{}{
		"Loja Centro SP", "LOJA-101", "12.345.678/0001-99", "Regional Sudeste",
		"01001-000", "Avenida Paulista", "1000", "Bela Vista", "São Paulo", "SP",
		"(11) 99999-9999", "Carlos Silva", "SIM", "NÃO", "America/Sao_Paulo",
		"Grade Padrão Áudio SP", "Grade Padrão Vídeo SP",
	}
	for colIdx, val := range exampleRow {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		f.SetCellValue(sheetName, cell, val)
	}

	// Ajusta largura das colunas
	for colIdx := range headers {
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		f.SetColWidth(sheetName, colName, colName, 22)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=modelo_importacao_filiais.xlsx")
	if err := f.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleImportStores processa a importação de filiais em lote via planilha Excel (.xlsx)
func HandleImportStores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/stores", http.StatusSeeOther)
		return
	}

	// 1. Validação de Sessão
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 2. Parse Multipart Form (limite de 10MB)
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Redirect(w, r, "/stores?error=Erro+ao+processar+formulario", http.StatusSeeOther)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/stores?error=Nenhum+arquivo+enviado", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// 3. Abrir planilha usando excelize
	excelFile, err := excelize.OpenReader(file)
	if err != nil {
		http.Redirect(w, r, "/stores?error=Formato+de+arquivo+invalido.+Envie+uma+planilha+Excel+.xlsx", http.StatusSeeOther)
		return
	}
	defer excelFile.Close()

	sheets := excelFile.GetSheetList()
	if len(sheets) == 0 {
		http.Redirect(w, r, "/stores?error=Planilha+sem+abas", http.StatusSeeOther)
		return
	}

	rows, err := excelFile.GetRows(sheets[0])
	if err != nil {
		http.Redirect(w, r, "/stores?error=Erro+ao+ler+linhas+da+planilha", http.StatusSeeOther)
		return
	}

	if len(rows) <= 1 {
		http.Redirect(w, r, "/stores?error=A+planilha+esta+vazia+ou+contem+apenas+cabecalhos", http.StatusSeeOther)
		return
	}

	// 4. Carregar tenants e regionais para validação rápida em memória
	tenantsList, err := tenantRepo.GetAll(context.Background())
	if err != nil {
		http.Redirect(w, r, "/stores?error=Erro+ao+carregar+empresas+do+banco", http.StatusSeeOther)
		return
	}

	cleanCNPJ := func(s string) string {
		r := strings.NewReplacer(".", "", "-", "", "/", "", " ", "")
		return r.Replace(s)
	}

	// Mapear tenants por CNPJ
	tenantsMap := make(map[string]models.Tenant)
	for _, t := range tenantsList {
		tenantsMap[cleanCNPJ(t.DocumentNumber)] = t
	}

	// Mapear regionais por TenantID + RegionalName (lowercased)
	regionalsList, err := regionalRepo.GetAll(context.Background(), nil)
	if err != nil {
		regionalsList = []models.Regional{}
	}
	regionalsMap := make(map[string]models.Regional)
	for _, reg := range regionalsList {
		key := fmt.Sprintf("%s_%s", reg.TenantID.String(), strings.ToLower(strings.TrimSpace(reg.Name)))
		regionalsMap[key] = reg
	}

	// Carregar grades de programação semanais para cache em memória
	var gridsList []struct {
		ID        uuid.UUID
		TenantID  uuid.UUID
		MediaType string
		Name      string
	}
	gridRows, err := database.Pool.Query(context.Background(), "SELECT id, tenant_id, media_type, name FROM weekly_master_grids")
	if err == nil {
		defer gridRows.Close()
		for gridRows.Next() {
			var g struct {
				ID        uuid.UUID
				TenantID  uuid.UUID
				MediaType string
				Name      string
			}
			if errScan := gridRows.Scan(&g.ID, &g.TenantID, &g.MediaType, &g.Name); errScan == nil {
				gridsList = append(gridsList, g)
			}
		}
	}
	gridsMap := make(map[string]uuid.UUID)
	for _, g := range gridsList {
		key := fmt.Sprintf("%s_%s_%s", g.TenantID.String(), strings.ToLower(g.MediaType), strings.ToLower(strings.TrimSpace(g.Name)))
		gridsMap[key] = g.ID
	}

	getVal := func(row []string, colIdx int) string {
		if colIdx < len(row) {
			return strings.TrimSpace(row[colIdx])
		}
		return ""
	}

	var successCount, failCount int
	var failureMessages []string

	for idx, row := range rows[1:] {
		lineNum := idx + 2

		// Ignorar linhas completamente vazias
		isEmpty := true
		for _, col := range row {
			if strings.TrimSpace(col) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		storeName := getVal(row, 0)
		storeNumber := getVal(row, 1)
		cnpjMatriz := cleanCNPJ(getVal(row, 2))
		regionalName := getVal(row, 3)
		zipCode := getVal(row, 4)
		street := getVal(row, 5)
		number := getVal(row, 6)
		district := getVal(row, 7)
		city := getVal(row, 8)
		state := getVal(row, 9)
		phone := getVal(row, 10)
		managerName := getVal(row, 11)
		hasAudioStr := strings.ToLower(getVal(row, 12))
		hasVideoStr := strings.ToLower(getVal(row, 13))
		timezone := getVal(row, 14)
		audioGridName := getVal(row, 15)
		videoGridName := getVal(row, 16)

		// Validações obrigatórias
		if storeName == "" {
			failCount++
			failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Nome da Filial é obrigatório", lineNum))
			continue
		}
		if city == "" {
			failCount++
			failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Cidade é obrigatória", lineNum))
			continue
		}
		if state == "" {
			failCount++
			failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Estado é obrigatório", lineNum))
			continue
		}

		// Determina o TenantID
		var targetTenantID uuid.UUID
		if session.TenantID != nil {
			targetTenantID = *session.TenantID
			// Se informou CNPJ, ele precisa bater com o tenant logado
			if cnpjMatriz != "" {
				t, exists := tenantsMap[cnpjMatriz]
				if !exists || t.ID != targetTenantID {
					failCount++
					failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: CNPJ %s não pertence à sua empresa", lineNum, getVal(row, 2)))
					continue
				}
			}
		} else {
			// Administrador Master precisa de CNPJ na linha
			if cnpjMatriz == "" {
				failCount++
				failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: CNPJ da Matriz é obrigatório para administrador", lineNum))
				continue
			}
			t, exists := tenantsMap[cnpjMatriz]
			if !exists {
				failCount++
				failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Matriz com CNPJ %s não cadastrada", lineNum, getVal(row, 2)))
				continue
			}
			targetTenantID = t.ID
		}

		// Determina o RegionalID (opcional)
		var targetRegionalID *uuid.UUID
		if regionalName != "" {
			key := fmt.Sprintf("%s_%s", targetTenantID.String(), strings.ToLower(regionalName))
			reg, exists := regionalsMap[key]
			if !exists {
				failCount++
				failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Regional '%s' não encontrada para esta matriz", lineNum, regionalName))
				continue
			}
			targetRegionalID = &reg.ID
		}

		// Determina o AudioGridID (opcional)
		var targetAudioGridID *uuid.UUID
		if audioGridName != "" {
			keyTenant := fmt.Sprintf("%s_audio_%s", targetTenantID.String(), strings.ToLower(audioGridName))
			keyGlobal := fmt.Sprintf("00000000-0000-0000-0000-000000000000_audio_%s", strings.ToLower(audioGridName))
			if gid, exists := gridsMap[keyTenant]; exists {
				targetAudioGridID = &gid
			} else if gid, exists := gridsMap[keyGlobal]; exists {
				targetAudioGridID = &gid
			} else {
				failCount++
				failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Grade de Áudio '%s' não encontrada para esta matriz ou globalmente", lineNum, audioGridName))
				continue
			}
		}

		// Determina o VideoGridID (opcional)
		var targetVideoGridID *uuid.UUID
		if videoGridName != "" {
			keyTenant := fmt.Sprintf("%s_video_%s", targetTenantID.String(), strings.ToLower(videoGridName))
			keyGlobal := fmt.Sprintf("00000000-0000-0000-0000-000000000000_video_%s", strings.ToLower(videoGridName))
			if gid, exists := gridsMap[keyTenant]; exists {
				targetVideoGridID = &gid
			} else if gid, exists := gridsMap[keyGlobal]; exists {
				targetVideoGridID = &gid
			} else {
				failCount++
				failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Grade de Vídeo '%s' não encontrada para esta matriz ou globalmente", lineNum, videoGridName))
				continue
			}
		}

		// Se os IDs continuarem nulos, aplica os defaults do Tenant/Matriz
		var targetTenant models.Tenant
		for _, t := range tenantsList {
			if t.ID == targetTenantID {
				targetTenant = t
				break
			}
		}
		if targetAudioGridID == nil && targetTenant.DefaultAudioGridID != nil && *targetTenant.DefaultAudioGridID != uuid.Nil {
			targetAudioGridID = targetTenant.DefaultAudioGridID
		}
		if targetVideoGridID == nil && targetTenant.DefaultVideoGridID != nil && *targetTenant.DefaultVideoGridID != uuid.Nil {
			targetVideoGridID = targetTenant.DefaultVideoGridID
		}

		// Configura flags de Áudio e Vídeo (padrão Áudio: Sim, Vídeo: Não)
		hasAudio := true
		if hasAudioStr != "" {
			hasAudio = strings.HasPrefix(hasAudioStr, "s") || hasAudioStr == "1" || hasAudioStr == "true"
		}
		hasVideo := false
		if hasVideoStr != "" {
			hasVideo = strings.HasPrefix(hasVideoStr, "s") || hasVideoStr == "1" || hasVideoStr == "true"
		}

		newStore := models.Store{
			ID:          uuid.New(),
			TenantID:    targetTenantID,
			RegionalID:  targetRegionalID,
			AudioGridID: targetAudioGridID,
			VideoGridID: targetVideoGridID,
			StoreName:   storeName,
			StoreNumber: storeNumber,
			ZipCode:     zipCode,
			Street:      street,
			Number:      number,
			District:    district,
			City:        city,
			State:       state,
			Timezone:    timezone,
			Phone:       phone,
			ManagerName: managerName,
			HasAudio:    hasAudio,
			HasVideo:    hasVideo,
			IsActive:    true,
		}

		// Grava no banco
		err = storeRepo.Create(context.Background(), newStore)
		if err != nil {
			failCount++
			failureMessages = append(failureMessages, fmt.Sprintf("Linha %d: Erro ao persistir no banco (%v)", lineNum, err))
			continue
		}

		successCount++
	}

	var redirectURL string
	if failCount > 0 {
		var detail string
		if len(failureMessages) > 2 {
			detail = strings.Join(failureMessages[:2], "; ") + fmt.Sprintf("... e mais %d erros", len(failureMessages)-2)
		} else {
			detail = strings.Join(failureMessages, "; ")
		}
		if successCount > 0 {
			redirectURL = fmt.Sprintf("/stores?success=Importadas+%d+filiais&error=Falha+em+%d+linhas+(%s)", successCount, failCount, url.QueryEscape(detail))
		} else {
			redirectURL = fmt.Sprintf("/stores?error=Falha+em+%d+linhas+(%s)", failCount, url.QueryEscape(detail))
		}
	} else {
		redirectURL = fmt.Sprintf("/stores?success=Todas+as+%d+filiais+foram+importadas+com+sucesso", successCount)
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func DownloadMusicTrack(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure token from the HTTP Headers
	token := r.Header.Get("X-Store-Token")
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized. Missing X-Store-Token header"}`))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate token
	var storeID uuid.UUID
	errToken := database.Pool.QueryRow(ctx, "SELECT id FROM stores WHERE token = $1 AND is_active = true", token).Scan(&storeID)
	if errToken != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Store not found or inactive for the provided token"}`))
		return
	}

	trackIDStr := r.URL.Query().Get("id")
	trackID, err := uuid.Parse(trackIDStr)
	if err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	var styleDir string
	var fileName string
	query := `
		SELECT s.base_directory, t.file_name 
		FROM music_library t
		JOIN music_styles s ON t.music_style_id = s.id
		WHERE t.id = $1`
	err = database.Pool.QueryRow(ctx, query, trackID).Scan(&styleDir, &fileName)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	// Read and serve file
	filePath := filepath.Join(styleDir, fileName)
	http.ServeFile(w, r, filePath)
}

func DownloadCommercial(w http.ResponseWriter, r *http.Request) {
	// 1. Extract the secure token from the HTTP Headers
	token := r.Header.Get("X-Store-Token")
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized. Missing X-Store-Token header"}`))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate token
	var storeID uuid.UUID
	var tenantID uuid.UUID
	errToken := database.Pool.QueryRow(ctx, "SELECT id, tenant_id FROM stores WHERE token = $1 AND is_active = true", token).Scan(&storeID, &tenantID)
	if errToken != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Store not found or inactive for the provided token"}`))
		return
	}

	commIDStr := r.URL.Query().Get("id")
	commID, err := uuid.Parse(commIDStr)
	if err != nil {
		http.Error(w, "Invalid commercial ID", http.StatusBadRequest)
		return
	}

	var fileName string
	var cTenantID uuid.UUID
	var cType string
	query := `
		SELECT tenant_id, file_name, commercial_type
		FROM commercials_and_events
		WHERE id = $1`
	err = database.Pool.QueryRow(ctx, query, commID).Scan(&cTenantID, &fileName, &cType)
	if err != nil {
		http.Error(w, "Commercial not found", http.StatusNotFound)
		return
	}

	// Validate ownership/distribution
	if cTenantID != tenantID && cTenantID != uuid.Nil {
		// Verify if it is distributed to this store
		var count int
		_ = database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM commercial_distributions WHERE commercial_id = $1 AND store_id = $2", commID, storeID).Scan(&count)
		if count == 0 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	// Serving file from midias/comerciais/<tenant_id>/<folderName>/<fileName>
	folderName := GetCommercialFolderName(models.CommercialType(cType))
	filePath := filepath.Join("midias", "comerciais", cTenantID.String(), folderName, fileName)
	http.ServeFile(w, r, filePath)
}

// IcecastSource defines the source details parsed from Icecast JSON API
type IcecastSource struct {
	ListenURL string      `json:"listenurl"`
	Listeners int         `json:"listeners"`
	Title     string      `json:"title"`
	Artist    string      `json:"artist"`
}

// RadioStatusRow contains display information for a server-side radio status row
type RadioStatusRow struct {
	StoreID         uuid.UUID
	StoreName       string
	StoreNumber     string
	TenantTradeName string
	MountPoint      string
	IsOnline        bool
	CurrentSong     string
	Listeners       int
	IcecastHost     string
	IcecastPort     int
}

// RadiosStatusPageData holds data passed to the radios_status.html template
type RadiosStatusPageData struct {
	Title          string
	LoggedUser     string
	User           map[string]string
	Theme          map[string]string
	Radios         []RadioStatusRow
	TotalRadios    int
	OnlineRadios   int
	OfflineRadios  int
}

// parseIcecastSources connects to local Icecast status-json.xsl and parses active streams
func parseIcecastSources() map[string]IcecastSource {
	sourcesMap := make(map[string]IcecastSource)
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:8000/status-json.xsl")
	if err != nil {
		return sourcesMap
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return sourcesMap
	}

	icestats, ok := data["icestats"].(map[string]interface{})
	if !ok {
		return sourcesMap
	}

	sourceVal, exists := icestats["source"]
	if !exists {
		return sourcesMap
	}

	processSourceMap := func(sMap map[string]interface{}) {
		var src IcecastSource
		if lUrl, ok := sMap["listenurl"].(string); ok {
			src.ListenURL = lUrl
		}
		if listenersVal, ok := sMap["listeners"]; ok {
			switch v := listenersVal.(type) {
			case float64:
				src.Listeners = int(v)
			case int:
				src.Listeners = v
			}
		}
		if titleVal, ok := sMap["title"].(string); ok {
			src.Title = titleVal
		}
		if artistVal, ok := sMap["artist"].(string); ok {
			src.Artist = artistVal
		}
		u, err := url.Parse(src.ListenURL)
		if err == nil {
			sourcesMap[u.Path] = src
		} else {
			if idx := strings.LastIndex(src.ListenURL, "/"); idx != -1 {
				sourcesMap[src.ListenURL[idx:]] = src
			}
		}
	}

	switch val := sourceVal.(type) {
	case []interface{}:
		for _, item := range val {
			if sMap, ok := item.(map[string]interface{}); ok {
				processSourceMap(sMap)
			}
		}
	case map[string]interface{}:
		processSourceMap(val)
	}

	return sourcesMap
}

// GetStoreMountPoint derives a safe, consistent Icecast mount path for a store
func GetStoreMountPoint(store models.Store) string {
	if store.StoreNumber != "" {
		clean := strings.ToLower(strings.ReplaceAll(store.StoreNumber, " ", "_"))
		tenantStr := store.TenantID.String()
		tenantSuffix := "0000"
		if len(tenantStr) >= 4 {
			tenantSuffix = tenantStr[len(tenantStr)-4:]
		}
		clean = "m" + tenantSuffix + "_l" + strings.TrimPrefix(clean, "/")
		return "/" + clean
	}
	return "/loja_" + store.ID.String()[:8]
}

// HandleRadiosStatusPage displays the active server-side radios status dashboard
func HandleRadiosStatusPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch all stores in database
	allStores, err := storeRepo.GetAll(context.Background(), session.TenantID)
	if err != nil {
		allStores = []models.Store{}
	}

	// Retrieve active sources from Icecast
	icecastSources := parseIcecastSources()

	// Filter and compile status for Server-side radios
	var radios []RadioStatusRow
	var onlineCount int
	var offlineCount int

	for _, s := range allStores {
		if s.RadioType != "server" {
			continue
		}

		mount := GetStoreMountPoint(s)
		source, isOnline := icecastSources[mount]

		var currentSong string
		if isOnline {
			onlineCount++
			if source.Title != "" {
				currentSong = source.Title
				if source.Artist != "" {
					currentSong = source.Artist + " - " + source.Title
				}
			}
		} else {
			offlineCount++
		}

		radios = append(radios, RadioStatusRow{
			StoreID:         s.ID,
			StoreName:       s.StoreName,
			StoreNumber:     s.StoreNumber,
			TenantTradeName: s.TenantTradeName,
			MountPoint:      mount,
			IsOnline:        isOnline,
			CurrentSong:     currentSong,
			Listeners:       source.Listeners,
			IcecastHost:     "stream.indoormedia.com.br", // Standard Icecast server host
			IcecastPort:     8000,
		})
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "/static/img/logo-imc.png",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	data := RadiosStatusPageData{
		Title:         "Status das Rádios do Servidor",
		LoggedUser:    session.UserName,
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:         theme,
		Radios:        radios,
		TotalRadios:   len(radios),
		OnlineRadios:  onlineCount,
		OfflineRadios: offlineCount,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/radios_status.html"))
	tmpl.Execute(w, data)
}

// EnsureStorePlayHistoryTableExists garante que a tabela store_play_history exista no PostgreSQL
func EnsureStorePlayHistoryTableExists() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	CREATE TABLE IF NOT EXISTS store_play_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		track_id VARCHAR(255) NOT NULL DEFAULT '',
		track_type VARCHAR(50) NOT NULL DEFAULT 'music',
		display_name TEXT NOT NULL DEFAULT '',
		artist TEXT NOT NULL DEFAULT '',
		played_at TIMESTAMP WITH TIME ZONE NOT NULL,
		duration_seconds INTEGER NOT NULL DEFAULT 0,
		style_name VARCHAR(100) NOT NULL DEFAULT '',
		play_status VARCHAR(50) NOT NULL DEFAULT 'executada',
		planned_block_time VARCHAR(20) NOT NULL DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT uq_store_track_played UNIQUE (store_id, track_id, played_at)
	);
	CREATE INDEX IF NOT EXISTS idx_sph_store_played_at ON store_play_history(store_id, played_at);
	CREATE INDEX IF NOT EXISTS idx_sph_tenant_played_at ON store_play_history(tenant_id, played_at);
	CREATE INDEX IF NOT EXISTS idx_sph_track_type ON store_play_history(track_type);
	`
	_, err := database.Pool.Exec(ctx, query)
	if err != nil {
		fmt.Println("[DB Error] Falha ao criar tabela store_play_history:", err)
	}
}

// HandleSyncPlayHistoryBatch recebe lote de histórico de execuções (playout) do player da loja
func HandleSyncPlayHistoryBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("X-Store-Token")
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized. Missing X-Store-Token header"}`))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var store models.Store
	err := database.Pool.QueryRow(ctx, "SELECT id, tenant_id FROM stores WHERE token = $1 AND is_active = true", token).Scan(&store.ID, &store.TenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Store not found for token"}`))
		return
	}

	var req models.PlayHistoryBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid JSON body: ` + err.Error() + `"}`))
		return
	}

	if len(req.Records) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.PlayHistoryBatchResponse{
			Status:        "success",
			IngestedCount: 0,
			Message:       "No records to ingest",
		})
		return
	}

	EnsureStorePlayHistoryTableExists()

	ingested := 0
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	if loc == nil {
		loc = time.Local
	}

	for _, rec := range req.Records {
		playedAt, parseErr := time.ParseInLocation("2006-01-02 15:04:05", rec.PlayedAt, loc)
		if parseErr != nil {
			playedAt, parseErr = time.Parse(time.RFC3339, rec.PlayedAt)
			if parseErr != nil {
				playedAt = time.Now().In(loc)
			}
		}

		insertQuery := `
			INSERT INTO store_play_history (
				store_id, tenant_id, track_id, track_type, display_name, artist, 
				played_at, duration_seconds, style_name, play_status, planned_block_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (store_id, track_id, played_at) DO NOTHING
		`
		tag, execErr := database.Pool.Exec(ctx, insertQuery,
			store.ID, store.TenantID, rec.TrackID, rec.TrackType, rec.DisplayName, rec.Artist,
			playedAt, rec.DurationSeconds, rec.StyleName, rec.PlayStatus, rec.PlannedBlockTime,
		)
		if execErr == nil && tag.RowsAffected() > 0 {
			ingested++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models.PlayHistoryBatchResponse{
		Status:        "success",
		IngestedCount: ingested,
		Message:       fmt.Sprintf("Ingested %d of %d records", ingested, len(req.Records)),
	})
}

// PurgeOldPlayHistory remove registros de histórico com mais de 90 dias do PostgreSQL central
func PurgeOldPlayHistory() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `DELETE FROM store_play_history WHERE played_at < NOW() - INTERVAL '90 days';`
	tag, err := database.Pool.Exec(ctx, query)
	if err != nil {
		fmt.Printf("[Retention Error] Falha ao expirar histórico antigo: %v\n", err)
	} else if tag.RowsAffected() > 0 {
		fmt.Printf("[Retention] Expirados %d registros de play_history com mais de 90 dias.\n", tag.RowsAffected())
	}
}

