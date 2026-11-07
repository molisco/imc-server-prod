package repository

import (
	"context"
	"fmt"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository encapsula todas as operações de banco de dados para a entidade Users.
type UserRepository struct{}

// NewUserRepository cria uma nova instância do repositório de usuários.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// GetByEmail busca um usuário ativo no banco através do e-mail corporativo ou login.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, string, error) {
	return r.GetByLoginOrEmail(ctx, email)
}

// GetByLoginOrEmail busca um usuário ativo pelo e-mail ou pelo nome de login.
func (r *UserRepository) GetByLoginOrEmail(ctx context.Context, identifier string) (*models.User, string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user models.User
	var dbPasswordHash string

	query := `
		SELECT id, tenant_id, store_id, permission_level_id, name, login, email, 
		       COALESCE(phone, ''), COALESCE(job_function, ''), COALESCE(registration_id, ''), COALESCE(supervisor_name, ''),
		       password_hash, is_active, created_at
		FROM users 
		WHERE LOWER(email) = LOWER($1) OR LOWER(login) = LOWER($1)`

	err := database.Pool.QueryRow(queryCtx, query, identifier).Scan(
		&user.ID,
		&user.TenantID,
		&user.StoreID,
		&user.PermissionLevelID,
		&user.Name,
		&user.Login,
		&user.Email,
		&user.Phone,
		&user.JobFunction,
		&user.RegistrationID,
		&user.SupervisorName,
		&dbPasswordHash,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, "", err
	}

	return &user, dbPasswordHash, nil
}

// GetByID busca um usuário por seu UUID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user models.User
	query := `
		SELECT id, tenant_id, store_id, permission_level_id, name, login, email, 
		       COALESCE(phone, ''), COALESCE(job_function, ''), COALESCE(registration_id, ''), COALESCE(supervisor_name, ''),
		       is_active, created_at
		FROM users 
		WHERE id = $1`

	err := database.Pool.QueryRow(queryCtx, query, id).Scan(
		&user.ID,
		&user.TenantID,
		&user.StoreID,
		&user.PermissionLevelID,
		&user.Name,
		&user.Login,
		&user.Email,
		&user.Phone,
		&user.JobFunction,
		&user.RegistrationID,
		&user.SupervisorName,
		&user.IsActive,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAll lista todos os usuários com detalhes de tenant, store e permissão, respeitando isolamento multi-tenant.
func (r *UserRepository) GetAll(ctx context.Context, tenantID *uuid.UUID) ([]models.UserWithDetails, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var users []models.UserWithDetails

	query := `
		SELECT u.id, u.tenant_id, u.store_id, u.permission_level_id, u.name, u.login, u.email,
		       COALESCE(u.phone, ''), COALESCE(u.job_function, ''), COALESCE(u.registration_id, ''), COALESCE(u.supervisor_name, ''),
		       u.is_active, u.created_at,
		       COALESCE(t.trade_name, 'Sistema Central (Geral)') AS tenant_name,
		       COALESCE(s.store_name, 'Todas as Lojas') AS store_name,
		       COALESCE(p.name, 'Indefinido') AS permission_level_name
		FROM users u
		LEFT JOIN tenants t ON u.tenant_id = t.id
		LEFT JOIN stores s ON u.store_id = s.id
		LEFT JOIN permission_levels p ON u.permission_level_id = p.id
		WHERE ($1::uuid IS NULL OR u.tenant_id = $1::uuid)
		ORDER BY u.name ASC`

	rows, err := database.Pool.Query(queryCtx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar usuários: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u models.UserWithDetails
		err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.StoreID,
			&u.PermissionLevelID,
			&u.Name,
			&u.Login,
			&u.Email,
			&u.Phone,
			&u.JobFunction,
			&u.RegistrationID,
			&u.SupervisorName,
			&u.IsActive,
			&u.CreatedAt,
			&u.TenantName,
			&u.StoreName,
			&u.PermissionLevelName,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao mapear usuário: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

// GetPermissionLevels retorna todos os níveis de permissão cadastrados no banco.
func (r *UserRepository) GetPermissionLevels(ctx context.Context) ([]models.PermissionLevel, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var levels []models.PermissionLevel
	query := `SELECT id, name, COALESCE(description, '') FROM permission_levels ORDER BY id ASC`

	rows, err := database.Pool.Query(queryCtx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar níveis de permissão: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lvl models.PermissionLevel
		if err := rows.Scan(&lvl.ID, &lvl.Name, &lvl.Description); err != nil {
			return nil, err
		}
		levels = append(levels, lvl)
	}

	return levels, nil
}

// Create insere um novo usuário com senha criptografada via Bcrypt.
func (r *UserRepository) Create(ctx context.Context, user *models.User, plainPassword string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	query := `
		INSERT INTO users (
			tenant_id, store_id, permission_level_id, name, login, password_hash, email, 
			phone, job_function, registration_id, supervisor_name, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`

	return database.Pool.QueryRow(
		queryCtx, query,
		user.TenantID,
		user.StoreID,
		user.PermissionLevelID,
		user.Name,
		user.Login,
		string(hashBytes),
		user.Email,
		user.Phone,
		user.JobFunction,
		user.RegistrationID,
		user.SupervisorName,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt)
}

// Update atualiza as informações de um usuário existente, alterando a senha somente se fornecida.
func (r *UserRepository) Update(ctx context.Context, user *models.User, newPassword *string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if newPassword != nil && *newPassword != "" {
		hashBytes, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("erro ao gerar hash da nova senha: %w", err)
		}

		query := `
			UPDATE users 
			SET tenant_id = $1, store_id = $2, permission_level_id = $3, name = $4, login = $5, email = $6,
			    phone = $7, job_function = $8, registration_id = $9, supervisor_name = $10,
			    is_active = $11, password_hash = $12
			WHERE id = $13`

		_, err = database.Pool.Exec(
			queryCtx, query,
			user.TenantID,
			user.StoreID,
			user.PermissionLevelID,
			user.Name,
			user.Login,
			user.Email,
			user.Phone,
			user.JobFunction,
			user.RegistrationID,
			user.SupervisorName,
			user.IsActive,
			string(hashBytes),
			user.ID,
		)
		return err
	}

	query := `
		UPDATE users 
		SET tenant_id = $1, store_id = $2, permission_level_id = $3, name = $4, login = $5, email = $6,
		    phone = $7, job_function = $8, registration_id = $9, supervisor_name = $10,
		    is_active = $11
		WHERE id = $12`

	_, err := database.Pool.Exec(
		queryCtx, query,
		user.TenantID,
		user.StoreID,
		user.PermissionLevelID,
		user.Name,
		user.Login,
		user.Email,
		user.Phone,
		user.JobFunction,
		user.RegistrationID,
		user.SupervisorName,
		user.IsActive,
		user.ID,
	)
	return err
}

// Delete remove um usuário por ID.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `DELETE FROM users WHERE id = $1`
	_, err := database.Pool.Exec(queryCtx, query, id)
	return err
}
