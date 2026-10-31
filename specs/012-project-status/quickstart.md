# Guía rápida: HU-12 – Consulta del Estado de un Proyecto

## Requisitos previos

- Go 1.27.1 (`go version`), Docker Desktop en ejecución y `golang-migrate/migrate` según el
  [README.md](../../README.md).
- HU-01, HU-14, HU-20a y HU-20b en `main` (ya están). Para la verificación manual, además HU-22 y HU-23.
- Rama: `git switch main; git pull; git switch -c HU-12`. Solo si no hay cambios sin commitear.
- `DATABASE_URL` con el puerto de tu `docker-compose.yml` o `docker-compose.override.yml`:
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- Base al día: `migrate -path db/migrations -database $env:DATABASE_URL up`. HU-12 no agrega migraciones.

## Pruebas

```powershell
chcp 65001
go test -count=1 -p 1 ./...
go test -count=1 -p 1 -run "StatusOn|ProgressPercentage|ProjectStatus|HU12" -v ./src/project/... ./src/web/... ./test/...
go test -count=1 -p 1 -cover ./src/project/...
gofmt -l src/project src/web test
go vet ./...
```

`-count=1` evita la caché y `-p 1` evita que los paquetes de integración se pisen en la misma base. `gofmt -l` no
tiene que listar ningún archivo nuevo.

## Escenarios de validación requeridos

Los valores esperados están en "Datos de ejemplo" de [spec.md](spec.md) y en las tablas de
[data-model.md](data-model.md).

1. **Reglas** (sin base):
   - `StatusOn` con los cinco proyectos del Issue, las 00:00 y las 23:59 de los bordes, el día anterior al inicio,
     el día siguiente al fin y el caso en UTC-3;
   - `ProgressPercentage`: 4 de 8 → "50.00"; 3 de 3 → "100.00"; 0 de 0 → "0.00"; 2 de 3 → "66.67"; 4 de 7 → "57.14";
     1 de 32 → "3.13".
2. **Catálogo** (PostgreSQL): «Alfa» con 2, 1, 1 y 4; «Gamma» con 0, 0, 0 y 3; «Beta» en ceros; nombre y fechas;
   nada de otro proyecto; proyecto inexistente → `NotFound`.
3. **Vista**: "50,00 %", fechas en dd/mm/aaaa, cuatro filas con ceros, "Sprint 2 en curso" y "No hay un Sprint en
   curso".
4. **Pantalla**: escenarios 1 a 10 con el reloj en 28/10/2026, `POST` con 405 y rendimiento.

## Verificación manual

1. Levantar la aplicación (`go run ./src/cmd/server`).
2. Crear «Alfa» (01/10/2026 al 30/11/2026), elegirlo y, con HU-22 y HU-23, cargar 8 historias y dejar el Sprint 2
   activo ("Pagos online") con el reparto de "Datos de ejemplo" (2 "Pendiente", 1 "En Sprint", 1 "En progreso" y 4
   "Hecho"; HU-14 para mover los estados).
3. Abrir `http://localhost:8080/estado`: "Estado: En curso" (si hoy está entre las fechas), "4 de 8 historias
   completadas (50,00 %)", las cuatro cantidades y "Sprint 2 en curso" con "Objetivo: Pagos online" y sus fechas.
4. Crear un proyecto con fechas futuras («Beta»): "Estado: No iniciado", cantidades en 0, "0 de 0 historias
   completadas (0,00 %)" y "No hay un Sprint en curso".
5. Crear un proyecto que empiece hoy y otro que termine hoy: los dos "En curso".
6. Borrar la cookie del proyecto y abrir `/estado`: se pide elegir uno y, al elegirlo, vuelve a "Estado del
   proyecto".
7. Detener la base (`docker compose stop`) y recargar: página de error 503, sin datos.

## Evidencias a guardar en el PR

```powershell
chcp 65001
cmd /c "go test -count=1 -p 1 -v ./... 2>&1" | Out-File -Encoding utf8 specs/012-project-status/evidencia-pruebas.txt
cmd /c "go test -count=1 -p 1 -cover ./... 2>&1" | Out-File -Encoding utf8 specs/012-project-status/cobertura.txt
```

`evidencia-red.txt` se completa por secciones (Reglas, Catálogo, Pantalla), y cada sección se guarda antes de escribir
el código de su bloque.

## Resultado de la verificación manual

Realizada el 30/10/2026 con la aplicación en ejecución (`go run ./src/cmd/server`) y PostgreSQL local. Los datos se
cargaron con un script SQL (proyectos «HU12 manual Alfa», «Beta», «Gamma», «Delta» y «Epsilon», con las fechas de
"Datos de ejemplo"; «Delta» empieza y «Epsilon» termina el 30/10/2026, la fecha de hoy) y se borraron al terminar.

| Escenario | Proyecto | Resultado observado | Estado |
|---|---|---|---|
| 1 Proyecto en curso | Alfa | "Estado: En curso", "Del 01/10/2026 al 30/11/2026", "4 de 8 historias completadas (50,00 %)", Pendiente 2, En Sprint 1, En progreso 1, Hecho 4, "Sprint 2 en curso", "Objetivo: Pagos online", "Del 19/10/2026 al 01/11/2026" | OK |
| 2 No iniciado | Beta | "Estado: No iniciado" | OK |
| 3 Finalizado | Gamma | "Estado: Finalizado", "3 de 3 historias completadas (100,00 %)" | OK |
| 4 Empieza hoy | Delta | "Estado: En curso" | OK |
| 5 Termina hoy | Epsilon | "Estado: En curso" | OK |
| 6 Sin historias | Beta | cuatro cantidades en 0 y "0 de 0 historias completadas (0,00 %)", sin error | OK |
| 7 Sin Sprint en curso | Gamma | "No hay un Sprint en curso" y el resto igual | OK |
| 8 Otro proyecto | Alfa | 8 historias y 4 "Hecho"; no aparecen las 3 de «Gamma» (nada de "7 de 11") | OK |
| 9 Sin proyecto | — | en ventana privada, `/estado` pide elegir proyecto y, al elegirlo, vuelve a "Estado del proyecto" | OK |
| 10 Error al consultar | cualquiera | con la base detenida, página de error sin datos del proyecto ni texto técnico; al volver la base, la pantalla funciona sin reiniciar la aplicación | OK |
| Solo lectura | Alfa | el código de la página no tiene `<script` ni `<form` | OK |