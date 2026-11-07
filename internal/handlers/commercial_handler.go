package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"imc-server/internal/models"
	"imc-server/internal/repository"
	"imc-server/internal/utils"

	"github.com/google/uuid"
)

var commRepo = repository.NewCommercialRepository()

func GetCommercialFolderName(cType models.CommercialType) string {
	switch cType {
	case models.CommercialInstitutional:
		return "institutional"
	case models.CommercialBlockOpening:
		return "open_blocks"
	case models.CommercialBlockClosing:
		return "close_blocks"
	case models.CommercialStandard:
		return "advertisements"
	case models.CommercialOffer:
		return "offers"
	case models.CommercialEvent:
		return "events"
	case models.CommercialGeneralInfo:
		return "general_information"
	case models.CommercialRightTime:
		return "right_time"
	case models.CommercialTemperature:
		return "temperature"
	default:
		return "advertisements"
	}
}

func HandleCommercialsPage(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()
	
	var stores []models.Store
	var commercials []models.CommercialsAndEvents
	
	if session.TenantID == nil {
		commercials, _ = commRepo.GetAll(ctx)
		stores, _ = storeRepo.GetAll(ctx, nil)
	} else {
		commercials, _ = commRepo.GetByTenant(ctx, *session.TenantID)
		stores, _ = storeRepo.GetAll(ctx, session.TenantID)
	}

	if commercials == nil {
		commercials = []models.CommercialsAndEvents{}
	}
	if stores == nil {
		stores = []models.Store{}
	}

	// Fetch tenant trade names
	tenants, _ := tenantRepo.GetAll(ctx)
	tenantMap := make(map[uuid.UUID]string)
	for _, t := range tenants {
		tenantMap[t.ID] = t.TradeName
	}
	tenantMap[uuid.MustParse("00000000-0000-0000-0000-000000000000")] = "Global / Geral (Nacional)"

	// Fetch regionals
	regionals, _ := regionalRepo.GetAll(ctx, session.TenantID)
	if regionals == nil {
		regionals = []models.Regional{}
	}

	// Build store mapping lists for fast client-side filtering attributes
	storeToRegionalMap := make(map[uuid.UUID]string)
	tenantStoresMap := make(map[uuid.UUID][]string)
	tenantRegionalsMap := make(map[uuid.UUID][]string)
	
	var allStoreIDs []string
	var allRegionalIDs []string
	
	// Pre-populate unique regional map to avoid duplicates
	tenantRegionalsSet := make(map[uuid.UUID]map[string]bool)
	allRegionalsSet := make(map[string]bool)
	
	for _, s := range stores {
		sIDStr := s.ID.String()
		allStoreIDs = append(allStoreIDs, sIDStr)
		tenantStoresMap[s.TenantID] = append(tenantStoresMap[s.TenantID], sIDStr)
		
		if s.RegionalID != nil {
			regIDStr := s.RegionalID.String()
			storeToRegionalMap[s.ID] = regIDStr
			
			if !allRegionalsSet[regIDStr] {
				allRegionalsSet[regIDStr] = true
				allRegionalIDs = append(allRegionalIDs, regIDStr)
			}
			
			if _, exists := tenantRegionalsSet[s.TenantID]; !exists {
				tenantRegionalsSet[s.TenantID] = make(map[string]bool)
			}
			if !tenantRegionalsSet[s.TenantID][regIDStr] {
				tenantRegionalsSet[s.TenantID][regIDStr] = true
				tenantRegionalsMap[s.TenantID] = append(tenantRegionalsMap[s.TenantID], regIDStr)
			}
		}
	}

	type CommercialDetail struct {
		models.CommercialsAndEvents
		TenantName  string
		StoreIDs    string
		RegionalIDs string
	}
	
	var detailedCommercials []CommercialDetail
	for _, c := range commercials {
		var cStoreIDs []string
		var cRegionalIDs []string
		
		if c.DistributeToAll {
			if c.TenantID == uuid.Nil || c.TenantID.String() == "00000000-0000-0000-0000-000000000000" {
				cStoreIDs = allStoreIDs
				cRegionalIDs = allRegionalIDs
			} else {
				cStoreIDs = tenantStoresMap[c.TenantID]
				cRegionalIDs = tenantRegionalsMap[c.TenantID]
			}
		} else {
			// Query specific distribution from DB
			sUUIDs, _ := commRepo.GetDistributedStores(ctx, c.ID)
			cStoreIDs = make([]string, 0, len(sUUIDs))
			cRegionalIDs = make([]string, 0, len(sUUIDs))
			regSet := make(map[string]bool)
			
			for _, sID := range sUUIDs {
				cStoreIDs = append(cStoreIDs, sID.String())
				if regID, exists := storeToRegionalMap[sID]; exists && regID != "" {
					if !regSet[regID] {
						regSet[regID] = true
						cRegionalIDs = append(cRegionalIDs, regID)
					}
				}
			}
		}

		detailedCommercials = append(detailedCommercials, CommercialDetail{
			CommercialsAndEvents: c,
			TenantName:           tenantMap[c.TenantID],
			StoreIDs:             strings.Join(cStoreIDs, ","),
			RegionalIDs:          strings.Join(cRegionalIDs, ","),
		})
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	data := map[string]interface{}{
		"Title":         "Gerenciamento de Comerciais & Eventos",
		"LoggedUser":    session.UserName,
		"Commercials":   detailedCommercials,
		"Stores":        stores,
		"Tenants":       tenants,
		"Regionals":     regionals,
		"User": map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		"Theme":         theme,
		"IsGlobalAdmin": session.TenantID == nil,
	}

	tmpl := template.New("commercials.html").Funcs(template.FuncMap{
		"formatDuration": func(secs int) string {
			m := secs / 60
			s := secs % 60
			return fmt.Sprintf("%02d:%02d", m, s)
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return "--"
			}
			return t.Format("02/01/2006")
		},
		"formatType": func(t models.CommercialType) string {
			switch t {
			case models.CommercialInstitutional:
				return "Institucional"
			case models.CommercialBlockOpening:
				return "Abertura de Bloco"
			case models.CommercialBlockClosing:
				return "Fechamento de Bloco"
			case models.CommercialStandard:
				return "Comercial"
			case models.CommercialOffer:
				return "Oferta"
			case models.CommercialEvent:
				return "Evento"
			case models.CommercialGeneralInfo:
				return "Informativo"
			case models.CommercialRightTime:
				return "Hora Certa"
			case models.CommercialTemperature:
				return "Temperatura"
			default:
				return string(t)
			}
		},
		"isOverFiveMinutes": func(cueIn, cueOut float64) bool {
			return (cueOut - cueIn) > 300.0
		},
		"formatDateTime": func(t time.Time) string {
			if t.IsZero() {
				return "--"
			}
			return t.Format("02/01/2006 15:04:05")
		},
	})
	tmpl = template.Must(tmpl.ParseFiles("web/templates/commercials.html"))
	tmpl.Execute(w, data)
}

func HandleCreateCommercial(w http.ResponseWriter, r *http.Request) {
	var lastCreated *models.CommercialsAndEvents
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/commercials", http.StatusSeeOther)
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

	err = r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Limite de tamanho excedido", http.StatusBadRequest)
		return
	}

	cTypeStr := r.FormValue("commercial_type")
	insertionsStr := r.FormValue("insertions_per_day")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	startTimeStr := r.FormValue("start_time")
	endTimeStr := r.FormValue("end_time")
	executionTimeStr := r.FormValue("execution_time")
	eventInterruptionMode := r.FormValue("event_interruption_mode")
	if eventInterruptionMode == "" {
		eventInterruptionMode = "SOFT"
	}

	distributeToAll := r.FormValue("distribute_to_all") == "true" || r.FormValue("distribute_to_all") == "on"
	noEndDate := r.FormValue("no_end_date") == "true" || r.FormValue("no_end_date") == "on"

	runSunday := r.FormValue("run_on_sunday") != ""
	runMonday := r.FormValue("run_on_monday") != ""
	runTuesday := r.FormValue("run_on_tuesday") != ""
	runWednesday := r.FormValue("run_on_wednesday") != ""
	runThursday := r.FormValue("run_on_thursday") != ""
	runFriday := r.FormValue("run_on_friday") != ""
	runSaturday := r.FormValue("run_on_saturday") != ""

	if !runSunday && !runMonday && !runTuesday && !runWednesday && !runThursday && !runFriday && !runSaturday {
		runSunday, runMonday, runTuesday, runWednesday, runThursday, runFriday, runSaturday = true, true, true, true, true, true, true
	}

	insertions, _ := strconv.Atoi(insertionsStr)
	if startTimeStr == "" {
		startTimeStr = "00:00:00"
	} else if len(startTimeStr) == 5 {
		startTimeStr += ":00"
	}
	if endTimeStr == "" {
		endTimeStr = "23:59:59"
	} else if len(endTimeStr) == 5 {
		endTimeStr += ":59"
	}

	var startDate, endDate *time.Time
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}
	// If noEndDate is checked, endDate is set to nil (NULL in database)
	if !noEndDate && endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &t
		}
	}

	var targetTenantIDs []uuid.UUID
	if session.TenantID == nil {
		tenantPicks := r.Form["tenant_id"]
		for _, pick := range tenantPicks {
			if tID, err := uuid.Parse(pick); err == nil {
				targetTenantIDs = append(targetTenantIDs, tID)
			}
		}
		if len(targetTenantIDs) == 0 {
			targetTenantIDs = append(targetTenantIDs, uuid.MustParse("00000000-0000-0000-0000-000000000000")) // Global Tenant
		}
	} else {
		targetTenantIDs = append(targetTenantIDs, *session.TenantID)
	}

	// Retrieve multiple files from multipart form
	formFiles := r.MultipartForm.File["files"]
	if len(formFiles) == 0 {
		http.Error(w, "Nenhum arquivo enviado", http.StatusBadRequest)
		return
	}

	// 1. Validation Phase: check for duplicate names in the batch and in the database for each target tenant
	batchNames := make(map[string]bool)
	for _, fileHeader := range formFiles {
		fileName := filepath.Base(fileHeader.Filename)
		
		// Prevent duplicates within the same batch upload
		if batchNames[fileName] {
			http.Error(w, fmt.Sprintf("Nome de arquivo duplicado no mesmo envio: %s", fileName), http.StatusBadRequest)
			return
		}
		batchNames[fileName] = true

		// Prevent duplicates in database for the selected tenants
		for _, tenantID := range targetTenantIDs {
			exists, err := commRepo.ExistsByName(r.Context(), tenantID, fileName)
			if err != nil {
				http.Error(w, "Falha ao validar duplicidade no banco: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if exists {
				http.Error(w, fmt.Sprintf("Já existe um comercial cadastrado com o nome '%s' para a empresa selecionada.", fileName), http.StatusBadRequest)
				return
			}
		}
	}

	// 2. Process Phase: save files, extract metadata, create records for each target tenant
	for _, targetTenantID := range targetTenantIDs {
		folderName := GetCommercialFolderName(models.CommercialType(cTypeStr))
		baseDir := filepath.Join("midias", "comerciais", targetTenantID.String(), folderName)
		_ = os.MkdirAll(baseDir, 0755)

		var parsedStoreIDs []uuid.UUID
		if !distributeToAll {
			tenantStores, _ := storeRepo.GetAll(r.Context(), &targetTenantID)
			tenantStoreSet := make(map[uuid.UUID]bool)
			for _, ts := range tenantStores {
				tenantStoreSet[ts.ID] = true
			}

			storeIDs := r.Form["store_ids"]
			for _, sIDStr := range storeIDs {
				if sID, err := uuid.Parse(sIDStr); err == nil {
					if tenantStoreSet[sID] {
						parsedStoreIDs = append(parsedStoreIDs, sID)
					}
				}
			}
		}

		for _, fileHeader := range formFiles {
			safeFileName := filepath.Base(fileHeader.Filename)
			fullPath := filepath.Join(baseDir, safeFileName)

			// Open uploaded file
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Falha ao abrir arquivo enviado: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Create file on disk
			out, err := os.Create(fullPath)
			if err != nil {
				file.Close()
				http.Error(w, "Falha ao criar arquivo no disco", http.StatusInternalServerError)
				return
			}

			_, err = io.Copy(out, file)
			file.Close()
			out.Close()
			if err != nil {
				_ = os.Remove(fullPath)
				http.Error(w, "Falha ao gravar arquivo no disco", http.StatusInternalServerError)
				return
			}

			meta, errMeta := utils.ExtractMetadata(fullPath)
			if errMeta != nil {
				_ = os.Remove(fullPath)
				http.Error(w, "Upload rejeitado: "+errMeta.Error(), http.StatusBadRequest)
				return
			}
			duration := meta.Duration

			var executionTime *string
			if models.CommercialType(cTypeStr) == models.CommercialEvent && executionTimeStr != "" {
				if len(executionTimeStr) == 5 {
					executionTimeStr += ":00"
				}
				executionTime = &executionTimeStr
			}

			c := models.CommercialsAndEvents{
				ID:               uuid.New(),
				TenantID:         targetTenantID,
				MediaType:        models.MediaAudio,
				CommercialType:   models.CommercialType(cTypeStr),
				Name:             safeFileName, // o nome ficaria sendo o nome do arquivo
				FileName:         safeFileName,
				DurationSeconds:  duration,
				InsertionsPerDay: insertions,
				StartDate:        startDate,
				EndDate:          endDate,
				StartTime:        startTimeStr,
				EndTime:          endTimeStr,
				RunOnSunday:      runSunday,
				RunOnMonday:      runMonday,
				RunOnTuesday:     runTuesday,
				RunOnWednesday:   runWednesday,
				RunOnThursday:    runThursday,
				RunOnFriday:      runFriday,
				RunOnSaturday:    runSaturday,
				DistributeToAll:  distributeToAll,
				IsActive:         true,
				
				LUFS:             meta.LUFS,
				LRA:              meta.LRA,
				LoudnessScore:    meta.LoudnessScore,
				CueIn:            meta.CueIn,
				CueOut:           meta.CueOut,
				FileSizeBytes:    meta.FileSizeBytes,
				BitrateKbps:      meta.BitRateKbps,
				SampleRateHz:     meta.SampleRateHz,
				RiscoRuido:       meta.NoiseRisk,
				IsSonic:          meta.IsSonic,
				ManualCue:        false,
				ExecutionTime:    executionTime,
				EventInterruptionMode: eventInterruptionMode,
			}

			err = commRepo.Create(r.Context(), c)
			if err != nil {
				_ = os.Remove(fullPath)
				http.Error(w, "Falha ao cadastrar no banco de dados: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Save store list distributions
			if !distributeToAll && len(parsedStoreIDs) > 0 {
				err = commRepo.SaveDistribution(r.Context(), c.ID, parsedStoreIDs)
				if err != nil {
					http.Error(w, "Falha ao salvar distribuição das lojas: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

			lastCreated = &c
		}
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"success": true}
		if lastCreated != nil {
			resp["name"] = lastCreated.Name
			resp["loudness_score"] = lastCreated.LoudnessScore
			resp["duration"] = lastCreated.DurationSeconds
			resp["lufs"] = lastCreated.LUFS
			resp["risco_ruido"] = lastCreated.RiscoRuido
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	http.Redirect(w, r, "/commercials", http.StatusSeeOther)
}

func HandleDeleteCommercial(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/commercials", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, id)
	if err != nil {
		http.Redirect(w, r, "/commercials", http.StatusSeeOther)
		return
	}

	folderName := GetCommercialFolderName(c.CommercialType)
	filePath := filepath.Join("midias", "comerciais", c.TenantID.String(), folderName, c.FileName)
	_ = os.Remove(filePath)

	_ = commRepo.Delete(ctx, id)

	http.Redirect(w, r, "/commercials", http.StatusSeeOther)
}

func HandleGetCommercialDistribution(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Comercial não encontrado", http.StatusNotFound)
		return
	}

	storeIDs, err := commRepo.GetDistributedStores(ctx, id)
	if err != nil {
		storeIDs = []uuid.UUID{}
	}

	response := map[string]interface{}{
		"commercial_id":     c.ID,
		"distribute_to_all": c.DistributeToAll,
		"store_ids":         storeIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HandleSaveCommercialDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CommercialID    string   `json:"commercial_id"`
		DistributeToAll bool     `json:"distribute_to_all"`
		StoreIDs        []string `json:"store_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	cID, err := uuid.Parse(req.CommercialID)
	if err != nil {
		http.Error(w, "ID do comercial inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, cID)
	if err != nil {
		http.Error(w, "Comercial não encontrado", http.StatusNotFound)
		return
	}

	c.DistributeToAll = req.DistributeToAll
	err = commRepo.Update(ctx, *c)
	if err != nil {
		http.Error(w, "Falha ao atualizar dados de distribuição: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var stores []uuid.UUID
	if !req.DistributeToAll {
		for _, sStr := range req.StoreIDs {
			if sID, err := uuid.Parse(sStr); err == nil {
				stores = append(stores, sID)
			}
		}
	}

	err = commRepo.SaveDistribution(ctx, cID, stores)
	if err != nil {
		http.Error(w, "Falha ao salvar lojas no banco: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

func HandleUpdateCommercial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/commercials", http.StatusSeeOther)
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

	err = r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Limite de tamanho excedido", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		http.Error(w, "Comercial não encontrado", http.StatusNotFound)
		return
	}

	cTypeStr := r.FormValue("commercial_type")
	insertionsStr := r.FormValue("insertions_per_day")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	startTimeStr := r.FormValue("start_time")
	endTimeStr := r.FormValue("end_time")
	executionTimeStr := r.FormValue("execution_time")
	eventInterruptionMode := r.FormValue("event_interruption_mode")
	if eventInterruptionMode == "" {
		eventInterruptionMode = "SOFT"
	}

	distributeToAll := r.FormValue("distribute_to_all") == "true" || r.FormValue("distribute_to_all") == "on"
	noEndDate := r.FormValue("no_end_date") == "true" || r.FormValue("no_end_date") == "on"

	runSunday := r.FormValue("run_on_sunday") != ""
	runMonday := r.FormValue("run_on_monday") != ""
	runTuesday := r.FormValue("run_on_tuesday") != ""
	runWednesday := r.FormValue("run_on_wednesday") != ""
	runThursday := r.FormValue("run_on_thursday") != ""
	runFriday := r.FormValue("run_on_friday") != ""
	runSaturday := r.FormValue("run_on_saturday") != ""

	if !runSunday && !runMonday && !runTuesday && !runWednesday && !runThursday && !runFriday && !runSaturday {
		runSunday, runMonday, runTuesday, runWednesday, runThursday, runFriday, runSaturday = true, true, true, true, true, true, true
	}

	insertions, _ := strconv.Atoi(insertionsStr)
	if startTimeStr == "" {
		startTimeStr = "00:00:00"
	} else if len(startTimeStr) == 5 {
		startTimeStr += ":00"
	}
	if endTimeStr == "" {
		endTimeStr = "23:59:59"
	} else if len(endTimeStr) == 5 {
		endTimeStr += ":59"
	}

	var startDate, endDate *time.Time
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}
	if !noEndDate && endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &t
		}
	}

	var targetTenantID uuid.UUID
	if session.TenantID == nil {
		tenantPick := r.FormValue("tenant_id")
		if tenantPick != "" {
			targetTenantID, _ = uuid.Parse(tenantPick)
		}
		if targetTenantID == uuid.Nil {
			targetTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		}
	} else {
		targetTenantID = *session.TenantID
	}

	var executionTime *string
	if models.CommercialType(cTypeStr) == models.CommercialEvent && executionTimeStr != "" {
		if len(executionTimeStr) == 5 {
			executionTimeStr += ":00"
		}
		executionTime = &executionTimeStr
	}

	formFiles := r.MultipartForm.File["files"]
	hasNewFile := len(formFiles) > 0

	oldFolderName := GetCommercialFolderName(c.CommercialType)
	oldFilePath := filepath.Join("midias", "comerciais", c.TenantID.String(), oldFolderName, c.FileName)

	newFolderName := GetCommercialFolderName(models.CommercialType(cTypeStr))
	newBaseDir := filepath.Join("midias", "comerciais", targetTenantID.String(), newFolderName)
	_ = os.MkdirAll(newBaseDir, 0755)

	if hasNewFile {
		fileHeader := formFiles[0]
		newFileName := filepath.Base(fileHeader.Filename)
		
		if newFileName != c.FileName || targetTenantID != c.TenantID {
			exists, err := commRepo.ExistsByName(ctx, targetTenantID, newFileName)
			if err == nil && exists {
				http.Error(w, fmt.Sprintf("Já existe um comercial cadastrado com o nome '%s' para esta empresa.", newFileName), http.StatusBadRequest)
				return
			}
		}

		_ = os.Remove(oldFilePath)

		newFilePath := filepath.Join(newBaseDir, newFileName)
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Falha ao abrir arquivo enviado: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		out, err := os.Create(newFilePath)
		if err != nil {
			http.Error(w, "Falha ao criar arquivo no disco", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			_ = os.Remove(newFilePath)
			http.Error(w, "Falha ao gravar arquivo no disco", http.StatusInternalServerError)
			return
		}

		meta, errMeta := utils.ExtractMetadata(newFilePath)
		if errMeta != nil {
			_ = os.Remove(newFilePath)
			http.Error(w, "Upload rejeitado: "+errMeta.Error(), http.StatusBadRequest)
			return
		}
		duration := meta.Duration

		c.Name = newFileName
		c.FileName = newFileName
		c.DurationSeconds = duration
		c.LUFS = meta.LUFS
		c.LRA = meta.LRA
		c.LoudnessScore = meta.LoudnessScore
		c.CueIn = meta.CueIn
		c.CueOut = meta.CueOut
		c.FileSizeBytes = meta.FileSizeBytes
		c.BitrateKbps = meta.BitRateKbps
		c.SampleRateHz = meta.SampleRateHz
		c.RiscoRuido = meta.NoiseRisk
		c.IsSonic = meta.IsSonic
		c.ManualCue = false
	} else {
		newFilePath := filepath.Join(newBaseDir, c.FileName)
		if oldFilePath != newFilePath {
			if _, err := os.Stat(oldFilePath); err == nil {
				errMove := os.Rename(oldFilePath, newFilePath)
				if errMove != nil {
					errCopy := copyFile(oldFilePath, newFilePath)
					if errCopy == nil {
						_ = os.Remove(oldFilePath)
					} else {
						http.Error(w, "Falha ao mover arquivo de mídia: "+errCopy.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}

	c.TenantID = targetTenantID
	c.CommercialType = models.CommercialType(cTypeStr)
	c.InsertionsPerDay = insertions
	c.StartDate = startDate
	c.EndDate = endDate
	c.StartTime = startTimeStr
	c.EndTime = endTimeStr
	c.RunOnSunday = runSunday
	c.RunOnMonday = runMonday
	c.RunOnTuesday = runTuesday
	c.RunOnWednesday = runWednesday
	c.RunOnThursday = runThursday
	c.RunOnFriday = runFriday
	c.RunOnSaturday = runSaturday
	c.DistributeToAll = distributeToAll
	c.ExecutionTime = executionTime
	c.EventInterruptionMode = eventInterruptionMode

	err = commRepo.Update(ctx, *c)
	if err != nil {
		http.Error(w, "Falha ao atualizar no banco de dados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var parsedStoreIDs []uuid.UUID
	if !distributeToAll {
		storeIDs := r.Form["store_ids"]
		for _, sIDStr := range storeIDs {
			if sID, err := uuid.Parse(sIDStr); err == nil {
				parsedStoreIDs = append(parsedStoreIDs, sID)
			}
		}
	}
	err = commRepo.SaveDistribution(ctx, c.ID, parsedStoreIDs)
	if err != nil {
		http.Error(w, "Falha ao atualizar distribuição das lojas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"name": c.Name,
			"loudness_score": c.LoudnessScore,
			"duration": c.DurationSeconds,
			"lufs": c.LUFS,
			"risco_ruido": c.RiscoRuido,
		})
		return
	}

	http.Redirect(w, r, "/commercials", http.StatusSeeOther)
}

func HandlePlayCommercial(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		http.Error(w, "Commercial not found", http.StatusNotFound)
		return
	}

	folderName := GetCommercialFolderName(c.CommercialType)
	filePath := filepath.Join("midias", "comerciais", c.TenantID.String(), folderName, c.FileName)
	
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Audio file not found on disk", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeFile(w, r, filePath)
}

func HandleGetCommercialDetails(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		http.Error(w, "Comercial não encontrado", http.StatusNotFound)
		return
	}

	storeIDs, err := commRepo.GetDistributedStores(ctx, id)
	if err != nil {
		storeIDs = []uuid.UUID{}
	}

	response := map[string]interface{}{
		"success": true,
		"id": c.ID,
		"tenant_id": c.TenantID,
		"media_type": c.MediaType,
		"commercial_type": c.CommercialType,
		"name": c.Name,
		"file_name": c.FileName,
		"duration_seconds": c.DurationSeconds,
		"insertions_per_day": c.InsertionsPerDay,
		"start_date": c.StartDate,
		"end_date": c.EndDate,
		"start_time": c.StartTime,
		"end_time": c.EndTime,
		"run_on_sunday": c.RunOnSunday,
		"run_on_monday": c.RunOnMonday,
		"run_on_tuesday": c.RunOnTuesday,
		"run_on_wednesday": c.RunOnWednesday,
		"run_on_thursday": c.RunOnThursday,
		"run_on_friday": c.RunOnFriday,
		"run_on_saturday": c.RunOnSaturday,
		"distribute_to_all": c.DistributeToAll,
		"is_active": c.IsActive,
		"execution_time": c.ExecutionTime,
		"event_interruption_mode": c.EventInterruptionMode,
		"store_ids": storeIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HandleValidateCommercialNames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	var req struct {
		TenantIDs []string `json:"tenant_ids"`
		TenantID  string   `json:"tenant_id"`
		Names     []string `json:"names"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var tIDs []uuid.UUID
	if len(req.TenantIDs) > 0 {
		for _, tidStr := range req.TenantIDs {
			if tid, err := uuid.Parse(tidStr); err == nil {
				tIDs = append(tIDs, tid)
			}
		}
	} else if req.TenantID != "" {
		if tid, err := uuid.Parse(req.TenantID); err == nil {
			tIDs = append(tIDs, tid)
		}
	}
	if len(tIDs) == 0 {
		tIDs = append(tIDs, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
	}

	duplicates := []string{}
	for _, name := range req.Names {
		for _, tID := range tIDs {
			exists, err := commRepo.ExistsByName(r.Context(), tID, name)
			if err == nil && exists {
				duplicates = append(duplicates, name)
				break // Already found duplicate for this name
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"duplicates": duplicates,
	})
}

func HandleBatchDeleteCommercials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	var req struct {
		IDs []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	for _, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		c, err := commRepo.GetByID(ctx, id)
		if err == nil && c != nil {
			folderName := GetCommercialFolderName(c.CommercialType)
			filePath := filepath.Join("midias", "comerciais", c.TenantID.String(), folderName, c.FileName)
			_ = os.Remove(filePath)
			_ = commRepo.Delete(ctx, id)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandleBatchReprocessCommercials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	var req struct {
		IDs []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	results := []map[string]interface{}{}

	for _, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		c, err := commRepo.GetByID(ctx, id)
		if err != nil || c == nil {
			continue
		}

		folderName := GetCommercialFolderName(c.CommercialType)
		filePath := filepath.Join("midias", "comerciais", c.TenantID.String(), folderName, c.FileName)

		meta, errMeta := utils.ExtractMetadata(filePath)
		if errMeta != nil {
			results = append(results, map[string]interface{}{
				"id":      idStr,
				"name":    c.Name,
				"success": false,
				"error":   errMeta.Error(),
			})
			continue
		}

		c.LUFS = meta.LUFS
		c.LRA = meta.LRA
		c.LoudnessScore = meta.LoudnessScore
		c.FileSizeBytes = meta.FileSizeBytes
		c.BitrateKbps = meta.BitRateKbps
		c.SampleRateHz = meta.SampleRateHz
		c.RiscoRuido = meta.NoiseRisk

		if !c.ManualCue {
			c.CueIn = meta.CueIn
			c.CueOut = meta.CueOut
		}
		c.DurationSeconds = int(c.CueOut - c.CueIn)
		if c.DurationSeconds < 1 {
			c.DurationSeconds = 1
		}
		c.IsSonic = (c.CueOut - c.CueIn) <= 300.0

		err = commRepo.Update(ctx, *c)
		if err != nil {
			results = append(results, map[string]interface{}{
				"id":      idStr,
				"name":    c.Name,
				"success": false,
				"error":   err.Error(),
			})
			continue
		}

		results = append(results, map[string]interface{}{
			"id":             idStr,
			"name":           c.Name,
			"success":        true,
			"loudness_score": c.LoudnessScore,
			"lufs":           c.LUFS,
			"risco_ruido":    c.RiscoRuido,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "results": results})
}

func HandleSaveCommercialCue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	var req struct {
		ID     string  `json:"id"`
		CueIn  float64 `json:"cue_in"`
		CueOut float64 `json:"cue_out"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cID, err := uuid.Parse(req.ID)
	if err != nil {
		http.Error(w, "Invalid commercial ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, cID)
	if err != nil || c == nil {
		http.Error(w, "Commercial not found", http.StatusNotFound)
		return
	}

	c.CueIn = req.CueIn
	c.CueOut = req.CueOut
	c.ManualCue = true
	c.IsSonic = (c.CueOut - c.CueIn) <= 300.0

	if err := commRepo.Update(ctx, *c); err != nil {
		http.Error(w, "Failed to update cue points: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Cue points salvos manualmente com sucesso."})
}

func HandleResetCommercialCue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cID, err := uuid.Parse(req.ID)
	if err != nil {
		http.Error(w, "Invalid commercial ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	c, err := commRepo.GetByID(ctx, cID)
	if err != nil || c == nil {
		http.Error(w, "Commercial not found", http.StatusNotFound)
		return
	}

	folderName := GetCommercialFolderName(c.CommercialType)
	filePath := filepath.Join("midias", "comerciais", c.TenantID.String(), folderName, c.FileName)

	meta, errMeta := utils.ExtractMetadata(filePath)
	if errMeta != nil {
		http.Error(w, "Failed to extract original metadata: "+errMeta.Error(), http.StatusInternalServerError)
		return
	}

	c.CueIn = meta.CueIn
	c.CueOut = meta.CueOut
	c.ManualCue = false
	c.IsSonic = (c.CueOut - c.CueIn) <= 300.0

	if err := commRepo.Update(ctx, *c); err != nil {
		http.Error(w, "Failed to update cue points: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cue points resetados para o modo automático.",
		"cue_in":  meta.CueIn,
		"cue_out": meta.CueOut,
	})
}
