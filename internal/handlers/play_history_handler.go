package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"
)

// PlayHistoryViewRecord representa uma linha detalhada para renderização na tabela de relatório
type PlayHistoryViewRecord struct {
	ID                string
	StoreID           string
	StoreName         string
	StoreNumber       string
	TenantID          string
	TenantName        string
	TrackID           string
	TrackType         string
	DisplayName       string
	Artist            string
	PlayedAt          time.Time
	PlayedAtFormatted string
	DurationSeconds   int
	DurationFormatted string
	StyleName         string
	PlayStatus        string
	PlannedBlockTime  string
	IsOperatorAction  bool
	OperatorName      string
}

// PlayHistoryMetrics métricas agregadas do período consultado
type PlayHistoryMetrics struct {
	TotalMusic       int
	TotalCommercials int
	TotalWarnings    int
	TotalDurationSec int
	TotalDurationFmt string
	TotalRecords     int
}

// PlayHistoryPageData dados completos enviados ao template play_history.html
type PlayHistoryPageData struct {
	LoggedUser         string
	Tenants            []models.Tenant
	Stores             []models.Store
	Records            []PlayHistoryViewRecord
	Metrics            PlayHistoryMetrics
	FilterTenantID     string
	FilterStoreID      string
	FilterTrackType    string
	FilterDateStart    string
	FilterDateEnd      string
	FilterSearch       string
	SelectedStoreName  string
	SelectedTenantName string
	CurrentPage        int
	TotalPages         int
	PageSize           int
	TotalRecords       int
	HasPrevPage        bool
	HasNextPage        bool
	PrevPage           int
	NextPage           int
}

// HandlePlayHistoryPage renderiza a tela de relatórios e auditoria consolidada de playout
func HandlePlayHistoryPage(w http.ResponseWriter, r *http.Request) {
	session, err := GetSessionFromRequest(r)
	if err != nil || session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Operadores são isolados no seu Web Player
	if session.PermissionLevelID == 3 && session.StoreID != nil {
		http.Redirect(w, r, fmt.Sprintf("/player/%s", session.StoreID.String()), http.StatusSeeOther)
		return
	}

	EnsureStorePlayHistoryTableExists()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 1. Extração dos parâmetros de busca e filtros
	q := r.URL.Query()
	filterTenantID := strings.TrimSpace(q.Get("tenant_id"))
	filterStoreID := strings.TrimSpace(q.Get("store_id"))
	filterTrackType := strings.TrimSpace(q.Get("track_type"))
	filterDateStart := strings.TrimSpace(q.Get("date_start"))
	filterDateEnd := strings.TrimSpace(q.Get("date_end"))
	filterSearch := strings.TrimSpace(q.Get("search"))

	// Default para os últimos 7 dias se a página for acessada sem query params
	now := time.Now()
	if !q.Has("date_start") && filterDateStart == "" {
		filterDateStart = now.AddDate(0, 0, -7).Format("2006-01-02")
	}
	if !q.Has("date_end") && filterDateEnd == "" {
		filterDateEnd = now.Format("2006-01-02")
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("limit"))
	if pageSize < 10 || pageSize > 500 {
		pageSize = 50
	}

	// 2. Isolamento de Tenant se não for Admin Master
	if session.TenantID != nil {
		filterTenantID = session.TenantID.String()
	}
	if session.StoreID != nil {
		filterStoreID = session.StoreID.String()
	}

	// 3. Montagem da Query Dinâmica
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	// Filtro de Loja ou Tenant
	if filterStoreID != "" && filterStoreID != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.store_id = $%d", argIdx))
		args = append(args, filterStoreID)
		argIdx++
	} else if filterTenantID != "" && filterTenantID != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.tenant_id = $%d", argIdx))
		args = append(args, filterTenantID)
		argIdx++
	}

	// Filtro de Tipo de Faixa
	if filterTrackType != "" && filterTrackType != "all" {
		if filterTrackType == "commercial_all" {
			whereClauses = append(whereClauses, "h.track_type IN ('commercial', 'offer', 'institutional', 'event')")
		} else if filterTrackType == "warning_all" {
			whereClauses = append(whereClauses, "h.track_type IN ('warning', 'placa', 'pre_formatted_warning')")
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("h.track_type = $%d", argIdx))
			args = append(args, filterTrackType)
			argIdx++
		}
	}

	// Filtro de Período
	if filterDateStart != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.played_at >= $%d", argIdx))
		args = append(args, filterDateStart+" 00:00:00")
		argIdx++
	}
	if filterDateEnd != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.played_at <= $%d", argIdx))
		args = append(args, filterDateEnd+" 23:59:59")
		argIdx++
	}

	// Filtro de Busca Livre
	if filterSearch != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(h.display_name ILIKE $%d OR h.artist ILIKE $%d OR s.store_name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filterSearch+"%")
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 4. Executar agregação de métricas
	metricsSQL := fmt.Sprintf(`
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE h.track_type = 'music'), 0) AS total_music,
			COALESCE(COUNT(*) FILTER (WHERE h.track_type IN ('commercial', 'offer', 'institutional', 'event')), 0) AS total_comm,
			COALESCE(COUNT(*) FILTER (WHERE h.track_type IN ('warning', 'placa', 'pre_formatted_warning')), 0) AS total_warn,
			COALESCE(SUM(h.duration_seconds), 0) AS total_dur,
			COALESCE(COUNT(*), 0) AS total_count
		FROM store_play_history h
		JOIN stores s ON s.id = h.store_id
		JOIN tenants t ON t.id = h.tenant_id
		%s
	`, whereSQL)

	var metrics PlayHistoryMetrics
	_ = database.Pool.QueryRow(ctx, metricsSQL, args...).Scan(
		&metrics.TotalMusic,
		&metrics.TotalCommercials,
		&metrics.TotalWarnings,
		&metrics.TotalDurationSec,
		&metrics.TotalRecords,
	)

	// Formatar duração total
	hours := metrics.TotalDurationSec / 3600
	mins := (metrics.TotalDurationSec % 3600) / 60
	secs := metrics.TotalDurationSec % 60
	if hours > 0 {
		metrics.TotalDurationFmt = fmt.Sprintf("%dh %02dm %02ds", hours, mins, secs)
	} else {
		metrics.TotalDurationFmt = fmt.Sprintf("%dm %02ds", mins, secs)
	}

	// 5. Paginação
	totalPages := (metrics.TotalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	// 6. Consulta dos Registros Paginados
	dataSQL := fmt.Sprintf(`
		SELECT 
			h.id, h.store_id, s.store_name, COALESCE(s.store_number, ''),
			h.tenant_id, t.trade_name AS tenant_name,
			h.track_id, h.track_type, h.display_name, COALESCE(h.artist, ''),
			h.played_at, h.duration_seconds, COALESCE(h.style_name, ''),
			COALESCE(h.play_status, 'executada'), COALESCE(h.planned_block_time, '')
		FROM store_play_history h
		JOIN stores s ON s.id = h.store_id
		JOIN tenants t ON t.id = h.tenant_id
		%s
		ORDER BY h.played_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)
	rows, err := database.Pool.Query(ctx, dataSQL, queryArgs...)
	var records []PlayHistoryViewRecord
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rec PlayHistoryViewRecord
			var playedAt time.Time
			errScan := rows.Scan(
				&rec.ID, &rec.StoreID, &rec.StoreName, &rec.StoreNumber,
				&rec.TenantID, &rec.TenantName,
				&rec.TrackID, &rec.TrackType, &rec.DisplayName, &rec.Artist,
				&playedAt, &rec.DurationSeconds, &rec.StyleName,
				&rec.PlayStatus, &rec.PlannedBlockTime,
			)
			if errScan == nil {
				rec.PlayedAt = playedAt
				rec.PlayedAtFormatted = playedAt.Local().Format("02/01/2006 15:04:05")

				m := rec.DurationSeconds / 60
				s := rec.DurationSeconds % 60
				rec.DurationFormatted = fmt.Sprintf("%02d:%02d", m, s)

				// Detectar se foi disparado por um operador
				if strings.Contains(rec.Artist, "Disparado por:") || strings.Contains(rec.Artist, "Operador:") {
					rec.IsOperatorAction = true
					parts := strings.Split(rec.Artist, "por:")
					if len(parts) > 1 {
						rec.OperatorName = strings.TrimSpace(parts[1])
					} else {
						rec.OperatorName = rec.Artist
					}
				}

				records = append(records, rec)
			}
		}
	} else {
		log.Printf("❌ Erro ao consultar store_play_history: %v\n", err)
	}

	// 7. Lista de Matrizes e Lojas para os dropdowns de filtro
	tenantsList := []models.Tenant{}
	storesList := []models.Store{}
	selectedTenantName := "Todas as Matrizes"
	selectedStoreName := "Todas as Filiais"

	tRows, _ := database.Pool.Query(ctx, "SELECT id, trade_name FROM tenants ORDER BY trade_name ASC")
	if tRows != nil {
		defer tRows.Close()
		for tRows.Next() {
			var t models.Tenant
			if err := tRows.Scan(&t.ID, &t.TradeName); err == nil {
				if filterTenantID != "" && filterTenantID == t.ID.String() {
					selectedTenantName = t.TradeName
				}
				tenantsList = append(tenantsList, t)
			}
		}
	}

	sSQL := "SELECT id, tenant_id, store_name, COALESCE(store_number, '') FROM stores ORDER BY store_name ASC"
	if session.TenantID != nil {
		sSQL = fmt.Sprintf("SELECT id, tenant_id, store_name, COALESCE(store_number, '') FROM stores WHERE tenant_id = '%s' ORDER BY store_name ASC", session.TenantID.String())
	}
	sRows, _ := database.Pool.Query(ctx, sSQL)
	if sRows != nil {
		defer sRows.Close()
		for sRows.Next() {
			var s models.Store
			var sNum string
			if err := sRows.Scan(&s.ID, &s.TenantID, &s.StoreName, &sNum); err == nil {
				s.StoreNumber = sNum
				if filterStoreID != "" && filterStoreID == s.ID.String() {
					selectedStoreName = s.StoreName
				}
				storesList = append(storesList, s)
			}
		}
	}

	data := PlayHistoryPageData{
		LoggedUser:         session.UserName,
		Tenants:            tenantsList,
		Stores:             storesList,
		Records:            records,
		Metrics:            metrics,
		FilterTenantID:     filterTenantID,
		FilterStoreID:      filterStoreID,
		FilterTrackType:    filterTrackType,
		FilterDateStart:    filterDateStart,
		FilterDateEnd:      filterDateEnd,
		FilterSearch:       filterSearch,
		SelectedStoreName:  selectedStoreName,
		SelectedTenantName: selectedTenantName,
		CurrentPage:        page,
		TotalPages:         totalPages,
		PageSize:           pageSize,
		TotalRecords:       metrics.TotalRecords,
		HasPrevPage:        page > 1,
		HasNextPage:        page < totalPages,
		PrevPage:           page - 1,
		NextPage:           page + 1,
	}

	funcs := template.FuncMap{
		"formatTrackTypeBadge": formatTrackTypeBadgeHTML,
	}

	tmpl, err := template.New("play_history.html").Funcs(funcs).ParseFiles("web/templates/play_history.html")
	if err != nil {
		http.Error(w, "Erro ao carregar template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, data)
}

// formatTrackTypeBadgeHTML helper de formatação de badges de tipo de mídia
func formatTrackTypeBadgeHTML(trackType string, displayName string) template.HTML {
	t := strings.ToLower(trackType)
	name := strings.ToLower(displayName)

	if t == "manual_override" || strings.Contains(t, "override") || strings.Contains(name, "troca manual") {
		return template.HTML(`<span class="badge bg-warning text-dark border border-warning"><i class="fa-solid fa-sliders me-1"></i>Troca Manual</span>`)
	}
	if t == "manual_override_cancel" || strings.Contains(name, "retorno para a grade") || strings.Contains(name, "grade restaurada") {
		return template.HTML(`<span class="badge bg-success bg-opacity-25 text-success border border-success border-opacity-50"><i class="fa-solid fa-rotate-left me-1"></i>Grade Restaurada</span>`)
	}

	if t == "streaming" || strings.Contains(t, "stream") || strings.Contains(name, "streaming ao vivo") {
		return template.HTML(`<span class="badge" style="background-color: #0284c7; color: #fff;"><i class="fa-solid fa-tower-broadcast me-1"></i>Streaming Ao Vivo</span>`)
	}
	if t == "fallback" || strings.Contains(name, "sem programação") || strings.Contains(name, "contingência") {
		return template.HTML(`<span class="badge" style="background-color: #e11d48; color: #fff;"><i class="fa-solid fa-triangle-exclamation me-1"></i>Sem Programação</span>`)
	}
	if t == "opening" || t == "block_opening" || strings.Contains(name, "abertura de bloco") || strings.HasPrefix(name, "abertura") {
		return template.HTML(`<span class="badge" style="background-color: #6366f1; color: #fff;"><i class="fa-solid fa-door-open me-1"></i>Abertura</span>`)
	}
	if t == "closing" || t == "block_closing" || strings.Contains(name, "fechamento de bloco") || strings.HasPrefix(name, "fechamento") {
		return template.HTML(`<span class="badge" style="background-color: #8b5cf6; color: #fff;"><i class="fa-solid fa-door-closed me-1"></i>Fechamento</span>`)
	}
	if t == "institutional" || t == "institucional" || strings.Contains(name, "institutional") || strings.Contains(name, "institucional") {
		return template.HTML(`<span class="badge" style="background-color: #06b6d4; color: #fff;"><i class="fa-solid fa-building me-1"></i>Institucional</span>`)
	}
	if t == "offer" || t == "oferta" || strings.Contains(name, "oferta") {
		return template.HTML(`<span class="badge" style="background-color: #f59e0b; color: #000;"><i class="fa-solid fa-tag me-1"></i>Oferta</span>`)
	}
	if t == "right_time" || t == "hora_certa" || strings.Contains(name, "hora certa") {
		return template.HTML(`<span class="badge" style="background-color: #10b981; color: #fff;"><i class="fa-solid fa-clock me-1"></i>Hora Certa</span>`)
	}
	if t == "temperature" || t == "temperatura" || strings.Contains(name, "temperatura") {
		return template.HTML(`<span class="badge" style="background-color: #ec4899; color: #fff;"><i class="fa-solid fa-temperature-half me-1"></i>Temperatura</span>`)
	}
	if t == "warning" || t == "aviso" || t == "placa" || strings.Contains(name, "aviso") || strings.Contains(name, "placa") {
		return template.HTML(`<span class="badge" style="background-color: #eab308; color: #000;"><i class="fa-solid fa-bullhorn me-1"></i>📢 Aviso / Placa</span>`)
	}
	if t == "event" || t == "evento" {
		return template.HTML(`<span class="badge" style="background-color: #a855f7; color: #fff;"><i class="fa-solid fa-calendar-star me-1"></i>Evento</span>`)
	}
	if t == "commercial" || t == "comercial" {
		return template.HTML(`<span class="badge" style="background-color: #0ea5e9; color: #fff;"><i class="fa-solid fa-rectangle-ad me-1"></i>Comercial</span>`)
	}
	return template.HTML(`<span class="badge bg-secondary"><i class="fa-solid fa-music me-1"></i>Música</span>`)
}

// HandleExportPlayHistoryCSV exporta o relatório filtrado em formato CSV com codificação UTF-8 com BOM
func HandleExportPlayHistoryCSV(w http.ResponseWriter, r *http.Request) {
	session, err := GetSessionFromRequest(r)
	if err != nil || session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	q := r.URL.Query()
	filterTenantID := strings.TrimSpace(q.Get("tenant_id"))
	filterStoreID := strings.TrimSpace(q.Get("store_id"))
	filterTrackType := strings.TrimSpace(q.Get("track_type"))
	filterDateStart := strings.TrimSpace(q.Get("date_start"))
	filterDateEnd := strings.TrimSpace(q.Get("date_end"))
	filterSearch := strings.TrimSpace(q.Get("search"))

	if session.TenantID != nil {
		filterTenantID = session.TenantID.String()
	}
	if session.StoreID != nil {
		filterStoreID = session.StoreID.String()
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filterStoreID != "" && filterStoreID != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.store_id = $%d", argIdx))
		args = append(args, filterStoreID)
		argIdx++
	} else if filterTenantID != "" && filterTenantID != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.tenant_id = $%d", argIdx))
		args = append(args, filterTenantID)
		argIdx++
	}

	if filterTrackType != "" && filterTrackType != "all" {
		if filterTrackType == "commercial_all" {
			whereClauses = append(whereClauses, "h.track_type IN ('commercial', 'offer', 'institutional', 'event')")
		} else if filterTrackType == "warning_all" {
			whereClauses = append(whereClauses, "h.track_type IN ('warning', 'placa', 'pre_formatted_warning')")
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("h.track_type = $%d", argIdx))
			args = append(args, filterTrackType)
			argIdx++
		}
	}
	if filterDateStart != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.played_at >= $%d", argIdx))
		args = append(args, filterDateStart+" 00:00:00")
		argIdx++
	}
	if filterDateEnd != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("h.played_at <= $%d", argIdx))
		args = append(args, filterDateEnd+" 23:59:59")
		argIdx++
	}
	if filterSearch != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(h.display_name ILIKE $%d OR h.artist ILIKE $%d OR s.store_name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filterSearch+"%")
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	dataSQL := fmt.Sprintf(`
		SELECT 
			s.store_name, COALESCE(s.store_number, ''),
			t.trade_name AS tenant_name,
			h.track_type, h.display_name, COALESCE(h.artist, ''),
			h.played_at, h.duration_seconds, COALESCE(h.style_name, ''),
			COALESCE(h.play_status, 'executada'), COALESCE(h.planned_block_time, '')
		FROM store_play_history h
		JOIN stores s ON s.id = h.store_id
		JOIN tenants t ON t.id = h.tenant_id
		%s
		ORDER BY h.played_at DESC
		LIMIT 10000
	`, whereSQL)

	rows, err := database.Pool.Query(ctx, dataSQL, args...)
	if err != nil {
		http.Error(w, "Erro ao exportar dados: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	filename := fmt.Sprintf("relatorio_veiculacao_imc_%s.csv", time.Now().Format("2006-01-02_1504"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// UTF-8 BOM para compatibilidade com Microsoft Excel
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	writer.Comma = ';' // Padrão Excel em português

	// Cabeçalhos
	_ = writer.Write([]string{
		"Data/Hora de Execucao",
		"Matriz / Cliente",
		"Filial / Loja",
		"Numero Loja",
		"Tipo de Midia",
		"Titulo / Faixa",
		"Artista / Anunciante / Operador",
		"Estilo / Categoria",
		"Duracao (Segundos)",
		"Status",
		"Horario Programado",
	})

	for rows.Next() {
		var storeName, storeNum, tenantName, trackType, displayName, artist, styleName, playStatus, plannedTime string
		var playedAt time.Time
		var durSec int

		if errScan := rows.Scan(
			&storeName, &storeNum, &tenantName,
			&trackType, &displayName, &artist,
			&playedAt, &durSec, &styleName,
			&playStatus, &plannedTime,
		); errScan == nil {
			_ = writer.Write([]string{
				playedAt.Local().Format("02/01/2006 15:04:05"),
				tenantName,
				storeName,
				storeNum,
				trackType,
				displayName,
				artist,
				styleName,
				strconv.Itoa(durSec),
				playStatus,
				plannedTime,
			})
		}
	}
	writer.Flush()
}
