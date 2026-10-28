# Guía rápida: HU-10 – Cálculo de Métricas de Calidad y Avance

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-06 (defectos) y HU-15 (cierre de Sprint) en `main`, además de HU-07, HU-14, HU-20a y HU-20b. Comprobarlo con
  `git log --oneline main` y `git grep -n "sprint_story_results\|CREATE TABLE defects" -- db/migrations`.
- Rama: `git switch main; git pull; git switch -c HU-10`. Solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-10 no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "Quality|HU10" -v ./src/metrics/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/metrics/...
gofmt -l src/metrics src/web src/cmd/server test
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo.

## Escenarios de validación requeridos

1. **Cálculo puro**: 3 de 4 → "75.00"; 0 de 3 → "0.00"; sin historias → "No disponible"; 2 de 3 → "66.67"; 1 de 8
   → "12.50"; Story Points 3, 5, 8 y 2 con 3 y 5 hechas → 18 y 8; sin Story Points (una y varias); una "Hecho" sin
   Story Points; conteos de defectos sin cambios.
2. **Servicio**:
   - identificadores 0 → `ErrSprintNotFound` sin llamar al almacén;
   - `ErrSprintNotFound` del almacén se devuelve tal cual;
   - Sprint activo → `ErrSprintNotFinished`;
   - otra falla del almacén se envuelve.
3. **Adaptador** (PostgreSQL), con datos propios:
   - escenarios 1 a 9;
   - Sprint de otro proyecto e inexistente;
   - Sprint activo;
   - Sprint finalizado sin registro;
   - prueba de coherencia con la velocidad de HU-07.
4. **Catálogo** (PostgreSQL): orden `number DESC`, solo finalizados, número del activo, proyecto vacío.
5. **Pantalla**:
   - escenarios 10a, 10b, 10c, 11 y 12;
   - `sprint` inválido, activo y ajeno;
   - formato "75,00 %" y advertencia en singular y plural;
   - sin proyecto, `POST` con 405 y rendimiento.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`) y elegir un proyecto (`/contexto/proyecto`).
2. Con HU-04, HU-23 y HU-14, armar el Sprint 1 con cuatro historias de 3, 5, 8 y 2 Story Points. Pasar a "Hecho" las
   de 3 y 5 y cerrar el Sprint con HU-15.
3. Con HU-06, registrar tres defectos en el Sprint 2 (activo) y resolver uno en él con HU-19. Cerrar el Sprint 2.
4. Abrir `http://localhost:8080/calidad`:
   - la lista muestra "Sprint 2" y "Sprint 1";
   - aparecen los datos del Sprint 2: 3 defectos detectados y 1 resuelto.
5. Elegir el Sprint 1: "2 de 4 historias completadas", "50,00 %", 18 Story Points planificados y 8 completados.
6. Con el Sprint 3 activo, la pantalla muestra "El Sprint 3 sigue en curso; sus métricas se calculan al cerrarlo".
7. Reabrir el defecto resuelto (HU-19) y recargar: en el Sprint 2 pasan a 0 resueltos y 3 detectados.
8. Abrir `/calidad?sprint=abc` y `/calidad?sprint=<id de otro proyecto>`: se muestra el Sprint más reciente, sin
   error.
9. En un proyecto sin Sprints finalizados: "Aún no hay Sprints finalizados".

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/010-quality-metrics/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/010-quality-metrics/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/010-quality-metrics/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Dominio, Almacenamiento, Pantalla), y cada sección se guarda antes de
escribir el código de su bloque.
