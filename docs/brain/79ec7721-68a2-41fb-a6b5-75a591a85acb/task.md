# Task - PoP Logging, Execução Real Tab, Auto-Catchup

## 1. PoP Logging (play_history)
- `[x]` Add `duration_seconds` and `skipped_by_catchup` columns via ALTER in `db.go`
- `[x]` Add index on `track_id` for faster PoP queries
- `[x]` Update `recordPlayHistory` in `scheduler.go` to save `duration_seconds`

## 2. Aba "Execução Real" no Diagnóstico
- `[x]` Add `PlayHistoryRecord` struct and `PlayHistory` + `CurrentDesvioSecs` fields to `DiagnosticResponse`
- `[x]` Query today's `play_history` in `handleAPIDiagnostics`
- `[x]` Add 4th tab button "Execução Real (Hoje)" in `diagnosticsHTML`
- `[x]` Add tab content with table (Horário, Tipo, Título, Artista, Duração, Status)
- `[x]` Add JS rendering for real execution timeline table with color-coded types
- `[x]` Add desvio badge and stats counter (músicas, comerciais, puladas)
- `[x]` Update `switchTab` JS for new tab

## 3. Auto-Catchup (Skip Automático)
- `[x]` Add `elapsed_seconds` to Liquidsoap `/getmeta` response in `watchdog.go`
- `[x]` Add `StartCatchupMonitor()` goroutine in `scheduler.go` 
- `[x]` Add `calcCurrentDesvio()` helper function in `scheduler.go`
- `[x]` Launch goroutine in `main.go`

## 4. Verificação
- `[x]` `go build ./...` - compilação OK
- `[x]` Restart player com `go run .`
