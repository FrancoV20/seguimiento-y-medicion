# Guía rápida: HU-18 – Planning Poker: Nueva Ronda y Estimación Acordada

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- **HU-05b integrada en `main`** (y con ella HU-05a), además de HU-02, HU-20a y HU-20b. Sin HU-05b no se empieza
  (T001).
- Rama: `git switch main; git pull; git switch -c HU-18`.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base con todas las migraciones de `main` aplicadas (`migrate -path db/migrations -database $env:DATABASE_URL up`).
  HU-18 no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "HU18|CheckNextRound|CheckAgree|OpenNextRound|AgreeEstimate|NextRound|PokerCatalog" -v ./src/poker/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/poker/...
```

Como en el CI: `gofmt -l .` (sin salida), `go vet ./...` y `go test ./... -p 1 -race`.

## Escenarios de validación requeridos

1. **Reglas** (sin base): las tablas de `TestCheckNextRound` y `TestCheckAgree` de [data-model.md](data-model.md),
   incluidos los órdenes de rechazo (abierta antes que "no es la última"; sin cerrar antes que "no es la última").
2. **Nueva ronda** (PostgreSQL): «Login» 1 → 2 y después 2 → 3; desde la 1 con la 2 cerrada → `ErrNotLatestRound`;
   «Reportes» → `ErrRoundAlreadyOpen`; «Alta» → `ErrStoryInFinishedSprint`; inexistente y de «Beta» →
   `ErrRoundNotFound`; dos a la vez → una sola ronda nueva; falla sin cambios.
3. **Estimación** (PostgreSQL): «Pagos» 5 y después 8 (reemplaza); «Login» 5 con divergencia; «Reportes» desde la 2
   (sin cerrar) y desde la 1 (no es la última) → 3 sin cambios; «Alta» → 3 sin cambios; contra el cierre de Sprint en
   los dos órdenes, sin bloqueo mutuo; contra "Nueva ronda" desde la misma ronda; falla sin cambios.
4. **Catálogo**: solo la última ronda cerrada de cada historia tiene `Latest`; «Alta» tiene `InFinishedSprint`.
5. **Pantalla**: los 12 escenarios con sus códigos (303/422/409/404/500), acciones solo en la última ronda cerrada,
   recordatorio con divergencia, acciones deshabilitadas en «Alta», Backlog con los Story Points nuevos, recarga sin
   reenviar y títulos escapados.

## Verificación manual

1. Con «Alfa», Ana, Beto y Caro y las historias «Login», «Pagos», «Reportes» y «Alta» (HU-21, HU-22), votar y cerrar
   con HU-05a y HU-05b las rondas 1 con los valores de [spec.md](spec.md), "Datos de ejemplo". Para «Reportes», abrir
   después la ronda 2 y votar como Ana. Darle 3 Story Points a «Reportes» y «Alta» desde el Backlog.
2. Para «Alta», asignarla a un Sprint 1 y finalizarlo con HU-15; si HU-15 no está, en SQL:
   `UPDATE sprints SET status = 'Finalizado' WHERE id = …` (dejando su asignación).
3. En `http://localhost:8080/planning-poker`, "Rondas cerradas": «Login», «Pagos» y «Alta» muestran "Nueva ronda" y
   "Estimación acordada"; la ronda 1 de «Reportes», no. «Login» muestra el recordatorio de divergencia; «Alta» muestra
   las acciones deshabilitadas con "Sprint finalizado".
4. "Nueva ronda" en «Login»: "Ronda 2 abierta para la historia «Login»"; la ronda 2 aparece abierta y la 1 sigue
   cerrada con Ana 2, Beto 13 y Caro 5.
5. Como Ana, votar 8 en la ronda 2; como Beto, ver "Votó" sin valor. Votar como Beto 8 y Caro 5 y cerrar la ronda 2.
6. Registrar 8 en la ronda 2 de «Login»: "Estimación registrada: «Login» queda con 8 Story Points"; `/backlog` muestra
   8.
7. Registrar 5 en «Pagos»: «Pagos» queda con 5. Registrar 4 cambiando el valor a mano: "El valor no pertenece a la
   serie de Fibonacci".
8. Desde una pestaña abierta antes de abrir la ronda 2 de «Reportes», enviar "Nueva ronda" y "Registrar estimación" de
   su ronda 1: "Ya hay una ronda abierta para esta historia" y "Hay una ronda más reciente de esta historia".
9. Enviar la estimación de «Alta» quitando `disabled` con las herramientas del navegador: "La historia pertenece a un
   Sprint finalizado; sus Story Points no se pueden modificar"; «Alta» sigue con 3.

## Evidencias a guardar en el PR

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 -Append specs/018-poker-consensus/evidencia-red.txt
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/018-poker-consensus/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/018-poker-consensus/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Reglas, Nueva ronda, Estimación, Pantalla), cada una antes del código de
su bloque.
