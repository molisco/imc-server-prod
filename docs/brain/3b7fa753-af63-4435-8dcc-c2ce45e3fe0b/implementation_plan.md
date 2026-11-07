# Implementation Plan - Automated Caddy Configuration & Diagnostics Routing

This plan addresses the requirement to make diagnostic ports (e.g. `5003`) automatically active and routed when a store is created or updated in the Go Central Server (`imc-server`).

---

## Technical Details & Architecture

1. **Docker Network & Host Port Mappings**:
   - The Caddy container (`imc-caddy`) runs on the `imc-audio-local_imc-network` Docker network. It binds host ports `5001-5003`.
   - Radio containers also run on the same network. Because host ports `5001-5003` are bound by Caddy, the radio containers *cannot* map these ports on the host. Instead, they expose them internally, and Caddy proxies them inside the docker network.
   - Caddyfile needs a dedicated block for each active server store to route the traffic from the host to the correct container.

2. **Automating configuration updates**:
   - We will write a Go function `RebuildAndReloadCaddy()` inside the Go Central Server utility `internal/utils/docker_orchestrator.go`.
   - This function will:
     1. Query all active `server` stores with mapped `harbor_port` from the PostgreSQL database.
     2. Construct the `Caddyfile` configuration dynamically, mapping each local diagnostics port (computed as `5000 + (harbor_port - 7000)`) using wildcard `:port` site blocks (e.g., `:5003`). This ensures both domain-based (`https://imc-audioteste.duckdns.org:5003`) and local host IP-based (`https://192.168.15.250:5003`) access work seamlessly.
     3. Overwrite the host Caddyfile at `/home/neoroute/caddy/Caddyfile`.
     4. Execute `docker exec imc-caddy caddy reload --config /etc/caddy/Caddyfile` to hot-reload Caddy without dropping connections.
   - This function will be triggered inside:
     - `StartStoreDockerContainer()`: after starting a container.
     - `StopStoreDockerContainer()`: after stopping/removing a container.

---

## Proposed Changes

### [Component Name] Go Central Server Playout Orchestration

#### [MODIFY] [docker_orchestrator.go](file:///P:/home/neoroute/imc-server/internal/utils/docker_orchestrator.go)
- Add context and database imports.
- Implement the `RebuildAndReloadCaddy()` function.
- Invoke `RebuildAndReloadCaddy()` at the end of `StartStoreDockerContainer()` and `StopStoreDockerContainer()`.

---

## Verification Plan

### Automated Tests
- Run `go build ./...` inside `P:\home\neoroute\imc-server` to verify no Go syntax or dependency issues.

### Manual Verification
- We will rebuild the `imc-server` binary and restart the service using `systemctl --user restart imc-server`.
- We will trigger a reload manually or edit/re-save a store from the backoffice to confirm:
  1. The Caddyfile at `/home/neoroute/caddy/Caddyfile` is updated automatically.
  2. Caddy hot-reloads successfully.
  3. The diagnostics page of the store (e.g. `https://192.168.15.250:5003/diagnostics`) is instantly accessible from inside the network.
