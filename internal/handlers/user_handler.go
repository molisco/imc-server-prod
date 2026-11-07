package handlers

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"imc-server/internal/models"

	"github.com/google/uuid"
)

// UserPageData representa o payload de dados enviado ao template HTML de usuários
type UserPageData struct {
	Title            string
	LoggedUser       string
	ErrorMessage     string
	SuccessMessage   string
	Form             map[string]string
	Users            []models.UserWithDetails
	Tenants          []models.Tenant
	Stores           []models.Store
	PermissionLevels []models.PermissionLevel
	User             map[string]string
	Theme            map[string]string
	CurrentUserID    string
}

// HandleUsersPage renderiza a tela de gerenciamento de usuários
func HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 1. Busca lista de Clientes (Tenants) ativos
	tenantsList, err := tenantRepo.GetAll(ctx)
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

	// 2. Busca lista de Lojas (Stores) ativas
	storesList, err := storeRepo.GetAll(ctx, session.TenantID)
	if err != nil {
		storesList = []models.Store{}
	}

	// 3. Busca lista de Níveis de Permissão
	permLevels, err := userRepo.GetPermissionLevels(ctx)
	if err != nil {
		permLevels = []models.PermissionLevel{}
	}

	// 4. Busca lista de Usuários cadastrados
	usersList, err := userRepo.GetAll(ctx, session.TenantID)
	if err != nil {
		usersList = []models.UserWithDetails{}
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

	data := UserPageData{
		Title:            "Gerenciamento de Usuários",
		LoggedUser:       session.UserName,
		CurrentUserID:    session.UserID.String(),
		Users:            usersList,
		Tenants:          tenantsList,
		Stores:           storesList,
		PermissionLevels: permLevels,
		Form:             make(map[string]string),
		User: map[string]string{
			"Name":  session.UserName,
			"Email": session.UserName,
			"Role":  roleName,
		},
		Theme:          theme,
		ErrorMessage:   r.URL.Query().Get("error"),
		SuccessMessage: r.URL.Query().Get("success"),
	}

	tmpl, err := template.ParseFiles("web/templates/users.html")
	if err != nil {
		http.Error(w, "Erro ao carregar o template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// HandleCreateUser processa a criação de um novo usuário
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSessionFromRequest(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if session.PermissionLevelID != 1 && session.PermissionLevelID != 7 {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	login := strings.TrimSpace(r.FormValue("login"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	jobFunction := strings.TrimSpace(r.FormValue("job_function"))
	registrationID := strings.TrimSpace(r.FormValue("registration_id"))
	supervisorName := strings.TrimSpace(r.FormValue("supervisor_name"))
	permLevelStr := r.FormValue("permission_level_id")
	tenantIDStr := r.FormValue("tenant_id")
	storeIDStr := r.FormValue("store_id")
	isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "1" || r.FormValue("is_active") == "on"

	if name == "" || login == "" || email == "" || password == "" || permLevelStr == "" {
		http.Redirect(w, r, "/users?error="+urlQuery("Preencha todos os campos obrigatórios (Nome, Login, E-mail, Senha e Perfil)."), http.StatusSeeOther)
		return
	}

	permLevel, err := strconv.Atoi(permLevelStr)
	if err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Nível de permissão inválido."), http.StatusSeeOther)
		return
	}

	var tenantUUID *uuid.UUID
	if session.TenantID != nil {
		tenantUUID = session.TenantID
	} else if tenantIDStr != "" {
		parsed, err := uuid.Parse(tenantIDStr)
		if err == nil {
			tenantUUID = &parsed
		}
	}

	var storeUUID *uuid.UUID
	if storeIDStr != "" {
		parsed, err := uuid.Parse(storeIDStr)
		if err == nil {
			storeUUID = &parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Valida se login ou email já existem
	existingUser, _, _ := userRepo.GetByLoginOrEmail(ctx, login)
	if existingUser != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Já existe um usuário cadastrado com este Login."), http.StatusSeeOther)
		return
	}
	existingUser, _, _ = userRepo.GetByLoginOrEmail(ctx, email)
	if existingUser != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Já existe um usuário cadastrado com este E-mail."), http.StatusSeeOther)
		return
	}

	newUser := models.User{
		TenantID:          tenantUUID,
		StoreID:           storeUUID,
		PermissionLevelID: permLevel,
		Name:              name,
		Login:             login,
		Email:             email,
		Phone:             phone,
		JobFunction:       jobFunction,
		RegistrationID:    registrationID,
		SupervisorName:    supervisorName,
		IsActive:          isActive,
	}

	if err := userRepo.Create(ctx, &newUser, password); err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Erro ao cadastrar usuário: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/users?success="+urlQuery("Usuário cadastrado com sucesso!"), http.StatusSeeOther)
}

// HandleUpdateUser processa a atualização dos dados de um usuário
func HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSessionFromRequest(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if session.PermissionLevelID != 1 && session.PermissionLevelID != 7 {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	userIDStr := r.FormValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("ID de usuário inválido."), http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	login := strings.TrimSpace(r.FormValue("login"))
	email := strings.TrimSpace(r.FormValue("email"))
	newPassword := strings.TrimSpace(r.FormValue("password"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	jobFunction := strings.TrimSpace(r.FormValue("job_function"))
	registrationID := strings.TrimSpace(r.FormValue("registration_id"))
	supervisorName := strings.TrimSpace(r.FormValue("supervisor_name"))
	permLevelStr := r.FormValue("permission_level_id")
	tenantIDStr := r.FormValue("tenant_id")
	storeIDStr := r.FormValue("store_id")
	isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "1" || r.FormValue("is_active") == "on"

	if name == "" || login == "" || email == "" || permLevelStr == "" {
		http.Redirect(w, r, "/users?error="+urlQuery("Nome, Login, E-mail e Perfil são obrigatórios."), http.StatusSeeOther)
		return
	}

	permLevel, err := strconv.Atoi(permLevelStr)
	if err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Nível de permissão inválido."), http.StatusSeeOther)
		return
	}

	var tenantUUID *uuid.UUID
	if session.TenantID != nil {
		tenantUUID = session.TenantID
	} else if tenantIDStr != "" {
		parsed, err := uuid.Parse(tenantIDStr)
		if err == nil {
			tenantUUID = &parsed
		}
	}

	var storeUUID *uuid.UUID
	if storeIDStr != "" {
		parsed, err := uuid.Parse(storeIDStr)
		if err == nil {
			storeUUID = &parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user := models.User{
		ID:                userID,
		TenantID:          tenantUUID,
		StoreID:           storeUUID,
		PermissionLevelID: permLevel,
		Name:              name,
		Login:             login,
		Email:             email,
		Phone:             phone,
		JobFunction:       jobFunction,
		RegistrationID:    registrationID,
		SupervisorName:    supervisorName,
		IsActive:          isActive,
	}

	var passPtr *string
	if newPassword != "" {
		passPtr = &newPassword
	}

	if err := userRepo.Update(ctx, &user, passPtr); err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Erro ao atualizar usuário: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/users?success="+urlQuery("Usuário atualizado com sucesso!"), http.StatusSeeOther)
}

// HandleDeleteUser remove um usuário do sistema
func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if session.PermissionLevelID != 1 && session.PermissionLevelID != 7 {
		http.Error(w, "Acesso negado", http.StatusForbidden)
		return
	}

	userIDStr := r.URL.Query().Get("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("ID inválido."), http.StatusSeeOther)
		return
	}

	if userID == session.UserID {
		http.Redirect(w, r, "/users?error="+urlQuery("Você não pode excluir o seu próprio usuário."), http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := userRepo.Delete(ctx, userID); err != nil {
		http.Redirect(w, r, "/users?error="+urlQuery("Erro ao excluir usuário: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/users?success="+urlQuery("Usuário excluído com sucesso!"), http.StatusSeeOther)
}

func urlQuery(s string) string {
	return url.QueryEscape(s)
}
