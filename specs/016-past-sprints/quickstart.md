# Guía rápida: HU-16 – Consulta de Sprints Anteriores

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- **HU-15 integrada en `main`**, con su migración de `sprint_story_results`, además de HU-07, HU-20a y HU-20b. Sin
  HU-15 no se empieza (T001). Para la verificación manual, además HU-21, HU-22, HU-23 y HU-14.
- Rama: `git switch main; git pull; git switch -c HU-16`. Solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día, con la migración de HU-15: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-16 no
  agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "PastSprint|HU16" -v ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/web/...
gofmt -l src/web test
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo nuevo.

## Escenarios de validación requeridos

Los valores esperados están en "Datos de ejemplo" de [spec.md](spec.md) y en las tablas de
[data-model.md](data-model.md).

1. **Vista** (sin base): "Sprint 2" antes que "Sprint 1"; fechas en dd/mm/aaaa; "Sin estimar"; las cuatro
   combinaciones de `buildPastSprintDetailView` (Sprint 2, Sprint 1, «Gamma» 1 y «Gamma» 2 sin registro).
2. **Catálogo** (PostgreSQL):
   - la lista de «Alfa» es [2, 1] y la de «Beta», vacía;
   - el Sprint 2 trae «Login», «Tablero», «Perfil» y «Reportes» en ese orden, con 5, 3, 2 y 8 y sus estados al
     cerrar;
   - el Sprint 3 (en curso), un Sprint de «Gamma» pedido con «Alfa» y el identificador 2147483647 dan `NotFound`.
3. **Coherencia**: la velocidad de HU-07 es igual a la suma de las completadas del registro (3, 10 y 0).
4. **Pantalla**: los escenarios 1 a 9, la redirección sin proyecto, `POST` con 405, los textos escapados y el
   rendimiento.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`).
2. Con HU-21, HU-22, HU-23 y HU-14, crear «Alfa» con las historias de "Datos de ejemplo" y armar el Sprint 1 con
   «Alta»; pasarla a "Hecho" y cerrarlo con HU-15. Armar el Sprint 2 con «Login», «Tablero», «Perfil» y «Reportes»;
   pasar las tres primeras a "Hecho" y «Reportes» a "En progreso", y cerrarlo. Crear el Sprint 3 ("Exportar e
   informar") y dejarlo en curso.
3. Abrir `http://localhost:8080/sprints-anteriores`: "Sprint 2" y después "Sprint 1", con objetivo y fechas; el 3 no
   figura.
4. Entrar al Sprint 2:
   - "Velocidad: 10 Story Points";
   - Completadas: «Login» 5, «Tablero» 3 y «Perfil» 2;
   - No completadas: «Reportes» 8, "En progreso", "volvió al Backlog";
   - no hay formularios ni botones.
5. Entrar al Sprint 1: «Alta» 3, "Velocidad: 3 Story Points" y "Todas las historias se completaron".
6. Cambiar la dirección por la del Sprint 3 y por `/sprints-anteriores/abc`: página 404.
7. Con «Beta» (sin Sprints) como proyecto activo: "Aún no hay Sprints anteriores".
8. Borrar la cookie del proyecto y abrir el detalle del Sprint 2: se pide elegir proyecto y, al elegirlo, vuelve a ese
   detalle.
9. Detener la base (`docker compose stop`) y recargar: página de error 503, sin datos.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/016-past-sprints/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/016-past-sprints/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/016-past-sprints/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Catálogo, Lista, Detalle), y cada sección se guarda antes de escribir
el código de su bloque.
