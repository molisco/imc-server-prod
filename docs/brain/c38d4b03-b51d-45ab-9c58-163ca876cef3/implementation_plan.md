# Implementation Plan - Automated Playout Orchestration (Docker)

This plan details the addition of automatic Docker container management to the `imc-server` panel when creating, updating, or deleting stores configured as server-side radios.

## Proposed Changes

We will introduce a `harbor_port` column to the database, implement dynamic port allocation, and add orchestration triggers in the backend to start, stop, and clean up Docker containers dynamically.

---

### Database Migration

#### [MODIFY] [postgres.go](file:///P:/home/neoroute/imc-server/internal/database/postgres.go)
We will add an automatic schema patch query to add the `harbor_port` column to the `stores` table:
```sql
ALTER TABLE stores ADD COLUMN IF NOT EXISTS harbor_port INT DEFAULT NULL
```

---

### Models & Repository Updates

#### [MODIFY] [store.go](file:///P:/home/neoroute/imc-server/internal/models/store.go)
Add the `HarborPort` field to the `Store` model struct:
```go
HarborPort *int `json:"harbor_port,omitempty" db:"harbor_port"`
```

#### [MODIFY] [store_repository.go](file:///P:/home/neoroute/imc-server/internal/repository/store_repository.go)
Update `Create`, `GetAll`, and `Update` queries to fetch and save `harbor_port`:
- **Create**: Insert `store.HarborPort`.
- **GetAll**: Select `s.harbor_port` and scan it.
- **Update**: Update `harbor_port = $21` (where the new parameter index is `$21` and `WHERE id = $1` becomes `$1`).

---

### Playout Orchestrator (Orchestration Logic)

#### [NEW] [docker_orchestrator.go](file:///P:/home/neoroute/imc-server/internal/utils/docker_orchestrator.go)
We will create a helper package to manage Docker operations:
- `StartStoreDockerContainer(store models.Store) error`:
  - Stops and removes any pre-existing container with the name `imc-radio-<store_id>`.
  - Creates the host persistent cache directory.
  - Spins up a new container using the image `imc-audio-local-radio_loja_01:latest`, exposing the dynamically allocated harbor port to the host `-p 127.0.0.1:harbor_port:harbor_port`.
- `StopStoreDockerContainer(storeID uuid.UUID) error`:
  - Enforces graceful shutdown and removal of the container `imc-radio-<store_id>`.

---

### Go Handler Updates

#### [MODIFY] [store_handler.go](file:///P:/home/neoroute/imc-server/internal/handlers/store_handler.go)
- **Port Allocation Helper**: If a store needs a harbor port, we find the maximum harbor port in the database and assign `MAX(harbor_port) + 1` (starting at 7001).
- **HandleCreateStore**:
  - If the store is `server`, assign a `harbor_port`, write to the DB, and trigger `StartStoreDockerContainer`.
- **HandleUpdateStore**:
  - If type changed to `server`, assign a `harbor_port` if missing, and trigger `StartStoreDockerContainer`.
  - If type changed to `local`, trigger `StopStoreDockerContainer`.
- **HandleDeleteStore**:
  - Trigger `StopStoreDockerContainer` before deleting the database record.

---

## Verification Plan

### Automated / Manual Verification
1. **CRUD Testing**: Create a new store on the panel and verify:
   - A unique harbor port (e.g. 7003) is allocated in the database.
   - A corresponding container `imc-radio-<id>` is running (`docker ps`).
   - The mount point (e.g. `/3`) is registered with Icecast.
2. **Type Switching**: Update the store to `Local` and verify the container is stopped and removed.
3. **Deletions**: Delete the store and verify the container is completely cleaned up.
