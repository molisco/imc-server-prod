package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

const defaultStreamSecret = "imc-streaming-secure-token-secret-2026-key"

// MaxConcurrentStreamsPerStore limits simultaneous streams per store to prevent link sharing
const MaxConcurrentStreamsPerStore = 2

// activeStoreStreams keeps an in-memory counter of active audio listeners per store
var activeStoreStreams sync.Map

func getStreamSecret() []byte {
	sec := os.Getenv("STREAM_SECRET")
	if sec == "" {
		sec = defaultStreamSecret
	}
	return []byte(sec)
}

// GenerateStreamToken generates an HMAC-SHA256 signature for a store ID with expiration timestamp
func GenerateStreamToken(storeID uuid.UUID, ttl time.Duration) (string, int64) {
	exp := time.Now().Add(ttl).Unix()
	msg := fmt.Sprintf("%s:%d", storeID.String(), exp)
	h := hmac.New(sha256.New, getStreamSecret())
	h.Write([]byte(msg))
	token := hex.EncodeToString(h.Sum(nil))
	return token, exp
}

// ValidateStreamToken verifies whether a token is valid and unexpired for the given storeID
func ValidateStreamToken(storeID uuid.UUID, token string, exp int64) bool {
	if token == "" || exp <= 0 {
		return false
	}
	if time.Now().Unix() > exp {
		return false // Token expired
	}
	msg := fmt.Sprintf("%s:%d", storeID.String(), exp)
	h := hmac.New(sha256.New, getStreamSecret())
	h.Write([]byte(msg))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(token), []byte(expected))
}

// CheckStreamAccess verifies if a stream request is authorized via session or signed token
func CheckStreamAccess(r *http.Request, store *models.Store) error {
	// 1. Check for valid authenticated user session with access to this store
	session, errSession := GetSessionFromRequest(r)
	if errSession == nil && session != nil {
		if errAccess := checkStoreAccess(session, store); errAccess == nil {
			return nil // Authorized via legitimate logged-in web session
		}
	}

	// 2. Check for signed HMAC token in query params
	token := r.URL.Query().Get("token")
	expStr := r.URL.Query().Get("exp")
	if token != "" && expStr != "" {
		exp, err := strconv.ParseInt(expStr, 10, 64)
		if err == nil {
			if ValidateStreamToken(store.ID, token, exp) {
				return nil // Authorized via valid HMAC token
			}
			if time.Now().Unix() > exp {
				return fmt.Errorf("token de streaming expirado")
			}
			return fmt.Errorf("token de streaming inválido")
		}
	}

	return fmt.Errorf("autenticação necessária: login ou token válido obrigatório")
}

// AcquireStreamSlot attempts to increment the active stream count for a store, enforcing MaxConcurrentStreamsPerStore
func AcquireStreamSlot(storeID uuid.UUID) (func(), error) {
	key := storeID.String()
	val, _ := activeStoreStreams.LoadOrStore(key, new(int32))
	counter := val.(*int32)

	current := atomic.AddInt32(counter, 1)
	if current > MaxConcurrentStreamsPerStore {
		atomic.AddInt32(counter, -1)
		return nil, fmt.Errorf("limite de conexões de áudio simultâneas (%d) atingido para esta loja", MaxConcurrentStreamsPerStore)
	}

	release := func() {
		atomic.AddInt32(counter, -1)
	}
	return release, nil
}

// FindStoreByMount attempts to resolve a store from an Icecast mountpoint (e.g. /m0001_l1, /loja_ce275c7f)
func FindStoreByMount(ctx context.Context, mount string) (*models.Store, *models.Tenant, error) {
	cleanMount := strings.TrimSpace(mount)
	if cleanMount == "" {
		return nil, nil, fmt.Errorf("mount vazio")
	}

	// If mount starts with /loja_
	if strings.HasPrefix(cleanMount, "/loja_") {
		prefix := strings.TrimPrefix(cleanMount, "/loja_")
		var storeID uuid.UUID
		err := database.Pool.QueryRow(ctx, `SELECT id FROM stores WHERE id::text LIKE $1 || '%' LIMIT 1`, prefix).Scan(&storeID)
		if err == nil {
			return findStoreAndTenant(ctx, storeID.String())
		}
		return findStoreAndTenant(ctx, prefix)
	}

	// If mount starts with /m (e.g. /mcf4f_l001 or /m0001_l1)
	if strings.HasPrefix(cleanMount, "/m") {
		parts := strings.Split(cleanMount, "_l")
		if len(parts) == 2 {
			tenantSuffix := strings.TrimPrefix(parts[0], "/m")
			cleanStoreNum := parts[1]
			var storeID uuid.UUID
			err := database.Pool.QueryRow(ctx, `
				SELECT id FROM stores 
				WHERE LOWER(REPLACE(store_number, ' ', '_')) = LOWER($1) 
				  AND tenant_id::text LIKE '%' || $2 
				LIMIT 1
			`, cleanStoreNum, tenantSuffix).Scan(&storeID)
			if err == nil {
				return findStoreAndTenant(ctx, storeID.String())
			}
			// Fallback: search just by store_number
			return findStoreAndTenant(ctx, cleanStoreNum)
		}
	}

	return nil, nil, fmt.Errorf("loja não identificada pelo ponto de montagem %s", cleanMount)
}
