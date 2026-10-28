# Quickstart: Actualización de estado de historias (Tablero)

## Prerequisites

- Go 1.27.1 instalado; verificar con `go version`.
- Docker Desktop instalado y en ejecución.
- Herramienta `golang-migrate/migrate` instalada según el [README.md](../../README.md).
- HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`; en tu rama, traerlas con
  `git merge origin/main`.
- Variable `DATABASE_URL` configurada. Ajustar el puerto al que publica tu `docker-compose.yml` o
  `docker-compose.override.yml` (por ejemplo 5432 o 5433):
  `postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable`.
- No subir al repositorio claves personales; usar `docker-compose.override.yml`.

## Run the application

```powershell
docker compose up -d postgres
migrate -path db/migrations -database $env:DATABASE_URL up
```

Para probar a mano hace falta un proyecto activo con un Sprint activo y historias asignadas:

1. Crear el proyecto (HU-21) y elegirlo en `http://localhost:8080/contexto/proyecto` (HU-20b).
2. Crear las historias (HU-22) y un Sprint con tres historias (HU-23). Si esas historias todavía no
   están integradas, insertar los datos por SQL respetando el esquema de `data-model.md` y dejar las
   historias en `'En Sprint'`.
3. Abrir `http://localhost:8080/tablero`.

```powershell
$env:DATABASE_URL = "postgres://sym_user:sym_pass@localhost:5432/seguimiento_y_medicion?sslmode=disable"
go run ./src/cmd/server
```

Las pruebas automáticas no dependen de HU-21, HU-22 ni HU-23: crean sus propios proyectos, historias,
Sprints y horas. Esta feature no agrega migraciones.

## Run the tests

```powershell
docker compose up -d postgres
go test -count=1 -p 1 ./...
```

`-count=1` evita resultados en caché y `-p 1` evita que los paquetes de integración se pisen al
compartir una base.

## Required validation scenarios

Las pruebas deben cubrir como mínimo:

1. Regla de transiciones (unitaria): las tres permitidas, y rechazadas En Sprint → Hecho, En progreso →
   En Sprint, Hecho → En Sprint, cualquier destino Pendiente y las repetidas; `NextAction` y `ActionTransition`
   de cada estado.
2. Servicio y almacenamiento (PostgreSQL real):
   - comenzar, completar y reabrir actualizan el estado;
   - completar sin horas devuelve la advertencia; con horas, no;
   - historia sin Sprint, de un Sprint finalizado, de otro proyecto o inexistente: rechazada y sin
     cambios;
   - estado ya aplicado: no cambia y no es un error de almacenamiento;
   - estado esperado distinto del real: `ErrStatusChanged` y sin cambios;
   - acción que no corresponde al estado real (`comenzar` sobre Hecho, `reabrir` sobre En Sprint):
     `ErrStatusChanged` y sin cambios;
   - dos cambios simultáneos sobre la misma historia: solo uno se aplica.
3. Handler (con `cookiejar` y proyecto activo elegido por el flujo de HU-20b):
   - cada acción válida: 303 a `/tablero`, mensaje una sola vez y estado nuevo; con dos proyectos, solo
     afecta al activo;
   - Story Points completados antes y después de completar, reabrir y enviar dos veces;
   - cada rechazo con su mensaje y código (409 o 422) y el estado sin cambios;
   - falla de guardado: "La operación no pudo completarse", sin éxito y sin cambios;
   - recarga después de un cambio: no se repite ni repite el mensaje;
   - sin proyecto activo: redirección de HU-20b y nada cambia; un campo de proyecto enviado se ignora.
4. Vista:
   - Sprint, grupos por estado, Story Points completados de planificados y una sola acción por historia;
   - sin Sprint activo y Sprint sin historias: sus mensajes y ninguna acción;
   - historia sin Story Points: "Sin estimar"; título con `<script>` se ve como texto;
   - falla de lectura: página de error de la aplicación.
5. Seguridad: ninguna respuesta de error contiene texto de consultas ni rutas.
6. PostgreSQL: `PostgreSQLStoryStatusStore` y `PostgreSQLBoardCatalog` contra PostgreSQL real con
   fixtures propias.

## Acceptance evidence

Para cada escenario, conservar el caso de prueba y su resultado en la revisión del cambio. Las pruebas
ejecutan las reglas reales y el almacenamiento real; solo se usa un almacenamiento que falla para
simular la falla de guardado y de lectura.

Guardar la evidencia en `specs/014-story-status/`. En PowerShell, ejecutar `chcp 65001` antes para que
las tildes y la ñ no salgan como `├®`:

```powershell
chcp 65001
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/evidencia-red.txt
```

Esa primera ejecución se guarda antes de implementar, con las pruebas nuevas fallando. Al cerrar la
historia, la ejecución final y la cobertura:

```powershell
go test -count=1 -p 1 -v ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/evidencia-pruebas.txt
go test -count=1 -p 1 -cover ./... 2>&1 | Out-File -Encoding utf8 specs/014-story-status/cobertura.txt
```

## Prueba manual (datos de demo por SQL)

El puerto 8080 puede estar ocupado por otro programa (por ejemplo un contenedor de otro proyecto): usar otro puerto con `ADDR`. Reemplazar `USUARIO` por el de `DATABASE_URL`.

```powershell
@"
WITH p AS (
  INSERT INTO projects (name, start_date, end_date, status)
  VALUES ('Demo HU14 - Proyecto Tablero', DATE '2026-10-01', DATE '2026-12-31', 'Activo') RETURNING id
),
m AS (INSERT INTO members (display_name) VALUES ('Demo HU14 - Integrante') RETURNING id),
pm AS (INSERT INTO project_members (project_id, member_id) SELECT p.id, m.id FROM p, m RETURNING project_id),
s AS (
  INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
  SELECT id, 1, 'Demo del Tablero', DATE '2026-10-20', DATE '2026-11-03', 'Activo' FROM p RETURNING id, project_id
),
u AS (
  INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
  SELECT s.project_id, v.title, 'Historia de demo', 'Media', v.points, v.status, s.id
  FROM s, (VALUES ('Demo: login', 5, 'En Sprint'), ('Demo: reporte', 3, 'En progreso'),
                  ('Demo: exportar', 8, 'Hecho')) AS v(title, points, status)
  RETURNING id, active_sprint_id
)
INSERT INTO sprint_story_assignments (sprint_id, story_id) SELECT active_sprint_id, id FROM u;
"@ | docker exec -i seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion

$env:ADDR = ":8081"
go run ./src/cmd/server
```

1. Abrir `http://localhost:8081/tablero` sin elegir proyecto: redirige a la selección y, al elegir "Demo HU14 - Proyecto Tablero", vuelve al Tablero.
2. Comprobar las tres columnas, "8 de 16" Story Points completados y una sola acción por historia.
3. Comenzar, Completar (sin horas: advertencia en el mismo mensaje) y Reabrir; recargar con F5 después de un cambio: el mensaje no se repite.
4. Con `UPDATE sprints SET status = 'Finalizado' ...` se ve "No hay un Sprint activo"; sin historias, "El Sprint no tiene historias asignadas".

Atención: `go test ./...` con `DATABASE_URL` definida vacía la base local (la prueba de HU-01 hace `TRUNCATE`). Probar a mano antes de correr la suite, o volver a cargar los datos.

Para borrar los datos de demo:

```powershell
@"
DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE title LIKE 'Demo: %');
DELETE FROM user_stories WHERE project_id IN (SELECT id FROM projects WHERE name = 'Demo HU14 - Proyecto Tablero');
DELETE FROM projects WHERE name = 'Demo HU14 - Proyecto Tablero';
DELETE FROM members WHERE display_name = 'Demo HU14 - Integrante';
"@ | docker exec -i seguimiento-y-medicion-db psql -U USUARIO -d seguimiento_y_medicion
```