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

## Verificación manual (T033)

Con la base al día, `DATABASE_URL` y `COOKIE_SECRET` definidas, ejecutar `go run ./src/cmd/server` (con
`$env:ADDR = ":8081"` si el 8080 está ocupado), elegir el proyecto desde **Cambiar proyecto** y abrir **Dashboard**.
El proyecto «Alfa» se arma con las pantallas de Backlog, Sprints, Tablero, Registro de esfuerzo, Cierre de Sprint y
Defectos, como la tabla "Datos de ejemplo" de la spec.

| # | Escenario | Pasos | Resultado esperado con «Alfa» |
|---|---|---|---|
| 1 | Sin datos suficientes | Crear un proyecto nuevo y abrir el Dashboard | Solo "Aún no hay datos suficientes para calcular métricas", sin gráficos |
| 2 | Historias por estado | Abrir el Dashboard de «Alfa» | Gráfico y tabla con 2, 1, 2 y 3 |
| 3 | Sprint en curso | Ídem | "Sprint 3 en curso", 1, 2 y 1 historias, "11 Story Points planificados y 2 completados" |
| 4 | Sin Sprint en curso | Proyecto con historias en el Backlog y sin Sprint activo | "No hay un Sprint en curso" y el resto igual |
| 5 | Velocidad | Abrir el Dashboard de «Alfa» | Barras Sprint 1 (8) y Sprint 2 (13), "Velocidad acumulada: 21 Story Points", "Promedio: 10,50 Story Points por Sprint" |
| 6 | Sin Sprints finalizados | Proyecto con un Sprint activo y ninguno cerrado | El mensaje en el bloque de velocidad y desviación; se ven las historias y el Sprint en curso |
| 7 | Desviación | Abrir el Dashboard de «Alfa» | "Desviación de esfuerzo del Sprint 2: +20,00 %", sin aviso parcial |
| 8 | Desviación no disponible | Cerrar un Sprint cuya historia "Hecho" no tiene horas estimadas | "No disponible" y la velocidad igual se muestra |
| 9 | Defectos abiertos | Registrar 5 defectos en «Alfa» y resolver 2 | "3 defectos abiertos" |
| 10 | Otro proyecto | Cambiar a otro proyecto | No aparece nada de «Alfa» |
| 11 | Sin JavaScript | Desactivar JavaScript en el navegador y recargar | Se ven igual los gráficos y las tablas |
| 12 | Error al leer | Detener la base (`docker compose stop`) y volver a entrar desde el menú | Página "Servicio no disponible" (503), sin gráficos ni números |

Los resultados observados están en [evidencia-manual.txt](evidencia-manual.txt). La verificación se hizo
con el proyecto «PRUEBA» y no con «Alfa», así que ahí se anotan los valores esperados con esos datos y los que se
vieron. Al final de ese archivo está el "Recorrido del revisor", para que quien apruebe el PR haga su propia
verificación.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/008-dashboard/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/008-dashboard/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/008-dashboard/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Dominio de velocidad, Catálogo, y Vista, gráficos y pantalla, que
se trabajaron como un solo bloque rojo/verde para reducir commits), y cada sección se guarda antes de escribir el
código de su bloque.
