# Walkthrough - Automated Caddy Configuration & Player Updates

We have successfully updated the player codebase, resolved port conflicts on the host, rebuilt the docker image, and verified that both diagnostics and skip reasons work correctly.

## Changes Made

### 1. Remote Player Code Synced & Fixed
- Synced the updated player source files (`db.go`, `scheduler.go`, `web_dashboard.go`) from the Windows host to the remote server directory `/home/neoroute/imc-audio-local/`.
- Fixed a bug in [scheduler.go](file:///P:/home/neoroute/imc-audio-local/scheduler.go#L537-L547):
  - Previously, the currently playing track (`na_ponta`) was always marked as `executada` when the next track started.
  - We updated this query to use a `CASE WHEN` clause: if the track has a `skip_reason` or `skipped_by_catchup = 1`, its status is correctly set to `pulada`. Otherwise, it becomes `executada`. This ensures skipped tracks display with the correct status badge.

### 2. Port conflict resolutions on Host
- **Central Go Server**: Removed host port mapping `-p localPort:localPort` from [docker_orchestrator.go](file:///P:/home/neoroute/imc-server/internal/utils/docker_orchestrator.go#L58-L63) for dynamically started store containers.
- **Docker Compose**: Removed host port mappings `- "5001:5001"` and `- "5002:5002"` from [docker-compose.yml](file:///P:/home/neoroute/imc-audio-local/docker-compose.yml).
- *Rationale*: Since the Caddy container (`imc-caddy`) already binds host ports `5001-5003` to act as a reverse proxy, mapping the same ports directly to the player containers on the host caused a `port is already allocated` conflict. Now, Caddy maps the host ports and routes the traffic internally to the player containers over the shared Docker network `imc-audio-local_imc-network` without host port conflicts.

### 3. Rebuild & Re-deployment
- Rebuilt the player image on the remote host:
  ```bash
  docker build -t imc-audio-local-radio_loja_01:latest .
  ```
- Recreated and started all compose-managed player containers:
  ```bash
  docker compose down && docker compose up -d
  ```
- Recreated all dynamic server player containers (e.g. `Loja 03` on port 5003) via the orchestrator.
- Checked container ownership on the server and resolved `/home/neoroute/imc-audio-local/stores` permissions.

---

## Verification Results

### 1. Diagnostics Pages
We verified that both diagnostics endpoints are successfully accessible and proxied by Caddy over HTTPS:
- **Loja 01**:
  ```bash
  curl.exe -k -I --resolve imc-audioteste.duckdns.org:5001:192.168.15.250 https://imc-audioteste.duckdns.org:5001/diagnostics
  HTTP/1.1 200 OK
  Via: 1.1 Caddy
  ```
- **Loja 03**:
  ```bash
  curl.exe -k -I --resolve imc-audioteste.duckdns.org:5003:192.168.15.250 https://imc-audioteste.duckdns.org:5003/diagnostics
  HTTP/1.1 200 OK
  Via: 1.1 Caddy
  ```

### 2. Skip Reasons & Queue
- Skipped tracks now correctly save their `skip_reason` in the local cache database.
- The diagnostics dashboard correctly checks the database values and displays `⏩ Pulada (Repetição)` or `⏩ Pulada (Tempo)`.
