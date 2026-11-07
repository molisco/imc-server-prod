# Walkthrough - Metadata Columns, Dynamic Scan Updates, Stop Buttons & First-Run Metadata Capture

We have implemented and verified all the requested audio metadata enhancements, dynamic scanner updates, client-side table filters, sticky headers, contrast adjustments, cancellation (Stop) controls, metadata capture on first-run scan, and sum of durations.

---

## 1. Additional Metadata Columns (First-Run & Reprocess)
- **Database Schema**: Added three new columns to the `music_library` table:
  - `file_size_bytes BIGINT DEFAULT 0`
  - `bitrate_kbps INT DEFAULT 0`
  - `sample_rate_hz INT DEFAULT 0`
- **Models**: Added corresponding fields `FileSizeBytes`, `BitRateKbps`, and `SampleRateHz` to the `MusicTrack` struct in [music_track.go](file:///P:/imc-server/internal/models/music_track.go).
- **Repository**: Updated CRUD functions and SQL queries in [music_track_repository.go](file:///P:/imc-server/internal/repository/music_track_repository.go) to handle these fields, and added an `Update` method to persist modifications.
- **First-Run Extraction Fix**: Modified `syncMusicStyleDirectory` in [music_style_handler.go](file:///P:/imc-server/internal/handlers/music_style_handler.go) to populate the `FileSizeBytes`, `BitRateKbps`, and `SampleRateHz` fields when registering new tracks for the first time. Previously, these metrics were only retrieved during reprocessing.
- **Extraction**: Modified `ExtractMetadata` in [metadata.go](file:///P:/imc-server/internal/utils/metadata.go) to parse size, bitrate, and sample rate from `ffprobe` JSON format/stream sections, falling back to `os.Stat` for size if necessary.

---

## 2. Decimals & Bounds Calculations
- **Cue Out Decimals**: Adjusted parser in [metadata.go](file:///P:/imc-server/internal/utils/metadata.go) to preserve exact decimal values for outro silences. We only reset `lastSilenceStart = -1.0` if `silence_end` is less than `floatDuration - 0.5`. This prevents trailing silences extending to the EOF from being discarded.
- **Cue Bounds Check**: Added clamp checks for cue points to keep them within `[0.0, floatDuration]`. If `cue_in >= cue_out` is detected, it automatically resets `cue_in = 0.0` and `cue_out = floatDuration`.

---

## 3. Dynamic Scan Updates & Tracks Partial
- **Table Partial**: Created [tracks_table_partial.html](file:///P:/imc-server/web/templates/tracks_table_partial.html) containing the inner tracks table view with the new size (formatted), bitrate, and sample rate columns.
- **Dynamic Render Endpoint**: Registered `/dashboard/music-styles/tracks-html` GET handler to render only the tracks table block.
- **Live Updating**: Connected `pollSyncStatus` to fetch the table partial dynamically on every poll tick, updating the DOM container. Checkbox states, selections, and active filters are saved and reapplied during update.
- **Scan Auto-Expand**: The page checks for styles with a `scanning` status on load and automatically expands their tracks row to show the scanner progress.

---

## 4. Client-side Search, Filters & Sticky Headers
- **Sticky Headers**: Added a `.sticky-thead` CSS class to style table headers with `position: sticky` and a high z-index, keeping the header fixed at the top of the container during scroll.
- **Client-Side Filters**: Added a search bar and a loudness score grade dropdown select (Nota A, B, C, D) inside the style tracks header block. Filtering is instantaneous and checks filename, title, artist, and album.
- **Description Contrast**: Changed the description cell class from `text-muted` to `text-white-50` in [music_styles.html](file:///P:/imc-server/web/templates/music_styles.html) to improve contrast against the dark background.

---

## 5. Sync and Reprocess Cancellation (Stop Buttons)
- **General Scan Stop**:
  - Implemented `/dashboard/music-styles/stop-sync` (POST) to stop a running background directory scan.
  - Tracked goroutine execution contexts using a thread-safe registry mapping Style IDs to `context.CancelFunc`.
  - Added a visual `🛑 Parar` button next to the active scan progress bar in [music_styles.html](file:///P:/imc-server/web/templates/music_styles.html) which issues the stop command.
- **Selected Reprocess Stop**:
  - Leveraged Web APIs (`AbortController` / `AbortSignal`) on the client-side to cancel the HTTP request when the user hits the cancel button.
  - Enhanced the backend handler in [music_style_handler.go](file:///P:/imc-server/internal/handlers/music_style_handler.go) to verify if the request context (`r.Context()`) has been closed on every loop step and immediately abort further track calculations.
  - Rendered a `🛑 Cancelar Reprocessamento` button when selected tracks reprocessing is underway.

---

## 6. Time Summation Display
- **HH:MM:SS format**: Added duration summation helper `formatTotalDuration` and updated the main page renderer `HandleMusicStylesPage` as well as the error template rendering `renderMusicStylesError` and dynamic HTML endpoint `/tracks-html` to calculate and render the total sum next to the track count.

---

## 7. Timestamps, IRR and Multi-Style Sync (New Enhancements)
- **Timestamps**:
  - Added `updated_at` timestamps to `music_styles` and `music_library` tables.
  - Automatically update the style's `updated_at` timestamp when tracks are added, deleted, or reprocessed.
- **Noise Risk Score (IRR)**:
  - Combined `ebur128`, `silencedetect`, and `volumedetect` into a single `ffmpeg` call per track to minimize CPU/I/O load.
  - Implemented IRR percentage calculator (0% to 100%) based on warning checks (Clipping: +40, Junk data: +10, lost frames/sync errors: +25 per occurrence).
  - Added color-coded badges to display the IRR score (🟢 0-15%, 🟡 16-49%, 🔴 50-100%).
- **Multi-Style Batch Sync**:
  - Added checkboxes to select multiple styles on the main page.
  - Added a "Sincronizar Selecionados" button to trigger concurrent sync operations in the background.
  - Fixed an issue where the page reloaded prematurely when the first batch style completed, which aborted polling and visual updates for other concurrently running syncs. Introduced a client-side `activeSyncs` Set to track running scans and defer the page reload until all selected styles are fully synchronized.
  - Introduced a batch synchronization option modal (`batchSyncModal`) prompting the user to choose between **"Sincronizar Novos"** (fast scan for new files only) and **"Reprocessar Tudo"** (re-extraction of all files and recalculation of LUFS, LRA, Cue, and IRR). This prevents the sync from instantly finishing when there are no new files.
- **Sort Preservation & Default Sort**:
  - Default track lists are sorted in ascending alphabetical order by "Nome do Arquivo" (colIndex 2) on first open.
  - Added sorting persistence using `localStorage`. The selected column and direction are preserved across dynamic refreshes (AJAX table updates) and browser reloads (F5).

---

## 8. Milliseconds & Waveform Editor (Latest Enhancements)
- **Millisecond Precision**:
  - Modified the Go `formatDateTime` helper to output day/month/year hour:minute:second.milliseconds (`02/01/2006 15:04:05.000`).
  - Rewrote the client-side `parseDateToTimestamp` parser in `music_styles.html` to parse seconds and milliseconds, ensuring that sorting by "Última Atualização" prioritizes the exact order of execution.
- **Player Text Contrast**:
  - Changed the CSS styling of `#player-track-artist` in the bottom premium player from `text-muted` (which was too dark) to `text-white-50`, making the artist name clearly readable against the dark background.
- **Waveform Cue Editor (Sound Forge style)**:
  - Added a `✂️` edit button in each music row of the table.
  - Implemented the `cueEditorModal` using **Wavesurfer.js** (loaded via CDN) with its Regions and Timeline plugins.
  - Clicking `✂️` loads the track dynamically and displays an interactive, draggable, and resizable visual region representing the current `Cue In` and `Cue Out` points.
  - Resizing or dragging the region instantly updates the input values. Typing values manually in the inputs also updates the region.
  - Added POST endpoints `/dashboard/music-styles/save-cue` and `/dashboard/music-styles/reset-cue` to save manual adjustments or restore automatic values.
  - Added `manual_cue` boolean column to the `music_library` table to track manual edits. Manually adjusted cue points are colored in cyan (`text-info`) in the tracks list.
  - Modified `syncMusicStyleDirectory` and `HandleReprocessMusicTracks` to skip overwriting cue points if `manual_cue = true` during subsequent synchronizations or reprocesses.

---

## 9. Verification and Tests
- Fixed compilation and unused variable issues in test files (`internal/handlers/music_style_handler_test.go` and `internal/utils/metadata_test.go`).
- Verified all Go unit tests compiled and passed successfully on the remote server `dreams`.
- Recompiled and restarted the central Go brain server on the remote `dreams` host, verifying that all schema patches and dynamic JS sorting work perfectly.

---

## 10. Cue Points Editor Tweaks (Seconds Display & Play Button Reset)
- **Seconds Display**:
  - Integrated dynamic seconds alongside the formatted time in the editor controls: `00:00.00 (0.00s) / 00:00.00 (0.00s)`.
  - Current elapsed seconds are updated continuously using wavesurfer's `timeupdate` event listener.
  - Total duration in seconds is updated only once when the audio wavesurfer instance emits its `ready` event.
  - Adjusted the controls container columns to `col-md-5`, `col-md-4`, `col-md-3` to accommodate the extra labels without wrapping.
- **Play Button Reset Bug**:
  - Created a robust `cleanupCueEditor()` helper function that pauses and destroys the wavesurfer instance, sets the reference to `null`, and resets the play button text to `"▶️ Play"`.
  - Attached a listener to Bootstrap's `hidden.bs.modal` event on the editor modal. This guarantees that `cleanupCueEditor()` is executed whenever the modal is hidden (via escape, clicking outside, close button, saving, or resetting).
  - Explicitly call `cleanupCueEditor()` and reset time labels to `"00:00.00 (0.00s)"` inside `openCueEditor()` before initializing the new wavesurfer instance, preventing stale states from showing up.
- **Autofocus & Keyboard Shortcuts**:
  - Configured `shown.bs.modal` to automatically `.focus()` on `#btn-editor-play` once the modal finishes transitioning open, letting the user immediately play/pause using Space.
  - Implemented a modal-level `keydown` listener that toggles play/pause with the Spacebar (`Space` or `' '`).
  - Added smart filtering to ignore Spacebar keydown events if focus is inside any numeric/text input fields, textareas, or other buttons (e.g., Save, Close, Reset), preventing conflicts with default browser input/button behaviors.
- **Verification**:
  - Checked that all Go unit tests pass on the remote server.
  - Confirmed the template updates are parsed live by the Go server.

---

## 11. Font Color Contrast Improvements (Description & Directory Columns)
- **Description Column Contrast**:
  - Replaced the low-contrast `text-muted` class (which rendered as a very dark grey on the dark background) with `text-white-50` (50% opaque white) in the tables of:
    - [grids_audio.html](file:///p:/imc-server/web/templates/grids_audio.html) (Audio Grids list)
    - [grids_video.html](file:///p:/imc-server/web/templates/grids_video.html) (Video Grids list)
    - [video_styles.html](file:///p:/imc-server/web/templates/video_styles.html) (Video Styles list)
- **Base Directory Contrast**:
  - Changed `text-muted` to `text-white-50` for the base directory columns in [music_styles.html](file:///p:/imc-server/web/templates/music_styles.html) and [video_styles.html](file:///p:/imc-server/web/templates/video_styles.html), making the directory paths readable.

---

## 12. Active Track Row Highlight (Premium Playback Feedback)
- **CSS Styling**:
  - Created a custom CSS class `.playing-track-row` in [music_styles.html](file:///p:/imc-server/web/templates/music_styles.html) which highlights the currently playing track with a subtle semi-transparent blue background (`rgba(67, 100, 247, 0.15)`) and a colored left border (`border-left: 4px solid var(--primary-color)`), giving a glowing interactive feedback.
- **Dynamic State Tracking**:
  - Added a global `currentlyPlayingTrackId` state variable to track the currently playing track ID.
  - In `playMusic(trackId, ...)`, the old highlights are cleared and the `.playing-track-row` class is dynamically added to the row `#track-row-` + `trackId`.
  - In `closeAudioPlayer()`, the `currentlyPlayingTrackId` is reset to `null` and any active row highlight is removed.
- **Dynamic Refresh Preservation**:
  - In `updateTracksTable(styleId)`, we check if `currentlyPlayingTrackId` matches any track inside the refreshed table and automatically re-apply the `.playing-track-row` class, keeping the highlight active when the scanner completes or table updates in the background.

---

## 13. Automatic Database Cleanup on Deleted Files/Directories (Latest Enhancements)
- **Directory Deletion Cleanup**:
  - Updated `syncMusicStyleDirectory` in [music_style_handler.go](file:///p:/imc-server/internal/handlers/music_style_handler.go) to verify if the configured base directory exists. If the directory is missing or is not a folder (e.g. deleted or renamed), the scanner logs the event, deletes all registered tracks for that style ID from the database, updates the style's `updated_at` timestamp, updates the scan progress to `completed` and exits cleanly. This prevents orphaned tracks from persisting when their folders are removed.
- **Track Deletion Logging & Error Handling**:
  - Added logging to print an info log with the exact ID and filename of any missing track deleted from the database.
  - Added log statements to capture and print database errors if a deletion fails.
- **Unit Tests**:
  - Added `TestSyncMusicStyleDirectory_MissingDirectory` to [music_style_handler_test.go](file:///p:/imc-server/internal/handlers/music_style_handler_test.go) which creates a style, scans a mock folder containing a track, deletes the folder, runs the sync again, and verifies that the database records for the style are completely cleaned up and the sync finishes with a `completed` progress status.

---

## 14. Commercials & Events Module with Store Distribution Engine
- **Database Schema Patches**:
  - Added enums `GENERAL_INFORMATION`, `RIGHT_TIME`, and `TEMPERATURE` to `commercial_type_enum`.
  - Added a `distribute_to_all` boolean column to the `commercials_and_events` table (defaulting to true).
  - Created the N:N mapping table `commercial_distributions` to link commercials directly to target stores.
- **Extended Models**:
  - Added the new categories to the `CommercialType` enum in [media..go](file:///p:/imc-server/internal/models/media..go).
  - Added `DistributeToAll` to the `CommercialsAndEvents` model struct.
- **Repository Implementation**:
  - Implemented CRUD methods, global/tenant filters, and transaction-safe store distribution saving in [commercial_repository.go](file:///p:/imc-server/internal/repository/commercial_repository.go).
  - Verified logic with automated unit tests in [commercial_repository_test.go](file:///p:/imc-server/internal/repository/commercial_repository_test.go).
- **Control Handlers**:
  - Added file upload logic with category subfolder organization (under `midias/comerciais/[TENANT_ID]/[CATEGORY_NAME]/`) in [commercial_handler.go](file:///p:/imc-server/internal/handlers/commercial_handler.go).
  - Integrated `ffprobe` to automatically calculate duration and extract audio details.
  - Implemented target distribution retrieve/save endpoints.
  - Registered web routes in [main.go](file:///p:/imc-server/cmd/server/main.go).
- **Stunning Frontend UI & Sidebar Integration**:
  - Created a modern, dark-themed commercials dashboard in [commercials.html](file:///p:/imc-server/web/templates/commercials.html).
  - Embedded an interactive target store selection modal allowing operators to target campaigns to specific stores.
  - Synchronized navigation across the system by adding the "Comerciais & Eventos" link to all sidebar layout templates.
  - **Category Tabs & Client-Side Filtering (Latest)**: Split the dashboard into a vertical side-tab list navigation on the left ( Todos, Comerciais, Ofertas, Institucionais, Aberturas, Fechamentos, Eventos, Informativos, Hora Certa, Temperatura) and the content list table on the right. Track counts in each category are updated dynamically via Javascript.
  - **Validations & Warning Alerts (Latest)**: Modified dropdown matrix selection to include a default disabled placeholder, renamed the global option to just `Global / Geral`, added a dynamic alert box warning that global selections target all stores system-wide, and added submit-interception checks requiring confirmation before saving global campaigns.
  - **Hierarchical Store Selector (Latest)**: Re-structured the distribution modal list into collapsible company check-groups. Operators can expand/collapse matrices, toggle a "Select All" check per company, or check individual stores.
- **Testing & Verification**:
  - Compiled the binary successfully and ran all tests on the remote server `dreams`. All tests pass cleanly.
  - Relaunched the server process in the background, checking that database migrations successfully applied.
---

## 15. Commercials & Events Form Extensions & Audio Quality Engine (Latest Enhancements)
- **Audio Quality Engine & Metadata Extraction**:
  - Integrated the full audio quality analyzer (`utils.ExtractMetadata`) into `HandleCreateCommercial` in [commercial_handler.go](file:///p:/imc-server/internal/handlers/commercial_handler.go).
  - Every uploaded commercial is analyzed via `ffmpeg` to extract LUFS, LRA, Loudness Score, Cue points, Bitrate, Sample Rate, and Noise Risk Score (Risco Ruído / IRR).
  - Added visual quality badges to the commercials list table in [commercials.html](file:///p:/imc-server/web/templates/commercials.html) for LUFS/LRA, Cue In/Out, Loudness Score (Nota A/B/C/D), and Noise Risk percentage, mirroring the premium design of the music library.
- **Form Tab locking**:
  - Converted the category sidebars into a clean, horizontal list of tabs.
  - Selecting any category tab locks the upload form's `commercial_type` dropdown selection to that category (`disabled = true`). Selecting the "Todos" tab unlocks selection.
  - Before form submission, Javascript enables the dropdown to ensure the value is included in the POST request.
- **Wavesurfer.js Local Audio Preview**:
  - Imported `Wavesurfer.js` (v7) via CDN.
  - Added an inline preview container inside the upload form. When a local file is selected, it loads as an Object URL, generates the waveform, and offers a Play/Pause button with timestamp trackers.
- **Inline Store Distribution checklist**:
  - Added an inline "Distribuição por Filial" checklist inside the upload form.
  - Unchecking "Distribuir para Todas as Lojas automaticamente" dynamically reveals the list.
  - Shows stores grouped by Matrix. If a specific Matrix is chosen in the form dropdown, only that matrix's stores list is displayed. If "Global / Geral" is chosen, all matrices are rendered inside collapsible containers.
  - Changes made to the inline checklist are parsed in the Go backend handler and saved directly to the N:N mapping table `commercial_distributions`.
- **Nullable Date Validity & Event Execution Times**:
  - Added a `Sem data de término` checkbox to the campaign end date field. Checking it disables the input and sets the database `end_date` value to `NULL` (infinite campaign).
  - Added an `Horário de Execução Exata` input field that is displayed only when the commercial type is `EVENT`. It registers the specific time (`execution_time TIME` column) when the event audio should trigger.
- **Matrix & Stores Seeding**:
  - Configured pgx startup migrations in [postgres.go](file:///p:/imc-server/internal/database/postgres.go) to seed 10 Matrices (`Matriz 01` to `Matriz 10`) and 100 Stores per matrix (named `Matriz [I] - Loja [J]` with unique C# connection tokens).
- **Store Scan Fix**:
  - Fixed a scanning error in `TestStoreCRUD` in [store_repository.go](file:///p:/imc-server/internal/repository/store_repository.go) by wrapping `zip_code` and `street` columns with SQL `COALESCE` statements, ensuring `NULL` database values map to safe empty strings in Go.
  - Verified that all unit tests in the codebase build and pass cleanly.

---

## 16. Multiple Commercials Batch Upload & Sidebar Cleanup (Latest Enhancements)
- **Batch Audio File Uploads**:
  - Modified the upload file input in [commercials.html](file:///p:/imc-server/web/templates/commercials.html) to support selecting multiple files at once (`multiple` attribute, name `files`).
  - Removed the manual `"Nome do Comercial / Evento"` text input field. The commercial name now defaults automatically to each file's uploaded filename.
  - Updated the backend handler `HandleCreateCommercial` in [commercial_handler.go](file:///p:/imc-server/internal/handlers/commercial_handler.go) to loop through all files in the multipart form, extracting the metadata, creating the database records, and mapping the target store distributions for each commercial.
- **Duplicate Validation Rules**:
  - Added the helper function `ExistsByName` in [commercial_repository.go](file:///p:/imc-server/internal/repository/commercial_repository.go) to verify if a commercial name already exists in the database for the selected tenant.
  - Added a two-tier validation check in the Go backend before any files are processed:
    1. Rejects the request if there are duplicate filenames inside the uploaded batch itself.
    2. Rejects the request with a bad request explanation if any of the filenames match an existing commercial name in the database for the selected tenant.
- **Visual Wavesurfer.js Batch Previewer**:
  - Refactored the Wavesurfer script in [commercials.html](file:///p:/imc-server/web/templates/commercials.html) to handle multiple local files.
  - Generates a visual list of badges/buttons for all selected files. Clicking on any file loads it directly into the audio waveform previewer, updating the play/pause button and cue point estimates.
- **Sidebar Menu Cleanup**:
  - Removed the `🎵 Banco de Mídias` link from the sidebar menu in all 9 dashboard layout templates, ensuring clean, unified, and consistent navigation:
    - [commercials.html](file:///p:/imc-server/web/templates/commercials.html)
    - [dashboard.html](file:///p:/imc-server/web/templates/dashboard.html)
    - [grids_audio.html](file:///p:/imc-server/web/templates/grids_audio.html)
    - [grids_video.html](file:///p:/imc-server/web/templates/grids_video.html)
    - [music_styles.html](file:///p:/imc-server/web/templates/music_styles.html)
    - [regionals.html](file:///p:/imc-server/web/templates/regionals.html)
    - [stores.html](file:///p:/imc-server/web/templates/stores.html)
    - [tenants.html](file:///p:/imc-server/web/templates/tenants.html)
    - [video_styles.html](file:///p:/imc-server/web/templates/video_styles.html)

---## 17. Premium Checkbox Dropdowns & Multi-Matrix Commercials
- **Combobox Dropdown Filters**:
  - Replaced the Matrix filter checklist box with a compact dropdown button displaying checkboxes in its menu, styled as a combobox.
  - Dynamically updates the button text to show the active selections (e.g. `Todas as Empresas`, `Matriz 01`, or `[N] Empresas selecionadas`).
- **Multi-Matrix Registration Form**:
  - Replaced the single-select Matrix dropdown in the registration form with a matching dropdown checklist combobox.
  - Supports selecting multiple matrices during creation, allowing batch registration of uploaded audio commercials across all checked matrices at once.
  - The inline stores checklist dynamically shows only the stores belonging to the checked matrices.
  - Handles mutual exclusivity between specific matrices and the `Global / Geral` option.
  - Enforces single-select radio-like behavior when in Edit Mode.

---

## 18. Final Verification & Launch (Current Enhancements)
- **Automated Tests**:
  - Checked compilation and ran tests via SSH on the remote server `dreams`. All unit tests passed cleanly.
- **Server Execution**:
  - Recompiled the binary `server` and started the server process in the background.
  - Verified login page accessibility via curl (HTTP `200`).

---

## 19. Dynamic Registration Buttons & Event Form Dynamic Validation (Resumed Enhancements)
- **Form Titles & Dynamic Submit Buttons**:
  - Implemented dynamic titles (e.g., `Cadastrar Novo Evento Especial`, `Cadastrar Nova Oferta`) and save buttons (e.g., `Salvar Evento`, `Salvar Oferta`) inside the creation form based on the active tab category or selected type dropdown.
  - Dynamically updates the form title in Edit Mode based on the edited commercial's type (e.g., `Editar Evento Especial`, `Editar Oferta`).
- **Tab Category Preservation on Reset**:
  - Updated `cancelEditMode()` to preserve and re-apply the locked category selection when the form is cleared or created, resolving a bug where resetting the form defaulted the category dropdown to `COMMERCIAL`.
- **Event Validation & Execution Time**:
  - Fully verified that selecting `Eventos` locks the type to `EVENT`, showing the `execution_time` field and hiding daily start and end times, automatically defaulting daily start/end to `00:00` and `23:59`.
- **Remote Host Compilation & Run**:
  - Recompiled the server binary on the remote host `dreams` and verified all automated tests passed cleanly. Relaunched the server process in the background.

---

## 20. Table Row Hover States, Commercial Name Search, Regional Combobox Filter, and Saved Pagination (Latest)
- **Row Hover States**:
  - Implemented custom CSS row hover styles on all tables across the platform (Matrizes/Clientes, Regionais, Filiais, Estilos Musicais, Estilos de Vídeo, Comerciais & Eventos, Grades de Áudio, Grades de Vídeo, and tracks lists).
  - Highlighting is achieved via a subtle white background translucency (`rgba(255, 255, 255, 0.075)`) and a subtle white border outline (`1px solid rgba(255, 255, 255, 0.15)`) to improve line readability when the mouse cursor is hovering.
- **Commercial Filters**:
  - Converted the single-select Regional dropdown filter to a premium multi-select checklist combobox dropdown, matching the Matrix dropdown filter and displaying the count of selected items.
  - Implemented a live "Nome do Comercial" text search filter to search commercials by name instantaneously.
  - Converted the single-select Store (Filial) dropdown filter to a premium multi-select checklist dropdown combobox with checkboxes and a live search input inside.
- **Client-Side Saved Pagination**:
  - Integrated full client-side pagination on all primary system tables (10, 20, 50, 100, ALL rows per page, defaulting to 50).
  - Page-size limits are synchronized and stored in `localStorage` under the key `imc_table_limit`, maintaining the preference across all pages and reloads.
  - Added full navigation controls (`«`, `‹`, `›`, `»`) showing standard index bounds (`Showing X-Y of Z lines`).
  - Added custom tracks list pagination states in `music_styles.html` and `tracks_table_partial.html` so expanded track sections paginate independently without colliding.
- **Target Store Selection Search (Distribution Engine)**:
  - Added live search inputs to both the target store distribution modal (`distributionModal`) and the inline campaign distribution checklist inside the creation/edit form.
  - Users can filter list items by store name or matrix name in real time, automatically collapsing matrix headers with no matching stores. Search query states are automatically reset when opening modals or canceling forms.
- **Premium Lojas (Filiais) Filters**:
  - Rebuilt the filter block on the Filiais (Lojas) dashboard screen (`stores.html`) to include premium checkbox dropdown comboboxes for Matrizes, Regionais, and Filiais.
  - Each combobox dropdown features a live search input, support for selecting multiple options simultaneously, automatic label counts, and cascading dependencies (e.g. selecting a matrix dynamically limits the visible regionals and stores inside their respective checklist menus).
- **Grid block legacy / deleted styles unselection**:
  - Solved an issue in [grids_audio.html](file:///P:/imc-server/web/templates/grids_audio.html) and [grids_video.html](file:///P:/imc-server/web/templates/grids_video.html) where styles deleted from the database catalog (e.g. `Estilo 01`) remained associated with grid blocks but could not be removed because they were missing from the selection combobox.
  - Implemented logic that merges a block's currently selected styles with the active catalog, ensuring any legacy or deleted style is displayed as checked in the combobox, allowing the user to simply uncheck and save to remove it.


