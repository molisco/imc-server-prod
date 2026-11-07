# Tasks - Automated Caddy Configuration

- [x] Modify `docker_orchestrator.go` in `imc-server`:
    - [x] Import `context` and `imc-server/internal/database`
    - [x] Implement `RebuildAndReloadCaddy()` function
    - [x] Invoke `RebuildAndReloadCaddy()` at the end of `StartStoreDockerContainer()`
    - [x] Invoke `RebuildAndReloadCaddy()` at the end of `StopStoreDockerContainer()`
- [x] Build & Verify implementation:
    - [x] Compile `imc-server` binary on server
    - [x] Restart `imc-server` user service
    - [x] Trigger store update or run manual query reload to test Caddyfile generation and hot-reload
    - [x] Test access to diagnostics page for Loja 03
