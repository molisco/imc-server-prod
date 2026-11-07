# Walkthrough - Bug Fix & Enhancements (Updated)

I have completed the requested bug fixes and verified that all dashboard elements are functioning as expected:

1. **Player Timezone Sync (`TZ`)**:
   - Updated the Docker orchestrator in the backend (`docker_orchestrator.go`) to automatically fetch each store's local timezone (e.g., `America/Sao_Paulo`) and inject it as the `TZ` environment variable in the player containers.
   - Recreated the containers with the local timezone, ensuring that the container clock matches the host's timezone.
   - Proof of Play (PoP) execution events are now recorded with the correct local time instead of a 3-hour UTC offset.

2. **Diagnostics Dashboard Early-Return Fix**:
   - Fixed a Javascript logic error in the dashboard's `updateDiagnostics()` function (in [web_dashboard.go](file:///C:/imc-audio-local/web_dashboard.go)).
   - Previously, if the store had no ads/commercials scheduled for the current day (`data.comm_slots` empty or null), the Javascript would immediately abort early via a `return;` statement, preventing the weekly grid, timeline, logs, and play history from ever loading (leaving the table stuck on "Carregando histórico...").
   - Wrapped the commercials loop in a standard element-existence check so the function continues executing all the way to the end. The Execution Real history log now loads and updates instantly.

3. **Style Name Translation in Block Structure**:
   - Resolved the issue where the "Estrutura dos Blocos" table displayed raw musical style UUIDs (hashes) instead of their human-readable names.
   - Added a `getStyleName` helper function in the player dashboard back-end to retrieve names from the local SQLite `media_styles` table.
   - Joined style UUIDs into their actual names (e.g., "Bossa Nova TOP", "MPB Essenciais") when assembling the `block_structures` response.

4. **Resource Volume Mounts for Alerts and Warnings**:
   - Added missing mounts for `/app/chimes`, `/app/voice_packs`, and `/app/pre_formatted_warnings` inside player containers in `docker_orchestrator.go`.
   - Corrected directories permissions on the host server (`chmod -R a+rX`) so files can be read properly by the container user.
   - Dynamic plate announcements and pre-formatted alerts now compile and play correctly without error.

---

## 1. Audio Grid Creation Bug Fix

### What Was the Problem?
When clicking the "Nova Grade" (New Grid) button, the dashboard failed to open the creation form layout because of a client-side JavaScript exception. 

- The JavaScript functions (`showCreateForm()`, `toggleDayEnabled()`, and `editGrid()`) did not contain safety checks when querying DOM elements.
- If any day's checkbox, fields div, or time input was missing or failed to query (or was evaluated during `recalculateAll()`'s initial metrics recalculation), the script would immediately throw a `TypeError` (e.g., *Cannot set properties of null*) and halt all execution, preventing the view toggles (`form-view`/`list-view`) from running.

### How Was it Fixed?
We refactored the JavaScript functions in both [grids_audio.html](file:///P:/home/neoroute/imc-server/web/templates/grids_audio.html) and [grids_video.html](file:///P:/home/neoroute/imc-server/web/templates/grids_video.html) to be completely null-safe and crash-proof:

1. **Null Checks**: Added validation checks (`if (el)`) before performing value assignments, class changes, or attribute modifications on queried DOM elements.
2. **Robust `toggleDayEnabled`**: Handled potential exceptions and checked for the existence of day checkbox/fields divs before querying child inputs.
3. **Robust `recalculateAll`**: Wrapped individual day metric updates in their own `try-catch` blocks. If any day has a rendering anomaly, it logs the warning but no longer crashes the other days or the page execution.
4. **Try-Catch Wrappers**: Protected the main entry points (`showCreateForm`, `editGrid`, `cancelEdit`) with try-catch blocks to guarantee UI rendering and show friendly alerts in case of unforeseen errors.

---

## 2. Music Filename Refactoring

The utility tool `tools/rename_music.py` recursive scans and strips number prefixes and sanitizes accents/punctuation.

### How to Run:
Navigate to the project root directory and run the command:

#### Preview Changes (Dry Run):
```powershell
python tools/rename_music.py --dir "P:\home\neoroute\imc-server\midias\musicas" --remove-accents --sanitize
```

#### Apply Changes (Rename Files):
```powershell
python tools/rename_music.py --dir "P:\home\neoroute\imc-server\midias\musicas" --remove-accents --sanitize --run
```

---

## 3. Server-Side Player Containerization (Docker)

To run the local player code (`imc-audio-local`) headlessly inside Linux containers on the server:

1. **Server Mode**: A new config field `ServerMode` skips local downloads, reads files directly from `/app/midias`, and bypasses the TUI Console Dashboard to run silently in the background.
2. **Icecast Integration**: Encodings are routed dynamically to Icecast mounts instead of local sound cards (`output.ao`).
3. **Cross-Platform Watchdog**: Liquidsoap process killing uses native `pkill` in Linux environments.
4. **Build Context**: A multi-stage [Dockerfile](file:///C:/imc-audio-local/Dockerfile) compiles the Go binary using `golang:alpine` and packages it into `savonet/liquidsoap` with `ENTRYPOINT` overrides.

---

## 4. Server-Side Radio Management and Monitoring

A new column `radio_type` in the `stores` table identifies whether a store runs a local playout terminal or a server-side streaming container.

1. **Auto-Patching Schema**: Added automatic schema migration inside `postgres.go` to patch database columns on start.
2. **Status Dashboard**: The new route `/dashboard/radios/status` decodes `status-json.xsl` from the Icecast server to display active streaming metadata (current track, listener counts, mount points) directly on the portal.

---

## 5. Automated Docker Playout Orchestration

placing the playout container control entirely in the Go backend handler logic.

### How it Works:
- **Port Allocation**: The system queries the database for `MAX(harbor_port)` and assigns `MAX(harbor_port) + 1` (starting at `7001`).
- **Dynamic Container Run**:
  `docker run -d --name imc-radio-<id> -p 127.0.0.1:<harbor_port>:<harbor_port> ...`
- **Graceful Deletion**: Deleting or deactivating a store triggers a background call to:
  `docker stop imc-radio-<id> && docker rm imc-radio-<id>`

---

## 6. Dynamic Synchronization Badges & Playout History Fixes

To improve real-time player analysis and secure access:

1. **Visual Sync Mode & Buffer Badges**:
   - Added a dynamic badge (`time-sync-mode-badge`) showing **Tempo Real** or **Streaming (-Xs)**.
   - Added a dynamic buffer status badge (`buffer-status-badge`) rendering `Buffer: Xs / Ys` calculated from queue length.
2. **Quick Warnings Interface**:
   - Integrated warning trigger inputs (Vehicle Plate and Pre-Formatted announcements) directly on the dashboard tab.
3. **Playout History Status Bugfixes**:
   - Changed `MarkTrackAsPlaying` logic to only mark previous playing tracks as `executada` when they actually play, and mark previous enqueued tracks in the past as `pulada` (skipped) rather than `executada`. This prevents future enqueued tracks with higher IDs from being prematurely marked as `executada`.
   - Added rendering support for `pulada` in the frontend diagnostics play history table (`⏩ Pulada`).
4. **Caddy HTTPS reverse-proxy**:
   - Routed diagnostics ports `5001-5003` internally on the Docker network, and mapped Caddy reverse proxy blocks so ports `5001-5003` are externally accessible over HTTPS with local SSL certificates (`imc-audioteste.duckdns.org:5001/diagnostics`).
