# Plan - PoP Logging, Execução Real Tab, e Auto-Catchup

## 1. Proof of Play (PoP) - Registro Estruturado de Logs

### [MODIFY] [db.go](file:///c:/imc-audio-local/db.go)
* Add `duration_seconds` and `skipped_by_catchup` columns to `play_history` via ALTER migration.

### [MODIFY] [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Expand `recordPlayHistory` to include `duration_seconds` in the INSERT.

---

## 2. Aba "Execução Real" no Diagnóstico

### [MODIFY] [web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go)
* Add `PlayHistory []PlayHistoryRecord` to `DiagnosticResponse`.
* Add `PlayHistoryRecord` struct with the fields: id, track_id, track_type, display_name, artist, played_at, duration_seconds, skipped_by_catchup.
* Query `play_history` for today in `handleAPIDiagnostics`, ordering by `played_at ASC`.
* Add a 4th tab button "Execução Real (Hoje)" in the tabs nav.
* Add `switchTab` case for the new tab.
* Add JS rendering loop that creates a table identical to the simulation timeline, but populated from real `play_history` data.

---

## 3. Auto-Catchup (Skip Automático para Recuperar Atraso)

### [NEW] Logic in [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Add `StartCatchupMonitor()` goroutine.
* Every 15 seconds, compare current clock time vs. the scheduled time of the item currently playing.
* If delay >= 3 minutes AND current music track has elapsed >= 50% of its duration, trigger skip via `http://127.0.0.1:7000/skip`.
* Log the auto-skip action in `play_history` and via `LogLocal`.
* Only skip music tracks (never skip commercials or block items).

### [MODIFY] [main.go](file:///c:/imc-audio-local/main.go)
* Launch `go StartCatchupMonitor()` alongside the other background goroutines.

---

## Verification Plan

### Automated Tests
* `go test -v ./...`

### Manual Verification
* Check `/diagnostics` tab "Execução Real" shows real play data.
* Verify auto-catchup behavior in logs when player falls behind.
