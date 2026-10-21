# Quickstart: FIX-HU-04 / Un solo Sprint activo por proyecto

## Prerequisites

- Go 1.27.1, Docker Desktop y `golang-migrate/migrate` instalados según el [README.md](../../README.md).
- HU-04 integrada en `main`; en tu rama `FIX-HU-04`, traer lo último con `git merge origin/main`.
- `DATABASE_URL` configurada con el puerto que publica tu contenedor (5432 o 5433).

## Antes de probar: revisar datos existentes

Este fix no agrega migraciones. Si tu base local tiene proyectos con dos Sprints activos, la regla nueva
no los corrige; revisarlo y cerrar uno a mano para no confundir las pruebas manuales:

```powershell
docker exec seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion -c "SELECT project_id, COUNT(*) FROM sprints WHERE status = 'Activo' GROUP BY project_id HAVING COUNT(*) > 1;"
```

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

La prueba de creación simultánea usa dos goroutines contra PostgreSQL real; se ejecuta con el resto.

## Required validation scenarios

1. Proyecto con un Sprint activo: crear otro se rechaza con el número del activo; no se crea nada ni se asigna ninguna historia.
2. Después del rechazo, al finalizar el Sprint activo (en las pruebas, cambiando su estado en la base) y crear uno nuevo, el número es el siguiente sin saltos.
3. Todos los Sprints finalizados: se puede crear un Sprint activo.
4. Proyecto A con Sprint activo y proyecto B sin ninguno: se puede crear en B.
5. Dos creaciones simultáneas: se crea un solo Sprint.
6. Asignar historias pendientes: quedan "En Sprint". Asignar una ya asignada: se rechaza con "ya está asignada al Sprint N".
7. Quitar una historia "En Sprint": vuelve a "Pendiente" y se puede asignar de nuevo.

## Acceptance evidence

En PowerShell, ejecutar `chcp 65001` antes:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/004-fix-single-active-sprint/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/004-fix-single-active-sprint/evidencia-pruebas.txt
```

La primera se guarda antes de implementar, con las pruebas nuevas fallando.
