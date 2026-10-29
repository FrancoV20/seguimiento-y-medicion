# Quickstart: FIX-HU-21 / Estado "Activo" al crear un proyecto

## Prerequisites

- Go 1.27.1, Docker Desktop y `golang-migrate/migrate` instalados según el [README.md](../../README.md).
- HU-01, HU-02 y HU-21 integradas en `main`; el fix se trabaja en la rama `FIX-HU-21` (traer lo último con `git merge origin/main`).
- `DATABASE_URL` configurada con el puerto que publica tu contenedor (5432 o 5433).

## Antes de probar: revisar datos existentes

Si creaste proyectos desde la web antes del fix, tu base los tiene con estado "Active". Para verlos:

```powershell
docker exec seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion -c "SELECT id, name, status FROM projects WHERE status <> 'Activo';"
```

La migración del fix los pasa a "Activo":

```powershell
migrate -path db/migrations -database $env:DATABASE_URL up
```

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

Las pruebas del fix están en `src/project/project_test.go`, `src/project/service_test.go`,
`src/project/postgres_store_integration_test.go`, `test/project_bdd_test.go` y
`test/project_backlog_flow_test.go` (esta última necesita `DATABASE_URL`; sin la variable se omite).

## Required validation scenarios

1. Crear un proyecto válido: queda con estado "Activo".
2. Crear un proyecto desde la pantalla Proyectos, seleccionarlo y cargarle una historia válida: la historia se registra en estado "Pendiente".
3. Un proyecto guardado como "Active" antes del fix: después de aplicar las migraciones queda "Activo" y admite historias.

## Acceptance evidence

- `evidencia-fix-estado-red.txt`: salida de `go test` con las pruebas nuevas fallando.
- `evidencia-fix-estado.txt`: salida de `go test` completa en verde.