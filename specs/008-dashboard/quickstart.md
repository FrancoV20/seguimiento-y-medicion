# Guía rápida: HU-08 – Visualización de Dashboard del Proyecto

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-10 en `main` (y con ella HU-06 y HU-15), además de HU-07, HU-14, HU-20a y HU-20b. Comprobarlo con
  `git log --oneline main` y
  `git grep -n "func (s \*QualityService) SprintQuality\|type QualityReader\|CREATE TABLE defects\|CREATE TABLE sprint_story_results"`.
- Rama: `git switch main; git pull; git switch -c HU-08`. Solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-08 no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "Velocity|Dashboard|HU08" -v ./src/metrics/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/metrics/...
gofmt -l src/metrics src/web src/cmd/server test
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo.

## Escenarios de validación requeridos

Los valores esperados y sus cuentas están en [data-model.md](data-model.md) y en "Datos de ejemplo" de
[spec.md](spec.md).

1. **Dominio** (`SummarizeVelocity`): la tabla de casos de `data-model.md` (21 → "10.50", 92 en 12 → "7.67",
   0,125 → "0.13", sin Sprints → "No disponible").
2. **Gráficos** (`dashboardBarHeights`): `[2, 1, 2, 3]` → `[80, 40, 80, 120]`; `[8, 13]` → `[74, 120]`; todo en 0
   → todo en 0; `[1, 1000]` → `[1, 120]`.
3. **Vista**: desviación "+20,00 %", "-8,50 %", "0,00 %" y "No disponible"; promedio con coma; defectos en
   singular, plural y ninguno.
4. **Catálogo** (PostgreSQL): cantidades 2, 1, 2 y 3 de «Alfa»; 3 defectos abiertos; Sprints 1 y 2 finalizados en
   orden; nada de «Beta»; proyecto vacío en ceros.
5. **Pantalla**:
   - escenarios 1 a 12 con los proyectos de prueba;
   - más de 10 Sprints (Sprints 3 a 12 en el gráfico, acumulada 92, promedio "7,67");
   - desviación parcial ("Cálculo parcial: basado en 1 de 2 historias");
   - coherencia de las barras con HU-10 y HU-07;
   - sin proyecto (303), `POST` (405) y rendimiento.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`) y crear un proyecto nuevo; abrir
   `http://localhost:8080/dashboard`: "Aún no hay datos suficientes para calcular métricas".
2. Con HU-22, HU-23, HU-24 y HU-14, armar el proyecto «Alfa» de "Datos de ejemplo" de la spec:
   - Sprint 1 con A (8) terminada y P1 (3) sin terminar; cerrarlo con HU-15;
   - Sprint 2 con B (13, 10 horas estimadas, 8 + 4 horas registradas) terminada y P2 (2) sin terminar; cerrarlo;
   - Sprint 3 activo con C (1) "En Sprint", D (3) y E (5) "En progreso", F (2) "Hecho".
3. Con HU-06 y HU-19, registrar 5 defectos y resolver 2.
4. Abrir el Dashboard y comprobar:
   - historias por estado 2, 1, 2 y 3, en el gráfico y en la tabla;
   - "Sprint 3 en curso", 1, 2 y 1 historias, "11 Story Points planificados y 2 completados";
   - barras Sprint 1 (8) y Sprint 2 (13), "Velocidad acumulada: 21 Story Points", "Promedio: 10,50 Story Points por
     Sprint";
   - "Desviación de esfuerzo del Sprint 2: +20,00 %", sin aviso parcial;
   - "3 defectos abiertos".
5. Desactivar JavaScript en el navegador y recargar: se ven igual los gráficos y las tablas.
6. Cambiar al proyecto «Beta»: no aparece nada de «Alfa».
7. Detener la base (`docker compose stop`) y recargar: página de error 503, sin gráficos ni números.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/008-dashboard/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/008-dashboard/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/008-dashboard/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Dominio de velocidad, Catálogo, Vista y gráficos, Pantalla), y cada
sección se guarda antes de escribir el código de su bloque.
