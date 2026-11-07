# Custom Rules and Guidelines - imc-server

This file contains coding guidelines, rules, and architectural definitions established for the `imc-server` workspace.

## 1. Web UI & JavaScript Guidelines

* **Null-Safety / Crash-Proofing**: Always verify that elements exist (`if (el)`) before performing operations such as `.value = ...`, `.innerText = ...`, `.classList.add(...)`, etc. client-side.
* **Metric Calculation Protection**: Wrap metrics and recalculation loops (such as those in day tabs) in local `try-catch` blocks. A rendering or calculation anomaly on one day's tab must never prevent the rendering of other tabs or block the view-toggling functions (`showCreateForm()`, `cancelEdit()`, `editGrid()`).
* **Hoisted Functions**: Keep core DOM manipulation functions clean and well-scoped. Avoid raw, non-safe queries in loop-initialization routines.
* **Continuous Audio Lifecycle & Watchdog**: The Web Player must maintain a persistent `userIntentPlay` state. Browsers' transient `pause` or `ended` events triggered during track transitions or `src` updates must never terminate playback. A 3-second heartbeat watchdog must continuously verify playback progression and trigger seamless reconnections upon buffer stalls.

## 2. Music Filename Sanitization Guidelines

* **Prefix Stripping**: Utility scripts modifying music files must strip leading track numbers and common delimiters matching the pattern:
  `r"^\d+(?:\s*[-_.]\s*|\s+)"`
* **Accents & Sanitization**:
  * Accents/diacritics should be stripped optionally (e.g. `--remove-accents`) to ensure compatibility with all OS filesystems.
  * Punctuation and special characters should be sanitized optionally (e.g. `--sanitize`).
  * Converting spaces to underscores (e.g. `--clean-names`) should be supported.
* **Collision / Duplicate Resolution**: When a rename action causes a collision in the same directory, resolve it by appending `[REPETIDO01]`, `[REPETIDO02]`, etc. to the filename, performing a case-insensitive check to avoid overwrites.

## 3. Playout & Liquidsoap Architecture Rules

* **Liquidsoap Instances**: Run small, isolated Liquidsoap instances (one per player/client) using local SQLite databases for schedules to ensure high resilience and independence.
* **Specialized 2x3 Template Matrix**: Playout scripts are generated dynamically based on output target (`SERVER_MODE`) and primary audio source (`acervo`, `streaming`, `hybrid`):
  1. `server_acervo.liq.tmpl` (Perfil A1.a): 100% Acervo MP3 local -> Icecast MP3 128k + HLS. Zero streaming decoders.
  2. `server_streaming.liq.tmpl` (Perfil A1.c): 100% Web Rádio Relay -> Icecast MP3 128k + HLS. Faders `/fade_to_stream`.
  3. `server_hybrid.liq.tmpl` (Perfil A1.b): Misto Acervo + Web Rádio -> Icecast MP3 128k + HLS com switch dinâmico.
  4. `local_acervo.liq.tmpl` (Perfil A2.a): 100% Acervo `.imc` offline -> Placa de som física (PulseAudio/AO).
  5. `local_streaming.liq.tmpl` (Perfil A2.c): 100% Web Rádio -> Placa de som física com injeção de comerciais locais.
  6. `local_hybrid.liq.tmpl` (Perfil A2.b): Misto Acervo + Web Rádio -> Placa de som física.
* **Injecting Alerts (request.queue & smooth_add)**: Avoid Telnet controls for security. Use `request.queue(id="warnings_queue")` for real-time audio injection (like vehicle announcements or chimes).
* **Warnings Queue & Ducking**: Warning announcements and vehicle plate warnings must duck the main background music to `3%` using a smooth add function:
  `smooth_add(p=0.03, warnings_queue, background_music)`
* **Control Endpoints**: Keep control and skipping mechanisms exposed via register harbor HTTP handlers (e.g., `harbor.http.register` on port `7000`/`7001`/`7002` for actions like `/getmeta`, `/skip`, `/push_warning`, `/push_commercial`).
* **Continuous Streaming Routes**: Never apply global request timeouts (`middleware.Timeout`) on streaming endpoints (`/stream/*`, `/m*`) or WebSockets (`/ws/*`). Streaming endpoints must remain indefinitely open for 24/7 broadcast.

## 4. Access Control & Role-Based Security (RBAC)

* **Operator Isolation**: Users with Operator role (`permission_level_id = 3`) are strictly restricted to their store Web Player (`/player/{store_id}`).
* **Administrative Route Protection**: All administrative endpoints (`/tenants`, `/stores`, `/users`, `/music-styles`, `/commercials`, `/grids/*`, `/voice-packs`, `/chimes`, etc.) and the root portal (`/`) must enforce `RequireAdminOrProgrammer` middleware, intercepting operators and redirecting them safely back to their Web Player.
* **Friendly Asset Resolution**: Chimes, voice packs, and pre-formatted warnings must be resolved and listed from database records rather than raw filesystem scans to avoid listing orphaned or deleted files.

## 5. Playout Schedule Boundaries & Off-Hours Behavior

* **Store-Level Isolation**: All operating hours, start/end times (`start_time`, `end_time`), and off-hours behaviors must be strictly scoped to the store's assigned Master Grid (`audio_grid_id`) and individual weekday schedule.
* **Default Off-Hours Policy (`SILENCE`)**: When outside the store's scheduled operating hours (e.g. after 23:00 and before 08:00), the default behavior is **Silence / Standby**:
  * The playout engine must not dispatch commercial blocks or music tracks.
  * In Acervo/Hybrid profiles, Liquidsoap's `mksafe(...)` emits clean silence (0 dB) keeping soundcard and streaming endpoints intact without crashing or spinning.
  * In Streaming profiles, the relay fader transitions to silence (`none`).
  * The Kiosk UI and Web Player must display a clear standby badge (e.g. `🌙 Standby (Fora do Horário - Retorna às HH:MM)`).
* **Optional Contingency Mode (`CONTINGENCY_MUSIC`)**: If explicitly enabled in the grid, playout continues in ambient music mode without commercials, strictly adhering to the Golden Rule (minimum play count rotation) and 45-minute artist separation.

## 6. Commercial Playout Idempotency & Resilience

* **Block Idempotency**: Playout engines must enforce database-backed idempotency checks before inserting `BLOCK_OPENING` or `BLOCK_CLOSING` vinhetas. If a block's opening or closing has already been executed on the current date, the engine must never inject duplicate vinhetas, even if the player process is restarted mid-block.
* **Atomic Sequential Delivery**: All items in a commercial block (Opening $\rightarrow$ Ads $\rightarrow$ Right Time/Temp $\rightarrow$ Closing) must execute in strict FIFO sequence before returning to musical rotation.
* **Zero-Gap Transitions**: Dynamic HTTP fetching in Liquidsoap templates must configure `settings.http.normalize_url.set(false)` to eliminate unnecessary HEAD request overhead and prevent audio gaps between tracks.

## 7. Music Curatorship Taxonomy & Ingestion Architecture

* **Vocal Profile vs. Genre Orthogonality**: "Instrumental" must never be treated as a music genre. It is a distinct **Vocal Profile / Execution Format** (`Instrumental Puro`, `Vocal Feminino`, `Vocal Masculino`, `Dueto`, `Chopped/Samples`). This allows orthogonal combination (e.g., `Genre: Pop` + `Vocal: Instrumental` for clinic lounges, `Genre: Jazz` + `Vocal: Instrumental` for upscale cafes).
* **Multi-Dimensional Taxonomy**: Tracks must be curated and filtered across distinct orthogonal dimensions:
  1. `Macro-Genre` & `Sub-Genres` (Specializations)
  2. `Vocal Profile` (Instrumental vs. Vocal gender)
  3. `BPM` (Tempo) & `Energy Level` (1 to 5)
  4. `Mood / Vibe` (e.g., Ensolarado, Elegante, Relaxante, Urbano, Vibrante)
  5. `Retail Fit / Segment` (e.g., Moda Jovem, Cafeteria, Farmácia, Supermercado)
  6. `Dayparting` (Manhã, Almoço, Tarde, Noite)
* **Playout CPU Isolation & Offline Ingestion**: Heavy DSP tasks (BPM detection, vocal activity detection, R128 -14 LUFS normalization, silence cue in/out calculation) must run in the dedicated offline **Curatorship Tool / IMC Music Studio**, keeping `imc-server` cloud instances free of audio CPU contention.
* **Storage & Encryption Lifecycle**:
  1. **Studio/Curator**: Ingests raw audio $\rightarrow$ Exports normalized MP3 (-14 LUFS) + `catalog_manifest.json`.
  2. **Server (`imc-server`)**: Stores treated MP3 + metadata (enables Web Player and preview with 0% CPU transcoding).
  3. **Store Player (`imc-audio-local`)**: Downloads via API and encrypts locally as `.imc` on disk for anti-piracy offline playback.


