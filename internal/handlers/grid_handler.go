package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"imc-server/internal/database"
	"imc-server/internal/models"
	"imc-server/internal/repository"

	"github.com/google/uuid"
)

var gridRepo = repository.NewGridRepository()

type DayInfo struct {
	Key  string
	Name string
}

type GridPageDetail struct {
	Grid      models.WeeklyMasterGrid                 `json:"grid"`
	Schedules map[string]*models.BasicSchedule          `json:"schedules"`
	Configs   map[string]*models.CommercialBlockConfig  `json:"configs"`
}

func (g GridPageDetail) MondayHours() string {
	s, ok := g.Schedules["mon"]
	if !ok || s == nil {
		return "---"
	}
	st := s.StartTime
	et := s.EndTime
	if len(st) >= 5 {
		st = st[:5]
	}
	if len(et) >= 5 {
		et = et[:5]
	}
	if st == "00:00" && et == "00:00" {
		return "24h"
	}
	return fmt.Sprintf("%s - %s", st, et)
}

func (g GridPageDetail) MondayCommercialCount() int {
	s, ok := g.Schedules["mon"]
	if !ok || s == nil {
		return 0
	}
	c, ok := g.Configs["mon"]
	if !ok || c == nil {
		return 0
	}
	
	opMin := 0
	startMin := parseTimeToMinutes(s.StartTime)
	endMin := parseTimeToMinutes(s.EndTime)
	if endMin == startMin {
		opMin = 24 * 60
	} else if endMin < startMin {
		opMin = (endMin + 24*60) - startMin
	} else {
		opMin = endMin - startMin
	}
	
	musicDuration := s.MusicalBlockDurationMinutes
	if musicDuration <= 0 {
		musicDuration = 60
	}
	intervalDuration := c.IntervalMinutes
	if intervalDuration <= 0 {
		intervalDuration = 15
	}
	
	count := 0
	currentPos := 0
	for currentPos < opMin {
		currentPos += musicDuration
		if currentPos < opMin {
			currentPos += intervalDuration
			count++
		}
	}
	return count
}

func (g GridPageDetail) MondayCommercialDuration() string {
	c, ok := g.Configs["mon"]
	if !ok || c == nil {
		return "---"
	}
	return fmt.Sprintf("%d min", c.IntervalMinutes)
}

func parseTimeToMinutes(t string) int {
	t = strings.TrimSpace(t)
	if len(t) < 5 {
		return 0
	}
	parts := strings.Split(t, ":")
	if len(parts) < 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

type StyleInfo struct {
	ID         string
	Name       string
	SourceType string
	StreamURL  string
}

type GridsPageData struct {
	Title          string
	LoggedUser     string
	ErrorMessage   string
	SuccessMessage string
	Form           map[string]string
	Tenants        []models.Tenant
	Grids          []GridPageDetail
	GridsJSON      template.JS
	User           map[string]string
	Theme          map[string]string
	Days           []DayInfo
	Styles         []StyleInfo
}

// HandleAudioGridsPage renderiza a tela de Grades de Áudio
func HandleAudioGridsPage(w http.ResponseWriter, r *http.Request) {
	renderGridsPage(w, r, models.MediaAudio, "Grades de Áudio", "grids_audio.html", "")
}

// HandleVideoGridsPage renderiza a tela de Grades de Vídeo
func HandleVideoGridsPage(w http.ResponseWriter, r *http.Request) {
	renderGridsPage(w, r, models.MediaVideo, "Grades de Vídeo", "grids_video.html", "")
}

// HandleCreateAudioGrid cria uma grade de áudio
func HandleCreateAudioGrid(w http.ResponseWriter, r *http.Request) {
	handleCreateGridWithType(w, r, models.MediaAudio, "/grids/audio", "grids_audio.html", "Grades de Áudio")
}

// HandleCreateVideoGrid cria uma grade de vídeo
func HandleCreateVideoGrid(w http.ResponseWriter, r *http.Request) {
	handleCreateGridWithType(w, r, models.MediaVideo, "/grids/video", "grids_video.html", "Grades de Vídeo")
}

// HandleUpdateAudioGrid atualiza uma grade de áudio
func HandleUpdateAudioGrid(w http.ResponseWriter, r *http.Request) {
	handleUpdateGridWithType(w, r, models.MediaAudio, "/grids/audio", "grids_audio.html", "Grades de Áudio")
}

// HandleUpdateVideoGrid atualiza uma grade de vídeo
func HandleUpdateVideoGrid(w http.ResponseWriter, r *http.Request) {
	handleUpdateGridWithType(w, r, models.MediaVideo, "/grids/video", "grids_video.html", "Grades de Vídeo")
}

// HandleDeleteAudioGrid deleta uma grade de áudio
func HandleDeleteAudioGrid(w http.ResponseWriter, r *http.Request) {
	handleDeleteGridWithType(w, r, "/grids/audio", "grids_audio.html", "Grades de Áudio")
}

// HandleDeleteVideoGrid deleta uma grade de vídeo
func HandleDeleteVideoGrid(w http.ResponseWriter, r *http.Request) {
	handleDeleteGridWithType(w, r, "/grids/video", "grids_video.html", "Grades de Vídeo")
}

// Auxiliar para renderizar a página filtrada por tipo de recurso
func renderGridsPage(w http.ResponseWriter, r *http.Request, mediaType models.MediaType, title string, templateName string, errMsg string) {
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

	rawGrids, err := gridRepo.GetAll(context.Background(), nil)
	if err != nil {
		rawGrids = []models.WeeklyMasterGrid{}
	}

	var gridsList []GridPageDetail
	for _, rg := range rawGrids {
		// Filtra pelo tipo correspondente ao "mundo"
		if rg.MediaType != mediaType {
			continue
		}
		g, schedules, configs, err := gridRepo.GetByID(context.Background(), rg.ID)
		if err == nil {
			gridsList = append(gridsList, GridPageDetail{
				Grid:      *g,
				Schedules: schedules,
				Configs:   configs,
			})
		}
	}

	gridsJSONBytes, _ := json.Marshal(gridsList)
	gridsJSON := template.JS(string(gridsJSONBytes))
	if len(gridsJSON) == 0 {
		gridsJSON = "[]"
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	formPreserve := make(map[string]string)
	r.ParseForm()
	for k, v := range r.Form {
		if len(v) > 0 {
			formPreserve[k] = v[0]
		}
	}

	days := []DayInfo{
		{Key: "mon", Name: "Segunda-feira"},
		{Key: "tue", Name: "Terça-feira"},
		{Key: "wed", Name: "Quarta-feira"},
		{Key: "thu", Name: "Quinta-feira"},
		{Key: "fri", Name: "Sexta-feira"},
		{Key: "sat", Name: "Sábado"},
		{Key: "sun", Name: "Domingo"},
		{Key: "holiday", Name: "Feriado"},
	}

	// Query styles from database
	var stylesList []StyleInfo
	var query string
	if mediaType == models.MediaAudio {
		query = "SELECT id, name, COALESCE(source_type, 'LOCAL'), COALESCE(stream_url, '') FROM music_styles ORDER BY name ASC"
	} else {
		query = "SELECT id, name, 'LOCAL', '' FROM video_styles ORDER BY name ASC"
	}
	rowsStyles, errStyles := database.Pool.Query(context.Background(), query)
	if errStyles == nil {
		defer rowsStyles.Close()
		for rowsStyles.Next() {
			var id, name, srcType, streamURL string
			if errScan := rowsStyles.Scan(&id, &name, &srcType, &streamURL); errScan == nil {
				stylesList = append(stylesList, StyleInfo{
					ID:         id,
					Name:       name,
					SourceType: srcType,
					StreamURL:  streamURL,
				})
			}
		}
	}
	if len(stylesList) == 0 {
		fallbackNames := []string{"Pop", "Rock", "Sertanejo", "MPB", "Eletrônica", "Jazz", "Bossa Nova", "Gospel", "Infantil", "Clássica", "Samba / Pagode", "Metal"}
		for i, name := range fallbackNames {
			stylesList = append(stylesList, StyleInfo{
				ID:   fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
				Name: name,
			})
		}
	}

	data := GridsPageData{
		Title:        title,
		LoggedUser:   session.UserName,
		ErrorMessage: errMsg,
		Tenants:      tenantsList,
		Grids:        gridsList,
		GridsJSON:    gridsJSON,
		Form:         formPreserve,
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:  theme,
		Days:   days,
		Styles: stylesList,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/" + templateName))
	tmpl.Execute(w, data)
}

func handleCreateGridWithType(w http.ResponseWriter, r *http.Request, mediaType models.MediaType, redirectURL string, templateName string, title string) {
	grid, schedules, configs, err := parseGridForm(r, mediaType)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, err.Error())
		return
	}

	err = gridRepo.Create(context.Background(), grid, schedules, configs)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, "Erro ao criar a grade: "+err.Error())
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func handleUpdateGridWithType(w http.ResponseWriter, r *http.Request, mediaType models.MediaType, redirectURL string, templateName string, title string) {
	grid, schedules, configs, err := parseGridForm(r, mediaType)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, err.Error())
		return
	}

	if grid.ID == uuid.Nil {
		renderGridsPage(w, r, mediaType, title, templateName, "ID da grade inválido para atualização")
		return
	}

	err = gridRepo.Update(context.Background(), grid, schedules, configs)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, "Erro ao atualizar a grade: "+err.Error())
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func handleDeleteGridWithType(w http.ResponseWriter, r *http.Request, redirectURL string, templateName string, title string) {
	// Pega o mediaType inferido da URL para poder renderizar em caso de erro
	mediaType := models.MediaAudio
	if strings.Contains(redirectURL, "video") {
		mediaType = models.MediaVideo
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

	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, "ID inválido para exclusão")
		return
	}

	err = gridRepo.Delete(context.Background(), id)
	if err != nil {
		renderGridsPage(w, r, mediaType, title, templateName, "Erro ao excluir a grade: "+err.Error())
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func parseGridForm(r *http.Request, mediaType models.MediaType) (*models.WeeklyMasterGrid, map[string]*models.BasicSchedule, map[string]*models.CommercialBlockConfig, error) {
	r.ParseForm()

	tenantIDStr := r.FormValue("tenant_id")
	if tenantIDStr == "" {
		tenantIDStr = "00000000-0000-0000-0000-000000000000"
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("matriz inválida: %w", err)
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return nil, nil, nil, errors.New("o nome da grade é obrigatório")
	}

	description := r.FormValue("description")

	gridIDStr := r.FormValue("id")
	var gridID uuid.UUID
	if gridIDStr != "" {
		gridID, _ = uuid.Parse(gridIDStr)
	}

	allowedStylesJSON := strings.TrimSpace(r.FormValue("allowed_manual_styles_json"))
	var allowedStylesPtr *string
	if allowedStylesJSON != "" && allowedStylesJSON != "[]" {
		allowedStylesPtr = &allowedStylesJSON
	}

	grid := &models.WeeklyMasterGrid{
		ID:                      gridID,
		TenantID:                tenantID,
		MediaType:               mediaType,
		Name:                    name,
		Description:             description,
		AllowedManualStylesJSON: allowedStylesPtr,
	}

	schedules := make(map[string]*models.BasicSchedule)
	configs := make(map[string]*models.CommercialBlockConfig)

	// Auxiliar para processar os parâmetros individuais de cada um dos 8 dias
	parseDayGroup := func(prefix string, displayName string) (*models.BasicSchedule, *models.CommercialBlockConfig) {
		// Segunda-feira ('mon') é sempre habilitada, outros dias dependem de prefix_enabled
		if prefix != "mon" {
			enabled := r.FormValue(prefix + "_enabled")
			if enabled != "true" && enabled != "on" {
				return nil, nil
			}
		}

		startTime := cleanTime(r.FormValue(prefix + "_start_time"))
		endTime := cleanTime(r.FormValue(prefix + "_end_time"))
		musicalDuration := parseIntOrDefault(r.FormValue(prefix+"_musical_block_duration_minutes"), 60)
		intervalMin := parseIntOrDefault(r.FormValue(prefix+"_interval_minutes"), 15)
		// block_limit_seconds is not a separate UI field — derive it from intervalMin.
		// If the form does submit an explicit value, use it; otherwise convert minutes → seconds.
		explicitLimit := r.FormValue(prefix + "_block_limit_seconds")
		limitSec := intervalMin * 60
		if explicitLimit != "" {
			limitSec = parseIntOrDefault(explicitLimit, intervalMin*60)
		}

		pctAnn := parseIntOrDefault(r.FormValue(prefix+"_percentage_time_announcement"), 0)
		pctTemp := parseIntOrDefault(r.FormValue(prefix+"_percentage_temperature"), 0)
		pctInst := parseIntOrDefault(r.FormValue(prefix+"_percentage_institutional"), 50)
		pctOffer := parseIntOrDefault(r.FormValue(prefix+"_percentage_offer"), 50)

		customDurationsJSON := strings.TrimSpace(r.FormValue(prefix + "_custom_durations_json"))
		customStylesJSON := strings.TrimSpace(r.FormValue(prefix + "_custom_styles_json"))

		var durPtr *string
		if customDurationsJSON != "" && customDurationsJSON != "{}" {
			durCopy := customDurationsJSON
			durPtr = &durCopy
		}
		var stylePtr *string
		if customStylesJSON != "" && customStylesJSON != "{}" {
			styleCopy := customStylesJSON
			stylePtr = &styleCopy
		}

		offHoursBehavior := strings.TrimSpace(r.FormValue(prefix + "_off_hours_behavior"))
		if offHoursBehavior == "" {
			offHoursBehavior = "SILENCE"
		}

		sched := &models.BasicSchedule{
			TenantID:                    tenantID,
			MediaType:                   mediaType,
			Name:                        fmt.Sprintf("%s - %s", name, displayName),
			Description:                 fmt.Sprintf("Expediente %s da grade %s", displayName, name),
			StartTime:                   startTime,
			EndTime:                     endTime,
			MusicalBlockDurationMinutes: musicalDuration,
			CustomDurationsJSON:        durPtr,
			CustomStylesJSON:           stylePtr,
			OffHoursBehavior:           offHoursBehavior,
		}

		conf := &models.CommercialBlockConfig{
			TenantID:                   tenantID,
			MediaType:                  mediaType,
			Name:                       fmt.Sprintf("%s - %s", name, displayName),
			Description:                fmt.Sprintf("Intervalos %s da grade %s", displayName, name),
			IntervalMinutes:            intervalMin,
			BlockLimitSeconds:          limitSec,
			PercentageTimeAnnouncement: pctAnn,
			PercentageTemperature:      pctTemp,
			PercentageInstitutional:    pctInst,
			PercentageOffer:            pctOffer,
		}

		return sched, conf
	}

	daysMapping := map[string]string{
		"mon":     "Segunda-feira",
		"tue":     "Terça-feira",
		"wed":     "Quarta-feira",
		"thu":     "Quinta-feira",
		"fri":     "Sexta-feira",
		"sat":     "Sábado",
		"sun":     "Domingo",
		"holiday": "Feriado",
	}

	for key, displayName := range daysMapping {
		s, c := parseDayGroup(key, displayName)
		schedules[key] = s
		configs[key] = c
	}

	return grid, schedules, configs, nil
}

func cleanTime(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "00:00:00"
	}
	if len(t) == 5 {
		return t + ":00"
	}
	return t
}

func parseIntOrDefault(s string, def int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return val
}
