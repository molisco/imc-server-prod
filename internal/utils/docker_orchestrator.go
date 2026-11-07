package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/google/uuid"
	"imc-server/internal/database"
	"imc-server/internal/models"
)

// GetStoreMountPoint derives a safe, consistent Icecast mount path for a store (matches handler logic)
func GetStoreMountPoint(store models.Store) string {
	if store.StoreNumber != "" {
		clean := strings.ToLower(strings.ReplaceAll(store.StoreNumber, " ", "_"))
		tenantStr := store.TenantID.String()
		tenantSuffix := "0000"
		if len(tenantStr) >= 4 {
			tenantSuffix = tenantStr[len(tenantStr)-4:]
		}
		clean = "m" + tenantSuffix + "_l" + strings.TrimPrefix(clean, "/")
		return "/" + clean
	}
	return "/loja_" + store.ID.String()[:8]
}

// StartStoreDockerContainer stops any previous container and starts a new player container
func StartStoreDockerContainer(store models.Store) error {
	if store.RadioType != "server" || store.HarborPort == nil || *store.HarborPort < 7001 {
		return nil
	}

	containerName := fmt.Sprintf("imc-radio-%s", store.ID.String())
	
	// 1. Stop and remove any prior container instance gracefully
	_ = exec.Command("docker", "stop", containerName).Run()
	_ = exec.Command("docker", "rm", containerName).Run()

	// 2. Set up paths
	cacheDir := fmt.Sprintf("/home/neoroute/imc-audio-local/stores/loja_%s/cache", store.ID.String()[:8])
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory on host: %w", err)
	}

	mountPoint := GetStoreMountPoint(store)

	localPort := 5000 + (*store.HarborPort - 7000)

	// 3. Start player container with dynamic environment configurations and expose ports automatically
	tz := store.Timezone
	if tz == "" {
		tz = "America/Sao_Paulo"
	}

	extStreamURL := ""
	if store.ExternalStreamURL != nil {
		extStreamURL = *store.ExternalStreamURL
	}

	cmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"--network", "imc-audio-local_imc-network",
		"-p", fmt.Sprintf("%d:%d", *store.HarborPort, *store.HarborPort),
		"-e", "SERVER_MODE=true",
		"-e", "SERVER_URL=http://200.141.23.44:8080",
		"-e", "STORE_TOKEN="+store.Token,
		"-e", "STORE_NAME="+store.StoreName,
		"-e", "EXTERNAL_STREAM_URL="+extStreamURL,
		"-e", fmt.Sprintf("LOCAL_PORT=%d", localPort),
		"-e", fmt.Sprintf("HARBOR_PORT=%d", *store.HarborPort),
		"-e", "ICECAST_HOST=imc-icecast",
		"-e", "ICECAST_PORT=8000",
		"-e", "ICECAST_PASSWORD=hackme",
		"-e", "ICECAST_MOUNT="+mountPoint,
		"-e", "TZ="+tz,
		"-v", "/home/neoroute/imc-server/midias:/app/midias:ro",
		"-v", fmt.Sprintf("%s:/app/cache", cacheDir),
		"-v", "/home/neoroute/imc-audio-local/chimes:/app/chimes:ro",
		"-v", "/home/neoroute/imc-audio-local/voice_packs:/app/voice_packs:ro",
		"-v", "/home/neoroute/imc-audio-local/pre_formatted_warnings:/app/pre_formatted_warnings:ro",
		"--log-opt", "max-size=10m",
		"--log-opt", "max-file=3",
		"--restart", "always",
		"imc-radio-base:latest",
		"/app/imc-audio-local",
		fmt.Sprintf("--store-name=%s", store.StoreName),
		fmt.Sprintf("--store-id=%s", store.ID.String()),
	)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %v, stderr: %s", err, errBuf.String())
	}

	if err := RebuildAndReloadCaddy(); err != nil {
		fmt.Printf("ERROR: Failed to rebuild and reload Caddy: %v\n", err)
	}

	return nil
}

// StopStoreDockerContainer gracefully stops and removes the playout container
func StopStoreDockerContainer(storeID uuid.UUID) error {
	containerName := fmt.Sprintf("imc-radio-%s", storeID.String())
	_ = exec.Command("docker", "stop", containerName).Run()
	_ = exec.Command("docker", "rm", containerName).Run()

	if err := RebuildAndReloadCaddy(); err != nil {
		fmt.Printf("ERROR: Failed to rebuild and reload Caddy: %v\n", err)
	}

	return nil
}

// RebuildAndReloadCaddy queries all active server stores, generates /home/neoroute/caddy/Caddyfile, and reloads the Caddy container
func RebuildAndReloadCaddy() error {
	ctx := context.Background()
	query := `
		SELECT id, harbor_port 
		FROM stores 
		WHERE harbor_port IS NOT NULL AND radio_type = 'server' AND is_active = true
		ORDER BY harbor_port ASC
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query stores for caddy configuration: %w", err)
	}
	defer rows.Close()

	var dynamicBlocks []string

	for rows.Next() {
		var id uuid.UUID
		var harborPort int
		if err := rows.Scan(&id, &harborPort); err != nil {
			continue
		}

		localPort := 5000 + (harborPort - 7000)
		containerHost := fmt.Sprintf("imc-radio-%s", id.String())

		// Configure block for domain and production IP address (HTTP only for diagnostic ports to avoid multiplexing conflicts)
		block := fmt.Sprintf(`# Store harbor: %d (local: %d)
http://stream.indoormedia.com.br:%d, http://app.indoormedia.com.br:%d, http://200.141.23.44:%d, http://localhost:%d {
    reverse_proxy %s:%d
}`, harborPort, localPort, localPort, localPort, localPort, localPort, containerHost, localPort)

		dynamicBlocks = append(dynamicBlocks, block)
	}

	caddyfileContent := `# ============================================================================
# IMC Production Caddyfile — indoormedia.com.br (200.141.23.44)
# ============================================================================

# 1. Domínios Oficiais com HTTPS Automático (Let's Encrypt)
indoormedia.com.br, www.indoormedia.com.br, app.indoormedia.com.br, player.indoormedia.com.br {
    @icecast {
        path /status* /admin* /server* /*.pls /*.m3u
    }
    reverse_proxy @icecast imc-icecast:8000
    reverse_proxy 200.141.23.44:8080
}

# 2. Subdomínio dedicado de Streaming Contínuo (Icecast + Proxy Go Protegido)
stream.indoormedia.com.br {
    @icecast {
        path /status* /admin* /server* /*.pls /*.m3u
    }
    reverse_proxy @icecast imc-icecast:8000
    reverse_proxy 200.141.23.44:8080
}

# 3. Fallback direto por IP Público (HTTP :80)
http://200.141.23.44, :80 {
    @icecast {
        path /status* /admin* /server* /*.pls /*.m3u
    }
    reverse_proxy @icecast imc-icecast:8000
    reverse_proxy 200.141.23.44:8080
}

`
	if len(dynamicBlocks) > 0 {
		caddyfileContent += strings.Join(dynamicBlocks, "\n\n") + "\n"
	}

	caddyfilePath := "/home/neoroute/caddy/Caddyfile"
	if err := os.WriteFile(caddyfilePath, []byte(caddyfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile to %s: %w", caddyfilePath, err)
	}

	// Reload Caddy container config
	cmd := exec.Command("docker", "exec", "imc-caddy", "caddy", "reload", "--config", "/etc/caddy/Caddyfile")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload caddy container: %v, stderr: %s", err, errBuf.String())
	}

	return nil
}
