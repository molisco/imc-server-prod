package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"imc-server/internal/database"
	"imc-server/internal/models"
)

// AuthLogEntry representa um registro de auditoria de autenticação
type AuthLogEntry struct {
	ID         uuid.UUID  `json:"id"`
	EventType  string     `json:"event_type"` // 'LOGIN', 'LOGOUT', 'LOGIN_FAILED'
	UserID     *uuid.UUID `json:"user_id"`
	UserName   string     `json:"user_name"`
	UserEmail  string     `json:"user_email"`
	UserRole   string     `json:"user_role"`
	TenantID   *uuid.UUID `json:"tenant_id"`
	TenantName string     `json:"tenant_name"`
	StoreID    *uuid.UUID `json:"store_id"`
	StoreName  string     `json:"store_name"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
}

// FormatCreatedAt retorna a data formatada para a interface
func (l AuthLogEntry) FormatCreatedAt() string {
	return l.CreatedAt.Format("02/01/2006 15:04:05")
}

// GetClientIP extrai o IP real do cliente tratando proxies reversos (Caddy/Nginx)
func GetClientIP(r *http.Request) string {
	// 1. Tenta X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// 2. Tenta X-Real-IP
	xrip := r.Header.Get("X-Real-IP")
	if xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// 3. Fallback para RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

// LogAuthEvent registra de forma assíncrona/segura o evento de login/logout no banco
func LogAuthEvent(ctx context.Context, r *http.Request, eventType string, userID *uuid.UUID, userName string, userEmail string, userRole string, tenantID *uuid.UUID, storeID *uuid.UUID) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ip := GetClientIP(r)
		ua := r.UserAgent()

		var tenantName, storeName string

		// Resolve nome do tenant se houver
		if tenantID != nil {
			_ = database.Pool.QueryRow(bgCtx, `SELECT COALESCE(trade_name, legal_name, '') FROM tenants WHERE id = $1`, *tenantID).Scan(&tenantName)
		}

		// Resolve nome da loja se houver
		if storeID != nil {
			_ = database.Pool.QueryRow(bgCtx, `SELECT COALESCE(store_name, '') FROM stores WHERE id = $1`, *storeID).Scan(&storeName)
		}

		if userRole == "" {
			userRole = "Usuário"
		}

		query := `
			INSERT INTO user_auth_logs (
				event_type, user_id, user_name, user_email, user_role,
				tenant_id, tenant_name, store_id, store_name,
				ip_address, user_agent, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		`
		_, err := database.Pool.Exec(bgCtx, query,
			eventType, userID, userName, userEmail, userRole,
			tenantID, tenantName, storeID, storeName,
			ip, ua,
		)
		if err != nil {
			fmt.Printf("⚠️ [AuthLog] Falha ao registrar log de autenticação: %v\n", err)
		}
	}()
}

// AuthLogsPageData estrutura os dados da tela de relatório
type AuthLogsPageData struct {
	User           models.SessionData
	Theme          map[string]string
	Logs           []AuthLogEntry
	Tenants        []models.Tenant
	Stores         []models.Store
	TotalLogins    int
	TotalLogouts   int
	TotalFailed    int
	SelectedTenant string
	SelectedStore  string
	SelectedEvent  string
	StartDate      string
	EndDate        string
	SearchQuery    string
}

// HandleAuthLogsPage renderiza a página de relatório e auditoria de acessos
func HandleAuthLogsPage(w http.ResponseWriter, r *http.Request) {
	session, errSession := GetAndRenewSession(w, r)
	if errSession != nil || session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parâmetros de filtro
	selectedTenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	selectedStore := strings.TrimSpace(r.URL.Query().Get("store_id"))
	selectedEvent := strings.TrimSpace(r.URL.Query().Get("event_type"))
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))

	now := time.Now()
	if startDate == "" {
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	// Se o usuário não for SuperAdmin (PermissionLevelID != 1), trava na Matriz dele
	if session.PermissionLevelID > 1 && session.TenantID != nil {
		selectedTenant = session.TenantID.String()
	}

	// Carrega lista de Tenants e Lojas para os selects
	var tenants []models.Tenant
	var stores []models.Store

	if session.PermissionLevelID == 1 {
		tRows, err := database.Pool.Query(ctx, `SELECT id, trade_name, legal_name FROM tenants ORDER BY trade_name ASC`)
		if err == nil {
			defer tRows.Close()
			for tRows.Next() {
				var t models.Tenant
				if errScan := tRows.Scan(&t.ID, &t.TradeName, &t.LegalName); errScan == nil {
					tenants = append(tenants, t)
				}
			}
		}
	}

	sQuery := `SELECT id, tenant_id, store_name, store_number FROM stores`
	if selectedTenant != "" {
		sQuery += fmt.Sprintf(` WHERE tenant_id = '%s'`, selectedTenant)
	}
	sQuery += ` ORDER BY store_name ASC`
	sRows, err := database.Pool.Query(ctx, sQuery)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var s models.Store
			if errScan := sRows.Scan(&s.ID, &s.TenantID, &s.StoreName, &s.StoreNumber); errScan == nil {
				stores = append(stores, s)
			}
		}
	}

	// Monta a query filtrada de logs
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, event_type, user_id, user_name, user_email, user_role,
		       tenant_id, COALESCE(tenant_name, ''), store_id, COALESCE(store_name, ''),
		       ip_address, COALESCE(user_agent, ''), created_at
		FROM user_auth_logs
		WHERE created_at >= $1::date AND created_at < ($2::date + INTERVAL '1 day')
	`)
	args := []interface{}{startDate, endDate}
	argIdx := 3

	if selectedTenant != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND tenant_id = $%d", argIdx))
		args = append(args, selectedTenant)
		argIdx++
	}

	if selectedStore != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND store_id = $%d", argIdx))
		args = append(args, selectedStore)
		argIdx++
	}

	if selectedEvent != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND event_type = $%d", argIdx))
		args = append(args, selectedEvent)
		argIdx++
	}

	if searchQuery != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND (user_name ILIKE $%d OR user_email ILIKE $%d OR ip_address ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+searchQuery+"%")
		argIdx++
	}

	queryBuilder.WriteString(" ORDER BY created_at DESC LIMIT 500")

	rows, err := database.Pool.Query(ctx, queryBuilder.String(), args...)
	var logs []AuthLogEntry
	totalLogins, totalLogouts, totalFailed := 0, 0, 0

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l AuthLogEntry
			errScan := rows.Scan(
				&l.ID, &l.EventType, &l.UserID, &l.UserName, &l.UserEmail, &l.UserRole,
				&l.TenantID, &l.TenantName, &l.StoreID, &l.StoreName,
				&l.IPAddress, &l.UserAgent, &l.CreatedAt,
			)
			if errScan == nil {
				logs = append(logs, l)
				switch l.EventType {
				case "LOGIN":
					totalLogins++
				case "LOGOUT":
					totalLogouts++
				case "LOGIN_FAILED":
					totalFailed++
				}
			}
		}
	}

	theme := map[string]string{
		"LogoURL": "/static/img/logo.png",
	}

	data := AuthLogsPageData{
		User:           *session,
		Theme:          theme,
		Logs:           logs,
		Tenants:        tenants,
		Stores:         stores,
		TotalLogins:    totalLogins,
		TotalLogouts:   totalLogouts,
		TotalFailed:    totalFailed,
		SelectedTenant: selectedTenant,
		SelectedStore:  selectedStore,
		SelectedEvent:  selectedEvent,
		StartDate:      startDate,
		EndDate:        endDate,
		SearchQuery:    searchQuery,
	}

	tmpl, errTmpl := template.ParseFiles("web/templates/auth_logs_report.html")
	if errTmpl != nil {
		http.Error(w, "Erro ao carregar template do relatório: "+errTmpl.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// HandleAuthLogsExportCSV exporta os logs filtrados em formato CSV para download
func HandleAuthLogsExportCSV(w http.ResponseWriter, r *http.Request) {
	session, errSession := GetSessionFromRequest(r)
	if errSession != nil || session == nil {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	selectedTenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	selectedStore := strings.TrimSpace(r.URL.Query().Get("store_id"))
	selectedEvent := strings.TrimSpace(r.URL.Query().Get("event_type"))
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("search"))

	now := time.Now()
	if startDate == "" {
		startDate = now.AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	if session.PermissionLevelID > 1 && session.TenantID != nil {
		selectedTenant = session.TenantID.String()
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT event_type, user_name, user_email, user_role,
		       COALESCE(tenant_name, ''), COALESCE(store_name, ''),
		       ip_address, COALESCE(user_agent, ''), created_at
		FROM user_auth_logs
		WHERE created_at >= $1::date AND created_at < ($2::date + INTERVAL '1 day')
	`)
	args := []interface{}{startDate, endDate}
	argIdx := 3

	if selectedTenant != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND tenant_id = $%d", argIdx))
		args = append(args, selectedTenant)
		argIdx++
	}
	if selectedStore != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND store_id = $%d", argIdx))
		args = append(args, selectedStore)
		argIdx++
	}
	if selectedEvent != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND event_type = $%d", argIdx))
		args = append(args, selectedEvent)
		argIdx++
	}
	if searchQuery != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND (user_name ILIKE $%d OR user_email ILIKE $%d OR ip_address ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+searchQuery+"%")
		argIdx++
	}

	queryBuilder.WriteString(" ORDER BY created_at DESC")

	rows, err := database.Pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		http.Error(w, "Erro ao buscar registros para exportação", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=relatorio_acessos_%s_%s.csv", startDate, endDate))
	// Escreve UTF-8 BOM para abrir perfeitamente no Excel
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	writer.Comma = ';' // Padrão Excel em português

	// Cabeçalho CSV
	_ = writer.Write([]string{
		"Data e Hora",
		"Evento",
		"Usuário",
		"E-mail",
		"Perfil",
		"Matriz / Cliente",
		"Filial (Loja)",
		"Endereço IP",
		"Dispositivo / Navegador",
	})

	for rows.Next() {
		var eventType, userName, userEmail, userRole, tenantName, storeName, ip, ua string
		var createdAt time.Time
		if errScan := rows.Scan(&eventType, &userName, &userEmail, &userRole, &tenantName, &storeName, &ip, &ua, &createdAt); errScan == nil {
			eventLabel := eventType
			switch eventType {
			case "LOGIN":
				eventLabel = "Login (Sucesso)"
			case "LOGOUT":
				eventLabel = "Logout (Sessão Encerrada)"
			case "LOGIN_FAILED":
				eventLabel = "Tentativa Falha"
			}
			_ = writer.Write([]string{
				createdAt.Format("02/01/2006 15:04:05"),
				eventLabel,
				userName,
				userEmail,
				userRole,
				tenantName,
				storeName,
				ip,
				ua,
			})
		}
	}
	writer.Flush()
}
