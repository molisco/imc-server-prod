package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"imc-server/internal/database"
)

type PrelaunchLeadRequest struct {
	Name    string `json:"name"`
	Company string `json:"company"`
	Email   string `json:"email"`
	Plan    string `json:"plan"`
}

type PrelaunchLeadRow struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Company      string    `json:"company"`
	Contact      string    `json:"contact"`
	PlanInterest string    `json:"plan_interest"`
	SourceHost   string    `json:"source_host"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
}

// extractHostname strips the port if present and lowercases the host
func extractHostname(hostHeader string) string {
	h := strings.TrimSpace(strings.ToLower(hostHeader))
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	return h
}

// IsLandingHost returns true if the request came through the root commercial domain (indoormedia.com.br / www.indoormedia.com.br)
func IsLandingHost(r *http.Request) bool {
	if r.URL.Query().Get("view") == "landing" {
		return true
	}
	host := extractHostname(r.Host)
	return host == "indoormedia.com.br" || host == "www.indoormedia.com.br"
}

// IsPlayerHost returns true if the request came through player.indoormedia.com.br
func IsPlayerHost(r *http.Request) bool {
	host := extractHostname(r.Host)
	return host == "player.indoormedia.com.br"
}

// HandleLandingPage serves the commercial Landing Page (Em Construção • Lançamento em Dezembro)
func HandleLandingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	http.ServeFile(w, r, "web/landing/index.html")
}

// HandleCreatePrelaunchLead stores a VIP pre-launch lead from the Landing Page in PostgreSQL
func HandleCreatePrelaunchLead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Método não permitido"})
		return
	}

	var req PrelaunchLeadRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "JSON inválido"})
			return
		}
	} else {
		req.Name = r.FormValue("name")
		req.Company = r.FormValue("company")
		req.Email = r.FormValue("email")
		req.Plan = r.FormValue("plan")
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Company = strings.TrimSpace(req.Company)
	req.Email = strings.TrimSpace(req.Email)
	req.Plan = strings.TrimSpace(req.Plan)

	if req.Name == "" || req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Nome e contato são obrigatórios"})
		return
	}
	if req.Plan == "" {
		req.Plan = "IMC Business Pro (Redes e Varejo)"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var leadID string
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO prelaunch_leads (name, company, contact, plan_interest, source_host, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text
	`, req.Name, req.Company, req.Email, req.Plan, r.Host, r.RemoteAddr, r.UserAgent()).Scan(&leadID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Falha ao registrar interesse no banco de dados"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"lead_id": leadID,
		"message": "Cadastro registrado com sucesso!",
	})
}

// HandleListPrelaunchLeads returns all registered VIP pre-launch leads for administrators
func HandleListPrelaunchLeads(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx, `
		SELECT id::text, name, company, contact, plan_interest, COALESCE(source_host, ''), COALESCE(ip_address, ''), created_at
		FROM prelaunch_leads
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Erro ao consultar leads: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	leads := make([]PrelaunchLeadRow, 0)
	for rows.Next() {
		var row PrelaunchLeadRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Company, &row.Contact, &row.PlanInterest, &row.SourceHost, &row.IPAddress, &row.CreatedAt); err == nil {
			leads = append(leads, row)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total": len(leads),
		"leads": leads,
	})
}
