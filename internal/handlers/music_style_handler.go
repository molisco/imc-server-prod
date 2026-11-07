package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"imc-server/internal/models"
	"imc-server/internal/repository"
	"imc-server/internal/utils"

	"github.com/google/uuid"
)

var musicStyleRepo = repository.NewMusicStyleRepository()
var musicTrackRepo = repository.NewMusicTrackRepository()

var stylesFuncMap = template.FuncMap{
	"formatDuration": func(secs int) string {
		m := secs / 60
		s := secs % 60
		return fmt.Sprintf("%02d:%02d", m, s)
	},
	"formatSize": func(b int64) string {
		if b <= 0 {
			return "0 B"
		}
		const unit = 1024
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
	},
	"formatSampleRate": func(hz int) string {
		if hz <= 0 {
			return "--"
		}
		if hz%1000 == 0 {
			return fmt.Sprintf("%d kHz", hz/1000)
		}
		return fmt.Sprintf("%.1f kHz", float64(hz)/1000.0)
	},
	"formatTotalDuration": func(secs int) string {
		h := secs / 3600
		m := (secs % 3600) / 60
		s := secs % 60
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	},
	"formatDateTime": func(t time.Time) string {
		if t.IsZero() {
			return "--"
		}
		return t.Format("02/01/2006 15:04:05.000")
	},
	"isOverFiveMinutes": func(cueIn, cueOut float64) bool {
		return (cueOut - cueIn) > 300.0
	},
}

type MusicStyleDetail struct {
	models.MusicStyle
	TracksCount          int
	Tracks               []models.MusicTrack
	SyncStatus           string
	TotalDurationSeconds int
}

type MusicStylePageData struct {
	Title                string
	LoggedUser           string
	ErrorMessage         string
	SuccessMessage       string
	Form                 map[string]string
	Styles               []MusicStyleDetail
	User                 map[string]string
	Theme                map[string]string
	AvailableDirectories []string
}

type ScanProgress struct {
	TotalFiles     int    `json:"total_files"`
	ProcessedFiles int    `json:"processed_files"`
	CurrentFile    string `json:"current_file"`
	Status         string `json:"status"` // "idle", "scanning", "completed", "failed"
	Error          string `json:"error"`
}

var (
	scanMutex          sync.RWMutex
	scanRegistry       = make(map[uuid.UUID]*ScanProgress)
	scanCancelRegistry = make(map[uuid.UUID]context.CancelFunc)
)

func startDirectorySync(styleID uuid.UUID, baseDir string, reprocess bool) {
	ctx, cancel := context.WithCancel(context.Background())
	scanMutex.Lock()
	if oldCancel, exists := scanCancelRegistry[styleID]; exists && oldCancel != nil {
		oldCancel()
	}
	scanCancelRegistry[styleID] = cancel
	
	// Synchronously initialize scan status to scanning to prevent race conditions during page redirect!
	p, exists := scanRegistry[styleID]
	if !exists {
		p = &ScanProgress{}
		scanRegistry[styleID] = p
	}
	p.Status = "scanning"
	p.TotalFiles = 0
	p.ProcessedFiles = 0
	p.CurrentFile = "Acessando diretório..."
	p.Error = ""
	scanMutex.Unlock()

	go syncMusicStyleDirectory(ctx, styleID, baseDir, reprocess)
}

func getOrCreateProgress(styleID uuid.UUID) *ScanProgress {
	scanMutex.Lock()
	defer scanMutex.Unlock()
	p, exists := scanRegistry[styleID]
	if !exists {
		p = &ScanProgress{Status: "idle"}
		scanRegistry[styleID] = p
	}
	return p
}

func updateProgress(styleID uuid.UUID, fn func(*ScanProgress)) {
	scanMutex.Lock()
	defer scanMutex.Unlock()
	p, exists := scanRegistry[styleID]
	if !exists {
		p = &ScanProgress{Status: "idle"}
		scanRegistry[styleID] = p
	}
	fn(p)
}

func getAvailableSubdirectories() []string {
	baseDir := filepath.Join("midias", "musicas")
	_ = os.MkdirAll(baseDir, 0755)
	
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}
	
	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		}
	}
	return subdirs
}

func syncMusicStyleDirectory(ctx context.Context, styleID uuid.UUID, baseDir string, reprocess bool) {
	if baseDir == "" {
		return
	}

	defer func() {
		scanMutex.Lock()
		delete(scanCancelRegistry, styleID)
		scanMutex.Unlock()
	}()

	// Initializing progress
	updateProgress(styleID, func(p *ScanProgress) {
		p.Status = "scanning"
		p.TotalFiles = 0
		p.ProcessedFiles = 0
		p.CurrentFile = "Acessando diretório..."
		p.Error = ""
	})

	// Check if directory exists
	info, err := os.Stat(baseDir)
	if err != nil || !info.IsDir() {
		log.Printf("[Sync] Directory %s does not exist or is not a folder. Cleaning up database records for style ID %s.", baseDir, styleID)
		
		// Delete all registered tracks for this style since the directory is missing
		_ = musicTrackRepo.DeleteByStyleID(ctx, styleID)
		
		// Update parent style timestamp
		_ = musicStyleRepo.UpdateTimestamp(ctx, styleID)
		
		updateProgress(styleID, func(p *ScanProgress) {
			p.Status = "completed"
			p.TotalFiles = 0
			p.ProcessedFiles = 0
			p.CurrentFile = ""
			p.Error = "Diretório não existe ou foi removido. Todos os registros desse estilo foram limpos do banco de dados."
		})
		return
	}

	// Read directory entries
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		updateProgress(styleID, func(p *ScanProgress) {
			p.Status = "failed"
			p.Error = "Falha ao ler diretório: " + err.Error()
		})
		return
	}

	var mp3Files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		nameLower := strings.ToLower(name)
		if strings.HasSuffix(nameLower, ".mp3") && !strings.Contains(nameLower, ".master.") && !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "._") {
			mp3Files = append(mp3Files, name)
		}
	}

	total := len(mp3Files)
	updateProgress(styleID, func(p *ScanProgress) {
		p.TotalFiles = total
	})

	activeFiles := make(map[string]bool)
	for idx, name := range mp3Files {
		select {
		case <-ctx.Done():
			updateProgress(styleID, func(p *ScanProgress) {
				p.Status = "failed"
				p.Error = "Sincronização interrompida pelo usuário"
			})
			return
		default:
		}

		activeFiles[name] = true

		updateProgress(styleID, func(p *ScanProgress) {
			p.CurrentFile = name
			p.ProcessedFiles = idx
		})

		// Check if already registered
		existing, err := musicTrackRepo.GetByStyleAndFileName(ctx, styleID, name)
		
		shouldProcess := false
		if err != nil || existing == nil {
			shouldProcess = true
		} else if reprocess {
			shouldProcess = true
		}

		if shouldProcess {
			fullPath := filepath.Join(baseDir, name)
			
			// Extract tags and calculate LUFS / LRA
			meta, errMeta := utils.ExtractMetadata(fullPath)
			if errMeta != nil {
				// Fallback defaults
				meta.Duration = 180
				meta.LUFS = -14.0
				meta.LRA = 5.0
				meta.LoudnessScore = "A"
				meta.CueIn = 0.0
				meta.CueOut = 180.0
			}

			displayName := meta.Title
			if displayName == "" {
				displayName = strings.TrimSuffix(name, filepath.Ext(name))
			}

			// If it has a manual cue, preserve it!
			cueIn := meta.CueIn
			cueOut := meta.CueOut
			isManual := false
			if existing != nil && existing.ManualCue {
				cueIn = existing.CueIn
				cueOut = existing.CueOut
				isManual = true
			}

			track := models.MusicTrack{
				MusicStyleID:    styleID,
				FileName:        name,
				DisplayName:     displayName,
				DurationSeconds: meta.Duration,
				Album:           meta.Album,
				Title:           meta.Title,
				Artist:          meta.Artist,
				Year:            meta.Year,
				LUFS:            meta.LUFS,
				LRA:             meta.LRA,
				LoudnessScore:   meta.LoudnessScore,
				CueIn:           cueIn,
				CueOut:          cueOut,
				FileSizeBytes:   meta.FileSizeBytes,
				BitRateKbps:     meta.BitRateKbps,
				SampleRateHz:    meta.SampleRateHz,
				RiscoRuido:      meta.NoiseRisk,
				ManualCue:       isManual,
				IsSonic:         (cueOut - cueIn) <= 300.0,
			}

			if existing != nil {
				track.ID = existing.ID
				track.CreatedAt = existing.CreatedAt
				_ = musicTrackRepo.Update(ctx, track)
			} else {
				track.ID = uuid.New()
				_ = musicTrackRepo.Create(ctx, track)
			}
		}
	}

	// Remove deleted tracks
	updateProgress(styleID, func(p *ScanProgress) {
		p.CurrentFile = "Limpando arquivos removidos..."
	})

	registered, err := musicTrackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		log.Printf("[Sync] Error retrieving registered tracks for style ID %s cleanup: %v", styleID, err)
	} else {
		for _, t := range registered {
			if !activeFiles[t.FileName] {
				log.Printf("[Sync] File %s was removed from directory. Deleting track ID %s from database.", t.FileName, t.ID)
				if errDel := musicTrackRepo.DeleteByID(ctx, t.ID); errDel != nil {
					log.Printf("[Sync] Error deleting track %s (ID %s) from database: %v", t.FileName, t.ID, errDel)
				}
			}
		}
	}

	_ = musicStyleRepo.UpdateTimestamp(ctx, styleID)

	updateProgress(styleID, func(p *ScanProgress) {
		p.Status = "completed"
		p.ProcessedFiles = total
		p.CurrentFile = ""
	})
}

func HandleMusicStylesPage(w http.ResponseWriter, r *http.Request) {
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

	stylesList, err := musicStyleRepo.GetAll(context.Background())
	if err != nil {
		stylesList = []models.MusicStyle{}
	}

	var detailedStyles []MusicStyleDetail
	for _, s := range stylesList {
		tracks, _ := musicTrackRepo.GetByStyleID(context.Background(), s.ID)
		if tracks == nil {
			tracks = []models.MusicTrack{}
		}
		var totalSeconds int
		for _, t := range tracks {
			totalSeconds += t.DurationSeconds
		}
		progress := getOrCreateProgress(s.ID)
		detailedStyles = append(detailedStyles, MusicStyleDetail{
			MusicStyle:           s,
			TracksCount:          len(tracks),
			Tracks:               tracks,
			SyncStatus:           progress.Status,
			TotalDurationSeconds: totalSeconds,
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

	data := MusicStylePageData{
		Title:      "Gerenciamento de Estilos Musicais",
		LoggedUser: session.UserName,
		Styles:     detailedStyles,
		Form:       make(map[string]string),
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:                theme,
		AvailableDirectories: getAvailableSubdirectories(),
	}

	tmpl := template.New("music_styles.html").Funcs(stylesFuncMap)
	tmpl = template.Must(tmpl.ParseFiles("web/templates/music_styles.html", "web/templates/tracks_table_partial.html"))
	tmpl.Execute(w, data)
}

func HandleCreateMusicStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
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
	name := r.FormValue("name")
	description := r.FormValue("description")
	baseDirectory := r.FormValue("base_directory")
	sourceType := strings.ToUpper(strings.TrimSpace(r.FormValue("source_type")))
	if sourceType != "STREAM" {
		sourceType = "LOCAL"
	}
	streamURL := strings.TrimSpace(r.FormValue("stream_url"))

	formData := map[string]string{
		"name":           name,
		"description":    description,
		"base_directory": baseDirectory,
		"source_type":    sourceType,
		"stream_url":     streamURL,
	}

	nameClean := strings.TrimSpace(name)

	if nameClean == "" {
		renderMusicStylesError(w, session, "❌ ATENÇÃO: O campo Nome do Estilo é obrigatório!", formData)
		return
	}

	if sourceType == "STREAM" && streamURL == "" {
		renderMusicStylesError(w, session, "❌ ATENÇÃO: Para estilos de Streaming, a URL do stream é obrigatória!", formData)
		return
	}

	style := models.MusicStyle{
		ID:            uuid.New(),
		Name:          nameClean,
		Description:   strings.TrimSpace(description),
		BaseDirectory: strings.TrimSpace(baseDirectory),
		SourceType:    sourceType,
		StreamURL:     streamURL,
	}

	err = musicStyleRepo.Create(context.Background(), style)
	if err != nil {
		renderMusicStylesError(w, session, "❌ Erro ao salvar o Estilo Musical: "+err.Error(), formData)
		return
	}

	if style.SourceType == "LOCAL" && style.BaseDirectory != "" {
		startDirectorySync(style.ID, style.BaseDirectory, false)
	}

	http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
}

func HandleUpdateMusicStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
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
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	baseDirectory := r.FormValue("base_directory")
	sourceType := strings.ToUpper(strings.TrimSpace(r.FormValue("source_type")))
	if sourceType != "STREAM" {
		sourceType = "LOCAL"
	}
	streamURL := strings.TrimSpace(r.FormValue("stream_url"))

	formData := map[string]string{
		"id":             idStr,
		"name":           name,
		"description":    description,
		"base_directory": baseDirectory,
		"source_type":    sourceType,
		"stream_url":     streamURL,
	}

	nameClean := strings.TrimSpace(name)

	if nameClean == "" {
		renderMusicStylesError(w, session, "❌ ATENÇÃO: O campo Nome do Estilo é obrigatório!", formData)
		return
	}

	if sourceType == "STREAM" && streamURL == "" {
		renderMusicStylesError(w, session, "❌ ATENÇÃO: Para estilos de Streaming, a URL do stream é obrigatória!", formData)
		return
	}

	style := models.MusicStyle{
		ID:            styleID,
		Name:          nameClean,
		Description:   strings.TrimSpace(description),
		BaseDirectory: strings.TrimSpace(baseDirectory),
		SourceType:    sourceType,
		StreamURL:     streamURL,
	}

	err = musicStyleRepo.Update(context.Background(), style)
	if err != nil {
		renderMusicStylesError(w, session, "❌ Erro ao atualizar o Estilo Musical: "+err.Error(), formData)
		return
	}

	if style.SourceType == "LOCAL" && style.BaseDirectory != "" {
		startDirectorySync(style.ID, style.BaseDirectory, false)
	}

	http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
}

func HandleDeleteMusicStyle(w http.ResponseWriter, r *http.Request) {
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
	styleID, err := uuid.Parse(idStr)
	if err == nil {
		_ = musicStyleRepo.Delete(context.Background(), styleID)
	}

	http.Redirect(w, r, "/music-styles", http.StatusSeeOther)
}

func renderMusicStylesError(w http.ResponseWriter, session models.SessionData, errMsg string, formData map[string]string) {
	stylesList, _ := musicStyleRepo.GetAll(context.Background())

	var detailedStyles []MusicStyleDetail
	for _, s := range stylesList {
		tracks, _ := musicTrackRepo.GetByStyleID(context.Background(), s.ID)
		if tracks == nil {
			tracks = []models.MusicTrack{}
		}
		var totalSeconds int
		for _, t := range tracks {
			totalSeconds += t.DurationSeconds
		}
		progress := getOrCreateProgress(s.ID)
		detailedStyles = append(detailedStyles, MusicStyleDetail{
			MusicStyle:           s,
			TracksCount:          len(tracks),
			Tracks:               tracks,
			SyncStatus:           progress.Status,
			TotalDurationSeconds: totalSeconds,
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

	data := MusicStylePageData{
		Title:        "Gerenciamento de Estilos Musicais",
		LoggedUser:   session.UserName,
		ErrorMessage: errMsg,
		Form:         formData,
		Styles:       detailedStyles,
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:                theme,
		AvailableDirectories: getAvailableSubdirectories(),
	}

	w.WriteHeader(http.StatusBadRequest)
	tmpl := template.New("music_styles.html").Funcs(stylesFuncMap)
	tmpl = template.Must(tmpl.ParseFiles("web/templates/music_styles.html", "web/templates/tracks_table_partial.html"))
	tmpl.Execute(w, data)
}

func HandleSyncMusicStyle(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.URL.Query().Get("id")
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid style ID", http.StatusBadRequest)
		return
	}

	style, err := musicStyleRepo.GetByID(context.Background(), styleID)
	if err != nil {
		http.Error(w, "Style not found", http.StatusNotFound)
		return
	}

	reprocessStr := r.URL.Query().Get("reprocess")
	reprocess := reprocessStr == "true"

	progress := getOrCreateProgress(styleID)
	scanMutex.RLock()
	isScanning := progress.Status == "scanning"
	scanMutex.RUnlock()

	if isScanning {
		http.Error(w, "Sincronização já está em andamento para este estilo", http.StatusConflict)
		return
	}

	// Trigger async scan
	startDirectorySync(style.ID, style.BaseDirectory, reprocess)

	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func HandleMusicStyleSyncStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid style ID", http.StatusBadRequest)
		return
	}

	progress := getOrCreateProgress(styleID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

func HandleStopSyncMusicStyle(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.URL.Query().Get("id")
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid style ID", http.StatusBadRequest)
		return
	}

	scanMutex.Lock()
	cancel, exists := scanCancelRegistry[styleID]
	if exists && cancel != nil {
		cancel()
		delete(scanCancelRegistry, styleID)
	}
	scanMutex.Unlock()

	updateProgress(styleID, func(p *ScanProgress) {
		p.Status = "failed"
		p.Error = "Sincronização interrompida pelo usuário"
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func HandlePlayMusicTrack(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	idStr := r.URL.Query().Get("id")
	trackID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	track, err := musicTrackRepo.GetByID(ctx, trackID)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	style, err := musicStyleRepo.GetByID(ctx, track.MusicStyleID)
	if err != nil {
		http.Error(w, "Music style not found", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(style.BaseDirectory, track.FileName)
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

func HandleDeleteMusicTracks(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var lastStyleID uuid.UUID
	for _, idStr := range req.IDs {
		trackID, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		if t, err := musicTrackRepo.GetByID(ctx, trackID); err == nil && t != nil {
			lastStyleID = t.MusicStyleID
		}
		_ = musicTrackRepo.DeleteByID(ctx, trackID)
	}
	if lastStyleID != uuid.Nil {
		_ = musicStyleRepo.UpdateTimestamp(ctx, lastStyleID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted_count": len(req.IDs)})
}

func HandleMusicStyleTracksHTML(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	_ = cookie

	idStr := r.URL.Query().Get("id")
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	style, err := musicStyleRepo.GetByID(r.Context(), styleID)
	if err != nil {
		http.Error(w, "Music Style not found", http.StatusNotFound)
		return
	}

	tracks, err := musicTrackRepo.GetByStyleID(r.Context(), styleID)
	if err != nil {
		tracks = []models.MusicTrack{}
	}

	var totalSeconds int
	for _, t := range tracks {
		totalSeconds += t.DurationSeconds
	}

	data := MusicStyleDetail{
		MusicStyle:           *style,
		TracksCount:          len(tracks),
		Tracks:               tracks,
		TotalDurationSeconds: totalSeconds,
	}

	tmpl := template.New("tracks_table_partial.html").Funcs(stylesFuncMap)
	tmpl = template.Must(tmpl.ParseFiles("web/templates/tracks_table_partial.html"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func HandleReprocessMusicTracks(w http.ResponseWriter, r *http.Request) {
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
		TrackIDs []string `json:"track_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Request JSON inválido: " + err.Error(),
		})
		return
	}

	if len(req.TrackIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Nenhum ID de música fornecido",
		})
		return
	}

	ctx := r.Context()
	var reprocessedCount int
	var errors []string

	for _, idStr := range req.TrackIDs {
		select {
		case <-ctx.Done():
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Processamento interrompido pelo usuário",
			})
			return
		default:
		}

		trackID, err := uuid.Parse(idStr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("ID inválido: %s", idStr))
			continue
		}

		track, err := musicTrackRepo.GetByID(ctx, trackID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Música não encontrada para o ID: %s", idStr))
			continue
		}

		style, err := musicStyleRepo.GetByID(ctx, track.MusicStyleID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Estilo não encontrado para a música: %s", track.FileName))
			continue
		}

		fullPath := filepath.Join(style.BaseDirectory, track.FileName)
		meta, err := utils.ExtractMetadata(fullPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Falha ao analisar mídias de %s: %v", track.FileName, err))
			continue
		}

		// Update track properties
		track.DurationSeconds = meta.Duration
		track.Album = meta.Album
		track.Title = meta.Title
		track.Artist = meta.Artist
		track.Year = meta.Year
		track.LUFS = meta.LUFS
		track.LRA = meta.LRA
		track.LoudnessScore = meta.LoudnessScore
		if !track.ManualCue {
			track.CueIn = meta.CueIn
			track.CueOut = meta.CueOut
		}
		track.FileSizeBytes = meta.FileSizeBytes
		track.BitRateKbps = meta.BitRateKbps
		track.SampleRateHz = meta.SampleRateHz

		track.RiscoRuido = meta.NoiseRisk
		track.IsSonic = (track.CueOut - track.CueIn) <= 300.0

		// If Tag has no title/artist, use clean filename format as fallback
		if track.Title == "" {
			nameWithoutExt := track.FileName
			if idx := strings.LastIndex(track.FileName, "."); idx != -1 {
				nameWithoutExt = track.FileName[:idx]
			}
			track.Title = nameWithoutExt
		}
		if track.Artist == "" {
			track.Artist = "Desconhecido"
		}

		err = musicTrackRepo.Update(ctx, *track)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Erro ao salvar dados de %s no BD: %v", track.FileName, err))
			continue
		}

		reprocessedCount++
	}

	if reprocessedCount > 0 {
		if len(req.TrackIDs) > 0 {
			if firstID, err := uuid.Parse(req.TrackIDs[0]); err == nil {
				if t, err := musicTrackRepo.GetByID(ctx, firstID); err == nil && t != nil {
					_ = musicStyleRepo.UpdateTimestamp(ctx, t.MusicStyleID)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 && reprocessedCount == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   strings.Join(errors, "; "),
		})
		return
	}

	msg := fmt.Sprintf("%d músicas reprocessadas com sucesso.", reprocessedCount)
	if len(errors) > 0 {
		msg += fmt.Sprintf(" Erros: %s", strings.Join(errors, "; "))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": msg,
	})
}

type SaveCueRequest struct {
	TrackID string  `json:"track_id"`
	CueIn   float64 `json:"cue_in"`
	CueOut  float64 `json:"cue_out"`
}

func HandleSaveMusicTrackCue(w http.ResponseWriter, r *http.Request) {
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

	var req SaveCueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	trackID, err := uuid.Parse(req.TrackID)
	if err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	track, err := musicTrackRepo.GetByID(ctx, trackID)
	if err != nil || track == nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	track.CueIn = req.CueIn
	track.CueOut = req.CueOut
	track.ManualCue = true
	track.IsSonic = (track.CueOut - track.CueIn) <= 300.0

	if err := musicTrackRepo.Update(ctx, *track); err != nil {
		http.Error(w, "Failed to update track cue points: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = musicStyleRepo.UpdateTimestamp(ctx, track.MusicStyleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Cue points salvos manualmente com sucesso."})
}

type ResetCueRequest struct {
	TrackID string `json:"track_id"`
}

func HandleResetMusicTrackCue(w http.ResponseWriter, r *http.Request) {
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

	var req ResetCueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	trackID, err := uuid.Parse(req.TrackID)
	if err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	track, err := musicTrackRepo.GetByID(ctx, trackID)
	if err != nil || track == nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	style, err := musicStyleRepo.GetByID(ctx, track.MusicStyleID)
	if err != nil || style == nil {
		http.Error(w, "Parent music style not found", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(style.BaseDirectory, track.FileName)
	meta, errMeta := utils.ExtractMetadata(fullPath)
	if errMeta != nil {
		http.Error(w, "Failed to extract original metadata: "+errMeta.Error(), http.StatusInternalServerError)
		return
	}

	track.CueIn = meta.CueIn
	track.CueOut = meta.CueOut
	track.ManualCue = false
	track.IsSonic = (track.CueOut - track.CueIn) <= 300.0

	if err := musicTrackRepo.Update(ctx, *track); err != nil {
		http.Error(w, "Failed to update track cue points: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = musicStyleRepo.UpdateTimestamp(ctx, track.MusicStyleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Cue points resetados para o modo automático."})
}

