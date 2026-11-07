# Checklist - Hotmix Radio Integration

- [x] Execute Postgres database migrations (add `external_stream_url` and `external_stream_name` to `stores` table)
- [x] Modify `models/store.go` in `imc-server`
- [x] Modify `repository/store_repository.go` in `imc-server`
- [x] Modify `handlers/store_handler.go` in `imc-server`
- [x] Modify `utils/docker_orchestrator.go` in `imc-server`
- [x] Modify Web UI template `web/templates/stores.html` in `imc-server`
- [x] Modify `config.go` in `imc-audio-local`
- [x] Modify `player.liq.tmpl` in `imc-audio-local`
- [x] Modify `scheduler.go` in `imc-audio-local` to implement `StartExternalStreamScheduler` and push logic
- [x] Disable standard scheduler/hard events check when in external stream mode
- [x] Create `sync_hotmix.py` scraper in `imc-external-fonts`
- [x] Verify Go backend and local player build compilation
- [x] Final Walkthrough
