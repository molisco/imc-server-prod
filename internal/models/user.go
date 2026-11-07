package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a portal backoffice operator, admin, or store client.
type User struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	TenantID          *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	StoreID           *uuid.UUID `json:"store_id,omitempty" db:"store_id"`
	PermissionLevelID int        `json:"permission_level_id" db:"permission_level_id"`
	Name              string     `json:"name" db:"name"`
	Login             string     `json:"login" db:"login"`
	PasswordHash      string     `json:"-" db:"password_hash"`
	Email             string     `json:"email" db:"email"`
	Phone             string     `json:"phone,omitempty" db:"phone"`
	JobFunction       string     `json:"job_function,omitempty" db:"job_function"`
	RegistrationID    string     `json:"registration_id,omitempty" db:"registration_id"`
	SupervisorName    string     `json:"supervisor_name,omitempty" db:"supervisor_name"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// PermissionLevel represents a user authorization role level.
type PermissionLevel struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
}

// UserWithDetails represents a user combined with related Tenant and Permission Level descriptions for display.
type UserWithDetails struct {
	User
	TenantName          string `json:"tenant_name"`
	StoreName           string `json:"store_name"`
	PermissionLevelName string `json:"permission_level_name"`
}

// SessionData represents the secure cookie structural footprint
type SessionData struct {
	UserID            uuid.UUID  `json:"user_id"`
	TenantID          *uuid.UUID `json:"tenant_id"`
	StoreID           *uuid.UUID `json:"store_id"`
	PermissionLevelID int        `json:"permission_level_id"`
	UserName          string     `json:"user_name"`

	// Impersonation Controls
	IsImpersonated     bool       `json:"is_impersonated"`
	ImpersonatorUserID *uuid.UUID `json:"impersonator_user_id"`
}
