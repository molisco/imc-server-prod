package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"imc-server/internal/models"
	"imc-server/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// Instanciamos o repositório de usuários para uso interno do handler
var userRepo = repository.NewUserRepository()

// SessionDuration define a validade padrão da sessão (30 dias para estações e terminais de loja)
const SessionDuration = 30 * 24 * time.Hour

// IssueSessionCookie define ou renova o Cookie de Sessão Protegido com janela deslizante de 30 dias.
func IssueSessionCookie(w http.ResponseWriter, session models.SessionData) {
	sessionBytes, err := json.Marshal(session)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "imc_session",
		Value:    url.QueryEscape(string(sessionBytes)),
		Path:     "/",
		Expires:  time.Now().Add(SessionDuration),
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   false,
	})
}

// GetAndRenewSession extrai a sessão atual e renova automaticamente a validade da janela deslizante.
func GetAndRenewSession(w http.ResponseWriter, r *http.Request) (*models.SessionData, error) {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		return nil, err
	}
	IssueSessionCookie(w, *session)
	return session, nil
}

// PageData estrutura o transporte de variáveis para dentro do HTML.
// Preparado para Internacionalização (i18n): podemos passar dicionários de tradução aqui dentro.
type PageData struct {
	Lang  string                 // Código do idioma atual (ex: "pt", "en")
	Error string                 // Mensagem ou código de erro para exibir na tela
	Data  map[string]interface{} // Dados genéricos extras
}

// ShowLoginPage renderiza a interface visual de login do Portal Administrativo.
func ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	// Captura o idioma preferido do navegador ou query string (padrão "pt")
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "pt"
	}

	// Carrega o arquivo HTML contido no servidor Linux
	tmpl, err := template.ParseFiles("web/templates/login.html")
	if err != nil {
		http.Error(w, "Erro crítico ao carregar o template visual: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Estrutura os dados iniciais da página
	data := PageData{
		Lang:  lang,
		Error: r.URL.Query().Get("error"), // Captura códigos como 'invalid' ou 'inactive'
	}

	// Executa a renderização do HTML enviando os dados limpos
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// HandleLogin processa o envio do formulário (POST) e autentica o usuário
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método HTTP não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Coleta os inputs sanitizados do formulário HTML
	inputEmail := r.FormValue("username")
	inputPassword := r.FormValue("password")

	// Padrão de Engenharia: Criamos um contexto com timeout para a requisição inteira
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// DESACOUPLAGEM EM AÇÃO: Pedimos os dados ao Repositório, sem saber o SQL interno
	user, dbPasswordHash, err := userRepo.GetByEmail(ctx, inputEmail)
	if err != nil {
		LogAuthEvent(ctx, r, "LOGIN_FAILED", nil, inputEmail, inputEmail, "Desconhecido", nil, nil)
		println("❌ [DIAGNÓSTICO] Repositório falhou ou e-mail não existe:", err.Error())
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	// Validação de Segurança 1: Conta do usuário está ativa no Portal?
	if !user.IsActive {
		LogAuthEvent(ctx, r, "LOGIN_FAILED", &user.ID, user.Name, user.Email, getRoleName(user.PermissionLevelID), user.TenantID, user.StoreID)
		println("❌ [DIAGNÓSTICO] Conta do usuário está inativa no sistema.")
		http.Redirect(w, r, "/login?error=inactive", http.StatusSeeOther)
		return
	}

	// Validação de Segurança 2: Decodificação e comparação com Bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(inputPassword))
	if err != nil {
		LogAuthEvent(ctx, r, "LOGIN_FAILED", &user.ID, user.Name, user.Email, getRoleName(user.PermissionLevelID), user.TenantID, user.StoreID)
		println("❌ [DIAGNÓSTICO] Senha inválida. Falha no casamento do Hash.")
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	// SUCESSO COMPROVADO: Preparamos o Footprint da Sessão Corporativa (SessionData)
	session := models.SessionData{
		UserID:            user.ID,
		TenantID:          user.TenantID,
		StoreID:           user.StoreID,
		PermissionLevelID: user.PermissionLevelID,
		UserName:          user.Name,
		IsImpersonated:    false, // Todo login padrão começa sem simulação ativa
	}

	// Registra log de login com sucesso
	LogAuthEvent(ctx, r, "LOGIN", &user.ID, user.Name, user.Email, getRoleName(user.PermissionLevelID), user.TenantID, user.StoreID)

	// Injeta o Cookie de Sessão Protegido com validade de 30 dias
	IssueSessionCookie(w, session)

	redirectURL := r.FormValue("redirect")
	if redirectURL == "" {
		redirectURL = r.URL.Query().Get("redirect")
	}

	if redirectURL != "" && strings.HasPrefix(redirectURL, "/") {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// Redirecionamento inteligente:
	if user.StoreID != nil {
		http.Redirect(w, r, fmt.Sprintf("/player/%s", user.StoreID.String()), http.StatusSeeOther)
		return
	}
	if user.TenantID != nil {
		http.Redirect(w, r, "/player", http.StatusSeeOther)
		return
	}

	println("✅ [DIAGNÓSTICO] Login bem-sucedido! Redirecionando:", user.Name)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetSessionFromRequest extrai e desserializa os dados de sessão do cookie de forma segura.
func GetSessionFromRequest(r *http.Request) (*models.SessionData, error) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		return nil, err
	}

	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil, err
	}

	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// HandleLogout invalida a sessão do usuário e redireciona para a tela de login.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if session, err := GetSessionFromRequest(r); err == nil && session != nil {
		LogAuthEvent(r.Context(), r, "LOGOUT", &session.UserID, session.UserName, "", "Usuário", session.TenantID, session.StoreID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "imc_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HomePageData estrutura os dados da tela inicial padrão
type HomePageData struct {
	User  map[string]string
	Theme map[string]string
}

// RequireAdminOrProgrammer middleware bloqueia o acesso de Operadores (PermissionLevelID == 3) a rotas administrativas.
func RequireAdminOrProgrammer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := GetAndRenewSession(w, r)
		if err != nil {
			http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.RequestURI), http.StatusSeeOther)
			return
		}

		// Se for Operador (PermissionLevelID = 3), restringe o acesso ao Web Player
		if session.PermissionLevelID == 3 {
			if session.StoreID != nil && session.StoreID.String() != "" {
				http.Redirect(w, r, fmt.Sprintf("/player/%s", session.StoreID.String()), http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/player", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HandleHomePage renderiza a tela inicial padrão (limpa, sem menu pré-selecionado)
func HandleHomePage(w http.ResponseWriter, r *http.Request) {
	if IsLandingHost(r) {
		HandleLandingPage(w, r)
		return
	}
	if IsPlayerHost(r) {
		HandleCustomerPlayerRoot(w, r)
		return
	}

	session, err := GetAndRenewSession(w, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Operador (PermissionLevelID = 3) nunca acessa a home administrativa: redireciona para o Player
	if session.PermissionLevelID == 3 {
		if session.StoreID != nil && session.StoreID.String() != "" {
			http.Redirect(w, r, fmt.Sprintf("/player/%s", session.StoreID.String()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/player", http.StatusSeeOther)
		return
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	roleName := "Administrador"
	switch session.PermissionLevelID {
	case 1:
		roleName = "Administrador"
	case 3:
		roleName = "Operador"
	case 5:
		roleName = "Programador"
	case 7:
		roleName = "Gestor de Acesso"
	}

	data := HomePageData{
		User: map[string]string{
			"Name": session.UserName,
			"Role": roleName,
		},
		Theme: theme,
	}

	tmpl, err := template.ParseFiles("web/templates/home.html")
	if err != nil {
		http.Error(w, "Falha crítica ao renderizar Página Inicial: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func getRoleName(permLevelID int) string {
	switch permLevelID {
	case 1:
		return "SuperAdmin"
	case 2:
		return "Gerente Matriz"
	case 3:
		return "Operador Loja"
	case 4:
		return "Programador Musical"
	case 5:
		return "Consulta / Acesso"
	default:
		return "Usuário"
	}
}
