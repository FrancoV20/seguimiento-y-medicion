# Quickstart: FIX-HU-01 / Fecha de fin posterior a la de inicio

## Prerequisites

- Go 1.27.1, Docker Desktop y `golang-migrate/migrate` instalados según el [README.md](../../README.md).
- HU-01 integrada en `main`; en tu rama `FIX-HU-01`, traer lo último con `git merge origin/main`.
- `DATABASE_URL` configurada con el puerto que publica tu contenedor (5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5433/seguimiento_y_medicion?sslmode=disable`.

## Antes de migrar: revisar datos existentes

Si tu base local tiene proyectos con fecha de inicio igual a la de fin, la migración falla. Revisarlo:

```powershell
docker exec seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion -c "SELECT id, name, start_date, end_date FROM projects WHERE end_date = start_date;"
```

Si devuelve filas, corregirlas a mano o recrear la base con `docker compose down -v` y migrar de nuevo.

## Probar la migración

El número de la migración es el último de `main` más uno (verificarlo en `db/migrations`). No se
edita la migración `001`.

```powershell
migrate -path db/migrations -database $env:DATABASE_URL up
migrate -path db/migrations -database $env:DATABASE_URL down 1
migrate -path db/migrations -database $env:DATABASE_URL up
```

Comprobar la restricción con `docker exec ... psql -c "\d projects"`: debe figurar `CHECK (end_date > start_date)`.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

## Required validation scenarios

1. Fin anterior al inicio: se rechaza con el mensaje de fechas.
2. Fin igual al inicio: se rechaza con el mensaje de fechas y no se crea el proyecto.
3. Fin un día posterior al inicio: se crea y se informa el éxito.
4. Guardar directamente en el almacenamiento con fechas iguales: la base lo rechaza y no queda ninguna fila.
5. Ninguna otra prueba del repositorio crea proyectos con fechas iguales.

Mientras HU-21 no esté integrada, verificar 1 a 3 con las pruebas del servicio; cuando lo esté,
probarlo también en la pantalla "Gestión de Proyectos".

## Acceptance evidence

En PowerShell, ejecutar `chcp 65001` antes:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/001-fix-project-dates/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/001-fix-project-dates/evidencia-pruebas.txt
```

La primera se guarda antes de implementar, con las pruebas nuevas fallando.
