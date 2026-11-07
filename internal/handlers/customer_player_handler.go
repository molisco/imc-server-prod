package handlers

import (
	"net"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	containerIPCache sync.Map
	httpClient       = &http.Client{Timeout: 5 * time.Second}
)

type ipCacheEntry struct {
	ip        string
	expiresAt time.Time
}

func getContainerIP(containerName string) (string, error) {
	if val, ok := containerIPCache.Load(containerName); ok {
		entry := val.(ipCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.ip, nil
		}
	}

	cmd := exec.Command("docker", "inspect", containerName, "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("container IP is empty for %s", containerName)
	}

	containerIPCache.Store(containerName, ipCacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(1 * time.Minute),
	})
	return ip, nil
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusBadGateway
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func proxyStoreRequest(store *models.Store, path string, method string, reqBody []byte) ([]byte, int, error) {
	if store.HarborPort == nil || *store.HarborPort <= 7000 {
		return nil, http.StatusBadRequest, fmt.Errorf("loja sem porta harbor configurada")
	}

	containerName := fmt.Sprintf("imc-radio-%s", store.ID.String())
	ip, err := getContainerIP(containerName)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("falha ao localizar container da radio: %w", err)
	}

	localPort := 5000 + (*store.HarborPort - 7000)
	targetURL := fmt.Sprintf("http://%s:%d%s", ip, localPort, path)

	var bodyReader io.Reader
	if len(reqBody) > 0 {
		bodyReader = bytes.NewReader(reqBody)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if len(reqBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 2500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return respData, resp.StatusCode, nil
}

type LiquidsoapMeta struct {
	Title            string
	Artist           string
	Type             string
	Filename         string
	ElapsedSeconds   float64
	RemainingSeconds float64
	DurationSeconds  float64
}

// getLiquidsoapMeta busca metadados em tempo real do Liquidsoap local da rádio
func getLiquidsoapMeta(store *models.Store) (*LiquidsoapMeta, error) {
	if store.HarborPort == nil || *store.HarborPort <= 7000 {
		return nil, fmt.Errorf("harbor port not configured")
	}

	targetURL := fmt.Sprintf("http://127.0.0.1:%d/getmeta", *store.HarborPort)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawEntries [][]string
	if err := json.NewDecoder(resp.Body).Decode(&rawEntries); err != nil {
		return nil, err
	}

	meta := &LiquidsoapMeta{}
	for _, entry := range rawEntries {
		if len(entry) >= 2 {
			k := strings.ToLower(entry[0])
			v := entry[1]
			switch k {
			case "title":
				meta.Title = v
			case "artist":
				meta.Artist = v
			case "type":
				meta.Type = v
			case "filename":
				meta.Filename = v
			case "elapsed_seconds", "elapsed":
				meta.ElapsedSeconds, _ = strconv.ParseFloat(v, 64)
			case "remaining_seconds", "remaining":
				meta.RemainingSeconds, _ = strconv.ParseFloat(v, 64)
			case "duration":
				meta.DurationSeconds, _ = strconv.ParseFloat(v, 64)
			}
		}
	}
	return meta, nil
}

// checkStoreAccess valida se o usuário logado na sessão tem permissão para a loja
func checkStoreAccess(session *models.SessionData, store *models.Store) error {
	if session == nil {
		return fmt.Errorf("não autenticado")
	}

	// 1. Administrador Master (TenantID == nil) tem acesso irrestrito a todas as rádios
	if session.TenantID == nil {
		return nil
	}

	// 2. Usuário de Empresa só pode acessar lojas da sua própria empresa
	if store.TenantID != *session.TenantID {
		return fmt.Errorf("você não tem permissão para acessar a rádio de outra empresa")
	}

	// 3. Usuário com loja específica vinculada só pode acessar a sua própria loja
	if session.StoreID != nil && *session.StoreID != store.ID {
		return fmt.Errorf("seu usuário tem permissão de acesso restrita apenas à sua própria filial")
	}

	return nil
}

// CustomerPlayerPageData carrega os dados visuais e de streaming para a tela do player do cliente
type CustomerPlayerPageData struct {
	Store            models.Store
	Tenant           models.Tenant
	Theme            map[string]string
	StreamURL        string
	MountPoint       string
	IsOnline         bool
	IsSuspended      bool
	SuspendedReason  string
	Listeners        int
	IsStreaming      bool
	StreamName       string
	InitialTitle     string
	InitialArtist    string
	InitialType      string
	InitialRemaining string
	InitialElapsed   string
	InitialProgress  float64
	User             models.SessionData
	IsStandby        bool
	StandbyReason    string
	NextOpenTime     string
}

// NowPlayingResponse representa o payload JSON retornado pela API para atualizacao em tempo real
type NowPlayingResponse struct {
	ManualOverride     interface{} `json:"manual_override"`
	Online             bool    `json:"online"`
	StoreName          string  `json:"store_name"`
	StoreNumber        string  `json:"store_number"`
	TenantName         string  `json:"tenant_name"`
	StreamURL          string  `json:"stream_url"`
	Listeners          int     `json:"listeners"`
	IsStreaming        bool    `json:"is_streaming"`
	StreamName         string  `json:"stream_name,omitempty"`
	Title              string  `json:"title"`
	Artist             string  `json:"artist"`
	Album              string  `json:"album"`
	MediaType          string  `json:"media_type"`
	DurationSec        int     `json:"duration_seconds"`
	ElapsedSec         int     `json:"elapsed_seconds"`
	RemainingSec       int     `json:"remaining_seconds"`
	FormattedDuration  string  `json:"formatted_duration"`
	FormattedElapsed   string  `json:"formatted_elapsed"`
	FormattedRemaining string  `json:"formatted_remaining"`
	ProgressPercent    float64     `json:"progress_percent"`
	ServerTime         string      `json:"server_time"`
	IsStandby          bool        `json:"is_standby"`
	StandbyReason      string      `json:"standby_reason,omitempty"`
	NextOpenTime       string      `json:"next_open_time,omitempty"`
}

// findStoreAndTenant busca uma loja e seu respectivo tenant por ID (UUID), token ou numero da loja
func findStoreAndTenant(ctx context.Context, identifier string) (*models.Store, *models.Tenant, error) {
	cleanID := strings.TrimSpace(identifier)

	query := `
		SELECT s.id, s.tenant_id, t.trade_name, s.regional_id, r.name,
		       s.store_name, s.store_number, s.city, s.state, s.timezone,
		       s.has_audio, s.has_video, s.radio_type, s.harbor_port,
		       s.external_stream_url, s.external_stream_name, s.is_active, s.created_at, s.updated_at,
		       t.logo_url, t.primary_color, t.background_color, t.font_color, t.field_background_color, t.is_active
		FROM stores s
		INNER JOIN tenants t ON s.tenant_id = t.id
		LEFT JOIN regionals r ON s.regional_id = r.id
		WHERE s.token = $1 OR LOWER(s.store_number) = LOWER($1) OR s.id::text = $1
		LIMIT 1
	`
	var s models.Store
	var t models.Tenant
	var harborPort *int
	var extURL, extName, regName, logoURL *string

	row := database.Pool.QueryRow(ctx, query, cleanID)
	err := row.Scan(
		&s.ID, &s.TenantID, &s.TenantTradeName, &s.RegionalID, &regName,
		&s.StoreName, &s.StoreNumber, &s.City, &s.State, &s.Timezone,
		&s.HasAudio, &s.HasVideo, &s.RadioType, &harborPort,
		&extURL, &extName, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		&logoURL, &t.PrimaryColor, &t.BackgroundColor, &t.FontColor, &t.FieldBackgroundColor, &t.IsActive,
	)
	if err != nil {
		return nil, nil, err
	}

	s.HarborPort = harborPort
	s.ExternalStreamURL = extURL
	s.ExternalStreamName = extName
	s.RegionalName = regName

	t.ID = s.TenantID
	t.TradeName = s.TenantTradeName
	t.LogoURL = logoURL

	return &s, &t, nil
}

// HandleCustomerPlayerRoot redireciona automaticamente o usuário logado para a rádio da sua loja
func HandleCustomerPlayerRoot(w http.ResponseWriter, r *http.Request) {
	session, err := GetAndRenewSession(w, r)
	if err != nil || session == nil {
		http.Redirect(w, r, "/login?redirect=/player", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Se o usuário tem loja específica vinculada:
	if session.StoreID != nil {
		http.Redirect(w, r, fmt.Sprintf("/player/%s", session.StoreID.String()), http.StatusSeeOther)
		return
	}

	// Se o usuário pertence a um Tenant:
	if session.TenantID != nil {
		var storeID uuid.UUID
		err := database.Pool.QueryRow(ctx, `SELECT id FROM stores WHERE tenant_id = $1 AND is_active = true ORDER BY store_name ASC LIMIT 1`, *session.TenantID).Scan(&storeID)
		if err == nil {
			http.Redirect(w, r, fmt.Sprintf("/player/%s", storeID.String()), http.StatusSeeOther)
			return
		}
		http.Error(w, "Nenhuma rádio ativa encontrada para a sua empresa.", http.StatusNotFound)
		return
	}

	// Master Admin:
	var storeID uuid.UUID
	err = database.Pool.QueryRow(ctx, `SELECT id FROM stores WHERE is_active = true ORDER BY store_name ASC LIMIT 1`).Scan(&storeID)
	if err == nil {
		http.Redirect(w, r, fmt.Sprintf("/player/%s", storeID.String()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/radios/status", http.StatusSeeOther)
}

func getLocalStoreNowPlaying(store *models.Store) (title string, artist string, isStream bool) {
	data, status, err := proxyStoreRequest(store, "/api/status", "GET", nil)
	if err == nil && status == 200 && len(data) > 0 {
		var resp struct {
			CurrentTrack struct {
				Title  string `json:"title"`
				Artist string `json:"artist"`
				Type   string `json:"type"`
			} `json:"current_track"`
		}
		if errJSON := json.Unmarshal(data, &resp); errJSON == nil {
			if resp.CurrentTrack.Title != "" && resp.CurrentTrack.Title != "Sem reprodução" {
				return resp.CurrentTrack.Title, resp.CurrentTrack.Artist, resp.CurrentTrack.Type == "STREAMING"
			}
		}
	}
	return "", "", false
}

// resolveActiveStreamName descobre o nome do streaming ativo para a loja
func resolveActiveStreamName(ctx context.Context, store *models.Store) string {
	if localTitle, _, isStream := getLocalStoreNowPlaying(store); isStream && localTitle != "" && localTitle != "Sem reprodução" {
		return localTitle
	}

	if store.ExternalStreamName != nil && strings.TrimSpace(*store.ExternalStreamName) != "" {
		return strings.TrimSpace(*store.ExternalStreamName)
	}

	return "Streaming Ao Vivo"
}

// HandleCustomerPlayerPage renderiza a interface do Web Player do Cliente (Now Playing)
func HandleCustomerPlayerPage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		http.Error(w, "Identificador da radio/loja nao fornecido", http.StatusBadRequest)
		return
	}

	// Validação de Sessão Obrigatória
	session, err := GetAndRenewSession(w, r)
	if err != nil || session == nil {
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, tenant, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		http.Error(w, "Radio ou Loja nao encontrada.", http.StatusNotFound)
		return
	}

	// Validação de Permissão Multi-Tenant
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		http.Error(w, fmt.Sprintf("Acesso Negado: %s", errAccess.Error()), http.StatusForbidden)
		return
	}

	mount := GetStoreMountPoint(*store)
	icecastSources := parseIcecastSources()
	source, isOnline := icecastSources[mount]

	isSuspended := !store.IsActive || !tenant.IsActive
	var suspendedReason string
	if !tenant.IsActive {
		suspendedReason = fmt.Sprintf("A empresa %s encontra-se temporariamente suspensa.", tenant.TradeName)
	} else if !store.IsActive {
		suspendedReason = fmt.Sprintf("A filial %s encontra-se temporariamente suspensa.", store.StoreName)
	}

	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := resolveActiveStreamName(ctx, store)
	title := "Radio no Ar"
	artist := store.StoreName
	mediaType := "Musica"
	formattedRemaining := "--:--"
	formattedElapsed := "00:00"
	progressPercent := 0.0

	if isSuspended {
		isOnline = false
		title = "Rádio Temporariamente Suspensa"
		artist = "Contate o Suporte Comercial / Financeiro"
		mediaType = "Suspenso"
	}

	// Tenta buscar metadados detalhados com tempo restante do Liquidsoap
	meta, errMeta := getLiquidsoapMeta(store)
	if errMeta == nil && meta != nil {
		isInfRemaining := math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0
		isCommercialActive := (meta.Type == "commercial" || meta.Type == "institutional" || meta.Type == "warning") && meta.RemainingSeconds > 0 && !isInfRemaining

		if isCommercialActive {
			if meta.Type == "warning" {
				mediaType = "Aviso / Comunicado"
			} else {
				mediaType = "Comercial / Evento"
			}
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			if meta.RemainingSeconds > 0 {
				remM := int(meta.RemainingSeconds) / 60
				remS := int(meta.RemainingSeconds) % 60
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remM, remS)
			}
			if meta.ElapsedSeconds > 0 {
				elM := int(meta.ElapsedSeconds) / 60
				elS := int(meta.ElapsedSeconds) % 60
				formattedElapsed = fmt.Sprintf("%02d:%02d", elM, elS)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		} else if isStreaming || isInfRemaining || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
			isStreaming = true
			mediaType = "Streaming Ao Vivo"
			title = streamName
			artist = "Transmissão Streaming Ao Vivo"
			formattedRemaining = "Contínuo"
			formattedElapsed = "AO VIVO"
			progressPercent = 100.0
		} else if meta.Title != "" {
			mediaType = "Música"
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			if meta.RemainingSeconds > 0 {
				remM := int(meta.RemainingSeconds) / 60
				remS := int(meta.RemainingSeconds) % 60
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remM, remS)
			} else if meta.ElapsedSeconds > 0 {
				formattedRemaining = "Quase no fim"
			}
			if meta.ElapsedSeconds > 0 {
				elM := int(meta.ElapsedSeconds) / 60
				elS := int(meta.ElapsedSeconds) % 60
				formattedElapsed = fmt.Sprintf("%02d:%02d", elM, elS)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		}
	} else if isOnline && source.Title != "" {
		title = source.Title
		if source.Artist != "" {
			artist = source.Artist
		}
	}

	var isStandby bool
	var standbyReason string
	var nextOpenTime string
	if localData, status, err := proxyStoreRequest(store, "/api/status", "GET", nil); err == nil && status == 200 && len(localData) > 0 {
		var localStatus struct {
			IsStandby     bool   `json:"is_standby"`
			StandbyReason string `json:"standby_reason"`
			NextOpenTime  string `json:"next_open_time"`
		}
		if errJSON := json.Unmarshal(localData, &localStatus); errJSON == nil {
			isStandby = localStatus.IsStandby
			standbyReason = localStatus.StandbyReason
			nextOpenTime = localStatus.NextOpenTime
		}
	}

	if isStandby {
		title = "🌙 Standby (Fora do Horário)"
		if nextOpenTime != "" {
			artist = fmt.Sprintf("Retorna às %s", nextOpenTime)
		} else {
			artist = "Aguardando horário de abertura"
		}
		mediaType = "Standby"
		formattedRemaining = "--:--"
		formattedElapsed = "--:--"
		progressPercent = 0.0
		isStreaming = false
	}

	// URL relativa para funcionar em qualquer porta externa (ex: 20080, 1080, 8080, 80, duckdns)
	streamURL := fmt.Sprintf("/stream/%s", store.ID.String())

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         tenant.PrimaryColor,
		"SecondaryColor":       "#161925",
		"BackgroundColor":      tenant.BackgroundColor,
		"FontColor":            tenant.FontColor,
		"FieldBackgroundColor": tenant.FieldBackgroundColor,
	}
	if tenant.LogoURL != nil && *tenant.LogoURL != "" {
		theme["LogoURL"] = *tenant.LogoURL
	}
	if theme["PrimaryColor"] == "" {
		theme["PrimaryColor"] = "#4364f7"
	}
	if theme["BackgroundColor"] == "" {
		theme["BackgroundColor"] = "#0f111a"
	}
	if theme["FontColor"] == "" {
		theme["FontColor"] = "#ffffff"
	}
	if theme["FieldBackgroundColor"] == "" {
		theme["FieldBackgroundColor"] = "#161925"
	}

	data := CustomerPlayerPageData{
		Store:            *store,
		Tenant:           *tenant,
		Theme:            theme,
		StreamURL:        streamURL,
		MountPoint:       mount,
		IsOnline:         isOnline,
		IsSuspended:      isSuspended,
		SuspendedReason:  suspendedReason,
		Listeners:        source.Listeners,
		IsStreaming:      isStreaming,
		StreamName:       streamName,
		InitialTitle:     title,
		InitialArtist:    artist,
		InitialType:      mediaType,
		InitialRemaining: formattedRemaining,
		InitialElapsed:   formattedElapsed,
		InitialProgress:  progressPercent,
		User:             *session,
		IsStandby:        isStandby,
		StandbyReason:    standbyReason,
		NextOpenTime:     nextOpenTime,
	}

	tmpl, err := template.ParseFiles("web/templates/customer_player.html")
	if err != nil {
		http.Error(w, "Erro ao carregar o player visual: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// HandlePlayerNowPlayingAPI responde com metadados em JSON para o player atualizar em tempo real
func HandlePlayerNowPlayingAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	store, tenant, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		http.Error(w, `{"error":"store not found"}`, http.StatusNotFound)
		return
	}

	mount := GetStoreMountPoint(*store)
	icecastSources := parseIcecastSources()
	source, isOnline := icecastSources[mount]

	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := resolveActiveStreamName(ctx, store)
	title := "Radio no Ar"
	artist := store.StoreName
	album := ""
	if !store.IsActive || !tenant.IsActive {
		resp := NowPlayingResponse{
			Online:      false,
			StoreName:   store.StoreName,
			StoreNumber: store.StoreNumber,
			TenantName:  tenant.TradeName,
			Title:       "Rádio Temporariamente Suspensa",
			Artist:      "Contate o Suporte Comercial / Financeiro",
			MediaType:   "Suspenso",
			ServerTime:  time.Now().Format("15:04:05"),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	mediaType := "Musica"
	durationSec := 0
	elapsedSec := 0
	remainingSec := 0
	formattedDuration := "--:--"
	formattedElapsed := "00:00"
	formattedRemaining := "--:--"
	progressPercent := 0.0

	// 1. Busca metadados de execução do Liquidsoap
	meta, errMeta := getLiquidsoapMeta(store)
	if errMeta == nil && meta != nil {
		isInfRemaining := math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0
		isCommercialActive := (meta.Type == "commercial" || meta.Type == "institutional" || meta.Type == "warning") && meta.RemainingSeconds > 0 && !isInfRemaining

		if isCommercialActive {
			if meta.Type == "warning" {
				mediaType = "Aviso / Comunicado"
			} else {
				mediaType = "Comercial / Evento"
			}
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			durationSec = int(meta.DurationSeconds)
			elapsedSec = int(meta.ElapsedSeconds)
			remainingSec = int(meta.RemainingSeconds)

			if durationSec > 0 {
				formattedDuration = fmt.Sprintf("%02d:%02d", durationSec/60, durationSec%60)
			}
			if elapsedSec > 0 {
				formattedElapsed = fmt.Sprintf("%02d:%02d", elapsedSec/60, elapsedSec%60)
			}
			if remainingSec > 0 {
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remainingSec/60, remainingSec%60)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		} else if isStreaming || isInfRemaining || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
			isStreaming = true
			mediaType = "Streaming Ao Vivo"
			title = streamName
			artist = "Transmissão Streaming Ao Vivo"
			formattedRemaining = "Contínuo"
			formattedElapsed = "AO VIVO"
			formattedDuration = "Ao Vivo"
			progressPercent = 100.0
		} else if meta.Title != "" {
			mediaType = "Música"
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			durationSec = int(meta.DurationSeconds)
			elapsedSec = int(meta.ElapsedSeconds)
			remainingSec = int(meta.RemainingSeconds)

			if durationSec > 0 {
				formattedDuration = fmt.Sprintf("%02d:%02d", durationSec/60, durationSec%60)
			}
			if elapsedSec > 0 {
				formattedElapsed = fmt.Sprintf("%02d:%02d", elapsedSec/60, elapsedSec%60)
			}
			if remainingSec > 0 {
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remainingSec/60, remainingSec%60)
			} else if elapsedSec > 0 {
				formattedRemaining = "Quase no fim"
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}

			// 2. Busca informações do álbum no acervo
			var dbAlbum string
			errQuery := database.Pool.QueryRow(ctx, `
				SELECT COALESCE(album, '')
				FROM media_library 
				WHERE (display_name = $1 OR file_name = $1 OR title = $1)
				LIMIT 1
			`, title).Scan(&dbAlbum)
			if errQuery == nil {
				album = dbAlbum
			}
		}
	} else if isOnline && source.Title != "" {
		title = source.Title
		if source.Artist != "" {
			artist = source.Artist
		}
	}

	var isStandby bool
	var standbyReason string
	var nextOpenTime string
	var manualOverride interface{}

	if localData, status, err := proxyStoreRequest(store, "/api/status", "GET", nil); err == nil && status == 200 && len(localData) > 0 {
		var localStatus struct {
			IsStandby      bool        `json:"is_standby"`
			StandbyReason  string      `json:"standby_reason"`
			NextOpenTime   string      `json:"next_open_time"`
			ManualOverride interface{} `json:"manual_override"`
		}
		if errJSON := json.Unmarshal(localData, &localStatus); errJSON == nil {
			isStandby = localStatus.IsStandby
			standbyReason = localStatus.StandbyReason
			nextOpenTime = localStatus.NextOpenTime
			manualOverride = localStatus.ManualOverride
		}
	}

	if isStandby {
		title = "🌙 Standby (Fora do Horário)"
		if nextOpenTime != "" {
			artist = fmt.Sprintf("Retorna às %s", nextOpenTime)
		} else {
			artist = "Aguardando horário de abertura"
		}
		mediaType = "Standby"
		formattedDuration = "--:--"
		formattedElapsed = "--:--"
		formattedRemaining = "--:--"
		progressPercent = 0.0
		isStreaming = false
	}

	streamURL := mount

	resp := NowPlayingResponse{
		ManualOverride:     manualOverride,
		Online:             isOnline,
		StoreName:          store.StoreName,
		StoreNumber:        store.StoreNumber,
		TenantName:         tenant.TradeName,
		StreamURL:          streamURL,
		Listeners:          source.Listeners,
		IsStreaming:        isStreaming,
		StreamName:         streamName,
		Title:              title,
		Artist:             artist,
		Album:              album,
		MediaType:          mediaType,
		DurationSec:        durationSec,
		ElapsedSec:         elapsedSec,
		RemainingSec:       remainingSec,
		FormattedDuration:  formattedDuration,
		FormattedElapsed:   formattedElapsed,
		FormattedRemaining: formattedRemaining,
		ProgressPercent:    progressPercent,
		ServerTime:         time.Now().Format("15:04:05"),
		IsStandby:          isStandby,
		StandbyReason:      standbyReason,
		NextOpenTime:       nextOpenTime,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(resp)
}

// HandlePlayerDiagnosticsAPI proxy do diagnostico completo do player local com controle de acesso
func HandlePlayerDiagnosticsAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	targetPath := "/api/diagnostics"
	if r.URL.RawQuery != "" {
		targetPath += "?" + r.URL.RawQuery
	}
	data, statusCode, err := proxyStoreRequest(store, targetPath, http.MethodGet, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerLogsAPI proxy dos logs centralizados e do Liquidsoap da rádio
func HandlePlayerLogsAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/centralized-logs", http.MethodGet, nil)
	if err != nil || statusCode >= 400 {
		if statusCode == http.StatusNotFound {
			data, statusCode, err = proxyStoreRequest(store, "/api/liquidsoap-logs", http.MethodGet, nil)
		}
		if err != nil {
			sendJSONError(w, err.Error(), statusCode)
			return
		}
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerVoicePacksAPI proxy da lista de pacotes de voz
func HandlePlayerVoicePacksAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/voice-packs", http.MethodGet, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerChimesAPI proxy da lista de sinais sonoros (chimes)
func HandlePlayerChimesAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/chimes", http.MethodGet, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerPreFormattedWarningsAPI proxy da lista de avisos gravados
func HandlePlayerPreFormattedWarningsAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/pre-formatted-warnings", http.MethodGet, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerTriggerVehicleWarningAPI proxy para disparar aviso de placa no ar
func HandlePlayerTriggerVehicleWarningAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSONError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var reqMap map[string]interface{}
	if errUn := json.Unmarshal(body, &reqMap); errUn == nil && reqMap != nil {
		if session != nil && session.UserName != "" {
			reqMap["triggered_by"] = session.UserName
		}
		if newBody, errMar := json.Marshal(reqMap); errMar == nil {
			body = newBody
		}
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/trigger-vehicle-warning", http.MethodPost, body)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerTriggerPreFormattedWarningAPI proxy para disparar aviso gravado no ar
func HandlePlayerTriggerPreFormattedWarningAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSONError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var reqMapPre map[string]interface{}
	if errUn := json.Unmarshal(body, &reqMapPre); errUn == nil && reqMapPre != nil {
		if session != nil && session.UserName != "" {
			reqMapPre["triggered_by"] = session.UserName
		}
		if newBody, errMar := json.Marshal(reqMapPre); errMar == nil {
			body = newBody
		}
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/trigger-pre-formatted-warning", http.MethodPost, body)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerSkipAPI proxy para pular musica atual
func HandlePlayerSkipAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/skip", http.MethodPost, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerSyncAPI dispara sincronismo forçado com o servidor central
func HandlePlayerSyncAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/sync", http.MethodPost, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerRedistributeAPI dispara redistribuição dos comerciais do dia
func HandlePlayerRedistributeAPI(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/redistribute", http.MethodPost, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandleStoreStreamProxy faz o proxy transparente do áudio Icecast diretamente pelo ID da loja
func HandleStoreStreamProxy(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	store, tenant, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		http.Error(w, "Radio nao encontrada", http.StatusNotFound)
		return
	}

	if !store.IsActive || (tenant != nil && !tenant.IsActive) {
		http.Error(w, "Radio temporariamente suspensa", http.StatusForbidden)
		return
	}

	mount := GetStoreMountPoint(*store)
	proxyIcecastStream(w, r, mount)
}

// HandleMountStreamProxy faz o proxy de qualquer rota de mount point (/m0001_l1, /mcf4f_l001, /loja_ce275c7f, etc.)
func HandleMountStreamProxy(w http.ResponseWriter, r *http.Request) {
	mount := r.URL.Path
	if strings.HasPrefix(mount, "/loja_") {
		prefix := strings.TrimPrefix(mount, "/loja_")
		var store models.Store
		err := database.Pool.QueryRow(r.Context(), "SELECT id, tenant_id, COALESCE(store_number, ''), is_active FROM stores WHERE id::text LIKE $1 LIMIT 1", prefix+"%").Scan(&store.ID, &store.TenantID, &store.StoreNumber, &store.IsActive)
		if err == nil {
			if !store.IsActive {
				http.Error(w, "Radio temporariamente suspensa", http.StatusForbidden)
				return
			}
			mount = GetStoreMountPoint(store)
		}
	}
	proxyIcecastStream(w, r, mount)
}

func proxyIcecastStream(w http.ResponseWriter, r *http.Request, mount string) {
	icecastURL := "http://127.0.0.1:8000" + mount

	// Cliente sem timeout para streaming contínuo ininterrupto com encerramento rápido
	streamClient := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		},
	}

	method := r.Method
	if method != http.MethodHead && method != http.MethodGet {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(r.Context(), method, icecastURL, nil)
	if err != nil {
		http.Error(w, "Erro ao criar requisicao de stream", http.StatusInternalServerError)
		return
	}

	// Copia headers da requisicao (como Range, User-Agent, etc.)
	for k, v := range r.Header {
		if strings.ToLower(k) != "host" {
			req.Header[k] = v
		}
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		http.Error(w, "Falha ao conectar ao servidor de transmissao Icecast", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, hasFlusher := w.(http.Flusher)
	w.WriteHeader(resp.StatusCode)
	if hasFlusher {
		flusher.Flush()
	}

	if method == http.MethodHead {
		return
	}

	buf := make([]byte, 8192)
	for {
		select {
		case <-r.Context().Done():
			return
		default:
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return
				}
				if hasFlusher {
					flusher.Flush()
				}
			}
			if readErr != nil {
				return
			}
		}
	}
}

// HandlePlayerManualOverride proxy para aplicar override temporário de estilo/streaming
func HandlePlayerManualOverride(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	data, statusCode, err := proxyStoreRequest(store, "/api/player/manual-override", http.MethodPost, bodyBytes)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}

// HandlePlayerManualOverrideCancel proxy para cancelar override temporário
func HandlePlayerManualOverrideCancel(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	store, _, err := findStoreAndTenant(ctx, idParam)
	if err != nil || store == nil {
		sendJSONError(w, "store not found", http.StatusNotFound)
		return
	}

	session, _ := GetAndRenewSession(w, r)
	if errAccess := checkStoreAccess(session, store); errAccess != nil {
		sendJSONError(w, errAccess.Error(), http.StatusForbidden)
		return
	}

	data, statusCode, err := proxyStoreRequest(store, "/api/player/manual-override/cancel", http.MethodPost, nil)
	if err != nil {
		sendJSONError(w, err.Error(), statusCode)
		return
	}

	if statusCode < 100 || statusCode > 599 {
		statusCode = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(data)
}


func getStoreManualOverrideState(store *models.Store) interface{} {
	data, statusCode, err := proxyStoreRequest(store, "/api/status", http.MethodGet, nil)
	if err != nil || statusCode != 200 {
		return nil
	}
	var raw struct {
		ManualOverride interface{} `json:"manual_override"`
	}
	if errJSON := json.Unmarshal(data, &raw); errJSON == nil {
		return raw.ManualOverride
	}
	return nil
}
